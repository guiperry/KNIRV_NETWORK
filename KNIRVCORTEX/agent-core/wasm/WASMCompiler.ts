/**
 * WASM Compiler - Backend WASM compilation pipeline for agent-core
 * Handles compilation of Rust code to WebAssembly for embedded execution
 */

import { spawn } from 'child_process';
import { promises as fs } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import pino from 'pino';

const logger = pino({ name: 'wasm-compiler' });

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export interface WASMCompilationOptions {
  target?: 'web' | 'nodejs' | 'bundler';
  optimize?: boolean;
  debug?: boolean;
  features?: string[];
  outputDir?: string;
}

export interface WASMModule {
  wasmBytes: Uint8Array;
  jsBindings: string;
  typeDefinitions: string;
  metadata: {
    size: number;
    compilationTime: number;
    features: string[];
    target: string;
  };
}

export class WASMCompiler {
  private ready = false;
  private rustWasmPath: string;
  private tempDir: string;

  constructor() {
    this.rustWasmPath = join(__dirname, '../../rust-wasm');
    this.tempDir = join(__dirname, '../../temp');
  }

  async initialize(): Promise<void> {
    logger.info('Initializing WASM Compiler...');

    try {
      // Ensure temp directory exists
      await fs.mkdir(this.tempDir, { recursive: true });

      // Check if wasm-pack is available
      await this.checkWasmPack();

      // Verify Rust toolchain
      await this.checkRustToolchain();

      this.ready = true;
      logger.info('WASM Compiler initialized successfully');
    } catch (error) {
      logger.error({ error }, 'Failed to initialize WASM Compiler');
      throw error;
    }
  }

  private async checkWasmPack(): Promise<void> {
    return new Promise((resolve, reject) => {
      const process = spawn('wasm-pack', ['--version'], { stdio: 'pipe' });
      
      process.on('close', (code) => {
        if (code === 0) {
          logger.info('wasm-pack is available');
          resolve();
        } else {
          reject(new Error('wasm-pack is not installed. Run: curl https://rustwasm.github.io/wasm-pack/installer/init.sh -sSf | sh'));
        }
      });

      process.on('error', () => {
        reject(new Error('wasm-pack is not installed. Run: curl https://rustwasm.github.io/wasm-pack/installer/init.sh -sSf | sh'));
      });
    });
  }

  private async checkRustToolchain(): Promise<void> {
    return new Promise((resolve, reject) => {
      const process = spawn('rustc', ['--version'], { stdio: 'pipe' });
      
      process.on('close', (code) => {
        if (code === 0) {
          logger.info('Rust toolchain is available');
          resolve();
        } else {
          reject(new Error('Rust toolchain is not installed. Visit: https://rustup.rs/'));
        }
      });

      process.on('error', () => {
        reject(new Error('Rust toolchain is not installed. Visit: https://rustup.rs/'));
      });
    });
  }

  /**
   * Compile Rust code to WebAssembly
   */
  async compile(rustCode: string, options: WASMCompilationOptions = {}): Promise<WASMModule> {
    if (!this.ready) {
      throw new Error('WASM Compiler not initialized');
    }

    const compilationId = this.generateId();
    const startTime = Date.now();
    
    logger.info({ compilationId, options }, 'Starting WASM compilation');

    try {
      // Create temporary project directory
      const projectDir = join(this.tempDir, compilationId);
      await fs.mkdir(projectDir, { recursive: true });

      // Write Rust code to lib.rs
      const srcDir = join(projectDir, 'src');
      await fs.mkdir(srcDir, { recursive: true });
      await fs.writeFile(join(srcDir, 'lib.rs'), rustCode);

      // Create Cargo.toml
      const cargoToml = this.generateCargoToml(options);
      await fs.writeFile(join(projectDir, 'Cargo.toml'), cargoToml);

      // Compile with wasm-pack
      const wasmModule = await this.runWasmPack(projectDir, options);

      // Cleanup temporary directory
      await fs.rm(projectDir, { recursive: true, force: true });

      const compilationTime = Date.now() - startTime;
      logger.info({ compilationId, compilationTime }, 'WASM compilation completed');

      return {
        ...wasmModule,
        metadata: {
          ...wasmModule.metadata,
          compilationTime
        }
      };

    } catch (error) {
      logger.error({ error, compilationId }, 'WASM compilation failed');
      throw error;
    }
  }

  private generateCargoToml(options: WASMCompilationOptions): string {
    const features = options.features || [];
    const featureList = features.length > 0 ? `\ndefault = [${features.map(f => `"${f}"`).join(', ')}]` : '';

    return `[package]
name = "knirv-cortex-wasm"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
wasm-bindgen = "0.2"
js-sys = "0.3"
web-sys = "0.3"
serde = { version = "1.0", features = ["derive"] }
serde-wasm-bindgen = "0.6"
console_error_panic_hook = "0.1"

[dependencies.web-sys]
version = "0.3"
features = [
  "console",
  "Performance",
  "Window",
]

[features]${featureList}

[profile.release]
opt-level = "s"
lto = true
codegen-units = 1
panic = "abort"
`;
  }

  private async runWasmPack(projectDir: string, options: WASMCompilationOptions): Promise<WASMModule> {
    const target = options.target || 'web';
    const outputDir = options.outputDir || 'pkg';
    const mode = options.debug ? 'dev' : 'release';

    const args = [
      'build',
      '--target', target,
      '--out-dir', outputDir,
      '--scope', 'knirv'
    ];

    if (!options.debug) {
      args.push('--release');
    }

    return new Promise((resolve, reject) => {
      const process = spawn('wasm-pack', args, {
        cwd: projectDir,
        stdio: 'pipe'
      });

      let stdout = '';
      let stderr = '';

      process.stdout.on('data', (data) => {
        stdout += data.toString();
      });

      process.stderr.on('data', (data) => {
        stderr += data.toString();
      });

      process.on('close', async (code) => {
        if (code === 0) {
          try {
            const wasmModule = await this.loadCompiledModule(projectDir, outputDir, options);
            resolve(wasmModule);
          } catch (error) {
            reject(error);
          }
        } else {
          logger.error({ stdout, stderr, code }, 'wasm-pack compilation failed');
          reject(new Error(`wasm-pack failed with code ${code}: ${stderr}`));
        }
      });

      process.on('error', (error) => {
        reject(error);
      });
    });
  }

  private async loadCompiledModule(projectDir: string, outputDir: string, options: WASMCompilationOptions): Promise<WASMModule> {
    const pkgDir = join(projectDir, outputDir);

    // Read WASM binary
    const wasmPath = join(pkgDir, 'knirv_cortex_wasm_bg.wasm');
    const wasmBytes = await fs.readFile(wasmPath);

    // Read JS bindings
    const jsPath = join(pkgDir, 'knirv_cortex_wasm.js');
    const jsBindings = await fs.readFile(jsPath, 'utf-8');

    // Read TypeScript definitions
    const dtsPath = join(pkgDir, 'knirv_cortex_wasm.d.ts');
    let typeDefinitions = '';
    try {
      typeDefinitions = await fs.readFile(dtsPath, 'utf-8');
    } catch (error) {
      logger.warn('TypeScript definitions not found');
    }

    return {
      wasmBytes,
      jsBindings,
      typeDefinitions,
      metadata: {
        size: wasmBytes.length,
        compilationTime: 0, // Will be set by caller
        features: options.features || [],
        target: options.target || 'web'
      }
    };
  }

  /**
   * Compile the default agent-core WASM module
   */
  async compileAgentCore(options: WASMCompilationOptions = {}): Promise<WASMModule> {
    logger.info('Compiling agent-core WASM module...');

    try {
      // Read the existing Rust code from rust-wasm directory
      const libRsPath = join(this.rustWasmPath, 'src', 'lib.rs');
      const rustCode = await fs.readFile(libRsPath, 'utf-8');

      return await this.compile(rustCode, {
        target: 'web',
        optimize: true,
        features: ['agent-core', 'lora-adapters'],
        ...options
      });
    } catch (error) {
      logger.error({ error }, 'Failed to compile agent-core WASM module');
      throw error;
    }
  }

  /**
   * Build the existing rust-wasm project
   */
  async buildExistingProject(): Promise<WASMModule> {
    logger.info('Building existing rust-wasm project...');

    try {
      const wasmModule = await this.runWasmPack(this.rustWasmPath, {
        target: 'web',
        optimize: true,
        outputDir: '../src/wasm-pkg'
      });

      logger.info('Existing rust-wasm project built successfully');
      return wasmModule;
    } catch (error) {
      logger.error({ error }, 'Failed to build existing rust-wasm project');
      throw error;
    }
  }

  isReady(): boolean {
    return this.ready;
  }

  async cleanup(): Promise<void> {
    logger.info('Cleaning up WASM Compiler...');
    
    try {
      // Clean up temp directory
      await fs.rm(this.tempDir, { recursive: true, force: true });
    } catch (error) {
      logger.warn({ error }, 'Failed to clean up temp directory');
    }

    this.ready = false;
  }

  private generateId(): string {
    return `wasm-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }
}

export default WASMCompiler;
