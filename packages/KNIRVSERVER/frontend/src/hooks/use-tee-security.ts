"use client";

import { useEffect, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  TEESecurityMetrics,
  TEEPerformanceMetrics,
  ThreatAlert,
  SecurityAudit,
  APIResponse
} from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

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
  const queryClient = useQueryClient();
  const queryKey = ['tee-security'];

  const {
    data: securityStatus = null,
    isLoading,
    error: queryError,
    refetch: fetchSecurityStatus
  } = useQuery<TEESecurityStatus>({
    queryKey,
    queryFn: async () => {
      const url = `${API_BASE_URL}/api/tee-security`;
      const response: APIResponse<TEESecurityStatus> = await apiRequest(url, { method: 'GET' });
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch TEE security status');
    },
    staleTime: 30000,
  });

  const error = queryError instanceof Error ? queryError.message : null;

  useEffect(() => {
    const handleTEESecurityUpdate = (payload: any) => {
      queryClient.setQueryData<TEESecurityStatus>(queryKey, (prevStatus) => {
        if (!prevStatus) return prevStatus;
        return {
          ...prevStatus,
          ...payload,
          attestation_status: payload.attestation_status ?? prevStatus.attestation_status,
          security_score: payload.security_score ?? prevStatus.security_score,
          threats_detected: payload.threats_detected ?? prevStatus.threats_detected,
          last_audit: payload.last_audit ?? prevStatus.last_audit,
        };
      });
    };

    webSocketService.on('tee-security-updated', handleTEESecurityUpdate);
    webSocketService.subscribe(['tee-security-updated']);

    return () => {
      webSocketService.off('tee-security-updated', handleTEESecurityUpdate);
    };
  }, [queryClient, queryKey]);

  const executeActionMutation = useMutation({
    mutationFn: async (action: TEESecurityAction) => {
      const url = `${API_BASE_URL}/api/tee-security/actions`;
      const response: APIResponse<TEESecurityStatus> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to execute action');
    },
    onSuccess: (data) => {
      queryClient.setQueryData(queryKey, data);
    }
  });

  const resolveThreatMutation = useMutation({
    mutationFn: async (threatId: string) => {
      const url = `${API_BASE_URL}/api/tee-security/threats/${threatId}/resolve`;
      const response: APIResponse = await apiRequest(url, { method: 'POST' });
      if (response.success) return threatId;
      throw new Error(response.error || 'Failed to resolve threat');
    },
    onSuccess: (threatId) => {
      queryClient.setQueryData<TEESecurityStatus>(queryKey, (prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          active_threats: prev.active_threats?.filter(t => t.id !== threatId) || []
        };
      });
    }
  });

  return {
    securityStatus,
    threats: securityStatus?.active_threats || [],
    auditHistory: securityStatus?.audit_history || [],
    performanceMetrics: securityStatus?.performance_metrics || null,
    isLoading,
    error,
    isConnected: webSocketService.getConnectionStatus(),
    executeAction: executeActionMutation.mutateAsync,
    resolveThreat: resolveThreatMutation.mutateAsync,
    refreshAll: fetchSecurityStatus,
  };
};

export default useTEESecurity;
