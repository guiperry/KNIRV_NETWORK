import { Frame, Registry } from './codec';
import { ModalityType } from './spec';
export declare class NRVReader {
    private path;
    private data;
    private registry;
    constructor(path: string, data: Uint8Array, registry: Registry);
    static open(path: string): Promise<NRVReader>;
    getFrame(id: string): Frame | null;
    getModality(frameId: string, modality: ModalityType): Uint8Array | null;
    streamFrames(modalityFilter?: ModalityType): Generator<Frame>;
    getRegistry(): Registry;
    verifyFrame(id: string, publicKey: Uint8Array, signature: Uint8Array): boolean;
    private decodeFrame;
    close(): void;
}
//# sourceMappingURL=reader.d.ts.map