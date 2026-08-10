"use client";

import { useQuery } from '@tanstack/react-query';

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
    throw new Error(`request failed: ${response.status}`);
  }
  const payload = await response.json();
  return (payload.data ?? payload) as T;
}

export function useOnboardingAdmin() {
  const applications = useQuery<{ applications: OnboardingApplication[] }>({
    queryKey: ['onboarding', 'applications'],
    queryFn: async () => unwrap<{ applications: OnboardingApplication[] }>(await fetch('/api/v1/onboarding/applications')),
    refetchInterval: 30000,
    staleTime: 15000,
    retry: 1,
  });

  const users = useQuery<{ users: OnboardingUser[] }>({
    queryKey: ['onboarding', 'users'],
    queryFn: async () => unwrap<{ users: OnboardingUser[] }>(await fetch('/api/v1/onboarding/users')),
    refetchInterval: 30000,
    staleTime: 15000,
    retry: 1,
  });

  const isLoading = applications.isLoading || users.isLoading;
  const error = applications.error instanceof Error ? applications.error.message : users.error instanceof Error ? users.error.message : null;

  return { applications, users, isLoading, error };
}

export default useOnboardingAdmin;
