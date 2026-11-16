import * as fs from 'fs';
import * as path from 'path';
import { publicView, encodeFramePayload, decodeFramePayload, FramePayload, PayloadError, PAYLOAD_MODALITY } from './payload';

// Shared cross-language fixture (also read by rust/src/payload.rs and go/pkg/nrv/payload_test.go).
const SHARED_CASES = path.resolve(__dirname, '../../../../../testdata/framepayload/cases.json');

const enc = (p: unknown) => new TextEncoder().encode(JSON.stringify(p));
const text = (p: FramePayload) => new TextDecoder().decode(encodeFramePayload(p));

const full: FramePayload = {
  token_window: { tokens: [1, 2, 3], start: 0, brackets: 2 },
  error_test_case: { id: 't1', input: '2+2', expected: '4' },
  trl_sample: { prompt: 'p', chosen: 'good', rejected: 'bad' },
  supervision_episode: {
    state: 'node healthy',
    choices: ['no-op', 'drain'],
    decision: 'no-op',
    tool_trace: [{ tool: 'manifest.get', tier: 0, outcome: 'execute' }],
    outcome: 'ok',
  },
  perceptual: { image_hashes: ['ab'], boxes: [{ x: 0.1, y: 0.2, w: 0.3, h: 0.4, label: 'cat' }], ocr: 'hi', embedding: [0.5] },
};

describe('FramePayload (mirrors go/pkg/nrv/payload.go and rust/src/payload.rs)', () => {
  test('round trips all five fields', () => {
    const back = decodeFramePayload(encodeFramePayload(full));
    expect(back).toEqual(full);
    expect(back.token_window && back.error_test_case && back.trl_sample && back.supervision_episode && back.perceptual).toBeTruthy();
  });

  test('publicView strips only the expected answer and does not mutate', () => {
    const pv = publicView(full);
    expect(pv.error_test_case?.expected).toBeUndefined();
    expect(pv.error_test_case?.input).toBe('2+2');
    expect(pv.trl_sample).toEqual(full.trl_sample);
    expect(full.error_test_case?.expected).toBe('4');
  });

  test('empty payload encodes as {}', () => {
    expect(new TextDecoder().decode(encodeFramePayload({}))).toBe('{}');
  });

  test('payload modality key matches the other runtimes', () => {
    expect(PAYLOAD_MODALITY).toBe('payload');
  });

  test('token ids and start: min, max, max+1, negative', () => {
    expect(() => encodeFramePayload({ token_window: { tokens: [0, 4294967295], start: 4294967295, brackets: 1024 } })).not.toThrow();
    expect(() => encodeFramePayload({ token_window: { tokens: [4294967296] } })).toThrow(PayloadError);
    expect(() => encodeFramePayload({ token_window: { tokens: [-1] } })).toThrow(PayloadError);
    expect(() => encodeFramePayload({ token_window: { tokens: [1], start: 4294967296 } })).toThrow(PayloadError);
    expect(() => encodeFramePayload({ token_window: { tokens: [1], start: -1 } })).toThrow(PayloadError);
    expect(() => encodeFramePayload({ token_window: { tokens: [1.5] } })).toThrow(PayloadError);
  });

  test('brackets frame cap: 1024 ok, 1025 named error', () => {
    expect(() => encodeFramePayload({ token_window: { tokens: [1], brackets: 1024 } })).not.toThrow();
    try {
      encodeFramePayload({ token_window: { tokens: [1], brackets: 1025 } });
      throw new Error('expected throw');
    } catch (e) {
      expect(e).toBeInstanceOf(PayloadError);
      expect((e as PayloadError).field).toBe('token_window.brackets');
    }
    expect(() => decodeFramePayload(enc({ token_window: { tokens: [1], brackets: -1 } }))).toThrow(PayloadError);
  });

  test('tool tier is 0..=3: 0-3 ok, 4, 255, 256 and -1 rejected', () => {
    const ep = (tier: number) => ({ supervision_episode: { state: 's', tool_trace: [{ tool: 't', tier, outcome: 'o' }] } });
    for (const t of [0, 1, 2, 3]) expect(() => encodeFramePayload(ep(t))).not.toThrow();
    for (const t of [4, 255, 256, -1]) expect(() => encodeFramePayload(ep(t))).toThrow(PayloadError);
  });

  test('box: in-frame boxes ok; origin, extent, past-edge, negative and non-finite rejected', () => {
    const box = (b: object) => ({ perceptual: { boxes: [b as never] } });
    expect(() => encodeFramePayload(box({ x: 0, y: 0, w: 1, h: 1 }))).not.toThrow();
    expect(() => encodeFramePayload(box({ x: 0.25, y: 0.25, w: 0.75, h: 0.75 }))).not.toThrow();
    expect(() => encodeFramePayload(box({ x: 1, y: 1, w: 0, h: 0 }))).not.toThrow();
    expect(() => encodeFramePayload(box({ x: -2.5, y: 0, w: 0, h: 0 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: 1.5, y: 0, w: 0, h: 0 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: 0.6, y: 0, w: 0.5, h: 0.1 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: 0, y: 0.6, w: 0.1, h: 0.5 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: 0.5, y: 0.5, w: 0.5, h: 0.5 }))).not.toThrow();
    expect(() => encodeFramePayload(box({ x: 0.5, y: 0, w: 0.500001, h: 0.1 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: 0, y: 0.5, w: 0.1, h: 0.500001 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: 1, y: 1, w: -0.5, h: 1 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: 1, y: 1, w: 1, h: -1 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: NaN, y: 1, w: 1, h: 1 }))).toThrow(PayloadError);
    expect(() => encodeFramePayload(box({ x: 1, y: 1, w: Infinity, h: 1 }))).toThrow(PayloadError);
  });

  test('empty means absent: empty strings, arrays and sub-objects', () => {
    expect(text({ error_test_case: { id: '', input: '', expected: '' } })).toBe('{}');
    expect(text({ token_window: { tokens: [], start: 0, brackets: 0 } })).toBe('{}');
    expect(text({ perceptual: { image_hashes: [], boxes: [], ocr: '', embedding: [] } })).toBe('{}');
    expect(text({ supervision_episode: {} })).toBe('{}');
    expect(text({ trl_sample: { prompt: 'p', completion: '', chosen: '', rejected: '' } })).toBe('{"trl_sample":{"prompt":"p"}}');
    expect(decodeFramePayload(enc({ perceptual: { boxes: [] } }))).toEqual({});
    expect(decodeFramePayload(enc({ token_window: null }))).toEqual({});
  });

  test('non-object top level and invalid JSON are rejected', () => {
    expect(() => decodeFramePayload(enc(null))).toThrow(PayloadError);
    expect(() => decodeFramePayload(enc([]))).toThrow(PayloadError);
    expect(() => decodeFramePayload(new TextEncoder().encode('{bad'))).toThrow(PayloadError);
  });
});

/** Parsed-JSON equality: 1 and 1.0 are the same number. */
describe('shared conformance fixture (testdata/framepayload/cases.json)', () => {
  const doc = JSON.parse(fs.readFileSync(SHARED_CASES, 'utf8')) as {
    cases: { name: string; input: unknown; expect: 'ok' | 'error'; canonical?: unknown }[];
  };

  test.each(doc.cases.map((c) => [c.name, c] as const))('%s', (_name, c) => {
    const bytes = enc(c.input);
    if (c.expect === 'error') {
      expect(() => decodeFramePayload(bytes)).toThrow(PayloadError);
      return;
    }
    const decoded = decodeFramePayload(bytes);
    const encoded = JSON.parse(new TextDecoder().decode(encodeFramePayload(decoded)));
    expect(encoded).toEqual(c.canonical);
  });
});
