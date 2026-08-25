import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Lock, Loader2, AlertTriangle } from 'lucide-react';
import { useSandbox } from './SandboxContext';

interface RequireSandboxProps {
  children: React.ReactNode;
}

const statusMessage: Record<string, string> = {
  idle: 'No sandbox running',
  provisioning: 'Sandbox starting…',
  stopping: 'Sandbox stopping…',
  stopped: 'Sandbox stopped',
  error: 'Sandbox failed to start',
};

export const RequireSandbox: React.FC<RequireSandboxProps> = ({ children }) => {
  const navigate = useNavigate();
  const { isReady, status, error, targetLabel } = useSandbox();

  if (isReady) {
    return <>{children}</>;
  }

  const message = status ? statusMessage[status] ?? 'No sandbox running' : 'No sandbox running';

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="w-full max-w-md rounded-2xl border border-slate-700/50 bg-slate-800/50 p-8 text-center">
          <div className="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-2xl border border-amber-400/30 bg-amber-500/10">
            <Lock className="h-8 w-8 text-amber-300" />
          </div>
          <h2 className="text-xl font-semibold text-white">Sandbox required</h2>
          <p className="mt-2 text-sm text-slate-400">{message}</p>
          {status === 'provisioning' && (
            <div className="mt-4 flex items-center justify-center gap-2 text-xs text-slate-500">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>provisioning namespace…</span>
            </div>
          )}
          {status === 'error' && (
            <div className="mt-4 flex items-center justify-center gap-2 text-xs text-red-300">
              <AlertTriangle className="h-4 w-4" />
              <span>{error || 'unknown error'}</span>
            </div>
          )}
          {targetLabel && (
            <p className="mt-3 text-xs text-slate-500">
              target: <span className="font-mono text-slate-400">{targetLabel}</span>
            </p>
          )}
          <button
            type="button"
            onClick={() => navigate('/sandbox/bubblewrap')}
            className="mt-6 inline-flex items-center gap-2 rounded-lg bg-fuchsia-500 px-4 py-2.5 text-sm font-medium text-white transition hover:bg-fuchsia-400 focus:outline-none focus:ring-2 focus:ring-fuchsia-300 focus:ring-offset-2 focus:ring-offset-slate-950"
          >
            Launch sandbox
          </button>
        </div>
      </div>
    </div>
  );
};

export default RequireSandbox;
