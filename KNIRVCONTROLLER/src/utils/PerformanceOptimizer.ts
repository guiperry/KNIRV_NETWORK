/**
 * Performance Optimizer
 * Advanced performance optimization utilities for KNIRV Controller
 */

interface PerformanceMetrics {
  memoryUsage: number;
  cpuUsage: number;
  renderTime: number;
  networkLatency: number;
  cacheHitRate: number;
  errorRate: number;
}

interface OptimizationConfig {
  enableCaching: boolean;
  enableLazyLoading: boolean;
  enableVirtualization: boolean;
  enableCompression: boolean;
  maxCacheSize: number;
  cacheTimeout: number;
  batchSize: number;
  throttleDelay: number;
}

class PerformanceOptimizer {
  private metrics: PerformanceMetrics;
  private config: OptimizationConfig;
  private cache: Map<string, { data: unknown; timestamp: number; hits: number }>;
  private observers: PerformanceObserver[];
  private isMonitoring: boolean;

  constructor(config: Partial<OptimizationConfig> = {}) {
    this.config = {
      enableCaching: true,
      enableLazyLoading: true,
      enableVirtualization: true,
      enableCompression: true,
      maxCacheSize: 100,
      cacheTimeout: 300000, // 5 minutes
      batchSize: 50,
      throttleDelay: 100,
      ...config
    };

    this.metrics = {
      memoryUsage: 0,
      cpuUsage: 0,
      renderTime: 0,
      networkLatency: 0,
      cacheHitRate: 0,
      errorRate: 0
    };

    this.cache = new Map();
    this.observers = [];
    this.isMonitoring = false;

    this.initializePerformanceMonitoring();
  }

  /**
   * Initialize performance monitoring
   */
  private initializePerformanceMonitoring(): void {
    if (typeof window !== 'undefined' && 'PerformanceObserver' in window) {
      // Monitor navigation timing
      const navObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (entry.entryType === 'navigation') {
            this.updateNavigationMetrics(entry as PerformanceNavigationTiming);
          }
        }
      });

      // Monitor resource timing
      const resourceObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (entry.entryType === 'resource') {
            this.updateResourceMetrics(entry as PerformanceResourceTiming);
          }
        }
      });

      // Monitor paint timing
      const paintObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (entry.entryType === 'paint') {
            this.updatePaintMetrics(entry);
          }
        }
      });

      try {
        navObserver.observe({ entryTypes: ['navigation'] });
        resourceObserver.observe({ entryTypes: ['resource'] });
        paintObserver.observe({ entryTypes: ['paint'] });

        this.observers.push(navObserver, resourceObserver, paintObserver);
        this.isMonitoring = true;
      } catch (error) {
        console.warn('Performance monitoring not supported:', error);
      }
    }
  }

  /**
   * Update navigation metrics
   */
  private updateNavigationMetrics(entry: PerformanceNavigationTiming): void {
    this.metrics.networkLatency = entry.responseStart - entry.requestStart;
    this.metrics.renderTime = entry.loadEventEnd - entry.fetchStart;
  }

  /**
   * Update resource metrics
   */
  private updateResourceMetrics(entry: PerformanceResourceTiming): void {
    const latency = entry.responseStart - entry.requestStart;
    this.metrics.networkLatency = (this.metrics.networkLatency + latency) / 2;
  }

  /**
   * Update paint metrics
   */
  private updatePaintMetrics(entry: PerformanceEntry): void {
    if (entry.name === 'first-contentful-paint') {
      this.metrics.renderTime = entry.startTime;
    }
  }

  /**
   * Optimized caching with LRU eviction
   */
  public cache_get<T>(key: string): T | null {
    const cached = this.cache.get(key);
    
    if (!cached) {
      return null;
    }

    // Check if cache entry has expired
    if (Date.now() - cached.timestamp > this.config.cacheTimeout) {
      this.cache.delete(key);
      return null;
    }

    // Update hit count and move to end (LRU)
    cached.hits++;
    this.cache.delete(key);
    this.cache.set(key, cached);

    this.updateCacheHitRate();
    return cached.data as T;
  }

  /**
   * Cache data with automatic cleanup
   */
  public cache_set<T>(key: string, data: T): void {
    if (!this.config.enableCaching) {
      return;
    }

    // Implement LRU eviction if cache is full
    if (this.cache.size >= this.config.maxCacheSize) {
      const firstKey = this.cache.keys().next().value;
      this.cache.delete(firstKey!);
    }

    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      hits: 0
    });
  }

  /**
   * Update cache hit rate metric
   */
  private updateCacheHitRate(): void {
    const totalRequests = Array.from(this.cache.values()).reduce(
      (sum, entry) => sum + entry.hits + 1, 0
    );
    const hits = Array.from(this.cache.values()).reduce(
      (sum, entry) => sum + entry.hits, 0
    );
    
    this.metrics.cacheHitRate = totalRequests > 0 ? (hits / totalRequests) * 100 : 0;
  }

  /**
   * Throttle function execution
   */
  public throttle<T extends (...args: unknown[]) => unknown>(
    func: T,
    delay: number = this.config.throttleDelay
  ): (...args: Parameters<T>) => void {
    let timeoutId: NodeJS.Timeout | null = null;
    let lastExecTime = 0;

    return (...args: Parameters<T>) => {
      const currentTime = Date.now();

      if (currentTime - lastExecTime > delay) {
        func(...args);
        lastExecTime = currentTime;
      } else {
        if (timeoutId) {
          clearTimeout(timeoutId);
        }
        
        timeoutId = setTimeout(() => {
          func(...args);
          lastExecTime = Date.now();
        }, delay - (currentTime - lastExecTime));
      }
    };
  }

  /**
   * Debounce function execution
   */
  public debounce<T extends (...args: unknown[]) => unknown>(
    func: T,
    delay: number = this.config.throttleDelay
  ): (...args: Parameters<T>) => void {
    let timeoutId: NodeJS.Timeout | null = null;

    return (...args: Parameters<T>) => {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }

      timeoutId = setTimeout(() => {
        func(...args);
      }, delay);
    };
  }

  /**
   * Batch process operations
   */
  public batchProcess<T, R>(
    items: T[],
    processor: (batch: T[]) => Promise<R[]>,
    batchSize: number = this.config.batchSize
  ): Promise<R[]> {
    const batches: T[][] = [];
    
    for (let i = 0; i < items.length; i += batchSize) {
      batches.push(items.slice(i, i + batchSize));
    }

    return Promise.all(
      batches.map(batch => processor(batch))
    ).then(results => results.flat());
  }

  /**
   * Lazy load component
   */
  public lazyLoad<T>(
    loader: () => Promise<T>,
    fallback?: T
  ): Promise<T> {
    if (!this.config.enableLazyLoading) {
      return loader();
    }

    const cacheKey = `lazy_${loader.toString().slice(0, 50)}`;
    const cached = this.cache_get<T>(cacheKey);
    
    if (cached) {
      return Promise.resolve(cached);
    }

    return loader().then(result => {
      this.cache_set(cacheKey, result);
      return result;
    }).catch(error => {
      if (fallback !== undefined) {
        return fallback;
      }
      throw error;
    });
  }

  /**
   * Optimize images for better performance
   */
  public optimizeImage(
    imageUrl: string,
    options: {
      width?: number;
      height?: number;
      quality?: number;
      format?: 'webp' | 'jpeg' | 'png';
    } = {}
  ): string {
    if (!this.config.enableCompression) {
      return imageUrl;
    }

    const params = new URLSearchParams();
    
    if (options.width) params.set('w', options.width.toString());
    if (options.height) params.set('h', options.height.toString());
    if (options.quality) params.set('q', options.quality.toString());
    if (options.format) params.set('f', options.format);

    return `${imageUrl}?${params.toString()}`;
  }

  /**
   * Memory usage monitoring
   */
  public getMemoryUsage(): number {
    if (typeof window !== 'undefined' && 'performance' in window && typeof performance !== 'undefined' && 'memory' in performance) {
      const perfWithMemory = performance as Performance & {
        memory?: {
          usedJSHeapSize: number;
          totalJSHeapSize: number;
        }
      };
      if (perfWithMemory.memory) {
        this.metrics.memoryUsage = (perfWithMemory.memory.usedJSHeapSize / perfWithMemory.memory.totalJSHeapSize) * 100;
        return this.metrics.memoryUsage;
      }
    }
    return 0;
  }

  /**
   * Get current performance metrics
   */
  public getMetrics(): PerformanceMetrics {
    this.getMemoryUsage();
    return { ...this.metrics };
  }

  /**
   * Generate performance report
   */
  public generateReport(): {
    metrics: PerformanceMetrics;
    recommendations: string[];
    cacheStats: {
      size: number;
      hitRate: number;
      oldestEntry: number;
    };
  } {
    const metrics = this.getMetrics();
    const recommendations: string[] = [];

    // Generate recommendations based on metrics
    if (metrics.memoryUsage > 80) {
      recommendations.push('High memory usage detected. Consider implementing memory cleanup.');
    }

    if (metrics.networkLatency > 1000) {
      recommendations.push('High network latency. Consider implementing request caching.');
    }

    if (metrics.renderTime > 3000) {
      recommendations.push('Slow render time. Consider implementing lazy loading.');
    }

    if (metrics.cacheHitRate < 50) {
      recommendations.push('Low cache hit rate. Review caching strategy.');
    }

    // Cache statistics
    const cacheEntries = Array.from(this.cache.values());
    const oldestTimestamp = Math.min(...cacheEntries.map(entry => entry.timestamp));

    return {
      metrics,
      recommendations,
      cacheStats: {
        size: this.cache.size,
        hitRate: metrics.cacheHitRate,
        oldestEntry: Date.now() - oldestTimestamp
      }
    };
  }

  /**
   * Clear cache
   */
  public clearCache(): void {
    this.cache.clear();
    this.metrics.cacheHitRate = 0;
  }

  /**
   * Cleanup and dispose
   */
  public dispose(): void {
    this.observers.forEach(observer => observer.disconnect());
    this.observers = [];
    this.clearCache();
    this.isMonitoring = false;
  }
}

// Singleton instance
export const performanceOptimizer = new PerformanceOptimizer();
export default PerformanceOptimizer;
