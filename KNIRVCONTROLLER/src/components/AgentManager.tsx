import * as React from 'react';
import { useState, useEffect } from 'react';
import { Bot, Shield, Users, Coins, Activity, Upload, Zap } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { NRV } from '../App';
import { agentManagementService, Agent, AgentUploadRequest, AgentDeploymentRequest } from '../services/AgentManagementService';
import { walletIntegrationService } from '../services/WalletIntegrationService';
import { CognitiveEngine } from '../sensory-shell/CognitiveEngine';

interface AgentManagerProps {
  agents?: Agent[]; // Legacy prop for backward compatibility
  nrvs: NRV[];
  selectedNRV: NRV | null;
  onAgentAssignment: (nrv: NRV, agent: Agent) => void;
  nrnBalance: number;
  cognitiveEngine?: CognitiveEngine;
}

export const AgentManager: React.FC<AgentManagerProps> = ({
  agents: _legacyAgents = [],
  nrvs,
  selectedNRV,
  onAgentAssignment,
  nrnBalance,
  cognitiveEngine: _cognitiveEngine
}) => {
  const navigate = useNavigate();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [deployedAgents, setDeployedAgents] = useState<Agent[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [uploadingAgent, setUploadingAgent] = useState(false);
  const [, setSelectedFile] = useState<File | null>(null);

  // Load agents from AgentManagementService
  useEffect(() => {
    const loadAgents = async () => {
      try {
        setIsLoading(true);

        // Load available agents
        const availableAgents = await agentManagementService.getAgents();
        setAgents(availableAgents);

        // Load deployed agents
        const deployedAgentsList = await agentManagementService.getDeployedAgents();
        setDeployedAgents(deployedAgentsList);

        console.log('Agents loaded:', availableAgents.length, 'available,', deployedAgentsList.length, 'deployed');
      } catch (error) {
        console.error('Failed to load agents:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadAgents();

    // Refresh agents every 30 seconds
    const interval = setInterval(loadAgents, 30000);
    return () => clearInterval(interval);
  }, []);

  // Handle agent file upload
  const handleAgentUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setSelectedFile(file);
    setUploadingAgent(true);

    try {
      // Determine agent type based on file extension
      let agentType: 'wasm' | 'lora' | 'hybrid' = 'wasm';
      if (file.name.endsWith('.wasm')) {
        agentType = 'wasm';
      } else if (file.name.endsWith('.json') || file.name.endsWith('.lora')) {
        agentType = 'lora';
      } else {
        agentType = 'hybrid';
      }

      const uploadRequest: AgentUploadRequest = {
        file,
        metadata: {
          name: file.name.replace(/\.[^/.]+$/, ''), // Remove file extension
          description: `Uploaded agent from ${file.name}`,
          author: 'User'
        },
        type: agentType
      };

      const newAgent = await agentManagementService.uploadAgent(uploadRequest);

      // Refresh agents list
      const updatedAgents = await agentManagementService.getAgents();
      setAgents(updatedAgents);

      console.log('Agent uploaded successfully:', newAgent);
    } catch (error) {
      console.error('Agent upload failed:', error);
      alert(`Agent upload failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setUploadingAgent(false);
      setSelectedFile(null);
      // Reset file input
      event.target.value = '';
    }
  };

  // Handle agent deployment
  const handleAgentDeployment = async (agent: Agent, targetNRV?: NRV) => {
    try {
      // Check wallet balance
      const currentAccount = walletIntegrationService.getCurrentAccount();
      if (!currentAccount) {
        alert('Please connect your wallet first');
        return;
      }

      if (nrnBalance < agent.nrnCost) {
        alert(`Insufficient NRN balance. Required: ${agent.nrnCost}, Available: ${nrnBalance}`);
        return;
      }

      const deploymentRequest: AgentDeploymentRequest = {
        agentId: agent.agentId,
        targetNRV: targetNRV?.id,
        configuration: {},
        resources: {
          memory: agent.metadata.requirements.memory,
          cpu: agent.metadata.requirements.cpu,
          timeout: 300000 // 5 minutes
        }
      };

      const deploymentId = await agentManagementService.deployAgent(deploymentRequest);

      // Refresh deployed agents list
      const updatedDeployedAgents = await agentManagementService.getDeployedAgents();
      setDeployedAgents(updatedDeployedAgents);

      console.log('Agent deployed successfully:', deploymentId);

      // Call the parent callback for UI updates
      if (targetNRV) {
        onAgentAssignment(targetNRV, agent);
      }
    } catch (error) {
      console.error('Agent deployment failed:', error);
      alert(`Agent deployment failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  // Handle agent click for navigation
  const handleAgentClick = (agent: Agent) => {
    navigate(`/agents/${agent.agentId}`);
  };



  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'wasm': return <Bot className="w-4 h-4" />;
      case 'lora': return <Users className="w-4 h-4" />;
      case 'hybrid': return <Shield className="w-4 h-4" />;
      default: return <Bot className="w-4 h-4" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Available': return 'text-green-400';
      case 'Deployed': return 'text-blue-400';
      case 'Compiling': return 'text-yellow-400';
      case 'Error': return 'text-red-500';
      default: return 'text-gray-400';
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'wasm': return 'bg-blue-500/20 text-blue-400';
      case 'lora': return 'bg-purple-500/20 text-purple-400';
      case 'hybrid': return 'bg-orange-500/20 text-orange-400';
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

      {/* Available Agents List */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-400">Available Agents</h3>
          <span className="text-xs text-gray-500">{agents.length} agents</span>
        </div>
        {agents.map((agent) => (
          <div
            key={agent.agentId}
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
                <div className="px-2 py-1 bg-purple-500/20 text-purple-400 text-xs rounded">
                  {agent.type.toUpperCase()}
                </div>
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
                  {agent.capabilities.join(', ')}
                </span>
              </div>
              <span className="text-sm font-medium text-yellow-400">
                {agent.nrnCost} NRN
              </span>
            </div>

            {/* Agent metadata */}
            <div className="flex items-center justify-between text-xs text-gray-500">
              <div className="flex items-center space-x-3">
                <span>Version: {agent.metadata.version}</span>
                <span>Memory: {agent.metadata.requirements.memory}MB</span>
                <span>CPU: {agent.metadata.requirements.cpu}</span>
              </div>
              {agent.lastActivity && (
                <span>Last: {new Date(agent.lastActivity).toLocaleTimeString()}</span>
              )}
            </div>

            {/* Agent description */}
            {agent.metadata.description && (
              <div className="text-xs text-gray-400">
                {String(agent.metadata.description)}
              </div>
            )}

            {selectedNRV && agent.status === 'Available' && (
              <button
                onClick={() => handleAgentDeployment(agent, selectedNRV)}
                disabled={nrnBalance < agent.nrnCost}
                className={`w-full py-2 px-3 rounded text-sm font-medium transition-colors ${
                  nrnBalance >= agent.nrnCost
                    ? 'bg-teal-500/20 text-teal-400 hover:bg-teal-500/30'
                    : 'bg-gray-700/50 text-gray-500 cursor-not-allowed'
                }`}
              >
                {nrnBalance >= agent.nrnCost ? 'Deploy Agent' : 'Insufficient NRN'}
              </button>
            )}
          </div>
        ))}

        {agents.length === 0 && (
          <div className="p-4 text-center text-gray-500">
            <Bot className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No agents available</p>
            <p className="text-xs">Upload a WASM agent to get started</p>
          </div>
        )}
      </div>

      {/* Deployed Agents Section */}
      {deployedAgents.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium text-gray-400">Deployed Agents</h3>
            <span className="text-xs text-gray-500">{deployedAgents.length} deployed</span>
          </div>
          {deployedAgents.map((agent) => (
            <div
              key={`deployed-${agent.agentId}`}
              className="p-3 bg-blue-500/10 rounded-lg border border-blue-500/20 space-y-2"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <div className={`p-1 rounded ${getTypeColor(agent.type)}`}>
                    {getTypeIcon(agent.type)}
                  </div>
                  <span className="text-white font-medium">{agent.name}</span>
                  <div className="px-2 py-1 bg-blue-500/20 text-blue-400 text-xs rounded">
                    DEPLOYED
                  </div>
                </div>
                <span className="text-sm font-medium text-blue-400">
                  {agent.status}
                </span>
              </div>
              <div className="text-xs text-gray-400">
                {agent.capabilities.join(', ')}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};