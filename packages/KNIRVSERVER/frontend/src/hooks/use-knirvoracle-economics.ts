"use client";

import { useQuery } from '@tanstack/react-query';
import { getAuthHeaders } from '@/lib/api';

export interface KNIRVOracleEconomicsData {
  total_supply: number;
  total_staked: number;
  total_burned: number;
  apy: number;
  fee_volume: number;
  rewards_issued: number;
}

async function unwrap<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }
  const payload = await response.json();
  return (payload.data ?? payload) as T;
}

export function useKNIRVOracleEconomics() {
  return useQuery<{ metrics: Record<string, unknown> }>({
    queryKey: ['knirvoracle', 'economics'],
    queryFn: async () => unwrap<{ metrics: Record<string, unknown> }>(await fetch('/api/v1/knirvoracle/economics', { headers: getAuthHeaders() })),
    refetchInterval: 30000,
    staleTime: 15000,
    retry: 1,
  });
}

export default useKNIRVOracleEconomics;
