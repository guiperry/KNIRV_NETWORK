export type DeltaType = 'I' | 'P';

export const DeltaTypeI: DeltaType = 'I';
export const DeltaTypeP: DeltaType = 'P';

export const BracketSize = 80;

export interface SyntacticProfile {
  syntactic: uint8;
  depHead: int8;
}

export interface IntentDomain {
  intentFlags: uint8;
  domainSig: uint16;
}

export interface BracketMeta {
  id: string;
  type: DeltaType;
  anchorId?: string;
  offset: number;
  driftScore: number;
}

export interface Bracket {
  id: string;
  projections: Uint8Array;
  subSecondUS: uint32;
  syntactic: uint8;
  depHead: int8;
  intentFlags: uint8;
  domainSig: uint16;
  goldenSeed: uint32;
  memory: Uint8Array;
  lshSalt: uint32;
  reserved: Uint8Array;
  meta?: BracketMeta;
  frameId: string;
  frameUnix: int64;
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
  };
}

type uint8 = number;
type uint16 = number;
type uint32 = number;
type int8 = number;
type int64 = number;

export function packSyntacticByte(post: uint8, tense: uint8, plurality: uint8): uint8 {
  return (post & 0x0F) | ((tense & 3) << 4) | ((plurality & 3) << 6);
}

export function unpackSyntacticByte(b: uint8): { post: uint8; tense: uint8; plurality: uint8 } {
  return {
    post: b & 0x0F,
    tense: (b >> 4) & 3,
    plurality: (b >> 6) & 3,
  };
}

export function getPOSTag(b: Bracket): uint8 {
  return b.syntactic & 0x0F;
}

export function setPOSTag(b: Bracket, v: uint8): void {
  b.syntactic = (b.syntactic & 0xF0) | (v & 0x0F);
}

export function getTense(b: Bracket): uint8 {
  return (b.syntactic >> 4) & 0x03;
}

export function setTense(b: Bracket, v: uint8): void {
  b.syntactic = (b.syntactic & 0xCF) | ((v & 0x03) << 4);
}

export function getPlurality(b: Bracket): uint8 {
  return (b.syntactic >> 6) & 0x03;
}

export function setPlurality(b: Bracket, v: uint8): void {
  b.syntactic = (b.syntactic & 0x3F) | ((v & 0x03) << 6);
}

export function encodeBracket(b: Bracket): Uint8Array {
  const buf = new Uint8Array(BracketSize);
  const view = new DataView(buf.buffer);

  buf.set(b.projections.slice(0, 32), 0);

  view.setUint32(32, b.subSecondUS, true);

  buf[36] = b.syntactic;
  buf[37] = b.depHead;
  buf[38] = b.intentFlags;

  view.setUint16(39, b.domainSig, true);

  view.setUint32(41, b.goldenSeed, true);

  buf.set(b.memory.slice(0, 14), 45);

  view.setUint32(59, b.lshSalt, true);

  buf.set(b.reserved.slice(0, 17), 63);

  return buf;
}

export function decodeBracket(buf: Uint8Array | ArrayBuffer): Bracket {
  const arr = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  if (arr.length < BracketSize) {
    throw new Error(`bracket buffer too small: got ${arr.length}, want ${BracketSize}`);
  }

  const view = new DataView(arr.buffer, arr.byteOffset);

  const projections = arr.slice(0, 32);
  const subSecondUS = view.getUint32(32, true);
  const syntactic = arr[36];
  const depHead = arr[37] > 127 ? arr[37] - 256 : arr[37];
  const intentFlags = arr[38];
  const domainSig = view.getUint16(39, true);
  const goldenSeed = view.getUint32(41, true);
  const memory = arr.slice(45, 59);
  const lshSalt = view.getUint32(59, true);
  const reserved = arr.slice(63, 80);

  return {
    id: '',
    projections,
    subSecondUS,
    syntactic,
    depHead,
    intentFlags,
    domainSig,
    goldenSeed,
    memory,
    lshSalt,
    reserved,
    frameId: '',
    frameUnix: 0,
  };
}

export function xorProjections(current: Uint8Array, anchor: Uint8Array): Uint8Array {
  if (current.length !== 32 || anchor.length !== 32) {
    throw new Error('projections must be 32 bytes');
  }

  const result = new Uint8Array(32);
  for (let i = 0; i < 32; i++) {
    result[i] = current[i] ^ anchor[i];
  }
  return result;
}

export function applyProjectionDelta(delta: Uint8Array, anchor: Uint8Array): Uint8Array {
  if (delta.length !== 32 || anchor.length !== 32) {
    throw new Error('delta and anchor must both be 32 bytes');
  }

  const result = new Uint8Array(32);
  for (let i = 0; i < 32; i++) {
    result[i] = delta[i] ^ anchor[i];
  }
  return result;
}

export function encodeSyntactic(sp: SyntacticProfile): uint8 {
  return sp.syntactic;
}

export function decodeSyntactic(val: uint8): SyntacticProfile {
  return {
    syntactic: val,
    depHead: 0,
  };
}

export function encodeIntentDomain(id: IntentDomain): uint32 {
  return (id.intentFlags & 0xFF) | (id.domainSig << 16);
}

export function decodeIntentDomain(val: uint32): IntentDomain {
  return {
    intentFlags: val & 0xFF,
    domainSig: (val >> 16) & 0xFFFF,
  };
}
