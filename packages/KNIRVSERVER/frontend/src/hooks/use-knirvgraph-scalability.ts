"use client";

import { useQuery } from '@tanstack/react-query';

export interface KnirvgraphMetric {
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

export function useKnirvgraphScalability() {
  return useQuery<{ metrics: KnirvgraphMetric[] }>({
    queryKey: ['knirvgraph', 'scalability'],
    queryFn: async () => unwrap<{ metrics: KnirvgraphMetric[] }>(await fetch('/api/v1/knirvgraph/scalability')),
    refetchInterval: 15000,
    staleTime: 10000,
    retry: 1,
  });
}

export function useKnirvgraphEmbeddings() {
  return useQuery<{ metrics: KnirvgraphMetric[] }>({
    queryKey: ['knirvgraph', 'embeddings'],
    queryFn: async () => unwrap<{ metrics: KnirvgraphMetric[] }>(await fetch('/api/v1/knirvgraph/embeddings')),
    refetchInterval: 15000,
    staleTime: 10000,
    retry: 1,
  });
}

export default useKnirvgraphScalability;
