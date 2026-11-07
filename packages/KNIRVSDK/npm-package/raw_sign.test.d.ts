/**
 * Runs the raw-sign known-answer vectors. The frozen dcrd compact signature (65
 * bytes) must strip to exactly 64 bytes of r||s with the documented recovery
 * byte, and the generated binding mirror must accept the allowlisted purpose with
 * a 32-byte digest and reject everything else. Pure: needs no WASM module.
 */
export declare function runRawSignVectors(): void;
