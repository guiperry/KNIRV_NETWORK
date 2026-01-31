import { createMetrics, metrics, defaultMetrics } from '../index';

describe('Monitoring Index', () => {
  describe('factory functions', () => {
    it('should create metrics instance', () => {
      const metrics = createMetrics();
      expect(metrics).toBeDefined();
      expect(metrics.blocksCommitted).toBeDefined();
    });
  });

  describe('default metrics', () => {
    it('should have default metrics instance', () => {
      expect(defaultMetrics).toBeDefined();
      expect(defaultMetrics.blocksCommitted).toBeDefined();
    });
  });

  describe('quick access metrics', () => {
    beforeEach(() => {
      // Reset default metrics before each test
      defaultMetrics.resetAll();
    });

    it('should provide quick access to block metrics', () => {
      expect(metrics.blocks.committed()).toBe(0);
      
      metrics.blocks.inc();
      expect(metrics.blocks.committed()).toBe(1);
      
      metrics.blocks.inc({ collection: 'users' });
      expect(metrics.blocks.committed()).toBe(2);
    });

    it('should provide quick access to memory metrics', () => {
      expect(metrics.memory.storeOps()).toBe(0);
      expect(metrics.memory.retrieveOps()).toBe(0);
      
      metrics.memory.incStoreOps();
      metrics.memory.incRetrieveOps();
      
      expect(metrics.memory.storeOps()).toBe(1);
      expect(metrics.memory.retrieveOps()).toBe(1);
    });

    it('should provide quick access to cache metrics', () => {
      expect(metrics.cache.hits()).toBe(0);
      expect(metrics.cache.misses()).toBe(0);
      expect(metrics.cache.hitRatio()).toBe(0);
      
      metrics.cache.incHits();
      metrics.cache.incHits();
      metrics.cache.incMisses();
      
      expect(metrics.cache.hits()).toBe(2);
      expect(metrics.cache.misses()).toBe(1);
      expect(metrics.cache.hitRatio()).toBe(2/3);
    });

    it('should provide quick access to network metrics', () => {
      expect(metrics.network.activeConnections()).toBe(0);
      expect(metrics.network.nrnBalance()).toBe(0);
      
      metrics.network.setActiveConnections(10);
      metrics.network.setNRNBalance(1000);
      
      expect(metrics.network.activeConnections()).toBe(10);
      expect(metrics.network.nrnBalance()).toBe(1000);
    });

    it('should provide quick access to query metrics', () => {
      expect(metrics.query.latency.average()).toBe(0);
      
      metrics.query.latency.observe(0.1);
      metrics.query.latency.observe(0.2);
      
      expect(metrics.query.latency.average()).toBe(0.15);
    });

    it('should provide quick access to error metrics', () => {
      expect(metrics.errors.count()).toBe(0);
      
      metrics.errors.inc();
      metrics.errors.inc({ type: 'network' });
      
      expect(metrics.errors.count()).toBe(2);
    });

    it('should provide quick access to index metrics', () => {
      expect(metrics.index.size()).toBe(0);
      
      metrics.index.setSize(1024000);
      
      expect(metrics.index.size()).toBe(1024000);
    });
  });

  describe('labeled quick access metrics', () => {
    beforeEach(() => {
      defaultMetrics.resetAll();
    });

    it('should handle labels in quick access', () => {
      metrics.blocks.inc({ collection: 'users' });
      metrics.blocks.inc({ collection: 'posts' });
      
      expect(metrics.blocks.committed()).toBe(0);
      expect(metrics.blocks.committed()).toBe(0); // Default (no labels)
      
      // Check that underlying metric has labeled values
      const allMetrics = defaultMetrics.getAllMetrics();
      expect(allMetrics.blocksCommitted).toBeDefined();
    });
  });
});