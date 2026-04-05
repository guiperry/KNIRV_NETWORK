import { Network } from '../network/network_manager';
import { Storage } from '../storage/storage';
import { CRDTOperation, SyncState } from '../types/types';
import { ModalityType } from '../storage/nrv/spec';
export interface Frame {
    id: string;
    vector: Float32Array;
    seed: Uint8Array;
    thermo: {
        tempCelsius: number;
        voltageV: number;
        freqMHz: number;
        fanRPM: number;
    };
    proof: Uint8Array;
}
export declare class LocalCollection {
    private name;
    private store;
    constructor(name: string, store: Storage);
    insert(doc: Record<string, any>): Promise<Record<string, any>>;
    update(id: string, update: Record<string, any>): Promise<number>;
    delete(id: string): Promise<number>;
    find(id: string): Promise<Record<string, any> | null>;
    findAll(): Promise<Record<string, any>[]>;
    getStore(): Storage;
    private cloneMap;
    private cloneSlice;
}
export declare class DistributedCollection {
    name: string;
    private network;
    private networkID;
    private syncStates;
    private operationLog;
    private maxLogSize;
    private local;
    constructor(name: string, network: Network, store: Storage);
    private setupMessageHandlers;
    attachToNetwork(networkID: string): Promise<void>;
    detachFromNetwork(): Promise<void>;
    insert(doc: Record<string, any>): Promise<Record<string, any>>;
    update(id: string, update: Record<string, any>): Promise<number>;
    delete(id: string): Promise<number>;
    find(id: string): Promise<Record<string, any> | null>;
    findAll(): Promise<Record<string, any>[]>;
    getSyncState(): SyncState | null;
    forceSync(): Promise<void>;
    streamFrames(modality?: ModalityType): AsyncGenerator<Frame, void, unknown>;
    getStore(): Storage;
    getOperationLog(): CRDTOperation[];
    private broadcastOperation;
    private handleRemoteOperation;
    private requestSync;
    private handleSyncRequest;
    private handleSyncResponse;
    private getCurrentVector;
    private pruneOperationLog;
}
//# sourceMappingURL=distributed_collection.d.ts.map