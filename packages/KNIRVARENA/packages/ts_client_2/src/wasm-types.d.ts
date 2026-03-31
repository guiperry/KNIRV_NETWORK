// TypeScript declarations for KNIRV-CONTROLLER AssemblyScript WASM module

declare module '../build/knirv-controller' {
  // Agent Core Functions
  export function createAgentCore(id: string): boolean;
  export function initializeAgent(): boolean;
  export function executeAgent(input: string, context: string): string;
  export function executeAgentTool(toolName: string, parameters: string, context: string): string;
  export function loadLoraAdapter(adapter: string): boolean;
  export function getAgentStatus(): string;

  // Model Functions
  export function createModel(type: string): boolean;
  export function loadModelWeights(weightsPtr: number, weightsLen: number): boolean;
  export function runModelInference(input: string, context: string): string;
  export function getModelInfo(): string;

  // Utility Functions
  export function getWasmVersion(): string;
  export function getSupportedFeatures(): string;
  export function allocateString(str: string): number;
  export function deallocateString(ptr: number): void;
  export function wasmInit(): void;

  // Memory
  export const memory: WebAssembly.Memory;
}

declare module '../build/knirv-controller.wasm' {
  const wasmModule: WebAssembly.Module;
  export default wasmModule;
}

// Legacy compatibility for existing Rust WASM imports
declare module '../wasm-pkg/knirv_cortex_wasm' {
  export class HRMCognitive {
    constructor();
    initialize_modules(l_count: number, h_count: number): void;
    process_cognitive_input(input_json: string): string;
    get_model_info(): string;
    load_weights(weights_data: Uint8Array): boolean;
    free(): void;
  }

  export function main(): void;
}

declare module '../wasm-pkg/knirv_cortex_wasm_bg.wasm' {
  const wasmModule: WebAssembly.Module;
  export default wasmModule;
}
