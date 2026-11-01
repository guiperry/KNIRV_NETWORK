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
import { BindingOperationError, BINDING_API_VERSION, type Envelope } from "./runtime.js";

export type LocalSource = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;

/** Thrown when a local operation runs before the core could be instantiated. */
export class LocalCoreNotInitializedError extends Error {
  constructor(cause?: unknown) {
    super(
      "the KNIRV core is not initialised and cannot be compiled synchronously here. " +
        "Call `await initLocal()` once at start-up (browser main thread), or " +
        "`await initLocal(wasmModule)` with a pre-compiled WebAssembly.Module (Cloudflare Workers)." +
        (cause instanceof Error ? ` Underlying error: ${cause.message}` : ""),
    );
    this.name = "LocalCoreNotInitializedError";
  }
}

let ready = false;
let pending: Promise<void> | undefined;

/** Instantiates the core. Idempotent; safe to call from several modules. */
export async function initLocal(source?: LocalSource): Promise<void> {
  if (ready) return;
  pending ??= initWasm({ module_or_path: source ?? inlinedWasmBytes }).then(() => {
    ready = true;
  });
  await pending;
}

function ensureReady(): void {
  if (ready) return;
  try {
    initSync({ module: inlinedWasmBytes });
    ready = true;
  } catch (cause) {
    throw new LocalCoreNotInitializedError(cause);
  }
}

// JSON cannot carry a bigint, and a u64 above 2^53 must not pass through a JS
// number. Tag bigints with a per-process marker and strip the quotes after
// stringifying so they reach the core as exact integer literals.
const MARK = `__u64_${Math.random().toString(36).slice(2)}__`;
const MARKED = new RegExp(`"${MARK}(\\d+)"`, "g");

/** JSON that carries bigints as exact integer literals. */
export function toJson(value: unknown): string {
  return JSON.stringify(value, (_key, v) => (typeof v === "bigint" ? `${MARK}${v}` : v)).replace(MARKED, "$1");
}

/** Calls one local operation synchronously and returns its payload. */
export function callLocal<T>(operation: string, payload: unknown = {}): T {
  ensureReady();
  const response = JSON.parse(invoke(toJson({ version: BINDING_API_VERSION, operation, payload }))) as Envelope<T>;
  if (response.error) throw new BindingOperationError(response.error);
  return response.payload as T;
}

/** Raw envelope call, for transports that forward arbitrary requests. */
export function callLocalEnvelope(request: Envelope): Envelope {
  ensureReady();
  return JSON.parse(invoke(toJson(request))) as Envelope;
}

// ---- shared encoding helpers ----

export type Uint64 = number | string | bigint;

export function toBase64(value: Uint8Array): string {
  let binary = "";
  for (let start = 0; start < value.length; start += 0x8000) {
    binary += String.fromCharCode(...value.subarray(start, start + 0x8000));
  }
  return btoa(binary);
}

export const fromBase64 = (value: string): Uint8Array => Uint8Array.from(atob(value), (c) => c.charCodeAt(0));

/** The core's `Option<Vec<u8>>` fields take a JSON array of numbers. */
export const byteArray = (value?: Uint8Array): number[] | undefined => (value === undefined ? undefined : Array.from(value));

/** u64 values come back as JSON numbers; the public types use bigint. */
export const big = (value: number | string | bigint | null | undefined): bigint => BigInt(value ?? 0);
