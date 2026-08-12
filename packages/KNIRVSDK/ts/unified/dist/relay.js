// knirv.controller.relay-envelope.v1 (plan section 8, Phase 8 task 2).
// Byte layout must stay identical to
// KNIRV_NETWORK/packages/KNIRVSDK/go/signing/relay.go.
export const RELAY_ENVELOPE_SCHEMA_VERSION = 'knirv.controller.relay-envelope.v1';
export const RELAY_TARGET_DVE_EXPERT_ADVISOR = 'dve_expert_advisor';
export const RELAY_TARGET_CLI_SUPERVISOR = 'cli_supervisor';
function concat(...values) {
    const length = values.reduce((total, value) => total + value.length, 0);
    const result = new Uint8Array(length);
    let offset = 0;
    for (const value of values) {
        result.set(value, offset);
        offset += value.length;
    }
    return result;
}
function varint(value) {
    let current = BigInt(value);
    if (current < 0n)
        current = BigInt.asUintN(64, current);
    const output = [];
    do {
        let next = Number(current & 0x7fn);
        current >>= 7n;
        if (current !== 0n)
            next |= 0x80;
        output.push(next);
    } while (current !== 0n);
    return Uint8Array.from(output);
}
function tag(field, wireType) {
    return varint(BigInt((field << 3) | wireType));
}
function bytesField(field, value) {
    if (!value?.length)
        return new Uint8Array();
    return concat(tag(field, 2), varint(value.length), value);
}
const encoder = new TextEncoder();
function stringField(field, value) {
    return value ? bytesField(field, encoder.encode(value)) : new Uint8Array();
}
function uintField(field, value) {
    if (value === undefined || BigInt(value) === 0n)
        return new Uint8Array();
    return concat(tag(field, 0), varint(value));
}
function normalizeRelayEnvelope(e) {
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
export function marshalRelayEnvelope(envelope) {
    const e = normalizeRelayEnvelope(envelope);
    return concat(stringField(1, e.schemaVersion), stringField(2, e.requestId), stringField(3, e.userSubject), stringField(4, e.deviceId), stringField(5, e.dveId), stringField(6, e.targetType), stringField(7, e.targetId), stringField(8, e.capability), uintField(9, e.sequence), uintField(10, e.leaseEpoch), varintAsInt(11, e.issuedAtUnix), varintAsInt(12, e.expiresAtUnix), stringField(13, e.payloadDigest));
}
// issuedAtUnix/expiresAtUnix are always positive in a valid envelope, so
// plain uintField is bit-identical to Go's appendInt64 for these fields.
function varintAsInt(field, value) {
    return uintField(field, value);
}
function readVarint(data, start) {
    let value = 0n;
    let shift = 0n;
    let i = start;
    for (; i < data.length; i++) {
        const byte = data[i];
        value |= BigInt(byte & 0x7f) << shift;
        if ((byte & 0x80) === 0)
            return { value, next: i + 1 };
        shift += 7n;
        if (shift >= 64n)
            throw new Error('varint overflow');
    }
    throw new Error('truncated varint');
}
function readFields(data) {
    const fields = [];
    let i = 0;
    while (i < data.length) {
        const { value: key, next: afterKey } = readVarint(data, i);
        i = afterKey;
        const number = Number(key >> 3n);
        const wireType = Number(key & 0x7n);
        if (wireType === 0) {
            const { value, next } = readVarint(data, i);
            fields.push({ number, wireType, uintValue: value });
            i = next;
        }
        else if (wireType === 2) {
            const { value: length, next: afterLen } = readVarint(data, i);
            const end = afterLen + Number(length);
            if (end > data.length)
                throw new Error('truncated length-delimited field');
            fields.push({ number, wireType, bytesValue: data.subarray(afterLen, end) });
            i = end;
        }
        else {
            throw new Error(`unsupported wire type ${wireType}`);
        }
    }
    return fields;
}
const decoder = new TextDecoder();
export function parseRelayEnvelope(data) {
    const fields = readFields(data);
    const byNumber = new Map();
    for (const field of fields)
        if (!byNumber.has(field.number))
            byNumber.set(field.number, field);
    const stringOf = (n) => {
        const field = byNumber.get(n);
        return field?.bytesValue ? decoder.decode(field.bytesValue) : '';
    };
    const uintOf = (n) => byNumber.get(n)?.uintValue ?? 0n;
    const envelope = {
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
