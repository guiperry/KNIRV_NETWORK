import { useState, useCallback, useRef, useEffect } from 'react';
import {
  startToolStream,
  stopToolStream,
  createToolStreamWebSocket,
  type ToolEvent,
} from '../services/sandboxToolService';
import { addToolReport } from '../services/toolReports';

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
  const eventsRef = useRef<ToolEvent[]>([]);
  const runRef = useRef<{ startedAt: string; startedAtMs: number; args: Record<string, unknown> } | null>(null);

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
    eventsRef.current = [];
    const runDetails = { startedAt: new Date().toISOString(), startedAtMs: Date.now(), args };
    runRef.current = runDetails;

    try {
      await startToolStream(sessionID, tool, args);
      const ws = createToolStreamWebSocket(sessionID, tool, {
        onMessage: (event) => {
          if (!mountedRef.current) return;
          try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'tool_event' && msg.event) {
              const toolEvent = msg.event as ToolEvent;
              eventsRef.current = [...eventsRef.current, toolEvent];
              setEvents(eventsRef.current);
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
      const message = err instanceof Error ? err.message : String(err);
      addToolReport({ tool, execution: 'stream', status: 'failed', sessionID, startedAt: runDetails.startedAt, completedAt: new Date().toISOString(), durationMs: Date.now() - runDetails.startedAtMs, args, output: '', error: message });
      runRef.current = null;
      if (mountedRef.current) {
        setError(message);
        setRunning(false);
      }
    } finally {
      if (mountedRef.current) {
        setStarting(false);
      }
    }
  }, [sessionID, tool]);

  const stop = useCallback(async () => {
    const runDetails = runRef.current;
    try {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      await stopToolStream(sessionID, tool);
      if (runDetails) {
        addToolReport({
          tool, execution: 'stream', status: 'completed', sessionID,
          startedAt: runDetails.startedAt, completedAt: new Date().toISOString(),
          durationMs: Date.now() - runDetails.startedAtMs, args: runDetails.args,
          output: eventsRef.current.map((event) => `[${event.timestamp}] ${event.rawLine || JSON.stringify(event.payload)}`).join('\n') || 'Stream stopped without events.',
        });
      }
    } catch {
      if (runDetails) {
        addToolReport({ tool, execution: 'stream', status: 'failed', sessionID, startedAt: runDetails.startedAt, completedAt: new Date().toISOString(), durationMs: Date.now() - runDetails.startedAtMs, args: runDetails.args, output: eventsRef.current.map((event) => event.rawLine || JSON.stringify(event.payload)).join('\n'), error: 'Unable to stop tool stream' });
      }
    } finally {
      runRef.current = null;
      if (mountedRef.current) {
        setRunning(false);
      }
    }
  }, [sessionID, tool]);

  const clearEvents = useCallback(() => {
    eventsRef.current = [];
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
