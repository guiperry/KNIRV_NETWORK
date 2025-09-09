import React, { useState } from 'react';
import { Search, Globe, Link, Database, Router, Layers } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface ExplorerTab {
  id: string;
  name: string;
  icon: React.ComponentType<any>;
  description: string;
}

const NetworkExplorers: React.FC = () => {
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState('graph');

  const explorers: ExplorerTab[] = [
    {
      id: 'graph',
      name: 'Graph',
      icon: Globe,
      description: 'Explore the KNIRVGRAPH network topology'
    },
    {
      id: 'chain',
      name: 'Chain',
      icon: Link,
      description: 'Browse KNIRVCHAIN transactions and blocks'
    },
    {
      id: 'oracle',
      name: 'Oracle',
      icon: Database,
      description: 'Query KNIRVORACLE data and services'
    },
    {
      id: 'router',
      name: 'Router',
      icon: Router,
      description: 'Analyze KNIRVROUTER network paths'
    },
    {
      id: 'nexus',
      name: 'Nexus',
      icon: Layers,
      description: 'Inspect KNIRVNEXUS development environments'
    }
  ];

  const renderExplorerContent = () => {
    switch (activeTab) {
      case 'graph':
        return (
          <div className="space-y-6">
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h3 className="text-lg font-semibold text-white mb-4">Graph Network Topology</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-purple-400">1,247</div>
                  <div className="text-sm text-slate-400">Total Nodes</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-blue-400">3,891</div>
                  <div className="text-sm text-slate-400">Connections</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-green-400">98.7%</div>
                  <div className="text-sm text-slate-400">Connectivity</div>
                </div>
              </div>
            </div>
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h4 className="text-md font-medium text-white mb-3">Recent Graph Events</h4>
              <div className="space-y-2">
                <div className="text-sm text-slate-300">• New skill node registered: error-handler-v2</div>
                <div className="text-sm text-slate-300">• Capability node updated: file-processor</div>
                <div className="text-sm text-slate-300">• Property node created: ai-model-weights</div>
              </div>
            </div>
          </div>
        );
      
      case 'chain':
        return (
          <div className="space-y-6">
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h3 className="text-lg font-semibold text-white mb-4">Blockchain Explorer</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-blue-400">15,432</div>
                  <div className="text-sm text-slate-400">Block Height</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-green-400">2.3s</div>
                  <div className="text-sm text-slate-400">Avg Block Time</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-purple-400">847</div>
                  <div className="text-sm text-slate-400">Pending Txs</div>
                </div>
              </div>
            </div>
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h4 className="text-md font-medium text-white mb-3">Latest Blocks</h4>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Block #15432</span>
                  <span className="text-slate-400">23 transactions</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Block #15431</span>
                  <span className="text-slate-400">18 transactions</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Block #15430</span>
                  <span className="text-slate-400">31 transactions</span>
                </div>
              </div>
            </div>
          </div>
        );
      
      case 'oracle':
        return (
          <div className="space-y-6">
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h3 className="text-lg font-semibold text-white mb-4">Oracle Services</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-yellow-400">12</div>
                  <div className="text-sm text-slate-400">Active Services</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-green-400">99.9%</div>
                  <div className="text-sm text-slate-400">Uptime</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-blue-400">1.2M</div>
                  <div className="text-sm text-slate-400">Queries/Day</div>
                </div>
              </div>
            </div>
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h4 className="text-md font-medium text-white mb-3">Service Status</h4>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Agent Registry</span>
                  <span className="text-green-400">Online</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Tunnel Registry</span>
                  <span className="text-green-400">Online</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Notary System</span>
                  <span className="text-green-400">Online</span>
                </div>
              </div>
            </div>
          </div>
        );
      
      case 'router':
        return (
          <div className="space-y-6">
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h3 className="text-lg font-semibold text-white mb-4">Network Routing</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-purple-400">2,847</div>
                  <div className="text-sm text-slate-400">Active Routes</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-green-400">45ms</div>
                  <div className="text-sm text-slate-400">Avg Latency</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-blue-400">98.2%</div>
                  <div className="text-sm text-slate-400">Success Rate</div>
                </div>
              </div>
            </div>
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h4 className="text-md font-medium text-white mb-3">Top Routes</h4>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Agent → Oracle</span>
                  <span className="text-slate-400">1,247 requests</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Client → Nexus</span>
                  <span className="text-slate-400">892 requests</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Router → Chain</span>
                  <span className="text-slate-400">634 requests</span>
                </div>
              </div>
            </div>
          </div>
        );
      
      case 'nexus':
        return (
          <div className="space-y-6">
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h3 className="text-lg font-semibold text-white mb-4">Development Environments</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-blue-400">47</div>
                  <div className="text-sm text-slate-400">Active DVEs</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-green-400">156</div>
                  <div className="text-sm text-slate-400">Total Projects</div>
                </div>
                <div className="bg-slate-800/50 rounded-lg p-4">
                  <div className="text-2xl font-bold text-purple-400">23</div>
                  <div className="text-sm text-slate-400">Active Users</div>
                </div>
              </div>
            </div>
            <div className="bg-slate-700/30 rounded-lg p-6">
              <h4 className="text-md font-medium text-white mb-3">Recent Activity</h4>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">New DVE created</span>
                  <span className="text-slate-400">5 minutes ago</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">Project deployed</span>
                  <span className="text-slate-400">12 minutes ago</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300">User session started</span>
                  <span className="text-slate-400">18 minutes ago</span>
                </div>
              </div>
            </div>
          </div>
        );
      
      default:
        return <div className="text-slate-400">Select an explorer to view details</div>;
    }
  };

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-purple-500/20 rounded-lg">
          <Search className="w-6 h-6 text-purple-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Network Explorers</h1>
          <p className="text-slate-400">Explore different aspects of the KNIRV network</p>
        </div>
      </div>

      {/* Explorer Tabs */}
      <div className="flex flex-wrap gap-2 mb-6">
        {explorers.map((explorer) => {
          const Icon = explorer.icon;
          return (
            <button
              key={explorer.id}
              onClick={() => setActiveTab(explorer.id)}
              className={`flex items-center space-x-2 px-4 py-2 rounded-lg transition-colors ${
                activeTab === explorer.id
                  ? 'bg-purple-600/30 text-purple-300 border border-purple-500/50'
                  : 'bg-slate-700/30 text-slate-400 hover:bg-slate-600/30 hover:text-slate-300'
              }`}
            >
              <Icon className="w-4 h-4" />
              <span className="font-medium">{explorer.name}</span>
            </button>
          );
        })}
      </div>

      {/* Explorer Description */}
      <div className="bg-slate-800/30 border border-slate-700/50 rounded-lg p-4 mb-6">
        <p className="text-slate-300">
          {explorers.find(e => e.id === activeTab)?.description}
        </p>
      </div>

      {/* Explorer Content */}
      <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-6">
        {renderExplorerContent()}
      </div>
    </div>
  );
};

export default NetworkExplorers;
