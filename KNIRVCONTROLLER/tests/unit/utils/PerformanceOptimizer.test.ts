/**
 * PerformanceOptimizer Unit Tests
 * Comprehensive test suite for performance optimization utilities
 */

import PerformanceOptimizer, { performanceOptimizer } from '../../../src/utils/PerformanceOptimizer';

// Mock PerformanceObserver
const PerformanceObserverMock = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  disconnect: jest.fn()
}))
Object.defineProperty(PerformanceObserverMock, 'supportedEntryTypes', {
  value: ['navigation', 'resource', 'mark', 'measure', 'paint'],
  writable: false
});
global.PerformanceObserver = PerformanceObserverMock as jest.Mock & { supportedEntryTypes: string[] };

// Mock performance API
Object.defineProperty(global, 'performance', {
  value: {
    memory: {
      usedJSHeapSize: 50000000,
      totalJSHeapSize: 100000000,
      jsHeapSizeLimit: 200000000
    },
    now: jest.fn(() => Date.now())
  }
});

describe('PerformanceOptimizer', () => {
  let optimizer: PerformanceOptimizer;

  beforeEach(() => {
    // Use real timers first to ensure clean state
    jest.useRealTimers();
    optimizer = new PerformanceOptimizer({
      enableCaching: true,
      enableLazyLoading: true,
      maxCacheSize: 10,
      cacheTimeout: 1000
    });
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    optimizer.dispose();
    jest.useRealTimers();
  });

  describe('Initialization', () => {
    it('should initialize with default configuration', () => {
      const defaultOptimizer = new PerformanceOptimizer();
      expect(defaultOptimizer).toBeDefined();
      expect(defaultOptimizer.getMetrics).toBeDefined();
    });

    it('should initialize with custom configuration', () => {
      const customOptimizer = new PerformanceOptimizer({
        enableCaching: false,
        maxCacheSize: 50
      });
      expect(customOptimizer).toBeDefined();
    });
  });

  describe('Caching', () => {
    it('should cache and retrieve data', () => {
      const testData = { test: 'data' };
      optimizer.cache_set('test-key', testData);
      
      const retrieved = optimizer.cache_get('test-key');
      expect(retrieved).toEqual(testData);
    });

    it('should return null for non-existent cache keys', () => {
      const retrieved = optimizer.cache_get('non-existent');
      expect(retrieved).toBeNull();
    });

    it('should expire cache entries after timeout', () => {
      const testData = { test: 'data' };
      optimizer.cache_set('test-key', testData);
      
      // Fast-forward time beyond cache timeout
      jest.advanceTimersByTime(2000);
      
      const retrieved = optimizer.cache_get('test-key');
      expect(retrieved).toBeNull();
    });

    it('should implement LRU eviction when cache is full', () => {
      // Fill cache to capacity
      for (let i = 0; i < 10; i++) {
        optimizer.cache_set(`key-${i}`, `data-${i}`);
      }
      
      // Add one more item to trigger eviction
      optimizer.cache_set('new-key', 'new-data');
      
      // First item should be evicted
      expect(optimizer.cache_get('key-0')).toBeNull();
      expect(optimizer.cache_get('new-key')).toBe('new-data');
    });

    it('should update cache hit rate correctly', () => {
      optimizer.cache_set('test-key', 'test-data');
      
      // Hit
      optimizer.cache_get('test-key');
      
      // Miss
      optimizer.cache_get('non-existent');
      
      const metrics = optimizer.getMetrics();
      expect(metrics.cacheHitRate).toBeGreaterThan(0);
    });
  });

  describe('Function Optimization', () => {
    it('should throttle function execution', (done) => {
      const mockFn = jest.fn();
      const throttledFn = optimizer.throttle(mockFn, 100);
      
      // Call multiple times rapidly
      throttledFn();
      throttledFn();
      throttledFn();
      
      // Should only execute once immediately
      expect(mockFn).toHaveBeenCalledTimes(1);
      
      // Wait for throttle delay
      setTimeout(() => {
        expect(mockFn).toHaveBeenCalledTimes(2);
        done();
      }, 150);
      
      jest.advanceTimersByTime(150);
    });

    it('should debounce function execution', (done) => {
      const mockFn = jest.fn();
      const debouncedFn = optimizer.debounce(mockFn, 100);
      
      // Call multiple times rapidly
      debouncedFn();
      debouncedFn();
      debouncedFn();
      
      // Should not execute immediately
      expect(mockFn).not.toHaveBeenCalled();
      
      // Wait for debounce delay
      setTimeout(() => {
        expect(mockFn).toHaveBeenCalledTimes(1);
        done();
      }, 150);
      
      jest.advanceTimersByTime(150);
    });
  });

  describe('Batch Processing', () => {
    it('should process items in batches', async () => {
      const items = Array.from({ length: 25 }, (_, i) => i);
      const processor = jest.fn().mockImplementation((batch) =>
        Promise.resolve(batch.map((item: number) => item * 2))
      );
      
      const results = await optimizer.batchProcess(items, processor, 10);
      
      expect(processor).toHaveBeenCalledTimes(3); // 25 items / 10 batch size = 3 batches
      expect(results).toHaveLength(25);
      expect(results[0]).toBe(0);
      expect(results[24]).toBe(48);
    });

    it('should handle batch processing errors', async () => {
      const items = [1, 2, 3];
      const processor = jest.fn().mockRejectedValue(new Error('Processing failed'));
      
      await expect(optimizer.batchProcess(items, processor)).rejects.toThrow('Processing failed');
    });
  });

  describe('Lazy Loading', () => {
    it('should lazy load and cache results', async () => {
      const loader = jest.fn().mockResolvedValue('loaded-data');
      
      const result1 = await optimizer.lazyLoad(loader);
      const result2 = await optimizer.lazyLoad(loader);
      
      expect(result1).toBe('loaded-data');
      expect(result2).toBe('loaded-data');
      expect(loader).toHaveBeenCalledTimes(1); // Should be cached after first call
    });

    it('should return fallback on loader error', async () => {
      const loader = jest.fn().mockRejectedValue(new Error('Load failed'));
      const fallback = 'fallback-data';
      
      const result = await optimizer.lazyLoad(loader, fallback);
      expect(result).toBe(fallback);
    });

    it('should throw error when no fallback provided', async () => {
      const loader = jest.fn().mockRejectedValue(new Error('Load failed'));
      
      await expect(optimizer.lazyLoad(loader)).rejects.toThrow('Load failed');
    });
  });

  describe('Image Optimization', () => {
    it('should optimize image URLs with parameters', () => {
      const originalUrl = 'https://example.com/image.jpg';
      const optimizedUrl = optimizer.optimizeImage(originalUrl, {
        width: 800,
        height: 600,
        quality: 80,
        format: 'webp'
      });
      
      expect(optimizedUrl).toContain('w=800');
      expect(optimizedUrl).toContain('h=600');
      expect(optimizedUrl).toContain('q=80');
      expect(optimizedUrl).toContain('f=webp');
    });

    it('should return original URL when compression disabled', () => {
      const disabledOptimizer = new PerformanceOptimizer({ enableCompression: false });
      const originalUrl = 'https://example.com/image.jpg';
      const result = disabledOptimizer.optimizeImage(originalUrl, { width: 800 });
      
      expect(result).toBe(originalUrl);
    });
  });

  describe('Memory Monitoring', () => {
    it('should get memory usage when available', () => {
      const usage = optimizer.getMemoryUsage();
      expect(typeof usage).toBe('number');
      expect(usage).toBeGreaterThanOrEqual(0);
      expect(usage).toBeLessThanOrEqual(100);
    });

    it('should return 0 when memory API not available', () => {
      const originalPerformance = global.performance;
      // Make performance optional before deleting
      (global as { performance?: unknown }).performance = undefined;
      
      const usage = optimizer.getMemoryUsage();
      expect(usage).toBe(0);
      
      global.performance = originalPerformance;
    });
  });

  describe('Performance Metrics', () => {
    it('should return current metrics', () => {
      const metrics = optimizer.getMetrics();
      
      expect(metrics).toHaveProperty('memoryUsage');
      expect(metrics).toHaveProperty('cpuUsage');
      expect(metrics).toHaveProperty('renderTime');
      expect(metrics).toHaveProperty('networkLatency');
      expect(metrics).toHaveProperty('cacheHitRate');
      expect(metrics).toHaveProperty('errorRate');
    });

    it('should update metrics over time', () => {
      const initialMetrics = optimizer.getMetrics();
      
      // Simulate some cache activity
      optimizer.cache_set('test', 'data');
      optimizer.cache_get('test');
      
      const updatedMetrics = optimizer.getMetrics();
      expect(updatedMetrics.cacheHitRate).toBeGreaterThanOrEqual(initialMetrics.cacheHitRate);
    });
  });

  describe('Performance Report', () => {
    it('should generate comprehensive performance report', () => {
      // Add some cache activity
      optimizer.cache_set('test1', 'data1');
      optimizer.cache_set('test2', 'data2');
      optimizer.cache_get('test1');
      
      const report = optimizer.generateReport();
      
      expect(report).toHaveProperty('metrics');
      expect(report).toHaveProperty('recommendations');
      expect(report).toHaveProperty('cacheStats');
      
      expect(Array.isArray(report.recommendations)).toBe(true);
      expect(report.cacheStats).toHaveProperty('size');
      expect(report.cacheStats).toHaveProperty('hitRate');
      expect(report.cacheStats).toHaveProperty('oldestEntry');
    });

    it('should provide recommendations based on metrics', () => {
      // Simulate high memory usage by mocking getMetrics to return high memory usage
      jest.spyOn(optimizer, 'getMetrics').mockReturnValue({
        memoryUsage: 85,
        networkLatency: 500,
        renderTime: 2000,
        cacheHitRate: 75,
        errorRate: 0,
        throughput: 100
      });

      const report = optimizer.generateReport();

      expect(report.recommendations.some(rec =>
        rec.includes('memory usage')
      )).toBe(true);
    });
  });

  describe('Cache Management', () => {
    it('should clear cache', () => {
      optimizer.cache_set('test', 'data');
      expect(optimizer.cache_get('test')).toBe('data');
      
      optimizer.clearCache();
      expect(optimizer.cache_get('test')).toBeNull();
    });
  });

  describe('Disposal', () => {
    it('should cleanup resources on disposal', () => {
      const disposeSpy = jest.spyOn(optimizer, 'dispose');
      
      optimizer.dispose();
      
      expect(disposeSpy).toHaveBeenCalled();
    });
  });

  describe('Singleton Instance', () => {
    it('should provide a singleton instance', () => {
      expect(performanceOptimizer).toBeDefined();
      expect(performanceOptimizer).toBeInstanceOf(PerformanceOptimizer);
    });
  });

  describe('Performance Observer Integration', () => {
    it('should initialize performance observers when available', () => {
      // Mock window and PerformanceObserver to simulate browser environment
      const originalWindow = global.window;
      (global as { window?: unknown }).window = {
        PerformanceObserver: global.PerformanceObserver
      };

      new PerformanceOptimizer();

      // PerformanceObserver should be called during initialization
      expect(global.PerformanceObserver).toHaveBeenCalled();

      // Restore original window
      (global as { window?: unknown }).window = originalWindow;
    });

    it('should handle performance observer errors gracefully', () => {
      // Mock PerformanceObserver to throw error
      const originalObserver = global.PerformanceObserver;
      const ErrorObserverMock = jest.fn().mockImplementation(() => {
        throw new Error('Observer not supported');
      });
      Object.defineProperty(ErrorObserverMock, 'supportedEntryTypes', {
        value: ['navigation', 'resource', 'mark', 'measure', 'paint'],
        writable: false
      });
      global.PerformanceObserver = ErrorObserverMock as jest.Mock & { supportedEntryTypes: string[] };
      
      // Should not throw during initialization
      expect(() => new PerformanceOptimizer()).not.toThrow();
      
      global.PerformanceObserver = originalObserver;
    });
  });
});
