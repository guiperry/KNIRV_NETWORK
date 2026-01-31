/**
 * In-memory Prometheus-like metrics implementation
 * This provides a compatible interface without requiring the full Prometheus client
 */

export abstract class Metric {
  protected name: string;
  protected help: string;
  protected labelNames: string[];

  constructor(options: { name: string; help: string; labelNames?: string[] }) {
    this.name = options.name;
    this.help = options.help;
    this.labelNames = options.labelNames || [];
  }

  getName(): string {
    return this.name;
  }

  getHelp(): string {
    return this.help;
  }

  getLabelNames(): string[] {
    return this.labelNames;
  }

  abstract getValues(): any;
}

export class Counter extends Metric {
  private value: number = 0;
  private labeledValues: Map<string, number> = new Map();

  constructor(options: { name: string; help: string; labelNames?: string[] }) {
    super(options);
  }

  inc(labels?: Record<string, string | number>): void {
    if (labels) {
      const key = this.getLabelsKey(labels);
      this.labeledValues.set(key, (this.labeledValues.get(key) || 0) + 1);
    } else {
      this.value++;
    }
  }

  add(value: number, labels?: Record<string, string | number>): void {
    if (labels) {
      const key = this.getLabelsKey(labels);
      this.labeledValues.set(key, (this.labeledValues.get(key) || 0) + value);
    } else {
      this.value += value;
    }
  }

  get(): number {
    return this.value;
  }

  getWithLabels(labels: Record<string, string | number>): number {
    const key = this.getLabelsKey(labels);
    return this.labeledValues.get(key) || 0;
  }

  reset(): void {
    this.value = 0;
    this.labeledValues.clear();
  }

  private getLabelsKey(labels: Record<string, string | number>): string {
    return this.labelNames
      .map(name => `${name}="${labels[name] || ''}"`)
      .join(',');
  }

  getValues(): any {
    const values: any = {};
    if (this.value > 0) {
      values.default = this.value;
    }
    
    this.labeledValues.forEach((value, key) => {
      values[key] = value;
    });
    
    return values;
  }
}

export class Gauge extends Metric {
  private value: number = 0;
  private labeledValues: Map<string, number> = new Map();

  constructor(options: { name: string; help: string; labelNames?: string[] }) {
    super(options);
  }

  set(value: number, labels?: Record<string, string | number>): void {
    if (labels) {
      const key = this.getLabelsKey(labels);
      this.labeledValues.set(key, value);
    } else {
      this.value = value;
    }
  }

  inc(labels?: Record<string, string | number>): void {
    if (labels) {
      const key = this.getLabelsKey(labels);
      this.labeledValues.set(key, (this.labeledValues.get(key) || 0) + 1);
    } else {
      this.value++;
    }
  }

  dec(labels?: Record<string, string | number>): void {
    if (labels) {
      const key = this.getLabelsKey(labels);
      this.labeledValues.set(key, (this.labeledValues.get(key) || 0) - 1);
    } else {
      this.value--;
    }
  }

  add(value: number, labels?: Record<string, string | number>): void {
    if (labels) {
      const key = this.getLabelsKey(labels);
      this.labeledValues.set(key, (this.labeledValues.get(key) || 0) + value);
    } else {
      this.value += value;
    }
  }

  sub(value: number, labels?: Record<string, string | number>): void {
    if (labels) {
      const key = this.getLabelsKey(labels);
      this.labeledValues.set(key, (this.labeledValues.get(key) || 0) - value);
    } else {
      this.value -= value;
    }
  }

  get(): number {
    return this.value;
  }

  getWithLabels(labels: Record<string, string | number>): number {
    const key = this.getLabelsKey(labels);
    return this.labeledValues.get(key) || 0;
  }

  private getLabelsKey(labels: Record<string, string | number>): string {
    return this.labelNames
      .map(name => `${name}="${labels[name] || ''}"`)
      .join(',');
  }

  getValues(): any {
    const values: any = {};
    values.default = this.value;
    
    this.labeledValues.forEach((value, key) => {
      values[key] = value;
    });
    
    return values;
  }
}

export class Histogram extends Metric {
  private buckets: number[];
  private observations: number[] = [];
  private labeledObservations: Map<string, number[]> = new Map();

  constructor(options: { name: string; help: string; buckets?: number[]; labelNames?: string[] }) {
    super(options);
    this.buckets = options.buckets || [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10];
  }

  observe(value: number, labels?: Record<string, string | number>): void {
    if (labels) {
      const key = this.getLabelsKey(labels);
      const observations = this.labeledObservations.get(key) || [];
      observations.push(value);
      this.labeledObservations.set(key, observations);
    } else {
      this.observations.push(value);
    }
  }

  getBuckets(): number[] {
    return [...this.buckets];
  }

  getBucketCounts(labels?: Record<string, string | number>): number[] {
    const observations = labels 
      ? this.labeledObservations.get(this.getLabelsKey(labels)) || []
      : this.observations;
    
    return this.buckets.map(bucket => 
      observations.filter(value => value <= bucket).length
    );
  }

  getCount(labels?: Record<string, string | number>): number {
    const observations = labels 
      ? this.labeledObservations.get(this.getLabelsKey(labels)) || []
      : this.observations;
    
    return observations.length;
  }

  getSum(labels?: Record<string, string | number>): number {
    const observations = labels 
      ? this.labeledObservations.get(this.getLabelsKey(labels)) || []
      : this.observations;
    
    return observations.reduce((sum, value) => sum + value, 0);
  }

  private getLabelsKey(labels: Record<string, string | number>): string {
    return this.labelNames
      .map(name => `${name}="${labels[name] || ''}"`)
      .join(',');
  }

  getValues(): any {
    const result: any = {
      buckets: this.buckets,
      observations: {
        default: {
          count: this.getCount(),
          sum: this.getSum(),
          bucket_counts: this.getBucketCounts()
        }
      }
    };
    
    this.labeledObservations.forEach((observations, key) => {
      result.observations[key] = {
        count: observations.length,
        sum: observations.reduce((sum, value) => sum + value, 0),
        bucket_counts: this.getBucketCounts(
          Object.fromEntries(
            key.split(',').map(pair => {
              const [name, value] = pair.split('=');
              return [name, value.replace(/"/g, '')];
            })
          )
        )
      };
    });
    
    return result;
  }

  reset(): void {
    this.observations = [];
    this.labeledObservations.clear();
  }
}