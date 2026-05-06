'use client';

import React, { useState, useEffect } from 'react';
import { Users, X, Search, Activity, Bot, GitBranch, Wifi, User, RefreshCw, Plus } from 'lucide-react';
import { useDemoMode } from '@/contexts/demo-mode-context';
import { webSocketService, WS_EVENTS } from '@/lib/websocket-service';

// DVE Supervisor Agent — the KNIRVAGENT is auto-provisioned by DVECreationService
const AGENT_FRAMEWORKS = [
  { id: 'knirvagent', label: 'DVE Supervisor Agent', description: 'Auto-provisioned KNIRVAGENT supervisor embedded within the DVE' },
] as const;

type AgentFramework = typeof AGENT_FRAMEWORKS[number]['id'];

interface AddAgentModalProps {
  onClose: () => void;
}

const AddAgentModal: React.FC<AddAgentModalProps> = ({ onClose }) => {
  return (
    <div className="absolute inset-0 z-[80] bg-black/70 flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-blue-600/40 rounded-xl w-full max-w-sm shadow-2xl">
        <div className="flex items-center justify-between p-4 border-b border-slate-800">
          <div className="flex items-center gap-2">
            <Bot className="w-4 h-4 text-purple-400" />
            <h3 className="text-sm font-bold text-purple-200 uppercase tracking-wide">DVE Supervisor Agent</h3>
          </div>
          <button onClick={onClose} className="text-slate-500 hover:text-white p-1 rounded transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-4 space-y-4">
          <div className="bg-purple-500/10 border border-purple-500/20 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <Bot className="w-5 h-5 text-purple-400" />
              <span className="text-xs font-bold text-purple-300 uppercase">KNIRVAGENT</span>
            </div>
            <p className="text-[11px] text-slate-300 leading-relaxed">
              The DVE Supervisor Agent is automatically provisioned for each DVE
              when it is created. It runs as an embedded KNIRVAGENT subprocess
              within the DVE's secure enclave and handles all agent communication
              including terminal I/O and task execution.
            </p>
          </div>

          <div className="bg-slate-950 border border-slate-800 rounded-lg p-3 space-y-2">
            <div className="flex items-center justify-between text-[10px]">
              <span className="text-slate-500">Framework</span>
              <span className="text-purple-300 font-mono font-bold">KNIRVAGENT</span>
            </div>
            <div className="flex items-center justify-between text-[10px]">
              <span className="text-slate-500">Lifecycle</span>
              <span className="text-green-400 font-bold">Auto-managed</span>
            </div>
            <div className="flex items-center justify-between text-[10px]">
              <span className="text-slate-500">Communication</span>
              <span className="text-blue-300 font-mono font-bold">Unix Socket / WebSocket</span>
            </div>
          </div>

          <button
            onClick={onClose}
            className="w-full py-2 text-xs font-bold bg-purple-600 hover:bg-purple-700 text-white rounded-lg transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export interface ActiveWorker {
  id: string;
  name: string;
  type: 'agent' | 'workflow' | 'user' | 'connection';
  status: 'active' | 'idle' | 'error' | 'disconnected';
  lastActivity: string;
  tasksCompleted: number;
  cpuUsage?: number;
  memoryUsage?: number;
  metadata?: Record<string, string>;
}

interface ConnectionsPanelProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectWorker?: (worker: ActiveWorker) => void;
}

const ConnectionsPanel: React.FC<ConnectionsPanelProps> = ({ isOpen, onClose, onSelectWorker }) => {
  const [selectedWorker, setSelectedWorker] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const { isDemoMode } = useDemoMode();
  const [workers, setWorkers] = useState<ActiveWorker[]>([]);
  const [showAddAgent, setShowAddAgent] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  useEffect(() => {
    if (isOpen) {
      loadWorkers();
      subscribeToP2PEvents();
    }
    return () => {
      unsubscribeFromP2PEvents();
    };
  }, [isOpen, isDemoMode]);

  const subscribeToP2PEvents = () => {
    webSocketService.subscribe([
      WS_EVENTS.P2P_PEERS,
      WS_EVENTS.P2P_TOPOLOGY,
      WS_EVENTS.NODE_STATUS,
      WS_EVENTS.TASK_COMPLETE,
    ]);

    webSocketService.on(WS_EVENTS.P2P_PEERS, handlePeersUpdate);
    webSocketService.on(WS_EVENTS.P2P_TOPOLOGY, handleTopologyUpdate);
    webSocketService.on(WS_EVENTS.NODE_STATUS, handleNodeStatusUpdate);
    webSocketService.on(WS_EVENTS.TASK_COMPLETE, handleTaskComplete);
  };

  const unsubscribeFromP2PEvents = () => {
    webSocketService.off(WS_EVENTS.P2P_PEERS, handlePeersUpdate);
    webSocketService.off(WS_EVENTS.P2P_TOPOLOGY, handleTopologyUpdate);
    webSocketService.off(WS_EVENTS.NODE_STATUS, handleNodeStatusUpdate);
    webSocketService.off(WS_EVENTS.TASK_COMPLETE, handleTaskComplete);
  };

  const handlePeersUpdate = (payload: { peers?: ActiveWorker[] }) => {
    if (payload.peers && payload.peers.length > 0) {
      setWorkers(payload.peers);
      setLastUpdate(new Date());
    }
  };

  const handleTopologyUpdate = (payload: { connections?: ActiveWorker[] }) => {
    if (payload.connections && payload.connections.length > 0) {
      setWorkers(prev => {
        const existingIds = new Set(prev.map(w => w.id));
        const newWorkers = payload.connections!.filter(w => !existingIds.has(w.id));
        return [...prev, ...newWorkers];
      });
      setLastUpdate(new Date());
    }
  };

  const handleNodeStatusUpdate = (payload: { nodeId: string; status: string }) => {
    setWorkers(prev => prev.map(w => 
      w.id === payload.nodeId ? { ...w, status: payload.status as ActiveWorker['status'] } : w
    ));
  };

  const handleTaskComplete = (payload: { workerId: string }) => {
    setWorkers(prev => prev.map(w => 
      w.id === payload.workerId ? { ...w, tasksCompleted: w.tasksCompleted + 1 } : w
    ));
  };

  const loadWorkers = async () => {
    setIsLoading(true);
    if (isDemoMode) {
      setWorkers(getDemoWorkers());
    } else {
      try {
        const response = await fetch('/api/v1/dve/workers');
        if (response.ok) {
          const data = await response.json();
          setWorkers(data.workers || []);
        } else {
          setWorkers([]);
        }
      } catch (error) {
        console.error('Failed to load workers:', error);
        setWorkers([]);
      }
    }
    setIsLoading(false);
    setLastUpdate(new Date());
  };

  const getDemoWorkers = (): ActiveWorker[] => [
    { id: 'WORKER-001', name: 'Claude Agent Alpha', type: 'agent', status: 'active', lastActivity: new Date().toISOString(), tasksCompleted: 47, cpuUsage: 23, memoryUsage: 512 },
    { id: 'WORKER-002', name: 'Data Pipeline Workflow', type: 'workflow', status: 'active', lastActivity: new Date().toISOString(), tasksCompleted: 12, cpuUsage: 45, memoryUsage: 1024 },
    { id: 'WORKER-003', name: 'User Session #4821', type: 'user', status: 'active', lastActivity: new Date().toISOString(), tasksCompleted: 3, cpuUsage: 5, memoryUsage: 128 },
    { id: 'WORKER-004', name: 'API Gateway Connection', type: 'connection', status: 'idle', lastActivity: new Date(Date.now() - 300000).toISOString(), tasksCompleted: 156, cpuUsage: 2, memoryUsage: 64 },
    { id: 'WORKER-005', name: 'GPT-4 Agent Beta', type: 'agent', status: 'error', lastActivity: new Date(Date.now() - 120000).toISOString(), tasksCompleted: 89, cpuUsage: 0, memoryUsage: 0, metadata: { error: 'Connection timeout' } },
    { id: 'WORKER-006', name: 'Image Processing Flow', type: 'workflow', status: 'active', lastActivity: new Date().toISOString(), tasksCompleted: 8, cpuUsage: 78, memoryUsage: 2048 },
    { id: 'WORKER-007', name: 'User Session #4822', type: 'user', status: 'disconnected', lastActivity: new Date(Date.now() - 600000).toISOString(), tasksCompleted: 0 },
    { id: 'WORKER-008', name: 'WebSocket Pool', type: 'connection', status: 'active', lastActivity: new Date().toISOString(), tasksCompleted: 234, cpuUsage: 12, memoryUsage: 256 },
  ];

  const filteredWorkers = workers.filter(worker =>
    worker.id.toLowerCase().includes(searchTerm.toLowerCase()) ||
    worker.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    worker.type.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const getWorkerIcon = (type: string) => {
    switch (type) {
      case 'agent': return <Bot className="w-4 h-4 text-blue-400" />;
      case 'workflow': return <GitBranch className="w-4 h-4 text-purple-400" />;
      case 'user': return <User className="w-4 h-4 text-green-400" />;
      case 'connection': return <Wifi className="w-4 h-4 text-amber-400" />;
      default: return <Activity className="w-4 h-4 text-slate-400" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-500/20 text-green-400 border-green-500/30';
      case 'idle': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
      case 'error': return 'bg-red-500/20 text-red-400 border-red-500/30';
      case 'disconnected': return 'bg-slate-500/20 text-slate-400 border-slate-500/30';
      default: return 'bg-slate-500/20 text-slate-400 border-slate-500/30';
    }
  };

  const formatLastActivity = (timestamp: string) => {
    const diff = Date.now() - new Date(timestamp).getTime();
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (seconds < 60) return `${seconds}s ago`;
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return new Date(timestamp).toLocaleDateString();
  };

  if (!isOpen) return null;

  return (
    <div
      className="absolute left-0 top-0 h-full z-[60] transition-slide duration-200 gpu-accelerated bg-slate-950 border-r border-blue-600/50 shadow-[10px_0_40px_rgba(0,0,0,0.5)] overflow-hidden"
      style={{
        width: '300px',
        paddingTop: '1rem',
      }}
    >
      {showAddAgent && (
        <AddAgentModal
          onClose={() => setShowAddAgent(false)}
        />
      )}

      <div className="h-full flex flex-col p-4 overflow-hidden">
        <div className="flex items-center justify-between mb-6 border-b border-blue-600/30 pb-4">
          <div className="flex items-center space-x-2">
            <Users className="w-5 h-5 text-blue-400 animate-pulse" />
            <h2 className="text-sm font-bold uppercase tracking-widest text-blue-300">
              Active Workers
            </h2>
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setShowAddAgent(true)}
              className="text-slate-400 hover:text-blue-300 hover:bg-blue-600/20 p-1.5 rounded transition-colors duration-150"
              title="Add Agent Connection"
            >
              <Plus className="w-4 h-4" />
            </button>
            <button
              onClick={onClose}
              className="text-slate-500 hover:text-white hover:bg-slate-800 p-1 rounded transition-colors duration-150"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div className="mb-4 space-y-2">
          <div className="relative group">
            <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-500 group-focus-within:text-blue-400 transition-colors" />
            <input
              type="text"
              placeholder="Filter by ID, Name, Type..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-slate-900 border border-slate-700 group-hover:border-blue-600/50 rounded-full pl-10 pr-4 py-2 text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 transition-colors duration-150"
            />
          </div>
          <div className="flex items-center justify-between">
            <button
              onClick={loadWorkers}
              disabled={isLoading}
              className="text-[10px] text-slate-500 hover:text-blue-400 flex items-center gap-1 transition-colors"
            >
              <RefreshCw className={`w-3 h-3 ${isLoading ? 'animate-spin' : ''}`} />
              Refresh
            </button>
            {lastUpdate && (
              <span className="text-[9px] text-slate-600">
                Updated: {lastUpdate.toLocaleTimeString()}
              </span>
            )}
          </div>
        </div>

        <div className="flex items-center justify-between mb-3 px-1">
          <span className="text-[10px] font-bold text-slate-500 uppercase">
            {filteredWorkers.length} Workers
          </span>
          <div className="flex space-x-2">
            <span className="text-[9px] text-green-400 flex items-center">
              <span className="w-1.5 h-1.5 rounded-full bg-green-500 mr-1" />
              {workers.filter(w => w.status === 'active').length}
            </span>
            <span className="text-[9px] text-yellow-400 flex items-center">
              <span className="w-1.5 h-1.5 rounded-full bg-yellow-500 mr-1" />
              {workers.filter(w => w.status === 'idle').length}
            </span>
            <span className="text-[9px] text-red-400 flex items-center">
              <span className="w-1.5 h-1.5 rounded-full bg-red-500 mr-1" />
              {workers.filter(w => w.status === 'error').length}
            </span>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto pr-2 custom-scrollbar space-y-2">
          {filteredWorkers.map(worker => (
            <div
              key={worker.id}
              onClick={() => {
                setSelectedWorker(worker.id);
                onSelectWorker?.(worker);
              }}
              className={`group relative p-3 rounded-lg cursor-pointer transition-colors duration-150 border-l-4 ${
                selectedWorker === worker.id
                  ? 'bg-blue-600/20 border-blue-500 shadow-[0_0_15px_rgba(59,130,246,0.2)]'
                  : 'bg-slate-900/50 hover:bg-slate-800 border-slate-800 hover:border-slate-700'
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center space-x-2">
                  {getWorkerIcon(worker.type)}
                  <span className="font-mono text-[10px] text-blue-400/80 font-bold">{worker.id}</span>
                </div>
                <span className={`flex items-center text-[9px] font-black uppercase px-1.5 py-0.5 rounded border ${getStatusColor(worker.status)}`}>
                  {worker.status}
                </span>
              </div>
              <div className="text-xs font-bold text-slate-100 group-hover:text-white transition-colors mb-1 line-clamp-1">{worker.name}</div>
              <div className="flex items-center justify-between text-[10px]">
                <span className="text-slate-500 capitalize">{worker.type}</span>
                <span className="text-slate-600 font-mono">{formatLastActivity(worker.lastActivity)}</span>
              </div>
              
              {worker.tasksCompleted > 0 && (
                <div className="mt-2 pt-2 border-t border-slate-800/50">
                  <div className="flex items-center justify-between text-[9px]">
                    <span className="text-slate-500">Tasks</span>
                    <span className="text-blue-400 font-bold">{worker.tasksCompleted}</span>
                  </div>
                </div>
              )}

              {worker.cpuUsage !== undefined && worker.memoryUsage !== undefined && worker.status === 'active' && (
                <div className="mt-2 pt-2 border-t border-slate-800/50 grid grid-cols-2 gap-2">
                  <div className="flex items-center justify-between text-[9px]">
                    <span className="text-slate-500">CPU</span>
                    <span className="text-purple-400 font-mono">{worker.cpuUsage}%</span>
                  </div>
                  <div className="flex items-center justify-between text-[9px]">
                    <span className="text-slate-500">MEM</span>
                    <span className="text-amber-400 font-mono">{worker.memoryUsage}MB</span>
                  </div>
                </div>
              )}

              {worker.metadata?.error && (
                <div className="mt-2 pt-2 border-t border-red-500/20">
                  <div className="text-[9px] text-red-400 italic">{worker.metadata.error}</div>
                </div>
              )}
              
              {selectedWorker === worker.id && (
                <div className="absolute -right-1 top-1/2 -translate-y-1/2 w-1 h-8 bg-blue-500 rounded-full animate-pulse" />
              )}
            </div>
          ))}
          
          {filteredWorkers.length === 0 && (
            <div className="text-center py-10">
              <Activity className="w-8 h-8 text-slate-800 mx-auto mb-2" />
              <div className="text-xs text-slate-600">No active workers matching filters</div>
            </div>
          )}
        </div>
        
        <div className="mt-4 pt-4 border-t border-blue-600/20">
          <div className="flex items-center justify-between text-[10px] text-slate-500 px-1">
            <span>Total Sessions:</span>
            <span className="text-blue-500 font-bold">{workers.length}</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ConnectionsPanel;
