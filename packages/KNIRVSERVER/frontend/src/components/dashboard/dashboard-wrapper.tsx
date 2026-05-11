'use client';

import React, { Suspense, useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTEESecurity } from "@/hooks/use-tee-security";
import { useAuth, ROLES } from '@/lib/auth-context';
import { UserProfile } from '@/components/auth/user-profile';
import { RoleGuard, DVEAccess, ValidationAccess, SystemAccess } from '@/components/auth/role-guard';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { NetworkAccessModal } from '@/components/nap/nap-access-modal';

import DVEWorkspacePanel from './dve-workspace-panel'; // Modular DVE Workspace
import DVECreationManagement from '@/components/dve-management/dve-creation-management';
import { KNIRVEngineModal } from '@/components/knirvengine/knirvengine-modal';
import { CognitiveEnginePanel } from '@/components/dashboard/cognitive-engine-panel';
import { CognitiveEngineStatusCard } from '@/components/dashboard/cognitive-engine-panel';
import NeuralDesktopPanel from '@/components/dashboard/neural-desktop-panel';
import { PredictiveAnalyticsPanel } from '@/components/dashboard/predictive-analytics-panel';
import { GuardrailViolationsPanel, GuardrailStatisticsCard } from '@/components/dashboard/guardrail-violations-panel';
import { BadgeLabPanel } from '@/components/dashboard/badge-lab-panel';
import { DVENodesPanel } from '@/components/dashboard/dve-nodes-panel';
import { DVECreationForm } from '@/components/dashboard/dve-creation-form';
import { ModuleLogViewer } from '@/components/dashboard/module-log-viewer';
import { useOnboarding } from "@/contexts/onboarding-context";
import OnboardingGuide from "@/components/onboarding/onboarding-guide";
import PaymentGatewayModal from './payment-gateway-modal';
import { WebguiIframeModal } from './webgui-iframe-modal';
import type { DVENode } from '@/types/api';
import { useDashboardStore } from '@/lib/store';
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
  WifiOff,
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
  Brain,
  Bell,
  Award,
  Coins
} from 'lucide-react';

interface DashboardWrapperProps {
  children: React.ReactNode;
  onRentDVE?: () => void;
}

function DashboardWrapperInner({ children, onRentDVE }: DashboardWrapperProps) {
  const router = useRouter();
  const { user, isLoading } = useAuth();
  const { state: onboardingState, updateState: updateOnboardingState, completeOnboarding, resetOnboarding } = useOnboarding();
  const { securityStatus: teeSecurityStatus, isLoading: teeLoading } = useTEESecurity();

  // ── Controlled tab state (allows postMessage navigation from Electron menu) ──
  const searchParams = useSearchParams();
  const [mainTab, setMainTab] = useState<string>('system');
  const [resourceTab, setResourceTab] = useState<string>('overview');

  // ── Shared navigation dispatch (used by both URL & postMessage paths) ──
  const applyNavSection = (section: string) => {
    if (section === 'admin') {
      setShowAdminAccess(true);
      return;
    }
    if (section === 'p2p-webgui') {
      goToWebgui('dashboard');
      return;
    }
    if (section === 'cognitive' || section === 'nodes' || section === 'badgelab' || section === 'overview') {
      setMainTab('system');
      setResourceTab(section);
    } else {
      setMainTab(section);
    }
  };

  // ── 1) Listen for postMessage (Electron desktop renderer) ──
  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      const { type, section, modal } = event.data || {};
      if (type === 'navigate' && section) applyNavSection(section);
      else if (type === 'open-modal' && modal === 'p2p-webgui') goToWebgui('dashboard');
    };
    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, []);

  // ── 2) Sync from URL ?nav= param on every navigation (web PWA / browser) ──
  useEffect(() => {
    const nav = searchParams.get('nav');
    if (nav) applyNavSection(nav);
  }, [searchParams]);

  const [cdeModalOpen, setCdeModalOpen] = useState(false);
  const [dveCreationModalOpen, setDveCreationModalOpen] = useState(false);
  const [knirvEngineModalOpen, setKnirvEngineModalOpen] = useState(false);
  const [paymentGatewayOpen, setPaymentGatewayOpen] = useState(false);
  const [selectedNode, setSelectedNode] = useState<DVENode | null>(null);
  const [showAdminAccess, setShowAdminAccess] = useState(false);
  const [webguiIframeOpen, setWebguiIframeOpen] = useState(false);
  const [webguiIframePage, setWebguiIframePage] = useState<string>('');

  const goToWebgui = (pageId: string) => {
    setWebguiIframePage(pageId);
    setWebguiIframeOpen(true);
  };

  // Persist active DVE nodes across navigation using Zustand store (survives tab switches)
  const { activeNodeIds, setActiveNodeId } = useDashboardStore();

  // Report generation helper
  const generateReport = (reportType: string, data: any) => {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const filename = `${reportType}-report-${timestamp}.json`;
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  const handleTaskReports = () => {
    generateReport('task-report', {
      type: 'Active Task Report',
      generatedAt: new Date().toISOString(),
      summary: {
        activeTasks: 127,
        status: 'Processing',
        lastUpdated: new Date().toISOString()
      },
      details: {
        tasks: Array.from({ length: 10 }, (_, i) => ({
          id: `task-${i + 1}`,
          name: `Validation Task ${i + 1}`,
          status: i < 7 ? 'active' : 'pending',
          priority: ['high', 'medium', 'low'][i % 3]
        }))
      }
    });
  };

  const handlePerformanceReports = () => {
    generateReport('performance-report', {
      type: 'Performance Report',
      generatedAt: new Date().toISOString(),
      summary: {
        completedToday: 1847,
        successRate: 98.2,
        averageLatency: 145,
        throughput: 1250
      },
      metrics: {
        accuracy: 98.2,
        errorRate: 1.8,
        uptime: 99.9
      }
    });
  };

  const handleSecurityReports = () => {
    generateReport('security-report', {
      type: 'SGX Enclaves Security Report',
      generatedAt: new Date().toISOString(),
      summary: {
        enclaveCount: teeSecurityStatus?.enclave_count ?? 0,
        attestationStatus: teeSecurityStatus?.attestation_status ?? 'unknown',
        securityScore: teeSecurityStatus?.security_score ?? 0,
        threatsDetected: teeSecurityStatus?.threats_detected ?? 0,
        lastAudit: teeSecurityStatus?.last_audit ?? null,
        lastAttestation: teeSecurityStatus?.last_attestation ?? null,
        monitoringEnabled: teeSecurityStatus?.monitoring_enabled ?? false,
        teeType: teeSecurityStatus?.tee_type ?? 'SGX'
      },
      activeThreats: teeSecurityStatus?.active_threats ?? [],
      auditHistory: teeSecurityStatus?.audit_history ?? [],
      performanceMetrics: teeSecurityStatus?.performance_metrics ?? null
    });
  };

  const handleVMReports = () => {
    generateReport('vm-report', {
      type: 'SEV-SNP VM Report',
      generatedAt: new Date().toISOString(),
      summary: {
        attestationStatus: teeSecurityStatus?.attestation_status ?? 'unknown',
        securityScore: teeSecurityStatus?.security_score ?? 0,
        threatsDetected: teeSecurityStatus?.threats_detected ?? 0,
        teeType: teeSecurityStatus?.tee_type ?? 'SEV-SNP',
        monitoringEnabled: teeSecurityStatus?.monitoring_enabled ?? false
      },
      performanceMetrics: teeSecurityStatus?.performance_metrics ?? null,
      activeThreats: teeSecurityStatus?.active_threats ?? []
    });
  };

  const handleTrustReports = () => {
    generateReport('trust-report', {
      type: 'TDX Trust Domain Report',
      generatedAt: new Date().toISOString(),
      summary: {
        attestationStatus: teeSecurityStatus?.attestation_status ?? 'unknown',
        securityScore: teeSecurityStatus?.security_score ?? 0,
        lastAttestation: teeSecurityStatus?.last_attestation ?? null,
        teeType: teeSecurityStatus?.tee_type ?? 'TDX',
        threatsDetected: teeSecurityStatus?.threats_detected ?? 0
      },
      auditHistory: teeSecurityStatus?.audit_history ?? [],
      activeThreats: teeSecurityStatus?.active_threats ?? [],
      performanceMetrics: teeSecurityStatus?.performance_metrics ?? null
    });
  };

  const handleMonthlyUsageReport = () => {
    generateReport('monthly-usage', {
      type: 'Monthly Usage Report',
      generatedAt: new Date().toISOString(),
      period: new Date().toLocaleDateString('en-US', { month: 'long', year: 'numeric' }),
      account: {
        username: user?.user,
        role: user?.role,
        nodeId: user?.node_id
      },
      usage: {
        computeHours: 247.3,
        storageUsed: '1.2TB',
        apiCalls: 12400,
        nrnSpent: 1847
      }
    });
  };

  const handleBillingStatement = () => {
    generateReport('billing-statement', {
      type: 'Billing Statement',
      generatedAt: new Date().toISOString(),
      period: new Date().toLocaleDateString('en-US', { month: 'long', year: 'numeric' }),
      account: {
        username: user?.user,
        role: user?.role,
        nodeId: user?.node_id,
        permissions: user?.nexus_access
      },
      billing: {
        totalNRNSpent: 1847,
        computeHours: 247.3,
        storageUsed: '1.2TB',
        apiCalls: 12400
      }
    });
  };

  const handlePerformanceAnalytics = () => {
    generateReport('performance-analytics', {
      type: 'Performance Analytics',
      generatedAt: new Date().toISOString(),
      summary: {
        activeTasks: 127,
        completedToday: 1847,
        successRate: 98.2,
        averageLatency: 145,
        throughput: 1250,
        errorRate: 1.8,
        uptime: 99.9
      },
      teePerformance: teeSecurityStatus?.performance_metrics ?? null,
      teeSecurityScore: teeSecurityStatus?.security_score ?? 0
    });
  };

  const handleSecurityAuditLog = () => {
    generateReport('security-audit-log', {
      type: 'Security Audit Log',
      generatedAt: new Date().toISOString(),
      summary: {
        securityScore: teeSecurityStatus?.security_score ?? 0,
        attestationStatus: teeSecurityStatus?.attestation_status ?? 'unknown',
        threatsDetected: teeSecurityStatus?.threats_detected ?? 0,
        lastAudit: teeSecurityStatus?.last_audit ?? null,
        lastAttestation: teeSecurityStatus?.last_attestation ?? null,
        monitoringEnabled: teeSecurityStatus?.monitoring_enabled ?? false,
        teeType: teeSecurityStatus?.tee_type ?? 'unknown',
        enclaveCount: teeSecurityStatus?.enclave_count ?? 0
      },
      auditHistory: teeSecurityStatus?.audit_history ?? [],
      activeThreats: teeSecurityStatus?.active_threats ?? []
    });
  };

  const handleResourceUtilization = () => {
    generateReport('resource-utilization', {
      type: 'Resource Utilization',
      generatedAt: new Date().toISOString(),
      resources: {
        computeHours: 247.3,
        storageUsed: '1.2TB',
        apiCalls: 12400,
        nrnSpent: 1847
      },
      teeMetrics: teeSecurityStatus?.performance_metrics ?? null,
      enclaveCount: teeSecurityStatus?.enclave_count ?? 0
    });
  };

  const handleDownloadAll = () => {
    handleMonthlyUsageReport();
    handleBillingStatement();
    handlePerformanceAnalytics();
    handleSecurityAuditLog();
    handleResourceUtilization();
  };

  const handleExportReport = () => {
    generateReport('full-export', {
      type: 'Full Reports Export',
      generatedAt: new Date().toISOString(),
      account: {
        username: user?.user,
        role: user?.role,
        nodeId: user?.node_id,
        permissions: user?.nexus_access
      },
      validationTasks: {
        activeTasks: 127,
        completedToday: 1847,
        successRate: 98.2,
        averageLatency: 145,
        throughput: 1250
      },
      teeSecurity: teeSecurityStatus ? {
        securityScore: teeSecurityStatus.security_score,
        attestationStatus: teeSecurityStatus.attestation_status,
        enclaveCount: teeSecurityStatus.enclave_count,
        threatsDetected: teeSecurityStatus.threats_detected,
        lastAudit: teeSecurityStatus.last_audit,
        teeType: teeSecurityStatus.tee_type,
        activeThreats: teeSecurityStatus.active_threats,
        auditHistory: teeSecurityStatus.audit_history,
        performanceMetrics: teeSecurityStatus.performance_metrics
      } : null,
      usage: {
        computeHours: 247.3,
        storageUsed: '1.2TB',
        apiCalls: 12400,
        nrnSpent: 1847
      }
    });
  };

  const handleShareReport = (reportType: string, reportData: any) => {
    const json = JSON.stringify(reportData, null, 2);
    navigator.clipboard?.writeText(json).catch(() => {});
  };

  const handleNodeAccess = (node: DVENode) => {
    setSelectedNode(node);
    setCdeModalOpen(true);
  };

  const handleOpenKNIRVEngine = () => {
    setCdeModalOpen(false);
    setKnirvEngineModalOpen(true);
  };

  const handleDVEManagement = () => {
    setDveCreationModalOpen(true);
  };

  const handleOnboardingComplete = async (config: any) => {
    // First update local state
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

    // Get user info for the API call
    const token = localStorage.getItem('knirv_nexus_token');
    const userId = localStorage.getItem('knirv_user_id') || 'default';

    // Call the onboarding completion API to save to KNIRVBASE and optionally commit policies to blockchain
    try {
      const response = await fetch('/api/icme/onboarding/complete', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
        },
        body: JSON.stringify({
          user_id: userId,
          wallet_name: config.walletName,
          fabric_inputs: config.fabricInputs,
          guardrails: config.guardrails,
          connection_data: config.connectionData,
          privacy_settings: config.privacySettings,
          commit_to_chain: true // Commit policies to blockchain
        })
      });

      if (response.ok) {
        const result = await response.json();
        console.log('Onboarding completed and policies committed:', result);
        if (result.policy_tx_hashes && result.policy_tx_hashes.length > 0) {
          console.log('Policy transaction hashes:', result.policy_tx_hashes);
        }
      } else {
        console.warn('Failed to complete onboarding via API, data saved locally');
      }
    } catch (error) {
      console.error('Error completing onboarding:', error);
      // Data is still saved locally via updateOnboardingState
    }
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

  useEffect(() => {
    if (!isLoading && !user?.authenticated) {
      router.push('/login');
    }
  }, [isLoading, user, router]);

  if (isLoading || !user?.authenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div data-testid="loading-spinner" className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen aether-dashboard-bg text-gray-100 overflow-hidden font-inter selection:bg-indigo-500/30">
      <div className="aether-scanline"></div>
      {/* Header with user profile */}
      <header className="h-16 border-b border-gray-800/50 flex items-center justify-between px-6 bg-[#02040a]/80 backdrop-blur-xl z-20">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl aether-bevel-dark flex items-center justify-center">
              <Shield className="w-6 h-6 text-indigo-400" />
            </div>
            <div>
              <h1 className="text-lg font-black tracking-tighter text-white uppercase tracking-widest">KNIRV Server</h1>
              <p className="text-[9px] text-gray-500 font-mono">Deterministic Validation Environment</p>
            </div>
          </div>
        </div>
        
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-sm">
            <span className="text-gray-500">Welcome,</span>
            <span className="font-medium text-gray-300">{user.user}</span>
            <Badge variant={user.role === 'admin' ? 'destructive' : user.role === 'validator' ? 'secondary' : 'outline'}>
              {ROLES[user.role]?.displayName || user.role.toUpperCase()}
            </Badge>
          </div>
          <button
            onClick={() => router.push('/menu')}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-mono font-bold tracking-widest border border-indigo-500/40 text-indigo-400/70 bg-indigo-500/5 hover:bg-indigo-500/15 hover:text-indigo-400 hover:border-indigo-400/60 transition-all duration-200 rounded-sm"
            title="Return to Constellation Menu"
          >
            ⬡ MENU
          </button>
          <UserProfile />
        </div>
      </header>

      {/* Role-based navigation */}
      <nav className="border-b border-gray-800/50 bg-[#02040a]/60 backdrop-blur-xl">
        <div className="container mx-auto px-4 py-2">
          <Tabs value={mainTab} onValueChange={setMainTab} className="w-full">
            <TabsList className="inline-flex h-12 w-full bg-transparent">
              <TabsTrigger value="setup" className="flex items-center space-x-2 text-gray-400 data-[state=active]:text-indigo-400 data-[state=active]:bg-indigo-500/10">
                <Settings className="w-4 h-4" />
                <span>Setup</span>
              </TabsTrigger>

              {user?.nexus_access?.includes('compliance:read') && (
                <TabsTrigger value="compliance" className="flex items-center space-x-2 text-gray-400 data-[state=active]:text-indigo-400 data-[state=active]:bg-indigo-500/10">
                  <Scale className="w-4 h-4" />
                  <span>Compliance</span>
                </TabsTrigger>
              )}

              <SystemAccess operation="read" showError={false}>
                <TabsTrigger value="system" className="flex items-center space-x-2 text-gray-400 data-[state=active]:text-indigo-400 data-[state=active]:bg-indigo-500/10">
                  <Settings className="w-4 h-4" />
                  <span>Network & Resources</span>
                </TabsTrigger>
              </SystemAccess>

              <RoleGuard allowedRoles={['admin']} showError={false}>
                <button
                  onClick={() => setShowAdminAccess(true)}
                  className="inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-md border border-white/20 px-2 py-1 text-sm font-medium whitespace-nowrap transition-colors duration-150 text-gray-400 hover:bg-indigo-500/20 hover:text-indigo-400 hover:border-indigo-400/50 disabled:pointer-events-none disabled:opacity-50"
                >
                  <User className="w-4 h-4" />
                  <span>Admin Access</span>
                </button>
              </RoleGuard>

              <TabsTrigger value="profile" className="flex items-center space-x-2 text-gray-400 data-[state=active]:text-indigo-400 data-[state=active]:bg-indigo-500/10">
                <Eye className="w-4 h-4" />
                <span>Reports</span>
              </TabsTrigger>
            </TabsList>
            
            <div className="py-6">
              <TabsContent value="setup">
                {!onboardingState.isOnboardingComplete ? (
                  <div className="rounded-3xl overflow-hidden aether-bevel-dark">
                    <OnboardingGuide 
                      onComplete={handleOnboardingComplete} 
                      onReset={resetOnboarding}
                    />
                  </div>
                ) : (
                  <div className="space-y-6">
                    <div className="rounded-3xl overflow-hidden aether-bevel-dark p-6">
                      <div className="flex items-center justify-between mb-6 pb-4 border-b border-gray-800">
                        <div>
                          <h2 className="text-2xl font-bold text-gray-200">Network Settings</h2>
                          <p className="text-gray-500">
                            Configure global network-wide settings for your KNIRV deployment
                          </p>
                        </div>
                          <Button 
                            onClick={() => {
                              updateOnboardingState({
                                currentStep: 'guide',
                                isOnboardingComplete: false
                              });
                            }}
                            className="bg-indigo-600 hover:bg-indigo-500 text-white"
                          >
                            <Settings className="w-4 h-4 mr-2" />
                            Reconfigure Settings
                          </Button>
                      </div>
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                          <Card className="aether-bevel-dark-light rounded-2xl">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2 text-gray-300">
                                <Shield className="w-5 h-5 text-indigo-400" />
                                <span>Data Wallet Configuration</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Wallet Name:</span>
                                  <span className="font-mono text-gray-300">{onboardingState.dataWalletConfig?.walletName || 'Not configured'}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Fabric Inputs:</span>
                                  <span className="font-mono text-gray-300">{onboardingState.dataWalletConfig?.fabricInputs?.length || 0} configured</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Guardrails:</span>
                                  <Badge variant="outline">
                                    {onboardingState.dataWalletConfig?.guardrails 
                                      ? Object.values(onboardingState.dataWalletConfig.guardrails).filter(Boolean).length 
                                      : 0} active
                                  </Badge>
                                </div>
                              </div>
                            </CardContent>
                          </Card>

                          <Card className="aether-bevel-dark-light rounded-2xl">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2 text-gray-300">
                                <Lock className="w-5 h-5 text-amber-400" />
                                <span>Privacy Preferences</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Encryption:</span>
                                  <Badge variant="outline">{onboardingState.privacyPreferences?.dataEncryption ? 'Enabled' : 'Disabled'}</Badge>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Analytics:</span>
                                  <Badge variant="outline">{onboardingState.privacyPreferences?.allowAnalytics ? 'Enabled' : 'Disabled'}</Badge>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Local Processing:</span>
                                  <Badge variant="outline">{onboardingState.privacyPreferences?.localProcessing ? 'Enabled' : 'Disabled'}</Badge>
                                </div>
                              </div>
                            </CardContent>
                          </Card>

                          <Card className="aether-bevel-dark-light rounded-2xl">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2 text-gray-300">
                                <Network className="w-5 h-5 text-green-400" />
                                <span>Connection Status</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-gray-500">API Keys:</span>
                                  <span className="font-mono text-gray-300">{onboardingState.connectionData?.apiKeys?.length || 0}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-500">MCP Servers:</span>
                                  <span className="font-mono text-gray-300">{onboardingState.connectionData?.mcpServers?.length || 0}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-500">Onboarding:</span>
                                  <Badge className="bg-green-500">Complete</Badge>
                                </div>
                              </div>
                            </CardContent>
                          </Card>
                        </div>

                        <div className="mt-6 p-4 aether-glass-panel rounded-xl">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center space-x-3">
                              <CheckCircle className="w-5 h-5 text-green-400" />
                              <div>
                                <p className="font-medium text-gray-200">Onboarding Completed</p>
                                <p className="text-sm text-gray-500">
                                  Your network is configured and ready. Click "Reconfigure Settings" to modify your network configuration.
                                </p>
                              </div>
                            </div>
                            <div className="text-right text-sm text-gray-500">
                              <p>Status: Active</p>
                            </div>
                          </div>
                        </div>
                    </div>
                  </div>
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
                    {/* FinTech dashboard will be loaded dynamically when the plugin runs */}
                  </div>
                </TabsContent>
              )}
              

              
              <TabsContent value="system">
                <SystemAccess operation="read">
                  <div className="space-y-6">
                    {/* Main Title */}
                    <div className="text-center space-y-2 pb-6 border-b border-gray-800">
                      <div className="flex items-center justify-center gap-2">
                        <h1 className="text-4xl font-bold knirv-gradient-text">KNIRV-SERVER DVE</h1>
                        <Badge className="bg-green-500"><Wifi className="w-3 h-3 mr-1" /> Live</Badge>
                      </div>
                      <p className="text-lg text-gray-500">
                        The Crucible of Verifiable AI Intelligence
                      </p>
                    </div>

                    {/* NEW: Top Navigation Bar — moved up below the main title */}
                    <Tabs value={resourceTab} onValueChange={setResourceTab} className="space-y-6">
                      <TabsList className="grid w-full grid-cols-4 bg-gray-900/50 border border-gray-800">
                        <TabsTrigger value="overview" className="text-gray-400 data-[state=active]:text-indigo-400 data-[state=active]:bg-indigo-500/10">Overview</TabsTrigger>
                        <TabsTrigger value="nodes" className="text-gray-400 data-[state=active]:text-indigo-400 data-[state=active]:bg-indigo-500/10">DVE Nodes</TabsTrigger>
                        <TabsTrigger value="cognitive" className="text-gray-400 data-[state=active]:text-indigo-400 data-[state=active]:bg-indigo-500/10">Cognitive Engine</TabsTrigger>
                        <TabsTrigger value="badgelab" className="text-gray-400 data-[state=active]:text-amber-400 data-[state=active]:bg-amber-500/10">Badge Lab</TabsTrigger>
                      </TabsList>

                      {/* ── Overview Tab: All 8 Card Tiles ── */}
                      <TabsContent value="overview" className="space-y-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <h2 className="text-2xl font-bold text-gray-200">Network & Resource Explorer</h2>
                            <p className="text-gray-500">
                              Monitor network topology, resource allocation, and system performance across the KNIRV ecosystem.
                            </p>
                          </div>
                          <Button variant="outline" className="flex items-center space-x-2 border-green-700/50 text-green-400 hover:bg-green-600/20 hover:text-green-300 hover:border-green-500/50" onClick={() => setPaymentGatewayOpen(true)}>
                            <Coins className="w-4 h-4" />
                            <span>Buy NRN</span>
                          </Button>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                          {/* 1. KNIRVORACLE → WebGUI Network Inference DAO */}
                          <Card 
                            className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive"
                            onClick={() => goToWebgui('network-inference-dao')}
                          >
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                              <div className="flex items-center space-x-2">
                                <CardTitle className="text-sm font-medium text-gray-300">KNIRVORACLE</CardTitle>
                                <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse shadow-[0_0_8px_rgba(34,197,94,0.6)]" title="Status: Active" />
                              </div>
                              <Eye className="h-4 w-4 text-gray-500" />
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">Root Node</div>
                              <p className="text-xs text-gray-500 mb-3">Verifiable data feeds & attestation</p>
                              <div className="bg-black/40 rounded-lg p-2 font-mono text-[9px] text-green-400/80 h-16 overflow-hidden">
                                <div className="line-clamp-1">[10:45:21] Oracle heartbeat OK</div>
                                <div className="line-clamp-1">[10:45:22] Attestation round #882</div>
                                <div className="line-clamp-1">[10:45:23] Price feed updated</div>
                              </div>
                            </CardContent>
                          </Card>

                          {/* 2. Reasoning Graph (KNIRVGRAPH) */}
                          <Card 
                            className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive"
                            onClick={() => goToWebgui('graph-explorer')}
                          >
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                              <div className="flex items-center space-x-2">
                                <CardTitle className="text-sm font-medium text-gray-300">Reasoning Graph (KNIRVGRAPH)</CardTitle>
                                <div className="w-2 h-2 rounded-full bg-indigo-500 animate-pulse shadow-[0_0_8px_rgba(99,102,241,0.6)]" title="Status: Synced" />
                              </div>
                              <Network className="h-4 w-4 text-gray-500" />
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">142 Traces</div>
                              <p className="text-xs text-gray-500 mb-3">Context records in .md fabric</p>
                              <div className="bg-black/40 rounded-lg p-2 font-mono text-[9px] text-indigo-400/80 h-16 overflow-hidden">
                                <ModuleLogViewer module="knirvgraph" maxLines={3} maxHeight="h-14" />
                              </div>
                            </CardContent>
                          </Card>

                          {/* 3. Solution Vault (KNIRVCHAIN) */}
                          <Card 
                            className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive"
                            onClick={() => goToWebgui('chain-explorer')}
                          >
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                              <div className="flex items-center space-x-2">
                                <CardTitle className="text-sm font-medium text-gray-300">Solution Vault (KNIRVCHAIN)</CardTitle>
                                <div className="w-2 h-2 rounded-full bg-indigo-500 animate-pulse shadow-[0_0_8px_rgba(99,102,241,0.6)]" title="Status: Minting" />
                              </div>
                              <Lock className="h-4 w-4 text-gray-500" />
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">28 Nodes</div>
                              <p className="text-xs text-gray-500 mb-3">Verifiable executable logic</p>
                              <div className="bg-black/40 rounded-lg p-2 font-mono text-[9px] text-amber-400/80 h-16 overflow-hidden">
                                <ModuleLogViewer module="knirvchain" maxLines={3} maxHeight="h-14" />
                              </div>
                            </CardContent>
                          </Card>

                          {/* 4. P2P Transport */}
                          <Card 
                            className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive"
                            onClick={() => goToWebgui('dashboard')}
                          >
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                              <div className="flex items-center space-x-2">
                                <CardTitle className="text-sm font-medium text-gray-300">P2P Transport</CardTitle>
                                <div className="w-2 h-2 rounded-full bg-indigo-500 animate-pulse shadow-[0_0_8px_rgba(99,102,241,0.6)]" title="Status: Active" />
                              </div>
                              <Globe className="h-4 w-4 text-gray-500" />
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">TURN Active</div>
                              <p className="text-xs text-gray-500 mb-3">Secure NAT traversal established</p>
                              <div className="bg-black/40 rounded-lg p-2 font-mono text-[9px] text-indigo-400/80 h-16 overflow-hidden">
                                <ModuleLogViewer module="knirvgateway" maxLines={3} maxHeight="h-14" />
                              </div>
                            </CardContent>
                          </Card>

                          {/* 5. Active Memory (KNIRVBASE) */}
                          <Card 
                            className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive"
                            onClick={() => goToWebgui('dashboard')}
                          >
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                              <div className="flex items-center space-x-2">
                                <CardTitle className="text-sm font-medium text-gray-300">Active Memory (KNIRVBASE)</CardTitle>
                                <div className="w-2 h-2 rounded-full bg-indigo-500 animate-pulse shadow-[0_0_8px_rgba(99,102,241,0.6)]" title="Status: Online" />
                              </div>
                              <Database className="h-4 w-4 text-gray-500" />
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">Encrypted</div>
                              <p className="text-xs text-gray-500 mb-3">PQC Markdown persistence active</p>
                              <div className="bg-black/40 rounded-lg p-2 font-mono text-[9px] text-indigo-400/80 h-16 overflow-hidden">
                                <div className="line-clamp-1">[10:45:21] Kyber-768 Handshake OK</div>
                                <div className="line-clamp-1">[10:45:22] Committing .md fabric slice...</div>
                                <div className="line-clamp-1">[10:45:23] Dilithium-3 Signature Valid</div>
                              </div>
                            </CardContent>
                          </Card>

                          {/* 6. KNIRVHASHER */}
                          <Card 
                            className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive"
                            onClick={() => setShowAdminAccess(true)}
                          >
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                              <div className="flex items-center space-x-2">
                                <CardTitle className="text-sm font-medium text-gray-300">KNIRVHASHER</CardTitle>
                                <div className="w-2 h-2 rounded-full bg-indigo-500 animate-pulse shadow-[0_0_8px_rgba(99,102,241,0.6)]" title="Status: Active" />
                              </div>
                              <Cpu className="h-4 w-4 text-gray-500" />
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">Hashing</div>
                              <p className="text-xs text-gray-500 mb-3">PQC-accelerated hash operations</p>
                              <div className="bg-black/40 rounded-lg p-2 font-mono text-[9px] text-cyan-400/80 h-16 overflow-hidden">
                                <div className="line-clamp-1">[10:45:21] Dilithium hash chain</div>
                                <div className="line-clamp-1">[10:45:22] Merkle root computed</div>
                                <div className="line-clamp-1">[10:45:23] Batch verified</div>
                              </div>
                            </CardContent>
                          </Card>

                          {/* 7. KNIRVARENA */}
                          <Card 
                            className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive"
                            onClick={() => setShowAdminAccess(true)}
                          >
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                              <div className="flex items-center space-x-2">
                                <CardTitle className="text-sm font-medium text-gray-300">KNIRVARENA</CardTitle>
                                <div className="w-2 h-2 rounded-full bg-purple-500 animate-pulse shadow-[0_0_8px_rgba(168,85,247,0.6)]" title="Status: Active" />
                              </div>
                              <Monitor className="h-4 w-4 text-gray-500" />
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">3D RTS</div>
                              <p className="text-xs text-gray-500 mb-3">Real-time strategy simulation</p>
                              <div className="bg-black/40 rounded-lg p-2 font-mono text-[9px] text-purple-400/80 h-16 overflow-hidden">
                                <div className="line-clamp-1">[10:45:21] Arena session active</div>
                                <div className="line-clamp-1">[10:45:22] Agent vs agent match</div>
                                <div className="line-clamp-1">[10:45:23] Telemetry stream OK</div>
                              </div>
                            </CardContent>
                          </Card>

                          {/* 8. KNIRVBRIDGE */}
                          <Card 
                            className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive"
                            onClick={() => setShowAdminAccess(true)}
                          >
                            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                              <div className="flex items-center space-x-2">
                                <CardTitle className="text-sm font-medium text-gray-300">KNIRVBRIDGE</CardTitle>
                                <div className="w-2 h-2 rounded-full bg-amber-500 animate-pulse shadow-[0_0_8px_rgba(245,158,11,0.6)]" title="Status: Active" />
                              </div>
                              <Share2 className="h-4 w-4 text-gray-500" />
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">IBC Bridge</div>
                              <p className="text-xs text-gray-500 mb-3">Cross-chain asset and data relay</p>
                              <div className="bg-black/40 rounded-lg p-2 font-mono text-[9px] text-amber-400/80 h-16 overflow-hidden">
                                <div className="line-clamp-1">[10:45:21] IBC packet received</div>
                                <div className="line-clamp-1">[10:45:22] Validating light client</div>
                                <div className="line-clamp-1">[10:45:23] Acknowledgement sent</div>
                              </div>
                            </CardContent>
                          </Card>
                        </div>
                      </TabsContent>

                      {/* ── DVE Nodes Tab ── */}
                      <TabsContent value="nodes" className="space-y-4">
                        <DVENodesPanel
                          onRentClick={handleDVEManagement}
                          onNodeConnect={handleNodeAccess}
                          effectiveActiveNodeIdss={activeNodeIds}
                          onActiveNodeChange={(nodeId, isActive) => {
                            setActiveNodeId(nodeId, isActive);
                          }}
                        />
                      </TabsContent>

                      {/* ── Cognitive Engine Tab ── */}
                      <TabsContent value="cognitive" className="space-y-4">
                        {/* Engine Controls - Full Width */}
                        <CognitiveEnginePanel />

                        {/* Neural Desktop + Cognitive Engine Status - same row */}
                        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                          <div className="space-y-4">
                            <NeuralDesktopPanel />
                          </div>
                          <div className="space-y-4">
                            <CognitiveEngineStatusCard />
                          </div>
                        </div>

                        {/* Predictive Analytics + Guardrails - bottom row */}
                        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                          <div className="space-y-4">
                            <PredictiveAnalyticsPanel />
                          </div>
                          <div className="space-y-4">
                            <GuardrailStatisticsCard />
                            <GuardrailViolationsPanel />
                          </div>
                        </div>
                      </TabsContent>

                      {/* ── Badge Lab Tab ── */}
                      <TabsContent value="badgelab" className="space-y-4">
                        <BadgeLabPanel />
                      </TabsContent>
                    </Tabs>
                  </div>
                </SystemAccess>
              </TabsContent>
              
              <TabsContent value="profile">
                <div className="space-y-6">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-2xl font-bold text-gray-200">Reports</h2>
                      <p className="text-gray-500">
                        Validation, Security, and Usage Reports
                      </p>
                    </div>
                    <Button variant="outline" className="flex items-center space-x-2 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleExportReport}>
                      <Download className="w-4 h-4" />
                      <span>Export Report</span>
                    </Button>
                  </div>

                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {/* Account Information */}
                    <Card className="aether-bevel-dark rounded-2xl">
                      <CardHeader>
                        <CardTitle className="text-gray-300">Account Information</CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-gray-500">Username</p>
                            <p className="text-lg text-gray-200">{user.user}</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-gray-500">Role</p>
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
                          <p className="text-sm font-medium text-muted-foreground mb-2">SERVER Permissions</p>
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

                    {/* Usage Summary */}
                    <Card className="aether-bevel-dark rounded-2xl">
                      <CardHeader>
                        <CardTitle className="flex items-center space-x-2 text-gray-300">
                          <BarChart3 className="w-5 h-5 text-indigo-400" />
                          <span>Usage Summary</span>
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-gray-500">Compute Hours</p>
                            <p className="text-2xl font-bold text-gray-200">247.3</p>
                            <p className="text-xs text-gray-600">This month</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-gray-500">Storage Used</p>
                            <p className="text-2xl font-bold text-gray-200">1.2TB</p>
                            <p className="text-xs text-gray-600">Current usage</p>
                          </div>
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <p className="text-sm font-medium text-gray-500">API Calls</p>
                            <p className="text-2xl font-bold text-gray-200">12.4K</p>
                            <p className="text-xs text-gray-600">This month</p>
                          </div>
                          <div>
                            <p className="text-sm font-medium text-gray-500">NRN Spent</p>
                            <p className="text-2xl font-bold text-gray-200">1,847</p>
                            <p className="text-xs text-gray-600">Total this month</p>
                            <Button 
                              variant="outline" 
                              size="sm" 
                              className="mt-2 border-green-600/50 text-green-400 hover:bg-green-600/20"
                              onClick={() => setPaymentGatewayOpen(true)}
                            >
                              <Coins className="w-3 h-3 mr-1" />
                              Buy NRN
                            </Button>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>

                  {/* Validation Tasks Section */}
                  <div className="space-y-4 mt-6">
                      <h3 className="text-lg font-semibold text-gray-200">Validation Tasks</h3>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <Card className="aether-bevel-dark rounded-2xl">
                          <CardHeader>
                            <CardTitle className="flex items-center space-x-2 text-gray-300">
                              <Activity className="w-5 h-5 text-indigo-400" />
                              <span>Active Tasks</span>
                            </CardTitle>
                          </CardHeader>
                          <CardContent>
                            <div className="text-3xl font-bold text-gray-200">127</div>
                            <p className="text-sm text-gray-500">Currently processing</p>
                            <Button variant="outline" size="sm" className="w-full mt-4 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleTaskReports}>
                              <Download className="w-3 h-3 mr-1" />
                              Task Reports
                            </Button>
                          </CardContent>
                        </Card>
                        <Card className="aether-bevel-dark rounded-2xl">
                          <CardHeader>
                            <CardTitle className="flex items-center space-x-2 text-gray-300">
                              <BarChart3 className="w-5 h-5 text-indigo-400" />
                              <span>Completed Today</span>
                            </CardTitle>
                          </CardHeader>
                          <CardContent>
                            <div className="text-3xl font-bold text-gray-200">1,847</div>
                            <p className="text-sm text-gray-500">98.2% success rate</p>
                            <Button variant="outline" size="sm" className="w-full mt-4 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handlePerformanceReports}>
                              <Download className="w-3 h-3 mr-1" />
                              Performance Reports
                            </Button>
                          </CardContent>
                        </Card>
                      </div>
                    </div>

                    {/* TEE Security Section - Moved from Network & Resources */}
                    {teeSecurityStatus && (
                      <div className="space-y-4">
                        <h3 className="text-lg font-semibold text-gray-200">TEE Security</h3>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                          <Card className="aether-bevel-dark rounded-2xl">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2 text-gray-300">
                                <Shield className="w-5 h-5 text-indigo-400" />
                                <span>SGX Enclaves</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">18</div>
                              <p className="text-sm text-gray-500">Active secure enclaves</p>
                              <Button variant="outline" size="sm" className="w-full mt-4 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleSecurityReports}>
                                <Download className="w-3 h-3 mr-1" />
                                Security Reports
                              </Button>
                            </CardContent>
                          </Card>
                          <Card className="aether-bevel-dark rounded-2xl">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2 text-gray-300">
                                <Lock className="w-5 h-5 text-amber-400" />
                                <span>SEV-SNP</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">6</div>
                              <p className="text-sm text-gray-500">Secure VMs running</p>
                              <Button variant="outline" size="sm" className="w-full mt-4 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleVMReports}>
                                <Download className="w-3 h-3 mr-1" />
                                VM Reports
                              </Button>
                            </CardContent>
                          </Card>
                          <Card className="aether-bevel-dark rounded-2xl">
                            <CardHeader>
                              <CardTitle className="flex items-center space-x-2 text-gray-300">
                                <Zap className="w-5 h-5 text-indigo-400" />
                                <span>TDX</span>
                              </CardTitle>
                            </CardHeader>
                            <CardContent>
                              <div className="text-2xl font-bold text-gray-200">3</div>
                              <p className="text-sm text-gray-500">Trust domains active</p>
                              <Button variant="outline" size="sm" className="w-full mt-4 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleTrustReports}>
                                <Download className="w-3 h-3 mr-1" />
                                Trust Reports
                              </Button>
                            </CardContent>
                          </Card>
                        </div>
                      </div>
                    )}

                  {/* Billing & Usage Reports */}
                  <Card className="aether-bevel-dark rounded-2xl">
                    <CardHeader>
                      <div className="flex items-center justify-between">
                        <CardTitle className="flex items-center space-x-2 text-gray-300">
                          <Download className="w-5 h-5 text-indigo-400" />
                          <span>Billing & Usage Reports</span>
                        </CardTitle>
                        <div className="flex space-x-2">
                          <Button variant="outline" size="sm" className="border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleDownloadAll}>
                            <Download className="w-4 h-4 mr-2" />
                            Download All
                          </Button>
                          <Button variant="outline" size="sm" className="border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={() => handleShareReport('full-export', { type: 'Full Reports Export', generatedAt: new Date().toISOString(), account: { username: user?.user, role: user?.role }, teeSecurity: teeSecurityStatus })}>
                            <Share2 className="w-4 h-4 mr-2" />
                            Share Reports
                          </Button>
                        </div>
                      </div>
                      <CardDescription className="text-gray-500">
                        Generate and download detailed usage and billing reports for your account.
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        <Card className="aether-bevel-dark-light rounded-xl">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm text-gray-300">Monthly Usage Report</CardTitle>
                            <CardDescription className="text-xs text-gray-500">
                              Detailed breakdown of compute, storage, and API usage
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span className="text-gray-500">Last Generated:</span>
                              <span className="text-gray-300">Dec 15, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleMonthlyUsageReport}>
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={() => handleShareReport('monthly-usage', { type: 'Monthly Usage Report', account: { username: user?.user, role: user?.role }, usage: { computeHours: 247.3, storageUsed: '1.2TB', apiCalls: 12400, nrnSpent: 1847 } })}>
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="aether-bevel-dark-light rounded-xl">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm text-gray-300">Billing Statement</CardTitle>
                            <CardDescription className="text-xs text-gray-500">
                              NRN transactions and payment history
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span className="text-gray-500">Last Generated:</span>
                              <span className="text-gray-300">Dec 1, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleBillingStatement}>
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={() => handleShareReport('billing-statement', { type: 'Billing Statement', account: { username: user?.user, role: user?.role }, billing: { totalNRNSpent: 1847 } })}>
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="aether-bevel-dark-light rounded-xl">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm text-gray-300">Performance Analytics</CardTitle>
                            <CardDescription className="text-xs text-gray-500">
                              Task completion rates and efficiency metrics
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span className="text-gray-500">Last Generated:</span>
                              <span className="text-gray-300">Dec 16, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handlePerformanceAnalytics}>
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={() => handleShareReport('performance-analytics', { type: 'Performance Analytics', summary: { completedTasks: 1847, successRate: 98.2, activeTasks: 127 } })}>
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="aether-bevel-dark-light rounded-xl">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm text-gray-300">Security Audit Log</CardTitle>
                            <CardDescription className="text-xs text-gray-500">
                              Access logs and security events
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span className="text-gray-500">Last Generated:</span>
                              <span className="text-gray-300">Dec 16, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleSecurityAuditLog}>
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={() => handleShareReport('security-audit-log', { type: 'Security Audit Log', summary: { securityScore: teeSecurityStatus?.security_score, attestationStatus: teeSecurityStatus?.attestation_status, threatsDetected: teeSecurityStatus?.threats_detected }, auditHistory: teeSecurityStatus?.audit_history ?? [] })}>
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="aether-bevel-dark-light rounded-xl">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm text-gray-300">Resource Utilization</CardTitle>
                            <CardDescription className="text-xs text-gray-500">
                              Detailed resource consumption patterns
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span className="text-gray-500">Last Generated:</span>
                              <span className="text-gray-300">Dec 15, 2024</span>
                            </div>
                            <div className="flex space-x-2">
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleResourceUtilization}>
                                <Download className="w-3 h-3 mr-1" />
                                Download
                              </Button>
                              <Button variant="outline" size="sm" className="flex-1 border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={() => handleShareReport('resource-utilization', { type: 'Resource Utilization', resources: { computeHours: 247.3, storageUsed: '1.2TB', apiCalls: 12400 }, teeMetrics: teeSecurityStatus?.performance_metrics ?? null })}>
                                <Share2 className="w-3 h-3 mr-1" />
                                Share
                              </Button>
                            </div>
                          </CardContent>
                        </Card>

                        <Card className="aether-bevel-dark-light rounded-xl">
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm text-gray-300">Custom Report</CardTitle>
                            <CardDescription className="text-xs text-gray-500">
                              Generate custom reports with specific parameters
                            </CardDescription>
                          </CardHeader>
                          <CardContent className="space-y-2">
                            <div className="flex justify-between text-sm">
                              <span className="text-gray-500">Status:</span>
                              <span className="text-gray-300">Ready to generate</span>
                            </div>
                            <Button variant="outline" size="sm" className="w-full border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400" onClick={handleExportReport}>
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

      {/* DVE Workspace Access Modal / Panel */}
      {selectedNode && (
        <DVEWorkspacePanel
          isOpen={cdeModalOpen}
          onClose={() => setCdeModalOpen(false)}
          node={selectedNode}
        />
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

      {/* DVE Sovereign Creation & Management Modal */}
      <DVECreationManagement
        isOpen={dveCreationModalOpen}
        onClose={() => setDveCreationModalOpen(false)}
        defaultTab="create"
      />

      {/* WebGUI Iframe Modal */}
      <WebguiIframeModal
        isOpen={webguiIframeOpen}
        onClose={() => setWebguiIframeOpen(false)}
        page={webguiIframePage}
      />

      {/* Payment Gateway Modal */}
      <PaymentGatewayModal
        isOpen={paymentGatewayOpen}
        onClose={() => setPaymentGatewayOpen(false)}
        currentBalance={1847}
      />
    </div>
  );
}

export function DashboardWrapper(props: DashboardWrapperProps) {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    }>
      <DashboardWrapperInner {...props} />
    </Suspense>
  );
}