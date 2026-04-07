import { KNIRVBasemetrics } from './monitoring';
import { Counter, Gauge, Histogram } from './metrics';
// Export all classes and interfaces
export { KNIRVBasemetrics, Counter, Gauge, Histogram };
/**
 * Factory function to create KNIRVBASE metrics
 */
export function createMetrics() {
    return new KNIRVBasemetrics();
}
/**
 * Create individual metrics
 */
export function createCounter(options) {
    return new Counter(options);
}
export function createGauge(options) {
    return new Gauge(options);
}
export function createHistogram(options) {
    return new Histogram(options);
}
/**
 * Default metrics instance
 */
export const defaultMetrics = createMetrics();
/**
 * Quick access to common metrics
 */
export const metrics = {
    blocks: {
        committed: () => defaultMetrics.blocksCommitted.get(),
        total: () => defaultMetrics.blocksCommitted.getTotal(),
        inc: (labels) => defaultMetrics.blocksCommitted.inc(labels)
    },
    memory: {
        storeOps: () => defaultMetrics.memoryStoreOps.get(),
        totalStoreOps: () => defaultMetrics.memoryStoreOps.getTotal(),
        incStoreOps: (labels) => defaultMetrics.memoryStoreOps.inc(labels),
        retrieveOps: () => defaultMetrics.memoryRetrieveOps.get(),
        totalRetrieveOps: () => defaultMetrics.memoryRetrieveOps.getTotal(),
        incRetrieveOps: (labels) => defaultMetrics.memoryRetrieveOps.inc(labels)
    },
    cache: {
        hits: () => defaultMetrics.cacheHits.get(),
        totalHits: () => defaultMetrics.cacheHits.getTotal(),
        incHits: (labels) => defaultMetrics.cacheHits.inc(labels),
        misses: () => defaultMetrics.cacheMisses.get(),
        totalMisses: () => defaultMetrics.cacheMisses.getTotal(),
        incMisses: (labels) => defaultMetrics.cacheMisses.inc(labels),
        hitRatio: () => defaultMetrics.getCacheHitRatio()
    },
    network: {
        activeConnections: () => defaultMetrics.activeConnections.get(),
        setActiveConnections: (value, labels) => defaultMetrics.activeConnections.set(value, labels),
        nrnBalance: () => defaultMetrics.nrnBalance.get(),
        setNRNBalance: (value, labels) => defaultMetrics.nrnBalance.set(value, labels)
    },
    query: {
        latency: {
            observe: (value, labels) => defaultMetrics.queryLatency.observe(value, labels),
            average: () => defaultMetrics.getAverageQueryLatency()
        }
    },
    errors: {
        count: () => defaultMetrics.errorCount.get(),
        totalCount: () => defaultMetrics.errorCount.getTotal(),
        inc: (labels) => defaultMetrics.errorCount.inc(labels)
    },
    index: {
        size: () => defaultMetrics.indexSize.get(),
        setSize: (value, labels) => defaultMetrics.indexSize.set(value, labels)
    }
};
//# sourceMappingURL=index.js.map