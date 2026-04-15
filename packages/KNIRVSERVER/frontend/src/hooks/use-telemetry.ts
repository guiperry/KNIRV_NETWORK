'use client';

import { useQuery } from '@tanstack/react-query';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export interface SystemTelemetry {
  timestamp: string;
  source: 'system-health';
  cpu_time_ns: number;
  memory_bytes: number;
  net_tx_bytes: number;
  net_rx_bytes: number;
  network_throughput: number;
  active_connections: number;
  context_switches: number;
  page_faults: number;
  goroutines: number;
  heap_alloc_bytes: number;
  gc_count: number;
  cpu_pressure: number;
  memory_pressure: number;
  cpu_usage: number;
  memory_usage: number;
}

export interface TelemetryResponse {
  success: boolean;
  data?: unknown;
  message?: string;
  error?: string;
  timestamp: string;
}

export const useTelemetry = () => {
  const queryKey = ['telemetry'];

  const telemetry = useQuery<SystemTelemetry>({
    queryKey,
    queryFn: async () => {
      const response = await apiRequest<{
        system_load?: number;
        memory_usage?: number;
        disk_usage?: number;
        network_throughput?: number;
        active_connections?: number;
        goroutine_count?: number;
        cpu_usage?: number;
      }>(`${API_BASE_URL}/api/system-health/metrics`);

      if (!response.success || !response.data) {
        throw new Error(response.error || 'Failed to fetch telemetry');
      }

      const metrics = response.data;
      return {
        timestamp: response.timestamp,
        source: 'system-health',
        cpu_time_ns: 0,
        memory_bytes: 0,
        net_tx_bytes: 0,
        net_rx_bytes: 0,
        network_throughput: metrics.network_throughput ?? 0,
        active_connections: metrics.active_connections ?? 0,
        context_switches: 0,
        page_faults: 0,
        goroutines: metrics.goroutine_count ?? 0,
        heap_alloc_bytes: 0,
        gc_count: 0,
        cpu_pressure: metrics.cpu_usage ?? 0,
        memory_pressure: metrics.memory_usage ?? 0,
        cpu_usage: metrics.cpu_usage ?? 0,
        memory_usage: metrics.memory_usage ?? 0,
      };
    },
    staleTime: 10000,
  });

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
  };

  const formatCPU = (ns: number): string => {
    const seconds = ns / 1e9;
    if (seconds < 60) return `${seconds.toFixed(2)} s`;
    const minutes = seconds / 60;
    if (minutes < 60) return `${minutes.toFixed(2)} min`;
    const hours = minutes / 60;
    return `${hours.toFixed(2)} h`;
  };

  return {
    telemetry,
    formatBytes,
    formatCPU,
    refetch: () => telemetry.refetch(),
  };
};

export default useTelemetry;
