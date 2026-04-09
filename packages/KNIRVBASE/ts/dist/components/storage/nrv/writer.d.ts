import * as fs from 'fs';
import { Frame, Registry } from './codec';
import { WAL } from './wal';
export interface Signer {
    sign(data: Uint8Array): Promise<string>;
}
export declare class NRVWriter {
    private path;
    private keyPair?;
    private wal;
    private file;
    private registry;
    private position;
    constructor(path: string, file: fs.promises.FileHandle, registry: Registry, wal: WAL, position: number, keyPair?: Signer);
    static create(path: string, keyPair?: Signer): Promise<NRVWriter>;
    appendFrame(frame: Frame, verified: boolean, ergoRank: number): Promise<void>;
    saveRegistry(): Promise<void>;
    /** Mark a frame as tombstoned in the registry (soft-delete). */
    setTombstone(id: string): Promise<void>;
    getRegistry(): Registry;
    close(): Promise<void>;
}
//# sourceMappingURL=writer.d.ts.map