"use client";

import { useQuery } from '@tanstack/react-query';

export interface KnirvchainHealthData {
  url: string;
  status: string;
  lastCheck: string;
}

async function unwrap<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }
  const payload = await response.json();
  return (payload.data ?? payload) as T;
}

export function useKnirvchainHealth() {
  return useQuery<{ health: KnirvchainHealthData }>({
    queryKey: ['knirvchain', 'health'],
    queryFn: async () => unwrap<{ health: KnirvchainHealthData }>(await fetch('/api/v1/knirvchain/health')),
    refetchInterval: 15000,
    staleTime: 10000,
    retry: 1,
  });
}

export default useKnirvchainHealth;
