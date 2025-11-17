"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  DVENode,
  DVENodeFilter,
  RegisterNodeRequest,
  APIResponse,
  DVENodeUpdate,
  TEEEndpoint,
  SSHSession,
  ValidationSession,
  ErrorResolutionSession
} from '@/types/api';
import { apiRequest, API_BASE_URL, buildQueryString } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export const useDVENodes = () => {
  const [nodes, setNodes] = useState<DVENode[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [socket, setSocket] = useState<WebSocket | null>(null);

  // Fetch DVE nodes with optional filtering
  const fetchNodes = useCallback(async (filter?: DVENodeFilter) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const queryString = buildQueryString(filter || {});
      const url = `${API_BASE_URL}/api/dve-nodes${queryString}`;
      console.log('[DVE Nodes] Fetching from:', url);
      console.log('[DVE Nodes] Filter:', filter);
      
      const response: APIResponse<DVENode[]> = await apiRequest(url, { method: 'GET' });
      console.log('[DVE Nodes] Raw API Response:', JSON.stringify(response, null, 2));
      
      if (response.success) {
        if (Array.isArray(response.data)) {
          console.log('[DVE Nodes] Received nodes count:', response.data.length);
          console.log('[DVE Nodes] Nodes data:', response.data);
          setNodes(response.data);
        } else {
          console.warn('[DVE Nodes] Response data is not an array:', typeof response.data, response.data);
          // If data is not an array but success is true, set empty array
          setNodes([]);
        }
      } else {
        console.error('[DVE Nodes] API returned error:', response.error);
        throw new Error(response.error || 'Failed to fetch DVE nodes');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('[DVE Nodes] Exception during fetch:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Get a specific DVE node by ID
  const getNode = useCallback(async (nodeId: string): Promise<DVENode | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}`;
      const response: APIResponse<DVENode> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch DVE node');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch DVE node:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Register a new DVE node
  const registerNode = useCallback(async (nodeData: RegisterNodeRequest): Promise<DVENode | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-nodes`;
      const response: APIResponse<DVENode> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(nodeData),
      });

      if (response.success && response.data && !Array.isArray(response.data)) {
        // Add the new node to the current list
        setNodes(prevNodes => [...prevNodes, response.data as DVENode]);
        return response.data as DVENode;
      } else {
        throw new Error(response.error || 'Failed to register DVE node');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to register DVE node:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Update a DVE node
  const updateNode = useCallback(async (nodeId: string, updates: Partial<DVENode>): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}`;
      const response: APIResponse<DVENode> = await apiRequest(url, {
        method: 'PUT',
        body: JSON.stringify(updates),
      });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        // Update the node in the current list
        setNodes(prevNodes => 
          prevNodes.map(node => 
            node.id === nodeId ? response.data as DVENode : node
          )
        );
        return true;
      } else {
        throw new Error(response.error || 'Failed to update DVE node');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to update DVE node:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Delete a DVE node
  const deleteNode = useCallback(async (nodeId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });
      
      if (response.success) {
        // Remove the node from the current list
        setNodes(prevNodes => prevNodes.filter(node => node.id !== nodeId));
        return true;
      } else {
        throw new Error(response.error || 'Failed to delete DVE node');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to delete DVE node:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    if (webSocketService.getConnectionStatus()) {
      setIsConnected(true);
      return;
    }

    // Set up event handlers
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
      if (data.connected) {
        console.log('DVE Nodes WebSocket connected');
        setError(null);
      }
    };

    const handleDVENodeUpdate = (payload: any) => {
      console.log('[DVE Nodes] WebSocket update received:', payload);
      // Update specific node in the list, or add if it doesn't exist
      setNodes(prevNodes => {
        const existingIndex = prevNodes.findIndex(node => node.id === payload.id);
        if (existingIndex >= 0) {
          // Update existing node
          return prevNodes.map(node =>
            node.id === payload.id
              ? {
                  ...node,
                  cpu_usage: payload.cpu_usage || node.cpu_usage,
                  memory_usage: payload.memory_usage || node.memory_usage,
                  status: payload.status || node.status,
                  last_heartbeat: payload.last_heartbeat || node.last_heartbeat
                }
              : node
          );
        } else if (payload.id && payload.name) {
          // Add new node if it has required fields
          console.log('[DVE Nodes] Adding new node from WebSocket:', payload);
          return [...prevNodes, payload as DVENode];
        }
        return prevNodes;
      });
    };

    const handleSystemNotification = (payload: any) => {
      console.log('[DVE Nodes] System notification:', payload);
    };

    const handleDVENodeDiscovered = (payload: any) => {
      console.log('[DVE Nodes] Node discovered via WebSocket:', payload);
      // Add newly discovered node to the list
      if (payload.id && payload.name) {
        setNodes(prevNodes => {
          const exists = prevNodes.some(node => node.id === payload.id);
          if (!exists) {
            return [...prevNodes, payload as DVENode];
          }
          return prevNodes;
        });
      }
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    webSocketService.on('dve-node-updated', handleDVENodeUpdate);
    webSocketService.on('dve-node-discovered', handleDVENodeDiscovered);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe(['dve-node-updated', 'dve-node-discovered', 'system-notification']);

    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Return cleanup function
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('dve-node-updated', handleDVENodeUpdate);
      webSocketService.off('dve-node-discovered', handleDVENodeDiscovered);
      webSocketService.off('system-notification', handleSystemNotification);
    };
  }, []);

  const disconnectWebSocket = useCallback(() => {
    // Individual hooks don't disconnect the shared service
    setIsConnected(false);
  }, []);

  // Update node status specifically
  const updateNodeStatus = useCallback(async (nodeId: string, status: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}/status`;
      const response: APIResponse<DVENode> = await apiRequest(url, {
        method: 'PUT',
        body: JSON.stringify({ status }),
      });

      if (response.success && response.data && !Array.isArray(response.data)) {
        // Update the node in the current list
        setNodes(prevNodes =>
          prevNodes.map(node =>
            node.id === nodeId ? response.data as DVENode : node
          )
        );
        return true;
      } else {
        throw new Error(response.error || 'Failed to update node status');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to update node status:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // ⭐ NEW: Get all endpoints for a specific DVE node
  const getNodeEndpoints = useCallback(async (nodeId: string): Promise<TEEEndpoint[]> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}/endpoints`;
      const response: APIResponse<any> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data) {
        // Backend returns a map object, convert it to array of TEEEndpoint objects
        const endpointsMap = response.data;
        const endpoints: TEEEndpoint[] = [];
        
        // Convert SSH endpoint
        if (endpointsMap.ssh && endpointsMap.ssh.host) {
          endpoints.push({
            id: `${nodeId}-ssh`,
            rental_id: '', // Will be populated when node is rented
            container_id: '', // Will be populated when container is created
            endpoint_type: 'ssh',
            host: endpointsMap.ssh.host,
            port: endpointsMap.ssh.port || 22,
            protocol: 'ssh',
            status: 'active',
            created_at: new Date().toISOString(),
            expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(), // 24 hours
          });
        }
        
        // Convert validation endpoint
        if (endpointsMap.validation && endpointsMap.validation.host) {
          endpoints.push({
            id: `${nodeId}-validation`,
            rental_id: '', // Will be populated when node is rented
            container_id: '', // Will be populated when container is created
            endpoint_type: 'validation',
            host: endpointsMap.validation.host,
            port: endpointsMap.validation.port || 8080,
            protocol: 'http',
            status: 'active',
            created_at: new Date().toISOString(),
            expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(), // 24 hours
          });
        }
        
        // Convert error resolution endpoint
        if (endpointsMap.error_resolution && endpointsMap.error_resolution.host) {
          endpoints.push({
            id: `${nodeId}-error-resolution`,
            rental_id: '', // Will be populated when node is rented
            container_id: '', // Will be populated when container is created
            endpoint_type: 'error-resolution',
            host: endpointsMap.error_resolution.host,
            port: endpointsMap.error_resolution.port || 8081,
            protocol: 'http',
            status: 'active',
            created_at: new Date().toISOString(),
            expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(), // 24 hours
          });
        }
        
        return endpoints;
      } else {
        throw new Error(response.error || 'Failed to fetch node endpoints');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch node endpoints:', err);
      return [];
    } finally {
      setIsLoading(false);
    }
  }, []);

  // ⭐ NEW: Get SSH endpoint for a specific DVE node
  const getNodeSSHEndpoint = useCallback(async (nodeId: string): Promise<TEEEndpoint | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}/ssh-endpoint`;
      const response: APIResponse<TEEEndpoint> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch SSH endpoint');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch SSH endpoint:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // ⭐ NEW: Get validation endpoint for a specific DVE node
  const getNodeValidationEndpoint = useCallback(async (nodeId: string): Promise<TEEEndpoint | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}/validation-endpoint`;
      const response: APIResponse<TEEEndpoint> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch validation endpoint');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch validation endpoint:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // ⭐ NEW: Get error resolution endpoint for a specific DVE node
  const getNodeErrorResolutionEndpoint = useCallback(async (nodeId: string): Promise<TEEEndpoint | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}/error-resolution-endpoint`;
      const response: APIResponse<TEEEndpoint> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch error resolution endpoint');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch error resolution endpoint:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Convenience methods for common operations
  const getOnlineNodes = useCallback(() => fetchNodes({ status: 'online' }), [fetchNodes]);
  const getNodesByTEE = useCallback((teeType: string) => fetchNodes({ tee_type: teeType }), [fetchNodes]);
  const refreshNodes = useCallback(() => fetchNodes(), [fetchNodes]);

  // Initial fetch and WebSocket connection on mount
  useEffect(() => {
    fetchNodes();
    return connectWebSocket();
  }, [fetchNodes, connectWebSocket]);

  return {
    nodes,
    isLoading,
    error,
    isConnected,
    fetchNodes,
    getNode,
    registerNode,
    updateNode,
    updateNodeStatus,
    deleteNode,
    getOnlineNodes,
    getNodesByTEE,
    refreshNodes,
    connectWebSocket,
    disconnectWebSocket,
    // ⭐ NEW endpoint methods
    getNodeEndpoints,
    getNodeSSHEndpoint,
    getNodeValidationEndpoint,
    getNodeErrorResolutionEndpoint,
  };
};

export default useDVENodes;
