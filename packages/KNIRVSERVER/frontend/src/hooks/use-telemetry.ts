'use client';

import { useQuery } from '@tanstack/react-query';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export interface SystemTelemetry {
  timestamp: string;
  cpu_time_ns: number;
  memory_bytes: number;
  net_tx_bytes: number;
  net_rx_bytes: number;
  context_switches: number;
  page_faults: number;
  goroutines: number;
  heap_alloc_bytes: number;
  gc_count: number;
  cpu_pressure: number;
  memory_pressure: number;
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
      const response = await apiRequest<SystemTelemetry>(`${API_BASE_URL}/api/cognitive/telemetry`);
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch telemetry');
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