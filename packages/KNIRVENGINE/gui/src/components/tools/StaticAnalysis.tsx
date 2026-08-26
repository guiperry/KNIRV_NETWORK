import React from 'react';
import { ScanSearch, GitBranch } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import { useSandboxSession } from '../../hooks/useSandboxSession';
import Semgrep from './staticanalysis/Semgrep';
import TreeSitter from './staticanalysis/TreeSitter';

export const StaticAnalysis: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();
  const { session } = useSandboxSession();

  const isSubRoute = location.pathname !== '/static-analysis';

  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/semgrep" element={<Semgrep />} />
        <Route path="/tree-sitter" element={<TreeSitter />} />
      </Routes>
    );
  }

  const cards = [
    { id: 'semgrep', name: 'Semgrep', icon: ScanSearch, color: 'emerald' },
    { id: 'tree-sitter', name: 'Tree-sitter', icon: GitBranch, color: 'lime' },
  ] as const;

  const colorClasses: Record<string, string> = {
    emerald: 'bg-emerald-500/20 text-emerald-400 group-hover:bg-emerald-500/30',
    lime: 'bg-lime-500/20 text-lime-400 group-hover:bg-lime-500/30',
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      {session?.targetLabel && (
        <div className="mb-4 inline-flex items-center gap-2 rounded-lg border border-slate-700/50 bg-slate-800/40 px-3 py-1.5 text-xs font-mono text-slate-400">
          <span className="text-slate-500">sandbox target</span>
          <span className="text-slate-200">{session.targetLabel}</span>
        </div>
      )}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-emerald-500/20 rounded-lg">
          <ScanSearch className="w-6 h-6 text-emerald-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Static Analysis</h1>
          <p className="text-slate-400">AST pattern matching, syntax tree inspection, and secret scanning</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {cards.map(c => canAccessSubPage('static-analysis', c.id) && (
          <button
            key={c.id}
            onClick={() => navigate(`/static-analysis/${c.id}`)}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className={`p-2 rounded-lg transition-colors ${colorClasses[c.color]}`}>
                <c.icon className="w-6 h-6" />
              </div>
               <h3 className="text-lg font-semibold text-white">{c.name}</h3>
            </div>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-emerald-500 rounded-full"></div>
              <span className="text-slate-300">available</span>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
};
