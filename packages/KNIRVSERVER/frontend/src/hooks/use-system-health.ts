"use client";

import { useEffect, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { APIResponse } from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export interface SystemHealth {
  overall_status: "healthy" | "degraded" | "critical";
  timestamp: string;
  uptime: number;
  components: Record<string, ComponentHealth>;
  component_summary?: ComponentSummary;
  active_alerts?: number;
  alerts: SystemAlert[];
  metrics: SystemMetrics;
}

export interface ComponentHealth {
  status: "healthy" | "warning" | "critical";
  message: string;
  metrics?: Record<string, any>;
}

export interface ComponentSummary {
  total_components: number;
  healthy_components: number;
  warning_components: number;
  critical_components: number;
}

export interface SystemAlert {
  id: string;
  severity: "low" | "medium" | "high" | "critical";
  component: string;
  message: string;
  timestamp: string;
  resolved: boolean;
  resolved_at?: string;
  metadata?: Record<string, any>;
}

export interface SystemMetrics {
  system_load: number;
  memory_usage: number;
  disk_usage: number;
  network_throughput: number;
  active_connections: number;
  goroutine_count: number;
  cpu_usage: number;
}

export interface SystemHealthAction {
  action: "run_diagnostics" | "add_alert" | "resolve_alert";
  parameters?: Record<string, any>;
}

export const useSystemHealth = (detailed: boolean = true) => {
  const queryClient = useQueryClient();
  const queryKey = ['system-health', detailed];

  const {
    data: systemHealth = null,
    isLoading,
    error: queryError,
    refetch: fetchSystemHealth
  } = useQuery<SystemHealth>({
    queryKey,
    queryFn: async () => {
      const url = `${API_BASE_URL}/api/system-health?detailed=${detailed}`;
      const response: APIResponse<SystemHealth> = await apiRequest(url, { method: 'GET' });
      if (response.success && response.data) {
        return response.data;
      }
      throw new Error(response.error || 'Failed to fetch system health');
    },
    staleTime: 10000, // 10 seconds for health data
  });

  const error = queryError instanceof Error ? queryError.message : null;

  useEffect(() => {
    const handleSystemHealthUpdate = (payload: any) => {
      queryClient.setQueryData<SystemHealth>(queryKey, (prevHealth) => {
        if (!prevHealth) return prevHealth;
        return {
          ...prevHealth,
          overall_status: payload.overall_status ?? prevHealth.overall_status,
          metrics: payload.metrics ?? prevHealth.metrics,
          components: payload.components ?? prevHealth.components,
        };
      });
    };

    const handleSystemAlertAdded = (payload: any) => {
      queryClient.setQueryData<SystemHealth>(queryKey, (prevHealth) => {
        if (!prevHealth) return prevHealth;
        return {
          ...prevHealth,
          alerts: [payload, ...(prevHealth.alerts || [])],
        };
      });
    };

    webSocketService.on('system-health-updated', handleSystemHealthUpdate);
    webSocketService.on('system-alert-added', handleSystemAlertAdded);
    webSocketService.subscribe(['system-health-updated', 'system-alert-added']);

    return () => {
      webSocketService.off('system-health-updated', handleSystemHealthUpdate);
      webSocketService.off('system-alert-added', handleSystemAlertAdded);
    };
  }, [queryClient, queryKey]);

  const executeActionMutation = useMutation({
    mutationFn: async (action: SystemHealthAction) => {
      const url = `${API_BASE_URL}/api/system-health/actions`;
      const response: APIResponse<any> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      if (response.success) return response.data;
      throw new Error(response.error || 'Failed to execute system health action');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
    }
  });

  return {
    systemHealth,
    alerts: systemHealth?.alerts || [],
    metrics: systemHealth?.metrics || null,
    components: systemHealth?.components || {},
    isLoading,
    error,
    isConnected: webSocketService.getConnectionStatus(),
    fetchSystemHealth,
    executeAction: executeActionMutation.mutateAsync,
    refreshAll: fetchSystemHealth,
  };
};

export default useSystemHealth;
