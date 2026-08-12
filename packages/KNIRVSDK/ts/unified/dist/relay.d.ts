export declare const RELAY_ENVELOPE_SCHEMA_VERSION = "knirv.controller.relay-envelope.v1";
export declare const RELAY_TARGET_DVE_EXPERT_ADVISOR = "dve_expert_advisor";
export declare const RELAY_TARGET_CLI_SUPERVISOR = "cli_supervisor";
export type RelayUint64 = number | string | bigint;
export interface RelayEnvelope {
    schemaVersion?: string;
    requestId: string;
    userSubject: string;
    deviceId: string;
    dveId?: string;
    targetType: string;
    targetId: string;
    capability: string;
    sequence: RelayUint64;
    leaseEpoch?: RelayUint64;
    issuedAtUnix: RelayUint64;
    expiresAtUnix: RelayUint64;
    payloadDigest: string;
}
export declare function marshalRelayEnvelope(envelope: RelayEnvelope): Uint8Array;
export declare function parseRelayEnvelope(data: Uint8Array): RelayEnvelope;
//# sourceMappingURL=relay.d.ts.map