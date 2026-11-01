/**
 * Arena relay WASM worker client.
 *
 * Loads the verified `controller-relay.wasm` artifact (the same crate the
 * Controller runs) into a module worker and exposes the envelope validation
 * / sequence-checking / canonicalization operations the arena needs to verify
 * every `arena-event` it receives from the paired CLI.
 *
 * The artifact is the SDK's published `controller-relay` module
 * (https://releases.knirv.com/sdk/<version>/modules/controller-relay.wasm),
 * fetched by `@knirv/sdk/modules` and checked against the SDK manifest's SHA-256
 * before use. `init` lets tests inject the bytes directly.
 */

interface RelayWorkerRequest {
  id: string;
  type: 'init' | 'self_test' | 'validate_envelope' | 'canonicalize' | 'compute_digest' | 'check_sequence' | 'zeroize' | 'sign_envelope';
  bytes?: ArrayBuffer;
}

interface RelayWorkerResponse {
  id: string;
  ok: boolean;
  result?: unknown;
  error?: string;
}

let workerInstance: Worker | null = null;
let initPromise: Promise<void> | null = null;
let requestId = 0;
const pending = new Map<string, { resolve: (value: unknown) => void; reject: (error: Error) => void }>();

function getWorker(): Worker {
  if (!workerInstance) {
    workerInstance = new Worker(new URL('./arenaRelay.worker.ts', import.meta.url), { type: 'module' });
    workerInstance.onmessage = (event: MessageEvent<RelayWorkerResponse>) => {
      const { id, ok, result, error } = event.data;
      const entry = pending.get(id);
      if (!entry) return;
      pending.delete(id);
      if (ok) entry.resolve(result);
      else entry.reject(new Error(error || 'Worker error'));
    };
    workerInstance.onerror = () => {
      const entries = Array.from(pending.entries());
      pending.clear();
      for (const [, req] of entries) req.reject(new Error('Relay worker crashed'));
      workerInstance = null;
    };
  }
  return workerInstance;
}

function sendRequest(type: string, params: Record<string, unknown> = {}): Promise<unknown> {
  const worker = getWorker();
  const id = `arena-relay-req-${requestId++}`;
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    worker.postMessage({ id, type, ...params });
  });
}

export const arenaRelayWorkerClient = {
  async init(wasmBytes: ArrayBuffer): Promise<void> {
    initPromise ??= (async () => {
      await sendRequest('init', { bytes: wasmBytes });
    })();
    await initPromise;
  },

  async initFromArtifact(): Promise<void> {
    const { fetchWasmModule, wasmModuleMetadata } = await import('@knirv/sdk/modules');
    const expected = wasmModuleMetadata('controller-relay').sha256;
    const bytes = await fetchWasmModule('controller-relay');
    const digest = Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', bytes)), (b) => b.toString(16).padStart(2, '0')).join('');
    if (digest !== expected) throw new Error('controller-relay.wasm does not match the SDK manifest digest');
    await this.init(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer);
  },

  async selfTest(): Promise<unknown> {
    return sendRequest('self_test');
  },

  async validateEnvelope(envelope: RelayEnvelopeLike): Promise<{
    valid: boolean;
    request_id: string;
    target_type: string;
    target_id: string;
    capability: string;
    sequence: number;
    lease_epoch: number;
    issued_at_unix: number;
    expires_at_unix: number;
    payload_digest: string;
    envelope_digest: string;
    error?: string;
  }> {
    const { marshalRelayEnvelope } = await import('@knirv/sdk/signing');
    const bytes = marshalRelayEnvelope(envelope as RelayEnvelope);
    return sendRequest('validate_envelope', { bytes: bytes.buffer }) as Promise<{
      valid: boolean;
      request_id: string;
      target_type: string;
      target_id: string;
      capability: string;
      sequence: number;
      lease_epoch: number;
      issued_at_unix: number;
      expires_at_unix: number;
      payload_digest: string;
      envelope_digest: string;
      error?: string;
    }>;
  },

  async checkSequence(envelope: RelayEnvelopeLike): Promise<{ accepted: boolean }> {
    const { marshalRelayEnvelope } = await import('@knirv/sdk/signing');
    const bytes = marshalRelayEnvelope(envelope as RelayEnvelope);
    return sendRequest('check_sequence', { bytes: bytes.buffer }) as Promise<{ accepted: boolean }>;
  },

  async canonicalize(envelope: RelayEnvelopeLike): Promise<string> {
    const { marshalRelayEnvelope } = await import('@knirv/sdk/signing');
    const bytes = marshalRelayEnvelope(envelope as RelayEnvelope);
    return sendRequest('canonicalize', { bytes: bytes.buffer }) as Promise<string>;
  },

  async generateKeyPair(pairingId: string): Promise<{ public_key: string; address: string }> {
    return sendRequest('generate_keypair', { pairingId }) as Promise<{ public_key: string; address: string }>;
  },

  /**
   * Signs a marshaled relay envelope as a knirv.message.v1 relay-request with
   * the pairing's own secp256k1 key (the format the CLI verifies).
   */
  async signRelayRequest(pairingId: string, relayEnvelope: Uint8Array, fields: RelaySignatureFields): Promise<SignedMessageEnvelope> {
    const bytes = relayEnvelope.slice().buffer;
    return sendRequest('sign_envelope', { bytes, pairingId, ...fields }) as Promise<SignedMessageEnvelope>;
  },

  async zeroize(): Promise<void> {
    if (!initPromise) return;
    await sendRequest('zeroize');
  },

  isInitialized(): boolean {
    return initPromise !== null;
  },

  terminate(): void {
    if (workerInstance) {
      workerInstance.terminate();
      workerInstance = null;
    }
    pending.clear();
    initPromise = null;
  },
};

// Re-exported for typing convenience; the worker validates the marshaled bytes.
import type { RelayEnvelope, SignedMessageEnvelope } from '@knirv/sdk/signing';
import type { RelaySignatureFields } from './arenaRelayKeys';
export type RelayEnvelopeLike = RelayEnvelope;