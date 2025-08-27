"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  TEESecurityMetrics,
  TEEPerformanceMetrics,
  ThreatAlert,
  SecurityAudit,
  APIResponse,
  TEESecurityUpdate
} from '@/types/api';
import { apiRequest, API_BASE_URL, StandardWebSocket } from '@/lib/api';

export interface TEESecurityStatus {
  attestation_status: "verified" | "pending" | "failed";
  enclave_count: number;
  security_score: number;
  last_audit: string;
  threats_detected: number;
  active_threats: ThreatAlert[];
  audit_history: SecurityAudit[];
  performance_metrics: TEEPerformanceMetrics;
  tee_type: string;
  last_attestation: string;
  monitoring_enabled: boolean;
}

export interface TEESecurityAction {
  action: "run_security_scan" | "perform_attestation" | "update_attestation" | "resolve_threat";
  parameters?: Record<string, any>;
}

export const useTEESecurity = () => {
  const [securityStatus, setSecurityStatus] = useState<TEESecurityStatus | null>(null);
  const [metrics, setMetrics] = useState<TEESecurityMetrics | null>(null);
  const [threats, setThreats] = useState<ThreatAlert[]>([]);
  const [auditHistory, setAuditHistory] = useState<SecurityAudit[]>([]);
  const [performanceMetrics, setPerformanceMetrics] = useState<TEEPerformanceMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [socket, setSocket] = useState<StandardWebSocket | null>(null);

  // Fetch TEE security status
  const fetchSecurityStatus = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/tee-security`;
      const response: APIResponse<TEESecurityStatus> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setSecurityStatus(response.data);
        setThreats(response.data.active_threats || []);
        setAuditHistory(response.data.audit_history || []);
        setPerformanceMetrics(response.data.performance_metrics);
      } else {
        throw new Error(response.error || 'Failed to fetch TEE security status');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch TEE security status:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch TEE security metrics
  const fetchMetrics = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/tee-security/metrics`;
      const response: APIResponse<TEESecurityMetrics> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setMetrics(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch TEE security metrics');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch TEE security metrics:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch threats
  const fetchThreats = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/tee-security/threats`;
      const response: APIResponse<ThreatAlert[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setThreats(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch threats');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch threats:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch audit history
  const fetchAuditHistory = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/tee-security/audit-history`;
      const response: APIResponse<SecurityAudit[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setAuditHistory(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch audit history');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch audit history:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch performance metrics
  const fetchPerformanceMetrics = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/tee-security/performance`;
      const response: APIResponse<TEEPerformanceMetrics> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setPerformanceMetrics(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch performance metrics');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch performance metrics:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Execute TEE security action
  const executeAction = useCallback(async (action: TEESecurityAction): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/tee-security/actions`;
      const response: APIResponse<TEESecurityStatus> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      
      if (response.success) {
        // Update security status with the response
        if (response.data) {
          setSecurityStatus(response.data);
          setThreats(response.data.active_threats || []);
          setAuditHistory(response.data.audit_history || []);
          setPerformanceMetrics(response.data.performance_metrics);
        }
        return true;
      } else {
        throw new Error(response.error || 'Failed to execute action');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to execute TEE security action:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Resolve a specific threat
  const resolveThreat = useCallback(async (threatId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/tee-security/threats/${threatId}/resolve`;
      const response: APIResponse = await apiRequest(url, { method: 'POST' });
      
      if (response.success) {
        // Remove the threat from the active threats list
        setThreats(prevThreats => prevThreats.filter(threat => threat.id !== threatId));
        return true;
      } else {
        throw new Error(response.error || 'Failed to resolve threat');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to resolve threat:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Convenience methods for common actions
  const runSecurityScan = useCallback(() => executeAction({ action: 'run_security_scan' }), [executeAction]);
  const performAttestation = useCallback(() => executeAction({ action: 'perform_attestation' }), [executeAction]);
  const updateAttestationStatus = useCallback((status: string) => 
    executeAction({ action: 'update_attestation', parameters: { status } }), [executeAction]);

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    if (socket?.isConnected()) return;

    const ws = new StandardWebSocket();
    
    ws.onOpen = () => {
      console.log('TEE Security WebSocket connected');
      setIsConnected(true);
      setError(null);
      
      // Subscribe to TEE security updates
      ws.subscribe(['tee-security-updated', 'system-notification']);
    };

    ws.onMessage = (message) => {
      if (message.event === 'tee-security-updated' && message.payload) {
        // Update security metrics
        setSecurityStatus(prevStatus => {
          if (!prevStatus) return prevStatus;
          return {
            ...prevStatus,
            attestation_status: message.payload.attestation_status || prevStatus.attestation_status,
            security_score: message.payload.security_score || prevStatus.security_score,
            threats_detected: message.payload.threats_detected || prevStatus.threats_detected,
            last_audit: message.payload.last_audit || prevStatus.last_audit,
          };
        });
      } else if (message.event === 'connected') {
        console.log('TEE Security WebSocket welcome:', message.payload);
      }
    };

    ws.onClose = () => {
      console.log('TEE Security WebSocket disconnected');
      setIsConnected(false);
    };

    ws.onError = (error) => {
      console.error('TEE Security WebSocket error:', error);
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

  // Refresh all data
  const refreshAll = useCallback(async () => {
    await Promise.all([
      fetchSecurityStatus(),
      fetchMetrics(),
      fetchThreats(),
      fetchAuditHistory(),
      fetchPerformanceMetrics(),
    ]);
  }, [fetchSecurityStatus, fetchMetrics, fetchThreats, fetchAuditHistory, fetchPerformanceMetrics]);

  // Initial fetch and WebSocket connection on mount
  useEffect(() => {
    fetchSecurityStatus();
    connectWebSocket();
  }, [fetchSecurityStatus, connectWebSocket]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnectWebSocket();
    };
  }, [disconnectWebSocket]);

  return {
    securityStatus,
    metrics,
    threats,
    auditHistory,
    performanceMetrics,
    isLoading,
    error,
    isConnected,
    fetchSecurityStatus,
    fetchMetrics,
    fetchThreats,
    fetchAuditHistory,
    fetchPerformanceMetrics,
    executeAction,
    resolveThreat,
    runSecurityScan,
    performAttestation,
    updateAttestationStatus,
    refreshAll,
    connectWebSocket,
    disconnectWebSocket,
  };
};

export default useTEESecurity;
