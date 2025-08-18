import React from 'react';
import { Server, Activity, Shield, BarChart3, Radio, Wifi } from 'lucide-react';

interface ValidatorDashboardProps {
  user: {
    user: string;
    role: string;
    node_id?: string;
  };
  systemRealtime: {
    connected: boolean;
    connectionType: string;
    messages: any[];
  };
}

export function ValidatorDashboard({ user, systemRealtime }: ValidatorDashboardProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">My Validator Dashboard</h2>
          <p className="text-gray-600">Node: {user.node_id || 'Not assigned'}</p>
        </div>
        <div className="flex items-center space-x-2">
          <Radio className={`w-4 h-4 ${systemRealtime.connected ? 'text-green-500' : 'text-gray-400'}`} />
          <span className="text-sm text-gray-500">{systemRealtime.connectionType}</span>
        </div>
      </div>

      {/* My Node Status */}
      <div className="knirv-card-gradient p-6 rounded-lg">
        <h3 className="text-lg font-semibold mb-4 flex items-center">
          <Server className="w-5 h-5 mr-2 text-blue-600" />
          My Node Status
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="text-center p-4 bg-green-50 rounded-lg">
            <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-2">
              <Wifi className="w-6 h-6 text-green-600" />
            </div>
            <p className="text-sm text-gray-600">Status</p>
            <p className="text-lg font-semibold text-green-600">Online</p>
          </div>
          <div className="text-center p-4 bg-blue-50 rounded-lg">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-2">
              <Activity className="w-6 h-6 text-blue-600" />
            </div>
            <p className="text-sm text-gray-600">Validations</p>
            <p className="text-lg font-semibold text-blue-600">24</p>
          </div>
          <div className="text-center p-4 bg-purple-50 rounded-lg">
            <div className="w-12 h-12 bg-purple-100 rounded-full flex items-center justify-center mx-auto mb-2">
              <Shield className="w-6 h-6 text-purple-600" />
            </div>
            <p className="text-sm text-gray-600">Reputation</p>
            <p className="text-lg font-semibold text-purple-600">98.5%</p>
          </div>
        </div>
      </div>

      {/* Performance Metrics */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="knirv-card-gradient p-6 rounded-lg">
          <h3 className="text-lg font-semibold mb-4 flex items-center">
            <BarChart3 className="w-5 h-5 mr-2 text-blue-500" />
            Node Performance
          </h3>
          <div className="space-y-4">
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">CPU Usage</span>
                <span className="font-medium">32%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-blue-600 h-2 rounded-full" style={{width: '32%'}}></div>
              </div>
            </div>
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">Memory Usage</span>
                <span className="font-medium">58%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-green-600 h-2 rounded-full" style={{width: '58%'}}></div>
              </div>
            </div>
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">Network I/O</span>
                <span className="font-medium">15%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-yellow-600 h-2 rounded-full" style={{width: '15%'}}></div>
              </div>
            </div>
          </div>
        </div>

        <div className="knirv-card-gradient p-6 rounded-lg">
          <h3 className="text-lg font-semibold mb-4">Earnings & Rewards</h3>
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Today's Earnings</span>
              <span className="font-semibold text-green-600">+125 NRN</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">This Week</span>
              <span className="font-semibold text-green-600">+890 NRN</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Total Staked</span>
              <span className="font-semibold">1,000 NRN</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Reputation Score</span>
              <span className="font-semibold text-blue-600">98.5%</span>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Tasks */}
      <div className="knirv-card-gradient rounded-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">My Recent Tasks</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Task ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Reward</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Completed</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              <tr>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">task-001</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">AI Model Validation</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                    Completed
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-green-600 font-medium">+50 NRN</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">2 hours ago</td>
              </tr>
              <tr>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">task-002</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">Data Integrity Check</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                    Running
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">Pending</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">In progress</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="knirv-card-gradient p-6 rounded-lg">
        <h3 className="text-lg font-semibold mb-4">Quick Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <button className="p-4 bg-blue-50 hover:bg-blue-100 rounded-lg transition-colors text-left">
            <Activity className="w-6 h-6 text-blue-600 mb-2" />
            <p className="font-medium text-blue-900">View Node Metrics</p>
            <p className="text-sm text-blue-600">Check detailed performance</p>
          </button>
          <button className="p-4 bg-green-50 hover:bg-green-100 rounded-lg transition-colors text-left">
            <Shield className="w-6 h-6 text-green-600 mb-2" />
            <p className="font-medium text-green-900">Update Configuration</p>
            <p className="text-sm text-green-600">Modify node settings</p>
          </button>
          <button className="p-4 bg-purple-50 hover:bg-purple-100 rounded-lg transition-colors text-left">
            <BarChart3 className="w-6 h-6 text-purple-600 mb-2" />
            <p className="font-medium text-purple-900">View Earnings</p>
            <p className="text-sm text-purple-600">Check reward history</p>
          </button>
        </div>
      </div>
    </div>
  );
}
