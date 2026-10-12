import { HRMLoRABridge, HRMAdapterHost } from '../HRMLoRABridge';

jest.mock('../../services/serverLLM', () => ({ authToken: jest.fn(() => 'tok') }));
jest.mock('../../config/runtimeConfig', () => ({ getKnirvServerUrl: () => 'https://k' }));

const bind = { adapter_rank: 2, scaling_alpha: 8, layers: { 'self_attn.q_proj': { a: [[1]], b: [[1]] } }, model_version: 'v7', content_hash: 'h1' };
const applied = { content_hash: 'h1', tensors_changed: 4, roles_applied: ['self_attn.q_proj'], roles_skipped: [] };

function host(version = 'v7') {
  let adapter: typeof applied | null = null;
  const h: HRMAdapterHost & { version: string; calls: unknown[] } = {
    version, calls: [],
    applyAdapter: jest.fn(async (b) => { h.calls.push(b); adapter = applied; return applied; }),
    clearAdapter: jest.fn(async () => { adapter = null; }),
    getAdapter: () => adapter,
    getAdaptableTensors: () => ['l.0.attn.q.weight', 'l.0.attn.k.weight', 'h.0.attn.q.weight', 'h.0.attn.o.weight', 'h.0.mlp.up.weight'],
    getModelInfo: () => ({ model_version: h.version }),
    isLocal: () => true,
  };
  return h;
}
const okFetch = () => jest.fn(async () => ({ ok: true, status: 200, json: async () => ({ success: true, data: bind }) }) as unknown as Response);

describe('HRMLoRABridge', () => {
  it('maps uLoRA roles onto the real HRM tensor names', async () => {
    const b = new HRMLoRABridge({}, { fetchImpl: okFetch() });
    b.setHRMBridge(host());
    await b.start();
    const m = b.getMappings();
    expect([...m.keys()].sort()).toEqual(['self_attn.k_proj', 'self_attn.o_proj', 'self_attn.q_proj']);
    expect(m.get('self_attn.q_proj')!.hrmTensors).toEqual(['l.0.attn.q.weight', 'h.0.attn.q.weight']);
    await b.stop();
  });

  it('binds on the server with the caller token, applies once, and re-applies only for a new model version', async () => {
    const f = okFetch();
    const h = host();
    const b = new HRMLoRABridge({ strength: 0.5, stacks: ['h'] }, { fetchImpl: f });
    b.setHRMBridge(h);
    await b.start();
    await b.setBundle('bundle-1');
    expect(f).toHaveBeenCalledTimes(1);
    const [url, init] = (f as jest.Mock).mock.calls[0];
    expect(url).toBe('https://k/api/hrm/adapters/bind');
    expect(init.headers.Authorization).toBe('Bearer tok');
    expect(JSON.parse(init.body)).toEqual({ bundle_id: 'bundle-1' });
    expect(h.calls[0]).toMatchObject({ stacks: ['h'], strength: 0.5, content_hash: 'h1' });
    expect(b.getMappings().get('self_attn.q_proj')).toMatchObject({ applied: true, hrmTensors: ['h.0.attn.q.weight'] });

    await b.forceSyncNow();                    // current: no refetch
    expect(f).toHaveBeenCalledTimes(1);
    h.version = 'v8';                          // a newly promoted HRM replaced the model
    (h as { getAdapter: () => null }).getAdapter = () => null;
    await b.forceSyncNow();
    expect(f).toHaveBeenCalledTimes(2);
    await b.stop();
  });

  it('reports a failed bind and leaves the model alone', async () => {
    const f = jest.fn(async () => ({ ok: false, status: 409, json: async () => ({ success: false, error: 'bundle content hash mismatch' }) }) as unknown as Response);
    const h = host();
    const b = new HRMLoRABridge({}, { fetchImpl: f });
    b.setHRMBridge(h);
    const errors: Error[] = [];
    b.on('syncError', (e: unknown) => errors.push(e as Error));
    await b.start();
    await expect(b.setBundle('x')).rejects.toThrow(/hash mismatch/);
    expect(h.applyAdapter).not.toHaveBeenCalled();
    expect(errors).toHaveLength(1);
    await b.stop();
  });

  it('accepts bound factors directly and refuses anything else', async () => {
    const h = host();
    const b = new HRMLoRABridge();
    b.setHRMBridge(h);
    const errors: Error[] = [];
    b.on('syncError', (e: unknown) => errors.push(e as Error));
    b.updateLoRAWeights({ base_hidden_1: { A: [[1]], B: [[1]] } });   // an EnhancedLoRAAdapter export: not HRM weights
    expect(h.applyAdapter).not.toHaveBeenCalled();
    expect(errors).toHaveLength(1);
    b.updateLoRAWeights(bind as unknown as Record<string, unknown>);
    await new Promise((r) => setTimeout(r, 0));
    expect(h.applyAdapter).toHaveBeenCalledTimes(1);
  });

  it('clear() removes the adapter and the bundle', async () => {
    const h = host();
    const b = new HRMLoRABridge({}, { fetchImpl: okFetch() });
    b.setHRMBridge(h);
    await b.start();
    await b.setBundle('bundle-1');
    await b.setBundle(null);
    expect(h.clearAdapter).toHaveBeenCalled();
    expect(b.getStatus()).toMatchObject({ bundleId: null, applied: null });
    await b.stop();
  });
});
