import { Counter, Gauge, Histogram } from './metrics';
/**
 * Comprehensive metrics collection for KNIRVBASE
 */
export class KNIRVBasemetrics {
    constructor() {
        // Block operations
        this.blocksCommitted = new Counter({
            name: 'knirvbase_blocks_committed_total',
            help: 'Total number of blocks committed to chain'
        });
        this.blockCommitDuration = new Histogram({
            name: 'knirvbase_block_commit_duration_seconds',
            help: 'Time taken to commit a block',
            buckets: [0.001, 0.002, 0.004, 0.008, 0.016, 0.032, 0.064, 0.128, 0.256, 0.512, 1.024]
        });
        // Memory operations
        this.memoryStoreOps = new Counter({
            name: 'knirvbase_memory_store_ops_total',
            help: 'Total number of memory store operations'
        });
        this.memoryRetrieveOps = new Counter({
            name: 'knirvbase_memory_retrieve_ops_total',
            help: 'Total number of memory retrieve operations'
        });
        // Cache performance
        this.cacheHits = new Counter({
            name: 'knirvbase_cache_hits_total',
            help: 'Total number of cache hits'
        });
        this.cacheMisses = new Counter({
            name: 'knirvbase_cache_misses_total',
            help: 'Total number of cache misses'
        });
        // Network metrics
        this.activeConnections = new Gauge({
            name: 'knirvbase_active_connections',
            help: 'Number of active connections'
        });
        this.nrnBalance = new Gauge({
            name: 'knirvbase_nrn_balance',
            help: 'Current NRN token balance'
        });
        // Query performance
        this.queryLatency = new Histogram({
            name: 'knirvbase_query_latency_seconds',
            help: 'Query latency distribution',
            buckets: [0.01, 0.02, 0.04, 0.08, 0.16, 0.32, 0.64, 1.28, 2.56, 5.12, 10.24]
        });
        // Error tracking
        this.errorCount = new Counter({
            name: 'knirvbase_errors_total',
            help: 'Total number of errors'
        });
        // Index metrics
        this.indexSize = new Gauge({
            name: 'knirvbase_index_size_bytes',
            help: 'Size of index in bytes'
        });
    }
    /**
     * Get all metrics as a plain object for serialization
     */
    getAllMetrics() {
        return {
            blocksCommitted: this.blocksCommitted.getValues(),
            blockCommitDuration: this.blockCommitDuration.getValues(),
            memoryStoreOps: this.memoryStoreOps.getValues(),
            memoryRetrieveOps: this.memoryRetrieveOps.getValues(),
            cacheHits: this.cacheHits.getValues(),
            cacheMisses: this.cacheMisses.getValues(),
            activeConnections: this.activeConnections.getValues(),
            nrnBalance: this.nrnBalance.getValues(),
            queryLatency: this.queryLatency.getValues(),
            errorCount: this.errorCount.getValues(),
            indexSize: this.indexSize.getValues()
        };
    }
    /**
     * Reset all metrics (useful for testing)
     */
    resetAll() {
        this.blocksCommitted.reset();
        this.blockCommitDuration.reset();
        this.memoryStoreOps.reset();
        this.memoryRetrieveOps.reset();
        this.cacheHits.reset();
        this.cacheMisses.reset();
        this.indexSize.reset();
    }
    /**
     * Get cache hit ratio
     */
    getCacheHitRatio() {
        const hits = this.cacheHits.get();
        const misses = this.cacheMisses.get();
        const total = hits + misses;
        return total > 0 ? hits / total : 0;
    }
    /**
     * Get average block commit duration
     */
    getAverageBlockCommitDuration() {
        const count = this.blockCommitDuration.getCount();
        const sum = this.blockCommitDuration.getSum();
        return count > 0 ? sum / count : 0;
    }
    /**
     * Get average query latency
     */
    getAverageQueryLatency() {
        const count = this.queryLatency.getCount();
        const sum = this.queryLatency.getSum();
        return count > 0 ? sum / count : 0;
    }
}
//# sourceMappingURL=monitoring.js.map