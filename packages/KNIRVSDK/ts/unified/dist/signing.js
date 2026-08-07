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
function stringField(field, value) {
    return value ? bytesField(field, encoder.encode(value)) : new Uint8Array();
}
function uintField(field, value) {
    if (value === undefined || BigInt(value) === 0n)
        return new Uint8Array();
    return concat(tag(field, 0), varint(value));
}
function any(typeUrl, value) {
    return concat(stringField(1, typeUrl), bytesField(2, value));
}
function toBase64(value) {
    let binary = '';
    for (let start = 0; start < value.length; start += 0x8000) {
        binary += String.fromCharCode(...value.subarray(start, start + 0x8000));
    }
    return btoa(binary);
}
function fromBase64(value) {
    const binary = atob(value);
    return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}
function toHex(value) {
    return Array.from(value, (byte) => byte.toString(16).padStart(2, '0')).join('').toUpperCase();
}
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
function marshalFee(fee) {
    if (!fee)
        return new Uint8Array();
    let amount = new Uint8Array();
    if (fee.denom || fee.amount) {
        amount = concat(stringField(1, fee.denom), stringField(2, fee.amount));
    }
    return concat(bytesField(1, amount), uintField(2, fee.gasLimit), stringField(3, fee.payer), stringField(4, fee.granter));
}
export function buildDirectSignDoc(request, compressedPublicKey) {
    if (!request.chainId.trim())
        throw new Error('chainId is required');
    if (compressedPublicKey.length !== 33)
        throw new Error('compressed secp256k1 public key must be 33 bytes');
    const bodyBytes = bytesField(1, any(ACTION_TYPE_URL, marshalAction(request.action)));
    const publicKey = any(SECP256K1_TYPE_URL, bytesField(1, compressedPublicKey));
    const modeInfo = bytesField(1, uintField(1, SIGN_MODE_DIRECT));
    const signerInfo = concat(bytesField(1, publicKey), bytesField(2, modeInfo), uintField(3, request.sequence));
    const authInfoBytes = concat(bytesField(1, signerInfo), bytesField(2, marshalFee(request.fee)));
    const signDoc = concat(bytesField(1, bodyBytes), bytesField(2, authInfoBytes), stringField(3, request.chainId), uintField(4, request.accountNumber));
    return { bodyBytes, authInfoBytes, signDoc };
}
export function marshalTxRaw(bodyBytes, authInfoBytes, signatures) {
    return concat(bytesField(1, bodyBytes), bytesField(2, authInfoBytes), ...signatures.map((s) => bytesField(3, s)));
}
export function publicKeyToKNIRVAddress(compressedPublicKey) {
    if (compressedPublicKey.length !== 33)
        throw new Error('compressed secp256k1 public key must be 33 bytes');
    return bech32.encode('knirv', bech32.toWords(ripemd160(sha256(compressedPublicKey))));
}
export async function signDirectTransaction(privateKey, request) {
    if (privateKey.length !== 32)
        throw new Error('secp256k1 private key must be 32 bytes');
    const { pubkey } = await Secp256k1.makeKeypair(privateKey);
    const publicKey = Secp256k1.compressPubkey(pubkey);
    const { bodyBytes, authInfoBytes, signDoc } = buildDirectSignDoc(request, publicKey);
    const signature = (await Secp256k1.createSignature(sha256(signDoc), privateKey)).toFixedLength();
    const txRaw = marshalTxRaw(bodyBytes, authInfoBytes, [signature]);
    return {
        body_bytes: toBase64(bodyBytes),
        auth_info_bytes: toBase64(authInfoBytes),
        signatures: [toBase64(signature)],
        public_key: toBase64(publicKey),
        address: publicKeyToKNIRVAddress(publicKey),
        hash: toHex(sha256(txRaw)),
    };
}
export function marshalMessageEnvelope(envelope) {
    const schema = envelope.schemaVersion || MESSAGE_SCHEMA_VERSION;
    if (schema !== MESSAGE_SCHEMA_VERSION)
        throw new Error(`Unsupported message schema: ${schema}`);
    if (!envelope.domain.trim() || !envelope.purpose.trim() || !envelope.chainId.trim() || !envelope.nonce.trim()) {
        throw new Error('domain, purpose, chainId, and nonce are required');
    }
    if (BigInt(envelope.issuedAtUnix) <= 0n || BigInt(envelope.expiresAtUnix) <= BigInt(envelope.issuedAtUnix)) {
        throw new Error('message envelope validity window is invalid');
    }
    return concat(stringField(1, schema), stringField(2, envelope.domain), stringField(3, envelope.purpose), stringField(4, envelope.chainId), stringField(5, envelope.nonce), uintField(6, envelope.issuedAtUnix), uintField(7, envelope.expiresAtUnix), bytesField(8, envelope.payload));
}
export async function signMessageEnvelope(privateKey, envelope) {
    const bytes = marshalMessageEnvelope(envelope);
    const { pubkey } = await Secp256k1.makeKeypair(privateKey);
    const publicKey = Secp256k1.compressPubkey(pubkey);
    const signature = (await Secp256k1.createSignature(sha256(bytes), privateKey)).toFixedLength();
    return {
        envelope: toBase64(bytes), signature: toBase64(signature), public_key: toBase64(publicKey),
        address: publicKeyToKNIRVAddress(publicKey),
    };
}
export async function verifyMessageEnvelope(signed, expected) {
    const envelope = marshalMessageEnvelope(expected);
    const receivedEnvelope = fromBase64(signed.envelope);
    if (envelope.length !== receivedEnvelope.length || envelope.some((byte, index) => byte !== receivedEnvelope[index]))
        return false;
    const publicKey = fromBase64(signed.public_key);
    if (publicKey.length !== 33 || publicKeyToKNIRVAddress(publicKey) !== signed.address)
        return false;
    const rawSignature = fromBase64(signed.signature);
    if (rawSignature.length !== 64)
        return false;
    return Secp256k1.verifySignature(Secp256k1Signature.fromFixedLength(rawSignature), sha256(envelope), publicKey);
}
