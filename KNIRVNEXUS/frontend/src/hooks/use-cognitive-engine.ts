"use client";

import { useState, useEffect, useCallback } from 'react';

export interface CognitiveEngine {
  status: "active" | "idle" | "learning" | "error";
  accuracy: number;
  tasks_processed: number;
  adaptation_rate: number;
  model_version: string;
  uptime: number;
  last_training: string;
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

import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';



export const useCognitiveEngine = () => {
  const [cognitiveEngine, setCognitiveEngine] = useState<CognitiveEngine | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isPolling, setIsPolling] = useState(false);
  const [isConnected, setIsConnected] = useState(false);

  // Fetch cognitive engine status
  const fetchCognitiveEngine = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/cognitive-engine`;
      const response: CognitiveEngineResponse = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setCognitiveEngine(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch cognitive engine data');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch cognitive engine:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Perform cognitive engine action
  const performAction = useCallback(async (action: CognitiveEngineAction): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/cognitive-engine`;
      const response: CognitiveEngineResponse = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      
      if (response.success && response.data) {
        setCognitiveEngine(response.data);
        return true;
      } else {
        throw new Error(response.error || 'Failed to perform action');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to perform cognitive engine action:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Convenience methods for common actions
  const startTraining = useCallback(() => performAction({ action: 'start_training' }), [performAction]);
  const stopTraining = useCallback(() => performAction({ action: 'stop_training' }), [performAction]);
  const resetMetrics = useCallback(() => performAction({ action: 'reset_metrics' }), [performAction]);
  const clearConversationHistory = useCallback(() => performAction({ action: 'clear_conversation_history' }), [performAction]);
  
  const updateModel = useCallback((modelVersion: string) =>
    performAction({ action: 'update_model', parameters: { model_version: modelVersion } }),
    [performAction]
  );

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    // Set up event handlers
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
      if (data.connected) {
        console.log('Cognitive Engine WebSocket connected');
        setError(null);
      } else {
        console.log('Cognitive Engine WebSocket disconnected');
      }
    };

    const handleCognitiveEngineUpdate = (payload: any) => {
      setCognitiveEngine(payload);
    };

    const handleSystemNotification = (payload: any) => {
      console.log('Cognitive Engine system notification:', payload);
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    webSocketService.on('cognitive-engine-updated', handleCognitiveEngineUpdate);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe(['cognitive-engine-updated', 'system-notification']);

    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Return cleanup function
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('cognitive-engine-updated', handleCognitiveEngineUpdate);
      webSocketService.off('system-notification', handleSystemNotification);
    };
  }, []);

  const disconnectWebSocket = useCallback(() => {
    // Individual hooks don't disconnect the shared service
    setIsConnected(false);
  }, []);

  // Start/stop polling for real-time updates
  const startPolling = useCallback((interval: number = 5000) => {
    if (isPolling) return;
    
    setIsPolling(true);
    const pollInterval = setInterval(() => {
      fetchCognitiveEngine();
    }, interval);

    // Store interval ID for cleanup
    (window as any).__cognitiveEnginePollingInterval = pollInterval;
  }, [fetchCognitiveEngine, isPolling]);

  const stopPolling = useCallback(() => {
    if (!isPolling) return;
    
    const pollInterval = (window as any).__cognitiveEnginePollingInterval;
    if (pollInterval) {
      clearInterval(pollInterval);
      delete (window as any).__cognitiveEnginePollingInterval;
    }
    setIsPolling(false);
  }, [isPolling]);

  // Initial fetch and WebSocket connection on mount
  useEffect(() => {
    fetchCognitiveEngine();
    const cleanup = connectWebSocket();
    return () => {
      stopPolling();
      if (cleanup) cleanup();
    };
  }, [fetchCognitiveEngine, connectWebSocket, stopPolling]);

  return {
    cognitiveEngine,
    isLoading,
    error,
    isPolling,
    isConnected,
    fetchCognitiveEngine,
    performAction,
    startTraining,
    stopTraining,
    resetMetrics,
    clearConversationHistory,
    updateModel,
    startPolling,
    stopPolling,
    connectWebSocket,
    disconnectWebSocket,
  };
};

export default useCognitiveEngine;
