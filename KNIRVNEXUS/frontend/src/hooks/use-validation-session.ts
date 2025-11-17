"use client";

import { useState, useCallback } from 'react';
import type { ValidationSession, APIResponse } from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export const useValidationSession = () => {
  const [currentSession, setCurrentSession] = useState<ValidationSession | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Create validation session for a rental
  const createSession = useCallback(async (rentalId: string): Promise<ValidationSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-rental/rentals/${rentalId}/validation-session`;
      const response: APIResponse<ValidationSession> = await apiRequest(url, { method: 'POST' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        setCurrentSession(response.data);
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

  // Get existing validation session
  const getSession = useCallback(async (rentalId: string): Promise<ValidationSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-rental/rentals/${rentalId}/validation-session`;
      const response: APIResponse<ValidationSession> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        setCurrentSession(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to get validation session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to get validation session:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Terminate validation session
  const terminateSession = useCallback(async (rentalId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-rental/rentals/${rentalId}/validation-session`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });

      if (response.success) {
        setCurrentSession(null);
        return true;
      } else {
        throw new Error(response.error || 'Failed to terminate validation session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to terminate validation session:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Open validation interface in new tab
  const openValidationInterface = useCallback((session: ValidationSession) => {
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
      console.error('Failed to open validation interface:', err);
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
    openValidationInterface,
    clearSession,
  };
};

export default useValidationSession;
