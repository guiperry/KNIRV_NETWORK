'use client';

import React, { useState, useEffect } from 'react';
import { X, Shield, MapPin, Activity, Clock, Terminal, CheckCircle, Zap, Info, Database, Cpu, Bot, GitBranch, Wifi, User } from 'lucide-react';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { useDVENodes } from '@/hooks/use-dve-nodes';
import { useSSHSession } from '@/hooks/use-ssh-session';
import { useValidationSession } from '@/hooks/use-validation-session';
import { useErrorResolutionSession } from '@/hooks/use-error-resolution-session';
import type { DVENode, TEEEndpoint } from '@/types/api';
import type { ActiveWorker } from './connections-panel';

interface MetadataPanelProps {
  isOpen: boolean;
  onClose: () => void;
  node?: DVENode;
  worker?: ActiveWorker | null;
  isMonitorOpen?: boolean;
  isSidebarOpen?: boolean;
}

const MetadataPanel: React.FC<MetadataPanelProps> = ({ isOpen, onClose, node, worker, isMonitorOpen, isSidebarOpen }) => {
  const { toast } = useToast();
  const { getNodeEndpoints } = useDVENodes();
  const sshSession = useSSHSession();
  const validationSession = useValidationSession();
  const errorResolutionSession = useErrorResolutionSession();

  const [endpoints, setEndpoints] = useState<TEEEndpoint[]>([]);
  const [loadingEndpoints, setLoadingEndpoints] = useState(false);

  const isWorkerMode = !!worker;

  useEffect(() => {
    if (isOpen && node && !isWorkerMode) {
      loadEndpoints();
    }
  }, [isOpen, node, isWorkerMode]);

  if (!isOpen) return null;

  const safeNode = node ? {
    id: node.id || '',
    name: node.name || 'Unknown Node',
    status: node.status || 'offline',
    tee_type: node.tee_type || 'software',
    stake_amount: node.stake_amount ?? 0,
    reputation_score: node.reputation_score ?? 0,
    location: node.location || 'Distributed',
    ip_address: node.ip_address || '',
    public_key: node.public_key || '',
    capabilities: node.capabilities || [],
    last_heartbeat: node.last_heartbeat || new Date().toISOString(),
    created_at: node.created_at || new Date().toISOString(),
    updated_at: node.updated_at || new Date().toISOString(),
    cpu_usage: node.cpu_usage ?? 0,
    memory_usage: node.memory_usage ?? 0,
    network_latency: node.network_latency ?? 0,
  } : null;

  const loadEndpoints = async () => {
    if (!safeNode) return;
    setLoadingEndpoints(true);
    try {
      const nodeEndpoints = await getNodeEndpoints(safeNode.id);
      setEndpoints(nodeEndpoints || []);
    } catch (error) {
      console.error('Failed to load endpoints:', error);
      toast({
        title: "Failed to Load Endpoints",
        description: "Could not retrieve endpoint information for this node.",
        variant: "destructive",
      });
    } finally {
      setLoadingEndpoints(false);
    }
  };

  const handleSSHConnect = async () => {
    if (!safeNode) return;
    try {
      const session = await sshSession.createSession(safeNode.id);
      if (session) {
        await sshSession.downloadPrivateKey(session.id);
        const command = `ssh -i dve-ssh-key.pem ${session.username}@${session.endpoint} -p ${session.port}`;
        navigator.clipboard.writeText(command);
        toast({
          title: "SSH Session Created",
          description: "SSH command copied to clipboard. Private key downloaded.",
        });
      }
    } catch (error) {
      toast({
        title: "SSH Connection Failed",
        description: "Failed to create SSH session. Please try again.",
        variant: "destructive",
      });
    }
  };

  const handleValidationConnect = async () => {
    if (!safeNode) return;
    try {
      const session = await validationSession.createSession(safeNode.id);
      if (session) {
        validationSession.openValidationInterface(session);
      }
    } catch (error) {
      toast({
        title: "Validation Connection Failed",
        description: "Failed to create validation session. Please try again.",
        variant: "destructive",
      });
    }
  };

  const handleErrorResolutionConnect = async () => {
    if (!safeNode) return;
    try {
      const session = await errorResolutionSession.createSession(safeNode.id);
      if (session) {
        errorResolutionSession.openErrorResolutionInterface(session);
      }
    } catch (error) {
      toast({
        title: "Error Resolution Connection Failed",
        description: "Failed to create error resolution session. Please try again.",
        variant: "destructive",
      });
    }
  };

  const getTEEIcon = (teeType?: string) => {
    const type = teeType?.toUpperCase() || 'SOFTWARE';
    switch (type) {
      case 'SGX': return <Shield className="w-5 h-5 text-blue-500" />;
      case 'SEV-SNP': return <Shield className="w-5 h-5 text-green-500" />;
      case 'TDX': return <Shield className="w-5 h-5 text-purple-500" />;
      default: return <Shield className="w-5 h-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status?: string) => {
    const s = status?.toLowerCase() || 'offline';
    switch (s) {
      case 'online': case 'active': return 'bg-green-500/20 text-green-400 border-green-500/30';
      case 'offline': case 'disconnected': return 'bg-red-500/20 text-red-400 border-red-500/30';
      case 'idle': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
      case 'error': return 'bg-red-600/20 text-red-500 border-red-600/30';
      case 'maintenance': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
      default: return 'bg-gray-500/20 text-gray-400 border-gray-500/30';
    }
  };

  const getWorkerIcon = (type?: string) => {
    switch (type) {
      case 'agent': return <Bot className="w-5 h-5 text-blue-400" />;
      case 'workflow': return <GitBranch className="w-5 h-5 text-purple-400" />;
      case 'user': return <User className="w-5 h-5 text-green-400" />;
      case 'connection': return <Wifi className="w-5 h-5 text-amber-400" />;
      default: return <Activity className="w-5 h-5 text-slate-400" />;
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

  const sidebarOffset = isSidebarOpen ? 300 : 0;

  return (
    <div
      className="absolute z-[100] transition-all duration-500 transform ease-in-out bg-slate-950 border border-blue-600/50 shadow-[0_0_50px_rgba(0,0,0,0.8)] overflow-hidden rounded-2xl flex flex-col"
      style={{
        left: sidebarOffset + 20,
        top: isMonitorOpen ? '45%' : '50%',
        transform: 'translate(0, -50%)',
        width: isMonitorOpen ? '500px' : '550px',
        maxHeight: isMonitorOpen ? '60vh' : '80vh',
      }}
    >
      {isWorkerMode && worker ? (
        <>
          <div className="bg-slate-900 border-b border-blue-600/30 p-5 flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="p-2 bg-blue-600/10 rounded-xl border border-blue-600/20">
                {getWorkerIcon(worker.type)}
              </div>
              <div>
                <h2 className="text-lg font-black text-blue-100 uppercase tracking-tighter">{worker.name}</h2>
                <div className="text-[9px] font-mono text-slate-500 flex items-center space-x-2">
                  <span>ID: {worker.id}</span>
                  <span className="text-blue-500/50">•</span>
                  <span className="capitalize">{worker.type} Worker</span>
                </div>
              </div>
            </div>
            <button
              onClick={onClose}
              className="text-slate-500 hover:text-white hover:bg-slate-800 p-1.5 rounded-lg transition-all"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-5 space-y-5 custom-scrollbar">
            <div className="flex items-center justify-between bg-slate-900/50 p-2.5 rounded-xl border border-slate-800">
              <div className="flex items-center space-x-3">
                <Badge variant="outline" className={`${getStatusColor(worker.status)} font-black text-[9px] uppercase px-2`}>
                  {worker.status}
                </Badge>
                <div className="flex items-center text-[9px] font-bold text-slate-500 uppercase tracking-widest">
                  <Clock className="w-3 h-3 mr-1" />
                  Last Activity: {formatLastActivity(worker.lastActivity)}
                </div>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-3 space-y-2">
                <div className="flex justify-between items-center">
                  <div className="flex items-center text-[9px] font-black text-slate-400 uppercase">
                    <Cpu className="w-3 h-3 mr-1.5 text-blue-500" />
                    CPU Usage
                  </div>
                  <span className="text-[10px] font-mono text-blue-400 font-bold">{worker.cpuUsage ?? 0}%</span>
                </div>
                <Progress value={worker.cpuUsage ?? 0} className="h-1 bg-slate-800" />
              </div>

              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-3 space-y-2">
                <div className="flex justify-between items-center">
                  <div className="flex items-center text-[9px] font-black text-slate-400 uppercase">
                    <Zap className="w-3 h-3 mr-1.5 text-purple-500" />
                    Memory
                  </div>
                  <span className="text-[10px] font-mono text-purple-400 font-bold">{worker.memoryUsage ?? 0}MB</span>
                </div>
                <Progress value={Math.min((worker.memoryUsage ?? 0) / 20, 100)} className="h-1 bg-slate-800" />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Type</p>
                  <div className="flex items-center space-x-1.5">
                    {getWorkerIcon(worker.type)}
                    <span className="text-[10px] font-bold text-slate-200 capitalize">{worker.type}</span>
                  </div>
                </div>
              </div>

              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Tasks Completed</p>
                  <span className="text-[10px] font-bold text-green-500">{worker.tasksCompleted}</span>
                </div>
              </div>
            </div>

            {worker.metadata && Object.keys(worker.metadata).length > 0 && (
              <div className="space-y-3">
                <h3 className="text-[9px] font-black text-blue-500 uppercase tracking-[0.2em]">Additional Metadata</h3>
                <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 space-y-2">
                  {Object.entries(worker.metadata).map(([key, value]) => (
                    <div key={key} className="flex justify-between text-[10px]">
                      <span className="text-slate-500 font-bold uppercase">{key}</span>
                      <span className="text-slate-300 font-mono">{value}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="bg-slate-900 border-t border-blue-600/30 p-3 flex justify-end">
            <Button
              variant="ghost"
              onClick={onClose}
              className="text-slate-500 hover:text-white text-[9px] font-bold uppercase tracking-widest h-8"
            >
              Close Worker Info
            </Button>
          </div>
        </>
      ) : safeNode ? (
        <>
          <div className="bg-slate-900 border-b border-blue-600/30 p-5 flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="p-2 bg-blue-600/10 rounded-xl border border-blue-600/20">
                <Database className="w-5 h-5 text-blue-400" />
              </div>
              <div>
                <h2 className="text-lg font-black text-blue-100 uppercase tracking-tighter">{safeNode.name}</h2>
                <div className="text-[9px] font-mono text-slate-500 flex items-center space-x-2">
                  <span>ID: {safeNode.id}</span>
                  <span className="text-blue-500/50">•</span>
                  <span>FABRIC CONTEXT</span>
                </div>
              </div>
            </div>
            <button
              onClick={onClose}
              className="text-slate-500 hover:text-white hover:bg-slate-800 p-1.5 rounded-lg transition-all"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-5 space-y-5 custom-scrollbar">
            <div className="flex items-center justify-between bg-slate-900/50 p-2.5 rounded-xl border border-slate-800">
              <div className="flex items-center space-x-3">
                <Badge variant="outline" className={`${getStatusColor(safeNode.status)} font-black text-[9px] uppercase px-2`}>
                  {safeNode.status}
                </Badge>
                <div className="flex items-center text-[9px] font-bold text-slate-500 uppercase tracking-widest">
                  <Clock className="w-3 h-3 mr-1" />
                  {new Date(safeNode.last_heartbeat).toLocaleTimeString()}
                </div>
              </div>
              <div className="flex items-center text-[9px] font-bold text-blue-500 uppercase tracking-widest">
                <Activity className="w-3 h-3 mr-1" />
                SYNC: 100%
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-3 space-y-2">
                <div className="flex justify-between items-center">
                  <div className="flex items-center text-[9px] font-black text-slate-400 uppercase">
                    <Cpu className="w-3 h-3 mr-1.5 text-blue-500" />
                    Compute
                  </div>
                  <span className="text-[10px] font-mono text-blue-400 font-bold">{safeNode.cpu_usage}%</span>
                </div>
                <Progress value={safeNode.cpu_usage} className="h-1 bg-slate-800" />
              </div>

              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-3 space-y-2">
                <div className="flex justify-between items-center">
                  <div className="flex items-center text-[9px] font-black text-slate-400 uppercase">
                    <Zap className="w-3 h-3 mr-1.5 text-purple-500" />
                    Memory
                  </div>
                  <span className="text-[10px] font-mono text-purple-400 font-bold">{safeNode.memory_usage}%</span>
                </div>
                <Progress value={safeNode.memory_usage} className="h-1 bg-slate-800" />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Hardware TEE</p>
                  <div className="flex items-center space-x-1.5">
                    {getTEEIcon(safeNode.tee_type)}
                    <span className="text-[10px] font-bold text-slate-200">{safeNode.tee_type}</span>
                  </div>
                </div>
              </div>

              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Geographic Loc</p>
                  <div className="flex items-center space-x-1.5">
                    <MapPin className="w-3.5 h-3.5 text-red-500" />
                    <span className="text-[10px] font-bold text-slate-200">{safeNode.location || 'Distributed'}</span>
                  </div>
                </div>
              </div>

              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Economic Stake</p>
                  <span className="text-[10px] font-bold text-green-500">{safeNode.stake_amount.toLocaleString()} NRN</span>
                </div>
              </div>

              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Trust Score</p>
                  <span className={`text-[10px] font-bold ${safeNode.reputation_score > 80 ? 'text-blue-400' : 'text-yellow-500'}`}>
                    {safeNode.reputation_score}/100 VERIFIED
                  </span>
                </div>
              </div>
            </div>

            <div className="space-y-3">
              <h3 className="text-[9px] font-black text-blue-500 uppercase tracking-[0.2em]">Hardware Endpoints</h3>
              
              {loadingEndpoints ? (
                <div className="flex items-center justify-center py-4">
                  <Activity className="w-4 h-4 animate-spin text-blue-500" />
                </div>
              ) : endpoints.length === 0 ? (
                <div className="bg-slate-900/30 border border-dashed border-slate-800 rounded-xl p-6 text-center">
                  <Info className="w-6 h-6 text-slate-700 mx-auto mb-1.5" />
                  <p className="text-[10px] text-slate-500">No active Fabric endpoints detected for this context.</p>
                </div>
              ) : (
                <div className="space-y-2">
                  {endpoints.map((endpoint, index) => (
                    <div key={index} className="flex items-center justify-between p-3 bg-slate-900 hover:bg-slate-800 transition-colors border border-slate-800 rounded-xl group">
                      <div className="flex items-center space-x-3">
                        <div className="p-1.5 bg-slate-950 rounded-lg border border-slate-800 group-hover:border-blue-500/30 transition-colors">
                          {endpoint.endpoint_type === 'ssh' && <Terminal className="w-3.5 h-3.5 text-green-400" />}
                          {endpoint.endpoint_type === 'validation' && <CheckCircle className="w-3.5 h-3.5 text-blue-400" />}
                          {endpoint.endpoint_type === 'error-resolution' && <Zap className="w-3.5 h-3.5 text-orange-400" />}
                        </div>
                        <div>
                          <p className="text-[10px] font-black text-slate-200 uppercase tracking-tight">
                            {endpoint.endpoint_type.replace('-', ' ')} ACCESS
                          </p>
                          <p className="text-[9px] font-mono text-slate-500">
                            {endpoint.host}:{endpoint.port} • {endpoint.protocol.toUpperCase()}
                          </p>
                        </div>
                      </div>
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-7 text-[9px] font-black uppercase border-blue-600/30 text-blue-400 hover:bg-blue-600 hover:text-white px-3"
                        onClick={() => {
                          if (endpoint.endpoint_type === 'ssh') {
                            handleSSHConnect();
                          } else if (endpoint.endpoint_type === 'validation') {
                            handleValidationConnect();
                          } else if (endpoint.endpoint_type === 'error-resolution') {
                            handleErrorResolutionConnect();
                          }
                        }}
                      >
                        Initialize
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <div className="bg-slate-900 border-t border-blue-600/30 p-3 flex justify-end">
            <Button
              variant="ghost"
              onClick={onClose}
              className="text-slate-500 hover:text-white text-[9px] font-bold uppercase tracking-widest h-8"
            >
              Close Metadata
            </Button>
          </div>
        </>
      ) : null}
    </div>
  );
};

export default MetadataPanel;
