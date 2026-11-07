import { RAW_SIGN_KAT, validateRawSign } from './signing.js';
function hexToBytes(s) {
    const clean = s.replace(/^0x/i, '');
    const out = new Uint8Array(clean.length / 2);
    for (let i = 0; i < out.length; i++)
        out[i] = parseInt(clean.substr(i * 2, 2), 16);
    return out;
}
function fail(message) {
    throw new Error(message);
}
/**
 * Runs the raw-sign known-answer vectors. The frozen dcrd compact signature (65
 * bytes) must strip to exactly 64 bytes of r||s with the documented recovery
 * byte, and the generated binding mirror must accept the allowlisted purpose with
 * a 32-byte digest and reject everything else. Pure: needs no WASM module.
 */
export function runRawSignVectors() {
    const compact = hexToBytes(RAW_SIGN_KAT.compactHex);
    if (compact.length !== 65)
        fail(`compact length ${compact.length}, want 65`);
    if (compact[0] !== RAW_SIGN_KAT.recoveryByte)
        fail(`recovery byte ${compact[0]}, want ${RAW_SIGN_KAT.recoveryByte}`);
    const sig64 = hexToBytes(RAW_SIGN_KAT.sig64Hex);
    if (sig64.length !== 64)
        fail(`sig64 length ${sig64.length}, want 64`);
    for (let i = 0; i < 64; i++)
        if (compact[i + 1] !== sig64[i])
            fail('compact[1:] != sig64');
    validateRawSign({ purpose: 'failover-identity', digestHex: RAW_SIGN_KAT.digestHex });
    for (const bad of ['', 'root', 'boot', 'checkpoint-signer', 'failover', 'anything']) {
        try {
            validateRawSign({ purpose: bad, digestHex: RAW_SIGN_KAT.digestHex });
            fail(`purpose ${bad} must be rejected`);
        }
        catch {
            /* expected */
        }
    }
    for (const bad of ['zz', 'aabbcc', 'g g']) {
        try {
            validateRawSign({ purpose: 'failover-identity', digestHex: bad });
            fail(`digest ${bad} must be rejected`);
        }
        catch {
            /* expected */
        }
    }
}
runRawSignVectors();
