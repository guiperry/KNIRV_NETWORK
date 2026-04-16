"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  APIResponse,
  KnowledgeBase,
  KnowledgeBaseResourceLimits,
  KnowledgeBaseRuntimeInstance,
  KnowledgeBaseResourceUsage,
  KnowledgeBaseDeployment,
  KnowledgeBaseMetrics,
  KnowledgeBaseLog,
  KnowledgeBaseEvent,
  KnowledgeBaseSummary,
  KnowledgeBaseTemplate,
  KnowledgeBaseAction,
  // Also import aliases for backward compatibility
  Fabric,
  FabricResourceLimits,
  FabricRuntimeInstance,
  FabricResourceUsage,
  FabricDeployment,
  FabricMetrics,
  FabricLog,
  FabricEvent,
  FabricSummary,
  FabricTemplate,
  FabricAction
} from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

// Additional interfaces not in main types file
export interface FabricFilter {
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

export const useFabricManagement = () => {
  const [fabrics, setFabrics] = useState<Fabric[]>([]);
  const [selectedFabric, setSelectedFabric] = useState<Fabric | null>(null);
  const [fabricMetrics, setFabricMetrics] = useState<Record<string, FabricMetrics[]>>({});
  const [fabricLogs, setFabricLogs] = useState<Record<string, FabricLog[]>>({});
  const [fabricEvents, setFabricEvents] = useState<FabricEvent[]>([]);
  const [templates, setTemplates] = useState<FabricTemplate[]>([]);
  const [summary, setSummary] = useState<FabricSummary | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // Fetch all knowledge bases with optional filtering
  // Backend API uses /api/v1/knowledge-base/ as source of truth
  const fetchFabrics = useCallback(async (filter?: FabricFilter) => {
    setIsLoading(true);
    setError(null);
    
    try {
      let url = `${API_BASE_URL}/api/v1/knowledge-base/objects`;
      
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
      
      const response: APIResponse<Fabric[]> = await apiRequest(url, { method: 'GET' });

      if (response.success) {
        setFabrics(response.data || []);
      } else {
        throw new Error(response.error || 'Failed to fetch fabric items');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch fabric items:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch a specific knowledge base item
  const fetchFabric = useCallback(async (fabricId: string) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/objects/${fabricId}`;
      const response: APIResponse<Fabric> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setSelectedFabric(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch fabric item');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch fabric item:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Create a new fabric item
  const createFabric = useCallback(async (fabric: Partial<Fabric>): Promise<Fabric | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/objects`;
      const response: APIResponse<KnowledgeBase> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(fabric),
      });
      
      if (response.success && response.data) {
        setFabrics(prevFabrics => [...prevFabrics, response.data!]);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create fabric item');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create fabric item:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Update an existing knowledge base item
  const updateFabric = useCallback(async (fabricId: string, updates: Partial<Fabric>): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/objects/${fabricId}`;
      const response: APIResponse = await apiRequest(url, {
        method: 'PUT',
        body: JSON.stringify(updates),
      });
      
      if (response.success) {
        // Update the fabric in the local state
        setFabrics(prevFabrics => 
          prevFabrics.map(fabric => 
            fabric.id === fabricId ? { ...fabric, ...updates } : fabric
          )
        );
        
        if (selectedFabric?.id === fabricId) {
          setSelectedFabric(prev => prev ? { ...prev, ...updates } : null);
        }
        
        return true;
      } else {
        throw new Error(response.error || 'Failed to update fabric item');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to update fabric item:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [selectedFabric]);

  // Delete a knowledge base item
  const deleteFabric = useCallback(async (fabricId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/objects/${fabricId}`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });
      
      if (response.success) {
        setFabrics(prevFabrics => prevFabrics.filter(fabric => fabric.id !== fabricId));
        
        if (selectedFabric?.id === fabricId) {
          setSelectedFabric(null);
        }
        
        return true;
      } else {
        throw new Error(response.error || 'Failed to delete fabric item');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to delete fabric item:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [selectedFabric]);

  // Execute an action on a knowledge base item
  const executeFabricAction = useCallback(async (fabricId: string, action: FabricAction): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/objects/${fabricId}/actions`;
      const response: APIResponse = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      
      if (response.success) {
        // Refresh the fabric data after action
        await fetchFabric(fabricId);
        await fetchFabrics();
        return true;
      } else {
        throw new Error(response.error || 'Failed to execute action');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to execute fabric action:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [fetchFabric, fetchFabrics]);

  // Convenience methods for common actions
  const deployFabric = useCallback((fabricId: string, parameters?: Record<string, any>) => 
    executeFabricAction(fabricId, { action: 'deploy', parameters }), [executeFabricAction]);
  
  const startFabric = useCallback((fabricId: string, parameters?: Record<string, any>) => 
    executeFabricAction(fabricId, { action: 'start', parameters }), [executeFabricAction]);
  
  const stopFabric = useCallback((fabricId: string) => 
    executeFabricAction(fabricId, { action: 'stop' }), [executeFabricAction]);
  
  const restartFabric = useCallback((fabricId: string, parameters?: Record<string, any>) => 
    executeFabricAction(fabricId, { action: 'restart', parameters }), [executeFabricAction]);

  // Fetch knowledge base metrics
  const fetchFabricMetrics = useCallback(async (fabricId: string, limit: number = 100) => {
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/objects/${fabricId}/metrics?limit=${limit}`;
      const response: APIResponse<FabricMetrics[]> = await apiRequest(url, { method: 'GET' });

      if (response.success) {
        const data = response.data || [];
        setFabricMetrics(prev => ({ ...prev, [fabricId]: data }));
        return data;
      }
    } catch (err) {
      console.error('Failed to fetch fabric metrics:', err);
    }
    return [];
  }, []);

  // Fetch knowledge base logs
  const fetchFabricLogs = useCallback(async (fabricId: string, limit: number = 100) => {
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/objects/${fabricId}/logs?limit=${limit}`;
      const response: APIResponse<FabricLog[]> = await apiRequest(url, { method: 'GET' });

      if (response.success) {
        const data = response.data || [];
        setFabricLogs(prev => ({ ...prev, [fabricId]: data }));
        return data;
      }
    } catch (err) {
      console.error('Failed to fetch fabric logs:', err);
    }
    return [];
  }, []);

  // Fetch knowledge base events
  const fetchFabricEvents = useCallback(async (fabricId?: string, limit: number = 100) => {
    try {
      const url = fabricId 
        ? `${API_BASE_URL}/api/v1/knowledge-base/objects/${fabricId}/events?limit=${limit}`
        : `${API_BASE_URL}/api/v1/knowledge-base/events?limit=${limit}`;
      const response: APIResponse<FabricEvent[]> = await apiRequest(url, { method: 'GET' });

      if (response.success) {
        const data = response.data || [];
        setFabricEvents(data);
        return data;
      }
    } catch (err) {
      console.error('Failed to fetch fabric events:', err);
    }
    return [];
  }, []);

  // Fetch knowledge base templates
  const fetchTemplates = useCallback(async () => {
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/templates`;
      const response: APIResponse<FabricTemplate[]> = await apiRequest(url, { method: 'GET' });

      if (response.success) {
        const data = response.data || [];
        setTemplates(data);
        return data;
      }
    } catch (err) {
      console.error('Failed to fetch templates:', err);
    }
    return [];
  }, []);

  // Create knowledge base template
  const createTemplate = useCallback(async (template: Partial<FabricTemplate>): Promise<FabricTemplate | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/templates`;
      const response: APIResponse<FabricTemplate> = await apiRequest(url, {
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

  // Fetch knowledge base summary
  const fetchSummary = useCallback(async () => {
    try {
      const url = `${API_BASE_URL}/api/v1/knowledge-base/summary`;
      const response: APIResponse<FabricSummary> = await apiRequest(url, { method: 'GET' });
      
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
        console.log('Fabric Management WebSocket connected');
        setError(null);
      } else {
        console.log('Fabric Management WebSocket disconnected');
      }
    };

    const handleFabricUpdate = (payload: any) => {
      // Update fabric in the list
      setFabrics(prevFabrics =>
        prevFabrics.map(fabric =>
          fabric.id === payload.id ? { ...fabric, ...payload } : fabric
        )
      );

      // Update selected fabric if it matches
      if (selectedFabric?.id === payload.id) {
        setSelectedFabric(prev => prev ? { ...prev, ...payload } : null);
      }
    };

    const handleFabricStatusChange = (payload: any) => {
      // Update fabric status
      setFabrics(prevFabrics =>
        prevFabrics.map(fabric =>
          fabric.id === payload.model_id
            ? { ...fabric, status: payload.status }
            : fabric
        )
      );
    };

    const handleSystemNotification = (payload: any) => {
      console.log('Fabric Management system notification:', payload);
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    // Note: Assuming backend sends 'model-updated' etc., we still listen to those events
    // Ideally backend should also be updated to 'fabric-updated'
    webSocketService.on('model-updated', handleFabricUpdate);
    webSocketService.on('model-status-changed', handleFabricStatusChange);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe(['model-updated', 'model-status-changed', 'system-notification']);

    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Return cleanup function
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('model-updated', handleFabricUpdate);
      webSocketService.off('model-status-changed', handleFabricStatusChange);
      webSocketService.off('system-notification', handleSystemNotification);
    };
  }, [selectedFabric]);

  const disconnectWebSocket = useCallback(() => {
    // Individual hooks don't disconnect the shared service
    setIsConnected(false);
  }, []);

  // Refresh all data
  const refreshAll = useCallback(async () => {
    await Promise.all([
      fetchFabrics(),
      fetchTemplates(),
      fetchSummary(),
      fetchFabricEvents(),
    ]);
  }, [fetchFabrics, fetchTemplates, fetchSummary, fetchFabricEvents]);

  // Initial fetch and WebSocket connection on mount
  useEffect(() => {
    fetchFabrics();
    fetchSummary();
    return connectWebSocket();
  }, [fetchFabrics, fetchSummary, connectWebSocket]);

  return {
    fabrics,
    selectedFabric,
    fabricMetrics,
    fabricLogs,
    fabricEvents,
    templates,
    summary,
    isLoading,
    error,
    isConnected,
    fetchFabrics,
    fetchFabric,
    createFabric,
    updateFabric,
    deleteFabric,
    executeFabricAction,
    deployFabric,
    startFabric,
    stopFabric,
    restartFabric,
    fetchFabricMetrics,
    fetchFabricLogs,
    fetchFabricEvents,
    fetchTemplates,
    createTemplate,
    fetchSummary,
    refreshAll,
    connectWebSocket,
    disconnectWebSocket,
    setSelectedFabric,
  };
};

export default useFabricManagement;
