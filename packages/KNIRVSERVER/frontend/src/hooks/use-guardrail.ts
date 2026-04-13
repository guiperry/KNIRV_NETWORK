'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiRequest, API_BASE_URL } from '@/lib/api';

export interface GuardrailViolation {
  id: string;
  node_id: string;
  policy_id: string;
  violation_type: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  details: string;
  timestamp: string;
  resolved: boolean;
  resolved_at?: string;
}

export interface GuardrailStatistics {
  total_violations: number;
  active_violations: number;
  resolved_violations: number;
  policies: {
    id: string;
    name: string;
    violations_count: number;
    enabled: boolean;
  }[];
}

export interface GuardrailPolicy {
  id: string;
  name: string;
  description: string;
  rules: Record<string, unknown>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface GuardrailResponse {
  success: boolean;
  data?: unknown;
  message?: string;
  error?: string;
  timestamp: string;
}

const GUARDRAIL_QUERY_KEY = ['guardrail'];

export const useGuardrail = () => {
  const queryClient = useQueryClient();

  const violations = useQuery<GuardrailViolation[]>({
    queryKey: [...GUARDRAIL_QUERY_KEY, 'violations'],
    queryFn: async () => {
      const response = await apiRequest<{ violations: GuardrailViolation[] }>(
        `${API_BASE_URL}/api/guardrail/violations`
      );
      if (response.success && response.data) return response.data.violations;
      throw new Error(response.error || 'Failed to fetch violations');
    },
    staleTime: 15000,
  });

  const statistics = useQuery<GuardrailStatistics>({
    queryKey: [...GUARDRAIL_QUERY_KEY, 'statistics'],
    queryFn: async () => {
      const response = await apiRequest<GuardrailStatistics>(
        `${API_BASE_URL}/api/guardrail/statistics`
      );
      if (response.success && response.data) return response.data;
      throw new Error(response.error || 'Failed to fetch statistics');
    },
    staleTime: 30000,
  });

  const policies = useQuery<GuardrailPolicy[]>({
    queryKey: [...GUARDRAIL_QUERY_KEY, 'policies'],
    queryFn: async () => {
      const response = await apiRequest<{ policies: GuardrailPolicy[] }>(
        `${API_BASE_URL}/api/guardrail/policies`
      );
      if (response.success && response.data) return response.data.policies;
      throw new Error(response.error || 'Failed to fetch policies');
    },
    staleTime: 60000,
  });

  const resolveViolation = useMutation({
    mutationFn: async (violationId: string) => {
      const response = await apiRequest<{ status: string }>(
        `${API_BASE_URL}/api/guardrail/violations/${violationId}/resolve`,
        { method: 'POST' }
      );
      if (!response.success) {
        throw new Error(response.error || 'Failed to resolve violation');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...GUARDRAIL_QUERY_KEY, 'violations'] });
      queryClient.invalidateQueries({ queryKey: [...GUARDRAIL_QUERY_KEY, 'statistics'] });
    },
  });

  const createPolicy = useMutation({
    mutationFn: async (policy: Omit<GuardrailPolicy, 'id' | 'created_at' | 'updated_at'>) => {
      const response = await apiRequest<{ status: string; policy: GuardrailPolicy }>(
        `${API_BASE_URL}/api/guardrail/policies`,
        {
          method: 'POST',
          body: JSON.stringify(policy),
        }
      );
      if (!response.success) {
        throw new Error(response.error || 'Failed to create policy');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...GUARDRAIL_QUERY_KEY, 'policies'] });
    },
  });

  const updatePolicy = useMutation({
    mutationFn: async (policy: GuardrailPolicy) => {
      const response = await apiRequest<{ status: string }>(
        `${API_BASE_URL}/api/guardrail/policies/${policy.id}`,
        {
          method: 'PUT',
          body: JSON.stringify(policy),
        }
      );
      if (!response.success) {
        throw new Error(response.error || 'Failed to update policy');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...GUARDRAIL_QUERY_KEY, 'policies'] });
    },
  });

  const deletePolicy = useMutation({
    mutationFn: async (policyId: string) => {
      const response = await apiRequest<{ status: string }>(
        `${API_BASE_URL}/api/guardrail/policies/${policyId}`,
        { method: 'DELETE' }
      );
      if (!response.success) {
        throw new Error(response.error || 'Failed to delete policy');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...GUARDRAIL_QUERY_KEY, 'policies'] });
    },
  });

  const commitPolicy = useMutation({
    mutationFn: async (policyId: string) => {
      const response = await apiRequest<{ status: string; tx_hash: string }>(
        `${API_BASE_URL}/api/guardrail/policies/${policyId}/commit`,
        { method: 'POST' }
      );
      if (!response.success) {
        throw new Error(response.error || 'Failed to commit policy');
      }
      return response.data;
    },
  });

  return {
    violations,
    statistics,
    policies,
    resolveViolation: resolveViolation.mutateAsync,
    createPolicy: createPolicy.mutateAsync,
    updatePolicy: updatePolicy.mutateAsync,
    deletePolicy: deletePolicy.mutateAsync,
    commitPolicy: commitPolicy.mutateAsync,
    refetchViolations: () => violations.refetch(),
    refetchStatistics: () => statistics.refetch(),
    refetchPolicies: () => policies.refetch(),
  };
};

export const useGuardrailViolationsForNode = (nodeId: string) => {
  return useQuery<GuardrailViolation[]>({
    queryKey: [...GUARDRAIL_QUERY_KEY, 'violations', nodeId],
    queryFn: async () => {
      const response = await apiRequest<{ violations: GuardrailViolation[] }>(
        `${API_BASE_URL}/api/guardrail/violations?node_id=${nodeId}`
      );
      if (response.success && response.data) return response.data.violations;
      throw new Error(response.error || 'Failed to fetch violations for node');
    },
    enabled: !!nodeId,
    staleTime: 15000,
  });
};

export default useGuardrail;