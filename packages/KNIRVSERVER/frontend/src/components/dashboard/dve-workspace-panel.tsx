'use client';

import React, { useState } from 'react';
import { X, Terminal, Shield, Monitor, Network, Settings, Info, Box, Activity, Zap, Globe } from 'lucide-react';
import ConsolePanel from './console-panel';
import PolicyEditor from './policy-editor';
import MonitorPanel, { type DVETask } from './monitor-panel';
import ConnectionsPanel, { type ActiveWorker } from './connections-panel';
import MetadataPanel from './metadata-panel';
import DVESolverPanel from './dve-solver-panel';
import ViewportPanel from './viewport-panel';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import type { DVENode } from '@/types/api';

interface DVEWorkspacePanelProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName?: string;
  nodeId?: string;
  isModular?: boolean;
  onToggleMode?: () => void;
  node?: DVENode;
}

const DVEWorkspacePanel: React.FC<DVEWorkspacePanelProps> = ({ 
  isOpen, 
  onClose, 
  nodeName, 
  nodeId,
  isModular,
  onToggleMode,
  node
}) => {
  const [showConsole, setShowConsole] = useState(false);
  const [showPolicy, setShowPolicy] = useState(false);
  const [showMonitor, setShowMonitor] = useState(false);
  const [showConnections, setShowConnections] = useState(false);
  const [showMetadata, setShowMetadata] = useState(false);
  const [showSolver, setShowSolver] = useState(false);
  const [showViewport, setShowViewport] = useState(false);
  const [selectedWorker, setSelectedWorker] = useState<ActiveWorker | null>(null);
  const [selectedTask, setSelectedTask] = useState<DVETask | null>(null);

  if (!isOpen) return null;

  const actualNodeId = node?.id || nodeId;
  const actualNodeName = node?.name || nodeName;

  return (
    <div className="fixed inset-0 z-50 bg-black/70 transition-all duration-300">
      {/* Main DVE Workspace Container */}
      <div className="absolute inset-0 flex flex-col bg-[#03050a] border-l border-blue-600/50">
        
        {/* Unified Header */}
        <div className="h-16 border-b border-blue-600/30 bg-slate-900/80 backdrop-blur-sm p-4 flex items-center justify-between z-50">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <Box className="w-6 h-6 text-blue-500 animate-pulse" />
              <div>
                <h1 className="text-sm font-black tracking-tighter text-blue-100 uppercase">
                  Sovereign DVE Workspace
                </h1>
                <p className="text-[10px] font-mono text-slate-500">
                  Secure Enclave Context: {actualNodeName || 'UNSET'} ({actualNodeId || 'unknown'})
                </p>
              </div>
            </div>
            <div className="h-8 w-px bg-slate-800" />
            <div className="flex space-x-2">
              <Badge variant="outline" className="bg-green-500/10 text-green-400 border-green-500/20 text-[9px]">
                TEE: VERIFIED
              </Badge>
              <Badge variant="outline" className="bg-blue-500/10 text-blue-400 border-blue-500/20 text-[9px]">
                FABRIC: SYNCED
              </Badge>
            </div>
          </div>

          <div className="flex items-center space-x-3">
            <div className="flex bg-slate-950/50 p-1 rounded-lg border border-slate-800">
              <button
                onClick={() => setShowConnections(!showConnections)}
                className={`p-2 rounded-md transition-all ${showConnections ? 'bg-blue-600 text-white shadow-lg' : 'text-slate-400 hover:text-blue-300'}`}
                title="Toggle Connections"
              >
                <Network className="w-4 h-4" />
              </button>
              <button
                onClick={() => setShowConsole(!showConsole)}
                className={`p-2 rounded-md transition-all ${showConsole ? 'bg-blue-600 text-white shadow-lg' : 'text-slate-400 hover:text-blue-300'}`}
                title="Toggle Console"
              >
                <Terminal className="w-4 h-4" />
              </button>
              <button
                onClick={() => setShowPolicy(!showPolicy)}
                className={`p-2 rounded-md transition-all ${showPolicy ? 'bg-blue-600 text-white shadow-lg' : 'text-slate-400 hover:text-blue-300'}`}
                title="Toggle Policy"
              >
                <Shield className="w-4 h-4" />
              </button>
              <button
                onClick={() => setShowMonitor(!showMonitor)}
                className={`p-2 rounded-md transition-all ${showMonitor ? 'bg-blue-600 text-white shadow-lg' : 'text-slate-400 hover:text-blue-300'}`}
                title="Toggle Monitor"
              >
                <Monitor className="w-4 h-4" />
              </button>
              <button
                onClick={() => setShowViewport(!showViewport)}
                className={`p-2 rounded-md transition-all ${showViewport ? 'bg-purple-600 text-white shadow-lg' : 'text-slate-400 hover:text-purple-300'}`}
                title="Toggle Container Viewport"
              >
                <Globe className="w-4 h-4" />
              </button>
            </div>
            
            <div className="h-8 w-px bg-slate-800" />
            
            <button
              onClick={onClose}
              className="bg-red-900/20 hover:bg-red-600 text-red-500 hover:text-white p-2 rounded-lg transition-all border border-red-500/20"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Main Content Area */}
        <div className="flex-1 overflow-hidden relative">
          
          {/* Background Grid Pattern */}
          <div className="absolute inset-0 opacity-10 pointer-events-none" style={{ backgroundImage: 'radial-gradient(#1e40af 1px, transparent 1px)', backgroundSize: '24px 24px' }} />

          {/* Connections Sidebar - Modular Slide-out */}
          {showConnections && (
            <div
              className={`absolute left-0 top-0 bottom-0 z-[60] transition-all duration-200 ease-in-out translate-x-0`}
            >
              <ConnectionsPanel 
                isOpen={showConnections} 
                onClose={() => setShowConnections(false)}
                onSelectWorker={(worker) => {
                  setSelectedWorker(worker);
                  setShowMetadata(true);
                }}
              />
            </div>
          )}

          {/* Centered Main Area */}
          <div 
            className={`h-full flex flex-col items-center justify-center p-8 transition-all duration-200 ${showConnections ? 'ml-[300px]' : 'ml-0'}`}
          >
            <div className="max-w-3xl w-full space-y-8 text-center">
              <div className="inline-flex p-4 rounded-3xl bg-blue-600/10 border border-blue-600/30 relative">
                <Settings className="w-16 h-16 text-blue-500 animate-spin-slow" />
                <div className="absolute -right-2 -top-2 bg-blue-600 text-white text-[10px] font-black px-2 py-1 rounded-full shadow-lg">
                  v1.1
                </div>
              </div>
              
              <div className="space-y-4">
                <h2 className="text-4xl font-black text-slate-100 tracking-tight">
                  <span className="text-blue-500">Sovereign Memory</span> Fabric Interface
                </h2>
                <p className="text-slate-400 text-sm leading-relaxed max-w-xl mx-auto">
                  You are currently operating within a hardware-isolated Trusted Execution Environment. 
                  All interactions are monitored by eBPF-based high-stakes validation guardrails.
                </p>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div className="p-4 rounded-xl bg-slate-900/50 border border-slate-800 text-left space-y-2">
                  <div className="flex items-center space-x-2 text-blue-400">
                    <Info className="w-4 h-4" />
                    <span className="text-xs font-bold uppercase">Workspace ID</span>
                  </div>
                  <p className="text-lg font-mono text-slate-200">DVE-{actualNodeId?.substring(0, 8) || 'NONE'}</p>
                </div>
                <div className="p-4 rounded-xl bg-slate-900/50 border border-slate-800 text-left space-y-2">
                  <div className="flex items-center space-x-2 text-green-400">
                    <Activity className="w-4 h-4" />
                    <span className="text-xs font-bold uppercase">Uptime</span>
                  </div>
                  <p className="text-lg font-mono text-slate-200">02:45:12</p>
                </div>
                <div className="p-4 rounded-xl bg-slate-900/50 border border-slate-800 text-left space-y-2">
                  <div className="flex items-center space-x-2 text-purple-400">
                    <Zap className="w-4 h-4" />
                    <span className="text-xs font-bold uppercase">Validation</span>
                  </div>
                  <p className="text-lg font-mono text-slate-200">99.9%</p>
                </div>
              </div>
              
              <div className="pt-8">
                <Button
                  onClick={() => {
                    setShowConsole(true);
                    setShowPolicy(true);
                    setShowMonitor(true);
                    setShowConnections(true);
                  }}
                  className="bg-blue-600 hover:bg-blue-700 text-white font-black px-8 py-6 rounded-2xl shadow-[0_0_30px_rgba(37,99,235,0.3)] transition-all transform hover:scale-105 active:scale-95"
                >
                  OPEN WORKSPACE PANELS
                </Button>
                {node?.ip_address && (
                  <Button
                    variant="outline"
                    onClick={() => setShowViewport(true)}
                    className="border-purple-500/40 text-purple-400 hover:bg-purple-600/20 font-bold px-6 py-6 rounded-2xl transition-all"
                  >
                    <Globe className="w-5 h-5 mr-2" />
                    OPEN CONTAINER VIEWPORT
                  </Button>
                )}
              </div>
            </div>
          </div>

          {/* Metadata Panel - Modular Center View */}
          {showMetadata && (
            <MetadataPanel 
              isOpen={showMetadata} 
              onClose={() => {
                setShowMetadata(false);
                setSelectedWorker(null);
              }} 
              node={node || ({
                id: actualNodeId || '',
                name: actualNodeName || 'Unknown Node',
                status: 'online',
                tee_type: 'software',
                stake_amount: 0,
                reputation_score: 100,
                location: 'Distributed',
                ip_address: '',
                public_key: '',
                capabilities: [],
                last_heartbeat: new Date().toISOString(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                cpu_usage: 0,
                memory_usage: 0,
                network_latency: 0,
              } as DVENode)}
              worker={selectedWorker}
              isMonitorOpen={showMonitor}
              isSidebarOpen={showConnections}
            />
          )}

          {/* DVE Solver Panel - Modular Center View */}
          <DVESolverPanel 
            isOpen={showSolver} 
            onClose={() => {
              setShowSolver(false);
              setSelectedTask(null);
            }} 
            isMonitorOpen={showMonitor}
            initialTask={selectedTask}
            onTaskSelect={setSelectedTask}
          />

          {/* Console Panel - Modular Slide-out */}
          <ConsolePanel 
            isOpen={showConsole} 
            onClose={() => setShowConsole(false)} 
            nodeId={actualNodeId}
            isMonitorOpen={showMonitor}
          />

          {/* Policy Editor - Modular Slide-out */}
          <PolicyEditor 
            isOpen={showPolicy} 
            onClose={() => setShowPolicy(false)} 
            nodeId={actualNodeId}
            isMonitorOpen={showMonitor}
          />

          {/* Monitor Panel - Modular Slide-up */}
          <MonitorPanel
            isOpen={showMonitor}
            onClose={() => setShowMonitor(false)}
            nodeId={actualNodeId}
            isSidebarOpen={showConnections}
            onOpenSolver={() => setShowSolver(true)}
            onSelectTask={(task) => setSelectedTask(task)}
          />

          {/* Viewport Panel - Container default web UI */}
          <ViewportPanel
            isOpen={showViewport}
            onClose={() => setShowViewport(false)}
            nodeIp={node?.ip_address || ''}
            nodeName={actualNodeName}
            isSidebarOpen={showConnections}
            isMonitorOpen={showMonitor}
          />
        </div>
      </div>
    </div>
  );
};

export default DVEWorkspacePanel;
