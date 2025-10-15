"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  APIResponse,
  Model,
  ModelResourceLimits,
  ModelRuntimeInstance,
  ModelResourceUsage,
  ModelDeployment,
  ModelMetrics,
  ModelLog,
  ModelEvent,
  ModelSummary,
  ModelTemplate,
  ModelAction
} from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

// Additional interfaces not in main types file
export interface ModelFilter {
  status?: string[];
  type?: string[];
  tags?: string[];
  author?: string;
  environment?: string;
  created_after?: string;
  created_before?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export const useModelManagement = () => {
  const [models, setModels] = useState<Model[]>([]);
  const [selectedModel, setSelectedModel] = useState<Model | null>(null);
  const [modelMetrics, setModelMetrics] = useState<Record<string, ModelMetrics[]>>({});
  const [modelLogs, setModelLogs] = useState<Record<string, ModelLog[]>>({});
  const [modelEvents, setModelEvents] = useState<ModelEvent[]>([]);
  const [templates, setTemplates] = useState<ModelTemplate[]>([]);
  const [summary, setSummary] = useState<ModelSummary | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // Fetch all models with optional filtering
  const fetchModels = useCallback(async (filter?: ModelFilter) => {
    setIsLoading(true);
    setError(null);
    
    try {
      let url = `${API_BASE_URL}/api/model-management/models`;
      
      if (filter) {
        const params = new URLSearchParams();
        if (filter.status?.length) params.append('status', filter.status.join(','));
        if (filter.type?.length) params.append('type', filter.type.join(','));
        if (filter.author) params.append('author', filter.author);
        if (filter.search) params.append('search', filter.search);
        if (filter.limit) params.append('limit', filter.limit.toString());
        if (filter.offset) params.append('offset', filter.offset.toString());
        
        if (params.toString()) {
          url += `?${params.toString()}`;
        }
      }
      
      const response: APIResponse<Model[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setModels(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch models');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch models:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch a specific model
  const fetchModel = useCallback(async (modelId: string) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/model-management/models/${modelId}`;
      const response: APIResponse<Model> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setSelectedModel(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch model');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch model:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Create a new model
  const createModel = useCallback(async (model: Partial<Model>): Promise<Model | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/model-management/models`;
      const response: APIResponse<Model> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(model),
      });
      
      if (response.success && response.data) {
        setModels(prevModels => [...prevModels, response.data!]);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create model');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create model:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Update an existing model
  const updateModel = useCallback(async (modelId: string, updates: Partial<Model>): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/model-management/models/${modelId}`;
      const response: APIResponse = await apiRequest(url, {
        method: 'PUT',
        body: JSON.stringify(updates),
      });
      
      if (response.success) {
        // Update the model in the local state
        setModels(prevModels => 
          prevModels.map(model => 
            model.id === modelId ? { ...model, ...updates } : model
          )
        );
        
        if (selectedModel?.id === modelId) {
          setSelectedModel(prev => prev ? { ...prev, ...updates } : null);
        }
        
        return true;
      } else {
        throw new Error(response.error || 'Failed to update model');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to update model:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [selectedModel]);

  // Delete an model
  const deleteModel = useCallback(async (modelId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/model-management/models/${modelId}`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });
      
      if (response.success) {
        setModels(prevModels => prevModels.filter(model => model.id !== modelId));
        
        if (selectedModel?.id === modelId) {
          setSelectedModel(null);
        }
        
        return true;
      } else {
        throw new Error(response.error || 'Failed to delete model');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to delete model:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [selectedModel]);

  // Execute an action on an model
  const executeModelAction = useCallback(async (modelId: string, action: ModelAction): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/model-management/models/${modelId}/actions`;
      const response: APIResponse = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      
      if (response.success) {
        // Refresh the model data after action
        await fetchModel(modelId);
        await fetchModels();
        return true;
      } else {
        throw new Error(response.error || 'Failed to execute action');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to execute model action:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [fetchModel, fetchModels]);

  // Convenience methods for common actions
  const deployModel = useCallback((modelId: string, parameters?: Record<string, any>) => 
    executeModelAction(modelId, { action: 'deploy', parameters }), [executeModelAction]);
  
  const startModel = useCallback((modelId: string, parameters?: Record<string, any>) => 
    executeModelAction(modelId, { action: 'start', parameters }), [executeModelAction]);
  
  const stopModel = useCallback((modelId: string) => 
    executeModelAction(modelId, { action: 'stop' }), [executeModelAction]);
  
  const restartModel = useCallback((modelId: string, parameters?: Record<string, any>) => 
    executeModelAction(modelId, { action: 'restart', parameters }), [executeModelAction]);

  // Fetch model metrics
  const fetchModelMetrics = useCallback(async (modelId: string, limit: number = 100) => {
    try {
      const url = `${API_BASE_URL}/api/model-management/models/${modelId}/metrics?limit=${limit}`;
      const response: APIResponse<ModelMetrics[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setModelMetrics(prev => ({ ...prev, [modelId]: response.data! }));
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch model metrics:', err);
    }
    return [];
  }, []);

  // Fetch model logs
  const fetchModelLogs = useCallback(async (modelId: string, limit: number = 100) => {
    try {
      const url = `${API_BASE_URL}/api/model-management/models/${modelId}/logs?limit=${limit}`;
      const response: APIResponse<ModelLog[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setModelLogs(prev => ({ ...prev, [modelId]: response.data! }));
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch model logs:', err);
    }
    return [];
  }, []);

  // Fetch model events
  const fetchModelEvents = useCallback(async (modelId?: string, limit: number = 100) => {
    try {
      const url = modelId 
        ? `${API_BASE_URL}/api/model-management/models/${modelId}/events?limit=${limit}`
        : `${API_BASE_URL}/api/model-management/events?limit=${limit}`;
      const response: APIResponse<ModelEvent[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setModelEvents(response.data);
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch model events:', err);
    }
    return [];
  }, []);

  // Fetch model templates
  const fetchTemplates = useCallback(async () => {
    try {
      const url = `${API_BASE_URL}/api/model-management/templates`;
      const response: APIResponse<ModelTemplate[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setTemplates(response.data);
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch templates:', err);
    }
    return [];
  }, []);

  // Create model template
  const createTemplate = useCallback(async (template: Partial<ModelTemplate>): Promise<ModelTemplate | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/model-management/templates`;
      const response: APIResponse<ModelTemplate> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(template),
      });
      
      if (response.success && response.data) {
        setTemplates(prev => [...prev, response.data!]);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create template');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create template:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch model summary
  const fetchSummary = useCallback(async () => {
    try {
      const url = `${API_BASE_URL}/api/model-management/summary`;
      const response: APIResponse<ModelSummary> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setSummary(response.data);
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch summary:', err);
    }
    return null;
  }, []);

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    // Set up event handlers
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
      if (data.connected) {
        console.log('Model Management WebSocket connected');
        setError(null);
      } else {
        console.log('Model Management WebSocket disconnected');
      }
    };

    const handleModelUpdate = (payload: any) => {
      // Update model in the list
      setModels(prevModels =>
        prevModels.map(model =>
          model.id === payload.id ? { ...model, ...payload } : model
        )
      );

      // Update selected model if it matches
      if (selectedModel?.id === payload.id) {
        setSelectedModel(prev => prev ? { ...prev, ...payload } : null);
      }
    };

    const handleModelStatusChange = (payload: any) => {
      // Update model status
      setModels(prevModels =>
        prevModels.map(model =>
          model.id === payload.model_id
            ? { ...model, status: payload.status }
            : model
        )
      );
    };

    const handleSystemNotification = (payload: any) => {
      console.log('Model Management system notification:', payload);
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    webSocketService.on('model-updated', handleModelUpdate);
    webSocketService.on('model-status-changed', handleModelStatusChange);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe(['model-updated', 'model-status-changed', 'system-notification']);

    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Return cleanup function
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('model-updated', handleModelUpdate);
      webSocketService.off('model-status-changed', handleModelStatusChange);
      webSocketService.off('system-notification', handleSystemNotification);
    };
  }, [selectedModel]);

  const disconnectWebSocket = useCallback(() => {
    // Individual hooks don't disconnect the shared service
    setIsConnected(false);
  }, []);

  // Refresh all data
  const refreshAll = useCallback(async () => {
    await Promise.all([
      fetchModels(),
      fetchTemplates(),
      fetchSummary(),
      fetchModelEvents(),
    ]);
  }, [fetchModels, fetchTemplates, fetchSummary, fetchModelEvents]);

  // Initial fetch and WebSocket connection on mount
  useEffect(() => {
    fetchModels();
    fetchSummary();
    return connectWebSocket();
  }, [fetchModels, fetchSummary, connectWebSocket]);

  return {
    models,
    selectedModel,
    modelMetrics,
    modelLogs,
    modelEvents,
    templates,
    summary,
    isLoading,
    error,
    isConnected,
    fetchModels,
    fetchModel,
    createModel,
    updateModel,
    deleteModel,
    executeModelAction,
    deployModel,
    startModel,
    stopModel,
    restartModel,
    fetchModelMetrics,
    fetchModelLogs,
    fetchModelEvents,
    fetchTemplates,
    createTemplate,
    fetchSummary,
    refreshAll,
    connectWebSocket,
    disconnectWebSocket,
    setSelectedModel,
  };
};

export default useModelManagement;
