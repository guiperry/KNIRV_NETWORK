// Main library exports for KNIRVBASE
// Core types
export * from '../components/types/types';
// Clock
export * from '../components/clock/vector_clock';
// Collection
export * from '../components/collection/distributed_collection';
// Network
export * from '../components/network/network_manager';
// Storage
export * from '../components/storage/storage';
export * from '../components/storage/index';
// Resolver
export * from '../components/resolver/crdt_resolver';
// Crypto
export * from '../components/crypto/pqc/keys';
export * from '../components/crypto/pqc/encryption';
// Query
export * from '../components/query/index';
// Authentication
export * from '../components/auth/index';
// Logging
export * from '../components/logging/index';
// Security
export * from '../components/security/index';
// Monitoring
export * from '../components/monitoring/index';
// Main Database class
import { NetworkManager } from '../components/network/network_manager';
import { FileStorage } from '../components/storage/storage';
import { DistributedCollection } from '../components/collection/distributed_collection';
export class DB {
    constructor(options) {
        this.distributed = false;
        this.collections = new Map();
        this.dhtCache = new Map();
        this.store = new FileStorage(options.dataDir);
        this.network = new NetworkManager();
        this.distributed = options.distributedEnabled;
    }
    async initialize() {
        await this.network.initialize();
        if (this.distributed) {
            // Distributed initialization handled in DistributedDatabase
        }
    }
    async createNetwork(cfg) {
        return this.network.createNetwork(cfg);
    }
    async joinNetwork(networkID, bootstrapPeers) {
        await this.network.joinNetwork(networkID, bootstrapPeers);
    }
    async leaveNetwork(networkID) {
        await this.network.leaveNetwork(networkID);
    }
    collection(name) {
        if (this.collections.has(name)) {
            return new CollectionAdapter(this.collections.get(name));
        }
        const coll = new DistributedCollection(name, this.network, this.store);
        this.collections.set(name, coll);
        return new CollectionAdapter(coll);
    }
    // Key-Value operations
    async put(key, value) {
        return this.store.put(key, value);
    }
    async get(key) {
        return this.store.get(key);
    }
    async deleteKey(key) {
        return this.store.deleteKey(key);
    }
    // JSON Object operations
    async storeObject(key, obj) {
        return this.store.storeObject(key, obj);
    }
    async getObject(key) {
        return this.store.getObject(key);
    }
    // DHT operations
    async putDHT(key, value, ttlMs = 60000) {
        const expiry = Date.now() + ttlMs;
        this.dhtCache.set(key, { value, expiry });
        // Broadcast to network if available
        const network = this.network;
        if (network.putDHT) {
            await network.putDHT(key, value, ttlMs);
        }
    }
    async getDHT(key) {
        // Check local cache first
        const cached = this.dhtCache.get(key);
        if (cached) {
            if (cached.expiry > Date.now()) {
                return [cached.value];
            }
            else {
                this.dhtCache.delete(key);
            }
        }
        // Query network
        const network = this.network;
        if (network.getDHT) {
            return network.getDHT(key);
        }
        return null;
    }
    // Index management
    async createIndex(collection, name, indexType, fields, unique, partialExpr, options) {
        return this.store.createIndex(collection, name, indexType, fields, unique, partialExpr, options);
    }
    async dropIndex(collection, name) {
        return this.store.dropIndex(collection, name);
    }
    getIndex(collection, name) {
        return this.store.getIndex(collection, name);
    }
    getIndexesForCollection(collection) {
        return this.store.getIndexesForCollection(collection);
    }
    async queryIndex(collection, indexName, query) {
        return this.store.queryIndex(collection, indexName, query);
    }
    // Master key for encryption
    setMasterKey(keyPair) {
        const fs = this.store;
        if (fs.setMasterKey) {
            fs.setMasterKey(keyPair);
        }
    }
    // Markdown projection
    async projectToMarkdown(key, targetPath) {
        return this.store.projectToMarkdown(key, targetPath);
    }
    // Get network manager
    getNetworkManager() {
        return this.network;
    }
    async shutdown() {
        await this.network.shutdown();
        await this.store.close();
    }
}
export class DistributedDatabase {
    constructor(ctx, opts, store, mockNet) {
        this.distributed = false;
        this.collections = new Map();
        this.storage = store;
        if (mockNet) {
            this.network = mockNet;
        }
        else {
            this.network = new NetworkManager();
        }
        this.distributed = opts.distributed.enabled;
        if (this.distributed) {
            this.initializeDistributed(opts);
        }
    }
    async initializeDistributed(opts) {
        await this.network.initialize();
        if (opts.distributed.networkId) {
            if (opts.distributed.bootstrapPeers && opts.distributed.bootstrapPeers.length > 0) {
                await this.network.joinNetwork(opts.distributed.networkId, opts.distributed.bootstrapPeers);
            }
            else {
                const networkConfig = {
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
    collection(name) {
        if (this.collections.has(name)) {
            return this.collections.get(name);
        }
        const coll = new DistributedCollection(name, this.network, this.storage);
        this.collections.set(name, coll);
        return coll;
    }
    async createNetwork(cfg) {
        if (!this.network) {
            throw new Error('network manager not initialized');
        }
        return this.network.createNetwork(cfg);
    }
    async joinNetwork(networkID, bootstrapPeers) {
        if (!this.network) {
            throw new Error('network manager not initialized');
        }
        await this.network.joinNetwork(networkID, bootstrapPeers);
    }
    async leaveNetwork(networkID) {
        if (!this.network) {
            throw new Error('network manager not initialized');
        }
        await this.network.leaveNetwork(networkID);
    }
    async addCollectionToNetwork(networkID, collectionName) {
        const coll = this.collections.get(collectionName);
        if (!coll) {
            throw new Error('collection not found');
        }
        await coll.attachToNetwork(networkID);
    }
    async removeCollectionFromNetwork(collectionName) {
        const coll = this.collections.get(collectionName);
        if (!coll) {
            return;
        }
        await coll.detachFromNetwork();
    }
    getNetworkManager() {
        return this.network;
    }
    setMasterKey(keyPair) {
        const fs = this.storage;
        if (fs.setMasterKey) {
            fs.setMasterKey(keyPair);
        }
    }
    async put(key, value) {
        return this.storage.put(key, value);
    }
    async get(key) {
        return this.storage.get(key);
    }
    async deleteKey(key) {
        return this.storage.deleteKey(key);
    }
    async storeObject(key, obj) {
        return this.storage.storeObject(key, obj);
    }
    async getObject(key) {
        return this.storage.getObject(key);
    }
    async projectToMarkdown(key, targetPath) {
        return this.storage.projectToMarkdown(key, targetPath);
    }
    async createIndex(collection, name, indexType, fields, unique, partialExpr, options) {
        return this.storage.createIndex(collection, name, indexType, fields, unique, partialExpr, options);
    }
    async dropIndex(collection, name) {
        return this.storage.dropIndex(collection, name);
    }
    getIndex(collection, name) {
        return this.storage.getIndex(collection, name);
    }
    getIndexesForCollection(collection) {
        return this.storage.getIndexesForCollection(collection);
    }
    async queryIndex(collection, indexName, query) {
        return this.storage.queryIndex(collection, indexName, query);
    }
    async shutdown() {
        if (this.network) {
            await this.network.shutdown();
        }
    }
}
class CollectionAdapter {
    constructor(coll) {
        this.coll = coll;
    }
    async insert(doc) {
        return this.coll.insert(doc);
    }
    async update(id, update) {
        return this.coll.update(id, update);
    }
    async delete(id) {
        return this.coll.delete(id);
    }
    async find(id) {
        return this.coll.find(id);
    }
    async findAll() {
        return this.coll.findAll();
    }
    async attachToNetwork(networkID) {
        await this.coll.attachToNetwork(networkID);
    }
    async detachFromNetwork() {
        await this.coll.detachFromNetwork();
    }
    async forceSync() {
        await this.coll.forceSync();
    }
}
// Factory function
export async function New(ctx, opts) {
    const db = new DB(opts);
    await db.initialize();
    return db;
}
export async function NewDistributedDatabase(ctx, opts, store, mockNet) {
    return new DistributedDatabase(ctx, opts, store, mockNet);
}
//# sourceMappingURL=index.js.map