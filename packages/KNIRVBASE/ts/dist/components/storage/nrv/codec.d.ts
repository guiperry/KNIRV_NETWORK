import { ModalityMap } from './spec';
export interface Header {
    magic: number;
    version: number;
    totalLength: number;
}
export interface Frame {
    id: string;
    vector: Float32Array;
    seed: Uint8Array;
    thermo: ThermoData;
    proof: Uint8Array;
}
export interface ThermoData {
    tempCelsius: number;
    voltageV: number;
    freqMHz: number;
    fanRPM: number;
}
export interface FrameEntry {
    id: string;
    offset: number;
    length: number;
    tombstone?: number;
    verified: boolean;
    ergoRank: number;
    modalities: ModalityMap;
}
export interface GlobalMetrics {
    featureMin: Float32Array;
    featureMax: Float32Array;
    featureMean: Float32Array;
    featureStd: Float32Array;
    thermoCorrelationCoefficient: number;
    ergoRankSum: number;
    verifiedFrameCount: number;
    compactedAt?: string;
}
export interface PQCManifest {
    keyId: string;
    algorithm: string;
    fileSignature: string;
    frameSignatures: Record<string, string>;
}
export interface Registry {
    version: number;
    datasetId: string;
    datasetVersion: Record<string, number>;
    chunk0Length: number;
    frameCount: number;
    tombstoneCount: number;
    globalMetrics: GlobalMetrics;
    frames: FrameEntry[];
    pqcManifest: PQCManifest;
}
export declare function encodeHeader(totalLength: number): Uint8Array;
export declare function decodeHeader(data: Uint8Array): Header;
export declare function encodeFrame(frame: Frame): {
    data: Uint8Array;
    modalities: ModalityMap;
};
export declare function decodeFrame(data: Uint8Array, entry: FrameEntry): Frame;
export declare function createDefaultRegistry(datasetId: string, keyId?: string): Registry;
//# sourceMappingURL=codec.d.ts.map