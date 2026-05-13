"use client";

import React, { useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useToast } from '@/hooks/use-toast';
import {
  Shield,
  Server,
  Clock,
  Coins,
  RefreshCw,
  Plus,
  Square,
  Trash2,
  Eye,
  Settings,
  BarChart3,
  Zap,
  Cpu,
  HardDrive,
  Network,
  CheckCircle,
  AlertTriangle,
  Timer,
  X,
  Lock
} from 'lucide-react';
import { useDVEManagement } from '@/hooks/use-dve-management';
import type { DVECreation, DVECreationRequest, DVEAccessInfo, ResourceLimits } from '@/types/api';
import { useAuth } from '@/lib/auth-context';
import DVEAccessFlow from './dve-access-flow';

interface DVECreationManagementProps {
  isOpen: boolean;
  onClose: () => void;
  onCreated?: () => void;
  defaultTab?: 'creations' | 'create' | 'stats';
}

export default function DVECreationManagement({ isOpen, onClose, onCreated, defaultTab = 'creations' }: DVECreationManagementProps) {
  const { toast } = useToast();
  const { user } = useAuth();
  const [mounted, setMounted] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);

  useEffect(() => { setMounted(true); }, []);
  const [name, setName] = useState('');
  const [stakeAmount, setStakeAmount] = useState(10000);
  const [teeType, setTeeType] = useState('software');
  const [persistent, setPersistent] = useState(true);
  
  const [showAccessModal, setShowAccessModal] = useState(false);
  const [selectedCreation, setSelectedCreation] = useState<DVECreation | null>(null);
  const [accessInfo, setAccessInfo] = useState<DVEAccessInfo | null>(null);

  const {
    creations,
    stats,
    isLoading,
    error,
    createDVE,
    fetchCreations,
    fetchStats,
    getFullAccessInfo,
  } = useDVEManagement();

  const handleCreateDVE = async () => {
    if (!name.trim()) {
      toast({
        title: "Validation Error",
        description: "Please provide a name for your DVE instance",
        variant: "destructive",
      });
      return;
    }

    const resourceLimits: ResourceLimits = {
      cpu_cores: 2,
      memory_gb: 4,
      disk_gb: 20,
      bandwidth_mbps: 100
    };

    const request: DVECreationRequest = {
      name: name.trim(),
      owner_id: user?.user || 'anonymous',
      owner_address: '0x...', // Would come from wallet
      tee_type: teeType,
      tee_attestation: 'direct-attestation-stub',
      stake_amount: stakeAmount,
      capabilities: ['reasoning', 'validation'],
      resource_limits: resourceLimits,
      persistent: persistent
    };

    const result = await createDVE(request);
    if (result) {
      toast({
        title: "Sovereign DVE Created",
        description: `Successfully created ${name} with ${stakeAmount} NRN stake`,
      });
      setName('');
      setStakeAmount(10000);
      onCreated?.();
    }
  };

  const handleAccessDVE = async (creation: DVECreation) => {
    try {
      const info = await getFullAccessInfo(creation.id);
      if (info) {
        setSelectedCreation(creation);
        setAccessInfo(info);
        setShowAccessModal(true);
      } else {
        toast({
          title: "Access Information Unavailable",
          description: "Could not retrieve access information for this DVE.",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Access Failed",
        description: "Failed to retrieve access information. Please try again.",
        variant: "destructive",
      });
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "active":
        return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" /> Active</Badge>;
      case "pending":
        return <Badge className="bg-yellow-500"><Timer className="w-3 h-3 mr-1" /> Pending</Badge>;
      case "suspended":
        return <Badge className="bg-red-500"><AlertTriangle className="w-3 h-3 mr-1" /> Suspended</Badge>;
      default:
        return <Badge className="bg-gray-500">{status}</Badge>;
    }
  };

  if (!isOpen) return null;

  const modalContent = (
    <div
      className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-[99999] p-8"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="w-full max-w-7xl max-h-[90vh] bg-background border shadow-2xl rounded-lg overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b bg-gradient-to-r from-blue-900/20 to-purple-900/20">
          <div className="flex items-center space-x-4">
            <div className="w-12 h-12 bg-gradient-to-r from-blue-600 to-purple-600 rounded-lg flex items-center justify-center">
              <Shield className="w-6 h-6 text-white" />
            </div>
            <div>
              <h2 className="text-2xl font-bold tracking-tight">Sovereign DVE Management</h2>
              <p className="text-sm text-muted-foreground">
                Stake NRN to create and manage your Deterministic Validation Environments
              </p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <Button variant="outline" size="sm" onClick={() => { fetchCreations(); fetchStats(); }} disabled={isLoading}>
              <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
              Sync
            </Button>
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="w-6 h-6" />
            </Button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-6">
          <Tabs defaultValue={defaultTab} className="space-y-4">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="creations">My DVE Instances</TabsTrigger>
              <TabsTrigger value="create">New Sovereign DVE</TabsTrigger>
              <TabsTrigger value="stats">Network Stats</TabsTrigger>
            </TabsList>

            <TabsContent value="creations" className="space-y-4">
              <div className="grid grid-cols-1 gap-4">
                {creations.map((creation) => (
                  <Card key={creation.id} className="knirv-card-gradient border hover:border-blue-500/50 transition-interactive">
                    <CardContent className="p-6">
                      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            <h3 className="text-lg font-bold">{creation.name}</h3>
                            {getStatusBadge(creation.status)}
                            {creation.persistent && <Badge variant="outline" className="border-blue-500/50 text-blue-400"><Lock className="w-3 h-3 mr-1" /> Sovereign</Badge>}
                          </div>
                          <p className="text-xs font-mono text-muted-foreground">ID: {creation.id}</p>
                          <div className="flex items-center gap-4 mt-2">
                            <div className="flex items-center text-xs text-muted-foreground">
                              <Coins className="w-3 h-3 mr-1" /> {creation.stake_amount.toLocaleString()} NRN Staked
                            </div>
                            <div className="flex items-center text-xs text-muted-foreground">
                              <Cpu className="w-3 h-3 mr-1" /> {creation.tee_type.toUpperCase()}
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Button 
                            variant="default" 
                            size="sm" 
                            className="bg-blue-600 hover:bg-blue-500"
                            onClick={() => handleAccessDVE(creation)}
                          >
                            <Zap className="w-4 h-4 mr-2" />
                            Access Fabric
                          </Button>
                          <Button variant="outline" size="sm">
                            <Settings className="w-4 h-4" />
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
                {creations.length === 0 && (
                  <div className="text-center py-12 border-2 border-dashed rounded-xl border-slate-800">
                    <Server className="w-12 h-12 text-slate-700 mx-auto mb-4" />
                    <h3 className="text-lg font-medium text-slate-300">No DVE Instances Found</h3>
                    <p className="text-slate-500 mb-6 text-sm">You haven't created any sovereign DVE nodes yet.</p>
                    <Button onClick={() => document.querySelector('[value="create"]')?.dispatchEvent(new MouseEvent('click', {bubbles: true}))}>
                      <Plus className="w-4 h-4 mr-2" />
                      Create Your First DVE
                    </Button>
                  </div>
                )}
              </div>
            </TabsContent>

            <TabsContent value="create">
              <Card className="max-w-2xl mx-auto knirv-card-gradient">
                <CardHeader>
                  <CardTitle>Initialize Sovereign DVE</CardTitle>
                  <CardDescription>
                    Staking NRN creates a permanent, secure validation environment linked to your identity.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="space-y-2">
                    <Label>DVE Instance Name</Label>
                    <Input 
                      placeholder="e.g. My-Secure-Reasoning-Node" 
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>TEE Architecture</Label>
                      <Select value={teeType} onValueChange={setTeeType}>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="software">Software (Simulated)</SelectItem>
                          <SelectItem value="sgx">Intel SGX</SelectItem>
                          <SelectItem value="sev-snp">AMD SEV-SNP</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>NRN Stake Amount</Label>
                      <Input 
                        type="number" 
                        value={stakeAmount}
                        onChange={(e) => setStakeAmount(parseInt(e.target.value) || 0)}
                      />
                    </div>
                  </div>

                  <div className="p-4 bg-blue-900/10 border border-blue-500/20 rounded-lg space-y-3">
                    <h4 className="text-sm font-bold flex items-center text-blue-400">
                      <Zap className="w-4 h-4 mr-2" /> Resource Allocation
                    </h4>
                    <div className="grid grid-cols-3 gap-2 text-xs">
                      <div className="flex flex-col">
                        <span className="text-slate-500">CPU</span>
                        <span className="font-mono">2 Cores</span>
                      </div>
                      <div className="flex flex-col">
                        <span className="text-slate-500">RAM</span>
                        <span className="font-mono">4 GB</span>
                      </div>
                      <div className="flex flex-col">
                        <span className="text-slate-500">Storage</span>
                        <span className="font-mono">20 GB</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center space-x-2 bg-slate-900/50 p-3 rounded border border-slate-800">
                    <input 
                      type="checkbox" 
                      id="persistent" 
                      checked={persistent} 
                      onChange={(e) => setPersistent(e.target.checked)}
                      className="rounded border-slate-700 bg-slate-800 text-blue-600"
                    />
                    <Label htmlFor="persistent" className="text-xs cursor-pointer">
                      Enable Sovereign Persistence (Stake remains locked for permanent node ownership)
                    </Label>
                  </div>

                  <Button className="w-full bg-blue-600 hover:bg-blue-500 h-12 text-lg font-bold" onClick={handleCreateDVE} disabled={isLoading}>
                    {isLoading ? <Activity className="animate-spin mr-2" /> : <Plus className="w-5 h-5 mr-2" />}
                    Initialize Sovereign Node
                  </Button>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="stats">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <Card className="knirv-card-gradient text-center p-6">
                  <p className="text-3xl font-bold text-blue-400">{stats?.total_creations || 0}</p>
                  <p className="text-xs uppercase tracking-widest text-slate-500 mt-2">Total Creations</p>
                </Card>
                <Card className="knirv-card-gradient text-center p-6">
                  <p className="text-3xl font-bold text-green-400">{stats?.active_creations || 0}</p>
                  <p className="text-xs uppercase tracking-widest text-slate-500 mt-2">Active Nodes</p>
                </Card>
                <Card className="knirv-card-gradient text-center p-6">
                  <p className="text-3xl font-bold text-purple-400">{(stats?.total_stake || 0).toLocaleString()}</p>
                  <p className="text-xs uppercase tracking-widest text-slate-500 mt-2">Total NRN Staked</p>
                </Card>
                <Card className="knirv-card-gradient text-center p-6">
                  <p className="text-3xl font-bold text-orange-400">{(stats?.average_stake || 0).toLocaleString()}</p>
                  <p className="text-xs uppercase tracking-widest text-slate-500 mt-2">Avg Stake</p>
                </Card>
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </div>

      {/* Access Modal */}
      {showAccessModal && selectedCreation && accessInfo && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-md flex items-center justify-center z-[100000] p-4">
          <div className="bg-slate-900 border-2 border-blue-500/30 rounded-xl shadow-2xl max-w-6xl w-full max-h-[90vh] flex flex-col">
            <div className="p-6 border-b border-slate-800 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="p-2 bg-blue-600/20 rounded-lg">
                  <Zap className="w-6 h-6 text-blue-400" />
                </div>
                <div>
                  <h3 className="text-xl font-bold">DVE Workspace: {selectedCreation.name}</h3>
                  <p className="text-sm text-slate-500">Secure Active Memory Fabric Access</p>
                </div>
              </div>
              <Button variant="ghost" size="sm" onClick={() => setShowAccessModal(false)}>
                <X className="w-6 h-6" />
              </Button>
            </div>
            <div className="flex-1 overflow-auto p-6">
              <DVEAccessFlow
                creationId={selectedCreation.id}
                accessInfo={accessInfo}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );

  if (!mounted) return null;
  return createPortal(modalContent, document.body);
}

// Minimal Activity icon since it was missing from imports
function Activity({ className }: { className?: string }) {
  return (
    <svg 
      xmlns="http://www.w3.org/2000/svg" 
      width="24" 
      height="24" 
      viewBox="0 0 24 24" 
      fill="none" 
      stroke="currentColor" 
      strokeWidth="2" 
      strokeLinecap="round" 
      strokeLinejoin="round" 
      className={className}
    >
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  );
}
