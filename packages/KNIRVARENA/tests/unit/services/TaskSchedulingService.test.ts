/**
 * TaskSchedulingService Unit Tests
 * Comprehensive test suite for task scheduling functionality
 */

import { taskSchedulingService, TaskSchedulingService, ScheduledTask, TaskAction } from '../../../src/services/TaskSchedulingService';

// Mock fetch globally
global.fetch = jest.fn();

describe('TaskSchedulingService', () => {
  let service: TaskSchedulingService;
  const mockFetch = global.fetch as jest.MockedFunction<typeof fetch>;

  beforeEach(() => {
    service = new TaskSchedulingService('http://localhost:3001');
    mockFetch.mockClear();
    jest.clearAllTimers();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
    mockFetch.mockRestore();
  });

  describe('Initialization', () => {
    it('should initialize with default settings', () => {
      expect(service).toBeDefined();
      expect(service.getAllTasks).toBeDefined();
      expect(service.createTask).toBeDefined();
    });
  });

  describe('Scheduler Management', () => {
    it('should start scheduler successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      await service.start();
      
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:3001/api/scheduler/start',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' }
        })
      );
    });

    it('should handle start failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        statusText: 'Server Error'
      } as Response);

      await expect(service.start()).rejects.toThrow('Failed to start scheduler: Server Error');
    });

    it('should stop scheduler successfully', async () => {
      // First start the scheduler
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);
      await service.start();

      // Then stop it
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      await service.stop();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:3001/api/scheduler/stop',
        expect.objectContaining({
          method: 'POST'
        })
      );
    });

    it('should not start if already running', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      await service.start();
      mockFetch.mockClear();
      
      await service.start(); // Second call
      expect(mockFetch).not.toHaveBeenCalled();
    });
  });

  describe('Task Creation', () => {
    it('should create a new task successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const taskData = {
        name: 'Test Task',
        description: 'A test task',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: {
          type: 'once' as const,
          startTime: new Date(Date.now() + 60000) // 1 minute from now
        },
        action: {
          type: 'api_call' as const,
          target: 'http://example.com/api',
          parameters: { method: 'GET' }
        },
        metadata: {}
      };

      const task = await service.createTask(taskData);
      
      expect(task).toHaveProperty('id');
      expect(task).toHaveProperty('createdAt');
      expect(task).toHaveProperty('runCount', 0);
      expect(task).toHaveProperty('successCount', 0);
      expect(task).toHaveProperty('failureCount', 0);
      expect(task.name).toBe('Test Task');
      expect(task.description).toBe('A test task');
      expect(task.type).toBe('custom');
      expect(task.priority).toBe('medium');
    });

    it('should calculate next run time for once tasks', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const futureTime = new Date(Date.now() + 60000);
      const taskData = {
        name: 'Once Task',
        description: 'A one-time task',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: {
          type: 'once' as const,
          startTime: futureTime
        },
        action: {
          type: 'api_call' as const,
          target: 'http://example.com/api',
          parameters: {}
        },
        metadata: {}
      };

      const task = await service.createTask(taskData);
      
      expect(task.nextRun).toEqual(futureTime);
    });

    it('should calculate next run time for recurring tasks', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const startTime = new Date(Date.now() - 30000); // 30 seconds ago
      const taskData = {
        name: 'Recurring Task',
        description: 'A recurring task',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: {
          type: 'recurring' as const,
          startTime: startTime,
          interval: 60000 // 1 minute
        },
        action: {
          type: 'api_call' as const,
          target: 'http://example.com/api',
          parameters: {}
        },
        metadata: {}
      };

      const task = await service.createTask(taskData);
      
      expect(task.nextRun).toBeDefined();
      expect(task.nextRun!.getTime()).toBeGreaterThan(Date.now());
    });
  });

  describe('Task Management', () => {
    let testTask: ScheduledTask;

    beforeEach(async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      testTask = await service.createTask({
        name: 'Test Task',
        description: 'A test task',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: {
          type: 'once' as const,
          startTime: new Date(Date.now() + 60000)
        },
        action: {
          type: 'api_call' as const,
          target: 'http://example.com/api',
          parameters: {}
        },
        metadata: {}
      });
    });

    it('should update task successfully', async () => {
      const updates = {
        name: 'Updated Task',
        priority: 'high' as const
      };

      const updatedTask = await service.updateTask(testTask.id, updates);
      
      expect(updatedTask.name).toBe('Updated Task');
      expect(updatedTask.priority).toBe('high');
    });

    it('should delete task successfully', async () => {
      await service.deleteTask(testTask.id);
      
      const task = service.getTask(testTask.id);
      expect(task).toBeUndefined();
    });

    it('should throw error when updating non-existent task', async () => {
      await expect(service.updateTask('non-existent-id', {}))
        .rejects.toThrow('Task non-existent-id not found');
    });

    it('should throw error when deleting non-existent task', async () => {
      await expect(service.deleteTask('non-existent-id'))
        .rejects.toThrow('Task non-existent-id not found');
    });
  });

  describe('Task Execution', () => {
    let testTask: ScheduledTask;

    beforeEach(async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true, result: 'API call successful' })
      } as Response);

      testTask = await service.createTask({
        name: 'Executable Task',
        description: 'A task for execution testing',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: {
          type: 'once' as const,
          startTime: new Date(Date.now() + 60000)
        },
        action: {
          type: 'api_call' as const,
          target: 'http://example.com/api',
          parameters: { method: 'GET' }
        },
        metadata: {}
      });
    });

    it('should execute task successfully', async () => {
      const execution = await service.executeTask(testTask.id);
      
      expect(execution).toHaveProperty('id');
      expect(execution).toHaveProperty('taskId', testTask.id);
      expect(execution).toHaveProperty('startTime');
      expect(execution).toHaveProperty('endTime');
      expect(execution).toHaveProperty('status', 'completed');
      expect(execution).toHaveProperty('result');
      expect(execution).toHaveProperty('duration');
      expect(execution.logs).toContain('Starting task execution: Executable Task');
    });

    it('should handle task execution failure', async () => {
      // Mock API call to fail
      mockFetch.mockRejectedValueOnce(new Error('API Error'));

      const execution = await service.executeTask(testTask.id);
      
      expect(execution.status).toBe('failed');
      expect(execution.error).toBe('API Error');
      expect(execution.logs).toContain('Task failed: API Error');
    });

    it('should update task statistics after execution', async () => {
      await service.executeTask(testTask.id);
      
      const updatedTask = service.getTask(testTask.id);
      expect(updatedTask!.runCount).toBe(1);
      expect(updatedTask!.successCount).toBe(1);
      expect(updatedTask!.failureCount).toBe(0);
      expect(updatedTask!.lastRun).toBeDefined();
    });

    it('should throw error when executing non-existent task', async () => {
      await expect(service.executeTask('non-existent-id'))
        .rejects.toThrow('Task non-existent-id not found');
    });
  });

  describe('Task Retrieval', () => {
    beforeEach(async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      // Create test tasks
      await service.createTask({
        name: 'Pending Task',
        description: 'A pending task',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'high' as const,
        schedule: { type: 'once' as const, startTime: new Date() },
        action: { type: 'api_call' as const, target: 'http://example.com', parameters: {} },
        metadata: {}
      });

      await service.createTask({
        name: 'Running Task',
        description: 'A running task',
        type: 'agent_deployment' as const,
        status: 'running' as const,
        priority: 'medium' as const,
        schedule: { type: 'once' as const, startTime: new Date() },
        action: { type: 'agent_invoke' as const, target: 'agent-123', parameters: {} },
        metadata: {}
      });
    });

    it('should get all tasks', () => {
      const tasks = service.getAllTasks();
      expect(tasks).toHaveLength(2);
    });

    it('should get tasks by status', () => {
      const pendingTasks = service.getTasksByStatus('pending');
      const runningTasks = service.getTasksByStatus('running');
      
      expect(pendingTasks).toHaveLength(1);
      expect(runningTasks).toHaveLength(1);
      expect(pendingTasks[0].status).toBe('pending');
      expect(runningTasks[0].status).toBe('running');
    });

    it('should get task executions', () => {
      const tasks = service.getAllTasks();
      const executions = service.getTaskExecutions(tasks[0].id);
      
      expect(Array.isArray(executions)).toBe(true);
    });
  });

  describe('Workflow Management', () => {
    it('should create workflow successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const workflowData = {
        name: 'Test Workflow',
        description: 'A test workflow',
        steps: [
          {
            id: 'step1',
            name: 'First Step',
            type: 'task' as const,
            action: {
              type: 'api_call' as const,
              target: 'http://example.com/step1',
              parameters: {}
            },
            dependencies: [],
            onSuccess: 'step2',
            onFailure: 'end'
          }
        ],
        variables: { env: 'test' }
      };

      const workflow = await service.createWorkflow(workflowData);
      
      expect(workflow).toHaveProperty('id');
      expect(workflow).toHaveProperty('createdAt');
      expect(workflow.name).toBe('Test Workflow');
      expect(workflow.steps).toHaveLength(1);
    });

    it('should execute workflow successfully', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const workflow = await service.createWorkflow({
        name: 'Executable Workflow',
        description: 'A workflow for execution testing',
        steps: [],
        variables: {}
      });

      const taskId = await service.executeWorkflow(workflow.id, { customVar: 'value' });
      
      expect(typeof taskId).toBe('string');
    });

    it('should throw error when executing non-existent workflow', async () => {
      await expect(service.executeWorkflow('non-existent-id', {}))
        .rejects.toThrow('Workflow non-existent-id not found');
    });
  });

  describe('Action Execution', () => {
    it('should execute API call action', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ result: 'success' })
      } as Response);

      const action = {
        type: 'api_call' as const,
        target: 'http://example.com/api',
        parameters: { method: 'POST', body: { data: 'test' } }
      };

      // Access private method for testing
      const result = await (service as unknown as { executeTaskAction: (action: TaskAction) => Promise<unknown> }).executeTaskAction(action);
      expect(result).toEqual({ result: 'success' });
    });

    it('should handle API call failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        statusText: 'Bad Request'
      } as Response);

      const action = {
        type: 'api_call' as const,
        target: 'http://example.com/api',
        parameters: {}
      };

      await expect((service as unknown as { executeTaskAction: (action: TaskAction) => Promise<unknown> }).executeTaskAction(action))
        .rejects.toThrow('API call failed: Bad Request');
    });
  });

  describe('Singleton Instance', () => {
    it('should provide a singleton instance', () => {
      expect(taskSchedulingService).toBeDefined();
      expect(taskSchedulingService).toBeInstanceOf(TaskSchedulingService);
    });
  });

  describe('Error Handling', () => {
    it('should handle network errors gracefully', async () => {
      mockFetch.mockRejectedValue(new Error('Network error'));

      // These should not throw but log errors
      await expect(service.start()).rejects.toThrow();
      await expect(service.stop()).resolves.not.toThrow();
    });
  });

  describe('ID Generation', () => {
    it('should generate unique task IDs', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      } as Response);

      const taskData = {
        name: 'Test Task',
        description: 'A test task',
        type: 'custom' as const,
        status: 'pending' as const,
        priority: 'medium' as const,
        schedule: { type: 'once' as const, startTime: new Date() },
        action: { type: 'api_call' as const, target: 'http://example.com', parameters: {} },
        metadata: {}
      };

      const task1 = await service.createTask(taskData);
      const task2 = await service.createTask(taskData);
      
      expect(task1.id).not.toBe(task2.id);
    });
  });
});
