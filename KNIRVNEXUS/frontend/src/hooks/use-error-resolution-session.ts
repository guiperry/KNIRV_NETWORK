"use client";

import { useState, useCallback } from 'react';
import type { ErrorResolutionSession, APIResponse } from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export const useErrorResolutionSession = () => {
  const [currentSession, setCurrentSession] = useState<ErrorResolutionSession | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Create error resolution session for a rental
  const createSession = useCallback(async (rentalId: string): Promise<ErrorResolutionSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-rental/rentals/${rentalId}/error-resolution-session`;
      const response: APIResponse<ErrorResolutionSession> = await apiRequest(url, { method: 'POST' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        setCurrentSession(response.data);
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

  // Get existing error resolution session
  const getSession = useCallback(async (rentalId: string): Promise<ErrorResolutionSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-rental/rentals/${rentalId}/error-resolution-session`;
      const response: APIResponse<ErrorResolutionSession> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        setCurrentSession(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to get error resolution session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to get error resolution session:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Terminate error resolution session
  const terminateSession = useCallback(async (rentalId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-rental/rentals/${rentalId}/error-resolution-session`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });

      if (response.success) {
        setCurrentSession(null);
        return true;
      } else {
        throw new Error(response.error || 'Failed to terminate error resolution session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to terminate error resolution session:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Open error resolution interface in new tab
  const openErrorResolutionInterface = useCallback((session: ErrorResolutionSession) => {
    if (!session.endpoint_url || !session.session_token) {
      setError('Invalid session data');
      return false;
    }

    try {
      const url = `${session.endpoint_url}?token=${encodeURIComponent(session.session_token)}&session_id=${encodeURIComponent(session.id)}`;
      window.open(url, '_blank', 'noopener,noreferrer');
      return true;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to open error resolution interface:', err);
      return false;
    }
  }, []);

  // Clear current session
  const clearSession = useCallback(() => {
    setCurrentSession(null);
    setError(null);
  }, []);

  return {
    currentSession,
    isLoading,
    error,
    createSession,
    getSession,
    terminateSession,
    openErrorResolutionInterface,
    clearSession,
  };
};

export default useErrorResolutionSession;
