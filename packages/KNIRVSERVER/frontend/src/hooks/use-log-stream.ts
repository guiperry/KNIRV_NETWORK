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
  const eventSourceRef = useRef<EventSource | null>(null);

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      return;
    }

    const url = module 
      ? `/api/logs/module/${module}`
      : '/api/logs/stream';
    
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setIsConnected(true);
      console.log(`[LogStream] Connected to ${module || 'all modules'}`);
    };

    es.onmessage = (event) => {
      try {
        const log: ModuleLog = JSON.parse(event.data);
        setLogs(prev => {
          const updated = [...prev, log];
          if (updated.length > 500) {
            return updated.slice(-500);
          }
          return updated;
        });
        onLog?.(log);
      } catch (err) {
        console.error('[LogStream] Failed to parse log:', err);
      }
    };

    es.onerror = () => {
      setIsConnected(false);
      console.warn('[LogStream] Connection error, reconnecting...');
      eventSourceRef.current = null;
      setTimeout(() => {
        if (autoConnect) connect();
      }, 3000);
    };
  }, [module, autoConnect, onLog]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
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