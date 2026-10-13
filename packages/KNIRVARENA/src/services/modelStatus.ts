/**
 * Platform model status (GET /api/models/status): which model does what, right now.
 *
 * The wire types and `roles()` are a copy of KNIRVSDK's `models` module (npm-package/src/models),
 * kept here until `@knirv/sdk` is republished with that module. `modelStatus.test.ts` runs this
 * copy against the SDK's shared fixture (testvectors/models/platform_status.json), so it
 * cannot drift from the Go and TS twins unnoticed.
 */

import { getKnirvServerUrl } from '../config/runtimeConfig';

export interface ModelStatus {
  name: string; role: string; version: string; ready: boolean; loaded: boolean; trained: boolean;
  backend: string; since?: string; error?: string;
}


/** The backend's authenticated route for the platform-wide model view. */
export const STATUS_PATH = "/api/models/status";

export interface LlamaAdapter { name: string; loaded: boolean; base_sha256?: string; bundle_hash?: string }
/** The embedded llama.cpp. `base_model` is a file name, never a path. */
export interface LlamaStatus { ready: boolean; base_model?: string; base_sha256?: string; adapters?: LlamaAdapter[]; error?: string }
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

const sha12 = (s: string) => (s.length > 12 ? s.slice(0, 12) : s);

/** The fixed list of role lines for a status document (same logic as the Go twin). */
export function roles(s: PlatformStatus): RoleLine[] {
  const byName = new Map((s.daemon ?? []).map((m) => [m.name, m]));
  const daemon = (role: string, label: string, name: string): RoleLine => {
    const l: RoleLine = { role, label, model: name, version: "", state: "unavailable" };
    const m = byName.get(name);
    if (!m) {
      l.detail = s.daemon_error ? s.daemon_error : "not served by the models daemon";
    } else if (m.ready) {
      l.state = "ready"; l.version = m.version;
    } else if (!m.loaded && !m.error) {
      l.state = "idle"; l.version = m.version; l.detail = "not built yet; built on first use";
    } else {
      l.state = "not ready"; l.version = m.version; l.detail = m.error || "not trained yet";
    }
    return l;
  };
  const llama: RoleLine = { role: "server_tasks", label: "server tasks and skills", model: "llama", version: "", state: "unavailable", detail: "llama is not running" };
  if (s.llama) {
    llama.state = "not ready"; llama.detail = s.llama.error ?? "";
    if (s.llama.ready) {
      llama.state = "ready";
      if (s.llama.base_sha256) llama.version = "gguf-" + sha12(s.llama.base_sha256);
      const loaded = (s.llama.adapters ?? []).filter((a) => a.loaded).map((a) => a.name);
      llama.detail = loaded.length ? "adapters: " + loaded.join(", ") : "no skill adapter loaded";
    } else if (!llama.detail) {
      llama.detail = "starting";
    }
  }
  const ext: RoleLine = { role: "external", label: "fallback and planning", model: "external", version: "provider-chain", state: "ready",
    detail: "chosen per request; the only model that leaves this machine" };
  const st = s.stats?.["external"];
  if (st && st.circuit && st.circuit !== "closed") { ext.state = "not ready"; ext.detail = `circuit ${st.circuit}: ${st.last_error ?? ""}`; }
  return [daemon("decide", "supervision decisions", "head"), daemon("generate", "text generation", "heart"),
    daemon("validate", "validation", "hasher"), llama, ext];
}

/** Reads the status document with the signed-in user's token. Throws on any failure. */
export async function fetchModelStatus(token: string | null, limit = 10, fetchImpl: typeof fetch = fetch): Promise<PlatformStatus> {
  if (!token) throw new Error('Sign in to KNIRV to see model status');
  const res = await fetchImpl(`${getKnirvServerUrl()}${STATUS_PATH}?limit=${limit}`, { headers: { Authorization: `Bearer ${token}` } });
  if (res.status === 404) throw new Error('This KNIRV server does not report model status yet');
  if (!res.ok) throw new Error(`Model status unavailable (HTTP ${res.status})`);
  return (await res.json()) as PlatformStatus;
}
