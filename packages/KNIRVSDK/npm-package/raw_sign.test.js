import { validateRawSign } from './signing.js';
/** Independent known-answer vector (core/src/secrets.rs raw_sign_kat; test data only). Frozen once:
 * private key 1 → the secp256k1 generator point G. The OpenBao plugin signs with
 * dcrd and must reproduce `sig64Hex` for `digestHex` exactly, proving the 64-byte
 * r||s compact format is byte-identical to legacy dcrd SignCompact(..., true)[1:]. */
const RAW_SIGN_KAT = {
    privHex: '0000000000000000000000000000000000000000000000000000000000000001',
    digestHex: '27a63df12ab9b1c25967ac3a371c19b65d6f326fcd0748b2b3f657bfd0e64e22',
    compactHex: '20031002a1c9f5e00ac84199972dc2c02dec4580d703d9f572d0fcbd960d33c53e10055df6e29204b5c780de6f0cf6a3ed9e4f017150c04feab3423f43cf2074e4',
    sig64Hex: '031002a1c9f5e00ac84199972dc2c02dec4580d703d9f572d0fcbd960d33c53e10055df6e29204b5c780de6f0cf6a3ed9e4f017150c04feab3423f43cf2074e4',
    pubHex: '0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798',
    recoveryByte: 0x20,
};
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
        let accepted = false;
        try {
            validateRawSign({ purpose: bad, digestHex: RAW_SIGN_KAT.digestHex });
            accepted = true;
        }
        catch {
            /* expected: the core rejects it */
        }
        if (accepted)
            fail(`purpose ${bad} must be rejected`);
    }
    for (const bad of ['zz', 'aabbcc', 'g g']) {
        let accepted = false;
        try {
            validateRawSign({ purpose: 'failover-identity', digestHex: bad });
            accepted = true;
        }
        catch {
            /* expected: the core rejects it */
        }
        if (accepted)
            fail(`digest ${bad} must be rejected`);
    }
}
runRawSignVectors();
