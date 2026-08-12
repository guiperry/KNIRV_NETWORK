export declare const WASM_PUBLICATION_SCHEMA_VERSION = "knirv.wasm_publication.v1";
export declare const WASM_MANIFEST_SCHEMA_VERSION = "knirv.wasm_manifest.v1";
export declare const CONTROLLER_DOMAIN = "knirv.controller";
export declare const PURPOSE_WASM_PUBLICATION = "wasm-publication";
export declare const PURPOSE_WASM_ASSIGNMENT = "wasm-assignment";
export declare const PURPOSE_WASM_DOWNLOAD_GRANT = "wasm-download-grant";
export declare const PURPOSE_RELAY_REQUEST = "relay-request";
export declare const PURPOSE_RELAY_RESPONSE = "relay-response";
export type WasmUint64 = number | string | bigint;
export interface WasmPublicationPayload {
    schemaVersion?: string;
    networkId: string;
    networkFingerprint: string;
    artifactDigest: string;
    byteSize: WasmUint64;
    moduleKind: string;
    abiVersion: WasmUint64;
    moduleSchemaVersion: WasmUint64;
    buildId: string;
    toolchainDigest?: string;
    selfTestDigest?: string;
    publisherAddress: string;
    dveTemplateId?: string;
}
export interface WasmManifestModule {
    moduleKind: string;
    artifactDigest: string;
    byteSize: WasmUint64;
    abiVersion: WasmUint64;
    moduleSchemaVersion: WasmUint64;
    capabilitiesJson?: string;
    configurationDigest?: string;
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
    leaseEpoch?: WasmUint64;
    modules: WasmManifestModule[];
    relayTargetType?: string;
    relayTargetId?: string;
    assignmentId: string;
    assignmentVersion?: WasmUint64;
    supersedesAssignmentId?: string;
}
export declare function marshalWasmPublicationPayload(payload: WasmPublicationPayload): Uint8Array;
export declare function marshalWasmManifestPayload(payload: WasmManifestPayload): Uint8Array;
export declare function parseWasmPublicationPayload(data: Uint8Array): WasmPublicationPayload;
export declare function parseWasmManifestPayload(data: Uint8Array): WasmManifestPayload;
//# sourceMappingURL=wasm.d.ts.map