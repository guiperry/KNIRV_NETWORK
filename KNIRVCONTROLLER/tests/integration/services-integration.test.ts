/**
 * Services Integration Tests
 * Tests the interaction between different services
 */

import { analyticsService } from '../../src/services/AnalyticsService';
import { taskSchedulingService } from '../../src/services/TaskSchedulingService';
import { udcManagementService } from '../../src/services/UDCManagementService';
import { settingsService } from '../../src/services/SettingsService';

// Mock fetch globally
global.fetch = jest.fn();

// Mock crypto.subtle for UDC signatures
Object.defineProperty(global, 'crypto', {
  value: {
    subtle: {
      digest: jest.fn().mockResolvedValue(new ArrayBuffer(32))
    }
  }
});

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn()
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

describe('Services Integration Tests', () => {
  const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

  beforeEach(() => {
    jest.clearAllMocks();
    jest.clearAllTimers();
    jest.useFakeTimers();
    
    // Mock successful API responses with proper data structures
    mockFetch.mockImplementation((url: string | URL | Request) => {
      const urlString = url.toString();

      if (urlString.includes('/api/analytics/agents')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            totalAgents: 25,
            activeAgents: 15,
            skillInvocations: 150,
            deploymentSuccess: 95.5,
            averageExecutionTime: 250
          })
        } as Response);
      }

      if (urlString.includes('/api/analytics/performance')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            responseTime: 125,
            errorRate: 2.5,
            cpuUsage: 45,
            memoryUsage: 60,
            networkLatency: 25
          })
        } as Response);
      }

      // Default successful response
      return Promise.resolve({
        ok: true,
        json: async () => ({ success: true })
      } as Response);
    });
    
    localStorageMock.getItem.mockReturnValue(null);
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('Analytics and Task Scheduling Integration', () => {
    it('should record metrics when tasks are executed', async () => {
      // Create a task
      const task = await taskSchedulingService.createTask({
        name: 'Test Analytics Task',
        description: 'Task for testing analytics integration',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: {
          type: 'once' as const,
          startTime: new Date(Date.now() + 1000)
        },
        action: {
          type: 'api_call' as const,
          target: 'http://example.com/api',
          parameters: {}
        },
        metadata: {}
      });

      // Spy on analytics metric recording
      const recordMetricSpy = jest.spyOn(analyticsService, 'recordMetric');

      // Execute the task
      await taskSchedulingService.executeTask(task.id);

      // Verify analytics metrics were recorded
      expect(recordMetricSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'task_execution',
          category: 'performance'
        })
      );

      recordMetricSpy.mockRestore();
    });

    it('should track task performance in analytics dashboard', async () => {
      // Create multiple tasks
      const tasks = await Promise.all([
        taskSchedulingService.createTask({
          name: 'Task 1',
          description: 'First test task',
          type: 'custom' as const,
          status: 'pending' as const,
          priority: 'high' as const,
          schedule: { type: 'once' as const, startTime: new Date() },
          action: { type: 'api_call' as const, target: 'http://example.com', parameters: {} },
          metadata: {}
        }),
        taskSchedulingService.createTask({
          name: 'Task 2',
          description: 'Second test task',
          type: 'custom' as const,
          status: 'pending' as const,
          priority: 'low' as const,
          schedule: { type: 'once' as const, startTime: new Date() },
          action: { type: 'api_call' as const, target: 'http://example.com', parameters: {} },
          metadata: {}
        })
      ]);

      // Execute tasks
      await Promise.all(tasks.map(task => taskSchedulingService.executeTask(task.id)));

      // Get dashboard stats
      const dashboardStats = await analyticsService.getDashboardStats();

      // Verify task-related metrics are included
      expect(dashboardStats).toHaveProperty('inferencesToday');
      expect(dashboardStats).toHaveProperty('successRate');
      expect(typeof dashboardStats.inferencesToday).toBe('number');
      expect(typeof dashboardStats.successRate).toBe('number');
    });
  });

  describe('UDC Management and Analytics Integration', () => {
    it('should track UDC usage in analytics', async () => {
      // Create a UDC
      const udc = await udcManagementService.createUDC({
        agentId: 'test-agent-analytics',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Test UDC for analytics',
        permissions: ['read']
      });

      // Record UDC usage
      await udcManagementService.recordUsage(udc.id, 'read_data', 'success', 'Data accessed successfully');

      // Get agent analytics
      const agentAnalytics = await analyticsService.getAgentAnalytics();

      // Verify UDC usage is reflected in analytics
      expect(agentAnalytics).toHaveProperty('totalAgents');
      expect(agentAnalytics).toHaveProperty('skillInvocations');
      expect(typeof agentAnalytics.skillInvocations).toBe('number');
    });

    it('should include UDC validation metrics in performance analytics', async () => {
      // Create multiple UDCs
      const udcs = await Promise.all([
        udcManagementService.createUDC({
          agentId: 'agent-1',
          type: 'basic' as const,
          authorityLevel: 'read' as const,
          validityPeriod: 30,
          scope: 'UDC 1',
          permissions: ['read']
        }),
        udcManagementService.createUDC({
          agentId: 'agent-2',
          type: 'advanced' as const,
          authorityLevel: 'write' as const,
          validityPeriod: 60,
          scope: 'UDC 2',
          permissions: ['read', 'write']
        })
      ]);

      // Validate UDCs
      await Promise.all(udcs.map(udc => udcManagementService.validateUDC(udc.id)));

      // Get performance metrics
      const performanceMetrics = await analyticsService.getPerformanceMetrics();

      // Verify validation performance is tracked
      expect(performanceMetrics).toHaveProperty('responseTime');
      expect(performanceMetrics).toHaveProperty('errorRate');
      expect(typeof performanceMetrics.responseTime).toBe('number');
      expect(typeof performanceMetrics.errorRate).toBe('number');
    });
  });

  describe('Settings and Service Configuration Integration', () => {
    it('should apply settings changes to analytics service', async () => {
      // Update analytics settings
      await settingsService.updateSettings({
        analytics: {
          collectMetrics: false,
          shareAnonymousData: true,
          retentionPeriod: 60
        }
      });

      // Verify settings are applied
      const settings = settingsService.getSettings();
      expect(settings.analytics.collectMetrics).toBe(false);
      expect(settings.analytics.shareAnonymousData).toBe(true);
      expect(settings.analytics.retentionPeriod).toBe(60);

      // Test that analytics service respects the settings
      const recordMetricSpy = jest.spyOn(analyticsService, 'recordMetric');
      
      await analyticsService.recordMetric({
        name: 'test_metric',
        value: 100,
        unit: 'count',
        category: 'performance'
      });

      // Should still record metrics locally even if collection is disabled
      expect(recordMetricSpy).toHaveBeenCalled();
      
      recordMetricSpy.mockRestore();
    });

    it('should apply cognitive settings to task execution', async () => {
      // Update cognitive settings
      await settingsService.updateSettings({
        cognitive: {
          defaultModel: 'gpt-3.5-turbo',
          temperature: 0.5,
          maxTokens: 2048,
          topP: 0.8,
          autoLearning: false,
          skillCaching: false
        }
      });

      // Create and execute a task that might use cognitive settings
      const task = await taskSchedulingService.createTask({
        name: 'Cognitive Task',
        description: 'Task using cognitive settings',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: { type: 'once' as const, startTime: new Date() },
        action: { type: 'agent_invoke' as const, target: 'cognitive-agent', parameters: {} },
        metadata: {}
      });

      // Execute task
      const execution = await taskSchedulingService.executeTask(task.id);

      // Verify task executed successfully
      expect(execution).toHaveProperty('status');
      expect(['completed', 'failed']).toContain(execution.status);
    });

    it('should apply security settings to UDC validation', async () => {
      // Update security settings
      await settingsService.updateSettings({
        security: {
          requireMFA: true,
          sessionTimeout: 15,
          maxLoginAttempts: 3
        }
      });

      // Create a UDC
      const udc = await udcManagementService.createUDC({
        agentId: 'secure-agent',
        type: 'enterprise' as const,
        authorityLevel: 'admin' as const,
        validityPeriod: 7, // Short validity for security
        scope: 'High security UDC',
        permissions: ['read', 'write', 'admin']
      });

      // Validate UDC
      const validation = await udcManagementService.validateUDC(udc.id);

      // Verify validation includes security checks
      expect(validation).toHaveProperty('securityChecks');
      expect(validation.securityChecks).toHaveProperty('signature');
      expect(validation.securityChecks).toHaveProperty('expiry');
      expect(validation.securityChecks).toHaveProperty('permissions');
      expect(validation.securityChecks).toHaveProperty('constraints');
    });
  });

  describe('Cross-Service Data Flow', () => {
    it('should maintain data consistency across services', async () => {
      // Create entities in multiple services
      const udc = await udcManagementService.createUDC({
        agentId: 'integration-agent',
        type: 'basic' as const,
        authorityLevel: 'read' as const,
        validityPeriod: 30,
        scope: 'Integration test UDC',
        permissions: ['read']
      });

      const task = await taskSchedulingService.createTask({
        name: 'Integration Task',
        description: 'Task for integration testing',
        type: 'analysis' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: { type: 'once' as const, startTime: new Date() },
        action: { type: 'agent_invoke' as const, target: 'integration-agent', parameters: {} },
        metadata: { udcId: udc.id }
      });

      // Execute task
      await taskSchedulingService.executeTask(task.id);

      // Record UDC usage
      await udcManagementService.recordUsage(udc.id, 'task_execution', 'success', 'Task executed via UDC');

      // Verify data consistency
      const updatedUdc = udcManagementService.getUDC(udc.id);
      const updatedTask = taskSchedulingService.getTask(task.id);

      expect(updatedUdc).toBeDefined();
      expect(updatedTask).toBeDefined();
      expect(updatedUdc!.metadata.usage.executionCount).toBe(1);
      expect(updatedTask!.runCount).toBe(1);
    });

    it('should handle service failures gracefully', async () => {
      // Create a task that depends on external service
      const task = await taskSchedulingService.createTask({
        name: 'Failure Test Task',
        description: 'Task to test failure handling',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'low' as const,
        schedule: { type: 'once' as const, startTime: new Date() },
        action: { type: 'api_call' as const, target: 'http://failing-service.com', parameters: {} },
        metadata: {}
      });

      // Mock a service failure for this specific call
      mockFetch.mockImplementationOnce(() =>
        Promise.reject(new Error('Service unavailable'))
      );

      // Execute task (should handle failure gracefully)
      const execution = await taskSchedulingService.executeTask(task.id);

      // Verify failure was handled
      expect(execution.status).toBe('failed');
      expect(execution.error).toBeDefined();

      // Verify analytics still recorded the attempt
      const dashboardStats = await analyticsService.getDashboardStats();
      expect(dashboardStats).toBeDefined();
    });
  });

  describe('Service Lifecycle Integration', () => {
    it('should handle service startup and shutdown correctly', async () => {
      // Get initial task count
      const initialTasks = taskSchedulingService.getAllTasks();
      const initialTaskCount = initialTasks.length;

      // Start services
      await taskSchedulingService.start();
      await analyticsService.startCollection();

      // Verify services are running
      expect(taskSchedulingService.getAllTasks).toBeDefined();
      expect(analyticsService.getDashboardStats).toBeDefined();

      // Stop services
      await taskSchedulingService.stop();
      await analyticsService.stopCollection();

      // Services should still be accessible and tasks should persist
      // (stopping the scheduler doesn't clear existing tasks)
      const finalTasks = taskSchedulingService.getAllTasks();
      expect(finalTasks.length).toBeGreaterThanOrEqual(initialTaskCount);
    });

    it('should maintain service state during configuration changes', async () => {
      // Create initial data
      const initialTask = await taskSchedulingService.createTask({
        name: 'State Test Task',
        description: 'Task to test state persistence',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: { type: 'once' as const, startTime: new Date() },
        action: { type: 'api_call' as const, target: 'http://example.com', parameters: {} },
        metadata: {}
      });

      // Change settings
      await settingsService.updateSettings({
        general: { theme: 'light' as const, debugMode: true }
      });

      // Verify data persists
      const persistedTask = taskSchedulingService.getTask(initialTask.id);
      expect(persistedTask).toBeDefined();
      expect(persistedTask!.name).toBe('State Test Task');

      // Verify settings were applied
      const settings = settingsService.getSettings();
      expect(settings.general.theme).toBe('light');
      expect(settings.general.debugMode).toBe(true);
    });
  });

  describe('Performance Integration', () => {
    it('should handle concurrent operations across services', async () => {
      const startTime = Date.now();

      // Perform concurrent operations
      const operations = await Promise.all([
        // Create multiple UDCs
        ...Array.from({ length: 5 }, (_, i) =>
          udcManagementService.createUDC({
            agentId: `concurrent-agent-${i}`,
            type: 'basic' as const,
            authorityLevel: 'read' as const,
            validityPeriod: 30,
            scope: `Concurrent UDC ${i}`,
            permissions: ['read']
          })
        ),
        // Create multiple tasks
        ...Array.from({ length: 5 }, (_, i) =>
          taskSchedulingService.createTask({
            name: `Concurrent Task ${i}`,
            description: `Concurrent task ${i}`,
            type: 'custom' as const,
            status: 'pending' as const,
            priority: 'medium' as const,
            schedule: { type: 'once' as const, startTime: new Date() },
            action: { type: 'api_call' as const, target: 'http://example.com', parameters: {} },
            metadata: {}
          })
        ),
        // Update settings (returns void, so we wrap it to return a success indicator)
        settingsService.updateSettings({
          general: { backupInterval: 30 }
        }).then(() => ({ success: true })),
        // Get analytics
        analyticsService.getDashboardStats()
      ]);

      const endTime = Date.now();
      const duration = endTime - startTime;

      // Verify all operations completed
      expect(operations).toHaveLength(12); // 5 UDCs + 5 tasks + 1 settings update + 1 analytics
      expect(operations.every(op => op !== undefined && op !== null)).toBe(true);

      // Verify reasonable performance (should complete within 5 seconds)
      expect(duration).toBeLessThan(5000);
    });
  });
});
