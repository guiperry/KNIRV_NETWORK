import { useState, useCallback, useRef, useEffect } from 'react';
import {
  attachTool,
  detachTool,
  getToolAttachWebSocketUrl,
  type ToolAttachState,
} from '../services/sandboxToolService';
import { addToolReport } from '../services/toolReports';

interface UseToolAttachOptions {
  sessionID: string;
  tool: string;
}

interface UseToolAttachReturn {
  attached: boolean;
  attaching: boolean;
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
  const [attaching, setAttaching] = useState(false);
  const [pid, setPid] = useState<number | null>(null);
  const [log, setLog] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const mountedRef = useRef(true);
  const logRef = useRef<string[]>([]);
  const attachRef = useRef<{ startedAt: string; startedAtMs: number; args: Record<string, unknown> } | null>(null);

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
    const attachDetails = { startedAt: new Date().toISOString(), startedAtMs: Date.now(), args: { pid: targetPid, ...args } };
    setAttaching(true);
    setError(null);
    setLog([]);
    logRef.current = [];

    try {
      const state: ToolAttachState = await attachTool(sessionID, tool, targetPid, args);
      if (!mountedRef.current) return;

      setAttached(state.attached);
      setPid(state.pid ?? targetPid);
      setLog(state.log ?? []);
      logRef.current = state.log ?? [];
      attachRef.current = attachDetails;

      const wsUrl = getToolAttachWebSocketUrl(sessionID, tool);
      const ws = new WebSocket(wsUrl);

      ws.onmessage = (event) => {
        if (!mountedRef.current) return;
        try {
          const msg = JSON.parse(event.data);
          switch (msg.type) {
            case 'attach_log':
              logRef.current = [...logRef.current, msg.line as string];
              setLog(logRef.current);
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
      const message = err instanceof Error ? err.message : String(err);
      addToolReport({ tool, execution: 'attach', status: 'failed', sessionID, startedAt: attachDetails.startedAt, completedAt: new Date().toISOString(), durationMs: Date.now() - attachDetails.startedAtMs, args: attachDetails.args, output: '', error: message });
      if (mountedRef.current) {
        setError(message);
        setAttached(false);
      }
    } finally {
      if (mountedRef.current) {
        setAttaching(false);
      }
    }
  }, [sessionID, tool]);

  const detach = useCallback(async () => {
    const attachDetails = attachRef.current;
    try {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      await detachTool(sessionID, tool);
      if (attachDetails) addToolReport({ tool, execution: 'attach', status: 'completed', sessionID, startedAt: attachDetails.startedAt, completedAt: new Date().toISOString(), durationMs: Date.now() - attachDetails.startedAtMs, args: attachDetails.args, output: logRef.current.join('\n') || 'Detached without bridge output.' });
    } catch {
      if (attachDetails) addToolReport({ tool, execution: 'attach', status: 'failed', sessionID, startedAt: attachDetails.startedAt, completedAt: new Date().toISOString(), durationMs: Date.now() - attachDetails.startedAtMs, args: attachDetails.args, output: logRef.current.join('\n'), error: 'Unable to detach tool' });
    } finally {
      attachRef.current = null;
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
    logRef.current = [];
    setLog([]);
  }, []);

  return {
    attached,
    attaching,
    pid,
    log,
    error,
    attach,
    detach,
    send,
    clearLog,
  };
}
