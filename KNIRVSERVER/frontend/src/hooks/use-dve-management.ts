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

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
    };

    const handleDVEUpdate = (payload: any) => {
      setCreations(prev =>
        prev.map(c => (c.id === payload.id ? { ...c, ...payload } : c))
      );
    };

    webSocketService.on('connection', handleConnection);
    webSocketService.on('dve-updated', handleDVEUpdate);
    webSocketService.subscribe(['dve-updated']);

    setIsConnected(webSocketService.getConnectionStatus());

    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('dve-updated', handleDVEUpdate);
    };
  }, []);

  const disconnectWebSocket = useCallback(() => {
    setIsConnected(false);
  }, []);

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

  // Initial fetch on mount
  useEffect(() => {
    fetchCreations();
    fetchStats();
    return connectWebSocket();
  }, [fetchCreations, fetchStats, connectWebSocket]);

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
