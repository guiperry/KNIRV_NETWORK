import { getKnirvServerUrl } from '../config/runtimeConfig';
import { authToken } from '../services/serverLLM';

/** `GET /api/head/models` (arena_fixes.md §2.6, point 5). */
export interface HEADManifest {
  version: string;
  /** `head_config.json` contents. */
  config: Record<string, unknown>;
  sha256: string;
  size: number;
  signature?: string;
}

export interface HEADModelFiles {
  manifest: HEADManifest;
  weights: Uint8Array;
  fromCache: boolean;
}

export class HEADModelUnavailableError extends Error {
  constructor(message: string, readonly status?: number) {
    super(message);
    this.name = 'HEADModelUnavailableError';
  }
}

const CACHE_NAME = 'knirv-head-models';

export async function sha256Hex(data: Uint8Array): Promise<string> {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle) throw new HEADModelUnavailableError('WebCrypto unavailable; cannot verify model');
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
export async function fetchLatestModel(fetchImpl: typeof fetch = fetch): Promise<HEADModelFiles> {
  const token = authToken();
  if (!token) throw new HEADModelUnavailableError('Sign in to KNIRV to load the HEAD', 401);
  const headers = { Authorization: `Bearer ${token}` };
  const base = getKnirvServerUrl();

  const res = await fetchImpl(`${base}/api/head/models`, { headers });
  if (!res.ok) throw new HEADModelUnavailableError(`HEAD manifest unavailable (HTTP ${res.status})`, res.status);
  const body = (await res.json()) as { success?: boolean; data?: HEADManifest } & Partial<HEADManifest>;
  const manifest = (body.data ?? body) as HEADManifest;
  if (!manifest?.version || !manifest.sha256 || !manifest.config) {
    throw new HEADModelUnavailableError('HEAD manifest is malformed');
  }

  const key = `${base}/api/head/models/${encodeURIComponent(manifest.version)}/weights`;
  const cache = await openCache();
  const cached = cache ? await cache.match(key) : undefined;
  if (cached) {
    const weights = new Uint8Array(await cached.arrayBuffer());
    if ((await sha256Hex(weights)) === manifest.sha256) return { manifest, weights, fromCache: true };
    await cache?.delete(key);
  }

  const w = await fetchImpl(key, { headers });
  if (!w.ok) throw new HEADModelUnavailableError(`HEAD weights unavailable (HTTP ${w.status})`, w.status);
  const weights = new Uint8Array(await w.arrayBuffer());
  if ((await sha256Hex(weights)) !== manifest.sha256) {
    throw new HEADModelUnavailableError('HEAD weights failed SHA-256 verification');
  }
  try { await cache?.put(key, new Response(weights as BodyInit)); } catch { /* cache is best-effort */ }
  return { manifest, weights, fromCache: false };
}
