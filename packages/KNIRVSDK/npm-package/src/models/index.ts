/**
 * Wire contract for the KNIRVMODELS daemon (TypeScript twin of
 * go-package/models). Both are tested against testvectors/models so they cannot
 * drift silently.
 */

/** Where the daemon serves each model. One namespace for all three. */
export const HEAD_PREFIX = "/models/head";
/** The previous release's HEAD prefix; the daemon still answers there (removed in P9). */
export const LEGACY_HEAD_PREFIX = "/heart/head";
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

export const BYTE_V1 = { EOS: 259, PAD: 256, BOS: 257 } as const;

export function findDecision(r: Reasoning, name: string): Decision | undefined {
  return r.decisions.find((d) => d.name === name);
}

/** Plan text: bytes up to the first EOS, specials dropped. */
export function planText(r: Reasoning): string {
  const bytes: number[] = [];
  for (const t of r.plan_tokens) {
    if (t === BYTE_V1.EOS) break;
    if (t < 256) bytes.push(t);
  }
  return new TextDecoder().decode(Uint8Array.from(bytes));
}

/** Tokens the model takes: its sequence length minus the summary slots. */
export function inputWidth(m: Manifest): number {
  const max = m.config.max_seq_len ?? 0;
  const slots = m.config.summary_slots ?? 0;
  if (max <= slots) throw new Error("model config has no input room");
  return max - slots;
}

/** BOS + the text's UTF-8 bytes, truncated or PAD-filled to `width`. */
export function encodeByteV1(text: string, width: number): number[] {
  const toks = [BYTE_V1.BOS as number];
  for (const b of new TextEncoder().encode(text)) {
    if (toks.length === width) break;
    toks.push(b);
  }
  while (toks.length < width) toks.push(BYTE_V1.PAD);
  return toks;
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

const ADVICE_ACTIONS = new Set(["allow", "request_review", "request_evidence", "ask_operator", "recover"]);
const SECRETS = [
  /AKIA[0-9A-Z]{16}/g,
  /gh[pousr]_[A-Za-z0-9]{36}/g,
  /(?:password|passwd|token|secret|api[_-]?key)\s*[=:]\s*[^\s]{8,}/gi,
];

/** Canonical model input; identical to the Go BuildAdvisePrompt and the trainer's `prompt`. */
export function buildAdvisePrompt(c: AdviseContext): string {
  let req = c.task_summary?.request ?? "";
  for (const p of SECRETS) req = req.replace(p, "[REDACTED]");
  const kv = (m?: Record<string, number>) =>
    Object.keys(m ?? {}).sort().map((k) => `${k}=${m![k]}`).join(",");
  const classes = (c.evidence_summary ?? []).map((e) => e.class).sort();
  return (
    `phase:${c.current_phase ?? ""} intent:${c.task_summary?.intent ?? ""} risk:${c.task_summary?.risk ?? ""} ` +
    `budget:${kv(c.remaining_budget)} agents:${(c.agent_chain ?? []).join(">")} evidence:${classes.join(",")}\n` +
    `request:${req}`
  );
}

export function abstainAdvice(reason: string, modelVersion: string, manifest: string): SupervisorAdvice {
  return {
    recommended_action: "", next_phase: "", confidence: 0, abstain: true, abstain_reason: reason,
    model_version: modelVersion, dataset_manifest_hash: manifest, policy_bundle_hash: "",
    received_at: new Date().toISOString(),
  };
}

/** Maps decision heads onto SupervisorAdvice; every doubtful case abstains. */
export function adviceFromReasoning(r: Reasoning, c: AdviseContext): SupervisorAdvice {
  const find = (name: string) => r.decisions.find((d) => d.name === name && d.kind === "choice");
  const ab = (reason: string) => abstainAdvice(reason, r.model_version, r.dataset_manifest_hash);

  const action = find("action");
  if (!action) return ab("model has no action head");
  if (action.abstain || action.selected == null) return ab("action confidence below the fitted abstention threshold");
  if (!ADVICE_ACTIONS.has(action.selected)) return ab("action is not in the advisory vocabulary");
  let conf = action.confidence ?? 0;

  let next = c.current_phase ?? "";
  const np = find("next_phase");
  if (np) {
    if (np.abstain || np.selected == null) return ab("next_phase confidence below the fitted abstention threshold");
    next = np.selected;
    if (np.confidence != null && np.confidence < conf) conf = np.confidence;
  }
  return {
    recommended_action: action.selected, next_phase: next, confidence: conf, abstain: false,
    model_version: r.model_version, dataset_manifest_hash: r.dataset_manifest_hash,
    policy_bundle_hash: r.policy_bundle_hash || c.policy_bundle_hash || "",
    received_at: new Date().toISOString(),
  };
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

  manifest(): Promise<Manifest> { return this.json<Manifest>(`${HEAD_PREFIX}/models`); }

  reason(tokens: number[]): Promise<Reasoning> {
    return this.json<Reasoning>(`${HEAD_PREFIX}/reason`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tokens }),
    });
  }

  /** manifest → encode → reason → advice; every failure is an abstention. */
  async advise(c: AdviseContext): Promise<SupervisorAdvice> {
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
