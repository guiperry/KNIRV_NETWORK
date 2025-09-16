"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  DVENode,
  DVENodeFilter,
  RegisterNodeRequest,
  APIResponse,
  DVENodeUpdate
} from '@/types/api';
import { apiRequest, API_BASE_URL, buildQueryString } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export const useDVENodes = () => {
  const [nodes, setNodes] = useState<DVENode[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [socket, setSocket] = useState<StandardWebSocket | null>(null);

  // Fetch DVE nodes with optional filtering
  const fetchNodes = useCallback(async (filter?: DVENodeFilter) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const queryString = buildQueryString(filter || {});
      const url = `${API_BASE_URL}/api/dve-nodes${queryString}`;
      const response: APIResponse<DVENode[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && Array.isArray(response.data)) {
        setNodes(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch DVE nodes');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch DVE nodes:', err);
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
      // Update specific node in the list
      setNodes(prevNodes =>
        prevNodes.map(node =>
          node.id === payload.id
            ? {
                ...node,
                cpu_usage: payload.cpu_usage || node.cpu_usage,
                memory_usage: payload.memory_usage || node.memory_usage,
                status: payload.status || node.status,
                last_heartbeat: payload.last_heartbeat || node.last_heartbeat
              }
            : node
        )
      );
    };

    const handleSystemNotification = (payload: any) => {
      console.log('DVE Nodes system notification:', payload);
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    webSocketService.on('dve-node-updated', handleDVENodeUpdate);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe(['dve-node-updated', 'system-notification']);

    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Return cleanup function
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('dve-node-updated', handleDVENodeUpdate);
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
  };
};

export default useDVENodes;
