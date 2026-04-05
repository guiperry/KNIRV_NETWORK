export interface WALEntry {
    frameId: string;
    lastGoodLength: number;
    committed: boolean;
}
export declare class WAL {
    private path;
    constructor(path: string);
    begin(entry: WALEntry): Promise<void>;
    commit(frameId: string): Promise<void>;
    recover(): Promise<number>;
    truncate(): Promise<void>;
    private readEntries;
}
//# sourceMappingURL=wal.d.ts.map