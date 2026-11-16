/**
 * ML-DSA-65 (FIPS 204) verification, key derivation and signing.
 *
 * Typed wrappers over the Rust core's `mldsa.*` operations (core/src/mldsa.rs,
 * the source of truth), reached through the binding envelope in `./local.ts`.
 * Nothing cryptographic is implemented in TypeScript.
 */
import { callLocal, toBase64, fromBase64 } from "./local.js";
/** FIPS 204 Table 2 sizes for ML-DSA-65. */
export const ML_DSA65_PUBLIC_KEY_LENGTH = 1952;
export const ML_DSA65_SIGNATURE_LENGTH = 3309;
export const ML_DSA65_SEED_LENGTH = 32;
export const ML_DSA65_MAX_CONTEXT_LENGTH = 255;
/**
 * Verifies an ML-DSA-65 signature. Argument order matches the Rust core and the
 * Go binding: (publicKey, message, context, signature). `context` defaults to the
 * empty string (pass `undefined` to take the default).
 */
export function mlDsa65Verify(publicKey, message, context = new Uint8Array(), signature) {
    return callLocal("mldsa.ml_dsa65_verify", {
        public_key_base64: toBase64(publicKey),
        message_base64: toBase64(message),
        context_base64: toBase64(context),
        signature_base64: toBase64(signature),
    });
}
/** Derives the 1952-byte public key from a 32-byte seed (FIPS 204 KeyGen). */
export function mlDsa65PublicKey(seed) {
    if (seed.length !== ML_DSA65_SEED_LENGTH) {
        throw new RangeError(`ML-DSA-65 seed must be ${ML_DSA65_SEED_LENGTH} bytes`);
    }
    const out = callLocal("mldsa.ml_dsa65_public_key", {
        seed_base64: toBase64(seed),
    });
    return fromBase64(out.public_key_base64);
}
/** Signs with the deterministic FIPS 204 variant. `context` defaults to empty. */
export function mlDsa65Sign(seed, message, context = new Uint8Array()) {
    if (seed.length !== ML_DSA65_SEED_LENGTH) {
        throw new RangeError(`ML-DSA-65 seed must be ${ML_DSA65_SEED_LENGTH} bytes`);
    }
    const out = callLocal("mldsa.ml_dsa65_sign", {
        seed_base64: toBase64(seed),
        message_base64: toBase64(message),
        context_base64: toBase64(context),
    });
    return fromBase64(out.signature_base64);
}
