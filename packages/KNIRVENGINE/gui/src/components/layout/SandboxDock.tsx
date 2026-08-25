import React, { lazy, Suspense, useState, useRef, useCallback } from 'react';
import { ChevronDown, ChevronUp, GripHorizontal, Square, Loader2 } from 'lucide-react';
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
  const { session, status, isReady, stop } = useSandbox();
  const [collapsed, setCollapsed] = useState(false);
  const [height, setHeight] = useState(300);
  const [connected, setConnected] = useState(false);
  const draggingRef = useRef<{ startY: number; startHeight: number } | null>(null);

  const onStatus = useCallback((c: boolean) => setConnected(c), []);

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
        <div className="relative min-h-0 flex-1">
          <Suspense fallback={<div className="h-full w-full bg-black" />}>
            <SandboxVncCanvas wsUrl={isReady && session.vncWsPath ? getWebSocketUrl(session.vncWsPath) : undefined} onStatus={onStatus} />
          </Suspense>
          {!isReady && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/60 text-xs font-mono text-slate-500">
              {status === 'provisioning' ? 'starting framebuffer…' : 'sandbox not running'}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default SandboxDock;
