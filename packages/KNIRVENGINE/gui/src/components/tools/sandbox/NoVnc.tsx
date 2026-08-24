import React, { useState } from 'react';
import { MonitorPlay, Wifi, Clipboard, RefreshCw } from 'lucide-react';

const NoVnc: React.FC = () => {
  const [connected, setConnected] = useState(true);
  const [host, setHost] = useState('127.0.0.1');
  const [port, setPort] = useState('6080');
  const [path, setPath] = useState('websockify');
  const [quality, setQuality] = useState(6);
  const [viewOnly, setViewOnly] = useState(false);

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-pink-500/20 rounded-lg">
            <MonitorPlay className="w-6 h-6 text-pink-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">noVNC</h1>
            <p className="text-slate-400 text-sm font-mono">RFB over WebSocket · x11vnc bridging Xvfb :99</p>
          </div>
        </div>
        <div className={`flex items-center space-x-2 px-3 py-1.5 rounded-lg text-xs font-mono border ${
          connected ? 'bg-green-500/15 border-green-500/30 text-green-300' : 'bg-slate-800/50 border-slate-700/50 text-slate-400'
        }`}>
          <Wifi className="w-3.5 h-3.5" />
          <span>{connected ? 'connected' : 'disconnected'}</span>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-4 gap-4">
        {/* connection panel */}
        <div className="xl:col-span-1 bg-slate-800/50 border border-slate-700/50 rounded-lg p-4 space-y-3">
          <div className="text-xs text-slate-500 uppercase mb-1">Connection</div>
          <div className="space-y-2 font-mono text-xs">
            <div>
              <div className="text-slate-500 mb-1">host</div>
              <input value={host} onChange={e => setHost(e.target.value)} className="w-full bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1.5 text-slate-200" />
            </div>
            <div>
              <div className="text-slate-500 mb-1">port</div>
              <input value={port} onChange={e => setPort(e.target.value)} className="w-full bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1.5 text-slate-200" />
            </div>
            <div>
              <div className="text-slate-500 mb-1">path</div>
              <input value={path} onChange={e => setPath(e.target.value)} className="w-full bg-slate-900/60 border border-slate-700/50 rounded px-2 py-1.5 text-slate-200" />
            </div>
          </div>
          <button
            onClick={() => setConnected(v => !v)}
            className={`w-full text-xs px-2 py-1.5 rounded ${
              connected ? 'bg-slate-700/50 text-slate-300 hover:bg-slate-700' : 'bg-pink-500/20 text-pink-300 hover:bg-pink-500/30'
            }`}
          >
            {connected ? 'Disconnect' : 'Connect'}
          </button>

          <div className="pt-2 border-t border-slate-700/50 space-y-2">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500 font-mono">quality</span>
              <span className="text-slate-300 font-mono">{quality}/9</span>
            </div>
            <input
              type="range"
              min={0}
              max={9}
              value={quality}
              onChange={e => setQuality(Number(e.target.value))}
              className="w-full accent-pink-500"
            />
            <label className="flex items-center space-x-2 text-xs font-mono text-slate-300">
              <input type="checkbox" checked={viewOnly} onChange={e => setViewOnly(e.target.checked)} className="accent-pink-500" />
              <span>view only</span>
            </label>
            <button className="flex items-center space-x-1 text-xs text-slate-400 hover:text-white">
              <Clipboard className="w-3.5 h-3.5" />
              <span>sync clipboard</span>
            </button>
          </div>
        </div>

        {/* canvas */}
        <div className="xl:col-span-3 bg-black/60 border border-slate-700/50 rounded-lg relative overflow-hidden" style={{ minHeight: 440 }}>
          {connected ? (
            <div className="absolute inset-0 flex flex-col">
              <div className="flex items-center justify-between px-3 py-1.5 bg-slate-900/80 text-[10px] font-mono text-slate-500 border-b border-slate-800">
                <span>rfb://{host}:{port}/{path} — 1280x800 · 24bpp</span>
                <RefreshCw className="w-3 h-3" />
              </div>
              <div className="flex-1 flex items-center justify-center">
                <div className="text-center">
                  <div className="w-16 h-16 mx-auto mb-3 rounded-lg bg-slate-800/60 flex items-center justify-center border border-slate-700/50">
                    <MonitorPlay className="w-8 h-8 text-pink-400/60" />
                  </div>
                  <div className="text-slate-500 text-xs font-mono">Xvfb :99 — target_binary window</div>
                  <div className="text-slate-700 text-[10px] font-mono mt-1">framebuffer stream renders here once the RFB session is live</div>
                </div>
              </div>
            </div>
          ) : (
            <div className="absolute inset-0 flex items-center justify-center text-slate-700 text-sm font-mono">
              not connected — press Connect
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default NoVnc;
