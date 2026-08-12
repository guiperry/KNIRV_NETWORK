// knirv.controller.relay-envelope.v1 (plan section 8, Phase 8 task 2).
// Byte layout must stay identical to
// KNIRV_NETWORK/packages/KNIRVSDK/go/signing/relay.go.

export const RELAY_ENVELOPE_SCHEMA_VERSION = 'knirv.controller.relay-envelope.v1';

export const RELAY_TARGET_DVE_EXPERT_ADVISOR = 'dve_expert_advisor';
export const RELAY_TARGET_CLI_SUPERVISOR = 'cli_supervisor';

export type RelayUint64 = number | string | bigint;

export interface RelayEnvelope {
  schemaVersion?: string;
  requestId: string;
  userSubject: string;
  deviceId: string;
  dveId?: string;
  targetType: string; // dve_expert_advisor | cli_supervisor
  targetId: string;
  capability: string;
  sequence: RelayUint64;
  leaseEpoch?: RelayUint64;
  issuedAtUnix: RelayUint64;
  expiresAtUnix: RelayUint64;
  payloadDigest: string; // sha256:<hex> of the opaque relay message payload
}

function concat(...values: Uint8Array[]): Uint8Array {
  const length = values.reduce((total, value) => total + value.length, 0);
  const result = new Uint8Array(length);
  let offset = 0;
  for (const value of values) {
    result.set(value, offset);
    offset += value.length;
  }
  return result;
}

function varint(value: RelayUint64): Uint8Array {
  let current = BigInt(value);
  if (current < 0n) current = BigInt.asUintN(64, current);
  const output: number[] = [];
  do {
    let next = Number(current & 0x7fn);
    current >>= 7n;
    if (current !== 0n) next |= 0x80;
    output.push(next);
  } while (current !== 0n);
  return Uint8Array.from(output);
}

function tag(field: number, wireType: 0 | 2): Uint8Array {
  return varint(BigInt((field << 3) | wireType));
}

function bytesField(field: number, value?: Uint8Array): Uint8Array {
  if (!value?.length) return new Uint8Array();
  return concat(tag(field, 2), varint(value.length), value);
}

const encoder = new TextEncoder();

function stringField(field: number, value?: string): Uint8Array {
  return value ? bytesField(field, encoder.encode(value)) : new Uint8Array();
}

function uintField(field: number, value?: RelayUint64): Uint8Array {
  if (value === undefined || BigInt(value) === 0n) return new Uint8Array();
  return concat(tag(field, 0), varint(value));
}

function normalizeRelayEnvelope(e: RelayEnvelope): RelayEnvelope {
  const schemaVersion = e.schemaVersion || RELAY_ENVELOPE_SCHEMA_VERSION;
  if (schemaVersion !== RELAY_ENVELOPE_SCHEMA_VERSION) {
    throw new Error(`Unsupported relay envelope schema: ${schemaVersion}`);
  }
  if (!e.requestId.trim() || !e.userSubject.trim() || !e.deviceId.trim()) {
    throw new Error('requestId, userSubject, and deviceId are required');
  }
  if (e.targetType !== RELAY_TARGET_DVE_EXPERT_ADVISOR && e.targetType !== RELAY_TARGET_CLI_SUPERVISOR) {
    throw new Error('targetType must be dve_expert_advisor or cli_supervisor');
  }
  if (!e.targetId.trim() || !e.capability.trim()) {
    throw new Error('targetId and capability are required');
  }
  if (BigInt(e.sequence) === 0n) {
    throw new Error('sequence must be a positive monotonic counter');
  }
  if (BigInt(e.issuedAtUnix) <= 0n || BigInt(e.expiresAtUnix) <= BigInt(e.issuedAtUnix)) {
    throw new Error('valid issuedAtUnix and expiresAtUnix are required');
  }
  if (!e.payloadDigest.trim()) {
    throw new Error('payloadDigest is required');
  }
  return { ...e, schemaVersion };
}

export function marshalRelayEnvelope(envelope: RelayEnvelope): Uint8Array {
  const e = normalizeRelayEnvelope(envelope);
  return concat(
    stringField(1, e.schemaVersion),
    stringField(2, e.requestId),
    stringField(3, e.userSubject),
    stringField(4, e.deviceId),
    stringField(5, e.dveId),
    stringField(6, e.targetType),
    stringField(7, e.targetId),
    stringField(8, e.capability),
    uintField(9, e.sequence),
    uintField(10, e.leaseEpoch),
    varintAsInt(11, e.issuedAtUnix),
    varintAsInt(12, e.expiresAtUnix),
    stringField(13, e.payloadDigest),
  );
}

// issuedAtUnix/expiresAtUnix are always positive in a valid envelope, so
// plain uintField is bit-identical to Go's appendInt64 for these fields.
function varintAsInt(field: number, value: RelayUint64): Uint8Array {
  return uintField(field, value);
}

interface WireField {
  number: number;
  wireType: 0 | 2;
  bytesValue?: Uint8Array;
  uintValue?: bigint;
}

function readVarint(data: Uint8Array, start: number): { value: bigint; next: number } {
  let value = 0n;
  let shift = 0n;
  let i = start;
  for (; i < data.length; i++) {
    const byte = data[i];
    value |= BigInt(byte & 0x7f) << shift;
    if ((byte & 0x80) === 0) return { value, next: i + 1 };
    shift += 7n;
    if (shift >= 64n) throw new Error('varint overflow');
  }
  throw new Error('truncated varint');
}

function readFields(data: Uint8Array): WireField[] {
  const fields: WireField[] = [];
  let i = 0;
  while (i < data.length) {
    const { value: key, next: afterKey } = readVarint(data, i);
    i = afterKey;
    const number = Number(key >> 3n);
    const wireType = Number(key & 0x7n) as 0 | 2;
    if (wireType === 0) {
      const { value, next } = readVarint(data, i);
      fields.push({ number, wireType, uintValue: value });
      i = next;
    } else if (wireType === 2) {
      const { value: length, next: afterLen } = readVarint(data, i);
      const end = afterLen + Number(length);
      if (end > data.length) throw new Error('truncated length-delimited field');
      fields.push({ number, wireType, bytesValue: data.subarray(afterLen, end) });
      i = end;
    } else {
      throw new Error(`unsupported wire type ${wireType}`);
    }
  }
  return fields;
}

const decoder = new TextDecoder();

export function parseRelayEnvelope(data: Uint8Array): RelayEnvelope {
  const fields = readFields(data);
  const byNumber = new Map<number, WireField>();
  for (const field of fields) if (!byNumber.has(field.number)) byNumber.set(field.number, field);

  const stringOf = (n: number) => {
    const field = byNumber.get(n);
    return field?.bytesValue ? decoder.decode(field.bytesValue) : '';
  };
  const uintOf = (n: number) => byNumber.get(n)?.uintValue ?? 0n;

  const envelope: RelayEnvelope = {
    schemaVersion: stringOf(1),
    requestId: stringOf(2),
    userSubject: stringOf(3),
    deviceId: stringOf(4),
    dveId: stringOf(5),
    targetType: stringOf(6),
    targetId: stringOf(7),
    capability: stringOf(8),
    sequence: uintOf(9),
    leaseEpoch: uintOf(10),
    issuedAtUnix: uintOf(11),
    expiresAtUnix: uintOf(12),
    payloadDigest: stringOf(13),
  };
  return normalizeRelayEnvelope(envelope);
}
