// Which model does what, right now (plan §7). Read-only: it shows the server's status
// document (daemon models, llama and its skill adapters, recent routing) and changes nothing.
import React, { useCallback, useEffect, useState } from 'react';
import { authToken } from '../services/serverLLM';
import { fetchModelStatus, roles, type PlatformStatus, type RoleState } from '../services/modelStatus';

const REFRESH_MS = 10_000;

const STATE_STYLE: Record<RoleState, string> = {
  ready: 'text-green-400',
  idle: 'text-gray-300',
  'not ready': 'text-yellow-400',
  unavailable: 'text-red-400',
};

const errorText = (err: unknown) => (err instanceof Error ? err.message : String(err));

export const ModelStatusPanel: React.FC = () => {
  const [status, setStatus] = useState<PlatformStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      setStatus(await fetchModelStatus(authToken(), 8));
      setError(null);
    } catch (err) {
      setError(errorText(err));
    }
  }, []);

  useEffect(() => {
    void refresh();
    const id = setInterval(() => { void refresh(); }, REFRESH_MS);
    return () => clearInterval(id);
  }, [refresh]);

  if (error && !status) {
    return <div className="text-sm text-red-300" role="alert" data-testid="model-status-error">{error}</div>;
  }
  if (!status) return <div className="text-sm text-gray-400">Reading model status…</div>;

  return (
    <div className="space-y-4" data-testid="model-status">
      {error && <div className="text-xs text-yellow-300" role="alert">Showing the last reading: {error}</div>}
      <div className="space-y-2">
        {roles(status).map((r) => (
          <div key={r.role} className="p-3 bg-gray-800/50 rounded-lg border border-gray-700/50" data-testid={`role-${r.role}`}>
            <div className="flex items-center justify-between">
              <span className="text-white font-medium">{r.label}</span>
              <span className={`text-sm font-medium ${STATE_STYLE[r.state]}`}>{r.state}</span>
            </div>
            <div className="text-xs text-gray-400 mt-1">
              {r.model}{r.version ? ` · ${r.version}` : ''}{r.detail ? ` · ${r.detail}` : ''}
            </div>
          </div>
        ))}
      </div>
      <div>
        <div className="text-sm text-gray-400 mb-2">Recent routing</div>
        {status.routing.length === 0 && <div className="text-xs text-gray-500">No routing decisions since the server started.</div>}
        <ul className="space-y-1 text-xs text-gray-300">
          {status.routing.map((ev) => (
            <li key={ev.id} data-testid="routing-event">
              <span className="text-gray-500">{new Date(ev.time).toLocaleTimeString()}</span>{' '}
              {ev.kind} {ev.error ? <span className="text-red-300">failed: {ev.error}</span> : <>→ {ev.chosen}{ev.adapter ? ` +${ev.adapter}` : ''}</>}
              {ev.fallbacks && ev.fallbacks.length > 0 && <span className="text-yellow-300"> after failing: {ev.fallbacks.join(', ')}</span>}
              {ev.degraded && <span className="text-red-300"> DEGRADED: skill did not run</span>}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
};
