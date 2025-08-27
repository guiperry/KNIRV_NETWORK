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



export const useCognitiveEngine = () => {
  const [cognitiveEngine, setCognitiveEngine] = useState<CognitiveEngine | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isPolling, setIsPolling] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [socket, setSocket] = useState<WebSocket | null>(null);

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
    if (socket?.readyState === WebSocket.OPEN) return;

    const wsUrl = `${API_BASE_URL.replace('http', 'ws')}/ws`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('WebSocket connected to KNIRV-NEXUS');
      setIsConnected(true);
      setError(null);

      // Subscribe to cognitive engine updates
      ws.send(JSON.stringify({
        type: 'subscribe',
        topics: ['cognitive-engine-updated', 'system-notification']
      }));
    };

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);

        if (message.event === 'cognitive-engine-updated' && message.payload) {
          setCognitiveEngine(message.payload);
        } else if (message.event === 'connected') {
          console.log('WebSocket welcome:', message.payload);
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      setIsConnected(false);
      setSocket(null);

      // Attempt to reconnect after 5 seconds
      setTimeout(() => {
        if (!socket || socket.readyState === WebSocket.CLOSED) {
          connectWebSocket();
        }
      }, 5000);
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
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
    connectWebSocket();
  }, [fetchCognitiveEngine, connectWebSocket]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      stopPolling();
      disconnectWebSocket();
    };
  }, [stopPolling, disconnectWebSocket]);

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
