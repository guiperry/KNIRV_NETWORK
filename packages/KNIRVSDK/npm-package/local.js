/**
 * Loader and call path for the Rust core's local operations (signing, addresses,
 * envelopes, relay and WASM payloads, URIs, authority records).
 *
 * The Rust core is the source of truth: `@knirv/sdk/signing` and
 * `@knirv/sdk/uri` are typed wrappers that call it through the binding
 * envelope. Nothing cryptographic is implemented in TypeScript.
 *
 * Initialisation. The core is a WebAssembly module that must be instantiated
 * once before any call:
 *
 *   - Node, web workers and most edge runtimes: nothing to do. The first call
 *     compiles the bundled module synchronously.
 *   - Browser main thread: call `await initLocal()` once at start-up. Browsers
 *     refuse to compile a module of this size synchronously on the main thread.
 *   - Cloudflare Workers: `WebAssembly.compile` from bytes is forbidden, so pass
 *     a pre-compiled module: `import wasm from '@knirv/sdk/wasm/knirv_sdk_core_bg.wasm'`
 *     then `await initLocal(wasm)`.
 */
import initWasm, { initSync, invoke } from "./wasm/knirv_sdk_core.js";
import { inlinedWasmBytes } from "./wasm/inlined_base64.js";
import { BindingOperationError, BINDING_API_VERSION } from "./runtime.js";
/** Thrown when a local operation runs before the core could be instantiated. */
export class LocalCoreNotInitializedError extends Error {
    constructor(cause) {
        super("the KNIRV core is not initialised and cannot be compiled synchronously here. " +
            "Call `await initLocal()` once at start-up (browser main thread), or " +
            "`await initLocal(wasmModule)` with a pre-compiled WebAssembly.Module (Cloudflare Workers)." +
            (cause instanceof Error ? ` Underlying error: ${cause.message}` : ""));
        this.name = "LocalCoreNotInitializedError";
    }
}
let ready = false;
let pending;
/** Instantiates the core. Idempotent; safe to call from several modules. */
export async function initLocal(source) {
    if (ready)
        return;
    pending ??= initWasm({ module_or_path: source ?? inlinedWasmBytes }).then(() => {
        ready = true;
    });
    await pending;
}
function ensureReady() {
    if (ready)
        return;
    try {
        initSync({ module: inlinedWasmBytes });
        ready = true;
    }
    catch (cause) {
        throw new LocalCoreNotInitializedError(cause);
    }
}
// JSON cannot carry a bigint, and a u64 above 2^53 must not pass through a JS
// number. Tag bigints with a per-process marker and strip the quotes after
// stringifying so they reach the core as exact integer literals.
const MARK = `__u64_${Math.random().toString(36).slice(2)}__`;
const MARKED = new RegExp(`"${MARK}(\\d+)"`, "g");
/** JSON that carries bigints as exact integer literals. */
export function toJson(value) {
    return JSON.stringify(value, (_key, v) => (typeof v === "bigint" ? `${MARK}${v}` : v)).replace(MARKED, "$1");
}
/** Calls one local operation synchronously and returns its payload. */
export function callLocal(operation, payload = {}) {
    ensureReady();
    const response = JSON.parse(invoke(toJson({ version: BINDING_API_VERSION, operation, payload })));
    if (response.error)
        throw new BindingOperationError(response.error);
    return response.payload;
}
/** Raw envelope call, for transports that forward arbitrary requests. */
export function callLocalEnvelope(request) {
    ensureReady();
    return JSON.parse(invoke(toJson(request)));
}
export function toBase64(value) {
    let binary = "";
    for (let start = 0; start < value.length; start += 0x8000) {
        binary += String.fromCharCode(...value.subarray(start, start + 0x8000));
    }
    return btoa(binary);
}
export const fromBase64 = (value) => Uint8Array.from(atob(value), (c) => c.charCodeAt(0));
/** The core's `Option<Vec<u8>>` fields take a JSON array of numbers. */
export const byteArray = (value) => (value === undefined ? undefined : Array.from(value));
/** u64 values come back as JSON numbers; the public types use bigint. */
export const big = (value) => BigInt(value ?? 0);
