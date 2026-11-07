/**
 * Wire contract for the KNIRVMODELS daemon. The logic (byte-v1 tokenizer, advice
 * prompt and redaction, advice mapping, role lines) lives in the Rust core
 * (core/src/models.rs) and is reached through `models.*` operations, so this file
 * holds types, the HTTP client and thin calls. The core must be initialised first
 * (`initLocal()` or `SdkClient.init()`); `ModelsClient.advise` does it itself.
 * testvectors/models gate the core and the still-separate Go copy.
 */
import { callLocal, fromBase64, initLocal } from "../local.js";


/** Where the daemon serves each model. One namespace for all three. */
export const HEAD_PREFIX = "/models/head";
export const REGISTRY_PATH = "/models/registry";
export const HEALTH_PATH = "/models/health";
export const HEART_PREFIX = "/models/heart";
export const HEART_DECISION_PREFIX = "/models/heart/decide";
export const HASHER_PREFIX = "/models/hasher";

/**
 * HANDS (server_tasks) namespace. Typed transport mirror of the Go SDK's
 * `sdk.HandsPrefix` and `sdk.RoleHands`; the Rust core (core/src/models.rs) is
 * the source of truth, so these are wire-value aliases plus thin HTTP calls.
 */
export const HandsPrefix = "/models/hands";
export const RoleHands = "server_tasks";

// Role keys are emitted by the Rust-core contract (core/src/models.rs roles());
// a HANDS ModelStatus carries RoleHands ("server_tasks"). Add to this union
// whenever the core adds a role, or the parity type test below fails to compile.
export type ModelRole =
  | "decide"
  | "generate"
  | "validate"
  | "vision"
  | "server_tasks"
  | "external"
  | "validator"
  | "reasoner";

// Parity type test: this assignment only compiles while "server_tasks" is a
// member of ModelRole, i.e. while the core contract still reports the HANDS
// server_tasks role. If the core drops it, the build fails here first.
export function serverTasksRole(): ModelRole {
  return RoleHands;
}

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

export interface Health { status: string; time: string; models: number }
export interface HeartHealth { available: boolean; status: string; loaded: boolean; ready: boolean; progress: number }
export interface HasherValidateRequest { prediction: number; domain: string; context?: Record<string, unknown> }
export interface HasherValidateResult {
  prediction: number; valid: boolean; domain: string; rules_applied: number; errors?: string[]; latency_ms: number;
}
export interface HasherInferResult {
  prediction: number; confidence: number; mode: string; valid_passes: number; total_passes: number; latency_ms: number;
}
export interface HasherStatus {
  loaded: boolean; mode?: string; hardware_tier?: string; using_hardware?: boolean;
  total_inferences: number; hardware_inferences: number; software_fallbacks: number;
}

// ---- HANDS (server_tasks) ----------------------------------------------------
// Transport-only mirror of the KNIRVMODELS HANDS facade (pkg/models/hands.go) and
// the Go SDK's sdk.HandsPrefix / sdk.RoleHands. Core (core/src/models.rs) is the
// source of truth; these types describe the wire shapes, not contract logic.

export interface HandsRequest { n?: number; temperature?: number; topK?: number }

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
export interface HandsWeights { name: string; version: string; backend: string; checkpoint: string; progress: number }
/** HANDS generation (POST /models/hands/{generate,plan,act}). */
export interface HandsGenerateResponse { text: string; stage?: string; frame_hash?: string }

// ---- HEAD VLM + mirrored text generator ---------------------------------------
// Transport mirror of the Go SDK's head_vlm.go and the KNIRVMODELS Head facade
// (pkg/models/head.go). The canonical wire contract lives in knirv-sdk
// core/data/vlm_contract.json; these types and the methods below are bindings only.
export interface ImageRef { url?: string; hash?: string; kind?: string }
export interface HeadCapabilities { vision: boolean; text_mirror: boolean; embed: boolean }
export interface HeadStatus {
  backend: string; ready: boolean; capabilities: HeadCapabilities; detail?: string;
}
export interface HeadGenerateRequest {
  prompt: string; perceptual_context?: string; images?: ImageRef[]; frame?: string;
}
export interface HeadGenerateResponse {
  text: string; model: string; version?: string; refuted?: boolean; notes?: string[];
}
export interface VisionRequest {
  operation: "embed" | "describe" | "ground" | "visual-reason";
  text?: string; images?: ImageRef[]; context?: string;
}
export interface VisionResponse {
  operation: string; model?: string; version?: string; result?: Record<string, unknown>;
  refuted?: boolean; notes?: string[];
}
export interface IndexSearchRequest {
  text: string; subject?: string; property?: string; value?: string;
  k?: number; mode?: "index" | "flash" | "hybrid"; include_quarantine?: boolean;
  /** Forwarded as X-KNIRV-Tenant, never serialized into the request body. */
  tenant?: string;
}
export interface IndexSearchHit {
  span_id: string; snapshot: string; start: number; end: number; text: string;
  source_tier: string; title?: string; url?: string; license?: string; fetched_at?: string;
  score: number; matched?: string[]; fact?: boolean; source?: string; revision?: string;
}
export interface IndexSearchResult {
  hits: IndexSearchHit[]; retrieval: string; index_version: string; embedder_id: string;
  flash_used: boolean; outbound_queries?: string[]; degraded: boolean; stale: boolean;
}
export interface OrganizationOntology {
  trade_secrets?: string[]; business_logic?: string[]; user_data?: string[];
  rules?: string[]; regulations?: string[]; procedures?: string[]; policies?: string[];
  faqs?: string[]; customer_service?: string[]; industry_jargon?: Record<string, string>;
  stakeholder_hierarchy?: Record<string, unknown>; decision_rights?: Record<string, unknown>;
	/** Tenant-approved alternate entity/property spelling -> canonical spelling. */
  aliases?: Record<string, string>;
}
export interface OntologyConfig {
  id?: string; organization_id: string; name?: string; ontology: OrganizationOntology;
}
export interface OntologySyncResult {
  tenant: string; ontology_version: string; previous_version?: string; unchanged: boolean;
  passages: number; facts: number; not_indexed: string[]; segment?: string;
}
export interface OntologyClaim {
  text?: string; subject: string; property: string; value: string;
  /** Forwarded as X-KNIRV-Tenant, never serialized into the request body. */
  tenant?: string;
}
export interface OntologyVerdict {
  outcome: "no_opinion" | "supported" | "refuted" | "uncertain";
  reason: string; ontology_version?: string; ontology_values?: string[]; hits?: IndexSearchHit[];
}
export interface OntologyRemoveResult { tenant: string; status: string }

/** The model is up but cannot answer yet (HTTP 503). */
export class NotReadyError extends Error {
  constructor() { super("model not ready"); this.name = "NotReadyError"; }
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
  /** Per-candidate probabilities of a choice-set (knirv-heart/v2) answer, in request order. */
  candidates?: { id: string; text: string; probability: number }[];
}

/** One choice-set candidate with its probability (knirv-heart/v2). */
export interface Candidate {
  id: string;
  text: string;
  probability: number;
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

/** One choice-set question for a knirv-heart/v2 model. `choices` may be omitted
 *  for null-kind set heads (the model supplies yes/no). */
export interface SetQuestion {
  name: string;
  question?: string;
  state?: string;
  choices?: { id: string; text: string }[];
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
  config: { max_seq_len?: number; summary_slots?: number; dataset_manifest_hash?: string; [k: string]: unknown };
  sha256: string;
  size: number;
  signature?: string;
}

export const BYTE_V1 = { EOS: 259, PAD: 256, BOS: 257, SEP: 258, OPT_OPEN: 260, OPT_END: 261, VOCAB: 262 } as const; // token ids; the tokenizer itself is in the core

export function findDecision(r: Reasoning, name: string): Decision | undefined {
  return r.decisions.find((d) => d.name === name);
}

/** Plan text: bytes up to the first EOS, specials dropped. */
export function planText(r: Reasoning): string {
  const { bytes_base64 } = callLocal<{ bytes_base64: string }>("models.plan_text", { plan_tokens: r.plan_tokens });
  return new TextDecoder().decode(fromBase64(bytes_base64));
}

/** Tokens the model takes: its sequence length minus the summary slots. */
export function inputWidth(m: Manifest): number {
  return callLocal<{ width: number }>("models.input_width", { config: m.config }).width;
}

/** BOS + the text's UTF-8 bytes, truncated or PAD-filled to `width`. */
export function encodeByteV1(text: string, width: number): number[] {
  return callLocal<{ tokens: number[] }>("models.encode_byte_v1", { text, width }).tokens;
}

/**
 * Leakage-safe choice-set HEAD input after the model-owned summary slot:
 * BOS question SEP state SEP (OPT_OPEN choice OPT_END)*.
 */
export function encodeChoiceSetV1(question: string, state: string, choices: readonly string[], width: number): number[] {
	return callLocal<{ tokens: number[] }>("models.encode_choice_set_v1", { question, state, choices: [...choices], width }).tokens;
}

export interface AdviseContext {
  current_phase?: string;
  previous_phase?: string;
  remaining_budget?: Record<string, number>;
  loop_iterations?: Record<string, number>;
  agent_chain?: string[];
  policy_bundle_hash?: string;
  evidence_summary?: { class: string }[];
  task_summary?: { request?: string; intent?: string; risk?: string };
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

const now = () => new Date().toISOString();

/** Canonical model input; the trainer's `prompt` is pinned to the same vector. */
export function buildAdvisePrompt(c: AdviseContext): string {
  return callLocal<{ prompt: string }>("models.build_advise_prompt", c).prompt;
}

export function abstainAdvice(reason: string, modelVersion: string, manifest: string): SupervisorAdvice {
  return callLocal<SupervisorAdvice>("models.abstain_advice", { reason, model_version: modelVersion, dataset_manifest_hash: manifest, now: now() });
}

/** Maps decision heads onto SupervisorAdvice; every doubtful case abstains. */
export function adviceFromReasoning(r: Reasoning, c: AdviseContext): SupervisorAdvice {
  return callLocal<SupervisorAdvice>("models.advice_from_reasoning", { reasoning: r, context: c, now: now() });
}

export class NoModelError extends Error {
  constructor() { super("no HEAD model has been published yet"); }
}

/** Client for the daemon's HEAD routes; `baseUrl` is the backend or daemon origin. */
export class ModelsClient {
  constructor(private baseUrl: string, private fetchImpl: typeof fetch = fetch, private headers: Record<string, string> = {}) {}

  /** Reads a non-2xx, non-503 response body so transport errors surface the
   *  server's contract text instead of a bare status — e.g. the Rust core's
   *  "no choice-set heads" rejection of a v1 /sets choice-set request. Reads the
   *  body once (as text), then optionally JSON-parses it to prefer the `error`
   *  field; falls back to the raw text so a non-JSON body is not lost. */
  private async errorDetail(res: Response): Promise<string> {
    let text: string;
    try {
      text = await res.text();
    } catch { return ""; }
    if (!text || text.length === 0) return "";
    try {
      const body = JSON.parse(text) as { error?: unknown };
      if (body && typeof body.error === "string" && body.error.length > 0) return body.error;
    } catch { /* body was not JSON; fall through to raw text */ }
    return text.slice(0, 240);
  }

  private async json<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await this.fetchImpl(this.baseUrl + path, { ...init, headers: { ...this.headers, ...(init?.headers ?? {}) } });
    if (res.status === 503) throw new NoModelError();
    if (!res.ok) {
      const detail = await this.errorDetail(res);
      throw new Error(detail ? `head: HTTP ${res.status}: ${detail}` : `head: HTTP ${res.status}`);
    }
    return (await res.json()) as T;
  }

  registry(): Promise<ModelStatus[]> { return this.json<ModelStatus[]>(REGISTRY_PATH); }
  health(): Promise<Health> { return this.json<Health>(HEALTH_PATH); }
  heartHealth(): Promise<HeartHealth> { return this.json<HeartHealth>(`${HEART_PREFIX}/health`); }
  hasherStatus(): Promise<HasherStatus> { return this.json<HasherStatus>(`${HASHER_PREFIX}/status`); }

  // ---- HANDS (server_tasks) -------------------------------------------------
  // Thin transport mirror of the Go SDK's HandsPrefix/RoleHands and the
  // KNIRVMODELS HANDS facade (pkg/models/hands.go). Untrained HANDS returns 503
  // (NotReadyError) for generate/plan/act/weights; status/health/models stay 200.

  handsStatus(): Promise<HandsStatus> { return this.json<HandsStatus>(`${HandsPrefix}/status`); }
  handsHealth(): Promise<HandsStatus> { return this.json<HandsStatus>(`${HandsPrefix}/health`); }
  handsModels(): Promise<HandsModels> { return this.json<HandsModels>(`${HandsPrefix}/models`); }
  /** 503 -> NotReadyError until HANDS has a checkpoint. */
  handsWeights(): Promise<HandsWeights> { return this.requestWithHeaders<HandsWeights>(`${HandsPrefix}/weights`, "GET", undefined, {}); }
  handsGenerate(prompt: string, opts: HandsRequest = {}): Promise<HandsGenerateResponse> {
    return this.post<HandsGenerateResponse>(`${HandsPrefix}/generate`, { prompt, ...opts });
  }
  handsPlan(prompt: string, opts: HandsRequest = {}): Promise<HandsGenerateResponse> {
    return this.post<HandsGenerateResponse>(`${HandsPrefix}/plan`, { prompt, ...opts });
  }
  handsAct(prompt: string, opts: HandsRequest = {}): Promise<HandsGenerateResponse> {
    return this.post<HandsGenerateResponse>(`${HandsPrefix}/act`, { prompt, ...opts });
  }

  // ---- HEAD VLM (vision / multimodal) ----------------------------------------
  // Transport mirror of the Go SDK HeadVision / HeadGenerate / HeadStatus. The four
  // vision operations each reach their own route; an unknown operation is rejected
  // client-side before any request leaves the process.
  headStatus(): Promise<HeadStatus> { return this.json<HeadStatus>(`${HEAD_PREFIX}/status`); }
  headCapabilities(): Promise<HeadCapabilities> { return this.json<HeadCapabilities>(`${HEAD_PREFIX}/capabilities`); }
  headGenerate(req: HeadGenerateRequest): Promise<HeadGenerateResponse> {
    return this.post<HeadGenerateResponse>(`${HEAD_PREFIX}/generate`, req);
  }
  headVision(req: VisionRequest): Promise<VisionResponse> {
    if (!["embed", "describe", "ground", "visual-reason"].includes(req.operation)) {
      return Promise.reject(new Error(`head: unsupported vision operation ${JSON.stringify(req.operation)} (want one of embed, describe, ground, visual-reason)`));
    }
    return this.post<VisionResponse>(`${HEAD_PREFIX}/${req.operation}`, req);
  }

  private async post<T>(path: string, body: unknown): Promise<T> {
    const res = await this.fetchImpl(this.baseUrl + path, {
      method: "POST", headers: { ...this.headers, "Content-Type": "application/json" }, body: JSON.stringify(body),
    });
    if (res.status === 503) throw new NotReadyError();
    if (!res.ok) {
      const detail = await this.errorDetail(res);
      throw new Error(detail ? `models: HTTP ${res.status}: ${detail}` : `models: HTTP ${res.status}`);
    }
    return (await res.json()) as T;
  }

  /** HEART text; NotReadyError until HEART has a trained generator. */
  async heartGenerate(prompt: string): Promise<string> {
    const out = await this.post<{ text?: string; error?: string }>(`${HEART_PREFIX}/generate`, { prompt });
    if (out.error) throw new Error(`heart: ${out.error}`);
    return out.text ?? "";
  }

  /** Decision-aware HEART generate. When resolve is true HEART consults the
   *  decision core; otherwise it degrades to plain generation and never fails
   *  just because the core is unavailable. Returns the full response. */
  async heartGenerateWithDecision(prompt: string, resolve: boolean, decision?: DecisionEnvelope): Promise<HeartGenerateResponse> {
    const body: HeartGenerateRequest = { prompt };
    if (resolve) body.with_decision = true;
    if (decision) body.decision = decision;
    const out = await this.post<HeartGenerateResponse>(`${HEART_PREFIX}/generate`, body);
    if (out.error) throw new Error(`heart: ${out.error}`);
    return out;
  }
  hasherValidate(req: HasherValidateRequest): Promise<HasherValidateResult> { return this.post(`${HASHER_PREFIX}/validate`, req); }
  hasherInfer(text: string): Promise<HasherInferResult> { return this.post(`${HASHER_PREFIX}/infer`, { text }); }
  hasherIndexSearch(req: IndexSearchRequest): Promise<IndexSearchResult> {
    const { tenant, ...body } = req;
    return this.postWithHeaders(`${HASHER_PREFIX}/index/search`, body, tenant ? { "X-KNIRV-Tenant": tenant } : {});
  }
  /** Sync is operator-only; authority is sent by the client, never in ontology content. */
  hasherOntologySync(config: OntologyConfig): Promise<OntologySyncResult> {
    return this.postWithHeaders(`${HASHER_PREFIX}/ontology/sync`, config, { "X-KNIRV-Privilege": "operator" });
  }
  hasherOntologyRemove(tenant: string): Promise<OntologyRemoveResult> {
    return this.requestWithHeaders(`${HASHER_PREFIX}/ontology?tenant=${encodeURIComponent(tenant)}`, "DELETE", undefined, { "X-KNIRV-Privilege": "operator" });
  }
  hasherOntologyCheck(claim: OntologyClaim): Promise<OntologyVerdict> {
    const { tenant, ...body } = claim;
    return this.postWithHeaders(`${HASHER_PREFIX}/ontology/check`, body, tenant ? { "X-KNIRV-Tenant": tenant } : {});
  }
  /** Verify a text or claims against the evidence index. A degraded answer (judge down) is a normal result. */
  hasherVerify(req: VerifyRequest): Promise<VerifyResponse> {
    const bad = callLocal<{ error: string | null }>("models.verify_request_check", { request: req }).error;
    if (bad) return Promise.reject(new Error(bad));
    const { tenant, ...body } = req;
    return this.postWithHeaders(VERIFY_PATH, body, tenant ? { "X-KNIRV-Tenant": tenant } : {});
  }
  /** Read a deferred job; only the tenant that submitted it can. */
  hasherVerifyJob(id: string, tenant?: string): Promise<VerifyResponse> {
    return this.requestWithHeaders(VERIFY_JOBS_PATH + encodeURIComponent(id), "GET", undefined, tenant ? { "X-KNIRV-Tenant": tenant } : {});
  }

  private async postWithHeaders<T>(path: string, body: unknown, headers: Record<string, string>): Promise<T> {
    const res = await this.fetchImpl(this.baseUrl + path, {
      method: "POST", headers: { ...this.headers, ...headers, "Content-Type": "application/json" }, body: JSON.stringify(body),
    });
    if (res.status === 503) throw new NotReadyError();
    if (!res.ok) {
      const detail = await this.errorDetail(res);
      throw new Error(detail ? `models: HTTP ${res.status}: ${detail}` : `models: HTTP ${res.status}`);
    }
    return (await res.json()) as T;
  }

  private async requestWithHeaders<T>(path: string, method: string, body: unknown, headers: Record<string, string>): Promise<T> {
    const res = await this.fetchImpl(this.baseUrl + path, {
      method, headers: { ...this.headers, ...headers, ...(body === undefined ? {} : { "Content-Type": "application/json" }) },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    if (res.status === 503) throw new NotReadyError();
    if (!res.ok) {
      const detail = await this.errorDetail(res);
      throw new Error(detail ? `models: HTTP ${res.status}: ${detail}` : `models: HTTP ${res.status}`);
    }
    return (await res.json()) as T;
  }

  /** HEAD VLM manifest (vision / mirrored text). Narrow, VLM-specific contract. */
  manifest(): Promise<Manifest> { return this.json<Manifest>(`${HEAD_PREFIX}/models`); }

  /** HEART decision core manifest; the manifest Advise needs. */
  decisionManifest(): Promise<Manifest> { return this.json<Manifest>(`${HEART_DECISION_PREFIX}/models`); }

  reason(tokens: number[]): Promise<Reasoning> {
    return this.json<Reasoning>(`${HEART_DECISION_PREFIX}/reason`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tokens }),
    });
  }

  /** Choice-set questions (knirv-heart/v2 models). A v1 model answers with a
   *  "no choice-set heads" error, rejecting the request just like every other runtime. */
  reasonSets(questions: SetQuestion[]): Promise<Reasoning> {
    return this.json<Reasoning>(`${HEART_DECISION_PREFIX}/sets`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ questions }),
    });
  }

  /** decisionManifest → encode → reason → advice; every failure is an abstention. */
  async advise(c: AdviseContext): Promise<SupervisorAdvice> {
    await initLocal();
    let m: Manifest;
    try { m = await this.decisionManifest(); } catch { return abstainAdvice("no decision model available", "none", ""); }
    let width: number;
    try { width = inputWidth(m); } catch { return abstainAdvice("no decision model available", "none", ""); }
    try {
      return adviceFromReasoning(await this.reason(encodeByteV1(buildAdvisePrompt(c), width)), c);
    } catch (e) {
      return abstainAdvice(e instanceof NoModelError ? "no decision model available" : "decision model could not answer", "none", "");
    }
  }
}

// ---- platform status (GET /api/models/status) -------------------------------------------

/** The backend's authenticated route for the platform-wide model view. */
export const STATUS_PATH = "/api/models/status";

export interface LlamaAdapter { name: string; loaded: boolean; base_sha256?: string; bundle_hash?: string }
/** The embedded llama.cpp. `base_model` is a file name, never a path. */
/** The llama wrapper's role-frame check. `unframed` must stay 0 in warn mode before enforce is switched on. */
export interface FrameStatus { mode: "off" | "warn" | "enforce"; constitution_version: string; framed: number; unframed: number; invalid: number; rejected: number; teacher: number }
/** The wrapper's admission queue by priority class (P0..P3). */
export interface QueueStatus { free_slots: number; depth: [number, number, number, number]; admitted: [number, number, number, number] }
/** The embedding server KNIRVMODELS provisions and supervises (Task 3.2). No path, no secret. */
export interface OllamaStatus {
  enabled: boolean; endpoint: string; model: string; digest?: string; pinned: boolean; version?: string;
  phase: "idle" | "disabled" | "adopting" | "downloading" | "starting" | "pulling" | "ready" | "degraded" | "failed" | "stopped";
  ready: boolean; adopted: boolean; bind?: string; error?: string;
}
/** The capability manifest the daemon serves: version, counts and a hash over every frame hash. */
export interface ManifestStatus { constitution_version: string; stages: number; tools: number; hash: string; error?: string }
export interface LlamaStatus { frames?: FrameStatus; queue?: QueueStatus; ready: boolean; base_model?: string; base_sha256?: string; adapters?: LlamaAdapter[]; error?: string }
/** One harness decision, without any prompt or output text. */
export interface RoutingEvent {
  id: string; time: string; kind: string; skill?: string; chosen?: string; model_version?: string; adapter?: string;
  degraded?: boolean; fallbacks?: string[]; reason?: string; notes?: string[]; latency_ms: number; error?: string;
}
export interface RouteStats { calls: number; failures: number; p50_ms: number; p95_ms: number; circuit: string; last_error?: string }
export interface PlatformStatus {
  time: string; daemon: ModelStatus[]; daemon_error?: string; llama?: LlamaStatus; ollama?: OllamaStatus; manifest?: ManifestStatus; routing: RoutingEvent[]; stats: Record<string, RouteStats>;
}
export type RoleState = "ready" | "idle" | "not ready" | "unavailable";
/** One row of "which model does what": role -> model -> version. */
export interface RoleLine { role: string; label: string; model: string; version: string; state: RoleState; detail?: string }

/** The fixed list of role lines for a status document (core/src/models.rs). */
export function roles(s: PlatformStatus): RoleLine[] {
  return callLocal<{ roles: RoleLine[] }>("models.platform_roles", { status: s }).roles;
}

export * from "./frame.js";

/** GET /models/hasher/index/status: what the evidence index holds (twin of Go IndexStatus). */
export interface IndexStatus {
  index_version: string; key_version: string; embedder_ids: string[]; retrieval: "lexical" | "hybrid";
  segments: number; spans: number; tiers: Record<string, number>; chunk_words: number;
  searches: number; hit_rate: number; last_batch?: string; last_ingest?: string; stale_seconds?: number;
}
export const INDEX_STATUS_PATH = `${HASHER_PREFIX}/index/status`;

/** POST /models/hasher/verify (Task 3.4); twin of Go VerifyRequest. */
export const VERIFY_PATH = `${HASHER_PREFIX}/verify`;
export const VERIFY_JOBS_PATH = `${HASHER_PREFIX}/verify/jobs/`;
export interface VerifyClaimIn { text: string; subject?: string; property?: string; value?: string }
export interface VerifyRequest {
  text?: string; claims?: VerifyClaimIn[];
  /** flash and hybrid are accepted but answered from the index until live lookup is enabled. */
  mode?: "index" | "flash" | "hybrid";
  /** deferred (the default) returns at once with a job id; inline waits up to budget_ms. */
  delivery?: "inline" | "deferred";
  budget_ms?: number; domains?: string[]; max_claims?: number;
  /** Forwarded as X-KNIRV-Tenant, never serialized into the request body. */
  tenant?: string;
}
export interface VerifyEvidence {
  span_id: string; snapshot: string; source_tier: string; text: string; title?: string; url?: string; fetched_at?: string; license?: string;
  /** Present only when the span is included in a witnessed batch root (Task 3.9). */
  witness?: { batch_root: string; proof: string; attestation: string };
}
export type GateName = "refused" | "discouraged" | "uncommitted" | "seen" | "attested" | "accepted" | "committed";
export interface VerifyClaim {
  text: string; subject?: string; property?: string; value?: string; type: string;
  verdict: "SUPPORTED" | "REFUTED" | "NEI"; source: "ontology" | "structured" | "judge" | "none";
  confidence: number; calibrated: number; gate: GateName; gate_level: number;
  evidence: VerifyEvidence[]; cited?: string[]; agreement: boolean; degraded?: boolean; reason?: string; cached?: boolean;
}
export interface VerifyResponse {
  status: "done" | "pending"; job_id?: string;
  gate: GateName; gate_level: number; gate_score: number;
  verdict: "supported" | "refuted" | "mixed" | "nei" | "unchecked"; confidence: number;
  claims: VerifyClaim[]; unchecked?: { text: string; why?: string }[]; unchecked_fraction: number;
  degraded: boolean; chargeable: boolean; calibrated: boolean; unverified?: boolean;
  index_version: string; embedder_id: string; judge?: { model: string; sha256?: string; adapters?: string };
  mode: string; delivery: string; domains?: string[]; notes?: string[];
  /** True when live providers were consulted; outbound_queries is exactly what left the machine. */
  flash_used?: boolean; outbound_queries?: string[];
  refused: boolean; is_accurate: boolean; citations: number[]; text?: string;
}
export const GATE_NAMES: readonly GateName[] = ["refused", "discouraged", "uncommitted", "seen", "attested", "accepted", "committed"];

/** The first way a response breaks the contract, or null. The invariants live in the SDK core (models.verify_check); twin of Go VerifyResponse.Check. */
export function checkVerify(r: VerifyResponse): string | null {
  return callLocal<{ error: string | null }>("models.verify_check", { response: r }).error;
}
