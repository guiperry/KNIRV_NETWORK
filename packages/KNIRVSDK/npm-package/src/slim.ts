import initWasm, * as core from "./wasm/knirv_sdk_core.js";
import { SdkClient as Client, WasmBindingTransport } from "./runtime.js";

export * from "./runtime.js";
export * from "./modules.js";

let initialization: Promise<unknown> | undefined;

/** Explicit-source build for Workers, CDNs, and bundler-managed assets. */
export async function init(wasmSource: InitInput): Promise<void> {
  if (!wasmSource) throw new TypeError("@knirv/sdk/slim requires a WASM URL, Response, ArrayBuffer, Uint8Array, or WebAssembly.Module");
  if (!initialization) initialization = initWasm({ module_or_path: wasmSource });
  await initialization;
}

export class SdkClient extends Client {
  static async init(wasmSource: InitInput): Promise<SdkClient> {
    await init(wasmSource);
    return new SdkClient(new WasmBindingTransport(core), core);
  }
}

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;
