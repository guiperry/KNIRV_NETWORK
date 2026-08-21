import type * as WasmCore from "./wasm/knirv_sdk_core.js";
import { type WasmModuleMetadata, type WasmModuleSource } from "./modules.js";
export declare const BINDING_API_VERSION: 1;
export type BindingErrorCode = "INVALID_ARGUMENT" | "AUTHENTICATION" | "TIMEOUT" | "TRANSPORT" | "API" | "CRYPTO" | "UNSUPPORTED" | "INTERNAL_PANIC";
export interface BindingError {
    code: BindingErrorCode;
    message: string;
    http_status?: number;
    retryable: boolean;
}
export interface Envelope<T = unknown> {
    version: number;
    operation: string;
    payload?: T;
    error?: BindingError;
}
export interface BindingTransport {
    call(request: Envelope): Promise<Envelope>;
}
export declare class BindingOperationError extends Error {
    readonly bindingError: BindingError;
    constructor(bindingError: BindingError);
}
type Core = typeof WasmCore;
export declare class WasmBindingTransport implements BindingTransport {
    private readonly core;
    constructor(core: Core);
    call(request: Envelope): Promise<Envelope>;
}
export declare class SdkClient {
    private readonly transport;
    private readonly core?;
    protected constructor(transport: BindingTransport, core?: Core | undefined);
    /** Create a client around a native, remote, or application-provided transport. */
    static create(transport: BindingTransport): Promise<SdkClient>;
    /** @internal Creates the default browser/edge client after WASM initialization. */
    static fromWasm(core: Core): SdkClient;
    call<T = unknown>(operation: string, payload?: unknown): Promise<Envelope<T>>;
    sha256(data: string): Promise<Envelope<{
        digest: string;
    }>>;
    wasmManifest(): Promise<Envelope>;
    /** Returns verified, caller-owned module bytes. Module assets stay separate
     * from the SDK runtime WASM binary to preserve browser and edge size limits. */
    moduleBytes(id: string, source?: WasmModuleSource): Promise<{
        metadata: WasmModuleMetadata;
        bytes: Uint8Array;
    }>;
    sha256Hex(data: string | Uint8Array): Promise<string>;
    verifySha256(data: string | Uint8Array, expectedHex: string): Promise<boolean>;
    coreVersion(): string | undefined;
}
export {};
