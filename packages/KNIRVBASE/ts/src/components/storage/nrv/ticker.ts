// FrameTicker — port of Go's internal/storage/nrv_ticker.go.
// Brackets are buffered and flushed as one frame per tick. The first bracket
// (and every 50th, and any bracket that drifts past the threshold) is an
// I-bracket; the rest are P-brackets stored as an XOR delta against the last
// I-bracket. Slots 6-8 carry a rolling XOR history of everything seen so far.
import * as crypto from 'crypto';
import { Bracket, BracketMeta, DeltaType, LinguisticMapping, ThermoAtmosphere, encodeBracket, xorProjections } from './bracket';
import { NRVWriter } from './writer';

export const I_FRAME_INTERVAL = 50;
export const DRIFT_THRESHOLD = 0.25;

interface Pending {
  bracket: Bracket;
  deltaType: DeltaType;
  anchorId?: string;
  driftScore: number;
  historyXOR: [number, number, number];
}

function cloneBracket(b: Bracket): Bracket {
  return {
    ...b,
    projections: b.projections.slice(),
    memory: b.memory.slice(),
    reserved: b.reserved.slice(),
    meta: undefined,
  };
}

/** Euclidean distance between the 16 little-endian uint16 projections, normalised to [0,1]. */
export function euclideanDrift(a: Uint8Array, b: Uint8Array): number {
  let sum = 0;
  for (let i = 0; i < 16; i++) {
    const av = (a[i * 2] | (a[i * 2 + 1] << 8)) / 65535.0;
    const bv = (b[i * 2] | (b[i * 2 + 1] << 8)) / 65535.0;
    const d = av - bv;
    sum += d * d;
  }
  return Math.sqrt(sum);
}

function hashBracket(b: Bracket): number {
  let h = b.goldenSeed >>> 0;
  const dv = new DataView(b.projections.buffer, b.projections.byteOffset, 32);
  for (let i = 0; i < 32; i += 4) {
    h = (h ^ dv.getUint32(i, true)) >>> 0;
  }
  return (h ^ b.subSecondUS) >>> 0;
}

/** Mean of the samples with float32 accumulation, as Go does. */
export function aggregateThermo(samples: ThermoAtmosphere[]): ThermoAtmosphere {
  if (samples.length === 0) return { avgTempC: 0, peakVoltV: 0, clockMHz: 0 };
  let t = 0;
  let v = 0;
  let c = 0;
  for (const s of samples) {
    t = Math.fround(t + Math.fround(s.avgTempC));
    v = Math.fround(v + Math.fround(s.peakVoltV));
    c = Math.fround(c + Math.fround(s.clockMHz));
  }
  const n = Math.fround(samples.length);
  return {
    avgTempC: Math.fround(t / n),
    peakVoltV: Math.fround(v / n),
    clockMHz: Math.fround(c / n),
  };
}

function writeHistory(dst: Uint8Array, hx: [number, number, number]): void {
  const dv = new DataView(dst.buffer, dst.byteOffset, dst.byteLength);
  dv.setUint32(0, hx[0], true);
  dv.setUint32(4, hx[1], true);
  dv.setUint32(8, hx[2], true);
  dst[12] = 0;
  dst[13] = 0;
}

export class FrameTicker {
  private pending: Pending[] = [];
  private thermoSamples: ThermoAtmosphere[] = [];
  private linguistic: LinguisticMapping = { token: '', unit: '' };
  private lastI: Bracket | null = null;
  private lastIId = '';
  private count = 0;
  private history: [number, number, number] = [0, 0, 0];
  private timer: ReturnType<typeof setInterval>;
  private flushChain: Promise<void> = Promise.resolve();
  private lastErr: Error | null = null;
  private stopped = false;

  constructor(private writer: NRVWriter, intervalMs: number) {
    this.timer = setInterval(() => void this.scheduleFlush(), intervalMs);
    this.timer.unref?.();
  }

  /** Most recent flush error, if any. */
  lastFlushError(): Error | null {
    return this.lastErr;
  }

  setLinguistic(token: string, unit: string): void {
    this.linguistic = { token, unit };
  }

  appendBracket(b: Bracket, thermo: ThermoAtmosphere): void {
    if (this.stopped) throw new Error('ticker stopped');
    if (!b.id) b.id = crypto.randomUUID();

    const historyXOR = this.computeMemoryXOR(b);
    let deltaType: DeltaType;
    let anchorId: string | undefined;
    let driftScore = 0;

    if (this.lastI === null || this.count % I_FRAME_INTERVAL === 0) {
      deltaType = 'I';
      this.lastI = cloneBracket(b);
      this.lastIId = b.id;
    } else {
      driftScore = euclideanDrift(b.projections, this.lastI.projections);
      if (driftScore > DRIFT_THRESHOLD) {
        deltaType = 'I';
        this.lastI = cloneBracket(b);
        this.lastIId = b.id;
      } else {
        deltaType = 'P';
        anchorId = this.lastIId;
        b.projections = xorProjections(b.projections, this.lastI.projections);
      }
    }

    this.pending.push({ bracket: b, deltaType, anchorId, driftScore, historyXOR });
    this.thermoSamples.push(thermo);
    this.count++;
  }

  private computeMemoryXOR(b: Bracket): [number, number, number] {
    const s6 = (this.history[0] ^ hashBracket(b)) >>> 0;
    const s7 = (this.history[1] ^ s6) >>> 0;
    const s8 = (this.history[2] ^ s7) >>> 0;
    this.history = [s6, s7, s8];
    return [s6, s7, s8];
  }

  /** Write pending brackets as a frame now; rejects with the write's error. */
  async flush(): Promise<void> {
    await this.scheduleFlush();
    if (this.lastErr) throw this.lastErr;
  }

  /** Stop the timer and write whatever is pending. */
  async stop(): Promise<void> {
    this.stopped = true;
    clearInterval(this.timer);
    await this.scheduleFlush();
  }

  private scheduleFlush(): Promise<void> {
    this.flushChain = this.flushChain.then(() => this.doFlush());
    return this.flushChain;
  }

  private async doFlush(): Promise<void> {
    const pending = this.pending;
    const thermo = this.thermoSamples;
    const linguistic = this.linguistic;
    this.pending = [];
    this.thermoSamples = [];
    if (pending.length === 0) return;

    const metas: BracketMeta[] = pending.map((p, i) => {
      writeHistory(p.bracket.memory, p.historyXOR);
      return { id: p.bracket.id, type: p.deltaType, anchorId: p.anchorId, offset: i * 80, driftScore: p.driftScore };
    });
    const buf = new Uint8Array(pending.length * 80);
    pending.forEach((p, i) => buf.set(encodeBracket(p.bracket), i * 80));

    try {
      await this.writer.appendBrackets(crypto.randomUUID(), buf, metas, aggregateThermo(thermo), linguistic);
      this.lastErr = null;
    } catch (e: any) {
      this.lastErr = e instanceof Error ? e : new Error(String(e));
    }
  }
}
