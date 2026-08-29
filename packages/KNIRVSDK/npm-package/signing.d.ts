export declare const KNIRV_HD_PATH = "m/44'/118'/0'/0/i";
export declare const ACTION_SCHEMA_VERSION = "knirv.action.v1";
export declare const MESSAGE_SCHEMA_VERSION = "knirv.message.v1";
export declare const ACTION_TYPE_URL = "/knirv.signing.v1.Action";
export declare const SECP256K1_TYPE_URL = "/cosmos.crypto.secp256k1.PubKey";
export declare const SIGN_MODE_DIRECT = 1;
export declare const RELAY_ENVELOPE_SCHEMA_VERSION = "knirv.controller.relay-envelope.v1";
export declare const WASM_PUBLICATION_SCHEMA_VERSION = "knirv.wasm_publication.v1";
export declare const WASM_MANIFEST_SCHEMA_VERSION = "knirv.wasm_manifest.v1";
export declare const CONTROLLER_DOMAIN = "knirv.controller";
export declare const PURPOSE_WASM_PUBLICATION = "wasm-publication";
export declare const PURPOSE_WASM_ASSIGNMENT = "wasm-assignment";
export declare const PURPOSE_WASM_DOWNLOAD_GRANT = "wasm-download-grant";
export declare const PURPOSE_RELAY_REQUEST = "relay-request";
export declare const PURPOSE_RELAY_RESPONSE = "relay-response";
export declare const RELAY_TARGET_DVE_EXPERT_ADVISOR = "dve_expert_advisor";
export declare const RELAY_TARGET_CLI_SUPERVISOR = "cli_supervisor";
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
export interface RelayEnvelope {
    schemaVersion?: string;
    requestId: string;
    userSubject: string;
    deviceId: string;
    dveId?: string;
    targetType: typeof RELAY_TARGET_DVE_EXPERT_ADVISOR | typeof RELAY_TARGET_CLI_SUPERVISOR;
    targetId: string;
    capability: string;
    sequence: Uint64;
    leaseEpoch?: Uint64;
    issuedAtUnix: Uint64;
    expiresAtUnix: Uint64;
    payloadDigest: string;
}
export interface ParsedRelayEnvelope {
    schemaVersion: string;
    requestId: string;
    userSubject: string;
    deviceId: string;
    dveId: string;
    targetType: RelayEnvelope['targetType'];
    targetId: string;
    capability: string;
    sequence: bigint;
    leaseEpoch: bigint;
    issuedAtUnix: bigint;
    expiresAtUnix: bigint;
    payloadDigest: string;
}
export interface WasmPublicationPayload {
    schemaVersion?: string;
    networkId: string;
    networkFingerprint: string;
    artifactDigest: string;
    byteSize: Uint64;
    moduleKind: string;
    abiVersion?: Uint64;
    moduleSchemaVersion?: Uint64;
    buildId: string;
    toolchainDigest?: string;
    selfTestDigest?: string;
    publisherAddress: string;
    dveTemplateId?: string;
}
export interface WasmManifestModule {
    moduleKind: string;
    artifactDigest: string;
    byteSize: Uint64;
    abiVersion: Uint64;
    moduleSchemaVersion: Uint64;
    capabilitiesJson: string;
    configurationDigest: string;
    downloadPath: string;
    publisherAddress: string;
    publicationStatementDigest: string;
}
export interface WasmManifestPayload {
    schemaVersion?: string;
    manifestId: string;
    networkId: string;
    chainId: string;
    networkFingerprint: string;
    userSubject: string;
    deviceId: string;
    dveId?: string;
    leaseEpoch?: Uint64;
    modules: WasmManifestModule[];
    relayTargetType?: string;
    relayTargetId?: string;
    assignmentId: string;
    assignmentVersion?: Uint64;
    supersedesAssignmentId?: string;
}
export declare function marshalAction(action: KNIRVAction): Uint8Array;
export declare function buildDirectSignDoc(request: DirectSignRequest, compressedPublicKey: Uint8Array): {
    bodyBytes: Uint8Array<ArrayBuffer>;
    authInfoBytes: Uint8Array<ArrayBuffer>;
    signDoc: Uint8Array<ArrayBuffer>;
};
export declare const marshalTxRaw: (bodyBytes: Uint8Array, authInfoBytes: Uint8Array, signatures: Uint8Array[]) => Uint8Array<ArrayBuffer>;
export declare function publicKeyToKNIRVAddress(compressedPublicKey: Uint8Array): string;
export declare function signDirectTransaction(privateKey: Uint8Array, request: DirectSignRequest): Promise<SignedDirectTransaction>;
export declare function marshalMessageEnvelope(envelope: MessageEnvelope): Uint8Array;
export declare function marshalRelayEnvelope(envelope: RelayEnvelope): Uint8Array;
export declare function parseRelayEnvelope(data: Uint8Array): ParsedRelayEnvelope;
export declare function marshalWasmPublicationPayload(payload: WasmPublicationPayload): Uint8Array;
export declare function marshalWasmManifestPayload(payload: WasmManifestPayload): Uint8Array;
export declare function parseWasmManifestPayload(data: Uint8Array): WasmManifestPayload;
export declare function signMessageEnvelope(privateKey: Uint8Array, envelope: MessageEnvelope): Promise<SignedMessageEnvelope>;
export declare function verifyMessageEnvelope(signed: SignedMessageEnvelope, expected: MessageEnvelope): Promise<boolean>;
export declare function parseMessageEnvelope(data: Uint8Array): ParsedMessageEnvelope;
export declare function verifyMessage(signed: SignedMessageEnvelope, expectedDomain: string, expectedPurpose: string, expectedChainId: string, expectedNonce: string, now: Date): Promise<void>;
export declare function verifyMessagePayload(signed: SignedMessageEnvelope, expectedDomain: string, expectedPurpose: string, expectedChainId: string, expectedPayload: Uint8Array, now: Date): Promise<void>;
