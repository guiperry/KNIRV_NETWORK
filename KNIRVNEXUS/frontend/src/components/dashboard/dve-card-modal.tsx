'use client';

import React, { useState, useEffect } from 'react';
import { X, Shield, MapPin, Activity, Clock, Terminal, CheckCircle, Zap, ExternalLink } from 'lucide-react';
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
    switch (teeType) {
      case 'SGX': return <Shield className="w-5 h-5 text-blue-500" />;
      case 'SEV-SNP': return <Shield className="w-5 h-5 text-green-500" />;
      case 'TDX': return <Shield className="w-5 h-5 text-purple-500" />;
      default: return <Shield className="w-5 h-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online': return 'bg-green-500';
      case 'offline': return 'bg-red-500';
      case 'maintenance': return 'bg-yellow-500';
      case 'error': return 'bg-red-600';
      default: return 'bg-gray-500';
    }
  };

  const formatLastHeartbeat = (heartbeat: string) => {
    const date = new Date(heartbeat);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ago`;
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-gradient-to-br from-slate-900 via-blue-950 to-slate-950 rounded-lg border-2 border-blue-600/50 shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-gradient-to-r from-slate-900 to-blue-950 border-b border-blue-600/30 p-6 flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-blue-300">{node.name}</h2>
            <p className="text-sm text-slate-400 mt-1">ID: {node.id}</p>
          </div>
          <button
            onClick={onClose}
            className="bg-slate-700 hover:bg-slate-600 p-2 rounded-lg transition-colors"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Status Badge */}
          <div className="flex items-center space-x-3">
            <Badge 
              variant="secondary" 
              className={`${getStatusColor(node.status)} text-white text-sm`}
            >
              {node.status.toUpperCase()}
            </Badge>
            <span className="text-sm text-slate-400">Last heartbeat: {formatLastHeartbeat(node.last_heartbeat)}</span>
          </div>

          {/* Performance Metrics */}
          <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-4 space-y-4">
            <h3 className="text-lg font-semibold text-blue-300">Performance Metrics</h3>
            
            <div>
              <div className="flex justify-between items-center mb-2">
                <label className="text-sm font-medium text-slate-300">CPU Usage</label>
                <span className="text-sm font-semibold text-blue-400">{node.cpu_usage}%</span>
              </div>
              <Progress value={node.cpu_usage} className="h-2" />
            </div>

            <div>
              <div className="flex justify-between items-center mb-2">
                <label className="text-sm font-medium text-slate-300">Memory Usage</label>
                <span className="text-sm font-semibold text-blue-400">{node.memory_usage}%</span>
              </div>
              <Progress value={node.memory_usage} className="h-2" />
            </div>
          </div>

          {/* Node Details Grid */}
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-4">
              <p className="text-xs text-slate-400 mb-2">TEE Type</p>
              <div className="flex items-center space-x-2">
                {getTEEIcon(node.tee_type)}
                <span className="font-semibold text-slate-200">{node.tee_type}</span>
              </div>
            </div>

            <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-4">
              <p className="text-xs text-slate-400 mb-2">Stake Amount</p>
              <p className="font-semibold text-slate-200">{node.stake_amount.toLocaleString()} NRN</p>
            </div>

            <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-4">
              <p className="text-xs text-slate-400 mb-2">Reputation Score</p>
              <p className="font-semibold text-slate-200">{node.reputation_score}/100</p>
            </div>

            {node.location && (
              <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-4">
                <p className="text-xs text-slate-400 mb-2">Location</p>
                <div className="flex items-center space-x-2">
                  <MapPin className="w-4 h-4 text-blue-400" />
                  <span className="font-semibold text-slate-200">{node.location}</span>
                </div>
              </div>
            )}
          </div>

          {/* ⭐ NEW: Endpoint Access Section */}
          <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-4">
            <h3 className="text-lg font-semibold text-blue-300 mb-4">Access Endpoints</h3>

            {loadingEndpoints ? (
              <div className="flex items-center justify-center py-4">
                <Activity className="w-5 h-5 animate-spin mr-2" />
                <span className="text-sm text-slate-400">Loading endpoints...</span>
              </div>
            ) : endpoints.length === 0 ? (
              <div className="text-center py-4">
                <p className="text-sm text-slate-400">No active endpoints available</p>
                <p className="text-xs text-slate-500 mt-1">Rent this node to access endpoints</p>
              </div>
            ) : (
              <div className="space-y-3">
                {endpoints.map((endpoint, index) => (
                  <div key={index} className="flex items-center justify-between p-3 bg-slate-700/50 rounded-lg">
                    <div className="flex items-center space-x-3">
                      {endpoint.endpoint_type === 'ssh' && <Terminal className="w-4 h-4 text-green-400" />}
                      {endpoint.endpoint_type === 'validation' && <CheckCircle className="w-4 h-4 text-blue-400" />}
                      {endpoint.endpoint_type === 'error-resolution' && <Zap className="w-4 h-4 text-orange-400" />}
                      <div>
                        <p className="text-sm font-medium text-slate-200 capitalize">
                          {endpoint.endpoint_type.replace('-', ' ')}
                        </p>
                        <p className="text-xs text-slate-400">
                          {endpoint.host}:{endpoint.port} ({endpoint.protocol})
                        </p>
                      </div>
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      className="text-xs"
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
                      <ExternalLink className="w-3 h-3 mr-1" />
                      Connect
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Additional Information */}
          <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-4">
            <h3 className="text-sm font-semibold text-blue-300 mb-3">Additional Information</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-slate-400">TEE Version:</span>
                <span className="text-slate-200">v2.1.0</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-400">Last Update:</span>
                <span className="text-slate-200">2 hours ago</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-400">Uptime:</span>
                <span className="text-slate-200">45 days</span>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="sticky bottom-0 bg-gradient-to-r from-slate-900 to-blue-950 border-t border-blue-600/30 p-4 flex justify-end space-x-2">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-100 font-medium rounded-lg transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default DVECardModal;
