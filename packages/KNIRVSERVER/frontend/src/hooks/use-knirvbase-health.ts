"use client";

import { useQuery } from '@tanstack/react-query';
import { getAuthHeaders } from '@/lib/api';

export interface KnirvbaseHealthData {
  url: string;
  status: string;
  blocksCommitted: number;
  errorRate: number;
  cacheHitRatio: number;
  activeConnections: number;
  lastCheck: string;
}

async function unwrap<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }
  const payload = await response.json();
  return (payload.data ?? payload) as T;
}

export function useKnirvbaseHealth() {
  return useQuery<{ health: KnirvbaseHealthData }>({
    queryKey: ['knirvbase', 'health'],
    queryFn: async () => unwrap<{ health: KnirvbaseHealthData }>(await fetch('/api/v1/knirvbase/health', { headers: getAuthHeaders() })),
    refetchInterval: 15000,
    staleTime: 10000,
    retry: 1,
  });
}

export default useKnirvbaseHealth;
