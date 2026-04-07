// LSA Reducer — port of Go's internal/embedding/lsa.go
// Implements Latent Semantic Analysis for dimensionality reduction via power iteration.

function dotProduct(a: number[], b: number[]): number {
  if (a.length !== b.length) return 0;
  let sum = 0;
  for (let i = 0; i < a.length; i++) sum += a[i] * b[i];
  return sum;
}

function vectorNorm(v: number[]): number {
  let sum = 0;
  for (const val of v) sum += val * val;
  return Math.sqrt(sum);
}

export class LSAReducer {
  private components: number[][] = [];
  private mean: number[] = [];
  private targetDim: number;
  private _sourceDim = 0;

  constructor(targetDim: number) {
    this.targetDim = targetDim;
  }

  sourceDimension(): number {
    return this._sourceDim;
  }

  fit(vectors: number[][]): void {
    if (vectors.length === 0) throw new Error('no vectors provided');

    this._sourceDim = vectors[0].length;
    const numVectors = vectors.length;

    // Compute mean vector
    this.mean = new Array<number>(this._sourceDim).fill(0);
    for (const vec of vectors) {
      for (let i = 0; i < this._sourceDim; i++) {
        this.mean[i] += vec[i];
      }
    }
    for (let i = 0; i < this._sourceDim; i++) {
      this.mean[i] /= numVectors;
    }

    // Center the data
    const centered = vectors.map(vec =>
      vec.map((val, i) => val - this.mean[i])
    );

    // Extract top k principal components using power iteration + deflation
    this.components = [];
    const workVectors = centered.map(v => [...v]);

    for (let k = 0; k < this.targetDim && k < this._sourceDim; k++) {
      const component = this.extractComponent(workVectors);
      this.components.push(component);

      // Deflation: remove this component's contribution
      for (const vec of workVectors) {
        const projection = dotProduct(vec, component);
        for (let j = 0; j < vec.length; j++) {
          vec[j] -= projection * component[j];
        }
      }
    }
  }

  private extractComponent(vectors: number[][]): number[] {
    if (vectors.length === 0 || this._sourceDim === 0) {
      return new Array<number>(this._sourceDim).fill(0);
    }

    const initVal = 1.0 / Math.sqrt(this._sourceDim);
    let component = new Array<number>(this._sourceDim).fill(initVal);

    for (let iter = 0; iter < 100; iter++) {
      const newComponent = new Array<number>(this._sourceDim).fill(0);
      for (const vec of vectors) {
        const proj = dotProduct(vec, component);
        for (let j = 0; j < newComponent.length; j++) {
          newComponent[j] += proj * vec[j];
        }
      }

      const norm = vectorNorm(newComponent);
      if (norm > 0) {
        for (let j = 0; j < newComponent.length; j++) {
          newComponent[j] /= norm;
        }
      }

      // Check convergence
      let diff = 0;
      for (let j = 0; j < component.length; j++) {
        diff += Math.abs(newComponent[j] - component[j]);
      }
      component = newComponent;
      if (diff < 1e-6) break;
    }

    return component;
  }

  transform(vector: number[]): number[] {
    if (vector.length !== this._sourceDim) {
      throw new Error(`vector dimension mismatch: expected ${this._sourceDim}, got ${vector.length}`);
    }

    const centered = vector.map((val, i) => val - this.mean[i]);

    const reduced = new Array<number>(this.targetDim).fill(0);
    for (let i = 0; i < this.components.length && i < this.targetDim; i++) {
      reduced[i] = dotProduct(centered, this.components[i]);
    }

    return reduced;
  }

  fitTransform(vectors: number[][]): number[][] {
    this.fit(vectors);
    return vectors.map(v => this.transform(v));
  }
}
