import { KNIRVBasemetrics } from './monitoring';
import { Counter, Gauge, Histogram } from './metrics';
import { CounterOptions, GaugeOptions, HistogramOptions, MetricOptions } from './types';

// Export all classes and interfaces
export {
  KNIRVBasemetrics,
  Counter,
  Gauge,
  Histogram,
  CounterOptions,
  GaugeOptions,
  HistogramOptions,
  MetricOptions
};

/**
 * Factory function to create KNIRVBASE metrics
 */
export function createMetrics(): KNIRVBasemetrics {
  return new KNIRVBasemetrics();
}

/**
 * Create individual metrics
 */
export function createCounter(options: CounterOptions): Counter {
  return new Counter(options);
}

export function createGauge(options: GaugeOptions): Gauge {
  return new Gauge(options);
}

export function createHistogram(options: HistogramOptions): Histogram {
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
    inc: (labels?: Record<string, string | number>) => defaultMetrics.blocksCommitted.inc(labels)
  },
  memory: {
    storeOps: () => defaultMetrics.memoryStoreOps.get(),
    incStoreOps: (labels?: Record<string, string | number>) => defaultMetrics.memoryStoreOps.inc(labels),
    retrieveOps: () => defaultMetrics.memoryRetrieveOps.get(),
    incRetrieveOps: (labels?: Record<string, string | number>) => defaultMetrics.memoryRetrieveOps.inc(labels)
  },
  cache: {
    hits: () => defaultMetrics.cacheHits.get(),
    incHits: (labels?: Record<string, string | number>) => defaultMetrics.cacheHits.inc(labels),
    misses: () => defaultMetrics.cacheMisses.get(),
    incMisses: (labels?: Record<string, string | number>) => defaultMetrics.cacheMisses.inc(labels),
    hitRatio: () => defaultMetrics.getCacheHitRatio()
  },
  network: {
    activeConnections: () => defaultMetrics.activeConnections.get(),
    setActiveConnections: (value: number, labels?: Record<string, string | number>) => 
      defaultMetrics.activeConnections.set(value, labels),
    nrnBalance: () => defaultMetrics.nrnBalance.get(),
    setNRNBalance: (value: number, labels?: Record<string, string | number>) => 
      defaultMetrics.nrnBalance.set(value, labels)
  },
  query: {
    latency: {
      observe: (value: number, labels?: Record<string, string | number>) => 
        defaultMetrics.queryLatency.observe(value, labels),
      average: () => defaultMetrics.getAverageQueryLatency()
    }
  },
  errors: {
    count: () => defaultMetrics.errorCount.get(),
    inc: (labels?: Record<string, string | number>) => defaultMetrics.errorCount.inc(labels)
  },
  index: {
    size: () => defaultMetrics.indexSize.get(),
    setSize: (value: number, labels?: Record<string, string | number>) => 
      defaultMetrics.indexSize.set(value, labels)
  }
};