"use client";

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from '@/lib/api';

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
  const queryClient = useQueryClient();
  const queryKey = ['root-failover'];
  const query = useQuery({
    queryKey,
    queryFn: fetchFailover,
    refetchInterval: 10_000,
    staleTime: 5_000,
    retry: 1,
  });

  // Reclaim doubles as the liveness proof: reaching the backend's handler at
  // all requires this request to land on the original root's own node,
  // authenticated, with that node's own decrypted root.key already loaded —
  // there's no separate signature for the browser to produce or hold.
  const reclaimMutation = useMutation({
    mutationFn: async () => {
      const response = await apiRequest('/api/v1/monitor/root-failover/reclaim', { method: 'POST' });
      if (!response.success) throw new Error(response.error || 'Reclaim request failed');
      return response.data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey }),
  });

  return {
    state: query.data ?? null,
    isLoading: query.isLoading,
    error: query.error instanceof Error ? query.error.message : null,
    refetch: query.refetch,
    requestReclaim: reclaimMutation.mutateAsync,
    isReclaiming: reclaimMutation.isPending,
    reclaimError: reclaimMutation.error instanceof Error ? reclaimMutation.error.message : null,
  };
}
