import React, { useState, useEffect } from 'react';
import {
  X,
  Target,
  Zap,
  Bot,
  Play,
  AlertCircle,
  Check,
  ArrowRight,
  Loader
} from 'lucide-react';
// import targetDiscoveryService from '../../services/targetDiscovery';

const AgentDeploymentModal = ({ isOpen, onClose, agent, onAgentDeployed }) => {
  const [selectedTarget, setSelectedTarget] = useState(null);
  const [selectedCapability, setSelectedCapability] = useState(null);
  const [deploymentStatus, setDeploymentStatus] = useState(null); // 'deploying', 'success', 'error', or null
  const [deploymentProgress, setDeploymentProgress] = useState(0);
  const [deploymentMessage, setDeploymentMessage] = useState('');
  const [errors, setErrors] = useState({});

  // Sample data for available targets and capabilities
  // In a real implementation, these would be fetched from the API
  const [availableTargets, setAvailableTargets] = useState([]);
  const [availableCapabilities, setAvailableCapabilities] = useState([]);

  useEffect(() => {
    if (isOpen && agent) {
      // Reset state when modal opens
      setSelectedTarget(null);
      setSelectedCapability(null);
      setDeploymentStatus(null);
      setDeploymentProgress(0);
      setDeploymentMessage('');
      setErrors({});

      // Load available targets (temporary implementation for testing)
      const loadTargets = async () => {
        try {
          // Temporary mock data for testing - in production this would use targetDiscoveryService
          const mockTargets = [
            {
              id: 'chrome-browser',
              name: 'Chrome Browser',
              type: 'browser',
              category: 'Web Browser',
              status: 'connected',
              description: 'Google Chrome web browser',
              capabilities: ['web_analysis', 'data_extraction', 'content_monitoring'],
              permissions: ['read', 'write'],
              security: 'high',
              icon: null,
              connectionMethod: 'Chrome Extension API',
              platform: 'Desktop'
            },
            {
              id: 'local-filesystem',
              name: 'Local File System',
              type: 'filesystem',
              category: 'Storage',
              status: 'connected',
              description: 'Local file system access',
              capabilities: ['file_analysis', 'document_processing', 'data_extraction'],
              permissions: ['read', 'write'],
              security: 'medium',
              icon: null,
              connectionMethod: 'File System API',
              platform: 'Desktop'
            }
          ];

          // Filter targets based on agent's targetTypes if specified
          const filteredTargets = agent && agent.targetTypes
            ? mockTargets.filter(target =>
                agent.targetTypes.some(type =>
                  type.toLowerCase() === target.type.toLowerCase() ||
                  type.toLowerCase().includes(target.type.toLowerCase()) ||
                  target.type.toLowerCase().includes(type.toLowerCase())
                )
              )
            : mockTargets;

          setAvailableTargets(filteredTargets);
        } catch (error) {
          console.error('Failed to load targets:', error);
          // Fallback to empty array if loading fails
          setAvailableTargets([]);
        }
      };

      loadTargets();
    }
  }, [isOpen, agent]);

  // Update available capabilities when target is selected
  useEffect(() => {
    if (selectedTarget) {
      // In a real implementation, this would be an API call
      // For now, we'll use sample data based on the selected target
      
      const capabilities = [
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
          id: 'content_monitoring',
          name: 'Content Monitoring',
          description: 'Monitor content for changes',
          estimatedTime: '1-2 seconds',
          provider: 'Browser MCP'
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
          id: 'data_mining',
          name: 'Data Mining',
          description: 'Extract patterns and insights from data',
          estimatedTime: '5-15 seconds',
          provider: 'Data MCP'
        },
        {
          id: 'code_analysis',
          name: 'Code Analysis',
          description: 'Analyze code quality and structure',
          estimatedTime: '3-8 seconds',
          provider: 'IDE MCP'
        },
        {
          id: 'quality_assessment',
          name: 'Quality Assessment',
          description: 'Assess code quality and identify issues',
          estimatedTime: '4-12 seconds',
          provider: 'IDE MCP'
        },
        {
          id: 'bug_detection',
          name: 'Bug Detection',
          description: 'Identify potential bugs and issues',
          estimatedTime: '3-10 seconds',
          provider: 'IDE MCP'
        },
        {
          id: 'security_analysis',
          name: 'Security Analysis',
          description: 'Analyze security vulnerabilities',
          estimatedTime: '5-15 seconds',
          provider: 'Security MCP'
        },
        {
          id: 'threat_detection',
          name: 'Threat Detection',
          description: 'Detect potential security threats',
          estimatedTime: '3-8 seconds',
          provider: 'Security MCP'
        },
        {
          id: 'network_monitoring',
          name: 'Network Monitoring',
          description: 'Monitor network traffic and activity',
          estimatedTime: '1-3 seconds',
          provider: 'Network MCP'
        },
        {
          id: 'image_processing',
          name: 'Image Processing',
          description: 'Process and analyze images',
          estimatedTime: '3-10 seconds',
          provider: 'Media MCP'
        },
        {
          id: 'media_analysis',
          name: 'Media Analysis',
          description: 'Analyze media content and metadata',
          estimatedTime: '4-12 seconds',
          provider: 'Media MCP'
        },
        {
          id: 'creative_enhancement',
          name: 'Creative Enhancement',
          description: 'Enhance creative content',
          estimatedTime: '5-15 seconds',
          provider: 'Media MCP'
        }
      ];
      
      // Filter capabilities based on selected target and agent capabilities
      const targetCapabilities = selectedTarget.capabilities || [];
      const agentCapabilities = agent && agent.capabilities 
        ? agent.capabilities.map(cap => cap.toLowerCase().replace(/\s+/g, '_')) 
        : [];
      
      // Find capabilities that match both the target and agent
      const filteredCapabilities = capabilities.filter(cap => 
        targetCapabilities.includes(cap.id) && 
        (
          agentCapabilities.includes(cap.id) || 
          agentCapabilities.some(agentCap => 
            cap.id.includes(agentCap) || 
            agentCap.includes(cap.id.replace(/_/g, ' '))
          )
        )
      );
      
      setAvailableCapabilities(filteredCapabilities);
      setSelectedCapability(null); // Reset selected capability when target changes
    } else {
      setAvailableCapabilities([]);
      setSelectedCapability(null);
    }
  }, [selectedTarget, agent]);

  const validateDeployment = () => {
    const newErrors = {};
    
    if (!selectedTarget) {
      newErrors.target = 'Please select a target system';
    }
    
    if (!selectedCapability) {
      newErrors.capability = 'Please select a capability';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleDeploy = async () => {
    if (!validateDeployment()) {
      return;
    }
    
    setDeploymentStatus('deploying');
    setDeploymentProgress(0);
    setDeploymentMessage('Initializing deployment...');
    
    // Simulate deployment process
    await simulateDeploymentStep('Connecting to target system...', 20);
    await simulateDeploymentStep('Initializing agent capabilities...', 40);
    await simulateDeploymentStep('Configuring security permissions...', 60);
    await simulateDeploymentStep('Establishing communication channels...', 80);
    await simulateDeploymentStep('Finalizing deployment...', 100);
    
    try {
      // In a real implementation, this would be an API call
      // For now, we'll simulate a successful deployment
      
      // Simulate API call
      // const response = await fetch(`/api/v1/agents/${agent.id}/deploy`, {
      //   method: 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify({
      //     target_id: selectedTarget.id,
      //     capability_id: selectedCapability.id
      //   }),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Failed to deploy agent');
      // }
      
      // const data = await response.json();
      
      setDeploymentStatus('success');
      setDeploymentMessage('Agent deployed successfully!');
      
      // Notify parent component of successful deployment
      if (onAgentDeployed) {
        const updatedAgent = {
          ...agent,
          status: 'active',
          currentTarget: selectedTarget.name,
          capability: selectedCapability.name,
          deployedSince: 'just now',
          lastActivity: 'just now'
        };
        
        onAgentDeployed(updatedAgent);
      }
      
      // Close modal after successful deployment
      setTimeout(() => {
        onClose();
      }, 2000);
      
    } catch (error) {
      console.error('Error deploying agent:', error);
      setDeploymentStatus('error');
      setDeploymentMessage('Failed to deploy agent. Please try again.');
    }
  };

  const simulateDeploymentStep = async (message, progress) => {
    setDeploymentMessage(message);
    setDeploymentProgress(progress);
    // Simulate network delay
    await new Promise(resolve => setTimeout(resolve, 800));
  };

  if (!isOpen || !agent) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
        onClick={onClose}
      ></div>
      
      {/* Modal */}
      <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-2xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-gradient-to-r from-green-500/20 to-emerald-500/20 rounded-lg border border-green-500/30">
              <Play className="w-5 h-5 text-green-400" />
            </div>
            <h2 className="text-xl font-bold text-white">Deploy Agent</h2>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
            disabled={deploymentStatus === 'deploying'}
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-200px)]">
          {/* Agent Info */}
          <div className="flex items-center space-x-4 mb-6 p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
            <img 
              src={agent.image} 
              alt={agent.name}
              className="w-16 h-16 rounded-lg object-cover"
            />
            <div>
              <h3 className="text-lg font-bold text-white">{agent.name}</h3>
              <p className="text-slate-400">{agent.collection}</p>
              <div className="flex flex-wrap gap-1 mt-1">
                {agent.capabilities && agent.capabilities.slice(0, 2).map((capability, index) => (
                  <span key={index} className="px-2 py-0.5 bg-slate-600/50 text-slate-300 text-xs rounded-full">
                    {capability}
                  </span>
                ))}
                {agent.capabilities && agent.capabilities.length > 2 && (
                  <span className="px-2 py-0.5 bg-slate-600/50 text-slate-300 text-xs rounded-full">
                    +{agent.capabilities.length - 2} more
                  </span>
                )}
              </div>
            </div>
          </div>
          
          {deploymentStatus === null ? (
            <div className="space-y-6">
              {/* Target Selection */}
              <div className="space-y-4">
                <div className="flex items-center space-x-2">
                  <Target className="w-5 h-5 text-blue-400" />
                  <h3 className="text-lg font-semibold text-white">Select Target System</h3>
                </div>
                
                <div className="grid grid-cols-1 gap-3">
                  {availableTargets.map((target) => (
                    <div
                      key={target.id}
                      onClick={() => setSelectedTarget(target)}
                      className={`p-4 rounded-lg cursor-pointer transition-all duration-200 border ${
                        selectedTarget?.id === target.id
                          ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 border-blue-500/30'
                          : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-3">
                          {target.icon && <target.icon className="w-6 h-6 text-blue-400" />}
                          <div>
                            <p className="text-white font-medium">{target.name}</p>
                            <p className="text-slate-400 text-sm">{target.description}</p>
                            <div className="flex items-center space-x-2 mt-1">
                              <span className="text-xs text-slate-500">{target.category}</span>
                              <span className="text-xs text-slate-500">•</span>
                              <span className="text-xs text-slate-500">{target.connectionMethod}</span>
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center space-x-2">
                          <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                            target.status === 'connected'
                              ? 'bg-green-500/20 text-green-400'
                              : target.status === 'monitoring'
                              ? 'bg-blue-500/20 text-blue-400'
                              : 'bg-yellow-500/20 text-yellow-400'
                          }`}>
                            {target.status}
                          </span>
                          {selectedTarget?.id === target.id && (
                            <Check className="w-4 h-4 text-blue-400" />
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
                {errors.target && (
                  <p className="text-sm text-red-400">{errors.target}</p>
                )}
              </div>
              
              {/* Capability Selection */}
              {selectedTarget && (
                <div className="space-y-4">
                  <div className="flex items-center space-x-2">
                    <Zap className="w-5 h-5 text-green-400" />
                    <h3 className="text-lg font-semibold text-white">Select Capability</h3>
                  </div>
                  
                  <div className="grid grid-cols-1 gap-3">
                    {availableCapabilities.length > 0 ? (
                      availableCapabilities.map((capability) => (
                        <div
                          key={capability.id}
                          onClick={() => setSelectedCapability(capability)}
                          className={`p-4 rounded-lg cursor-pointer transition-all duration-200 border ${
                            selectedCapability?.id === capability.id
                              ? 'bg-gradient-to-r from-green-500/20 to-emerald-500/20 border-green-500/30'
                              : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <div>
                              <p className="text-white font-medium">{capability.name}</p>
                              <p className="text-slate-400 text-sm">{capability.description}</p>
                              <div className="flex items-center space-x-4 mt-1 text-xs text-slate-500">
                                <span>Provider: {capability.provider}</span>
                                <span>Est. Time: {capability.estimatedTime}</span>
                              </div>
                            </div>
                            {selectedCapability?.id === capability.id && (
                              <Check className="w-4 h-4 text-green-400" />
                            )}
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
                        <div className="flex items-start space-x-3">
                          <AlertCircle className="w-5 h-5 text-yellow-400 mt-0.5" />
                          <div>
                            <p className="text-yellow-400 font-medium">No compatible capabilities</p>
                            <p className="text-yellow-300/70 text-sm">This agent doesn't have capabilities compatible with the selected target. Please select a different target or update the agent's capabilities.</p>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                  {errors.capability && (
                    <p className="text-sm text-red-400">{errors.capability}</p>
                  )}
                </div>
              )}
              
              {/* Deployment Summary */}
              {selectedTarget && selectedCapability && (
                <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                  <h4 className="text-white font-semibold mb-3">Deployment Summary</h4>
                  <div className="flex items-center space-x-4 text-sm">
                    <div className="flex items-center space-x-2">
                      <Bot className="w-4 h-4 text-purple-400" />
                      <span className="text-white">{agent.name}</span>
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
                  <p className="text-slate-400 text-sm mt-2">
                    Estimated deployment time: 5-10 seconds
                  </p>
                </div>
              )}
            </div>
          ) : (
            <div className="space-y-6">
              {/* Deployment Status */}
              <div className={`p-6 rounded-lg border ${
                deploymentStatus === 'deploying' ? 'bg-blue-500/20 border-blue-500/30' :
                deploymentStatus === 'success' ? 'bg-green-500/20 border-green-500/30' :
                'bg-red-500/20 border-red-500/30'
              }`}>
                <div className="flex items-center space-x-3 mb-4">
                  {deploymentStatus === 'deploying' && (
                    <Loader className="w-6 h-6 text-blue-400 animate-spin" />
                  )}
                  {deploymentStatus === 'success' && (
                    <Check className="w-6 h-6 text-green-400" />
                  )}
                  {deploymentStatus === 'error' && (
                    <AlertCircle className="w-6 h-6 text-red-400" />
                  )}
                  <h3 className={`text-lg font-semibold ${
                    deploymentStatus === 'deploying' ? 'text-blue-400' :
                    deploymentStatus === 'success' ? 'text-green-400' :
                    'text-red-400'
                  }`}>
                    {deploymentStatus === 'deploying' ? 'Deploying Agent...' :
                     deploymentStatus === 'success' ? 'Deployment Successful!' :
                     'Deployment Failed'}
                  </h3>
                </div>
                
                <p className={`text-sm ${
                  deploymentStatus === 'deploying' ? 'text-blue-300' :
                  deploymentStatus === 'success' ? 'text-green-300' :
                  'text-red-300'
                }`}>
                  {deploymentMessage}
                </p>
                
                {deploymentStatus === 'deploying' && (
                  <div className="mt-4">
                    <div className="w-full bg-slate-700 rounded-full h-2">
                      <div 
                        className="bg-gradient-to-r from-blue-500 to-cyan-500 h-2 rounded-full transition-all duration-500"
                        style={{ width: `${deploymentProgress}%` }}
                      ></div>
                    </div>
                    <p className="text-right text-xs text-blue-300 mt-1">{deploymentProgress}%</p>
                  </div>
                )}
              </div>
              
              {/* Deployment Details */}
              {deploymentStatus === 'success' && (
                <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                  <h4 className="text-white font-semibold mb-3">Deployment Details</h4>
                  <div className="space-y-2 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="text-slate-400">Agent:</span>
                      <span className="text-white">{agent.name}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-slate-400">Target:</span>
                      <span className="text-white">{selectedTarget?.name}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-slate-400">Capability:</span>
                      <span className="text-white">{selectedCapability?.name}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-slate-400">Deployed At:</span>
                      <span className="text-white">{new Date().toLocaleTimeString()}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-slate-400">Status:</span>
                      <span className="text-green-400">Active</span>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
        
        {/* Footer */}
        <div className="p-6 border-t border-slate-700 bg-slate-800/80 mt-auto">
          <div className="flex items-center justify-between">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200"
              disabled={deploymentStatus === 'deploying'}
            >
              {deploymentStatus === 'success' ? 'Close' : 'Cancel'}
            </button>
            
            {deploymentStatus === null && (
              <button
                onClick={handleDeploy}
                className="px-6 py-2 bg-gradient-to-r from-green-500 to-emerald-500 text-white rounded-lg font-medium hover:from-green-600 hover:to-emerald-600 transition-all duration-200 flex items-center space-x-2"
              >
                <Play className="w-4 h-4" />
                <span>Deploy Agent</span>
              </button>
            )}
            
            {deploymentStatus === 'error' && (
              <button
                onClick={() => setDeploymentStatus(null)}
                className="px-6 py-2 bg-gradient-to-r from-blue-500 to-purple-500 text-white rounded-lg font-medium hover:from-blue-600 hover:to-purple-600 transition-all duration-200 flex items-center space-x-2"
              >
                <ArrowRight className="w-4 h-4" />
                <span>Try Again</span>
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default AgentDeploymentModal;