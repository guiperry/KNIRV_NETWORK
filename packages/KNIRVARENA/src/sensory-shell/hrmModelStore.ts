import { getKnirvServerUrl } from '../config/runtimeConfig';
import { authToken } from '../services/serverLLM';

/** `GET /api/hrm/models` (arena_fixes.md §2.6, point 5). */
export interface HRMManifest {
  version: string;
  /** `hrm_config.json` contents. */
  config: Record<string, unknown>;
  sha256: string;
  size: number;
  signature?: string;
}

export interface HRMModelFiles {
  manifest: HRMManifest;
  weights: Uint8Array;
  fromCache: boolean;
}

export class HRMModelUnavailableError extends Error {
  constructor(message: string, readonly status?: number) {
    super(message);
    this.name = 'HRMModelUnavailableError';
  }
}

const CACHE_NAME = 'knirv-hrm-models';

export async function sha256Hex(data: Uint8Array): Promise<string> {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle) throw new HRMModelUnavailableError('WebCrypto unavailable; cannot verify model');
  const digest = await subtle.digest('SHA-256', data as BufferSource);
  return Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, '0')).join('');
}

async function openCache(): Promise<Cache | null> {
  try { return typeof caches === 'undefined' ? null : await caches.open(CACHE_NAME); } catch { return null; }
}

/**
 * Fetches the latest model, verifying its SHA-256 against the manifest. Weights
 * are cached by version; a cached copy is re-verified before use. A model whose
 * hash does not match is never returned.
 */
export async function fetchLatestModel(fetchImpl: typeof fetch = fetch): Promise<HRMModelFiles> {
  const token = authToken();
  if (!token) throw new HRMModelUnavailableError('Sign in to KNIRV to load the HRM', 401);
  const headers = { Authorization: `Bearer ${token}` };
  const base = getKnirvServerUrl();

  const res = await fetchImpl(`${base}/api/hrm/models`, { headers });
  if (!res.ok) throw new HRMModelUnavailableError(`HRM manifest unavailable (HTTP ${res.status})`, res.status);
  const body = (await res.json()) as { success?: boolean; data?: HRMManifest } & Partial<HRMManifest>;
  const manifest = (body.data ?? body) as HRMManifest;
  if (!manifest?.version || !manifest.sha256 || !manifest.config) {
    throw new HRMModelUnavailableError('HRM manifest is malformed');
  }

  const key = `${base}/api/hrm/models/${encodeURIComponent(manifest.version)}/weights`;
  const cache = await openCache();
  const cached = cache ? await cache.match(key) : undefined;
  if (cached) {
    const weights = new Uint8Array(await cached.arrayBuffer());
    if ((await sha256Hex(weights)) === manifest.sha256) return { manifest, weights, fromCache: true };
    await cache?.delete(key);
  }

  const w = await fetchImpl(key, { headers });
  if (!w.ok) throw new HRMModelUnavailableError(`HRM weights unavailable (HTTP ${w.status})`, w.status);
  const weights = new Uint8Array(await w.arrayBuffer());
  if ((await sha256Hex(weights)) !== manifest.sha256) {
    throw new HRMModelUnavailableError('HRM weights failed SHA-256 verification');
  }
  try { await cache?.put(key, new Response(weights as BodyInit)); } catch { /* cache is best-effort */ }
  return { manifest, weights, fromCache: false };
}
