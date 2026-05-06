"use client";

import { useEffect } from "react";
import { useRouter } from 'next/navigation';
import { useAuth } from "@/lib/auth-context";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Progress } from "@/components/ui/progress";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useToast } from "@/hooks/use-toast";
import { useKnirvSocket } from "@/hooks/use-knirv-socket";
import { useDVENodes } from "@/hooks/use-dve-nodes";
import { useValidationTasks } from "@/hooks/use-validation-tasks";
import { useSystemHealth } from "@/hooks/use-system-health";
import { DVENodesPanel } from "@/components/dashboard/dve-nodes-panel";
import { CognitiveEnginePanel } from "@/components/dashboard/cognitive-engine-panel";
import { ActiveMemoryPanel } from "@/components/dashboard/active-memory/active-memory-panel";
import { PluginVaultPanel } from "@/components/dashboard/active-memory/plugin-vault-panel";

import { DashboardWrapper } from "@/components/dashboard/dashboard-wrapper";
import { DemoModeToggle } from "@/components/admin/demo-mode-toggle";
import { DHTToggle } from "@/components/admin/dht-toggle";
import { useDemoMode } from "@/contexts/demo-mode-context";
import { useDHT } from "@/contexts/dht-context";
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
  Layers,
  CreditCard,
  Database
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
  const router = useRouter();
  const { user } = useAuth();

  useEffect(() => {
    const token = localStorage.getItem('knirv_nexus_token') || localStorage.getItem('knirv_auth_token');
    if (!token) {
      router.push('/login');
      return;
    }
  }, [router]);

  // Use real backend hooks instead of mock data
  const { nodes: dveNodes, isLoading: dveLoading, error: dveError } = useDVENodes();
  const { tasks: validationTasks, isLoading: tasksLoading } = useValidationTasks();
  const { systemHealth, isLoading: healthLoading } = useSystemHealth();
  
  const {
    isConnected,
    dveNodeUpdates,
    validationTaskUpdates,
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
    if (securityAlerts.length === 0 && systemNotifications.length === 0) return;

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

    // Clear updates after processing to prevent redundant toasts on next render
    clearUpdates();
  }, [securityAlerts, systemNotifications, toast, clearUpdates]);

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
    <DashboardWrapper 
      onRentDVE={() => {}}
    >
      <div className="space-y-6">
        <>
            <div className="text-center space-y-2">
              <div className="flex items-center justify-center gap-2">
                <h1 className="text-4xl font-bold knirv-gradient-text">KNIRV-SERVER DVE</h1>
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

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <Card 
                className="knirv-card-gradient cursor-pointer hover:bg-primary/5 transition-colors border-white/20 hover:border-primary/50"
              >
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    Active Memory Fabric
                  </CardTitle>
                  <Database className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">Encrypted</div>
                  <p className="text-xs text-muted-foreground">
                    PQC Markdown persistence active
                  </p>
                </CardContent>
              </Card>

              <Card 
                className="knirv-card-gradient cursor-pointer hover:bg-primary/5 transition-colors border-white/20 hover:border-primary/50"
              >
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    Reasoning Graph
                  </CardTitle>
                  <Network className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">142 Traces</div>
                  <p className="text-xs text-muted-foreground">
                    Context records in .md fabric
                  </p>
                </CardContent>
              </Card>

              <Card 
                className="knirv-card-gradient cursor-pointer hover:bg-primary/5 transition-colors border-white/20 hover:border-primary/50"
              >
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    Solution Vault
                  </CardTitle>
                  <Lock className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">28 Nodes</div>
                  <p className="text-xs text-muted-foreground">
                    Verifiable executable logic
                  </p>
                </CardContent>
              </Card>

              <Card 
                className="knirv-card-gradient cursor-pointer hover:bg-primary/5 transition-colors border-white/20 hover:border-primary/50"
              >
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    P2P Transport
                  </CardTitle>
                  <Globe className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">TURN Active</div>
                  <p className="text-xs text-muted-foreground">
                    Secure NAT traversal established
                  </p>
                </CardContent>
              </Card>
            </div>

            <Tabs defaultValue="fabric" className="space-y-4">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="fabric">Markdown Fabric</TabsTrigger>
                <TabsTrigger value="admin">Admin</TabsTrigger>
              </TabsList>

              <TabsContent value="fabric" className="space-y-4">
                <Tabs defaultValue="traces">
                  <TabsList className="mb-4">
                    <TabsTrigger value="traces">Reasoning Explorer</TabsTrigger>
                    <TabsTrigger value="vault">Solution Vault</TabsTrigger>
                  </TabsList>
                  <TabsContent value="traces">
                    <ActiveMemoryPanel />
                  </TabsContent>
                  <TabsContent value="vault">
                    <PluginVaultPanel />
                  </TabsContent>
                </Tabs>
              </TabsContent>

              <TabsContent value="admin" className="space-y-4">
                <div className="grid gap-4">
                  <DemoModeToggle />
                  <DHTToggle />
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
                          <span className="ml-2 text-muted-foreground">http://localhost:8090</span>
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
          </>
        </div>
    </DashboardWrapper>
  );
}
