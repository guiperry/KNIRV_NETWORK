import { Index, IndexManager, IndexType } from '../storage/index';
import { Doc, QueryHost } from './executor';

/**
 * In-memory QueryHost backed by the storage IndexManager. Useful for tests,
 * tools and embedding the executor without a full DistributedCollection.
 * `indexDir` is where IndexManager persists index metadata.
 */
export class MemoryQueryHost implements QueryHost {
  private docs = new Map<string, Map<string, Doc>>();
  readonly indexes: IndexManager;

  constructor(indexDir: string) {
    this.indexes = new IndexManager(indexDir);
  }

  private coll(name: string): Map<string, Doc> {
    let c = this.docs.get(name);
    if (!c) this.docs.set(name, (c = new Map()));
    return c;
  }

  async findAll(collection: string): Promise<Doc[]> {
    return [...this.coll(collection).values()];
  }

  async find(collection: string, id: string): Promise<Doc | null> {
    return this.coll(collection).get(id) ?? null;
  }

  getIndexes(collection: string): Index[] {
    return this.indexes.getIndexesForCollection(collection);
  }

  queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]> {
    return this.indexes.queryIndex(collection, indexName, query);
  }

  async insert(collection: string, doc: Doc): Promise<Doc> {
    if (typeof doc.id !== 'string') throw new Error('document id must be a string');
    this.coll(collection).set(doc.id, doc);
    await this.indexes.insert(collection, doc);
    return doc;
  }

  async delete(collection: string, id: string): Promise<void> {
    this.coll(collection).delete(id);
    await this.indexes.delete(collection, id);
  }

  createIndex(
    collection: string, name: string, type: IndexType, fields: string[],
    unique: boolean, partialExpr: string, options: Record<string, any>
  ): Promise<void> {
    return this.indexes.createIndex(collection, name, type, fields, unique, partialExpr, options);
  }

  dropIndex(collection: string, name: string): Promise<void> {
    return this.indexes.dropIndex(collection, name);
  }
}
