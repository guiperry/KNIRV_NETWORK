import React, { useState } from 'react';
import { Radio, Play, Pause, Trash2, RotateCcw, ShieldAlert, Circle } from 'lucide-react';

type FlowMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';

interface Flow {
  id: number;
  method: FlowMethod;
  host: string;
  path: string;
  status: number | null;
  contentType: string;
  size: string;
  time: string;
  tls: boolean;
  intercepted: boolean;
}

const seedFlows: Flow[] = [
  { id: 1, method: 'GET', host: 'api.knirv.network', path: '/v1/agents/registry', status: 200, contentType: 'application/json', size: '4.2kb', time: '182ms', tls: true, intercepted: false },
  { id: 2, method: 'POST', host: 'oracle.knirv.network', path: '/oracle/checkpoint', status: 201, contentType: 'application/json', size: '812b', time: '96ms', tls: true, intercepted: false },
  { id: 3, method: 'GET', host: 'cdn.assets.io', path: '/fonts/inter-var.woff2', status: 200, contentType: 'font/woff2', size: '48.1kb', time: '41ms', tls: true, intercepted: false },
  { id: 4, method: 'POST', host: 'auth.targetapp.local', path: '/api/session/refresh', status: null, contentType: '-', size: '-', time: '-', tls: true, intercepted: true },
  { id: 5, method: 'PUT', host: 'gateway.internal', path: '/api/knirvshell/exec', status: 403, contentType: 'application/json', size: '96b', time: '12ms', tls: false, intercepted: false },
  { id: 6, method: 'GET', host: 'telemetry.targetapp.local', path: '/collect?ev=app_open', status: 204, contentType: '-', size: '0b', time: '58ms', tls: true, intercepted: false },
  { id: 7, method: 'DELETE', host: 'api.knirv.network', path: '/v1/skills/sk_88f2', status: 200, contentType: 'application/json', size: '212b', time: '73ms', tls: true, intercepted: false },
];

const methodColor: Record<FlowMethod, string> = {
  GET: 'text-blue-400',
  POST: 'text-green-400',
  PUT: 'text-yellow-400',
  DELETE: 'text-red-400',
  PATCH: 'text-purple-400',
};

const statusColor = (status: number | null) => {
  if (status === null) return 'text-slate-500';
  if (status >= 500) return 'text-red-400';
  if (status >= 400) return 'text-orange-400';
  if (status >= 300) return 'text-yellow-400';
  return 'text-green-400';
};

export const Proxy: React.FC = () => {
  const [running, setRunning] = useState(true);
  const [intercepting, setIntercepting] = useState(false);
  const [filter, setFilter] = useState('~tls');
  const [flows, setFlows] = useState(seedFlows);
  const [selectedId, setSelectedId] = useState<number | null>(4);

  const selected = flows.find(f => f.id === selectedId) ?? null;

  const filtered = flows.filter(f => {
    if (!filter.trim()) return true;
    const q = filter.trim().toLowerCase();
    if (q === '~tls') return f.tls;
    if (q.startsWith('~m ')) return f.method.toLowerCase() === q.slice(3).trim();
    if (q.startsWith('~d ')) return f.host.toLowerCase().includes(q.slice(3).trim());
    return `${f.method} ${f.host}${f.path}`.toLowerCase().includes(q);
  });

  const releaseFlow = (id: number) => {
    setFlows(prev => prev.map(f => f.id === id ? { ...f, intercepted: false, status: 200, contentType: 'application/json', size: '1.1kb', time: '104ms' } : f));
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-orange-500/20 rounded-lg">
            <Radio className="w-6 h-6 text-orange-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">mitmproxy</h1>
            <p className="text-slate-400 text-sm font-mono">listening on 127.0.0.1:8080 · transparent mode · upstream cert: mitmproxy-ca.pem</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <button
            onClick={() => setIntercepting(v => !v)}
            className={`flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium border transition-colors ${
              intercepting
                ? 'bg-red-500/20 border-red-500/40 text-red-300'
                : 'bg-slate-800/50 border-slate-700/50 text-slate-300 hover:text-white'
            }`}
            title="Toggle intercept filter (i)"
          >
            <ShieldAlert className="w-4 h-4" />
            <span>{intercepting ? 'Intercept: ON' : 'Intercept: OFF'}</span>
          </button>
          <button
            onClick={() => setRunning(v => !v)}
            className="flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium bg-slate-800/50 border border-slate-700/50 text-slate-300 hover:text-white"
          >
            {running ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
            <span>{running ? 'Pause' : 'Resume'}</span>
          </button>
          <button
            onClick={() => setFlows([])}
            className="flex items-center space-x-2 px-3 py-2 rounded-lg text-sm font-medium bg-slate-800/50 border border-slate-700/50 text-slate-300 hover:text-white"
          >
            <Trash2 className="w-4 h-4" />
            <span>Clear</span>
          </button>
        </div>
      </div>

      {/* filter bar */}
      <div className="flex items-center space-x-2 mb-4">
        <span className="font-mono text-slate-500 text-sm">filter:</span>
        <input
          value={filter}
          onChange={e => setFilter(e.target.value)}
          placeholder="~m POST, ~d example.com, ~tls, !~d cdn.assets.io"
          className="flex-1 bg-slate-800/50 border border-slate-700/50 rounded-lg px-3 py-2 text-sm font-mono text-slate-200 placeholder-slate-600 focus:outline-none focus:border-orange-500/50"
        />
        <div className="flex items-center space-x-1 text-xs text-slate-500">
          <Circle className={`w-2 h-2 fill-current ${running ? 'text-green-400' : 'text-slate-600'}`} />
          <span>{flows.length} flows</span>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-5 gap-4">
        {/* flow table */}
        <div className="xl:col-span-3 bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-hidden">
          <table className="w-full text-sm font-mono">
            <thead>
              <tr className="border-b border-slate-700/50 text-slate-500 text-xs uppercase">
                <th className="text-left px-3 py-2 font-medium">Method</th>
                <th className="text-left px-3 py-2 font-medium">Host</th>
                <th className="text-left px-3 py-2 font-medium">Path</th>
                <th className="text-right px-3 py-2 font-medium">Status</th>
                <th className="text-right px-3 py-2 font-medium">Size</th>
                <th className="text-right px-3 py-2 font-medium">Time</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(flow => (
                <tr
                  key={flow.id}
                  onClick={() => setSelectedId(flow.id)}
                  className={`border-b border-slate-800 cursor-pointer transition-colors ${
                    selectedId === flow.id ? 'bg-orange-500/10' : 'hover:bg-slate-700/30'
                  } ${flow.intercepted ? 'bg-red-500/5' : ''}`}
                >
                  <td className={`px-3 py-2 font-semibold ${methodColor[flow.method]}`}>{flow.method}</td>
                  <td className="px-3 py-2 text-slate-300">{flow.host}</td>
                  <td className="px-3 py-2 text-slate-400 truncate max-w-[220px]">{flow.path}</td>
                  <td className={`px-3 py-2 text-right ${statusColor(flow.status)}`}>
                    {flow.intercepted ? <span className="text-red-400">held</span> : (flow.status ?? '-')}
                  </td>
                  <td className="px-3 py-2 text-right text-slate-500">{flow.size}</td>
                  <td className="px-3 py-2 text-right text-slate-500">{flow.time}</td>
                </tr>
              ))}
              {filtered.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-3 py-6 text-center text-slate-600">no flows match this filter</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* flow detail */}
        <div className="xl:col-span-2 bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          {selected ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-white font-semibold text-sm">
                  <span className={methodColor[selected.method]}>{selected.method}</span>{' '}
                  {selected.host}{selected.path}
                </h3>
                {selected.intercepted && (
                  <div className="flex items-center space-x-1">
                    <button
                      onClick={() => releaseFlow(selected.id)}
                      className="text-xs px-2 py-1 rounded bg-green-500/20 text-green-300 hover:bg-green-500/30"
                    >
                      Resume
                    </button>
                    <button className="text-xs px-2 py-1 rounded bg-slate-700/50 text-slate-300 hover:bg-slate-700">
                      Drop
                    </button>
                  </div>
                )}
              </div>

              <div>
                <div className="text-xs text-slate-500 uppercase mb-1">Request Headers</div>
                <pre className="text-xs font-mono text-slate-300 bg-slate-900/60 rounded p-2 overflow-x-auto">
{`:authority: ${selected.host}
:method: ${selected.method}
:path: ${selected.path}
user-agent: TargetApp/4.2 (Android 14; Build 3311)
authorization: Bearer eyJhbGciOiJIUzI1NiIs...
accept-encoding: gzip, deflate, br`}
                </pre>
              </div>

              {!selected.intercepted && (
                <div>
                  <div className="text-xs text-slate-500 uppercase mb-1">Response {selected.status}</div>
                  <pre className="text-xs font-mono text-slate-300 bg-slate-900/60 rounded p-2 overflow-x-auto">
{`content-type: ${selected.contentType}
content-length: ${selected.size}

{ "ok": true, "cursor": "eyJvZmZzZXQiOjQyfQ==" }`}
                  </pre>
                </div>
              )}

              {selected.intercepted && (
                <div className="flex items-center space-x-2 text-xs text-red-300 bg-red-500/10 border border-red-500/20 rounded px-2 py-1.5">
                  <ShieldAlert className="w-3.5 h-3.5" />
                  <span>Held by intercept filter — edit body then resume, or drop.</span>
                </div>
              )}

              <button className="w-full flex items-center justify-center space-x-1 text-xs px-2 py-1.5 rounded bg-slate-700/50 text-slate-300 hover:bg-slate-700">
                <RotateCcw className="w-3.5 h-3.5" />
                <span>Replay flow</span>
              </button>
            </div>
          ) : (
            <div className="text-slate-600 text-sm text-center py-12">select a flow to inspect</div>
          )}
        </div>
      </div>
    </div>
  );
};
