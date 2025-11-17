'use client';

import React, { useState } from 'react';
import { Network, X, Search } from 'lucide-react';

interface NRV {
  id: string;
  title: string;
  domain: string;
  severity: 'high' | 'medium' | 'low';
  timestamp: string;
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
    { id: 'NRV-001', title: 'Hallucination in Medical Diagnosis', domain: 'Healthcare', severity: 'high', timestamp: '2025-10-21T10:30:00Z' },
    { id: 'NRV-002', title: 'Code Generation Security Flaw', domain: 'Security', severity: 'high', timestamp: '2025-10-21T09:15:00Z' },
    { id: 'NRV-003', title: 'Math Calculation Error', domain: 'Finance', severity: 'medium', timestamp: '2025-10-21T08:45:00Z' },
  ];

  const filteredNRVs = mockNRVs.filter(nrv =>
    nrv.id.toLowerCase().includes(searchTerm.toLowerCase()) ||
    nrv.title.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (!isOpen) return null;

  return (
    <div
      className="fixed left-0 top-0 h-screen z-20 transition-all duration-300 bg-gradient-to-r from-blue-950 to-slate-900 border-r-2 border-blue-600 shadow-lg overflow-hidden"
      style={{
        width: '280px',
        paddingTop: '1.5rem',
      }}
    >
      <div className="h-full flex flex-col p-4 overflow-hidden">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-bold flex items-center text-blue-300 flex-1">
            <Network className="w-4 h-4 mr-2" />
            Connections
          </h2>
          <button
            onClick={onClose}
            className="bg-slate-700 hover:bg-slate-600 p-1 rounded transition-colors flex-shrink-0"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="mb-3">
          <div className="relative">
            <Search className="w-3 h-3 absolute left-2 top-2.5 text-slate-500" />
            <input
              type="text"
              placeholder="Search..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded pl-7 pr-2 py-1.5 text-xs focus:outline-none focus:border-blue-500"
            />
          </div>
        </div>

        <div className="space-y-2 overflow-y-auto flex-1">
          {filteredNRVs.map(nrv => (
            <div
              key={nrv.id}
              onClick={() => {
                setSelectedNRV(nrv.id);
                onSelectNRV?.(nrv);
              }}
              className={`p-2.5 rounded cursor-pointer transition-all text-xs border ${
                selectedNRV === nrv.id
                  ? 'bg-blue-900 border-blue-500 shadow-md'
                  : 'bg-slate-800 hover:bg-slate-700 border-slate-700'
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-mono text-blue-300 text-xs">{nrv.id}</span>
                <span className={`text-xs px-1.5 py-0.5 rounded ${
                  nrv.severity === 'high' ? 'bg-red-900 text-red-200' :
                  nrv.severity === 'medium' ? 'bg-yellow-900 text-yellow-200' :
                  'bg-green-900 text-green-200'
                }`}>
                  {nrv.severity}
                </span>
              </div>
              <div className="text-xs font-semibold text-gray-100 mb-0.5 line-clamp-1">{nrv.title}</div>
              <div className="text-xs text-slate-400">{nrv.domain}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default ConnectionsPanel;