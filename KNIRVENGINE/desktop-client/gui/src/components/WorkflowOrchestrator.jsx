import React, { useState, useEffect } from 'react';
import {
  Play,
  Pause,
  Square,
  Settings,
  Clock,
  CheckCircle,
  AlertCircle,
  Loader,
  GitBranch,
  Bot,
  Target,
  Zap,
  Eye,
  Download,
  BarChart3,
  ArrowRight,
  Plus,
  Trash2,
  RefreshCw
} from 'lucide-react';

export const WorkflowOrchestrator = () => {
  const [agents, setAgents] = useState([]);
  const [targets, setTargets] = useState([]);
  const [capabilities, setCapabilities] = useState([]);
  const [workflows, setWorkflows] = useState([]);
  const [selectedAgent, setSelectedAgent] = useState(null);
  const [selectedTarget, setSelectedTarget] = useState(null);
  const [selectedCapability, setSelectedCapability] = useState(null);
  const [promptInput, setPromptInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(null);

  // Fetch data on component mount
  useEffect(() => {
    fetchAgents();
    fetchTargets();
    fetchCapabilities();
    fetchWorkflows();

    // Set up auto-refresh for workflows
    const interval = setInterval(() => {
      fetchWorkflows();
    }, 5000); // Refresh every 5 seconds
    setRefreshInterval(interval);

    // Clean up interval on unmount
    return () => {
      if (refreshInterval) {
        clearInterval(refreshInterval);
      }
    };
  }, []);

  const fetchAgents = async () => {
    // In a real implementation, this would be an API call
    // For now, we'll use sample data
    setAgents([
      {
        id: '1',
        name: 'CyberPunk Agent #7804',
        collection: 'CyberPunk Collective',
        image: 'https://images.pexels.com/photos/5380664/pexels-photo-5380664.jpeg?auto=compress&cs=tinysrgb&w=200',
        status: 'active',
        capabilities: ['web_analysis', 'data_extraction', 'content_monitoring']
      },
      {
        id: '2',
        name: 'Data Miner #3749',
        collection: 'Industrial Agents',
        image: 'https://images.pexels.com/photos/5380617/pexels-photo-5380617.jpeg?auto=compress&cs=tinysrgb&w=200',
        status: 'active',
        capabilities: ['file_analysis', 'document_processing', 'data_mining']
      },
      {
        id: '3',
        name: 'Security Guardian #182',
        collection: 'Defense Protocol',
        image: 'https://images.pexels.com/photos/5380613/pexels-photo-5380613.jpeg?auto=compress&cs=tinysrgb&w=200',
        status: 'active',
        capabilities: ['security_analysis', 'threat_detection', 'network_monitoring']
      }
    ]);
  };

  const fetchTargets = async () => {
    // In a real implementation, this would be an API call
    // For now, we'll use sample data
    setTargets([
      {
        id: '1',
        name: 'Chrome Browser',
        type: 'browser',
        status: 'connected',
        capabilities: ['web_analysis', 'data_extraction', 'content_monitoring']
      },
      {
        id: '2',
        name: 'Local File System',
        type: 'filesystem',
        status: 'connected',
        capabilities: ['file_analysis', 'document_processing', 'data_mining']
      },
      {
        id: '3',
        name: 'VS Code Editor',
        type: 'application',
        status: 'limited',
        capabilities: ['code_analysis', 'quality_assessment', 'bug_detection']
      },
      {
        id: '4',
        name: 'Terminal/PowerShell',
        type: 'system',
        status: 'connected',
        capabilities: ['command_execution', 'process_monitoring', 'system_information']
      }
    ]);
  };

  const fetchCapabilities = async () => {
    // In a real implementation, this would be an API call
    // For now, we'll use sample data
    setCapabilities([
      {
        id: 'web_analysis',
        name: 'Web Analysis',
        description: 'Analyze web content and structure',
        estimatedTime: '2-5 seconds',
        provider: 'Browser MCP'
      },
      {
        id: 'data_extraction',
        name: 'Data Extraction',
        description: 'Extract structured data from content',
        estimatedTime: '3-8 seconds',
        provider: 'Data MCP'
      },
      {
        id: 'file_analysis',
        name: 'File Analysis',
        description: 'Analyze file content and metadata',
        estimatedTime: '2-6 seconds',
        provider: 'FileSystem MCP'
      },
      {
        id: 'document_processing',
        name: 'Document Processing',
        description: 'Process and analyze document content',
        estimatedTime: '4-10 seconds',
        provider: 'FileSystem MCP'
      },
      {
        id: 'security_analysis',
        name: 'Security Analysis',
        description: 'Analyze security vulnerabilities',
        estimatedTime: '5-15 seconds',
        provider: 'Security MCP'
      }
    ]);
  };

  const fetchWorkflows = async () => {
    try {
      // In a real implementation, this would be an API call
      // For now, we'll use sample data and simulate API behavior
      
      // Simulate API call delay
      // await new Promise(resolve => setTimeout(resolve, 500));
      
      // For demo purposes, we'll keep the existing workflows and just update their status
      // In a real implementation, you would fetch the latest workflow data from the server
      
      // If we don't have any workflows yet, create some sample ones
      if (workflows.length === 0) {
        setWorkflows([
          {
            id: '1',
            status: 'completed',
            startTime: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
            endTime: new Date(Date.now() - 3540000).toISOString(), // 59 minutes ago
            agentID: '1',
            targetID: '1',
            capabilityID: 'web_analysis',
            input: { prompt: 'Analyze the structure of example.com' },
            output: { text: 'Analysis complete: example.com has a responsive layout with 5 main sections and proper SEO metadata.' },
            dataExtracted: '5 sections'
          },
          {
            id: '2',
            status: 'running',
            startTime: new Date(Date.now() - 120000).toISOString(), // 2 minutes ago
            agentID: '2',
            targetID: '2',
            capabilityID: 'file_analysis',
            input: { prompt: 'Analyze the contents of the documents folder' }
          },
          {
            id: '3',
            status: 'failed',
            startTime: new Date(Date.now() - 1800000).toISOString(), // 30 minutes ago
            endTime: new Date(Date.now() - 1790000).toISOString(), // 29 minutes 50 seconds ago
            agentID: '3',
            targetID: '3',
            capabilityID: 'code_analysis',
            input: { prompt: 'Analyze the code quality of the project' },
            error: 'Access denied to target system'
          }
        ]);
      }
    } catch (error) {
      console.error('Error fetching workflows:', error);
    }
  };

  const getAgentById = (id) => {
    return agents.find(agent => agent.id === id) || { name: 'Unknown Agent' };
  };

  const getTargetById = (id) => {
    return targets.find(target => target.id === id) || { name: 'Unknown Target' };
  };

  const getCapabilityById = (id) => {
    return capabilities.find(capability => capability.id === id) || { name: 'Unknown Capability' };
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'running':
        return <Loader className="w-5 h-5 text-blue-400 animate-spin" />;
      case 'failed':
        return <AlertCircle className="w-5 h-5 text-red-400" />;
      case 'cancelled':
        return <Square className="w-5 h-5 text-orange-400" />;
      default:
        return <Clock className="w-5 h-5 text-slate-400" />;
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed':
        return 'text-green-400 bg-green-500/20 border-green-500/30';
      case 'running':
        return 'text-blue-400 bg-blue-500/20 border-blue-500/30';
      case 'failed':
        return 'text-red-400 bg-red-500/20 border-red-500/30';
      case 'cancelled':
        return 'text-orange-400 bg-orange-500/20 border-orange-500/30';
      default:
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
    }
  };

  const handleStartWorkflow = async () => {
    if (!selectedAgent || !selectedTarget || !selectedCapability || !promptInput) {
      alert('Please select an agent, target, capability, and enter a prompt');
      return;
    }

    setIsLoading(true);

    try {
      // In a real implementation, this would be an API call
      // For now, we'll simulate it
      
      // Simulate API call delay
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // Create a new workflow
      const newWorkflow = {
        id: Date.now().toString(), // Use timestamp as ID for demo
        status: 'running',
        startTime: new Date().toISOString(),
        agentID: selectedAgent.id,
        targetID: selectedTarget.id,
        capabilityID: selectedCapability.id,
        input: { prompt: promptInput }
      };
      
      // Add to workflows list
      setWorkflows(prev => [newWorkflow, ...prev]);
      
      // Simulate workflow completion after a delay
      setTimeout(() => {
        setWorkflows(prev => prev.map(workflow => {
          if (workflow.id === newWorkflow.id) {
            return {
              ...workflow,
              status: 'completed',
              endTime: new Date().toISOString(),
              output: {
                text: `Response from ${selectedAgent.name} using ${selectedCapability.name} on ${selectedTarget.name}: Successfully processed '${promptInput}' and extracted relevant data.`
              },
              dataExtracted: 'Analysis complete'
            };
          }
          return workflow;
        }));
      }, 5000); // Complete after 5 seconds
      
      // Reset form
      setPromptInput('');
    } catch (error) {
      console.error('Error starting workflow:', error);
      alert('Failed to start workflow');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancelWorkflow = async (workflowId) => {
    try {
      // In a real implementation, this would be an API call
      // For now, we'll simulate it
      
      // Update the workflow status
      setWorkflows(prev => prev.map(workflow => {
        if (workflow.id === workflowId && workflow.status === 'running') {
          return {
            ...workflow,
            status: 'cancelled',
            endTime: new Date().toISOString(),
            error: 'Workflow cancelled by user'
          };
        }
        return workflow;
      }));
    } catch (error) {
      console.error('Error cancelling workflow:', error);
      alert('Failed to cancel workflow');
    }
  };

  const formatDateTime = (isoString) => {
    const date = new Date(isoString);
    return date.toLocaleString();
  };

  const calculateDuration = (startTime, endTime) => {
    if (!endTime) return 'In progress';
    
    const start = new Date(startTime);
    const end = new Date(endTime);
    const durationMs = end - start;
    
    if (durationMs < 1000) return `${durationMs}ms`;
    if (durationMs < 60000) return `${Math.round(durationMs / 1000)}s`;
    
    const minutes = Math.floor(durationMs / 60000);
    const seconds = Math.round((durationMs % 60000) / 1000);
    return `${minutes}m ${seconds}s`;
  };

  // Filter available capabilities based on selected agent and target
  const getAvailableCapabilities = () => {
    if (!selectedAgent || !selectedTarget) return [];
    
    // Find capabilities that match both the agent and target
    return capabilities.filter(capability => 
      selectedAgent.capabilities.includes(capability.id) && 
      selectedTarget.capabilities.includes(capability.id)
    );
  };

  return (
    <div className="p-6 space-y-8">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">Workflow Orchestrator</h1>
          <p className="text-slate-400">Create and manage complex workflows by connecting agents, targets, and capabilities.</p>
        </div>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          <button
            onClick={() => fetchWorkflows()}
            className="bg-slate-800/50 border border-slate-700/50 text-slate-300 px-4 py-2 rounded-lg hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2"
          >
            <RefreshCw className="w-4 h-4" />
            <span>Refresh</span>
          </button>
          <button
            onClick={handleStartWorkflow}
            disabled={!selectedAgent || !selectedTarget || !selectedCapability || !promptInput || isLoading}
            className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? (
              <>
                <Loader className="w-5 h-5 animate-spin" />
                <span>Executing...</span>
              </>
            ) : (
              <>
                <GitBranch className="w-5 h-5" />
                <span>Execute Workflow</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Workflow Builder */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
        <h2 className="text-xl font-bold text-white mb-6">Create New Workflow</h2>
        
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-6 items-center">
          {/* Agent Selection */}
          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Bot className="w-5 h-5 text-purple-400" />
              <h3 className="text-lg font-semibold text-white">Agent</h3>
            </div>
            <div className="space-y-2">
              {agents.map((agent) => (
                <div
                  key={agent.id}
                  onClick={() => setSelectedAgent(agent)}
                  className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                    selectedAgent?.id === agent.id
                      ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 border-purple-500/30'
                      : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                  }`}
                >
                  <div className="flex items-center space-x-2">
                    <img 
                      src={agent.image} 
                      alt={agent.name}
                      className="w-8 h-8 rounded-lg object-cover"
                    />
                    <div>
                      <p className="text-white font-medium text-sm">{agent.name}</p>
                      <p className="text-slate-400 text-xs">{agent.collection}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Arrow */}
          <div className="flex justify-center">
            <ArrowRight className="w-6 h-6 text-slate-400" />
          </div>

          {/* Target Selection */}
          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Target className="w-5 h-5 text-blue-400" />
              <h3 className="text-lg font-semibold text-white">Target</h3>
            </div>
            <div className="space-y-2">
              {targets.map((target) => (
                <div
                  key={target.id}
                  onClick={() => setSelectedTarget(target)}
                  className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                    selectedTarget?.id === target.id
                      ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 border-blue-500/30'
                      : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                  }`}
                >
                  <p className="text-white font-medium text-sm">{target.name}</p>
                  <p className="text-slate-400 text-xs">{target.type}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Arrow */}
          <div className="flex justify-center">
            <ArrowRight className="w-6 h-6 text-slate-400" />
          </div>

          {/* Capability Selection */}
          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Zap className="w-5 h-5 text-green-400" />
              <h3 className="text-lg font-semibold text-white">Capability</h3>
            </div>
            <div className="space-y-2">
              {getAvailableCapabilities().length > 0 ? (
                getAvailableCapabilities().map((capability) => (
                  <div
                    key={capability.id}
                    onClick={() => setSelectedCapability(capability)}
                    className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                      selectedCapability?.id === capability.id
                        ? 'bg-gradient-to-r from-green-500/20 to-emerald-500/20 border-green-500/30'
                        : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                    }`}
                  >
                    <p className="text-white font-medium text-sm">{capability.name}</p>
                    <p className="text-slate-400 text-xs">{capability.provider}</p>
                  </div>
                ))
              ) : (
                <div className="p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
                  <div className="flex items-start space-x-3">
                    <AlertCircle className="w-5 h-5 text-yellow-400 mt-0.5" />
                    <div>
                      <p className="text-yellow-400 font-medium">No compatible capabilities</p>
                      <p className="text-yellow-300/70 text-sm">
                        {!selectedAgent && !selectedTarget 
                          ? 'Select an agent and target first'
                          : !selectedAgent 
                            ? 'Select an agent first'
                            : !selectedTarget
                              ? 'Select a target first'
                              : 'No capabilities compatible with both the selected agent and target'}
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Prompt Input */}
        {selectedAgent && selectedTarget && selectedCapability && (
          <div className="mt-6">
            <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30 mb-4">
              <h4 className="text-white font-semibold mb-3">Workflow Configuration</h4>
              <div className="flex items-center space-x-4 text-sm mb-4">
                <div className="flex items-center space-x-2">
                  <Bot className="w-4 h-4 text-purple-400" />
                  <span className="text-white">{selectedAgent.name}</span>
                </div>
                <ArrowRight className="w-4 h-4 text-slate-400" />
                <div className="flex items-center space-x-2">
                  <Target className="w-4 h-4 text-blue-400" />
                  <span className="text-white">{selectedTarget.name}</span>
                </div>
                <ArrowRight className="w-4 h-4 text-slate-400" />
                <div className="flex items-center space-x-2">
                  <Zap className="w-4 h-4 text-green-400" />
                  <span className="text-white">{selectedCapability.name}</span>
                </div>
              </div>
              
              <div className="space-y-3">
                <label className="block text-sm font-medium text-slate-300">
                  Prompt
                </label>
                <textarea
                  value={promptInput}
                  onChange={(e) => setPromptInput(e.target.value)}
                  placeholder="Enter your prompt or instructions for the agent..."
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 min-h-[100px]"
                />
              </div>
            </div>
            
            <div className="flex justify-end">
              <button
                onClick={handleStartWorkflow}
                disabled={isLoading || !promptInput}
                className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isLoading ? (
                  <>
                    <Loader className="w-5 h-5 animate-spin" />
                    <span>Starting Workflow...</span>
                  </>
                ) : (
                  <>
                    <GitBranch className="w-5 h-5" />
                    <span>Start Workflow</span>
                  </>
                )}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Workflow History */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700/50 overflow-hidden">
        <div className="p-6 border-b border-slate-700/50">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-bold text-white">Workflow History</h2>
            <div className="flex items-center space-x-2">
              <button className="text-purple-400 hover:text-purple-300 text-sm font-medium">Export</button>
              <button className="text-slate-400 hover:text-white text-sm font-medium">Clear All</button>
            </div>
          </div>
        </div>
        
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-slate-700/30">
              <tr>
                <th className="text-left py-3 px-6 text-slate-300 font-medium">Status</th>
                <th className="text-left py-3 px-6 text-slate-300 font-medium">Agent</th>
                <th className="text-left py-3 px-6 text-slate-300 font-medium">Target</th>
                <th className="text-left py-3 px-6 text-slate-300 font-medium">Capability</th>
                <th className="text-left py-3 px-6 text-slate-300 font-medium">Started</th>
                <th className="text-left py-3 px-6 text-slate-300 font-medium">Duration</th>
                <th className="text-left py-3 px-6 text-slate-300 font-medium">Result</th>
                <th className="text-left py-3 px-6 text-slate-300 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-700/50">
              {workflows.length > 0 ? (
                workflows.map((workflow) => (
                  <tr key={workflow.id} className="hover:bg-slate-700/20 transition-colors duration-200">
                    <td className="py-4 px-6">
                      <div className="flex items-center space-x-2">
                        {getStatusIcon(workflow.status)}
                        <span className={`px-2 py-1 rounded-full text-xs font-medium border ${getStatusColor(workflow.status)}`}>
                          {workflow.status}
                        </span>
                      </div>
                    </td>
                    <td className="py-4 px-6 text-white font-medium">
                      {getAgentById(workflow.agentID).name}
                    </td>
                    <td className="py-4 px-6 text-slate-300">
                      {getTargetById(workflow.targetID).name}
                    </td>
                    <td className="py-4 px-6 text-slate-300">
                      {getCapabilityById(workflow.capabilityID).name}
                    </td>
                    <td className="py-4 px-6 text-slate-300">
                      {formatDateTime(workflow.startTime)}
                    </td>
                    <td className="py-4 px-6 text-slate-300">
                      {calculateDuration(workflow.startTime, workflow.endTime)}
                    </td>
                    <td className="py-4 px-6">
                      {workflow.output ? (
                        <div>
                          <p className="text-white text-sm">{workflow.output.text.substring(0, 50)}...</p>
                          {workflow.dataExtracted && (
                            <p className="text-slate-400 text-xs">Data: {workflow.dataExtracted}</p>
                          )}
                        </div>
                      ) : workflow.error ? (
                        <div>
                          <p className="text-red-400 text-sm">{workflow.error.substring(0, 50)}...</p>
                        </div>
                      ) : (
                        <span className="text-slate-500">—</span>
                      )}
                    </td>
                    <td className="py-4 px-6">
                      <div className="flex items-center space-x-2">
                        {workflow.status === 'running' && (
                          <button
                            onClick={() => handleCancelWorkflow(workflow.id)}
                            className="p-1 text-red-400 hover:text-red-300 transition-colors duration-200"
                            title="Cancel workflow"
                          >
                            <Square className="w-4 h-4" />
                          </button>
                        )}
                        {(workflow.status === 'completed' || workflow.output) && (
                          <>
                            <button
                              className="p-1 text-slate-400 hover:text-white transition-colors duration-200"
                              title="View details"
                            >
                              <Eye className="w-4 h-4" />
                            </button>
                            <button
                              className="p-1 text-slate-400 hover:text-white transition-colors duration-200"
                              title="Download results"
                            >
                              <Download className="w-4 h-4" />
                            </button>
                          </>
                        )}
                        <button
                          className="p-1 text-slate-400 hover:text-white transition-colors duration-200"
                          title="View analytics"
                        >
                          <BarChart3 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan="8" className="py-8 text-center text-slate-400">
                    No workflows found. Create a new workflow to get started.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default WorkflowOrchestrator;