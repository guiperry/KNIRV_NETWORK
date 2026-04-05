export const NRV_MAGIC = 0x4E525621;
export const NRV_VERSION = 1;
export const NRV_ALIGNMENT = 8;
export const NRV_REGISTRY_PADDING = 5 * 1024 * 1024;
export const NRV_HEADER_SIZE = 12;
export var ModalityType;
(function (ModalityType) {
    ModalityType["Vector"] = "vector";
    ModalityType["Seed"] = "seed";
    ModalityType["Thermo"] = "thermo";
    ModalityType["Proof"] = "proof";
})(ModalityType || (ModalityType = {}));
export function align8(n) {
    return (n + 7) & ~7;
}
export function createModalityMap(proofLength) {
    const alignedProof = align8(proofLength);
    return {
        [ModalityType.Vector]: { offset: 0, length: 48 },
        [ModalityType.Seed]: { offset: 48, length: 32 },
        [ModalityType.Thermo]: { offset: 80, length: 16 },
        [ModalityType.Proof]: { offset: 96, length: proofLength },
    };
}
//# sourceMappingURL=spec.js.map