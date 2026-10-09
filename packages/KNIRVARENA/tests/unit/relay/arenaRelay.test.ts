/**
 * Arena relay: per-pairing secp256k1 keys, CLI-verifiable request signing, the
 * hub request transport, and CLI response verification (arena_fixes.md §2).
 */
import '../../utils/realWebAssembly'; // must stay first: see the file
import { webcrypto } from 'node:crypto';
import {
  CONTROLLER_DOMAIN,
  MESSAGE_SCHEMA_VERSION,
  PURPOSE_RELAY_REQUEST,
  PURPOSE_RELAY_RESPONSE,
  RELAY_TARGET_CLI_SUPERVISOR,
  marshalMessageEnvelope,
  marshalRelayEnvelope,
  parseMessageEnvelope,
  parseRelayEnvelope,
  verifyMessagePayload,
} from '@knirv/sdk/signing';
import { ArenaRelayKeys, MemoryArenaKeyStore, ARENA_RELAY_CHAIN_ID } from '../../../src/relay/arenaRelayKeys';
import { ArenaRelayClient, ArenaRelayError, toBase64, type SignedArenaRelayRequest } from '../../../src/relay/arenaRelayClient';
import { verifyCliResponse } from '../../../src/relay/verifyCliResponse';

const wc = webcrypto as unknown as Crypto;

const fromB64 = (b64: string) => Uint8Array.from(Buffer.from(b64, 'base64'));
const PAIRING = 'pairing-abc';

function newClient(keys = new ArenaRelayKeys(new MemoryArenaKeyStore(), wc)) {
  return { keys, client: new ArenaRelayClient((env, fields) => keys.signRelayRequest(PAIRING, env, fields), wc) };
}

/** Builds a response the way the CLI's buildSignedResponse does, signed with ed25519. */
async function cliResponse(signed: SignedArenaRelayRequest, result: unknown, key: CryptoKeyPair) {
  const payload = new TextEncoder().encode(JSON.stringify(result));
  const digest = Buffer.from(await wc.subtle.digest('SHA-256', payload)).toString('hex');
  const relay = marshalRelayEnvelope({ ...signed.relayEnvelope, payloadDigest: `sha256:${digest}` });
  const envelope = marshalMessageEnvelope({
    schemaVersion: MESSAGE_SCHEMA_VERSION, domain: CONTROLLER_DOMAIN, purpose: PURPOSE_RELAY_RESPONSE,
    chainId: ARENA_RELAY_CHAIN_ID, nonce: signed.requestId, issuedAtUnix: 1n, expiresAtUnix: 9_999_999_999n, payload: relay,
  });
  const signature = new Uint8Array(await wc.subtle.sign('Ed25519', key.privateKey, envelope));
  const publicKey = new Uint8Array(await wc.subtle.exportKey('raw', key.publicKey));
  return {
    request_id: signed.requestId,
    message_envelope: toBase64(envelope),
    signature: toBase64(signature),
    public_key: toBase64(publicKey),
    payload_b64: toBase64(payload),
  };
}

const ed25519 = () => wc.subtle.generateKey('Ed25519', true, ['sign', 'verify']) as Promise<CryptoKeyPair>;

function fakeFetch(status: number, body: unknown) {
  const calls: Array<{ url: string; init: RequestInit }> = [];
  const impl = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return { ok: status >= 200 && status < 300, status, text: async () => (typeof body === 'string' ? body : JSON.stringify(body)) };
  }) as unknown as typeof fetch;
  return { impl, calls };
}

describe('ArenaRelayKeys', () => {
  it('creates one secp256k1 key per pairing and stores it only encrypted', async () => {
    const store = new MemoryArenaKeyStore();
    const keys = new ArenaRelayKeys(store, wc);
    const first = await keys.ensureKey(PAIRING);
    expect(fromB64(first.publicKey)).toHaveLength(33);
    expect(first.address).toMatch(/^knirv1[0-9a-z]+$/);
    expect(await keys.ensureKey(PAIRING)).toEqual(first);
    expect((await keys.ensureKey('other-pairing')).publicKey).not.toEqual(first.publicKey);

    const stored = await store.getKey(PAIRING);
    expect(stored?.ciphertext.length).toBe(32 + 16); // AES-GCM: 32-byte key + 16-byte tag
    expect((await store.getWrappingKey())?.extractable).toBe(false);
  });

  it('signs a knirv.message.v1 relay-request that the SDK verifier (the CLI rule) accepts', async () => {
    const keys = new ArenaRelayKeys(new MemoryArenaKeyStore(), wc);
    const relayBytes = new Uint8Array([1, 2, 3]);
    const now = Math.floor(Date.now() / 1000);
    const signed = await keys.signRelayRequest(PAIRING, relayBytes, { requestId: 'req-1', issuedAtUnix: now, expiresAtUnix: now + 60 });
    await expect(verifyMessagePayload(signed, CONTROLLER_DOMAIN, PURPOSE_RELAY_REQUEST, 'knirv-1', relayBytes, new Date())).resolves.toBeUndefined();
    expect(parseMessageEnvelope(fromB64(signed.envelope)).nonce).toBe('req-1');
    expect(signed.public_key).toBe((await keys.ensureKey(PAIRING)).publicKey);
  });
});

describe('ArenaRelayClient', () => {
  it('builds a cli_supervisor relay with the payload and posts it to the hub', async () => {
    const { client } = newClient();
    const payload = new TextEncoder().encode(JSON.stringify({ error_node_id: 'err-1', agent: 'claude' }));
    const signed = await client.createSignedRequest({ userSubject: 'u', deviceId: 'd', pairingId: PAIRING, capability: 'arena.agent.deploy', payload, leaseEpoch: 3 });

    const relay = parseRelayEnvelope(parseMessageEnvelope(fromB64(signed.body.signed_envelope)).payload);
    expect(relay.targetType).toBe(RELAY_TARGET_CLI_SUPERVISOR);
    expect(relay.targetId).toBe(PAIRING);
    expect(relay.capability).toBe('arena.agent.deploy');
    expect(relay.requestId).toBe(signed.requestId);
    expect(Number(relay.leaseEpoch)).toBe(3);
    expect(relay.payloadDigest).toBe(`sha256:${Buffer.from(await wc.subtle.digest('SHA-256', payload)).toString('hex')}`);
    expect(fromB64(signed.body.payload_b64)).toEqual(payload);

    const cliKey = await ed25519();
    const { impl, calls } = fakeFetch(200, await cliResponse(signed, { accepted: true }, cliKey));
    const result = await client.send(signed, { serverUrl: 'https://gw.example/', token: 'jwt', fetchImpl: impl, webCrypto: wc });
    expect(JSON.parse(new TextDecoder().decode(result.payload))).toEqual({ accepted: true });
    expect(calls[0].url).toBe(`https://gw.example/api/v1/cli-supervisor/sessions/${PAIRING}/relay`);
    const headers = calls[0].init.headers as Record<string, string>;
    expect(headers.Authorization).toBe('Bearer jwt');
    expect(headers['X-Relay-Request-ID']).toBe(signed.requestId);
    expect(headers['X-Relay-Capability']).toBe('arena.agent.deploy');
  });

  it('rejects a lease epoch below 1 and a missing pairing', async () => {
    const { client } = newClient();
    await expect(client.createSignedRequest({ userSubject: 'u', deviceId: 'd', pairingId: PAIRING, capability: 'arena.status', payload: new Uint8Array(), leaseEpoch: 0 })).rejects.toThrow(/lease epoch/);
    await expect(client.createSignedRequest({ userSubject: 'u', deviceId: 'd', pairingId: '', capability: 'arena.status', payload: new Uint8Array(), leaseEpoch: 1 })).rejects.toThrow(/Pair the arena/);
  });

  it('surfaces hub errors with their code', async () => {
    const { client } = newClient();
    const signed = await client.createSignedRequest({ userSubject: 'u', deviceId: 'd', pairingId: PAIRING, capability: 'arena.status', payload: new Uint8Array(), leaseEpoch: 1 });
    const { impl } = fakeFetch(404, { error: { code: 'CLI_SESSION_NOT_CONNECTED', message: 'CLI supervisor session is not connected' } });
    const err = await client.send(signed, { serverUrl: 'https://gw', token: 't', fetchImpl: impl, webCrypto: wc }).catch((e) => e);
    expect(err).toBeInstanceOf(ArenaRelayError);
    expect(err.status).toBe(404);
    expect(err.code).toBe('CLI_SESSION_NOT_CONNECTED');
  });
});

describe('verifyCliResponse', () => {
  async function signedPair() {
    const { client } = newClient();
    const signed = await client.createSignedRequest({ userSubject: 'u', deviceId: 'd', pairingId: PAIRING, capability: 'chat.invoke', payload: new Uint8Array([7]), leaseEpoch: 1 });
    return { signed, key: await ed25519() };
  }

  it('accepts a correctly signed response and reports the key to pin', async () => {
    const { signed, key } = await signedPair();
    const response = await cliResponse(signed, { text: 'hi' }, key);
    const out = await verifyCliResponse(signed.requestId, response, ARENA_RELAY_CHAIN_ID, PAIRING, { webCrypto: wc });
    expect(out.ok).toBe(true);
    expect(out.publicKey).toBe(response.public_key);
  });

  it('rejects a tampered payload, another request, a forged signature and an unpinned key', async () => {
    const { signed, key } = await signedPair();
    const good = await cliResponse(signed, { text: 'hi' }, key);

    expect((await verifyCliResponse(signed.requestId, { ...good, payload_b64: toBase64(new TextEncoder().encode('{"text":"evil"}')) }, ARENA_RELAY_CHAIN_ID, PAIRING, { webCrypto: wc })).error).toMatch(/digest/);
    expect((await verifyCliResponse('other-request', good, ARENA_RELAY_CHAIN_ID, PAIRING, { webCrypto: wc })).error).toMatch(/request ID/);
    expect((await verifyCliResponse(signed.requestId, good, ARENA_RELAY_CHAIN_ID, 'other-pairing', { webCrypto: wc })).error).toMatch(/target/);

    const other = await ed25519();
    const forged = await cliResponse(signed, { text: 'hi' }, other);
    expect((await verifyCliResponse(signed.requestId, { ...good, signature: forged.signature }, ARENA_RELAY_CHAIN_ID, PAIRING, { webCrypto: wc })).error).toMatch(/signature verification failed/);
    expect((await verifyCliResponse(signed.requestId, forged, ARENA_RELAY_CHAIN_ID, PAIRING, { webCrypto: wc, pinnedPublicKey: good.public_key })).error).toMatch(/pinned/);
  });
});
