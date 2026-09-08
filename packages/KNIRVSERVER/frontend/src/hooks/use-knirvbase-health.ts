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

type RawKnirvbaseHealthData = Omit<KnirvbaseHealthData, 'blocksCommitted' | 'errorRate' | 'cacheHitRatio' | 'activeConnections' | 'lastCheck'> & {
  blocks_committed?: number;
  error_rate?: number;
  cache_hit_ratio?: number;
  active_connections?: number;
  last_check?: string;
};

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
    queryFn: async () => {
      const payload = await unwrap<{ health: RawKnirvbaseHealthData }>(await fetch('/api/v1/knirvbase/health', { headers: getAuthHeaders() }));
      const health = payload.health;
      return {
        health: {
          ...health,
          blocksCommitted: health.blocks_committed ?? 0,
          errorRate: health.error_rate ?? 0,
          cacheHitRatio: health.cache_hit_ratio ?? 0,
          activeConnections: health.active_connections ?? 0,
          lastCheck: health.last_check ?? '',
        },
      };
    },
    refetchInterval: 15000,
    staleTime: 10000,
    retry: 1,
  });
}

export default useKnirvbaseHealth;
