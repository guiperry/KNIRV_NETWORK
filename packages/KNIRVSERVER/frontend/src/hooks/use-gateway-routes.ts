"use client";

import { useQuery } from '@tanstack/react-query';
import { getAuthHeaders } from '@/lib/api';

export interface GatewayRoute {
  name: string;
  pathPrefix: string;
  target: string;
  protocol: string;
  status: string;
  latencyMs: number;
}

async function unwrap<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }
  const payload = await response.json();
  return (payload.data ?? payload) as T;
}

export function useGatewayRoutes() {
  return useQuery<{ routes: GatewayRoute[] }>({
    queryKey: ['gateway', 'routes'],
    queryFn: async () => unwrap<{ routes: GatewayRoute[] }>(await fetch('/api/v1/gateway/routes', { headers: getAuthHeaders() })),
    refetchInterval: 15000,
    staleTime: 10000,
    retry: 1,
  });
}

export default useGatewayRoutes;
