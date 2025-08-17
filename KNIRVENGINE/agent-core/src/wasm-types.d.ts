// TypeScript declarations for KNIRV-CORTEX WASM module

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
