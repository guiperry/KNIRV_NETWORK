/**
 * Language-model calls through KNIRVSERVER (arena_fixes.md §3.5).
 *
 * Provider keys never live in the browser. A user stores their own keys in
 * KNIRVSERVER secret storage (write-only from here) and the server calls the
 * provider with them; the local llama.cpp model (`llama`) needs no key.
 */

import { getKnirvServerUrl } from '../config/runtimeConfig';

export type ServerLLMProvider = 'llama' | 'openai' | 'anthropic' | 'gemini' | 'deepseek' | 'cerebras';
export type KeyedProvider = Exclude<ServerLLMProvider, 'llama'>;

/** Model used when a caller does not choose one. */
export const DEFAULT_MODELS: Record<KeyedProvider, string> = {
  openai: 'gpt-4o-mini',
  anthropic: 'claude-haiku-4-5',
  gemini: 'gemini-1.5-flash',
  deepseek: 'deepseek-chat',
  cerebras: 'llama3.1-8b',
};

export interface ServerLLMMessage {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

export interface ServerLLMRequest {
  provider: ServerLLMProvider;
  model?: string;
  messages: ServerLLMMessage[];
  maxTokens?: number;
  /** The HRM's plan/decision (§2.6); the model verbalises it. */
  hrmPlan?: Record<string, unknown>;
}

export class ServerLLMError extends Error {
  constructor(message: string, readonly code?: string, readonly status?: number) {
    super(message);
    this.name = 'ServerLLMError';
  }
}

const AUTH_TOKEN_KEY = 'knirv_auth_token';

export function authToken(): string | null {
  try { return localStorage.getItem(AUTH_TOKEN_KEY); } catch { return null; }
}

async function call(path: string, init: RequestInit, fetchImpl: typeof fetch = fetch): Promise<Response> {
  const token = authToken();
  if (!token) throw new ServerLLMError('Sign in to KNIRV to use language models', 'UNAUTHENTICATED', 401);
  return fetchImpl(`${getKnirvServerUrl()}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...(init.headers ?? {}) },
  });
}

export async function completeOnServer(request: ServerLLMRequest, fetchImpl: typeof fetch = fetch): Promise<string> {
  const model = request.provider === 'llama' ? undefined : request.model ?? DEFAULT_MODELS[request.provider];
  const response = await call('/api/llm/complete', {
    method: 'POST',
    body: JSON.stringify({
      provider: request.provider,
      model,
      messages: request.messages,
      max_tokens: request.maxTokens,
      hrm_plan: request.hrmPlan,
    }),
  }, fetchImpl);
  let body: { success?: boolean; data?: { text?: string }; error?: string; code?: string } = {};
  try { body = await response.json(); } catch { /* non-JSON error */ }
  if (!response.ok || body.success === false) {
    throw new ServerLLMError(body.error ?? `completion failed with HTTP ${response.status}`, body.code, response.status);
  }
  return body.data?.text ?? '';
}

/** Name under which a provider key is stored in KNIRVSERVER secrets. */
export const providerSecretName = (provider: KeyedProvider) => `provider:${provider}`;

/**
 * Stores (or replaces) the user's key for a provider. Keys are write-only from
 * the browser: they are never read back into the page.
 */
export async function saveProviderKey(provider: KeyedProvider, key: string, fetchImpl: typeof fetch = fetch): Promise<void> {
  if (!key.trim()) throw new ServerLLMError('Enter a key');
  await removeProviderKey(provider, fetchImpl);
  const response = await call('/api/secrets/create', {
    method: 'POST',
    body: JSON.stringify({ name: providerSecretName(provider), type: 'api_key', value: key.trim() }),
  }, fetchImpl);
  if (!response.ok) throw new ServerLLMError(`could not save the ${provider} key (HTTP ${response.status})`, undefined, response.status);
}

/** Which providers the user has stored keys for (names only, never values). */
export async function listProviderKeys(fetchImpl: typeof fetch = fetch): Promise<KeyedProvider[]> {
  const response = await call('/api/secrets/list', { method: 'GET' }, fetchImpl);
  if (!response.ok) throw new ServerLLMError(`could not list keys (HTTP ${response.status})`, undefined, response.status);
  const body = (await response.json()) as { secrets?: Array<{ name?: string }> | null };
  const names = new Set((body.secrets ?? []).map((s) => s.name ?? ''));
  return (Object.keys(DEFAULT_MODELS) as KeyedProvider[]).filter((p) => names.has(providerSecretName(p)));
}

export async function removeProviderKey(provider: KeyedProvider, fetchImpl: typeof fetch = fetch): Promise<void> {
  const response = await call('/api/secrets/list', { method: 'GET' }, fetchImpl);
  if (!response.ok) return;
  const body = (await response.json()) as { secrets?: Array<{ id?: string; name?: string }> | null };
  for (const secret of body.secrets ?? []) {
    if (secret.name === providerSecretName(provider) && secret.id) {
      await call(`/api/secrets/${encodeURIComponent(secret.id)}`, { method: 'DELETE' }, fetchImpl);
    }
  }
}
