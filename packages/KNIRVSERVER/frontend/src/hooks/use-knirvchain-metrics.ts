"use client";

import { useQuery } from '@tanstack/react-query';
import { getAuthHeaders } from '@/lib/api';

export interface KnirvchainMetric {
  name: string;
  value: number;
  help: string;
  type: string;
}

async function unwrap<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }
  const payload = await response.json();
  return (payload.data ?? payload) as T;
}

export function useKnirvchainMetrics() {
  return useQuery<{ metrics: KnirvchainMetric[] }>({
    queryKey: ['knirvchain', 'metrics'],
    queryFn: async () => unwrap<{ metrics: KnirvchainMetric[] }>(await fetch('/api/v1/knirvchain/metrics', { headers: getAuthHeaders() })),
    refetchInterval: 15000,
    staleTime: 10000,
    retry: 1,
  });
}

export default useKnirvchainMetrics;
