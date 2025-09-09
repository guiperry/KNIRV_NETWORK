import React, { useState, useEffect } from 'react';
import { Server, Globe, Key, Copy, Eye, EyeOff, Plus, Trash2, Edit, Activity, Link, Settings } from 'lucide-react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { GlassCard } from './common';

interface APIEndpoint {
  id: string;
  name: string;
  url: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  description: string;
  isActive: boolean;
  createdDate: Date;
  lastUsed: Date;
  requestCount: number;
  apiKey?: string;
  tunnelId?: string;
}

// Sub-component for Personal API Endpoints
const PersonalAPIEndpoints: React.FC = () => {
  const [endpoints, setEndpoints] = useState<APIEndpoint[]>([]);
  const [showApiKey, setShowApiKey] = useState<{ [key: string]: boolean }>({});

  useEffect(() => {
    // Load personal API endpoints from TunnelRegistry
    const loadEndpoints = async () => {
      try {
        // TODO: Replace with actual TunnelRegistry API call
        // const response = await fetch('/api/tunnel-registry/user/endpoints');
        // const endpointsData = await response.json();

        // Mock personal API endpoints data
        const mockEndpoints: APIEndpoint[] = [
          {
            id: 'api-1',
            name: 'My Agent Status API',
            url: 'https://tunnel.knirv.network/user123/agents/status',
            method: 'GET',
            description: 'Personal API endpoint for monitoring my agent status',
            isActive: true,
            createdDate: new Date('2024-01-15'),
            lastUsed: new Date(),
            requestCount: 1247,
            apiKey: 'knirv_sk_1234567890abcdef',
            tunnelId: 'tunnel-1'
          },
          {
            id: 'api-2',
            name: 'My Model Training API',
            url: 'https://tunnel.knirv.network/user123/models/train',
            method: 'POST',
            description: 'Personal API endpoint for submitting model training requests',
            isActive: true,
            createdDate: new Date('2024-02-01'),
            lastUsed: new Date(Date.now() - 3600000),
            requestCount: 89,
            apiKey: 'knirv_sk_abcdef1234567890',
            tunnelId: 'tunnel-2'
          },
          {
            id: 'api-3',
            name: 'My Analytics API',
            url: 'https://tunnel.knirv.network/user123/analytics',
            method: 'GET',
            description: 'Personal API endpoint for accessing my usage analytics',
            isActive: false,
            createdDate: new Date('2024-01-20'),
            lastUsed: new Date(Date.now() - 86400000),
            requestCount: 456,
            apiKey: 'knirv_sk_fedcba0987654321',
            tunnelId: 'tunnel-3'
          }
        ];

        setEndpoints(mockEndpoints);
      } catch (error) {
        console.error('Failed to load personal API endpoints:', error);
      }
    };

    loadEndpoints();
  }, []);
  const toggleApiKeyVisibility = (endpointId: string) => {
    setShowApiKey(prev => ({
      ...prev,
      [endpointId]: !prev[endpointId]
    }));
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    // TODO: Show toast notification
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center space-x-3">
          <div className="p-2 bg-blue-500/20 rounded-lg">
            <Server className="w-6 h-6 text-blue-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Personal API Endpoints</h1>
            <p className="text-slate-400">Your personal API endpoints from TunnelRegistry</p>
          </div>
        </div>

        <button className="flex items-center space-x-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors">
          <Plus className="w-4 h-4" />
          <span>Create Endpoint</span>
        </button>
      </div>

      {/* API Endpoints Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {endpoints.map((endpoint) => (
          <div key={endpoint.id} className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
            <div className="flex items-start justify-between mb-4">
              <div className="flex-1">
                <div className="flex items-center space-x-2 mb-2">
                  <h3 className="text-lg font-semibold text-white">{endpoint.name}</h3>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${
                    endpoint.isActive ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                  }`}>
                    {endpoint.isActive ? 'Active' : 'Inactive'}
                  </span>
                </div>
                <p className="text-slate-400 text-sm mb-3">{endpoint.description}</p>

                <div className="space-y-2 text-sm">
                  <div className="flex items-center space-x-2">
                    <span className="text-slate-500">Method:</span>
                    <span className={`px-2 py-1 rounded text-xs font-medium ${
                      endpoint.method === 'GET' ? 'bg-blue-500/20 text-blue-400' :
                      endpoint.method === 'POST' ? 'bg-green-500/20 text-green-400' :
                      endpoint.method === 'PUT' ? 'bg-yellow-500/20 text-yellow-400' :
                      'bg-red-500/20 text-red-400'
                    }`}>
                      {endpoint.method}
                    </span>
                  </div>

                  <div className="flex items-center space-x-2">
                    <span className="text-slate-500">URL:</span>
                    <code className="bg-slate-700/50 px-2 py-1 rounded text-slate-300 text-xs flex-1 truncate">
                      {endpoint.url}
                    </code>
                    <button
                      onClick={() => copyToClipboard(endpoint.url)}
                      className="text-slate-400 hover:text-white transition-colors"
                    >
                      <Copy className="w-4 h-4" />
                    </button>
                  </div>

                  {endpoint.apiKey && (
                    <div className="flex items-center space-x-2">
                      <span className="text-slate-500">API Key:</span>
                      <code className="bg-slate-700/50 px-2 py-1 rounded text-slate-300 text-xs flex-1">
                        {showApiKey[endpoint.id] ? endpoint.apiKey : '••••••••••••••••'}
                      </code>
                      <button
                        onClick={() => toggleApiKeyVisibility(endpoint.id)}
                        className="text-slate-400 hover:text-white transition-colors"
                      >
                        {showApiKey[endpoint.id] ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                      <button
                        onClick={() => copyToClipboard(endpoint.apiKey!)}
                        className="text-slate-400 hover:text-white transition-colors"
                      >
                        <Copy className="w-4 h-4" />
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>

            <div className="flex items-center justify-between text-sm">
              <div className="text-slate-500">
                <div>Requests: {endpoint.requestCount.toLocaleString()}</div>
                <div>Last used: {endpoint.lastUsed.toLocaleDateString()}</div>
              </div>
              <div className="flex space-x-2">
                <button className="bg-slate-600/50 text-slate-300 px-3 py-1 rounded text-sm hover:bg-slate-600 transition-colors">
                  <Edit className="w-4 h-4" />
                </button>
                <button className="bg-red-600/20 text-red-400 px-3 py-1 rounded text-sm hover:bg-red-600/30 transition-colors">
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Stats Overview */}
      <div className="mt-8 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">API Usage Overview</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">{endpoints.length}</div>
            <div className="text-sm text-slate-400">Total Endpoints</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">{endpoints.filter(e => e.isActive).length}</div>
            <div className="text-sm text-slate-400">Active</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">{endpoints.reduce((sum, e) => sum + e.requestCount, 0).toLocaleString()}</div>
            <div className="text-sm text-slate-400">Total Requests</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-400">99.9%</div>
            <div className="text-sm text-slate-400">Uptime</div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Main API component with nested navigation
export const API: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { canAccessSubPage } = useAuth();

  // Check if we're on a sub-route
  const isSubRoute = location.pathname !== '/api';

  // If we're on a sub-route, render the sub-component
  if (isSubRoute) {
    return (
      <Routes>
        <Route path="/personal-endpoints" element={<PersonalAPIEndpoints />} />
      </Routes>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-blue-500/20 rounded-lg">
          <Server className="w-6 h-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">API</h1>
          <p className="text-slate-400">Personal API endpoints and TunnelRegistry management</p>
        </div>
      </div>

      {/* API Options */}
      <div className="grid grid-cols-1 md:grid-cols-1 gap-6">
        {canAccessSubPage('api', 'personal-endpoints') && (
          <button
            onClick={() => navigate('/api/personal-endpoints')}
            className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6 hover:bg-slate-700/50 transition-colors text-left group"
          >
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                <Server className="w-6 h-6 text-blue-400" />
              </div>
              <h3 className="text-lg font-semibold text-white">Personal API Endpoints</h3>
            </div>
            <p className="text-slate-400 mb-4">
              Manage your personal API endpoints configured through the TunnelRegistry. Create, monitor, and control access to your custom API endpoints with authentication and usage analytics.
            </p>
            <div className="flex items-center space-x-4 text-sm">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span className="text-slate-300">3 endpoints</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span className="text-slate-300">2 active</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
                <span className="text-slate-300">1.8K requests</span>
              </div>
            </div>
          </button>
        )}
      </div>

      {/* Quick Overview */}
      <GlassCard title="API Overview" className="mt-8">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-400">3</div>
            <div className="text-sm text-slate-400">Personal Endpoints</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-400">2</div>
            <div className="text-sm text-slate-400">Active</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-400">1.8K</div>
            <div className="text-sm text-slate-400">Total Requests</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-400">99.9%</div>
            <div className="text-sm text-slate-400">Uptime</div>
          </div>
        </div>
      </GlassCard>

      {/* TunnelRegistry Info */}
      <GlassCard title="About TunnelRegistry" className="mt-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="glass-card-dark rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Link className="w-5 h-5 text-blue-400" />
              <h3 className="text-white font-medium">Secure Tunneling</h3>
            </div>
            <p className="text-slate-400 text-sm">
              TunnelRegistry provides secure, encrypted tunnels for your personal API endpoints, allowing safe access to your local services from anywhere.
            </p>
          </div>

          <div className="glass-card-dark rounded-lg p-4">
            <div className="flex items-center space-x-2 mb-3">
              <Key className="w-5 h-5 text-green-400" />
              <h3 className="text-white font-medium">API Key Management</h3>
            </div>
            <p className="text-slate-400 text-sm">
              Automatically generated API keys with configurable permissions and usage limits for each of your personal endpoints.
            </p>
          </div>
        </div>
      </GlassCard>

      {/* Recent Activity */}
      <div className="mt-6 bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Recent API Activity</h2>
        <div className="space-y-3">
          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">My Agent Status API called 15 times</div>
              <div className="text-xs text-slate-400">2 minutes ago</div>
            </div>
            <div className="text-xs text-green-400">Success</div>
          </div>

          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">New API key generated for Model Training API</div>
              <div className="text-xs text-slate-400">1 hour ago</div>
            </div>
            <div className="text-xs text-blue-400">Created</div>
          </div>

          <div className="flex items-center space-x-3 p-3 bg-slate-700/20 rounded-lg">
            <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
            <div className="flex-1">
              <div className="text-sm text-white">Analytics API endpoint disabled</div>
              <div className="text-xs text-slate-400">3 hours ago</div>
            </div>
            <div className="text-xs text-yellow-400">Disabled</div>
          </div>
        </div>
      </div>
    </div>
  );
};


