import React from 'react';
import { Cpu, Waypoints, Activity } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import Frida from './instrumentation/Frida';
import ProxychainsNg from './instrumentation/ProxychainsNg';
import Bpftrace from './instrumentation/Bpftrace';

export const Instrumentation: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  const isSubRoute = location.pathname !== '/instrumentation';

  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/frida" element={<Frida />} />
        <Route path="/proxychains-ng" element={<ProxychainsNg />} />
        <Route path="/bpftrace" element={<Bpftrace />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-cyan-500/20 rounded-lg">
          <Cpu className="w-6 h-6 text-cyan-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Instrumentation</h1>
          <p className="text-slate-400">Runtime dynamic instrumentation, socket redirection, and kernel tracing</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {canAccessSubPage('instrumentation', 'frida') && (
          <button
            onClick={() => navigate('/instrumentation/frida')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-cyan-500/20 rounded-lg group-hover:bg-cyan-500/30 transition-colors">
                <Cpu className="w-6 h-6 text-cyan-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Frida</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              frida-core · attach to a PID, inject a JS agent, hook exports and Java/ObjC methods live.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">2 attached</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('instrumentation', 'proxychains-ng') && (
          <button
            onClick={() => navigate('/instrumentation/proxychains-ng')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                <Waypoints className="w-6 h-6 text-blue-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">proxychains-ng</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              LD_PRELOAD socket redirection — force a target's TCP calls through a SOCKS/HTTP chain.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span className="text-slate-300">dynamic_chain</span>
              </div>
            </div>
          </button>
        )}

        {canAccessSubPage('instrumentation', 'bpftrace') && (
          <button
            onClick={() => navigate('/instrumentation/bpftrace')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-purple-500/20 rounded-lg group-hover:bg-purple-500/30 transition-colors">
                <Activity className="w-6 h-6 text-purple-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">bpftrace</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              eBPF one-liners and scripts — trace syscalls, uprobes, and kernel events with near-zero overhead.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
                <span className="text-slate-300">idle</span>
              </div>
            </div>
          </button>
        )}
      </div>
    </div>
  );
};
