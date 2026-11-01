import { SdkClient as Client, type SdkClientOptions } from "./runtime.js";
export * from "./runtime.js";
export * from "./modules.js";
export * from "./actuarial.js";
/** Initializes the SDK from a caller source, or the bundled inline fallback. */
export declare function init(wasmSource?: InitInput): Promise<void>;
export declare class SdkClient extends Client {
    static init(wasmSource?: InitInput, options?: SdkClientOptions): Promise<SdkClient>;
}
export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;
