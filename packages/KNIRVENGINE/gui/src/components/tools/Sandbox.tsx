import React, { lazy, Suspense } from 'react';
import { Box, MonitorPlay } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import Bubblewrap from './sandbox/Bubblewrap';

// The RFB client is sizeable and is only needed for the dedicated noVNC page.
const NoVnc = lazy(() => import('./sandbox/NoVnc'));

export const Sandbox: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  const isSubRoute = location.pathname !== '/sandbox';

  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/bubblewrap" element={<Bubblewrap />} />
        <Route path="/novnc" element={<Suspense fallback={<div className="h-full bg-slate-900" />}><NoVnc /></Suspense>} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-fuchsia-500/20 rounded-lg">
          <Box className="w-6 h-6 text-fuchsia-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Sandbox</h1>
          <p className="text-slate-400">Isolated execution namespaces and remote GUI streaming</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {canAccessSubPage('sandbox', 'bubblewrap') && (
          <button
            onClick={() => navigate('/sandbox/bubblewrap')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-fuchsia-500/20 rounded-lg group-hover:bg-fuchsia-500/30 transition-colors">
                <Box className="w-6 h-6 text-fuchsia-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Bubblewrap</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              Unprivileged user-namespace sandbox for the target binary — bind mounts, netns, and Xvfb.
            </p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-green-500 rounded-full"></div>
              <span className="text-slate-300">1 namespace active</span>
            </div>
          </button>
        )}

        {canAccessSubPage('sandbox', 'novnc') && (
          <button
            onClick={() => navigate('/sandbox/novnc')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-pink-500/20 rounded-lg group-hover:bg-pink-500/30 transition-colors">
                <MonitorPlay className="w-6 h-6 text-pink-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">noVNC</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              HTML5 RFB client — watch and interact with the sandboxed display over a WebSocket bridge.
            </p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-pink-500 rounded-full"></div>
              <span className="text-slate-300">:99 connected</span>
            </div>
          </button>
        )}
      </div>
    </div>
  );
};
