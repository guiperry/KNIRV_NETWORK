// TF-IDF Vectorizer — port of Go's internal/embedding/tfidf.go

function tokenize(text: string): string[] {
  const lower = text.toLowerCase();
  const tokens: string[] = [];
  let current = '';

  for (const ch of lower) {
    if (/[a-z0-9]/.test(ch)) {
      current += ch;
    } else {
      if (current.length > 1) tokens.push(current);
      current = '';
    }
  }
  if (current.length > 1) tokens.push(current);

  return tokens;
}

export class TFIDFVectorizer {
  private vocabulary: Map<string, number> = new Map();
  private idf: Map<string, number> = new Map();
  private docCount = 0;
  private wordDocCounts: Map<string, number> = new Map();

  fit(documents: string[]): void {
    if (documents.length === 0) throw new Error('no documents provided');

    this.vocabulary = new Map();
    this.idf = new Map();
    this.wordDocCounts = new Map();
    this.docCount = documents.length;

    for (const doc of documents) {
      const tokens = tokenize(doc);
      const uniqueWords = new Set(tokens);
      for (const word of uniqueWords) {
        this.wordDocCounts.set(word, (this.wordDocCounts.get(word) ?? 0) + 1);
      }
    }

    let idx = 0;
    for (const [word, docFreq] of this.wordDocCounts) {
      this.vocabulary.set(word, idx++);
      this.idf.set(word, Math.log(this.docCount / docFreq));
    }
  }

  fitIncremental(document: string): void {
    const tokens = tokenize(document);
    const uniqueWords = new Set(tokens);
    this.docCount++;

    for (const word of uniqueWords) {
      const count = (this.wordDocCounts.get(word) ?? 0) + 1;
      this.wordDocCounts.set(word, count);

      if (!this.vocabulary.has(word)) {
        this.vocabulary.set(word, this.vocabulary.size);
      }

      this.idf.set(word, Math.log(this.docCount / count));
    }

    // Update IDF for words not in this document
    for (const [word] of this.vocabulary) {
      if (!uniqueWords.has(word)) {
        const df = this.wordDocCounts.get(word) ?? 1;
        this.idf.set(word, Math.log(this.docCount / df));
      }
    }
  }

  transform(document: string): number[] {
    if (this.vocabulary.size === 0) throw new Error('vectorizer not fitted');

    const tokens = tokenize(document);
    const tf = new Map<string, number>();
    for (const token of tokens) {
      tf.set(token, (tf.get(token) ?? 0) + 1);
    }

    const docLength = tokens.length;
    if (docLength > 0) {
      for (const [word, freq] of tf) {
        tf.set(word, freq / docLength);
      }
    }

    const vector = new Array<number>(this.vocabulary.size).fill(0);
    for (const [word, freq] of tf) {
      const idx = this.vocabulary.get(word);
      if (idx !== undefined) {
        vector[idx] = freq * (this.idf.get(word) ?? 0);
      }
    }

    return vector;
  }

  fitTransform(documents: string[]): number[][] {
    this.fit(documents);
    return documents.map(doc => this.transform(doc));
  }

  vocabularySize(): number {
    return this.vocabulary.size;
  }

  getVocabulary(): Record<string, number> {
    return Object.fromEntries(this.vocabulary);
  }

  getIdf(): Record<string, number> {
    return Object.fromEntries(this.idf);
  }

  getDocCount(): number {
    return this.docCount;
  }

  getWordDocCounts(): Record<string, number> {
    return Object.fromEntries(this.wordDocCounts);
  }

  restoreVocabulary(
    vocab: Record<string, number>,
    idf: Record<string, number>,
    wordDocCounts: Record<string, number>,
    docCount: number,
  ): void {
    this.vocabulary = new Map(Object.entries(vocab));
    this.idf = new Map(Object.entries(idf));
    this.wordDocCounts = new Map(Object.entries(wordDocCounts));
    this.docCount = docCount;
  }
}
