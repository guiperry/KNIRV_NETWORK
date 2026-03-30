'use client';

import React, { useState, useEffect } from 'react';
import { X, Shield, MapPin, Activity, Clock, Terminal, CheckCircle, Zap, ExternalLink, Info, Database, Cpu } from 'lucide-react';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { useDVENodes } from '@/hooks/use-dve-nodes';
import { useSSHSession } from '@/hooks/use-ssh-session';
import { useValidationSession } from '@/hooks/use-validation-session';
import { useErrorResolutionSession } from '@/hooks/use-error-resolution-session';
import type { DVENode, TEEEndpoint } from '@/types/api';

interface DVECardModalProps {
  isOpen: boolean;
  onClose: () => void;
  node: DVENode;
}

const DVECardModal: React.FC<DVECardModalProps> = ({ isOpen, onClose, node }) => {
  const { toast } = useToast();
  const { getNodeEndpoints } = useDVENodes();
  const sshSession = useSSHSession();
  const validationSession = useValidationSession();
  const errorResolutionSession = useErrorResolutionSession();

  const [endpoints, setEndpoints] = useState<TEEEndpoint[]>([]);
  const [loadingEndpoints, setLoadingEndpoints] = useState(false);

  // Load endpoints when modal opens
  useEffect(() => {
    if (isOpen && node) {
      loadEndpoints();
    }
  }, [isOpen, node]);

  const loadEndpoints = async () => {
    setLoadingEndpoints(true);
    try {
      const nodeEndpoints = await getNodeEndpoints(node.id);
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

  // ⭐ NEW: Handle SSH connection
  const handleSSHConnect = async () => {
    try {
      const session = await sshSession.createSession(node.id);
      if (session) {
        // Download private key
        await sshSession.downloadPrivateKey(session.id);

        // Show connection command
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

  // ⭐ NEW: Handle validation connection
  const handleValidationConnect = async () => {
    try {
      const session = await validationSession.createSession(node.id);
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

  // ⭐ NEW: Handle error resolution connection
  const handleErrorResolutionConnect = async () => {
    try {
      const session = await errorResolutionSession.createSession(node.id);
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

  if (!isOpen) return null;

  const getTEEIcon = (teeType: string) => {
    switch (teeType.toUpperCase()) {
      case 'SGX': return <Shield className="w-5 h-5 text-blue-500" />;
      case 'SEV-SNP': return <Shield className="w-5 h-5 text-green-500" />;
      case 'TDX': return <Shield className="w-5 h-5 text-purple-500" />;
      default: return <Shield className="w-5 h-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online': return 'bg-green-500/20 text-green-400 border-green-500/30';
      case 'offline': return 'bg-red-500/20 text-red-400 border-red-500/30';
      case 'maintenance': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
      case 'error': return 'bg-red-600/20 text-red-500 border-red-600/30';
      default: return 'bg-gray-500/20 text-gray-400 border-gray-500/30';
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-center justify-center p-4">
      <div className="bg-slate-950 border border-blue-600/50 shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden rounded-2xl flex flex-col">
        
        {/* Header */}
        <div className="bg-slate-900 border-b border-blue-600/30 p-6 flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="p-3 bg-blue-600/10 rounded-xl border border-blue-600/20">
              <Database className="w-6 h-6 text-blue-400" />
            </div>
            <div>
              <h2 className="text-xl font-black text-blue-100 uppercase tracking-tighter">{node.name}</h2>
              <div className="text-[10px] font-mono text-slate-500 flex items-center space-x-2">
                <span>ID: {node.id}</span>
                <span className="text-blue-500/50">•</span>
                <span>FABRIC CONTEXT</span>
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-500 hover:text-white hover:bg-slate-800 p-2 rounded-lg transition-interactive"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6 custom-scrollbar">
          {/* Status Bar */}
          <div className="flex items-center justify-between bg-slate-900/50 p-3 rounded-xl border border-slate-800">
            <div className="flex items-center space-x-3">
              <Badge variant="outline" className={`${getStatusColor(node.status)} font-black text-[10px] uppercase px-3`}>
                {node.status}
              </Badge>
              <div className="flex items-center text-[10px] font-bold text-slate-500 uppercase tracking-widest">
                <Clock className="w-3 h-3 mr-1.5" />
                Heartbeat: {new Date(node.last_heartbeat).toLocaleTimeString()}
              </div>
            </div>
            <div className="flex items-center text-[10px] font-bold text-blue-500 uppercase tracking-widest">
              <Activity className="w-3 h-3 mr-1.5" />
              Sync: 100%
            </div>
          </div>

          {/* Performance Metrics */}
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-4 space-y-3">
              <div className="flex justify-between items-center">
                <div className="flex items-center text-[10px] font-black text-slate-400 uppercase">
                  <Cpu className="w-3 h-3 mr-2 text-blue-500" />
                  Compute Load
                </div>
                <span className="text-xs font-mono text-blue-400 font-bold">{node.cpu_usage}%</span>
              </div>
              <Progress value={node.cpu_usage} className="h-1 bg-slate-800" />
            </div>

            <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-4 space-y-3">
              <div className="flex justify-between items-center">
                <div className="flex items-center text-[10px] font-black text-slate-400 uppercase">
                  <Zap className="w-3 h-3 mr-2 text-purple-500" />
                  Memory Pool
                </div>
                <span className="text-xs font-mono text-purple-400 font-bold">{node.memory_usage}%</span>
              </div>
              <Progress value={node.memory_usage} className="h-1 bg-slate-800" />
            </div>
          </div>

          {/* Node Metadata Grid */}
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Hardware TEE</p>
                <div className="flex items-center space-x-2">
                  {getTEEIcon(node.tee_type)}
                  <span className="text-xs font-bold text-slate-200">{node.tee_type}</span>
                </div>
              </div>
            </div>

            <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Geographic Loc</p>
                <div className="flex items-center space-x-2">
                  <MapPin className="w-4 h-4 text-red-500" />
                  <span className="text-xs font-bold text-slate-200">{node.location || 'Distributed'}</span>
                </div>
              </div>
            </div>

            <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Economic Stake</p>
                <span className="text-xs font-bold text-green-500">{node.stake_amount.toLocaleString()} NRN</span>
              </div>
            </div>

            <div className="bg-slate-950 border border-slate-800 rounded-xl p-4 flex items-center justify-between">
              <div className="space-y-1">
                <p className="text-[9px] font-black text-slate-500 uppercase tracking-widest">Trust Score</p>
                <span className={`text-xs font-bold ${node.reputation_score > 80 ? 'text-blue-400' : 'text-yellow-500'}`}>
                  {node.reputation_score}/100 VERIFIED
                </span>
              </div>
            </div>
          </div>

          {/* Endpoints */}
          <div className="space-y-4">
            <h3 className="text-[10px] font-black text-blue-500 uppercase tracking-[0.2em]">Hardware Endpoints</h3>
            
            {loadingEndpoints ? (
              <div className="flex items-center justify-center py-8">
                <Activity className="w-5 h-5 animate-spin text-blue-500" />
              </div>
            ) : endpoints.length === 0 ? (
              <div className="bg-slate-900/30 border border-dashed border-slate-800 rounded-xl p-8 text-center">
                <Info className="w-8 h-8 text-slate-700 mx-auto mb-2" />
                <p className="text-xs text-slate-500">No active Fabric endpoints detected for this context.</p>
              </div>
            ) : (
              <div className="space-y-2">
                {endpoints.map((endpoint, index) => (
                  <div key={index} className="flex items-center justify-between p-4 bg-slate-900 hover:bg-slate-800 transition-colors border border-slate-800 rounded-xl group">
                    <div className="flex items-center space-x-4">
                      <div className="p-2 bg-slate-950 rounded-lg border border-slate-800 group-hover:border-blue-500/30 transition-colors">
                        {endpoint.endpoint_type === 'ssh' && <Terminal className="w-4 h-4 text-green-400" />}
                        {endpoint.endpoint_type === 'validation' && <CheckCircle className="w-4 h-4 text-blue-400" />}
                        {endpoint.endpoint_type === 'error-resolution' && <Zap className="w-4 h-4 text-orange-400" />}
                      </div>
                      <div>
                        <p className="text-xs font-black text-slate-200 uppercase tracking-tight">
                          {endpoint.endpoint_type.replace('-', ' ')} ACCESS
                        </p>
                        <p className="text-[10px] font-mono text-slate-500">
                          {endpoint.host}:{endpoint.port} • {endpoint.protocol.toUpperCase()}
                        </p>
                      </div>
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      className="text-[10px] font-black uppercase border-blue-600/30 text-blue-400 hover:bg-blue-600 hover:text-white"
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

        {/* Footer */}
        <div className="bg-slate-900 border-t border-blue-600/30 p-4 flex justify-end">
          <Button
            variant="ghost"
            onClick={onClose}
            className="text-slate-500 hover:text-white text-xs font-bold uppercase tracking-widest"
          >
            Close Metadata
          </Button>
        </div>
      </div>
    </div>
  );
};

export default DVECardModal;
