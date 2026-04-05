export * from './nrv';
export declare enum IndexType {
    BTree = "btree",
    GIN = "gin",
    HNSW = "hnsw",
    Tag = "tag"
}
export declare class BTreeIndex {
    data: Map<string, string[]>;
}
export declare class GINIndex {
    data: Map<string, string[]>;
}
export declare class HNSWIndex {
    dimensions: number;
    vectors: Map<string, number[]>;
    neighbors: Map<string, string[]>;
    constructor(dimensions: number);
}
export interface Block {
    id: string;
    timestamp: number;
    category: string;
    vector?: number[];
    tags: string[];
}
export declare class TagIndex {
    private blocksByTag;
    private blocksById;
    add(block: Block): void;
    remove(id: string): void;
    search(tags: string[]): string[];
    getBlock(id: string): Block | undefined;
    clear(): void;
}
export declare class Index {
    name: string;
    collection: string;
    type: IndexType;
    fields: string[];
    unique: boolean;
    partialExpr: string;
    options: Record<string, any>;
    btreeIndex?: BTreeIndex;
    ginIndex?: GINIndex;
    hnswIndex?: HNSWIndex;
    tagIndex?: TagIndex;
    constructor(name: string, collection: string, type: IndexType, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>);
}
export declare class IndexManager {
    private baseDir;
    private indexes;
    constructor(baseDir: string);
    createIndex(collection: string, name: string, indexType: IndexType, fields: string[], unique: boolean, partialExpr: string, options: Record<string, any>): Promise<void>;
    dropIndex(collection: string, name: string): Promise<void>;
    getIndex(collection: string, name: string): Index | null;
    getIndexesForCollection(collection: string): Index[];
    insert(collection: string, doc: Record<string, any>): Promise<void>;
    delete(collection: string, docID: string): Promise<void>;
    queryIndex(collection: string, indexName: string, query: Record<string, any>): Promise<string[]>;
    loadIndexes(): void;
    private matchesPartial;
    private insertBTree;
    private deleteBTree;
    private queryBTree;
    private insertGIN;
    private deleteGIN;
    private queryGIN;
    private insertHNSW;
    private deleteHNSW;
    private queryHNSW;
    private insertTag;
    private deleteTag;
    private queryTag;
    private buildCompositeKey;
    private tokenizeJSON;
    private tokenizeValue;
    private cosineSimilarity;
    private saveIndexMetadata;
}
//# sourceMappingURL=index.d.ts.map