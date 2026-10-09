/**
 * Verifies a CLI supervisor relay-response before the arena acts on it.
 *
 * Ported from the Controller (`controller/src/react-app/hooks/verifyCliResponse.ts`)
 * so both clients apply the same rules:
 *   1. the response request_id equals the request we sent;
 *   2. the knirv.message.v1 envelope is domain `knirv.controller`, purpose
 *      `relay-response`, and on the expected chain;
 *   3. the inner relay envelope targets the same CLI supervisor and request;
 *   4. the payload digest matches the returned payload bytes;
 *   5. the CLI's ed25519 signature over the envelope verifies.
 * The arena additionally pins the CLI key per pairing (D8): once a pairing has
 * answered, a response signed by any other key is rejected.
 */

import {
  CONTROLLER_DOMAIN,
  PURPOSE_RELAY_RESPONSE,
  RELAY_TARGET_CLI_SUPERVISOR,
  parseMessageEnvelope,
  parseRelayEnvelope,
} from '@knirv/sdk/signing';

export interface CliResponsePayload {
  request_id: string;
  envelope?: string;
  message_envelope?: string;
  signature?: string;
  public_key?: string;
  purpose?: string;
  payload_b64?: string;
  error?: string;
}

export interface CliVerification {
  ok: boolean;
  error?: string;
  /** The CLI's base64 ed25519 key, to pin for this pairing. */
  publicKey?: string;
  /** Decoded response payload, when present. */
  payload?: Uint8Array;
}

const fromB64 = (b64: string): Uint8Array => Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));

async function sha256Hex(webCrypto: Crypto, data: Uint8Array): Promise<string> {
  const digest = new Uint8Array(await webCrypto.subtle.digest('SHA-256', data));
  return Array.from(digest, (b) => b.toString(16).padStart(2, '0')).join('');
}

export async function verifyCliResponse(
  requestId: string,
  response: CliResponsePayload,
  expectedChainId: string,
  expectedTarget: string,
  options: { pinnedPublicKey?: string; webCrypto?: Crypto } = {},
): Promise<CliVerification> {
  const webCrypto = options.webCrypto ?? globalThis.crypto;
  if (response.request_id !== requestId) {
    return { ok: false, error: `request ID mismatch: expected ${requestId}, got ${response.request_id}` };
  }
  if (response.error && !response.message_envelope) {
    return { ok: false, error: `CLI denied or failed the request: ${response.error}` };
  }
  if (!response.message_envelope || !response.signature || !response.public_key) {
    return { ok: false, error: 'CLI response is missing message_envelope, signature or public_key' };
  }
  if (options.pinnedPublicKey && options.pinnedPublicKey !== response.public_key) {
    return { ok: false, error: 'CLI response was signed by a different key than this pairing is pinned to' };
  }

  let envelopeBytes: Uint8Array;
  let fields: ReturnType<typeof parseMessageEnvelope>;
  try {
    envelopeBytes = fromB64(response.message_envelope);
    fields = parseMessageEnvelope(envelopeBytes);
  } catch (err) {
    return { ok: false, error: `invalid response message_envelope: ${err instanceof Error ? err.message : String(err)}` };
  }
  if (fields.domain !== CONTROLLER_DOMAIN || fields.purpose !== PURPOSE_RELAY_RESPONSE || fields.chainId !== expectedChainId) {
    return { ok: false, error: `unexpected response envelope: domain=${fields.domain}, purpose=${fields.purpose}, chainId=${fields.chainId}` };
  }

  let payload: Uint8Array | undefined;
  try {
    const relay = parseRelayEnvelope(fields.payload);
    if (relay.targetType !== RELAY_TARGET_CLI_SUPERVISOR || relay.targetId !== expectedTarget || relay.requestId !== requestId) {
      return { ok: false, error: `response relay target mismatch: type=${relay.targetType}, targetId=${relay.targetId}, requestId=${relay.requestId}` };
    }
    if (response.payload_b64) {
      payload = fromB64(response.payload_b64);
      if (relay.payloadDigest !== `sha256:${await sha256Hex(webCrypto, payload)}`) {
        return { ok: false, error: 'response payload digest mismatch' };
      }
    }
  } catch (err) {
    return { ok: false, error: `invalid response relay envelope: ${err instanceof Error ? err.message : String(err)}` };
  }

  const publicKey = fromB64(response.public_key);
  const signature = fromB64(response.signature);
  if (publicKey.byteLength !== 32 || signature.byteLength !== 64) {
    return { ok: false, error: 'invalid ed25519 key or signature length' };
  }
  let valid = false;
  try {
    const key = await webCrypto.subtle.importKey('raw', publicKey, { name: 'Ed25519' } as AlgorithmIdentifier, false, ['verify']);
    valid = await webCrypto.subtle.verify({ name: 'Ed25519' } as AlgorithmIdentifier, key, signature, envelopeBytes);
  } catch {
    return { ok: false, error: 'ed25519 verification is not supported in this browser' };
  }
  if (!valid) return { ok: false, error: 'ed25519 signature verification failed' };
  return { ok: true, publicKey: response.public_key, payload };
}
