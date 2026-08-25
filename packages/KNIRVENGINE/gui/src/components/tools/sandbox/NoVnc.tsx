import React, { useCallback, useEffect, useState } from 'react';
import { Clipboard, MonitorPlay, Wifi } from 'lucide-react';
import { useSandbox } from '../../SandboxContext';
import { getWebSocketUrl } from '../../../utils/apiBase';
import SandboxVncCanvas from './SandboxVncCanvas';

const NoVnc: React.FC = () => {
  const { session, isReady } = useSandbox();
  const [connected, setConnected] = useState(false);
  const [connectRequested, setConnectRequested] = useState(true);
  const [quality, setQuality] = useState(6);
  const [viewOnly, setViewOnly] = useState(false);

  const onStatus = useCallback((isConnected: boolean) => setConnected(isConnected), []);

  // A newly launched session connects automatically; a stopped session must
  // not retain a stale connected indicator.
  useEffect(() => {
    setConnectRequested(true);
    if (!isReady) setConnected(false);
  }, [isReady, session?.id]);

  const canConnect = Boolean(session && isReady && session.vncWsPath);
  const wsUrl = canConnect && connectRequested && session?.vncWsPath
    ? getWebSocketUrl(session.vncWsPath)
    : undefined;

  const syncClipboard = async () => {
    if (!navigator.clipboard) return;
    try {
      await navigator.clipboard.readText();
    } catch {
      // Clipboard access requires a user-granted browser permission.
    }
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div className="rounded-lg bg-pink-500/20 p-2"><MonitorPlay className="h-6 w-6 text-pink-400" /></div>
          <div>
            <h1 className="text-2xl font-bold text-white">noVNC</h1>
            <p className="font-mono text-sm text-slate-400">RFB over WebSocket · x11vnc bridging Xvfb :99</p>
          </div>
        </div>
        <div className={`flex items-center space-x-2 rounded-lg border px-3 py-1.5 font-mono text-xs ${
          connected ? 'border-green-500/30 bg-green-500/15 text-green-300' : 'border-slate-700/50 bg-slate-800/50 text-slate-400'
        }`}>
          <Wifi className="h-3.5 w-3.5" />
          <span>{connected ? 'connected' : canConnect ? 'connecting' : 'disconnected'}</span>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
        <div className="space-y-3 rounded-lg border border-slate-700/50 bg-slate-800/50 p-4 xl:col-span-1">
          <div className="mb-1 text-xs uppercase text-slate-500">Connection</div>
          <div className="space-y-2 font-mono text-xs">
            <div className="text-slate-500">sandbox session</div>
            <div className="break-all rounded bg-slate-900/60 px-2 py-1.5 text-slate-200">{session?.id ?? 'none'}</div>
            <div className="text-slate-500">websocket path</div>
            <div className="break-all rounded bg-slate-900/60 px-2 py-1.5 text-slate-200">{session?.vncWsPath ?? 'launch a sandbox to connect'}</div>
          </div>
          <button disabled={!canConnect} onClick={() => setConnectRequested((requested) => !requested)} className="w-full rounded bg-pink-500/20 px-2 py-1.5 text-xs text-pink-300 hover:bg-pink-500/30 disabled:cursor-not-allowed disabled:opacity-50">
            {connectRequested && canConnect ? 'Disconnect' : 'Connect'}
          </button>

          <div className="space-y-2 border-t border-slate-700/50 pt-2">
            <div className="flex items-center justify-between text-xs"><span className="font-mono text-slate-500">quality</span><span className="font-mono text-slate-300">{quality}/9</span></div>
            <input type="range" min={0} max={9} value={quality} onChange={(event) => setQuality(Number(event.target.value))} className="w-full accent-pink-500" />
            <label className="flex items-center space-x-2 font-mono text-xs text-slate-300"><input type="checkbox" checked={viewOnly} onChange={(event) => setViewOnly(event.target.checked)} className="accent-pink-500" /><span>view only</span></label>
            <button onClick={syncClipboard} className="flex items-center space-x-1 text-xs text-slate-400 hover:text-white"><Clipboard className="h-3.5 w-3.5" /><span>request clipboard access</span></button>
          </div>
        </div>

        <div className="relative min-h-[440px] overflow-hidden rounded-lg border border-slate-700/50 bg-black/60 xl:col-span-3">
          {wsUrl ? <SandboxVncCanvas wsUrl={wsUrl} viewOnly={viewOnly} quality={quality} onStatus={onStatus} /> : (
            <div className="absolute inset-0 flex items-center justify-center font-mono text-sm text-slate-500">{session ? 'sandbox framebuffer is not running' : 'launch a sandbox to open its framebuffer'}</div>
          )}
        </div>
      </div>
    </div>
  );
};

export default NoVnc;
