/**
 * Task Scheduling Service
 * Handles automated workflow management, task scheduling, and execution
 */

import { analyticsService } from './AnalyticsService';

export interface ScheduledTask {
  id: string;
  name: string;
  description: string;
  type: 'agent_deployment' | 'data_sync' | 'maintenance' | 'backup' | 'analysis' | 'custom';
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  priority: 'low' | 'medium' | 'high' | 'critical';
  schedule: TaskSchedule;
  action: TaskAction;
  metadata: Record<string, unknown>;
  createdAt: Date;
  lastRun?: Date;
  nextRun?: Date;
  runCount: number;
  successCount: number;
  failureCount: number;
}

export interface TaskSchedule {
  type: 'once' | 'recurring' | 'cron';
  startTime: Date;
  endTime?: Date;
  interval?: number; // in milliseconds for recurring tasks
  cronExpression?: string; // for cron-based scheduling
  timezone?: string;
}

export interface TaskAction {
  type: 'api_call' | 'agent_invoke' | 'system_command' | 'workflow';
  target: string;
  parameters: Record<string, unknown>;
  timeout?: number;
  retryCount?: number;
  retryDelay?: number;
}

export interface TaskExecution {
  id: string;
  taskId: string;
  startTime: Date;
  endTime?: Date;
  status: 'running' | 'completed' | 'failed' | 'timeout';
  result?: unknown;
  error?: string;
  logs: string[];
  duration?: number;
}

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  steps: WorkflowStep[];
  variables: Record<string, unknown>;
  createdAt: Date;
}

export interface WorkflowStep {
  id: string;
  name: string;
  type: 'task' | 'condition' | 'loop' | 'parallel';
  action?: TaskAction;
  condition?: string;
  dependencies: string[];
  onSuccess?: string;
  onFailure?: string;
}

export class TaskSchedulingService {
  private tasks: Map<string, ScheduledTask> = new Map();
  private executions: Map<string, TaskExecution> = new Map();
  private workflows: Map<string, WorkflowTemplate> = new Map();
  private activeTimers: Map<string, NodeJS.Timeout> = new Map();
  private baseUrl: string;
  private isRunning: boolean = false;

  constructor(baseUrl: string = 'http://localhost:3001') {
    this.baseUrl = baseUrl;
    this.initializeScheduler();
  }

  private initializeScheduler(): void {
    console.log('Task Scheduling Service initialized');
  }

  /**
   * Start the task scheduler
   */
  async start(): Promise<void> {
    if (this.isRunning) {
      return;
    }

    try {
      const response = await fetch(`${this.baseUrl}/api/scheduler/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        throw new Error(`Failed to start scheduler: ${response.statusText}`);
      }

      this.isRunning = true;
      this.scheduleAllTasks();
      console.log('Task Scheduler started');
    } catch (error) {
      console.error('Failed to start task scheduler:', error);
      throw error;
    }
  }

  /**
   * Stop the task scheduler
   */
  async stop(): Promise<void> {
    if (!this.isRunning) {
      return;
    }

    try {
      await fetch(`${this.baseUrl}/api/scheduler/stop`, {
        method: 'POST'
      });

      this.isRunning = false;
      this.clearAllTimers();
      console.log('Task Scheduler stopped');
    } catch (error) {
      console.error('Failed to stop task scheduler:', error);
    }
  }

  /**
   * Create a new scheduled task
   */
  async createTask(taskData: Omit<ScheduledTask, 'id' | 'createdAt' | 'runCount' | 'successCount' | 'failureCount'>): Promise<ScheduledTask> {
    const task: ScheduledTask = {
      ...taskData,
      id: this.generateTaskId(),
      createdAt: new Date(),
      runCount: 0,
      successCount: 0,
      failureCount: 0
    };

    // Calculate next run time
    task.nextRun = this.calculateNextRun(task.schedule);

    // Store task
    this.tasks.set(task.id, task);

    // Schedule task if scheduler is running
    if (this.isRunning) {
      this.scheduleTask(task);
    }

    // Send to backend
    try {
      await fetch(`${this.baseUrl}/api/scheduler/tasks`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(task)
      });
    } catch (error) {
      console.error('Failed to save task to backend:', error);
    }

    return task;
  }

  /**
   * Update an existing task
   */
  async updateTask(taskId: string, updates: Partial<ScheduledTask>): Promise<ScheduledTask> {
    const task = this.tasks.get(taskId);
    if (!task) {
      throw new Error(`Task ${taskId} not found`);
    }

    const updatedTask = { ...task, ...updates };
    
    // Recalculate next run if schedule changed
    if (updates.schedule) {
      updatedTask.nextRun = this.calculateNextRun(updatedTask.schedule);
    }

    this.tasks.set(taskId, updatedTask);

    // Reschedule task
    if (this.isRunning) {
      this.clearTaskTimer(taskId);
      this.scheduleTask(updatedTask);
    }

    // Send to backend
    try {
      await fetch(`${this.baseUrl}/api/scheduler/tasks/${taskId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updatedTask)
      });
    } catch (error) {
      console.error('Failed to update task in backend:', error);
    }

    return updatedTask;
  }

  /**
   * Delete a task
   */
  async deleteTask(taskId: string): Promise<void> {
    const task = this.tasks.get(taskId);
    if (!task) {
      throw new Error(`Task ${taskId} not found`);
    }

    // Clear timer and remove task
    this.clearTaskTimer(taskId);
    this.tasks.delete(taskId);

    // Send to backend
    try {
      await fetch(`${this.baseUrl}/api/scheduler/tasks/${taskId}`, {
        method: 'DELETE'
      });
    } catch (error) {
      console.error('Failed to delete task from backend:', error);
    }
  }

  /**
   * Execute a task immediately
   */
  async executeTask(taskId: string): Promise<TaskExecution> {
    const task = this.tasks.get(taskId);
    if (!task) {
      throw new Error(`Task ${taskId} not found`);
    }

    const execution: TaskExecution = {
      id: this.generateExecutionId(),
      taskId,
      startTime: new Date(),
      status: 'running',
      logs: []
    };

    this.executions.set(execution.id, execution);

    try {
      execution.logs.push(`Starting task execution: ${task.name}`);
      
      // Execute the task action
      const result = await this.executeTaskAction(task.action);
      
      execution.endTime = new Date();
      execution.duration = execution.endTime.getTime() - execution.startTime.getTime();
      execution.status = 'completed';
      execution.result = result;
      execution.logs.push(`Task completed successfully in ${execution.duration}ms`);

      // Update task statistics
      task.runCount++;
      task.successCount++;
      task.lastRun = execution.startTime;

      // Record analytics metric for task execution
      await analyticsService.recordMetric({
        name: 'task_execution',
        value: execution.duration || 0,
        unit: 'milliseconds',
        category: 'performance',
        metadata: {
          taskId: task.id,
          taskName: task.name,
          taskType: task.type,
          status: execution.status
        }
      });
      
      // Calculate next run for recurring tasks
      if (task.schedule.type === 'recurring' || task.schedule.type === 'cron') {
        task.nextRun = this.calculateNextRun(task.schedule);
        this.scheduleTask(task);
      }

    } catch (error) {
      execution.endTime = new Date();
      execution.duration = execution.endTime.getTime() - execution.startTime.getTime();
      execution.status = 'failed';
      execution.error = error instanceof Error ? error.message : 'Unknown error';
      execution.logs.push(`Task failed: ${execution.error}`);

      // Update task statistics
      task.runCount++;
      task.failureCount++;
      task.lastRun = execution.startTime;

      // Record analytics metric for failed task execution
      await analyticsService.recordMetric({
        name: 'task_execution',
        value: execution.duration || 0,
        unit: 'milliseconds',
        category: 'performance',
        metadata: {
          taskId: task.id,
          taskName: task.name,
          taskType: task.type,
          status: execution.status,
          error: execution.error
        }
      });
    }

    this.executions.set(execution.id, execution);
    this.tasks.set(taskId, task);

    return execution;
  }

  /**
   * Get all tasks
   */
  getAllTasks(): ScheduledTask[] {
    return Array.from(this.tasks.values());
  }

  /**
   * Get task by ID
   */
  getTask(taskId: string): ScheduledTask | undefined {
    return this.tasks.get(taskId);
  }

  /**
   * Get tasks by status
   */
  getTasksByStatus(status: ScheduledTask['status']): ScheduledTask[] {
    return Array.from(this.tasks.values()).filter(task => task.status === status);
  }

  /**
   * Get task executions
   */
  getTaskExecutions(taskId: string): TaskExecution[] {
    return Array.from(this.executions.values()).filter(exec => exec.taskId === taskId);
  }

  /**
   * Create workflow template
   */
  async createWorkflow(workflow: Omit<WorkflowTemplate, 'id' | 'createdAt'>): Promise<WorkflowTemplate> {
    const fullWorkflow: WorkflowTemplate = {
      ...workflow,
      id: this.generateWorkflowId(),
      createdAt: new Date()
    };

    this.workflows.set(fullWorkflow.id, fullWorkflow);

    try {
      await fetch(`${this.baseUrl}/api/scheduler/workflows`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(fullWorkflow)
      });
    } catch (error) {
      console.error('Failed to save workflow to backend:', error);
    }

    return fullWorkflow;
  }

  /**
   * Execute workflow
   */
  async executeWorkflow(workflowId: string, variables: Record<string, unknown> = {}): Promise<string> {
    const workflow = this.workflows.get(workflowId);
    if (!workflow) {
      throw new Error(`Workflow ${workflowId} not found`);
    }

    // Create a task for the workflow execution
    const task = await this.createTask({
      name: `Workflow: ${workflow.name}`,
      description: `Executing workflow: ${workflow.description}`,
      type: 'custom',
      status: 'pending',
      priority: 'medium',
      schedule: {
        type: 'once',
        startTime: new Date()
      },
      action: {
        type: 'workflow',
        target: workflowId,
        parameters: { ...workflow.variables, ...variables }
      },
      metadata: {
        workflowId,
        isWorkflowExecution: true
      }
    });

    // Execute immediately
    await this.executeTask(task.id);
    
    return task.id;
  }

  private scheduleAllTasks(): void {
    for (const task of this.tasks.values()) {
      if (task.status === 'pending' && task.nextRun) {
        this.scheduleTask(task);
      }
    }
  }

  private scheduleTask(task: ScheduledTask): void {
    if (!task.nextRun) {
      return;
    }

    const delay = task.nextRun.getTime() - Date.now();
    
    if (delay <= 0) {
      // Execute immediately if scheduled time has passed
      this.executeTask(task.id);
    } else {
      // Schedule for future execution
      const timer = setTimeout(() => {
        this.executeTask(task.id);
      }, delay);
      
      this.activeTimers.set(task.id, timer);
    }
  }

  private clearTaskTimer(taskId: string): void {
    const timer = this.activeTimers.get(taskId);
    if (timer) {
      clearTimeout(timer);
      this.activeTimers.delete(taskId);
    }
  }

  private clearAllTimers(): void {
    for (const timer of this.activeTimers.values()) {
      clearTimeout(timer);
    }
    this.activeTimers.clear();
  }

  private calculateNextRun(schedule: TaskSchedule): Date | undefined {
    const now = new Date();
    
    switch (schedule.type) {
      case 'once':
        return schedule.startTime > now ? schedule.startTime : undefined;
      
      case 'recurring': {
        if (!schedule.interval) return undefined;

        let nextRun = new Date(schedule.startTime);
        while (nextRun <= now) {
          nextRun = new Date(nextRun.getTime() + schedule.interval);
        }
        
        if (schedule.endTime && nextRun > schedule.endTime) {
          return undefined;
        }
        
        return nextRun;
      }
      case 'cron':
        // Simple cron implementation - would use a proper cron library in production
        return this.calculateCronNextRun(schedule.cronExpression!, now);
      
      default:
        return undefined;
    }
  }

  private calculateCronNextRun(cronExpression: string, from: Date): Date {
    // Simplified cron calculation - would use a proper library like node-cron
    // For now, just add 1 hour as a placeholder
    return new Date(from.getTime() + 60 * 60 * 1000);
  }

  private async executeTaskAction(action: TaskAction): Promise<unknown> {
    switch (action.type) {
      case 'api_call':
        return await this.executeApiCall(action);
      case 'agent_invoke':
        return await this.executeAgentInvoke(action);
      case 'system_command':
        return await this.executeSystemCommand(action);
      case 'workflow':
        return await this.executeWorkflowAction(action);
      
      default:
        throw new Error(`Unknown action type: ${action.type}`);
    }
  }

  private async executeApiCall(action: TaskAction): Promise<unknown> {
    const params = action.parameters as {
      method?: string;
      headers?: Record<string, string>;
      body?: unknown;
    };

    const response = await fetch(action.target, {
      method: params.method || 'GET',
      headers: params.headers || {},
      body: params.body ? JSON.stringify(params.body) : undefined
    });

    if (!response.ok) {
      throw new Error(`API call failed: ${response.statusText}`);
    }

    return await response.json();
  }

  private async executeAgentInvoke(action: TaskAction): Promise<unknown> {
    // Integrate with AgentManagementService
    const response = await fetch(`${this.baseUrl}/api/agents/${action.target}/invoke`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(action.parameters)
    });

    if (!response.ok) {
      throw new Error(`Agent invocation failed: ${response.statusText}`);
    }

    return await response.json();
  }

  private async executeSystemCommand(action: TaskAction): Promise<unknown> {
    // Integrate with TerminalCommandService
    const response = await fetch(`${this.baseUrl}/api/terminal/execute`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        command: action.target,
        args: action.parameters.args || []
      })
    });

    if (!response.ok) {
      throw new Error(`System command failed: ${response.statusText}`);
    }

    return await response.json();
  }

  private async executeWorkflowAction(action: TaskAction): Promise<unknown> {
    const workflow = this.workflows.get(action.target);
    if (!workflow) {
      throw new Error(`Workflow ${action.target} not found`);
    }

    // Execute workflow steps sequentially
    const results: unknown[] = [];
    
    for (const step of workflow.steps) {
      if (step.action) {
        const result = await this.executeTaskAction(step.action);
        results.push(result);
      }
    }

    return results;
  }

  private generateTaskId(): string {
    return `task_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateExecutionId(): string {
    return `exec_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateWorkflowId(): string {
    return `workflow_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Export singleton instance
export const taskSchedulingService = new TaskSchedulingService();
