"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  APIResponse,
  DVERentalPlan,
  DVERental,
  DVERentalStats,
  RentalRequest,
  ExtendRentalRequest
} from '@/types/api';
import { apiRequest, API_BASE_URL, StandardWebSocket } from '@/lib/api';

export const useDVERental = () => {
  const [plans, setPlans] = useState<DVERentalPlan[]>([]);
  const [rentals, setRentals] = useState<DVERental[]>([]);
  const [stats, setStats] = useState<DVERentalStats | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [socket, setSocket] = useState<StandardWebSocket | null>(null);

  // Fetch available rental plans
  const fetchPlans = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-rental/plans`;
      const response: APIResponse<DVERentalPlan[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && Array.isArray(response.data)) {
        setPlans(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch rental plans');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch rental plans:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch user's active rentals
  const fetchRentals = useCallback(async (userId?: string) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const queryParams = userId ? `?user_id=${userId}` : '';
      const url = `${API_BASE_URL}/api/dve-rental/rentals${queryParams}`;
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

  // Fetch rental statistics
  const fetchStats = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-rental/stats`;
      const response: APIResponse<DVERentalStats> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data && !Array.isArray(response.data)) {
        setStats(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch rental stats');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch rental stats:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Create a new rental
  const createRental = useCallback(async (rentalRequest: RentalRequest): Promise<DVERental | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/dve-rental/rentals`;
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
      const url = `${API_BASE_URL}/api/dve-rental/rentals/${rentalId}/extend`;
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
      const url = `${API_BASE_URL}/api/dve-rental/rentals/${rentalId}`;
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
    if (socket?.isConnected()) return;

    const ws = new StandardWebSocket();

    ws.onOpen = () => {
      console.log('DVE Rental WebSocket connected');
      setIsConnected(true);
      setError(null);

      // Subscribe to rental updates
      ws.subscribe(['dve-rental-updated', 'dve-rental-expired', 'system-notification']);
    };

    ws.onMessage = (message) => {
      if (message.event === 'dve-rental-updated' && message.payload) {
        // Update specific rental in the list
        setRentals(prevRentals =>
          prevRentals.map(rental =>
            rental.id === message.payload.id
              ? { ...rental, ...message.payload }
              : rental
          )
        );
      } else if (message.event === 'dve-rental-expired' && message.payload) {
        // Mark rental as expired
        setRentals(prevRentals =>
          prevRentals.map(rental =>
            rental.id === message.payload.id
              ? { ...rental, status: 'expired' }
              : rental
          )
        );
      }
    };

    ws.onClose = () => {
      console.log('DVE Rental WebSocket disconnected');
      setIsConnected(false);
    };

    ws.onError = (error) => {
      console.error('DVE Rental WebSocket error:', error);
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

  // Convenience methods
  const getActiveRentals = useCallback(() => 
    rentals.filter(rental => rental.status === 'active'), [rentals]);
  
  const getTotalCost = useCallback(() => 
    rentals.reduce((total, rental) => total + rental.total_cost, 0), [rentals]);

  // Initial fetch on mount
  useEffect(() => {
    fetchPlans();
    fetchStats();
    connectWebSocket();
  }, [fetchPlans, fetchStats, connectWebSocket]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnectWebSocket();
    };
  }, [disconnectWebSocket]);

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
  };
};

export default useDVERental;
