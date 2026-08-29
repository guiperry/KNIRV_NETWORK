"use client";

import { useState, useEffect } from 'react';
import { apiRequest, APIResponse } from '@/lib/api';
import type { DVESession } from '@/types/api';

export function useDVESessions(creationId: string | null) {
  const [sessions, setSessions] = useState<DVESession[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!creationId) {
      setSessions([]);
      return;
    }
    setLoading(true);
    setError(null);
    apiRequest<DVESession[]>(`/api/dve-creation/nodes/${creationId}/sessions`, { method: 'GET' })
      .then((data: APIResponse<DVESession[]>) => {
        setSessions(Array.isArray(data.data) ? data.data : []);
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [creationId]);

  return { sessions, loading, error };
}

export default useDVESessions;
