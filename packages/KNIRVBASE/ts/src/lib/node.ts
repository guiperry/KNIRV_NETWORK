// Node entry point (`@knirvcorp/knirvbase-ts/node`): storage, network, crypto,
// consensus-adjacent services. Not browser-safe. Express/JWT auth is in `./server`.

// Core types
export * from '../components/types/types';

// Clock
export * from '../components/clock/vector_clock';

// Collection
export * from '../components/collection/distributed_collection';
export { CollectionEventHandler } from '../components/collection/consensus_handler';

// Network
export * from '../components/network/network_manager';

// Storage
export { FileStorage, Storage } from '../components/storage/storage';
export { NRVStorage } from '../components/storage/nrv/nrv_storage';
export { NRVWriter, pqcSigner } from '../components/storage/nrv/writer';
export type { Signer } from '../components/storage/nrv/writer';
export { NRVReader } from '../components/storage/nrv/reader';
export { WAL } from '../components/storage/nrv/wal';
export { FrameTicker, euclideanDrift, aggregateThermo } from '../components/storage/nrv/ticker';

// NRV
export { Frame } from '../components/collection/distributed_collection';

// Resolver
export * from '../components/resolver/crdt_resolver';

// Crypto
export * from '../components/crypto/pqc/keys';
export * from '../components/crypto/pqc/encryption';

// Query
export * from '../components/query/index';

// Logging
export * from '../components/logging/index';

// Security
export * from '../components/security/index';

// Monitoring
export * from '../components/monitoring/index';

// Embedding (TF-IDF + LSA)
export * from '../components/embedding/index';

// Indexing module
export * from '../components/indexing';

// Distributed tracing
export * from '../components/tracing';

// NRV Storage imports
import { Bracket, ThermoAtmosphere } from '../components/storage/nrv/bracket';
import { FlightServer, FlightClient, bracketsToFlightData, flightDataToBrackets } from '../components/storage/nrv/flight';

// Main Database class
import { NetworkManager, Network } from '../components/network/network_manager';
import { FileStorage, Storage } from '../components/storage/storage';
import { NRVStorage } from '../components/storage/nrv/nrv_storage';
import { DistributedCollection } from '../components/collection/distributed_collection';
import { NetworkConfig, NetworkStats } from '../components/types/types';
import { PQCKeyPair } from '../components/crypto/pqc/keys';
import { ConsensusConfig, ConsensusManager, defaultConsensusConfig } from '../components/p2pconsensus';
import { ConsensusFactory, startDatabaseConsensus } from '../components/collection/database_consensus';

export interface Options {
  dataDir: string;
  distributedEnabled: boolean;
  distributedNetworkID?: string;
  distributedBootstrapPeers?: string[];
  /** P2P consensus (Go: Options.Consensus). Not started unless `enabled` is true. */
  consensus?: ConsensusConfig;
  /** Test/embedding hook replacing the default P2PConsensusManager. */
  consensusFactory?: ConsensusFactory;
}

export interface DistributedDbOptions {
  distributed: {
    enabled: boolean;
    networkId?: string;
    bootstrapPeers?: string[];
  };
  consensus?: ConsensusConfig;
  consensusFactory?: ConsensusFactory;
}

export class DB {
  private store: Storage;
  private network: Network;
  private distributed: boolean = false;
  private consensus: ConsensusConfig;
  private consensusManager?: ConsensusManager;
  private consensusFactory?: ConsensusFactory;
  private collections: Map<string, DistributedCollection> = new Map();
  private dhtCache: Map<string, { value: any; expiry: number }> = new Map();

  constructor(options: Options) {
    this.store = new FileStorage(options.dataDir);
    this.network = new NetworkManager();
    this.distributed = options.distributedEnabled;
    // Consensus is opt-in: without an explicit config nothing probes gateways or opens libp2p.
    this.consensus = options.consensus ?? { ...defaultConsensusConfig(), enabled: false };
    this.consensusFactory = options.consensusFactory;
    if (this.distributed && (!this.consensus.network_id || this.consensus.network_id === 'knirvbase-default')) {
      this.consensus = { ...this.consensus, network_id: options.distributedNetworkID ?? this.consensus.network_id };
    }
  }

  getConsensusConfig(): ConsensusConfig {
    return this.consensus;
  }

  async initialize(): Promise<void> {
    await this.network.initialize();

    if (this.consensus.enabled && !this.consensusManager) {
      this.consensusManager = await startDatabaseConsensus(this.consensus, () => this.collections, this.consensusFactory);
      for (const c of this.collections.values()) c.setConsensus(this.consensusManager);
    }
  }

  getConsensusManager(): ConsensusManager | undefined {
    return this.consensusManager;
  }

  async createNetwork(cfg: NetworkConfig): Promise<string> {
    return this.network.createNetwork(cfg);
  }

  async joinNetwork(networkID: string, bootstrapPeers: string[]): Promise<void> {
    await this.network.joinNetwork(networkID, bootstrapPeers);
  }

  async leaveNetwork(networkID: string): Promise<void> {
    await this.network.leaveNetwork(networkID);
  }

  /** Attach a collection (created on demand) to a network. */
  async addCollectionToNetwork(networkID: string, collectionName: string): Promise<void> {
    await this.collection(collectionName).attachToNetwork(networkID);
  }

  async removeCollectionFromNetwork(collectionName: string): Promise<void> {
    const c = this.collections.get(collectionName);
    if (c) await c.detachFromNetwork();
  }

  /** Transport counters for a joined network, or null if it is unknown. */
  networkStats(networkID: string): NetworkStats | null {
    return this.network.getNetworkStats(networkID);
  }

  collection(name: string): Collection {
    if (this.collections.has(name)) {
      return new CollectionAdapter(this.collections.get(name)!);
    }
    const coll = new DistributedCollection(name, this.network, this.store);
    coll.setConsensus(this.consensusManager);
    this.collections.set(name, coll);
    return new CollectionAdapter(coll);
  }

  // Key-Value operations
  async put(key: string, value: Uint8Array): Promise<void> {
    return this.store.put(key, value);
  }

  async get(key: string): Promise<Uint8Array | null> {
    return this.store.get(key);
  }

  async deleteKey(key: string): Promise<void> {
    return this.store.deleteKey(key);
  }

  // JSON Object operations
  async storeObject(key: string, obj: any): Promise<void> {
    return this.store.storeObject(key, obj);
  }

  async getObject<T = any>(key: string): Promise<T | null> {
    return this.store.getObject(key);
  }

  // DHT operations
  async putDHT(key: string, value: any, ttlMs: number = 60000): Promise<void> {
    const expiry = Date.now() + ttlMs;
    this.dhtCache.set(key, { value, expiry });
    
    // Broadcast to network if available
    const network = this.network as any;
    if (network.putDHT) {
      await network.putDHT(key, value, ttlMs);
    }
  }

  async getDHT(key: string): Promise<any[] | null> {
    // Check local cache first
    const cached = this.dhtCache.get(key);
    if (cached) {
      if (cached.expiry > Date.now()) {
        return [cached.value];
      } else {
        this.dhtCache.delete(key);
      }
    }
    
    // Query network
    const network = this.network as any;
    if (network.getDHT) {
      return network.getDHT(key);
    }
    
    return null;
  }

  // Index management
  async createIndex(collection: string, name: string, indexType: any, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>): Promise<void> {
    return this.store.createIndex(collection, name, indexType, fields, unique, partialExpr, options);
  }

  async dropIndex(collection: string, name: string): Promise<void> {
    return this.store.dropIndex(collection, name);
  }

  getIndex(collection: string, name: string): any {
    return this.store.getIndex(collection, name);
  }

  getIndexesForCollection(collection: string): any[] {
    return this.store.getIndexesForCollection(collection);
  }

  async queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]> {
    return this.store.queryIndex(collection, indexName, query);
  }

  // Master key for encryption
  setMasterKey(keyPair: PQCKeyPair): void {
    const fs = this.store as FileStorage;
    if (fs.setMasterKey) {
      fs.setMasterKey(keyPair);
    }
  }

  // Markdown projection
  async projectToMarkdown(key: string, targetPath: string): Promise<void> {
    return this.store.projectToMarkdown(key, targetPath);
  }

  // Get network manager
  getNetworkManager(): Network {
    return this.network;
  }

  dataset(name: string): NRVDataset {
    const nrvStore = this.store as NRVStorage;
    if (!(nrvStore instanceof NRVStorage)) {
      throw new Error('DB was not created with NRVStorage — use NewNRV() constructor');
    }
    const coll = this.collection(name);
    return new NRVDataset(name, nrvStore, (coll as any).coll);
  }

  async shutdown(): Promise<void> {
    await this.consensusManager?.stop();
    await this.network.shutdown();
    await this.store.close();
  }
}

export class DistributedDatabase {
  private network: Network;
  private storage: Storage;
  private distributed: boolean = false;
  private collections: Map<string, DistributedCollection> = new Map();
  private consensusManager?: ConsensusManager;
  private initializing: Promise<void> = Promise.resolve();

  constructor(ctx: any, opts: DistributedDbOptions, store: Storage, mockNet?: Network) {
    this.storage = store;
    
    if (mockNet) {
      this.network = mockNet;
    } else {
      this.network = new NetworkManager();
    }
    
    this.distributed = opts.distributed.enabled;
    
    const consensus = opts.consensus;
    this.initializing = (async () => {
      if (this.distributed) await this.initializeDistributed(opts);
      if (consensus?.enabled) {
        this.consensusManager = await startDatabaseConsensus(consensus, () => this.collections, opts.consensusFactory);
        for (const c of this.collections.values()) c.setConsensus(this.consensusManager);
      }
    })();
    // Surfaced through ready(); avoid an unhandled rejection if nobody awaits it.
    this.initializing.catch(() => undefined);
  }

  /**
   * Resolves once network setup and (if enabled) consensus startup finish; rejects
   * with their error. Go does this work inside NewDistributedDatabase.
   */
  ready(): Promise<void> {
    return this.initializing;
  }

  getConsensusManager(): ConsensusManager | undefined {
    return this.consensusManager;
  }

  private async initializeDistributed(opts: DistributedDbOptions): Promise<void> {
    await this.network.initialize();
    
    if (opts.distributed.networkId) {
      if (opts.distributed.bootstrapPeers && opts.distributed.bootstrapPeers.length > 0) {
        await this.network.joinNetwork(opts.distributed.networkId, opts.distributed.bootstrapPeers);
      } else {
        const networkConfig: NetworkConfig = {
          networkId: opts.distributed.networkId,
          name: `Network ${opts.distributed.networkId}`,
          collections: {},
          bootstrapPeers: [],
          defaultPostingNetwork: '',
          autoPostClassifications: [],
          privateByDefault: true,
          encryption: { enabled: false, sharedSecret: '' },
          replication: { factor: 3, strategy: 'full' },
          discovery: { mdns: true, bootstrap: true },
        };
        await this.network.createNetwork(networkConfig);
      }
    }
  }

  collection(name: string): DistributedCollection {
    if (this.collections.has(name)) {
      return this.collections.get(name)!;
    }
    
    const coll = new DistributedCollection(name, this.network, this.storage);
    coll.setConsensus(this.consensusManager);
    this.collections.set(name, coll);
    return coll;
  }

  async createNetwork(cfg: NetworkConfig): Promise<string> {
    if (!this.network) {
      throw new Error('network manager not initialized');
    }
    return this.network.createNetwork(cfg);
  }

  async joinNetwork(networkID: string, bootstrapPeers: string[]): Promise<void> {
    if (!this.network) {
      throw new Error('network manager not initialized');
    }
    await this.network.joinNetwork(networkID, bootstrapPeers);
  }

  async leaveNetwork(networkID: string): Promise<void> {
    if (!this.network) {
      throw new Error('network manager not initialized');
    }
    await this.network.leaveNetwork(networkID);
  }

  async addCollectionToNetwork(networkID: string, collectionName: string): Promise<void> {
    const coll = this.collections.get(collectionName);
    if (!coll) {
      throw new Error('collection not found');
    }
    await coll.attachToNetwork(networkID);
  }

  async removeCollectionFromNetwork(collectionName: string): Promise<void> {
    const coll = this.collections.get(collectionName);
    if (!coll) {
      return;
    }
    await coll.detachFromNetwork();
  }

  getNetworkManager(): Network {
    return this.network;
  }

  setMasterKey(keyPair: PQCKeyPair): void {
    const fs = this.storage as FileStorage;
    if (fs.setMasterKey) {
      fs.setMasterKey(keyPair);
    }
  }

  async put(key: string, value: Uint8Array): Promise<void> {
    return this.storage.put(key, value);
  }

  async get(key: string): Promise<Uint8Array | null> {
    return this.storage.get(key);
  }

  async deleteKey(key: string): Promise<void> {
    return this.storage.deleteKey(key);
  }

  async storeObject(key: string, obj: any): Promise<void> {
    return this.storage.storeObject(key, obj);
  }

  async getObject<T = any>(key: string): Promise<T | null> {
    return this.storage.getObject(key);
  }

  async projectToMarkdown(key: string, targetPath: string): Promise<void> {
    return this.storage.projectToMarkdown(key, targetPath);
  }

  async createIndex(collection: string, name: string, indexType: any, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>): Promise<void> {
    return this.storage.createIndex(collection, name, indexType, fields, unique, partialExpr, options);
  }

  async dropIndex(collection: string, name: string): Promise<void> {
    return this.storage.dropIndex(collection, name);
  }

  getIndex(collection: string, name: string): any {
    return this.storage.getIndex(collection, name);
  }

  getIndexesForCollection(collection: string): any[] {
    return this.storage.getIndexesForCollection(collection);
  }

  async queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]> {
    return this.storage.queryIndex(collection, indexName, query);
  }

  async shutdown(): Promise<void> {
    await this.consensusManager?.stop();
    if (this.network) {
      await this.network.shutdown();
    }
  }
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

class CollectionAdapter implements Collection {
  constructor(private coll: DistributedCollection) {}

  async insert(doc: Record<string, any>): Promise<Record<string, any>> {
    return this.coll.insert(doc);
  }

  async update(id: string, update: Record<string, any>): Promise<number> {
    return this.coll.update(id, update);
  }

  async delete(id: string): Promise<number> {
    return this.coll.delete(id);
  }

  async find(id: string): Promise<Record<string, any> | null> {
    return this.coll.find(id);
  }

  async findAll(): Promise<Record<string, any>[]> {
    return this.coll.findAll();
  }

  async attachToNetwork(networkID: string): Promise<void> {
    await this.coll.attachToNetwork(networkID);
  }

  async detachFromNetwork(): Promise<void> {
    await this.coll.detachFromNetwork();
  }

  async forceSync(): Promise<void> {
    await this.coll.forceSync();
  }
}

// Factory function
export async function New(ctx: any, opts: Options): Promise<DB> {
  const db = new DB(opts);
  await db.initialize();
  return db;
}

export async function NewDistributedDatabase(ctx: any, opts: DistributedDbOptions, store: Storage, mockNet?: Network): Promise<DistributedDatabase> {
  const consensus = opts.consensus ?? defaultConsensusConfig();
  if (opts.distributed.enabled && (!consensus.network_id || consensus.network_id === 'knirvbase-default')) {
    consensus.network_id = opts.distributed.networkId ?? consensus.network_id;
    consensus.bootstrap_peers = opts.distributed.bootstrapPeers ?? consensus.bootstrap_peers;
  }
  return new DistributedDatabase(ctx, { ...opts, consensus }, store, mockNet);
}

export class NRVDataset {
  readonly name: string;
  private storage: NRVStorage;
  private collection: DistributedCollection;

  constructor(name: string, storage: NRVStorage, collection: DistributedCollection) {
    this.name = name;
    this.storage = storage;
    this.collection = collection;
  }

  async appendBracket(bracket: Bracket, thermo: ThermoAtmosphere): Promise<void> {
    return this.storage.appendBracket(this.name, bracket, thermo);
  }

  async *streamBrackets(goldOnly: boolean): AsyncIterableIterator<Bracket> {
    const reader = await (this.storage as any).getReader(this.name);
    if (!reader) {
      return;
    }

    const registry = reader.getRegistry();
    for (const entry of registry.frames) {
      if ((entry as any).tombstone !== undefined) {
        continue;
      }
      const frame = reader.getFrame(entry.id);
      if (frame && frame.brackets) {
        for (const bracket of frame.brackets) {
          if (!goldOnly || (bracket.meta && bracket.meta.type === 'P')) {
            yield bracket;
          }
        }
      }
    }
  }

  async getFrame(frameId: string): Promise<{ frame: any; brackets: Bracket[] } | null> {
    const reader = await (this.storage as any).getReader(this.name);
    if (!reader) {
      return null;
    }

    const frame = reader.getFrame(frameId);
    if (!frame) {
      return null;
    }

    return { frame, brackets: (frame as any).brackets || [] };
  }

  async setLinguistic(token: string, unit: string): Promise<void> {
    const registry = await (this.storage as any).getRegistry(this.name);
    if (!registry) {
      throw new Error('collection not found');
    }
    if (!registry.linguistic) {
      (registry as any).linguistic = {};
    }
    (registry as any).linguistic[token] = unit;
    await (this.storage as any).saveRegistry(this.name);
  }
}

export async function NewNRV(ctx: any, opts: Options, keyPair: PQCKeyPair): Promise<DB> {
  const store = new NRVStorage(opts.dataDir, keyPair);
  const db = new DB(opts);
  (db as any).store = store;
  await db.initialize();
  return db;
}

export interface DBWithDataset extends DB {
  dataset(name: string): NRVDataset;
}

export { Bracket, ThermoAtmosphere };
export { FlightServer, FlightClient, bracketsToFlightData, flightDataToBrackets };

// Namespaced: its PeerInfo/SyncRequest wire types differ from the CRDT ones above.
export * as p2pconsensus from '../components/p2pconsensus';
