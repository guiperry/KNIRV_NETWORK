export interface WasmModuleMetadata {
    id: string;
    artifact_version: string;
    module_kind: string;
    abi_version: number;
    byte_length: number;
    sha256: string;
    capabilities: readonly string[];
}
export interface WasmModuleManifest {
    version: number;
    modules: readonly WasmModuleMetadata[];
}
export type WasmModuleSource = RequestInfo | URL | Response | ArrayBuffer | Uint8Array;
export declare const wasmModulesManifest: WasmModuleManifest;
export declare function wasmModuleMetadata(id: string): WasmModuleMetadata;
/** Where versioned SDK artifacts are published. */
export declare const WASM_RELEASE_BASE_URL = "https://releases.knirv.com/sdk";
/**
 * Default location of a module: next to this package when it is bundled,
 * otherwise the published release for this SDK version (no `import.meta.url`
 * asset path, which bundlers and Workers handle inconsistently).
 */
export declare function wasmModuleURL(id: string): URL;
/** Loads a raw module asset. Integrity verification is performed by SdkClient. */
export declare function fetchWasmModule(id: string, source?: WasmModuleSource): Promise<Uint8Array>;
