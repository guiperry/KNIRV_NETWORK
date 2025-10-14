import React, { useState } from 'react';
import { NexusDashboard } from './NexusDashboard';
import {
  Shield,
  Server,
  Activity,
  Eye,
  User,
  Settings,
  BarChart3,
  Lock,
  Unlock,
  LogOut,
  Wifi,
  WifiOff,
  Bell
} from 'lucide-react';

interface DashboardWrapperProps {
  children: React.ReactNode;
  user: {
    user: string;
    role: 'admin' | 'validator' | 'observer';
    permissions: string[];
    nexus_access: string[];
    node_id?: string;
  };
  onLogout: () => void;
  isConnected: boolean;
  alerts: number;
}

export function DashboardWrapper({ 
  children, 
  user, 
  onLogout, 
  isConnected, 
  alerts 
}: DashboardWrapperProps) {
  const [activeTab, setActiveTab] = useState('overview');

  const hasPermission = (resource: string, operation: string) => {
    if (user.role === 'admin') return true;
    
    const permission = `${resource}:${operation}`;
    return user.permissions.includes(permission) || user.permissions.includes(`${resource}:*`) || user.permissions.includes('*:*');
  };

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'admin': return 'bg-red-500';
      case 'validator': return 'bg-blue-500';
      case 'observer': return 'bg-green-500';
      default: return 'bg-gray-500';
    }
  };

  const tabs = [
    { id: 'overview', label: 'Overview', icon: BarChart3, permission: null },
    { id: 'nodes', label: 'DVE Nodes', icon: Server, permission: 'dve:read' },
    { id: 'validation', label: 'Validation', icon: Activity, permission: 'validation:read' },
    { id: 'system', label: 'System', icon: Settings, permission: 'system:read' },
    { id: 'admin', label: 'Admin', icon: User, permission: 'admin:read', adminOnly: true },
    { id: 'profile', label: 'Profile', icon: Eye, permission: null }
  ];

  const visibleTabs = tabs.filter(tab => {
    if (tab.adminOnly && user.role !== 'admin') return false;
    if (tab.permission && !hasPermission(tab.permission.split(':')[0], tab.permission.split(':')[1])) return false;
    return true;
  });

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900">
      {/* Header */}
      <header className="bg-black/20 backdrop-blur-md border-b border-white/10">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-2">
                <Shield className="w-8 h-8 text-blue-400" />
                <div>
                  <h1 className="text-2xl font-bold text-white">KNIRV NEXUS</h1>
                  <p className="text-sm text-gray-300">Decentralized Validation Environment</p>
                </div>
              </div>
            </div>
            
            <div className="flex items-center space-x-4">
              {/* Connection Status */}
              <div className="flex items-center space-x-2">
                {isConnected ? (
                  <div className="flex items-center space-x-1 text-green-400">
                    <Wifi className="w-4 h-4" />
                    <span className="text-sm">Live</span>
                  </div>
                ) : (
                  <div className="flex items-center space-x-1 text-red-400">
                    <WifiOff className="w-4 h-4" />
                    <span className="text-sm">Offline</span>
                  </div>
                )}
                
                {/* Alerts */}
                {alerts > 0 && (
                  <div className="flex items-center space-x-1 text-red-400">
                    <Bell className="w-4 h-4" />
                    <span className="text-sm">{alerts}</span>
                  </div>
                )}
              </div>

              {/* User Info */}
              <div className="flex items-center space-x-2 text-sm">
                <span className="text-gray-300">Welcome,</span>
                <span className="font-medium text-white">{user.user}</span>
                <span className={`px-2 py-1 rounded text-xs font-medium text-white ${getRoleBadgeColor(user.role)}`}>
                  {user.role.toUpperCase()}
                </span>
              </div>

              {/* Logout Button */}
              <button
                onClick={onLogout}
                className="flex items-center space-x-1 text-gray-300 hover:text-white transition-colors"
              >
                <LogOut className="w-4 h-4" />
                <span className="text-sm">Logout</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation Tabs */}
      <nav className="bg-black/10 backdrop-blur-md border-b border-white/10">
        <div className="container mx-auto px-4">
          <div className="flex space-x-8 overflow-x-auto">
            {visibleTabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center space-x-2 py-4 px-2 border-b-2 font-medium text-sm transition-colors ${
                    activeTab === tab.id
                      ? 'border-blue-500 text-blue-400'
                      : 'border-transparent text-gray-400 hover:text-gray-200'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  <span>{tab.label}</span>
                </button>
              );
            })}
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        {activeTab === 'overview' && (
          <NexusDashboard isConnected={isConnected} alerts={alerts} />
        )}
        
        {activeTab === 'nodes' && hasPermission('dve', 'read') && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-white">DVE Nodes Management</h2>
            <p className="text-gray-300">
              Monitor and manage Decentralized Validation Environment nodes.
            </p>
            <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4">
              <div className="flex items-center space-x-2">
                <Server className="w-5 h-5 text-yellow-400" />
                <span className="text-yellow-400 font-medium">DVE Nodes interface is being loaded...</span>
              </div>
            </div>
          </div>
        )}
        
        {activeTab === 'validation' && hasPermission('validation', 'read') && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-white">Validation Tasks</h2>
            <p className="text-gray-300">
              View and manage validation tasks and results.
            </p>
            
            {hasPermission('validation', 'execute') ? (
              <div className="bg-green-500/10 border border-green-500/20 rounded-lg p-4">
                <div className="flex items-center space-x-2">
                  <Unlock className="w-5 h-5 text-green-400" />
                  <span className="text-green-400 font-medium">You have permission to execute validation tasks.</span>
                </div>
              </div>
            ) : (
              <div className="bg-blue-500/10 border border-blue-500/20 rounded-lg p-4">
                <div className="flex items-center space-x-2">
                  <Lock className="w-5 h-5 text-blue-400" />
                  <span className="text-blue-400 font-medium">You have read-only access to validation tasks.</span>
                </div>
              </div>
            )}
          </div>
        )}
        
        {activeTab === 'system' && hasPermission('system', 'read') && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-white">System Status</h2>
            <p className="text-gray-300">
              Monitor system health, metrics, and performance.
            </p>
          </div>
        )}
        
        {activeTab === 'admin' && user.role === 'admin' && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-white">Administration</h2>
            <p className="text-gray-300">
              Administrative controls and system configuration.
            </p>
            
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
                <div className="flex items-center space-x-2 mb-4">
                  <User className="w-5 h-5 text-blue-400" />
                  <h3 className="text-lg font-semibold text-white">User Management</h3>
                </div>
                <p className="text-gray-300 text-sm mb-4">Manage user roles and permissions</p>
                <button className="w-full bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded-lg transition-colors">
                  Manage Users
                </button>
              </div>
              
              <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
                <div className="flex items-center space-x-2 mb-4">
                  <Settings className="w-5 h-5 text-green-400" />
                  <h3 className="text-lg font-semibold text-white">System Config</h3>
                </div>
                <p className="text-gray-300 text-sm mb-4">Configure system parameters</p>
                <button className="w-full bg-green-600 hover:bg-green-700 text-white py-2 px-4 rounded-lg transition-colors">
                  Configure
                </button>
              </div>
              
              <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
                <div className="flex items-center space-x-2 mb-4">
                  <Shield className="w-5 h-5 text-purple-400" />
                  <h3 className="text-lg font-semibold text-white">Security</h3>
                </div>
                <p className="text-gray-300 text-sm mb-4">Security settings and audit logs</p>
                <button className="w-full bg-purple-600 hover:bg-purple-700 text-white py-2 px-4 rounded-lg transition-colors">
                  Security Center
                </button>
              </div>
            </div>
          </div>
        )}
        
        {activeTab === 'profile' && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-white">User Profile</h2>
            <p className="text-gray-300">
              Your account information and permissions.
            </p>
            
            <div className="bg-white/10 backdrop-blur-md rounded-lg p-6 border border-white/20">
              <h3 className="text-lg font-semibold text-white mb-4">Account Information</h3>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm font-medium text-gray-300">Username</p>
                    <p className="text-lg text-white">{user.user}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-300">Role</p>
                    <span className={`px-2 py-1 rounded text-sm font-medium text-white ${getRoleBadgeColor(user.role)}`}>
                      {user.role.toUpperCase()}
                    </span>
                  </div>
                </div>
                
                {user.node_id && (
                  <div>
                    <p className="text-sm font-medium text-gray-300">Assigned Node</p>
                    <span className="px-2 py-1 bg-blue-500/20 border border-blue-500/30 rounded text-sm text-blue-300">
                      {user.node_id}
                    </span>
                  </div>
                )}
                
                <div>
                  <p className="text-sm font-medium text-gray-300 mb-2">NEXUS Permissions</p>
                  <div className="flex flex-wrap gap-2">
                    {user.nexus_access.map((permission, index) => (
                      <span 
                        key={index} 
                        className="px-2 py-1 bg-green-500/20 border border-green-500/30 rounded text-xs text-green-300"
                      >
                        {permission}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
