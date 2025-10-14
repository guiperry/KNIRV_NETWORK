"use client";

import { useState, useEffect } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Progress } from "@/components/ui/progress";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useToast } from "@/hooks/use-toast";
import { useKnirvSocket } from "@/hooks/use-knirv-socket";
import { useDVENodes } from "@/hooks/use-dve-nodes";
import { useDVERental } from "@/hooks/use-dve-rental";
import { useValidationTasks } from "@/hooks/use-validation-tasks";
import { useCognitiveEngine } from "@/hooks/use-cognitive-engine";
import { useTEESecurity } from "@/hooks/use-tee-security";
import { useSystemHealth } from "@/hooks/use-system-health";
import QRCodeDisplay from "@/components/controller/qr-code-display";
import DNSManagement from "@/components/dns/dns-management";
import AgentManagement from "@/components/agents/agent-management";
import DVERentalManagement from "@/components/dve-rental/dve-rental-management";
import { useAuth } from "@/lib/auth-context";
import { DashboardWrapper } from "@/components/dashboard/dashboard-wrapper";
import { DemoModeToggle } from "@/components/admin/demo-mode-toggle";
import { useDemoMode } from "@/contexts/demo-mode-context";
import {
  Activity,
  Shield,
  Brain,
  Coins,
  Server,
  CheckCircle,
  AlertTriangle,
  Clock,
  Zap,
  Network,
  Lock,
  TrendingUp,
  Wifi,
  WifiOff,
  Bell,
  QrCode,
  Globe,
  Bot,
  CreditCard
} from "lucide-react";

interface DVENode {
  id: string;
  name: string;
  status: "online" | "offline" | "maintenance";
  cpu_usage: number;
  memory_usage: number;
  tee_type: "SGX" | "SEV-SNP" | "TDX" | "None";
  stake_amount: number;
  reputation_score: number;
  last_heartbeat: string;
}

interface ValidationTask {
  id: string;
  type: "skill_validation" | "llm_update" | "security_audit";
  status: "pending" | "running" | "completed" | "failed";
  priority: "low" | "medium" | "high" | "critical";
  assigned_node: string;
  progress: number;
  created_at: string;
  estimated_completion: string;
}

interface CognitiveEngine {
  status: "active" | "idle" | "learning" | "error";
  accuracy: number;
  tasks_processed: number;
  adaptation_rate: number;
  model_version: string;
}

interface TEESecurity {
  attestation_status: "verified" | "pending" | "failed";
  enclave_count: number;
  security_score: number;
  last_audit: string;
  threats_detected: number;
}

interface NRNStaking {
  total_staked: number;
  apy: number;
  rewards_24h: number;
  validators_count: number;
  slashing_events: number;
}

export default function Dashboard() {
  const { toast } = useToast();
  const { user } = useAuth();
  const [showQRCode, setShowQRCode] = useState(false);
  const [showDNSManagement, setShowDNSManagement] = useState(false);
  const [showAgentManagement, setShowAgentManagement] = useState(false);
  const [showDVERentalManagement, setShowDVERentalManagement] = useState(false);

  // Helper functions to check connection/setup status
  const isControllerConnected = () => {
    // Check if controller is connected (placeholder logic)
    return false;
  };

  const isDNSPorted = () => {
    // Check if DNS is ported (placeholder logic) 
    return false;
  };

  const hasAgentsAdded = () => {
    // Check if agents are added (placeholder logic)
    return false;
  };

  const hasDVERentals = () => {
    // Check if DVE rentals exist
    return (rentalStats?.active_rentals || 0) > 0;
  };

  // Use real backend hooks instead of mock data
  const { nodes: dveNodes, isLoading: dveLoading, error: dveError } = useDVENodes();
  const { rentals, stats: rentalStats, isLoading: rentalLoading } = useDVERental();
  const { tasks: validationTasks, isLoading: tasksLoading } = useValidationTasks();
  const { cognitiveEngine, isLoading: cognitiveLoading } = useCognitiveEngine();
  const { securityStatus: teeSecurityStatus, isLoading: teeLoading } = useTEESecurity();
  const { systemHealth, isLoading: healthLoading } = useSystemHealth();
  
  const {
    isConnected,
    dveNodeUpdates,
    validationTaskUpdates,
    cognitiveEngineUpdates,
    teeSecurityUpdates,
    nrnStakingUpdates,
    securityAlerts,
    systemNotifications,
    clearUpdates
  } = useKnirvSocket();

  // Error handling for backend connections
  useEffect(() => {
    if (dveError) {
      toast({
        title: "DVE Nodes Error",
        description: dveError,
        variant: "destructive",
      });
    }
  }, [dveError, toast]);



  // Handle security alerts and system notifications
  useEffect(() => {
    // Show security alerts as toasts
    securityAlerts.forEach(alert => {
      toast({
        title: `Security Alert - ${alert.severity.toUpperCase()}`,
        description: alert.message,
        variant: alert.severity === 'critical' ? 'destructive' : 'default'
      });
    });

    // Show system notifications as toasts
    systemNotifications.forEach(notification => {
      toast({
        title: notification.title,
        description: notification.message,
        variant: notification.type === 'error' ? 'destructive' :
                notification.type === 'success' ? 'default' : 'default'
      });
    });
  }, [securityAlerts, systemNotifications, toast]);

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

  const getPriorityBadge = (priority: string) => {
    switch (priority) {
      case "critical":
        return <Badge className="bg-red-500">{priority}</Badge>;
      case "high":
        return <Badge className="bg-orange-500">{priority}</Badge>;
      case "medium":
        return <Badge className="bg-yellow-500">{priority}</Badge>;
      case "low":
        return <Badge className="bg-green-500">{priority}</Badge>;
      default:
        return <Badge>{priority}</Badge>;
    }
  };

  return (
    <DashboardWrapper onRentDVE={() => setShowDVERentalManagement(true)}>
      <div className="space-y-6">
        {/* Header */}
        <div className="text-center space-y-2">
          <div className="flex items-center justify-center gap-2">
            <h1 className="text-4xl font-bold knirv-gradient-text">KNIRV-NEXUS DVE</h1>
            <div className="flex items-center gap-2">
              {isConnected ? (
                <Badge className="bg-green-500"><Wifi className="w-3 h-3 mr-1" /> Live</Badge>
              ) : (
                <Badge className="bg-red-500"><WifiOff className="w-3 h-3 mr-1" /> Offline</Badge>
              )}
              {securityAlerts.length > 0 && (
                <Badge className="bg-red-500"><Bell className="w-3 h-3 mr-1" /> {securityAlerts.length}</Badge>
              )}
            </div>
          </div>
          <p className="text-lg text-muted-foreground">
            The Crucible of Verifiable AI Intelligence
          </p>
        </div>

      {/* Getting Started Action Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card 
          className="knirv-card-gradient cursor-pointer hover:bg-primary/5 transition-colors border-white/20 hover:border-primary/50"
          onClick={() => setShowQRCode(true)}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {isControllerConnected() ? "Update Controller" : "Connect Controller"}
            </CardTitle>
            <QrCode className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {isControllerConnected() ? "Connected" : "Setup Required"}
            </div>
            <p className="text-xs text-muted-foreground">
              {isControllerConnected() 
                ? "Click to update controller settings" 
                : "Pair your device to get started"}
            </p>
          </CardContent>
        </Card>

        <Card 
          className="knirv-card-gradient cursor-pointer hover:bg-primary/5 transition-colors border-white/20 hover:border-primary/50"
          onClick={() => setShowDNSManagement(true)}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {isDNSPorted() ? "Manage DNS" : "Port DNS"}
            </CardTitle>
            <Globe className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {isDNSPorted() ? "Active" : "Setup Required"}
            </div>
            <p className="text-xs text-muted-foreground">
              {isDNSPorted() 
                ? "Click to manage DNS settings" 
                : "Configure DNS for network access"}
            </p>
          </CardContent>
        </Card>

        <Card 
          className="knirv-card-gradient cursor-pointer hover:bg-primary/5 transition-colors border-white/20 hover:border-primary/50"
          onClick={() => setShowAgentManagement(true)}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {hasAgentsAdded() ? "Manage Agents" : "Add Agents"}
            </CardTitle>
            <Bot className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {hasAgentsAdded() ? "Active" : "Setup Required"}
            </div>
            <p className="text-xs text-muted-foreground">
              {hasAgentsAdded() 
                ? "Click to manage your agents" 
                : "Deploy AI agents to the network"}
            </p>
          </CardContent>
        </Card>

        <Card 
          className="knirv-card-gradient cursor-pointer hover:bg-primary/5 transition-colors border-white/20 hover:border-primary/50"
          onClick={() => setShowDVERentalManagement(true)}
        >
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {hasDVERentals() ? "Manage DVE Rentals" : "Rent DVE Instance"}
            </CardTitle>
            <CreditCard className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {hasDVERentals() ? `${rentalStats?.active_rentals || 0} Active` : "Get Started"}
            </div>
            <p className="text-xs text-muted-foreground">
              {hasDVERentals() 
                ? "Click to manage your rentals" 
                : "Rent computing resources"}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content Tabs */}
      <Tabs defaultValue="cognitive" className="space-y-4">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="cognitive">Cognitive Engine</TabsTrigger>
          <TabsTrigger value="security">TEE Security</TabsTrigger>
          <TabsTrigger value="admin">Admin</TabsTrigger>
        </TabsList>





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
                    <p className="text-sm text-muted-foreground">Model Version</p>
                    <Badge variant="outline">{cognitiveEngine.model_version}</Badge>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </TabsContent>

        <TabsContent value="security" className="space-y-4">
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
            </div>
          )}
        </TabsContent>



        <TabsContent value="admin" className="space-y-4">
          <div className="grid gap-4">
            <DemoModeToggle />

            <Card className="knirv-card-gradient">
              <CardHeader>
                <CardTitle>System Information</CardTitle>
                <CardDescription>
                  Current system status and configuration
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="font-medium">WebSocket Status:</span>
                    <Badge variant={isConnected ? "default" : "destructive"} className="ml-2">
                      {isConnected ? "Connected" : "Disconnected"}
                    </Badge>
                  </div>
                  <div>
                    <span className="font-medium">Backend URL:</span>
                    <span className="ml-2 text-muted-foreground">http://localhost:8080</span>
                  </div>
                  <div>
                    <span className="font-medium">DVE Nodes:</span>
                    <span className="ml-2">{dveNodes.length} registered</span>
                  </div>
                  <div>
                    <span className="font-medium">Active Tasks:</span>
                    <span className="ml-2">{validationTasks.length} tasks</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
      </div>

      {/* QR Code Display Modal */}
      <QRCodeDisplay
        isOpen={showQRCode}
        onClose={() => setShowQRCode(false)}
        userId={user?.user || 'anonymous'}
        deviceType="desktop"
        capabilities={['remote_control', 'file_transfer', 'screen_share', 'dve_access']}
      />

      {/* DNS Management Modal */}
      <DNSManagement
        isOpen={showDNSManagement}
        onClose={() => setShowDNSManagement(false)}
      />

      {/* Agent Management Modal */}
      <AgentManagement
        isOpen={showAgentManagement}
        onClose={() => setShowAgentManagement(false)}
      />

      {/* DVE Rental Management Modal */}
      <DVERentalManagement
        isOpen={showDVERentalManagement}
        onClose={() => setShowDVERentalManagement(false)}
      />
    </DashboardWrapper>
  );
}