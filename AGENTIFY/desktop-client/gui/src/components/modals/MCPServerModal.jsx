import React, { useState, useEffect } from 'react';
import { 
  X, 
  Server, 
  Save, 
  AlertCircle,
  Check,
  Brain,
  Globe,
  Database,
  Shield,
  Terminal,
  Cpu,
  RefreshCw
} from 'lucide-react';

const MCPServerModal = ({ isOpen, onClose, server, onServerSaved }) => {
  const [formData, setFormData] = useState({
    name: '',
    type: 'llm',
    description: '',
    endpoint: '',
    apiKey: '',
    version: '1.0.0'
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [testingConnection, setTestingConnection] = useState(false);
  const [testResult, setTestResult] = useState(null); // 'success', 'error', or null
  const [showApiKey, setShowApiKey] = useState(false);

  // Available server types
  const serverTypes = [
    { id: 'llm', name: 'LLM Provider', icon: Brain, description: 'Language model provider for text generation and analysis' },
    { id: 'tool', name: 'Tool Provider', icon: Terminal, description: 'Provides tools and utilities for agents' },
    { id: 'data', name: 'Data Provider', icon: Database, description: 'Provides data access and storage capabilities' },
    { id: 'web', name: 'Web Service', icon: Globe, description: 'Web-based services and APIs' },
    { id: 'security', name: 'Security Service', icon: Shield, description: 'Security and authentication services' },
    { id: 'compute', name: 'Compute Service', icon: Cpu, description: 'Computational resources and processing' }
  ];

  // Initialize form data when server prop changes
  useEffect(() => {
    if (isOpen && server) {
      // Editing existing server
      setFormData({
        name: server.name || '',
        type: server.type || 'llm',
        description: server.description || '',
        endpoint: server.endpoint || '',
        apiKey: server.apiKey || '',
        version: server.version || '1.0.0'
      });
    } else if (isOpen) {
      // Creating new server
      setFormData({
        name: '',
        type: 'llm',
        description: '',
        endpoint: '',
        apiKey: '',
        version: '1.0.0'
      });
    }
    
    // Reset errors and status
    setErrors({});
    setSubmitStatus(null);
    setTestResult(null);
    setShowApiKey(false);
  }, [isOpen, server]);

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

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Server name is required';
    }
    
    if (!formData.type) {
      newErrors.type = 'Server type is required';
    }
    
    if (!formData.endpoint.trim()) {
      newErrors.endpoint = 'Endpoint URL is required';
    } else if (!isValidURL(formData.endpoint)) {
      newErrors.endpoint = 'Please enter a valid URL';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const isValidURL = (string) => {
    try {
      new URL(string);
      return true;
    } catch (_) {
      return false;
    }
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
      
      // Prepare the server data for the API
      const serverData = {
        ...formData,
        status: 'connected', // Default status for new servers
        lastSync: 'just now',
        id: server ? server.id : Date.now(), // Use existing ID or generate a new one
        capabilities: server ? server.capabilities : [],
        health: {
          status: 'healthy',
          uptime: '100%',
          latency: '50ms',
          lastChecked: 'just now'
        },
        icon: getServerTypeIcon(formData.type)
      };
      
      // Simulate API call
      // const response = await fetch(server ? `/api/v1/mcp/servers/${server.id}` : '/api/v1/mcp/servers', {
      //   method: server ? 'PUT' : 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(serverData),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Failed to save server');
      // }
      
      // const data = await response.json();
      
      setSubmitStatus('success');
      
      // Notify parent component of successful save
      if (onServerSaved) {
        onServerSaved(serverData);
      }
      
      // Close modal after successful submission
      setTimeout(() => {
        setSubmitStatus(null);
        onClose();
      }, 1500);
      
    } catch (error) {
      console.error('Error saving server:', error);
      setSubmitStatus('error');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleTestConnection = async () => {
    if (!formData.endpoint.trim() || !isValidURL(formData.endpoint)) {
      setErrors(prev => ({
        ...prev,
        endpoint: !formData.endpoint.trim() ? 'Endpoint URL is required' : 'Please enter a valid URL'
      }));
      return;
    }
    
    setTestingConnection(true);
    setTestResult(null);
    
    try {
      // In a real implementation, this would be an API call to test the connection
      // For now, we'll simulate a connection test
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // Simulate API call
      // const response = await fetch('/api/v1/mcp/servers/test-connection', {
      //   method: 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(formData),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Connection test failed');
      // }
      
      // const data = await response.json();
      
      // 80% chance of success for simulation
      const success = Math.random() < 0.8;
      
      if (success) {
        setTestResult('success');
      } else {
        throw new Error('Connection test failed');
      }
    } catch (error) {
      console.error('Error testing connection:', error);
      setTestResult('error');
    } finally {
      setTestingConnection(false);
    }
  };

  const getServerTypeIcon = (typeId) => {
    const serverType = serverTypes.find(type => type.id === typeId);
    return serverType ? serverType.icon : Server;
  };

  if (!isOpen) return null;

  const ServerTypeIcon = getServerTypeIcon(formData.type);

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
              <Server className="w-5 h-5 text-purple-400" />
            </div>
            <h2 className="text-xl font-bold text-white">
              {server ? 'Edit MCP Server' : 'Add MCP Server'}
            </h2>
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
                  Server Name
                </label>
                <input
                  type="text"
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Enter MCP server name (e.g., OpenAI MCP Server)"
                  className={`w-full bg-slate-700/50 border ${errors.name ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-red-400">{errors.name}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="type" className="block text-sm font-medium text-slate-300 mb-2">
                  Server Type
                </label>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                  {serverTypes.map((type) => {
                    const TypeIcon = type.icon;
                    return (
                      <div
                        key={type.id}
                        onClick={() => handleChange({ target: { name: 'type', value: type.id } })}
                        className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                          formData.type === type.id
                            ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 border-purple-500/30'
                            : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                        }`}
                      >
                        <div className="flex flex-col items-center text-center">
                          <TypeIcon className={`w-6 h-6 mb-2 ${formData.type === type.id ? 'text-purple-400' : 'text-slate-400'}`} />
                          <span className={`text-sm font-medium ${formData.type === type.id ? 'text-white' : 'text-slate-300'}`}>
                            {type.name}
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
                {errors.type && (
                  <p className="mt-1 text-sm text-red-400">{errors.type}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="description" className="block text-sm font-medium text-slate-300 mb-2">
                  Description
                </label>
                <textarea
                  id="description"
                  name="description"
                  value={formData.description}
                  onChange={handleChange}
                  placeholder="Enter a description of this MCP server"
                  rows="3"
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                ></textarea>
              </div>
            </div>
            
            {/* Connection Details */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Connection Details</h3>
              
              <div>
                <label htmlFor="endpoint" className="block text-sm font-medium text-slate-300 mb-2">
                  Endpoint URL
                </label>
                <input
                  type="text"
                  id="endpoint"
                  name="endpoint"
                  value={formData.endpoint}
                  onChange={handleChange}
                  placeholder={`Enter endpoint URL (e.g., ${
                    formData.type === 'llm' ? 'https://api.openai.com/v1' :
                    formData.type === 'tool' ? 'http://localhost:8081/tools' :
                    formData.type === 'data' ? 'http://localhost:8082/data' :
                    formData.type === 'web' ? 'https://api.example.com' :
                    formData.type === 'security' ? 'https://security.example.com/api' :
                    formData.type === 'compute' ? 'http://localhost:5000' :
                    'https://api.example.com'
                  })`}
                  className={`w-full bg-slate-700/50 border ${errors.endpoint ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500`}
                />
                {errors.endpoint && (
                  <p className="mt-1 text-sm text-red-400">{errors.endpoint}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="apiKey" className="block text-sm font-medium text-slate-300 mb-2">
                  API Key (if required)
                </label>
                <div className="relative">
                  <input
                    type={showApiKey ? "text" : "password"}
                    id="apiKey"
                    name="apiKey"
                    value={formData.apiKey}
                    onChange={handleChange}
                    placeholder="Enter API key if required for authentication"
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  />
                  <button
                    type="button"
                    onClick={() => setShowApiKey(!showApiKey)}
                    className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slate-400 hover:text-white"
                  >
                    {showApiKey ? "Hide" : "Show"}
                  </button>
                </div>
                <p className="mt-1 text-xs text-slate-400">API keys are stored securely and never exposed</p>
              </div>
              
              <div>
                <label htmlFor="version" className="block text-sm font-medium text-slate-300 mb-2">
                  Version
                </label>
                <input
                  type="text"
                  id="version"
                  name="version"
                  value={formData.version}
                  onChange={handleChange}
                  placeholder="Enter server version (e.g., 1.0.0)"
                  className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
                />
              </div>
              
              <div>
                <button
                  type="button"
                  onClick={handleTestConnection}
                  disabled={testingConnection}
                  className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200 flex items-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {testingConnection ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      <span>Testing Connection...</span>
                    </>
                  ) : (
                    <>
                      <ServerTypeIcon className="w-4 h-4" />
                      <span>Test Connection</span>
                    </>
                  )}
                </button>
                
                {testResult === 'success' && (
                  <div className="mt-2 p-3 bg-green-500/20 border border-green-500/30 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <Check className="w-5 h-5 text-green-400" />
                      <span className="text-green-400 font-medium">Connection successful!</span>
                    </div>
                    <p className="text-green-300/70 text-sm mt-1">MCP server is accessible and ready for use.</p>
                  </div>
                )}
                
                {testResult === 'error' && (
                  <div className="mt-2 p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <AlertCircle className="w-5 h-5 text-red-400" />
                      <span className="text-red-400 font-medium">Connection failed!</span>
                    </div>
                    <p className="text-red-300/70 text-sm mt-1">Unable to connect to the MCP server. Please check your endpoint URL and API key.</p>
                  </div>
                )}
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
                  <span>Server saved!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to save server</span>
                </div>
              )}
              
              <button
                onClick={handleSubmit}
                disabled={isSubmitting}
                className="px-6 py-2 bg-gradient-to-r from-purple-500 to-blue-500 text-white rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmitting ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4" />
                    <span>Save Server</span>
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

export default MCPServerModal;