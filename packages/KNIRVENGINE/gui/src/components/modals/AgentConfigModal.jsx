import React, { useState, useEffect } from 'react';
import {
  X,
  Bot,
  Zap,
  Target,
  Settings,
  Save,
  AlertCircle,
  Check,
  Sliders,
  Shield,
  Clock,
  Cpu,
  BarChart3,
  Key,
  Trash2
} from 'lucide-react';
import { deleteAgent } from '../../utils/api';

const AgentConfigModal = ({ isOpen, onClose, agent, onAgentUpdated, onDelete }) => {
  const [formData, setFormData] = useState({
    name: '',
    collection: '',
    capabilities: [],
    targetTypes: [],
    apiKeys: {
      openai: '',
      claude: '',
      gemini: '',
      deepseek: '',
      cerebras: ''
    },
    settings: {
      responseLength: 'medium',
      responseStyle: 'balanced',
      maxInferences: 100,
      securityLevel: 'standard',
      timeout: 30,
      autoRestart: true,
      logLevel: 'info'
    }
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [activeTab, setActiveTab] = useState('general');

  // Sample data for capabilities and target types
  const [availableCapabilities, setAvailableCapabilities] = useState([]);
  const [availableTargetTypes, setAvailableTargetTypes] = useState([]);

  useEffect(() => {
    if (isOpen && agent) {
      // Initialize form data with agent data
      setFormData({
        name: agent.name || '',
        collection: agent.collection || '',
        capabilities: agent.capabilities || [],
        targetTypes: agent.targetTypes || [],
        apiKeys: {
          openai: agent.apiKeys?.openai || '',
          claude: agent.apiKeys?.claude || '',
          gemini: agent.apiKeys?.gemini || '',
          deepseek: agent.apiKeys?.deepseek || '',
          cerebras: agent.apiKeys?.cerebras || ''
        },
        settings: {
          responseLength: agent.settings?.responseLength || 'medium',
          responseStyle: agent.settings?.responseStyle || 'balanced',
          maxInferences: agent.settings?.maxInferences || 100,
          securityLevel: agent.settings?.securityLevel || 'standard',
          timeout: agent.settings?.timeout || 30,
          autoRestart: agent.settings?.autoRestart !== undefined ? agent.settings.autoRestart : true,
          logLevel: agent.settings?.logLevel || 'info'
        }
      });
      
      // Reset errors and status
      setErrors({});
      setSubmitStatus(null);
      
      // Set default active tab
      setActiveTab('general');
      
      // Fetch available capabilities and target types
      // In a real implementation, these would be API calls
      setAvailableCapabilities([
        { id: 'web_analysis', name: 'Web Analysis' },
        { id: 'data_extraction', name: 'Data Extraction' },
        { id: 'content_monitoring', name: 'Content Monitoring' },
        { id: 'file_analysis', name: 'File Analysis' },
        { id: 'document_processing', name: 'Document Processing' },
        { id: 'data_mining', name: 'Data Mining' },
        { id: 'security_analysis', name: 'Security Analysis' },
        { id: 'threat_detection', name: 'Threat Detection' },
        { id: 'network_monitoring', name: 'Network Monitoring' },
        { id: 'code_analysis', name: 'Code Analysis' },
        { id: 'quality_assessment', name: 'Quality Assessment' },
        { id: 'bug_detection', name: 'Bug Detection' },
        { id: 'image_processing', name: 'Image Processing' },
        { id: 'media_analysis', name: 'Media Analysis' },
        { id: 'creative_enhancement', name: 'Creative Enhancement' }
      ]);
      
      setAvailableTargetTypes([
        { id: 'browser', name: 'Browser' },
        { id: 'filesystem', name: 'File System' },
        { id: 'application', name: 'Applications' },
        { id: 'system', name: 'System' },
        { id: 'network', name: 'Network' },
        { id: 'database', name: 'Database' },
        { id: 'api', name: 'API' },
        { id: 'mobile', name: 'Mobile' }
      ]);
    }
  }, [isOpen, agent]);

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

  const handleSettingChange = (e) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      settings: {
        ...prev.settings,
        [name]: type === 'checkbox' ? checked : value
      }
    }));

    // Clear error for this field if it exists
    if (errors[`settings.${name}`]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[`settings.${name}`];
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

  const handleCapabilityToggle = (capability) => {
    setFormData(prev => {
      const newCapabilities = prev.capabilities.includes(capability)
        ? prev.capabilities.filter(cap => cap !== capability)
        : [...prev.capabilities, capability];
      
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

  const handleTargetTypeToggle = (targetType) => {
    setFormData(prev => {
      const newTargetTypes = prev.targetTypes.includes(targetType)
        ? prev.targetTypes.filter(type => type !== targetType)
        : [...prev.targetTypes, targetType];
      
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

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Agent name is required';
    }
    
    if (!formData.collection.trim()) {
      newErrors.collection = 'Collection name is required';
    }
    
    if (formData.capabilities.length === 0) {
      newErrors.capabilities = 'Select at least one capability';
    }
    
    if (formData.targetTypes.length === 0) {
      newErrors.targetTypes = 'Select at least one target type';
    }
    
    // Validate settings
    if (formData.settings.maxInferences <= 0) {
      newErrors['settings.maxInferences'] = 'Max inferences must be greater than 0';
    }
    
    if (formData.settings.timeout <= 0) {
      newErrors['settings.timeout'] = 'Timeout must be greater than 0';
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
    
    try {
      // In a real implementation, this would be an API call
      // For now, we'll simulate a successful API call
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // Prepare the updated agent data
      const updatedAgent = {
        ...agent,
        name: formData.name,
        collection: formData.collection,
        capabilities: formData.capabilities,
        targetTypes: formData.targetTypes,
        apiKeys: formData.apiKeys,
        settings: formData.settings
      };
      
      // Simulate API call
      // const response = await fetch(`/api/v1/agents/${agent.id}`, {
      //   method: 'PUT',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(updatedAgent),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Failed to update agent');
      // }
      
      // const data = await response.json();
      
      setSubmitStatus('success');
      
      // Notify parent component of successful update
      if (onAgentUpdated) {
        onAgentUpdated(updatedAgent);
      }
      
      // Close modal after successful submission
      setTimeout(() => {
        setSubmitStatus(null);
        onClose();
      }, 1500);
      
    } catch (error) {
      console.error('Error updating agent:', error);
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!agent || isDeleting) return;

    if (!window.confirm(`Delete "${agent.name}"? This action cannot be undone.`)) return;

    setIsDeleting(true);
    try {
      await deleteAgent(agent.id);
      await onDelete?.(agent);
      onClose();
    } catch (error) {
      console.error('Error deleting agent:', error);
      setSubmitStatus('error');
    } finally {
      setIsDeleting(false);
    }
  };

  if (!isOpen || !agent) return null;

  const tabs = [
    { id: 'general', name: 'General', icon: Bot },
    { id: 'capabilities', name: 'Capabilities', icon: Zap },
    { id: 'targets', name: 'Target Types', icon: Target },
    { id: 'apikeys', name: 'API Keys', icon: Key },
    { id: 'performance', name: 'Performance', icon: BarChart3 },
    { id: 'security', name: 'Security', icon: Shield },
    { id: 'advanced', name: 'Advanced', icon: Settings }
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
        onClick={onClose}
      ></div>
      
      {/* Modal */}
      <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-4xl max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div className="flex items-center space-x-3">
            <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
              <Settings className="w-5 h-5 text-purple-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">Agent Configuration</h2>
              <p className="text-slate-400 text-sm">{agent.name}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        
        {/* Tabs */}
        <div className="flex border-b border-slate-700" role="tablist">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                role="tab"
                aria-selected={activeTab === tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center space-x-2 px-4 py-3 transition-all duration-200 ${
                  activeTab === tab.id
                    ? 'bg-slate-700/50 text-white border-b-2 border-purple-500'
                    : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
                }`}
              >
                <Icon className="w-4 h-4" />
                <span>{tab.name}</span>
              </button>
            );
          })}
        </div>
        
        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[calc(90vh-220px)]">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* General Settings */}
            {activeTab === 'general' && (
              <div className="space-y-4">
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
                    className={`w-full bg-slate-700/50 border ${errors.collection ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                  />
                  {errors.collection && (
                    <p className="mt-1 text-sm text-red-400">{errors.collection}</p>
                  )}
                </div>
                
                <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                  <h4 className="text-white font-medium mb-3">Agent Information</h4>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-slate-400">Status:</span>
                      <p className="text-white font-medium capitalize">{agent.status || 'Idle'}</p>
                    </div>
                    <div>
                      <span className="text-slate-400">ID:</span>
                      <p className="text-white font-medium">{agent.id}</p>
                    </div>
                    <div>
                      <span className="text-slate-400">Total Inferences:</span>
                      <p className="text-white font-medium">{agent.totalInferences?.toLocaleString() || '0'}</p>
                    </div>
                    <div>
                      <span className="text-slate-400">Success Rate:</span>
                      <p className="text-white font-medium">{agent.successRate || '0'}%</p>
                    </div>
                    <div>
                      <span className="text-slate-400">Last Activity:</span>
                      <p className="text-white font-medium">{agent.lastActivity || 'Never'}</p>
                    </div>
                    <div>
                      <span className="text-slate-400">Current Target:</span>
                      <p className="text-white font-medium">{agent.currentTarget || 'None'}</p>
                    </div>
                  </div>
                </div>
              </div>
            )}
            
            {/* Capabilities */}
            {activeTab === 'capabilities' && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <Zap className="w-5 h-5 text-purple-400" />
                    <h3 className="text-lg font-semibold text-white">Agent Capabilities</h3>
                  </div>
                  <span className="text-slate-400 text-sm">{formData.capabilities.length} selected</span>
                </div>
                
                <div className="grid grid-cols-2 gap-3">
                  {availableCapabilities.map((capability) => (
                    <div 
                      key={capability.id}
                      onClick={() => handleCapabilityToggle(capability.name)}
                      className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                        formData.capabilities.includes(capability.name)
                          ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 border-purple-500/30'
                          : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-white font-medium">{capability.name}</span>
                        {formData.capabilities.includes(capability.name) && (
                          <Check className="w-4 h-4 text-purple-400" />
                        )}
                      </div>
                    </div>
                  ))}
                </div>
                {errors.capabilities && (
                  <p className="text-sm text-red-400">{errors.capabilities}</p>
                )}
              </div>
            )}
            
            {/* Target Types */}
            {activeTab === 'targets' && (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <Target className="w-5 h-5 text-blue-400" />
                    <h3 className="text-lg font-semibold text-white">Target Types</h3>
                  </div>
                  <span className="text-slate-400 text-sm">{formData.targetTypes.length} selected</span>
                </div>
                
                <div className="grid grid-cols-2 gap-3">
                  {availableTargetTypes.map((targetType) => (
                    <div 
                      key={targetType.id}
                      onClick={() => handleTargetTypeToggle(targetType.name)}
                      className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                        formData.targetTypes.includes(targetType.name)
                          ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 border-blue-500/30'
                          : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-white font-medium">{targetType.name}</span>
                        {formData.targetTypes.includes(targetType.name) && (
                          <Check className="w-4 h-4 text-blue-400" />
                        )}
                      </div>
                    </div>
                  ))}
                </div>
                {errors.targetTypes && (
                  <p className="text-sm text-red-400">{errors.targetTypes}</p>
                )}
              </div>
            )}

            {/* API Keys */}
            {activeTab === 'apikeys' && (
              <div className="space-y-6">
                <div className="flex items-center space-x-2 mb-4">
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
            )}

            {/* Performance Settings */}
            {activeTab === 'performance' && (
              <div className="space-y-6">
                <div className="flex items-center space-x-2 mb-4">
                  <BarChart3 className="w-5 h-5 text-purple-400" />
                  <h3 className="text-lg font-semibold text-white">Performance Settings</h3>
                </div>
                
                <div className="space-y-4">
                  <div>
                    <label htmlFor="responseLength" className="block text-sm font-medium text-slate-300 mb-2">
                      Response Length
                    </label>
                    <select
                      id="responseLength"
                      name="responseLength"
                      value={formData.settings.responseLength}
                      onChange={handleSettingChange}
                      className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                    >
                      <option value="short">Short (Concise responses)</option>
                      <option value="medium">Medium (Balanced responses)</option>
                      <option value="long">Long (Detailed responses)</option>
                    </select>
                  </div>
                  
                  <div>
                    <label htmlFor="responseStyle" className="block text-sm font-medium text-slate-300 mb-2">
                      Response Style
                    </label>
                    <select
                      id="responseStyle"
                      name="responseStyle"
                      value={formData.settings.responseStyle}
                      onChange={handleSettingChange}
                      className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                    >
                      <option value="creative">Creative (More varied and innovative)</option>
                      <option value="balanced">Balanced (Mix of creative and precise)</option>
                      <option value="precise">Precise (More factual and accurate)</option>
                    </select>
                  </div>
                  
                  <div>
                    <label htmlFor="maxInferences" className="block text-sm font-medium text-slate-300 mb-2">
                      Maximum Inferences Per Day
                    </label>
                    <input
                      type="number"
                      id="maxInferences"
                      name="maxInferences"
                      value={formData.settings.maxInferences}
                      onChange={handleSettingChange}
                      min="1"
                      className={`w-full bg-slate-700/50 border ${errors['settings.maxInferences'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500`}
                    />
                    {errors['settings.maxInferences'] && (
                      <p className="mt-1 text-sm text-red-400">{errors['settings.maxInferences']}</p>
                    )}
                  </div>
                  
                  <div>
                    <label htmlFor="timeout" className="block text-sm font-medium text-slate-300 mb-2">
                      Inference Timeout (seconds)
                    </label>
                    <input
                      type="number"
                      id="timeout"
                      name="timeout"
                      value={formData.settings.timeout}
                      onChange={handleSettingChange}
                      min="1"
                      className={`w-full bg-slate-700/50 border ${errors['settings.timeout'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500`}
                    />
                    {errors['settings.timeout'] && (
                      <p className="mt-1 text-sm text-red-400">{errors['settings.timeout']}</p>
                    )}
                  </div>
                </div>
              </div>
            )}
            
            {/* Security Settings */}
            {activeTab === 'security' && (
              <div className="space-y-6">
                <div className="flex items-center space-x-2 mb-4">
                  <Shield className="w-5 h-5 text-purple-400" />
                  <h3 className="text-lg font-semibold text-white">Security Settings</h3>
                </div>
                
                <div className="space-y-4">
                  <div>
                    <label htmlFor="securityLevel" className="block text-sm font-medium text-slate-300 mb-2">
                      Security Level
                    </label>
                    <select
                      id="securityLevel"
                      name="securityLevel"
                      value={formData.settings.securityLevel}
                      onChange={handleSettingChange}
                      className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                    >
                      <option value="basic">Basic (Minimal restrictions)</option>
                      <option value="standard">Standard (Balanced security)</option>
                      <option value="strict">Strict (Maximum security)</option>
                    </select>
                  </div>
                  
                  <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                    <h4 className="text-white font-medium mb-3">Security Level Details</h4>
                    <div className="space-y-2 text-sm">
                      {formData.settings.securityLevel === 'basic' && (
                        <p className="text-slate-300">Basic security provides minimal restrictions, allowing the agent to access most resources. Suitable for trusted environments.</p>
                      )}
                      {formData.settings.securityLevel === 'standard' && (
                        <p className="text-slate-300">Standard security provides a balance between access and protection. The agent will have limited access to sensitive resources and operations.</p>
                      )}
                      {formData.settings.securityLevel === 'strict' && (
                        <p className="text-slate-300">Strict security enforces maximum restrictions. The agent will have minimal access to resources and will require explicit permission for most operations.</p>
                      )}
                    </div>
                  </div>
                  
                  <div className="p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
                    <div className="flex items-start space-x-3">
                      <AlertCircle className="w-5 h-5 text-yellow-400 mt-0.5" />
                      <div>
                        <p className="text-yellow-400 font-medium">Security Recommendation</p>
                        <p className="text-yellow-300/70 text-sm">For agents with access to sensitive systems or data, we recommend using at least Standard security level.</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}
            
            {/* Advanced Settings */}
            {activeTab === 'advanced' && (
              <div className="space-y-6">
                <div className="flex items-center space-x-2 mb-4">
                  <Sliders className="w-5 h-5 text-purple-400" />
                  <h3 className="text-lg font-semibold text-white">Advanced Settings</h3>
                </div>
                
                <div className="space-y-4">
                  <div className="flex items-center justify-between p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                    <div className="flex items-center space-x-3">
                      <Cpu className="w-5 h-5 text-purple-400" />
                      <div>
                        <p className="text-white font-medium">Auto Restart on Failure</p>
                        <p className="text-slate-400 text-sm">Automatically restart the agent if it crashes or fails</p>
                      </div>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        name="autoRestart"
                        checked={formData.settings.autoRestart}
                        onChange={handleSettingChange}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-slate-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-purple-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-purple-500"></div>
                    </label>
                  </div>
                  
                  <div>
                    <label htmlFor="logLevel" className="block text-sm font-medium text-slate-300 mb-2">
                      Log Level
                    </label>
                    <select
                      id="logLevel"
                      name="logLevel"
                      value={formData.settings.logLevel}
                      onChange={handleSettingChange}
                      className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
                    >
                      <option value="error">Error (Minimal logging)</option>
                      <option value="warn">Warning (Important issues only)</option>
                      <option value="info">Info (General information)</option>
                      <option value="debug">Debug (Detailed information)</option>
                      <option value="trace">Trace (Maximum detail)</option>
                    </select>
                  </div>
                  
                  <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                    <h4 className="text-white font-medium mb-3">Agent Metadata</h4>
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <span className="text-slate-400">Token ID:</span>
                        <p className="text-white font-medium">{agent.tokenID || 'N/A'}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">Contract Address:</span>
                        <p className="text-white font-medium">{agent.contractAddr || 'N/A'}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">Created:</span>
                        <p className="text-white font-medium">{agent.createdAt || 'Unknown'}</p>
                      </div>
                      <div>
                        <span className="text-slate-400">Last Updated:</span>
                        <p className="text-white font-medium">{agent.updatedAt || 'Never'}</p>
                      </div>
                    </div>
                  </div>
                  
                  <div className="p-4 bg-red-500/10 border border-red-500/20 rounded-lg">
                    <div className="flex items-start space-x-3">
                      <AlertCircle className="w-5 h-5 text-red-400 mt-0.5" />
                      <div>
                        <p className="text-red-400 font-medium">Danger Zone</p>
                        <p className="text-red-300/70 text-sm mb-3">These actions cannot be undone</p>
                        <button
                          type="button"
                          onClick={handleDelete}
                          disabled={isDeleting}
                          className="bg-red-500/20 hover:bg-red-500/30 border border-red-500/30 text-red-400 px-4 py-2 rounded-lg text-sm font-medium transition-colors duration-200"
                        >
                          <span className="flex items-center gap-2">
                            <Trash2 className="w-4 h-4" />
                            {isDeleting ? 'Deleting…' : 'Delete Agent'}
                          </span>
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </form>
        </div>
        
        {/* Footer */}
        <div className="p-6 border-t border-slate-700 bg-slate-800/80">
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
                  <span>Configuration saved!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to save configuration</span>
                </div>
              )}
              
              <button
                onClick={handleSubmit}
                disabled={isSubmitting}
                className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmitting ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4" />
                    <span>Save Configuration</span>
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

export default AgentConfigModal;
