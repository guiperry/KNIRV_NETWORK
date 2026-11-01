import * as core from "./wasm/knirv_sdk_core.js";
import { initLocal } from "./local.js";
import { SdkClient as Client, WasmBindingTransport } from "./runtime.js";
export * from "./runtime.js";
export * from "./modules.js";
/** Explicit-source build for Workers, CDNs, and bundler-managed assets. */
export async function init(wasmSource) {
    if (!wasmSource)
        throw new TypeError("@knirv/sdk/slim requires a WASM URL, Response, ArrayBuffer, Uint8Array, or WebAssembly.Module");
    await initLocal(wasmSource);
}
export class SdkClient extends Client {
    static async init(wasmSource, options = {}) {
        await init(wasmSource);
        return new SdkClient(new WasmBindingTransport(core, options.http, options.config), core);
    }
}
