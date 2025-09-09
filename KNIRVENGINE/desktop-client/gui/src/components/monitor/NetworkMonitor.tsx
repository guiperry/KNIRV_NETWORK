import React, { useState, useEffect } from 'react';
import { Monitor, Activity, Wifi, Server, AlertTriangle, CheckCircle, Clock } from 'lucide-react';
import { useAuth } from '../AuthContext';

interface NetworkNode {
  id: string;
  name: string;
  type: 'root' | 'bootnode' | 'peer' | 'client';
  status: 'online' | 'offline' | 'warning';
  lastSeen: Date;
  connections: number;
  latency: number;
  version: string;
}

interface NetworkMetrics {
  totalNodes: number;
  activeNodes: number;
  totalConnections: number;
  avgLatency: number;
  networkHealth: number;
}

const NetworkMonitor: React.FC = () => {
  const { user } = useAuth();
  const [nodes, setNodes] = useState<NetworkNode[]>([]);
  const [metrics, setMetrics] = useState<NetworkMetrics>({
    totalNodes: 0,
    activeNodes: 0,
    totalConnections: 0,
    avgLatency: 0,
    networkHealth: 0
  });
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Simulate loading network data
    const loadNetworkData = async () => {
      setIsLoading(true);
      
      // Simulate API call delay
      await new Promise(resolve => setTimeout(resolve, 1500));
      
      const mockNodes: NetworkNode[] = [
        {
          id: 'root-1',
          name: 'KNIRV Root Node',
          type: 'root',
          status: 'online',
          lastSeen: new Date(),
          connections: 12,
          latency: 15,
          version: '1.0.0'
        },
        {
          id: 'bootnode-1',
          name: 'Bootnode Alpha',
          type: 'bootnode',
          status: 'online',
          lastSeen: new Date(Date.now() - 30000),
          connections: 8,
          latency: 25,
          version: '1.0.0'
        },
        {
          id: 'bootnode-2',
          name: 'Bootnode Beta',
          type: 'bootnode',
          status: 'warning',
          lastSeen: new Date(Date.now() - 120000),
          connections: 5,
          latency: 85,
          version: '0.9.8'
        },
        {
          id: 'peer-1',
          name: 'Peer Node 1',
          type: 'peer',
          status: 'online',
          lastSeen: new Date(Date.now() - 10000),
          connections: 3,
          latency: 45,
          version: '1.0.0'
        },
        {
          id: 'peer-2',
          name: 'Peer Node 2',
          type: 'peer',
          status: 'offline',
          lastSeen: new Date(Date.now() - 300000),
          connections: 0,
          latency: 0,
          version: '1.0.0'
        }
      ];

      setNodes(mockNodes);
      
      const activeNodes = mockNodes.filter(n => n.status === 'online').length;
      const totalConnections = mockNodes.reduce((sum, n) => sum + n.connections, 0);
      const avgLatency = mockNodes.filter(n => n.status === 'online')
        .reduce((sum, n) => sum + n.latency, 0) / activeNodes;
      
      setMetrics({
        totalNodes: mockNodes.length,
        activeNodes,
        totalConnections,
        avgLatency: Math.round(avgLatency),
        networkHealth: Math.round((activeNodes / mockNodes.length) * 100)
      });
      
      setIsLoading(false);
    };

    loadNetworkData();
    
    // Set up periodic refresh
    const interval = setInterval(loadNetworkData, 30000);
    return () => clearInterval(interval);
  }, []);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'warning':
        return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
      case 'offline':
        return <div className="w-4 h-4 bg-red-500 rounded-full" />;
      default:
        return <Clock className="w-4 h-4 text-slate-400" />;
    }
  };

  const getNodeTypeColor = (type: string) => {
    switch (type) {
      case 'root':
        return 'text-purple-400 bg-purple-500/20';
      case 'bootnode':
        return 'text-blue-400 bg-blue-500/20';
      case 'peer':
        return 'text-green-400 bg-green-500/20';
      case 'client':
        return 'text-slate-400 bg-slate-500/20';
      default:
        return 'text-slate-400 bg-slate-500/20';
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full bg-slate-900">
        <div className="text-center">
          <div className="animate-spin w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
          <p className="text-slate-400">Loading network data...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full bg-slate-900 p-6">
      {/* Header */}
      <div className="flex items-center space-x-3 mb-6">
        <div className="p-2 bg-blue-500/20 rounded-lg">
          <Monitor className="w-6 h-6 text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">Network Monitor</h1>
          <p className="text-slate-400">Real-time KNIRV network status</p>
        </div>
      </div>

      {/* Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Server className="w-5 h-5 text-blue-400" />
            <span className="text-sm text-slate-400">Total Nodes</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{metrics.totalNodes}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Activity className="w-5 h-5 text-green-400" />
            <span className="text-sm text-slate-400">Active Nodes</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{metrics.activeNodes}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Wifi className="w-5 h-5 text-purple-400" />
            <span className="text-sm text-slate-400">Connections</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{metrics.totalConnections}</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <Clock className="w-5 h-5 text-yellow-400" />
            <span className="text-sm text-slate-400">Avg Latency</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{metrics.avgLatency}ms</div>
        </div>
        
        <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg p-4">
          <div className="flex items-center space-x-2">
            <CheckCircle className="w-5 h-5 text-green-400" />
            <span className="text-sm text-slate-400">Health</span>
          </div>
          <div className="text-2xl font-bold text-white mt-2">{metrics.networkHealth}%</div>
        </div>
      </div>

      {/* Nodes Table */}
      <div className="bg-slate-800/50 border border-slate-700/50 rounded-lg overflow-hidden">
        <div className="p-4 border-b border-slate-700/50">
          <h2 className="text-lg font-semibold text-white">Network Nodes</h2>
        </div>
        
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-slate-700/30">
              <tr>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Node</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Type</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Status</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Connections</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Latency</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Last Seen</th>
                <th className="text-left p-4 text-sm font-medium text-slate-300">Version</th>
              </tr>
            </thead>
            <tbody>
              {nodes.map((node) => (
                <tr key={node.id} className="border-t border-slate-700/30 hover:bg-slate-700/20">
                  <td className="p-4">
                    <div className="text-white font-medium">{node.name}</div>
                    <div className="text-xs text-slate-400">{node.id}</div>
                  </td>
                  <td className="p-4">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${getNodeTypeColor(node.type)}`}>
                      {node.type}
                    </span>
                  </td>
                  <td className="p-4">
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(node.status)}
                      <span className="text-sm text-white capitalize">{node.status}</span>
                    </div>
                  </td>
                  <td className="p-4 text-white">{node.connections}</td>
                  <td className="p-4 text-white">{node.latency}ms</td>
                  <td className="p-4 text-slate-400 text-sm">
                    {node.lastSeen.toLocaleTimeString()}
                  </td>
                  <td className="p-4 text-slate-400 text-sm">{node.version}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default NetworkMonitor;
