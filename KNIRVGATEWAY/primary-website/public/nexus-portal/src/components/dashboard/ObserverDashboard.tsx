import React from 'react';
import { Server, Activity, Shield, BarChart3, Eye, Radio, TrendingUp } from 'lucide-react';

interface ObserverDashboardProps {
  systemRealtime: {
    connected: boolean;
    connectionType: string;
    messages: any[];
  };
}

export function ObserverDashboard({ systemRealtime }: ObserverDashboardProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Network Observer</h2>
          <p className="text-gray-600">Read-only view of KNIRV NEXUS network</p>
        </div>
        <div className="flex items-center space-x-2">
          <Radio className={`w-4 h-4 ${systemRealtime.connected ? 'text-green-500' : 'text-gray-400'}`} />
          <span className="text-sm text-gray-500">{systemRealtime.connectionType}</span>
        </div>
      </div>

      {/* Network Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="knirv-card-gradient p-6 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Network Nodes</p>
              <p className="text-3xl font-bold text-gray-900">12</p>
              <p className="text-sm text-blue-600">10 active</p>
            </div>
            <Server className="w-8 h-8 text-blue-600" />
          </div>
        </div>
        <div className="knirv-card-gradient p-6 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Validations</p>
              <p className="text-3xl font-bold text-green-600">1,247</p>
              <p className="text-sm text-green-600">98.7% success</p>
            </div>
            <Shield className="w-8 h-8 text-green-600" />
          </div>
        </div>
        <div className="knirv-card-gradient p-6 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Network Health</p>
              <p className="text-3xl font-bold text-green-600">98.5%</p>
              <p className="text-sm text-green-600">Excellent</p>
            </div>
            <Activity className="w-8 h-8 text-green-600" />
          </div>
        </div>
        <div className="knirv-card-gradient p-6 rounded-lg">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Block Height</p>
              <p className="text-3xl font-bold text-purple-600">1.2M</p>
              <p className="text-sm text-purple-600">+156 today</p>
            </div>
            <BarChart3 className="w-8 h-8 text-purple-600" />
          </div>
        </div>
      </div>

      {/* Network Health Overview */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="knirv-card-gradient p-6 rounded-lg">
          <h3 className="text-lg font-semibold mb-4 flex items-center">
            <Activity className="w-5 h-5 mr-2 text-green-500" />
            Network Health
          </h3>
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Consensus Rate</span>
              <span className="text-green-600 font-medium">95.2%</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Active Validators</span>
              <span className="font-medium">10/12</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Average Latency</span>
              <span className="text-blue-600 font-medium">25ms</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-600">Network Uptime</span>
              <span className="text-green-600 font-medium">99.8%</span>
            </div>
          </div>
        </div>

        <div className="knirv-card-gradient p-6 rounded-lg">
          <h3 className="text-lg font-semibold mb-4 flex items-center">
            <TrendingUp className="w-5 h-5 mr-2 text-blue-500" />
            Performance Trends
          </h3>
          <div className="space-y-4">
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">Validation Success Rate</span>
                <span className="font-medium">98.7%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-green-600 h-2 rounded-full" style={{width: '98.7%'}}></div>
              </div>
            </div>
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">Network Utilization</span>
                <span className="font-medium">67%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-blue-600 h-2 rounded-full" style={{width: '67%'}}></div>
              </div>
            </div>
            <div>
              <div className="flex justify-between items-center mb-1">
                <span className="text-gray-600">Task Queue Load</span>
                <span className="font-medium">23%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div className="bg-yellow-600 h-2 rounded-full" style={{width: '23%'}}></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Node Status Overview */}
      <div className="knirv-card-gradient rounded-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900 flex items-center">
            <Eye className="w-5 h-5 mr-2" />
            Node Status Overview
          </h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Node</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">TEE Type</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Location</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Uptime</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Validations</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              <tr>
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center">
                    <div className="w-10 h-10 bg-green-100 rounded-full flex items-center justify-center">
                      <Server className="w-5 h-5 text-green-600" />
                    </div>
                    <div className="ml-4">
                      <div className="text-sm font-medium text-gray-900">validator-node-001</div>
                      <div className="text-sm text-gray-500">192.168.1.100</div>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                    Online
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">Intel SGX</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">US-East</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">99.8%</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">247</td>
              </tr>
              <tr>
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center">
                    <div className="w-10 h-10 bg-yellow-100 rounded-full flex items-center justify-center">
                      <Server className="w-5 h-5 text-yellow-600" />
                    </div>
                    <div className="ml-4">
                      <div className="text-sm font-medium text-gray-900">validator-node-002</div>
                      <div className="text-sm text-gray-500">192.168.1.101</div>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                    Syncing
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">AMD SEV</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">EU-West</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">97.2%</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">189</td>
              </tr>
              <tr>
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center">
                    <div className="w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
                      <Server className="w-5 h-5 text-red-600" />
                    </div>
                    <div className="ml-4">
                      <div className="text-sm font-medium text-gray-900">validator-node-003</div>
                      <div className="text-sm text-gray-500">192.168.1.102</div>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                    Offline
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">Intel TDX</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">Asia-Pacific</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">94.1%</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">156</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Recent Network Activity */}
      <div className="knirv-card-gradient rounded-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900 flex items-center">
            <Radio className="w-5 h-5 mr-2" />
            Live Network Activity
          </h3>
        </div>
        <div className="p-6">
          <div className="space-y-3 max-h-60 overflow-y-auto">
            {systemRealtime.messages.slice(0, 8).map((message, index) => (
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
                <p>Waiting for network activity...</p>
                <p className="text-sm">Real-time events will appear here</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Information Panel */}
      <div className="knirv-card-gradient p-6 rounded-lg bg-blue-50 border border-blue-200">
        <div className="flex items-start">
          <Eye className="w-6 h-6 text-blue-600 mt-1 mr-3" />
          <div>
            <h3 className="text-lg font-semibold text-blue-900 mb-2">Observer Mode</h3>
            <p className="text-blue-800 mb-2">
              You have read-only access to the KNIRV NEXUS network. You can monitor network health, 
              view node status, and observe validation activities in real-time.
            </p>
            <p className="text-sm text-blue-700">
              For administrative access or to become a validator, please contact the network administrators.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
