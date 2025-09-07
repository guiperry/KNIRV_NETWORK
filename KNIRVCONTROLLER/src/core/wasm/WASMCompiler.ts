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

// Jest-compatible module URL resolution
const getModuleUrl = () => {
  if (typeof process !== 'undefined' && process.env.NODE_ENV === 'test') {
    return 'file://' + process.cwd() + '/src/core/wasm/WASMCompiler.ts';
  }
  try {
    const importMeta = eval('import.meta');
    if (importMeta && importMeta.url) {
      return importMeta.url;
    }
  } catch {
    // Fallback for CommonJS
    return 'file://' + __filename;
  }
  return 'file://' + process.cwd();
};

const __filename = fileURLToPath(getModuleUrl());
const __dirname = dirname(__filename);

export interface WASMCompilationOptions {
  target?: 'web' | 'nodejs' | 'bundler';
  optimize?: boolean;
  debug?: boolean;
  features?: string[];
  outputDir?: string;
  // New options for LoRA adapter compilation
  loraAdapterMode?: boolean;
  dynamicCompilation?: boolean;
  embeddedChainIntegration?: boolean;
}

export interface LoRAAdapterCompilationRequest {
  adapterId: string;
  adapterName: string;
  baseModel: string;
  rank: number;
  alpha: number;
  weightsA: Float32Array;
  weightsB: Float32Array;
  targetArchitecture: 'wasm32' | 'wasm64';
  optimizationLevel: 'none' | 'basic' | 'aggressive';
}

export interface LoRAAdapterWASMModule extends WASMModule {
  adapterId: string;
  adapterName: string;
  applyWeights: (input: Float32Array) => Promise<Float32Array>;
  getAdapterInfo: () => { name: string; version: string; [key: string]: unknown };
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
  async compile(rustCode: string, _options: WASMCompilationOptions = {}): Promise<WASMModule> {
    if (!this.ready) {
      throw new Error('WASM Compiler not initialized');
    }

    const compilationId = this.generateId();
    const startTime = Date.now();

    logger.info({ compilationId, options: _options }, 'Starting WASM compilation');

    try {
      // Create temporary project directory
      const projectDir = join(this.tempDir, compilationId);
      await fs.mkdir(projectDir, { recursive: true });

      // Write Rust code to lib.rs
      const srcDir = join(projectDir, 'src');
      await fs.mkdir(srcDir, { recursive: true });
      await fs.writeFile(join(srcDir, 'lib.rs'), rustCode);

      // Create Cargo.toml
      const cargoToml = this.generateCargoToml(_options);
      await fs.writeFile(join(projectDir, 'Cargo.toml'), cargoToml);

      // Compile with wasm-pack
      const wasmModule = await this.runWasmPack(projectDir, _options);

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

  private generateCargoToml(_options: WASMCompilationOptions): string {
    const features = _options.features || [];
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

  private async runWasmPack(projectDir: string, _options: WASMCompilationOptions): Promise<WASMModule> {
    const target = _options.target || 'web';
    const outputDir = _options.outputDir || 'pkg';


    const args = [
      'build',
      '--target', target,
      '--out-dir', outputDir,
      '--scope', 'knirv'
    ];

    if (!_options.debug) {
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
            const wasmModule = await this.loadCompiledModule(projectDir, outputDir, _options);
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

  private async loadCompiledModule(projectDir: string, outputDir: string, _options: WASMCompilationOptions): Promise<WASMModule> {
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
    } catch {
      logger.warn('TypeScript definitions not found');
    }

    return {
      wasmBytes,
      jsBindings,
      typeDefinitions,
      metadata: {
        size: wasmBytes.length,
        compilationTime: 0, // Will be set by caller
        features: _options.features || [],
        target: _options.target || 'web'
      }
    };
  }

  /**
   * Compile the default agent-core WASM module
   */
  async compileAgentCore(_options: WASMCompilationOptions = {}): Promise<WASMModule> {
    logger.info('Compiling agent-core WASM module...');

    try {
      // Read the existing Rust code from rust-wasm directory
      const libRsPath = join(this.rustWasmPath, 'src', 'lib.rs');
      const rustCode = await fs.readFile(libRsPath, 'utf-8');

      return await this.compile(rustCode, {
        target: 'web',
        optimize: true,
        features: ['agent-core', 'lora-adapters'],
        ..._options
      });
    } catch (error) {
      logger.error({ error }, 'Failed to compile agent-core WASM module');
      throw error;
    }
  }

  /**
   * Dynamic LoRA adapter compilation for embedded KNIRVCHAIN
   * Compiles LoRA adapter weights into optimized WASM modules for real-time execution
   */
  async compileLoRAAdapter(request: LoRAAdapterCompilationRequest): Promise<LoRAAdapterWASMModule> {
    logger.info({ adapterId: request.adapterId }, 'Compiling LoRA adapter to WASM...');

    const startTime = Date.now();


    try {
      // Generate Rust code for LoRA adapter
      const rustCode = this.generateLoRAAdapterRustCode(request);

      // Compile with LoRA-specific optimizations
      const wasmModule = await this.compile(rustCode, {
        target: 'web',
        optimize: request.optimizationLevel !== 'none',
        features: ['lora-adapter', 'embedded-chain', 'wasm-bindgen'],
        loraAdapterMode: true,
        dynamicCompilation: true,
        embeddedChainIntegration: true
      });

      // Create enhanced LoRA adapter WASM module
      const loraWasmModule: LoRAAdapterWASMModule = {
        ...wasmModule,
        adapterId: request.adapterId,
        adapterName: request.adapterName,
        applyWeights: async (input: Float32Array) => {
          return await this.executeLoRAWeightApplication(wasmModule, input, request);
        },
        getAdapterInfo: () => ({
          name: request.adapterName,
          version: '1.0.0',
          adapterId: request.adapterId,
          adapterName: request.adapterName,
          baseModel: request.baseModel,
          rank: request.rank,
          alpha: request.alpha,
          compilationTime: Date.now() - startTime,
          wasmSize: wasmModule.wasmBytes.length
        })
      };

      logger.info({
        adapterId: request.adapterId,
        compilationTime: Date.now() - startTime,
        wasmSize: wasmModule.wasmBytes.length
      }, 'LoRA adapter WASM compilation completed');

      return loraWasmModule;

    } catch (error) {
      logger.error({ error, adapterId: request.adapterId }, 'Failed to compile LoRA adapter to WASM');
      throw error;
    }
  }

  /**
   * Generate Rust code for LoRA adapter WASM module
   */
  private generateLoRAAdapterRustCode(request: LoRAAdapterCompilationRequest): string {
    return `
use wasm_bindgen::prelude::*;
use js_sys::Float32Array;

// LoRA Adapter: ${request.adapterName}
// Base Model: ${request.baseModel}
// Rank: ${request.rank}, Alpha: ${request.alpha}

#[wasm_bindgen]
pub struct LoRAAdapter {
    rank: usize,
    alpha: f32,
    weights_a: Vec<f32>,
    weights_b: Vec<f32>,
}

#[wasm_bindgen]
impl LoRAAdapter {
    #[wasm_bindgen(constructor)]
    pub fn new() -> LoRAAdapter {
        LoRAAdapter {
            rank: ${request.rank},
            alpha: ${request.alpha},
            weights_a: vec![${Array.from(request.weightsA).join(', ')}],
            weights_b: vec![${Array.from(request.weightsB).join(', ')}],
        }
    }

    #[wasm_bindgen]
    pub fn apply_weights(&self, input: &Float32Array) -> Float32Array {
        let input_vec: Vec<f32> = input.to_vec();
        let mut output = vec![0.0; input_vec.len()];

        // Apply LoRA transformation: output = input + (alpha/rank) * (B * A * input)
        let scaling = self.alpha / self.rank as f32;

        // Simplified LoRA application for WASM
        for i in 0..input_vec.len().min(self.weights_a.len()) {
            let lora_contribution = scaling * self.weights_b[i % self.weights_b.len()] * self.weights_a[i];
            output[i] = input_vec[i] + lora_contribution;
        }

        Float32Array::from(&output[..])
    }

    #[wasm_bindgen]
    pub fn get_adapter_id(&self) -> String {
        "${request.adapterId}".to_string()
    }

    #[wasm_bindgen]
    pub fn get_adapter_name(&self) -> String {
        "${request.adapterName}".to_string()
    }

    #[wasm_bindgen]
    pub fn get_rank(&self) -> usize {
        self.rank
    }

    #[wasm_bindgen]
    pub fn get_alpha(&self) -> f32 {
        self.alpha
    }
}

// Export the adapter for embedded chain integration
#[wasm_bindgen(start)]
pub fn main() {
    console_error_panic_hook::set_once();
}
`;
  }

  /**
   * Execute LoRA weight application using compiled WASM module
   */
  private async executeLoRAWeightApplication(
    wasmModule: WASMModule,
    input: Float32Array,
    request: LoRAAdapterCompilationRequest
  ): Promise<Float32Array> {
    try {
      // In a real implementation, this would:
      // 1. Instantiate the WASM module
      // 2. Call the apply_weights function
      // 3. Return the transformed output

      // For now, simulate the LoRA weight application
      const output = new Float32Array(input.length);
      const scaling = request.alpha / request.rank;

      for (let i = 0; i < input.length; i++) {
        const aWeight = request.weightsA[i % request.weightsA.length];
        const bWeight = request.weightsB[i % request.weightsB.length];
        const loraContribution = scaling * bWeight * aWeight;
        output[i] = input[i] + loraContribution;
      }

      return output;

    } catch (error) {
      logger.error({ error }, 'Failed to execute LoRA weight application');
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

  /**
   * Communication channel with embedded WASM compiler toolchain within agent-core
   * Enables real-time LoRA adapter compilation and deployment
   */
  async establishEmbeddedChainCommunication(embeddedChainUrl: string): Promise<void> {
    logger.info({ embeddedChainUrl }, 'Establishing communication with embedded KNIRVCHAIN...');

    try {
      // Test connection to embedded chain
      const response = await fetch(`${embeddedChainUrl}/health`);
      if (!response.ok) {
        throw new Error(`Embedded chain not accessible: ${response.statusText}`);
      }

      // Register this WASM compiler with the embedded chain
      const registrationResponse = await fetch(`${embeddedChainUrl}/wasm-compiler/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          compilerId: `wasm-compiler-${Date.now()}`,
          capabilities: ['lora-adapter-compilation', 'dynamic-compilation', 'real-time-deployment'],
          version: '1.0.0',
          status: 'ready'
        }),
      });

      if (!registrationResponse.ok) {
        throw new Error(`Failed to register with embedded chain: ${registrationResponse.statusText}`);
      }

      logger.info('Successfully established communication with embedded KNIRVCHAIN');

    } catch (error) {
      logger.error({ error }, 'Failed to establish embedded chain communication');
      throw error;
    }
  }

  /**
   * Deploy compiled LoRA adapter to embedded KNIRVCHAIN
   */
  async deployLoRAAdapterToEmbeddedChain(
    loraWasmModule: LoRAAdapterWASMModule,
    embeddedChainUrl: string
  ): Promise<void> {
    logger.info({ adapterId: loraWasmModule.adapterId }, 'Deploying LoRA adapter to embedded chain...');

    try {
      const deploymentPayload = {
        adapterId: loraWasmModule.adapterId,
        adapterName: loraWasmModule.adapterName,
        wasmBytes: Array.from(loraWasmModule.wasmBytes),
        adapterInfo: loraWasmModule.getAdapterInfo(),
        deploymentTimestamp: Date.now()
      };

      const response = await fetch(`${embeddedChainUrl}/skills`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(deploymentPayload),
      });

      if (!response.ok) {
        throw new Error(`Deployment failed: ${response.statusText}`);
      }

      const result = await response.json();
      logger.info({
        adapterId: loraWasmModule.adapterId,
        deploymentResult: result
      }, 'LoRA adapter deployed successfully to embedded chain');

    } catch (error) {
      logger.error({ error, adapterId: loraWasmModule.adapterId }, 'Failed to deploy LoRA adapter');
      throw error;
    }
  }

  /**
   * Compile and deploy LoRA adapter in one operation
   */
  async compileAndDeployLoRAAdapter(
    request: LoRAAdapterCompilationRequest,
    embeddedChainUrl: string
  ): Promise<LoRAAdapterWASMModule> {
    logger.info({ adapterId: request.adapterId }, 'Compiling and deploying LoRA adapter...');

    try {
      // Compile LoRA adapter to WASM
      const loraWasmModule = await this.compileLoRAAdapter(request);

      // Deploy to embedded chain
      await this.deployLoRAAdapterToEmbeddedChain(loraWasmModule, embeddedChainUrl);

      return loraWasmModule;

    } catch (error) {
      logger.error({ error, adapterId: request.adapterId }, 'Failed to compile and deploy LoRA adapter');
      throw error;
    }
  }

  /**
   * Get compilation status and metrics
   */
  getCompilationMetrics(): unknown {
    return {
      isReady: this.ready,
      tempDir: this.tempDir,
      rustWasmPath: this.rustWasmPath,
      capabilities: [
        'rust-to-wasm',
        'agent-core-compilation',
        'lora-adapter-compilation',
        'dynamic-compilation',
        'embedded-chain-integration'
      ],
      timestamp: Date.now()
    };
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
