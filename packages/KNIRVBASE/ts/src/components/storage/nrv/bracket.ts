export type DeltaType = 'I' | 'P';

export const BracketSize = 80;

export interface BracketMeta {
  id: string;
  type: DeltaType;
  anchorId?: string;
  offset: number;
  driftScore: number;
}

export interface Bracket {
  id: string;
  lshSalt: number;
  projections: Uint8Array;
  subSecondUS: number;
  asicLoops: number;
  goldenSeed: number;
  meta?: BracketMeta;
}

export interface LinguisticMapping {
  token: string;
  unit: string;
}

export interface ThermoAtmosphere {
  avgTempC: number;
  peakVoltV: number;
  clockMHz: number;
}

export interface Z3Result {
  status: string;
  relevance: number;
}

export interface BracketBinaryMap {
  count: number;
  offset: number;
  length: number;
}

export interface FrameEntry {
  id: string;
  timestampUnix: number;
  tombstone?: number;
  linguistic: LinguisticMapping;
  thermo: ThermoAtmosphere;
  z3: Z3Result;
  brackets: BracketBinaryMap;
  bracketIndex: BracketMeta[];
}

export interface GlobalMetrics {
  avgTempCMean: number;
  avgTempCMax: number;
  peakVoltVMean: number;
  clockMHzMean: number;
  totalBracketCount: number;
  validFrameCount: number;
  invalidFrameCount: number;
  compactedAt?: string;
  ergoRankSum: number;
  verifiedFrameCount: number;
}

export function createDefaultGlobalMetrics(): GlobalMetrics {
  return {
    avgTempCMean: 0,
    avgTempCMax: 0,
    peakVoltVMean: 0,
    clockMHzMean: 0,
    totalBracketCount: 0,
    validFrameCount: 0,
    invalidFrameCount: 0,
    ergoRankSum: 0,
    verifiedFrameCount: 0,
  };
}

export function encodeBracket(b: Bracket): Uint8Array {
  const buf = new Uint8Array(BracketSize);
  const view = new DataView(buf.buffer);

  view.setUint32(0, b.lshSalt, true);
  buf.set(b.projections.slice(0, 64), 4);
  view.setUint32(68, b.subSecondUS, true);
  view.setUint32(72, b.asicLoops, true);
  view.setUint32(76, b.goldenSeed, true);

  return buf;
}

export function decodeBracket(buf: Uint8Array): Bracket {
  if (buf.length < BracketSize) {
    throw new Error(`bracket buffer too small: got ${buf.length}, want ${BracketSize}`);
  }

  const view = new DataView(buf.buffer, buf.byteOffset);

  const lshSalt = view.getUint32(0, true);
  const projections = buf.slice(4, 68);
  const subSecondUS = view.getUint32(68, true);
  const asicLoops = view.getUint32(72, true);
  const goldenSeed = view.getUint32(76, true);

  return {
    id: '',
    lshSalt,
    projections,
    subSecondUS,
    asicLoops,
    goldenSeed,
  };
}

export function xorProjections(current: Uint8Array, anchor: Uint8Array): Uint8Array {
  if (current.length !== 64 || anchor.length !== 64) {
    throw new Error('projections must be 64 bytes');
  }

  const result = new Uint8Array(64);
  for (let i = 0; i < 64; i++) {
    result[i] = current[i] ^ anchor[i];
  }
  return result;
}

export function applyProjectionDelta(delta: Uint8Array, anchor: Uint8Array): Uint8Array {
  if (delta.length !== 64 || anchor.length !== 64) {
    throw new Error('delta and anchor must both be 64 bytes');
  }

  const result = new Uint8Array(64);
  for (let i = 0; i < 64; i++) {
    result[i] = delta[i] ^ anchor[i];
  }
  return result;
}