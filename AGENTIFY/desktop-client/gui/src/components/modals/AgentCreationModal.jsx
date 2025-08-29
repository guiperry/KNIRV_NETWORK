import React, { useState, useEffect } from 'react';
import {
  X,
  Upload,
  Check,
  AlertCircle,
  Zap,
  Target,
  Bot,
  Key,
  Loader2,
  Plus
} from 'lucide-react';
import { createAgent, buildAgentPlugin, getAgentBuildStatus, fetchCompiledPlugins, fetchAvailableAgents } from '../../utils/api';
import { sampleAgentImages, getDefaultAgentImage } from '../../utils/imageUrls';
import { useDefaultAgentImage } from '../../hooks/useAssetPath';
import AgentImage from '../common/AgentImage';
import { handleApiError } from '../../utils/errorHandler';

const AgentCreationModal = ({ isOpen, onClose, onAgentCreated }) => {
  const defaultAgentImage = useDefaultAgentImage();
  const [formData, setFormData] = useState({
    name: '',
    collection: '',
    imageURL: defaultAgentImage, // Set default image to Agentify logo
    capabilities: [],
    targetTypes: [],
    buildTarget: 'plugin', // Default to plugin build
    apiKeys: {
      openai: '',
      claude: '',
      gemini: '',
      deepseek: '',
      cerebras: ''
    }
  });

  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [availableCapabilities, setAvailableCapabilities] = useState([]);
  const [availableTargetTypes, setAvailableTargetTypes] = useState([]);

  // Agent creation mode: 'select' (choose existing) or 'create' (build new)
  const [creationMode, setCreationMode] = useState('select');
  const [availableAgents, setAvailableAgents] = useState([]);
  const [selectedAgent, setSelectedAgent] = useState(null);
  const [loadingAgents, setLoadingAgents] = useState(false);

  // Agent creation progress state
  const [creationProgress, setCreationProgress] = useState(0);
  const [creationStatus, setCreationStatus] = useState('');

  // Load available pre-made agents from both compiled plugins and discovered agents
  const loadAvailableAgents = async () => {
    setLoadingAgents(true);
    try {
      // Get both compiled plugins and discovered agents
      const [compiledPlugins, discoveredAgentIds] = await Promise.all([
        fetchCompiledPlugins().catch(() => []),
        fetchAvailableAgents().catch(() => [])
      ]);

      // Transform compiled plugins into available agents
      const pluginAgents = compiledPlugins.map(plugin => ({
        id: plugin.agent_id,
        name: plugin.agent_id.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()),
        description: `Pre-built agent: ${plugin.agent_id}`,
        plugin_info: {
          plugin_id: plugin.agent_id,
          plugin_version: plugin.version,
          plugin_path: plugin.path,
          filename: plugin.filename
        },
        capabilities: ['General Purpose'], // Default capabilities
        targetTypes: ['application'], // Default target types
        type: 'plugin',
        created_at: plugin.created_at
      }));

      // Transform discovered agent IDs into agent format
      const discoveredAgents = discoveredAgentIds.map(agentId => {
        // Parse agent ID to extract name and version
        const parts = agentId.split('_');
        const version = parts.pop();
        const name = parts.join('_');
        const isWasm = agentId.includes('wasm') || !pluginAgents.find(p => p.id === agentId);

        return {
          id: agentId,
          name: name.charAt(0).toUpperCase() + name.slice(1).replace(/_/g, ' '),
          description: `${isWasm ? 'WASM' : 'Plugin'} agent: ${name} (v${version})`,
          capabilities: ['General Purpose'],
          targetTypes: ['application'],
          type: isWasm ? 'wasm' : 'plugin',
          plugin_info: {
            plugin_id: name,
            plugin_version: version,
            file_type: isWasm ? '.wasm' : '.so'
          },
          created_at: new Date().toISOString()
        };
      });

      // Combine and deduplicate agents (prefer compiled plugins over discovered ones)
      const allAgents = [...pluginAgents];
      discoveredAgents.forEach(discovered => {
        if (!pluginAgents.find(plugin => plugin.id === discovered.id)) {
          allAgents.push(discovered);
        }
      });

      setAvailableAgents(allAgents);
    } catch (error) {
      console.error('Error loading available agents:', error);
      handleApiError(error, {
        operation: 'load_available_agents',
        component: 'AgentCreationModal',
        timestamp: new Date().toISOString(),
        context: 'Loading available agents for agent creation'
      });
      setAvailableAgents([]);
    } finally {
      setLoadingAgents(false);
    }
  };

  // Create agent with automatic plugin building
  const createAgentWithPlugin = async () => {
    setCreationProgress(10);
    setCreationStatus('Preparing agent configuration...');

    try {
      // Generate a unique agent ID from the name
      const agentId = formData.name.toLowerCase().replace(/[^a-z0-9]/g, '');

      setCreationProgress(20);
      setCreationStatus('Building agent plugin...');

      // Build configuration for the plugin
      const buildConfig = {
        agent_name: agentId,
        agent_description: formData.description || `AI Agent: ${formData.name}`,
        model: 'deepseek-chat',
        instruction: `You are ${formData.name}. ${formData.description || 'You are a helpful AI assistant.'} Your capabilities include: ${formData.capabilities.join(', ')}.`,
        agent_type: 'llm',
        build_target: formData.buildTarget, // Include build target
        capabilities: formData.capabilities,
        target_types: formData.targetTypes,
        api_keys: formData.apiKeys
      };

      // Start the plugin build
      const buildResult = await buildAgentPlugin(agentId, 'default', buildConfig);

      setCreationProgress(40);
      setCreationStatus('Compiling agent...');

      // Poll for build status
      let attempts = 0;
      const maxAttempts = 120; // 120 seconds timeout (2 minutes) for WASM builds

      while (attempts < maxAttempts) {
        await new Promise(resolve => setTimeout(resolve, 1000)); // Wait 1 second

        try {
          const status = await getAgentBuildStatus(agentId);

          if (status.status === 'completed' || status.status === 'success') {
            setCreationProgress(80);
            setCreationStatus('Registering agent...');

            // Create the agent record
            const agentData = {
              name: formData.name,
              type: 'llm',
              config: JSON.stringify({
                description: formData.description,
                capabilities: formData.capabilities,
                target_types: formData.targetTypes,
                api_keys: formData.apiKeys,
                status: 'idle', // Ensure new agents start with idle status
                plugin_info: {
                  plugin_id: agentId,
                  plugin_version: '1.0',
                  plugin_path: status.plugin_path
                }
              })
            };

            const createdAgent = await createAgent(agentData);

            setCreationProgress(100);
            setCreationStatus('Agent created successfully!');

            return createdAgent;
          } else if (status.status === 'failed' || status.status === 'error') {
            throw new Error(status.message || 'Agent plugin build failed');
          } else {
            setCreationStatus(status.message || 'Building agent...');
            setCreationProgress(Math.min(40 + (attempts * 1.2), 75));
          }
        } catch (statusError) {
          console.warn('Error checking build status:', statusError);
        }

        attempts++;
      }

      if (attempts >= maxAttempts) {
        throw new Error('Agent creation timed out');
      }

    } catch (error) {
      console.error('Error creating agent:', error);
      throw error;
    }
  };

  // Sample data for capabilities and target types
  // In a real implementation, these would be fetched from the API
  useEffect(() => {
    // Load available agents when modal opens
    if (isOpen) {
      loadAvailableAgents();
    }

    // Set up available capabilities and target types
    setAvailableCapabilities([
      { id: 'web_analysis', name: 'Web Analysis' },
      { id: 'data_extraction', name: 'Data Extraction' },
      { id: 'content_monitoring', name: 'Content Monitoring' },
      { id: 'file_analysis', name: 'File Analysis' },
      { id: 'document_processing', name: 'Document Processing' },
      { id: 'security_analysis', name: 'Security Analysis' },
      { id: 'code_analysis', name: 'Code Analysis' },
      { id: 'natural_language_processing', name: 'Natural Language Processing' },
      { id: 'content_generation', name: 'Content Generation' },
      { id: 'research', name: 'Research' },
      { id: 'problem_solving', name: 'Problem Solving' }
    ]);

    setAvailableTargetTypes([
      { id: 'browser', name: 'Browser' },
      { id: 'filesystem', name: 'File System' },
      { id: 'application', name: 'Applications' },
      { id: 'system', name: 'System' },
      { id: 'network', name: 'Network' },
      { id: 'api', name: 'API Services' },
      { id: 'security', name: 'Security Systems' }
    ]);
  }, [isOpen]);

  // Use imported sample images from imageUrls.js
  const sampleImages = sampleAgentImages;

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    
    // Clear error for this field if it exists
    if (errors[name]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
  };

  const handleImageSelect = (url) => {
    setFormData(prev => ({ ...prev, imageURL: url }));
    
    // Clear error for imageURL if it exists
    if (errors.imageURL) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors.imageURL;
        return newErrors;
      });
    }
  };

  const handleCapabilityToggle = (capabilityId) => {
    setFormData(prev => {
      const newCapabilities = prev.capabilities.includes(capabilityId)
        ? prev.capabilities.filter(id => id !== capabilityId)
        : [...prev.capabilities, capabilityId];
      
      return { ...prev, capabilities: newCapabilities };
    });
    
    // Clear error for capabilities if it exists
    if (errors.capabilities) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors.capabilities;
        return newErrors;
      });
    }
  };

  const handleTargetTypeToggle = (targetTypeId) => {
    setFormData(prev => {
      const newTargetTypes = prev.targetTypes.includes(targetTypeId)
        ? prev.targetTypes.filter(id => id !== targetTypeId)
        : [...prev.targetTypes, targetTypeId];

      return { ...prev, targetTypes: newTargetTypes };
    });

    // Clear error for targetTypes if it exists
    if (errors.targetTypes) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors.targetTypes;
        return newErrors;
      });
    }
  };

  const handleApiKeyChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      apiKeys: {
        ...prev.apiKeys,
        [name]: value
      }
    }));
  };

  const validateForm = () => {
    const newErrors = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Agent name is required';
    }

    if (!formData.collection.trim()) {
      newErrors.collection = 'Collection name is required';
    }

    if (!formData.imageURL) {
      newErrors.imageURL = 'Please select an image for your agent';
    }

    // For 'select' mode, require an agent selection
    if (creationMode === 'select' && !selectedAgent) {
      newErrors.selectedAgent = 'Please select an existing agent';
    }

    // For 'create' mode, require capabilities and target types
    if (creationMode === 'create') {
      if (formData.capabilities.length === 0) {
        newErrors.capabilities = 'Select at least one capability';
      }

      if (formData.targetTypes.length === 0) {
        newErrors.targetTypes = 'Select at least one target type';
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);
    setSubmitStatus(null);
    setCreationProgress(0);
    setCreationStatus('');

    try {
      let createdAgent;

      if (creationMode === 'select') {
        // Use existing agent/plugin
        setCreationStatus('Setting up agent...');
        setCreationProgress(50);

        const configObj = {
          collection: formData.collection,
          image_url: formData.imageURL,
          capabilities: selectedAgent.capabilities,
          target_types: selectedAgent.targetTypes,
          api_keys: formData.apiKeys,
          status: 'idle',
          plugin_info: selectedAgent.plugin_info,
          // Add build target information for proper agent type identification
          build_target: selectedAgent.type || 'plugin',
          // Store the original agent ID for WASM/plugin loading
          source_agent_id: selectedAgent.id
        };

        const agentData = {
          name: formData.name,
          type: selectedAgent.type || 'plugin', // Use the actual agent type (wasm or plugin)
          config: JSON.stringify(configObj)
        };

        createdAgent = await createAgent(agentData);
        setCreationProgress(100);
        setCreationStatus('Agent created successfully!');

      } else {
        // Create new agent with automatic plugin building
        createdAgent = await createAgentWithPlugin();
      }

      setSubmitStatus('success');
      
      // Notify parent component of successful creation
      if (onAgentCreated) {
        onAgentCreated(createdAgent);
      }
      
      // Reset form after successful submission
      setTimeout(() => {
        setFormData({
          name: '',
          collection: '',
          imageURL: getDefaultAgentImage(),
          capabilities: [],
          targetTypes: [],
          apiKeys: {
            openai: '',
            claude: '',
            gemini: '',
            deepseek: '',
            cerebras: ''
          }
        });
        setSelectedAgent(null);
        setCreationMode('select');
        setSubmitStatus(null);
        setCreationProgress(0);
        setCreationStatus('');
        onClose();
      }, 1500);
      
    } catch (error) {
      console.error('Error creating agent:', error);
      handleApiError(error, {
        operation: 'create_agent',
        component: 'AgentCreationModal',
        agentName: formData.name,
        agentType: formData.type,
        timestamp: new Date().toISOString(),
        context: 'User attempted to create a new agent'
      });
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
        onClick={onClose}
      ></div>
      
      {/* Modal */}
      <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-3xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
              <Bot className="w-5 h-5 text-purple-400" />
            </div>
            <h2 className="text-xl font-bold text-white">Create New NFT-Agent</h2>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-200px)]">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Basic Information */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Basic Information</h3>
              
              <div>
                <label htmlFor="name" className="block text-sm font-medium text-slate-300 mb-2">
                  Agent Name
                </label>
                <input
                  type="text"
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Enter agent name (e.g., CyberPunk Agent #7804)"
                  className={`w-full bg-slate-700/50 border ${errors.name ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-red-400">{errors.name}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="collection" className="block text-sm font-medium text-slate-300 mb-2">
                  Collection
                </label>
                <input
                  type="text"
                  id="collection"
                  name="collection"
                  value={formData.collection}
                  onChange={handleChange}
                  placeholder="Enter collection name (e.g., CyberPunk Collective)"
                  className={`w-full bg-slate-700/50 border ${errors.collection ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                />
                {errors.collection && (
                  <p className="mt-1 text-sm text-red-400">{errors.collection}</p>
                )}
              </div>

              {/* Agent Creation Mode */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-3">
                  Agent Type *
                </label>

                {/* Mode Selection */}
                <div className="flex space-x-3 mb-4">
                  <button
                    type="button"
                    onClick={() => setCreationMode('select')}
                    className={`flex-1 p-3 rounded-lg border transition-colors ${
                      creationMode === 'select'
                        ? 'border-purple-500 bg-purple-500/20 text-white'
                        : 'border-slate-600 bg-slate-700/50 text-slate-300 hover:border-slate-500'
                    }`}
                  >
                    <div className="text-center">
                      <Bot className="w-6 h-6 mx-auto mb-2" />
                      <p className="font-medium">Use Existing Agent</p>
                      <p className="text-xs opacity-75">Choose from pre-built agents</p>
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={() => setCreationMode('create')}
                    className={`flex-1 p-3 rounded-lg border transition-colors ${
                      creationMode === 'create'
                        ? 'border-purple-500 bg-purple-500/20 text-white'
                        : 'border-slate-600 bg-slate-700/50 text-slate-300 hover:border-slate-500'
                    }`}
                  >
                    <div className="text-center">
                      <Plus className="w-6 h-6 mx-auto mb-2" />
                      <p className="font-medium">Create New Agent</p>
                      <p className="text-xs opacity-75">Build a custom agent</p>
                    </div>
                  </button>
                </div>

                {/* Build Target Selection (for 'create' mode) */}
                {creationMode === 'create' && (
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-3">
                      Build Target *
                    </label>
                    <div className="grid grid-cols-2 gap-3">
                      <button
                        type="button"
                        onClick={() => setFormData(prev => ({ ...prev, buildTarget: 'plugin' }))}
                        className={`p-3 rounded-lg border transition-colors ${
                          formData.buildTarget === 'plugin'
                            ? 'border-blue-500 bg-blue-500/20 text-white'
                            : 'border-slate-600 bg-slate-700/50 text-slate-300 hover:border-slate-500'
                        }`}
                      >
                        <div className="text-center">
                          <div className="w-8 h-8 mx-auto mb-2 bg-blue-500/20 rounded-lg flex items-center justify-center">
                            <span className="text-blue-400 font-bold text-sm">SO</span>
                          </div>
                          <p className="font-medium">Plugin (.so)</p>
                          <p className="text-xs opacity-75">Native Go plugin</p>
                        </div>
                      </button>

                      <button
                        type="button"
                        onClick={() => setFormData(prev => ({ ...prev, buildTarget: 'wasm' }))}
                        className={`p-3 rounded-lg border transition-colors ${
                          formData.buildTarget === 'wasm'
                            ? 'border-green-500 bg-green-500/20 text-white'
                            : 'border-slate-600 bg-slate-700/50 text-slate-300 hover:border-slate-500'
                        }`}
                      >
                        <div className="text-center">
                          <div className="w-8 h-8 mx-auto mb-2 bg-green-500/20 rounded-lg flex items-center justify-center">
                            <span className="text-green-400 font-bold text-sm">WA</span>
                          </div>
                          <p className="font-medium">WebAssembly</p>
                          <p className="text-xs opacity-75">Cross-platform WASM</p>
                        </div>
                      </button>
                    </div>

                    {/* Build Target Info */}
                    <div className="mt-3 p-3 bg-slate-700/30 rounded-lg border border-slate-600/30">
                      {formData.buildTarget === 'plugin' ? (
                        <div className="text-sm text-slate-300">
                          <p className="font-medium text-blue-400 mb-1">Go Plugin (.so)</p>
                          <p>Native performance with direct system access. Best for production environments with consistent architecture.</p>
                        </div>
                      ) : (
                        <div className="text-sm text-slate-300">
                          <p className="font-medium text-green-400 mb-1">WebAssembly (.wasm)</p>
                          <p>Cross-platform compatibility with sandboxed execution. Ideal for distributed deployments and enhanced security.</p>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Agent Selection (for 'select' mode) */}
                {creationMode === 'select' && (
                  <div>
                    {loadingAgents ? (
                      <div className="flex items-center space-x-2 text-slate-400 p-4">
                        <Loader2 className="w-4 h-4 animate-spin" />
                        <span>Loading available agents...</span>
                      </div>
                    ) : availableAgents.length === 0 ? (
                      <div className="text-slate-400 text-sm p-4 text-center border border-slate-600 rounded-lg">
                        No pre-built agents available. Try creating a new agent instead.
                      </div>
                    ) : (
                      <div className="space-y-2 max-h-32 overflow-y-auto">
                        {availableAgents.map((agent, index) => (
                          <div
                            key={index}
                            className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                              selectedAgent === agent
                                ? 'border-purple-500 bg-purple-500/20'
                                : 'border-slate-600 bg-slate-700/50 hover:border-slate-500'
                            }`}
                            onClick={() => setSelectedAgent(agent)}
                          >
                            <div className="flex items-center justify-between">
                              <div className="flex-1">
                                <div className="flex items-center justify-between mb-1">
                                  <p className="text-white font-medium">{agent.name}</p>
                                  <div className="flex items-center space-x-1">
                                    {agent.plugin_info && agent.plugin_info.filename && agent.plugin_info.filename.endsWith('.wasm') ? (
                                      <>
                                        <div className="w-2 h-2 bg-green-400 rounded-full"></div>
                                        <span className="text-green-400 text-xs font-medium">WASM</span>
                                      </>
                                    ) : (
                                      <>
                                        <div className="w-2 h-2 bg-blue-400 rounded-full"></div>
                                        <span className="text-blue-400 text-xs font-medium">Plugin</span>
                                      </>
                                    )}
                                  </div>
                                </div>
                                <p className="text-slate-400 text-sm">{agent.description}</p>
                                <p className="text-slate-500 text-xs">ID: {agent.id}</p>
                              </div>
                              {selectedAgent === agent && (
                                <Check className="w-5 h-5 text-purple-400 ml-2" />
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}

                    {errors.selectedAgent && (
                      <p className="mt-2 text-sm text-red-400">{errors.selectedAgent}</p>
                    )}
                  </div>
                )}
              </div>
            </div>
            
            {/* Agent Image */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Agent Image</h3>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Select an image for your agent
                </label>
                <div className="grid grid-cols-3 gap-3">
                  {sampleImages.map((url, index) => (
                    <div 
                      key={index}
                      onClick={() => handleImageSelect(url)}
                      className={`relative cursor-pointer rounded-lg overflow-hidden border-2 transition-all duration-200 ${
                        formData.imageURL === url 
                          ? 'border-purple-500 ring-2 ring-purple-500/50' 
                          : 'border-transparent hover:border-slate-500'
                      }`}
                    >
                      <AgentImage
                        src={url}
                        alt={`Agent image option ${index + 1}`}
                        className="w-full h-24 object-contain p-2"
                      />
                      {formData.imageURL === url && (
                        <div className="absolute top-2 right-2 bg-purple-500 rounded-full p-1">
                          <Check className="w-3 h-3 text-white" />
                        </div>
                      )}
                    </div>
                  ))}
                </div>
                {errors.imageURL && (
                  <p className="mt-1 text-sm text-red-400">{errors.imageURL}</p>
                )}
              </div>
            </div>
            
            {/* Capabilities - Only show when creating new agent */}
            {creationMode === 'create' && (
              <div className="space-y-4">
                <div className="flex items-center space-x-3">
                  <Zap className="w-5 h-5 text-purple-400" />
                  <h3 className="text-lg font-semibold text-white">Capabilities</h3>
                </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Select capabilities for your agent
                </label>
                <div className="grid grid-cols-2 gap-3">
                  {availableCapabilities.map((capability) => (
                    <div 
                      key={capability.id}
                      onClick={() => handleCapabilityToggle(capability.id)}
                      className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                        formData.capabilities.includes(capability.id)
                          ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 border-purple-500/30'
                          : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-white font-medium">{capability.name}</span>
                        {formData.capabilities.includes(capability.id) && (
                          <Check className="w-4 h-4 text-purple-400" />
                        )}
                      </div>
                    </div>
                  ))}
                </div>
                {errors.capabilities && (
                  <p className="mt-1 text-sm text-red-400">{errors.capabilities}</p>
                )}
              </div>
              </div>
            )}

            {/* Target Types - Only show when creating new agent */}
            {creationMode === 'create' && (
              <div className="space-y-4">
                <div className="flex items-center space-x-3">
                  <Target className="w-5 h-5 text-purple-400" />
                  <h3 className="text-lg font-semibold text-white">Target Types</h3>
                </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Select target types your agent can interact with
                </label>
                <div className="grid grid-cols-2 gap-3">
                  {availableTargetTypes.map((targetType) => (
                    <div 
                      key={targetType.id}
                      onClick={() => handleTargetTypeToggle(targetType.id)}
                      className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                        formData.targetTypes.includes(targetType.id)
                          ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 border-purple-500/30'
                          : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-white font-medium">{targetType.name}</span>
                        {formData.targetTypes.includes(targetType.id) && (
                          <Check className="w-4 h-4 text-purple-400" />
                        )}
                      </div>
                    </div>
                  ))}
                </div>
                {errors.targetTypes && (
                  <p className="mt-1 text-sm text-red-400">{errors.targetTypes}</p>
                )}
              </div>
              </div>
            )}

            {/* API Keys */}
            <div className="space-y-4">
              <div className="flex items-center space-x-3">
                <Key className="w-5 h-5 text-purple-400" />
                <h3 className="text-lg font-semibold text-white">API Keys</h3>
              </div>

              <div className="space-y-4">
                <div className="p-4 bg-blue-500/10 border border-blue-500/30 rounded-lg">
                  <div className="flex items-start space-x-3">
                    <AlertCircle className="w-5 h-5 text-blue-400 mt-0.5" />
                    <div>
                      <p className="text-blue-400 font-medium">Security Notice</p>
                      <p className="text-blue-300/70 text-sm">API keys are encrypted and stored securely. They are only used by your agent instances for LLM inference.</p>
                    </div>
                  </div>
                </div>

                <div>
                  <label htmlFor="openai" className="block text-sm font-medium text-slate-300 mb-2">
                    OpenAI API Key
                  </label>
                  <input
                    type="password"
                    id="openai"
                    name="openai"
                    value={formData.apiKeys.openai}
                    onChange={handleApiKeyChange}
                    placeholder="sk-..."
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  />
                </div>

                <div>
                  <label htmlFor="claude" className="block text-sm font-medium text-slate-300 mb-2">
                    Claude API Key
                  </label>
                  <input
                    type="password"
                    id="claude"
                    name="claude"
                    value={formData.apiKeys.claude}
                    onChange={handleApiKeyChange}
                    placeholder="sk-ant-..."
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  />
                </div>

                <div>
                  <label htmlFor="gemini" className="block text-sm font-medium text-slate-300 mb-2">
                    Google Gemini API Key
                  </label>
                  <input
                    type="password"
                    id="gemini"
                    name="gemini"
                    value={formData.apiKeys.gemini}
                    onChange={handleApiKeyChange}
                    placeholder="AIza..."
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  />
                </div>

                <div>
                  <label htmlFor="deepseek" className="block text-sm font-medium text-slate-300 mb-2">
                    Deepseek API Key
                  </label>
                  <input
                    type="password"
                    id="deepseek"
                    name="deepseek"
                    value={formData.apiKeys.deepseek}
                    onChange={handleApiKeyChange}
                    placeholder="sk-..."
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  />
                </div>

                <div>
                  <label htmlFor="cerebras" className="block text-sm font-medium text-slate-300 mb-2">
                    Cerebras API Key
                  </label>
                  <input
                    type="password"
                    id="cerebras"
                    name="cerebras"
                    value={formData.apiKeys.cerebras}
                    onChange={handleApiKeyChange}
                    placeholder="csk-..."
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  />
                </div>
              </div>
            </div>
          </form>
        </div>
        
        {/* Footer */}
        <div className="p-6 border-t border-slate-700 bg-slate-800/80 mt-auto">
          <div className="flex items-center justify-between">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200"
            >
              Cancel
            </button>
            
            <div className="flex items-center space-x-3">
              {submitStatus === 'success' && (
                <div className="flex items-center text-green-400 space-x-1">
                  <Check className="w-4 h-4" />
                  <span>Agent created successfully!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to create agent</span>
                </div>
              )}
              
              <button
                onClick={handleSubmit}
                disabled={isSubmitting}
                className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmitting ? (
                  <div className="flex items-center space-x-2">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <div className="flex flex-col items-start">
                      <span>{creationStatus || 'Creating...'}</span>
                      {creationMode === 'create' && creationProgress > 0 && (
                        <div className="w-24 h-1 bg-white/20 rounded-full mt-1">
                          <div
                            className="h-full bg-white rounded-full transition-all duration-300"
                            style={{ width: `${creationProgress}%` }}
                          />
                        </div>
                      )}
                    </div>
                  </div>
                ) : (
                  <>
                    <Bot className="w-4 h-4" />
                    <span>{creationMode === 'create' ? 'Create New Agent' : 'Use Selected Agent'}</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AgentCreationModal;