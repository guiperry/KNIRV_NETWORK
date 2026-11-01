// Binding over the Rust core's replication module. The head format, its
// canonical hash, the chain rules and pull authentication live in
// core/src/replication.rs; this file only carries types and op calls.
import { callLocal } from '../local.js';
/** Canonical bytes are decided by the core; returned base64-encoded. */
export function headPayload(head) {
    const out = callLocal('replication.head_payload', { head });
    return Uint8Array.from(atob(out.payload_base64), (c) => c.charCodeAt(0));
}
export function headHash(head) {
    return callLocal('replication.head_hash', { head });
}
export function validateHead(head) {
    const out = callLocal('replication.head_validate', { head });
    return out.valid ? true : out;
}
export function compareHeads(local, next) {
    return callLocal('replication.compare', { local, next });
}
export function verifyHead(signed, trustAnchor = '', nowUnix = Math.floor(Date.now() / 1000)) {
    return callLocal('replication.verify_head', {
        signed: signed.signed,
        trust_anchor: trustAnchor,
        now_unix: nowUnix,
    });
}
export function pullPayload(request) {
    const out = callLocal('replication.pull_payload', { request });
    return Uint8Array.from(atob(out.payload_base64), (c) => c.charCodeAt(0));
}
export function verifyPull(signed, allowed, nowUnix = Math.floor(Date.now() / 1000), skewSeconds = 120) {
    return callLocal('replication.verify_pull', {
        signed,
        allowed,
        now_unix: nowUnix,
        skew_seconds: skewSeconds,
    });
}
