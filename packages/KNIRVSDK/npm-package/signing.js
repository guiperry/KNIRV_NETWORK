/**
 * KNIRV direct-transaction and signed-message primitives.
 *
 * Every function here is a typed wrapper over the Rust core (the source of
 * truth), reached through the binding envelope in `./local.ts`. Nothing
 * cryptographic is implemented in TypeScript. See `./local.ts` for the
 * one-time initialisation contract (`initLocal`), which differs by platform.
 */
import { big, byteArray, callLocal, fromBase64, toBase64 } from './local.js';
export { initLocal, LocalCoreNotInitializedError } from './local.js';
export const KNIRV_HD_PATH = "m/44'/118'/0'/0/i";
export const ACTION_SCHEMA_VERSION = 'knirv.action.v1';
export const MESSAGE_SCHEMA_VERSION = 'knirv.message.v1';
export const ACTION_TYPE_URL = '/knirv.signing.v1.Action';
export const SECP256K1_TYPE_URL = '/cosmos.crypto.secp256k1.PubKey';
export const SIGN_MODE_DIRECT = 1;
export const RELAY_ENVELOPE_SCHEMA_VERSION = 'knirv.controller.relay-envelope.v1';
export const WASM_PUBLICATION_SCHEMA_VERSION = 'knirv.wasm_publication.v1';
export const WASM_MANIFEST_SCHEMA_VERSION = 'knirv.wasm_manifest.v1';
export const CONTROLLER_DOMAIN = 'knirv.controller';
export const SCENES_DOMAIN = 'knirv.scenes';
export const PURPOSE_SCENE_MANIFEST = 'scene-manifest';
export const ROLE_SCENE_SIGNER = 'scene-signer';
export const PURPOSE_WASM_PUBLICATION = 'wasm-publication';
export const PURPOSE_WASM_ASSIGNMENT = 'wasm-assignment';
export const PURPOSE_WASM_DOWNLOAD_GRANT = 'wasm-download-grant';
export const PURPOSE_RELAY_REQUEST = 'relay-request';
export const PURPOSE_RELAY_RESPONSE = 'relay-response';
export const RELAY_TARGET_DVE_EXPERT_ADVISOR = 'dve_expert_advisor';
export const RELAY_TARGET_CLI_SUPERVISOR = 'cli_supervisor';
// Registry / bond escrow domain and purposes (root tx-chain design rev 2).
export const REGISTRY_DOMAIN = 'knirv.registry';
export const PURPOSE_ROOT_TX_CHAIN_RECORD = 'root-tx-chain-record';
export const PURPOSE_BOND_HOLD = 'bond-hold';
export const PURPOSE_BOND_RELEASE = 'bond-release';
// Boot-node enrollment (DEC-2 / 2d.2) domain and purpose.
export const AUTHORITY_DOMAIN = 'knirv.authority';
export const PURPOSE_BOOTNODE_ENROLL = 'bootnode-enroll';
const bytesOf = (reply) => fromBase64(reply.bytes_base64);
const wireAction = (a) => ({
    schema_version: a.schemaVersion, action: a.action, sender: a.sender, recipient: a.recipient, amount: a.amount,
    payload: byteArray(a.payload), timestamp_unix: a.timestampUnix,
});
const wireFee = (f) => (f ? { denom: f.denom, amount: f.amount, gas_limit: f.gasLimit, payer: f.payer, granter: f.granter } : undefined);
/** @internal */
export const wireRequest = (r) => ({ action: wireAction(r.action), chain_id: r.chainId, account_number: r.accountNumber, sequence: r.sequence, fee: wireFee(r.fee) });
/** @internal */
export const wireEnvelope = (e) => ({
    schema_version: e.schemaVersion, domain: e.domain, purpose: e.purpose, chain_id: e.chainId, nonce: e.nonce,
    issued_at_unix: e.issuedAtUnix, expires_at_unix: e.expiresAtUnix, payload: byteArray(e.payload),
});
const unixSeconds = (now) => Math.floor(now.getTime() / 1000);
const wireRelay = (e) => ({
    schema_version: e.schemaVersion, request_id: e.requestId, user_subject: e.userSubject, device_id: e.deviceId, dve_id: e.dveId,
    target_type: e.targetType, target_id: e.targetId, capability: e.capability, sequence: e.sequence, lease_epoch: e.leaseEpoch,
    issued_at_unix: e.issuedAtUnix, expires_at_unix: e.expiresAtUnix, payload_digest: e.payloadDigest,
});
export function marshalAction(action) {
    return bytesOf(callLocal('signing.marshal_action', wireAction(action)));
}
export function buildDirectSignDoc(request, compressedPublicKey) {
    const out = callLocal('signing.build_sign_doc', {
        request: wireRequest(request), public_key_base64: toBase64(compressedPublicKey),
    });
    return { bodyBytes: fromBase64(out.body_base64), authInfoBytes: fromBase64(out.auth_info_base64), signDoc: fromBase64(out.sign_doc_base64) };
}
export const marshalTxRaw = (bodyBytes, authInfoBytes, signatures) => bytesOf(callLocal('signing.marshal_tx_raw', {
    body_base64: toBase64(bodyBytes), auth_info_base64: toBase64(authInfoBytes), signatures_base64: signatures.map(toBase64),
}));
export function publicKeyToKNIRVAddress(compressedPublicKey) {
    return callLocal('address.encode', { public_key_base64: toBase64(compressedPublicKey) }).address;
}
export async function signDirectTransaction(privateKey, request) {
    return callLocal('signing.sign_direct_transaction', { private_key_base64: toBase64(privateKey), request: wireRequest(request) });
}
export function marshalMessageEnvelope(envelope) {
    return bytesOf(callLocal('signing.marshal_message_envelope', wireEnvelope(envelope)));
}
export async function signMessageEnvelope(privateKey, envelope) {
    return callLocal('signing.sign_message_envelope', { private_key_base64: toBase64(privateKey), envelope: wireEnvelope(envelope) });
}
export function parseMessageEnvelope(data) {
    const p = callLocal('signing.parse_message_envelope', { bytes_base64: toBase64(data) });
    return { schemaVersion: p.schema_version, domain: p.domain, purpose: p.purpose, chainId: p.chain_id, nonce: p.nonce, issuedAtUnix: big(p.issued_at_unix), expiresAtUnix: big(p.expires_at_unix), payload: fromBase64(p.payload_base64) };
}
export async function verifyMessage(signed, expectedDomain, expectedPurpose, expectedChainId, expectedNonce, now) {
    callLocal('signing.verify_message', { signed, domain: expectedDomain, purpose: expectedPurpose, chain_id: expectedChainId, nonce: expectedNonce, now_unix: unixSeconds(now) });
}
export async function verifyMessagePayload(signed, expectedDomain, expectedPurpose, expectedChainId, expectedPayload, now) {
    callLocal('signing.verify_message_payload', { signed, domain: expectedDomain, purpose: expectedPurpose, chain_id: expectedChainId, payload_base64: toBase64(expectedPayload), now_unix: unixSeconds(now) });
}
// ---- Root tx-chain registry record signing (CRITICAL: unauthenticated POST /register) ----
const wireRegistryRecord = (r) => ({
    chain_id: r.chainId, tx_chain_address: r.txChainAddress, owner_pub_key: r.ownerPubKey,
    nonce: r.nonce, expires_at_unix: r.expiresAtUnix,
});
const wireBondHold = (h) => ({
    hold_id: h.holdId, operator: h.operator, amount: h.amount,
    root_registry_id: h.rootRegistryId, expires_at_unix: h.expiresAtUnix,
});
const wireBondRelease = (r) => ({
    hold_id: r.holdId, boot_node_chain_id: r.bootNodeChainId, boot_node_address: r.bootNodeAddress,
});
/** Signs a root-tx-chain registry record (secp256k1 via the Rust core). */
export async function signRegistryRecord(privateKey, chainId, record, issuedAtUnix) {
    return callLocal('signing.sign_registry_record', {
        private_key_base64: toBase64(privateKey), chain_id: chainId, record: wireRegistryRecord(record), issued_at_unix: issuedAtUnix,
    });
}
/**
 * Verifies a signed root-tx-chain registry record. `expectedRootAddress` and
 * `expectedRootRegistryId` are the trusted root identity / immutable root registry
 * id and MUST come from configuration — never from the request or record.
 */
export async function verifyRegistryRecord(signed, expectedRootAddress, expectedRootRegistryId, now) {
    const out = callLocal('signing.verify_registry_record', {
        signed, expected_root_address: expectedRootAddress, expected_root_registry_id: expectedRootRegistryId, now_unix: unixSeconds(now),
    });
    return out.record;
}
/** Signs a bond-hold request bound to the root registry id. */
export async function signBondHold(privateKey, rootRegistryId, hold, issuedAtUnix) {
    return callLocal('signing.sign_bond_hold', {
        private_key_base64: toBase64(privateKey), root_registry_id: rootRegistryId, hold: wireBondHold(hold), issued_at_unix: issuedAtUnix,
    });
}
/** Verifies a bond-hold request. */
export async function verifyBondHold(signed, expectedRootRegistryId, now) {
    const out = callLocal('signing.verify_bond_hold', {
        signed, root_registry_id: expectedRootRegistryId, now_unix: unixSeconds(now),
    });
    return out.hold;
}
/** Signs a bond-release request bound to the boot node chain id. */
export async function signBondRelease(privateKey, bootNodeChainId, release, issuedAtUnix) {
    return callLocal('signing.sign_bond_release', {
        private_key_base64: toBase64(privateKey), boot_node_chain_id: bootNodeChainId, release: wireBondRelease(release), issued_at_unix: issuedAtUnix,
    });
}
/** Verifies a bond-release request. */
export async function verifyBondRelease(signed, expectedBootNodeChainId, now) {
    const out = callLocal('signing.verify_bond_release', {
        signed, boot_node_chain_id: expectedBootNodeChainId, now_unix: unixSeconds(now),
    });
    return out.release;
}
/** Signs the new boot-node key's enrollment challenge, binding the operator claim into the
 * signed payload (operator-to-key binding, DEC-2). Thin pass-through to the Rust core. */
export async function signBootnodeEnrollChallenge(privateKey, chainId, nonce, issuedAtUnix, expiresAtUnix, operator, claims) {
    return callLocal('signing.sign_bootnode_enroll_challenge', {
        private_key_base64: toBase64(privateKey), chain_id: chainId, nonce,
        issued_at_unix: issuedAtUnix, expires_at_unix: expiresAtUnix, operator,
        claims: toBase64(claims),
    });
}
/** Signs the root owner's approval of a challenge; the approval payload is the exact base64
 * envelope of the challenge (approval-to-key binding). Thin pass-through to the Rust core. */
export async function signBootnodeEnrollApproval(rootKey, chainId, nonce, issuedAtUnix, expiresAtUnix, challengeEnvelope) {
    return callLocal('signing.sign_bootnode_enroll_approval', {
        private_key_base64: toBase64(rootKey), chain_id: chainId, nonce,
        issued_at_unix: issuedAtUnix, expires_at_unix: expiresAtUnix, challenge_envelope: challengeEnvelope,
    });
}
/** Verifies an approved enrollment, enforcing operator match and replay prevention. Cryptographic
 * verification is delegated to the Rust core, which authenticates the full approval→challenge→root
 * binding and returns the consumed nonce. The host then durably consumes that nonce through its own
 * atomic NonceStore: a replay (already consumed) yields false even though the crypto check passed.
 * Thin pass-through to the Rust core. */
export async function verifyBootnodeEnrollment(enrollment, expectedRootPublicKey, expectedOperator, store, now) {
    const out = await callLocal('signing.verify_bootnode_enrollment', {
        enrollment, expected_root_public_key: expectedRootPublicKey, expected_operator: expectedOperator,
        now_unix: unixSeconds(now),
    });
    if (!out.valid)
        return false;
    return store.consume(out.nonce);
}
/** True when `signed` is exactly the canonical encoding of `expected` with a valid signature. */
export async function verifyMessageEnvelope(signed, expected) {
    try {
        const canonical = marshalMessageEnvelope(expected);
        const received = fromBase64(signed.envelope);
        if (canonical.length !== received.length || canonical.some((byte, index) => byte !== received[index]))
            return false;
        // The envelope bytes are identical, so checking at its own issue time
        // exercises the signature and address without a wall-clock dependency.
        callLocal('signing.verify_message', {
            signed, domain: expected.domain, purpose: expected.purpose, chain_id: expected.chainId, nonce: expected.nonce,
            now_unix: BigInt(expected.issuedAtUnix),
        });
        return true;
    }
    catch {
        return false;
    }
}
export function marshalRelayEnvelope(envelope) {
    return bytesOf(callLocal('relay.marshal', wireRelay(envelope)));
}
export function parseRelayEnvelope(data) {
    const e = callLocal('relay.parse', { bytes_base64: toBase64(data) });
    return {
        schemaVersion: e.schema_version ?? '', requestId: e.request_id, userSubject: e.user_subject, deviceId: e.device_id, dveId: e.dve_id ?? '',
        targetType: e.target_type, targetId: e.target_id, capability: e.capability, sequence: big(e.sequence), leaseEpoch: big(e.lease_epoch),
        issuedAtUnix: big(e.issued_at_unix), expiresAtUnix: big(e.expires_at_unix), payloadDigest: e.payload_digest,
    };
}
/** Creates a CLI-supervisor relay envelope with a SHA-256 payload binding. */
export function newCLISupervisorRelay(params) {
    if (!Number.isSafeInteger(params.ttlSeconds))
        throw new Error('ttlSeconds must be a safe integer');
    const e = callLocal('relay.new_cli_supervisor', {
        request_id: params.requestId, user_subject: params.userSubject, device_id: params.deviceId, target_id: params.targetId,
        capability: params.capability, sequence: params.sequence, lease_epoch: params.leaseEpoch, payload: byteArray(params.payload),
        ttl_seconds: params.ttlSeconds, issued_at_unix: Math.floor(Date.now() / 1000),
    });
    return {
        schemaVersion: e.schema_version, requestId: e.request_id, userSubject: e.user_subject, deviceId: e.device_id,
        ...(e.dve_id ? { dveId: e.dve_id } : {}), targetType: e.target_type, targetId: e.target_id, capability: e.capability,
        sequence: e.sequence, leaseEpoch: e.lease_epoch ?? 0, issuedAtUnix: e.issued_at_unix, expiresAtUnix: e.expires_at_unix,
        payloadDigest: e.payload_digest,
    };
}
/** Signs the canonical relay-response message derived from a relay envelope. */
export async function signCLISupervisorRelayResponse(privateKey, envelope, chainId) {
    return callLocal('relay.sign_cli_supervisor_response', { private_key_base64: toBase64(privateKey), envelope: wireRelay(envelope), chain_id: chainId });
}
const wirePublication = (p) => ({
    schema_version: p.schemaVersion, network_id: p.networkId, network_fingerprint: p.networkFingerprint, artifact_digest: p.artifactDigest,
    byte_size: p.byteSize, module_kind: p.moduleKind, abi_version: p.abiVersion, module_schema_version: p.moduleSchemaVersion,
    build_id: p.buildId, toolchain_digest: p.toolchainDigest, self_test_digest: p.selfTestDigest, publisher_address: p.publisherAddress,
    dve_template_id: p.dveTemplateId,
});
export function marshalWasmPublicationPayload(payload) {
    return bytesOf(callLocal('wasm.publication.marshal', wirePublication(payload)));
}
export function marshalWasmManifestPayload(p) {
    return bytesOf(callLocal('wasm.assignment.marshal', {
        schema_version: p.schemaVersion, manifest_id: p.manifestId, network_id: p.networkId, chain_id: p.chainId,
        network_fingerprint: p.networkFingerprint, user_subject: p.userSubject, device_id: p.deviceId, dve_id: p.dveId,
        lease_epoch: p.leaseEpoch, relay_target_type: p.relayTargetType, relay_target_id: p.relayTargetId,
        assignment_id: p.assignmentId, assignment_version: p.assignmentVersion, supersedes_assignment_id: p.supersedesAssignmentId,
        modules: p.modules.map((m) => ({
            module_kind: m.moduleKind, artifact_digest: m.artifactDigest, byte_size: m.byteSize, abi_version: m.abiVersion,
            module_schema_version: m.moduleSchemaVersion, capabilities_json: m.capabilitiesJson, configuration_digest: m.configurationDigest,
            download_path: m.downloadPath, publisher_address: m.publisherAddress, publication_statement_digest: m.publicationStatementDigest,
        })),
    }));
}
export function parseWasmManifestPayload(data) {
    const p = callLocal('wasm.assignment.parse', { bytes_base64: toBase64(data) });
    return {
        schemaVersion: p.schema_version, manifestId: p.manifest_id, networkId: p.network_id, chainId: p.chain_id,
        networkFingerprint: p.network_fingerprint, userSubject: p.user_subject, deviceId: p.device_id, dveId: p.dve_id,
        leaseEpoch: big(p.lease_epoch), relayTargetType: p.relay_target_type, relayTargetId: p.relay_target_id,
        assignmentId: p.assignment_id, assignmentVersion: big(p.assignment_version), supersedesAssignmentId: p.supersedes_assignment_id,
        modules: p.modules.map((m) => ({
            moduleKind: m.module_kind, artifactDigest: m.artifact_digest, byteSize: big(m.byte_size), abiVersion: big(m.abi_version),
            moduleSchemaVersion: big(m.module_schema_version), capabilitiesJson: m.capabilities_json, configurationDigest: m.configuration_digest,
            downloadPath: m.download_path, publisherAddress: m.publisher_address, publicationStatementDigest: m.publication_statement_digest,
        })),
    };
}
// ---- Raw (registry-compatible compact) signing contract -----------------------
//
// The raw compact endpoint signs a precomputed 32-byte digest with a named
// identity and returns a hex 64-byte r||s secp256k1 signature (the leading
// recovery byte is stripped) — byte-identical to the registry's legacy dcrd
// SignCompact(priv, digest, true)[1:]. The canonical contract lives in the Rust
// core (core/src/secrets.rs RAW_SIGN_*); this is the generated TypeScript binding
// mirror so the purpose allowlist and 32-byte digest rule cannot drift, and the
// OpenBao plugin enforces the same contract server-side as the outer gate
// (OpenBao.md §4.6).
export const RAW_SIGN_ALLOWED_PURPOSES = ['failover-identity'];
/** Throws when `request` violates the canonical raw-sign contract. */
export function validateRawSign(request) {
    const purpose = request.purpose.trim();
    if (!RAW_SIGN_ALLOWED_PURPOSES.includes(purpose)) {
        throw new Error(`raw signing not permitted for purpose ${JSON.stringify(request.purpose)}; allowed: ${JSON.stringify(RAW_SIGN_ALLOWED_PURPOSES)}`);
    }
    const digest = hexToBytes(request.digestHex.trim());
    if (digest.length !== 32) {
        throw new Error(`raw sign requires a 32-byte digest, got ${digest.length}`);
    }
}
function hexToBytes(s) {
    const clean = s.replace(/^0x/i, '');
    if (clean.length % 2 !== 0 || !/^[0-9a-fA-F]*$/.test(clean)) {
        throw new Error('raw sign digest must be hex');
    }
    const out = new Uint8Array(clean.length / 2);
    for (let i = 0; i < out.length; i++)
        out[i] = parseInt(clean.substr(i * 2, 2), 16);
    return out;
}
/** Independent known-answer vector (core/src/secrets.rs raw_sign_kat). Frozen once:
 * private key 1 → the secp256k1 generator point G. The OpenBao plugin signs with
 * dcrd and must reproduce `sig64Hex` for `digestHex` exactly, proving the 64-byte
 * r||s compact format is byte-identical to legacy dcrd SignCompact(..., true)[1:]. */
export const RAW_SIGN_KAT = {
    privHex: '0000000000000000000000000000000000000000000000000000000000000001',
    digestHex: '27a63df12ab9b1c25967ac3a371c19b65d6f326fcd0748b2b3f657bfd0e64e22',
    compactHex: '20031002a1c9f5e00ac84199972dc2c02dec4580d703d9f572d0fcbd960d33c53e10055df6e29204b5c780de6f0cf6a3ed9e4f017150c04feab3423f43cf2074e4',
    sig64Hex: '031002a1c9f5e00ac84199972dc2c02dec4580d703d9f572d0fcbd960d33c53e10055df6e29204b5c780de6f0cf6a3ed9e4f017150c04feab3423f43cf2074e4',
    pubHex: '0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798',
    recoveryByte: 0x20,
};
