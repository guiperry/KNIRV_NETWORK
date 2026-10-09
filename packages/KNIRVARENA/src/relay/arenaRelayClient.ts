/**
 * Arena relay client (arena_fixes.md §2.2–2.4).
 *
 * The arena reaches the paired CLI through the same signed relay the
 * Controller uses: it builds a `knirv.controller.relay-envelope.v1`, signs it
 * as a `knirv.message.v1` relay-request with its own per-pairing secp256k1 key
 * (D8, see arenaRelayKeys.ts), and POSTs it to the CLISupervisorHub at
 * `POST /api/v1/cli-supervisor/sessions/{pairingId}/relay`. The hub checks the
 * caller owns the pairing and forwards it to the CLI, which re-verifies the
 * signature, purpose, expiry and payload digest before dispatching. The CLI's
 * signed response is verified here (verifyCliResponse.ts) before use.
 */

import {
  RELAY_ENVELOPE_SCHEMA_VERSION,
  RELAY_TARGET_CLI_SUPERVISOR,
  marshalRelayEnvelope,
  type RelayEnvelope,
  type SignedMessageEnvelope,
} from '@knirv/sdk/signing';
import { ARENA_RELAY_CHAIN_ID, type RelaySignatureFields } from './arenaRelayKeys';
import { verifyCliResponse, type CliResponsePayload } from './verifyCliResponse';

export const RELAY_CAPABILITY_CHAT_INVOKE = 'chat.invoke';
export const RELAY_CAPABILITY_ARENA_STATUS = 'arena.status';
export const RELAY_CAPABILITY_ARENA_AGENT_DEPLOY = 'arena.agent.deploy';
export const RELAY_CAPABILITY_ARENA_AGENT_RECALL = 'arena.agent.recall';

/** Matches the CLI's 300-second relay validity ceiling. */
const MAX_TTL_SECONDS = 300;

export interface ArenaRelayRequest {
  userSubject: string;
  deviceId: string;
  pairingId: string;
  capability: string;
  payload: Uint8Array;
  leaseEpoch: number;
  ttlSeconds?: number;
}

/** Wire body accepted by the hub and parsed by the CLI's parseAndVerifyRelay. */
export interface RelayWireBody {
  signed_envelope: string;
  signature: string;
  public_key: string;
  address: string;
  payload_b64: string;
}

export interface SignedArenaRelayRequest {
  requestId: string;
  pairingId: string;
  capability: string;
  relayEnvelope: RelayEnvelope;
  body: RelayWireBody;
}

export type ArenaRelaySigner = (relayEnvelope: Uint8Array, fields: RelaySignatureFields) => Promise<SignedMessageEnvelope>;

export interface SendOptions {
  serverUrl: string;
  token: string;
  chainId?: string;
  /** The CLI key this pairing is pinned to, if it has answered before. */
  pinnedPublicKey?: string;
  fetchImpl?: typeof fetch;
  webCrypto?: Crypto;
}

export interface ArenaRelayResponse {
  payload?: Uint8Array;
  /** The CLI's ed25519 key that signed this response (pin it per pairing). */
  cliPublicKey: string;
}

export class ArenaRelayError extends Error {
  constructor(message: string, readonly status?: number, readonly code?: string) {
    super(message);
    this.name = 'ArenaRelayError';
  }
}

export function toBase64(bytes: Uint8Array): string {
  let binary = '';
  for (let i = 0; i < bytes.length; i += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  }
  return btoa(binary);
}

async function digestPayload(webCrypto: Crypto, payload: Uint8Array): Promise<string> {
  const digest = new Uint8Array(await webCrypto.subtle.digest('SHA-256', payload));
  return `sha256:${Array.from(digest, (byte) => byte.toString(16).padStart(2, '0')).join('')}`;
}

export class ArenaRelayClient {
  private sequence = 0;

  constructor(
    private readonly sign: ArenaRelaySigner,
    private readonly webCrypto: Crypto = globalThis.crypto,
  ) {}

  async createSignedRequest(request: ArenaRelayRequest, chainId = ARENA_RELAY_CHAIN_ID): Promise<SignedArenaRelayRequest> {
    if (!request.pairingId) throw new ArenaRelayError('Pair the arena with your CLI first');
    if (!Number.isSafeInteger(request.leaseEpoch) || request.leaseEpoch < 1) {
      throw new ArenaRelayError('A valid relay lease epoch is required');
    }
    const issuedAtUnix = Math.floor(Date.now() / 1000);
    const expiresAtUnix = issuedAtUnix + Math.min(Math.max(request.ttlSeconds ?? 60, 1), MAX_TTL_SECONDS);
    const requestId = this.webCrypto.randomUUID();
    const relayEnvelope: RelayEnvelope = {
      schemaVersion: RELAY_ENVELOPE_SCHEMA_VERSION,
      requestId,
      userSubject: request.userSubject,
      deviceId: request.deviceId,
      dveId: request.pairingId,
      targetType: RELAY_TARGET_CLI_SUPERVISOR,
      targetId: request.pairingId,
      capability: request.capability,
      sequence: ++this.sequence,
      leaseEpoch: request.leaseEpoch,
      issuedAtUnix,
      expiresAtUnix,
      payloadDigest: await digestPayload(this.webCrypto, request.payload),
    };
    const signed = await this.sign(marshalRelayEnvelope(relayEnvelope), { requestId, issuedAtUnix, expiresAtUnix, chainId });
    return {
      requestId,
      pairingId: request.pairingId,
      capability: request.capability,
      relayEnvelope,
      body: {
        signed_envelope: signed.envelope,
        signature: signed.signature,
        public_key: signed.public_key,
        address: signed.address,
        payload_b64: toBase64(request.payload),
      },
    };
  }

  /** Sends a signed request through the hub and verifies the CLI's response. */
  async send(signed: SignedArenaRelayRequest, options: SendOptions): Promise<ArenaRelayResponse> {
    const fetchImpl = options.fetchImpl ?? fetch;
    const url = `${options.serverUrl.replace(/\/$/, '')}/api/v1/cli-supervisor/sessions/${encodeURIComponent(signed.pairingId)}/relay`;
    const response = await fetchImpl(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${options.token}`,
        'X-Relay-Request-ID': signed.requestId,
        'X-Relay-Capability': signed.capability,
      },
      body: JSON.stringify(signed.body),
    });
    const text = await response.text();
    if (!response.ok) {
      let code: string | undefined;
      let message = text.trim() || `relay failed with HTTP ${response.status}`;
      try {
        const parsed = JSON.parse(text) as { error?: { code?: string; message?: string } };
        code = parsed.error?.code;
        message = parsed.error?.message ?? message;
      } catch { /* plain-text error */ }
      throw new ArenaRelayError(message, response.status, code);
    }
    let body: CliResponsePayload;
    try {
      body = JSON.parse(text) as CliResponsePayload;
    } catch {
      throw new ArenaRelayError('CLI returned a response that is not JSON');
    }
    const verified = await verifyCliResponse(signed.requestId, body, options.chainId ?? ARENA_RELAY_CHAIN_ID, signed.pairingId, {
      pinnedPublicKey: options.pinnedPublicKey,
      webCrypto: options.webCrypto ?? this.webCrypto,
    });
    if (!verified.ok || !verified.publicKey) throw new ArenaRelayError(verified.error ?? 'CLI response failed verification');
    return { payload: verified.payload, cliPublicKey: verified.publicKey };
  }

  resetSequence(): void {
    this.sequence = 0;
  }
}
