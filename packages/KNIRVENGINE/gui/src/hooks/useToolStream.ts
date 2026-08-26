import { useState, useCallback, useRef, useEffect } from 'react';
import {
  startToolStream,
  stopToolStream,
  createToolStreamWebSocket,
  type ToolEvent,
} from '../services/sandboxToolService';

interface UseToolStreamOptions {
  sessionID: string;
  tool: string;
}

interface UseToolStreamReturn {
  events: ToolEvent[];
  starting: boolean;
  running: boolean;
  error: string | null;
  start: (args?: Record<string, unknown>) => Promise<void>;
  stop: () => Promise<void>;
  clearEvents: () => void;
}

/**
 * useToolStream is the hook for Lane 2 (streaming daemon) tools.
 * It manages a long-lived subprocess whose output is fanned out over
 * a WebSocket, with start/stop lifecycle.
 */
export function useToolStream({ sessionID, tool }: UseToolStreamOptions): UseToolStreamReturn {
  const [events, setEvents] = useState<ToolEvent[]>([]);
  const [starting, setStarting] = useState(false);
  const [running, setRunning] = useState(false);
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

  const start = useCallback(async (args: Record<string, unknown> = {}) => {
    setStarting(true);
    setRunning(true);
    setError(null);
    setEvents([]);

    try {
      await startToolStream(sessionID, tool, args);
      const ws = createToolStreamWebSocket(sessionID, tool, {
        onMessage: (event) => {
          if (!mountedRef.current) return;
          try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'tool_event' && msg.event) {
              setEvents((prev) => [...prev, msg.event as ToolEvent]);
            }
          } catch {
            // Ignore malformed messages.
          }
        },
        onClose: () => {
          if (mountedRef.current) {
            setRunning(false);
          }
        },
        onError: () => {
          if (mountedRef.current) {
            setError('WebSocket connection failed');
            setRunning(false);
          }
        },
      });
      wsRef.current = ws;
    } catch (err) {
      if (mountedRef.current) {
        setError(err instanceof Error ? err.message : String(err));
        setRunning(false);
      }
    } finally {
      if (mountedRef.current) {
        setStarting(false);
      }
    }
  }, [sessionID, tool]);

  const stop = useCallback(async () => {
    try {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      await stopToolStream(sessionID, tool);
    } catch {
      // Ignore stop errors.
    } finally {
      if (mountedRef.current) {
        setRunning(false);
      }
    }
  }, [sessionID, tool]);

  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  return {
    events,
    starting,
    running,
    error,
    start,
    stop,
    clearEvents,
  };
}
