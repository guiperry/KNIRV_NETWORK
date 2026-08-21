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
const encoder = new TextEncoder();
const decoder = new TextDecoder();
const concat = (...values) => { const output = new Uint8Array(values.reduce((total, value) => total + value.length, 0)); let offset = 0; for (const value of values) {
    output.set(value, offset);
    offset += value.length;
} return output; };
function varint(value) { let current = BigInt(value); if (current < 0n)
    current = BigInt.asUintN(64, current); const output = []; do {
    let next = Number(current & 0x7fn);
    current >>= 7n;
    if (current !== 0n)
        next |= 0x80;
    output.push(next);
} while (current !== 0n); return Uint8Array.from(output); }
const tag = (field, wireType) => varint(BigInt((field << 3) | wireType));
const bytesField = (field, value) => !value?.length ? new Uint8Array() : concat(tag(field, 2), varint(value.length), value);
const stringField = (field, value) => value ? bytesField(field, encoder.encode(value)) : new Uint8Array();
const uintField = (field, value) => value === undefined || BigInt(value) === 0n ? new Uint8Array() : concat(tag(field, 0), varint(value));
const any = (typeUrl, value) => concat(stringField(1, typeUrl), bytesField(2, value));
function toBase64(value) { let binary = ''; for (let start = 0; start < value.length; start += 0x8000)
    binary += String.fromCharCode(...value.subarray(start, start + 0x8000)); return btoa(binary); }
const fromBase64 = (value) => Uint8Array.from(atob(value), (character) => character.charCodeAt(0));
const toHex = (value) => Array.from(value, (byte) => byte.toString(16).padStart(2, '0')).join('').toUpperCase();
export function marshalAction(action) {
    const schema = action.schemaVersion || ACTION_SCHEMA_VERSION;
    if (schema !== ACTION_SCHEMA_VERSION)
        throw new Error(`Unsupported action schema: ${schema}`);
    if (!action.action.trim() || !action.sender.trim())
        throw new Error('action and sender are required');
    if (BigInt(action.timestampUnix) <= 0n)
        throw new Error('action timestamp is required');
    return concat(stringField(1, schema), stringField(2, action.action), stringField(3, action.sender), stringField(4, action.recipient), uintField(5, action.amount), bytesField(6, action.payload), uintField(7, action.timestampUnix));
}
function marshalFee(fee) { if (!fee)
    return new Uint8Array(); const amount = fee.denom || fee.amount ? concat(stringField(1, fee.denom), stringField(2, fee.amount)) : new Uint8Array(); return concat(bytesField(1, amount), uintField(2, fee.gasLimit), stringField(3, fee.payer), stringField(4, fee.granter)); }
export function buildDirectSignDoc(request, compressedPublicKey) {
    if (!request.chainId.trim())
        throw new Error('chainId is required');
    if (compressedPublicKey.length !== 33)
        throw new Error('compressed secp256k1 public key must be 33 bytes');
    const bodyBytes = bytesField(1, any(ACTION_TYPE_URL, marshalAction(request.action)));
    const publicKey = any(SECP256K1_TYPE_URL, bytesField(1, compressedPublicKey));
    const signerInfo = concat(bytesField(1, publicKey), bytesField(2, bytesField(1, uintField(1, SIGN_MODE_DIRECT))), uintField(3, request.sequence));
    const authInfoBytes = concat(bytesField(1, signerInfo), bytesField(2, marshalFee(request.fee)));
    return { bodyBytes, authInfoBytes, signDoc: concat(bytesField(1, bodyBytes), bytesField(2, authInfoBytes), stringField(3, request.chainId), uintField(4, request.accountNumber)) };
}
export const marshalTxRaw = (bodyBytes, authInfoBytes, signatures) => concat(bytesField(1, bodyBytes), bytesField(2, authInfoBytes), ...signatures.map((signature) => bytesField(3, signature)));
export function publicKeyToKNIRVAddress(compressedPublicKey) { if (compressedPublicKey.length !== 33)
    throw new Error('compressed secp256k1 public key must be 33 bytes'); return bech32.encode('knirv', bech32.toWords(ripemd160(sha256(compressedPublicKey)))); }
export async function signDirectTransaction(privateKey, request) {
    if (privateKey.length !== 32)
        throw new Error('secp256k1 private key must be 32 bytes');
    const { pubkey } = await Secp256k1.makeKeypair(privateKey);
    const publicKey = Secp256k1.compressPubkey(pubkey);
    const { bodyBytes, authInfoBytes, signDoc } = buildDirectSignDoc(request, publicKey);
    const signature = (await Secp256k1.createSignature(sha256(signDoc), privateKey)).toFixedLength().slice(0, 64);
    const txRaw = marshalTxRaw(bodyBytes, authInfoBytes, [signature]);
    return { body_bytes: toBase64(bodyBytes), auth_info_bytes: toBase64(authInfoBytes), signatures: [toBase64(signature)], public_key: toBase64(publicKey), address: publicKeyToKNIRVAddress(publicKey), hash: toHex(sha256(txRaw)) };
}
export function marshalMessageEnvelope(envelope) {
    const schema = envelope.schemaVersion || MESSAGE_SCHEMA_VERSION;
    if (schema !== MESSAGE_SCHEMA_VERSION)
        throw new Error(`Unsupported message schema: ${schema}`);
    if (!envelope.domain.trim() || !envelope.purpose.trim() || !envelope.chainId.trim() || !envelope.nonce.trim())
        throw new Error('domain, purpose, chainId, and nonce are required');
    if (BigInt(envelope.issuedAtUnix) <= 0n || BigInt(envelope.expiresAtUnix) <= BigInt(envelope.issuedAtUnix))
        throw new Error('message envelope validity window is invalid');
    return concat(stringField(1, schema), stringField(2, envelope.domain), stringField(3, envelope.purpose), stringField(4, envelope.chainId), stringField(5, envelope.nonce), uintField(6, envelope.issuedAtUnix), uintField(7, envelope.expiresAtUnix), bytesField(8, envelope.payload));
}
export async function signMessageEnvelope(privateKey, envelope) {
    if (privateKey.length !== 32)
        throw new Error('secp256k1 private key must be 32 bytes');
    const bytes = marshalMessageEnvelope(envelope);
    const { pubkey } = await Secp256k1.makeKeypair(privateKey);
    const publicKey = Secp256k1.compressPubkey(pubkey);
    const signature = (await Secp256k1.createSignature(sha256(bytes), privateKey)).toFixedLength().slice(0, 64);
    return { envelope: toBase64(bytes), signature: toBase64(signature), public_key: toBase64(publicKey), address: publicKeyToKNIRVAddress(publicKey) };
}
export async function verifyMessageEnvelope(signed, expected) { const envelope = marshalMessageEnvelope(expected); const received = fromBase64(signed.envelope); if (envelope.length !== received.length || envelope.some((byte, index) => byte !== received[index]))
    return false; const publicKey = fromBase64(signed.public_key); if (publicKey.length !== 33 || !signed.address || publicKeyToKNIRVAddress(publicKey) !== signed.address)
    return false; const signature = fromBase64(signed.signature); return signature.length === 64 && Secp256k1.verifySignature(Secp256k1Signature.fromFixedLength(signature), sha256(envelope), publicKey); }
function readVarint(data, start) { let value = 0n, shift = 0n, i = start; for (; i < data.length; i++) {
    const byte = data[i];
    value |= BigInt(byte & 0x7f) << shift;
    if ((byte & 0x80) === 0)
        return { value, next: i + 1 };
    shift += 7n;
    if (shift >= 64n)
        throw new Error('varint overflow');
} throw new Error('truncated varint'); }
function readFields(data) { const fields = []; let i = 0; while (i < data.length) {
    const key = readVarint(data, i);
    i = key.next;
    const number = Number(key.value >> 3n), wireType = Number(key.value & 0x7n);
    if (wireType === 0) {
        const value = readVarint(data, i);
        fields.push({ number, wireType, uintValue: value.value });
        i = value.next;
    }
    else if (wireType === 2) {
        const length = readVarint(data, i);
        const end = length.next + Number(length.value);
        if (end > data.length)
            throw new Error('truncated length-delimited field');
        fields.push({ number, wireType, bytesValue: data.subarray(length.next, end) });
        i = end;
    }
    else
        throw new Error(`unsupported wire type ${wireType}`);
} return fields; }
export function parseMessageEnvelope(data) { const fields = new Map(); for (const field of readFields(data))
    if (!fields.has(field.number))
        fields.set(field.number, field); const stringOf = (n) => fields.get(n)?.bytesValue ? decoder.decode(fields.get(n)?.bytesValue) : ''; const parsed = { schemaVersion: stringOf(1), domain: stringOf(2), purpose: stringOf(3), chainId: stringOf(4), nonce: stringOf(5), issuedAtUnix: fields.get(6)?.uintValue ?? 0n, expiresAtUnix: fields.get(7)?.uintValue ?? 0n, payload: fields.get(8)?.bytesValue ?? new Uint8Array() }; if (parsed.schemaVersion !== MESSAGE_SCHEMA_VERSION || !parsed.domain || !parsed.purpose || !parsed.chainId || !parsed.nonce)
    throw new Error('signed message envelope is incomplete'); return parsed; }
export async function verifyMessage(signed, expectedDomain, expectedPurpose, expectedChainId, expectedNonce, now) { const envelope = fromBase64(signed.envelope); const fields = parseMessageEnvelope(envelope); if (fields.domain !== expectedDomain || fields.purpose !== expectedPurpose || fields.chainId !== expectedChainId || fields.nonce !== expectedNonce)
    throw new Error('message signing domain does not match request'); const nowUnix = BigInt(Math.floor(now.getTime() / 1000)); if (nowUnix < fields.issuedAtUnix - 60n || nowUnix > fields.expiresAtUnix)
    throw new Error('signed message is outside its validity window'); const publicKey = fromBase64(signed.public_key); if (publicKey.length !== 33)
    throw new Error('compressed secp256k1 public key must be 33 bytes'); if (!signed.address || signed.address !== publicKeyToKNIRVAddress(publicKey))
    throw new Error('message address does not match public key'); const signature = fromBase64(signed.signature); if (signature.length !== 64)
    throw new Error('signature must use Cosmos 64-byte r|s encoding'); if (!await Secp256k1.verifySignature(Secp256k1Signature.fromFixedLength(signature), sha256(envelope), publicKey))
    throw new Error('signature verification failed'); }
export async function verifyMessagePayload(signed, expectedDomain, expectedPurpose, expectedChainId, expectedPayload, now) { const fields = parseMessageEnvelope(fromBase64(signed.envelope)); await verifyMessage(signed, expectedDomain, expectedPurpose, expectedChainId, fields.nonce, now); if (fields.payload.length !== expectedPayload.length || fields.payload.some((byte, index) => byte !== expectedPayload[index]))
    throw new Error('signed message payload does not match request'); }
