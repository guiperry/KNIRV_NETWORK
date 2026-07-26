import { KNIRVBasemetrics, createMetrics } from '../monitoring';

describe('KNIRVBasemetrics', () => {
  let metrics: KNIRVBasemetrics;

  beforeEach(() => {
    metrics = createMetrics();
  });

  describe('constructor', () => {
    it('should initialize all metrics', () => {
      expect(metrics.blocksCommitted).toBeDefined();
      expect(metrics.blockCommitDuration).toBeDefined();
      expect(metrics.memoryStoreOps).toBeDefined();
      expect(metrics.memoryRetrieveOps).toBeDefined();
      expect(metrics.cacheHits).toBeDefined();
      expect(metrics.cacheMisses).toBeDefined();
      expect(metrics.activeConnections).toBeDefined();
      expect(metrics.nrnBalance).toBeDefined();
      expect(metrics.queryLatency).toBeDefined();
      expect(metrics.errorCount).toBeDefined();
      expect(metrics.indexSize).toBeDefined();
    });

    it('should have correct metric names', () => {
      expect(metrics.blocksCommitted.getName()).toBe('knirvbase_blocks_committed_total');
      expect(metrics.queryLatency.getName()).toBe('knirvbase_query_latency_seconds');
      expect(metrics.activeConnections.getName()).toBe('knirvbase_active_connections');
    });
  });

  describe('block metrics', () => {
    it('should track blocks committed', () => {
      metrics.blocksCommitted.inc();
      metrics.blocksCommitted.inc();
      expect(metrics.blocksCommitted.get()).toBe(2);
    });

    it('should track block commit duration', () => {
      metrics.blockCommitDuration.observe(0.1);
      metrics.blockCommitDuration.observe(0.2);
      metrics.blockCommitDuration.observe(0.05);

      expect(metrics.blockCommitDuration.getCount()).toBe(3);
      expect(metrics.blockCommitDuration.getSum()).toBeCloseTo(0.35, 5);
      expect(metrics.getAverageBlockCommitDuration()).toBeCloseTo(0.1167, 3);
    });
  });

  describe('memory metrics', () => {
    it('should track memory store operations', () => {
      metrics.memoryStoreOps.inc();
      metrics.memoryStoreOps.add(5);

      expect(metrics.memoryStoreOps.get()).toBe(6);
    });

    it('should track memory retrieve operations', () => {
      metrics.memoryRetrieveOps.inc();
      expect(metrics.memoryRetrieveOps.get()).toBe(1);
    });
  });

  describe('cache metrics', () => {
    it('should track cache hits and misses', () => {
      metrics.cacheHits.inc();
      metrics.cacheHits.add(4);
      metrics.cacheMisses.inc();

      expect(metrics.cacheHits.get()).toBe(5);
      expect(metrics.cacheMisses.get()).toBe(1);
      expect(metrics.getCacheHitRatio()).toBe(5 / 6);
    });

    it('should handle empty cache metrics', () => {
      expect(metrics.getCacheHitRatio()).toBe(0);
    });
  });

  describe('network metrics', () => {
    it('should track active connections', () => {
      metrics.activeConnections.set(10);
      expect(metrics.activeConnections.get()).toBe(10);

      metrics.activeConnections.inc();
      expect(metrics.activeConnections.get()).toBe(11);

      metrics.activeConnections.dec();
      expect(metrics.activeConnections.get()).toBe(10);
    });

    it('should track NRN balance', () => {
      metrics.nrnBalance.set(1000.5);
      expect(metrics.nrnBalance.get()).toBe(1000.5);
    });
  });

  describe('query metrics', () => {
    it('should track query latency', () => {
      metrics.queryLatency.observe(0.05);
      metrics.queryLatency.observe(0.1);
      metrics.queryLatency.observe(0.15);

      expect(metrics.queryLatency.getCount()).toBe(3);
      expect(metrics.queryLatency.getSum()).toBeCloseTo(0.3, 5);
      expect(metrics.getAverageQueryLatency()).toBeCloseTo(0.1, 5);
    });
  });

  describe('error metrics', () => {
    it('should track error count', () => {
      metrics.errorCount.inc();
      metrics.errorCount.add(3);

      expect(metrics.errorCount.get()).toBe(4);
    });
  });

  describe('index metrics', () => {
    it('should track index size', () => {
      metrics.indexSize.set(1024000);
      expect(metrics.indexSize.get()).toBe(1024000);
    });
  });

  describe('labeled metrics', () => {
    it('should handle metrics with labels', () => {
      const labels = { operation: 'read', collection: 'users' };
      
      metrics.memoryStoreOps.inc();
      metrics.memoryStoreOps.inc(labels);
      metrics.memoryStoreOps.inc(labels);

      expect(metrics.memoryStoreOps.get()).toBe(1);
      expect(metrics.memoryStoreOps.getWithLabels(labels)).toBe(2);
    });
  });

  describe('getAllMetrics', () => {
    it('should return all metrics as object', () => {
      metrics.blocksCommitted.inc();
      metrics.activeConnections.set(5);

      const all = metrics.getAllMetrics();

      expect(all.blocksCommitted).toBeDefined();
      expect(all.activeConnections).toBeDefined();
      expect(all.blocksCommitted.default).toBe(1);
      expect(all.activeConnections.default).toBe(5);
    });
  });

  describe('resetAll', () => {
    it('should reset all metrics', () => {
      metrics.blocksCommitted.inc();
      metrics.activeConnections.set(5);
      metrics.queryLatency.observe(0.1);

      metrics.resetAll();

      expect(metrics.blocksCommitted.get()).toBe(0);
      expect(metrics.activeConnections.get()).toBe(0);
      expect(metrics.queryLatency.getCount()).toBe(0);
    });
  });
});