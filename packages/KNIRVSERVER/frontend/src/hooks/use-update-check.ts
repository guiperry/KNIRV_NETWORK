'use client';

import { useCallback, useEffect, useState } from 'react';

export interface UpdateStatus {
  available: boolean;
  latest_tag: string;
  current_version: string;
  notes?: string;
  checked_at: string;
}

const POLL_INTERVAL_MS = 5 * 60 * 1000;

export function useUpdateCheck() {
  const [status, setStatus] = useState<UpdateStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [applying, setApplying] = useState(false);

  const checkUpdate = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/system/update');
      if (!res.ok) return;
      const data: UpdateStatus = await res.json();
      setStatus(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Update check failed');
    }
  }, []);

  const applyUpdate = useCallback(async () => {
    setApplying(true);
    try {
      await fetch('/api/v1/system/update/apply', { method: 'POST' });
    } catch {
      // Expected — server restarts and drops the connection.
    }
  }, []);

  useEffect(() => {
    checkUpdate();
    const interval = setInterval(checkUpdate, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [checkUpdate]);

  return { status, error, applying, checkUpdate, applyUpdate };
}
