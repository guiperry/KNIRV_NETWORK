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
export declare function headPayload(head: Head): Uint8Array;
export declare function headHash(head: Head): HeadHashResult | Failure;
export declare function validateHead(head: Head): true | Failure;
export declare function compareHeads(local: Head, next: Head): {
    result: FollowResult;
    valid: boolean;
    message?: string;
};
export declare function verifyHead(signed: SignedHead, trustAnchor?: string, nowUnix?: number): {
    valid: true;
    head: Head;
} | Failure;
export declare function pullPayload(request: PullRequest): Uint8Array;
export declare function verifyPull(signed: SignedHead['signed'], allowed: string[], nowUnix?: number, skewSeconds?: number): {
    valid: true;
    request: PullRequest;
} | Failure;
export {};
