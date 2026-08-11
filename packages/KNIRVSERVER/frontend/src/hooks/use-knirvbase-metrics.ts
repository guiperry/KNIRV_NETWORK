"use client";

import { useQuery } from '@tanstack/react-query';
import { getAuthHeaders } from '@/lib/api';

export interface KnirvbaseMetric {
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

export function useKnirvbaseMetrics() {
  return useQuery<{ metrics: KnirvbaseMetric[] }>({
    queryKey: ['knirvbase', 'metrics'],
    queryFn: async () => unwrap<{ metrics: KnirvbaseMetric[] }>(await fetch('/api/v1/knirvbase/metrics', { headers: getAuthHeaders() })),
    refetchInterval: 15000,
    staleTime: 10000,
    retry: 1,
  });
}

export default useKnirvbaseMetrics;
