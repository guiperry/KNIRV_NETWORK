import React from 'react';
import { ScanSearch, GitBranch, KeySquare } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import Semgrep from './staticanalysis/Semgrep';
import TreeSitter from './staticanalysis/TreeSitter';
import TruffleHog from './staticanalysis/TruffleHog';

export const StaticAnalysis: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  const isSubRoute = location.pathname !== '/static-analysis';

  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/semgrep" element={<Semgrep />} />
        <Route path="/tree-sitter" element={<TreeSitter />} />
        <Route path="/trufflehog" element={<TruffleHog />} />
      </Routes>
    );
  }

  const cards = [
    { id: 'semgrep', name: 'Semgrep', icon: ScanSearch, color: 'emerald', desc: 'Pattern-based static analysis — rule packs for OWASP, secrets, and custom AST patterns.', tag: '18 findings' },
    { id: 'tree-sitter', name: 'Tree-sitter', icon: GitBranch, color: 'lime', desc: 'Incremental parsing — inspect a live syntax tree and run S-expression queries.', tag: 'parser: javascript' },
    { id: 'trufflehog', name: 'TruffleHog', icon: KeySquare, color: 'amber', desc: 'Verified secret scanning across filesystems, git history, and live repos.', tag: '3 verified' },
  ] as const;

  const colorClasses: Record<string, string> = {
    emerald: 'bg-emerald-500/20 text-emerald-400 group-hover:bg-emerald-500/30',
    lime: 'bg-lime-500/20 text-lime-400 group-hover:bg-lime-500/30',
    amber: 'bg-amber-500/20 text-amber-400 group-hover:bg-amber-500/30',
  };

  return (
    <div className="h-full bg-slate-900 p-6">
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
            <p className="text-slate-400 mb-4 font-mono text-sm">{c.desc}</p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-emerald-500 rounded-full"></div>
              <span className="text-slate-300">{c.tag}</span>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
};
