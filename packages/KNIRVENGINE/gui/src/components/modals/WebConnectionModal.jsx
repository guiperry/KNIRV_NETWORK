import React, { useState, useEffect } from 'react';
import { handleApiError } from '../../utils/errorHandler';
import {
  X,
  Globe,
  Save,
  AlertCircle,
  Check,
  Key,
  Lock,
  Eye,
  EyeOff,
  RefreshCw,
  FileJson,
  Upload,
  ExternalLink,
  Plus
} from 'lucide-react';

const WebConnectionModal = ({ isOpen, onClose, connection, onConnectionSaved }) => {
  const [formData, setFormData] = useState({
    name: '',
    type: 'oauth',
    provider: '',
    description: '',
    scopes: [],
    connectionDetails: {
      clientId: '',
      clientSecret: '',
      redirectUri: 'https://app.picaos.com/authkit/callback',
      tokenEndpoint: '',
      authEndpoint: '',
      apiKey: '',
      baseUrl: '',
      accessKeyId: '',
      secretAccessKey: '',
      region: '',
      bucket: ''
    }
  });
  const [errors, setErrors] = useState({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState(null); // 'success', 'error', or null
  const [showSecrets, setShowSecrets] = useState({
    clientSecret: false,
    apiKey: false,
    secretAccessKey: false
  });
  const [scopeInput, setScopeInput] = useState('');

  // Available connection types
  const connectionTypes = [
    { id: 'oauth', name: 'OAuth 2.0', description: 'Connect using OAuth 2.0 authorization' },
    { id: 'apikey', name: 'API Key', description: 'Connect using an API key or token' }
  ];

  // Common providers
  const commonProviders = [
    { id: 'google', name: 'Google', type: 'oauth', icon: 'https://upload.wikimedia.org/wikipedia/commons/5/53/Google_%22G%22_Logo.svg' },
    { id: 'github', name: 'GitHub', type: 'oauth', icon: 'https://github.githubassets.com/assets/GitHub-Mark-ea2971cee799.png' },
    { id: 'slack', name: 'Slack', type: 'oauth', icon: 'https://upload.wikimedia.org/wikipedia/commons/d/d5/Slack_icon_2019.svg' },
    { id: 'salesforce', name: 'Salesforce', type: 'oauth', icon: 'https://upload.wikimedia.org/wikipedia/commons/f/f9/Salesforce.com_logo.svg' },
    { id: 'microsoft', name: 'Microsoft', type: 'oauth', icon: 'https://upload.wikimedia.org/wikipedia/commons/4/44/Microsoft_logo.svg' },
    { id: 'openai', name: 'OpenAI', type: 'apikey', icon: 'https://upload.wikimedia.org/wikipedia/commons/0/04/ChatGPT_logo.svg' },
    { id: 'aws', name: 'AWS', type: 'apikey', icon: 'https://upload.wikimedia.org/wikipedia/commons/9/93/Amazon_Web_Services_Logo.svg' },
    { id: 'azure', name: 'Azure', type: 'apikey', icon: 'https://upload.wikimedia.org/wikipedia/commons/f/fa/Microsoft_Azure.svg' },
    { id: 'custom', name: 'Custom', type: 'both', icon: 'https://upload.wikimedia.org/wikipedia/commons/5/5c/Gears_0.png' }
  ];

  // Initialize form data when connection prop changes
  useEffect(() => {
    if (isOpen && connection) {
      // Editing existing connection
      setFormData({
        name: connection.name || '',
        type: connection.type || 'oauth',
        provider: connection.provider || '',
        description: connection.description || '',
        scopes: connection.scopes || [],
        connectionDetails: {
          clientId: connection.connectionDetails?.clientId || '',
          clientSecret: connection.connectionDetails?.clientSecret || '',
          redirectUri: connection.connectionDetails?.redirectUri || 'https://app.picaos.com/authkit/callback',
          tokenEndpoint: connection.connectionDetails?.tokenEndpoint || '',
          authEndpoint: connection.connectionDetails?.authEndpoint || '',
          apiKey: connection.connectionDetails?.apiKey || '',
          baseUrl: connection.connectionDetails?.baseUrl || '',
          accessKeyId: connection.connectionDetails?.accessKeyId || '',
          secretAccessKey: connection.connectionDetails?.secretAccessKey || '',
          region: connection.connectionDetails?.region || '',
          bucket: connection.connectionDetails?.bucket || ''
        }
      });
    } else if (isOpen) {
      // Creating new connection
      setFormData({
        name: '',
        type: 'oauth',
        provider: '',
        description: '',
        scopes: [],
        connectionDetails: {
          clientId: '',
          clientSecret: '',
          redirectUri: 'https://app.picaos.com/authkit/callback',
          tokenEndpoint: '',
          authEndpoint: '',
          apiKey: '',
          baseUrl: '',
          accessKeyId: '',
          secretAccessKey: '',
          region: '',
          bucket: ''
        }
      });
    }
    
    // Reset errors and status
    setErrors({});
    setSubmitStatus(null);
    setShowSecrets({
      clientSecret: false,
      apiKey: false,
      secretAccessKey: false
    });
    setScopeInput('');
  }, [isOpen, connection]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    
    // Handle nested properties in connectionDetails
    if (name.includes('.')) {
      const [parent, child] = name.split('.');
      setFormData(prev => ({
        ...prev,
        [parent]: {
          ...prev[parent],
          [child]: value
        }
      }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
    
    // Clear error for this field if it exists
    if (errors[name]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[name];
        return newErrors;
      });
    }
    
    // If provider changes, pre-fill endpoints for known providers
    if (name === 'provider') {
      const selectedProvider = commonProviders.find(p => p.id === value);
      if (selectedProvider) {
        if (selectedProvider.id === 'google') {
          setFormData(prev => ({
            ...prev,
            provider: selectedProvider.name,
            connectionDetails: {
              ...prev.connectionDetails,
              authEndpoint: 'https://accounts.google.com/o/oauth2/auth',
              tokenEndpoint: 'https://oauth2.googleapis.com/token'
            }
          }));
        } else if (selectedProvider.id === 'github') {
          setFormData(prev => ({
            ...prev,
            provider: selectedProvider.name,
            connectionDetails: {
              ...prev.connectionDetails,
              authEndpoint: 'https://github.com/login/oauth/authorize',
              tokenEndpoint: 'https://github.com/login/oauth/access_token'
            }
          }));
        } else if (selectedProvider.id === 'slack') {
          setFormData(prev => ({
            ...prev,
            provider: selectedProvider.name,
            connectionDetails: {
              ...prev.connectionDetails,
              authEndpoint: 'https://slack.com/oauth/v2/authorize',
              tokenEndpoint: 'https://slack.com/api/oauth.v2.access'
            }
          }));
        } else if (selectedProvider.id === 'salesforce') {
          setFormData(prev => ({
            ...prev,
            provider: selectedProvider.name,
            connectionDetails: {
              ...prev.connectionDetails,
              authEndpoint: 'https://login.salesforce.com/services/oauth2/authorize',
              tokenEndpoint: 'https://login.salesforce.com/services/oauth2/token'
            }
          }));
        } else if (selectedProvider.id === 'microsoft') {
          setFormData(prev => ({
            ...prev,
            provider: selectedProvider.name,
            connectionDetails: {
              ...prev.connectionDetails,
              authEndpoint: 'https://login.microsoftonline.com/common/oauth2/v2.0/authorize',
              tokenEndpoint: 'https://login.microsoftonline.com/common/oauth2/v2.0/token'
            }
          }));
        } else if (selectedProvider.id === 'openai') {
          setFormData(prev => ({
            ...prev,
            provider: selectedProvider.name,
            connectionDetails: {
              ...prev.connectionDetails,
              baseUrl: 'https://api.openai.com/v1'
            }
          }));
        } else if (selectedProvider.id === 'aws') {
          setFormData(prev => ({
            ...prev,
            provider: selectedProvider.name,
            connectionDetails: {
              ...prev.connectionDetails,
              region: 'us-east-1'
            }
          }));
        }
      }
    }
    
    // If type changes, reset connection details
    if (name === 'type') {
      setFormData(prev => ({
        ...prev,
        connectionDetails: value === 'oauth' 
          ? {
              clientId: '',
              clientSecret: '',
              redirectUri: 'https://app.picaos.com/authkit/callback',
              tokenEndpoint: '',
              authEndpoint: ''
            }
          : {
              apiKey: '',
              baseUrl: '',
              accessKeyId: '',
              secretAccessKey: '',
              region: '',
              bucket: ''
            }
      }));
    }
  };

  const handleAddScope = () => {
    if (!scopeInput.trim()) return;
    
    // Add scope if it doesn't already exist
    if (!formData.scopes.includes(scopeInput.trim())) {
      setFormData(prev => ({
        ...prev,
        scopes: [...prev.scopes, scopeInput.trim()]
      }));
    }
    
    // Clear input
    setScopeInput('');
  };

  const handleRemoveScope = (scope) => {
    setFormData(prev => ({
      ...prev,
      scopes: prev.scopes.filter(s => s !== scope)
    }));
  };

  const handleToggleShowSecret = (field) => {
    setShowSecrets(prev => ({
      ...prev,
      [field]: !prev[field]
    }));
  };

  const validateForm = () => {
    const newErrors = {};
    
    if (!formData.name.trim()) {
      newErrors.name = 'Connection name is required';
    }
    
    if (!formData.provider.trim()) {
      newErrors.provider = 'Provider is required';
    }
    
    if (!formData.description.trim()) {
      newErrors.description = 'Description is required';
    }
    
    if (formData.type === 'oauth') {
      if (!formData.connectionDetails.clientId.trim()) {
        newErrors['connectionDetails.clientId'] = 'Client ID is required';
      }
      
      if (!formData.connectionDetails.clientSecret.trim()) {
        newErrors['connectionDetails.clientSecret'] = 'Client Secret is required';
      }
      
      if (!formData.connectionDetails.authEndpoint.trim()) {
        newErrors['connectionDetails.authEndpoint'] = 'Authorization Endpoint is required';
      }
      
      if (!formData.connectionDetails.tokenEndpoint.trim()) {
        newErrors['connectionDetails.tokenEndpoint'] = 'Token Endpoint is required';
      }
    } else if (formData.type === 'apikey') {
      if (formData.provider === 'AWS') {
        if (!formData.connectionDetails.accessKeyId.trim()) {
          newErrors['connectionDetails.accessKeyId'] = 'Access Key ID is required';
        }
        
        if (!formData.connectionDetails.secretAccessKey.trim()) {
          newErrors['connectionDetails.secretAccessKey'] = 'Secret Access Key is required';
        }
      } else {
        if (!formData.connectionDetails.apiKey.trim()) {
          newErrors['connectionDetails.apiKey'] = 'API Key is required';
        }
      }
    }
    
    if (formData.scopes.length === 0) {
      newErrors.scopes = 'At least one scope is required';
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
      // In a real implementation, this would be an API call to picaos.com/authkit
      // For now, we'll simulate a successful API call
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      // Prepare the connection data for the API
      const connectionData = {
        ...formData,
        id: connection ? connection.id : Date.now(),
        status: connection ? connection.status : 'inactive',
        lastUsed: connection ? connection.lastUsed : 'Never',
        expiresAt: connection ? connection.expiresAt : null,
        createdAt: connection ? connection.createdAt : new Date().toISOString(),
        createdBy: connection ? connection.createdBy : 'Current User',
        icon: getProviderIcon(formData.provider)
      };
      
      // Simulate API call
      // const response = await fetch('https://app.picaos.com/authkit/api/connections', {
      //   method: 'POST',
      //   headers: {
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify(connectionData),
      // });
      
      // if (!response.ok) {
      //   throw new Error('Failed to save connection');
      // }
      
      // const data = await response.json();
      
      setSubmitStatus('success');
      
      // Notify parent component of successful save
      if (onConnectionSaved) {
        onConnectionSaved(connectionData);
      }
      
      // Close modal after successful submission
      setTimeout(() => {
        setSubmitStatus(null);
        onClose();
      }, 1500);
      
    } catch (error) {
      console.error('Error saving connection:', error);
      setSubmitStatus('error');
      // Handle error with AI Error Engine
      handleApiError(error, {
        operation: 'save_connection',
        component: 'WebConnectionModal',
        connectionType: formData.type,
        provider: formData.provider,
        timestamp: new Date().toISOString(),
        context: 'User attempted to save a web connection configuration'
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const getProviderIcon = (providerName) => {
    const provider = commonProviders.find(p => 
      p.name.toLowerCase() === providerName.toLowerCase() || 
      p.id.toLowerCase() === providerName.toLowerCase()
    );
    
    return provider ? provider.icon : 'https://upload.wikimedia.org/wikipedia/commons/5/5c/Gears_0.png';
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
            <div className="p-2 bg-gradient-to-r from-blue-500/20 to-cyan-500/20 rounded-lg border border-blue-500/30">
              <Globe className="w-5 h-5 text-blue-400" />
            </div>
            <h2 className="text-xl font-bold text-white">
              {connection ? 'Edit Connection' : 'Add Connection'}
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
                  Connection Name
                </label>
                <input
                  type="text"
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Enter connection name (e.g., Google Drive)"
                  className={`w-full bg-slate-700/50 border ${errors.name ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-red-400">{errors.name}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="type" className="block text-sm font-medium text-slate-300 mb-2">
                  Connection Type
                </label>
                <div className="grid grid-cols-2 gap-3">
                  {connectionTypes.map((type) => (
                    <div
                      key={type.id}
                      onClick={() => handleChange({ target: { name: 'type', value: type.id } })}
                      className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                        formData.type === type.id
                          ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 border-blue-500/30'
                          : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                      }`}
                    >
                      <div className="flex flex-col items-center text-center">
                        {type.id === 'oauth' ? (
                          <Lock className="w-6 h-6 text-blue-400" />
                        ) : (
                          <Key className="w-6 h-6 text-blue-400" />
                        )}
                        <span className={`text-sm font-medium mt-2 ${formData.type === type.id ? 'text-white' : 'text-slate-300'}`}>
                          {type.name}
                        </span>
                        <span className="text-xs text-slate-400 mt-1">{type.description}</span>
                      </div>
                    </div>
                  ))}
                </div>
                {errors.type && (
                  <p className="mt-1 text-sm text-red-400">{errors.type}</p>
                )}
              </div>
              
              <div>
                <label htmlFor="provider" className="block text-sm font-medium text-slate-300 mb-2">
                  Provider
                </label>
                <div className="grid grid-cols-3 gap-3 mb-3">
                  {commonProviders
                    .filter(p => p.type === 'both' || p.type === formData.type)
                    .map((provider) => (
                      <div
                        key={provider.id}
                        onClick={() => handleChange({ target: { name: 'provider', value: provider.id } })}
                        className={`p-3 rounded-lg cursor-pointer transition-all duration-200 border ${
                          formData.provider === provider.name
                            ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 border-blue-500/30'
                            : 'bg-slate-700/30 hover:bg-slate-700/50 border-slate-600/30'
                        }`}
                      >
                        <div className="flex flex-col items-center text-center">
                          <img 
                            src={provider.icon} 
                            alt={provider.name} 
                            className="w-8 h-8 object-contain bg-white p-1 rounded"
                          />
                          <span className={`text-sm font-medium mt-2 ${formData.provider === provider.name ? 'text-white' : 'text-slate-300'}`}>
                            {provider.name}
                          </span>
                        </div>
                      </div>
                    ))}
                </div>
                
                <input
                  type="text"
                  id="provider"
                  name="provider"
                  value={formData.provider}
                  onChange={handleChange}
                  placeholder="Enter provider name (e.g., Google, GitHub, Custom)"
                  className={`w-full bg-slate-700/50 border ${errors.provider ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                />
                {errors.provider && (
                  <p className="mt-1 text-sm text-red-400">{errors.provider}</p>
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
                  placeholder="Enter a description of this connection"
                  rows="3"
                  className={`w-full bg-slate-700/50 border ${errors.description ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                ></textarea>
                {errors.description && (
                  <p className="mt-1 text-sm text-red-400">{errors.description}</p>
                )}
              </div>
            </div>
            
            {/* Connection Details */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Connection Details</h3>
              
              {formData.type === 'oauth' ? (
                <>
                  <div>
                    <label htmlFor="clientId" className="block text-sm font-medium text-slate-300 mb-2">
                      Client ID
                    </label>
                    <input
                      type="text"
                      id="clientId"
                      name="connectionDetails.clientId"
                      value={formData.connectionDetails.clientId}
                      onChange={handleChange}
                      placeholder="Enter OAuth client ID"
                      className={`w-full bg-slate-700/50 border ${errors['connectionDetails.clientId'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                    />
                    {errors['connectionDetails.clientId'] && (
                      <p className="mt-1 text-sm text-red-400">{errors['connectionDetails.clientId']}</p>
                    )}
                  </div>
                  
                  <div>
                    <label htmlFor="clientSecret" className="block text-sm font-medium text-slate-300 mb-2">
                      Client Secret
                    </label>
                    <div className="relative">
                      <input
                        type={showSecrets.clientSecret ? "text" : "password"}
                        id="clientSecret"
                        name="connectionDetails.clientSecret"
                        value={formData.connectionDetails.clientSecret}
                        onChange={handleChange}
                        placeholder="Enter OAuth client secret"
                        className={`w-full bg-slate-700/50 border ${errors['connectionDetails.clientSecret'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 pr-10 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                      />
                      <button
                        type="button"
                        onClick={() => handleToggleShowSecret('clientSecret')}
                        className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slate-400 hover:text-white"
                      >
                        {showSecrets.clientSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                    </div>
                    {errors['connectionDetails.clientSecret'] && (
                      <p className="mt-1 text-sm text-red-400">{errors['connectionDetails.clientSecret']}</p>
                    )}
                  </div>
                  
                  <div>
                    <label htmlFor="redirectUri" className="block text-sm font-medium text-slate-300 mb-2">
                      Redirect URI
                    </label>
                    <div className="relative">
                      <input
                        type="text"
                        id="redirectUri"
                        name="connectionDetails.redirectUri"
                        value={formData.connectionDetails.redirectUri}
                        onChange={handleChange}
                        placeholder="Enter OAuth redirect URI"
                        className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 pr-10 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        readOnly
                      />
                      <button
                        type="button"
                        onClick={() => {
                          navigator.clipboard.writeText(formData.connectionDetails.redirectUri);
                          // Show a temporary "Copied" message
                          const button = document.getElementById('copyRedirectButton');
                          if (button) {
                            const originalText = button.innerHTML;
                            button.innerHTML = 'Copied!';
                            setTimeout(() => {
                              button.innerHTML = originalText;
                            }, 2000);
                          }
                        }}
                        id="copyRedirectButton"
                        className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slate-400 hover:text-white"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                        </svg>
                      </button>
                    </div>
                    <p className="mt-1 text-xs text-slate-400">This is the callback URL to configure in your OAuth provider</p>
                  </div>
                  
                  <div>
                    <label htmlFor="authEndpoint" className="block text-sm font-medium text-slate-300 mb-2">
                      Authorization Endpoint
                    </label>
                    <input
                      type="text"
                      id="authEndpoint"
                      name="connectionDetails.authEndpoint"
                      value={formData.connectionDetails.authEndpoint}
                      onChange={handleChange}
                      placeholder="Enter OAuth authorization endpoint URL"
                      className={`w-full bg-slate-700/50 border ${errors['connectionDetails.authEndpoint'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                    />
                    {errors['connectionDetails.authEndpoint'] && (
                      <p className="mt-1 text-sm text-red-400">{errors['connectionDetails.authEndpoint']}</p>
                    )}
                  </div>
                  
                  <div>
                    <label htmlFor="tokenEndpoint" className="block text-sm font-medium text-slate-300 mb-2">
                      Token Endpoint
                    </label>
                    <input
                      type="text"
                      id="tokenEndpoint"
                      name="connectionDetails.tokenEndpoint"
                      value={formData.connectionDetails.tokenEndpoint}
                      onChange={handleChange}
                      placeholder="Enter OAuth token endpoint URL"
                      className={`w-full bg-slate-700/50 border ${errors['connectionDetails.tokenEndpoint'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                    />
                    {errors['connectionDetails.tokenEndpoint'] && (
                      <p className="mt-1 text-sm text-red-400">{errors['connectionDetails.tokenEndpoint']}</p>
                    )}
                  </div>
                </>
              ) : (
                <>
                  {formData.provider === 'AWS' ? (
                    <>
                      <div>
                        <label htmlFor="accessKeyId" className="block text-sm font-medium text-slate-300 mb-2">
                          Access Key ID
                        </label>
                        <input
                          type="text"
                          id="accessKeyId"
                          name="connectionDetails.accessKeyId"
                          value={formData.connectionDetails.accessKeyId}
                          onChange={handleChange}
                          placeholder="Enter AWS Access Key ID"
                          className={`w-full bg-slate-700/50 border ${errors['connectionDetails.accessKeyId'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                        />
                        {errors['connectionDetails.accessKeyId'] && (
                          <p className="mt-1 text-sm text-red-400">{errors['connectionDetails.accessKeyId']}</p>
                        )}
                      </div>
                      
                      <div>
                        <label htmlFor="secretAccessKey" className="block text-sm font-medium text-slate-300 mb-2">
                          Secret Access Key
                        </label>
                        <div className="relative">
                          <input
                            type={showSecrets.secretAccessKey ? "text" : "password"}
                            id="secretAccessKey"
                            name="connectionDetails.secretAccessKey"
                            value={formData.connectionDetails.secretAccessKey}
                            onChange={handleChange}
                            placeholder="Enter AWS Secret Access Key"
                            className={`w-full bg-slate-700/50 border ${errors['connectionDetails.secretAccessKey'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 pr-10 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                          />
                          <button
                            type="button"
                            onClick={() => handleToggleShowSecret('secretAccessKey')}
                            className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slate-400 hover:text-white"
                          >
                            {showSecrets.secretAccessKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                          </button>
                        </div>
                        {errors['connectionDetails.secretAccessKey'] && (
                          <p className="mt-1 text-sm text-red-400">{errors['connectionDetails.secretAccessKey']}</p>
                        )}
                      </div>
                      
                      <div>
                        <label htmlFor="region" className="block text-sm font-medium text-slate-300 mb-2">
                          Region
                        </label>
                        <input
                          type="text"
                          id="region"
                          name="connectionDetails.region"
                          value={formData.connectionDetails.region}
                          onChange={handleChange}
                          placeholder="Enter AWS region (e.g., us-east-1)"
                          className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      </div>
                      
                      <div>
                        <label htmlFor="bucket" className="block text-sm font-medium text-slate-300 mb-2">
                          S3 Bucket (optional)
                        </label>
                        <input
                          type="text"
                          id="bucket"
                          name="connectionDetails.bucket"
                          value={formData.connectionDetails.bucket}
                          onChange={handleChange}
                          placeholder="Enter S3 bucket name (if applicable)"
                          className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      </div>
                    </>
                  ) : (
                    <>
                      <div>
                        <label htmlFor="apiKey" className="block text-sm font-medium text-slate-300 mb-2">
                          API Key
                        </label>
                        <div className="relative">
                          <input
                            type={showSecrets.apiKey ? "text" : "password"}
                            id="apiKey"
                            name="connectionDetails.apiKey"
                            value={formData.connectionDetails.apiKey}
                            onChange={handleChange}
                            placeholder="Enter API key"
                            className={`w-full bg-slate-700/50 border ${errors['connectionDetails.apiKey'] ? 'border-red-500/50' : 'border-slate-600/50'} rounded-lg px-3 py-2 pr-10 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500`}
                          />
                          <button
                            type="button"
                            onClick={() => handleToggleShowSecret('apiKey')}
                            className="absolute right-3 top-1/2 transform -translate-y-1/2 text-slate-400 hover:text-white"
                          >
                            {showSecrets.apiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                          </button>
                        </div>
                        {errors['connectionDetails.apiKey'] && (
                          <p className="mt-1 text-sm text-red-400">{errors['connectionDetails.apiKey']}</p>
                        )}
                      </div>
                      
                      <div>
                        <label htmlFor="baseUrl" className="block text-sm font-medium text-slate-300 mb-2">
                          Base URL (optional)
                        </label>
                        <input
                          type="text"
                          id="baseUrl"
                          name="connectionDetails.baseUrl"
                          value={formData.connectionDetails.baseUrl}
                          onChange={handleChange}
                          placeholder="Enter API base URL (e.g., https://api.example.com/v1)"
                          className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                      </div>
                    </>
                  )}
                </>
              )}
            </div>
            
            {/* Scopes */}
            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-white">Scopes</h3>
              
              <div>
                <label htmlFor="scopes" className="block text-sm font-medium text-slate-300 mb-2">
                  Add Scopes
                </label>
                <div className="flex space-x-2">
                  <input
                    type="text"
                    id="scopes"
                    value={scopeInput}
                    onChange={(e) => setScopeInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        handleAddScope();
                      }
                    }}
                    placeholder="Enter scope (e.g., read:user, write:files)"
                    className="flex-1 bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <button
                    type="button"
                    onClick={handleAddScope}
                    className="px-4 py-2 bg-blue-500/20 text-blue-400 rounded-lg hover:bg-blue-500/30 transition-colors duration-200 flex items-center space-x-2"
                  >
                    <Plus className="w-4 h-4" />
                    <span>Add</span>
                  </button>
                </div>
                {errors.scopes && (
                  <p className="mt-1 text-sm text-red-400">{errors.scopes}</p>
                )}
              </div>
              
              <div>
                <div className="flex flex-wrap gap-2 mt-2">
                  {formData.scopes.map((scope, index) => (
                    <div 
                      key={index}
                      className="px-3 py-1 bg-slate-700/50 text-slate-300 text-sm rounded-full border border-slate-600/50 flex items-center space-x-2"
                    >
                      <span>{scope}</span>
                      <button
                        type="button"
                        onClick={() => handleRemoveScope(scope)}
                        className="text-slate-400 hover:text-red-400"
                      >
                        <X className="w-3 h-3" />
                      </button>
                    </div>
                  ))}
                  
                  {formData.scopes.length === 0 && (
                    <p className="text-slate-500 text-sm">No scopes added yet</p>
                  )}
                </div>
                
                {formData.type === 'oauth' && formData.provider && (
                  <div className="mt-3">
                    <p className="text-sm text-slate-400">Common scopes for {formData.provider}:</p>
                    <div className="flex flex-wrap gap-2 mt-1">
                      {formData.provider === 'Google' && (
                        <>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('https://www.googleapis.com/auth/drive.readonly');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            drive.readonly
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('https://www.googleapis.com/auth/drive.file');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            drive.file
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('https://www.googleapis.com/auth/gmail.readonly');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            gmail.readonly
                          </button>
                        </>
                      )}
                      
                      {formData.provider === 'GitHub' && (
                        <>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('repo');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            repo
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('user');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            user
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('read:org');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            read:org
                          </button>
                        </>
                      )}
                      
                      {formData.provider === 'Slack' && (
                        <>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('channels:read');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            channels:read
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('chat:write');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            chat:write
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setScopeInput('users:read');
                              handleAddScope();
                            }}
                            className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                          >
                            users:read
                          </button>
                        </>
                      )}
                      
                      {formData.type === 'apikey' && (
                        <button
                          type="button"
                          onClick={() => {
                            setScopeInput('*');
                            handleAddScope();
                          }}
                          className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
                        >
                          all (*) 
                        </button>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
            
            {/* AuthKit Information */}
            <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
              <div className="flex items-start space-x-3">
                <FileJson className="w-5 h-5 text-blue-400 mt-0.5" />
                <div>
                  <h4 className="text-white font-medium">Picaos AuthKit Integration</h4>
                  <p className="text-slate-300 text-sm mt-1">This connection will be managed by Picaos AuthKit for secure credential storage and token management.</p>
                  <div className="mt-2 text-xs text-slate-400">
                    <p>• Credentials are securely stored and encrypted</p>
                    <p>• OAuth tokens are automatically refreshed</p>
                    <p>• Connections can be managed at <a href="https://app.picaos.com/authkit" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:underline flex items-center space-x-1 inline-flex"><span>app.picaos.com/authkit</span> <ExternalLink className="w-3 h-3" /></a></p>
                  </div>
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
                  <span>Connection saved!</span>
                </div>
              )}
              
              {submitStatus === 'error' && (
                <div className="flex items-center text-red-400 space-x-1">
                  <AlertCircle className="w-4 h-4" />
                  <span>Failed to save connection</span>
                </div>
              )}
              
              <button
                onClick={handleSubmit}
                disabled={isSubmitting}
                className="px-6 py-2 bg-gradient-to-r from-blue-500 to-cyan-500 text-white rounded-lg font-medium hover:from-blue-600 hover:to-cyan-600 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
              >
                {isSubmitting ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>Saving...</span>
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4" />
                    <span>Save Connection</span>
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

export default WebConnectionModal;