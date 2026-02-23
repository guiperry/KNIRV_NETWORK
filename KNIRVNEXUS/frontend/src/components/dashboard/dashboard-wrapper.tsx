'use client';

import React, { useState } from 'react';
import { useTEESecurity } from "@/hooks/use-tee-security";
import { useCognitiveEngine } from "@/hooks/use-cognitive-engine";
import { useAuth, ROLES } from '@/lib/auth-context';
import { LoginForm } from '@/components/auth/login-form';
import { UserProfile } from '@/components/auth/user-profile';
import { RoleGuard, DVEAccess, ValidationAccess, SystemAccess } from '@/components/auth/role-guard';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { NetworkAccessModal } from '@/components/cde/cde-access-modal';
import { ActiveMemoryAccessModal, KNIRVGraphAccessModal, KNIRVChainAccessModal, P2PTransportAccessModal } from '@/components/cde/access-panels';
import CDEPanel from './cde-panel'; // Modular CDE Panel
import { KNIRVEngineModal } from '@/components/knirvengine/knirvengine-modal';
import { CognitiveEnginePanel } from '@/components/dashboard/cognitive-engine-panel';
import { DVENodesPanel } from '@/components/dashboard/dve-nodes-panel';
import { FinancialComplianceDashboard } from '@/components/dashboard/financial-compliance-dashboard';
import { useOnboarding } from "@/contexts/onboarding-context";
import OnboardingGuide from "@/components/onboarding/onboarding-guide";
import type { DVENode } from '@/types/api';
import {
  Shield,
  Server,
  Activity,
  Eye,
  User,
  Settings,
  BarChart3,
  Lock,
  Unlock,
  Network,
  Cpu,
  HardDrive,
  Wifi,
  Globe,
  Database,
  Monitor,
  Zap,
  Download,
  Share2,
  Scale,
  CheckCircle,
  AlertTriangle,
  Clock,
  Brain
} from 'lucide-react';

interface DashboardWrapperProps {
  children: React.ReactNode;
  onRentDVE?: () => void;
  useModularCDE: boolean;
  setUseModularCDE: (value: boolean) => void;
}

export function DashboardWrapper({ children, onRentDVE, useModularCDE, setUseModularCDE }: DashboardWrapperProps) {
  const { user, isLoading } = useAuth();
  const { state: onboardingState, updateState: updateOnboardingState, completeOnboarding, resetOnboarding } = useOnboarding();
  const { securityStatus: teeSecurityStatus, isLoading: teeLoading } = useTEESecurity();
  const { cognitiveEngine, isLoading: cognitiveLoading } = useCognitiveEngine();
  const [cdeModalOpen, setCdeModalOpen] = useState(false);
  const [activeMemoryOpen, setActiveMemoryOpen] = useState(false);
  const [knirvGraphOpen, setKnirvGraphOpen] = useState(false);
  const [knirvChainOpen, setKnirvChainOpen] = useState(false);
  const [p2pTransportOpen, setP2PTransportOpen] = useState(false);
  const [knirvEngineModalOpen, setKnirvEngineModalOpen] = useState(false);
  const [selectedNode, setSelectedNode] = useState<DVENode | null>(null);
  const [showAdminAccess, setShowAdminAccess] = useState(false);

  const handleNodeAccess = (node: DVENode) => {
    setSelectedNode(node);
    setUseModularCDE(true);
    setCdeModalOpen(true);
  };

  const handleOpenKNIRVEngine = () => {
    setCdeModalOpen(false);
    setKnirvEngineModalOpen(true);
  };

  const handleOnboardingComplete = (config: any) => {
    updateOnboardingState({
      dataWalletConfig: {
        walletName: config.walletName,
        fabricInputs: config.fabricInputs,
        guardrails: config.guardrails,
        connectionData: config.connectionData,
        completedConnections: config.completedConnections
      },
      privacyPreferences: config.privacySettings,
      isOnboardingComplete: true,
      currentStep: 'hosting'
    });
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "online":
      case "active":
      case "completed":
      case "verified":
        return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" /> {status}</Badge>;
      case "offline":
      case "failed":
        return <Badge className="bg-red-500"><AlertTriangle className="w-3 h-3 mr-1" /> {status}</Badge>;
      case "maintenance":
      case "pending":
        return <Badge className="bg-yellow-500"><Clock className="w-3 h-3 mr-1" /> {status}</Badge>;
      case "running":
      case "learning":
        return <Badge className="bg-blue-500"><Activity className="w-3 h-3 mr-1" /> {status}</Badge>;
      default:
        return <Badge>{status}</Badge>;
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div data-testid="loading-spinner" className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    );
  }

  if (!user?.authenticated) {
    return <LoginForm />;
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background to-muted">
      {/* Header with user profile */}
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                <Shield className="w-8 h-8 text-primary" />
                <div>
                  <h1 className="text-2xl font-bold">KNIRV NEXUS</h1>
                  <p className="text-sm text-muted-foreground">Deterministic Validation Environment</p>
                </div>
              </div>
            </div>
            
            <div className="flex items-center space-x-4">

              <div className="flex items-center space-x-2 text-sm">
                <span className="text-muted-foreground">Welcome,</span>
                <span className="font-medium">{user.user}</span>
                <Badge variant={user.role === 'admin' ? 'destructive' : user.role === 'validator' ? 'secondary' : 'outline'}>
                  {ROLES[user.role]?.displayName || user.role.toUpperCase()}
                </Badge>
              </div>
              <UserProfile />
            </div>
          </div>
        </div>
      </header>

      {/* Role-based navigation */}
      <nav className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto px-4">
          <Tabs defaultValue="overview" className="w-full">
            <TabsList className="inline-flex h-12 w-full">
              <TabsTrigger value="overview" className="flex items-center space-x-2">
                <BarChart3 className="w-4 h-4" />
                <span>Overview</span>
              </TabsTrigger>

              {user?.nexus_access?.includes('compliance:read') && (
                <TabsTrigger value="compliance" className="flex items-center space-x-2">
                  <Scale className="w-4 h-4" />
                  <span>Compliance</span>
                </TabsTrigger>
              )}

              <SystemAccess operation="read" showError={false}>
                <TabsTrigger value="system" className="flex items-center space-x-2">
                  <Settings className="w-4 h-4" />
                  <span>Network & Resources</span>
                </TabsTrigger>
              </SystemAccess>

              <RoleGuard allowedRoles={['admin']} showError={false}>
                <button
                  onClick={() => setShowAdminAccess(true)}
                  className="inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-md border border-white/20 px-2 py-1 text-sm font-medium whitespace-nowrap transition-all text-foreground hover:bg-blue-500/20 hover:border-blue-400/50 disabled:pointer-events-none disabled:opacity-50"
                >
                  <User className="w-4 h-4" />
                  <span>Admin Access</span>
                </button>
              </RoleGuard>

              <TabsTrigger value="profile" className="flex items-center space-x-2">
                <Eye className="w-4 h-4" />
                <span>Profile</span>
              </TabsTrigger>
            </TabsList>
            
            <div className="py-6">
              <TabsContent value="overview">
                {!onboardingState.isOnboardingComplete ? (
                  <div className="rounded-xl overflow-hidden border border-blue-500/30 shadow-[0_0_20px_rgba(59,130,246,0.1)]">
                    <OnboardingGuide 
                      onComplete={handleOnboardingComplete} 
                      onReset={resetOnboarding}
                    />
                  </div>
                ) : (
                  children
                )}
              </TabsContent>
              
              {user?.nexus_access?.includes('compliance:read') && (
                <TabsContent value="compliance">
                  <div className="space-y-6">
                    <div>
                      <h2 className="text-2xl font-bold">Financial Compliance Dashboard</h2>
                      <p className="text-muted-foreground">
                        Deterministic Validation of Financial AI Agents - Evidence Packs, Fidelity Scoring, and Regulatory Compliance
                      </p>
                    </div>
                    <FinancialComplianceDashboard />
                  </div>
                </TabsContent>
              )}
              

              
              <TabsContent value="system">
                <SystemAccess operation="read">
                  <div className="space-y-6">
                    <div className="flex items-center justify-between">
                      <div>
                        <h2 className="text-2xl font-bold">Network & Resource Explorer</h2>
                        <p className="text-muted-foreground">
                          Monitor network topology, resource allocation, and system performance across the KNIRV ecosystem.
                        </p>
                      </div>
                      <Button variant="outline" className="flex items-center space-x-2">
                        <Download className="w-4 h-4" />
                        <span>Export Report</span>
                      </Button>
                    </div>

                    {/* Network Overview Cards */}
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                      <Card 
                        className="knirv-card-gradient border hover:border-blue-500/50 transition-all group cursor-pointer"
                        onClick={() => setActiveMemoryOpen(true)}
                      >
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                          <div className="flex items-center space-x-2">
                            <CardTitle className="text-sm font-medium">Active Memory (KNIRVBASE)</CardTitle>
                            <div className="w-2 h-2 rounded-full bg-blue-500 animate-pulse shadow-[0_0_8px_rgba(59,130,246,0.6)]" title="Status: Online" />
                          </div>
                          <Database className="h-4 w-4 text-muted-foreground group-hover:text-blue-400 transition-colors" />
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">Encrypted</div>
                          <p className="text-xs text-muted-foreground mb-3">
                            PQC Markdown persistence active
                          </p>
                          <div className="bg-black/40 rounded p-2 font-mono text-[9px] text-blue-400/80 h-16 overflow-hidden">
                            <div className="line-clamp-1">[10:45:21] Kyber-768 Handshake OK</div>
                            <div className="line-clamp-1">[10:45:22] Committing .md fabric slice...</div>
                            <div className="line-clamp-1">[10:45:23] Dilithium-3 Signature Valid</div>
                          </div>
                        </CardContent>
                      </Card>

                      <Card 
                        className="knirv-card-gradient border hover:border-blue-500/50 transition-all group cursor-pointer"
                        onClick={() => setKnirvGraphOpen(true)}
                      >
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                          <div className="flex items-center space-x-2">
                            <CardTitle className="text-sm font-medium">Reasoning Graph (KNIRVGRAPH)</CardTitle>
                            <div className="w-2 h-2 rounded-full bg-blue-500 animate-pulse shadow-[0_0_8px_rgba(59,130,246,0.6)]" title="Status: Synced" />
                          </div>
                          <Network className="h-4 w-4 text-muted-foreground group-hover:text-blue-400 transition-colors" />
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">142 Traces</div>
                          <p className="text-xs text-muted-foreground mb-3">
                            Context records in .md fabric
                          </p>
                          <div className="bg-black/40 rounded p-2 font-mono text-[9px] text-blue-400/80 h-16 overflow-hidden">
                            <div className="line-clamp-1">[10:45:18] Querying NRV-8472...</div>
                            <div className="line-clamp-1">[10:45:20] Edge verified via Consensus</div>
                            <div className="line-clamp-1">[10:45:23] Graph re-indexing complete</div>
                          </div>
                        </CardContent>
                      </Card>

                      <Card 
                        className="knirv-card-gradient border hover:border-blue-500/50 transition-all group cursor-pointer"
                        onClick={() => setKnirvChainOpen(true)}
                      >
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                          <div className="flex items-center space-x-2">
                            <CardTitle className="text-sm font-medium">Solution Vault (KNIRVCHAIN)</CardTitle>
                            <div className="w-2 h-2 rounded-full bg-blue-500 animate-pulse shadow-[0_0_8px_rgba(59,130,246,0.6)]" title="Status: Minting" />
                          </div>
                          <Lock className="h-4 w-4 text-muted-foreground group-hover:text-amber-400 transition-colors" />
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">28 Nodes</div>
                          <p className="text-xs text-muted-foreground mb-3">
                            Verifiable executable logic
                          </p>
                          <div className="bg-black/40 rounded p-2 font-mono text-[9px] text-amber-400/80 h-16 overflow-hidden">
                            <div className="line-clamp-1">[10:45:15] Validating Proof-of-Skill...</div>
                            <div className="line-clamp-1">[10:45:19] New block minted: 0x7a8b...</div>
                            <div className="line-clamp-1">[10:45:23] Distributing rewards...</div>
                          </div>
                        </CardContent>
                      </Card>

                      <Card 
                        className="knirv-card-gradient border hover:border-blue-500/50 transition-all group cursor-pointer"
                        onClick={() => setP2PTransportOpen(true)}
                      >
                        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                          <div className="flex items-center space-x-2">
                            <CardTitle className="text-sm font-medium">P2P Transport</CardTitle>
                            <div className="w-2 h-2 rounded-full bg-blue-500 animate-pulse shadow-[0_0_8px_rgba(59,130,246,0.6)]" title="Status: Active" />
                          </div>
                          <Globe className="h-4 w-4 text-muted-foreground group-hover:text-blue-400 transition-colors" />
                        </CardHeader>
                        <CardContent>
                          <div className="text-2xl font-bold">TURN Active</div>
                          <p className="text-xs text-muted-foreground mb-3">
                            Secure NAT traversal established
                          </p>
                          <div className="bg-black/40 rounded p-2 font-mono text-[9px] text-blue-400/80 h-16 overflow-hidden">
                            <div className="line-clamp-1">[10:45:10] Relay node assigned: BK-4</div>
                            <div className="line-clamp-1">[10:45:16] Hole-punching successful</div>
                            <div className="line-clamp-1">[10:45:23] Connected peers: 24</div>
                          </div>
                        </CardContent>
                      </Card>
                    </div>

                    {/* Resource Explorer Tabs */}
                    <Tabs defaultValue="nodes" className="space-y-4">
                      <TabsList className="grid w-full grid-cols-4">
                        <TabsTrigger value="nodes">DVE Nodes</TabsTrigger>
                        <TabsTrigger value="validation">Validation Tasks</TabsTrigger>
                        <TabsTrigger value="tee">TEE Security</TabsTrigger>
                        <TabsTrigger value="cognitive">Cognitive Engine</TabsTrigger>
                      </TabsList>

                      <TabsContent value="nodes" className="space-y-4">
                        <DVENodesPanel onRentClick={onRentDVE} onNodeConnect={handleNodeAccess} />
                      </TabsContent>

                      <TabsContent value="validation" className="space-y-4">
                        <div className="flex items-center justify-between">
                          <h3 className="text-lg font-semibold">Validation Task Status</h3>
                          <div className="flex space-x-2">
                            <Button variant="outline" size="sm">
                              <Download className="w-4 h-4 mr-2" />
                              Export
                            </Button>
                            <Button variant="outline" size="sm">
                              <Share2 className="w-4 h-4 mr-2" />
                              Share
                            </Button>
                          </div>
                        </div>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <Card className="knirv-card-gradient">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2">
                                <Activity className="w-5 h-5" />
                                <span>Active Tasks</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-3xl font-bold">127</div>
                              <p className="text-sm text-muted-foreground">Currently processing</p>
                              <Button variant="outline" size="sm" className="w-full mt-4">
                                <Download className="w-3 h-3 mr-1" />
                                Task Reports
                              </Button>
                            </CardContent>
                          </Card>
                          <Card className="knirv-card-gradient">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2">
                                <BarChart3 className="w-5 h-5" />
                                <span>Completed Today</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-3xl font-bold">1,847</div>
                              <p className="text-sm text-muted-foreground">98.2% success rate</p>
                              <Button variant="outline" size="sm" className="w-full mt-4">
                                <Download className="w-3 h-3 mr-1" />
                                Performance Reports
                              </Button>
                            </CardContent>
                          </Card>
                        </div>
                      </TabsContent>

                      <TabsContent value="tee" className="space-y-4">
                        {teeSecurityStatus && (
                          <div className="grid gap-4">
                            <Card className="knirv-card-gradient">
                              <CardHeader>
                                <CardTitle className="flex items-center gap-2">
                                  <Shield className="h-5 w-5" />
                                  TEE Security Status
                                </CardTitle>
                              </CardHeader>
                              <CardContent>
                                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                                  <div>
                                    <p className="text-sm text-muted-foreground">Attestation Status</p>
                                    {getStatusBadge(teeSecurityStatus.attestation_status)}
                                  </div>
                                  <div>
                                    <p className="text-sm text-muted-foreground">Security Score</p>
                                    <div className="flex items-center gap-2">
                                      <Progress value={teeSecurityStatus.security_score} className="flex-1" />
                                      <span className="text-sm">{teeSecurityStatus.security_score}%</span>
                                    </div>
                                  </div>
                                  <div>
                                    <p className="text-sm text-muted-foreground">Active Enclaves</p>
                                    <p className="text-2xl font-bold">{teeSecurityStatus.enclave_count}</p>
                                  </div>
                                </div>
                                <div className="mt-4 grid grid-cols-2 gap-4">
                                  <div>
                                    <p className="text-sm text-muted-foreground">Last Audit</p>
                                    <p className="font-semibold">{new Date(teeSecurityStatus.last_audit).toLocaleString()}</p>
                                  </div>
                                  <div>
                                    <p className="text-sm text-muted-foreground">Threats Detected</p>
                                    <p className="font-semibold">{teeSecurityStatus.threats_detected}</p>
                                  </div>
                                </div>
                              </CardContent>
                            </Card>
                            
                            {teeSecurityStatus.threats_detected > 0 && (
                              <Alert>
                                <AlertTriangle className="h-4 w-4" />
                                <AlertDescription>
                                  {teeSecurityStatus.threats_detected} potential threats detected. Security team has been notified.
                                </AlertDescription>
                              </Alert>
                            )}

                            <div className="flex items-center justify-between mt-6">
                              <h3 className="text-lg font-semibold">TEE Technology Overview</h3>
                              <div className="flex space-x-2">
                                <Button variant="outline" size="sm">
                                  <Download className="w-4 h-4 mr-2" />
                                  Export
                                </Button>
                                <Button variant="outline" size="sm">
                                  <Share2 className="w-4 h-4 mr-2" />
                                  Share
                                </Button>
                              </div>
                            </div>
                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
                              <Card className="knirv-card-gradient">
                                <CardHeader>
                                  <CardTitle className="flex items-center space-x-2">
                                    <Shield className="w-5 h-5" />
                                    <span>SGX Enclaves</span>
                                  </CardTitle>
                                </CardHeader>
                                <CardContent>
                                  <div className="text-2xl font-bold">18</div>
                                  <p className="text-sm text-muted-foreground">Active secure enclaves</p>
                                  <Button variant="outline" size="sm" className="w-full mt-4">
                                    <Download className="w-3 h-3 mr-1" />
                                    Security Reports
                                  </Button>
                                </CardContent>
                              </Card>
                              <Card className="knirv-card-gradient">
                                <CardHeader>
                                  <CardTitle className="flex items-center space-x-2">
                                    <Lock className="w-5 h-5" />
                                    <span>SEV-SNP</span>
                                  </CardTitle>
                                </CardHeader>
                                <CardContent>
                                  <div className="text-2xl font-bold">6</div>
                                  <p className="text-sm text-muted-foreground">Secure VMs running</p>
                                  <Button variant="outline" size="sm" className="w-full mt-4">
                                    <Download className="w-3 h-3 mr-1" />
                                    VM Reports
                                  </Button>
                                </CardContent>
                              </Card>
                              <Card className="knirv-card-gradient">
                                <CardHeader>
                                  <CardTitle className="flex items-center space-x-2">
                                    <Zap className="w-5 h-5" />
                                    <span>TDX</span>
                                  </CardTitle>
                                </CardHeader>
                                <CardContent>
                                  <div className="text-2xl font-bold">3</div>
                                  <p className="text-sm text-muted-foreground">Trust domains active</p>
                                  <Button variant="outline" size="sm" className="w-full mt-4">
                                    <Download className="w-3 h-3 mr-1" />
                                    Trust Reports
                                  </Button>
                                </CardContent>
                              </Card>
                            </div>
                          </div>
                        )}
                      </TabsContent>

                      <TabsContent value="cognitive" className="space-y-4">
                        {cognitiveEngine && (
                          <div className="grid gap-4">
                            <Card className="knirv-card-gradient">
                              <CardHeader>
                                <CardTitle className="flex items-center gap-2">
                                  <Brain className="h-5 w-5" />
                                  Cognitive Engine Status
                                </CardTitle>
                              </CardHeader>
                              <CardContent>
                                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                                  <div>
                                    <p className="text-sm text-muted-foreground">Status</p>
                                    {getStatusBadge(cognitiveEngine.status)}
                                  </div>
                                  <div>
                                    <p className="text-sm text-muted-foreground">Accuracy</p>
                                    <div className="flex items-center gap-2">
                                      <Progress value={cognitiveEngine.accuracy} className="flex-1" />
                                      <span className="text-sm">{cognitiveEngine.accuracy}%</span>
                                    </div>
                                  </div>
                                  <div>
                                    <p className="text-sm text-muted-foreground">Tasks Processed</p>
                                    <p className="text-2xl font-bold">{cognitiveEngine.tasks_processed.toLocaleString()}</p>
                                  </div>
                                  <div>
                                    <p className="text-sm text-muted-foreground">Adaptation Rate</p>
                                    <div className="flex items-center gap-2">
                                      <Progress value={cognitiveEngine.adaptation_rate * 100} className="flex-1" />
                                      <span className="text-sm">{(cognitiveEngine.adaptation_rate * 100).toFixed(1)}%</span>
                                    </div>
                                  </div>
                                </div>
                                <div className="mt-4">
                                  <p className="text-sm text-muted-foreground">Fabric Version</p>
                                  <Badge variant="outline">{cognitiveEngine.model_version}</Badge>
                                </div>
                              </CardContent>
                            </Card>
                          </div>
                        )}
                        <CognitiveEnginePanel />
                      </TabsContent>
                    </Tabs>
                  </div>
                </SystemAccess>
              </TabsContent>
              
              <TabsContent value="profile">
                <div className="space-y-6">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-2xl font-bold">User Profile</h2>
                      <p className="text-muted-foreground">
                        Your account information, permissions, and usage analytics.
                      </p>
                    </div>
                    <Button variant="outline" className="flex items-center space-x-2">
                      <Download className="w-4 h-4" />
                      <span>Export Profile</span>
                    </Button>
                  </div>

                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {/* Account Information */}
                    <Card>
                      <CardHeader>
                        <CardTitle>Account Information</CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Username</p>
                            <p className="text-lg">{user.user}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Role</p>
                            <Badge variant={user.role === 'admin' ? 'destructive' : user.role === 'validator' ? 'secondary' : 'outline'}>
                              {user.role.toUpperCase()}
                            </Badge>
                          </div>
                        </div>

                        {user.node_id && (
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Assigned Node</p>
                            <Badge variant="outline">{user.node_id}</Badge>
                          </div>
                        )}

                        <div>
                          <p className="text-sm font-medium text-muted-foreground mb-2">NEXUS Permissions</p>
                          <div className="flex flex-wrap gap-2">
                            {user.nexus_access.map((permission, index) => (
                              <Badge key={index} variant="secondary" className="text-xs">
                                {permission}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    {/* Billing & Usage Summary */}
                    <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center space-x-2">
                          <BarChart3 className="w-5 h-5" />
                          <span>Usage Summary</span>
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Compute Hours</p>
                            <p className="text-2xl font-bold">247.3</p>
                            <p className="text-xs text-muted-foreground">This month</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">Storage Used</p>
                            <p className="text-2xl font-bold">1.2TB</p>
                            <p className="text-xs text-muted-foreground">Current usage</p>
                          </div>
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">API Calls</p>
                            <p className="text-2xl font-bold">12.4K</p>
                            <p className="text-xs text-muted-foreground">This month</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-muted-foreground">NRN Spent</p>
                            <p className="text-2xl font-bold">1,847</p>
                            <p className="text-xs text-muted-foreground">Total this month</p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>

                  {/* Billing & Usage Reports */}
                  <Card>
                    <CardHeader>
                      <div className="flex items-center justify-between">
                        <CardTitle className="flex items-center space-x-2">
                          <Download className="w-5 h-5" />
                          <span>Billing & Usage Reports</span>
                        </CardTitle>
                        <div className="flex space-x-2">
                          <Button variant="outline" size="sm">
                            <Download className="w-4 h-4 mr-2" />
                            Download All
                          </Button>
                          <Button variant="outline" size="sm">
                            <Share2 className="w-4 h-4 mr-2" />
                            Share Reports
                          </Button>
                        </div>
                      </div>
                      <CardDescription>
                        Generate and download detailed usage and billing reports for your account.
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Monthly Usage Report</CardTitle>
                            <CardDescription className="text-xs">
                              Detailed breakdown of compute, storage, and API usage
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 15, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Billing Statement</CardTitle>
                            <CardDescription className="text-xs">
                              NRN transactions and payment history
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 1, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Performance Analytics</CardTitle>
                            <CardDescription className="text-xs">
                              Task completion rates and efficiency metrics
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 16, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Security Audit Log</CardTitle>
                            <CardDescription className="text-xs">
                              Access logs and security events
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 16, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Resource Utilization</CardTitle>
                            <CardDescription className="text-xs">
                              Detailed resource consumption patterns
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Last Generated:</span>
                              <span>Dec 15, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1">
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1">
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="knirv-card-gradient">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">Custom Report</CardTitle>
                            <CardDescription className="text-xs">
                              Generate custom reports with specific parameters
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span>Status:</span>
                              <span>Ready to generate</span>
                            </div>
                            <Button variant="outline" size="sm" className="w-full">
                              <Settings className="w-3 h-3 mr-1" />
                              Configure & Generate
                            </Button>
                          </CardContent>
                        </Card>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>
            </div>
          </Tabs>
        </div>
      </nav>

      {/* CDE Access Modal / Panel */}
      {selectedNode && (
        useModularCDE ? (
          <CDEPanel
            isOpen={cdeModalOpen}
            onClose={() => setCdeModalOpen(false)}
            node={selectedNode}
            isModular={true}
            onToggleMode={() => {
              setUseModularCDE(false);
              setCdeModalOpen(true);
            }}
          />
        ) : (
          <NetworkAccessModal
            isOpen={cdeModalOpen}
            onClose={() => setCdeModalOpen(false)}
            nodeId={selectedNode.id}
            nodeName={selectedNode.name}
            onOpenKNIRVEngine={handleOpenKNIRVEngine}
            isModular={false}
            onToggleMode={() => {
              setUseModularCDE(true);
              setCdeModalOpen(true);
            }}
          />
        )
      )}

      {/* KNIRVENGINE Modal */}
      {selectedNode && (
        <KNIRVEngineModal
          isOpen={knirvEngineModalOpen}
          onClose={() => setKnirvEngineModalOpen(false)}
          nodeId={selectedNode.id}
        />
      )}

      {/* Admin Access Modal */}
      <NetworkAccessModal
        isOpen={showAdminAccess}
        onClose={() => setShowAdminAccess(false)}
        nodeId="ADMIN-NODE-01"
        nodeName="Administrative Node"
        onOpenKNIRVEngine={() => {
          setShowAdminAccess(false);
        }}
      />

      {/* Network Resource Access Modals */}
      <ActiveMemoryAccessModal
        isOpen={activeMemoryOpen}
        onClose={() => setActiveMemoryOpen(false)}
      />

      <KNIRVGraphAccessModal
        isOpen={knirvGraphOpen}
        onClose={() => setKnirvGraphOpen(false)}
      />

      <KNIRVChainAccessModal
        isOpen={knirvChainOpen}
        onClose={() => setKnirvChainOpen(false)}
      />

      <P2PTransportAccessModal
        isOpen={p2pTransportOpen}
        onClose={() => setP2PTransportOpen(false)}
      />
    </div>
  );
}
