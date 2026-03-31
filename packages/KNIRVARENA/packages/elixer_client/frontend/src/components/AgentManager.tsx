import React from 'react';

import { useState, useEffect } from 'react';
import { Bot, Coins, Activity, Upload, Zap } from 'lucide-react';
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
  const [cortexModels, setCortexModels] = useState<Agent[]>([]);
  const [keyAgent, setKeyAgent] = useState<Agent | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [uploadingModel, setUploadingModel] = useState(false);
  const [, setSelectedFile] = useState<File | null>(null);

  // Load cortex models and key agent from AgentManagementService
  useEffect(() => {
    const loadModelsAndAgent = async () => {
      try {
        setIsLoading(true);

        // Load available cortex.wasm models
        const availableModels = await agentManagementService.getAgents();
        const cortexModelsOnly = availableModels.filter(agent => agent.type === 'wasm');
        setCortexModels(cortexModelsOnly);

        // Load key agent (first available agent or create default)
        const allAgents = await agentManagementService.getAgents();
        const keyAgentData = allAgents.find(agent => agent.type !== 'wasm') || allAgents[0];
        if (keyAgentData) {
          setKeyAgent(keyAgentData);
        }

        console.log('Cortex models loaded:', cortexModelsOnly.length, 'models, key agent:', keyAgentData?.name);
      } catch (error) {
        console.error('Failed to load models and agent:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadModelsAndAgent();

    // Refresh every 30 seconds
    const interval = setInterval(loadModelsAndAgent, 30000);
    return () => clearInterval(interval);
  }, []);

  // Handle cortex model file upload
  const handleModelUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setSelectedFile(file);
    setUploadingModel(true);

    try {
      const uploadRequest: AgentUploadRequest = {
        file,
        metadata: {
          name: file.name.replace(/\.[^/.]+$/, ''), // Remove file extension
          description: `Uploaded cortex.wasm model from ${file.name}`,
          author: 'User'
        },
        type: 'wasm'
      };

      const newModel = await agentManagementService.uploadAgent(uploadRequest);

      // Refresh cortex models list
      const updatedModels = await agentManagementService.getAgents();
      const cortexModelsOnly = updatedModels.filter(agent => agent.type === 'wasm');
      setCortexModels(cortexModelsOnly);

      console.log('Cortex model uploaded successfully:', newModel);
    } catch (error) {
      console.error('Model upload failed:', error);
      alert(`Model upload failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setUploadingModel(false);
      setSelectedFile(null);
      // Reset file input
      event.target.value = '';
    }
  };

  // Handle agent/model deployment
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



  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Available': return 'text-green-400';
      case 'Deployed': return 'text-blue-400';
      case 'Compiling': return 'text-yellow-400';
      case 'Error': return 'text-red-500';
      default: return 'text-gray-400';
    }
  };

  const getAgentAvatar = (status: string) => {
    switch (status) {
      case 'Available': return '/assets/avatar/bot_green.png';
      case 'Deployed': return '/assets/avatar/bot_blue.png';
      case 'Compiling': return '/assets/avatar/bot_orange.png';
      case 'Error': return '/assets/avatar/bot_red.png';
      default: return '/assets/avatar/bot_default.png';
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

      {/* Key Agent Section */}
      <div className="space-y-2">
        <h3 className="text-sm font-medium text-gray-400">Key Agent</h3>
        {keyAgent ? (
          <div className="p-4 bg-gray-800/50 rounded-lg border border-gray-700/50 space-y-3">
            <div className="flex items-center space-x-4">
              <img
                src={getAgentAvatar(keyAgent.status)}
                alt="Agent Avatar"
                className="w-12 h-12 rounded-full border-2 border-gray-600"
              />
              <div className="flex-1">
                <h4 className="text-white font-medium">{keyAgent.name}</h4>
                <p className="text-xs text-gray-400">{keyAgent.metadata.description}</p>
                <div className="flex items-center space-x-2 mt-1">
                  <span className={`text-sm font-medium ${getStatusColor(keyAgent.status)}`}>
                    {keyAgent.status}
                  </span>
                  {keyAgent.status === 'Available' && (
                    <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                  )}
                </div>
              </div>
            </div>
            <div className="flex items-center justify-between text-xs text-gray-500">
              <div className="flex items-center space-x-3">
                <span>Version: {keyAgent.metadata.version}</span>
                <span>Memory: {keyAgent.metadata.requirements.memory}MB</span>
                <span>CPU: {keyAgent.metadata.requirements.cpu}</span>
              </div>
              {keyAgent.lastActivity && (
                <span>Last: {new Date(keyAgent.lastActivity).toLocaleTimeString()}</span>
              )}
            </div>
            <div className="flex items-center space-x-2">
              <Zap className="w-3 h-3 text-yellow-400" />
              <span className="text-xs text-gray-400">
                {keyAgent.capabilities.join(', ')}
              </span>
            </div>
          </div>
        ) : (
          <div className="p-4 text-center text-gray-500 bg-gray-800/30 rounded-lg border border-gray-700/30">
            <Bot className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No key agent configured</p>
          </div>
        )}
      </div>

      {/* Cortex Models Upload */}
      <div className="p-3 bg-gray-800/50 rounded-lg border border-gray-700/50">
        <div className="flex items-center justify-between mb-2">
          <span className="text-white font-medium">Upload Cortex Model</span>
          <Activity className="w-4 h-4 text-blue-400" />
        </div>
        <div className="flex items-center space-x-2">
          <input
            type="file"
            accept=".wasm"
            onChange={handleModelUpload}
            disabled={uploadingModel}
            className="hidden"
            id="model-upload"
          />
          <label
            htmlFor="model-upload"
            className={`flex items-center space-x-2 px-3 py-2 rounded text-sm font-medium transition-colors cursor-pointer ${
              uploadingModel
                ? 'bg-gray-700/50 text-gray-500 cursor-not-allowed'
                : 'bg-blue-500/20 text-blue-400 hover:bg-blue-500/30'
            }`}
          >
            <Upload className="w-4 h-4" />
            <span>{uploadingModel ? 'Uploading...' : 'Upload WASM Model'}</span>
          </label>
          <span className="text-xs text-gray-400">
            Upload .wasm files for neural intelligence models
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

      {/* Cortex Models List */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-400">Cortex Models</h3>
          <span className="text-xs text-gray-500">{cortexModels.length} models</span>
        </div>
        {cortexModels.map((model) => (
          <div
            key={model.agentId}
            className="p-3 bg-gray-800/50 rounded-lg border border-gray-700/50 space-y-2"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <div className="p-1 rounded bg-blue-500/20 text-blue-400">
                  <Bot className="w-4 h-4" />
                </div>
                <button
                  onClick={() => handleAgentClick(model)}
                  className="text-white font-medium hover:text-blue-400 transition-colors cursor-pointer"
                >
                  {model.name}
                </button>
                <div className="px-2 py-1 bg-blue-500/20 text-blue-400 text-xs rounded">
                  CORTEX
                </div>
              </div>
              <div className="flex items-center space-x-2">
                <span className={`text-sm font-medium ${getStatusColor(model.status)}`}>
                  {model.status}
                </span>
                {model.status === 'Available' && (
                  <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                )}
              </div>
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Zap className="w-3 h-3 text-yellow-400" />
                <span className="text-xs text-gray-400">
                  {model.capabilities.join(', ')}
                </span>
              </div>
              <span className="text-sm font-medium text-yellow-400">
                {model.nrnCost} NRN
              </span>
            </div>

            {/* Model metadata */}
            <div className="flex items-center justify-between text-xs text-gray-500">
              <div className="flex items-center space-x-3">
                <span>Version: {model.metadata.version}</span>
                <span>Memory: {model.metadata.requirements.memory}MB</span>
                <span>CPU: {model.metadata.requirements.cpu}</span>
              </div>
              {model.lastActivity && (
                <span>Last: {new Date(model.lastActivity).toLocaleTimeString()}</span>
              )}
            </div>

            {/* Model description */}
            {model.metadata.description && (
              <div className="text-xs text-gray-400">
                {String(model.metadata.description)}
              </div>
            )}

            {selectedNRV && model.status === 'Available' && (
              <button
                onClick={() => handleAgentDeployment(model, selectedNRV)}
                disabled={nrnBalance < model.nrnCost}
                className={`w-full py-2 px-3 rounded text-sm font-medium transition-colors ${
                  nrnBalance >= model.nrnCost
                    ? 'bg-teal-500/20 text-teal-400 hover:bg-teal-500/30'
                    : 'bg-gray-700/50 text-gray-500 cursor-not-allowed'
                }`}
              >
                {nrnBalance >= model.nrnCost ? 'Deploy Model' : 'Insufficient NRN'}
              </button>
            )}
          </div>
        ))}

        {cortexModels.length === 0 && (
          <div className="p-4 text-center text-gray-500">
            <Bot className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No cortex models available</p>
            <p className="text-xs">Upload a WASM model to get started</p>
          </div>
        )}
      </div>
    </div>
  );
};
