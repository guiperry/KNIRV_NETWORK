import { Secp256k1, Secp256k1Signature, sha256 } from '@cosmjs/crypto';
import { ripemd160 } from '@noble/hashes/ripemd160';
import { bech32 } from 'bech32';

export * from './wasm';
export * from './relay';

export const KNIRV_HD_PATH = "m/44'/118'/0'/0/i";
export const ACTION_SCHEMA_VERSION = 'knirv.action.v1';
export const MESSAGE_SCHEMA_VERSION = 'knirv.message.v1';
export const ACTION_TYPE_URL = '/knirv.signing.v1.Action';
export const SECP256K1_TYPE_URL = '/cosmos.crypto.secp256k1.PubKey';
export const SIGN_MODE_DIRECT = 1;

export type Uint64 = number | string | bigint;

export interface KNIRVAction {
  schemaVersion?: string;
  action: string;
  sender: string;
  recipient?: string;
  amount?: Uint64;
  payload?: Uint8Array;
  timestampUnix: Uint64;
}

export interface KNIRVFee {
  denom?: string;
  amount?: string;
  gasLimit?: Uint64;
  payer?: string;
  granter?: string;
}

export interface DirectSignRequest {
  action: KNIRVAction;
  chainId: string;
  accountNumber: Uint64;
  sequence: Uint64;
  fee?: KNIRVFee;
}

export interface SignedDirectTransaction {
  body_bytes: string;
  auth_info_bytes: string;
  signatures: string[];
  public_key: string;
  address: string;
  hash: string;
}

export interface MessageEnvelope {
  schemaVersion?: string;
  domain: string;
  purpose: string;
  chainId: string;
  nonce: string;
  issuedAtUnix: Uint64;
  expiresAtUnix: Uint64;
  payload: Uint8Array;
}

export interface SignedMessageEnvelope {
  envelope: string;
  signature: string;
  public_key: string;
  address: string;
}

const encoder = new TextEncoder();

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

function varint(value: Uint64): Uint8Array {
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

function stringField(field: number, value?: string): Uint8Array {
  return value ? bytesField(field, encoder.encode(value)) : new Uint8Array();
}

function uintField(field: number, value?: Uint64): Uint8Array {
  if (value === undefined || BigInt(value) === 0n) return new Uint8Array();
  return concat(tag(field, 0), varint(value));
}

function any(typeUrl: string, value: Uint8Array): Uint8Array {
  return concat(stringField(1, typeUrl), bytesField(2, value));
}

function toBase64(value: Uint8Array): string {
  let binary = '';
  for (let start = 0; start < value.length; start += 0x8000) {
    binary += String.fromCharCode(...value.subarray(start, start + 0x8000));
  }
  return btoa(binary);
}

function fromBase64(value: string): Uint8Array {
  const binary = atob(value);
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

function toHex(value: Uint8Array): string {
  return Array.from(value, (byte) => byte.toString(16).padStart(2, '0')).join('').toUpperCase();
}

export function marshalAction(action: KNIRVAction): Uint8Array {
  const schema = action.schemaVersion || ACTION_SCHEMA_VERSION;
  if (schema !== ACTION_SCHEMA_VERSION) throw new Error(`Unsupported action schema: ${schema}`);
  if (!action.action.trim() || !action.sender.trim()) throw new Error('action and sender are required');
  if (BigInt(action.timestampUnix) <= 0n) throw new Error('action timestamp is required');
  return concat(
    stringField(1, schema),
    stringField(2, action.action),
    stringField(3, action.sender),
    stringField(4, action.recipient),
    uintField(5, action.amount),
    bytesField(6, action.payload),
    uintField(7, action.timestampUnix),
  );
}

function marshalFee(fee?: KNIRVFee): Uint8Array {
  if (!fee) return new Uint8Array();
  let amount = new Uint8Array();
  if (fee.denom || fee.amount) {
    amount = concat(stringField(1, fee.denom), stringField(2, fee.amount));
  }
  return concat(
    bytesField(1, amount),
    uintField(2, fee.gasLimit),
    stringField(3, fee.payer),
    stringField(4, fee.granter),
  );
}

export function buildDirectSignDoc(request: DirectSignRequest, compressedPublicKey: Uint8Array) {
  if (!request.chainId.trim()) throw new Error('chainId is required');
  if (compressedPublicKey.length !== 33) throw new Error('compressed secp256k1 public key must be 33 bytes');
  const bodyBytes = bytesField(1, any(ACTION_TYPE_URL, marshalAction(request.action)));
  const publicKey = any(SECP256K1_TYPE_URL, bytesField(1, compressedPublicKey));
  const modeInfo = bytesField(1, uintField(1, SIGN_MODE_DIRECT));
  const signerInfo = concat(bytesField(1, publicKey), bytesField(2, modeInfo), uintField(3, request.sequence));
  const authInfoBytes = concat(bytesField(1, signerInfo), bytesField(2, marshalFee(request.fee)));
  const signDoc = concat(
    bytesField(1, bodyBytes),
    bytesField(2, authInfoBytes),
    stringField(3, request.chainId),
    uintField(4, request.accountNumber),
  );
  return { bodyBytes, authInfoBytes, signDoc };
}

export function marshalTxRaw(bodyBytes: Uint8Array, authInfoBytes: Uint8Array, signatures: Uint8Array[]) {
  return concat(bytesField(1, bodyBytes), bytesField(2, authInfoBytes), ...signatures.map((s) => bytesField(3, s)));
}

export function publicKeyToKNIRVAddress(compressedPublicKey: Uint8Array): string {
  if (compressedPublicKey.length !== 33) throw new Error('compressed secp256k1 public key must be 33 bytes');
  return bech32.encode('knirv', bech32.toWords(ripemd160(sha256(compressedPublicKey))));
}

export async function signDirectTransaction(privateKey: Uint8Array, request: DirectSignRequest): Promise<SignedDirectTransaction> {
  if (privateKey.length !== 32) throw new Error('secp256k1 private key must be 32 bytes');
  const { pubkey } = await Secp256k1.makeKeypair(privateKey);
  const publicKey = Secp256k1.compressPubkey(pubkey);
  const { bodyBytes, authInfoBytes, signDoc } = buildDirectSignDoc(request, publicKey);
  // ExtendedSecp256k1Signature.toFixedLength() is r|s|recovery (65 bytes);
  // the Cosmos-compatible wire format is the plain 64-byte r|s pair (see Go
  // SignTransaction's compact[1:] slice in direct.go) — drop the recovery byte.
  const signature = (await Secp256k1.createSignature(sha256(signDoc), privateKey)).toFixedLength().slice(0, 64);
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

export function marshalMessageEnvelope(envelope: MessageEnvelope): Uint8Array {
  const schema = envelope.schemaVersion || MESSAGE_SCHEMA_VERSION;
  if (schema !== MESSAGE_SCHEMA_VERSION) throw new Error(`Unsupported message schema: ${schema}`);
  if (!envelope.domain.trim() || !envelope.purpose.trim() || !envelope.chainId.trim() || !envelope.nonce.trim()) {
    throw new Error('domain, purpose, chainId, and nonce are required');
  }
  if (BigInt(envelope.issuedAtUnix) <= 0n || BigInt(envelope.expiresAtUnix) <= BigInt(envelope.issuedAtUnix)) {
    throw new Error('message envelope validity window is invalid');
  }
  return concat(
    stringField(1, schema), stringField(2, envelope.domain), stringField(3, envelope.purpose),
    stringField(4, envelope.chainId), stringField(5, envelope.nonce), uintField(6, envelope.issuedAtUnix),
    uintField(7, envelope.expiresAtUnix), bytesField(8, envelope.payload),
  );
}

export async function signMessageEnvelope(privateKey: Uint8Array, envelope: MessageEnvelope): Promise<SignedMessageEnvelope> {
  const bytes = marshalMessageEnvelope(envelope);
  const { pubkey } = await Secp256k1.makeKeypair(privateKey);
  const publicKey = Secp256k1.compressPubkey(pubkey);
  // See signDirectTransaction above: strip the trailing recovery byte to get
  // the plain 64-byte Cosmos r|s encoding.
  const signature = (await Secp256k1.createSignature(sha256(bytes), privateKey)).toFixedLength().slice(0, 64);
  return {
    envelope: toBase64(bytes), signature: toBase64(signature), public_key: toBase64(publicKey),
    address: publicKeyToKNIRVAddress(publicKey),
  };
}

export async function verifyMessageEnvelope(signed: SignedMessageEnvelope, expected: MessageEnvelope): Promise<boolean> {
  const envelope = marshalMessageEnvelope(expected);
  const receivedEnvelope = fromBase64(signed.envelope);
  if (envelope.length !== receivedEnvelope.length || envelope.some((byte, index) => byte !== receivedEnvelope[index])) return false;
  const publicKey = fromBase64(signed.public_key);
  if (publicKey.length !== 33 || publicKeyToKNIRVAddress(publicKey) !== signed.address) return false;
  const rawSignature = fromBase64(signed.signature);
  if (rawSignature.length !== 64) return false;
  return Secp256k1.verifySignature(Secp256k1Signature.fromFixedLength(rawSignature), sha256(envelope), publicKey);
}

export interface ParsedMessageEnvelope {
  schemaVersion: string;
  domain: string;
  purpose: string;
  chainId: string;
  nonce: string;
  issuedAtUnix: bigint;
  expiresAtUnix: bigint;
  payload: Uint8Array;
}

interface EnvelopeWireField {
  number: number;
  wireType: 0 | 2;
  bytesValue?: Uint8Array;
  uintValue?: bigint;
}

function readEnvelopeVarint(data: Uint8Array, start: number): { value: bigint; next: number } {
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

function readEnvelopeFields(data: Uint8Array): EnvelopeWireField[] {
  const fields: EnvelopeWireField[] = [];
  let i = 0;
  while (i < data.length) {
    const { value: key, next: afterKey } = readEnvelopeVarint(data, i);
    i = afterKey;
    const number = Number(key >> 3n);
    const wireType = Number(key & 0x7n) as 0 | 2;
    if (wireType === 0) {
      const { value, next } = readEnvelopeVarint(data, i);
      fields.push({ number, wireType, uintValue: value });
      i = next;
    } else if (wireType === 2) {
      const { value: length, next: afterLen } = readEnvelopeVarint(data, i);
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

/**
 * parseMessageEnvelope independently decodes envelope wire bytes — parity
 * with Go ParseMessageEnvelope. Used by verifyMessage so validity-window and
 * domain/purpose/chain/nonce checks are enforced against what was actually
 * signed, not against caller-supplied values (see
 * KNIRV_CORP/packages/controller/stateless_pwa_controller.md section 3.4).
 */
export function parseMessageEnvelope(data: Uint8Array): ParsedMessageEnvelope {
  const fields = readEnvelopeFields(data);
  const byNumber = new Map<number, EnvelopeWireField>();
  for (const field of fields) if (!byNumber.has(field.number)) byNumber.set(field.number, field);

  const stringOf = (n: number) => {
    const field = byNumber.get(n);
    return field?.bytesValue ? decoder.decode(field.bytesValue) : '';
  };
  const uintOf = (n: number) => byNumber.get(n)?.uintValue ?? 0n;
  const bytesOf = (n: number) => byNumber.get(n)?.bytesValue ?? new Uint8Array();

  const parsed: ParsedMessageEnvelope = {
    schemaVersion: stringOf(1),
    domain: stringOf(2),
    purpose: stringOf(3),
    chainId: stringOf(4),
    nonce: stringOf(5),
    issuedAtUnix: uintOf(6),
    expiresAtUnix: uintOf(7),
    payload: bytesOf(8),
  };
  if (parsed.schemaVersion !== MESSAGE_SCHEMA_VERSION || !parsed.domain || !parsed.purpose || !parsed.chainId || !parsed.nonce) {
    throw new Error('signed message envelope is incomplete');
  }
  return parsed;
}

const decoder = new TextDecoder();

/**
 * verifyMessage is the TypeScript parity implementation of Go
 * VerifyMessage: it independently parses the envelope (rather than trusting
 * a caller-reconstructed "expected" envelope), checks domain/purpose/chainId
 * /nonce, and enforces the validity window against wall-clock time with the
 * same 60-second issued-at clock-skew allowance as the Go implementation.
 * Throws on any verification failure.
 */
export async function verifyMessage(
  signed: SignedMessageEnvelope,
  expectedDomain: string,
  expectedPurpose: string,
  expectedChainId: string,
  expectedNonce: string,
  now: Date,
): Promise<void> {
  const envelope = fromBase64(signed.envelope);
  const fields = parseMessageEnvelope(envelope);

  if (
    fields.domain !== expectedDomain ||
    fields.purpose !== expectedPurpose ||
    fields.chainId !== expectedChainId ||
    fields.nonce !== expectedNonce
  ) {
    throw new Error('message signing domain does not match request');
  }

  const nowUnix = BigInt(Math.floor(now.getTime() / 1000));
  if (nowUnix < fields.issuedAtUnix - 60n || nowUnix > fields.expiresAtUnix) {
    throw new Error('signed message is outside its validity window');
  }

  const publicKey = fromBase64(signed.public_key);
  if (publicKey.length !== 33) throw new Error('compressed secp256k1 public key must be 33 bytes');
  const address = publicKeyToKNIRVAddress(publicKey);
  if (signed.address && signed.address !== address) {
    throw new Error('message address does not match public key');
  }

  const rawSignature = fromBase64(signed.signature);
  if (rawSignature.length !== 64) throw new Error('signature must use Cosmos 64-byte r|s encoding');

  const digest = sha256(envelope);
  const verified = await Secp256k1.verifySignature(Secp256k1Signature.fromFixedLength(rawSignature), digest, publicKey);
  if (!verified) throw new Error('signature verification failed');
}

/**
 * verifyMessagePayload is the TypeScript parity implementation of Go
 * VerifyMessagePayload: verifyMessage using the envelope's own nonce, plus a
 * byte-for-byte check of the decoded payload against expectedPayload.
 */
export async function verifyMessagePayload(
  signed: SignedMessageEnvelope,
  expectedDomain: string,
  expectedPurpose: string,
  expectedChainId: string,
  expectedPayload: Uint8Array,
  now: Date,
): Promise<void> {
  const envelope = fromBase64(signed.envelope);
  const fields = parseMessageEnvelope(envelope);
  await verifyMessage(signed, expectedDomain, expectedPurpose, expectedChainId, fields.nonce, now);
  if (
    fields.payload.length !== expectedPayload.length ||
    fields.payload.some((byte, index) => byte !== expectedPayload[index])
  ) {
    throw new Error('signed message payload does not match request');
  }
}
