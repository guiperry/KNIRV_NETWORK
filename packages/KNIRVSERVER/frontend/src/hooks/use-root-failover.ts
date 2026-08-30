"use client";

import { useQuery } from '@tanstack/react-query';

export type RootFailoverPhase = 'NORMAL' | 'VOTING' | 'ACTING_ROOT' | 'CONFIRMED' | 'RECLAIMED' | 'UNKNOWN';

export interface RootFailoverState {
  chainID: string;
  phase: RootFailoverPhase;
  currentRootID: string;
  previousRootID?: string;
  since: number;
  round: number;
  rootStatus: 0 | 1;
  reclaimDeadline?: number;
  consensus?: {
    activeValidators: string[];
    missing: string[];
    healthy: string[];
    unhealthy: string[];
    votes: number;
    winner: string | null;
    rootDown: boolean;
  };
}

async function fetchFailover(): Promise<RootFailoverState> {
  const response = await fetch('/api/v1/monitor/root-failover');
  if (!response.ok) throw new Error(`root failover request failed: ${response.status}`);
  const payload = await response.json();
  return (payload.data ?? payload) as RootFailoverState;
}

export function useRootFailover() {
  const query = useQuery({
    queryKey: ['root-failover'],
    queryFn: fetchFailover,
    refetchInterval: 10_000,
    staleTime: 5_000,
    retry: 1,
  });
  return { state: query.data ?? null, isLoading: query.isLoading, error: query.error instanceof Error ? query.error.message : null, refetch: query.refetch };
}
