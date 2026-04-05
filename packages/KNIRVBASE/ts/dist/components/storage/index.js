export * from './nrv';
import * as fs from 'fs';
import * as path from 'path';
export var IndexType;
(function (IndexType) {
    IndexType["BTree"] = "btree";
    IndexType["GIN"] = "gin";
    IndexType["HNSW"] = "hnsw";
    IndexType["Tag"] = "tag";
})(IndexType || (IndexType = {}));
export class BTreeIndex {
    constructor() {
        this.data = new Map();
    }
}
export class GINIndex {
    constructor() {
        this.data = new Map();
    }
}
export class HNSWIndex {
    constructor(dimensions) {
        this.vectors = new Map();
        this.neighbors = new Map();
        this.dimensions = dimensions;
    }
}
export class TagIndex {
    constructor() {
        this.blocksByTag = new Map();
        this.blocksById = new Map();
    }
    add(block) {
        this.blocksById.set(block.id, block);
        for (const tag of block.tags) {
            if (!this.blocksByTag.has(tag)) {
                this.blocksByTag.set(tag, new Set());
            }
            this.blocksByTag.get(tag).add(block.id);
        }
    }
    remove(id) {
        const block = this.blocksById.get(id);
        if (block) {
            for (const tag of block.tags) {
                const tagSet = this.blocksByTag.get(tag);
                if (tagSet) {
                    tagSet.delete(id);
                    if (tagSet.size === 0) {
                        this.blocksByTag.delete(tag);
                    }
                }
            }
            this.blocksById.delete(id);
        }
    }
    search(tags) {
        if (tags.length === 0) {
            return Array.from(this.blocksById.keys());
        }
        const resultSets = [];
        for (const tag of tags) {
            const tagSet = this.blocksByTag.get(tag);
            if (tagSet) {
                resultSets.push(new Set(tagSet));
            }
        }
        if (resultSets.length === 0) {
            return [];
        }
        let intersection = resultSets[0];
        for (let i = 1; i < resultSets.length; i++) {
            intersection = new Set([...intersection].filter(x => resultSets[i].has(x)));
        }
        return Array.from(intersection);
    }
    getBlock(id) {
        return this.blocksById.get(id);
    }
    clear() {
        this.blocksByTag.clear();
        this.blocksById.clear();
    }
}
export class Index {
    constructor(name, collection, type, fields, unique, partialExpr, options) {
        this.name = name;
        this.collection = collection;
        this.type = type;
        this.fields = fields;
        this.unique = unique;
        this.partialExpr = partialExpr;
        this.options = options;
        switch (type) {
            case IndexType.BTree:
                this.btreeIndex = new BTreeIndex();
                break;
            case IndexType.GIN:
                this.ginIndex = new GINIndex();
                break;
            case IndexType.HNSW:
                const dim = options.dimensions || 768;
                this.hnswIndex = new HNSWIndex(dim);
                break;
            case IndexType.Tag:
                this.tagIndex = new TagIndex();
                break;
        }
    }
}
export class IndexManager {
    constructor(baseDir) {
        this.indexes = new Map();
        this.baseDir = baseDir;
    }
    async createIndex(collection, name, indexType, fields, unique, partialExpr, options) {
        const key = `${collection}:${name}`;
        if (this.indexes.has(key)) {
            throw new Error(`index ${key} already exists`);
        }
        const index = new Index(name, collection, indexType, fields, unique, partialExpr, options);
        this.indexes.set(key, index);
        await this.saveIndexMetadata(index);
    }
    async dropIndex(collection, name) {
        const key = `${collection}:${name}`;
        if (!this.indexes.has(key)) {
            throw new Error(`index ${key} does not exist`);
        }
        this.indexes.delete(key);
        const indexDir = path.join(this.baseDir, collection, 'indexes', name);
        fs.rmSync(indexDir, { recursive: true, force: true });
    }
    getIndex(collection, name) {
        const key = `${collection}:${name}`;
        return this.indexes.get(key) || null;
    }
    getIndexesForCollection(collection) {
        const indexes = [];
        for (const index of this.indexes.values()) {
            if (index.collection === collection) {
                indexes.push(index);
            }
        }
        return indexes;
    }
    async insert(collection, doc) {
        const indexes = this.getIndexesForCollection(collection);
        for (const idx of indexes) {
            if (!this.matchesPartial(idx, doc)) {
                continue;
            }
            const docID = doc.id;
            switch (idx.type) {
                case IndexType.BTree:
                    this.insertBTree(idx, docID, doc);
                    break;
                case IndexType.GIN:
                    this.insertGIN(idx, docID, doc);
                    break;
                case IndexType.HNSW:
                    this.insertHNSW(idx, docID, doc);
                    break;
                case IndexType.Tag:
                    this.insertTag(idx, docID, doc);
                    break;
            }
        }
    }
    async delete(collection, docID) {
        const indexes = this.getIndexesForCollection(collection);
        for (const idx of indexes) {
            switch (idx.type) {
                case IndexType.BTree:
                    this.deleteBTree(idx, docID);
                    break;
                case IndexType.GIN:
                    this.deleteGIN(idx, docID);
                    break;
                case IndexType.HNSW:
                    this.deleteHNSW(idx, docID);
                    break;
                case IndexType.Tag:
                    this.deleteTag(idx, docID);
                    break;
            }
        }
    }
    async queryIndex(collection, indexName, query) {
        const idx = this.getIndex(collection, indexName);
        if (!idx) {
            throw new Error(`index ${collection}:${indexName} not found`);
        }
        switch (idx.type) {
            case IndexType.BTree:
                return this.queryBTree(idx, query);
            case IndexType.GIN:
                return this.queryGIN(idx, query);
            case IndexType.HNSW:
                return this.queryHNSW(idx, query);
            case IndexType.Tag:
                return this.queryTag(idx, query);
            default:
                throw new Error(`unsupported index type: ${idx.type}`);
        }
    }
    loadIndexes() {
        const collectionsDir = this.baseDir;
        try {
            const entries = fs.readdirSync(collectionsDir, { withFileTypes: true });
            for (const entry of entries) {
                if (!entry.isDirectory())
                    continue;
                const collection = entry.name;
                const indexesDir = path.join(collectionsDir, collection, 'indexes');
                try {
                    const indexEntries = fs.readdirSync(indexesDir, { withFileTypes: true });
                    for (const idxEntry of indexEntries) {
                        if (!idxEntry.isDirectory())
                            continue;
                        const indexName = idxEntry.name;
                        const metaPath = path.join(indexesDir, indexName, 'metadata.json');
                        try {
                            const data = fs.readFileSync(metaPath, 'utf8');
                            const idxData = JSON.parse(data);
                            const idx = new Index(idxData.name, idxData.collection, idxData.type, idxData.fields, idxData.unique, idxData.partialExpr, idxData.options);
                            const key = `${collection}:${indexName}`;
                            this.indexes.set(key, idx);
                        }
                        catch { }
                    }
                }
                catch { }
            }
        }
        catch { }
    }
    matchesPartial(idx, doc) {
        if (!idx.partialExpr)
            return true;
        const parts = idx.partialExpr.split('=');
        if (parts.length !== 2)
            return true;
        const field = parts[0].trim();
        const expected = parts[1].trim();
        const val = doc[field];
        return val !== undefined && String(val) === expected;
    }
    insertBTree(idx, docID, doc) {
        const key = this.buildCompositeKey(idx.fields, doc);
        if (idx.unique) {
            if (idx.btreeIndex.data.get(key)?.length) {
                return;
            }
        }
        const existing = idx.btreeIndex.data.get(key) || [];
        existing.push(docID);
        idx.btreeIndex.data.set(key, existing);
    }
    deleteBTree(idx, docID) {
        for (const [key, docIDs] of idx.btreeIndex.data) {
            const index = docIDs.indexOf(docID);
            if (index !== -1) {
                docIDs.splice(index, 1);
                if (docIDs.length === 0) {
                    idx.btreeIndex.data.delete(key);
                }
                else {
                    idx.btreeIndex.data.set(key, docIDs);
                }
                break;
            }
        }
    }
    queryBTree(idx, query) {
        if (query.value !== undefined) {
            const key = String(query.value);
            return idx.btreeIndex.data.get(key) || [];
        }
        return [];
    }
    insertGIN(idx, docID, doc) {
        const tokens = this.tokenizeJSON(doc);
        for (const token of tokens) {
            const existing = idx.ginIndex.data.get(token) || [];
            existing.push(docID);
            idx.ginIndex.data.set(token, existing);
        }
    }
    deleteGIN(idx, docID) {
        for (const [token, docIDs] of idx.ginIndex.data) {
            const index = docIDs.indexOf(docID);
            if (index !== -1) {
                docIDs.splice(index, 1);
                if (docIDs.length === 0) {
                    idx.ginIndex.data.delete(token);
                }
                else {
                    idx.ginIndex.data.set(token, docIDs);
                }
            }
        }
    }
    queryGIN(idx, query) {
        const token = query.token;
        if (token) {
            return idx.ginIndex.data.get(token) || [];
        }
        return [];
    }
    insertHNSW(idx, docID, doc) {
        const payload = doc.payload;
        if (payload?.vector) {
            const vec = payload.vector.map(v => Number(v));
            idx.hnswIndex.vectors.set(docID, vec);
            idx.hnswIndex.neighbors.set(docID, []);
        }
    }
    deleteHNSW(idx, docID) {
        idx.hnswIndex.vectors.delete(docID);
        idx.hnswIndex.neighbors.delete(docID);
    }
    queryHNSW(idx, query) {
        const queryVec = query.vector;
        if (queryVec) {
            const results = [];
            for (const [docID, vec] of idx.hnswIndex.vectors) {
                const score = this.cosineSimilarity(queryVec, vec);
                results.push({ id: docID, score });
            }
            results.sort((a, b) => b.score - a.score);
            const limit = query.limit || 10;
            return results.slice(0, limit).map(r => r.id);
        }
        return [];
    }
    insertTag(idx, docID, doc) {
        const payload = doc.payload;
        const tags = payload?.tags || [];
        const block = {
            id: docID,
            timestamp: doc._timestamp || Date.now(),
            category: doc.category || '',
            vector: payload?.vector,
            tags,
        };
        idx.tagIndex.add(block);
    }
    deleteTag(idx, docID) {
        idx.tagIndex.remove(docID);
    }
    queryTag(idx, query) {
        let tags = query.tags;
        if (!tags) {
            const tag = query.tag;
            if (tag) {
                tags = [tag];
            }
            else {
                return [];
            }
        }
        return idx.tagIndex.search(tags);
    }
    buildCompositeKey(fields, doc) {
        const parts = [];
        for (const field of fields) {
            const val = doc[field];
            if (val !== undefined) {
                parts.push(String(val));
            }
        }
        return parts.join('|');
    }
    tokenizeJSON(doc) {
        const tokens = [];
        this.tokenizeValue(doc, tokens);
        return tokens;
    }
    tokenizeValue(val, tokens) {
        if (typeof val === 'string') {
            const words = val.toLowerCase().split(/\s+/);
            tokens.push(...words);
        }
        else if (typeof val === 'object' && val !== null) {
            if (Array.isArray(val)) {
                for (const item of val) {
                    this.tokenizeValue(item, tokens);
                }
            }
            else {
                for (const value of Object.values(val)) {
                    this.tokenizeValue(value, tokens);
                }
            }
        }
    }
    cosineSimilarity(a, b) {
        if (a.length !== b.length)
            return 0;
        let dotProduct = 0;
        let normA = 0;
        let normB = 0;
        for (let i = 0; i < a.length; i++) {
            dotProduct += a[i] * b[i];
            normA += a[i] * a[i];
            normB += b[i] * b[i];
        }
        if (normA === 0 || normB === 0)
            return 0;
        return dotProduct / (Math.sqrt(normA) * Math.sqrt(normB));
    }
    async saveIndexMetadata(idx) {
        const indexDir = path.join(this.baseDir, idx.collection, 'indexes', idx.name);
        fs.mkdirSync(indexDir, { recursive: true });
        const metaPath = path.join(indexDir, 'metadata.json');
        const data = JSON.stringify({
            name: idx.name,
            collection: idx.collection,
            type: idx.type,
            fields: idx.fields,
            unique: idx.unique,
            partialExpr: idx.partialExpr,
            options: idx.options,
        });
        fs.writeFileSync(metaPath, data, 'utf8');
    }
}
//# sourceMappingURL=index.js.map