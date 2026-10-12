import { webcrypto } from 'crypto';
import { HRMBridge, HRMUnavailableError } from '../HRMBridge';
import { HRMEngine, HRMReasoning } from '../hrmEngine';
import { decodeTokens, encodeSamples, encodeText, padTo, BOS, PAD } from '../hrmTokenizer';
import { sha256Hex } from '../hrmModelStore';

const config = { l_module_count: 1, h_module_count: 1, enable_adaptation: false, processing_timeout: 1000 };

const reasoning = (plan = 'ok'): HRMReasoning => ({
  plan_tokens: Array.from(new TextEncoder().encode(plan)),
  halting_steps: 2, max_steps: 4, halt_probability: 0.7, confidence: 0.8,
  decisions: [
    { name: 'action', kind: 'choice', options: ['allow', 'ask_operator'], probabilities: [0.2, 0.8], selected: 'ask_operator', value: null, confidence: 0.8, abstain: false },
    { name: 'risk', kind: 'score', options: [], probabilities: [], selected: null, value: 0.4321, confidence: 0.5679, abstain: false },
  ],
  l_module_activations: [0.1, 0.2], h_module_activations: [0.3, 0.4],
  model_version: 'v7', dataset_manifest_hash: 'd', policy_bundle_hash: 'p',
});

function fakeEngine(): HRMEngine & { seen: number[][] } {
  const seen: number[][] = [];
  return {
    seen,
    load: jest.fn().mockResolvedValue({ loaded: true, total_parameters: 123, l_modules: 2, h_modules: 2, max_input_len: 64, tokenizer: 'byte-v1', model_version: 'v7' }),
    reason: jest.fn(async (t: number[]) => { seen.push(t); return reasoning(); }),
    dispose: jest.fn(),
  };
}

const model = (tokenizer = 'byte-v1') => async () => ({
  manifest: { version: 'v7', sha256: 'x', size: 1, config: { tokenizer } },
  weights: new Uint8Array([1, 2, 3]), fromCache: false,
});

describe('hrmTokenizer', () => {
  it('round-trips text and quantises samples', () => {
    expect(decodeTokens(encodeText('héllo', 64))).toBe('héllo');
    expect(encodeText('abcdef', 3)).toEqual([BOS, 97, 98]);
    expect(encodeSamples([0, 0.5, 1, 2, NaN], 10)).toEqual([BOS, 0, 128, 255, 255, 0]);
  });
});

describe('sha256Hex', () => {
  it('matches the known digest of "abc"', async () => {
    Object.defineProperty(globalThis, 'crypto', { value: webcrypto, configurable: true });
    expect(await sha256Hex(new TextEncoder().encode('abc'))).toBe('ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad');
  });
});

describe('HRMBridge', () => {
  const signIn = (token: string | null) => (localStorage.getItem as jest.Mock).mockImplementation((k: string) => (k === 'knirv_auth_token' ? token : null));
  beforeEach(() => signIn('tok'));
  afterEach(() => (localStorage.getItem as jest.Mock).mockReset());

  it('refuses to run before initialize', async () => {
    await expect(new HRMBridge(config).processTextInput('x')).rejects.toBeInstanceOf(HRMUnavailableError);
    expect(new HRMBridge(config).isReady()).toBe(false);
  });

  it('runs the loaded model and maps real outputs, with no mock path', async () => {
    const engine = fakeEngine();
    const b = new HRMBridge(config, { engine, fetchModel: model() });
    await b.initialize();
    expect(b.isLocal()).toBe(true);
    const out = await b.processTextInput('why');
    expect(engine.seen[0]).toEqual(padTo(encodeText('why', 64), 64));
    expect(engine.seen[0]).toHaveLength(64);
    expect(out).toMatchObject({ source: 'wasm', halting_steps: 2, plan: 'ok', model_version: 'v7', confidence: 0.8 });
    expect(out.reasoning_result).toBe('action=ask_operator (0.80); risk=0.432');
    expect(out.l_module_activations).toEqual([0.1, 0.2]);
    expect(b.getModelInfo()).toMatchObject({ total_parameters: 123, l_modules: 2, h_modules: 2 });
  });

  it('uses byte tokens for voice/visual data', async () => {
    const engine = fakeEngine();
    const b = new HRMBridge(config, { engine, fetchModel: model() });
    await b.initialize();
    await b.processVisualInput([0, 1]);
    expect(engine.seen[0].slice(0, 4)).toEqual([BOS, 0, 255, PAD]);
    expect(engine.seen[0]).toHaveLength(64);
  });

  it('falls back to the server when no model loads, and reports server as the source', async () => {
    const fetchImpl = jest.fn().mockResolvedValue({ ok: true, json: async () => ({ success: true, data: reasoning('srv') }) });
    const b = new HRMBridge(config, { fetchModel: async () => { throw new Error('404'); }, fetchImpl: fetchImpl as unknown as typeof fetch });
    const unavailable = jest.fn();
    b.on('unavailable', unavailable);
    await b.initialize();
    expect(unavailable).toHaveBeenCalled();
    expect(b.isLocal()).toBe(false);
    const out = await b.processTextInput('x');
    expect(out.source).toBe('server');
    const [url, init] = fetchImpl.mock.calls[0];
    expect(String(url)).toMatch(/\/api\/hrm\/reason$/);
    expect(JSON.parse(init.body)).toMatchObject({ tokenizer: 'byte-v1' });
  });

  it('throws HRMUnavailableError rather than inventing output when the server also fails', async () => {
    const fetchImpl = jest.fn().mockResolvedValue({ ok: false, status: 404, json: async () => ({}) });
    const b = new HRMBridge(config, { fetchModel: async () => { throw new Error('x'); }, fetchImpl: fetchImpl as unknown as typeof fetch });
    await b.initialize();
    await expect(b.processTextInput('x')).rejects.toBeInstanceOf(HRMUnavailableError);
  });

  it('does not use a model whose tokenizer the browser cannot run', async () => {
    const engine = fakeEngine();
    const b = new HRMBridge(config, { engine, fetchModel: model('tiktoken-cl100k') });
    await b.initialize();
    expect(b.isLocal()).toBe(false);
    expect(engine.load).not.toHaveBeenCalled();
  });

  it('requires sign-in for server reasoning', async () => {
    signIn(null);
    const fetchImpl = jest.fn();
    const b = new HRMBridge(config, { fetchModel: async () => { throw new Error('x'); }, fetchImpl: fetchImpl as unknown as typeof fetch });
    await b.initialize();
    await expect(b.processTextInput('x')).rejects.toThrow(/sign in/i);
    expect(fetchImpl).not.toHaveBeenCalled();
  });

  it('destroy disposes the engine', async () => {
    const engine = fakeEngine();
    const b = new HRMBridge(config, { engine, fetchModel: model() });
    await b.initialize();
    await b.destroy();
    expect(engine.dispose).toHaveBeenCalled();
    expect(b.isReady()).toBe(false);
  });
});

describe('HRMBridge adapters (uLoRA)', () => {
  const bind = { adapter_rank: 2, scaling_alpha: 8, layers: { 'self_attn.q_proj': { a: [[1]], b: [[1]] } }, model_version: 'v7', content_hash: 'h1' };
  const applied = { content_hash: 'h1', tensors_changed: 4, roles_applied: ['self_attn.q_proj'], roles_skipped: [] };

  async function ready() {
    const engine = fakeEngine();
    (engine.load as jest.Mock).mockImplementation(async (_c: string, _w: Uint8Array, adapter?: string) => ({
      loaded: true, total_parameters: 123, l_modules: 2, h_modules: 2, max_input_len: 64, tokenizer: 'byte-v1', model_version: 'v7',
      adaptable_tensors: ['l.0.attn.q.weight', 'h.0.attn.q.weight'], ...(adapter ? { adapter: applied } : {}),
    }));
    const bridge = new HRMBridge(config, { engine, fetchModel: model() });
    await bridge.initialize();
    return { engine, bridge };
  }

  it('merges into the verified base weights and can remove the adapter again', async () => {
    const { engine, bridge } = await ready();
    expect(await bridge.applyAdapter(bind)).toEqual(applied);
    const call = (engine.load as jest.Mock).mock.calls.at(-1);
    expect(Array.from(call[1] as Uint8Array)).toEqual([1, 2, 3]);      // the base weights, not a previous merge
    expect(JSON.parse(call[2]).layers['self_attn.q_proj']).toBeDefined();
    expect(bridge.getAdapter()).toEqual(applied);
    await bridge.clearAdapter();
    expect(bridge.getAdapter()).toBeNull();
    expect((engine.load as jest.Mock).mock.calls.at(-1)[2]).toBeUndefined();
  });

  it('refuses an adapter bound for a different model version, and one with no local model', async () => {
    const { bridge } = await ready();
    await expect(bridge.applyAdapter({ ...bind, model_version: 'v6' })).rejects.toThrow(/bound for model v6/);
    const server = new HRMBridge(config, { fetchModel: async () => { throw new Error('none'); } });
    await server.initialize();
    await expect(server.applyAdapter(bind)).rejects.toBeInstanceOf(HRMUnavailableError);
  });
});
