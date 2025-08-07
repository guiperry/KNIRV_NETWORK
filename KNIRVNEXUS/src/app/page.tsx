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
  Bell
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
  const [dveNodes, setDveNodes] = useState<DVENode[]>([]);
  const [validationTasks, setValidationTasks] = useState<ValidationTask[]>([]);
  const [cognitiveEngine, setCognitiveEngine] = useState<CognitiveEngine | null>(null);
  const [teeSecurity, setTeeSecurity] = useState<TEESecurity | null>(null);
  const [nrnStaking, setNrnStaking] = useState<NRNStaking | null>(null);
  const { toast } = useToast();
  
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

  useEffect(() => {
    // Initialize with mock data
    const mockData = {
      dveNodes: [
        {
          id: "dve-001",
          name: "KNIRV-Node-Alpha",
          status: "online" as const,
          cpu_usage: 45,
          memory_usage: 62,
          tee_type: "SGX" as const,
          stake_amount: 50000,
          reputation_score: 98,
          last_heartbeat: new Date().toISOString()
        },
        {
          id: "dve-002",
          name: "KNIRV-Node-Beta",
          status: "online" as const,
          cpu_usage: 78,
          memory_usage: 85,
          tee_type: "SEV-SNP" as const,
          stake_amount: 75000,
          reputation_score: 95,
          last_heartbeat: new Date().toISOString()
        },
        {
          id: "dve-003",
          name: "KNIRV-Node-Gamma",
          status: "maintenance" as const,
          cpu_usage: 0,
          memory_usage: 0,
          tee_type: "TDX" as const,
          stake_amount: 60000,
          reputation_score: 92,
          last_heartbeat: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
        }
      ],
      validationTasks: [
        {
          id: "task-001",
          type: "skill_validation" as const,
          status: "running" as const,
          priority: "high" as const,
          assigned_node: "dve-001",
          progress: 75,
          created_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
          estimated_completion: new Date(Date.now() + 30 * 60 * 1000).toISOString()
        },
        {
          id: "task-002",
          type: "llm_update" as const,
          status: "pending" as const,
          priority: "critical" as const,
          assigned_node: "",
          progress: 0,
          created_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
          estimated_completion: new Date(Date.now() + 4 * 60 * 60 * 1000).toISOString()
        },
        {
          id: "task-003",
          type: "security_audit" as const,
          status: "completed" as const,
          priority: "medium" as const,
          assigned_node: "dve-002",
          progress: 100,
          created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
          estimated_completion: new Date(Date.now() - 30 * 60 * 1000).toISOString()
        }
      ],
      cognitiveEngine: {
        status: "active" as const,
        accuracy: 94.5,
        tasks_processed: 15420,
        adaptation_rate: 0.85,
        model_version: "CLEAN-v2.0.1"
      },
      teeSecurity: {
        attestation_status: "verified" as const,
        enclave_count: 12,
        security_score: 98,
        last_audit: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
        threats_detected: 0
      },
      nrnStaking: {
        total_staked: 2500000,
        apy: 12.5,
        rewards_24h: 856.25,
        validators_count: 45,
        slashing_events: 0
      }
    };

    setDveNodes(mockData.dveNodes);
    setValidationTasks(mockData.validationTasks);
    setCognitiveEngine(mockData.cognitiveEngine);
    setTeeSecurity(mockData.teeSecurity);
    setNrnStaking(mockData.nrnStaking);
  }, []);

  // Handle real-time updates
  useEffect(() => {
    // Update DVE nodes with real-time data
    dveNodeUpdates.forEach(update => {
      setDveNodes(prev => prev.map(node => 
        node.id === update.id 
          ? { ...node, ...update, last_heartbeat: update.last_heartbeat }
          : node
      ));
    });

    // Update validation tasks with real-time data
    validationTaskUpdates.forEach(update => {
      setValidationTasks(prev => prev.map(task => 
        task.id === update.id 
          ? { ...task, ...update }
          : task
      ));
    });

    // Update cognitive engine with real-time data
    cognitiveEngineUpdates.forEach(update => {
      if (cognitiveEngine) {
        setCognitiveEngine(prev => prev ? { ...prev, ...update } : null);
      }
    });

    // Update TEE security with real-time data
    teeSecurityUpdates.forEach(update => {
      if (teeSecurity) {
        setTeeSecurity(prev => prev ? { ...prev, ...update } : null);
      }
    });

    // Update NRN staking with real-time data
    nrnStakingUpdates.forEach(update => {
      if (nrnStaking) {
        setNrnStaking(prev => prev ? { ...prev, ...update } : null);
      }
    });

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
  }, [
    dveNodeUpdates,
    validationTaskUpdates,
    cognitiveEngineUpdates,
    teeSecurityUpdates,
    nrnStakingUpdates,
    securityAlerts,
    systemNotifications,
    cognitiveEngine,
    teeSecurity,
    nrnStaking,
    toast
  ]);

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
    <div className="min-h-screen p-4 space-y-6">
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
          Decentralized Validation Environment - The Crucible of Verifiable AI Intelligence
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
            <div className="text-2xl font-bold">{teeSecurity?.security_score || 0}%</div>
            <p className="text-xs text-muted-foreground">
              {teeSecurity?.enclave_count || 0} active enclaves
            </p>
          </CardContent>
        </Card>

        <Card className="knirv-card-gradient">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">NRN Staked</CardTitle>
            <Coins className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{((nrnStaking?.total_staked || 0) / 1000000).toFixed(1)}M</div>
            <p className="text-xs text-muted-foreground">
              {nrnStaking?.apy || 0}% APY
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content Tabs */}
      <Tabs defaultValue="nodes" className="space-y-4">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="nodes">DVE Nodes</TabsTrigger>
          <TabsTrigger value="tasks">Validation Tasks</TabsTrigger>
          <TabsTrigger value="cognitive">Cognitive Engine</TabsTrigger>
          <TabsTrigger value="security">TEE Security</TabsTrigger>
          <TabsTrigger value="staking">NRN Staking</TabsTrigger>
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
                      {getPriorityBadge(task.priority)}
                      {getStatusBadge(task.status)}
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">Progress</p>
                      <div className="flex items-center gap-2">
                        <Progress value={task.progress} className="flex-1" />
                        <span className="text-sm">{task.progress}%</span>
                      </div>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Assigned Node</p>
                      <p className="font-semibold">{task.assigned_node || "Unassigned"}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Created</p>
                      <p className="font-semibold">{new Date(task.created_at).toLocaleString()}</p>
                    </div>
                  </div>
                  {task.estimated_completion && (
                    <div className="mt-4">
                      <p className="text-sm text-muted-foreground">Estimated Completion</p>
                      <p className="font-semibold">{new Date(task.estimated_completion).toLocaleString()}</p>
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
          {teeSecurity && (
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
                      {getStatusBadge(teeSecurity.attestation_status)}
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Security Score</p>
                      <div className="flex items-center gap-2">
                        <Progress value={teeSecurity.security_score} className="flex-1" />
                        <span className="text-sm">{teeSecurity.security_score}%</span>
                      </div>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Active Enclaves</p>
                      <p className="text-2xl font-bold">{teeSecurity.enclave_count}</p>
                    </div>
                  </div>
                  <div className="mt-4 grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">Last Audit</p>
                      <p className="font-semibold">{new Date(teeSecurity.last_audit).toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Threats Detected</p>
                      <p className="font-semibold">{teeSecurity.threats_detected}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
              
              {teeSecurity.threats_detected > 0 && (
                <Alert>
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>
                    {teeSecurity.threats_detected} potential threats detected. Security team has been notified.
                  </AlertDescription>
                </Alert>
              )}
            </div>
          )}
        </TabsContent>

        <TabsContent value="staking" className="space-y-4">
          {nrnStaking && (
            <div className="grid gap-4">
              <Card className="knirv-card-gradient">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Coins className="h-5 w-5" />
                    NRN Staking Overview
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">Total Staked</p>
                      <p className="text-2xl font-bold">{(nrnStaking.total_staked / 1000000).toFixed(1)}M NRN</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">APY</p>
                      <p className="text-2xl font-bold">{nrnStaking.apy}%</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">24h Rewards</p>
                      <p className="text-2xl font-bold">{nrnStaking.rewards_24h.toFixed(2)} NRN</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Validators</p>
                      <p className="text-2xl font-bold">{nrnStaking.validators_count}</p>
                    </div>
                  </div>
                  <div className="mt-4">
                    <p className="text-sm text-muted-foreground">Slashing Events (24h)</p>
                    <p className="font-semibold">{nrnStaking.slashing_events}</p>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}