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
export const HASHER_PREFIX = "/models/hasher";

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

  private async json<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await this.fetchImpl(this.baseUrl + path, { ...init, headers: { ...this.headers, ...(init?.headers ?? {}) } });
    if (res.status === 503) throw new NoModelError();
    if (!res.ok) throw new Error(`head: HTTP ${res.status}`);
    return (await res.json()) as T;
  }

  registry(): Promise<ModelStatus[]> { return this.json<ModelStatus[]>(REGISTRY_PATH); }
  health(): Promise<Health> { return this.json<Health>(HEALTH_PATH); }
  heartHealth(): Promise<HeartHealth> { return this.json<HeartHealth>(`${HEART_PREFIX}/health`); }
  hasherStatus(): Promise<HasherStatus> { return this.json<HasherStatus>(`${HASHER_PREFIX}/status`); }

  private async post<T>(path: string, body: unknown): Promise<T> {
    const res = await this.fetchImpl(this.baseUrl + path, {
      method: "POST", headers: { ...this.headers, "Content-Type": "application/json" }, body: JSON.stringify(body),
    });
    if (res.status === 503) throw new NotReadyError();
    if (!res.ok) throw new Error(`models: HTTP ${res.status}`);
    return (await res.json()) as T;
  }

  /** HEART text; NotReadyError until HEART has a trained generator. */
  async heartGenerate(prompt: string): Promise<string> {
    const out = await this.post<{ text?: string; error?: string }>(`${HEART_PREFIX}/generate`, { prompt });
    if (out.error) throw new Error(`heart: ${out.error}`);
    return out.text ?? "";
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

  private async postWithHeaders<T>(path: string, body: unknown, headers: Record<string, string>): Promise<T> {
    const res = await this.fetchImpl(this.baseUrl + path, {
      method: "POST", headers: { ...this.headers, ...headers, "Content-Type": "application/json" }, body: JSON.stringify(body),
    });
    if (res.status === 503) throw new NotReadyError();
    if (!res.ok) throw new Error(`models: HTTP ${res.status}`);
    return (await res.json()) as T;
  }

  private async requestWithHeaders<T>(path: string, method: string, body: unknown, headers: Record<string, string>): Promise<T> {
    const res = await this.fetchImpl(this.baseUrl + path, {
      method, headers: { ...this.headers, ...headers, ...(body === undefined ? {} : { "Content-Type": "application/json" }) },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    if (res.status === 503) throw new NotReadyError();
    if (!res.ok) throw new Error(`models: HTTP ${res.status}`);
    return (await res.json()) as T;
  }

  manifest(): Promise<Manifest> { return this.json<Manifest>(`${HEAD_PREFIX}/models`); }

  reason(tokens: number[]): Promise<Reasoning> {
    return this.json<Reasoning>(`${HEAD_PREFIX}/reason`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tokens }),
    });
  }

  /** manifest → encode → reason → advice; every failure is an abstention. */
  async advise(c: AdviseContext): Promise<SupervisorAdvice> {
    await initLocal();
    let m: Manifest;
    try { m = await this.manifest(); } catch { return abstainAdvice("no HEAD model available", "none", ""); }
    let width: number;
    try { width = inputWidth(m); } catch { return abstainAdvice("no HEAD model available", "none", ""); }
    try {
      return adviceFromReasoning(await this.reason(encodeByteV1(buildAdvisePrompt(c), width)), c);
    } catch (e) {
      return abstainAdvice(e instanceof NoModelError ? "no HEAD model available" : "HEAD could not answer", "none", "");
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
export interface LlamaStatus { frames?: FrameStatus; queue?: QueueStatus; ready: boolean; base_model?: string; base_sha256?: string; adapters?: LlamaAdapter[]; error?: string }
/** One harness decision, without any prompt or output text. */
export interface RoutingEvent {
  id: string; time: string; kind: string; skill?: string; chosen?: string; model_version?: string; adapter?: string;
  degraded?: boolean; fallbacks?: string[]; reason?: string; notes?: string[]; latency_ms: number; error?: string;
}
export interface RouteStats { calls: number; failures: number; p50_ms: number; p95_ms: number; circuit: string; last_error?: string }
export interface PlatformStatus {
  time: string; daemon: ModelStatus[]; daemon_error?: string; llama?: LlamaStatus; routing: RoutingEvent[]; stats: Record<string, RouteStats>;
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
  searches: number; hit_rate: number; last_batch?: string; last_ingest?: string;
}
export const INDEX_STATUS_PATH = `${HASHER_PREFIX}/index/status`;
