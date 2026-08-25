import React, { useState } from 'react';
import { Settings as SettingsIcon, User, Shield, Bell, Palette, Database, Key, Globe } from 'lucide-react';
import { useAuth } from './AuthContext';
import { GlassCard } from './common';

const Settings: React.FC = () => {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState('profile');

  const tabs = [
    { id: 'profile', name: 'Profile', icon: User },
    { id: 'security', name: 'Security', icon: Shield },
    { id: 'notifications', name: 'Notifications', icon: Bell },
    { id: 'appearance', name: 'Appearance', icon: Palette },
    { id: 'data', name: 'Data & Privacy', icon: Database },
    { id: 'api', name: 'API Keys', icon: Key },
    { id: 'network', name: 'Network', icon: Globe }
  ];

  return (
    <div className="p-6">
      <div className="flex items-center space-x-3 mb-6">
        <SettingsIcon className="w-8 h-8 text-blue-400" />
        <h1 className="text-2xl font-bold text-white">Settings</h1>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
        {/* Settings Navigation */}
        <div className="lg:w-64">
          <GlassCard title="Settings Menu">
            <nav className="space-y-2">
              {tabs.map((tab) => {
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`w-full flex items-center space-x-3 px-3 py-2 rounded-lg transition-colors ${
                      activeTab === tab.id
                        ? 'bg-blue-600/30 text-blue-400 border border-blue-500/30'
                        : 'text-slate-400 hover:text-white hover:bg-slate-700/30'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    <span>{tab.name}</span>
                  </button>
                );
              })}
            </nav>
          </GlassCard>
        </div>

        {/* Settings Content */}
        <div className="flex-1">
          {activeTab === 'profile' && (
            <GlassCard title="Profile Settings">
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Username
                  </label>
                  <input
                    type="text"
                    value={user?.username || ''}
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    readOnly
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Email
                  </label>
                  <input
                    type="email"
                    value={user?.email || ''}
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    readOnly
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Role
                  </label>
                  <input
                    type="text"
                    value={user?.role || ''}
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-4 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    readOnly
                  />
                </div>
              </div>
            </GlassCard>
          )}

          {activeTab === 'security' && (
            <GlassCard title="Security Settings">
              <div className="space-y-6">
                <div className="bg-yellow-600/20 border border-yellow-500/30 rounded-lg p-4">
                  <h3 className="text-yellow-400 font-medium mb-2">Two-Factor Authentication</h3>
                  <p className="text-slate-300 text-sm mb-3">
                    Enhance your account security with two-factor authentication.
                  </p>
                  <button className="bg-yellow-600 text-white px-4 py-2 rounded-lg hover:bg-yellow-700 transition-colors">
                    Enable 2FA
                  </button>
                </div>
                <div>
                  <h3 className="text-white font-medium mb-3">Active Sessions</h3>
                  <div className="bg-slate-700/30 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-white">Current Session</p>
                        <p className="text-slate-400 text-sm">Desktop Client - {new Date().toLocaleDateString()}</p>
                      </div>
                      <span className="text-green-400 text-sm">Active</span>
                    </div>
                  </div>
                </div>
              </div>
            </GlassCard>
          )}

          {activeTab === 'notifications' && (
            <GlassCard title="Notification Settings">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-white font-medium">Network Alerts</h3>
                    <p className="text-slate-400 text-sm">Get notified about network status changes</p>
                  </div>
                  <input type="checkbox" className="w-4 h-4" defaultChecked />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-white font-medium">Agent Updates</h3>
                    <p className="text-slate-400 text-sm">Notifications about agent status and activities</p>
                  </div>
                  <input type="checkbox" className="w-4 h-4" defaultChecked />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-white font-medium">System Maintenance</h3>
                    <p className="text-slate-400 text-sm">Important system updates and maintenance notices</p>
                  </div>
                  <input type="checkbox" className="w-4 h-4" defaultChecked />
                </div>
              </div>
            </GlassCard>
          )}

          {activeTab === 'appearance' && (
            <GlassCard title="Appearance Settings">
              <div className="space-y-6">
                <div>
                  <h3 className="text-white font-medium mb-3">Theme</h3>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="bg-slate-700/30 rounded-lg p-4 border border-blue-500/30">
                      <div className="w-full h-20 bg-gradient-to-br from-slate-900 via-purple-900 to-slate-900 rounded mb-2"></div>
                      <p className="text-white text-sm">Dark (Current)</p>
                    </div>
                    <div className="bg-slate-700/30 rounded-lg p-4 opacity-50">
                      <div className="w-full h-20 bg-gradient-to-br from-blue-100 to-purple-100 rounded mb-2"></div>
                      <p className="text-slate-400 text-sm">Light (Coming Soon)</p>
                    </div>
                  </div>
                </div>
              </div>
            </GlassCard>
          )}

          {activeTab === 'data' && (
            <GlassCard title="Data & Privacy Settings">
              <div className="space-y-6">
                <div className="bg-blue-600/20 border border-blue-500/30 rounded-lg p-4">
                  <h3 className="text-blue-400 font-medium mb-2">Data Collection</h3>
                  <p className="text-slate-300 text-sm mb-3">
                    We collect minimal data to improve your experience. All data is stored locally and encrypted.
                  </p>
                </div>
                <div>
                  <h3 className="text-white font-medium mb-3">Privacy Controls</h3>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-slate-300">Analytics Data</span>
                      <input type="checkbox" className="w-4 h-4" defaultChecked />
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-slate-300">Usage Statistics</span>
                      <input type="checkbox" className="w-4 h-4" defaultChecked />
                    </div>
                  </div>
                </div>
              </div>
            </GlassCard>
          )}

          {activeTab === 'api' && (
            <GlassCard title="API Keys">
              <div className="space-y-6">
                <div className="bg-purple-600/20 border border-purple-500/30 rounded-lg p-4">
                  <h3 className="text-purple-400 font-medium mb-2">Personal API Key</h3>
                  <p className="text-slate-300 text-sm mb-3">
                    Use this key to access KNIRV Network APIs programmatically.
                  </p>
                  <div className="flex items-center space-x-2">
                    <input
                      type="password"
                      value="knirv_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                      className="flex-1 bg-slate-700/50 border border-slate-600/50 rounded-lg px-4 py-2 text-white"
                      readOnly
                    />
                    <button className="bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition-colors">
                      Regenerate
                    </button>
                  </div>
                </div>
              </div>
            </GlassCard>
          )}

          {activeTab === 'network' && (
            <GlassCard title="Network Settings">
              <div className="space-y-6">
                <div>
                  <h3 className="text-white font-medium mb-3">Connection Status</h3>
                  <div className="bg-green-600/20 border border-green-500/30 rounded-lg p-4">
                    <div className="flex items-center space-x-2">
                      <div className="w-3 h-3 bg-green-400 rounded-full"></div>
                      <span className="text-green-400 font-medium">Connected to KNIRV Network</span>
                    </div>
                    <p className="text-slate-300 text-sm mt-2">
                      Node ID: knirv_node_xxxxxxxxxxxxxxxx
                    </p>
                  </div>
                </div>
                <div>
                  <h3 className="text-white font-medium mb-3">Network Preferences</h3>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-slate-300">Auto-connect on startup</span>
                      <input type="checkbox" className="w-4 h-4" defaultChecked />
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-slate-300">Enable peer discovery</span>
                      <input type="checkbox" className="w-4 h-4" defaultChecked />
                    </div>
                  </div>
                </div>
              </div>
            </GlassCard>
          )}
        </div>
      </div>
    </div>
  );
};

export default Settings;
