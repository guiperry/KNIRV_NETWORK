/**
 * Browser-compatible Storage implementation for KNIRVBASE
 * Uses RemoteStorageClient to communicate with backend server
 */

import { RemoteStorageClient, Document } from './RemoteStorageClient';
import { Index, IndexType } from '../../../../KNIRVBASE/ts/src/components/storage/index';
import { EntryType } from '../../../../KNIRVBASE/ts/src/components/types/types';

export interface Storage {
  insert(collection: string, doc: Record<string, any>): Promise<void>;
  update(collection: string, id: string, update: Record<string, any>): Promise<number>;
  delete(collection: string, id: string): Promise<number>;
  find(collection: string, id: string): Promise<Record<string, any> | null>;
  findAll(collection: string): Promise<Record<string, any>[]>;

  // Index management
  createIndex(collection: string, name: string, indexType: IndexType, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>): Promise<void>;
  dropIndex(collection: string, name: string): Promise<void>;
  getIndex(collection: string, name: string): Index | null;
  getIndexesForCollection(collection: string): Index[];
  queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]>;
}

export class BrowserStorage implements Storage {
  private client: RemoteStorageClient;
  private sessionId: string;

  constructor(sessionId: string, baseUrl?: string) {
    this.sessionId = sessionId;
    this.client = new RemoteStorageClient({ sessionId, baseUrl });
  }

  async initialize(): Promise<void> {
    await this.client.initialize();
  }

  private async ensureInitialized(): Promise<void> {
    try {
      await this.client.getInfo();
    } catch (error) {
      // If not initialized, initialize it
      await this.client.initialize();
    }
  }

  async insert(collection: string, doc: Record<string, any>): Promise<void> {
    await this.ensureInitialized();
    
    // Handle MEMORY blob (convert to JSON serializable format)
    const docCopy = this.deepCopyDoc(doc);
    if (docCopy.entryType === EntryType.Memory) {
      const payload = docCopy.payload;
      if (payload && payload.blob !== undefined) {
        // Store blob as JSON string for browser compatibility
        payload.blobRef = JSON.stringify(payload.blob);
        delete payload.blob;
      }
    }

    await this.client.insert(collection, docCopy as Document);
  }

  async update(collection: string, id: string, update: Record<string, any>): Promise<number> {
    await this.ensureInitialized();
    return this.client.update(collection, id, update);
  }

  async delete(collection: string, id: string): Promise<number> {
    await this.ensureInitialized();
    return this.client.delete(collection, id);
  }

  async find(collection: string, id: string): Promise<Record<string, any> | null> {
    await this.ensureInitialized();
    const doc = await this.client.find(collection, id);
    
    if (!doc) {
      return null;
    }

    // Handle blob restoration
    if (doc.entryType === EntryType.Memory) {
      const payload = doc.payload;
      if (payload && payload.blobRef) {
        try {
          payload.blob = JSON.parse(payload.blobRef);
          delete payload.blobRef;
        } catch (e) {
          console.warn('Failed to parse blobRef:', e);
        }
      }
    }

    return doc;
  }

  async findAll(collection: string): Promise<Record<string, any>[]> {
    await this.ensureInitialized();
    const docs = await this.client.findAll(collection);
    
    // Handle blob restoration for all documents
    return docs.map(doc => {
      if (doc.entryType === EntryType.Memory) {
        const payload = doc.payload;
        if (payload && payload.blobRef) {
          try {
            payload.blob = JSON.parse(payload.blobRef);
            delete payload.blobRef;
          } catch (e) {
            console.warn('Failed to parse blobRef:', e);
          }
        }
      }
      return doc;
    });
  }

  // Index management - simplified for browser (indexes managed on backend)
  async createIndex(collection: string, name: string, indexType: IndexType, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>): Promise<void> {
    // In browser mode, indexes are managed by the backend
    // This is a no-op for now, but could be extended to call the backend
    console.log(`Index creation not supported in browser mode for collection ${collection}, index ${name}`);
  }

  async dropIndex(collection: string, name: string): Promise<void> {
    console.log(`Index dropping not supported in browser mode for collection ${collection}, index ${name}`);
  }

  getIndex(collection: string, name: string): Index | null {
    console.log(`Index retrieval not supported in browser mode for collection ${collection}, index ${name}`);
    return null;
  }

  getIndexesForCollection(collection: string): Index[] {
    console.log(`Index list retrieval not supported in browser mode for collection ${collection}`);
    return [];
  }

  async queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]> {
    console.log(`Index query not supported in browser mode for collection ${collection}, index ${indexName}`);
    return [];
  }

  private deepCopyDoc(doc: Record<string, any>): Record<string, any> {
    return JSON.parse(JSON.stringify(doc));
  }

  async shutdown(): Promise<void> {
    try {
      await this.client.close();
    } catch (error) {
      console.warn('Error closing storage client:', error);
    }
  }
}