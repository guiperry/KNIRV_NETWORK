import '../../utils/realWebAssembly'; // must stay first: the SDK core is wasm and needs the real TextDecoder
import * as sdk from '@knirv/sdk/models';
const { ModelsClient } = sdk;
import { KnirvModelsService } from '../../../src/sensory-shell/KnirvModelsService';

const BASE = 'https://testnet-gateway.knirv.com/api/knirvmodels';

type Reply = { status: number; body?: unknown };

/** Fake fetch that answers by path suffix and records every URL it was asked for. */
const routedFetch = (routes: Record<string, Reply>) => {
  const calls: string[] = [];
  const fetchImpl = (async (input: RequestInfo | URL) => {
    const url = String(input);
    calls.push(url);
    const match = Object.keys(routes).find((suffix) => url.endsWith(suffix));
    const reply: Reply = match ? routes[match] : { status: 404, body: { error: 'no route' } };
    const text = JSON.stringify(reply.body ?? {});
    return {
      ok: reply.status >= 200 && reply.status < 300,
      status: reply.status,
      statusText: '',
      headers: new Headers(),
      text: async () => text,
      json: async () => JSON.parse(text),
    } as unknown as Response;
  }) as unknown as typeof fetch;
  return { fetchImpl, calls };
};

const serviceWith = (routes: Record<string, Reply>) => {
  const { fetchImpl, calls } = routedFetch(routes);
  const client = new ModelsClient(BASE, fetchImpl);
  return { service: new KnirvModelsService({ client, sdk }), calls };
};

const heartReady = (text: string) => ({ status: 200, body: { ready: true, progress: 1, text } });
const verdict = (v: 'supported' | 'refuted') => ({
  status: 200,
  body: { status: 'done', gate: 'index', gate_level: 1, gate_score: 1, verdict: v, confidence: 0.9, claims: [] },
});

describe('KnirvModelsService', () => {
  it('generates with HEART and verifies with HASHER through the gateway', async () => {
    const { service, calls } = serviceWith({
      '/models/heart/generate': heartReady('Paris is the capital of France.'),
      '/models/hasher/verify': verdict('supported'),
    });

    const out = await service.generateVerifiedText('capital of France?');

    expect(out).toEqual(expect.objectContaining({ text: 'Paris is the capital of France.', source: 'heart' }));
    expect(calls.map((u) => u.replace(BASE, ''))).toEqual(
      expect.arrayContaining([expect.stringMatching(/\/models\/heart\/generate$/), expect.stringMatching(/\/models\/hasher\/verify$/)]),
    );
    expect(calls.every((u) => u.startsWith(BASE))).toBe(true);
  });

  it('never returns text that HASHER refutes', async () => {
    const { service } = serviceWith({
      '/models/heart/generate': heartReady('Paris is in Germany.'),
      '/models/hasher/verify': verdict('refuted'),
    });

    await expect(service.generateVerifiedText('where is Paris?')).resolves.toBeNull();
  });

  it('falls back to the HEAD text mirror when HEART is not ready', async () => {
    const { service, calls } = serviceWith({
      '/models/heart/generate': { status: 200, body: { ready: false, progress: 0 } },
      '/models/head/generate': { status: 200, body: { text: 'head answer', model: 'head' } },
    });

    const out = await service.generateVerifiedText('hello');

    expect(out).toEqual({ text: 'head answer', source: 'head', verification: null });
    expect(calls.some((u) => u.endsWith('/models/hasher/verify'))).toBe(false);
  });

  it('returns null when neither HEART nor HEAD can answer (503 not ready)', async () => {
    const { service } = serviceWith({
      '/models/heart/generate': { status: 503, body: { error: 'not ready', ready: false, progress: 0 } },
      '/models/head/generate': { status: 503, body: { error: 'no vision backend' } },
    });

    await expect(service.generateVerifiedText('hello')).resolves.toBeNull();
  });

  it('returns null instead of throwing when the gateway is unreachable', async () => {
    const failing = (async () => { throw new TypeError('network down'); }) as unknown as typeof fetch;
    const service = new KnirvModelsService({ client: new ModelsClient(BASE, failing), sdk });

    await expect(service.generateVerifiedText('hello')).resolves.toBeNull();
  });

  it('describes images through HEAD and returns null without a vision backend', async () => {
    const ok = serviceWith({
      '/models/head/visual-reason': { status: 200, body: { operation: 'describe', result: { description: 'a red bridge' } } },
      '/models/head/describe': { status: 200, body: { operation: 'describe', result: { description: 'a red bridge' } } },
    });
    await expect(ok.service.describeImages([{ url: 'https://img.test/a.png' }])).resolves.toBe('a red bridge');

    const none = serviceWith({ '/models/head/describe': { status: 503, body: { error: 'none' } } });
    await expect(none.service.describeImages([{ url: 'https://img.test/a.png' }])).resolves.toBeNull();
  });
});
