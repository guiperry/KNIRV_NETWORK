import initWasm, * as core from "./wasm/knirv_sdk_core.js";
import { inlinedWasmBytes } from "./wasm/inlined_base64.js";
import { SdkClient as Client, WasmBindingTransport } from "./runtime.js";

export * from "./runtime.js";
export * from "./modules.js";

let initialization: Promise<unknown> | undefined;

/** Initializes the SDK from a caller source, or the bundled inline fallback. */
export async function init(wasmSource?: InitInput): Promise<void> {
  if (!initialization) initialization = initWasm({ module_or_path: wasmSource ?? inlinedWasmBytes });
  await initialization;
}

export class SdkClient extends Client {
  static async init(wasmSource?: InitInput): Promise<SdkClient> {
    await init(wasmSource);
    return new SdkClient(new WasmBindingTransport(core), core);
  }
}

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;
