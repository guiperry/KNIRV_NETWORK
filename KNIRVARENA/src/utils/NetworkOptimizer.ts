/**
 * Network Optimizer
 * Advanced network optimization and request management for KNIRV Controller
 */

import { errorHandler, ErrorCategory, ErrorSeverity } from './ErrorHandler';
import { performanceOptimizer } from './PerformanceOptimizer';

interface RequestConfig {
  url: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
  retries?: number;
  retryDelay?: number;
  cache?: boolean;
  priority?: 'low' | 'normal' | 'high';
  currentRetries?: number;
}

interface QueuedRequest extends RequestConfig {
  resolve: () => void;
}

interface NetworkMetrics {
  totalRequests: number;
  successfulRequests: number;
  failedRequests: number;
  averageLatency: number;
  cacheHitRate: number;
  bytesTransferred: number;
  activeConnections: number;
}

interface ConnectionPool {
  maxConnections: number;
  activeConnections: number;
  queuedRequests: RequestConfig[];
}

class NetworkOptimizer {
  private metrics: NetworkMetrics;
  private connectionPool: ConnectionPool;
  private requestQueue: Map<string, QueuedRequest[]> = new Map();
  private abortControllers: Map<string, AbortController> = new Map();
  private retryQueue: RequestConfig[] = [];
  private isProcessingQueue: boolean = false;

  constructor() {
    this.metrics = {
      totalRequests: 0,
      successfulRequests: 0,
      failedRequests: 0,
      averageLatency: 0,
      cacheHitRate: 0,
      bytesTransferred: 0,
      activeConnections: 0
    };

    this.connectionPool = {
      maxConnections: 6, // Browser default
      activeConnections: 0,
      queuedRequests: []
    };

    this.startQueueProcessor();
  }

  /**
   * Optimized fetch with automatic retries, caching, and connection pooling
   */
  public async optimizedFetch<T>(config: RequestConfig): Promise<T> {
    const startTime = Date.now();
    const requestId = this.generateRequestId();

    try {
      // Check cache first
      if (config.cache !== false && config.method === 'GET') {
        const cached = performanceOptimizer.cache_get<T>(this.getCacheKey(config));
        if (cached) {
          this.updateCacheMetrics(true);
          return cached;
        }
        this.updateCacheMetrics(false);
      }

      // Check connection pool
      if (this.connectionPool.activeConnections >= this.connectionPool.maxConnections) {
        await this.queueRequest(config);
      }

      // Execute request
      const response = await this.executeRequest(config, requestId);
      const data = await this.parseResponse<T>(response);

      // Cache successful GET requests
      if (config.cache !== false && config.method === 'GET' && response.ok) {
        performanceOptimizer.cache_set(this.getCacheKey(config), data);
      }

      // Update metrics
      this.updateSuccessMetrics(startTime, response);

      return data;

    } catch (error) {
      this.updateFailureMetrics(startTime);
      
      // Handle retries
      if (this.shouldRetry(config, error as Error)) {
        return this.retryRequest(config);
      }

      // Log error
      await errorHandler.handleError(
        error as Error,
        {
          component: 'NetworkOptimizer',
          action: 'optimizedFetch',
          additionalData: { url: config.url, method: config.method }
        },
        ErrorCategory.NETWORK,
        ErrorSeverity.MEDIUM
      );

      throw error;
    } finally {
      this.connectionPool.activeConnections--;
      this.abortControllers.delete(requestId);
    }
  }

  /**
   * Execute the actual network request
   */
  private async executeRequest(config: RequestConfig, requestId: string): Promise<Response> {
    this.connectionPool.activeConnections++;
    this.metrics.totalRequests++;

    // Create abort controller for timeout and cancellation
    const abortController = new AbortController();
    this.abortControllers.set(requestId, abortController);

    // Set timeout
    const timeout = config.timeout || 30000;
    const timeoutId = setTimeout(() => {
      abortController.abort();
    }, timeout);

    try {
      const response = await fetch(config.url, {
        method: config.method,
        headers: {
          'Content-Type': 'application/json',
          ...config.headers
        },
        body: config.body ? JSON.stringify(config.body) : undefined,
        signal: abortController.signal
      });

      clearTimeout(timeoutId);

      // Ensure response is valid
      if (!response) {
        throw new Error('Network request returned undefined response');
      }

      return response;

    } catch (error) {
      clearTimeout(timeoutId);

      // Handle specific error types
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new Error('Request timeout');
        }
        if (error.message.includes('fetch')) {
          throw new Error(`Network error: ${error.message}`);
        }
      }

      throw error;
    }
  }

  /**
   * Parse response based on content type
   */
  private async parseResponse<T>(response: Response | undefined): Promise<T> {
    if (!response) {
      throw new Error('Response is undefined');
    }

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const contentType = response.headers.get('content-type');

    if (contentType?.includes('application/json')) {
      return response.json();
    } else if (contentType?.includes('text/')) {
      return response.text() as unknown as T;
    } else {
      return response.blob() as unknown as T;
    }
  }

  /**
   * Queue request when connection pool is full
   */
  private async queueRequest(config: RequestConfig): Promise<void> {
    return new Promise((resolve) => {
      const priority = config.priority || 'normal';
      
      if (!this.requestQueue.has(priority)) {
        this.requestQueue.set(priority, []);
      }
      
      this.requestQueue.get(priority)!.push({
        ...config,
        resolve: resolve
      });
    });
  }

  /**
   * Process queued requests
   */
  private startQueueProcessor(): void {
    setInterval(() => {
      if (this.isProcessingQueue || this.connectionPool.activeConnections >= this.connectionPool.maxConnections) {
        return;
      }

      this.processQueue();
    }, 100);
  }

  /**
   * Process requests from queue based on priority
   */
  private processQueue(): void {
    if (this.isProcessingQueue) return;
    
    this.isProcessingQueue = true;

    try {
      const priorities = ['high', 'normal', 'low'];
      
      for (const priority of priorities) {
        const queue = this.requestQueue.get(priority);
        if (queue && queue.length > 0) {
          const config = queue.shift()!;
          if (config.resolve) {
            config.resolve();
          }
          break;
        }
      }
    } finally {
      this.isProcessingQueue = false;
    }
  }

  /**
   * Determine if request should be retried
   */
  private shouldRetry(config: RequestConfig, error: Error): boolean {
    const retries = config.retries || 3;
    const currentRetries = config.currentRetries || 0;

    if (currentRetries >= retries) {
      return false;
    }

    // Retry on network errors, timeouts, and 5xx status codes
    const retryableErrors = [
      'NetworkError',
      'TimeoutError',
      'AbortError'
    ];

    return retryableErrors.some(errorType => 
      error.name.includes(errorType) || error.message.includes(errorType)
    );
  }

  /**
   * Retry failed request with exponential backoff
   */
  private async retryRequest<T>(config: RequestConfig): Promise<T> {
    const currentRetries = config.currentRetries || 0;
    const retryDelay = (config.retryDelay || 1000) * Math.pow(2, currentRetries);

    // Wait before retry
    await new Promise(resolve => setTimeout(resolve, retryDelay));

    // Update retry count
    const retryConfig = {
      ...config,
      currentRetries: currentRetries + 1
    };

    return this.optimizedFetch<T>(retryConfig);
  }

  /**
   * Batch multiple requests for efficiency
   */
  public async batchRequests<T>(configs: RequestConfig[]): Promise<T[]> {
    const batchSize = 5; // Limit concurrent requests
    const results: T[] = [];

    for (let i = 0; i < configs.length; i += batchSize) {
      const batch = configs.slice(i, i + batchSize);
      const batchPromises = batch.map(config => this.optimizedFetch<T>(config));
      
      try {
        const batchResults = await Promise.allSettled(batchPromises);
        
        batchResults.forEach((result, index) => {
          if (result.status === 'fulfilled') {
            results.push(result.value);
          } else {
            console.error(`Batch request ${i + index} failed:`, result.reason);
            results.push(null as unknown as T); // Placeholder for failed request
          }
        });
      } catch (error) {
        console.error('Batch processing failed:', error);
      }
    }

    return results;
  }

  /**
   * Cancel request by ID
   */
  public cancelRequest(requestId: string): void {
    const controller = this.abortControllers.get(requestId);
    if (controller) {
      controller.abort();
      this.abortControllers.delete(requestId);
    }
  }

  /**
   * Cancel all active requests
   */
  public cancelAllRequests(): void {
    this.abortControllers.forEach(controller => controller.abort());
    this.abortControllers.clear();
  }

  /**
   * Preload resources for better performance
   */
  public preloadResource(url: string): void {
    if (typeof document !== 'undefined') {
      const link = document.createElement('link');
      link.rel = 'preload';
      link.href = url;
      link.as = 'fetch';
      link.crossOrigin = 'anonymous';
      document.head.appendChild(link);
    }
  }

  /**
   * Update success metrics
   */
  private updateSuccessMetrics(startTime: number, response: Response): void {
    const latency = Date.now() - startTime;
    
    this.metrics.successfulRequests++;
    this.metrics.averageLatency = (
      (this.metrics.averageLatency * (this.metrics.successfulRequests - 1)) + latency
    ) / this.metrics.successfulRequests;

    // Estimate bytes transferred
    const contentLength = response.headers.get('content-length');
    if (contentLength) {
      this.metrics.bytesTransferred += parseInt(contentLength, 10);
    }
  }

  /**
   * Update failure metrics
   */
  private updateFailureMetrics(startTime: number): void {
    this.metrics.failedRequests++;

    // Calculate and track failure response time
    const failureTime = Date.now() - startTime;
    if (!this.metrics.averageLatency) {
      this.metrics.averageLatency = failureTime;
    } else {
      // Update running average including failed request time
      this.metrics.averageLatency =
        (this.metrics.averageLatency + failureTime) / 2;
    }
  }

  /**
   * Update cache metrics
   */
  private updateCacheMetrics(hit: boolean): void {
    const totalCacheRequests = this.metrics.successfulRequests + this.metrics.failedRequests;
    if (hit) {
      this.metrics.cacheHitRate = ((this.metrics.cacheHitRate * totalCacheRequests) + 1) / (totalCacheRequests + 1);
    } else {
      this.metrics.cacheHitRate = (this.metrics.cacheHitRate * totalCacheRequests) / (totalCacheRequests + 1);
    }
  }

  /**
   * Generate cache key for request
   */
  private getCacheKey(config: RequestConfig): string {
    const params = new URLSearchParams();
    if (config.body) {
      params.set('body', JSON.stringify(config.body));
    }
    return `${config.method}:${config.url}:${params.toString()}`;
  }

  /**
   * Generate unique request ID
   */
  private generateRequestId(): string {
    return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Get current network metrics
   */
  public getMetrics(): NetworkMetrics {
    return { ...this.metrics };
  }

  /**
   * Get connection pool status
   */
  public getConnectionPoolStatus(): ConnectionPool {
    return { ...this.connectionPool };
  }

  /**
   * Reset metrics
   */
  public resetMetrics(): void {
    this.metrics = {
      totalRequests: 0,
      successfulRequests: 0,
      failedRequests: 0,
      averageLatency: 0,
      cacheHitRate: 0,
      bytesTransferred: 0,
      activeConnections: 0
    };
  }

  /**
   * Generate network performance report
   */
  public generateReport(): {
    metrics: NetworkMetrics;
    connectionPool: ConnectionPool;
    recommendations: string[];
  } {
    const recommendations: string[] = [];

    // Generate recommendations based on metrics
    if (this.metrics.failedRequests / this.metrics.totalRequests > 0.1) {
      recommendations.push('High failure rate detected. Check network connectivity and server status.');
    }

    if (this.metrics.averageLatency > 2000) {
      recommendations.push('High average latency. Consider implementing request optimization.');
    }

    if (this.metrics.cacheHitRate < 30) {
      recommendations.push('Low cache hit rate. Review caching strategy.');
    }

    if (this.connectionPool.queuedRequests.length > 10) {
      recommendations.push('High number of queued requests. Consider increasing connection pool size.');
    }

    return {
      metrics: this.getMetrics(),
      connectionPool: this.getConnectionPoolStatus(),
      recommendations
    };
  }
}

// Singleton instance
export const networkOptimizer = new NetworkOptimizer();
export default NetworkOptimizer;
