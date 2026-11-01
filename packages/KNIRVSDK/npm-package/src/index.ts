import * as core from "./wasm/knirv_sdk_core.js";
import { initLocal } from "./local.js";
import { SdkClient as Client, WasmBindingTransport, type SdkClientOptions } from "./runtime.js";

export * from "./runtime.js";
export * from "./modules.js";
export * from "./actuarial.js";

/** Initializes the SDK from a caller source, or the bundled inline fallback. */
export async function init(wasmSource?: InitInput): Promise<void> {
  await initLocal(wasmSource);
}

export class SdkClient extends Client {
  static async init(wasmSource?: InitInput, options: SdkClientOptions = {}): Promise<SdkClient> {
    await init(wasmSource);
    return new SdkClient(new WasmBindingTransport(core, options.http, options.config), core);
  }
}

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;
