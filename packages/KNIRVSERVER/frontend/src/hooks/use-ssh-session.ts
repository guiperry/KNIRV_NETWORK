"use client";

import { useState, useCallback } from 'react';
import type { SSHSession, APIResponse } from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export const useSSHSession = () => {
  const [currentSession, setCurrentSession] = useState<SSHSession | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Create SSH session for a creation
  const createSession = useCallback(async (creationId: string): Promise<SSHSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-creation/nodes/${creationId}/ssh-session`;
      const response: APIResponse<SSHSession> = await apiRequest(url, { method: 'POST' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        setCurrentSession(response.data);
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

  // Get existing SSH session
  const getSession = useCallback(async (creationId: string): Promise<SSHSession | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-creation/nodes/${creationId}/ssh-session`;
      const response: APIResponse<SSHSession> = await apiRequest(url, { method: 'GET' });

      if (response.success && response.data && !Array.isArray(response.data)) {
        setCurrentSession(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to get SSH session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to get SSH session:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Terminate SSH session
  const terminateSession = useCallback(async (creationId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);

    try {
      const url = `${API_BASE_URL}/api/dve-creation/nodes/${creationId}/ssh-session`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });

      if (response.success) {
        setCurrentSession(null);
        return true;
      } else {
        throw new Error(response.error || 'Failed to terminate SSH session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to terminate SSH session:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Download private key (one-time use)
  const downloadPrivateKey = useCallback(async (sessionId: string): Promise<boolean> => {
    try {
      const url = `${API_BASE_URL}/api/sessions/ssh/${sessionId}/private-key`;
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`,
        },
      });

      if (response.ok) {
        const blob = await response.blob();
        const downloadUrl = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = downloadUrl;
        a.download = 'dve-ssh-key.pem';
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(downloadUrl);
        document.body.removeChild(a);
        return true;
      } else {
        throw new Error('Failed to download private key');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to download private key:', err);
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
    downloadPrivateKey,
    clearSession,
  };
};

export default useSSHSession;
