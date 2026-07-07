/**
 * AssemblyScript WASM Loader
 * Loads and manages the AssemblyScript-compiled WASM module
 */

import pino from 'pino';

const logger = pino({ name: 'assemblyscript-wasm' });

export interface AssemblyScriptWASMModule {
  // Agent Core Functions
  createAgentCore(id: string): boolean;
  initializeAgent(): boolean;
  executeAgent(input: string, context: string): string;
  executeAgentTool(toolName: string, parameters: string, context: string): string;
  loadLoraAdapter(adapter: string): boolean;
  getAgentStatus(): string;

  // Model Functions
  createModel(type: string): boolean;
  loadModelWeights(weightsPtr: number, weightsLen: number): boolean;
  runModelInference(input: string, context: string): string;
  getModelInfo(): string;

  // Utility Functions
  getWasmVersion(): string;
  getSupportedFeatures(): string;
  allocateString(str: string): number;
  deallocateString(ptr: number): void;
  wasmInit(): void;

  // Memory
  memory: WebAssembly.Memory;
}

export class AssemblyScriptWASMLoader {
  private wasmModule: WebAssembly.Module | null = null;
  private wasmInstance: WebAssembly.Instance | null = null;
  private wasmExports: AssemblyScriptWASMModule | null = null;
  private isInitialized = false;

  async initialize(): Promise<void> {
    try {
      logger.info('Loading AssemblyScript WASM module...');

      // Load WASM module
      const wasmBytes = await this.loadWASMBytes();
      this.wasmModule = await WebAssembly.compile(wasmBytes);

      // Create imports object
      const imports = this.createImports();

      // Instantiate WASM module
      this.wasmInstance = await WebAssembly.instantiate(this.wasmModule, imports);
      this.wasmExports = this.wasmInstance.exports as unknown as AssemblyScriptWASMModule;

      // Initialize the WASM module
      if (this.wasmExports.wasmInit) {
        this.wasmExports.wasmInit();
      }

      this.isInitialized = true;
      logger.info('AssemblyScript WASM module loaded successfully');

    } catch (error) {
      logger.error({ error }, 'Failed to load AssemblyScript WASM module');
      throw error;
    }
  }

  private async loadWASMBytes(): Promise<Uint8Array> {
    const isBrowser = typeof window !== 'undefined';
    
    if (isBrowser) {
      // Browser: fetch from build directory
      const response = await fetch('/build/knirv-controller.wasm');
      if (!response.ok) {
        throw new Error(`Failed to fetch WASM: ${response.statusText}`);
      }
      return new Uint8Array(await response.arrayBuffer());
    } else {
      // Node.js: read from file system
      const { promises: fs } = await import('fs');
      const { join } = await import('path');
      const wasmPath = join(process.cwd(), 'build/knirv-controller.wasm');
      return await fs.readFile(wasmPath);
    }
  }

  private createImports(): WebAssembly.Imports {
    return {
      env: {
        // Console logging
        console_log: (ptr: number, len: number) => {
          if (this.wasmExports?.memory) {
            const memory = new Uint8Array(this.wasmExports.memory.buffer);
            const message = new TextDecoder().decode(memory.slice(ptr, ptr + len));
            console.log(`[WASM]: ${message}`);
          }
        },

        // Error handling
        abort: (messagePtr: number, filePtr: number, line: number, column: number) => {
          let message = 'WASM abort';
          if (this.wasmExports?.memory) {
            const memory = new Uint8Array(this.wasmExports.memory.buffer);
            if (messagePtr) {
              const messageLen = memory.indexOf(0, messagePtr) - messagePtr;
              message = new TextDecoder().decode(memory.slice(messagePtr, messagePtr + messageLen));
            }
          }
          throw new Error(`WASM abort: ${message} at line ${line}, column ${column}`);
        },

        // Math functions (if needed)
        ...(typeof Math !== 'undefined' && {
          Math: {
            random: Math.random,
            floor: Math.floor,
            ceil: Math.ceil,
            round: Math.round,
            abs: Math.abs,
            min: Math.min,
            max: Math.max,
            pow: Math.pow,
            sqrt: Math.sqrt
          }
        }),
        ...(typeof Date !== 'undefined' && {
          Date: {
            now: (() => Date.now()) as any
          }
        }),
      }
    } as any;
  }

  // Agent Core Methods
  createAgentCore(id: string): boolean {
    this.ensureInitialized();
    return this.wasmExports!.createAgentCore(id);
  }

  initializeAgent(): boolean {
    this.ensureInitialized();
    return this.wasmExports!.initializeAgent();
  }

  executeAgent(input: string, context: string): string {
    this.ensureInitialized();
    return this.wasmExports!.executeAgent(input, context);
  }

  executeAgentTool(toolName: string, parameters: string, context: string): string {
    this.ensureInitialized();
    return this.wasmExports!.executeAgentTool(toolName, parameters, context);
  }

  loadLoraAdapter(adapter: string): boolean {
    this.ensureInitialized();
    return this.wasmExports!.loadLoraAdapter(adapter);
  }

  getAgentStatus(): string {
    this.ensureInitialized();
    return this.wasmExports!.getAgentStatus();
  }

  // Model Methods
  createModel(type: string): boolean {
    this.ensureInitialized();
    return this.wasmExports!.createModel(type);
  }

  loadModelWeights(weightsPtr: number, weightsLen: number): boolean {
    this.ensureInitialized();
    return this.wasmExports!.loadModelWeights(weightsPtr, weightsLen);
  }

  runModelInference(input: string, context: string): string {
    this.ensureInitialized();
    return this.wasmExports!.runModelInference(input, context);
  }

  getModelInfo(): string {
    this.ensureInitialized();
    return this.wasmExports!.getModelInfo();
  }

  // Utility Methods
  getWasmVersion(): string {
    this.ensureInitialized();
    return this.wasmExports!.getWasmVersion();
  }

  getSupportedFeatures(): string {
    this.ensureInitialized();
    return this.wasmExports!.getSupportedFeatures();
  }

  // Memory Management
  allocateString(str: string): number {
    this.ensureInitialized();
    return this.wasmExports!.allocateString(str);
  }

  deallocateString(ptr: number): void {
    this.ensureInitialized();
    this.wasmExports!.deallocateString(ptr);
  }

  // Status
  isReady(): boolean {
    return this.isInitialized && this.wasmExports !== null;
  }

  getMemory(): WebAssembly.Memory | null {
    return this.wasmExports?.memory || null;
  }

  private ensureInitialized(): void {
    if (!this.isInitialized || !this.wasmExports) {
      throw new Error('AssemblyScript WASM module not initialized');
    }
  }

  // Cleanup
  dispose(): void {
    this.wasmModule = null;
    this.wasmInstance = null;
    this.wasmExports = null;
    this.isInitialized = false;
    logger.info('AssemblyScript WASM module disposed');
  }
}

// Singleton instance
let wasmLoader: AssemblyScriptWASMLoader | null = null;

export async function getAssemblyScriptWASM(): Promise<AssemblyScriptWASMLoader> {
  if (!wasmLoader) {
    wasmLoader = new AssemblyScriptWASMLoader();
    await wasmLoader.initialize();
  }
  return wasmLoader;
}

export default AssemblyScriptWASMLoader;
