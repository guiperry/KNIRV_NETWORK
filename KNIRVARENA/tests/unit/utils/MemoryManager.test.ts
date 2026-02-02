/**
 * MemoryManager Unit Tests
 * Comprehensive test suite for memory management and leak detection utilities
 */

import MemoryManager, { memoryManager } from '../../../src/utils/MemoryManager';

// Type definitions for test mocks
interface MemoryInfo {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
}

interface PerformanceWithMemory {
  memory?: MemoryInfo;
}

interface WindowWithGC {
  gc?: jest.Mock;
  performance?: PerformanceWithMemory;
}

interface GlobalWithPerformance extends Global {
  performance?: PerformanceWithMemory;
  window?: WindowWithGC;
}

declare const global: GlobalWithPerformance;

// Mock performance.memory API
Object.defineProperty(global, 'performance', {
  value: {
    memory: {
      usedJSHeapSize: 50000000,
      totalJSHeapSize: 100000000,
      jsHeapSizeLimit: 200000000
    }
  }
});

// Mock window.gc
Object.defineProperty(global, 'window', {
  value: {
    gc: jest.fn()
  }
});

describe('MemoryManager', () => {
  let manager: MemoryManager;

  beforeEach(() => {
    // Mock performance.memory for both global and window environments
    const mockMemory = {
      usedJSHeapSize: 50000000,
      totalJSHeapSize: 100000000,
      jsHeapSizeLimit: 200000000
    };

    // Mock for global (Node.js environment)
    Object.defineProperty(global, 'performance', {
      value: { memory: mockMemory },
      writable: true,
      configurable: true
    });

    // Mock for window (browser environment simulation)
    Object.defineProperty(global, 'window', {
      value: {
        performance: { memory: mockMemory },
        gc: jest.fn()
      },
      writable: true,
      configurable: true
    });

    manager = new MemoryManager({
      enableMonitoring: false, // Disable auto-monitoring for tests
      enableAutoCleanup: true,
      monitoringInterval: 100, // Shorter interval for tests
      thresholds: {
        warning: 70,
        critical: 85,
        cleanup: 90
      },
      maxHistorySize: 10
    });
    jest.clearAllMocks();

    // Use fake timers for this specific test file
    jest.useFakeTimers();
  });

  afterEach(() => {
    manager.dispose();
    jest.useRealTimers();

    // Clean up mocks
    delete global.performance;
    delete global.window;
  });

  describe('Initialization', () => {
    it('should initialize with default configuration', () => {
      const defaultManager = new MemoryManager();
      expect(defaultManager).toBeDefined();
      expect(defaultManager.getCurrentMetrics).toBeDefined();
    });

    it('should initialize with custom configuration', () => {
      const customManager = new MemoryManager({
        enableMonitoring: false,
        thresholds: { warning: 60, critical: 80, cleanup: 95 }
      });
      expect(customManager).toBeDefined();
    });
  });

  describe('Memory Metrics Collection', () => {
    it('should get current memory metrics', () => {
      const metrics = manager.getCurrentMetrics();
      
      expect(metrics).toHaveProperty('usedJSHeapSize', 50000000);
      expect(metrics).toHaveProperty('totalJSHeapSize', 100000000);
      expect(metrics).toHaveProperty('jsHeapSizeLimit', 200000000);
      expect(metrics).toHaveProperty('usagePercentage', 25); // 50MB / 200MB = 25%
      expect(metrics).toHaveProperty('timestamp');
    });

    it('should return null when memory API not available', () => {
      const originalPerformance = global.performance;
      const originalWindow = global.window;
      delete global.performance;
      delete global.window;

      const metrics = manager.getCurrentMetrics();
      expect(metrics).toBeNull();

      global.performance = originalPerformance;
      global.window = originalWindow;
    });

    it('should calculate usage percentage correctly', () => {
      // Mock different memory values
      if (global.performance) {
        global.performance.memory = {
          usedJSHeapSize: 100000000,
          totalJSHeapSize: 150000000,
          jsHeapSizeLimit: 200000000
        };
      }

      const metrics = manager.getCurrentMetrics();
      expect(metrics?.usagePercentage).toBe(50); // 100MB / 200MB = 50%
    });
  });

  describe('Memory Monitoring', () => {
    it('should start monitoring', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      manager.startMonitoring();
      
      expect(consoleSpy).toHaveBeenCalledWith('Memory monitoring started');
      consoleSpy.mockRestore();
    });

    it('should stop monitoring', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      manager.startMonitoring();
      manager.stopMonitoring();
      
      expect(consoleSpy).toHaveBeenCalledWith('Memory monitoring stopped');
      consoleSpy.mockRestore();
    });

    it('should not start monitoring if already monitoring', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();

      manager.startMonitoring(); // First call
      expect(consoleSpy).toHaveBeenCalledWith('Memory monitoring started');

      consoleSpy.mockClear(); // Clear the spy to reset call count

      manager.startMonitoring(); // Second call

      expect(consoleSpy).toHaveBeenCalledTimes(0);
      consoleSpy.mockRestore();
    });

    it('should collect metrics at intervals', () => {
      const monitoringManager = new MemoryManager({
        enableMonitoring: true,
        monitoringInterval: 100
      });

      // Fast-forward time
      jest.advanceTimersByTime(300);

      const history = monitoringManager.getMetricsHistory();
      expect(history.length).toBeGreaterThan(0);

      monitoringManager.dispose();
    });
  });

  describe('Memory Cleanup', () => {
    it('should trigger cleanup manually', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
      
      manager.triggerCleanup();
      
      expect(consoleSpy).toHaveBeenCalledWith('Triggering memory cleanup...');
      expect(consoleSpy).toHaveBeenCalledWith('Memory cleanup completed');
      consoleSpy.mockRestore();
    });

    it('should execute cleanup callbacks', () => {
      const callback1 = jest.fn();
      const callback2 = jest.fn();
      
      manager.registerCleanupCallback(callback1);
      manager.registerCleanupCallback(callback2);
      
      manager.triggerCleanup();
      
      expect(callback1).toHaveBeenCalled();
      expect(callback2).toHaveBeenCalled();
    });

    it('should handle cleanup callback errors', () => {
      const errorCallback = jest.fn().mockImplementation(() => {
        throw new Error('Cleanup error');
      });
      const goodCallback = jest.fn();
      
      manager.registerCleanupCallback(errorCallback);
      manager.registerCleanupCallback(goodCallback);
      
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      
      manager.triggerCleanup();
      
      expect(errorCallback).toHaveBeenCalled();
      expect(goodCallback).toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Cleanup callback 0 failed:'),
        expect.any(Error)
      );
      
      consoleSpy.mockRestore();
    });

    it('should force garbage collection when available', () => {
      const gcSpy = global.window?.gc as jest.Mock;
      gcSpy.mockClear();

      manager.triggerCleanup();

      expect(gcSpy).toHaveBeenCalled();
    });

    it('should handle garbage collection errors', () => {
      if (global.window?.gc) {
        const gcSpy = global.window.gc as jest.Mock;
        gcSpy.mockImplementation(() => {
          throw new Error('GC error');
        });
        const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();

        manager.triggerCleanup();

        expect(consoleSpy).toHaveBeenCalledWith(
          'Failed to force garbage collection:',
          expect.any(Error)
        );

        gcSpy.mockRestore();
        consoleSpy.mockRestore();
      }
    });
  });

  describe('Cleanup Callback Management', () => {
    it('should register cleanup callbacks', () => {
      const callback = jest.fn();
      
      manager.registerCleanupCallback(callback);
      manager.triggerCleanup();
      
      expect(callback).toHaveBeenCalled();
    });

    it('should unregister cleanup callbacks', () => {
      const callback = jest.fn();
      
      manager.registerCleanupCallback(callback);
      manager.unregisterCleanupCallback(callback);
      manager.triggerCleanup();
      
      expect(callback).not.toHaveBeenCalled();
    });
  });

  describe('Memory Listeners', () => {
    it('should add memory listeners', () => {
      const listener = jest.fn();

      manager.addMemoryListener(listener);

      // Simulate metric collection
      const monitoringManager = new MemoryManager({
        enableMonitoring: true,
        monitoringInterval: 50 // Short interval for testing
      });
      monitoringManager.addMemoryListener(listener);

      jest.advanceTimersByTime(100);

      expect(listener).toHaveBeenCalled();

      monitoringManager.dispose();
    });

    it('should remove memory listeners', () => {
      const listener = jest.fn();
      
      manager.addMemoryListener(listener);
      manager.removeMemoryListener(listener);
      
      // Listeners should not be called after removal
      expect(listener).not.toHaveBeenCalled();
    });

    it('should handle listener errors gracefully', () => {
      const errorListener = jest.fn().mockImplementation(() => {
        throw new Error('Listener error');
      });

      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();

      const monitoringManager = new MemoryManager({
        enableMonitoring: true,
        monitoringInterval: 50 // Short interval for testing
      });
      monitoringManager.addMemoryListener(errorListener);

      jest.advanceTimersByTime(100);

      expect(consoleSpy).toHaveBeenCalledWith(
        'Memory listener failed:',
        expect.any(Error)
      );

      consoleSpy.mockRestore();
      monitoringManager.dispose();
    });
  });

  describe('Memory Usage Trends', () => {
    it('should detect increasing trend', () => {
      // Simulate increasing memory usage
      const metrics = [
        { usagePercentage: 50, timestamp: Date.now(), usedJSHeapSize: 50000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 55, timestamp: Date.now() + 1000, usedJSHeapSize: 55000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 60, timestamp: Date.now() + 2000, usedJSHeapSize: 60000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 65, timestamp: Date.now() + 3000, usedJSHeapSize: 65000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 }
      ];

      // Mock the metrics history
      jest.spyOn(manager, 'getMetricsHistory').mockReturnValue(metrics);

      const trend = manager.getUsageTrend(4);
      
      expect(trend.trend).toBe('increasing');
      expect(trend.averageChange).toBeGreaterThan(0);
      expect(trend.currentUsage).toBe(65);
    });

    it('should detect decreasing trend', () => {
      const metrics = [
        { usagePercentage: 70, timestamp: Date.now(), usedJSHeapSize: 70000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 65, timestamp: Date.now() + 1000, usedJSHeapSize: 65000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 60, timestamp: Date.now() + 2000, usedJSHeapSize: 60000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 55, timestamp: Date.now() + 3000, usedJSHeapSize: 55000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 }
      ];

      jest.spyOn(manager, 'getMetricsHistory').mockReturnValue(metrics);

      const trend = manager.getUsageTrend(4);
      
      expect(trend.trend).toBe('decreasing');
      expect(trend.averageChange).toBeLessThan(0);
    });

    it('should detect stable trend', () => {
      const metrics = [
        { usagePercentage: 50, timestamp: Date.now(), usedJSHeapSize: 50000000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 50.2, timestamp: Date.now() + 1000, usedJSHeapSize: 50200000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 49.8, timestamp: Date.now() + 2000, usedJSHeapSize: 49800000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 },
        { usagePercentage: 50.1, timestamp: Date.now() + 3000, usedJSHeapSize: 50100000, totalJSHeapSize: 100000000, jsHeapSizeLimit: 200000000 }
      ];

      jest.spyOn(manager, 'getMetricsHistory').mockReturnValue(metrics);

      const trend = manager.getUsageTrend(4);
      
      expect(trend.trend).toBe('stable');
      expect(Math.abs(trend.averageChange)).toBeLessThan(0.5);
    });

    it('should handle insufficient data', () => {
      jest.spyOn(manager, 'getMetricsHistory').mockReturnValue([]);

      const trend = manager.getUsageTrend();
      
      expect(trend.trend).toBe('stable');
      expect(trend.averageChange).toBe(0);
      expect(trend.currentUsage).toBe(0);
    });
  });

  describe('Memory Leak Detection', () => {
    it('should detect potential memory leaks', () => {
      // Simulate consistent memory growth
      const metrics = Array.from({ length: 20 }, (_, i) => ({
        usagePercentage: 30 + i * 2, // Increasing from 30% to 68%
        timestamp: Date.now() + i * 1000,
        usedJSHeapSize: 30000000 + i * 2000000,
        totalJSHeapSize: 100000000,
        jsHeapSizeLimit: 200000000
      }));

      jest.spyOn(manager, 'getMetricsHistory').mockReturnValue(metrics);

      const detection = manager.detectMemoryLeaks();
      
      expect(detection.hasLeak).toBe(true);
      expect(detection.confidence).toBeGreaterThan(0);
      expect(detection.details).toContain('memory growth');
    });

    it('should not detect leaks with stable memory', () => {
      const metrics = Array.from({ length: 20 }, (_, i) => ({
        usagePercentage: 50 + (Math.random() - 0.5) * 2, // Stable around 50%
        timestamp: Date.now() + i * 1000,
        usedJSHeapSize: 50000000 + (Math.random() - 0.5) * 2000000,
        totalJSHeapSize: 100000000,
        jsHeapSizeLimit: 200000000
      }));

      jest.spyOn(manager, 'getMetricsHistory').mockReturnValue(metrics);

      const detection = manager.detectMemoryLeaks();
      
      expect(detection.hasLeak).toBe(false);
      expect(detection.details).toContain('No significant memory leak patterns');
    });

    it('should handle insufficient data for leak detection', () => {
      const metrics = Array.from({ length: 5 }, (_, i) => ({
        usagePercentage: 50,
        timestamp: Date.now() + i * 1000,
        usedJSHeapSize: 50000000,
        totalJSHeapSize: 100000000,
        jsHeapSizeLimit: 200000000
      }));

      jest.spyOn(manager, 'getMetricsHistory').mockReturnValue(metrics);

      const detection = manager.detectMemoryLeaks();
      
      expect(detection.hasLeak).toBe(false);
      expect(detection.confidence).toBe(0);
      expect(detection.details).toContain('Insufficient data');
    });
  });

  describe('Memory Reports', () => {
    it('should generate comprehensive memory report', () => {
      const report = manager.generateReport();
      
      expect(report).toHaveProperty('current');
      expect(report).toHaveProperty('trend');
      expect(report).toHaveProperty('leakDetection');
      expect(report).toHaveProperty('recommendations');
      
      expect(Array.isArray(report.recommendations)).toBe(true);
    });

    it('should provide recommendations based on memory state', () => {
      // Mock high memory usage
      jest.spyOn(manager, 'getCurrentMetrics').mockReturnValue({
        usedJSHeapSize: 170000000,
        totalJSHeapSize: 180000000,
        jsHeapSizeLimit: 200000000,
        usagePercentage: 85,
        timestamp: Date.now()
      });

      const report = manager.generateReport();
      
      expect(report.recommendations.some(rec => 
        rec.includes('Memory usage is high')
      )).toBe(true);
    });

    it('should recommend monitoring for increasing trends', () => {
      jest.spyOn(manager, 'getUsageTrend').mockReturnValue({
        trend: 'increasing',
        averageChange: 2.5,
        currentUsage: 60
      });

      const report = manager.generateReport();
      
      expect(report.recommendations.some(rec => 
        rec.includes('trending upward')
      )).toBe(true);
    });
  });

  describe('History Management', () => {
    it('should get metrics history', () => {
      const history = manager.getMetricsHistory();
      expect(Array.isArray(history)).toBe(true);
    });

    it('should clear metrics history', () => {
      manager.clearHistory();
      const history = manager.getMetricsHistory();
      expect(history).toHaveLength(0);
    });

    it('should limit history size', () => {
      const limitedManager = new MemoryManager({
        enableMonitoring: true,
        monitoringInterval: 10,
        maxHistorySize: 3
      });

      // Let it collect more than 3 metrics
      jest.advanceTimersByTime(100);

      const history = limitedManager.getMetricsHistory();
      expect(history.length).toBeLessThanOrEqual(3);

      limitedManager.dispose();
    });
  });

  describe('Threshold Monitoring', () => {
    it('should trigger cleanup on critical threshold', () => {
      // Mock high memory usage
      if (global.performance) {
        global.performance.memory = {
          usedJSHeapSize: 180000000,
          totalJSHeapSize: 190000000,
          jsHeapSizeLimit: 200000000 // 90% usage
        };
      }

      const monitoringManager = new MemoryManager({
        enableMonitoring: true,
        enableAutoCleanup: true,
        monitoringInterval: 50, // Short interval for testing
        thresholds: { warning: 70, critical: 85, cleanup: 90 }
      });

      const cleanupSpy = jest.spyOn(monitoringManager, 'triggerCleanup');

      jest.advanceTimersByTime(100);

      expect(cleanupSpy).toHaveBeenCalled();

      cleanupSpy.mockRestore();
      monitoringManager.dispose();
    });
  });

  describe('Disposal', () => {
    it('should cleanup resources on disposal', () => {
      manager.startMonitoring();
      
      const stopSpy = jest.spyOn(manager, 'stopMonitoring');
      const clearSpy = jest.spyOn(manager, 'clearHistory');
      
      manager.dispose();
      
      expect(stopSpy).toHaveBeenCalled();
      expect(clearSpy).toHaveBeenCalled();
    });
  });

  describe('Singleton Instance', () => {
    it('should provide a singleton instance', () => {
      expect(memoryManager).toBeDefined();
      expect(memoryManager).toBeInstanceOf(MemoryManager);
    });
  });
});
