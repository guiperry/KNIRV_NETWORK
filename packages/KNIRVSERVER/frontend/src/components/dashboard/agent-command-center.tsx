'use client';

import React, { useEffect, useState, useCallback } from 'react';
import {
  Bot,
  Play,
  Square,
  Plus,
  RefreshCw,
  CheckCircle,
  AlertCircle,
  Clock,
  Activity,
  Loader2,
  ChevronDown,
  ChevronUp,
  Monitor,
  ListTodo,
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { useToast } from '@/hooks/use-toast';
import { apiRequest, API_BASE_URL } from '@/lib/api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type AgentStatus = 'running' | 'stopped' | 'starting' | 'stopping';

export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed';

export type TaskType = 'research' | 'coding' | 'validation' | 'analysis';

export type TaskPriority = 'low' | 'medium' | 'high' | 'critical';

export interface AgentStatusResponse {
  dve_id: string;
  status: AgentStatus;
  viewport_url?: string;
  started_at?: string;
  task_count?: number;
}

export interface AgentTask {
  id: string;
  dve_id: string;
  title: string;
  description: string;
  type: TaskType;
  priority: TaskPriority;
  status: TaskStatus;
  created_at: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
  error_message?: string;
  result?: Record<string, any>;
}

export interface CreateAgentTaskRequest {
  title: string;
  description: string;
  type: TaskType;
  priority: TaskPriority;
}

interface AgentCommandCenterProps {
  dveId: string;
  className?: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const getTaskStatusColor = (status: TaskStatus): string => {
  switch (status) {
    case 'pending':   return 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/30';
    case 'running':   return 'bg-blue-500/20 text-blue-400 border border-blue-500/30';
    case 'completed': return 'bg-green-500/20 text-green-400 border border-green-500/30';
    case 'failed':    return 'bg-red-500/20 text-red-400 border border-red-500/30';
    default:          return 'bg-muted text-muted-foreground';
  }
};

const getTaskStatusIcon = (status: TaskStatus) => {
  switch (status) {
    case 'pending':   return <Clock className="w-3 h-3" />;
    case 'running':   return <Activity className="w-3 h-3 animate-pulse" />;
    case 'completed': return <CheckCircle className="w-3 h-3" />;
    case 'failed':    return <AlertCircle className="w-3 h-3" />;
    default:          return <Clock className="w-3 h-3" />;
  }
};

const getPriorityColor = (priority: TaskPriority): string => {
  switch (priority) {
    case 'low':      return 'text-slate-400';
    case 'medium':   return 'text-yellow-400';
    case 'high':     return 'text-orange-400';
    case 'critical': return 'text-red-400';
    default:         return 'text-slate-400';
  }
};

const formatRelativeTime = (isoTimestamp: string): string => {
  const date = new Date(isoTimestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
};

// ---------------------------------------------------------------------------
// Default form state
// ---------------------------------------------------------------------------

const defaultFormState: CreateAgentTaskRequest = {
  title: '',
  description: '',
  type: 'research',
  priority: 'medium',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const AgentCommandCenter: React.FC<AgentCommandCenterProps> = ({
  dveId,
  className,
}) => {
  const { toast } = useToast();

  // Agent status state
  const [agentStatus, setAgentStatus] = useState<AgentStatusResponse | null>(null);
  const [isStatusLoading, setIsStatusLoading] = useState(false);
  const [isAgentActionPending, setIsAgentActionPending] = useState(false);

  // Task list state
  const [tasks, setTasks] = useState<AgentTask[]>([]);
  const [isTasksLoading, setIsTasksLoading] = useState(false);
  const [tasksError, setTasksError] = useState<string | null>(null);

  // Task form state
  const [showTaskForm, setShowTaskForm] = useState(false);
  const [formState, setFormState] = useState<CreateAgentTaskRequest>(defaultFormState);
  const [isSubmittingTask, setIsSubmittingTask] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // Viewport state
  const [showViewport, setShowViewport] = useState(false);

  // Selected task detail state
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null);

  // ---------------------------------------------------------------------------
  // API helpers
  // ---------------------------------------------------------------------------

  const fetchAgentStatus = useCallback(async () => {
    setIsStatusLoading(true);
    try {
      const response = await apiRequest<AgentStatusResponse>(
        `${API_BASE_URL}/api/dve/${dveId}/agent/status`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setAgentStatus(response.data);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch agent status';
      console.error('[AgentCommandCenter] fetchAgentStatus error:', err);
      setAgentStatus(prev => prev); // keep stale data
      toast({
        title: 'Status Error',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setIsStatusLoading(false);
    }
  }, [dveId, toast]);

  const fetchTasks = useCallback(async () => {
    setIsTasksLoading(true);
    setTasksError(null);
    try {
      const response = await apiRequest<AgentTask[]>(
        `${API_BASE_URL}/api/dve/${dveId}/agent/tasks`,
        { method: 'GET' }
      );
      if (response.success && response.data) {
        setTasks(Array.isArray(response.data) ? response.data : []);
      } else {
        setTasks([]);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch tasks';
      setTasksError(message);
      console.error('[AgentCommandCenter] fetchTasks error:', err);
    } finally {
      setIsTasksLoading(false);
    }
  }, [dveId]);

  const launchAgent = async () => {
    setIsAgentActionPending(true);
    try {
      const response = await apiRequest<AgentStatusResponse>(
        `${API_BASE_URL}/api/dve/${dveId}/agent/launch`,
        { method: 'POST' }
      );
      if (response.success && response.data) {
        setAgentStatus(response.data);
        toast({
          title: 'Agent Launched',
          description: `Agent for DVE ${dveId} is starting.`,
        });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to launch agent';
      toast({
        title: 'Launch Failed',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setIsAgentActionPending(false);
    }
  };

  const stopAgent = async () => {
    setIsAgentActionPending(true);
    try {
      await apiRequest(
        `${API_BASE_URL}/api/dve/${dveId}/agent`,
        { method: 'DELETE' }
      );
      setAgentStatus(prev => prev ? { ...prev, status: 'stopped', viewport_url: undefined } : null);
      setShowViewport(false);
      toast({
        title: 'Agent Stopped',
        description: `Agent for DVE ${dveId} has been terminated.`,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to stop agent';
      toast({
        title: 'Stop Failed',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setIsAgentActionPending(false);
    }
  };

  const submitTask = async () => {
    if (!formState.title.trim()) {
      setFormError('Task title is required.');
      return;
    }
    setFormError(null);
    setIsSubmittingTask(true);
    try {
      const response = await apiRequest<AgentTask>(
        `${API_BASE_URL}/api/dve/${dveId}/agent/tasks`,
        {
          method: 'POST',
          body: JSON.stringify(formState),
        }
      );
      if (response.success && response.data) {
        setTasks(prev => [response.data!, ...prev]);
        setFormState(defaultFormState);
        setShowTaskForm(false);
        toast({
          title: 'Task Created',
          description: `"${response.data.title}" has been queued.`,
        });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create task';
      setFormError(message);
      toast({
        title: 'Task Creation Failed',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setIsSubmittingTask(false);
    }
  };

  // ---------------------------------------------------------------------------
  // Effects
  // ---------------------------------------------------------------------------

  useEffect(() => {
    fetchAgentStatus();
    fetchTasks();
  }, [fetchAgentStatus, fetchTasks]);

  // ---------------------------------------------------------------------------
  // Derived values
  // ---------------------------------------------------------------------------

  const isAgentRunning = agentStatus?.status === 'running';
  const isAgentStarting = agentStatus?.status === 'starting';
  const isAgentStopping = agentStatus?.status === 'stopping';
  const isAgentBusy = isAgentStarting || isAgentStopping || isAgentActionPending;

  const taskStats = {
    total: tasks.length,
    pending: tasks.filter(t => t.status === 'pending').length,
    running: tasks.filter(t => t.status === 'running').length,
    completed: tasks.filter(t => t.status === 'completed').length,
    failed: tasks.filter(t => t.status === 'failed').length,
  };

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className={`space-y-4 ${className ?? ''}`} data-testid="agent-command-center">
      {/* Header Card — Agent Status + Launch/Stop */}
      <Card className="knirv-card-gradient">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center space-x-2">
              <Bot className="w-5 h-5 text-primary" />
              <span>Agent Command Center</span>
            </CardTitle>
            <div className="flex items-center space-x-2">
              {/* Status badge */}
              <Badge
                variant="secondary"
                className={`text-[10px] font-black uppercase ${
                  isAgentRunning
                    ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                    : isAgentStarting || isAgentStopping
                    ? 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/30'
                    : 'bg-muted text-muted-foreground'
                }`}
                data-testid="agent-status-badge"
              >
                <div className="flex items-center space-x-1">
                  {isAgentRunning ? (
                    <Activity className="w-3 h-3 animate-pulse" />
                  ) : isAgentStarting || isAgentStopping ? (
                    <Loader2 className="w-3 h-3 animate-spin" />
                  ) : (
                    <Square className="w-3 h-3" />
                  )}
                  <span>{agentStatus?.status ?? 'unknown'}</span>
                </div>
              </Badge>

              {/* Refresh */}
              <Button
                variant="outline"
                size="sm"
                onClick={() => { fetchAgentStatus(); fetchTasks(); }}
                disabled={isStatusLoading || isTasksLoading}
                data-testid="refresh-button"
              >
                <RefreshCw className={`w-4 h-4 ${isStatusLoading ? 'animate-spin' : ''}`} />
              </Button>

              {/* Launch / Stop */}
              {isAgentRunning ? (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={stopAgent}
                  disabled={isAgentBusy}
                  data-testid="stop-agent-button"
                  className="font-black uppercase text-[10px]"
                >
                  {isAgentBusy ? (
                    <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                  ) : (
                    <Square className="w-3 h-3 mr-1 fill-current" />
                  )}
                  Stop Agent
                </Button>
              ) : (
                <Button
                  variant="default"
                  size="sm"
                  onClick={launchAgent}
                  disabled={isAgentBusy || isStatusLoading}
                  data-testid="launch-agent-button"
                  className="bg-blue-600 hover:bg-blue-500 text-white shadow-lg shadow-blue-600/20 font-black uppercase text-[10px]"
                >
                  {isAgentBusy ? (
                    <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                  ) : (
                    <Play className="w-3 h-3 mr-1 fill-current" />
                  )}
                  Launch Agent
                </Button>
              )}
            </div>
          </div>
          <CardDescription className="text-[11px] font-mono">
            DVE: {dveId}
            {agentStatus?.started_at && (
              <span className="ml-2 text-slate-500">
                | Started {formatRelativeTime(agentStatus.started_at)}
              </span>
            )}
          </CardDescription>
        </CardHeader>
      </Card>

      {/* Viewport — only shown when agent is running and has a viewport URL */}
      {isAgentRunning && agentStatus?.viewport_url && (
        <Card className="knirv-card-gradient">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center space-x-2 text-sm">
                <Monitor className="w-4 h-4 text-blue-400" />
                <span>Agent Viewport</span>
              </CardTitle>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowViewport(v => !v)}
                data-testid="toggle-viewport-button"
                className="text-[10px] font-black uppercase text-slate-400 hover:text-white"
              >
                {showViewport ? (
                  <>
                    <ChevronUp className="w-3 h-3 mr-1" />
                    Collapse
                  </>
                ) : (
                  <>
                    <ChevronDown className="w-3 h-3 mr-1" />
                    Expand
                  </>
                )}
              </Button>
            </div>
          </CardHeader>
          {showViewport && (
            <CardContent>
              <div className="rounded-lg overflow-hidden border border-slate-700 bg-slate-950">
                <iframe
                  src={agentStatus.viewport_url}
                  title="Agent Viewport"
                  className="w-full"
                  style={{ height: '480px', border: 'none' }}
                  data-testid="agent-viewport-iframe"
                />
              </div>
            </CardContent>
          )}
        </Card>
      )}

      {/* Task Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <Card className="knirv-card-gradient">
          <CardContent className="p-3">
            <div className="text-2xl font-bold">{taskStats.total}</div>
            <div className="text-[10px] uppercase tracking-widest text-muted-foreground">Total</div>
          </CardContent>
        </Card>
        <Card className="knirv-card-gradient">
          <CardContent className="p-3">
            <div className="text-2xl font-bold text-yellow-400">{taskStats.pending}</div>
            <div className="text-[10px] uppercase tracking-widest text-muted-foreground">Pending</div>
          </CardContent>
        </Card>
        <Card className="knirv-card-gradient">
          <CardContent className="p-3">
            <div className="text-2xl font-bold text-green-400">{taskStats.completed}</div>
            <div className="text-[10px] uppercase tracking-widest text-muted-foreground">Completed</div>
          </CardContent>
        </Card>
        <Card className="knirv-card-gradient">
          <CardContent className="p-3">
            <div className="text-2xl font-bold text-red-400">{taskStats.failed}</div>
            <div className="text-[10px] uppercase tracking-widest text-muted-foreground">Failed</div>
          </CardContent>
        </Card>
      </div>

      {/* Task List + New Task Form */}
      <Card className="knirv-card-gradient">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center space-x-2">
              <ListTodo className="w-5 h-5" />
              <span>Tasks</span>
            </CardTitle>
            <Button
              variant="default"
              size="sm"
              onClick={() => setShowTaskForm(v => !v)}
              data-testid="new-task-toggle-button"
              className="bg-primary hover:bg-primary/90 font-black uppercase text-[10px]"
            >
              <Plus className="w-3 h-3 mr-1" />
              New Task
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Inline Task Form */}
          {showTaskForm && (
            <div
              className="bg-slate-900/60 border border-slate-700 rounded-lg p-4 space-y-3"
              data-testid="task-form"
            >
              <p className="text-xs font-black uppercase tracking-wider text-slate-400">
                Submit New Task
              </p>

              {formError && (
                <Alert className="border-red-500/30 bg-red-950/20">
                  <AlertCircle className="h-4 w-4 text-red-400" />
                  <AlertDescription className="text-red-300 text-xs">{formError}</AlertDescription>
                </Alert>
              )}

              {/* Title */}
              <div className="space-y-1">
                <label className="text-[10px] font-bold uppercase tracking-wider text-slate-400">
                  Title <span className="text-red-400">*</span>
                </label>
                <input
                  type="text"
                  value={formState.title}
                  onChange={e => setFormState(prev => ({ ...prev, title: e.target.value }))}
                  placeholder="Brief task title"
                  className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 placeholder:text-slate-600"
                  data-testid="task-title-input"
                />
              </div>

              {/* Description */}
              <div className="space-y-1">
                <label className="text-[10px] font-bold uppercase tracking-wider text-slate-400">
                  Description
                </label>
                <textarea
                  value={formState.description}
                  onChange={e => setFormState(prev => ({ ...prev, description: e.target.value }))}
                  placeholder="Detailed task description..."
                  rows={3}
                  className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500 placeholder:text-slate-600 resize-none"
                  data-testid="task-description-input"
                />
              </div>

              {/* Type + Priority row */}
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-[10px] font-bold uppercase tracking-wider text-slate-400">
                    Type
                  </label>
                  <select
                    value={formState.type}
                    onChange={e =>
                      setFormState(prev => ({ ...prev, type: e.target.value as TaskType }))
                    }
                    className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500"
                    data-testid="task-type-select"
                  >
                    <option value="research">Research</option>
                    <option value="coding">Coding</option>
                    <option value="validation">Validation</option>
                    <option value="analysis">Analysis</option>
                  </select>
                </div>
                <div className="space-y-1">
                  <label className="text-[10px] font-bold uppercase tracking-wider text-slate-400">
                    Priority
                  </label>
                  <select
                    value={formState.priority}
                    onChange={e =>
                      setFormState(prev => ({ ...prev, priority: e.target.value as TaskPriority }))
                    }
                    className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-blue-500"
                    data-testid="task-priority-select"
                  >
                    <option value="low">Low</option>
                    <option value="medium">Medium</option>
                    <option value="high">High</option>
                    <option value="critical">Critical</option>
                  </select>
                </div>
              </div>

              {/* Form actions */}
              <div className="flex items-center space-x-2 pt-1">
                <Button
                  variant="default"
                  size="sm"
                  onClick={submitTask}
                  disabled={isSubmittingTask}
                  data-testid="submit-task-button"
                  className="bg-blue-600 hover:bg-blue-500 text-white font-black uppercase text-[10px]"
                >
                  {isSubmittingTask ? (
                    <>
                      <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                      Submitting...
                    </>
                  ) : (
                    <>
                      <Plus className="w-3 h-3 mr-1" />
                      Submit Task
                    </>
                  )}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setShowTaskForm(false);
                    setFormState(defaultFormState);
                    setFormError(null);
                  }}
                  data-testid="cancel-task-button"
                  className="font-black uppercase text-[10px] border-slate-700 text-slate-400 hover:text-white"
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}

          {/* Task Error */}
          {tasksError && (
            <Alert className="border-red-200 bg-red-50">
              <AlertCircle className="h-4 w-4 text-red-600" />
              <AlertDescription className="text-red-700 text-sm">{tasksError}</AlertDescription>
            </Alert>
          )}

          {/* Loading state */}
          {isTasksLoading ? (
            <div className="flex items-center justify-center py-8" data-testid="tasks-loading">
              <Activity className="w-5 h-5 animate-spin mr-2 text-primary" />
              <span className="text-sm text-muted-foreground">Loading tasks...</span>
            </div>
          ) : tasks.length === 0 ? (
            <div
              className="text-center py-8 text-muted-foreground text-sm"
              data-testid="tasks-empty"
            >
              No tasks found. Submit a new task to get started.
            </div>
          ) : (
            <div className="space-y-2" data-testid="task-list">
              {tasks.map(task => (
                <div
                  key={task.id}
                  className="bg-slate-900/50 border border-slate-800 rounded-lg p-3 hover:border-slate-600 transition-colors cursor-pointer"
                  onClick={() =>
                    setExpandedTaskId(prev => (prev === task.id ? null : task.id))
                  }
                  data-testid={`task-item-${task.id}`}
                >
                  {/* Task header row */}
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-2 min-w-0">
                      <span
                        className={`text-[10px] font-black uppercase ${getPriorityColor(task.priority)}`}
                      >
                        {task.priority}
                      </span>
                      <span className="text-sm font-semibold text-slate-200 truncate">
                        {task.title}
                      </span>
                    </div>
                    <div className="flex items-center space-x-2 flex-shrink-0 ml-2">
                      <Badge
                        variant="secondary"
                        className={`text-[9px] font-black uppercase ${getTaskStatusColor(task.status)}`}
                        data-testid={`task-status-${task.id}`}
                      >
                        <div className="flex items-center space-x-1">
                          {getTaskStatusIcon(task.status)}
                          <span>{task.status}</span>
                        </div>
                      </Badge>
                      <Badge
                        variant="outline"
                        className="text-[9px] font-black uppercase border-slate-700 text-slate-400"
                      >
                        {task.type}
                      </Badge>
                      {expandedTaskId === task.id ? (
                        <ChevronUp className="w-3 h-3 text-slate-500" />
                      ) : (
                        <ChevronDown className="w-3 h-3 text-slate-500" />
                      )}
                    </div>
                  </div>

                  {/* Expanded detail */}
                  {expandedTaskId === task.id && (
                    <div
                      className="mt-3 pt-3 border-t border-slate-800 space-y-2 text-xs"
                      data-testid={`task-detail-${task.id}`}
                    >
                      {task.description && (
                        <p className="text-slate-400 leading-relaxed">{task.description}</p>
                      )}
                      <div className="grid grid-cols-2 gap-2 text-[10px] font-mono bg-slate-950/50 p-2 rounded">
                        <div>
                          <span className="text-slate-500 uppercase">ID:</span>{' '}
                          <span className="text-slate-300">{task.id.substring(0, 16)}...</span>
                        </div>
                        <div>
                          <span className="text-slate-500 uppercase">Created:</span>{' '}
                          <span className="text-slate-300">
                            {formatRelativeTime(task.created_at)}
                          </span>
                        </div>
                        {task.started_at && (
                          <div>
                            <span className="text-slate-500 uppercase">Started:</span>{' '}
                            <span className="text-slate-300">
                              {formatRelativeTime(task.started_at)}
                            </span>
                          </div>
                        )}
                        {task.completed_at && (
                          <div>
                            <span className="text-slate-500 uppercase">Completed:</span>{' '}
                            <span className="text-slate-300">
                              {formatRelativeTime(task.completed_at)}
                            </span>
                          </div>
                        )}
                      </div>
                      {task.error_message && (
                        <Alert className="border-red-500/30 bg-red-950/20 py-1">
                          <AlertCircle className="h-3 w-3 text-red-400" />
                          <AlertDescription className="text-red-300 text-[10px]">
                            {task.error_message}
                          </AlertDescription>
                        </Alert>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default AgentCommandCenter;
