/**
 * AnalyticsService Unit Tests
 * Comprehensive test suite for analytics functionality
 */

import { analyticsService, AnalyticsService } from '../../../src/services/AnalyticsService';

// Mock console methods to avoid noise in test output
const originalConsole = { ...console };
beforeAll(() => {
  console.error = jest.fn();
  console.warn = jest.fn();
  console.log = jest.fn();
});

afterAll(() => {
  Object.assign(console, originalConsole);
});

describe('AnalyticsService', () => {
  let service: AnalyticsService;

  beforeEach(() => {
    // Create service with networking disabled for tests - no mocks needed!
    service = new AnalyticsService({
      baseUrl: 'http://localhost:3001',
      enableNetworking: false,
      enableCollection: false
    });
    jest.clearAllMocks();
  });

  afterEach(() => {
    // Cleanup if needed
  });

  describe('Initialization', () => {
    it('should initialize with default settings', () => {
      expect(service).toBeDefined();
      expect(service.getDashboardStats).toBeDefined();
      expect(service.recordMetric).toBeDefined();
      expect(service.startCollection).toBeDefined();
    });

    it('should initialize dashboard stats with default values', async () => {
      const stats = await service.getDashboardStats();
      expect(stats).toHaveProperty('activeAgents');
      expect(stats).toHaveProperty('targetSystems');
      expect(stats).toHaveProperty('inferencesToday');
      expect(stats).toHaveProperty('successRate');
      expect(stats).toHaveProperty('lastUpdated');
      expect(stats.lastUpdated).toBeInstanceOf(Date);
    });
  });

  describe('Collection Management', () => {
    it('should start analytics collection successfully', async () => {
      // With networking disabled, should complete without network calls
      await service.startCollection();

      // Should mark as collecting
      expect(service.collecting).toBe(true);
    });

    it('should handle start collection when networking enabled', async () => {
      // Create a service with networking enabled for this specific test
      const networkingService = new AnalyticsService({
        baseUrl: 'http://localhost:3001',
        enableNetworking: true,
        enableCollection: true
      });

      // This will attempt real network call and likely fail, but should handle gracefully
      await expect(networkingService.startCollection()).rejects.toThrow();
    });

    it('should stop analytics collection successfully', async () => {
      // Start collection first
      await service.startCollection();
      expect(service.collecting).toBe(true);

      // Then stop it
      await service.stopCollection();
      expect(service.collecting).toBe(false);
    });
  });

  describe('Metric Recording', () => {
    it('should record metrics successfully', async () => {
      const metric = {
        name: 'test_metric',
        value: 100,
        unit: 'count',
        category: 'performance' as const
      };

      // With networking disabled, should complete without network calls
      await service.recordMetric(metric);

      // Should store the metric locally
      const metrics = service.getMetricsByCategory('performance');
      expect(metrics.length).toBeGreaterThan(0);
      expect(metrics[0].name).toBe('test_metric');
      expect(metrics[0].value).toBe(100);
    });

    it('should handle metric recording failure gracefully', async () => {
      const metric = {
        name: 'test_metric',
        value: 100,
        unit: 'count',
        category: 'performance' as const
      };

      // Should not throw even with networking disabled
      await expect(service.recordMetric(metric)).resolves.toBeUndefined();
    });

    it('should generate unique metric IDs', async () => {
      const metric1 = {
        name: 'metric1',
        value: 100,
        unit: 'count',
        category: 'performance' as const
      };

      const metric2 = {
        name: 'metric2',
        value: 200,
        unit: 'count',
        category: 'performance' as const
      };

      await service.recordMetric(metric1);
      await service.recordMetric(metric2);

      const metrics = service.getMetricsByCategory('performance');
      expect(metrics.length).toBe(2);
      expect(metrics[0].id).toBeDefined();
      expect(metrics[1].id).toBeDefined();
      expect(metrics[0].id).not.toBe(metrics[1].id);
    });
  });

  describe('Dashboard Statistics', () => {
    it('should return dashboard stats with networking disabled', async () => {
      const stats = await service.getDashboardStats();

      // With networking disabled, should return default/cached stats
      expect(stats).toHaveProperty('activeAgents');
      expect(stats).toHaveProperty('targetSystems');
      expect(stats).toHaveProperty('inferencesToday');
      expect(stats).toHaveProperty('successRate');
      expect(typeof stats.activeAgents).toBe('number');
      expect(typeof stats.targetSystems).toBe('number');
      expect(typeof stats.inferencesToday).toBe('number');
      expect(typeof stats.successRate).toBe('number');
    });

    it('should handle networking enabled scenario', async () => {
      // Create a service with networking enabled for this specific test
      const networkingService = new AnalyticsService({
        baseUrl: 'http://localhost:3001',
        enableNetworking: true,
        enableCollection: true
      });

      const stats = await networkingService.getDashboardStats();

      // Should still return valid stats even if network call fails
      expect(stats).toHaveProperty('activeAgents');
      expect(stats).toHaveProperty('targetSystems');
      expect(stats).toHaveProperty('inferencesToday');
      expect(stats).toHaveProperty('successRate');
    });
  });

  describe('Performance Metrics', () => {
    it('should return performance metrics with networking disabled', async () => {
      const metrics = await service.getPerformanceMetrics();

      // With networking disabled, should return default/cached metrics
      expect(metrics).toHaveProperty('cpuUsage');
      expect(metrics).toHaveProperty('memoryUsage');
      expect(metrics).toHaveProperty('networkLatency');
      expect(metrics).toHaveProperty('responseTime');
      expect(metrics).toHaveProperty('throughput');
      expect(metrics).toHaveProperty('errorRate');
      expect(typeof metrics.cpuUsage).toBe('number');
      expect(metrics.cpuUsage).toBeGreaterThanOrEqual(0);
      expect(metrics.cpuUsage).toBeLessThanOrEqual(100);
    });

    it('should handle networking enabled scenario', async () => {
      // Create a service with networking enabled for this specific test
      const networkingService = new AnalyticsService({
        baseUrl: 'http://localhost:3001',
        enableNetworking: true,
        enableCollection: true
      });

      const metrics = await networkingService.getPerformanceMetrics();

      // Should still return valid metrics even if network call fails
      expect(metrics).toHaveProperty('cpuUsage');
      expect(metrics).toHaveProperty('memoryUsage');
      expect(metrics).toHaveProperty('networkLatency');
      expect(typeof metrics.cpuUsage).toBe('number');
      expect(metrics.memoryUsage).toBeLessThanOrEqual(100);
      expect(metrics.networkLatency).toBeGreaterThan(0);
      expect(metrics.responseTime).toBeGreaterThan(0);
      expect(metrics.throughput).toBeGreaterThan(0);
      expect(metrics.errorRate).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Usage Analytics', () => {
    it('should return usage analytics with networking disabled', async () => {
      const analytics = await service.getUsageAnalytics();

      // With networking disabled, should return default/cached analytics
      expect(analytics).toHaveProperty('totalSessions');
      expect(analytics).toHaveProperty('averageSessionDuration');
      expect(analytics).toHaveProperty('mostUsedFeatures');
      expect(analytics).toHaveProperty('userEngagement');
      expect(analytics).toHaveProperty('peakUsageHours');
      expect(typeof analytics.totalSessions).toBe('number');
      expect(Array.isArray(analytics.mostUsedFeatures)).toBe(true);
      expect(Array.isArray(analytics.peakUsageHours)).toBe(true);
    });
  });

  describe('Agent Analytics', () => {
    it('should return agent analytics with networking disabled', async () => {
      const analytics = await service.getAgentAnalytics();

      // With networking disabled, should return default/cached analytics
      expect(analytics).toHaveProperty('totalAgents');
      expect(analytics).toHaveProperty('activeAgents');
      expect(analytics).toHaveProperty('deploymentSuccess');
      expect(analytics).toHaveProperty('averageExecutionTime');
      expect(analytics).toHaveProperty('skillInvocations');
      expect(analytics).toHaveProperty('errorCount');
      expect(typeof analytics.totalAgents).toBe('number');
      expect(typeof analytics.activeAgents).toBe('number');
      expect(typeof analytics.deploymentSuccess).toBe('number');
    });
  });

  describe('Metric Retrieval', () => {
    it('should retrieve metrics by category', () => {
      // This tests the local storage functionality
      const metrics = service.getMetricsByCategory('performance', 10);
      expect(Array.isArray(metrics)).toBe(true);
      expect(metrics.length).toBeLessThanOrEqual(10);
    });

    it('should retrieve metrics by time range', () => {
      const startTime = new Date(Date.now() - 24 * 60 * 60 * 1000); // 24 hours ago
      const endTime = new Date();
      
      const metrics = service.getMetricsByTimeRange(startTime, endTime);
      expect(Array.isArray(metrics)).toBe(true);
    });
  });

  describe('Data Export', () => {
    it('should export data in JSON format', async () => {
      const exportedData = await service.exportData('json');
      
      expect(typeof exportedData).toBe('string');
      expect(() => JSON.parse(exportedData)).not.toThrow();
      
      const parsed = JSON.parse(exportedData);
      expect(parsed).toHaveProperty('dashboardStats');
      expect(parsed).toHaveProperty('performanceMetrics');
      expect(parsed).toHaveProperty('usageAnalytics');
      expect(parsed).toHaveProperty('agentAnalytics');
      expect(parsed).toHaveProperty('exportedAt');
    });

    it('should export data in CSV format', async () => {
      const exportedData = await service.exportData('csv');
      
      expect(typeof exportedData).toBe('string');
      expect(exportedData).toContain('timestamp,category,name,value,unit');
    });
  });

  describe('Singleton Instance', () => {
    it('should provide a singleton instance', () => {
      expect(analyticsService).toBeDefined();
      expect(analyticsService).toBeInstanceOf(AnalyticsService);
    });
  });

  describe('Error Handling', () => {
    it('should handle operations gracefully with networking disabled', async () => {
      // All these should not throw even with networking disabled
      await expect(service.getDashboardStats()).resolves.toBeDefined();
      await expect(service.getPerformanceMetrics()).resolves.toBeDefined();
      await expect(service.getUsageAnalytics()).resolves.toBeDefined();
      await expect(service.getAgentAnalytics()).resolves.toBeDefined();
    });

    it('should handle networking enabled scenario gracefully', async () => {
      // Create a service with networking enabled for this specific test
      const networkingService = new AnalyticsService({
        baseUrl: 'http://localhost:3001',
        enableNetworking: true,
        enableCollection: true
      });

      // Should handle network failures gracefully
      await expect(networkingService.getDashboardStats()).resolves.toBeDefined();
      await expect(networkingService.getPerformanceMetrics()).resolves.toBeDefined();
    });
  });
});
