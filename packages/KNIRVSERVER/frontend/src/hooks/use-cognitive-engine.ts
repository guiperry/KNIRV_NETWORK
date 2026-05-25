"use client";

import { useEffect, useCallback, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export interface BackgroundTask {
  id: string;
  category: 'workflow' | 'ontology' | 'error_node' | 'dve_monitor' | 'agent_monitor' | 'guardrails' | 'cache';
  label: string;
  status: 'running' | 'completed' | 'failed' | 'queued';
  detail?: string;
  timestamp: number;
}

export interface CognitiveEngine {
  status: "active" | "idle" | "learning" | "error" | "stopped" | "degraded";
  accuracy: number;
  tasks_processed: number;
  adaptation_rate: number;
  model_version: string;
  uptime: number;
  last_training: string;
  current_task_status?: string;
  background_tasks?: BackgroundTask[];
  health_status?: string;
  last_validation?: string;
  moa_config?: {
    available_quality_modes: string[];
    current_quality_mode: string;
    iterations: number;
    max_parallel_agents: number;
  };
  performance_metrics: {
    inference_latency: number;
    throughput: number;
    error_rate: number;
  };
  learning_metrics: {
    training_accuracy: number;
    validation_accuracy: number;
    loss: number;
  };
}

export interface CognitiveEngineResponse {
  success: boolean;
  data?: CognitiveEngine;
  message?: string;
  error?: string;
  timestamp: string;
}

export interface CognitiveEngineAction {
  action: string;
  parameters?: Record<string, any>;
}

export const useQualityMode = () => {
  const [qualityMode, setQualityMode] = useState<'standard' | 'deep'>('standard');
  return { qualityMode, setQualityMode };
};

export const useCognitiveEngine = () => {
  const queryClient = useQueryClient();
  const queryKey = ['cognitive-engine'];

  const {
    data: cognitiveEngine = null,
    isLoading,
    error: queryError,
    refetch: fetchCognitiveEngine
  } = useQuery<CognitiveEngine>({
    queryKey,
    queryFn: async () => {
      const url = `${API_BASE_URL}/api/cognitive-engine`;
      const response: CognitiveEngineResponse = await apiRequest(url, { method: 'GET' });
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch cognitive engine data');
    },
    staleTime: 30000,
  });

  const error = queryError instanceof Error ? queryError.message : null;

  useEffect(() => {
    const handleCognitiveEngineUpdate = (payload: any) => {
      queryClient.setQueryData<CognitiveEngine>(queryKey, payload);
    };

    webSocketService.on('cognitive-engine-updated', handleCognitiveEngineUpdate);
    webSocketService.subscribe(['cognitive-engine-updated']);

    return () => {
      webSocketService.off('cognitive-engine-updated', handleCognitiveEngineUpdate);
    };
  }, [queryClient, queryKey]);

  const performActionMutation = useMutation({
    mutationFn: async (action: CognitiveEngineAction) => {
      const url = `${API_BASE_URL}/api/cognitive-engine`;
      const response: CognitiveEngineResponse = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to perform action');
    },
    onSuccess: (data) => {
      queryClient.setQueryData(queryKey, data);
    }
  });

  return {
    cognitiveEngine,
    isLoading,
    error,
    isConnected: webSocketService.getConnectionStatus(),
    fetchCognitiveEngine,
    startTraining: () => performActionMutation.mutateAsync({ action: 'start_training' }),
    stopTraining: () => performActionMutation.mutateAsync({ action: 'stop_training' }),
    startEngine: () => performActionMutation.mutateAsync({ action: 'start_engine' }),
    stopEngine: () => performActionMutation.mutateAsync({ action: 'stop_engine' }),
    healthCheck: () => performActionMutation.mutateAsync({ action: 'health_check' }),
    selfValidate: () => performActionMutation.mutateAsync({ action: 'self_validate' }),
    makeRequest: (message: string, quality?: string) => performActionMutation.mutateAsync({ action: 'make_request', parameters: { message, ...(quality ? { quality } : {}) } }),
    sendChat: async (message: string, sessionId?: string | null, quality?: string) => {
      const url = `${API_BASE_URL}/api/cognitive-engine/chat`;
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message,
          ...(sessionId ? { session_id: sessionId } : {}),
          ...(quality ? { quality } : {}),
        }),
      });
      return response.json();
    },
    resetMetrics: () => performActionMutation.mutateAsync({ action: 'reset_metrics' }),
    clearConversationHistory: () => performActionMutation.mutateAsync({ action: 'clear_conversation_history' }),
    updateFabricVersion: (modelVersion: string) => performActionMutation.mutateAsync({ action: 'update_model', parameters: { model_version: modelVersion } }),
    startPolling: () => {}, // Polling removed in favor of React Query / WebSockets
    stopPolling: () => {},
  };
};

export default useCognitiveEngine;
