"use client";

import { useEffect, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  DVENode,
  DVENodeFilter,
  RegisterNodeRequest,
  APIResponse,
  TEEEndpoint
} from '@/types/api';
import { apiRequest, API_BASE_URL, buildQueryString } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export const useDVENodes = (filter?: DVENodeFilter) => {
  const queryClient = useQueryClient();
  const queryString = buildQueryString(filter || {});
  const queryKey = ['dve-nodes', queryString];

  // Fetch DVE nodes with React Query
  const {
    data: nodes = [],
    isLoading,
    error: queryError,
    refetch: fetchNodes
  } = useQuery<DVENode[]>({
    queryKey,
    queryFn: async () => {
      const url = `${API_BASE_URL}/api/dve-nodes${queryString}`;
      const response: APIResponse<DVENode[]> = await apiRequest(url, { method: 'GET' });
      if (response.success && Array.isArray(response.data)) {
        return response.data;
      }
      throw new Error(response.error || 'Failed to fetch DVE nodes');
    },
    staleTime: 30000, // 30 seconds
  });

  const error = queryError instanceof Error ? queryError.message : null;

  // WebSocket connection and real-time cache updates
  useEffect(() => {
    const handleDVENodeUpdate = (payload: any) => {
      queryClient.setQueryData<DVENode[]>(queryKey, (prevNodes = []) => {
        const existingIndex = prevNodes.findIndex(node => node.id === payload.id);
        if (existingIndex >= 0) {
          return prevNodes.map(node =>
            node.id === payload.id
              ? {
                  ...node,
                  ...payload,
                  // Ensure we don't overwrite with undefined if WebSocket payload is partial
                  cpu_usage: payload.cpu_usage ?? node.cpu_usage,
                  memory_usage: payload.memory_usage ?? node.memory_usage,
                  status: payload.status ?? node.status,
                  last_heartbeat: payload.last_heartbeat ?? node.last_heartbeat
                }
              : node
          );
        } else if (payload.id && payload.name) {
          return [...prevNodes, payload as DVENode];
        }
        return prevNodes;
      });
    };

    const handleDVENodeDiscovered = (payload: any) => {
      queryClient.setQueryData<DVENode[]>(queryKey, (prevNodes = []) => {
        const exists = prevNodes.some(node => node.id === payload.id);
        if (!exists && payload.id && payload.name) {
          return [...prevNodes, payload as DVENode];
        }
        return prevNodes;
      });
    };

    webSocketService.on('dve-node-updated', handleDVENodeUpdate);
    webSocketService.on('dve-node-discovered', handleDVENodeDiscovered);
    webSocketService.subscribe(['dve-node-updated', 'dve-node-discovered']);

    return () => {
      webSocketService.off('dve-node-updated', handleDVENodeUpdate);
      webSocketService.off('dve-node-discovered', handleDVENodeDiscovered);
    };
  }, [queryClient, queryKey]);

  // Mutations
  const registerNodeMutation = useMutation({
    mutationFn: async (nodeData: RegisterNodeRequest) => {
      const url = `${API_BASE_URL}/api/dve-nodes`;
      const response: APIResponse<DVENode> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(nodeData),
      });
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to register DVE node');
    },
    onSuccess: (newNode) => {
      queryClient.setQueryData<DVENode[]>(queryKey, (prev = []) => [...prev, newNode]);
    }
  });

  const updateNodeMutation = useMutation({
    mutationFn: async ({ nodeId, updates }: { nodeId: string, updates: Partial<DVENode> }) => {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}`;
      const response: APIResponse<DVENode> = await apiRequest(url, {
        method: 'PUT',
        body: JSON.stringify(updates),
      });
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to update DVE node');
    },
    onSuccess: (updatedNode) => {
      queryClient.setQueryData<DVENode[]>(queryKey, (prev = []) => 
        prev.map(node => node.id === updatedNode.id ? updatedNode : node)
      );
    }
  });

  const deleteNodeMutation = useMutation({
    mutationFn: async (nodeId: string) => {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });
      if (response.success) return nodeId;
      throw new Error(response.error || 'Failed to delete DVE node');
    },
    onSuccess: (nodeId) => {
      queryClient.setQueryData<DVENode[]>(queryKey, (prev = []) => 
        prev.filter(node => node.id !== nodeId)
      );
    }
  });

  // Endpoints (Async helpers, not cached via useQuery for now as they are on-demand)
  const getNodeEndpoints = useCallback(async (nodeId: string): Promise<TEEEndpoint[]> => {
    try {
      const url = `${API_BASE_URL}/api/dve-nodes/${nodeId}/endpoints`;
      const response: APIResponse<any> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data) {
        const endpointsMap = response.data;
        const endpoints: TEEEndpoint[] = [];
        
        if (endpointsMap.ssh?.host) {
          endpoints.push({
            id: `${nodeId}-ssh`,
            rental_id: '',
            container_id: '',
            endpoint_type: 'ssh',
            host: endpointsMap.ssh.host,
            port: endpointsMap.ssh.port || 22,
            protocol: 'ssh',
            status: 'active',
            created_at: new Date().toISOString(),
            expires_at: new Date(Date.now() + 86400000).toISOString(),
          });
        }
        
        if (endpointsMap.validation?.host) {
          endpoints.push({
            id: `${nodeId}-validation`,
            rental_id: '',
            container_id: '',
            endpoint_type: 'validation',
            host: endpointsMap.validation.host,
            port: endpointsMap.validation.port || 8080,
            protocol: 'http',
            status: 'active',
            created_at: new Date().toISOString(),
            expires_at: new Date(Date.now() + 86400000).toISOString(),
          });
        }
        
        return endpoints;
      }
      return [];
    } catch (err) {
      console.error('Failed to fetch node endpoints:', err);
      return [];
    }
  }, []);

  return {
    nodes,
    isLoading,
    error,
    isConnected: webSocketService.getConnectionStatus(),
    fetchNodes,
    registerNode: registerNodeMutation.mutateAsync,
    updateNode: (nodeId: string, updates: Partial<DVENode>) => updateNodeMutation.mutateAsync({ nodeId, updates }),
    deleteNode: deleteNodeMutation.mutateAsync,
    refreshNodes: fetchNodes,
    getNodeEndpoints,
    // Add other missing methods as needed
    getOnlineNodes: () => fetchNodes(),
    getNodesByTEE: (_teeType: string) => fetchNodes(),
  };
};

export default useDVENodes;
