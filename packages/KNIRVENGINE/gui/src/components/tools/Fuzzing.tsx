import React from 'react';
import { Bug, Zap } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import { useSandboxSession } from '../../hooks/useSandboxSession';
import AflPlusPlus from './fuzzing/AflPlusPlus';

export const Fuzzing: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();
  const { session } = useSandboxSession();

  const isSubRoute = location.pathname !== '/fuzzing';

  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/aflplusplus" element={<AflPlusPlus />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {session?.targetLabel && (
        <div className="mb-4 inline-flex items-center gap-2 rounded-lg border border-slate-700/50 bg-slate-800/40 px-3 py-1.5 text-xs font-mono text-slate-400">
          <span className="text-slate-500">sandbox target</span>
          <span className="text-slate-200">{session.targetLabel}</span>
        </div>
      )}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-yellow-500/20 rounded-lg">
          <Bug className="w-6 h-6 text-yellow-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Fuzzing</h1>
          <p className="text-slate-400">Coverage-guided fuzzing harnesses and campaign monitoring</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {canAccessSubPage('fuzzing', 'aflplusplus') && (
          <button
            onClick={() => navigate('/fuzzing/aflplusplus')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-orange-500/20 rounded-lg group-hover:bg-orange-500/30 transition-colors">
                <Bug className="w-6 h-6 text-orange-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">AFL++</h3>
            </div>
            <div className="flex items-center space-x-1 text-sm">
              <Zap className="w-4 h-4 text-orange-400" />
              <span className="text-slate-300">launch harness</span>
            </div>
          </button>
        )}
      </div>
    </div>
  );
};
