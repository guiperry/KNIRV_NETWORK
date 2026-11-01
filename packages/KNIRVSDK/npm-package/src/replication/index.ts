// Binding over the Rust core's replication module. The head format, its
// canonical hash, the chain rules and pull authentication live in
// core/src/replication.rs; this file only carries types and op calls.
import { callLocal } from '../local.js';

export interface Head {
  version: number;
  network_id: string;
  oracle_chain_id: string;
  owner_address: string;
  epoch: number;
  seq: number;
  mmr_size: number;
  mmr_root: string;
  registry_hash: string;
  state_log_size: number;
  state_root: string;
  aux_hash: string;
  consensus_height: number;
  produced_at: number;
  prev_head_hash: string;
}

export interface SignedHead {
  head: Head;
  signed: {
    envelope: string;
    signature: string;
    public_key: string;
    address: string;
  };
}

export interface PullRequest {
  chain_id: string;
  head_seq_seen: number;
  nonce: string;
  ts: number;
}

export type FollowResult = 'accepted' | 'old' | 'gap' | 'diverged';

interface Failure {
  valid: false;
  code: string;
  message: string;
}

export interface HeadHashResult {
  valid: true;
  hash: string;
}

/** Canonical bytes are decided by the core; returned base64-encoded. */
export function headPayload(head: Head): Uint8Array {
  const out = callLocal<{ payload_base64: string }>('replication.head_payload', { head });
  return Uint8Array.from(atob(out.payload_base64), (c) => c.charCodeAt(0));
}

export function headHash(head: Head): HeadHashResult | Failure {
  return callLocal<HeadHashResult | Failure>('replication.head_hash', { head });
}

export function validateHead(head: Head): true | Failure {
  const out = callLocal<{ valid: boolean; code?: string; message?: string }>(
    'replication.head_validate',
    { head },
  );
  return out.valid ? true : (out as Failure);
}

export function compareHeads(local: Head, next: Head): { result: FollowResult; valid: boolean; message?: string } {
  return callLocal('replication.compare', { local, next });
}

export function verifyHead(
  signed: SignedHead,
  trustAnchor = '',
  nowUnix = Math.floor(Date.now() / 1000),
): { valid: true; head: Head } | Failure {
  return callLocal('replication.verify_head', {
    signed: signed.signed,
    trust_anchor: trustAnchor,
    now_unix: nowUnix,
  });
}

export function pullPayload(request: PullRequest): Uint8Array {
  const out = callLocal<{ payload_base64: string }>('replication.pull_payload', { request });
  return Uint8Array.from(atob(out.payload_base64), (c) => c.charCodeAt(0));
}

export function verifyPull(
  signed: SignedHead['signed'],
  allowed: string[],
  nowUnix = Math.floor(Date.now() / 1000),
  skewSeconds = 120,
): { valid: true; request: PullRequest } | Failure {
  return callLocal('replication.verify_pull', {
    signed,
    allowed,
    now_unix: nowUnix,
    skew_seconds: skewSeconds,
  });
}
