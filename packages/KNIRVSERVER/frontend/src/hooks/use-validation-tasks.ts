"use client";

import { useEffect, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  ValidationTask,
  ValidationTaskFilter,
  CreateTaskRequest,
  APIResponse
} from '@/types/api';
import { apiRequest, API_BASE_URL, buildQueryString } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export const useValidationTasks = (filter?: ValidationTaskFilter) => {
  const queryClient = useQueryClient();
  const queryString = buildQueryString(filter || {});
  const queryKey = ['validation-tasks', queryString];

  const {
    data: tasks = [],
    isLoading,
    error: queryError,
    refetch: fetchTasks
  } = useQuery<ValidationTask[]>({
    queryKey,
    queryFn: async () => {
      const url = `${API_BASE_URL}/api/validation-tasks${queryString}`;
      const response: APIResponse<ValidationTask[]> = await apiRequest(url, { method: 'GET' });
      if (response.success && Array.isArray(response.data)) {
        return response.data;
      }
      throw new Error(response.error || 'Failed to fetch validation tasks');
    },
    staleTime: 30000,
  });

  const error = queryError instanceof Error ? queryError.message : null;

  useEffect(() => {
    const handleValidationTaskUpdate = (payload: any) => {
      queryClient.setQueryData<ValidationTask[]>(queryKey, (prevTasks = []) => 
        prevTasks.map(task =>
          task.id === payload.id
            ? {
                ...task,
                status: payload.status ?? task.status,
                assigned_node_id: payload.assigned_node ?? task.assigned_node_id,
                updated_at: payload.updated_at ?? task.updated_at
              }
            : task
        )
      );
    };

    webSocketService.on('validation-task-updated', handleValidationTaskUpdate);
    webSocketService.subscribe(['validation-task-updated']);

    return () => {
      webSocketService.off('validation-task-updated', handleValidationTaskUpdate);
    };
  }, [queryClient, queryKey]);

  const createTaskMutation = useMutation({
    mutationFn: async (taskData: CreateTaskRequest) => {
      const url = `${API_BASE_URL}/api/validation-tasks`;
      const response: APIResponse<ValidationTask> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(taskData),
      });
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to create validation task');
    },
    onSuccess: (newTask) => {
      queryClient.setQueryData<ValidationTask[]>(queryKey, (prev = []) => [...prev, newTask]);
    }
  });

  const executeTaskMutation = useMutation({
    mutationFn: async (taskId: string) => {
      const url = `${API_BASE_URL}/api/validation-tasks/${taskId}/execute`;
      const response: APIResponse = await apiRequest(url, { method: 'POST' });
      if (response.success) return taskId;
      throw new Error(response.error || 'Failed to execute validation task');
    },
    onSuccess: (taskId) => {
      queryClient.setQueryData<ValidationTask[]>(queryKey, (prev = []) => 
        prev.map(task => task.id === taskId ? { ...task, status: 'running' } : task)
      );
    }
  });

  return {
    tasks,
    isLoading,
    error,
    isConnected: webSocketService.getConnectionStatus(),
    fetchTasks,
    createTask: createTaskMutation.mutateAsync,
    executeTask: executeTaskMutation.mutateAsync,
    refreshTasks: fetchTasks,
    getPendingTasks: () => fetchTasks(),
    getRunningTasks: () => fetchTasks(),
  };
};

export default useValidationTasks;
