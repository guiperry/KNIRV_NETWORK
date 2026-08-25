import React, { lazy, Suspense, useState, useRef, useCallback } from 'react';
import { ChevronDown, ChevronUp, GripHorizontal, Square, Loader2, Clipboard, Check } from 'lucide-react';
import { useSandbox } from '../SandboxContext';
import { getWebSocketUrl } from '../../utils/apiBase';

// noVNC includes the RFB client. It is only useful after a sandbox launches,
// so keep it out of the initial application bundle.
const SandboxVncCanvas = lazy(() => import('../tools/sandbox/SandboxVncCanvas'));

/**
 * Persistent bottom-docked panel that hosts the live sandbox framebuffer.
 * Always mounted in AppLayout (never conditionally unmounted) so the RFB
 * connection survives route navigation. Collapse state + drag-resize are local.
 */
export const SandboxDock: React.FC = () => {
  const { session, status, isReady, stop, log, error, clearLog } = useSandbox();
  const [collapsed, setCollapsed] = useState(false);
  const [height, setHeight] = useState(300);
  const [connected, setConnected] = useState(false);
  const [copyStatus, setCopyStatus] = useState<'idle' | 'copied' | 'failed'>('idle');
  const draggingRef = useRef<{ startY: number; startHeight: number } | null>(null);

  const onStatus = useCallback((c: boolean) => setConnected(c), []);

  const copyLog = async () => {
    const text = [error, ...log].filter((line): line is string => Boolean(line)).join('\n');
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopyStatus('copied');
    } catch {
      const textarea = document.createElement('textarea');
      textarea.value = text;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.select();
      const copied = document.execCommand('copy');
      textarea.remove();
      setCopyStatus(copied ? 'copied' : 'failed');
    }
  };

  const onDragStart = (e: React.MouseEvent) => {
    draggingRef.current = { startY: e.clientY, startHeight: height };
    const move = (ev: MouseEvent) => {
      if (!draggingRef.current) return;
      const delta = draggingRef.current.startY - ev.clientY;
      const next = Math.max(140, Math.min(640, draggingRef.current.startHeight + delta));
      setHeight(next);
    };
    const up = () => {
      draggingRef.current = null;
      window.removeEventListener('mousemove', move);
      window.removeEventListener('mouseup', up);
    };
    window.addEventListener('mousemove', move);
    window.addEventListener('mouseup', up);
  };

  if (!session) return null;

  return (
    <div
      className="flex shrink-0 flex-col border-t border-slate-700/50 bg-slate-900"
      style={{ height: collapsed ? 40 : height }}
    >
      <div className="flex h-10 shrink-0 items-center justify-between border-b border-slate-800 bg-slate-950/60 px-3">
        <div className="flex items-center gap-2">
          {!collapsed && (
            <span
              onMouseDown={onDragStart}
              className="cursor-ns-resize text-slate-600 hover:text-slate-400"
              title="Drag to resize"
            >
              <GripHorizontal className="h-4 w-4" />
            </span>
          )}
          <span className="text-xs font-medium uppercase tracking-wider text-fuchsia-300">
            Sandbox
          </span>
          <span className="font-mono text-xs text-slate-400">{session.targetLabel}</span>
          <span
            className={`rounded px-1.5 py-0.5 text-[10px] font-mono ${
              isReady
                ? 'bg-green-500/15 text-green-300'
                : status === 'provisioning'
                ? 'bg-amber-500/15 text-amber-300'
                : 'bg-slate-700/50 text-slate-400'
            }`}
          >
            {isReady ? (connected ? 'live' : 'starting…') : status || 'idle'}
          </span>
        </div>
        <div className="flex items-center gap-1">
          {isReady && (
            <button
              onClick={() => stop()}
              title="Stop sandbox"
              className="rounded p-1 text-slate-400 hover:bg-slate-700/50 hover:text-red-300"
            >
              <Square className="h-3.5 w-3.5" />
            </button>
          )}
          {status === 'provisioning' && (
            <Loader2 className="h-3.5 w-3.5 animate-spin text-amber-300" />
          )}
          <button
            onClick={() => setCollapsed((v) => !v)}
            title={collapsed ? 'Expand' : 'Collapse'}
            className="rounded p-1 text-slate-400 hover:bg-slate-700/50 hover:text-white"
          >
            {collapsed ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>
        </div>
      </div>

      {!collapsed && (
        <div className="grid min-h-0 flex-1 grid-cols-1 gap-px bg-slate-800 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,34%)]">
          <section className="flex min-h-0 flex-col bg-slate-900 p-3" aria-label="Namespace log">
            <div className="mb-2 flex items-center justify-between gap-2"><div className="text-[10px] font-medium uppercase tracking-wider text-slate-500">Namespace log</div><div className="flex items-center gap-1"><button type="button" onClick={clearLog} disabled={!error && log.length === 0} className="rounded px-2 py-1 text-xs text-slate-400 hover:bg-slate-700/50 hover:text-white disabled:cursor-not-allowed disabled:opacity-40">Clear</button><button type="button" onClick={copyLog} disabled={!error && log.length === 0} className="flex items-center gap-1 rounded px-2 py-1 text-xs text-slate-400 hover:bg-slate-700/50 hover:text-white disabled:cursor-not-allowed disabled:opacity-40" aria-label="Copy namespace log">{copyStatus === 'copied' ? <Check className="h-3.5 w-3.5 text-green-400" /> : <Clipboard className="h-3.5 w-3.5" />}<span>{copyStatus === 'copied' ? 'Copied' : copyStatus === 'failed' ? 'Copy failed' : 'Copy all'}</span></button></div></div>
            <div className="min-h-0 flex-1 select-text overflow-y-auto rounded bg-slate-950/60 p-2 font-mono text-xs text-slate-400">
              {error && <div className="whitespace-pre-wrap text-red-400">{error}</div>}
              {log.length ? log.map((line, index) => <div key={`${index}-${line}`} className="whitespace-pre-wrap text-green-400">{line}</div>) : <span className="text-slate-600">waiting for sandbox output</span>}
            </div>
          </section>
          <section className="relative min-h-[10rem] bg-black" aria-label="Target application view">
            <div className="absolute left-3 top-2 z-10 rounded bg-slate-950/80 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wider text-fuchsia-300">Target view</div>
            <Suspense fallback={<div className="h-full w-full bg-black" />}>
              <SandboxVncCanvas wsUrl={isReady && session.vncWsPath ? getWebSocketUrl(session.vncWsPath) : undefined} onStatus={onStatus} />
            </Suspense>
            {!isReady && (
              <div className="absolute inset-0 flex items-center justify-center bg-black/60 text-xs font-mono text-slate-500">
                {status === 'provisioning' ? 'starting framebuffer…' : 'sandbox not running'}
              </div>
            )}
          </section>
        </div>
      )}
    </div>
  );
};

export default SandboxDock;
