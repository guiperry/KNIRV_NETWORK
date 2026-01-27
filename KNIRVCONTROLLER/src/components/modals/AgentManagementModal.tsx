import React, { useState, useMemo } from 'react';

import {
  Bot,
  Wallet,
  Trophy,
  Star,
  ChevronLeft,
  ChevronRight,
  Zap,
  Shield,
  Brain,
  Activity,
  Settings,
  PlayCircle
} from 'lucide-react';
import { Agent } from '../../services/AgentManagementService';

// Types for Agent Management
interface SubAgent {
  id: string;
  name: string;
  status: 'idle' | 'training' | 'ready' | 'deployed';
  capabilities: string[];
  expertise: string[];
  trainingProgress: number;
  nrnCost: number;
  description: string;
}

interface KeyAgentData {
  name: string;
  avatar: string;
  status: string;
  capabilities: string[];
  version: string;
  totalTrained: number;
  successRate: number;
}

interface AgentManagementModalProps {
  isOpen: boolean;
  onClose: () => void;
  keyAgent: Agent | null;
  subAgents: SubAgent[];
  nrnBalance: number;
  onTrainSubAgent: (subAgentId: string, userInstructions: string) => void;
  onDeployAgent: (agentId: string) => void;
}

export const AgentManagementModal: React.FC<AgentManagementModalProps> = ({
  isOpen,
  onClose,
  keyAgent,
  subAgents,
  nrnBalance,
  onTrainSubAgent,
  onDeployAgent
}) => {
  const [selectedSubAgentId, setSelectedSubAgentId] = useState<string | null>(null);
  const [userInstructions, setUserInstructions] = useState('');
  const [isTraining, setIsTraining] = useState(false);

  const selectedSubAgent = useMemo(() => 
    subAgents.find(sa => sa.id === selectedSubAgentId), 
    [subAgents, selectedSubAgentId]
  );

  // Convert keyAgent to KeyAgentData format
  const keyAgentData: KeyAgentData | null = keyAgent ? {
    name: keyAgent.name,
    avatar: '/assets/avatar/key-agent.png',
    status: keyAgent.status,
    capabilities: keyAgent.capabilities,
    version: keyAgent.metadata.version,
    totalTrained: Math.floor(Math.random() * 50) + 10, // Mock data
    successRate: Math.floor(Math.random() * 30) + 70 // Mock data
  } : null;

  const handleStartTraining = () => {
    if (!selectedSubAgentId || !userInstructions.trim()) return;
    
    setIsTraining(true);
    onTrainSubAgent(selectedSubAgentId, userInstructions);
    
    // Simulate training completion
    setTimeout(() => {
      setIsTraining(false);
      setUserInstructions('');
    }, 3000);
  };

  const getCapabilityIcon = (capability: string) => {
    if (capability.toLowerCase().includes('security')) return Shield;
    if (capability.toLowerCase().includes('ai') || capability.toLowerCase().includes('learning')) return Brain;
    return Zap;
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="w-full max-w-7xl max-h-[90vh] bg-slate-950 rounded-2xl border border-slate-800 shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="sticky top-0 z-30 backdrop-blur supports-[backdrop-filter]:bg-slate-950/70 border-b border-slate-800">
          <div className="flex items-center justify-between px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-2xl bg-gradient-to-br from-indigo-500 to-purple-600 shadow-lg shadow-indigo-800/30">
                <Settings className="w-6 h-6" />
              </div>
              <div>
                <h1 className="text-xl md:text-2xl font-semibold tracking-tight">Agent Management</h1>
                <p className="text-xs text-slate-400">Train and manage your specialized sub-agents</p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-2 rounded-lg hover:bg-slate-800 transition-colors"
            >
              <ChevronRight className="w-5 h-5 text-slate-400 rotate-180" />
            </button>
          </div>
        </div>

        <div className="p-6 pb-10">
          <div className="grid grid-cols-12 gap-6">
            {/* Left: Agent Overview & Balance */}
            <div className="col-span-12 md:col-span-3 space-y-4">
              {/* Balance */}
              <Panel title="Resources" icon={Wallet}>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-300">NRN Balance</span>
                  <span className="text-lg font-bold text-emerald-400">{nrnBalance.toLocaleString()}</span>
                </div>
                <div className="mt-3 pt-3 border-t border-slate-800">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-slate-400">Active Agents</span>
                    <span className="text-slate-300">{subAgents.filter(sa => sa.status === 'deployed').length}/{subAgents.length}</span>
                  </div>
                </div>
              </Panel>

              {/* Training Stats */}
              <Panel title="Training Stats" icon={Trophy}>
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-slate-400">Total Trained</span>
                    <span className="text-sm font-medium text-slate-300">
                      {subAgents.filter(sa => sa.status === 'ready' || sa.status === 'deployed').length}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-slate-400">In Training</span>
                    <span className="text-sm font-medium text-blue-400">
                      {subAgents.filter(sa => sa.status === 'training').length}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-slate-400">Success Rate</span>
                    <span className="text-sm font-medium text-emerald-400">94%</span>
                  </div>
                </div>
              </Panel>
            </div>

            {/* Center: Sub-Agents Grid or Training Interface */}
            <div className="col-span-12 md:col-span-6">
              {!selectedSubAgent ? (
                <Panel title="Sub-Agent Training Center" icon={Activity}>
                  <div className="mb-4 text-sm text-slate-400">
                    Select a sub-agent to train with your Key Agent. Each agent can be specialized based on your specific instructions.
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    {subAgents.map((subAgent) => (
                      <button
                        key={subAgent.id}
                        onClick={() => setSelectedSubAgentId(subAgent.id)}
                        className={`p-4 rounded-xl border text-left transition-all hover:scale-[1.02] active:scale-[0.98] ${
                          subAgent.status === 'training' 
                            ? 'border-blue-600/40 bg-blue-950/20' 
                            : subAgent.status === 'ready'
                            ? 'border-emerald-600/40 bg-emerald-950/20'
                            : 'border-slate-700 bg-slate-900/60 hover:border-slate-600'
                        }`}
                      >
                        <div className="flex items-start justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                              subAgent.status === 'training' ? 'bg-blue-500/20 text-blue-400' :
                              subAgent.status === 'ready' ? 'bg-emerald-500/20 text-emerald-400' :
                              'bg-slate-700 text-slate-400'
                            }`}>
                              <Bot className="w-4 h-4" />
                            </div>
                            <div>
                              <div className="text-sm font-medium text-slate-200">{subAgent.name}</div>
                              <div className="text-[10px] text-slate-400">{subAgent.expertise[0]}</div>
                            </div>
                          </div>
                          <div className={`text-[10px] px-2 py-1 rounded-full border ${
                            subAgent.status === 'training' ? 'border-blue-500/40 text-blue-300' :
                            subAgent.status === 'ready' ? 'border-emerald-500/40 text-emerald-300' :
                            subAgent.status === 'deployed' ? 'border-purple-500/40 text-purple-300' :
                            'border-slate-600 text-slate-400'
                          }`}>
                            {subAgent.status}
                          </div>
                        </div>

                        <div className="text-xs text-slate-400 mb-3 line-clamp-2">
                          {subAgent.description}
                        </div>

                        {subAgent.status === 'training' && (
                          <div className="mb-2">
                            <div className="flex items-center justify-between text-xs mb-1">
                              <span className="text-blue-400">Training Progress</span>
                              <span>{subAgent.trainingProgress}%</span>
                            </div>
                            <div className="w-full bg-slate-700 rounded-full h-1.5">
                              <div 
                                className="bg-blue-500 h-1.5 rounded-full transition-all duration-500"
                                style={{ width: `${subAgent.trainingProgress}%` }}
                              />
                            </div>
                          </div>
                        )}

                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-1">
                            {subAgent.capabilities.slice(0, 2).map((cap, idx) => {
                              const Icon = getCapabilityIcon(cap);
                              return <Icon key={idx} className="w-3 h-3 text-slate-500" />;
                            })}
                            {subAgent.capabilities.length > 2 && (
                              <span className="text-[10px] text-slate-500">+{subAgent.capabilities.length - 2}</span>
                            )}
                          </div>
                          <div className="text-xs font-medium text-yellow-400">
                            {subAgent.nrnCost} NRN
                          </div>
                        </div>
                      </button>
                    ))}
                  </div>
                </Panel>
              ) : (
                <Panel title={`${selectedSubAgent.name} - Training Interface`} icon={Activity}>
                  <div className="space-y-4">
                    <button
                      onClick={() => setSelectedSubAgentId(null)}
                      className="flex items-center gap-1 text-xs px-2 py-1 rounded-lg border border-slate-700 hover:bg-slate-800"
                    >
                      <ChevronLeft className="w-4 h-4" />
                      Back to Agents
                    </button>

                    <div className="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
                      <div className="flex items-center gap-3 mb-3">
                        <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center">
                          <Bot className="w-6 h-6" />
                        </div>
                        <div>
                          <h3 className="text-lg font-medium text-slate-200">{selectedSubAgent.name}</h3>
                          <div className="flex items-center gap-2">
                            <div className={`w-2 h-2 rounded-full ${
                              selectedSubAgent.status === 'ready' ? 'bg-emerald-400' :
                              selectedSubAgent.status === 'training' ? 'bg-blue-400 animate-pulse' :
                              'bg-slate-400'
                            }`} />
                            <span className="text-sm text-slate-400">{selectedSubAgent.status}</span>
                          </div>
                        </div>
                      </div>

                      <div className="text-sm text-slate-300 mb-3">{selectedSubAgent.description}</div>
                      
                      <div className="flex flex-wrap gap-2 mb-4">
                        {selectedSubAgent.expertise.map((skill, idx) => (
                          <span key={idx} className="text-xs px-2 py-1 rounded-full bg-slate-800 text-slate-300">
                            {skill}
                          </span>
                        ))}
                      </div>

                      <div className="text-sm font-medium text-yellow-400 mb-4">
                        Training Cost: {selectedSubAgent.nrnCost} NRN
                      </div>
                    </div>

                    {/* Training Instructions */}
                    <div>
                      <label className="block text-sm font-medium text-slate-300 mb-2">
                        Training Instructions
                      </label>
                      <textarea
                        value={userInstructions}
                        onChange={(e) => setUserInstructions(e.target.value)}
                        placeholder="Provide specific instructions for training this sub-agent. Examples: 'Focus on debugging React components', 'Specialize in API integration patterns', 'Learn to handle error scenarios in microservices'..."
                        className="w-full h-32 px-3 py-2 bg-slate-900/60 border border-slate-700 rounded-lg text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-indigo-500 resize-none"
                      />
                    </div>

                    <div className="flex gap-3">
                      <button
                        onClick={handleStartTraining}
                        disabled={isTraining || !userInstructions.trim() || nrnBalance < selectedSubAgent.nrnCost}
                        className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                          isTraining || !userInstructions.trim() || nrnBalance < selectedSubAgent.nrnCost
                            ? 'bg-slate-700/50 text-slate-500 cursor-not-allowed'
                            : 'bg-indigo-600 text-white hover:bg-indigo-700'
                        }`}
                      >
                        {isTraining ? (
                          <>
                            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                            Training...
                          </>
                        ) : (
                          <>
                            <PlayCircle className="w-4 h-4" />
                            Start Training
                          </>
                        )}
                      </button>

                      {selectedSubAgent.status === 'ready' && (
                        <button
                          onClick={() => onDeployAgent(selectedSubAgent.id)}
                          className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium bg-emerald-600 text-white hover:bg-emerald-700"
                        >
                          <Zap className="w-4 h-4" />
                          Deploy Agent
                        </button>
                      )}
                    </div>
                  </div>
                </Panel>
              )}
            </div>

            {/* Right: Key Agent */}
            <div className="col-span-12 md:col-span-3">
              <Panel title="Key Agent" icon={Star}>
                {keyAgentData ? (
                  <div className="space-y-4">
                    <div className="flex items-center gap-3">
                      <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-lg">
                        <Bot className="w-6 h-6 text-white" />
                      </div>
                      <div>
                        <div className="text-sm font-medium text-slate-200">{keyAgentData.name}</div>
                        <div className="text-xs text-slate-400">v{keyAgentData.version}</div>
                      </div>
                    </div>

                    <div className="p-3 rounded-lg bg-slate-900/60 border border-slate-800">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs text-slate-400">Status</span>
                        <span className="text-xs font-medium text-emerald-400">{keyAgentData.status}</span>
                      </div>
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-xs text-slate-400">Agents Trained</span>
                        <span className="text-xs font-medium text-slate-300">{keyAgentData.totalTrained}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-xs text-slate-400">Success Rate</span>
                        <span className="text-xs font-medium text-emerald-400">{keyAgentData.successRate}%</span>
                      </div>
                    </div>

                    <div>
                      <div className="text-xs text-slate-400 mb-2">Core Capabilities</div>
                      <div className="space-y-2">
                        {keyAgentData.capabilities.map((capability, idx) => {
                          const Icon = getCapabilityIcon(capability);
                          return (
                            <div key={idx} className="flex items-center gap-2 text-xs">
                              <Icon className="w-3 h-3 text-indigo-400" />
                              <span className="text-slate-300">{capability}</span>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="text-center py-8">
                    <Bot className="w-8 h-8 mx-auto mb-2 opacity-50 text-slate-600" />
                    <p className="text-sm text-slate-500">No Key Agent Configured</p>
                  </div>
                )}
              </Panel>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Panel Component (reused from ecosystem-menu)
interface PanelProps {
  title: string;
  icon?: React.ComponentType<{ className?: string }>;
  children: React.ReactNode;
  className?: string;
}

function Panel({ title, icon: Icon, children, className = "" }: PanelProps) {
  return (
    <div className={`rounded-2xl border border-slate-800 bg-slate-900/60 shadow-xl shadow-black/20 ${className}`}>
      <div className="flex items-center gap-2 px-4 py-3 border-b border-slate-800/70">
        {Icon && <Icon className="w-4 h-4 text-slate-300" />}
        <h2 className="text-sm font-semibold tracking-wide text-slate-200">{title}</h2>
      </div>
      <div className="p-4">{children}</div>
    </div>
  );
}