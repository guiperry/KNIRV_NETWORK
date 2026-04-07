/**
 * In-memory Prometheus-like metrics implementation
 * This provides a compatible interface without requiring the full Prometheus client
 */
export declare abstract class Metric {
    protected name: string;
    protected help: string;
    protected labelNames: string[];
    constructor(options: {
        name: string;
        help: string;
        labelNames?: string[];
    });
    getName(): string;
    getHelp(): string;
    getLabelNames(): string[];
    abstract getValues(): any;
}
export declare class Counter extends Metric {
    private value;
    private labeledValues;
    constructor(options: {
        name: string;
        help: string;
        labelNames?: string[];
    });
    inc(labels?: Record<string, string | number>): void;
    add(value: number, labels?: Record<string, string | number>): void;
    get(): number;
    getWithLabels(labels: Record<string, string | number>): number;
    getTotal(): number;
    reset(): void;
    private getLabelsKey;
    getValues(): any;
}
export declare class Gauge extends Metric {
    private value;
    private labeledValues;
    constructor(options: {
        name: string;
        help: string;
        labelNames?: string[];
    });
    set(value: number, labels?: Record<string, string | number>): void;
    inc(labels?: Record<string, string | number>): void;
    dec(labels?: Record<string, string | number>): void;
    add(value: number, labels?: Record<string, string | number>): void;
    sub(value: number, labels?: Record<string, string | number>): void;
    get(): number;
    getWithLabels(labels: Record<string, string | number>): number;
    reset(): void;
    private getLabelsKey;
    getValues(): any;
}
export declare class Histogram extends Metric {
    private buckets;
    private observations;
    private labeledObservations;
    constructor(options: {
        name: string;
        help: string;
        buckets?: number[];
        labelNames?: string[];
    });
    observe(value: number, labels?: Record<string, string | number>): void;
    getBuckets(): number[];
    getBucketCounts(labels?: Record<string, string | number>): number[];
    getCount(labels?: Record<string, string | number>): number;
    getSum(labels?: Record<string, string | number>): number;
    private getLabelsKey;
    getValues(): any;
    reset(): void;
}
//# sourceMappingURL=metrics.d.ts.map