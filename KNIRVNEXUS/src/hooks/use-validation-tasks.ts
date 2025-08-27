"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  ValidationTask,
  ValidationTaskFilter,
  CreateTaskRequest,
  APIResponse,
  ValidationTaskUpdate
} from '@/types/api';
import { apiRequest, API_BASE_URL, buildQueryString, StandardWebSocket } from '@/lib/api';

export const useValidationTasks = () => {
  const [tasks, setTasks] = useState<ValidationTask[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [socket, setSocket] = useState<StandardWebSocket | null>(null);

  // Fetch validation tasks with optional filtering
  const fetchTasks = useCallback(async (filter?: ValidationTaskFilter) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const queryString = buildQueryString(filter || {});
      const url = `${API_BASE_URL}/api/validation-tasks${queryString}`;
      const response: APIResponse<ValidationTask[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && Array.isArray(response.data)) {
        setTasks(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch validation tasks');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch validation tasks:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Get a specific validation task by ID
  const getTask = useCallback(async (taskId: string): Promise<ValidationTask | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/validation-tasks/${taskId}`;
      const response: APIResponse<ValidationTask> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch validation task');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch validation task:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Create a new validation task
  const createTask = useCallback(async (taskData: CreateTaskRequest): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/validation-tasks`;
      const response: APIResponse<ValidationTask> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(taskData),
      });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        // Add the new task to the current list
        setTasks(prevTasks => [...prevTasks, response.data as ValidationTask]);
        return true;
      } else {
        throw new Error(response.error || 'Failed to create validation task');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create validation task:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Execute a validation task
  const executeTask = useCallback(async (taskId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/validation-tasks/${taskId}/execute`;
      const response: APIResponse = await apiRequest(url, { method: 'POST' });
      
      if (response.success) {
        // Update the task status in the current list
        setTasks(prevTasks => 
          prevTasks.map(task => 
            task.id === taskId ? { ...task, status: 'running' } : task
          )
        );
        return true;
      } else {
        throw new Error(response.error || 'Failed to execute validation task');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to execute validation task:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    if (socket?.isConnected()) return;

    const ws = new StandardWebSocket();

    ws.onOpen = () => {
      console.log('Validation Tasks WebSocket connected');
      setIsConnected(true);
      setError(null);

      // Subscribe to validation task updates
      ws.subscribe(['validation-task-updated', 'system-notification']);
    };

    ws.onMessage = (message) => {
      if (message.event === 'validation-task-updated' && message.payload) {
        // Update specific task in the list
        setTasks(prevTasks =>
          prevTasks.map(task =>
            task.id === message.payload.id
              ? {
                  ...task,
                  status: message.payload.status || task.status,
                  assigned_node_id: message.payload.assigned_node || task.assigned_node_id,
                  updated_at: message.payload.updated_at || task.updated_at
                }
              : task
          )
        );
      } else if (message.event === 'connected') {
        console.log('Validation Tasks WebSocket welcome:', message.payload);
      }
    };

    ws.onClose = () => {
      console.log('Validation Tasks WebSocket disconnected');
      setIsConnected(false);
    };

    ws.onError = (error) => {
      console.error('Validation Tasks WebSocket error:', error);
      setError('WebSocket connection failed');
    };

    setSocket(ws);
  }, [socket]);

  const disconnectWebSocket = useCallback(() => {
    if (socket) {
      socket.close();
      setSocket(null);
      setIsConnected(false);
    }
  }, [socket]);

  // Convenience methods for common operations
  const getPendingTasks = useCallback(() => fetchTasks({ status: 'pending' }), [fetchTasks]);
  const getRunningTasks = useCallback(() => fetchTasks({ status: 'running' }), [fetchTasks]);
  const getTasksByType = useCallback((type: string) => fetchTasks({ type }), [fetchTasks]);
  const refreshTasks = useCallback(() => fetchTasks(), [fetchTasks]);

  // Initial fetch and WebSocket connection on mount
  useEffect(() => {
    fetchTasks();
    connectWebSocket();
  }, [fetchTasks, connectWebSocket]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnectWebSocket();
    };
  }, [disconnectWebSocket]);

  return {
    tasks,
    isLoading,
    error,
    isConnected,
    fetchTasks,
    getTask,
    createTask,
    executeTask,
    getPendingTasks,
    getRunningTasks,
    getTasksByType,
    refreshTasks,
    connectWebSocket,
    disconnectWebSocket,
  };
};

export default useValidationTasks;
