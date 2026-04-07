// TFIDFEmbedder — port of Go's internal/embedding/embedder.go
// Combines TF-IDF vectorization with LSA dimensionality reduction.

import { TFIDFVectorizer } from './tfidf';
import { LSAReducer } from './lsa';

export interface Storage {
  put(key: string, value: Uint8Array): Promise<void>;
  get(key: string): Promise<Uint8Array | null>;
  has(key: string): Promise<boolean>;
}

export class TFIDFEmbedder {
  private vectorizer: TFIDFVectorizer;
  private reducer: LSAReducer;
  private storage: Storage | null;
  private dim: number;
  private fitted = false;

  constructor(storage: Storage | null, dimension: number) {
    if (dimension <= 0) throw new Error('dimension must be positive');
    this.vectorizer = new TFIDFVectorizer();
    this.reducer = new LSAReducer(dimension);
    this.storage = storage;
    this.dim = dimension;
  }

  dimension(): number {
    return this.dim;
  }

  async fit(documents: string[]): Promise<void> {
    if (documents.length === 0) throw new Error('no documents provided for training');

    this.vectorizer.fit(documents);

    const vectors = documents.map(doc => this.vectorizer.transform(doc));

    if (vectors.length > 0 && vectors[0].length >= this.dim) {
      this.reducer.fit(vectors);
    }

    if (this.storage) {
      await this.saveVocabulary();
    }

    this.fitted = true;
  }

  async fitIncremental(document: string): Promise<void> {
    if (!document) throw new Error('empty document provided');

    this.vectorizer.fitIncremental(document);

    if (this.storage) {
      await this.saveVocabulary();
    }

    this.fitted = true;
  }

  generate(text: string): number[] {
    if (!this.fitted) {
      return new Array<number>(this.dim).fill(0);
    }

    const tfidfVector = this.vectorizer.transform(text);

    if (tfidfVector.length < this.dim || this.reducer.sourceDimension() === 0) {
      return this.normalizeToTarget(tfidfVector);
    }

    const reduced = this.reducer.transform(tfidfVector);
    return reduced.map(v => v as number);
  }

  private normalizeToTarget(vector: number[]): number[] {
    const result = new Array<number>(this.dim).fill(0);
    const copyLen = Math.min(vector.length, this.dim);
    for (let i = 0; i < copyLen; i++) {
      result[i] = vector[i];
    }
    return result;
  }

  private async saveVocabulary(): Promise<void> {
    if (!this.storage) return;

    const enc = (obj: unknown) => new TextEncoder().encode(JSON.stringify(obj));

    await this.storage.put('embedding:vocabulary', enc(this.vectorizer.getVocabulary()));
    await this.storage.put('embedding:idf', enc(this.vectorizer.getIdf()));
    await this.storage.put('embedding:word_doc_counts', enc(this.vectorizer.getWordDocCounts()));
    await this.storage.put('embedding:doc_count', enc(this.vectorizer.getDocCount()));
  }

  async loadVocabulary(): Promise<void> {
    if (!this.storage) throw new Error('no storage configured');

    const exists = await this.storage.has('embedding:vocabulary');
    if (!exists) throw new Error('vocabulary not found in storage');

    const dec = async (key: string) => {
      const bytes = await this.storage!.get(key);
      if (!bytes) throw new Error(`missing key: ${key}`);
      return JSON.parse(new TextDecoder().decode(bytes));
    };

    const vocab = await dec('embedding:vocabulary') as Record<string, number>;
    const idf = await dec('embedding:idf') as Record<string, number>;
    const wordDocCounts = await dec('embedding:word_doc_counts') as Record<string, number>;
    const docCount = await dec('embedding:doc_count') as number;

    this.vectorizer.restoreVocabulary(vocab, idf, wordDocCounts, docCount);
    this.fitted = true;
  }
}
