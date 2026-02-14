'use client';

import React, { useEffect, useState } from 'react';
import { Server, Cpu, HardDrive, Shield, MapPin, Clock, AlertCircle, CheckCircle, Activity, CreditCard, Play, Square, Zap, Info } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { useDVENodes } from '@/hooks/use-dve-nodes';
import { useDVERental } from '@/hooks/use-dve-rental';
import { useAuth } from '@/lib/auth-context';
import { CDEAccessModal } from '@/components/cde/cde-access-modal';
import type { DVENode, DVEAccessInfo } from '@/types/api';
import { useToast } from '@/hooks/use-toast';

interface DVENodesPanelProps {
  className?: string;
  onRentClick?: () => void;
  onNodeConnect?: (node: DVENode) => void;
}

export const DVENodesPanel: React.FC<DVENodesPanelProps> = ({ className, onRentClick, onNodeConnect }) => {
  const { user } = useAuth();
  const { toast } = useToast();
  const {
    nodes,
    isLoading,
    error,
    refreshNodes,
    getOnlineNodes,
    getNodesByTEE,
    getNodeEndpoints,
  } = useDVENodes();

  const {
    getFullAccessInfo,
    createSSHSession,
    createValidationSession,
    createErrorResolutionSession,
  } = useDVERental();

  const [filter, setFilter] = useState<string>('all');
  const [selectedNode, setSelectedNode] = useState<DVENode | null>(null);
  const [cdeOpen, setCDEOpen] = useState(false);
  const [activeNodeId, setActiveNodeId] = useState<{ [key: string]: boolean }>({});
  const [provisioningNodes, setProvisioningNodes] = useState<Set<string>>(new Set());

  // ⭐ NEW: Handle Start button - provision container for rental
  const handleStartDVE = async (node: DVENode) => {
    if (provisioningNodes.has(node.id)) return;

    if (activeNodeId[node.id]) {
      // Stop logic
      setActiveNodeId(prev => ({ ...prev, [node.id]: false }));
      toast({
        title: "Node Stopped",
        description: `DVE instance ${node.name} has been disconnected.`,
      });
      return;
    }

    setProvisioningNodes(prev => new Set(prev).add(node.id));

    try {
      // Check if node is already rented by this user
      const endpoints = await getNodeEndpoints(node.id);

      if (endpoints && endpoints.length > 0) {
        // Node is already rented
        setActiveNodeId(prev => ({ ...prev, [node.id]: true }));
        toast({
          title: "Node Ready",
          description: "DVE instance is active. You can now access the modular CDE.",
        });
      } else {
        // Node not rented - redirect to rental modal
        if (onRentClick) {
          onRentClick();
        } else {
          toast({
            title: "Rental Required",
            description: "Please use the Rent button to create a rental first.",
          });
        }
      }
    } catch (error) {
      toast({
        title: "Provisioning Failed",
        description: "Failed to provision DVE container. Please try again.",
        variant: "destructive",
      });
    } finally {
      setProvisioningNodes(prev => {
        const newSet = new Set(prev);
        newSet.delete(node.id);
        return newSet;
      });
    }
  };

  const handleAccessDVE = (node: DVENode) => {
    if (onNodeConnect) {
      onNodeConnect(node);
    } else {
      setSelectedNode(node);
      setCDEOpen(true);
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

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online': return <CheckCircle className="w-4 h-4" />;
      case 'offline': return <AlertCircle className="w-4 h-4" />;
      case 'maintenance': return <Clock className="w-4 h-4" />;
      case 'error': return <AlertCircle className="w-4 h-4" />;
      default: return <Server className="w-4 h-4" />;
    }
  };

  const getTEEIcon = (teeType: string) => {
    switch (teeType.toUpperCase()) {
      case 'SGX': return <Shield className="w-4 h-4 text-blue-500" />;
      case 'SEV-SNP': return <Shield className="w-4 h-4 text-green-500" />;
      case 'TDX': return <Shield className="w-4 h-4 text-purple-500" />;
      default: return <Shield className="w-4 h-4 text-gray-500" />;
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

  const filteredNodes = nodes.filter(node => {
    if (filter === 'all') return true;
    if (filter === 'online') return node.status === 'online';
    if (filter === 'offline') return node.status === 'offline';
    if (filter === 'maintenance') return node.status === 'maintenance';
    return true;
  });

  const nodeStats = {
    total: nodes.length,
    online: nodes.filter(n => n.status === 'online').length,
    offline: nodes.filter(n => n.status === 'offline').length,
    maintenance: nodes.filter(n => n.status === 'maintenance').length,
  };

  if (error) {
    return (
      <div className={`space-y-4 ${className}`}>
        <Card className="border-red-200 bg-red-50">
          <CardHeader>
            <CardTitle className="flex items-center space-x-2 text-red-700">
              <AlertCircle className="w-5 h-5" />
              <span>DVE Nodes Error</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-red-600 text-sm">{error}</p>
            <Button 
              variant="outline" 
              size="sm" 
              className="mt-2"
              onClick={refreshNodes}
            >
              Retry
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Developer Role Scoped View Alert */}
      {user?.role === 'observer' && (
        <Alert className="border-blue-200 bg-blue-50">
          <Info className="h-4 w-4 text-blue-600" />
          <AlertDescription className="text-blue-800">
            <strong>Developer View:</strong> You are viewing only the DVE instances you have rented. 
            Root and Operator roles can see all network resources.
          </AlertDescription>
        </Alert>
      )}

      {/* Stats Overview */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card className="knirv-card-gradient">
          <CardContent className="p-4">
            <div className="flex items-center space-x-2">
              <Server className="w-5 h-5 text-primary" />
              <div>
                <div className="text-2xl font-bold">{nodeStats.total}</div>
                <div className="text-xs text-muted-foreground">Total Nodes</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card className="knirv-card-gradient">
          <CardContent className="p-4">
            <div className="flex items-center space-x-2">
              <CheckCircle className="w-5 h-5 text-green-500" />
              <div>
                <div className="text-2xl font-bold">{nodeStats.online}</div>
                <div className="text-xs text-muted-foreground">Online</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card className="knirv-card-gradient">
          <CardContent className="p-4">
            <div className="flex items-center space-x-2">
              <AlertCircle className="w-5 h-5 text-red-500" />
              <div>
                <div className="text-2xl font-bold">{nodeStats.offline}</div>
                <div className="text-xs text-muted-foreground">Offline</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card className="knirv-card-gradient">
          <CardContent className="p-4">
            <div className="flex items-center space-x-2">
              <Clock className="w-5 h-5 text-yellow-500" />
              <div>
                <div className="text-2xl font-bold">{nodeStats.maintenance}</div>
                <div className="text-xs text-muted-foreground">Maintenance</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filter Controls */}
      <Card className="knirv-card-gradient">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center space-x-2">
              <Server className="w-5 h-5" />
              <span>{user?.role === 'observer' ? 'My DVE Instances' : 'DVE Nodes'}</span>
            </CardTitle>
            <div className="flex space-x-2">
              <Button
                variant={filter === 'all' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setFilter('all')}
              >
                All
              </Button>
              <Button
                variant={filter === 'online' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setFilter('online')}
              >
                Online
              </Button>
              <Button
                variant={filter === 'offline' ? 'default' : 'outline'}
                size="sm"
                onClick={() => setFilter('offline')}
              >
                Offline
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={refreshNodes}
                disabled={isLoading}
              >
                <Activity className="w-4 h-4 mr-2" />
                Refresh
              </Button>
              {onRentClick && (
                <Button
                  variant="default"
                  size="sm"
                  onClick={onRentClick}
                  className="bg-primary hover:bg-primary/90"
                >
                  <CreditCard className="w-4 h-4 mr-2" />
                  Rent
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Activity className="w-6 h-6 animate-spin mr-2" />
              <span>Loading DVE nodes...</span>
            </div>
          ) : filteredNodes.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              No DVE nodes found
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {filteredNodes.map((node) => (
                <Card key={node.id} className="knirv-card-gradient border hover:border-blue-500/50 transition-all duration-300 group">
                  <CardHeader className="pb-2">
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-sm font-black uppercase tracking-tighter text-slate-200 group-hover:text-blue-400 transition-colors">{node.name}</CardTitle>
                      <div className="flex items-center space-x-2">
                        <Badge 
                          variant="secondary" 
                          className={`${
                            activeNodeId[node.id]
                              ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                              : 'bg-muted text-muted-foreground'
                          } text-[10px] font-black uppercase`}
                        >
                          <div className="flex items-center space-x-1">
                            {activeNodeId[node.id] ? (
                              <CheckCircle className="w-3 h-3" />
                            ) : (
                              <AlertCircle className="w-3 h-3" />
                            )}
                            <span>{activeNodeId[node.id] ? 'online' : 'offline'}</span>
                          </div>
                        </Badge>
                      </div>
                    </div>
                    <CardDescription className="text-[10px] font-mono">
                      ID: {node.id.substring(0, 12)}...
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {/* Performance Metrics */}
                    <div className="space-y-2">
                      <div className="flex justify-between items-center text-[11px] font-mono">
                        <div className="flex items-center space-x-1">
                          <Cpu className="w-3 h-3 text-blue-500" />
                          <span>CPU:</span>
                        </div>
                        <span className="text-slate-300">{node.cpu_usage}%</span>
                      </div>
                      <Progress value={node.cpu_usage} className="h-1 bg-slate-800" />
                      
                      <div className="flex justify-between items-center text-[11px] font-mono">
                        <div className="flex items-center space-x-1">
                          <HardDrive className="w-3 h-3 text-purple-500" />
                          <span>MEM:</span>
                        </div>
                        <span className="text-slate-300">{node.memory_usage}%</span>
                      </div>
                      <Progress value={node.memory_usage} className="h-1 bg-slate-800" />
                    </div>

                    {/* Node Details */}
                    <div className="space-y-1.5 text-[10px] font-bold uppercase tracking-wider bg-slate-950/30 p-2 rounded-lg border border-slate-800/50">
                      <div className="flex justify-between items-center">
                        <div className="flex items-center space-x-1">
                          {getTEEIcon(node.tee_type)}
                          <span className="text-slate-500">TEE:</span>
                        </div>
                        <span className="text-blue-400 font-black">{node.tee_type}</span>
                      </div>
                      
                      <div className="flex justify-between items-center">
                        <span className="text-slate-500">STAKE:</span>
                        <span className="text-slate-300">{node.stake_amount.toLocaleString()} NRN</span>
                      </div>
                      
                      <div className="flex justify-between items-center">
                        <span className="text-slate-500">REPUTATION:</span>
                        <span className={`font-black ${node.reputation_score > 80 ? 'text-green-500' : 'text-yellow-500'}`}>
                          {node.reputation_score}/100
                        </span>
                      </div>
                      
                      {node.location && (
                        <div className="flex justify-between items-center">
                          <div className="flex items-center space-x-1 text-slate-500">
                            <MapPin className="w-3 h-3" />
                            <span>LOC:</span>
                          </div>
                          <span className="text-slate-300">{node.location}</span>
                        </div>
                      )}
                    </div>

                    {/* Actions */}
                    <div className="grid grid-cols-2 gap-2 pt-2">
                      {/* Start/Stop Button */}
                      <Button
                        variant="default"
                        size="sm"
                        className={`text-[10px] font-black uppercase transition-all ${
                          activeNodeId[node.id]
                            ? 'bg-red-900/20 text-red-500 border border-red-500/20 hover:bg-red-600 hover:text-white'
                            : 'bg-blue-600 hover:bg-blue-500 text-white shadow-lg shadow-blue-600/20'
                        }`}
                        onClick={() => handleStartDVE(node)}
                        disabled={provisioningNodes.has(node.id)}
                      >
                        {provisioningNodes.has(node.id) ? (
                          <>
                            <Activity className="w-3 h-3 mr-1 animate-spin" />
                            Provisioning
                          </>
                        ) : activeNodeId[node.id] ? (
                          <>
                            <Square className="w-3 h-3 mr-1 fill-current" />
                            Terminate
                          </>
                        ) : (
                          <>
                            <Play className="w-3 h-3 mr-1 fill-current" />
                            Initialize
                          </>
                        )}
                      </Button>

                      {/* Access Button - Only shown when active */}
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={!activeNodeId[node.id]}
                        className={`text-[10px] font-black uppercase border-blue-600/30 ${activeNodeId[node.id] ? 'text-blue-400 hover:bg-blue-600 hover:text-white' : 'text-slate-600'}`}
                        onClick={() => handleAccessDVE(node)}
                      >
                        <Zap className="w-3 h-3 mr-1" />
                        Access
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* CDE Access Modal (Legacy fallback) */}
      {selectedNode && (
        <CDEAccessModal
          isOpen={cdeOpen}
          onClose={() => setCDEOpen(false)}
          nodeId={selectedNode.id}
          nodeName={selectedNode.name}
          onOpenKNIRVEngine={() => {}}
        />
      )}
    </div>
  );
};

export default DVENodesPanel;
