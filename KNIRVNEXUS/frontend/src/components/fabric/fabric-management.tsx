"use client";

import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useToast } from '@/hooks/use-toast';
import { 
  Bot, 
  Play, 
  Square, 
  RotateCcw, 
  Trash2, 
  RefreshCw, 
  Upload,
  Activity,
  CheckCircle,
  AlertTriangle,
  Clock,
  Eye,
  Settings,
  BarChart3,
  FileText,
  Zap,
  Cpu,
  HardDrive,
  Network,
  Layers
} from 'lucide-react';
import { useFabricManagement, FabricFilter } from '@/hooks/use-fabric-management';
import { useNexusEvents } from '@/hooks/use-nexus-events';
import type { Fabric, FabricAction } from '@/types/api';

interface FabricManagementProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function FabricManagement({ isOpen, onClose }: FabricManagementProps) {
  const { toast } = useToast();
  const [selectedFabric, setSelectedFabric] = useState<Fabric | null>(null);
  const [selectedStatus, setSelectedStatus] = useState<string>('all');
  const [selectedType, setSelectedType] = useState<string>('all');

  const {
    fabrics,
    summary,
    isLoading,
    error,
    deleteFabric,
    refreshAll
  } = useFabricManagement();

  const {
    events: nexusEvents,
    isLoading: nexusLoading,
    error: nexusError
  } = useNexusEvents();

  const handleFabricAction = async (fabricId: string, action: FabricAction['action']) => {
    // TODO: Implement fabric actions when backend API is available
    toast({
      title: "Fabric Action",
      description: `${action} action for fabric item ${fabricId} - Feature coming soon`,
      variant: "default",
    });
  };

  const handleDeleteFabric = async (fabricId: string, fabricName: string) => {
    const confirmed = window.confirm(`Are you sure you want to delete fabric item "${fabricName}"?`);
    if (!confirmed) return;

    const success = await deleteFabric(fabricId);
    if (success) {
      toast({
        title: "Fabric Deleted",
        description: `Successfully deleted fabric item "${fabricName}"`,
      });
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "running":
        return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" /> Running</Badge>;
      case "deployed":
        return <Badge className="bg-blue-500"><Zap className="w-3 h-3 mr-1" /> Deployed</Badge>;
      case "stopped":
        return <Badge className="bg-gray-500"><Square className="w-3 h-3 mr-1" /> Stopped</Badge>;
      case "error":
        return <Badge className="bg-red-500"><AlertTriangle className="w-3 h-3 mr-1" /> Error</Badge>;
      case "uploaded":
        return <Badge className="bg-yellow-500"><Upload className="w-3 h-3 mr-1" /> Uploaded</Badge>;
      case "archived":
        return <Badge className="bg-gray-400"><Clock className="w-3 h-3 mr-1" /> Archived</Badge>;
      default:
        return <Badge className="bg-gray-500">{status}</Badge>;
    }
  };

  const getTypeBadge = (type: string) => {
    const colors: Record<string, string> = {
      'WASM': 'bg-purple-500',
      'LoRA': 'bg-blue-500',
      'CodeT5': 'bg-green-500',
      'SEAL': 'bg-orange-500',
      'NRN': 'bg-pink-500'
    };
    return <Badge className={colors[type] || 'bg-gray-500'}>{type}</Badge>;
  };

  const filteredFabrics = fabrics.filter(fabric => {
    const statusMatch = selectedStatus === 'all' || fabric.status === selectedStatus;
    const typeMatch = selectedType === 'all' || fabric.type === selectedType;
    return statusMatch && typeMatch;
  });

  const formatFileSize = (bytes: number) => {
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    if (bytes === 0) return '0 Bytes';
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-8 pt-16 pb-16">
      <div className="w-full max-w-7xl max-h-[90vh] bg-background border shadow-2xl rounded-lg overflow-hidden">
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b bg-gradient-to-r from-primary/10 to-secondary/10">
            <div className="flex items-center space-x-4">
              <div className="w-12 h-12 bg-gradient-to-r from-primary to-secondary rounded-lg flex items-center justify-center">
                <Layers className="w-6 h-6 text-white" />
              </div>
              <div>
                <h2 className="text-2xl font-bold">Fabric Management</h2>
                <p className="text-muted-foreground">
                  Manage Agentic Memory Fabric and Validation Guardrails
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" onClick={refreshAll} disabled={isLoading}>
                <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Button variant="ghost" size="sm" onClick={onClose}>
                ×
              </Button>
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-auto p-6">
            {error && (
              <div className="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg">
                <p className="text-sm text-destructive">{error}</p>
              </div>
            )}

            <Tabs defaultValue="overview" className="space-y-4">
              <TabsList className="grid w-full grid-cols-5">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="fabric">Fabric Items</TabsTrigger>
                <TabsTrigger value="memory">Memory Fabric</TabsTrigger>
                <TabsTrigger value="monitoring">Monitoring</TabsTrigger>
                <TabsTrigger value="settings">Settings</TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="space-y-4">
                {/* Fabric Summary */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Activity className="h-5 w-5" />
                      Fabric Summary
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    {summary ? (
                      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
                        <div className="text-center">
                          <p className="text-2xl font-bold text-blue-500">{summary.total_models}</p>
                          <p className="text-sm text-muted-foreground">Total</p>
                        </div>
                        <div className="text-center">
                          <p className="text-2xl font-bold text-green-500">{summary.running_models}</p>
                          <p className="text-sm text-muted-foreground">Running</p>
                        </div>
                        <div className="text-center">
                          <p className="text-2xl font-bold text-blue-400">{summary.deployed_models}</p>
                          <p className="text-sm text-muted-foreground">Deployed</p>
                        </div>
                        <div className="text-center">
                          <p className="text-2xl font-bold text-gray-500">{summary.stopped_models}</p>
                          <p className="text-sm text-muted-foreground">Stopped</p>
                        </div>
                        <div className="text-center">
                          <p className="text-2xl font-bold text-red-500">{summary.error_models}</p>
                          <p className="text-sm text-muted-foreground">Error</p>
                        </div>
                        <div className="text-center">
                          <p className="text-2xl font-bold text-yellow-500">{summary.uploaded_models}</p>
                          <p className="text-sm text-muted-foreground">Uploaded</p>
                        </div>
                      </div>
                    ) : (
                      <p className="text-muted-foreground">Loading fabric summary...</p>
                    )}
                  </CardContent>
                </Card>

                {/* Quick Actions */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle>Quick Actions</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <Button className="h-20 flex flex-col items-center justify-center">
                        <Upload className="w-6 h-6 mb-2" />
                        Upload Fabric Item
                      </Button>
                      <Button variant="outline" className="h-20 flex flex-col items-center justify-center">
                        <Settings className="w-6 h-6 mb-2" />
                        Configure Runtime
                      </Button>
                      <Button variant="outline" className="h-20 flex flex-col items-center justify-center">
                        <BarChart3 className="w-6 h-6 mb-2" />
                        View Metrics
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="fabric" className="space-y-4">
                {/* Filters and Actions */}
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-4">
                    <div>
                      <Label htmlFor="status-filter">Status</Label>
                      <Select value={selectedStatus} onValueChange={setSelectedStatus}>
                        <SelectTrigger className="w-40">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All Status</SelectItem>
                          <SelectItem value="running">Running</SelectItem>
                          <SelectItem value="deployed">Deployed</SelectItem>
                          <SelectItem value="stopped">Stopped</SelectItem>
                          <SelectItem value="error">Error</SelectItem>
                          <SelectItem value="uploaded">Uploaded</SelectItem>
                          <SelectItem value="archived">Archived</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div>
                      <Label htmlFor="type-filter">Type</Label>
                      <Select value={selectedType} onValueChange={setSelectedType}>
                        <SelectTrigger className="w-32">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All Types</SelectItem>
                          <SelectItem value="WASM">WASM</SelectItem>
                          <SelectItem value="LoRA">LoRA</SelectItem>
                          <SelectItem value="CodeT5">CodeT5</SelectItem>
                          <SelectItem value="SEAL">SEAL</SelectItem>
                          <SelectItem value="NRN">NRN</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <Button>
                    <Upload className="w-4 h-4 mr-2" />
                    Upload Fabric Item
                  </Button>
                </div>

                {/* Fabrics List */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle>Fabric Items ({filteredFabrics.length})</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      {filteredFabrics.map((fabric) => (
                        <div key={fabric.id} className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50">
                          <div className="flex-1 grid grid-cols-1 md:grid-cols-5 gap-4">
                            <div>
                              <p className="font-medium">{fabric.name}</p>
                              <p className="text-sm text-muted-foreground">{fabric.description}</p>
                              <p className="text-xs text-muted-foreground">v{fabric.version} by {fabric.author}</p>
                            </div>
                            <div className="flex flex-col space-y-1">
                              {getTypeBadge(fabric.type)}
                              {getStatusBadge(fabric.status)}
                            </div>
                            <div>
                              <p className="text-sm font-mono">{formatFileSize(fabric.file_size)}</p>
                              <p className="text-xs text-muted-foreground">{fabric.capabilities?.length || 0} capabilities</p>
                            </div>
                            <div>
                              <p className="text-xs text-muted-foreground">Uploaded</p>
                              <p className="text-sm">{new Date(fabric.uploaded_at).toLocaleDateString()}</p>
                              {fabric.last_activity && (
                                <p className="text-xs text-muted-foreground">
                                  Active: {new Date(fabric.last_activity).toLocaleDateString()}
                                </p>
                              )}
                            </div>
                            <div className="flex items-center space-x-1">
                              {fabric.status === 'uploaded' && (
                                <Button 
                                  variant="outline" 
                                  size="sm" 
                                  onClick={() => handleFabricAction(fabric.id, 'deploy')}
                                  disabled={isLoading}
                                >
                                  <Zap className="w-3 h-3" />
                                </Button>
                              )}
                              {(fabric.status === 'deployed' || fabric.status === 'stopped') && (
                                <Button 
                                  variant="outline" 
                                  size="sm" 
                                  onClick={() => handleFabricAction(fabric.id, 'start')}
                                  disabled={isLoading}
                                >
                                  <Play className="w-3 h-3" />
                                </Button>
                              )}
                              {fabric.status === 'running' && (
                                <Button 
                                  variant="outline" 
                                  size="sm" 
                                  onClick={() => handleFabricAction(fabric.id, 'stop')}
                                  disabled={isLoading}
                                >
                                  <Square className="w-3 h-3" />
                                </Button>
                              )}
                              <Button 
                                variant="outline" 
                                size="sm" 
                                onClick={() => setSelectedFabric(fabric)}
                              >
                                <Eye className="w-3 h-3" />
                              </Button>
                              <Button 
                                variant="outline" 
                                size="sm" 
                                onClick={() => handleDeleteFabric(fabric.id, fabric.name)}
                                disabled={isLoading}
                              >
                                <Trash2 className="w-3 h-3" />
                              </Button>
                            </div>
                          </div>
                        </div>
                      ))}
                      {filteredFabrics.length === 0 && (
                        <div className="text-center py-8 text-muted-foreground">
                          No fabric items found matching the current filters.
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="memory" className="space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Zap className="h-5 w-5 text-yellow-500" />
                      Active Memory Fabric (Arrow Flight)
                    </CardTitle>
                    <CardDescription>
                      Real-time Intent-Action Correlation via eBPF Guardian
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      {nexusLoading ? (
                        <p className="text-center py-4">Loading memory fabric events...</p>
                      ) : nexusEvents.length === 0 ? (
                        <div className="text-center py-12 border-2 border-dashed rounded-lg">
                          <Bot className="w-12 h-12 mx-auto mb-4 text-muted-foreground" />
                          <p className="text-muted-foreground">No active agent sessions detected</p>
                          <p className="text-xs text-muted-foreground mt-1">Deploy an agent to see real-time correlation</p>
                        </div>
                      ) : (
                        <div className="rounded-md border">
                          <table className="min-w-full divide-y divide-border">
                            <thead className="bg-muted/50">
                              <tr>
                                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider">Timestamp</th>
                                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider">Agent ID</th>
                                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider">Intent</th>
                                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider">Observed Action</th>
                                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider">Status</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-border">
                              {nexusEvents.map((event, i) => (
                                <tr key={i} className="hover:bg-muted/30 transition-colors">
                                  <td className="px-4 py-3 text-xs font-mono">
                                    {new Date(event.timestamp / 1000000).toLocaleTimeString()}
                                  </td>
                                  <td className="px-4 py-3 text-sm font-medium">{event.agent_id}</td>
                                  <td className="px-4 py-3 text-sm">{event.intent}</td>
                                  <td className="px-4 py-3 text-sm font-mono text-blue-400">{event.observed_action}</td>
                                  <td className="px-4 py-3 text-sm">
                                    {event.verified ? (
                                      <Badge className="bg-green-500/20 text-green-500 border-green-500/50">Verified</Badge>
                                    ) : (
                                      <Badge className="bg-red-500/20 text-red-500 border-red-500/50">Anomaly</Badge>
                                    )}
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Card className="knirv-card-gradient">
                    <CardHeader>
                      <CardTitle className="text-sm">Zero-Copy Handover Status</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="flex items-center justify-between">
                        <p className="text-xs text-muted-foreground">Arrow Flight RPC Port</p>
                        <Badge variant="outline">50051</Badge>
                      </div>
                      <div className="mt-2 flex items-center justify-between">
                        <p className="text-xs text-muted-foreground">Active Streams</p>
                        <p className="text-sm font-bold">{nexusEvents.length > 0 ? 1 : 0}</p>
                      </div>
                    </CardContent>
                  </Card>
                  <Card className="knirv-card-gradient">
                    <CardHeader>
                      <CardTitle className="text-sm">eBPF Kernel Probes</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="flex items-center justify-between">
                        <p className="text-xs text-muted-foreground">Kprobe Status</p>
                        <Badge className="bg-green-500">Active</Badge>
                      </div>
                      <div className="mt-2 flex items-center justify-between">
                        <p className="text-xs text-muted-foreground">Intercepted Syscalls</p>
                        <p className="text-sm font-bold">sys_connect</p>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <TabsContent value="monitoring" className="space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <BarChart3 className="h-5 w-5" />
                      Fabric Monitoring
                    </CardTitle>
                    <CardDescription>
                      Real-time monitoring and metrics for running fabric items
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div className="p-4 border rounded-lg">
                          <div className="flex items-center gap-2 mb-2">
                            <Cpu className="w-4 h-4" />
                            <span className="text-sm font-medium">CPU Usage</span>
                          </div>
                          <p className="text-2xl font-bold">--</p>
                          <p className="text-xs text-muted-foreground">Average across all items</p>
                        </div>
                        <div className="p-4 border rounded-lg">
                          <div className="flex items-center gap-2 mb-2">
                            <HardDrive className="w-4 h-4" />
                            <span className="text-sm font-medium">Memory Usage</span>
                          </div>
                          <p className="text-2xl font-bold">--</p>
                          <p className="text-xs text-muted-foreground">Total memory allocated</p>
                        </div>
                        <div className="p-4 border rounded-lg">
                          <div className="flex items-center gap-2 mb-2">
                            <Network className="w-4 h-4" />
                            <span className="text-sm font-medium">Network I/O</span>
                          </div>
                          <p className="text-2xl font-bold">--</p>
                          <p className="text-xs text-muted-foreground">Combined throughput</p>
                        </div>
                      </div>
                      <div className="p-4 border rounded-lg bg-muted/50">
                        <p className="text-sm text-muted-foreground">
                          Real-time monitoring data will be displayed here when fabric items are running.
                          Select an item from the Fabric Items tab to view detailed metrics.
                        </p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="settings" className="space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Settings className="h-5 w-5" />
                      Fabric Runtime Configuration
                    </CardTitle>
                    <CardDescription>
                      Configure fabric runtime settings and resource limits
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      <div className="p-4 border rounded-lg bg-muted/50">
                        <p className="text-sm text-muted-foreground">
                          Fabric runtime configuration is managed through the backend configuration files.
                          Contact your system administrator to modify runtime settings.
                        </p>
                      </div>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                          <Label>Max Items</Label>
                          <p className="font-mono">100</p>
                        </div>
                        <div>
                          <Label>Max Instances per Item</Label>
                          <p className="font-mono">10</p>
                        </div>
                        <div>
                          <Label>Default Memory Limit</Label>
                          <p className="font-mono">512 MB</p>
                        </div>
                        <div>
                          <Label>Default CPU Limit</Label>
                          <p className="font-mono">50%</p>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </div>
    </div>
  );
}
