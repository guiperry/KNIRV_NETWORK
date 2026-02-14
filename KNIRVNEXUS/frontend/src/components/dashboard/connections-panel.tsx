'use client';

import React, { useState } from 'react';
import { Network, X, Search, Activity, Globe, ShieldAlert } from 'lucide-react';

interface NRV {
  id: string;
  title: string;
  domain: string;
  severity: 'high' | 'medium' | 'low';
  timestamp: string;
  sourceNode?: string;
}

interface ConnectionsPanelProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectNRV?: (nrv: NRV) => void;
}

const ConnectionsPanel: React.FC<ConnectionsPanelProps> = ({ isOpen, onClose, onSelectNRV }) => {
  const [selectedNRV, setSelectedNRV] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');

  const mockNRVs: NRV[] = [
    { id: 'NRV-001', title: 'Fabric Reasoning Hallucination', domain: 'Medical', severity: 'high', timestamp: '2025-10-21T10:30:00Z', sourceNode: 'NODE-AX-1' },
    { id: 'NRV-002', title: 'Kernel Privilege Escalation', domain: 'Cybersec', severity: 'high', timestamp: '2025-10-21T09:15:00Z', sourceNode: 'NODE-BK-4' },
    { id: 'NRV-003', title: 'Data Sovereignty Violation', domain: 'Compliance', severity: 'medium', timestamp: '2025-10-21T08:45:00Z', sourceNode: 'NODE-CX-2' },
    { id: 'NRV-004', title: 'Neural Weights Divergence', domain: 'Validation', severity: 'low', timestamp: '2025-10-21T07:20:00Z', sourceNode: 'NODE-AX-1' },
  ];

  const filteredNRVs = mockNRVs.filter(nrv =>
    nrv.id.toLowerCase().includes(searchTerm.toLowerCase()) ||
    nrv.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
    nrv.domain.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (!isOpen) return null;

  return (
    <div
      className="absolute left-0 top-0 h-full z-[60] transition-all duration-500 transform ease-in-out bg-slate-950 border-r border-blue-600/50 shadow-[10px_0_40px_rgba(0,0,0,0.5)] overflow-hidden"
      style={{
        width: '300px',
        paddingTop: '1rem', // adjusted for relative positioning
      }}
    >
      <div className="h-full flex flex-col p-4 overflow-hidden">
        <div className="flex items-center justify-between mb-6 border-b border-blue-600/30 pb-4">
          <div className="flex items-center space-x-2">
            <Globe className="w-5 h-5 text-blue-400 animate-spin-slow" />
            <h2 className="text-sm font-bold uppercase tracking-widest text-blue-300">
              Network Fabric
            </h2>
          </div>
          <button
            onClick={onClose}
            className="text-slate-500 hover:text-white hover:bg-slate-800 p-1 rounded transition-all"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="mb-6">
          <div className="relative group">
            <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-500 group-focus-within:text-blue-400 transition-colors" />
            <input
              type="text"
              placeholder="Filter by ID, Domain..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-slate-900 border border-slate-700 group-hover:border-blue-600/50 rounded-full pl-10 pr-4 py-2 text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 transition-all"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto pr-2 custom-scrollbar space-y-3">
          <div className="text-[10px] font-bold text-slate-500 uppercase mb-2 px-1">Active Failure Contexts</div>
          
          {filteredNRVs.map(nrv => (
            <div
              key={nrv.id}
              onClick={() => {
                setSelectedNRV(nrv.id);
                onSelectNRV?.(nrv);
              }}
              className={`group relative p-3 rounded-lg cursor-pointer transition-all duration-300 border-l-4 ${
                selectedNRV === nrv.id
                  ? 'bg-blue-600/20 border-blue-500 shadow-[0_0_15px_rgba(59,130,246,0.2)]'
                  : 'bg-slate-900/50 hover:bg-slate-800 border-slate-800 hover:border-slate-700'
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <span className="font-mono text-[10px] text-blue-400/80 font-bold">{nrv.id}</span>
                <div className={`flex items-center space-x-1 px-1.5 py-0.5 rounded text-[9px] font-black uppercase ${
                  nrv.severity === 'high' ? 'bg-red-500/20 text-red-400 border border-red-500/30' :
                  nrv.severity === 'medium' ? 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/30' :
                  'bg-green-500/20 text-green-400 border border-green-500/30'
                }`}>
                  <ShieldAlert className="w-2.5 h-2.5" />
                  <span>{nrv.severity}</span>
                </div>
              </div>
              <div className="text-xs font-bold text-slate-100 group-hover:text-white transition-colors mb-1 line-clamp-1">{nrv.title}</div>
              <div className="flex items-center justify-between text-[10px]">
                <span className="text-slate-500 italic">{nrv.domain}</span>
                <span className="text-slate-600 font-mono">{nrv.sourceNode}</span>
              </div>
              
              {selectedNRV === nrv.id && (
                <div className="absolute -right-1 top-1/2 -translate-y-1/2 w-1 h-8 bg-blue-500 rounded-full animate-pulse" />
              )}
            </div>
          ))}
          
          {filteredNRVs.length === 0 && (
            <div className="text-center py-10">
              <Activity className="w-8 h-8 text-slate-800 mx-auto mb-2" />
              <div className="text-xs text-slate-600">No active contexts matching filters</div>
            </div>
          )}
        </div>
        
        <div className="mt-4 pt-4 border-t border-blue-600/20">
          <div className="flex items-center justify-between text-[10px] text-slate-500 px-1">
            <span>Global Consensus:</span>
            <span className="text-green-500 font-bold">99.98% SYNC</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ConnectionsPanel;
