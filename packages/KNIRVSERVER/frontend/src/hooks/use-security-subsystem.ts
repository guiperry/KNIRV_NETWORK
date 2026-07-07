"use client";

import { useQuery } from '@tanstack/react-query';
import { API_BASE_URL, apiRequest } from '@/lib/api';

export interface SecuritySubsystemStatus {
  backend: 'ebpf' | 'native' | 'unavailable' | string;
  mode: 'auto' | 'native' | 'ebpf' | string;
  status: 'active' | 'fallback' | 'unavailable' | string;
  fallback_reason?: string;
  environment?: {
    is_containerized: boolean;
    container_runtime: string;
    ebpf_capable: boolean;
    reason: string;
  };
  capabilities?: {
    policy_enforcer: boolean;
    rate_limiter: boolean;
    container_isolator: boolean;
    syscall_tracer: boolean;
  };
  manager?: {
    initialized: boolean;
    programs_attached: number;
  };
  policy?: {
    enabled: boolean;
  };
  rate_limiter?: {
    enabled: boolean;
    initialized: boolean;
    allowed_packets: number;
    dropped_packets: number;
    top_attackers?: Array<{
      ip: string;
      packets_dropped: number;
      timestamp?: string;
    }>;
  };
  isolator?: {
    enabled: boolean;
    active_containers: number;
    containers?: Array<{
      id: number;
      root_pid: number;
      root_fs: string;
      network_allowed: boolean;
    }>;
  };
  tracer?: {
    enabled: boolean;
    active: boolean;
    tracked_pids: number;
    syscall_events: number;
  };
  guardian?: {
    running: boolean;
    skipped: boolean;
    reason: string;
  };
}

const fallbackStatus: SecuritySubsystemStatus = {
  backend: 'unavailable',
  mode: 'auto',
  status: 'unavailable',
  fallback_reason: 'security subsystem endpoint unavailable',
  environment: {
    is_containerized: false,
    container_runtime: '',
    ebpf_capable: false,
    reason: 'security subsystem endpoint unavailable',
  },
  capabilities: {
    policy_enforcer: false,
    rate_limiter: false,
    container_isolator: false,
    syscall_tracer: false,
  },
  manager: {
    initialized: false,
    programs_attached: 0,
  },
  policy: {
    enabled: false,
  },
  rate_limiter: {
    enabled: false,
    initialized: false,
    allowed_packets: 0,
    dropped_packets: 0,
    top_attackers: [],
  },
  isolator: {
    enabled: false,
    active_containers: 0,
    containers: [],
  },
  tracer: {
    enabled: false,
    active: false,
    tracked_pids: 0,
    syscall_events: 0,
  },
  guardian: {
    running: false,
    skipped: true,
    reason: 'security subsystem endpoint unavailable',
  },
};

const normalizeStatus = (value: unknown): SecuritySubsystemStatus => {
  const raw = value as any;
  const data = raw?.data ?? raw;
  return {
    ...fallbackStatus,
    ...data,
    environment: { ...fallbackStatus.environment, ...(data?.environment ?? {}) },
    capabilities: { ...fallbackStatus.capabilities, ...(data?.capabilities ?? {}) },
    manager: { ...fallbackStatus.manager, ...(data?.manager ?? {}) },
    policy: { ...fallbackStatus.policy, ...(data?.policy ?? {}) },
    rate_limiter: { ...fallbackStatus.rate_limiter, ...(data?.rate_limiter ?? {}) },
    isolator: { ...fallbackStatus.isolator, ...(data?.isolator ?? {}) },
    tracer: { ...fallbackStatus.tracer, ...(data?.tracer ?? {}) },
    guardian: { ...fallbackStatus.guardian, ...(data?.guardian ?? {}) },
  };
};

export const useSecuritySubsystem = () => {
  const query = useQuery<SecuritySubsystemStatus>({
    queryKey: ['security-subsystem'],
    queryFn: async () => {
      const response = await apiRequest<SecuritySubsystemStatus>(`${API_BASE_URL}/api/security/subsystem`);
      return normalizeStatus(response);
    },
    refetchInterval: 10000,
    staleTime: 5000,
    retry: 1,
  });

  return {
    securitySubsystem: query.data ?? fallbackStatus,
    isLoading: query.isLoading,
    error: query.error instanceof Error ? query.error.message : null,
    refetch: query.refetch,
  };
};

export const getSecurityBackendLabel = (status: SecuritySubsystemStatus | null | undefined): string => {
  if (!status) return 'Unavailable';
  if (status.backend === 'ebpf') return 'eBPF';
  if (status.backend === 'native') return 'Native Fallback';
  return 'Unavailable';
};

export default useSecuritySubsystem;
