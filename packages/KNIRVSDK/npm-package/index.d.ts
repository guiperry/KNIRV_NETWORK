import { SdkClient as Client } from "./runtime.js";
export * from "./runtime.js";
export * from "./modules.js";
/** Initializes the SDK from a caller source, or the bundled inline fallback. */
export declare function init(wasmSource?: InitInput): Promise<void>;
export declare class SdkClient extends Client {
    static init(wasmSource?: InitInput): Promise<SdkClient>;
}
export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;
