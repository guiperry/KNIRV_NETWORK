import { useState, useCallback, useRef, useEffect } from 'react';
import {
  attachTool,
  detachTool,
  getToolAttachWebSocketUrl,
  type ToolAttachState,
} from '../services/sandboxToolService';

interface UseToolAttachOptions {
  sessionID: string;
  tool: string;
}

interface UseToolAttachReturn {
  attached: boolean;
  pid: number | null;
  log: string[];
  error: string | null;
  attach: (pid: number, args?: Record<string, unknown>) => Promise<void>;
  detach: () => Promise<void>;
  send: (command: string, args?: Record<string, unknown>) => void;
  clearLog: () => void;
}

/**
 * useToolAttach is the hook for Lane 3 (RPC attach) tools.
 * It manages a bidirectional connection to a tool's RPC bridge process,
 * accepting commands and streaming responses over a WebSocket.
 */
export function useToolAttach({ sessionID, tool }: UseToolAttachOptions): UseToolAttachReturn {
  const [attached, setAttached] = useState(false);
  const [pid, setPid] = useState<number | null>(null);
  const [log, setLog] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, []);

  const attach = useCallback(async (targetPid: number, args: Record<string, unknown> = {}) => {
    setError(null);
    setLog([]);

    try {
      const state: ToolAttachState = await attachTool(sessionID, tool, targetPid, args);
      if (!mountedRef.current) return;

      setAttached(state.attached);
      setPid(state.pid ?? targetPid);
      setLog(state.log ?? []);

      const wsUrl = getToolAttachWebSocketUrl(sessionID, tool);
      const ws = new WebSocket(wsUrl);

      ws.onmessage = (event) => {
        if (!mountedRef.current) return;
        try {
          const msg = JSON.parse(event.data);
          switch (msg.type) {
            case 'attach_log':
              setLog((prev) => [...prev, msg.line as string]);
              break;
            case 'attach_detached':
              setAttached(false);
              break;
            default:
              break;
          }
        } catch {
          // Ignore malformed messages.
        }
      };

      ws.onerror = () => {
        if (mountedRef.current) {
          setError('WebSocket connection failed');
        }
      };

      wsRef.current = ws;
    } catch (err) {
      if (mountedRef.current) {
        setError(err instanceof Error ? err.message : String(err));
        setAttached(false);
      }
    }
  }, [sessionID, tool]);

  const detach = useCallback(async () => {
    try {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      await detachTool(sessionID, tool);
    } catch {
      // Ignore detach errors.
    } finally {
      if (mountedRef.current) {
        setAttached(false);
        setPid(null);
      }
    }
  }, [sessionID, tool]);

  const send = useCallback((command: string, args: Record<string, unknown> = {}) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ command, args }));
    }
  }, []);

  const clearLog = useCallback(() => {
    setLog([]);
  }, []);

  return {
    attached,
    pid,
    log,
    error,
    attach,
    detach,
    send,
    clearLog,
  };
}
