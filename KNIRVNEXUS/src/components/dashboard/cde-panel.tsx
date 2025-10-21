'use client';

import React, { useState } from 'react';
import { X, Terminal, Shield, Monitor, Network, Settings } from 'lucide-react';
import ConsolePanel from './console-panel';
import PolicyEditor from './policy-editor';
import MonitorPanel from './monitor-panel';
import ConnectionsPanel from './connections-panel';

interface CDEPanelProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName?: string;
  nodeId?: string;
}

const CDEPanel: React.FC<CDEPanelProps> = ({ isOpen, onClose, nodeName, nodeId }) => {
  const [showConsole, setShowConsole] = useState(false);
  const [showPolicy, setShowPolicy] = useState(false);
  const [showMonitor, setShowMonitor] = useState(false);
  const [showConnections, setShowConnections] = useState(false);
  const [activeView, setActiveView] = useState<'tools' | 'content'>('tools');

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm">
      {/* Main CDE Container */}
      <div className="absolute inset-0 flex flex-col bg-gradient-to-br from-slate-900 via-blue-950 to-slate-950 border border-blue-600/50">
        {/* Header */}
        <div className="border-b border-blue-600/30 bg-gradient-to-r from-slate-900 to-blue-950 p-4 flex items-center justify-between">
          <div>
            <h1 className="text-lg font-bold text-blue-300">Cloud Development Environment</h1>
            {nodeName && <p className="text-xs text-slate-400 mt-1">Node: {nodeName} ({nodeId})</p>}
          </div>
          <button
            onClick={onClose}
            className="bg-slate-700 hover:bg-slate-600 p-2 rounded-lg transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Main Content Area */}
        <div className="flex-1 overflow-hidden relative">
          {/* Connections Sidebar */}
          <div
            className={`absolute left-0 top-0 bottom-0 z-20 transition-all duration-300 ${
              showConnections ? 'translate-x-0' : '-translate-x-full'
            }`}
          >
            <ConnectionsPanel isOpen={showConnections} onClose={() => setShowConnections(false)} />
          </div>

          {/* Main Content Area with Slide-outs */}
          <div
            className="h-full overflow-auto transition-all duration-300 p-4"
            style={{
              marginLeft: showConnections ? '280px' : '0',
              marginBottom: showMonitor ? 'calc(33vh + 1rem)' : '0',
            }}
          >
            {/* Tools View */}
            {activeView === 'tools' && (
              <div className="space-y-4">
                <div className="bg-gradient-to-r from-slate-800 to-blue-950 rounded-lg p-6 border border-blue-600/30 shadow-lg">
                  <h2 className="text-lg font-bold text-blue-300 mb-6">Tools</h2>
                  <div className="grid grid-cols-2 gap-4">
                    {/* Console Button */}
                    <button
                      onClick={() => setShowConsole(!showConsole)}
                      className={`flex items-center justify-start p-4 rounded-lg font-semibold transition-all border-2 ${
                        showConsole
                          ? 'bg-red-900 border-red-600 text-red-100 shadow-lg shadow-red-500/30'
                          : 'bg-slate-800 border-slate-700 text-slate-300 hover:bg-slate-700 hover:border-blue-600'
                      }`}
                    >
                      <Terminal className="w-5 h-5 mr-3" />
                      Console
                    </button>

                    {/* Policy Button */}
                    <button
                      onClick={() => setShowPolicy(!showPolicy)}
                      className={`flex items-center justify-start p-4 rounded-lg font-semibold transition-all border-2 ${
                        showPolicy
                          ? 'bg-amber-900 border-amber-600 text-amber-100 shadow-lg shadow-amber-500/30'
                          : 'bg-slate-800 border-slate-700 text-slate-300 hover:bg-slate-700 hover:border-blue-600'
                      }`}
                    >
                      <Shield className="w-5 h-5 mr-3" />
                      Policy
                    </button>

                    {/* Monitor Button */}
                    <button
                      onClick={() => setShowMonitor(!showMonitor)}
                      className={`flex items-center justify-start p-4 rounded-lg font-semibold transition-all border-2 ${
                        showMonitor
                          ? 'bg-green-900 border-green-600 text-green-100 shadow-lg shadow-green-500/30'
                          : 'bg-slate-800 border-slate-700 text-slate-300 hover:bg-slate-700 hover:border-blue-600'
                      }`}
                    >
                      <Monitor className="w-5 h-5 mr-3" />
                      Monitor
                    </button>

                    {/* Connections Button */}
                    <button
                      onClick={() => setShowConnections(!showConnections)}
                      className={`flex items-center justify-start p-4 rounded-lg font-semibold transition-all border-2 ${
                        showConnections
                          ? 'bg-purple-900 border-purple-600 text-purple-100 shadow-lg shadow-purple-500/30'
                          : 'bg-slate-800 border-slate-700 text-slate-300 hover:bg-slate-700 hover:border-blue-600'
                      }`}
                    >
                      <Network className="w-5 h-5 mr-3" />
                      Connections
                    </button>
                  </div>
                </div>

                {/* Status Info */}
                <div className="bg-gradient-to-r from-slate-800 to-blue-950 rounded-lg p-4 border border-blue-600/30">
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <p className="text-slate-400">Status</p>
                      <p className="text-green-400 font-semibold">● Online</p>
                    </div>
                    <div>
                      <p className="text-slate-400">Active Panels</p>
                      <p className="text-blue-300 font-semibold">{[showConsole, showPolicy, showMonitor, showConnections].filter(Boolean).length}</p>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Console Panel - Top Left */}
          <div className="absolute left-0 top-0 z-30">
            <ConsolePanel isOpen={showConsole} onClose={() => setShowConsole(false)} />
          </div>

          {/* Policy Editor - Middle Left */}
          <div className="absolute left-0 z-30">
            <PolicyEditor isOpen={showPolicy} onClose={() => setShowPolicy(false)} />
          </div>

          {/* Monitor Panel - Bottom */}
          <div className="absolute bottom-0 left-0 right-0 z-30">
            <MonitorPanel isOpen={showMonitor} onClose={() => setShowMonitor(false)} />
          </div>
        </div>
      </div>
    </div>
  );
};

export default CDEPanel;