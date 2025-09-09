import React, { useState, useEffect } from 'react';
import { Settings, Globe, Key, Shield, CheckCircle, AlertTriangle, Save } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface APIProvider {
  id: string;
  name: string;
  type: 'openai' | 'anthropic' | 'google' | 'custom';
  status: 'active' | 'inactive' | 'error';
  endpoint: string;
  apiKey: string;
  priority: number;
  rateLimit: number;
  lastUsed: Date | null;
}

interface HOMConfig {
  enabled: boolean;
  threshold: number;
  fallbackDelay: number;
  retryAttempts: number;
  healthCheckInterval: number;
}

const FallbackConfig: React.FC = () => {
  const { user } = useAuth();
  const [providers, setProviders] = useState<APIProvider[]>([]);
  const [homConfig, setHomConfig] = useState<HOMConfig>({
    enabled: true,
    threshold: 85,
    fallbackDelay: 2000,
    retryAttempts: 3,
    healthCheckInterval: 30000
  });
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    // Load existing configuration
    const mockProviders: APIProvider[] = [
      {
        id: 'openai-1',
        name: 'OpenAI GPT-4',
        type: 'openai',
        status: 'active',
        endpoint: 'https://api.openai.com/v1',
        apiKey: 'sk-*********************',
        priority: 1,
        rateLimit: 10000,
        lastUsed: new Date(Date.now() - 3600000)
      },
      {
        id: 'anthropic-1',
        name: 'Anthropic Claude',
        type: 'anthropic',
        status: 'active',
        endpoint: 'https://api.anthropic.com/v1',
        apiKey: 'sk-ant-*********************',
        priority: 2,
        rateLimit: 5000,
        lastUsed: new Date(Date.now() - 7200000)
      },
      {
        id: 'google-1',
        name: 'Google Gemini',
        type: 'google',
        status: 'inactive',
        endpoint: 'https://generativelanguage.googleapis.com/v1',
        apiKey: 'AIza*********************',
        priority: 3,
        rateLimit: 8000,
        lastUsed: null
      }
    ];

    setProviders(mockProviders);
  }, []);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'error':
        return <AlertTriangle className="w-4 h-4 text-red-500" />;
      default:
        return <div className="w-4 h-4 bg-slate-500 rounded-full" />;
    }
  };

  const getProviderIcon = (type: string) => {
    switch (type) {
      case 'openai':
        return '🤖';
      case 'anthropic':
        return '🧠';
      case 'google':
        return '🔍';
      default:
        return '⚙️';
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    // Simulate save operation
    await new Promise(resolve => setTimeout(resolve, 1500));
    setIsSaving(false);
    setIsEditing(false);
  };

  const updateProvider = (id: string, updates: Partial<APIProvider>) => {
    setProviders(prev => prev.map(p => p.id === id ? { ...p, ...updates } : p));
  };

  const addProvider = () => {
    const newProvider: APIProvider = {
      id: `provider-${Date.now()}`,
      name: 'New Provider',
      type: 'custom',
      status: 'inactive',
      endpoint: '',
      apiKey: '',
      priority: providers.length + 1,
      rateLimit: 1000,
      lastUsed: null
    };
    setProviders(prev => [...prev, newProvider]);
    setIsEditing(true);
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-orange-500/20 rounded-lg">
            <Settings className="w-6 h-6 text-orange-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Fallback API & HOM Config</h1>
            <p className="text-slate-400">Configure API fallback providers and Health-Oriented Monitoring</p>
          </div>
        </div>
        
        <div className="flex space-x-2">
          <button
            onClick={() => setIsEditing(!isEditing)}
            className="flex items-center space-x-2 bg-slate-600 text-white px-4 py-2 rounded-lg hover:bg-slate-500 transition-colors"
          >
            <Settings className="w-4 h-4" />
            <span>{isEditing ? 'Cancel' : 'Edit'}</span>
          </button>
          
          {isEditing && (
            <button
              onClick={handleSave}
              disabled={isSaving}
              className="flex items-center space-x-2 bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 disabled:opacity-50 transition-colors"
            >
              {isSaving ? (
                <>
                  <div className="animate-spin w-4 h-4 border-2 border-white border-t-transparent rounded-full"></div>
                  <span>Saving...</span>
                </>
              ) : (
                <>
                  <Save className="w-4 h-4" />
                  <span>Save</span>
                </>
              )}
            </button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* API Providers */}
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-white">API Providers</h2>
            {isEditing && (
              <button
                onClick={addProvider}
                className="text-sm bg-blue-600 text-white px-3 py-1 rounded hover:bg-blue-700 transition-colors"
              >
                Add Provider
              </button>
            )}
          </div>
          
          <div className="space-y-4">
            {providers.map((provider) => (
              <div key={provider.id} className="bg-slate-700/30 rounded-lg p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center space-x-3">
                    <span className="text-2xl">{getProviderIcon(provider.type)}</span>
                    <div>
                      {isEditing ? (
                        <input
                          type="text"
                          value={provider.name}
                          onChange={(e) => updateProvider(provider.id, { name: e.target.value })}
                          className="bg-slate-600 text-white px-2 py-1 rounded text-sm"
                        />
                      ) : (
                        <div className="text-white font-medium">{provider.name}</div>
                      )}
                      <div className="text-xs text-slate-400">Priority: {provider.priority}</div>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    {getStatusIcon(provider.status)}
                    <span className="text-sm text-white capitalize">{provider.status}</span>
                  </div>
                </div>
                
                {isEditing && (
                  <div className="space-y-2">
                    <div>
                      <label className="text-xs text-slate-400">Endpoint</label>
                      <input
                        type="text"
                        value={provider.endpoint}
                        onChange={(e) => updateProvider(provider.id, { endpoint: e.target.value })}
                        className="w-full bg-slate-600 text-white px-2 py-1 rounded text-sm"
                        placeholder="https://api.example.com/v1"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-slate-400">API Key</label>
                      <input
                        type="password"
                        value={provider.apiKey}
                        onChange={(e) => updateProvider(provider.id, { apiKey: e.target.value })}
                        className="w-full bg-slate-600 text-white px-2 py-1 rounded text-sm"
                        placeholder="Enter API key"
                      />
                    </div>
                    <div className="grid grid-cols-2 gap-2">
                      <div>
                        <label className="text-xs text-slate-400">Priority</label>
                        <input
                          type="number"
                          value={provider.priority}
                          onChange={(e) => updateProvider(provider.id, { priority: parseInt(e.target.value) })}
                          className="w-full bg-slate-600 text-white px-2 py-1 rounded text-sm"
                          min="1"
                        />
                      </div>
                      <div>
                        <label className="text-xs text-slate-400">Rate Limit</label>
                        <input
                          type="number"
                          value={provider.rateLimit}
                          onChange={(e) => updateProvider(provider.id, { rateLimit: parseInt(e.target.value) })}
                          className="w-full bg-slate-600 text-white px-2 py-1 rounded text-sm"
                        />
                      </div>
                    </div>
                  </div>
                )}
                
                {!isEditing && (
                  <div className="text-xs text-slate-400">
                    Rate Limit: {provider.rateLimit.toLocaleString()}/hour
                    {provider.lastUsed && (
                      <span className="ml-4">Last used: {provider.lastUsed.toLocaleTimeString()}</span>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* HOM Configuration */}
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-white mb-4">Health-Oriented Monitoring</h2>
          
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-slate-300">Enable HOM</span>
              <button
                onClick={() => setHomConfig(prev => ({ ...prev, enabled: !prev.enabled }))}
                disabled={!isEditing}
                className={`w-12 h-6 rounded-full transition-colors ${
                  homConfig.enabled ? 'bg-green-600' : 'bg-slate-600'
                } ${!isEditing ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                <div className={`w-5 h-5 bg-white rounded-full transition-transform ${
                  homConfig.enabled ? 'translate-x-6' : 'translate-x-0.5'
                }`}></div>
              </button>
            </div>
            
            <div>
              <label className="text-sm text-slate-300 block mb-2">
                Health Threshold ({homConfig.threshold}%)
              </label>
              <input
                type="range"
                min="50"
                max="100"
                value={homConfig.threshold}
                onChange={(e) => setHomConfig(prev => ({ ...prev, threshold: parseInt(e.target.value) }))}
                disabled={!isEditing}
                className="w-full"
              />
            </div>
            
            <div>
              <label className="text-sm text-slate-300 block mb-2">Fallback Delay (ms)</label>
              <input
                type="number"
                value={homConfig.fallbackDelay}
                onChange={(e) => setHomConfig(prev => ({ ...prev, fallbackDelay: parseInt(e.target.value) }))}
                disabled={!isEditing}
                className="w-full bg-slate-600 text-white px-3 py-2 rounded disabled:opacity-50"
              />
            </div>
            
            <div>
              <label className="text-sm text-slate-300 block mb-2">Retry Attempts</label>
              <input
                type="number"
                value={homConfig.retryAttempts}
                onChange={(e) => setHomConfig(prev => ({ ...prev, retryAttempts: parseInt(e.target.value) }))}
                disabled={!isEditing}
                className="w-full bg-slate-600 text-white px-3 py-2 rounded disabled:opacity-50"
                min="1"
                max="10"
              />
            </div>
            
            <div>
              <label className="text-sm text-slate-300 block mb-2">Health Check Interval (ms)</label>
              <input
                type="number"
                value={homConfig.healthCheckInterval}
                onChange={(e) => setHomConfig(prev => ({ ...prev, healthCheckInterval: parseInt(e.target.value) }))}
                disabled={!isEditing}
                className="w-full bg-slate-600 text-white px-3 py-2 rounded disabled:opacity-50"
                min="5000"
                step="5000"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FallbackConfig;
