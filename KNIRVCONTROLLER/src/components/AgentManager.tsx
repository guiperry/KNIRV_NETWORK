import React, { useState, useEffect } from 'react';
import { Bot, Zap, Shield, Users, Coins, Activity, Upload, Play, Pause, Square } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { Agent, NRV } from '../App';
import { WASMAgentManager, AgentMetadata } from '../sensory-shell/WASMAgentManager';
import { CognitiveEngine } from '../sensory-shell/CognitiveEngine';

interface RealTimeAgent {
  id: string;
  name: string;
  type: string;
  status: 'Available' | 'Busy' | 'Offline' | 'Loading' | 'Error';
  specialization: string[];
  nrnCost: number;
  metadata?: AgentMetadata;
  wasmInstance?: boolean;
  activeSkills: string[];
  memoryUsage: number;
  uptime: number;
  lastActivity: Date;
}

interface AgentManagerProps {
  agents?: Agent[]; // Legacy prop for backward compatibility
  nrvs: NRV[];
  selectedNRV: NRV | null;
  onAgentAssignment: (nrv: NRV, agent: Agent | RealTimeAgent) => void;
  nrnBalance: number;
  cognitiveEngine?: CognitiveEngine;
}

export const AgentManager: React.FC<AgentManagerProps> = ({
  agents: legacyAgents = [],
  nrvs,
  selectedNRV,
  onAgentAssignment,
  nrnBalance,
  cognitiveEngine
}) => {
  const navigate = useNavigate();
  const [realTimeAgents, setRealTimeAgents] = useState<RealTimeAgent[]>([]);
  const [wasmAgentManager, setWasmAgentManager] = useState<WASMAgentManager | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [uploadingAgent, setUploadingAgent] = useState(false);

  // Initialize WASM Agent Manager
  useEffect(() => {
    const initializeWASMManager = async () => {
      try {
        const manager = new WASMAgentManager({
          maxMemoryMB: 512,
          enableLoRAAdapters: true,
          maxConcurrentSkills: 10,
          timeoutMs: 30000
        });

        // Set up event listeners for real-time updates
        manager.on('agentLoaded', (metadata: AgentMetadata) => {
          console.log('Agent loaded:', metadata);
          updateAgentStatus(metadata.name, 'Available');
        });

        manager.on('agentError', (error: any) => {
          console.error('Agent error:', error);
          updateAgentStatus(error.agentName || 'unknown', 'Error');
        });

        manager.on('skillInvoked', (data: any) => {
          console.log('Skill invoked:', data);
          updateAgentActivity(data.agentName, data.skillId);
        });

        setWasmAgentManager(manager);
        await loadAvailableAgents(manager);
      } catch (error) {
        console.error('Failed to initialize WASM Agent Manager:', error);
      } finally {
        setIsLoading(false);
      }
    };

    initializeWASMManager();
  }, [cognitiveEngine]);

  // Load available agents from various sources
  const loadAvailableAgents = async (manager: WASMAgentManager) => {
    const agents: RealTimeAgent[] = [];

    // Add default cognitive engine agent if available
    if (cognitiveEngine) {
      agents.push({
        id: 'cognitive-engine-default',
        name: 'KNIRV Cognitive Engine',
        type: 'KNIRV-CORTEX',
        status: 'Available',
        specialization: ['cognitive-processing', 'skill-invocation', 'error-handling'],
        nrnCost: 100,
        activeSkills: [],
        memoryUsage: 0,
        uptime: Date.now(),
        lastActivity: new Date()
      });
    }

    // Add any uploaded WASM agents
    if (manager.isReady()) {
      // TODO: Get list of uploaded agents from manager
      // This would be implemented when the manager has agent listing functionality
    }

    // Merge with legacy agents for backward compatibility
    legacyAgents.forEach(legacyAgent => {
      agents.push({
        id: legacyAgent.id,
        name: legacyAgent.name,
        type: legacyAgent.type,
        status: legacyAgent.status as any,
        specialization: legacyAgent.specialization,
        nrnCost: legacyAgent.nrnCost,
        activeSkills: [],
        memoryUsage: 0,
        uptime: Date.now(),
        lastActivity: new Date()
      });
    });

    setRealTimeAgents(agents);
  };

  // Update agent status in real-time
  const updateAgentStatus = (agentName: string, status: RealTimeAgent['status']) => {
    setRealTimeAgents(prev => prev.map(agent =>
      agent.name === agentName
        ? { ...agent, status, lastActivity: new Date() }
        : agent
    ));
  };

  // Update agent activity
  const updateAgentActivity = (agentName: string, skillId: string) => {
    setRealTimeAgents(prev => prev.map(agent =>
      agent.name === agentName
        ? {
            ...agent,
            activeSkills: [...agent.activeSkills.filter(s => s !== skillId), skillId].slice(-5),
            lastActivity: new Date()
          }
        : agent
    ));
  };
  // Handle agent file upload
  const handleAgentUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file || !wasmAgentManager) return;

    setUploadingAgent(true);
    try {
      const arrayBuffer = await file.arrayBuffer();
      const wasmBytes = new Uint8Array(arrayBuffer);

      const metadata = await wasmAgentManager.loadAgent(wasmBytes, {
        name: file.name.replace('.wasm', ''),
        version: '1.0.0',
        description: 'User uploaded agent',
        capabilities: ['custom-processing'],
        author: 'User',
        uploadedAt: new Date(),
        size: file.size,
        hash: await generateFileHash(arrayBuffer)
      });

      console.log('Agent uploaded successfully:', metadata);
      await loadAvailableAgents(wasmAgentManager);
    } catch (error) {
      console.error('Failed to upload agent:', error);
      alert('Failed to upload agent. Please check the file format.');
    } finally {
      setUploadingAgent(false);
      // Reset file input
      event.target.value = '';
    }
  };

  // Generate file hash for verification
  const generateFileHash = async (arrayBuffer: ArrayBuffer): Promise<string> => {
    const hashBuffer = await crypto.subtle.digest('SHA-256', arrayBuffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'KNIRV-CORTEX': return <Bot className="w-4 h-4" />;
      case 'KNIRVANA': return <Users className="w-4 h-4" />;
      case 'DVE': return <Shield className="w-4 h-4" />;
      default: return <Bot className="w-4 h-4" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Available': return 'text-green-400';
      case 'Busy': return 'text-yellow-400';
      case 'Offline': return 'text-red-400';
      case 'Loading': return 'text-blue-400';
      case 'Error': return 'text-red-500';
      default: return 'text-gray-400';
    }
  };

  const getAgentId = (agentName: string) => {
    return agentName.toLowerCase().replace(/[^a-z0-9]/g, '-');
  };

  const handleAgentClick = (agent: RealTimeAgent) => {
    const agentId = getAgentId(agent.name);
    navigate(`/manager/agent/${agentId}`);
  };

  const formatUptime = (uptime: number) => {
    const seconds = Math.floor((Date.now() - uptime) / 1000);
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m`;
    const hours = Math.floor(minutes / 60);
    return `${hours}h`;
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'KNIRV-CORTEX': return 'bg-blue-500/20 text-blue-400';
      case 'KNIRVANA': return 'bg-purple-500/20 text-purple-400';
      case 'DVE': return 'bg-orange-500/20 text-orange-400';
      default: return 'bg-gray-500/20 text-gray-400';
    }
  };

  const availableNRVs = nrvs.filter(nrv => nrv.status === 'Mapped' || nrv.status === 'Identified');

  if (isLoading) {
    return (
      <div className="space-y-4" data-testid="agent-manager">
        <div className="flex items-center justify-center p-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400"></div>
          <span className="ml-3 text-gray-400">Loading WASM Agent Manager...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="agent-manager">
      {/* Balance Display */}
      <div className="flex items-center justify-between p-3 bg-gray-800/50 rounded-lg border border-gray-700/50">
        <div className="flex items-center space-x-2">
          <Coins className="w-5 h-5 text-yellow-400" />
          <span className="text-white font-medium">NRN Balance</span>
        </div>
        <span className="text-lg font-bold text-yellow-400">{nrnBalance.toLocaleString()}</span>
      </div>

      {/* Agent Upload */}
      <div className="p-3 bg-gray-800/50 rounded-lg border border-gray-700/50">
        <div className="flex items-center justify-between mb-2">
          <span className="text-white font-medium">Upload Custom Agent</span>
          <Activity className="w-4 h-4 text-blue-400" />
        </div>
        <div className="flex items-center space-x-2">
          <input
            type="file"
            accept=".wasm"
            onChange={handleAgentUpload}
            disabled={uploadingAgent}
            className="hidden"
            id="agent-upload"
          />
          <label
            htmlFor="agent-upload"
            className={`flex items-center space-x-2 px-3 py-2 rounded text-sm font-medium transition-colors cursor-pointer ${
              uploadingAgent
                ? 'bg-gray-700/50 text-gray-500 cursor-not-allowed'
                : 'bg-blue-500/20 text-blue-400 hover:bg-blue-500/30'
            }`}
          >
            <Upload className="w-4 h-4" />
            <span>{uploadingAgent ? 'Uploading...' : 'Upload WASM Agent'}</span>
          </label>
          <span className="text-xs text-gray-400">
            Upload .wasm files compiled with agent-core interface
          </span>
        </div>
      </div>

      {/* Selected NRV */}
      {selectedNRV && (
        <div className="p-3 bg-teal-500/10 rounded-lg border border-teal-500/20">
          <div className="flex items-center space-x-2 mb-2">
            <div className="w-2 h-2 bg-teal-400 rounded-full"></div>
            <span className="text-sm font-medium text-teal-400">Selected NRV</span>
          </div>
          <p className="text-sm text-white mb-2">{selectedNRV.problemDescription}</p>
          <span className="text-xs text-gray-400">{selectedNRV.suggestedSolutionType}</span>
        </div>
      )}

      {/* Available NRVs */}
      {availableNRVs.length > 0 && (
        <div className="space-y-2">
          <h3 className="text-sm font-medium text-gray-400">Available NRVs</h3>
          {availableNRVs.map((nrv) => (
            <div
              key={nrv.id}
              className="p-2 bg-gray-800/30 rounded border border-gray-700/30 text-sm"
            >
              <p className="text-white truncate">{nrv.problemDescription}</p>
              <span className="text-xs text-gray-400">{nrv.severity} • {nrv.status}</span>
            </div>
          ))}
        </div>
      )}

      {/* Real-Time Agent List */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-400">Available Agents</h3>
          <span className="text-xs text-gray-500">{realTimeAgents.length} agents</span>
        </div>
        {realTimeAgents.map((agent) => (
          <div
            key={agent.id}
            className="p-3 bg-gray-800/50 rounded-lg border border-gray-700/50 space-y-2"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <div className={`p-1 rounded ${getTypeColor(agent.type)}`}>
                  {getTypeIcon(agent.type)}
                </div>
                <button
                  onClick={() => handleAgentClick(agent)}
                  className="text-white font-medium hover:text-blue-400 transition-colors cursor-pointer"
                >
                  {agent.name}
                </button>
                {agent.wasmInstance && (
                  <div className="px-2 py-1 bg-purple-500/20 text-purple-400 text-xs rounded">
                    WASM
                  </div>
                )}
              </div>
              <div className="flex items-center space-x-2">
                <span className={`text-sm font-medium ${getStatusColor(agent.status)}`}>
                  {agent.status}
                </span>
                {agent.status === 'Available' && (
                  <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                )}
              </div>
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Zap className="w-3 h-3 text-yellow-400" />
                <span className="text-xs text-gray-400">
                  {agent.specialization.join(', ')}
                </span>
              </div>
              <span className="text-sm font-medium text-yellow-400">
                {agent.nrnCost} NRN
              </span>
            </div>

            {/* Real-time agent stats */}
            <div className="flex items-center justify-between text-xs text-gray-500">
              <div className="flex items-center space-x-3">
                <span>Uptime: {formatUptime(agent.uptime)}</span>
                <span>Memory: {agent.memoryUsage}MB</span>
                {agent.activeSkills.length > 0 && (
                  <span>Active: {agent.activeSkills.length}</span>
                )}
              </div>
              <span>Last: {agent.lastActivity.toLocaleTimeString()}</span>
            </div>

            {/* Active skills display */}
            {agent.activeSkills.length > 0 && (
              <div className="flex flex-wrap gap-1">
                {agent.activeSkills.slice(0, 3).map((skill, index) => (
                  <span
                    key={index}
                    className="px-2 py-1 bg-blue-500/20 text-blue-400 text-xs rounded"
                  >
                    {skill}
                  </span>
                ))}
                {agent.activeSkills.length > 3 && (
                  <span className="px-2 py-1 bg-gray-500/20 text-gray-400 text-xs rounded">
                    +{agent.activeSkills.length - 3}
                  </span>
                )}
              </div>
            )}

            {selectedNRV && agent.status === 'Available' && (
              <button
                onClick={() => onAgentAssignment(selectedNRV, agent)}
                disabled={nrnBalance < agent.nrnCost}
                className={`w-full py-2 px-3 rounded text-sm font-medium transition-colors ${
                  nrnBalance >= agent.nrnCost
                    ? 'bg-teal-500/20 text-teal-400 hover:bg-teal-500/30'
                    : 'bg-gray-700/50 text-gray-500 cursor-not-allowed'
                }`}
              >
                {nrnBalance >= agent.nrnCost ? 'Assign Agent' : 'Insufficient NRN'}
              </button>
            )}
          </div>
        ))}

        {realTimeAgents.length === 0 && (
          <div className="p-4 text-center text-gray-500">
            <Bot className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No agents available</p>
            <p className="text-xs">Upload a WASM agent to get started</p>
          </div>
        )}
      </div>
    </div>
  );
};