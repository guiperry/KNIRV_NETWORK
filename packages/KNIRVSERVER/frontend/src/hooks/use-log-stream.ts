'use client';

import { useEffect, useRef, useState, useCallback } from 'react';

export interface ModuleLog {
  type: string;
  module: string;
  level: string;
  message: string;
  source: string;
  metadata?: Record<string, unknown>;
  timestamp: string;
}

interface UseLogStreamOptions {
  module?: string;
  autoConnect?: boolean;
  onLog?: (log: ModuleLog) => void;
}

export function useLogStream(options: UseLogStreamOptions = {}) {
  const { module = '', autoConnect = true, onLog } = options;
  const [isConnected, setIsConnected] = useState(false);
  const [logs, setLogs] = useState<ModuleLog[]>([]);
  const pollingRef = useRef<number | null>(null);
  const latestSeenRef = useRef<string>('');

  const buildHistoryUrl = useCallback(() => {
    const params = new URLSearchParams({ limit: '100' });
    if (module) {
      params.set('module', module);
    }
    return `/api/logs/history?${params.toString()}`;
  }, [module]);

  const mergeLogs = useCallback((incoming: ModuleLog[]) => {
    if (incoming.length === 0) {
      return;
    }

    setLogs(prev => {
      const existing = new Set(
        prev.map(log => `${log.timestamp}|${log.module}|${log.level}|${log.message}`)
      );
      const next = [...prev];

      for (const log of incoming) {
        const key = `${log.timestamp}|${log.module}|${log.level}|${log.message}`;
        if (existing.has(key)) {
          continue;
        }
        existing.add(key);
        next.push(log);
        onLog?.(log);
      }

      if (next.length > 500) {
        return next.slice(-500);
      }
      return next;
    });
  }, [onLog]);

  const pollLogs = useCallback(async () => {
    try {
      const response = await fetch(buildHistoryUrl());
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const payload = await response.json();
      const history = Array.isArray(payload.logs) ? payload.logs as ModuleLog[] : [];
      const ordered = [...history].reverse();

      if (ordered.length > 0) {
        const latest = ordered[ordered.length - 1];
        latestSeenRef.current = `${latest.timestamp}|${latest.module}|${latest.level}|${latest.message}`;
      }

      mergeLogs(ordered);
      setIsConnected(true);
    } catch (error) {
      setIsConnected(false);
      console.debug('[LogStream] History polling unavailable:', error);
    }
  }, [buildHistoryUrl, mergeLogs]);

  const connect = useCallback(() => {
    if (pollingRef.current !== null) {
      return;
    }

    void pollLogs();
    pollingRef.current = window.setInterval(() => {
      void pollLogs();
    }, 3000);
    console.log(`[LogStream] Polling ${module || 'all modules'}`);
  }, [module, pollLogs]);

  const disconnect = useCallback(() => {
    if (pollingRef.current !== null) {
      window.clearInterval(pollingRef.current);
      pollingRef.current = null;
      setIsConnected(false);
      console.log('[LogStream] Disconnected');
    }
  }, []);

  useEffect(() => {
    if (autoConnect) {
      connect();
    }
    return () => disconnect();
  }, [autoConnect, connect, disconnect]);

  const clearLogs = useCallback(() => {
    setLogs([]);
    latestSeenRef.current = '';
  }, []);

  return {
    logs,
    isConnected,
    connect,
    disconnect,
    clearLogs,
  };
}

export function useModuleLogs(module: string) {
  return useLogStream({ module, autoConnect: true });
}
