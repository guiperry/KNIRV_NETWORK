import * as React from 'react';
import { useState, useEffect, useCallback, useRef } from 'react';
import {
  Calendar,
  Clock,
  Play,
  Trash2,
  Plus,
  CheckCircle,
  XCircle,
  RefreshCw,
  AlertTriangle
} from 'lucide-react';
import { taskSchedulingService, ScheduledTask, TaskExecution } from '../services/TaskSchedulingService';

interface TaskSchedulerProps {
  isOpen: boolean;
  onClose: () => void;
}

const TaskScheduler: React.FC<TaskSchedulerProps> = ({ isOpen, onClose }) => {
  const [tasks, setTasks] = useState<ScheduledTask[]>([]);
  const [executions, setExecutions] = useState<Record<string, TaskExecution[]>>({});
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'tasks' | 'create' | 'executions'>('tasks');
  // const [selectedTask, setSelectedTask] = useState<ScheduledTask | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const intervalRef = useRef<number | null>(null);

  const loadTasks = useCallback(async () => {
    setIsLoading(true);
    try {
      const allTasks = taskSchedulingService.getAllTasks();
      setTasks(allTasks);

      // Load executions for each task
      const executionData: Record<string, TaskExecution[]> = {};
      for (const task of allTasks) {
        executionData[task.id] = taskSchedulingService.getTaskExecutions(task.id);
      }
      setExecutions(executionData);
    } catch (error) {
      console.error('Failed to load tasks:', error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Cleanup function for interval
  const clearRefreshInterval = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      loadTasks();
      // Auto-refresh every 30 seconds
      clearRefreshInterval(); // Clear any existing interval
      intervalRef.current = setInterval(loadTasks, 30000);
    } else {
      // Clear interval when modal is closed
      clearRefreshInterval();
    }

    // Cleanup on unmount
    return () => {
      clearRefreshInterval();
    };
  }, [isOpen, loadTasks, clearRefreshInterval]);

  const handleCreateTask = async (taskData: Omit<ScheduledTask, 'id' | 'createdAt' | 'runCount' | 'successCount' | 'failureCount'>) => {
    try {
      await taskSchedulingService.createTask(taskData);
      await loadTasks();
      setActiveTab('tasks');
      setIsCreating(false);
    } catch (error) {
      console.error('Failed to create task:', error);
    }
  };

  const handleExecuteTask = async (taskId: string) => {
    try {
      await taskSchedulingService.executeTask(taskId);
      await loadTasks();
    } catch (error) {
      console.error('Failed to execute task:', error);
    }
  };

  const handleDeleteTask = async (taskId: string) => {
    try {
      await taskSchedulingService.deleteTask(taskId);
      await loadTasks();
    } catch (error) {
      console.error('Failed to delete task:', error);
    }
  };

  const getStatusIcon = (status: ScheduledTask['status'] | TaskExecution['status']) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-4 h-4 text-green-400" />;
      case 'running':
        return <RefreshCw className="w-4 h-4 text-blue-400 animate-spin" />;
      case 'failed':
        return <XCircle className="w-4 h-4 text-red-400" />;
      case 'cancelled':
        return <XCircle className="w-4 h-4 text-gray-400" />;
      case 'pending':
        return <Clock className="w-4 h-4 text-yellow-400" />;
      case 'timeout':
        return <AlertTriangle className="w-4 h-4 text-orange-400" />;
      default:
        return <Clock className="w-4 h-4 text-yellow-400" />;
    }
  };

  const getPriorityColor = (priority: ScheduledTask['priority']) => {
    switch (priority) {
      case 'critical':
        return 'text-red-400 bg-red-500/20 border-red-500/20';
      case 'high':
        return 'text-orange-400 bg-orange-500/20 border-orange-500/20';
      case 'medium':
        return 'text-yellow-400 bg-yellow-500/20 border-yellow-500/20';
      default:
        return 'text-gray-400 bg-gray-500/20 border-gray-500/20';
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-gray-900/95 border border-gray-700/50 rounded-xl w-full max-w-6xl h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700/50">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-purple-500/20 rounded-lg flex items-center justify-center border border-purple-500/20">
              <Calendar className="w-5 h-5 text-purple-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">Task Scheduler</h2>
              <p className="text-sm text-gray-400">Automated workflow management</p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={loadTasks}
              disabled={isLoading}
              aria-label="Refresh tasks"
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={onClose}
              aria-label="Close task scheduler"
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
            >
              ×
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-700/50">
          {[
            { id: 'tasks', label: 'Tasks', icon: Calendar },
            { id: 'create', label: 'Create Task', icon: Plus },
            { id: 'executions', label: 'Executions', icon: Play }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as 'tasks' | 'create' | 'executions')}
              className={`flex items-center space-x-2 px-6 py-3 text-sm font-medium transition-all ${
                activeTab === tab.id
                  ? 'text-purple-400 border-b-2 border-purple-400 bg-purple-500/10'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700/30'
              }`}
            >
              <tab.icon className="w-4 h-4" />
              <span>{tab.label}</span>
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto h-full">
          {activeTab === 'tasks' && (
            <div className="space-y-4">
              {tasks.length === 0 ? (
                <div className="text-center py-12">
                  <Calendar className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                  <p className="text-gray-400">No scheduled tasks found</p>
                  <button
                    onClick={() => setActiveTab('create')}
                    className="mt-4 px-4 py-2 bg-purple-500/20 border border-purple-500/20 text-purple-400 rounded-lg hover:bg-purple-500/30 transition-all"
                  >
                    Create First Task
                  </button>
                </div>
              ) : (
                tasks.map(task => (
                  <div key={task.id} className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center space-x-3">
                        {getStatusIcon(task.status)}
                        <div>
                          <h3 className="font-semibold text-white">{task.name}</h3>
                          <p className="text-sm text-gray-400">{task.description}</p>
                        </div>
                      </div>
                      <div className="flex items-center space-x-2">
                        <span className={`px-2 py-1 text-xs rounded border ${getPriorityColor(task.priority)}`}>
                          {task.priority}
                        </span>
                        <button
                          onClick={() => handleExecuteTask(task.id)}
                          aria-label={`Execute task ${task.name}`}
                          className="p-1 hover:bg-gray-700/50 rounded text-gray-400 hover:text-white transition-all"
                        >
                          <Play className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDeleteTask(task.id)}
                          aria-label={`Delete task ${task.name}`}
                          className="p-1 hover:bg-gray-700/50 rounded text-gray-400 hover:text-red-400 transition-all"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                    
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                      <div>
                        <span className="text-gray-400">Type:</span>
                        <span className="ml-2 text-white">{task.type}</span>
                      </div>
                      <div>
                        <span className="text-gray-400">Schedule:</span>
                        <span className="ml-2 text-white">{task.schedule.type}</span>
                      </div>
                      <div>
                        <span className="text-gray-400">Runs:</span>
                        <span className="ml-2 text-white">{task.runCount}</span>
                      </div>
                      <div>
                        <span className="text-gray-400">Success Rate:</span>
                        <span className="ml-2 text-white">
                          {task.runCount > 0 ? Math.round((task.successCount / task.runCount) * 100) : 0}%
                        </span>
                      </div>
                    </div>

                    {task.nextRun && (
                      <div className="mt-3 text-sm">
                        <span className="text-gray-400">Next Run:</span>
                        <span className="ml-2 text-white">{task.nextRun.toLocaleString()}</span>
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          )}

          {activeTab === 'create' && (
            <TaskCreateForm onSubmit={handleCreateTask} onCancel={() => setActiveTab('tasks')} isSubmitting={isCreating} />
          )}

          {activeTab === 'executions' && (
            <div className="space-y-4">
              {Object.entries(executions).map(([taskId, taskExecutions]) => {
                const task = tasks.find(t => t.id === taskId);
                if (!task || taskExecutions.length === 0) return null;

                return (
                  <div key={taskId} className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
                    <h3 className="font-semibold text-white mb-3">{task.name}</h3>
                    <div className="space-y-2">
                      {taskExecutions.slice(0, 5).map(execution => (
                        <div key={execution.id} className="flex items-center justify-between p-2 bg-gray-700/30 rounded">
                          <div className="flex items-center space-x-3">
                            {getStatusIcon(execution.status)}
                            <span className="text-sm text-gray-300">
                              {execution.startTime.toLocaleString()}
                            </span>
                          </div>
                          <div className="text-sm text-gray-400">
                            {execution.duration ? `${execution.duration}ms` : 'Running...'}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

interface TaskCreateFormProps {
  onSubmit: (taskData: Omit<ScheduledTask, 'id' | 'createdAt' | 'runCount' | 'successCount' | 'failureCount'>) => void;
  onCancel: () => void;
}

interface TaskCreateFormProps {
  onSubmit: (taskData: Omit<ScheduledTask, 'id' | 'createdAt' | 'runCount' | 'successCount' | 'failureCount'>) => void;
  onCancel: () => void;
  isSubmitting: boolean;
}

const TaskCreateForm: React.FC<TaskCreateFormProps> = ({ onSubmit, onCancel, isSubmitting }) => {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    type: 'custom' as ScheduledTask['type'],
    priority: 'medium' as ScheduledTask['priority'],
    scheduleType: 'once' as 'once' | 'recurring' | 'cron',
    startTime: new Date().toISOString().slice(0, 16),
    interval: 3600000, // 1 hour in ms
    actionType: 'api_call' as 'api_call' | 'agent_invoke' | 'system_command',
    target: '',
    parameters: '{}'
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const taskData = {
      name: formData.name,
      description: formData.description,
      type: formData.type,
      status: 'pending' as const,
      priority: formData.priority,
      schedule: {
        type: formData.scheduleType,
        startTime: new Date(formData.startTime),
        interval: formData.scheduleType === 'recurring' ? formData.interval : undefined
      },
      action: {
        type: formData.actionType,
        target: formData.target,
        parameters: JSON.parse(formData.parameters || '{}')
      },
      metadata: {}
    };

    onSubmit(taskData);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6 max-w-2xl">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="task-name" className="block text-sm font-medium text-gray-300 mb-2">Task Name</label>
          <input
            id="task-name"
            type="text"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none"
            required
          />
        </div>

        <div>
          <label htmlFor="task-priority" className="block text-sm font-medium text-gray-300 mb-2">Priority</label>
          <select
            id="task-priority"
            value={formData.priority}
            onChange={(e) => setFormData({ ...formData, priority: e.target.value as ScheduledTask['priority'] })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none"
          >
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="critical">Critical</option>
          </select>
        </div>
      </div>

      <div>
        <label htmlFor="task-description" className="block text-sm font-medium text-gray-300 mb-2">Description</label>
        <textarea
          id="task-description"
          value={formData.description}
          onChange={(e) => setFormData({ ...formData, description: e.target.value })}
          className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none"
          rows={3}
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="schedule-type" className="block text-sm font-medium text-gray-300 mb-2">Schedule Type</label>
          <select
            id="schedule-type"
            value={formData.scheduleType}
            onChange={(e) => setFormData({ ...formData, scheduleType: e.target.value as 'once' | 'recurring' | 'cron' })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none"
          >
            <option value="once">Run Once</option>
            <option value="recurring">Recurring</option>
            <option value="cron">Cron Schedule</option>
          </select>
        </div>

        <div>
          <label htmlFor="start-time" className="block text-sm font-medium text-gray-300 mb-2">Start Time</label>
          <input
            id="start-time"
            type="datetime-local"
            value={formData.startTime}
            onChange={(e) => setFormData({ ...formData, startTime: e.target.value })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none"
            required
          />
        </div>
      </div>

      {formData.scheduleType === 'recurring' && (
        <div>
          <label htmlFor="interval" className="block text-sm font-medium text-gray-300 mb-2">Interval (minutes)</label>
          <input
            id="interval"
            type="number"
            value={formData.interval / 60000}
            onChange={(e) => setFormData({ ...formData, interval: parseInt(e.target.value) * 60000 })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none"
            min="1"
          />
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="action-type" className="block text-sm font-medium text-gray-300 mb-2">Action Type</label>
          <select
            id="action-type"
            value={formData.actionType}
            onChange={(e) => setFormData({ ...formData, actionType: e.target.value as 'api_call' | 'agent_invoke' | 'system_command' })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none"
          >
            <option value="api_call">API Call</option>
            <option value="agent_invoke">Agent Invoke</option>
            <option value="system_command">System Command</option>
          </select>
        </div>

        <div>
          <label htmlFor="target" className="block text-sm font-medium text-gray-300 mb-2">Target</label>
          <input
            id="target"
            type="text"
            value={formData.target}
            onChange={(e) => setFormData({ ...formData, target: e.target.value })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none"
            placeholder="URL, Agent ID, or Command"
            required
          />
        </div>
      </div>

      <div>
        <label htmlFor="parameters" className="block text-sm font-medium text-gray-300 mb-2">Parameters (JSON)</label>
        <textarea
          id="parameters"
          value={formData.parameters}
          onChange={(e) => setFormData({ ...formData, parameters: e.target.value })}
          className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-purple-400 focus:outline-none font-mono text-sm"
          rows={4}
          placeholder='{"key": "value"}'
        />
      </div>

      <div className="flex space-x-4">
        <button
          type="submit"
          className="px-6 py-2 bg-purple-500 text-white rounded-lg hover:bg-purple-600 transition-all disabled:opacity-50"
          disabled={isSubmitting}
        >
          {isSubmitting ? 'Creating...' : 'Create Task'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-6 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-all"
        >
          Cancel
        </button>
      </div>
    </form>
  );
};

export default TaskScheduler;
