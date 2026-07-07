"use client";

import { useQuery } from '@tanstack/react-query';

export interface NetworkMonitorStatus {
  name: string;
  overall_status: 'healthy' | 'degraded' | 'critical' | string;
  services_up: number;
  services_down: number;
  services_total: number;
  last_update: string;
}

export interface NetworkMonitorMetrics {
  cpu?: {
    usage_percent: number;
    load_average?: number[];
  };
  memory?: {
    total_bytes: number;
    used_bytes: number;
    available_bytes: number;
    usage_percent: number;
  };
  disk?: {
    total_bytes: number;
    used_bytes: number;
    available_bytes: number;
    usage_percent: number;
  };
  timestamp?: string;
}

const unwrap = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    throw new Error(`network monitor request failed: ${response.status}`);
  }
  const payload = await response.json();
  return (payload.data ?? payload) as T;
};

export const useNetworkMonitor = () => {
  const status = useQuery<NetworkMonitorStatus>({
    queryKey: ['network-monitor', 'status'],
    queryFn: async () => unwrap<NetworkMonitorStatus>(await fetch('/api/network-monitor/status')),
    refetchInterval: 10000,
    staleTime: 5000,
    retry: 1,
  });

  const metrics = useQuery<NetworkMonitorMetrics>({
    queryKey: ['network-monitor', 'metrics'],
    queryFn: async () => unwrap<NetworkMonitorMetrics>(await fetch('/api/network-monitor/metrics')),
    refetchInterval: 10000,
    staleTime: 5000,
    retry: 1,
  });

  return {
    status: status.data ?? null,
    metrics: metrics.data ?? null,
    isLoading: status.isLoading || metrics.isLoading,
    error: status.error instanceof Error ? status.error.message : metrics.error instanceof Error ? metrics.error.message : null,
    refetch: () => {
      status.refetch();
      metrics.refetch();
    },
  };
};

export default useNetworkMonitor;
