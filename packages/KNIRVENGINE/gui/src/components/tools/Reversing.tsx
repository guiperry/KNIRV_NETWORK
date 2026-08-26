import React from 'react';
import { Binary, Compass, Layers3, FileCode2 } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import { useSandboxSession } from '../../hooks/useSandboxSession';
import Cutter from './reversing/Cutter';
import ILSpy from './reversing/ILSpy';
import Jadx from './reversing/Jadx';

export const Reversing: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();
  const { session } = useSandboxSession();

  const isSubRoute = location.pathname !== '/reversing';

  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/cutter" element={<Cutter />} />
        <Route path="/ilspy" element={<ILSpy />} />
        <Route path="/jadx" element={<Jadx />} />
      </Routes>
    );
  }

  const cards = [
    { id: 'cutter', name: 'Cutter', icon: Compass, color: 'red' },
    { id: 'ilspy', name: 'ILSpy', icon: Layers3, color: 'blue' },
    { id: 'jadx', name: 'JADX', icon: FileCode2, color: 'orange' },
  ] as const;

  const colorClasses: Record<string, string> = {
    green: 'bg-green-500/20 text-green-400 group-hover:bg-green-500/30',
    red: 'bg-red-500/20 text-red-400 group-hover:bg-red-500/30',
    blue: 'bg-blue-500/20 text-blue-400 group-hover:bg-blue-500/30',
    orange: 'bg-orange-500/20 text-orange-400 group-hover:bg-orange-500/30',
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
        <div className="p-2 bg-green-500/20 rounded-lg">
          <Binary className="w-6 h-6 text-green-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Reversing</h1>
          <p className="text-slate-400">Disassembly and decompilation across native, .NET, and Android targets</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {cards.map(c => canAccessSubPage('reversing', c.id) && (
          <button
            key={c.id}
            onClick={() => navigate(`/reversing/${c.id}`)}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className={`p-2 rounded-lg transition-colors ${colorClasses[c.color]}`}>
                <c.icon className="w-6 h-6" />
              </div>
               <h3 className="text-lg font-semibold text-white">{c.name}</h3>
            </div>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-green-500 rounded-full"></div>
              <span className="text-slate-300">available</span>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
};
