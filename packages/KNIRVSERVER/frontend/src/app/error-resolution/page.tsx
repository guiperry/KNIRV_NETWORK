'use client';

import React, { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { AlertCircle } from 'lucide-react';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import type { ErrorResolutionSession } from '@/types/api';
import ErrorResolutionDashboard from '@/components/dve-management/error-resolution-dashboard';

function ErrorResolutionInner() {
  const searchParams = useSearchParams();
  const sessionId = searchParams.get('session_id');
  const [session, setSession] = useState<ErrorResolutionSession | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    if (!sessionId) {
      setError('Missing error-resolution session ID');
      return;
    }

    const loadSession = async () => {
      try {
        const response = await apiRequest<ErrorResolutionSession>(`${API_BASE_URL}/api/dve-creation/error-resolution-sessions/${sessionId}`);
        if (!response.success || !response.data) {
          throw new Error(response.error || 'Failed to load error-resolution session');
        }
        if (!cancelled) {
          setSession(response.data);
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : 'Failed to load error-resolution session');
        }
      }
    };

    loadSession();
    return () => {
      cancelled = true;
    };
  }, [sessionId]);

  if (error) {
    return (
      <div className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center p-8">
        <div className="flex items-center gap-3 text-red-400">
          <AlertCircle className="w-5 h-5" />
          <span>{error}</span>
        </div>
      </div>
    );
  }

  if (!session) {
    return <div className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center">Loading session...</div>;
  }

  return (
    <main className="min-h-screen bg-slate-950 text-slate-100 p-6">
      <div className="max-w-6xl mx-auto">
        <ErrorResolutionDashboard
          sessionId={session.id}
          supportedTypes={session.supported_error_types}
          port={session.port}
        />
      </div>
    </main>
  );
}

export default function ErrorResolutionPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center">Loading...</div>}>
      <ErrorResolutionInner />
    </Suspense>
  );
}
