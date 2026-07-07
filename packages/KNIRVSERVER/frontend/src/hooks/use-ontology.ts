'use client';

import { useQuery } from '@tanstack/react-query';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export interface OntologyEntity {
  id: string;
  type: string;
  label: string;
  properties: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface OntologyRelation {
  source_id: string;
  target_id: string;
  rel_type: string;
  properties: Record<string, unknown>;
  created_at: string;
}

export interface OntologyStats {
  entity_count: number;
  relation_count: number;
  entity_types: Record<string, number>;
}

export interface OntologyResponse {
  success: boolean;
  data?: unknown;
  message?: string;
  error?: string;
  timestamp: string;
}

const ONTOLOGY_QUERY_KEY = ['ontology'];

export const useOntology = () => {
  const stats = useQuery<OntologyStats>({
    queryKey: [...ONTOLOGY_QUERY_KEY, 'stats'],
    queryFn: async () => {
      const response = await apiRequest<OntologyStats>(`${API_BASE_URL}/api/ontology/stats`);
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch ontology stats');
    },
    staleTime: 60000,
  });

  const entities = useQuery<OntologyEntity[]>({
    queryKey: [...ONTOLOGY_QUERY_KEY, 'entities'],
    queryFn: async () => {
      const response = await apiRequest<{ entities: OntologyEntity[] }>(`${API_BASE_URL}/api/ontology/entities`);
      if (response.success && response.data) return response.data.entities;
      throw new Error(response.error || 'Failed to fetch entities');
    },
    staleTime: 60000,
  });

  const relations = useQuery<OntologyRelation[]>({
    queryKey: [...ONTOLOGY_QUERY_KEY, 'relations'],
    queryFn: async () => {
      const response = await apiRequest<{ relations: OntologyRelation[] }>(`${API_BASE_URL}/api/ontology/relations`);
      if (response.success && response.data) return response.data.relations;
      throw new Error(response.error || 'Failed to fetch relations');
    },
    staleTime: 60000,
  });

  return {
    stats,
    entities,
    relations,
    refetch: async () => {
      await Promise.all([
        stats.refetch(),
        entities.refetch(),
        relations.refetch(),
      ]);
    },
  };
};

export const useOntologyEntity = (id: string) => {
  return useQuery<OntologyEntity>({
    queryKey: [...ONTOLOGY_QUERY_KEY, 'entity', id],
    queryFn: async () => {
      const response = await apiRequest<OntologyEntity>(`${API_BASE_URL}/api/ontology/entities/${id}`);
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch entity');
    },
    enabled: !!id,
    staleTime: 60000,
  });
};

export const useOntologySearch = (query: string, type?: string) => {
  const params = new URLSearchParams({ q: query });
  if (type) params.set('type', type);

  return useQuery<OntologyEntity[]>({
    queryKey: [...ONTOLOGY_QUERY_KEY, 'search', query, type],
    queryFn: async () => {
      const response = await apiRequest<{ results: OntologyEntity[] }>(
        `${API_BASE_URL}/api/ontology/search?${params.toString()}`
      );
      if (response.success && response.data) return response.data.results;
      throw new Error(response.error || 'Failed to search entities');
    },
    enabled: query.length > 0,
    staleTime: 30000,
  });
};

export default useOntology;