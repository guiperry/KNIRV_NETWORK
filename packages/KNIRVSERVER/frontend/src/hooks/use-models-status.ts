"use client";

import { useQuery } from '@tanstack/react-query';
import { getAuthHeaders } from '@/lib/api';
import { STATUS_PATH, type PlatformStatus } from '@/lib/models-status';

export async function fetchModelsStatus(limit = 8): Promise<PlatformStatus> {
  const response = await fetch(`${STATUS_PATH}?limit=${limit}`, { headers: getAuthHeaders() });
  if (response.status === 404) throw new Error('This server does not report model status yet');
  if (!response.ok) throw new Error(`model status request failed: ${response.status}`);
  return (await response.json()) as PlatformStatus;
}

/** Polls the platform's model status (which model does what, recent routing). */
export function useModelsStatus(limit = 8) {
  return useQuery({
    queryKey: ['models-status', limit],
    queryFn: () => fetchModelsStatus(limit),
    refetchInterval: 10_000,
    staleTime: 5_000,
    retry: 1,
  });
}
