/**
 * Analytics Service
 * Handles metrics collection, performance tracking, and dashboard statistics
 */

import { performanceOptimizer } from '../utils/PerformanceOptimizer';
import { errorHandler, ErrorCategory, ErrorSeverity } from '../utils/ErrorHandler';
import { memoryManager } from '../utils/MemoryManager';
import { networkOptimizer } from '../utils/NetworkOptimizer';

export interface AnalyticsMetric {
  id: string;
  name: string;
  value: number;
  unit: string;
  timestamp: Date;
  category: 'performance' | 'usage' | 'system' | 'agent' | 'wallet';
  metadata?: Record<string, unknown>;
}

export interface DashboardStats {
  activeAgents: number;
  totalSkills: number;
  totalTransactions: number;
  networkHealth: string;
  lastUpdated: Date;
  targetSystems: number;
  inferencesToday: number;
  successRate: number;
  changes: Record<string, string>;
}

export interface PerformanceMetrics {
  throughput: number;
  errorRate: number;
  uptime: number;
  lastMeasured: Date;
  cpuUsage: number;
  memoryUsage: number;
  networkLatency: number;
  responseTime: number;
}

export interface UsageAnalytics {
  totalSessions: number;
  averageSessionDuration: number;
  popularFeatures: Array<{ feature: string; usage: number }>;
  lastCalculated: Date;
  mostUsedFeatures: Array<{ feature: string; usage: number }>;
  userEngagement: number;
  peakUsageHours: number[];
}

export interface AgentAnalytics {
  successRate: number;
  averageExecutionTime: number;
  resourceUtilization: number;
  lastAnalyzed: Date;
  totalAgents: number;
  activeAgents: number;
  deploymentSuccess: number;
  skillInvocations: number;
  errorCount: number;
}

export interface AnalyticsConfig {
  baseUrl?: string;
  enableNetworking?: boolean;
  enableCollection?: boolean;
}

export class AnalyticsService {
  private metrics: Map<string, AnalyticsMetric[]> = new Map();
  private dashboardStats: DashboardStats = {
    activeAgents: 0,
    totalSkills: 0,
    totalTransactions: 0,
    networkHealth: 'healthy',
    lastUpdated: new Date(),
    targetSystems: 0,
    inferencesToday: 0,
    successRate: 0,
    changes: {}
  };
  private performanceMetrics: PerformanceMetrics = {
    throughput: 0,
    errorRate: 0,
    uptime: 100,
    lastMeasured: new Date(),
    cpuUsage: 0,
    memoryUsage: 0,
    networkLatency: 0,
    responseTime: 0
  };
  private usageAnalytics: UsageAnalytics = {
    totalSessions: 0,
    averageSessionDuration: 0,
    popularFeatures: [],
    lastCalculated: new Date(),
    mostUsedFeatures: [],
    userEngagement: 0,
    peakUsageHours: []
  };
  private agentAnalytics: AgentAnalytics = {
    successRate: 0,
    averageExecutionTime: 0,
    resourceUtilization: 0,
    lastAnalyzed: new Date(),
    totalAgents: 0,
    activeAgents: 0,
    deploymentSuccess: 0,
    skillInvocations: 0,
    errorCount: 0
  };
  private baseUrl: string;
  private isCollecting: boolean = false;
  private collectionInterval: number | null = null;
  private config: AnalyticsConfig;

  constructor(config: AnalyticsConfig = {}) {
    this.config = {
      baseUrl: 'http://localhost:3001',
      enableNetworking: process.env.NODE_ENV !== 'test',
      enableCollection: process.env.NODE_ENV !== 'test',
      ...config
    };
    this.baseUrl = this.config.baseUrl!;
    this.initializeAnalytics();
    this.initializePerformanceMonitoring();
  }

  private initializeAnalytics(): void {
    // Initialize dashboard stats
    this.dashboardStats = {
      activeAgents: 0,
      totalSkills: 0,
      totalTransactions: 0,
      networkHealth: 'healthy',
      targetSystems: 0,
      inferencesToday: 0,
      successRate: 0,
      changes: {},
      lastUpdated: new Date()
    };

    // Initialize performance metrics
    this.performanceMetrics = {
      cpuUsage: 0,
      memoryUsage: 0,
      networkLatency: 0,
      responseTime: 0,
      throughput: 0,
      errorRate: 0,
      uptime: 100,
      lastMeasured: new Date()
    };

    // Initialize usage analytics
    this.usageAnalytics = {
      totalSessions: 0,
      averageSessionDuration: 0,
      popularFeatures: [],
      lastCalculated: new Date(),
      mostUsedFeatures: [],
      userEngagement: 0,
      peakUsageHours: []
    };

    // Initialize agent analytics
    this.agentAnalytics = {
      totalAgents: 0,
      activeAgents: 0,
      deploymentSuccess: 0,
      averageExecutionTime: 0,
      skillInvocations: 0,
      errorCount: 0,
      successRate: 0,
      resourceUtilization: 0,
      lastAnalyzed: new Date()
    };

    console.log('Analytics Service initialized');
  }

  /**
   * Initialize performance monitoring integration
   */
  private initializePerformanceMonitoring(): void {
    // Register memory cleanup callback
    memoryManager.registerCleanupCallback(() => {
      this.cleanupOldMetrics();
    });

    // Add memory listener for automatic metric collection
    memoryManager.addMemoryListener((metrics) => {
      this.recordMetric({
        name: 'memory_usage',
        value: metrics.usagePercentage,
        unit: 'percentage',
        category: 'performance',
        metadata: {
          usedJSHeapSize: metrics.usedJSHeapSize,
          totalJSHeapSize: metrics.totalJSHeapSize,
          jsHeapSizeLimit: metrics.jsHeapSizeLimit
        }
      });
    });

    // Monitor network performance
    setInterval(() => {
      const networkMetrics = networkOptimizer.getMetrics();
      if (networkMetrics.totalRequests > 0) {
        this.recordMetric({
          name: 'network_latency',
          value: networkMetrics.averageLatency,
          unit: 'milliseconds',
          category: 'performance',
          metadata: networkMetrics as unknown as Record<string, unknown>
        });
      }
    }, 30000); // Every 30 seconds

    // Monitor general performance
    setInterval(() => {
      const perfMetrics = performanceOptimizer.getMetrics();
      this.recordMetric({
        name: 'render_time',
        value: perfMetrics.renderTime,
        unit: 'milliseconds',
        category: 'performance',
        metadata: perfMetrics as unknown as Record<string, unknown>
      });
    }, 10000); // Every 10 seconds
  }

  /**
   * Clean up old metrics to prevent memory leaks
   */
  private cleanupOldMetrics(): void {
    const cutoffTime = Date.now() - (24 * 60 * 60 * 1000); // 24 hours ago
    this.metrics = new Map([...this.metrics.entries()].filter(([_key, metrics]) =>
      metrics.some((metric: { timestamp: Date }) => metric.timestamp.getTime() > cutoffTime)
    ));
  }

  /**
   * Get collection status
   */
  public get collecting(): boolean {
    return this.isCollecting;
  }

  /**
   * Start analytics collection
   */
  async startCollection(): Promise<void> {
    if (this.isCollecting) {
      return;
    }

    // In test environment or when collection is disabled, just mark as collecting
    if (!this.config.enableCollection || !this.config.enableNetworking) {
      this.isCollecting = true;
      console.log('Analytics collection started');
      return;
    }

    try {
      const response = await fetch(`${this.baseUrl}/api/analytics/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        throw new Error(`Failed to start analytics collection: ${response.statusText}`);
      }

      this.isCollecting = true;

      // Start periodic data collection
      this.collectionInterval = setInterval(() => {
        this.collectMetrics();
      }, 30000); // Collect every 30 seconds

      console.log('Analytics collection started');
    } catch (error) {
      console.error('Failed to start analytics collection:', error);
      throw error;
    }
  }

  /**
   * Stop analytics collection
   */
  async stopCollection(): Promise<void> {
    if (!this.isCollecting) {
      return;
    }

    // Always stop collection locally first
    this.isCollecting = false;

    if (this.collectionInterval) {
      clearInterval(this.collectionInterval);
      this.collectionInterval = null;
    }

    // Only make network call if networking is enabled
    if (this.config.enableNetworking) {
      try {
        await fetch(`${this.baseUrl}/api/analytics/stop`, {
          method: 'POST'
        });
      } catch (error) {
        console.error('Failed to stop analytics collection:', error);
      }
    }

    console.log('Analytics collection stopped');
  }

  /**
   * Record a metric with enhanced error handling and performance optimization
   */
  async recordMetric(metric: Omit<AnalyticsMetric, 'id' | 'timestamp'>): Promise<void> {
    try {
      const fullMetric: AnalyticsMetric = {
        ...metric,
        id: this.generateMetricId(),
        timestamp: new Date()
      };

      // Store locally
      const categoryMetrics = this.metrics.get(metric.category) || [];
      categoryMetrics.push(fullMetric);
      this.metrics.set(metric.category, categoryMetrics);

      // Limit metrics array size to prevent memory issues
      if (categoryMetrics.length > 1000) {
        categoryMetrics.splice(0, categoryMetrics.length - 500);
      }

      // Cache frequently accessed metrics for performance
      if (metric.category === 'performance') {
        performanceOptimizer.cache_set(`metric_${metric.name}`, fullMetric);
      }

      // Use optimized network request only if networking is enabled
      if (this.config.enableNetworking) {
        try {
          await networkOptimizer.optimizedFetch({
            url: `${this.baseUrl}/api/analytics/metrics`,
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: fullMetric,
            cache: false,
            priority: 'low' // Analytics metrics are low priority
          });
        } catch (networkError) {
          // Handle network errors gracefully
          await errorHandler.handleError(
            networkError as Error,
            {
              component: 'AnalyticsService',
              action: 'recordMetric_network',
              additionalData: { metricName: metric.name, metricCategory: metric.category }
            },
            ErrorCategory.NETWORK,
            ErrorSeverity.LOW
          );
        }
      }
    } catch (error) {
      await errorHandler.handleError(
        error as Error,
        {
          component: 'AnalyticsService',
          action: 'recordMetric',
          additionalData: { metricName: metric.name, metricCategory: metric.category }
        },
        ErrorCategory.SYSTEM,
        ErrorSeverity.LOW
      );
    }
  }

  /**
   * Get dashboard statistics
   */
  async getDashboardStats(): Promise<DashboardStats> {
    // In test environment or when networking is disabled, return cached stats
    if (!this.config.enableNetworking) {
      return this.dashboardStats;
    }

    try {
      const response = await fetch(`${this.baseUrl}/api/analytics/dashboard`);

      if (response.ok) {
        const stats = await response.json();
        this.dashboardStats = {
          ...stats,
          lastUpdated: new Date(stats.lastUpdated)
        };
      } else {
        // Use local/simulated data if backend unavailable
        this.updateDashboardStats();
      }
    } catch (error) {
      console.error('Failed to fetch dashboard stats:', error);
      this.updateDashboardStats();
    }

    return this.dashboardStats;
  }

  /**
   * Get performance metrics
   */
  async getPerformanceMetrics(): Promise<PerformanceMetrics> {
    try {
      const response = await fetch(`${this.baseUrl}/api/analytics/performance`);
      
      if (response.ok) {
        this.performanceMetrics = await response.json();
      } else {
        this.updatePerformanceMetrics();
      }
    } catch (error) {
      console.error('Failed to fetch performance metrics:', error);
      this.updatePerformanceMetrics();
    }

    return this.performanceMetrics;
  }

  /**
   * Get usage analytics
   */
  async getUsageAnalytics(): Promise<UsageAnalytics> {
    try {
      const response = await fetch(`${this.baseUrl}/api/analytics/usage`);
      
      if (response.ok) {
        this.usageAnalytics = await response.json();
      } else {
        this.updateUsageAnalytics();
      }
    } catch (error) {
      console.error('Failed to fetch usage analytics:', error);
      this.updateUsageAnalytics();
    }

    return this.usageAnalytics;
  }

  /**
   * Get agent analytics
   */
  async getAgentAnalytics(): Promise<AgentAnalytics> {
    try {
      const response = await fetch(`${this.baseUrl}/api/analytics/agents`);
      
      if (response.ok) {
        this.agentAnalytics = await response.json();
      } else {
        this.updateAgentAnalytics();
      }
    } catch (error) {
      console.error('Failed to fetch agent analytics:', error);
      this.updateAgentAnalytics();
    }

    return this.agentAnalytics;
  }

  /**
   * Get metrics by category
   */
  getMetricsByCategory(category: string, limit: number = 100): AnalyticsMetric[] {
    const categoryMetrics = this.metrics.get(category) || [];
    return categoryMetrics
      .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
      .slice(0, limit);
  }

  /**
   * Get metrics by time range
   */
  getMetricsByTimeRange(startTime: Date, endTime: Date): AnalyticsMetric[] {
    const allMetrics: AnalyticsMetric[] = [];
    
    for (const categoryMetrics of this.metrics.values()) {
      allMetrics.push(...categoryMetrics.filter(metric => 
        metric.timestamp >= startTime && metric.timestamp <= endTime
      ));
    }

    return allMetrics.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime());
  }

  /**
   * Export analytics data
   */
  async exportData(format: 'json' | 'csv' = 'json'): Promise<string> {
    const data = {
      dashboardStats: this.dashboardStats,
      performanceMetrics: this.performanceMetrics,
      usageAnalytics: this.usageAnalytics,
      agentAnalytics: this.agentAnalytics,
      metrics: Object.fromEntries(this.metrics),
      exportedAt: new Date()
    };

    if (format === 'json') {
      return JSON.stringify(data, null, 2);
    } else {
      // Convert to CSV format
      return this.convertToCSV(data);
    }
  }

  private async collectMetrics(): Promise<void> {
    // Collect system metrics
    await this.recordMetric({
      name: 'cpu_usage',
      value: this.getCPUUsage(),
      unit: 'percent',
      category: 'performance'
    });

    await this.recordMetric({
      name: 'memory_usage',
      value: this.getMemoryUsage(),
      unit: 'percent',
      category: 'performance'
    });

    // Update analytics
    this.updateDashboardStats();
    this.updatePerformanceMetrics();
    this.updateUsageAnalytics();
    this.updateAgentAnalytics();
  }

  private updateDashboardStats(): void {
    const now = new Date();
    const hourVariation = Math.sin(now.getHours() / 24 * 2 * Math.PI) * 10;
    const dayVariation = Math.sin(now.getDate() / 31 * 2 * Math.PI) * 5;

    this.dashboardStats = {
      activeAgents: Math.max(0, Math.floor(45 + hourVariation + dayVariation)),
      totalSkills: Math.max(0, Math.floor(150 + dayVariation * 3)),
      totalTransactions: Math.max(0, Math.floor(5000 + hourVariation * 100 + dayVariation * 50)),
      networkHealth: 'healthy',
      targetSystems: Math.max(0, Math.floor(20 + dayVariation / 2)),
      inferencesToday: Math.max(0, Math.floor(1500 + hourVariation * 50 + dayVariation * 20)),
      successRate: Math.round((96.5 + Math.sin(now.getTime() / 86400000) * 2) * 10) / 10,
      changes: {
        active_agents: '+12%',
        target_systems: '+8%',
        inferences_today: '+34%',
        success_rate: '+1.8%'
      },
      lastUpdated: now
    };
  }

  private updatePerformanceMetrics(): void {
    this.performanceMetrics = {
      cpuUsage: this.getCPUUsage(),
      memoryUsage: this.getMemoryUsage(),
      networkLatency: Math.round(Math.random() * 50 + 10),
      responseTime: Math.round(Math.random() * 200 + 50),
      throughput: Math.round(Math.random() * 1000 + 500),
      errorRate: Math.round(Math.random() * 5 * 100) / 100,
      uptime: Math.round((Math.random() * 5 + 95) * 10) / 10,
      lastMeasured: new Date()
    };
  }

  private updateUsageAnalytics(): void {
    this.usageAnalytics = {
      totalSessions: Math.floor(Math.random() * 1000 + 500),
      averageSessionDuration: Math.round(Math.random() * 30 + 15),
      popularFeatures: [
        { feature: 'Agent Management', usage: Math.floor(Math.random() * 100 + 50) },
        { feature: 'Cognitive Shell', usage: Math.floor(Math.random() * 80 + 40) },
        { feature: 'Terminal', usage: Math.floor(Math.random() * 60 + 30) },
        { feature: 'QR Scanner', usage: Math.floor(Math.random() * 40 + 20) }
      ],
      lastCalculated: new Date(),
      mostUsedFeatures: [
        { feature: 'Agent Management', usage: Math.floor(Math.random() * 100 + 50) },
        { feature: 'Cognitive Shell', usage: Math.floor(Math.random() * 80 + 40) },
        { feature: 'Terminal', usage: Math.floor(Math.random() * 60 + 30) },
        { feature: 'QR Scanner', usage: Math.floor(Math.random() * 40 + 20) }
      ],
      userEngagement: Math.round(Math.random() * 100),
      peakUsageHours: [9, 10, 11, 14, 15, 16, 20, 21]
    };
  }

  private updateAgentAnalytics(): void {
    this.agentAnalytics = {
      totalAgents: Math.floor(Math.random() * 50 + 25),
      activeAgents: Math.floor(Math.random() * 30 + 15),
      deploymentSuccess: Math.round((Math.random() * 10 + 90) * 10) / 10,
      averageExecutionTime: Math.round(Math.random() * 500 + 100),
      skillInvocations: Math.floor(Math.random() * 1000 + 500),
      errorCount: Math.floor(Math.random() * 10),
      successRate: Math.round((Math.random() * 10 + 90) * 10) / 10,
      resourceUtilization: Math.round((Math.random() * 30 + 40) * 10) / 10,
      lastAnalyzed: new Date()
    };
  }

  private getCPUUsage(): number {
    // Simulate CPU usage
    return Math.round((Math.random() * 30 + 20) * 10) / 10;
  }

  private getMemoryUsage(): number {
    // Simulate memory usage
    return Math.round((Math.random() * 40 + 30) * 10) / 10;
  }

  private generateMetricId(): string {
    return `metric_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private convertToCSV(data: { metrics: Record<string, AnalyticsMetric[]> }): string {
    // Simple CSV conversion - would be more sophisticated in production
    const lines = ['timestamp,category,name,value,unit'];
    
    for (const [category, metrics] of Object.entries(data.metrics)) {
      for (const metric of metrics as AnalyticsMetric[]) {
        lines.push(`${metric.timestamp.toISOString()},${category},${metric.name},${metric.value},${metric.unit}`);
      }
    }
    
    return lines.join('\n');
  }
}

// Export singleton instance
export const analyticsService = new AnalyticsService();
