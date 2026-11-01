/* tslint:disable */
/* eslint-disable */

export function core_version(): string;

/**
 * Runs one binding request (`{version, operation, payload}` as JSON) against
 * the Rust core's local operations and returns the response envelope JSON.
 * Unknown or network operations answer `UNSUPPORTED`.
 */
export function invoke(request_json: string): string;

/**
 * Like [`invoke`], and network operations run through `host`
 * (`{http_request(json): Promise<string>, sleep(ms): Promise<void>}`).
 */
export function invoke_async(request_json: string, host: any): Promise<string>;

/**
 * Hash arbitrary caller-owned bytes with the SDK's canonical SHA-256 helper.
 */
export function sha256_hex(data: Uint8Array): string;

/**
 * Fast integrity primitive for module assets fetched by a browser or edge host.
 */
export function sha256_matches(data: Uint8Array, expected_hex: string): boolean;

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;

export interface InitOutput {
    readonly memory: WebAssembly.Memory;
    readonly core_version: (a: number) => void;
    readonly invoke: (a: number, b: number, c: number) => void;
    readonly invoke_async: (a: number, b: number, c: number) => number;
    readonly sha256_hex: (a: number, b: number, c: number) => void;
    readonly sha256_matches: (a: number, b: number, c: number, d: number) => number;
    readonly __wasm_bindgen_func_elem_3345: (a: number, b: number, c: number, d: number) => void;
    readonly __wasm_bindgen_func_elem_3354: (a: number, b: number, c: number, d: number) => void;
    readonly __wbindgen_export: (a: number, b: number) => number;
    readonly __wbindgen_export2: (a: number, b: number, c: number, d: number) => number;
    readonly __wbindgen_export3: (a: number) => void;
    readonly __wbindgen_export4: (a: number, b: number) => void;
    readonly __wbindgen_add_to_stack_pointer: (a: number) => number;
    readonly __wbindgen_export5: (a: number, b: number, c: number) => void;
}

export type SyncInitInput = BufferSource | WebAssembly.Module;

/**
 * Instantiates the given `module`, which can either be bytes or
 * a precompiled `WebAssembly.Module`.
 *
 * @param {{ module: SyncInitInput }} module - Passing `SyncInitInput` directly is deprecated.
 *
 * @returns {InitOutput}
 */
export function initSync(module: { module: SyncInitInput } | SyncInitInput): InitOutput;

/**
 * If `module_or_path` is {RequestInfo} or {URL}, makes a request and
 * for everything else, calls `WebAssembly.instantiate` directly.
 *
 * @param {{ module_or_path: InitInput | Promise<InitInput> }} module_or_path - Passing `InitInput` directly is deprecated.
 *
 * @returns {Promise<InitOutput>}
 */
export default function __wbg_init (module_or_path?: { module_or_path: InitInput | Promise<InitInput> } | InitInput | Promise<InitInput>): Promise<InitOutput>;
