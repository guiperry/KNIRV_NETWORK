"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  APIResponse,
  DVERentalPlan,
  DVERental,
  DVERentalStats,
  RentalRequest,
  ExtendRentalRequest,
  DVEAccessInfo,
  SSHSession,
  ValidationSession,
  ErrorResolutionSession
} from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export const useDVERental = () => {
  const [plans, setPlans] = useState<DVERentalPlan[]>([]);
  const [rentals, setRentals] = useState<DVERental[]>([]);
  const [stats, setStats] = useState<DVERentalStats | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // Fetch available services
  const fetchPlans = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve/services`;
      console.log('[DVE] Fetching services from:', url);
      const response: APIResponse<DVERentalPlan[]> = await apiRequest(url, { method: 'GET' });
      console.log('[DVE] Services response:', response);

      if (response.success && Array.isArray(response.data)) {
        console.log('[DVE] Loaded services:', response.data.length, response.data);
        setPlans(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch available services');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch available services:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch user's DVE instances
  const fetchRentals = useCallback(async (userId?: string) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const queryParams = userId ? `?user_id=${userId}` : '';
      const url = `${API_BASE_URL}/api/dve/instances${queryParams}`;
      const response: APIResponse<DVERental[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && Array.isArray(response.data)) {
        setRentals(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch rentals');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch rentals:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch DVE statistics
  const fetchStats = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve/stats`;
      const response: APIResponse<DVERentalStats> = await apiRequest(url, { method: 'GET' });
      
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

  // Create a new rental
  const createRental = useCallback(async (rentalRequest: RentalRequest): Promise<DVERental | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve/instances`;
      const response: APIResponse<DVERental> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(rentalRequest),
      });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        // Add the new rental to the current list
        setRentals(prevRentals => [...prevRentals, response.data as DVERental]);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create rental');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create rental:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Extend an existing rental
  const extendRental = useCallback(async (
    rentalId: string, 
    additionalDuration: number, 
    paymentTxHash: string
  ): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve/instances/${rentalId}/extend`;
      const response: APIResponse = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify({
          additional_duration: additionalDuration,
          payment_tx_hash: paymentTxHash,
        }),
      });
      
      if (response.success) {
        // Refresh rentals to get updated data
        await fetchRentals();
        return true;
      } else {
        throw new Error(response.error || 'Failed to extend rental');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to extend rental:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [fetchRentals]);

  // Cancel a rental
  const cancelRental = useCallback(async (rentalId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve/instances/${rentalId}`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });
      
      if (response.success) {
        // Remove the rental from the current list
        setRentals(prevRentals => prevRentals.filter(rental => rental.id !== rentalId));
        return true;
      } else {
        throw new Error(response.error || 'Failed to cancel rental');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to cancel rental:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    // Set up event handlers
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
      if (data.connected) {
        console.log('DVE Rental WebSocket connected');
        setError(null);
      } else {
        console.log('DVE Rental WebSocket disconnected');
      }
    };

    const handleDVERentalUpdate = (payload: any) => {
      // Update specific rental in the list
      setRentals(prevRentals =>
        prevRentals.map(rental =>
          rental.id === payload.id
            ? { ...rental, ...payload }
            : rental
        )
      );
    };

    const handleDVERentalExpired = (payload: any) => {
      // Mark rental as expired
      setRentals(prevRentals =>
        prevRentals.map(rental =>
          rental.id === payload.id
            ? { ...rental, status: 'expired' }
            : rental
        )
      );
    };

    const handleSystemNotification = (payload: any) => {
      console.log('DVE Rental system notification:', payload);
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    webSocketService.on('dve-updated', handleDVERentalUpdate);
    webSocketService.on('dve-expired', handleDVERentalExpired);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe(['dve-updated', 'dve-expired', 'system-notification']);

    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Return cleanup function
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('dve-updated', handleDVERentalUpdate);
      webSocketService.off('dve-expired', handleDVERentalExpired);
      webSocketService.off('system-notification', handleSystemNotification);
    };
  }, []);

  const disconnectWebSocket = useCallback(() => {
    // Individual hooks don't disconnect the shared service
    setIsConnected(false);
  }, []);

  // ⭐ NEW: Get full access information for a rental
  const getFullAccessInfo = useCallback(async (rentalId: string): Promise<DVEAccessInfo | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve/instances/${rentalId}/full-access-info`;
      const response: APIResponse<DVEAccessInfo> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch access info');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch access info:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // ⭐ NEW: Create SSH session for a rental
  const createSSHSession = useCallback(async (rentalId: string): Promise<SSHSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve/instances/${rentalId}/ssh-session`;
      const response: APIResponse<SSHSession> = await apiRequest(url, { method: 'POST' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create SSH session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create SSH session:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // ⭐ NEW: Create validation session for a rental
  const createValidationSession = useCallback(async (rentalId: string): Promise<ValidationSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve/instances/${rentalId}/validation-session`;
      const response: APIResponse<ValidationSession> = await apiRequest(url, { method: 'POST' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create validation session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create validation session:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // ⭐ NEW: Create error resolution session for a rental
  const createErrorResolutionSession = useCallback(async (rentalId: string): Promise<ErrorResolutionSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve/instances/${rentalId}/error-resolution-session`;
      const response: APIResponse<ErrorResolutionSession> = await apiRequest(url, { method: 'POST' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create error resolution session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create error resolution session:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Convenience methods
  const getActiveRentals = useCallback(() =>
    rentals.filter(rental => rental.status === 'active'), [rentals]);

  const getTotalCost = useCallback(() =>
    rentals.reduce((total, rental) => total + rental.total_cost, 0), [rentals]);

  // Initial fetch on mount
  useEffect(() => {
    fetchPlans();
    fetchStats();
    return connectWebSocket();
  }, [fetchPlans, fetchStats, connectWebSocket]);

  return {
    plans,
    rentals,
    stats,
    isLoading,
    error,
    isConnected,
    fetchPlans,
    fetchRentals,
    fetchStats,
    createRental,
    extendRental,
    cancelRental,
    getActiveRentals,
    getTotalCost,
    connectWebSocket,
    disconnectWebSocket,
    // ⭐ NEW access and session methods
    getFullAccessInfo,
    createSSHSession,
    createValidationSession,
    createErrorResolutionSession,
  };
};

export default useDVERental;
