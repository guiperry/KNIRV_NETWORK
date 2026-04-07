import { Counter, Gauge, Histogram } from './metrics';
export declare function createMetrics(): KNIRVBasemetrics;
/**
 * Comprehensive metrics collection for KNIRVBASE
 */
export declare class KNIRVBasemetrics {
    readonly blocksCommitted: Counter;
    readonly blockCommitDuration: Histogram;
    readonly memoryStoreOps: Counter;
    readonly memoryRetrieveOps: Counter;
    readonly cacheHits: Counter;
    readonly cacheMisses: Counter;
    readonly activeConnections: Gauge;
    readonly nrnBalance: Gauge;
    readonly queryLatency: Histogram;
    readonly errorCount: Counter;
    readonly indexSize: Gauge;
    constructor();
    /**
     * Get all metrics as a plain object for serialization
     */
    getAllMetrics(): any;
    /**
     * Reset all metrics (useful for testing)
     */
    resetAll(): void;
    /**
     * Get cache hit ratio
     */
    getCacheHitRatio(): number;
    /**
     * Get average block commit duration
     */
    getAverageBlockCommitDuration(): number;
    /**
     * Get average query latency
     */
    getAverageQueryLatency(): number;
}
//# sourceMappingURL=monitoring.d.ts.map