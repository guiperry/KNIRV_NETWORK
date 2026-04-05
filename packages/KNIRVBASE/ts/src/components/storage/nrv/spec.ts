export const NRV_MAGIC = 0x4E525621;
export const NRV_VERSION = 1;
export const NRV_ALIGNMENT = 8;
export const NRV_REGISTRY_PADDING = 5 * 1024 * 1024;
export const NRV_HEADER_SIZE = 12;

export enum ModalityType {
  Vector = 'vector',
  Seed = 'seed',
  Thermo = 'thermo',
  Proof = 'proof',
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

export function align8(n: number): number {
  return (n + 7) & ~7;
}

export function createModalityMap(proofLength: number): ModalityMap {
  const alignedProof = align8(proofLength);
  return {
    [ModalityType.Vector]: { offset: 0, length: 48 },
    [ModalityType.Seed]: { offset: 48, length: 32 },
    [ModalityType.Thermo]: { offset: 80, length: 16 },
    [ModalityType.Proof]: { offset: 96, length: proofLength },
  };
}
