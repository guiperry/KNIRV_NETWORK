/** Where the daemon serves each model. One namespace for all three. */
export declare const HEAD_PREFIX = "/models/head";
export declare const REGISTRY_PATH = "/models/registry";
export declare const HEALTH_PATH = "/models/health";
export declare const HEART_PREFIX = "/models/heart";
export declare const HEART_DECISION_PREFIX = "/models/heart/decide";
export declare const HASHER_PREFIX = "/models/hasher";
/**
 * HANDS (server_tasks) namespace. Typed transport mirror of the Go SDK's
 * `sdk.HandsPrefix` and `sdk.RoleHands`; the Rust core (core/src/models.rs) is
 * the source of truth, so these are wire-value aliases plus thin HTTP calls.
 */
export declare const HandsPrefix = "/models/hands";
export declare const RoleHands = "server_tasks";
export type ModelRole = "decide" | "generate" | "validate" | "vision" | "server_tasks" | "external" | "validator" | "reasoner";
export declare function serverTasksRole(): ModelRole;
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
export interface HandsRequest {
    n?: number;
    temperature?: number;
    topK?: number;
}
/** HANDS status (GET /models/hands/{status,health}). `role` is RoleHands ("server_tasks"). */
export interface HandsStatus {
    name: string;
    role: string;
    ready: boolean;
    loaded: boolean;
    version?: string;
    trained?: boolean;
    backend: string;
    since?: string;
    error?: string;
}
/** HANDS capabilities (GET /models/hands/models). */
export interface HandsModels {
    name: string;
    role: string;
    backend: string;
    version: string;
    capabilities: string[];
    endpoints: string[];
}
/** HANDS checkpoint info (GET /models/hands/weights); 503 until trained. */
export interface HandsWeights {
    name: string;
    version: string;
    backend: string;
    checkpoint: string;
    progress: number;
}
/** HANDS generation (POST /models/hands/{generate,plan,act}). */
export interface HandsGenerateResponse {
    text: string;
    stage?: string;
    frame_hash?: string;
}
export interface ImageRef {
    url?: string;
    hash?: string;
    kind?: string;
}
export interface HeadCapabilities {
    vision: boolean;
    text_mirror: boolean;
    embed: boolean;
}
export interface HeadStatus {
    backend: string;
    ready: boolean;
    capabilities: HeadCapabilities;
    detail?: string;
}
export interface HeadGenerateRequest {
    prompt: string;
    perceptual_context?: string;
    images?: ImageRef[];
    frame?: string;
}
export interface HeadGenerateResponse {
    text: string;
    model: string;
    version?: string;
    refuted?: boolean;
    notes?: string[];
}
export interface VisionRequest {
    operation: "embed" | "describe" | "ground" | "visual-reason";
    text?: string;
    images?: ImageRef[];
    context?: string;
}
export interface VisionResponse {
    operation: string;
    model?: string;
    version?: string;
    result?: Record<string, unknown>;
    refuted?: boolean;
    notes?: string[];
}
export interface IndexSearchRequest {
    text: string;
    subject?: string;
    property?: string;
    value?: string;
    k?: number;
    mode?: "index" | "flash" | "hybrid";
    include_quarantine?: boolean;
    /** Forwarded as X-KNIRV-Tenant, never serialized into the request body. */
    tenant?: string;
}
export interface IndexSearchHit {
    span_id: string;
    snapshot: string;
    start: number;
    end: number;
    text: string;
    source_tier: string;
    title?: string;
    url?: string;
    license?: string;
    fetched_at?: string;
    score: number;
    matched?: string[];
    fact?: boolean;
    source?: string;
    revision?: string;
}
export interface IndexSearchResult {
    hits: IndexSearchHit[];
    retrieval: string;
    index_version: string;
    embedder_id: string;
    flash_used: boolean;
    outbound_queries?: string[];
    degraded: boolean;
    stale: boolean;
}
export interface OrganizationOntology {
    trade_secrets?: string[];
    business_logic?: string[];
    user_data?: string[];
    rules?: string[];
    regulations?: string[];
    procedures?: string[];
    policies?: string[];
    faqs?: string[];
    customer_service?: string[];
    industry_jargon?: Record<string, string>;
    stakeholder_hierarchy?: Record<string, unknown>;
    decision_rights?: Record<string, unknown>;
    /** Tenant-approved alternate entity/property spelling -> canonical spelling. */
    aliases?: Record<string, string>;
}
export interface OntologyConfig {
    id?: string;
    organization_id: string;
    name?: string;
    ontology: OrganizationOntology;
}
export interface OntologySyncResult {
    tenant: string;
    ontology_version: string;
    previous_version?: string;
    unchanged: boolean;
    passages: number;
    facts: number;
    not_indexed: string[];
    segment?: string;
}
export interface OntologyClaim {
    text?: string;
    subject: string;
    property: string;
    value: string;
    /** Forwarded as X-KNIRV-Tenant, never serialized into the request body. */
    tenant?: string;
}
export interface OntologyVerdict {
    outcome: "no_opinion" | "supported" | "refuted" | "uncertain";
    reason: string;
    ontology_version?: string;
    ontology_values?: string[];
    hits?: IndexSearchHit[];
}
export interface OntologyRemoveResult {
    tenant: string;
    status: string;
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
/** Decision envelope: the output of the decision core, or a caller-supplied decision. */
export interface DecisionEnvelope {
    decisions: Decision[];
    abstained?: boolean;
    reason?: string;
}
/** Decision-aware HEART generate request. Plain generation (no decision and
 *  with_decision false) never consults the decision core, so legacy clients keep
 *  working when the core is unavailable. The wire field is `with_decision` to
 *  match the Rust core / Go server contract. */
export interface HeartGenerateRequest {
    prompt: string;
    decision?: DecisionEnvelope;
    with_decision?: boolean;
}
/** Decision-aware HEART generate response; always carries the (possibly absent
 *  or abstaining) decision metadata alongside the text. */
export interface HeartGenerateResponse {
    text?: string;
    ready: boolean;
    progress: number;
    decision?: DecisionEnvelope;
    abstained?: boolean;
    error?: string;
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
    readonly SEP: 258;
    readonly OPT_OPEN: 260;
    readonly OPT_END: 261;
    readonly VOCAB: 262;
};
export declare function findDecision(r: Reasoning, name: string): Decision | undefined;
/** Plan text: bytes up to the first EOS, specials dropped. */
export declare function planText(r: Reasoning): string;
/** Tokens the model takes: its sequence length minus the summary slots. */
export declare function inputWidth(m: Manifest): number;
/** BOS + the text's UTF-8 bytes, truncated or PAD-filled to `width`. */
export declare function encodeByteV1(text: string, width: number): number[];
/**
 * Leakage-safe choice-set HEAD input after the model-owned summary slot:
 * BOS question SEP state SEP (OPT_OPEN choice OPT_END)*.
 */
export declare function encodeChoiceSetV1(question: string, state: string, choices: readonly string[], width: number): number[];
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
/** Canonical model input; the trainer's `prompt` is pinned to the same vector. */
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
    handsStatus(): Promise<HandsStatus>;
    handsHealth(): Promise<HandsStatus>;
    handsModels(): Promise<HandsModels>;
    /** 503 -> NotReadyError until HANDS has a checkpoint. */
    handsWeights(): Promise<HandsWeights>;
    handsGenerate(prompt: string, opts?: HandsRequest): Promise<HandsGenerateResponse>;
    handsPlan(prompt: string, opts?: HandsRequest): Promise<HandsGenerateResponse>;
    handsAct(prompt: string, opts?: HandsRequest): Promise<HandsGenerateResponse>;
    headStatus(): Promise<HeadStatus>;
    headCapabilities(): Promise<HeadCapabilities>;
    headGenerate(req: HeadGenerateRequest): Promise<HeadGenerateResponse>;
    headVision(req: VisionRequest): Promise<VisionResponse>;
    private post;
    /** HEART text; NotReadyError until HEART has a trained generator. */
    heartGenerate(prompt: string): Promise<string>;
    /** Decision-aware HEART generate. When resolve is true HEART consults the
     *  decision core; otherwise it degrades to plain generation and never fails
     *  just because the core is unavailable. Returns the full response. */
    heartGenerateWithDecision(prompt: string, resolve: boolean, decision?: DecisionEnvelope): Promise<HeartGenerateResponse>;
    hasherValidate(req: HasherValidateRequest): Promise<HasherValidateResult>;
    hasherInfer(text: string): Promise<HasherInferResult>;
    hasherIndexSearch(req: IndexSearchRequest): Promise<IndexSearchResult>;
    /** Sync is operator-only; authority is sent by the client, never in ontology content. */
    hasherOntologySync(config: OntologyConfig): Promise<OntologySyncResult>;
    hasherOntologyRemove(tenant: string): Promise<OntologyRemoveResult>;
    hasherOntologyCheck(claim: OntologyClaim): Promise<OntologyVerdict>;
    /** Verify a text or claims against the evidence index. A degraded answer (judge down) is a normal result. */
    hasherVerify(req: VerifyRequest): Promise<VerifyResponse>;
    /** Read a deferred job; only the tenant that submitted it can. */
    hasherVerifyJob(id: string, tenant?: string): Promise<VerifyResponse>;
    private postWithHeaders;
    private requestWithHeaders;
    /** HEAD VLM manifest (vision / mirrored text). Narrow, VLM-specific contract. */
    manifest(): Promise<Manifest>;
    /** HEART decision core manifest; the manifest Advise needs. */
    decisionManifest(): Promise<Manifest>;
    reason(tokens: number[]): Promise<Reasoning>;
    /** decisionManifest → encode → reason → advice; every failure is an abstention. */
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
/** The llama wrapper's role-frame check. `unframed` must stay 0 in warn mode before enforce is switched on. */
export interface FrameStatus {
    mode: "off" | "warn" | "enforce";
    constitution_version: string;
    framed: number;
    unframed: number;
    invalid: number;
    rejected: number;
    teacher: number;
}
/** The wrapper's admission queue by priority class (P0..P3). */
export interface QueueStatus {
    free_slots: number;
    depth: [number, number, number, number];
    admitted: [number, number, number, number];
}
/** The embedding server KNIRVMODELS provisions and supervises (Task 3.2). No path, no secret. */
export interface OllamaStatus {
    enabled: boolean;
    endpoint: string;
    model: string;
    digest?: string;
    pinned: boolean;
    version?: string;
    phase: "idle" | "disabled" | "adopting" | "downloading" | "starting" | "pulling" | "ready" | "degraded" | "failed" | "stopped";
    ready: boolean;
    adopted: boolean;
    bind?: string;
    error?: string;
}
/** The capability manifest the daemon serves: version, counts and a hash over every frame hash. */
export interface ManifestStatus {
    constitution_version: string;
    stages: number;
    tools: number;
    hash: string;
    error?: string;
}
export interface LlamaStatus {
    frames?: FrameStatus;
    queue?: QueueStatus;
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
    ollama?: OllamaStatus;
    manifest?: ManifestStatus;
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
/** The fixed list of role lines for a status document (core/src/models.rs). */
export declare function roles(s: PlatformStatus): RoleLine[];
export * from "./frame.js";
/** GET /models/hasher/index/status: what the evidence index holds (twin of Go IndexStatus). */
export interface IndexStatus {
    index_version: string;
    key_version: string;
    embedder_ids: string[];
    retrieval: "lexical" | "hybrid";
    segments: number;
    spans: number;
    tiers: Record<string, number>;
    chunk_words: number;
    searches: number;
    hit_rate: number;
    last_batch?: string;
    last_ingest?: string;
    stale_seconds?: number;
}
export declare const INDEX_STATUS_PATH = "/models/hasher/index/status";
/** POST /models/hasher/verify (Task 3.4); twin of Go VerifyRequest. */
export declare const VERIFY_PATH = "/models/hasher/verify";
export declare const VERIFY_JOBS_PATH = "/models/hasher/verify/jobs/";
export interface VerifyClaimIn {
    text: string;
    subject?: string;
    property?: string;
    value?: string;
}
export interface VerifyRequest {
    text?: string;
    claims?: VerifyClaimIn[];
    /** flash and hybrid are accepted but answered from the index until live lookup is enabled. */
    mode?: "index" | "flash" | "hybrid";
    /** deferred (the default) returns at once with a job id; inline waits up to budget_ms. */
    delivery?: "inline" | "deferred";
    budget_ms?: number;
    domains?: string[];
    max_claims?: number;
    /** Forwarded as X-KNIRV-Tenant, never serialized into the request body. */
    tenant?: string;
}
export interface VerifyEvidence {
    span_id: string;
    snapshot: string;
    source_tier: string;
    text: string;
    title?: string;
    url?: string;
    fetched_at?: string;
    license?: string;
    /** Present only when the span is included in a witnessed batch root (Task 3.9). */
    witness?: {
        batch_root: string;
        proof: string;
        attestation: string;
    };
}
export type GateName = "refused" | "discouraged" | "uncommitted" | "seen" | "attested" | "accepted" | "committed";
export interface VerifyClaim {
    text: string;
    subject?: string;
    property?: string;
    value?: string;
    type: string;
    verdict: "SUPPORTED" | "REFUTED" | "NEI";
    source: "ontology" | "structured" | "judge" | "none";
    confidence: number;
    calibrated: number;
    gate: GateName;
    gate_level: number;
    evidence: VerifyEvidence[];
    cited?: string[];
    agreement: boolean;
    degraded?: boolean;
    reason?: string;
    cached?: boolean;
}
export interface VerifyResponse {
    status: "done" | "pending";
    job_id?: string;
    gate: GateName;
    gate_level: number;
    gate_score: number;
    verdict: "supported" | "refuted" | "mixed" | "nei" | "unchecked";
    confidence: number;
    claims: VerifyClaim[];
    unchecked?: {
        text: string;
        why?: string;
    }[];
    unchecked_fraction: number;
    degraded: boolean;
    chargeable: boolean;
    calibrated: boolean;
    unverified?: boolean;
    index_version: string;
    embedder_id: string;
    judge?: {
        model: string;
        sha256?: string;
        adapters?: string;
    };
    mode: string;
    delivery: string;
    domains?: string[];
    notes?: string[];
    /** True when live providers were consulted; outbound_queries is exactly what left the machine. */
    flash_used?: boolean;
    outbound_queries?: string[];
    refused: boolean;
    is_accurate: boolean;
    citations: number[];
    text?: string;
}
export declare const GATE_NAMES: readonly GateName[];
/** The first way a response breaks the contract, or null. The invariants live in the SDK core (models.verify_check); twin of Go VerifyResponse.Check. */
export declare function checkVerify(r: VerifyResponse): string | null;
