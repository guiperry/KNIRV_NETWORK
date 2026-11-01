import * as core from "./wasm/knirv_sdk_core.js";
import { initLocal } from "./local.js";
import { SdkClient as Client, WasmBindingTransport } from "./runtime.js";
export * from "./runtime.js";
export * from "./modules.js";
export * from "./actuarial.js";
/** Initializes the SDK from a caller source, or the bundled inline fallback. */
export async function init(wasmSource) {
    await initLocal(wasmSource);
}
export class SdkClient extends Client {
    static async init(wasmSource, options = {}) {
        await init(wasmSource);
        return new SdkClient(new WasmBindingTransport(core, options.http, options.config), core);
    }
}
