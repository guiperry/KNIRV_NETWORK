/**
 * Performance Monitoring Service
 * Real-time performance monitoring and analytics for KNIRVCONTROLLER
 */

interface PerformanceMetric {
  name: string;
  value: number;
  timestamp: number;
  category: 'memory' | 'network' | 'rendering' | 'user-interaction' | 'game';
  metadata?: Record<string, unknown>;
}

interface PerformanceAlert {
  id: string;
  type: 'warning' | 'error' | 'critical';
  message: string;
  metric: string;
  threshold: number;
  currentValue: number;
  timestamp: number;
}

export class PerformanceMonitoringService {
  private metrics: PerformanceMetric[] = [];
  private alerts: PerformanceAlert[] = [];
  private observers: PerformanceObserver[] = [];
  private isMonitoring = false;
  private alertCallbacks: ((alert: PerformanceAlert) => void)[] = [];

  // Performance thresholds
  private thresholds = {
    memoryUsage: 100 * 1024 * 1024, // 100MB
    renderTime: 16, // 16ms for 60fps
    networkLatency: 2000, // 2 seconds
    gameFrameRate: 30, // 30fps minimum
    errorRate: 0.05 // 5% error rate
  };

  constructor() {
    this.setupPerformanceObservers();
  }

  /**
   * Start performance monitoring
   */
  startMonitoring(): void {
    if (this.isMonitoring) return;

    this.isMonitoring = true;
    this.startMemoryMonitoring();
    this.startNetworkMonitoring();
    this.startRenderingMonitoring();
    this.startGamePerformanceMonitoring();
    this.startUserInteractionMonitoring();

    console.log('Performance monitoring started');
  }

  /**
   * Stop performance monitoring
   */
  stopMonitoring(): void {
    this.isMonitoring = false;
    this.observers.forEach(observer => observer.disconnect());
    this.observers = [];

    console.log('Performance monitoring stopped');
  }

  /**
   * Record a custom performance metric
   */
  recordMetric(name: string, value: number, category: PerformanceMetric['category'], metadata?: Record<string, unknown>): void {
    const metric: PerformanceMetric = {
      name,
      value,
      timestamp: Date.now(),
      category,
      metadata
    };

    this.metrics.push(metric);
    this.checkThresholds(metric);

    // Keep only last 1000 metrics to prevent memory leaks
    if (this.metrics.length > 1000) {
      this.metrics = this.metrics.slice(-1000);
    }
  }

  /**
   * Get performance metrics by category
   */
  getMetrics(category?: PerformanceMetric['category'], timeRange?: number): PerformanceMetric[] {
    let filteredMetrics = this.metrics;

    if (category) {
      filteredMetrics = filteredMetrics.filter(m => m.category === category);
    }

    if (timeRange) {
      const cutoff = Date.now() - timeRange;
      filteredMetrics = filteredMetrics.filter(m => m.timestamp > cutoff);
    }

    return filteredMetrics;
  }

  /**
   * Get performance summary
   */
  getPerformanceSummary(): {
    memory: { current: number; peak: number; average: number };
    network: { averageLatency: number; errorRate: number };
    rendering: { averageFrameTime: number; droppedFrames: number };
    game: { averageFPS: number; totalOperations: number };
    alerts: PerformanceAlert[];
  } {
    const memoryMetrics = this.getMetrics('memory', 60000); // Last minute
    const networkMetrics = this.getMetrics('network', 60000);
    const renderingMetrics = this.getMetrics('rendering', 60000);
    const gameMetrics = this.getMetrics('game', 60000);

    return {
      memory: {
        current: this.getCurrentMemoryUsage(),
        peak: Math.max(...memoryMetrics.map(m => m.value), 0),
        average: memoryMetrics.length > 0 ? memoryMetrics.reduce((sum, m) => sum + m.value, 0) / memoryMetrics.length : 0
      },
      network: {
        averageLatency: networkMetrics.length > 0 ? networkMetrics.reduce((sum, m) => sum + m.value, 0) / networkMetrics.length : 0,
        errorRate: this.calculateErrorRate(networkMetrics)
      },
      rendering: {
        averageFrameTime: renderingMetrics.length > 0 ? renderingMetrics.reduce((sum, m) => sum + m.value, 0) / renderingMetrics.length : 0,
        droppedFrames: renderingMetrics.filter(m => m.value > 16).length
      },
      game: {
        averageFPS: this.calculateAverageFPS(gameMetrics),
        totalOperations: gameMetrics.filter(m => m.name === 'game-operation').length
      },
      alerts: this.alerts.slice(-10) // Last 10 alerts
    };
  }

  /**
   * Subscribe to performance alerts
   */
  onAlert(callback: (alert: PerformanceAlert) => void): () => void {
    this.alertCallbacks.push(callback);
    return () => {
      const index = this.alertCallbacks.indexOf(callback);
      if (index > -1) {
        this.alertCallbacks.splice(index, 1);
      }
    };
  }

  /**
   * Setup performance observers
   */
  private setupPerformanceObservers(): void {
    if (typeof PerformanceObserver === 'undefined') return;

    // Navigation timing
    const navObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.entryType === 'navigation') {
          const navEntry = entry as PerformanceNavigationTiming;
          this.recordMetric('page-load-time', navEntry.loadEventEnd - navEntry.startTime, 'network');
          this.recordMetric('dom-content-loaded', navEntry.domContentLoadedEventEnd - navEntry.startTime, 'rendering');
        }
      }
    });

    try {
      navObserver.observe({ entryTypes: ['navigation'] });
      this.observers.push(navObserver);
    } catch {
      console.warn('Navigation timing observer not supported');
    }

    // Resource timing
    const resourceObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.entryType === 'resource') {
          const resourceEntry = entry as PerformanceResourceTiming;
          this.recordMetric('resource-load-time', resourceEntry.responseEnd - resourceEntry.startTime, 'network', {
            name: resourceEntry.name,
            size: resourceEntry.transferSize
          });
        }
      }
    });

    try {
      resourceObserver.observe({ entryTypes: ['resource'] });
      this.observers.push(resourceObserver);
    } catch {
      console.warn('Resource timing observer not supported');
    }

    // Long tasks
    const longTaskObserver = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.entryType === 'longtask') {
          this.recordMetric('long-task', entry.duration, 'rendering', {
            startTime: entry.startTime
          });
        }
      }
    });

    try {
      longTaskObserver.observe({ entryTypes: ['longtask'] });
      this.observers.push(longTaskObserver);
    } catch {
      console.warn('Long task observer not supported');
    }
  }

  /**
   * Start memory monitoring
   */
  private startMemoryMonitoring(): void {
    const monitorMemory = () => {
      if (!this.isMonitoring) return;

      const memoryUsage = this.getCurrentMemoryUsage();
      this.recordMetric('memory-usage', memoryUsage, 'memory');

      setTimeout(monitorMemory, 5000); // Check every 5 seconds
    };

    monitorMemory();
  }

  /**
   * Start network monitoring
   */
  private startNetworkMonitoring(): void {
    // Monitor fetch requests
    const originalFetch = window.fetch;
    window.fetch = async (...args) => {
      const startTime = performance.now();
      try {
        const response = await originalFetch(...args);
        const endTime = performance.now();
        
        this.recordMetric('network-request', endTime - startTime, 'network', {
          url: args[0],
          status: response.status,
          success: response.ok
        });

        return response;
      } catch (error) {
        const endTime = performance.now();
        this.recordMetric('network-error', endTime - startTime, 'network', {
          url: args[0],
          error: error instanceof Error ? error.message : String(error)
        });
        throw error;
      }
    };
  }

  /**
   * Start rendering monitoring
   */
  private startRenderingMonitoring(): void {
    let lastFrameTime = performance.now();
    
    const measureFrame = () => {
      if (!this.isMonitoring) return;

      const currentTime = performance.now();
      const frameTime = currentTime - lastFrameTime;
      lastFrameTime = currentTime;

      this.recordMetric('frame-time', frameTime, 'rendering');

      requestAnimationFrame(measureFrame);
    };

    requestAnimationFrame(measureFrame);
  }

  /**
   * Start game performance monitoring
   */
  private startGamePerformanceMonitoring(): void {
    // Monitor game-specific operations
    const monitorGameOps = () => {
      if (!this.isMonitoring) return;

      // This would integrate with KnirvanaBridgeService
      try {
        const gameState = this.getGameState();
        if (gameState) {
          this.recordMetric('game-fps', gameState.fps || 0, 'game');
          this.recordMetric('game-agents', gameState.activeAgents || 0, 'game');
          this.recordMetric('game-errors-solved', gameState.errorsSolved || 0, 'game');
        }
      } catch {
        // Game not active
      }

      setTimeout(monitorGameOps, 1000); // Check every second
    };

    monitorGameOps();
  }

  /**
   * Start user interaction monitoring
   */
  private startUserInteractionMonitoring(): void {
    const interactionTypes = ['click', 'keydown', 'touchstart'];
    
    interactionTypes.forEach(type => {
      document.addEventListener(type, (event) => {
        this.recordMetric('user-interaction', 1, 'user-interaction', {
          type,
          target: (event.target as Element)?.tagName
        });
      });
    });
  }

  /**
   * Check performance thresholds and create alerts
   */
  private checkThresholds(metric: PerformanceMetric): void {
    let threshold: number | undefined;
    let alertType: PerformanceAlert['type'] = 'warning';

    switch (metric.name) {
      case 'memory-usage':
        threshold = this.thresholds.memoryUsage;
        alertType = metric.value > threshold * 1.5 ? 'critical' : 'warning';
        break;
      case 'frame-time':
        threshold = this.thresholds.renderTime;
        alertType = metric.value > threshold * 2 ? 'error' : 'warning';
        break;
      case 'network-request':
        threshold = this.thresholds.networkLatency;
        alertType = metric.value > threshold * 2 ? 'error' : 'warning';
        break;
      case 'game-fps':
        threshold = this.thresholds.gameFrameRate;
        alertType = metric.value < threshold / 2 ? 'critical' : 'warning';
        break;
    }

    if (threshold && (
      (metric.name === 'game-fps' && metric.value < threshold) ||
      (metric.name !== 'game-fps' && metric.value > threshold)
    )) {
      const alert: PerformanceAlert = {
        id: `alert-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        type: alertType,
        message: `Performance threshold exceeded for ${metric.name}`,
        metric: metric.name,
        threshold,
        currentValue: metric.value,
        timestamp: Date.now()
      };

      this.alerts.push(alert);
      this.alertCallbacks.forEach(callback => callback(alert));

      // Keep only last 100 alerts
      if (this.alerts.length > 100) {
        this.alerts = this.alerts.slice(-100);
      }
    }
  }

  /**
   * Get current memory usage
   */
  private getCurrentMemoryUsage(): number {
    if ('memory' in performance) {
      const perfWithMemory = performance as Performance & { memory?: { usedJSHeapSize: number } };
      return perfWithMemory.memory?.usedJSHeapSize || 0;
    }
    return 0;
  }

  /**
   * Calculate error rate from network metrics
   */
  private calculateErrorRate(metrics: PerformanceMetric[]): number {
    if (metrics.length === 0) return 0;
    
    const errorCount = metrics.filter(m => 
      m.name === 'network-error' || 
      (m.metadata?.status && typeof m.metadata.status === 'number' && m.metadata.status >= 400)
    ).length;
    
    return errorCount / metrics.length;
  }

  /**
   * Calculate average FPS from game metrics
   */
  private calculateAverageFPS(metrics: PerformanceMetric[]): number {
    const fpsMetrics = metrics.filter(m => m.name === 'game-fps');
    if (fpsMetrics.length === 0) return 0;
    
    return fpsMetrics.reduce((sum, m) => sum + m.value, 0) / fpsMetrics.length;
  }

  /**
   * Get game state (would integrate with actual game service)
   */
  private getGameState(): { fps: number; activeAgents: number; memoryUsage: number; networkLatency: number; errorsSolved?: number } {
    // This would integrate with KnirvanaBridgeService
    // For now, return mock data
    return {
      fps: 60,
      activeAgents: 3,
      memoryUsage: 45,
      networkLatency: 20,
      errorsSolved: 15
    };
  }

  /**
   * Export performance data
   */
  exportData(): string {
    return JSON.stringify({
      metrics: this.metrics,
      alerts: this.alerts,
      summary: this.getPerformanceSummary(),
      timestamp: new Date().toISOString()
    }, null, 2);
  }

  /**
   * Clear all performance data
   */
  clearData(): void {
    this.metrics = [];
    this.alerts = [];
  }
}
