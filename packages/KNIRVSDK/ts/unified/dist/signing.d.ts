export * from './wasm';
export * from './relay';
export declare const KNIRV_HD_PATH = "m/44'/118'/0'/0/i";
export declare const ACTION_SCHEMA_VERSION = "knirv.action.v1";
export declare const MESSAGE_SCHEMA_VERSION = "knirv.message.v1";
export declare const ACTION_TYPE_URL = "/knirv.signing.v1.Action";
export declare const SECP256K1_TYPE_URL = "/cosmos.crypto.secp256k1.PubKey";
export declare const SIGN_MODE_DIRECT = 1;
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
export declare function marshalAction(action: KNIRVAction): Uint8Array;
export declare function buildDirectSignDoc(request: DirectSignRequest, compressedPublicKey: Uint8Array): {
    bodyBytes: Uint8Array;
    authInfoBytes: Uint8Array;
    signDoc: Uint8Array;
};
export declare function marshalTxRaw(bodyBytes: Uint8Array, authInfoBytes: Uint8Array, signatures: Uint8Array[]): Uint8Array;
export declare function publicKeyToKNIRVAddress(compressedPublicKey: Uint8Array): string;
export declare function signDirectTransaction(privateKey: Uint8Array, request: DirectSignRequest): Promise<SignedDirectTransaction>;
export declare function marshalMessageEnvelope(envelope: MessageEnvelope): Uint8Array;
export declare function signMessageEnvelope(privateKey: Uint8Array, envelope: MessageEnvelope): Promise<SignedMessageEnvelope>;
export declare function verifyMessageEnvelope(signed: SignedMessageEnvelope, expected: MessageEnvelope): Promise<boolean>;
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
/**
 * parseMessageEnvelope independently decodes envelope wire bytes — parity
 * with Go ParseMessageEnvelope. Used by verifyMessage so validity-window and
 * domain/purpose/chain/nonce checks are enforced against what was actually
 * signed, not against caller-supplied values (see
 * KNIRV_CORP/packages/controller/stateless_pwa_controller.md section 3.4).
 */
export declare function parseMessageEnvelope(data: Uint8Array): ParsedMessageEnvelope;
/**
 * verifyMessage is the TypeScript parity implementation of Go
 * VerifyMessage: it independently parses the envelope (rather than trusting
 * a caller-reconstructed "expected" envelope), checks domain/purpose/chainId
 * /nonce, and enforces the validity window against wall-clock time with the
 * same 60-second issued-at clock-skew allowance as the Go implementation.
 * Throws on any verification failure.
 */
export declare function verifyMessage(signed: SignedMessageEnvelope, expectedDomain: string, expectedPurpose: string, expectedChainId: string, expectedNonce: string, now: Date): Promise<void>;
/**
 * verifyMessagePayload is the TypeScript parity implementation of Go
 * VerifyMessagePayload: verifyMessage using the envelope's own nonce, plus a
 * byte-for-byte check of the decoded payload against expectedPayload.
 */
export declare function verifyMessagePayload(signed: SignedMessageEnvelope, expectedDomain: string, expectedPurpose: string, expectedChainId: string, expectedPayload: Uint8Array, now: Date): Promise<void>;
//# sourceMappingURL=signing.d.ts.map