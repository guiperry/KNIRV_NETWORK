import { type Envelope } from "./runtime.js";
export type LocalSource = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;
/** Thrown when a local operation runs before the core could be instantiated. */
export declare class LocalCoreNotInitializedError extends Error {
    constructor(cause?: unknown);
}
/** Instantiates the core. Idempotent; safe to call from several modules. */
export declare function initLocal(source?: LocalSource): Promise<void>;
/** JSON that carries bigints as exact integer literals. */
export declare function toJson(value: unknown): string;
/** Calls one local operation synchronously and returns its payload. */
export declare function callLocal<T>(operation: string, payload?: unknown): T;
/** Raw envelope call, for transports that forward arbitrary requests. */
export declare function callLocalEnvelope(request: Envelope): Envelope;
export type Uint64 = number | string | bigint;
export declare function toBase64(value: Uint8Array): string;
export declare const fromBase64: (value: string) => Uint8Array;
/** The core's `Option<Vec<u8>>` fields take a JSON array of numbers. */
export declare const byteArray: (value?: Uint8Array) => number[] | undefined;
/** u64 values come back as JSON numbers; the public types use bigint. */
export declare const big: (value: number | string | bigint | null | undefined) => bigint;
