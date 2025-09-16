/**
 * Memory Manager
 * Advanced memory management and leak detection for KNIRV Controller
 */

interface MemoryMetrics {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
  usagePercentage: number;
  timestamp: number;
}

interface MemoryThresholds {
  warning: number; // Percentage
  critical: number; // Percentage
  cleanup: number; // Percentage
}

interface MemoryManagerConfig {
  enableMonitoring: boolean;
  enableAutoCleanup: boolean;
  monitoringInterval: number;
  thresholds: MemoryThresholds;
  maxHistorySize: number;
}

interface PerformanceMemory {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
}

interface GlobalWithPerformance {
  performance?: {
    memory?: PerformanceMemory;
  };
}

interface GlobalWithGC {
  gc?: () => void;
}

interface WindowWithPerformanceMemory extends Window {
  performance: Performance & {
    memory?: PerformanceMemory;
  };
}

class MemoryManager {
  private config: MemoryManagerConfig;
  private metrics: MemoryMetrics[] = [];
  private monitoringInterval: NodeJS.Timeout | null = null;
  private cleanupCallbacks: (() => void)[] = [];
  private memoryListeners: ((metrics: MemoryMetrics) => void)[] = [];
  private isMonitoring: boolean = false;

  constructor(config: Partial<MemoryManagerConfig> = {}) {
    this.config = {
      enableMonitoring: true,
      enableAutoCleanup: true,
      monitoringInterval: 5000, // 5 seconds
      thresholds: {
        warning: 70,
        critical: 85,
        cleanup: 90
      },
      maxHistorySize: 100,
      ...config
    };

    if (this.config.enableMonitoring) {
      this.startMonitoring();
    }
  }

  /**
   * Start memory monitoring
   */
  public startMonitoring(): void {
    if (this.isMonitoring) {
      return;
    }

    this.isMonitoring = true;
    this.monitoringInterval = setInterval(() => {
      this.collectMetrics();
    }, this.config.monitoringInterval);

    console.log('Memory monitoring started');
  }

  /**
   * Stop memory monitoring
   */
  public stopMonitoring(): void {
    if (this.monitoringInterval) {
      clearInterval(this.monitoringInterval);
      this.monitoringInterval = null;
    }
    this.isMonitoring = false;
    console.log('Memory monitoring stopped');
  }

  /**
   * Collect current memory metrics
   */
  private collectMetrics(): void {
    const currentMetrics = this.getCurrentMetrics();
    if (!currentMetrics) {
      return;
    }

    // Add to metrics history
    this.metrics.push(currentMetrics);

    // Limit history size
    if (this.metrics.length > this.config.maxHistorySize) {
      this.metrics.shift();
    }

    // Check thresholds and trigger actions
    this.checkThresholds(currentMetrics);

    // Notify listeners
    this.notifyListeners(currentMetrics);
  }

  /**
   * Check memory thresholds and trigger appropriate actions
   */
  private checkThresholds(metrics: MemoryMetrics): void {
    const usage = metrics.usagePercentage;

    if (usage >= this.config.thresholds.cleanup && this.config.enableAutoCleanup) {
      console.warn(`Memory usage critical (${usage.toFixed(1)}%). Triggering cleanup.`);
      this.triggerCleanup();
    } else if (usage >= this.config.thresholds.critical) {
      console.warn(`Memory usage critical: ${usage.toFixed(1)}%`);
    } else if (usage >= this.config.thresholds.warning) {
      console.warn(`Memory usage warning: ${usage.toFixed(1)}%`);
    }
  }

  /**
   * Trigger memory cleanup
   */
  public triggerCleanup(): void {
    console.log('Triggering memory cleanup...');

    // Execute all registered cleanup callbacks
    this.cleanupCallbacks.forEach((callback, index) => {
      try {
        callback();
      } catch (error) {
        console.error(`Cleanup callback ${index} failed:`, error);
      }
    });

    // Force garbage collection if available
    this.forceGarbageCollection();

    console.log('Memory cleanup completed');
  }

  /**
   * Force garbage collection (if available)
   */
  private forceGarbageCollection(): void {
    let gcCalled = false;

    // Check for browser environment
    if (typeof window !== 'undefined' && 'gc' in window) {
      try {
        (window as Window & { gc?: () => void }).gc?.();
        console.log('Forced garbage collection');
        gcCalled = true;
      } catch (error) {
        console.warn('Failed to force garbage collection:', error);
      }
    }
    // Check for global.window.gc (test environment)
    else if (typeof global !== 'undefined' && (global as any).window && 'gc' in (global as any).window) {
      try {
        (global as any).window.gc();
        console.log('Forced garbage collection');
        gcCalled = true;
      } catch (error) {
        console.warn('Failed to force garbage collection:', error);
      }
    }
    // Check for global gc (Node.js environment)
    else if (typeof global !== 'undefined' && 'gc' in global && (global as GlobalWithGC).gc) {
      try {
        (global as GlobalWithGC).gc!();
        console.log('Forced garbage collection');
        gcCalled = true;
      } catch (error) {
        console.warn('Failed to force garbage collection:', error);
      }
    }

    // If no gc was available or called, log a message
    if (!gcCalled) {
      console.log('Garbage collection not available in this environment');
    }
  }

  /**
   * Register a cleanup callback
   */
  public registerCleanupCallback(callback: () => void): void {
    this.cleanupCallbacks.push(callback);
  }

  /**
   * Unregister a cleanup callback
   */
  public unregisterCleanupCallback(callback: () => void): void {
    const index = this.cleanupCallbacks.indexOf(callback);
    if (index > -1) {
      this.cleanupCallbacks.splice(index, 1);
    }
  }

  /**
   * Add memory listener
   */
  public addMemoryListener(listener: (metrics: MemoryMetrics) => void): void {
    this.memoryListeners.push(listener);
  }

  /**
   * Remove memory listener
   */
  public removeMemoryListener(listener: (metrics: MemoryMetrics) => void): void {
    const index = this.memoryListeners.indexOf(listener);
    if (index > -1) {
      this.memoryListeners.splice(index, 1);
    }
  }

  /**
   * Notify all memory listeners
   */
  private notifyListeners(metrics: MemoryMetrics): void {
    this.memoryListeners.forEach(listener => {
      try {
        listener(metrics);
      } catch (error) {
        console.error('Memory listener failed:', error);
      }
    });
  }

  /**
   * Get current memory metrics
   */
  public getCurrentMetrics(): MemoryMetrics | null {
    let memory: PerformanceMemory | null = null;

    // Check for global performance mock (test environment) - prioritize this for tests
    if (typeof global !== 'undefined') {
      const globalWithPerf = global as GlobalWithPerformance;
      if (globalWithPerf.performance && globalWithPerf.performance.memory) {
        memory = globalWithPerf.performance.memory;
      }
    }

    // If not found in global, check for browser environment
    if (!memory && typeof window !== 'undefined' && window.performance && (window as WindowWithPerformanceMemory).performance.memory) {
      memory = (window as WindowWithPerformanceMemory).performance.memory;
    }

    // If still not found, check for standard performance API
    if (!memory && typeof performance !== 'undefined') {
      const perfWithMemory = performance as Performance & { memory?: PerformanceMemory };
      if (perfWithMemory.memory) {
        memory = perfWithMemory.memory;
      }
    }

    if (memory) {
      return {
        usedJSHeapSize: memory.usedJSHeapSize,
        totalJSHeapSize: memory.totalJSHeapSize,
        jsHeapSizeLimit: memory.jsHeapSizeLimit,
        usagePercentage: (memory.usedJSHeapSize / memory.jsHeapSizeLimit) * 100,
        timestamp: Date.now()
      };
    }

    return null;
  }

  /**
   * Get memory metrics history
   */
  public getMetricsHistory(): MemoryMetrics[] {
    return [...this.metrics];
  }

  /**
   * Get memory usage trend
   */
  public getUsageTrend(samples: number = 10): {
    trend: 'increasing' | 'decreasing' | 'stable';
    averageChange: number;
    currentUsage: number;
  } {
    const metricsHistory = this.getMetricsHistory();

    if (metricsHistory.length < 2) {
      return {
        trend: 'stable',
        averageChange: 0,
        currentUsage: metricsHistory[0]?.usagePercentage || 0
      };
    }

    const recentMetrics = metricsHistory.slice(-samples);
    const changes: number[] = [];

    for (let i = 1; i < recentMetrics.length; i++) {
      const change = recentMetrics[i].usagePercentage - recentMetrics[i - 1].usagePercentage;
      changes.push(change);
    }

    const averageChange = changes.reduce((sum, change) => sum + change, 0) / changes.length;
    const currentUsage = recentMetrics[recentMetrics.length - 1].usagePercentage;

    let trend: 'increasing' | 'decreasing' | 'stable';
    if (Math.abs(averageChange) < 0.1) {
      trend = 'stable';
    } else if (averageChange > 0) {
      trend = 'increasing';
    } else {
      trend = 'decreasing';
    }

    return {
      trend,
      averageChange,
      currentUsage
    };
  }

  /**
   * Detect potential memory leaks
   */
  public detectMemoryLeaks(): {
    hasLeak: boolean;
    confidence: number;
    details: string;
  } {
    const metricsHistory = this.getMetricsHistory();

    if (metricsHistory.length < 10) {
      return {
        hasLeak: false,
        confidence: 0,
        details: 'Insufficient data for leak detection'
      };
    }

    const trend = this.getUsageTrend(20);
    const recentMetrics = metricsHistory.slice(-20);
    
    // Check for consistent upward trend
    const consistentIncrease = recentMetrics.every((metric, index) => {
      if (index === 0) return true;
      return metric.usagePercentage >= recentMetrics[index - 1].usagePercentage - 1; // Allow small fluctuations
    });

    // Check for significant memory growth
    const startUsage = recentMetrics[0].usagePercentage;
    const endUsage = recentMetrics[recentMetrics.length - 1].usagePercentage;
    const totalGrowth = endUsage - startUsage;

    let hasLeak = false;
    let confidence = 0;
    let details = '';

    if (consistentIncrease && totalGrowth > 10) {
      hasLeak = true;
      confidence = Math.min(90, totalGrowth * 2);
      details = `Consistent memory growth of ${totalGrowth.toFixed(1)}% detected`;
    } else if (trend.trend === 'increasing' && trend.averageChange > 0.5) {
      hasLeak = true;
      confidence = Math.min(70, trend.averageChange * 20);
      details = `Increasing memory trend with average change of ${trend.averageChange.toFixed(2)}%`;
    } else {
      details = 'No significant memory leak patterns detected';
    }

    return {
      hasLeak,
      confidence,
      details
    };
  }

  /**
   * Generate memory report
   */
  public generateReport(): {
    current: MemoryMetrics | null;
    trend: ReturnType<MemoryManager['getUsageTrend']>;
    leakDetection: ReturnType<MemoryManager['detectMemoryLeaks']>;
    recommendations: string[];
  } {
    const current = this.getCurrentMetrics();
    const trend = this.getUsageTrend();
    const leakDetection = this.detectMemoryLeaks();
    const recommendations: string[] = [];

    // Generate recommendations
    if (current && current.usagePercentage > this.config.thresholds.warning) {
      recommendations.push('Memory usage is high. Consider implementing cleanup strategies.');
    }

    if (trend.trend === 'increasing') {
      recommendations.push('Memory usage is trending upward. Monitor for potential leaks.');
    }

    if (leakDetection.hasLeak) {
      recommendations.push(`Potential memory leak detected (${leakDetection.confidence}% confidence). ${leakDetection.details}`);
    }

    if (this.cleanupCallbacks.length === 0) {
      recommendations.push('No cleanup callbacks registered. Consider adding memory cleanup strategies.');
    }

    return {
      current,
      trend,
      leakDetection,
      recommendations
    };
  }

  /**
   * Clear metrics history
   */
  public clearHistory(): void {
    this.metrics = [];
  }

  /**
   * Dispose and cleanup
   */
  public dispose(): void {
    this.stopMonitoring();
    this.cleanupCallbacks = [];
    this.memoryListeners = [];
    this.clearHistory();
  }
}

// Singleton instance
export const memoryManager = new MemoryManager();
export default MemoryManager;
