import { useState, useEffect, useCallback } from 'react';

export interface BuiltinPluginInfo {
  id: string;
  name: string;
  category: string;
  description: string;
  version: string;
  enabled: boolean;
  running: boolean;
}

export function useBuiltinPlugins() {
  const [builtins, setBuiltins] = useState<BuiltinPluginInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchBuiltins = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/plugins/builtins');
      if (!res.ok) {
        throw new Error(`Failed to fetch builtin plugins: ${res.statusText}`);
      }
      const data = await res.json();
      setBuiltins(data.builtins ?? []);
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Unknown error';
      console.error('Failed to fetch builtin plugins', e);
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  const toggle = useCallback(async (id: string, enable: boolean) => {
    try {
      const res = await fetch(`/api/plugins/builtin/${id}/toggle`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enable }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error((body as { error?: string }).error ?? res.statusText);
      }
    } catch (e) {
      console.error(`Failed to toggle plugin ${id}`, e);
      throw e;
    } finally {
      await fetchBuiltins();
    }
  }, [fetchBuiltins]);

  const updateConfig = useCallback(async (id: string, patch: Record<string, unknown>) => {
    try {
      const res = await fetch(`/api/plugins/builtin/${id}/config`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(patch),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error((body as { error?: string }).error ?? res.statusText);
      }
    } catch (e) {
      console.error(`Failed to update config for plugin ${id}`, e);
      throw e;
    } finally {
      await fetchBuiltins();
    }
  }, [fetchBuiltins]);

  useEffect(() => {
    fetchBuiltins();
  }, [fetchBuiltins]);

  return { builtins, loading, error, toggle, updateConfig, refetch: fetchBuiltins };
}
