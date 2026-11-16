/** FIPS 204 Table 2 sizes for ML-DSA-65. */
export declare const ML_DSA65_PUBLIC_KEY_LENGTH = 1952;
export declare const ML_DSA65_SIGNATURE_LENGTH = 3309;
export declare const ML_DSA65_SEED_LENGTH = 32;
export declare const ML_DSA65_MAX_CONTEXT_LENGTH = 255;
/**
 * The core's verdict. `valid` is false with a stable `code` (for example
 * `public_key_length`, `signature_length`, `context_length`, `signature_invalid`)
 * when the input was refused.
 */
export interface MlDsaVerdict {
    valid: boolean;
    code?: string;
    message?: string;
}
/**
 * Verifies an ML-DSA-65 signature. Argument order matches the Rust core and the
 * Go binding: (publicKey, message, context, signature). `context` defaults to the
 * empty string (pass `undefined` to take the default).
 */
export declare function mlDsa65Verify(publicKey: Uint8Array, message: Uint8Array, context: Uint8Array | undefined, signature: Uint8Array): MlDsaVerdict;
/** Derives the 1952-byte public key from a 32-byte seed (FIPS 204 KeyGen). */
export declare function mlDsa65PublicKey(seed: Uint8Array): Uint8Array;
/** Signs with the deterministic FIPS 204 variant. `context` defaults to empty. */
export declare function mlDsa65Sign(seed: Uint8Array, message: Uint8Array, context?: Uint8Array): Uint8Array;
