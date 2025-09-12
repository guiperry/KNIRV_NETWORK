import * as React from 'react';
import { useState, useEffect } from 'react';
import { Key, Plus, Trash2, Eye, EyeOff, Copy, Calendar, Shield, Activity } from 'lucide-react';
import { apiKeyService, ApiKey, CreateApiKeyRequest } from '../services/ApiKeyService';

interface ApiKeyManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

export const ApiKeyManager: React.FC<ApiKeyManagerProps> = ({ isOpen, onClose }) => {
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newKeyData, setNewKeyData] = useState<CreateApiKeyRequest>({
    name: '',
    description: '',
    permissions: [],
    rateLimit: {
      requestsPerMinute: 60,
      requestsPerHour: 1000,
      requestsPerDay: 10000
    }
  });
  const [availablePermissions, setAvailablePermissions] = useState<string[]>([]);
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [showKey, setShowKey] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (isOpen) {
      loadApiKeys();
      loadPermissions();
    }
  }, [isOpen]);

  const loadApiKeys = async () => {
    try {
      const keys = await apiKeyService.getApiKeys();
      setApiKeys(keys);
    } catch (error) {
      console.error('Failed to load API keys:', error);
    }
  };

  const loadPermissions = () => {
    const permissions = apiKeyService.getAvailablePermissions();
    setAvailablePermissions(permissions);
  };

  const handleCreateKey = async () => {
    try {
      if (!newKeyData.name || !newKeyData.description || newKeyData.permissions.length === 0) {
        alert('Please fill in all required fields');
        return;
      }

      const apiKey = await apiKeyService.createApiKey(newKeyData);
      setCreatedKey(apiKey.key);
      setShowCreateForm(false);
      setNewKeyData({
        name: '',
        description: '',
        permissions: [],
        rateLimit: {
          requestsPerMinute: 60,
          requestsPerHour: 1000,
          requestsPerDay: 10000
        }
      });
      await loadApiKeys();
    } catch (error) {
      console.error('Failed to create API key:', error);
      alert('Failed to create API key');
    }
  };

  const handleDeleteKey = async (keyId: string) => {
    if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
      return;
    }

    try {
      await apiKeyService.deleteApiKey(keyId);
      await loadApiKeys();
    } catch (error) {
      console.error('Failed to delete API key:', error);
      alert('Failed to delete API key');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    // You could add a toast notification here
  };

  const toggleKeyVisibility = (keyId: string) => {
    setShowKey(prev => ({ ...prev, [keyId]: !prev[keyId] }));
  };

  const formatDate = (date: Date) => {
    return new Date(date).toLocaleDateString() + ' ' + new Date(date).toLocaleTimeString();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-gray-900 rounded-lg border border-gray-700 w-full max-w-6xl h-5/6 flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-gradient-to-r from-blue-500 to-purple-500 rounded-lg flex items-center justify-center">
              <Key className="w-6 h-6 text-white" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">API Key Manager</h2>
              <p className="text-sm text-gray-400">Manage programmatic access to KNIRVCONTROLLER</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 p-6 overflow-auto">
          {/* Created Key Display */}
          {createdKey && (
            <div className="mb-6 p-4 bg-green-900/20 border border-green-500/30 rounded-lg">
              <h3 className="text-lg font-medium text-green-400 mb-2">API Key Created Successfully!</h3>
              <p className="text-sm text-gray-300 mb-3">
                Please copy this key now. You won't be able to see it again for security reasons.
              </p>
              <div className="flex items-center space-x-2 p-3 bg-gray-800 rounded border">
                <code className="flex-1 text-green-400 font-mono text-sm">{createdKey}</code>
                <button
                  onClick={() => copyToClipboard(createdKey)}
                  className="p-2 text-gray-400 hover:text-white transition-colors"
                  title="Copy to clipboard"
                >
                  <Copy className="w-4 h-4" />
                </button>
              </div>
              <button
                onClick={() => setCreatedKey(null)}
                className="mt-3 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded transition-colors"
              >
                I've copied the key
              </button>
            </div>
          )}

          {/* Create New Key Button */}
          <div className="mb-6">
            <button
              onClick={() => setShowCreateForm(true)}
              className="flex items-center space-x-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors"
            >
              <Plus className="w-4 h-4" />
              <span>Create New API Key</span>
            </button>
          </div>

          {/* Create Form */}
          {showCreateForm && (
            <div className="mb-6 p-6 bg-gray-800 rounded-lg border border-gray-700">
              <h3 className="text-lg font-medium text-white mb-4">Create New API Key</h3>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">Name *</label>
                  <input
                    type="text"
                    value={newKeyData.name}
                    onChange={(e) => setNewKeyData(prev => ({ ...prev, name: e.target.value }))}
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white"
                    placeholder="My API Key"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">Description *</label>
                  <input
                    type="text"
                    value={newKeyData.description}
                    onChange={(e) => setNewKeyData(prev => ({ ...prev, description: e.target.value }))}
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white"
                    placeholder="Used for automated agent management"
                  />
                </div>
              </div>

              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-300 mb-2">Permissions *</label>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                  {availablePermissions.map(permission => (
                    <label key={permission} className="flex items-center space-x-2 text-sm">
                      <input
                        type="checkbox"
                        checked={newKeyData.permissions.includes(permission)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setNewKeyData(prev => ({ 
                              ...prev, 
                              permissions: [...prev.permissions, permission] 
                            }));
                          } else {
                            setNewKeyData(prev => ({ 
                              ...prev, 
                              permissions: prev.permissions.filter(p => p !== permission) 
                            }));
                          }
                        }}
                        className="rounded"
                      />
                      <span className="text-gray-300">{permission}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4 mb-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">Requests/Minute</label>
                  <input
                    type="number"
                    value={newKeyData.rateLimit?.requestsPerMinute || 60}
                    onChange={(e) => setNewKeyData(prev => ({ 
                      ...prev, 
                      rateLimit: { ...prev.rateLimit!, requestsPerMinute: parseInt(e.target.value) }
                    }))}
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">Requests/Hour</label>
                  <input
                    type="number"
                    value={newKeyData.rateLimit?.requestsPerHour || 1000}
                    onChange={(e) => setNewKeyData(prev => ({ 
                      ...prev, 
                      rateLimit: { ...prev.rateLimit!, requestsPerHour: parseInt(e.target.value) }
                    }))}
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">Requests/Day</label>
                  <input
                    type="number"
                    value={newKeyData.rateLimit?.requestsPerDay || 10000}
                    onChange={(e) => setNewKeyData(prev => ({ 
                      ...prev, 
                      rateLimit: { ...prev.rateLimit!, requestsPerDay: parseInt(e.target.value) }
                    }))}
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white"
                  />
                </div>
              </div>

              <div className="flex space-x-3">
                <button
                  onClick={handleCreateKey}
                  className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded transition-colors"
                >
                  Create API Key
                </button>
                <button
                  onClick={() => setShowCreateForm(false)}
                  className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded transition-colors"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}

          {/* API Keys List */}
          <div className="space-y-4">
            <h3 className="text-lg font-medium text-white">Existing API Keys</h3>
            {apiKeys.length === 0 ? (
              <div className="text-center py-8 text-gray-400">
                <Key className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p>No API keys created yet</p>
                <p className="text-sm">Create your first API key to get started</p>
              </div>
            ) : (
              apiKeys.map(key => (
                <div key={key.id} className="p-4 bg-gray-800 rounded-lg border border-gray-700">
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <h4 className="font-medium text-white">{key.name}</h4>
                      <p className="text-sm text-gray-400">{key.description}</p>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className={`px-2 py-1 rounded text-xs ${
                        key.isActive ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                      }`}>
                        {key.isActive ? 'Active' : 'Inactive'}
                      </span>
                      <button
                        onClick={() => handleDeleteKey(key.id)}
                        className="p-1 text-gray-400 hover:text-red-400 transition-colors"
                        title="Delete API key"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                    <div>
                      <div className="flex items-center space-x-2 text-gray-400 mb-1">
                        <Calendar className="w-4 h-4" />
                        <span>Created</span>
                      </div>
                      <p className="text-white">{formatDate(key.createdAt)}</p>
                    </div>
                    <div>
                      <div className="flex items-center space-x-2 text-gray-400 mb-1">
                        <Activity className="w-4 h-4" />
                        <span>Usage</span>
                      </div>
                      <p className="text-white">{key.usageCount} requests</p>
                    </div>
                    <div>
                      <div className="flex items-center space-x-2 text-gray-400 mb-1">
                        <Shield className="w-4 h-4" />
                        <span>Permissions</span>
                      </div>
                      <p className="text-white">{key.permissions.length} permissions</p>
                    </div>
                  </div>

                  {/* API Key Display with Visibility Toggle */}
                  <div className="mt-3 p-3 bg-gray-700 rounded border">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm text-gray-400">API Key</span>
                      <div className="flex items-center space-x-2">
                        <button
                          onClick={() => toggleKeyVisibility(key.id)}
                          className="p-1 text-gray-400 hover:text-white transition-colors"
                          title={showKey[key.id] ? "Hide key" : "Show key"}
                        >
                          {showKey[key.id] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </button>
                        <button
                          onClick={() => copyToClipboard(key.key)}
                          className="p-1 text-gray-400 hover:text-white transition-colors"
                          title="Copy to clipboard"
                        >
                          <Copy className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                    <code className="text-sm font-mono text-green-400">
                      {showKey[key.id] ? key.key : '••••••••••••••••••••••••••••••••'}
                    </code>
                  </div>

                  <div className="mt-3 flex flex-wrap gap-1">
                    {key.permissions.map(permission => (
                      <span
                        key={permission}
                        className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs"
                      >
                        {permission}
                      </span>
                    ))}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
