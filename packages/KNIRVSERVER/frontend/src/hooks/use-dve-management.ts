"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  APIResponse,
  DVECreation,
  DVECreationRequest,
  DVECreationStats,
  DVEAccessInfo,
  SSHSession,
  ValidationSession,
  ErrorResolutionSession
} from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export const useDVEManagement = () => {
  const [creations, setCreations] = useState<DVECreation[]>([]);
  const [stats, setStats] = useState<DVECreationStats | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // Fetch user's DVE creations
  const fetchCreations = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-creation/nodes`;
      const response: APIResponse<DVECreation[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && Array.isArray(response.data)) {
        setCreations(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch creations');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch creations:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch DVE creation statistics
  const fetchStats = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-creation/stats`;
      const response: APIResponse<DVECreationStats> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        setStats(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch DVE stats');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch DVE stats:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Create a new DVE (Sovereign Creation)
  const createDVE = useCallback(async (request: DVECreationRequest): Promise<DVECreation | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-creation/nodes`;
      const response: APIResponse<any> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(request),
      });
      
      if (response.success && response.data) {
        // Backend returns DVECreationResponse which contains dve_creation
        const creation = response.data.dve_creation as DVECreation;
        if (creation) {
          setCreations(prev => [...prev, creation]);
          return creation;
        }
        return null;
      } else {
        throw new Error(response.error || 'Failed to create DVE');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create DVE:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // WebSocket connection management — subscribe to real backend events
  const connectWebSocket = useCallback(() => {
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
    };

    // The backend broadcasts 'dve-node-updated' and 'dve-node-discovered'
    // from its periodic health/status sweep. When these arrive it means
    // something DVE-related changed, so re-fetch our creation list.
    const handleNodeEvent = () => {
      fetchCreations();
    };

    webSocketService.on('connection', handleConnection);
    webSocketService.on('dve-node-updated', handleNodeEvent);
    webSocketService.on('dve-node-discovered', handleNodeEvent);
    webSocketService.subscribe(['dve-node-updated', 'dve-node-discovered']);

    setIsConnected(webSocketService.getConnectionStatus());

    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('dve-node-updated', handleNodeEvent);
      webSocketService.off('dve-node-discovered', handleNodeEvent);
    };
  }, [fetchCreations]);

  const disconnectWebSocket = useCallback(() => {
    setIsConnected(false);
  }, []);

  // Periodic polling as a fallback — re-fetches every 15 seconds so
  // background provisioning (pending → active) shows up without
  // manual "Sync" clicks.  Cancelled on unmount.
  useEffect(() => {
    const interval = setInterval(() => {
      fetchCreations();
      fetchStats();
    }, 15_000);
    return () => clearInterval(interval);
  }, [fetchCreations, fetchStats]);

  // Initial fetch on mount
  useEffect(() => {
    fetchCreations();
    fetchStats();
    return connectWebSocket();
  }, [fetchCreations, fetchStats, connectWebSocket]);

  // Get full access information for a creation
  const getFullAccessInfo = useCallback(async (creationId: string): Promise<DVEAccessInfo | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-creation/nodes/${creationId}/access`;
      const response: APIResponse<DVEAccessInfo> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch access info');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    creations,
    stats,
    isLoading,
    error,
    isConnected,
    fetchCreations,
    fetchStats,
    createDVE,
    getFullAccessInfo,
    connectWebSocket,
    disconnectWebSocket,
  };
};

export default useDVEManagement;
