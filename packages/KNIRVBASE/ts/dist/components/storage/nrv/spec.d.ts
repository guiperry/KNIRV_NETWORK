export declare const NRV_MAGIC = 1314018849;
export declare const NRV_VERSION = 1;
export declare const NRV_ALIGNMENT = 8;
export declare const NRV_REGISTRY_PADDING: number;
export declare const NRV_HEADER_SIZE = 12;
export declare enum ModalityType {
    Vector = "vector",
    Seed = "seed",
    Thermo = "thermo",
    Proof = "proof"
}
export interface ModalityIndex {
    offset: number;
    length: number;
}
export interface ModalityMap {
    [ModalityType.Vector]: ModalityIndex;
    [ModalityType.Seed]: ModalityIndex;
    [ModalityType.Thermo]: ModalityIndex;
    [ModalityType.Proof]?: ModalityIndex;
}
export declare function align8(n: number): number;
export declare function createModalityMap(proofLength: number): ModalityMap;
//# sourceMappingURL=spec.d.ts.map