import React from 'react';
import { Waves, Fish } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthContext';
import { useSandboxSession } from '../../hooks/useSandboxSession';
import Wireshark from './packetcapture/Wireshark';
import Zeek from './packetcapture/Zeek';

export const PacketCapture: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();
  const { session } = useSandboxSession();

  const isSubRoute = location.pathname !== '/packet-capture';

  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/wireshark" element={<Wireshark />} />
        <Route path="/zeek" element={<Zeek />} />
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
        <div className="p-2 bg-sky-500/20 rounded-lg">
          <Waves className="w-6 h-6 text-sky-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Packet Capture</h1>
          <p className="text-slate-400">Protocol dissection and network security monitoring</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {canAccessSubPage('packet-capture', 'wireshark') && (
          <button
            onClick={() => navigate('/packet-capture/wireshark')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-sky-500/20 rounded-lg group-hover:bg-sky-500/30 transition-colors">
                <Waves className="w-6 h-6 text-sky-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Wireshark (TShark)</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              Live packet list with display filters, protocol columns, and follow-stream.
            </p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-green-500 rounded-full"></div>
              <span className="text-slate-300">capturing on eth0</span>
            </div>
          </button>
        )}

        {canAccessSubPage('packet-capture', 'zeek') && (
          <button
            onClick={() => navigate('/packet-capture/zeek')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-teal-500/20 rounded-lg group-hover:bg-teal-500/30 transition-colors">
                <Fish className="w-6 h-6 text-teal-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Zeek</h3>
            </div>
            <p className="text-slate-400 mb-4 font-mono text-sm">
              Protocol-aware logging — conn.log, dns.log, ssl.log, and notice.log from live traffic.
            </p>
            <div className="flex items-center space-x-1 text-sm">
              <div className="w-2 h-2 bg-teal-500 rounded-full"></div>
              <span className="text-slate-300">4 logs streaming</span>
            </div>
          </button>
        )}
      </div>
    </div>
  );
};
