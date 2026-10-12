/**
 * Wire contract for the KNIRVMODELS daemon (TypeScript twin of
 * go-package/models). Both are tested against testvectors/models so they cannot
 * drift silently.
 */
/** Where the daemon serves the HEAD (moves to /models/head with the registry). */
export declare const HEAD_PREFIX = "/heart/head";
export interface Decision {
    name: string;
    kind: string;
    options: string[];
    probabilities: number[];
    selected: string | null;
    value: number | null;
    confidence: number | null;
    abstain: boolean;
}
export interface Reasoning {
    plan_tokens: number[];
    halting_steps: number;
    max_steps?: number;
    halt_probability?: number;
    confidence: number;
    decisions: Decision[];
    l_module_activations?: number[];
    h_module_activations?: number[];
    model_version: string;
    dataset_manifest_hash: string;
    policy_bundle_hash: string;
}
export interface Manifest {
    version: string;
    config: {
        max_seq_len?: number;
        summary_slots?: number;
        dataset_manifest_hash?: string;
        [k: string]: unknown;
    };
    sha256: string;
    size: number;
    signature?: string;
}
export declare const BYTE_V1: {
    readonly EOS: 259;
    readonly PAD: 256;
    readonly BOS: 257;
};
export declare function findDecision(r: Reasoning, name: string): Decision | undefined;
/** Plan text: bytes up to the first EOS, specials dropped. */
export declare function planText(r: Reasoning): string;
/** Tokens the model takes: its sequence length minus the summary slots. */
export declare function inputWidth(m: Manifest): number;
/** BOS + the text's UTF-8 bytes, truncated or PAD-filled to `width`. */
export declare function encodeByteV1(text: string, width: number): number[];
export interface AdviseContext {
    current_phase?: string;
    previous_phase?: string;
    remaining_budget?: Record<string, number>;
    loop_iterations?: Record<string, number>;
    agent_chain?: string[];
    policy_bundle_hash?: string;
    evidence_summary?: {
        class: string;
    }[];
    task_summary?: {
        request?: string;
        intent?: string;
        risk?: string;
    };
}
export interface SupervisorAdvice {
    recommended_action: string;
    next_phase: string;
    missing_evidence?: string[];
    policy_references?: string[];
    confidence: number;
    abstain: boolean;
    abstain_reason?: string;
    retrieval_refs?: string[];
    model_version: string;
    dataset_manifest_hash: string;
    policy_bundle_hash: string;
    received_at: string;
}
/** Canonical model input; identical to the Go BuildAdvisePrompt and the trainer's `prompt`. */
export declare function buildAdvisePrompt(c: AdviseContext): string;
export declare function abstainAdvice(reason: string, modelVersion: string, manifest: string): SupervisorAdvice;
/** Maps decision heads onto SupervisorAdvice; every doubtful case abstains. */
export declare function adviceFromReasoning(r: Reasoning, c: AdviseContext): SupervisorAdvice;
export declare class NoModelError extends Error {
    constructor();
}
/** Client for the daemon's HEAD routes; `baseUrl` is the backend or daemon origin. */
export declare class ModelsClient {
    private baseUrl;
    private fetchImpl;
    private headers;
    constructor(baseUrl: string, fetchImpl?: typeof fetch, headers?: Record<string, string>);
    private json;
    manifest(): Promise<Manifest>;
    reason(tokens: number[]): Promise<Reasoning>;
    /** manifest → encode → reason → advice; every failure is an abstention. */
    advise(c: AdviseContext): Promise<SupervisorAdvice>;
}
