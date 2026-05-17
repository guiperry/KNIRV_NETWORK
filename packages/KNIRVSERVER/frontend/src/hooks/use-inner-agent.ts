'use client';

import { useState, useCallback } from 'react';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

export interface InnerAgentTool {
  name: string;
  binary: string;
  is_installed: boolean;
}

export interface InnerAgentSession {
  id: string;
  tool_name: string;
  status: 'running' | 'exited';
  started_at: string;
}

export function useInnerAgent(dveId: string | undefined) {
  const [tools, setTools] = useState<InnerAgentTool[]>([]);
  const [sessions, setSessions] = useState<InnerAgentSession[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const listTools = useCallback(async () => {
    if (!dveId) return;
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/dve/${dveId}/inner/tools`, {
        headers: getAuthHeaders(),
      });
      if (resp.ok) {
        const data = await resp.json();
        setTools(Array.isArray(data) ? data : data.tools ?? []);
      }
    } catch (err) {
      setError(String(err));
    }
  }, [dveId]);

  const listSessions = useCallback(async () => {
    if (!dveId) return;
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/dve/${dveId}/inner/sessions`, {
        headers: getAuthHeaders(),
      });
      if (resp.ok) {
        const data = await resp.json();
        setSessions(Array.isArray(data) ? data : data.sessions ?? []);
      }
    } catch (err) {
      setError(String(err));
    }
  }, [dveId]);

  const installTool = useCallback(async (toolName: string): Promise<boolean> => {
    if (!dveId) return false;
    setLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/dve/${dveId}/inner/install`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ tool: toolName }),
      });
      if (!resp.ok) {
        const body = await resp.text();
        setError(`Install failed: ${body}`);
        return false;
      }
      await listTools();
      return true;
    } catch (err) {
      setError(String(err));
      return false;
    } finally {
      setLoading(false);
    }
  }, [dveId, listTools]);

  const spawnSession = useCallback(async (toolName: string): Promise<string | null> => {
    if (!dveId) return null;
    setLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/dve/${dveId}/inner/spawn`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ tool: toolName }),
      });
      if (!resp.ok) {
        const body = await resp.text();
        setError(`Spawn failed: ${body}`);
        return null;
      }
      const data = await resp.json();
      const sessionId: string = data.id ?? data.session_id;
      if (sessionId) {
        setSessions(prev => [...prev, {
          id: sessionId,
          tool_name: toolName,
          status: 'running',
          started_at: new Date().toISOString(),
        }]);
      }
      return sessionId ?? null;
    } catch (err) {
      setError(String(err));
      return null;
    } finally {
      setLoading(false);
    }
  }, [dveId]);

  const terminateSession = useCallback(async (sessionId: string) => {
    if (!dveId) return;
    try {
      await fetch(`${API_BASE_URL}/api/v1/dve/${dveId}/inner/sessions/${sessionId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      setSessions(prev => prev.filter(s => s.id !== sessionId));
    } catch {
      // best-effort
    }
  }, [dveId]);

  return {
    tools, sessions, loading, error,
    listTools, listSessions, installTool, spawnSession, terminateSession,
  };
}
