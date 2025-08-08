import React, { useState, useEffect } from 'react';
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
} from 'lucide-react';

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

interface NexusDashboardProps {
  isConnected: boolean;
  alerts: number;
}

export function NexusDashboard({ isConnected, alerts }: NexusDashboardProps) {
  const [activeTab, setActiveTab] = useState('nodes');
  const [dveNodes, setDveNodes] = useState<DVENode[]>([]);
  const [validationTasks, setValidationTasks] = useState<ValidationTask[]>([]);
  const [cognitiveEngine, setCognitiveEngine] = useState<CognitiveEngine | null>(null);
  const [teeSecurity, setTeeSecurity] = useState<TEESecurity | null>(null);
  const [nrnStaking, setNrnStaking] = useState<NRNStaking | null>(null);

  useEffect(() => {
    // Initialize with mock data that matches the original NEXUS
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

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "online":
      case "active":
      case "completed":
      case "verified":
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800"><CheckCircle className="w-3 h-3 mr-1" /> {status}</span>;
      case "offline":
      case "failed":
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800"><AlertTriangle className="w-3 h-3 mr-1" /> {status}</span>;
      case "maintenance":
      case "pending":
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800"><Clock className="w-3 h-3 mr-1" /> {status}</span>;
      case "running":
      case "learning":
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800"><Activity className="w-3 h-3 mr-1" /> {status}</span>;
      default:
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">{status}</span>;
    }
  };

  const getPriorityBadge = (priority: string) => {
    switch (priority) {
      case "critical":
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">{priority}</span>;
      case "high":
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-orange-100 text-orange-800">{priority}</span>;
      case "medium":
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">{priority}</span>;
      case "low":
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">{priority}</span>;
      default:
        return <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">{priority}</span>;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="text-center space-y-2">
        <div className="flex items-center justify-center gap-2">
          <h1 className="text-4xl font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">KNIRV-NEXUS DVE</h1>
          <div className="flex items-center gap-2">
            {isConnected ? (
              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800"><Wifi className="w-3 h-3 mr-1" /> Live</span>
            ) : (
              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800"><WifiOff className="w-3 h-3 mr-1" /> Offline</span>
            )}
            {alerts > 0 && (
              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800"><Bell className="w-3 h-3 mr-1" /> {alerts}</span>
            )}
          </div>
        </div>
        <p className="text-lg text-gray-300">
          The Crucible of Verifiable AI Intelligence
        </p>
      </div>

      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-sm font-medium text-gray-300">Active DVE Nodes</h3>
            <Server className="h-4 w-4 text-gray-400" />
          </div>
          <div className="text-2xl font-bold text-white">{dveNodes.filter(n => n.status === 'online').length}</div>
          <p className="text-xs text-gray-400">
            {dveNodes.length} total nodes
          </p>
        </div>

        <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-sm font-medium text-gray-300">Validation Tasks</h3>
            <Activity className="h-4 w-4 text-gray-400" />
          </div>
          <div className="text-2xl font-bold text-white">{validationTasks.filter(t => t.status === 'running').length}</div>
          <p className="text-xs text-gray-400">
            {validationTasks.filter(t => t.status === 'pending').length} pending
          </p>
        </div>

        <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-sm font-medium text-gray-300">Security Score</h3>
            <Shield className="h-4 w-4 text-gray-400" />
          </div>
          <div className="text-2xl font-bold text-white">{teeSecurity?.security_score || 0}%</div>
          <p className="text-xs text-gray-400">
            {teeSecurity?.enclave_count || 0} active enclaves
          </p>
        </div>

        <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-sm font-medium text-gray-300">NRN Staked</h3>
            <Coins className="h-4 w-4 text-gray-400" />
          </div>
          <div className="text-2xl font-bold text-white">{((nrnStaking?.total_staked || 0) / 1000000).toFixed(1)}M</div>
          <p className="text-xs text-gray-400">
            {nrnStaking?.apy || 0}% APY
          </p>
        </div>
      </div>

      {/* Main Content Tabs */}
      <div className="space-y-4">
        <div className="flex space-x-8 border-b border-white/20">
          {[
            { id: 'nodes', label: 'DVE Nodes', icon: Server },
            { id: 'tasks', label: 'Validation Tasks', icon: Activity },
            { id: 'cognitive', label: 'Cognitive Engine', icon: Brain },
            { id: 'security', label: 'TEE Security', icon: Shield },
            { id: 'staking', label: 'NRN Staking', icon: Coins }
          ].map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center space-x-2 py-4 px-2 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-400'
                    : 'border-transparent text-gray-400 hover:text-gray-200'
                }`}
              >
                <Icon className="w-4 h-4" />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>

        {/* DVE Nodes Tab */}
        {activeTab === 'nodes' && (
          <div className="grid gap-4">
            {dveNodes.map((node) => (
              <div key={node.id} className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
                <div className="flex items-center justify-between mb-4">
                  <div>
                    <h3 className="flex items-center gap-2 text-lg font-semibold text-white">
                      <Server className="h-5 w-5" />
                      {node.name}
                    </h3>
                    <p className="text-gray-400">Node ID: {node.id}</p>
                  </div>
                  {getStatusBadge(node.status)}
                </div>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div>
                    <p className="text-sm text-gray-400">CPU Usage</p>
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-700 rounded-full h-2">
                        <div className="bg-blue-500 h-2 rounded-full" style={{width: `${node.cpu_usage}%`}}></div>
                      </div>
                      <span className="text-sm text-white">{node.cpu_usage}%</span>
                    </div>
                  </div>
                  <div>
                    <p className="text-sm text-gray-400">Memory Usage</p>
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-700 rounded-full h-2">
                        <div className="bg-green-500 h-2 rounded-full" style={{width: `${node.memory_usage}%`}}></div>
                      </div>
                      <span className="text-sm text-white">{node.memory_usage}%</span>
                    </div>
                  </div>
                  <div>
                    <p className="text-sm text-gray-400">TEE Type</p>
                    <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-purple-100 text-purple-800">{node.tee_type}</span>
                  </div>
                  <div>
                    <p className="text-sm text-gray-400">Reputation</p>
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-700 rounded-full h-2">
                        <div className="bg-yellow-500 h-2 rounded-full" style={{width: `${node.reputation_score}%`}}></div>
                      </div>
                      <span className="text-sm text-white">{node.reputation_score}%</span>
                    </div>
                  </div>
                </div>
                <div className="mt-4 grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-gray-400">Stake Amount</p>
                    <p className="font-semibold text-white">{node.stake_amount.toLocaleString()} NRN</p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-400">Last Heartbeat</p>
                    <p className="font-semibold text-white">{new Date(node.last_heartbeat).toLocaleString()}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Validation Tasks Tab */}
        {activeTab === 'tasks' && (
          <div className="grid gap-4">
            {validationTasks.map((task) => (
              <div key={task.id} className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
                <div className="flex items-center justify-between mb-4">
                  <div>
                    <h3 className="flex items-center gap-2 text-lg font-semibold text-white">
                      <Activity className="h-5 w-5" />
                      Task {task.id}
                    </h3>
                    <p className="text-gray-400">{task.type.replace('_', ' ').toUpperCase()}</p>
                  </div>
                  <div className="flex gap-2">
                    {getPriorityBadge(task.priority)}
                    {getStatusBadge(task.status)}
                  </div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div>
                    <p className="text-sm text-gray-400">Progress</p>
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-700 rounded-full h-2">
                        <div className="bg-blue-500 h-2 rounded-full" style={{width: `${task.progress}%`}}></div>
                      </div>
                      <span className="text-sm text-white">{task.progress}%</span>
                    </div>
                  </div>
                  <div>
                    <p className="text-sm text-gray-400">Assigned Node</p>
                    <p className="font-semibold text-white">{task.assigned_node || "Unassigned"}</p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-400">Created</p>
                    <p className="font-semibold text-white">{new Date(task.created_at).toLocaleString()}</p>
                  </div>
                </div>
                {task.estimated_completion && (
                  <div className="mt-4">
                    <p className="text-sm text-gray-400">Estimated Completion</p>
                    <p className="font-semibold text-white">{new Date(task.estimated_completion).toLocaleString()}</p>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Cognitive Engine Tab */}
        {activeTab === 'cognitive' && cognitiveEngine && (
          <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
            <h3 className="flex items-center gap-2 text-lg font-semibold text-white mb-6">
              <Brain className="h-5 w-5" />
              Cognitive Engine Status
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div>
                <p className="text-sm text-gray-400">Status</p>
                {getStatusBadge(cognitiveEngine.status)}
              </div>
              <div>
                <p className="text-sm text-gray-400">Accuracy</p>
                <div className="flex items-center gap-2">
                  <div className="flex-1 bg-gray-700 rounded-full h-2">
                    <div className="bg-green-500 h-2 rounded-full" style={{width: `${cognitiveEngine.accuracy}%`}}></div>
                  </div>
                  <span className="text-sm text-white">{cognitiveEngine.accuracy}%</span>
                </div>
              </div>
              <div>
                <p className="text-sm text-gray-400">Tasks Processed</p>
                <p className="text-2xl font-bold text-white">{cognitiveEngine.tasks_processed.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">Adaptation Rate</p>
                <div className="flex items-center gap-2">
                  <div className="flex-1 bg-gray-700 rounded-full h-2">
                    <div className="bg-purple-500 h-2 rounded-full" style={{width: `${cognitiveEngine.adaptation_rate * 100}%`}}></div>
                  </div>
                  <span className="text-sm text-white">{(cognitiveEngine.adaptation_rate * 100).toFixed(1)}%</span>
                </div>
              </div>
            </div>
            <div className="mt-4">
              <p className="text-sm text-gray-400">Model Version</p>
              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">{cognitiveEngine.model_version}</span>
            </div>
          </div>
        )}

        {/* TEE Security Tab */}
        {activeTab === 'security' && teeSecurity && (
          <div className="space-y-6">
            <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
              <h3 className="flex items-center gap-2 text-lg font-semibold text-white mb-6">
                <Shield className="h-5 w-5" />
                TEE Security Status
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <p className="text-sm text-gray-400">Attestation Status</p>
                  {getStatusBadge(teeSecurity.attestation_status)}
                </div>
                <div>
                  <p className="text-sm text-gray-400">Security Score</p>
                  <div className="flex items-center gap-2">
                    <div className="flex-1 bg-gray-700 rounded-full h-2">
                      <div className="bg-green-500 h-2 rounded-full" style={{width: `${teeSecurity.security_score}%`}}></div>
                    </div>
                    <span className="text-sm text-white">{teeSecurity.security_score}%</span>
                  </div>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Active Enclaves</p>
                  <p className="text-2xl font-bold text-white">{teeSecurity.enclave_count}</p>
                </div>
              </div>
              <div className="mt-4 grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-400">Last Audit</p>
                  <p className="font-semibold text-white">{new Date(teeSecurity.last_audit).toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Threats Detected</p>
                  <p className="font-semibold text-white">{teeSecurity.threats_detected}</p>
                </div>
              </div>
            </div>

            {teeSecurity.threats_detected > 0 && (
              <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-4">
                <div className="flex items-center space-x-2">
                  <AlertTriangle className="h-4 w-4 text-red-400" />
                  <span className="text-red-400 font-medium">
                    {teeSecurity.threats_detected} potential threats detected. Security team has been notified.
                  </span>
                </div>
              </div>
            )}
          </div>
        )}

        {/* NRN Staking Tab */}
        {activeTab === 'staking' && nrnStaking && (
          <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
            <h3 className="flex items-center gap-2 text-lg font-semibold text-white mb-6">
              <Coins className="h-5 w-5" />
              NRN Staking Overview
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div>
                <p className="text-sm text-gray-400">Total Staked</p>
                <p className="text-2xl font-bold text-white">{(nrnStaking.total_staked / 1000000).toFixed(1)}M NRN</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">APY</p>
                <p className="text-2xl font-bold text-white">{nrnStaking.apy}%</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">24h Rewards</p>
                <p className="text-2xl font-bold text-white">{nrnStaking.rewards_24h.toFixed(2)} NRN</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">Validators</p>
                <p className="text-2xl font-bold text-white">{nrnStaking.validators_count}</p>
              </div>
            </div>
            <div className="mt-4">
              <p className="text-sm text-gray-400">Slashing Events (24h)</p>
              <p className="font-semibold text-white">{nrnStaking.slashing_events}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
