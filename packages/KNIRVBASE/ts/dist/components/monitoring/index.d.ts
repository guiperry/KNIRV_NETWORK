import { KNIRVBasemetrics } from './monitoring';
import { Counter, Gauge, Histogram } from './metrics';
import { CounterOptions, GaugeOptions, HistogramOptions, MetricOptions } from './types';
export { KNIRVBasemetrics, Counter, Gauge, Histogram, CounterOptions, GaugeOptions, HistogramOptions, MetricOptions };
/**
 * Factory function to create KNIRVBASE metrics
 */
export declare function createMetrics(): KNIRVBasemetrics;
/**
 * Create individual metrics
 */
export declare function createCounter(options: CounterOptions): Counter;
export declare function createGauge(options: GaugeOptions): Gauge;
export declare function createHistogram(options: HistogramOptions): Histogram;
/**
 * Default metrics instance
 */
export declare const defaultMetrics: KNIRVBasemetrics;
/**
 * Quick access to common metrics
 */
export declare const metrics: {
    blocks: {
        committed: () => number;
        inc: (labels?: Record<string, string | number>) => void;
    };
    memory: {
        storeOps: () => number;
        incStoreOps: (labels?: Record<string, string | number>) => void;
        retrieveOps: () => number;
        incRetrieveOps: (labels?: Record<string, string | number>) => void;
    };
    cache: {
        hits: () => number;
        incHits: (labels?: Record<string, string | number>) => void;
        misses: () => number;
        incMisses: (labels?: Record<string, string | number>) => void;
        hitRatio: () => number;
    };
    network: {
        activeConnections: () => number;
        setActiveConnections: (value: number, labels?: Record<string, string | number>) => void;
        nrnBalance: () => number;
        setNRNBalance: (value: number, labels?: Record<string, string | number>) => void;
    };
    query: {
        latency: {
            observe: (value: number, labels?: Record<string, string | number>) => void;
            average: () => number;
        };
    };
    errors: {
        count: () => number;
        inc: (labels?: Record<string, string | number>) => void;
    };
    index: {
        size: () => number;
        setSize: (value: number, labels?: Record<string, string | number>) => void;
    };
};
//# sourceMappingURL=index.d.ts.map