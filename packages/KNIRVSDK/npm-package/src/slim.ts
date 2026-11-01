import * as core from "./wasm/knirv_sdk_core.js";
import { initLocal } from "./local.js";
import { SdkClient as Client, WasmBindingTransport, type SdkClientOptions } from "./runtime.js";

export * from "./runtime.js";
export * from "./modules.js";

/** Explicit-source build for Workers, CDNs, and bundler-managed assets. */
export async function init(wasmSource: InitInput): Promise<void> {
  if (!wasmSource) throw new TypeError("@knirv/sdk/slim requires a WASM URL, Response, ArrayBuffer, Uint8Array, or WebAssembly.Module");
  await initLocal(wasmSource);
}

export class SdkClient extends Client {
  static async init(wasmSource: InitInput, options: SdkClientOptions = {}): Promise<SdkClient> {
    await init(wasmSource);
    return new SdkClient(new WasmBindingTransport(core, options.http, options.config), core);
  }
}

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;
