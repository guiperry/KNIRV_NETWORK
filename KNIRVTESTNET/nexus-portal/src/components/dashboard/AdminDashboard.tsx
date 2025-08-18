import React from 'react';
import { Server, Activity, Shield, BarChart3, Users, Settings, AlertTriangle, Radio } from 'lucide-react';

interface AdminDashboardProps {
  systemRealtime: {
    connected: boolean;
    connectionType: string;
    messages: any[];
  };
}

export function AdminDashboard({ systemRealtime }: AdminDashboardProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Network Administration</h2>
          <p className="text-gray-600">Complete network overview and management</p>
        </div>
        <div className="flex items-center space-x-2">
          <Radio className={`w-4 h-4 ${systemRealtime.connected ? 'text-green-500' : 'text-gray-400'}`} />
          <span className="text-sm text-gray-500">{systemRealtime.connectionType}</span>
        </div>
      </div>

      {/* Network Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="knirv-card-gradient p-6 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Total Nodes</p>
              <p className="text-3xl font-bold text-gray-900">12</p>
              <p className="text-sm text-green-600">+2 this week</p>
            </div>
            <Server className="w-8 h-8 text-blue-600" />
          </div>
        </div>
        <div className="knirv-card-gradient p-6 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Active Validators</p>
              <p className="text-3xl font-bold text-green-600">10</p>
              <p className="text-sm text-gray-500">83.3% uptime</p>
            </div>
            <Shield className="w-8 h-8 text-green-600" />
          </div>
        </div>
        <div className="knirv-card-gradient p-6 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Total Tasks</p>
              <p className="text-3xl font-bold text-blue-600">1,247</p>
              <p className="text-sm text-blue-600">+156 today</p>
            </div>
            <Activity className="w-8 h-8 text-blue-600" />
          </div>
        </div>
        <div className="knirv-card-gradient p-6 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Network Health</p>
              <p className="text-3xl font-bold text-green-600">98.5%</p>
              <p className="text-sm text-green-600">Excellent</p>
            </div>
            <BarChart3 className="w-8 h-8 text-green-600" />
          </div>
        </div>
      </div>

      {/* Critical Alerts */}
      <div className="knirv-card-gradient rounded-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 bg-red-50">
          <h3 className="text-lg font-semibold text-red-900 flex items-center">
            <AlertTriangle className="w-5 h-5 mr-2" />
            Critical Alerts
          </h3>
        </div>
        <div className="p-6">
          <div className="space-y-3">
            <div className="flex items-start p-3 bg-red-50 border border-red-200 rounded-lg">
              <div className="w-2 h-2 bg-red-500 rounded-full mt-2 mr-3"></div>
              <div className="flex-1">
                <p className="text-sm font-medium text-red-800">Node Offline</p>
                <p className="text-xs text-red-600 mt-1">validator-node-003 has been offline for 15 minutes</p>
                <div className="flex space-x-2 mt-2">
                  <button className="text-xs bg-red-600 text-white px-2 py-1 rounded">Investigate</button>
                  <button className="text-xs bg-gray-600 text-white px-2 py-1 rounded">Acknowledge</button>
                </div>
              </div>
            </div>
            <div className="flex items-start p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
              <div className="w-2 h-2 bg-yellow-500 rounded-full mt-2 mr-3"></div>
              <div className="flex-1">
                <p className="text-sm font-medium text-yellow-800">High Resource Usage</p>
                <p className="text-xs text-yellow-600 mt-1">Multiple nodes showing high memory usage (>85%)</p>
                <div className="flex space-x-2 mt-2">
                  <button className="text-xs bg-yellow-600 text-white px-2 py-1 rounded">Scale Resources</button>
                  <button className="text-xs bg-gray-600 text-white px-2 py-1 rounded">Monitor</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Network Performance */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="knirv-card-gradient p-6 rounded-lg">
          <h3 className="text-lg font-semibold mb-4 flex items-center">
            <BarChart3 className="w-5 h-5 mr-2 text-blue-500" />
            Network Performance
          </h3>
          <div className="space-y-4">
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">Consensus Participation</span>
                <span className="font-medium">95.2%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-green-600 h-2 rounded-full" style={{width: '95.2%'}}></div>
              </div>
            </div>
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">Task Success Rate</span>
                <span className="font-medium">98.7%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-blue-600 h-2 rounded-full" style={{width: '98.7%'}}></div>
              </div>
            </div>
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">Network Latency</span>
                <span className="font-medium">25ms avg</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-yellow-600 h-2 rounded-full" style={{width: '30%'}}></div>
              </div>
            </div>
          </div>
        </div>

        <div className="knirv-card-gradient p-6 rounded-lg">
          <h3 className="text-lg font-semibold mb-4 flex items-center">
            <Users className="w-5 h-5 mr-2 text-purple-500" />
            User Management
          </h3>
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Total Users</span>
              <span className="font-semibold">47</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Active Validators</span>
              <span className="font-semibold text-green-600">12</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Observers</span>
              <span className="font-semibold text-blue-600">28</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Administrators</span>
              <span className="font-semibold text-purple-600">7</span>
            </div>
            <button className="w-full mt-4 bg-purple-600 text-white py-2 px-4 rounded-lg hover:bg-purple-700 transition-colors">
              Manage Users
            </button>
          </div>
        </div>
      </div>

      {/* Quick Admin Actions */}
      <div className="knirv-card-gradient p-6 rounded-lg">
        <h3 className="text-lg font-semibold mb-4">Quick Admin Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <button className="p-4 bg-blue-50 hover:bg-blue-100 rounded-lg transition-colors text-left">
            <Server className="w-6 h-6 text-blue-600 mb-2" />
            <p className="font-medium text-blue-900">Add Node</p>
            <p className="text-sm text-blue-600">Register new validator</p>
          </button>
          <button className="p-4 bg-green-50 hover:bg-green-100 rounded-lg transition-colors text-left">
            <Users className="w-6 h-6 text-green-600 mb-2" />
            <p className="font-medium text-green-900">User Management</p>
            <p className="text-sm text-green-600">Manage roles & permissions</p>
          </button>
          <button className="p-4 bg-purple-50 hover:bg-purple-100 rounded-lg transition-colors text-left">
            <Settings className="w-6 h-6 text-purple-600 mb-2" />
            <p className="font-medium text-purple-900">Network Config</p>
            <p className="text-sm text-purple-600">Update network settings</p>
          </button>
          <button className="p-4 bg-red-50 hover:bg-red-100 rounded-lg transition-colors text-left">
            <AlertTriangle className="w-6 h-6 text-red-600 mb-2" />
            <p className="font-medium text-red-900">Emergency Actions</p>
            <p className="text-sm text-red-600">Network emergency controls</p>
          </button>
        </div>
      </div>

      {/* Recent Network Activity */}
      <div className="knirv-card-gradient rounded-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">Recent Network Activity</h3>
        </div>
        <div className="p-6">
          <div className="space-y-3 max-h-60 overflow-y-auto">
            {systemRealtime.messages.slice(0, 10).map((message, index) => (
              <div key={index} className="flex items-start p-3 bg-gray-50 rounded-lg">
                <div className="w-2 h-2 bg-blue-500 rounded-full mt-2 mr-3"></div>
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-900">{message.type}</p>
                  <p className="text-xs text-gray-600 mt-1">Channel: {message.channel}</p>
                  <p className="text-xs text-gray-500 mt-1">
                    {new Date(message.timestamp).toLocaleTimeString()}
                  </p>
                </div>
              </div>
            ))}
            {systemRealtime.messages.length === 0 && (
              <div className="text-center py-8 text-gray-500">
                <Radio className="w-8 h-8 mx-auto mb-2 text-gray-400" />
                <p>No real-time activity yet</p>
                <p className="text-sm">Network events will appear here</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
