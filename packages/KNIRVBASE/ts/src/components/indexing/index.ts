export enum IndexType {
  Semantic = 'semantic',
  Temporal = 'temporal',
  Category = 'category',
  FullText = 'fulltext',
  Tag = 'tag',
}

export enum IndexMemoryCategory {
  Episode = 'episode',
  Semantic = 'semantic',
  Procedural = 'procedural',
  Intentional = 'intentional',
}

export interface IndexBlock {
  getBlockID(): string;
  getTimestamp(): number;
  getCategory(): IndexMemoryCategory;
  getSemanticVector(): Float32Array | null;
}

export class BlockImpl implements IndexBlock {
  constructor(
    public readonly id: string,
    public readonly timestamp: number,
    public readonly category: IndexMemoryCategory,
    public readonly semanticVector: Float32Array | null
  ) {}

  getBlockID(): string {
    return this.id;
  }

  getTimestamp(): number {
    return this.timestamp;
  }

  getCategory(): IndexMemoryCategory {
    return this.category;
  }

  getSemanticVector(): Float32Array | null {
    return this.semanticVector;
  }
}

export interface Index {
  add(ctx: any, block: IndexBlock): Promise<void>;
  search(ctx: any, query: any): Promise<string[]>;
  remove(ctx: any, blockID: string): Promise<void>;
  rebuild(ctx: any): Promise<void>;
}

export interface IndexConfig {
  dimensions?: number;
  efConstruction?: number;
  m?: number;
  maxNeighbors?: number;
}

export class HNSWIndex implements Index {
  private vectors: Map<string, Float32Array> = new Map();
  private neighbors: Map<string, string[]> = new Map();
  private entryPoints: string[] = [];
  private dimensions: number;
  private efConstruction: number;
  private m: number;
  private maxNeighbors: number;

  constructor(config: IndexConfig = {}) {
    this.dimensions = config.dimensions || 768;
    this.efConstruction = config.efConstruction || 200;
    this.m = config.m || 16;
    this.maxNeighbors = config.maxNeighbors || 50;
  }

  async add(ctx: any, block: IndexBlock): Promise<void> {
    const vector = block.getSemanticVector();
    if (!vector || vector.length !== this.dimensions) {
      return;
    }

    this.vectors.set(block.getBlockID(), vector);
    this.neighbors.set(block.getBlockID(), []);

    if (this.entryPoints.length === 0) {
      this.entryPoints.push(block.getBlockID());
    } else {
      const nearest = this.findNearest(vector, 1);
      if (nearest.length > 0) {
        this.neighbors.get(nearest[0])?.push(block.getBlockID());
        this.neighbors.set(nearest[0], this.neighbors.get(nearest[0])!);
      }
    }
  }

  async search(ctx: any, query: any): Promise<string[]> {
    const queryVector = query.vector as Float32Array;
    if (!queryVector) {
      return [];
    }

    const limit = query.limit || 10;
    return this.findNearest(queryVector, limit);
  }

  async remove(ctx: any, blockID: string): Promise<void> {
    this.vectors.delete(blockID);
    this.neighbors.delete(blockID);

    for (const neighbors of this.neighbors.values()) {
      const idx = neighbors.indexOf(blockID);
      if (idx !== -1) {
        neighbors.splice(idx, 1);
      }
    }
  }

  async rebuild(ctx: any): Promise<void> {
    const entries = Array.from(this.vectors.entries());
    this.vectors.clear();
    this.neighbors.clear();
    this.entryPoints = [];

    for (const [id, vector] of entries) {
      this.vectors.set(id, vector);
      this.neighbors.set(id, []);
    }

    if (entries.length > 0) {
      this.entryPoints = [entries[0][0]];
    }
  }

  private findNearest(query: Float32Array, limit: number): string[] {
    if (this.vectors.size === 0) {
      return [];
    }

    const results: { id: string; score: number }[] = [];

    for (const [id, vector] of this.vectors) {
      const score = this.cosineSimilarity(query, vector);
      results.push({ id, score });
    }

    results.sort((a, b) => b.score - a.score);
    return results.slice(0, limit).map(r => r.id);
  }

  private cosineSimilarity(a: Float32Array, b: Float32Array): number {
    if (a.length !== b.length) return 0;
    let dotProduct = 0;
    let normA = 0;
    let normB = 0;
    for (let i = 0; i < a.length; i++) {
      dotProduct += a[i] * b[i];
      normA += a[i] * a[i];
      normB += b[i] * b[i];
    }
    if (normA === 0 || normB === 0) return 0;
    return dotProduct / (Math.sqrt(normA) * Math.sqrt(normB));
  }
}

export class TemporalIndex implements Index {
  private blocks: Map<string, IndexBlock> = new Map();
  private byTimestamp: Map<number, string[]> = new Map();

  async add(ctx: any, block: IndexBlock): Promise<void> {
    this.blocks.set(block.getBlockID(), block);
    
    const ts = Math.floor(block.getTimestamp() / 1000) * 1000;
    if (!this.byTimestamp.has(ts)) {
      this.byTimestamp.set(ts, []);
    }
    this.byTimestamp.get(ts)!.push(block.getBlockID());
  }

  async search(ctx: any, query: any): Promise<string[]> {
    const startTime = query.startTime as number;
    const endTime = query.endTime as number;
    const limit = query.limit || 100;

    if (startTime === undefined || endTime === undefined) {
      return [];
    }

    const results: string[] = [];
    const startKey = Math.floor(startTime / 1000) * 1000;
    const endKey = Math.floor(endTime / 1000) * 1000;

    for (const [ts, ids] of this.byTimestamp) {
      if (ts >= startKey && ts <= endKey) {
        results.push(...ids);
      }
    }

    return results.slice(0, limit);
  }

  async remove(ctx: any, blockID: string): Promise<void> {
    const block = this.blocks.get(blockID);
    if (!block) return;

    const ts = Math.floor(block.getTimestamp() / 1000) * 1000;
    const ids = this.byTimestamp.get(ts);
    if (ids) {
      const idx = ids.indexOf(blockID);
      if (idx !== -1) {
        ids.splice(idx, 1);
      }
    }
    this.blocks.delete(blockID);
  }

  async rebuild(ctx: any): Promise<void> {
    const entries = Array.from(this.blocks.entries());
    this.blocks.clear();
    this.byTimestamp.clear();

    for (const [id, block] of entries) {
      await this.add(ctx, block);
    }
  }
}

export class CategoryIndex implements Index {
  private blocks: Map<string, IndexBlock> = new Map();
  private byCategory: Map<IndexMemoryCategory, Set<string>> = new Map();

  async add(ctx: any, block: IndexBlock): Promise<void> {
    this.blocks.set(block.getBlockID(), block);
    
    const cat = block.getCategory();
    if (!this.byCategory.has(cat)) {
      this.byCategory.set(cat, new Set());
    }
    this.byCategory.get(cat)!.add(block.getBlockID());
  }

  async search(ctx: any, query: any): Promise<string[]> {
    const category = query.category as IndexMemoryCategory;
    if (!category) {
      return [];
    }

    const ids = this.byCategory.get(category);
    return ids ? Array.from(ids) : [];
  }

  async remove(ctx: any, blockID: string): Promise<void> {
    const block = this.blocks.get(blockID);
    if (!block) return;

    const ids = this.byCategory.get(block.getCategory());
    if (ids) {
      ids.delete(blockID);
    }
    this.blocks.delete(blockID);
  }

  async rebuild(ctx: any): Promise<void> {
    const entries = Array.from(this.blocks.entries());
    this.blocks.clear();
    this.byCategory.clear();

    for (const [id, block] of entries) {
      await this.add(ctx, block);
    }
  }
}

export class FullTextIndex implements Index {
  private blocks: Map<string, IndexBlock> = new Map();
  private invertedIndex: Map<string, Set<string>> = new Map();

  async add(ctx: any, block: IndexBlock): Promise<void> {
    this.blocks.set(block.getBlockID(), block);
    // Full-text indexing would require access to text content
    // This is a simplified placeholder
  }

  async search(ctx: any, query: any): Promise<string[]> {
    const token = query.token as string;
    const tokens = query.tokens as string[];
    
    if (token) {
      const ids = this.invertedIndex.get(token);
      return ids ? Array.from(ids) : [];
    }

    if (tokens && tokens.length > 0) {
      const resultSets: Set<string>[] = [];
      for (const t of tokens) {
        const ids = this.invertedIndex.get(t);
        if (ids) {
          resultSets.push(new Set(ids));
        }
      }
      
      if (resultSets.length === 0) return [];
      
      let intersection = resultSets[0];
      for (let i = 1; i < resultSets.length; i++) {
        intersection = new Set([...intersection].filter(x => resultSets[i].has(x)));
      }
      
      return Array.from(intersection);
    }

    return [];
  }

  async remove(ctx: any, blockID: string): Promise<void> {
    this.blocks.delete(blockID);
    // Remove from inverted index - would need to track which tokens this block had
  }

  async rebuild(ctx: any): Promise<void> {
    // Full rebuild would re-index all content
  }
}

export class TagIndex implements Index {
  private blocks: Map<string, IndexBlock> = new Map();
  private byTag: Map<string, Set<string>> = new Map();

  async add(ctx: any, block: IndexBlock): Promise<void> {
    this.blocks.set(block.getBlockID(), block);
    // Tags would come from block metadata
  }

  async search(ctx: any, query: any): Promise<string[]> {
    const tags = query.tags as string[];
    
    if (!tags || tags.length === 0) {
      return Array.from(this.blocks.keys());
    }

    const resultSets: Set<string>[] = [];
    for (const tag of tags) {
      const ids = this.byTag.get(tag);
      if (ids) {
        resultSets.push(new Set(ids));
      }
    }

    if (resultSets.length === 0) return [];
    
    let intersection = resultSets[0];
    for (let i = 1; i < resultSets.length; i++) {
      intersection = new Set([...intersection].filter(x => resultSets[i].has(x)));
    }
    
    return Array.from(intersection);
  }

  async remove(ctx: any, blockID: string): Promise<void> {
    this.blocks.delete(blockID);
    // Remove from byTag
    for (const ids of this.byTag.values()) {
      ids.delete(blockID);
    }
  }

  async rebuild(ctx: any): Promise<void> {
    const entries = Array.from(this.blocks.entries());
    this.blocks.clear();
    this.byTag.clear();

    for (const [id, block] of entries) {
      await this.add(ctx, block);
    }
  }
}

export class MultiIndexManager {
  private indexes: Map<IndexType, Index> = new Map();
  private blockIndex: Map<string, IndexBlock> = new Map();

  constructor() {
    this.registerDefaultIndexes();
  }

  private registerDefaultIndexes(): void {
    this.registerIndex(IndexType.Semantic, new HNSWIndex());
    this.registerIndex(IndexType.Temporal, new TemporalIndex());
    this.registerIndex(IndexType.Category, new CategoryIndex());
    this.registerIndex(IndexType.FullText, new FullTextIndex());
    this.registerIndex(IndexType.Tag, new TagIndex());
  }

  registerIndex(indexType: IndexType, index: Index): void {
    this.indexes.set(indexType, index);
  }

  getIndex(indexType: IndexType): Index | undefined {
    return this.indexes.get(indexType);
  }

  async addIndexBlock(ctx: any, block: IndexBlock): Promise<void> {
    this.blockIndex.set(block.getBlockID(), block);
    
    for (const index of this.indexes.values()) {
      try {
        await index.add(ctx, block);
      } catch (e) {
        console.error('failed to add block to index:', e);
      }
    }
  }

  async search(ctx: any, indexType: IndexType, query: any): Promise<string[]> {
    const index = this.indexes.get(indexType);
    if (!index) {
      throw new Error(`index not registered: ${indexType}`);
    }
    return index.search(ctx, query);
  }

  async removeIndexBlock(ctx: any, blockID: string): Promise<void> {
    this.blockIndex.delete(blockID);
    
    for (const index of this.indexes.values()) {
      try {
        await index.remove(ctx, blockID);
      } catch (e) {
        console.error('failed to remove block from index:', e);
      }
    }
  }

  async rebuildAll(ctx: any): Promise<void> {
    const blocks = Array.from(this.blockIndex.values());
    
    for (const index of this.indexes.values()) {
      try {
        await index.rebuild(ctx);
      } catch (e) {
        console.error('failed to rebuild index:', e);
      }
    }

    for (const block of blocks) {
      for (const index of this.indexes.values()) {
        try {
          await index.add(ctx, block);
        } catch (e) {
          console.error('failed to re-add block to index:', e);
        }
      }
    }
  }

  getIndexBlock(blockID: string): IndexBlock | undefined {
    return this.blockIndex.get(blockID);
  }

  getAllIndexBlocks(): IndexBlock[] {
    return Array.from(this.blockIndex.values());
  }
}