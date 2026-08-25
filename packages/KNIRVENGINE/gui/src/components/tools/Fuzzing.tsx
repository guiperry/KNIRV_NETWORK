import React from 'react';
import { Bug, Boxes } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import { useSandboxSession } from '../../hooks/useSandboxSession';
import LibAFL from './fuzzing/LibAFL';
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
        <Route path="/libafl" element={<LibAFL />} />
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
        {canAccessSubPage('fuzzing', 'libafl') && (
          <button
            onClick={() => navigate('/fuzzing/libafl')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-yellow-500/20 rounded-lg group-hover:bg-yellow-500/30 transition-colors">
                <Boxes className="w-6 h-6 text-yellow-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">LibAFL</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              Rust fuzzing framework — compose custom executors, mutators, and corpus stages.
            </p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-green-500 rounded-full"></div>
              <span className="text-slate-300">2 campaigns running</span>
            </div>
          </button>
        )}

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
            <p className="text-slate-400 mb-4 font-mono text-sm">
              afl-fuzz — persistent-mode, QEMU/Frida instrumentation, CmpLog, and multi-core campaigns.
            </p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-orange-500 rounded-full"></div>
              <span className="text-slate-300">1 campaign running</span>
            </div>
          </button>
        )}
      </div>
    </div>
  );
};
