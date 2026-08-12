// Canonical schema versions and reserved domain/purpose pairs for the KNIRV
// Controller WASM control plane (see
// KNIRV_CORP/packages/controller/stateless_pwa_controller.md sections 3.4,
// 8.1, and 9). These payloads are opaque canonical bytes carried inside a
// signed MessageEnvelope (domain knirv.controller); they are never signed on
// their own. Byte layout must stay identical to
// KNIRV_NETWORK/packages/KNIRVSDK/go/signing/wasm.go — see
// KNIRV_NETWORK/packages/KNIRVSDK/testvectors/wasm_payloads.json for the
// cross-language golden vectors that lock this down.
export const WASM_PUBLICATION_SCHEMA_VERSION = 'knirv.wasm_publication.v1';
export const WASM_MANIFEST_SCHEMA_VERSION = 'knirv.wasm_manifest.v1';
export const CONTROLLER_DOMAIN = 'knirv.controller';
export const PURPOSE_WASM_PUBLICATION = 'wasm-publication';
export const PURPOSE_WASM_ASSIGNMENT = 'wasm-assignment';
export const PURPOSE_WASM_DOWNLOAD_GRANT = 'wasm-download-grant';
export const PURPOSE_RELAY_REQUEST = 'relay-request';
export const PURPOSE_RELAY_RESPONSE = 'relay-response';
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
function normalizeWasmPublicationPayload(p) {
    const schemaVersion = p.schemaVersion || WASM_PUBLICATION_SCHEMA_VERSION;
    if (schemaVersion !== WASM_PUBLICATION_SCHEMA_VERSION) {
        throw new Error(`Unsupported wasm publication schema: ${schemaVersion}`);
    }
    if (!p.networkId.trim() || !p.networkFingerprint.trim()) {
        throw new Error('networkId and networkFingerprint are required');
    }
    if (!p.artifactDigest.trim() || BigInt(p.byteSize) === 0n) {
        throw new Error('artifactDigest and byteSize are required');
    }
    if (!p.moduleKind.trim() || !p.buildId.trim()) {
        throw new Error('moduleKind and buildId are required');
    }
    if (!p.publisherAddress.trim()) {
        throw new Error('publisherAddress is required');
    }
    return { ...p, schemaVersion };
}
export function marshalWasmPublicationPayload(payload) {
    const p = normalizeWasmPublicationPayload(payload);
    return concat(stringField(1, p.schemaVersion), stringField(2, p.networkId), stringField(3, p.networkFingerprint), stringField(4, p.artifactDigest), uintField(5, p.byteSize), stringField(6, p.moduleKind), uintField(7, p.abiVersion), uintField(8, p.moduleSchemaVersion), stringField(9, p.buildId), stringField(10, p.toolchainDigest), stringField(11, p.selfTestDigest), stringField(12, p.publisherAddress), stringField(13, p.dveTemplateId));
}
function marshalWasmManifestModule(m) {
    return concat(stringField(1, m.moduleKind), stringField(2, m.artifactDigest), uintField(3, m.byteSize), uintField(4, m.abiVersion), uintField(5, m.moduleSchemaVersion), stringField(6, m.capabilitiesJson), stringField(7, m.configurationDigest), stringField(8, m.downloadPath), stringField(9, m.publisherAddress), stringField(10, m.publicationStatementDigest));
}
function normalizeWasmManifestPayload(p) {
    const schemaVersion = p.schemaVersion || WASM_MANIFEST_SCHEMA_VERSION;
    if (schemaVersion !== WASM_MANIFEST_SCHEMA_VERSION) {
        throw new Error(`Unsupported wasm manifest schema: ${schemaVersion}`);
    }
    if (!p.manifestId.trim() || !p.networkId.trim() || !p.chainId.trim()) {
        throw new Error('manifestId, networkId, and chainId are required');
    }
    if (!p.networkFingerprint.trim() || !p.userSubject.trim() || !p.deviceId.trim()) {
        throw new Error('networkFingerprint, userSubject, and deviceId are required');
    }
    if (!p.modules.length) {
        throw new Error('at least one module entry is required');
    }
    if (!p.assignmentId.trim()) {
        throw new Error('assignmentId is required');
    }
    return { ...p, schemaVersion };
}
export function marshalWasmManifestPayload(payload) {
    const p = normalizeWasmManifestPayload(payload);
    return concat(stringField(1, p.schemaVersion), stringField(2, p.manifestId), stringField(3, p.networkId), stringField(4, p.chainId), stringField(5, p.networkFingerprint), stringField(6, p.userSubject), stringField(7, p.deviceId), stringField(8, p.dveId), uintField(9, p.leaseEpoch), ...p.modules.map((m) => bytesField(10, marshalWasmManifestModule(m))), stringField(11, p.relayTargetType), stringField(12, p.relayTargetId), stringField(13, p.assignmentId), uintField(14, p.assignmentVersion), stringField(15, p.supersedesAssignmentId));
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
function stringOf(field) {
    return field?.bytesValue ? decoder.decode(field.bytesValue) : '';
}
function uintOf(field) {
    return field?.uintValue ?? 0n;
}
export function parseWasmPublicationPayload(data) {
    const fields = readFields(data);
    const byNumber = new Map();
    for (const field of fields)
        if (!byNumber.has(field.number))
            byNumber.set(field.number, field);
    const payload = {
        schemaVersion: stringOf(byNumber.get(1)),
        networkId: stringOf(byNumber.get(2)),
        networkFingerprint: stringOf(byNumber.get(3)),
        artifactDigest: stringOf(byNumber.get(4)),
        byteSize: uintOf(byNumber.get(5)),
        moduleKind: stringOf(byNumber.get(6)),
        abiVersion: uintOf(byNumber.get(7)),
        moduleSchemaVersion: uintOf(byNumber.get(8)),
        buildId: stringOf(byNumber.get(9)),
        toolchainDigest: stringOf(byNumber.get(10)),
        selfTestDigest: stringOf(byNumber.get(11)),
        publisherAddress: stringOf(byNumber.get(12)),
        dveTemplateId: stringOf(byNumber.get(13)),
    };
    return normalizeWasmPublicationPayload(payload);
}
function parseWasmManifestModule(data) {
    const fields = readFields(data);
    const byNumber = new Map();
    for (const field of fields)
        if (!byNumber.has(field.number))
            byNumber.set(field.number, field);
    return {
        moduleKind: stringOf(byNumber.get(1)),
        artifactDigest: stringOf(byNumber.get(2)),
        byteSize: uintOf(byNumber.get(3)),
        abiVersion: uintOf(byNumber.get(4)),
        moduleSchemaVersion: uintOf(byNumber.get(5)),
        capabilitiesJson: stringOf(byNumber.get(6)),
        configurationDigest: stringOf(byNumber.get(7)),
        downloadPath: stringOf(byNumber.get(8)),
        publisherAddress: stringOf(byNumber.get(9)),
        publicationStatementDigest: stringOf(byNumber.get(10)),
    };
}
export function parseWasmManifestPayload(data) {
    const fields = readFields(data);
    const byNumber = new Map();
    const modules = [];
    for (const field of fields) {
        if (field.number === 10 && field.wireType === 2 && field.bytesValue) {
            modules.push(parseWasmManifestModule(field.bytesValue));
            continue;
        }
        if (!byNumber.has(field.number))
            byNumber.set(field.number, field);
    }
    const payload = {
        schemaVersion: stringOf(byNumber.get(1)),
        manifestId: stringOf(byNumber.get(2)),
        networkId: stringOf(byNumber.get(3)),
        chainId: stringOf(byNumber.get(4)),
        networkFingerprint: stringOf(byNumber.get(5)),
        userSubject: stringOf(byNumber.get(6)),
        deviceId: stringOf(byNumber.get(7)),
        dveId: stringOf(byNumber.get(8)),
        leaseEpoch: uintOf(byNumber.get(9)),
        modules,
        relayTargetType: stringOf(byNumber.get(11)),
        relayTargetId: stringOf(byNumber.get(12)),
        assignmentId: stringOf(byNumber.get(13)),
        assignmentVersion: uintOf(byNumber.get(14)),
        supersedesAssignmentId: stringOf(byNumber.get(15)),
    };
    return normalizeWasmManifestPayload(payload);
}
