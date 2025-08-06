import React from 'react';
import { Bot, Zap, Shield, Users, Coins } from 'lucide-react';
import { Agent, NRV } from '../App';

interface AgentManagerProps {
  agents: Agent[];
  nrvs: NRV[];
  selectedNRV: NRV | null;
  onAgentAssignment: (nrv: NRV, agent: Agent) => void;
  nrnBalance: number;
}

export const AgentManager: React.FC<AgentManagerProps> = ({
  agents,
  nrvs,
  selectedNRV,
  onAgentAssignment,
  nrnBalance
}) => {
  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'KNIRV-AGENTIFIER': return <Bot className="w-4 h-4" />;
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
      default: return 'text-gray-400';
    }
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'KNIRV-AGENTIFIER': return 'bg-blue-500/20 text-blue-400';
      case 'KNIRVANA': return 'bg-purple-500/20 text-purple-400';
      case 'DVE': return 'bg-orange-500/20 text-orange-400';
      default: return 'bg-gray-500/20 text-gray-400';
    }
  };

  const availableNRVs = nrvs.filter(nrv => nrv.status === 'Mapped' || nrv.status === 'Identified');

  return (
    <div className="space-y-4">
      {/* Balance Display */}
      <div className="flex items-center justify-between p-3 bg-gray-800/50 rounded-lg border border-gray-700/50">
        <div className="flex items-center space-x-2">
          <Coins className="w-5 h-5 text-yellow-400" />
          <span className="text-white font-medium">NRN Balance</span>
        </div>
        <span className="text-lg font-bold text-yellow-400">{nrnBalance.toLocaleString()}</span>
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

      {/* Agent List */}
      <div className="space-y-2">
        <h3 className="text-sm font-medium text-gray-400">Available Agents</h3>
        {agents.map((agent) => (
          <div
            key={agent.id}
            className="p-3 bg-gray-800/50 rounded-lg border border-gray-700/50 space-y-2"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <div className={`p-1 rounded ${getTypeColor(agent.type)}`}>
                  {getTypeIcon(agent.type)}
                </div>
                <span className="text-white font-medium">{agent.name}</span>
              </div>
              <span className={`text-sm font-medium ${getStatusColor(agent.status)}`}>
                {agent.status}
              </span>
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
      </div>
    </div>
  );
};