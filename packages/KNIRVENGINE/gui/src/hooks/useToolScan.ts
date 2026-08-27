import { useState, useCallback, useRef, useEffect } from 'react';
import {
  runToolScan,
  runToolNative,
  type ToolScanResult,
} from '../services/sandboxToolService';
import { isLane6Tool } from './toolCapability';
import { addToolReport } from '../services/toolReports';

interface UseToolScanOptions {
  sessionID: string;
  tool: string;
  useNative?: boolean; // force Lane 6 path
}

interface UseToolScanReturn {
  result: ToolScanResult | null;
  rawOutput: string;
  structured: unknown;
  running: boolean;
  error: string | null;
  run: (args?: Record<string, unknown>) => Promise<void>;
  reset: () => void;
}

/**
 * useToolScan is the unified hook for Lane 1 (batch scan) and Lane 6 (native Go)
 * tool executions. Both lanes have the same request/response shape from the
 * frontend's perspective — the only difference is whether the backend spawned
 * a subprocess or called a Go function, which the frontend never needs to know.
 */
export function useToolScan({ sessionID, tool, useNative = false }: UseToolScanOptions): UseToolScanReturn {
  const [result, setResult] = useState<ToolScanResult | null>(null);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const run = useCallback(async (args: Record<string, unknown> = {}) => {
    const startedAt = new Date().toISOString();
    const startedAtMs = Date.now();
    setRunning(true);
    setError(null);
    try {
      const response = useNative || isLane6Tool(tool)
        ? await runToolNative(sessionID, tool, args)
        : await runToolScan(sessionID, tool, args);
      if (mountedRef.current) {
        setResult(response);
      }
      addToolReport({
        tool,
        execution: useNative || isLane6Tool(tool) ? 'analysis' : 'scan',
        status: 'completed',
        sessionID,
        startedAt: response.startedAt || startedAt,
        completedAt: new Date().toISOString(),
        durationMs: response.durationMs ?? Date.now() - startedAtMs,
        args,
        output: response.rawOutput || (response.structured ? JSON.stringify(response.structured, null, 2) : 'Completed with no output.'),
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      addToolReport({ tool, execution: useNative || isLane6Tool(tool) ? 'analysis' : 'scan', status: 'failed', sessionID, startedAt, completedAt: new Date().toISOString(), durationMs: Date.now() - startedAtMs, args, output: '', error: message });
      if (mountedRef.current) {
        setError(message);
      }
    } finally {
      if (mountedRef.current) {
        setRunning(false);
      }
    }
  }, [sessionID, tool, useNative]);

  const reset = useCallback(() => {
    setResult(null);
    setError(null);
  }, []);

  return {
    result,
    rawOutput: result?.rawOutput ?? '',
    structured: result?.structured ?? null,
    running,
    error,
    run,
    reset,
  };
}
