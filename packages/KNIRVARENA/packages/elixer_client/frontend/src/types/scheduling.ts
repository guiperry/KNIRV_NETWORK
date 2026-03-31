// Task scheduling types and interfaces
export interface TaskSchedule {
  type: 'once' | 'recurring' | 'cron';
  startTime: Date;
  endTime?: Date;
  interval?: number; // in milliseconds for recurring tasks
  cronExpression?: string; // for cron-based scheduling
  timezone?: string;
  maxExecutions?: number;
  retryPolicy?: {
    maxRetries: number;
    retryDelay: number;
    backoffMultiplier: number;
  };
}

export interface ScheduledTask {
  id: string;
  name: string;
  description: string;
  type: 'agent_deployment' | 'data_sync' | 'maintenance' | 'backup' | 'analysis' | 'custom';
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'paused';
  priority: 'low' | 'medium' | 'high' | 'critical';
  schedule: TaskSchedule;
  action: {
    type: 'function' | 'api_call' | 'script' | 'workflow';
    target: string;
    parameters: Record<string, unknown>;
    timeout?: number;
  };
  dependencies?: string[]; // IDs of tasks that must complete first
  createdAt: Date;
  updatedAt: Date;
  lastExecuted?: Date;
  nextExecution?: Date;
  runCount: number;
  successCount: number;
  failureCount: number;
  averageExecutionTime?: number;
  metadata: Record<string, unknown>;
}

export interface TaskExecution {
  id: string;
  taskId: string;
  status: 'running' | 'completed' | 'failed' | 'cancelled';
  startTime: Date;
  endTime?: Date;
  duration?: number;
  result?: unknown;
  error?: {
    code: string;
    message: string;
    stack?: string;
    details?: Record<string, unknown>;
  };
  logs: Array<{
    level: 'debug' | 'info' | 'warn' | 'error';
    message: string;
    timestamp: Date;
    metadata?: Record<string, unknown>;
  }>;
  metrics?: {
    cpuUsage: number;
    memoryUsage: number;
    networkIO: number;
    diskIO: number;
  };
}

export interface TaskQueue {
  id: string;
  name: string;
  description: string;
  maxConcurrency: number;
  priority: number;
  isActive: boolean;
  tasks: string[]; // Task IDs
  currentlyRunning: string[];
  statistics: {
    totalTasks: number;
    completedTasks: number;
    failedTasks: number;
    averageWaitTime: number;
    averageExecutionTime: number;
  };
  createdAt: Date;
  updatedAt: Date;
}

export interface TaskScheduler {
  id: string;
  name: string;
  isActive: boolean;
  queues: string[]; // Queue IDs
  config: {
    maxConcurrentTasks: number;
    taskTimeout: number;
    retryPolicy: {
      maxRetries: number;
      retryDelay: number;
      backoffMultiplier: number;
    };
    cleanupPolicy: {
      retainCompletedTasks: number; // days
      retainFailedTasks: number; // days
      maxLogEntries: number;
    };
  };
  statistics: {
    totalTasksScheduled: number;
    totalTasksCompleted: number;
    totalTasksFailed: number;
    averageExecutionTime: number;
    uptime: number;
  };
  createdAt: Date;
  lastActivity: Date;
}

export interface TaskTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  type: ScheduledTask['type'];
  defaultSchedule: Partial<TaskSchedule>;
  defaultAction: Partial<ScheduledTask['action']>;
  parameters: Array<{
    name: string;
    type: 'string' | 'number' | 'boolean' | 'object' | 'array';
    required: boolean;
    default?: unknown;
    description: string;
    validation?: {
      min?: number;
      max?: number;
      pattern?: string;
      enum?: unknown[];
    };
  }>;
  tags: string[];
  version: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface TaskNotification {
  id: string;
  taskId: string;
  type: 'task_started' | 'task_completed' | 'task_failed' | 'task_cancelled' | 'task_delayed';
  title: string;
  message: string;
  severity: 'info' | 'warning' | 'error' | 'success';
  timestamp: Date;
  isRead: boolean;
  metadata?: Record<string, unknown>;
}
