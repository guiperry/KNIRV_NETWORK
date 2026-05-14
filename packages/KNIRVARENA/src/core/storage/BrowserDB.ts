/**
 * Browser-compatible DB class for KNIRVBASE
 * Uses RemoteStorageClient to communicate with backend server
 */

import { BrowserStorage } from './BrowserStorage';
import { Collection } from '@knirvcorp/knirvbase-ts';

// Temporary type definitions (fallback)
// interface Collection {
//   // placeholder
// }
// interface NetworkConfig {
//   // placeholder
// }

export interface Options {
  dataDir?: string;
  distributedEnabled?: boolean;
  distributedNetworkID?: string;
  distributedBootstrapPeers?: string[];
  sessionId?: string;
  baseUrl?: string;
}

export class BrowserDB {
  private sessionId: string;
  private store: BrowserStorage;
  private collections: Map<string, Collection> = new Map();

  constructor(options: Options = {}) {
    this.sessionId = options.sessionId || `browser_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    this.store = new BrowserStorage(this.sessionId, options.baseUrl);
    
    // Initialize storage if distributed is not enabled
    if (options.distributedEnabled) {
      console.warn('Distributed mode not supported in browser, falling back to local');
    }
  }

  async initialize(): Promise<void> {
    await this.store.initialize();
  }

  collection(name: string): Collection {
    if (this.collections.has(name)) {
      return this.collections.get(name)!;
    }
    
    const coll = new BrowserCollection(name, this.store);
    this.collections.set(name, coll);
    return coll;
  }

  async shutdown(): Promise<void> {
    await this.store.shutdown();
    this.collections.clear();
  }

  getSessionId(): string {
    return this.sessionId;
  }
}

interface BrowserCollectionInterface {
  insert(doc: Record<string, unknown>): Promise<Record<string, unknown>>;
  update(id: string, update: Record<string, unknown>): Promise<number>;
  delete(id: string): Promise<number>;
  find(id: string): Promise<Record<string, unknown> | null>;
  findAll(): Promise<Record<string, unknown>[]>;
  attachToNetwork(networkID: string): Promise<void>;
  detachFromNetwork(): Promise<void>;
  forceSync(): Promise<void>;
}

class BrowserCollection implements Collection, BrowserCollectionInterface {
  constructor(private name: string, private store: BrowserStorage) {}

  async insert(doc: Record<string, unknown>): Promise<Record<string, unknown>> {
    if (!doc.id) {
      doc.id = `${this.name}_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    await this.store.insert(this.name, doc);
    return doc;
  }

  async update(id: string, update: Record<string, unknown>): Promise<number> {
    return this.store.update(this.name, id, update);
  }

  async delete(id: string): Promise<number> {
    return this.store.delete(this.name, id);
  }

  async find(id: string): Promise<Record<string, unknown> | null> {
    return this.store.find(this.name, id);
  }

  async findAll(): Promise<Record<string, unknown>[]> {
    return this.store.findAll(this.name);
  }

  async attachToNetwork(_networkID: string): Promise<void> {
    console.log(`Network attachment not supported in browser mode for collection ${this.name}`);
  }

  async detachFromNetwork(): Promise<void> {
    console.log(`Network detachment not supported in browser mode for collection ${this.name}`);
  }

  async forceSync(): Promise<void> {
    console.log(`Force sync not supported in browser mode for collection ${this.name}`);
  }
}

// Factory function
export async function NewBrowserDB(opts: Options): Promise<BrowserDB> {
  const db = new BrowserDB(opts);
  await db.initialize();
  return db;
}