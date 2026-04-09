export * from '../components/types/types';
export * from '../components/clock/vector_clock';
export * from '../components/collection/distributed_collection';
export * from '../components/network/network_manager';
export { FileStorage, Storage } from '../components/storage/storage';
export { NRVStorage } from '../components/storage/nrv/nrv_storage';
export { Frame } from '../components/collection/distributed_collection';
export * from '../components/resolver/crdt_resolver';
export * from '../components/crypto/pqc/keys';
export * from '../components/crypto/pqc/encryption';
export * from '../components/query/index';
export * from '../components/auth/index';
export * from '../components/logging/index';
export * from '../components/security/index';
export * from '../components/monitoring/index';
export * from '../components/embedding/index';
export * from '../components/indexing';
export * from '../components/tracing';
import { Bracket, ThermoAtmosphere } from '../components/storage/nrv/bracket';
import { FlightServer, FlightClient, bracketsToFlightData, flightDataToBrackets } from '../components/storage/nrv/flight';
import { Network } from '../components/network/network_manager';
import { Storage } from '../components/storage/storage';
import { NRVStorage } from '../components/storage/nrv/nrv_storage';
import { DistributedCollection } from '../components/collection/distributed_collection';
import { NetworkConfig } from '../components/types/types';
import { PQCKeyPair } from '../components/crypto/pqc/keys';
export interface Options {
    dataDir: string;
    distributedEnabled: boolean;
    distributedNetworkID?: string;
    distributedBootstrapPeers?: string[];
}
export interface DistributedDbOptions {
    distributed: {
        enabled: boolean;
        networkId?: string;
        bootstrapPeers?: string[];
    };
}
export declare class DB {
    private store;
    private network;
    private distributed;
    private collections;
    private dhtCache;
    constructor(options: Options);
    initialize(): Promise<void>;
    createNetwork(cfg: NetworkConfig): Promise<string>;
    joinNetwork(networkID: string, bootstrapPeers: string[]): Promise<void>;
    leaveNetwork(networkID: string): Promise<void>;
    collection(name: string): Collection;
    put(key: string, value: Uint8Array): Promise<void>;
    get(key: string): Promise<Uint8Array | null>;
    deleteKey(key: string): Promise<void>;
    storeObject(key: string, obj: any): Promise<void>;
    getObject<T = any>(key: string): Promise<T | null>;
    putDHT(key: string, value: any, ttlMs?: number): Promise<void>;
    getDHT(key: string): Promise<any[] | null>;
    createIndex(collection: string, name: string, indexType: any, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>): Promise<void>;
    dropIndex(collection: string, name: string): Promise<void>;
    getIndex(collection: string, name: string): any;
    getIndexesForCollection(collection: string): any[];
    queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]>;
    setMasterKey(keyPair: PQCKeyPair): void;
    projectToMarkdown(key: string, targetPath: string): Promise<void>;
    getNetworkManager(): Network;
    dataset(name: string): NRVDataset;
    shutdown(): Promise<void>;
}
export declare class DistributedDatabase {
    private network;
    private storage;
    private distributed;
    private collections;
    constructor(ctx: any, opts: DistributedDbOptions, store: Storage, mockNet?: Network);
    private initializeDistributed;
    collection(name: string): DistributedCollection;
    createNetwork(cfg: NetworkConfig): Promise<string>;
    joinNetwork(networkID: string, bootstrapPeers: string[]): Promise<void>;
    leaveNetwork(networkID: string): Promise<void>;
    addCollectionToNetwork(networkID: string, collectionName: string): Promise<void>;
    removeCollectionFromNetwork(collectionName: string): Promise<void>;
    getNetworkManager(): Network;
    setMasterKey(keyPair: PQCKeyPair): void;
    put(key: string, value: Uint8Array): Promise<void>;
    get(key: string): Promise<Uint8Array | null>;
    deleteKey(key: string): Promise<void>;
    storeObject(key: string, obj: any): Promise<void>;
    getObject<T = any>(key: string): Promise<T | null>;
    projectToMarkdown(key: string, targetPath: string): Promise<void>;
    createIndex(collection: string, name: string, indexType: any, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>): Promise<void>;
    dropIndex(collection: string, name: string): Promise<void>;
    getIndex(collection: string, name: string): any;
    getIndexesForCollection(collection: string): any[];
    queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]>;
    shutdown(): Promise<void>;
}
export interface Collection {
    insert(doc: Record<string, any>): Promise<Record<string, any>>;
    update(id: string, update: Record<string, any>): Promise<number>;
    delete(id: string): Promise<number>;
    find(id: string): Promise<Record<string, any> | null>;
    findAll(): Promise<Record<string, any>[]>;
    attachToNetwork(networkID: string): Promise<void>;
    detachFromNetwork(): Promise<void>;
    forceSync(): Promise<void>;
}
export declare function New(ctx: any, opts: Options): Promise<DB>;
export declare function NewDistributedDatabase(ctx: any, opts: DistributedDbOptions, store: Storage, mockNet?: Network): Promise<DistributedDatabase>;
export declare class NRVDataset {
    readonly name: string;
    private storage;
    private collection;
    constructor(name: string, storage: NRVStorage, collection: DistributedCollection);
    appendBracket(bracket: Bracket, thermo: ThermoAtmosphere): Promise<void>;
    streamBrackets(goldOnly: boolean): AsyncIterableIterator<Bracket>;
    getFrame(frameId: string): Promise<{
        frame: any;
        brackets: Bracket[];
    } | null>;
    setLinguistic(token: string, unit: string): Promise<void>;
}
export declare function NewNRV(ctx: any, opts: Options, keyPair: PQCKeyPair): Promise<DB>;
export interface DBWithDataset extends DB {
    dataset(name: string): NRVDataset;
}
export { Bracket, ThermoAtmosphere };
export { FlightServer, FlightClient, bracketsToFlightData, flightDataToBrackets };
//# sourceMappingURL=index.d.ts.map