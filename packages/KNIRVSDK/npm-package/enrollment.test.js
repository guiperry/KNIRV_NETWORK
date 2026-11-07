/**
 * Boot-node enrollment (DEC-2 / 2d.2) vector tests for the thin TS binding.
 *
 * These exercise the binding's delegation to the Rust core op
 * `signing.verify_bootnode_enrollment`, asserting the operator-to-key binding,
 * the claims-survive base64 contract (item 5), and replay prevention through a
 * host NonceStore. They run against a rebuilt WASM SDK; if the op is unavailable
 * they skip rather than fail.
 */
import { signBootnodeEnrollChallenge, signBootnodeEnrollApproval, verifyBootnodeEnrollment } from './signing.js';
import { MemoryNonceStore } from './nonceStore.js';
import { fromBase64 } from './local.js';
function hexToBytes(s) {
    const clean = s.replace(/^0x/i, '');
    const out = new Uint8Array(clean.length / 2);
    for (let i = 0; i < out.length; i++)
        out[i] = parseInt(clean.substr(i * 2, 2), 16);
    return out;
}
function claimsFromEnvelope(envelope) {
    const bytes = fromBase64(envelope);
    const payload = findProtoField(bytes, 8);
    const claim = JSON.parse(new TextDecoder().decode(payload));
    return Uint8Array.from(claim.claims);
}
// findProtoField returns the length-delimited bytes of the first wire-type-2 field
// with the given field number (mirrors the Go test helper).
function findProtoField(b, field) {
    const tag = (field << 3) | 2;
    let i = 0;
    while (i < b.length) {
        if (b[i] !== tag) {
            i++;
            continue;
        }
        i++;
        let length = 0;
        let shift = 0;
        let byte = 0;
        do {
            byte = b[i++];
            length |= (byte & 0x7f) << shift;
            shift += 7;
        } while (byte & 0x80 && i < b.length);
        if (i + length > b.length)
            return new Uint8Array();
        return b.slice(i, i + length);
    }
    return new Uint8Array();
}
function toHex(bytes) {
    let s = "";
    for (const b of bytes)
        s += b.toString(16).padStart(2, "0");
    return s;
}
export async function runBootnodeEnrollmentVectors() {
    const bootKey = hexToBytes('1a2b3c4d5e6f7081928374655647382910fedcba0987654321fedcba09876543');
    const rootKey = hexToBytes('ffffeeeeddddccccbbbbaaaa9999888877776666555544443333222211110000');
    const chainId = 'knirv-testnet-1';
    const operator = 'operator:alice';
    const nonce = 'enroll:vec-1';
    const claims = new Uint8Array([0x62, 0x6f, 0x6f, 0x74]);
    const now = new Date(1_700_000_000 * 1000);
    let challenge;
    try {
        challenge = await signBootnodeEnrollChallenge(bootKey, chainId, nonce, 1_700_000_000, 1_700_000_600, operator, claims);
    }
    catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        if (/unsupported operation|not initialised|LocalCoreNotInitialized/i.test(msg)) {
            return { passed: false, skipped: true, reason: `enrollment ops not in released WASM artifact yet (wasm re-release deferred): ${msg}` };
        }
        throw e;
    }
    // Item 5: the claims must survive the TS-base64 → Rust-base64-decode contract identically.
    const got = claimsFromEnvelope(challenge.envelope);
    if (got.length !== claims.length || got.some((b, i) => b !== claims[i])) {
        throw new Error(`claims did not survive the binding base64 contract: got ${toHex(got)} want ${toHex(claims)}`);
    }
    const approval = await signBootnodeEnrollApproval(rootKey, chainId, nonce, 1_700_000_000, 1_700_000_600, challenge.envelope);
    const expectedRootPk = approval.public_key;
    const enrollment = { challenge, approval };
    const store = new MemoryNonceStore();
    const ok = await verifyBootnodeEnrollment(enrollment, expectedRootPk, operator, store, now);
    if (!ok)
        throw new Error('expected valid enrollment');
    // Replay: the same nonce is now consumed, so a second verification is rejected.
    if (await verifyBootnodeEnrollment(enrollment, expectedRootPk, operator, store, now)) {
        throw new Error('replay of consumed nonce must be rejected');
    }
    // Operator substitution must be rejected.
    if (await verifyBootnodeEnrollment(enrollment, expectedRootPk, 'operator:bob', new MemoryNonceStore(), now)) {
        throw new Error('operator substitution must be rejected');
    }
    return { passed: true, skipped: false };
}
