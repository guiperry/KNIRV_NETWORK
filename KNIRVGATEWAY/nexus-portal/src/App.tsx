import React, { useState, useEffect } from 'react';
import { Shield, Server, Activity, Eye, User, Settings, BarChart3, Wifi, WifiOff, Radio } from 'lucide-react';
import { useNexusSystem, useNexusDVE, useNexusValidation } from './hooks/use-realtime';

// Types
interface AuthUser {
  user: string;
  role: 'admin' | 'validator' | 'observer';
  permissions: string[];
  nexus_access: string[];
  node_id?: string;
  authenticated: boolean;
}

interface SystemStatus {
  dve_manager: {
    status: string;
    service: string;
  };
  validation_core: {
    status: string;
    service: string;
  };
  timestamp: number;
}

function App() {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');
  const [token, setToken] = useState('');
  const [loginError, setLoginError] = useState('');

  // Real-time connections
  const systemRealtime = useNexusSystem(localStorage.getItem('knirv_nexus_token') || undefined);
  const dveRealtime = useNexusDVE(localStorage.getItem('knirv_nexus_token') || undefined);
  const validationRealtime = useNexusValidation(localStorage.getItem('knirv_nexus_token') || undefined);

  // Check for existing authentication
  useEffect(() => {
    const savedToken = localStorage.getItem('knirv_nexus_token');
    if (savedToken) {
      validateToken(savedToken);
    }
  }, []);

  // Fetch system status
  useEffect(() => {
    if (user?.authenticated) {
      fetchSystemStatus();
      const interval = setInterval(fetchSystemStatus, 30000); // Update every 30 seconds
      return () => clearInterval(interval);
    }
  }, [user]);

  const validateToken = async (authToken: string) => {
    try {
      const response = await fetch('/gateway/nexus/system/status', {
        headers: {
          'Authorization': `Bearer ${authToken}`
        }
      });

      if (response.ok) {
        // Mock user data based on token for demo
        const mockUsers: Record<string, AuthUser> = {
          'testnet-admin-123': {
            user: 'testnet-admin',
            role: 'admin',
            permissions: ['*:*'],
            nexus_access: ['dve:*', 'validation:*', 'system:*'],
            authenticated: true
          },
          'testnet-validator-456': {
            user: 'testnet-validator',
            role: 'validator',
            permissions: ['nexus:read', 'nexus:validate'],
            nexus_access: ['dve:read', 'validation:read', 'validation:execute'],
            node_id: 'validator-node-001',
            authenticated: true
          },
          'testnet-observer-789': {
            user: 'testnet-observer',
            role: 'observer',
            permissions: ['*:read'],
            nexus_access: ['dve:read', 'validation:read', 'system:read'],
            authenticated: true
          }
        };

        setUser(mockUsers[authToken] || {
          user: 'anonymous',
          role: 'observer',
          permissions: ['*:read'],
          nexus_access: ['dve:read', 'validation:read'],
          authenticated: true
        });
        setIsConnected(true);
      } else {
        localStorage.removeItem('knirv_nexus_token');
        setIsConnected(false);
      }
    } catch (error) {
      console.error('Token validation failed:', error);
      setIsConnected(false);
    }
  };

  const fetchSystemStatus = async () => {
    try {
      const token = localStorage.getItem('knirv_nexus_token');
      const response = await fetch('/gateway/nexus/system/status', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const status = await response.json();
        setSystemStatus(status);
        setIsConnected(true);
      }
    } catch (error) {
      console.error('Failed to fetch system status:', error);
      setIsConnected(false);
    }
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoginError('');

    try {
      await validateToken(token);
      if (user?.authenticated) {
        localStorage.setItem('knirv_nexus_token', token);
        setToken('');
      } else {
        setLoginError('Invalid token. Please check your credentials.');
      }
    } catch (error) {
      setLoginError('Login failed. Please try again.');
    }
  };

  const handleLogout = () => {
    setUser(null);
    setSystemStatus(null);
    setIsConnected(false);
    localStorage.removeItem('knirv_nexus_token');
  };

  const useTestnetToken = (testToken: string) => {
    setToken(testToken);
    setLoginError('');
  };

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'admin': return 'bg-red-500 text-white';
      case 'validator': return 'bg-blue-500 text-white';
      case 'observer': return 'bg-gray-500 text-white';
      default: return 'bg-gray-500 text-white';
    }
  };

  const hasPermission = (service: string, operation: string) => {
    if (!user?.authenticated) return false;
    if (user.role === 'admin') return true;
    
    const requiredPermission = `${service}:${operation}`;
    const hasWildcard = user.nexus_access.includes(`${service}:*`);
    const hasSpecific = user.nexus_access.includes(requiredPermission);
    
    return hasWildcard || hasSpecific;
  };

  // Login screen
  if (!user?.authenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-purple-50">
        <div className="w-full max-w-md space-y-6 p-6">
          <div className="text-center space-y-4">
            <div className="mx-auto w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center">
              <Shield className="w-8 h-8 text-blue-600" />
            </div>
            <h1 className="text-3xl font-bold knirv-gradient-text">KNIRV NEXUS Portal</h1>
            <p className="text-gray-600">Decentralized Validation Environment</p>
          </div>

          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Access Token
              </label>
              <input
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="Enter your access token"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              />
            </div>

            {loginError && (
              <div className="p-3 bg-red-100 border border-red-400 text-red-700 rounded">
                {loginError}
              </div>
            )}

            <button
              type="submit"
              className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
              disabled={!token.trim()}
            >
              Login
            </button>
          </form>

          {/* Testnet tokens */}
          <div className="border-t pt-6">
            <h3 className="text-sm font-medium text-gray-700 mb-3 text-center">
              Testnet Tokens (Development)
            </h3>
            <div className="space-y-2">
              <button
                onClick={() => useTestnetToken('testnet-admin-123')}
                className="w-full flex items-center justify-between p-2 bg-red-50 border border-red-200 rounded text-sm hover:bg-red-100"
              >
                <span className="flex items-center space-x-2">
                  <User className="w-4 h-4" />
                  <span>Admin</span>
                </span>
                <span className="bg-red-500 text-white px-2 py-1 rounded text-xs">Full Access</span>
              </button>

              <button
                onClick={() => useTestnetToken('testnet-validator-456')}
                className="w-full flex items-center justify-between p-2 bg-blue-50 border border-blue-200 rounded text-sm hover:bg-blue-100"
              >
                <span className="flex items-center space-x-2">
                  <Shield className="w-4 h-4" />
                  <span>Validator</span>
                </span>
                <span className="bg-blue-500 text-white px-2 py-1 rounded text-xs">Scoped</span>
              </button>

              <button
                onClick={() => useTestnetToken('testnet-observer-789')}
                className="w-full flex items-center justify-between p-2 bg-gray-50 border border-gray-200 rounded text-sm hover:bg-gray-100"
              >
                <span className="flex items-center space-x-2">
                  <Eye className="w-4 h-4" />
                  <span>Observer</span>
                </span>
                <span className="bg-gray-500 text-white px-2 py-1 rounded text-xs">Read Only</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Main dashboard
  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Shield className="w-8 h-8 text-blue-600" />
              <div>
                <h1 className="text-2xl font-bold knirv-gradient-text">KNIRV NEXUS Portal</h1>
                <p className="text-sm text-gray-600">Decentralized Validation Environment</p>
              </div>
            </div>

            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                {isConnected ? (
                  <span className="flex items-center space-x-1 text-green-600">
                    <Wifi className="w-4 h-4" />
                    <span className="text-sm">Connected</span>
                  </span>
                ) : (
                  <span className="flex items-center space-x-1 text-red-600">
                    <WifiOff className="w-4 h-4" />
                    <span className="text-sm">Disconnected</span>
                  </span>
                )}

                {/* Real-time connection status */}
                {user?.authenticated && (
                  <div className="flex items-center space-x-1">
                    <Radio className={`w-3 h-3 ${
                      systemRealtime.connected || dveRealtime.connected || validationRealtime.connected
                        ? 'text-green-500'
                        : 'text-gray-400'
                    }`} />
                    <span className="text-xs text-gray-500">
                      {systemRealtime.connectionType || 'none'}
                    </span>
                  </div>
                )}
              </div>

              <div className="flex items-center space-x-2">
                <span className="text-sm text-gray-600">Welcome,</span>
                <span className="font-medium">{user.user}</span>
                <span className={`px-2 py-1 rounded text-xs ${getRoleBadgeColor(user.role)}`}>
                  {user.role.toUpperCase()}
                </span>
              </div>

              <button
                onClick={handleLogout}
                className="text-sm text-gray-600 hover:text-gray-800"
              >
                Logout
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex space-x-8">
            <button
              onClick={() => setActiveTab('overview')}
              className={`py-4 px-2 border-b-2 font-medium text-sm ${
                activeTab === 'overview'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              <BarChart3 className="w-4 h-4 inline mr-2" />
              Overview
            </button>

            {hasPermission('dve', 'read') && (
              <button
                onClick={() => setActiveTab('nodes')}
                className={`py-4 px-2 border-b-2 font-medium text-sm ${
                  activeTab === 'nodes'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700'
                }`}
              >
                <Server className="w-4 h-4 inline mr-2" />
                DVE Nodes
              </button>
            )}

            {hasPermission('validation', 'read') && (
              <button
                onClick={() => setActiveTab('validation')}
                className={`py-4 px-2 border-b-2 font-medium text-sm ${
                  activeTab === 'validation'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700'
                }`}
              >
                <Activity className="w-4 h-4 inline mr-2" />
                Validation
              </button>
            )}

            {hasPermission('system', 'read') && (
              <button
                onClick={() => setActiveTab('system')}
                className={`py-4 px-2 border-b-2 font-medium text-sm ${
                  activeTab === 'system'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700'
                }`}
              >
                <Settings className="w-4 h-4 inline mr-2" />
                System
              </button>
            )}
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main className="max-w-7xl mx-auto px-4 py-8">
        {activeTab === 'overview' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <h2 className="text-2xl font-bold text-gray-900">System Overview</h2>
              <div className="flex items-center space-x-2 text-sm text-gray-500">
                <Radio className={`w-4 h-4 ${systemRealtime.connected ? 'text-green-500' : 'text-gray-400'}`} />
                <span>Real-time updates {systemRealtime.connected ? 'active' : 'inactive'}</span>
              </div>
            </div>

            {systemStatus && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="knirv-card-gradient p-6 rounded-lg">
                  <h3 className="text-lg font-semibold mb-4 flex items-center justify-between">
                    <div className="flex items-center">
                      <Server className="w-5 h-5 mr-2" />
                      DVE Manager
                    </div>
                    <div className="flex items-center space-x-1">
                      <Radio className={`w-3 h-3 ${dveRealtime.connected ? 'text-green-500' : 'text-gray-400'}`} />
                      <span className="text-xs text-gray-500">{dveRealtime.connectionType}</span>
                    </div>
                  </h3>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-gray-600">Status:</span>
                      <span className={`font-medium ${
                        systemStatus.dve_manager.status === 'running' ? 'text-green-600' : 'text-red-600'
                      }`}>
                        {systemStatus.dve_manager.status}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Service:</span>
                      <span className="font-medium">{systemStatus.dve_manager.service}</span>
                    </div>
                    {dveRealtime.lastMessage && (
                      <div className="text-xs text-gray-500 mt-2">
                        Last update: {new Date(dveRealtime.lastMessage.timestamp).toLocaleTimeString()}
                      </div>
                    )}
                  </div>
                </div>

                <div className="knirv-card-gradient p-6 rounded-lg">
                  <h3 className="text-lg font-semibold mb-4 flex items-center justify-between">
                    <div className="flex items-center">
                      <Activity className="w-5 h-5 mr-2" />
                      Validation Core
                    </div>
                    <div className="flex items-center space-x-1">
                      <Radio className={`w-3 h-3 ${validationRealtime.connected ? 'text-green-500' : 'text-gray-400'}`} />
                      <span className="text-xs text-gray-500">{validationRealtime.connectionType}</span>
                    </div>
                  </h3>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-gray-600">Status:</span>
                      <span className={`font-medium ${
                        systemStatus.validation_core.status === 'running' ? 'text-green-600' : 'text-red-600'
                      }`}>
                        {systemStatus.validation_core.status}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Service:</span>
                      <span className="font-medium">{systemStatus.validation_core.service}</span>
                    </div>
                    {validationRealtime.lastMessage && (
                      <div className="text-xs text-gray-500 mt-2">
                        Last update: {new Date(validationRealtime.lastMessage.timestamp).toLocaleTimeString()}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="knirv-card-gradient p-6 rounded-lg">
                <h3 className="text-lg font-semibold mb-4">Your Permissions</h3>
                <div className="grid grid-cols-2 gap-2">
                  {user.nexus_access.map((permission, index) => (
                    <span
                      key={index}
                      className="px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm"
                    >
                      {permission}
                    </span>
                  ))}
                </div>
              </div>

              <div className="knirv-card-gradient p-6 rounded-lg">
                <h3 className="text-lg font-semibold mb-4 flex items-center">
                  <Radio className="w-5 h-5 mr-2" />
                  Real-time Activity
                </h3>
                <div className="space-y-2 max-h-40 overflow-y-auto">
                  {[...systemRealtime.messages, ...dveRealtime.messages, ...validationRealtime.messages]
                    .sort((a, b) => b.timestamp - a.timestamp)
                    .slice(0, 5)
                    .map((message, index) => (
                      <div key={index} className="text-sm p-2 bg-white/50 rounded border-l-2 border-blue-500">
                        <div className="flex justify-between items-start">
                          <span className="font-medium text-blue-700">{message.type}</span>
                          <span className="text-xs text-gray-500">
                            {new Date(message.timestamp).toLocaleTimeString()}
                          </span>
                        </div>
                        <div className="text-gray-600 text-xs mt-1">
                          Channel: {message.channel}
                        </div>
                      </div>
                    ))}
                  {[...systemRealtime.messages, ...dveRealtime.messages, ...validationRealtime.messages].length === 0 && (
                    <div className="text-gray-500 text-sm text-center py-4">
                      No real-time messages yet
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'nodes' && hasPermission('dve', 'read') && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-gray-900">DVE Nodes</h2>
            <div className="knirv-card-gradient p-6 rounded-lg">
              <p className="text-gray-600">DVE Nodes management interface will be loaded here.</p>
              <p className="text-sm text-gray-500 mt-2">
                This section provides access to Decentralized Validation Environment nodes.
              </p>
            </div>
          </div>
        )}

        {activeTab === 'validation' && hasPermission('validation', 'read') && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-gray-900">Validation Tasks</h2>
            <div className="knirv-card-gradient p-6 rounded-lg">
              <p className="text-gray-600">Validation tasks and results interface will be loaded here.</p>
              {hasPermission('validation', 'execute') ? (
                <p className="text-sm text-green-600 mt-2">
                  ✓ You have permission to execute validation tasks.
                </p>
              ) : (
                <p className="text-sm text-gray-500 mt-2">
                  ℹ You have read-only access to validation tasks.
                </p>
              )}
            </div>
          </div>
        )}

        {activeTab === 'system' && hasPermission('system', 'read') && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-gray-900">System Status</h2>
            <div className="knirv-card-gradient p-6 rounded-lg">
              <p className="text-gray-600">System monitoring and metrics interface will be loaded here.</p>
              <p className="text-sm text-gray-500 mt-2">
                Monitor system health, performance metrics, and operational status.
              </p>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
