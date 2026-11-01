// Exercises every @knirv/sdk/signing wrapper end to end. The logic under test
// is the Rust core's (via WASM); this checks the TypeScript mapping around it:
// camelCase/snake_case, optional fields, bigint u64 handling, and error paths.
import assert from "node:assert/strict";
import {
  initLocal, publicKeyToKNIRVAddress, marshalAction, buildDirectSignDoc, marshalTxRaw, signDirectTransaction,
  marshalMessageEnvelope, signMessageEnvelope, parseMessageEnvelope, verifyMessage, verifyMessagePayload, verifyMessageEnvelope,
  marshalRelayEnvelope, parseRelayEnvelope, newCLISupervisorRelay, signCLISupervisorRelayResponse,
  marshalWasmPublicationPayload, marshalWasmManifestPayload, parseWasmManifestPayload,
  CONTROLLER_DOMAIN, PURPOSE_RELAY_RESPONSE, RELAY_TARGET_CLI_SUPERVISOR,
} from "../signing.js";

const key = new Uint8Array(32); key[31] = 1;
const ADDRESS = "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e";
const b64 = (s) => Uint8Array.from(Buffer.from(s, "base64"));
const eq = (a, b) => assert.deepEqual(Array.from(a), Array.from(b));

await initLocal();
await initLocal(); // idempotent

// Address of private key 1 is the same string Go and Rust derive.
const { pubkey } = await (async () => {
  const signed = await signMessageEnvelope(key, { domain: "d", purpose: "p", chainId: "c", nonce: "n", issuedAtUnix: 1_750_000_000, expiresAtUnix: 1_750_000_600, payload: new Uint8Array() });
  return { pubkey: b64(signed.public_key) };
})();
assert.equal(publicKeyToKNIRVAddress(pubkey), ADDRESS);
assert.throws(() => publicKeyToKNIRVAddress(new Uint8Array(32)), /33 bytes/);
assert.throws(() => publicKeyToKNIRVAddress(new Uint8Array(33)), /public key/); // not a curve point

// Direct transaction: deterministic, canonical hash casing, tamper-evident sign doc.
const request = {
  action: { action: "transfer", sender: "knirv1sender", recipient: "knirv1recipient", amount: 42, payload: new TextEncoder().encode('{"memo":"hello"}'), timestampUnix: 1_750_000_000 },
  chainId: "knirv-testnet-1", accountNumber: 7, sequence: 9, fee: { denom: "unrn", amount: "3", gasLimit: 200_000 },
};
const tx1 = await signDirectTransaction(key, request), tx2 = await signDirectTransaction(key, request);
assert.deepEqual(tx1, tx2);
assert.equal(tx1.address, ADDRESS);
assert.match(tx1.hash, /^[0-9A-F]{64}$/);
const doc = buildDirectSignDoc(request, pubkey);
eq(doc.bodyBytes, b64(tx1.body_bytes)); eq(doc.authInfoBytes, b64(tx1.auth_info_bytes));
assert.equal(marshalTxRaw(doc.bodyBytes, doc.authInfoBytes, [b64(tx1.signatures[0])]).length > 0, true);
assert.throws(() => marshalAction({ ...request.action, action: " " }), /required/);
assert.throws(() => buildDirectSignDoc({ ...request, chainId: "" }, pubkey), /chain_id/);

// u64 above 2^53 must reach the core exactly: 2^64-1 is nine 0xff-ish varint bytes then 0x01.
const huge = marshalAction({ ...request.action, amount: 2n ** 64n - 1n });
assert.ok(Buffer.from(huge).includes(Buffer.from([0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01])), "max u64 amount lost precision");

// Message envelopes: round trip, windows, domain separation, payload binding.
const now = new Date(1_750_000_100 * 1000);
const env = { domain: CONTROLLER_DOMAIN, purpose: "cli-sign", chainId: "knirv-testnet-1", nonce: "request-123", issuedAtUnix: 1_750_000_000, expiresAtUnix: 1_750_000_300, payload: new TextEncoder().encode("approve transfer") };
const signed = await signMessageEnvelope(key, env);
const parsed = parseMessageEnvelope(marshalMessageEnvelope(env));
assert.equal(parsed.issuedAtUnix, 1_750_000_000n); assert.equal(typeof parsed.expiresAtUnix, "bigint");
assert.equal(parsed.nonce, "request-123"); eq(parsed.payload, env.payload);
await verifyMessage(signed, env.domain, env.purpose, env.chainId, env.nonce, now);
await verifyMessagePayload(signed, env.domain, env.purpose, env.chainId, env.payload, now);
await assert.rejects(verifyMessage(signed, env.domain, "login", env.chainId, env.nonce, now), /domain/);
await assert.rejects(verifyMessage(signed, env.domain, env.purpose, env.chainId, env.nonce, new Date(1_750_001_000 * 1000)), /validity window/);
await assert.rejects(verifyMessagePayload(signed, env.domain, env.purpose, env.chainId, new TextEncoder().encode("other"), now), /payload/);
assert.equal(await verifyMessageEnvelope(signed, env), true);
assert.equal(await verifyMessageEnvelope(signed, { ...env, nonce: "different" }), false);
assert.equal(await verifyMessageEnvelope({ ...signed, signature: Buffer.from(new Uint8Array(64)).toString("base64") }, env), false);
assert.throws(() => parseMessageEnvelope(new Uint8Array([1, 2, 3])));

// Relay envelopes: new -> marshal -> parse -> sign response.
const relay = newCLISupervisorRelay({ requestId: "r-1", userSubject: "u", deviceId: "d", targetId: "t", capability: "cap", sequence: 5, leaseEpoch: 3, payload: new TextEncoder().encode("hi"), ttlSeconds: 120 });
assert.equal(relay.targetType, RELAY_TARGET_CLI_SUPERVISOR); assert.match(relay.payloadDigest, /^sha256:[0-9a-f]{64}$/);
const back = parseRelayEnvelope(marshalRelayEnvelope(relay));
assert.equal(back.sequence, 5n); assert.equal(back.leaseEpoch, 3n); assert.equal(back.dveId, ""); assert.equal(back.requestId, "r-1");
assert.throws(() => newCLISupervisorRelay({ requestId: "r", userSubject: "u", deviceId: "d", targetId: "t", capability: "c", sequence: 1, leaseEpoch: 0, payload: new Uint8Array(), ttlSeconds: 1.5 }), /safe integer/);
assert.throws(() => marshalRelayEnvelope({ ...relay, targetType: "nope" }), /target_type/);
const resp = await signCLISupervisorRelayResponse(key, relay, "knirv-testnet-1");
await verifyMessage(resp, CONTROLLER_DOMAIN, PURPOSE_RELAY_RESPONSE, "knirv-testnet-1", relay.requestId, new Date((relay.issuedAtUnix + 1) * 1000));

// WASM payloads.
const pub = marshalWasmPublicationPayload({ networkId: "n", networkFingerprint: "f", artifactDigest: "sha256:" + "a".repeat(64), byteSize: 10, moduleKind: "crypto", buildId: "b", publisherAddress: ADDRESS });
assert.ok(pub.length > 0);
assert.throws(() => marshalWasmPublicationPayload({ networkId: "", networkFingerprint: "f", artifactDigest: "d", byteSize: 1, moduleKind: "k", buildId: "b", publisherAddress: "p" }), /required/);
const manifest = { manifestId: "m", networkId: "n", chainId: "c", networkFingerprint: "f", userSubject: "u", deviceId: "d", assignmentId: "a", leaseEpoch: 4, assignmentVersion: 2, modules: [{ moduleKind: "crypto", artifactDigest: "sha256:x", byteSize: 99, abiVersion: 1, moduleSchemaVersion: 1, capabilitiesJson: "[]", configurationDigest: "", downloadPath: "/p", publisherAddress: ADDRESS, publicationStatementDigest: "" }] };
const m2 = parseWasmManifestPayload(marshalWasmManifestPayload(manifest));
assert.equal(m2.manifestId, "m"); assert.equal(m2.leaseEpoch, 4n); assert.equal(m2.modules[0].byteSize, 99n); assert.equal(m2.modules[0].downloadPath, "/p");
assert.throws(() => marshalWasmManifestPayload({ ...manifest, modules: [] }), /module/);

console.log("signing wrappers: all checks passed");
