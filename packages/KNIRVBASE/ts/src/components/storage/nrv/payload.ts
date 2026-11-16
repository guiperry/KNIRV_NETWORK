/**
 * Uniform open payload every `.nrv` frame may carry. Binding of the Rust core
 * (`rust/src/payload.rs`, source of truth); widths, range checks and the
 * absent-means-empty rule mirror it. The shared fixture
 * `testdata/framepayload/cases.json` pins them across Rust, Go and TS.
 *
 * A frame indexes its encoded payload under PAYLOAD_MODALITY in its modalities map.
 */

export const PAYLOAD_MODALITY = 'payload';

/** u32 bounds: token ids and token-window start. */
export const U32_MAX = 4294967295;
/** u16 bound for brackets, further capped by the frame limit below. */
export const U16_MAX = 65535;
/** Producer cap on brackets in one frame (go/pkg/nrv MaxBracketsPerFrame). */
export const MAX_BRACKETS_PER_FRAME = 1024;
/** u8 width for tool-call tier; the valid set is 0..=MAX_TOOL_TIER. */
export const U8_MAX = 255;
/**
 * Highest tool tier. The tier set is 0..=3: 0 execute, 1 ask, 2 deny, 3 escalate
 * (owner-decided; same as rust/src/payload.rs MAX_TOOL_TIER).
 */
export const MAX_TOOL_TIER = 3;
/** Tolerance for x + w <= 1 and y + h <= 1 on normalised boxes (float rounding). */
export const BOX_EPSILON = 1e-9;

export interface TokenWindowPayload {
  tokens?: number[];
  start?: number;
  brackets?: number;
}

export interface ErrorTestCasePayload {
  id?: string;
  input?: string;
  expected?: string;
}

export interface TRLSamplePayload {
  prompt?: string;
  completion?: string;
  chosen?: string;
  rejected?: string;
}

export interface ToolCallPayload {
  tool?: string;
  tier?: number;
  outcome?: string;
}

export interface SupervisionEpisodePayload {
  state?: string;
  choices?: string[];
  decision?: string;
  tool_trace?: ToolCallPayload[];
  outcome?: string;
}

export interface BoxPayload {
  x?: number;
  y?: number;
  w?: number;
  h?: number;
  label?: string;
}

export interface PerceptualPayload {
  image_hashes?: string[];
  boxes?: BoxPayload[];
  ocr?: string;
  embedding?: number[];
}

export interface FramePayload {
  token_window?: TokenWindowPayload;
  error_test_case?: ErrorTestCasePayload;
  trl_sample?: TRLSamplePayload;
  supervision_episode?: SupervisionEpisodePayload;
  perceptual?: PerceptualPayload;
}

/** Thrown for a wrong JSON type, an out-of-range value, or a non-object payload. */
export class PayloadError extends Error {
  readonly field: string;
  constructor(field: string, detail: string) {
    super(field ? `nrv payload field ${field}: ${detail}` : `nrv payload: ${detail}`);
    this.name = 'PayloadError';
    this.field = field;
  }
}

function isPlainObject(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

function expectInt(v: unknown, field: string, max: number): number {
  if (typeof v !== 'number' || !Number.isInteger(v)) {
    throw new PayloadError(field, `expected an integer, got ${JSON.stringify(v)}`);
  }
  if (v < 0 || v > max) {
    throw new PayloadError(field, `${v} outside 0..=${max}`);
  }
  return v;
}

function expectStr(v: unknown, field: string): string {
  if (typeof v !== 'string') throw new PayloadError(field, 'expected a string');
  return v;
}

function expectNum(v: unknown, field: string): number {
  if (typeof v !== 'number') throw new PayloadError(field, 'expected a number');
  if (!Number.isFinite(v)) throw new PayloadError(field, `${v} is not finite`);
  return v;
}

function optObject(v: unknown, field: string): Record<string, unknown> | undefined {
  if (v === undefined || v === null) return undefined;
  if (!isPlainObject(v)) throw new PayloadError(field, 'expected an object');
  return v;
}

function strList(v: unknown, field: string): string[] {
  if (v === undefined || v === null) return [];
  if (!Array.isArray(v)) throw new PayloadError(field, 'expected an array');
  return v.map((x, i) => expectStr(x, `${field}[${i}]`));
}

function arr(v: unknown, field: string): unknown[] {
  if (v === undefined || v === null) return [];
  if (!Array.isArray(v)) throw new PayloadError(field, 'expected an array');
  return v;
}

/**
 * Converts untrusted JSON to a normalized, validated payload: wrong types and
 * out-of-range values throw; empty strings, arrays and sub-objects become absent.
 */
function parseFramePayload(raw: unknown): FramePayload {
  if (!isPlainObject(raw)) throw new PayloadError('', 'payload must be a JSON object');
  const out: FramePayload = {};

  const tw = optObject(raw.token_window, 'token_window');
  if (tw) {
    const tokens = arr(tw.tokens, 'token_window.tokens').map((t, i) =>
      expectInt(t, `token_window.tokens[${i}]`, U32_MAX),
    );
    const start = tw.start === undefined || tw.start === null ? 0 : expectInt(tw.start, 'token_window.start', U32_MAX);
    const brackets =
      tw.brackets === undefined || tw.brackets === null ? 0 : expectInt(tw.brackets, 'token_window.brackets', U16_MAX);
    if (brackets > MAX_BRACKETS_PER_FRAME) {
      throw new PayloadError('token_window.brackets', `${brackets} exceeds the per-frame cap of ${MAX_BRACKETS_PER_FRAME}`);
    }
    if (tokens.length || start !== 0 || brackets !== 0) {
      out.token_window = {
        tokens: tokens.length ? tokens : undefined,
        start,
        brackets,
      };
    }
  }

  const ec = optObject(raw.error_test_case, 'error_test_case');
  if (ec) {
    const id = ec.id === undefined || ec.id === null ? '' : expectStr(ec.id, 'error_test_case.id');
    const input = ec.input === undefined || ec.input === null ? '' : expectStr(ec.input, 'error_test_case.input');
    const expected =
      ec.expected === undefined || ec.expected === null ? '' : expectStr(ec.expected, 'error_test_case.expected');
    if (id || input || expected) {
      out.error_test_case = { id: id || undefined, input: input || undefined, expected: expected || undefined };
    }
  }

  const trl = optObject(raw.trl_sample, 'trl_sample');
  if (trl) {
    const s = (k: keyof TRLSamplePayload) =>
      trl[k] === undefined || trl[k] === null ? '' : expectStr(trl[k], `trl_sample.${k}`);
    const v = { prompt: s('prompt'), completion: s('completion'), chosen: s('chosen'), rejected: s('rejected') };
    if (v.prompt || v.completion || v.chosen || v.rejected) {
      out.trl_sample = {
        prompt: v.prompt || undefined,
        completion: v.completion || undefined,
        chosen: v.chosen || undefined,
        rejected: v.rejected || undefined,
      };
    }
  }

  const se = optObject(raw.supervision_episode, 'supervision_episode');
  if (se) {
    const state = se.state === undefined || se.state === null ? '' : expectStr(se.state, 'supervision_episode.state');
    const decision =
      se.decision === undefined || se.decision === null ? '' : expectStr(se.decision, 'supervision_episode.decision');
    const outcome =
      se.outcome === undefined || se.outcome === null ? '' : expectStr(se.outcome, 'supervision_episode.outcome');
    const choices = strList(se.choices, 'supervision_episode.choices');
    const toolTrace = arr(se.tool_trace, 'supervision_episode.tool_trace').map((t, i) => {
      const f = `supervision_episode.tool_trace[${i}]`;
      if (!isPlainObject(t)) throw new PayloadError(f, 'expected an object');
      const tool = t.tool === undefined || t.tool === null ? '' : expectStr(t.tool, `${f}.tool`);
      const tier = t.tier === undefined || t.tier === null ? 0 : expectInt(t.tier, `${f}.tier`, MAX_TOOL_TIER);
      const to = t.outcome === undefined || t.outcome === null ? '' : expectStr(t.outcome, `${f}.outcome`);
      return { tool: tool || undefined, tier, outcome: to || undefined } as ToolCallPayload;
    });
    if (state || decision || outcome || choices.length || toolTrace.length) {
      out.supervision_episode = {
        state: state || undefined,
        choices: choices.length ? choices : undefined,
        decision: decision || undefined,
        tool_trace: toolTrace.length ? toolTrace : undefined,
        outcome: outcome || undefined,
      };
    }
  }

  const pc = optObject(raw.perceptual, 'perceptual');
  if (pc) {
    const hashes = strList(pc.image_hashes, 'perceptual.image_hashes');
    const boxes = arr(pc.boxes, 'perceptual.boxes').map((b, i) => {
      const f = `perceptual.boxes[${i}]`;
      if (!isPlainObject(b)) throw new PayloadError(f, 'expected an object');
      const x = expectNum(b.x ?? 0, `${f}.x`);
      const y = expectNum(b.y ?? 0, `${f}.y`);
      const w = expectNum(b.w ?? 0, `${f}.w`);
      const h = expectNum(b.h ?? 0, `${f}.h`);
      // Normalised frame coordinates: each in 0..=1, and the box stays inside the frame.
      for (const [name, v] of [['x', x], ['y', y], ['w', w], ['h', h]] as const) {
        if (v < 0 || v > 1) throw new PayloadError(`${f}.${name}`, `${v} is outside the normalised range 0..=1`);
      }
      if (x + w > 1 + BOX_EPSILON) throw new PayloadError(`${f}.x`, `x + w = ${x + w} extends past the frame (1)`);
      if (y + h > 1 + BOX_EPSILON) throw new PayloadError(`${f}.y`, `y + h = ${y + h} extends past the frame (1)`);
      const label = b.label === undefined || b.label === null ? '' : expectStr(b.label, `${f}.label`);
      return { x, y, w, h, label: label || undefined } as BoxPayload;
    });
    const ocr = pc.ocr === undefined || pc.ocr === null ? '' : expectStr(pc.ocr, 'perceptual.ocr');
    const embedding = arr(pc.embedding, 'perceptual.embedding').map((e, i) =>
      expectNum(e, `perceptual.embedding[${i}]`),
    );
    if (hashes.length || boxes.length || ocr || embedding.length) {
      out.perceptual = {
        image_hashes: hashes.length ? hashes : undefined,
        boxes: boxes.length ? boxes : undefined,
        ocr: ocr || undefined,
        embedding: embedding.length ? embedding : undefined,
      };
    }
  }

  return out;
}

/** The payload as a test-taker may see it: the ErrorTestCase expected answer is removed. */
export function publicView(p: FramePayload): FramePayload {
  if (!p.error_test_case) return { ...p };
  const { expected: _expected, ...rest } = p.error_test_case;
  return { ...p, error_test_case: rest };
}

/** Encodes after normalizing and validating; invalid payloads are refused. */
export function encodeFramePayload(p: FramePayload): Uint8Array {
  const normalized = parseFramePayload(p);
  return new TextEncoder().encode(JSON.stringify(normalized));
}

/** Decodes bytes: must be a JSON object; values are type- and range-checked. */
export function decodeFramePayload(bytes: Uint8Array): FramePayload {
  const text = new TextDecoder().decode(bytes);
  let raw: unknown;
  try {
    raw = JSON.parse(text);
  } catch (e) {
    throw new PayloadError('', `invalid JSON: ${(e as Error).message}`);
  }
  return parseFramePayload(raw);
}
