"use client";

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { getAuthHeaders } from '@/lib/api';

export interface OnboardingApplication {
  id: string;
  legal_name: string;
  kyc_status: string;
  status: string;
  created_at: string;
}

export interface OnboardingUser {
  id: string;
  username: string;
  email: string;
  role: string;
  account_status: string;
  plan: string;
}

async function unwrap<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const payload = await response.json().catch(() => null);
    const message = payload?.data?.error ?? payload?.error ?? `request failed: ${response.status}`;
    throw new Error(message);
  }
  const payload = await response.json();
  return (payload.data ?? payload) as T;
}

async function postJSON<T>(url: string, body: unknown): Promise<T> {
  return unwrap<T>(
    await fetch(url, {
      method: 'POST',
      headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  );
}

export function useOnboardingAdmin() {
  const applications = useQuery<{ applications: OnboardingApplication[] }>({
    queryKey: ['onboarding', 'applications'],
    queryFn: async () => unwrap<{ applications: OnboardingApplication[] }>(await fetch('/api/v1/onboarding/applications', { headers: getAuthHeaders() })),
    refetchInterval: 30000,
    staleTime: 15000,
    retry: 1,
  });

  const users = useQuery<{ users: OnboardingUser[] }>({
    queryKey: ['onboarding', 'users'],
    queryFn: async () => unwrap<{ users: OnboardingUser[] }>(await fetch('/api/v1/onboarding/users', { headers: getAuthHeaders() })),
    refetchInterval: 30000,
    staleTime: 15000,
    retry: 1,
  });

  const isLoading = applications.isLoading || users.isLoading;
  const error = applications.error instanceof Error ? applications.error.message : users.error instanceof Error ? users.error.message : null;

  return { applications, users, isLoading, error };
}

/** Approve or reject a pending operator application. */
export function useReviewOperatorApplication() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, decision, note }: { id: string; decision: 'approved' | 'rejected'; note?: string }) =>
      postJSON(`/api/v1/onboarding/applications/${encodeURIComponent(id)}/review`, { decision, note }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['onboarding', 'applications'] });
    },
  });
}

/** Change a user account's role (admin / validator / observer). */
export function useUpdateUserRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, role }: { id: string; role: 'admin' | 'validator' | 'observer' }) =>
      postJSON(`/api/v1/onboarding/users/${encodeURIComponent(id)}/role`, { role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['onboarding', 'users'] });
    },
  });
}

/** Change a user account's status (active / suspended / banned). */
export function useUpdateUserStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: 'active' | 'suspended' | 'banned' }) =>
      postJSON(`/api/v1/onboarding/users/${encodeURIComponent(id)}/status`, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['onboarding', 'users'] });
    },
  });
}

export default useOnboardingAdmin;
