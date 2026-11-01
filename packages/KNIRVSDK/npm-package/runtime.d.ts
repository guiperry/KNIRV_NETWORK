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
/** Client settings for the network operations; carried on each request. */
export interface BindingConfig {
    network?: string;
    api_key?: string;
    timeout_ms?: number;
    transaction_url?: string;
    gateway_url?: string;
    controller_url?: string;
    backend_url?: string;
    headers?: Record<string, string>;
}
/**
 * The one network seam. The Rust core builds requests, retries, fails over and
 * maps errors; the host only moves bytes. Wire format: core/src/http_transport.rs.
 * Both functions receive and return JSON strings / milliseconds.
 */
export interface HttpHost {
    http_request(requestJson: string): Promise<string>;
    sleep(ms: number): Promise<void>;
}
/** The default host: global `fetch`, with `timeout_ms` enforced by an abort. */
export declare function fetchHost(fetchImpl?: typeof fetch): HttpHost;
export declare class BindingOperationError extends Error {
    readonly bindingError: BindingError;
    constructor(bindingError: BindingError);
}
type Core = typeof WasmCore;
/** Runs envelopes against the WASM core; network operations use `host`. */
export declare class WasmBindingTransport {
    private readonly core;
    private readonly host;
    private readonly config?;
    constructor(core: Core, host?: HttpHost, config?: BindingConfig | undefined);
    call(request: Envelope & {
        config?: BindingConfig;
    }): Promise<Envelope>;
}
export interface SdkClientOptions {
    /** Moves bytes for the network operations. Defaults to global `fetch`. */
    http?: HttpHost;
    /** Default client settings (network, URLs, API key, timeout) for network operations. */
    config?: BindingConfig;
}
export declare class SdkClient {
    private readonly transport;
    private readonly core;
    protected constructor(transport: WasmBindingTransport, core: Core);
    /** @internal Creates the client after WASM initialization. */
    static fromWasm(core: Core, options?: SdkClientOptions): SdkClient;
    /** `config` overrides the client's default settings for a network operation. */
    call<T = unknown>(operation: string, payload?: unknown, config?: BindingConfig): Promise<Envelope<T>>;
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
