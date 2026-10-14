/**
 * Wire contract for the KNIRVMODELS daemon (TypeScript twin of
 * go-package/models). Both are tested against testvectors/models so they cannot
 * drift silently.
 */
/** Where the daemon serves each model. One namespace for all three. */
export declare const HEAD_PREFIX = "/models/head";
export declare const REGISTRY_PATH = "/models/registry";
export declare const HEALTH_PATH = "/models/health";
export declare const HEART_PREFIX = "/models/heart";
export declare const HASHER_PREFIX = "/models/hasher";
export type ModelRole = "validator" | "generator" | "reasoner";
/** One registry row. `loaded` is false for a model not built yet (HEART is lazy). */
export interface ModelStatus {
    name: string;
    role: ModelRole;
    version: string;
    ready: boolean;
    loaded: boolean;
    trained: boolean;
    backend: string;
    since?: string;
    error?: string;
    detail?: unknown;
}
export interface Health {
    status: string;
    time: string;
    models: number;
}
export interface HeartHealth {
    available: boolean;
    status: string;
    loaded: boolean;
    ready: boolean;
    progress: number;
}
export interface HasherValidateRequest {
    prediction: number;
    domain: string;
    context?: Record<string, unknown>;
}
export interface HasherValidateResult {
    prediction: number;
    valid: boolean;
    domain: string;
    rules_applied: number;
    errors?: string[];
    latency_ms: number;
}
export interface HasherInferResult {
    prediction: number;
    confidence: number;
    mode: string;
    valid_passes: number;
    total_passes: number;
    latency_ms: number;
}
export interface HasherStatus {
    loaded: boolean;
    mode?: string;
    hardware_tier?: string;
    using_hardware?: boolean;
    total_inferences: number;
    hardware_inferences: number;
    software_fallbacks: number;
}
/** The model is up but cannot answer yet (HTTP 503). */
export declare class NotReadyError extends Error {
    constructor();
}
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
    registry(): Promise<ModelStatus[]>;
    health(): Promise<Health>;
    heartHealth(): Promise<HeartHealth>;
    hasherStatus(): Promise<HasherStatus>;
    private post;
    /** HEART text; NotReadyError until HEART has a trained generator. */
    heartGenerate(prompt: string): Promise<string>;
    hasherValidate(req: HasherValidateRequest): Promise<HasherValidateResult>;
    hasherInfer(text: string): Promise<HasherInferResult>;
    manifest(): Promise<Manifest>;
    reason(tokens: number[]): Promise<Reasoning>;
    /** manifest → encode → reason → advice; every failure is an abstention. */
    advise(c: AdviseContext): Promise<SupervisorAdvice>;
}
/** The backend's authenticated route for the platform-wide model view. */
export declare const STATUS_PATH = "/api/models/status";
export interface LlamaAdapter {
    name: string;
    loaded: boolean;
    base_sha256?: string;
    bundle_hash?: string;
}
/** The embedded llama.cpp. `base_model` is a file name, never a path. */
export interface LlamaStatus {
    ready: boolean;
    base_model?: string;
    base_sha256?: string;
    adapters?: LlamaAdapter[];
    error?: string;
}
/** One harness decision, without any prompt or output text. */
export interface RoutingEvent {
    id: string;
    time: string;
    kind: string;
    skill?: string;
    chosen?: string;
    model_version?: string;
    adapter?: string;
    degraded?: boolean;
    fallbacks?: string[];
    reason?: string;
    notes?: string[];
    latency_ms: number;
    error?: string;
}
export interface RouteStats {
    calls: number;
    failures: number;
    p50_ms: number;
    p95_ms: number;
    circuit: string;
    last_error?: string;
}
export interface PlatformStatus {
    time: string;
    daemon: ModelStatus[];
    daemon_error?: string;
    llama?: LlamaStatus;
    routing: RoutingEvent[];
    stats: Record<string, RouteStats>;
}
export type RoleState = "ready" | "idle" | "not ready" | "unavailable";
/** One row of "which model does what": role -> model -> version. */
export interface RoleLine {
    role: string;
    label: string;
    model: string;
    version: string;
    state: RoleState;
    detail?: string;
}
/** The fixed list of role lines for a status document (same logic as the Go twin). */
export declare function roles(s: PlatformStatus): RoleLine[];
