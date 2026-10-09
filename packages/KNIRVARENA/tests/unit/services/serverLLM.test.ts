/**
 * Language-model calls and provider keys through KNIRVSERVER (arena_fixes.md §3.5).
 */
import {
  ServerLLMError,
  completeOnServer,
  listProviderKeys,
  removeProviderKey,
  saveProviderKey,
} from '../../../src/services/serverLLM';
import { LLMProviderService } from '../../../src/services/llmProviderService';
import { getKnirvServerUrl } from '../../../src/config/runtimeConfig';

type Call = { url: string; init: RequestInit };

function fakeFetch(routes: Record<string, (init: RequestInit) => { status: number; body: unknown }>) {
  const calls: Call[] = [];
  const impl = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    const path = url.replace(getKnirvServerUrl(), '');
    const route = routes[`${init.method} ${path}`];
    const { status, body } = route ? route(init) : { status: 404, body: { error: 'no route' } };
    return { ok: status >= 200 && status < 300, status, json: async () => body };
  }) as unknown as typeof fetch;
  return { impl, calls };
}

const bodyOf = (c: Call) => JSON.parse(String(c.init.body));

// The shared polyfill's localStorage discards writes; these tests need a real one.
const store = new Map<string, string>();
Object.defineProperty(global, 'localStorage', {
  configurable: true,
  writable: true,
  value: {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, String(v)),
    removeItem: (k: string) => void store.delete(k),
    clear: () => store.clear(),
  },
});

beforeEach(() => localStorage.setItem('knirv_auth_token', 'jwt'));
afterEach(() => localStorage.clear());

describe('completeOnServer', () => {
  it('posts the conversation with the bearer token and returns the text', async () => {
    const { impl, calls } = fakeFetch({ 'POST /api/llm/complete': () => ({ status: 200, body: { success: true, data: { text: 'hello' } } }) });
    const text = await completeOnServer({ provider: 'openai', messages: [{ role: 'user', content: 'hi' }], maxTokens: 50 }, impl);
    expect(text).toBe('hello');
    expect((calls[0].init.headers as Record<string, string>).Authorization).toBe('Bearer jwt');
    expect(bodyOf(calls[0])).toEqual({ provider: 'openai', model: 'gpt-4o-mini', messages: [{ role: 'user', content: 'hi' }], max_tokens: 50 });
  });

  it('sends no model for the local model', async () => {
    const { impl, calls } = fakeFetch({ 'POST /api/llm/complete': () => ({ status: 200, body: { success: true, data: { text: 'x' } } }) });
    await completeOnServer({ provider: 'llama', model: 'ignored', messages: [{ role: 'user', content: 'hi' }] }, impl);
    expect(bodyOf(calls[0]).model).toBeUndefined();
  });

  it('surfaces server error codes, and refuses when signed out', async () => {
    const { impl } = fakeFetch({ 'POST /api/llm/complete': () => ({ status: 428, body: { success: false, code: 'PROVIDER_KEY_REQUIRED', error: 'add your key' } }) });
    const err = await completeOnServer({ provider: 'gemini', messages: [{ role: 'user', content: 'hi' }] }, impl).catch((e) => e);
    expect(err).toBeInstanceOf(ServerLLMError);
    expect(err.code).toBe('PROVIDER_KEY_REQUIRED');
    expect(err.status).toBe(428);

    localStorage.clear();
    const { impl: unused, calls } = fakeFetch({});
    await expect(completeOnServer({ provider: 'llama', messages: [] }, unused)).rejects.toMatchObject({ code: 'UNAUTHENTICATED' });
    expect(calls).toHaveLength(0);
  });
});

describe('provider keys', () => {
  const secrets = [
    { id: 'sec_1', name: 'provider:openai' },
    { id: 'sec_2', name: 'something-else' },
    { id: 'sec_3', name: 'provider:deepseek' },
  ];

  it('lists only providers with a stored key', async () => {
    const { impl } = fakeFetch({ 'GET /api/secrets/list': () => ({ status: 200, body: { secrets } }) });
    expect(await listProviderKeys(impl)).toEqual(['openai', 'deepseek']);
  });

  it('replaces an existing key: deletes the old secret, then creates the new one', async () => {
    const { impl, calls } = fakeFetch({
      'GET /api/secrets/list': () => ({ status: 200, body: { secrets } }),
      'DELETE /api/secrets/sec_1': () => ({ status: 200, body: {} }),
      'POST /api/secrets/create': () => ({ status: 201, body: {} }),
    });
    await saveProviderKey('openai', '  sk-new  ', impl);
    expect(calls.map((c) => `${c.init.method} ${c.url.replace(getKnirvServerUrl(), '')}`)).toEqual([
      'GET /api/secrets/list', 'DELETE /api/secrets/sec_1', 'POST /api/secrets/create',
    ]);
    expect(bodyOf(calls[2])).toEqual({ name: 'provider:openai', type: 'api_key', value: 'sk-new' });
  });

  it('rejects an empty key without calling the server', async () => {
    const { impl, calls } = fakeFetch({});
    await expect(saveProviderKey('gemini', '   ', impl)).rejects.toThrow(/Enter a key/);
    expect(calls).toHaveLength(0);
  });

  it('removes only the matching provider secret', async () => {
    const { impl, calls } = fakeFetch({
      'GET /api/secrets/list': () => ({ status: 200, body: { secrets } }),
      'DELETE /api/secrets/sec_3': () => ({ status: 200, body: {} }),
    });
    await removeProviderKey('deepseek', impl);
    expect(calls.filter((c) => c.init.method === 'DELETE').map((c) => c.url)).toEqual([`${getKnirvServerUrl()}/api/secrets/sec_3`]);
  });
});

describe('LLMProviderService', () => {
  const realFetch = global.fetch;
  afterEach(() => { global.fetch = realFetch; });

  it('maps adaline to the local model and returns the server text', async () => {
    const { impl, calls } = fakeFetch({ 'POST /api/llm/complete': () => ({ status: 200, body: { success: true, data: { text: 'local answer' } } }) });
    global.fetch = impl;
    const service = new LLMProviderService();
    const reply = await service.chat('fix it', 'adaline', [{ id: '1', type: 'user', text: 'earlier', timestamp: 0 }]);
    expect(reply).toMatchObject({ text: 'local answer', provider: 'adaline' });
    expect(bodyOf(calls[0])).toMatchObject({
      provider: 'llama',
      messages: [{ role: 'user', content: 'earlier' }, { role: 'user', content: 'fix it' }],
    });
  });

  it('propagates server errors instead of inventing a reply', async () => {
    global.fetch = fakeFetch({ 'POST /api/llm/complete': () => ({ status: 503, body: { success: false, code: 'LOCAL_MODEL_UNAVAILABLE', error: 'down' } }) }).impl;
    await expect(new LLMProviderService().chat('hi', 'adaline')).rejects.toMatchObject({ code: 'LOCAL_MODEL_UNAVAILABLE' });
  });

  it('reports providers available only when signed in', () => {
    const service = new LLMProviderService();
    expect(service.getAvailableProviders()).toEqual(['adaline', 'gemini', 'openai', 'deepseek']);
    localStorage.clear();
    expect(service.getAvailableProviders()).toEqual([]);
  });
});
