/**
 * KNIRV direct-transaction and signed-message primitives.
 *
 * This entry point is intentionally independent of the WASM runtime so it is
 * usable by Node, browsers, and edge runtimes before `SdkClient.init()`.
 */
import { Secp256k1, Secp256k1Signature, sha256 } from '@cosmjs/crypto';
import { ripemd160 } from '@noble/hashes/ripemd160';
import { bech32 } from 'bech32';

export const KNIRV_HD_PATH = "m/44'/118'/0'/0/i";
export const ACTION_SCHEMA_VERSION = 'knirv.action.v1';
export const MESSAGE_SCHEMA_VERSION = 'knirv.message.v1';
export const ACTION_TYPE_URL = '/knirv.signing.v1.Action';
export const SECP256K1_TYPE_URL = '/cosmos.crypto.secp256k1.PubKey';
export const SIGN_MODE_DIRECT = 1;
export const RELAY_ENVELOPE_SCHEMA_VERSION = 'knirv.controller.relay-envelope.v1';
export const WASM_PUBLICATION_SCHEMA_VERSION = 'knirv.wasm_publication.v1';
export const WASM_MANIFEST_SCHEMA_VERSION = 'knirv.wasm_manifest.v1';
export const CONTROLLER_DOMAIN = 'knirv.controller';
export const PURPOSE_WASM_PUBLICATION = 'wasm-publication';
export const PURPOSE_WASM_ASSIGNMENT = 'wasm-assignment';
export const PURPOSE_WASM_DOWNLOAD_GRANT = 'wasm-download-grant';
export const PURPOSE_RELAY_REQUEST = 'relay-request';
export const PURPOSE_RELAY_RESPONSE = 'relay-response';
export const RELAY_TARGET_DVE_EXPERT_ADVISOR = 'dve_expert_advisor';
export const RELAY_TARGET_CLI_SUPERVISOR = 'cli_supervisor';

export type Uint64 = number | string | bigint;
export interface KNIRVAction { schemaVersion?: string; action: string; sender: string; recipient?: string; amount?: Uint64; payload?: Uint8Array; timestampUnix: Uint64; }
export interface KNIRVFee { denom?: string; amount?: string; gasLimit?: Uint64; payer?: string; granter?: string; }
export interface DirectSignRequest { action: KNIRVAction; chainId: string; accountNumber: Uint64; sequence: Uint64; fee?: KNIRVFee; }
export interface SignedDirectTransaction { body_bytes: string; auth_info_bytes: string; signatures: string[]; public_key: string; address: string; hash: string; }
export interface MessageEnvelope { schemaVersion?: string; domain: string; purpose: string; chainId: string; nonce: string; issuedAtUnix: Uint64; expiresAtUnix: Uint64; payload: Uint8Array; }
export interface SignedMessageEnvelope { envelope: string; signature: string; public_key: string; address: string; }
export interface ParsedMessageEnvelope { schemaVersion: string; domain: string; purpose: string; chainId: string; nonce: string; issuedAtUnix: bigint; expiresAtUnix: bigint; payload: Uint8Array; }
export interface RelayEnvelope { schemaVersion?: string; requestId: string; userSubject: string; deviceId: string; dveId?: string; targetType: typeof RELAY_TARGET_DVE_EXPERT_ADVISOR | typeof RELAY_TARGET_CLI_SUPERVISOR; targetId: string; capability: string; sequence: Uint64; leaseEpoch?: Uint64; issuedAtUnix: Uint64; expiresAtUnix: Uint64; payloadDigest: string; }
export interface ParsedRelayEnvelope { schemaVersion: string; requestId: string; userSubject: string; deviceId: string; dveId: string; targetType: RelayEnvelope['targetType']; targetId: string; capability: string; sequence: bigint; leaseEpoch: bigint; issuedAtUnix: bigint; expiresAtUnix: bigint; payloadDigest: string; }
export interface WasmPublicationPayload { schemaVersion?: string; networkId: string; networkFingerprint: string; artifactDigest: string; byteSize: Uint64; moduleKind: string; abiVersion?: Uint64; moduleSchemaVersion?: Uint64; buildId: string; toolchainDigest?: string; selfTestDigest?: string; publisherAddress: string; dveTemplateId?: string; }
export interface WasmManifestModule { moduleKind: string; artifactDigest: string; byteSize: Uint64; abiVersion: Uint64; moduleSchemaVersion: Uint64; capabilitiesJson: string; configurationDigest: string; downloadPath: string; publisherAddress: string; publicationStatementDigest: string; }
export interface WasmManifestPayload { schemaVersion?: string; manifestId: string; networkId: string; chainId: string; networkFingerprint: string; userSubject: string; deviceId: string; dveId?: string; leaseEpoch?: Uint64; modules: WasmManifestModule[]; relayTargetType?: string; relayTargetId?: string; assignmentId: string; assignmentVersion?: Uint64; supersedesAssignmentId?: string; }

const encoder = new TextEncoder();
const decoder = new TextDecoder();
const concat = (...values: Uint8Array[]) => { const output = new Uint8Array(values.reduce((total, value) => total + value.length, 0)); let offset = 0; for (const value of values) { output.set(value, offset); offset += value.length; } return output; };
function varint(value: Uint64): Uint8Array { let current = BigInt(value); if (current < 0n) current = BigInt.asUintN(64, current); const output: number[] = []; do { let next = Number(current & 0x7fn); current >>= 7n; if (current !== 0n) next |= 0x80; output.push(next); } while (current !== 0n); return Uint8Array.from(output); }
const tag = (field: number, wireType: 0 | 2) => varint(BigInt((field << 3) | wireType));
const bytesField = (field: number, value?: Uint8Array) => !value?.length ? new Uint8Array() : concat(tag(field, 2), varint(value.length), value);
const stringField = (field: number, value?: string) => value ? bytesField(field, encoder.encode(value)) : new Uint8Array();
const uintField = (field: number, value?: Uint64) => value === undefined || BigInt(value) === 0n ? new Uint8Array() : concat(tag(field, 0), varint(value));
const any = (typeUrl: string, value: Uint8Array) => concat(stringField(1, typeUrl), bytesField(2, value));
function toBase64(value: Uint8Array): string { let binary = ''; for (let start = 0; start < value.length; start += 0x8000) binary += String.fromCharCode(...value.subarray(start, start + 0x8000)); return btoa(binary); }
const fromBase64 = (value: string) => Uint8Array.from(atob(value), (character) => character.charCodeAt(0));
const toHex = (value: Uint8Array) => Array.from(value, (byte) => byte.toString(16).padStart(2, '0')).join('').toUpperCase();

export function marshalAction(action: KNIRVAction): Uint8Array {
  const schema = action.schemaVersion || ACTION_SCHEMA_VERSION;
  if (schema !== ACTION_SCHEMA_VERSION) throw new Error(`Unsupported action schema: ${schema}`);
  if (!action.action.trim() || !action.sender.trim()) throw new Error('action and sender are required');
  if (BigInt(action.timestampUnix) <= 0n) throw new Error('action timestamp is required');
  return concat(stringField(1, schema), stringField(2, action.action), stringField(3, action.sender), stringField(4, action.recipient), uintField(5, action.amount), bytesField(6, action.payload), uintField(7, action.timestampUnix));
}
function marshalFee(fee?: KNIRVFee): Uint8Array { if (!fee) return new Uint8Array(); const amount = fee.denom || fee.amount ? concat(stringField(1, fee.denom), stringField(2, fee.amount)) : new Uint8Array(); return concat(bytesField(1, amount), uintField(2, fee.gasLimit), stringField(3, fee.payer), stringField(4, fee.granter)); }
export function buildDirectSignDoc(request: DirectSignRequest, compressedPublicKey: Uint8Array) {
  if (!request.chainId.trim()) throw new Error('chainId is required');
  if (compressedPublicKey.length !== 33) throw new Error('compressed secp256k1 public key must be 33 bytes');
  const bodyBytes = bytesField(1, any(ACTION_TYPE_URL, marshalAction(request.action)));
  const publicKey = any(SECP256K1_TYPE_URL, bytesField(1, compressedPublicKey));
  const signerInfo = concat(bytesField(1, publicKey), bytesField(2, bytesField(1, uintField(1, SIGN_MODE_DIRECT))), uintField(3, request.sequence));
  const authInfoBytes = concat(bytesField(1, signerInfo), bytesField(2, marshalFee(request.fee)));
  return { bodyBytes, authInfoBytes, signDoc: concat(bytesField(1, bodyBytes), bytesField(2, authInfoBytes), stringField(3, request.chainId), uintField(4, request.accountNumber)) };
}
export const marshalTxRaw = (bodyBytes: Uint8Array, authInfoBytes: Uint8Array, signatures: Uint8Array[]) => concat(bytesField(1, bodyBytes), bytesField(2, authInfoBytes), ...signatures.map((signature) => bytesField(3, signature)));
export function publicKeyToKNIRVAddress(compressedPublicKey: Uint8Array): string { if (compressedPublicKey.length !== 33) throw new Error('compressed secp256k1 public key must be 33 bytes'); return bech32.encode('knirv', bech32.toWords(ripemd160(sha256(compressedPublicKey)))); }
export async function signDirectTransaction(privateKey: Uint8Array, request: DirectSignRequest): Promise<SignedDirectTransaction> {
  if (privateKey.length !== 32) throw new Error('secp256k1 private key must be 32 bytes');
  const { pubkey } = await Secp256k1.makeKeypair(privateKey); const publicKey = Secp256k1.compressPubkey(pubkey); const { bodyBytes, authInfoBytes, signDoc } = buildDirectSignDoc(request, publicKey);
  const signature = (await Secp256k1.createSignature(sha256(signDoc), privateKey)).toFixedLength().slice(0, 64); const txRaw = marshalTxRaw(bodyBytes, authInfoBytes, [signature]);
  return { body_bytes: toBase64(bodyBytes), auth_info_bytes: toBase64(authInfoBytes), signatures: [toBase64(signature)], public_key: toBase64(publicKey), address: publicKeyToKNIRVAddress(publicKey), hash: toHex(sha256(txRaw)) };
}
export function marshalMessageEnvelope(envelope: MessageEnvelope): Uint8Array {
  const schema = envelope.schemaVersion || MESSAGE_SCHEMA_VERSION;
  if (schema !== MESSAGE_SCHEMA_VERSION) throw new Error(`Unsupported message schema: ${schema}`);
  if (!envelope.domain.trim() || !envelope.purpose.trim() || !envelope.chainId.trim() || !envelope.nonce.trim()) throw new Error('domain, purpose, chainId, and nonce are required');
  if (BigInt(envelope.issuedAtUnix) <= 0n || BigInt(envelope.expiresAtUnix) <= BigInt(envelope.issuedAtUnix)) throw new Error('message envelope validity window is invalid');
  return concat(stringField(1, schema), stringField(2, envelope.domain), stringField(3, envelope.purpose), stringField(4, envelope.chainId), stringField(5, envelope.nonce), uintField(6, envelope.issuedAtUnix), uintField(7, envelope.expiresAtUnix), bytesField(8, envelope.payload));
}
const nonEmpty = (value: string) => value.trim().length > 0;
const messageString = (field: EnvelopeWireField | undefined) => field?.bytesValue ? decoder.decode(field.bytesValue) : '';
const messageUint = (field: EnvelopeWireField | undefined) => field?.uintValue ?? 0n;
const fieldMap = (data: Uint8Array) => { const fields = new Map<number, EnvelopeWireField[]>(); for (const field of readFields(data)) { const values = fields.get(field.number) ?? []; values.push(field); fields.set(field.number, values); } return fields; };
const first = (fields: Map<number, EnvelopeWireField[]>, number: number) => fields.get(number)?.[0];
const relayTarget = (value: string): RelayEnvelope['targetType'] => {
  if (value !== RELAY_TARGET_DVE_EXPERT_ADVISOR && value !== RELAY_TARGET_CLI_SUPERVISOR) throw new Error('target_type must be dve_expert_advisor or cli_supervisor');
  return value;
};
export function marshalRelayEnvelope(envelope: RelayEnvelope): Uint8Array {
  const schema = envelope.schemaVersion || RELAY_ENVELOPE_SCHEMA_VERSION;
  if (schema !== RELAY_ENVELOPE_SCHEMA_VERSION) throw new Error(`Unsupported relay envelope schema: ${schema}`);
  if (![envelope.requestId, envelope.userSubject, envelope.deviceId, envelope.targetId, envelope.capability, envelope.payloadDigest].every(nonEmpty)) throw new Error('relay envelope required fields are missing');
  relayTarget(envelope.targetType);
  if (BigInt(envelope.sequence) <= 0n || BigInt(envelope.issuedAtUnix) <= 0n || BigInt(envelope.expiresAtUnix) <= BigInt(envelope.issuedAtUnix)) throw new Error('relay sequence and validity window are invalid');
  return concat(stringField(1, schema), stringField(2, envelope.requestId), stringField(3, envelope.userSubject), stringField(4, envelope.deviceId), stringField(5, envelope.dveId), stringField(6, envelope.targetType), stringField(7, envelope.targetId), stringField(8, envelope.capability), uintField(9, envelope.sequence), uintField(10, envelope.leaseEpoch), uintField(11, envelope.issuedAtUnix), uintField(12, envelope.expiresAtUnix), stringField(13, envelope.payloadDigest));
}
export function parseRelayEnvelope(data: Uint8Array): ParsedRelayEnvelope {
  const fields = fieldMap(data); const parsed = { schemaVersion: messageString(first(fields, 1)), requestId: messageString(first(fields, 2)), userSubject: messageString(first(fields, 3)), deviceId: messageString(first(fields, 4)), dveId: messageString(first(fields, 5)), targetType: relayTarget(messageString(first(fields, 6))), targetId: messageString(first(fields, 7)), capability: messageString(first(fields, 8)), sequence: messageUint(first(fields, 9)), leaseEpoch: messageUint(first(fields, 10)), issuedAtUnix: BigInt.asIntN(64, messageUint(first(fields, 11))), expiresAtUnix: BigInt.asIntN(64, messageUint(first(fields, 12))), payloadDigest: messageString(first(fields, 13)) };
  if (parsed.schemaVersion !== RELAY_ENVELOPE_SCHEMA_VERSION || ![parsed.requestId, parsed.userSubject, parsed.deviceId, parsed.targetId, parsed.capability, parsed.payloadDigest].every(nonEmpty) || parsed.sequence <= 0n || parsed.issuedAtUnix <= 0n || parsed.expiresAtUnix <= parsed.issuedAtUnix) throw new Error('relay envelope is incomplete or invalid');
  return parsed;
}
function marshalManifestModule(module: WasmManifestModule): Uint8Array { return concat(stringField(1, module.moduleKind), stringField(2, module.artifactDigest), uintField(3, module.byteSize), uintField(4, module.abiVersion), uintField(5, module.moduleSchemaVersion), stringField(6, module.capabilitiesJson), stringField(7, module.configurationDigest), stringField(8, module.downloadPath), stringField(9, module.publisherAddress), stringField(10, module.publicationStatementDigest)); }
function parseManifestModule(data: Uint8Array): WasmManifestModule { const fields = fieldMap(data); return { moduleKind: messageString(first(fields, 1)), artifactDigest: messageString(first(fields, 2)), byteSize: messageUint(first(fields, 3)), abiVersion: messageUint(first(fields, 4)), moduleSchemaVersion: messageUint(first(fields, 5)), capabilitiesJson: messageString(first(fields, 6)), configurationDigest: messageString(first(fields, 7)), downloadPath: messageString(first(fields, 8)), publisherAddress: messageString(first(fields, 9)), publicationStatementDigest: messageString(first(fields, 10)) }; }
export function marshalWasmPublicationPayload(payload: WasmPublicationPayload): Uint8Array { const schema = payload.schemaVersion || WASM_PUBLICATION_SCHEMA_VERSION; if (schema !== WASM_PUBLICATION_SCHEMA_VERSION) throw new Error('unsupported wasm publication schema'); if (![payload.networkId, payload.networkFingerprint, payload.artifactDigest, payload.moduleKind, payload.buildId, payload.publisherAddress].every(nonEmpty) || BigInt(payload.byteSize) <= 0n) throw new Error('wasm publication required fields are missing'); return concat(stringField(1, schema), stringField(2, payload.networkId), stringField(3, payload.networkFingerprint), stringField(4, payload.artifactDigest), uintField(5, payload.byteSize), stringField(6, payload.moduleKind), uintField(7, payload.abiVersion), uintField(8, payload.moduleSchemaVersion), stringField(9, payload.buildId), stringField(10, payload.toolchainDigest), stringField(11, payload.selfTestDigest), stringField(12, payload.publisherAddress), stringField(13, payload.dveTemplateId)); }
export function marshalWasmManifestPayload(payload: WasmManifestPayload): Uint8Array { const schema = payload.schemaVersion || WASM_MANIFEST_SCHEMA_VERSION; if (schema !== WASM_MANIFEST_SCHEMA_VERSION) throw new Error('unsupported wasm manifest schema'); if (![payload.manifestId, payload.networkId, payload.chainId, payload.networkFingerprint, payload.userSubject, payload.deviceId, payload.assignmentId].every(nonEmpty) || payload.modules.length === 0) throw new Error('wasm manifest required fields are missing'); return concat(stringField(1, schema), stringField(2, payload.manifestId), stringField(3, payload.networkId), stringField(4, payload.chainId), stringField(5, payload.networkFingerprint), stringField(6, payload.userSubject), stringField(7, payload.deviceId), stringField(8, payload.dveId), uintField(9, payload.leaseEpoch), ...payload.modules.map((module) => bytesField(10, marshalManifestModule(module))), stringField(11, payload.relayTargetType), stringField(12, payload.relayTargetId), stringField(13, payload.assignmentId), uintField(14, payload.assignmentVersion), stringField(15, payload.supersedesAssignmentId)); }
export function parseWasmManifestPayload(data: Uint8Array): WasmManifestPayload { const fields = fieldMap(data); const payload: WasmManifestPayload = { schemaVersion: messageString(first(fields, 1)), manifestId: messageString(first(fields, 2)), networkId: messageString(first(fields, 3)), chainId: messageString(first(fields, 4)), networkFingerprint: messageString(first(fields, 5)), userSubject: messageString(first(fields, 6)), deviceId: messageString(first(fields, 7)), dveId: messageString(first(fields, 8)), leaseEpoch: messageUint(first(fields, 9)), modules: (fields.get(10) ?? []).map((field) => parseManifestModule(field.bytesValue ?? new Uint8Array())), relayTargetType: messageString(first(fields, 11)), relayTargetId: messageString(first(fields, 12)), assignmentId: messageString(first(fields, 13)), assignmentVersion: messageUint(first(fields, 14)), supersedesAssignmentId: messageString(first(fields, 15)) }; if (payload.schemaVersion !== WASM_MANIFEST_SCHEMA_VERSION || ![payload.manifestId, payload.networkId, payload.chainId, payload.networkFingerprint, payload.userSubject, payload.deviceId, payload.assignmentId].every(nonEmpty) || payload.modules.length === 0) throw new Error('wasm manifest is incomplete or invalid'); return payload; }
export async function signMessageEnvelope(privateKey: Uint8Array, envelope: MessageEnvelope): Promise<SignedMessageEnvelope> {
  if (privateKey.length !== 32) throw new Error('secp256k1 private key must be 32 bytes'); const bytes = marshalMessageEnvelope(envelope); const { pubkey } = await Secp256k1.makeKeypair(privateKey); const publicKey = Secp256k1.compressPubkey(pubkey); const signature = (await Secp256k1.createSignature(sha256(bytes), privateKey)).toFixedLength().slice(0, 64);
  return { envelope: toBase64(bytes), signature: toBase64(signature), public_key: toBase64(publicKey), address: publicKeyToKNIRVAddress(publicKey) };
}
export async function verifyMessageEnvelope(signed: SignedMessageEnvelope, expected: MessageEnvelope): Promise<boolean> { const envelope = marshalMessageEnvelope(expected); const received = fromBase64(signed.envelope); if (envelope.length !== received.length || envelope.some((byte, index) => byte !== received[index])) return false; const publicKey = fromBase64(signed.public_key); if (publicKey.length !== 33 || !signed.address || publicKeyToKNIRVAddress(publicKey) !== signed.address) return false; const signature = fromBase64(signed.signature); return signature.length === 64 && Secp256k1.verifySignature(Secp256k1Signature.fromFixedLength(signature), sha256(envelope), publicKey); }

interface EnvelopeWireField { number: number; wireType: 0 | 2; bytesValue?: Uint8Array; uintValue?: bigint; }
function readVarint(data: Uint8Array, start: number): { value: bigint; next: number } { let value = 0n, shift = 0n, i = start; for (; i < data.length; i++) { const byte = data[i]; value |= BigInt(byte & 0x7f) << shift; if ((byte & 0x80) === 0) return { value, next: i + 1 }; shift += 7n; if (shift >= 64n) throw new Error('varint overflow'); } throw new Error('truncated varint'); }
function readFields(data: Uint8Array): EnvelopeWireField[] { const fields: EnvelopeWireField[] = []; let i = 0; while (i < data.length) { const key = readVarint(data, i); i = key.next; const number = Number(key.value >> 3n), wireType = Number(key.value & 0x7n) as 0 | 2; if (wireType === 0) { const value = readVarint(data, i); fields.push({ number, wireType, uintValue: value.value }); i = value.next; } else if (wireType === 2) { const length = readVarint(data, i); const end = length.next + Number(length.value); if (end > data.length) throw new Error('truncated length-delimited field'); fields.push({ number, wireType, bytesValue: data.subarray(length.next, end) }); i = end; } else throw new Error(`unsupported wire type ${wireType}`); } return fields; }
export function parseMessageEnvelope(data: Uint8Array): ParsedMessageEnvelope { const fields = new Map<number, EnvelopeWireField>(); for (const field of readFields(data)) if (!fields.has(field.number)) fields.set(field.number, field); const stringOf = (n: number) => fields.get(n)?.bytesValue ? decoder.decode(fields.get(n)?.bytesValue) : ''; const parsed = { schemaVersion: stringOf(1), domain: stringOf(2), purpose: stringOf(3), chainId: stringOf(4), nonce: stringOf(5), issuedAtUnix: fields.get(6)?.uintValue ?? 0n, expiresAtUnix: fields.get(7)?.uintValue ?? 0n, payload: fields.get(8)?.bytesValue ?? new Uint8Array() }; if (parsed.schemaVersion !== MESSAGE_SCHEMA_VERSION || !parsed.domain || !parsed.purpose || !parsed.chainId || !parsed.nonce) throw new Error('signed message envelope is incomplete'); return parsed; }
export async function verifyMessage(signed: SignedMessageEnvelope, expectedDomain: string, expectedPurpose: string, expectedChainId: string, expectedNonce: string, now: Date): Promise<void> { const envelope = fromBase64(signed.envelope); const fields = parseMessageEnvelope(envelope); if (fields.domain !== expectedDomain || fields.purpose !== expectedPurpose || fields.chainId !== expectedChainId || fields.nonce !== expectedNonce) throw new Error('message signing domain does not match request'); const nowUnix = BigInt(Math.floor(now.getTime() / 1000)); if (nowUnix < fields.issuedAtUnix - 60n || nowUnix > fields.expiresAtUnix) throw new Error('signed message is outside its validity window'); const publicKey = fromBase64(signed.public_key); if (publicKey.length !== 33) throw new Error('compressed secp256k1 public key must be 33 bytes'); if (!signed.address || signed.address !== publicKeyToKNIRVAddress(publicKey)) throw new Error('message address does not match public key'); const signature = fromBase64(signed.signature); if (signature.length !== 64) throw new Error('signature must use Cosmos 64-byte r|s encoding'); if (!await Secp256k1.verifySignature(Secp256k1Signature.fromFixedLength(signature), sha256(envelope), publicKey)) throw new Error('signature verification failed'); }
export async function verifyMessagePayload(signed: SignedMessageEnvelope, expectedDomain: string, expectedPurpose: string, expectedChainId: string, expectedPayload: Uint8Array, now: Date): Promise<void> { const fields = parseMessageEnvelope(fromBase64(signed.envelope)); await verifyMessage(signed, expectedDomain, expectedPurpose, expectedChainId, fields.nonce, now); if (fields.payload.length !== expectedPayload.length || fields.payload.some((byte, index) => byte !== expectedPayload[index])) throw new Error('signed message payload does not match request'); }
