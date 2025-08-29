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
    <DashboardWrapper>
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
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowQRCode(true)}
                className="ml-2"
              >
                <QrCode className="w-4 h-4 mr-2" />
                Pair Device
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowDNSManagement(true)}
                className="ml-2"
              >
                <Globe className="w-4 h-4 mr-2" />
                DNS
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowAgentManagement(true)}
                className="ml-2"
              >
                <Bot className="w-4 h-4 mr-2" />
                Agents
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowDVERentalManagement(true)}
                className="ml-2"
              >
                <CreditCard className="w-4 h-4 mr-2" />
                DVE Rental
              </Button>
            </div>
          </div>
          <p className="text-lg text-muted-foreground">
            The Crucible of Verifiable AI Intelligence
          </p>
        </div>

      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="knirv-card-gradient">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active DVE Nodes</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{dveNodes.filter(n => n.status === 'online').length}</div>
            <p className="text-xs text-muted-foreground">
              {dveNodes.length} total nodes
            </p>
          </CardContent>
        </Card>

        <Card className="knirv-card-gradient">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Validation Tasks</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{validationTasks.filter(t => t.status === 'running').length}</div>
            <p className="text-xs text-muted-foreground">
              {validationTasks.filter(t => t.status === 'pending').length} pending
            </p>
          </CardContent>
        </Card>

        <Card className="knirv-card-gradient">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Security Score</CardTitle>
            <Shield className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{teeSecurityStatus?.security_score || 0}%</div>
            <p className="text-xs text-muted-foreground">
              {teeSecurityStatus?.enclave_count || 0} active enclaves
            </p>
          </CardContent>
        </Card>

        <Card className="knirv-card-gradient">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">DVE Rentals</CardTitle>
            <Coins className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{rentalStats?.active_rentals || 0}</div>
            <p className="text-xs text-muted-foreground">
              {rentalStats?.total_rentals || 0} total rentals
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content Tabs */}
      <Tabs defaultValue="nodes" className="space-y-4">
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="nodes">DVE Nodes</TabsTrigger>
          <TabsTrigger value="tasks">Validation Tasks</TabsTrigger>
          <TabsTrigger value="cognitive">Cognitive Engine</TabsTrigger>
          <TabsTrigger value="security">TEE Security</TabsTrigger>
          <TabsTrigger value="staking">DVE Rentals</TabsTrigger>
          <TabsTrigger value="admin">Admin</TabsTrigger>
        </TabsList>

        <TabsContent value="nodes" className="space-y-4">
          <div className="grid gap-4">
            {dveNodes.map((node) => (
              <Card key={node.id} className="knirv-card-gradient">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="flex items-center gap-2">
                        <Server className="h-5 w-5" />
                        {node.name}
                      </CardTitle>
                      <CardDescription>Node ID: {node.id}</CardDescription>
                    </div>
                    {getStatusBadge(node.status)}
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">CPU Usage</p>
                      <div className="flex items-center gap-2">
                        <Progress value={node.cpu_usage} className="flex-1" />
                        <span className="text-sm">{node.cpu_usage}%</span>
                      </div>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Memory Usage</p>
                      <div className="flex items-center gap-2">
                        <Progress value={node.memory_usage} className="flex-1" />
                        <span className="text-sm">{node.memory_usage}%</span>
                      </div>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">TEE Type</p>
                      <Badge variant="outline">{node.tee_type}</Badge>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Reputation</p>
                      <div className="flex items-center gap-2">
                        <Progress value={node.reputation_score} className="flex-1" />
                        <span className="text-sm">{node.reputation_score}%</span>
                      </div>
                    </div>
                  </div>
                  <div className="mt-4 grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">Stake Amount</p>
                      <p className="font-semibold">{node.stake_amount.toLocaleString()} NRN</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Last Heartbeat</p>
                      <p className="font-semibold">{new Date(node.last_heartbeat).toLocaleString()}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="tasks" className="space-y-4">
          <div className="grid gap-4">
            {validationTasks.map((task) => (
              <Card key={task.id} className="knirv-card-gradient">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="flex items-center gap-2">
                        <Activity className="h-5 w-5" />
                        Task {task.id}
                      </CardTitle>
                      <CardDescription>{task.type.replace('_', ' ').toUpperCase()}</CardDescription>
                    </div>
                    <div className="flex gap-2">
                      {getPriorityBadge(task.priority.toString())}
                      {getStatusBadge(task.status)}
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">Progress</p>
                      <div className="flex items-center gap-2">
                        <Progress value={task.completion_percentage || 0} className="flex-1" />
                        <span className="text-sm">{task.completion_percentage || 0}%</span>
                      </div>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Assigned Node</p>
                      <p className="font-semibold">{task.assigned_node_id || "Unassigned"}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Created</p>
                      <p className="font-semibold">{new Date(task.created_at).toLocaleString()}</p>
                    </div>
                  </div>
                  {task.estimated_completion_time && (
                    <div className="mt-4">
                      <p className="text-sm text-muted-foreground">Estimated Completion</p>
                      <p className="font-semibold">{new Date(task.estimated_completion_time).toLocaleString()}</p>
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
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

        <TabsContent value="staking" className="space-y-4">
          {rentalStats && (
            <div className="grid gap-4">
              <Card className="knirv-card-gradient">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Coins className="h-5 w-5" />
                    DVE Rental Overview
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">Total Rentals</p>
                      <p className="text-2xl font-bold">{rentalStats.total_rentals}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Active Rentals</p>
                      <p className="text-2xl font-bold">{rentalStats.active_rentals}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Total Revenue</p>
                      <p className="text-2xl font-bold">{rentalStats.total_revenue.toFixed(0)} NRN</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Avg Duration</p>
                      <p className="text-2xl font-bold">{rentalStats.average_duration.toFixed(1)}h</p>
                    </div>
                  </div>
                  {rentalStats.popular_plans.length > 0 && (
                    <div className="mt-4">
                      <p className="text-sm text-muted-foreground mb-2">Popular Plans</p>
                      <div className="space-y-1">
                        {rentalStats.popular_plans.slice(0, 3).map((plan, index) => (
                          <div key={plan.plan_id} className="flex justify-between text-sm">
                            <span>{plan.plan_name}</span>
                            <span className="font-semibold">{plan.rental_count} rentals</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
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