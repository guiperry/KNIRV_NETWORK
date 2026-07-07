'use client';

import { useCallback, useEffect, useState } from 'react';
import { API_BASE_URL } from '@/lib/api';

export interface SystemTelemetry {
  timestamp: string;
  source: 'cognitive-telemetry';
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

interface TelemetryState {
  data: SystemTelemetry | null;
  isLoading: boolean;
  error: string | null;
}

const EMPTY_STATE: TelemetryState = {
  data: null,
  isLoading: true,
  error: null,
};

const toTelemetry = (snapshot: Record<string, number | string>): SystemTelemetry => {
  const netTx = Number(snapshot.net_tx_bytes ?? 0);
  const netRx = Number(snapshot.net_rx_bytes ?? 0);
  const memoryBytes = Number(snapshot.memory_bytes ?? 0);
  const heapAlloc = Number(snapshot.heap_alloc_bytes ?? 0);

  return {
    timestamp: String(snapshot.timestamp ?? new Date().toISOString()),
    source: 'cognitive-telemetry',
    cpu_time_ns: Number(snapshot.cpu_time_ns ?? 0),
    memory_bytes: memoryBytes,
    net_tx_bytes: netTx,
    net_rx_bytes: netRx,
    network_throughput: (netTx + netRx) / 1024,
    active_connections: 0,
    context_switches: Number(snapshot.context_switches ?? 0),
    page_faults: Number(snapshot.page_faults ?? 0),
    goroutines: Number(snapshot.goroutines ?? 0),
    heap_alloc_bytes: heapAlloc,
    gc_count: Number(snapshot.gc_count ?? 0),
    cpu_pressure: Number(snapshot.cpu_pressure ?? 0),
    memory_pressure: Number(snapshot.memory_pressure ?? 0),
    cpu_usage: Number(snapshot.cpu_pressure ?? 0),
    memory_usage: memoryBytes > 0 ? (heapAlloc / memoryBytes) * 100 : Number(snapshot.memory_pressure ?? 0),
  };
};

export const useTelemetry = () => {
  const [telemetry, setTelemetry] = useState<TelemetryState>(EMPTY_STATE);
  const [refreshNonce, setRefreshNonce] = useState(0);

  const refetch = useCallback(() => {
    setRefreshNonce((value) => value + 1);
  }, []);

  useEffect(() => {
    let eventSource: EventSource | null = null;

    setTelemetry((prev) => ({ ...prev, isLoading: true, error: null }));

    try {
      eventSource = new EventSource(`${API_BASE_URL}/api/cognitive/telemetry?stream=1`);

      eventSource.addEventListener('telemetry', (event) => {
        const payload = JSON.parse((event as MessageEvent).data) as Record<string, number | string>;
        setTelemetry({
          data: toTelemetry(payload),
          isLoading: false,
          error: null,
        });
      });

      eventSource.onerror = () => {
        setTelemetry((prev) => ({
          data: prev.data,
          isLoading: false,
          error: prev.data ? null : 'Failed to connect to telemetry stream',
        }));
      };
    } catch (error) {
      setTelemetry({
        data: null,
        isLoading: false,
        error: error instanceof Error ? error.message : 'Failed to initialize telemetry stream',
      });
    }

    return () => {
      eventSource?.close();
    };
  }, [refreshNonce]);

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
    refetch,
  };
};

export default useTelemetry;
