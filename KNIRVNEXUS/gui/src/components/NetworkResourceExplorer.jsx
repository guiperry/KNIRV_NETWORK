import React, { useState, useEffect } from 'react';
import {
  Globe,
  Server,
  MapPin,
  Cpu,
  Database,
  HardDrive,
  Wifi,
  Shield,
  Zap,
  Clock,
  Filter,
  Search,
  RefreshCw,
  CheckCircle,
  AlertTriangle,
  Activity
} from 'lucide-react';

const NetworkResourceExplorer = () => {
  const [nodes, setNodes] = useState([]);
  const [selectedNode, setSelectedNode] = useState(null);
  const [filterRegion, setFilterRegion] = useState('all');
  const [filterTeeType, setFilterTeeType] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(false);
  const [viewMode, setViewMode] = useState('list'); // 'list' or 'map'

  // Mock network nodes data
  useEffect(() => {
    const mockNodes = [
      {
        id: 'node-001',
        name: 'KNIRV-SGX-US-East-1',
        region: 'US East',
        location: 'Virginia, USA',
        coordinates: { lat: 38.13, lng: -78.45 },
        teeType: 'Intel SGX',
        status: 'online',
        load: 68,
        capacity: {
          cpu: { used: 68, total: 100 },
          memory: { used: 45, total: 64 },
          storage: { used: 120, total: 500 },
          teeSlots: { used: 12, total: 16 }
        },
        specs: {
          cpuCores: 32,
          memoryGB: 64,
          storageGB: 500,
          teeVersion: 'SGX v2.0',
          networkSpeed: '10 Gbps'
        },
        uptime: '99.8%',
        latency: '12ms',
        activeTasks: 8
      },
      {
        id: 'node-002',
        name: 'KNIRV-SEV-EU-West-1',
        region: 'EU West',
        location: 'Frankfurt, Germany',
        coordinates: { lat: 50.11, lng: 8.68 },
        teeType: 'AMD SEV-SNP',
        status: 'online',
        load: 45,
        capacity: {
          cpu: { used: 45, total: 100 },
          memory: { used: 28, total: 128 },
          storage: { used: 200, total: 1000 },
          teeSlots: { used: 6, total: 24 }
        },
        specs: {
          cpuCores: 64,
          memoryGB: 128,
          storageGB: 1000,
          teeVersion: 'SEV-SNP v1.0',
          networkSpeed: '25 Gbps'
        },
        uptime: '99.9%',
        latency: '8ms',
        activeTasks: 12
      },
      {
        id: 'node-003',
        name: 'KNIRV-TDX-ASIA-1',
        region: 'Asia Pacific',
        location: 'Tokyo, Japan',
        coordinates: { lat: 35.68, lng: 139.69 },
        teeType: 'Intel TDX',
        status: 'maintenance',
        load: 0,
        capacity: {
          cpu: { used: 0, total: 100 },
          memory: { used: 0, total: 256 },
          storage: { used: 50, total: 2000 },
          teeSlots: { used: 0, total: 32 }
        },
        specs: {
          cpuCores: 128,
          memoryGB: 256,
          storageGB: 2000,
          teeVersion: 'TDX v1.5',
          networkSpeed: '100 Gbps'
        },
        uptime: '99.5%',
        latency: '25ms',
        activeTasks: 0
      }
    ];
    setNodes(mockNodes);
  }, []);

  const getStatusColor = (status) => {
    switch (status) {
      case 'online':
        return 'text-green-400';
      case 'maintenance':
        return 'text-yellow-400';
      case 'offline':
        return 'text-red-400';
      default:
        return 'text-gray-400';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'online':
        return <CheckCircle className="w-4 h-4 text-green-400" />;
      case 'maintenance':
        return <Clock className="w-4 h-4 text-yellow-400" />;
      case 'offline':
        return <AlertTriangle className="w-4 h-4 text-red-400" />;
      default:
        return <Activity className="w-4 h-4 text-gray-400" />;
    }
  };

  const getLoadColor = (load) => {
    if (load < 50) return 'from-green-500 to-emerald-500';
    if (load < 80) return 'from-yellow-500 to-orange-500';
    return 'from-red-500 to-pink-500';
  };

  const filteredNodes = nodes.filter(node => {
    const matchesRegion = filterRegion === 'all' || node.region === filterRegion;
    const matchesTeeType = filterTeeType === 'all' || node.teeType === filterTeeType;
    const matchesSearch = node.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         node.location.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesRegion && matchesTeeType && matchesSearch;
  });

  const ResourceBar = ({ label, used, total, unit = '', color = 'from-knirv-primary to-knirv-secondary' }) => (
    <div className="mb-3">
      <div className="flex justify-between text-sm mb-1">
        <span className="text-slate-300">{label}</span>
        <span className="text-white">{used}{unit} / {total}{unit}</span>
      </div>
      <div className="w-full bg-slate-600 rounded-full h-2">
        <div
          className={`h-2 rounded-full bg-gradient-to-r ${color}`}
          style={{ width: `${(used / total) * 100}%` }}
        />
      </div>
    </div>
  );

  return (
    <div className="min-h-screen bg-knirv-gradient p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center space-x-3">
            <Globe className="w-8 h-8 text-knirv-primary" />
            <div>
              <h1 className="text-3xl font-bold text-white">Network Status & Resource Explorer</h1>
              <p className="text-slate-300">Explore the decentralized KNIRV-NEXUS network topology and resources</p>
            </div>
          </div>
          <div className="flex items-center space-x-4">
            <div className="flex bg-slate-800 rounded-lg border border-slate-600">
              <button
                onClick={() => setViewMode('list')}
                className={`px-4 py-2 rounded-l-lg transition-colors ${
                  viewMode === 'list' ? 'bg-knirv-primary text-white' : 'text-slate-300 hover:text-white'
                }`}
              >
                List
              </button>
              <button
                onClick={() => setViewMode('map')}
                className={`px-4 py-2 rounded-r-lg transition-colors ${
                  viewMode === 'map' ? 'bg-knirv-primary text-white' : 'text-slate-300 hover:text-white'
                }`}
              >
                Map
              </button>
            </div>
            <button
              onClick={() => setLoading(true)}
              className="flex items-center space-x-2 px-4 py-2 bg-knirv-primary text-white rounded-lg hover:bg-knirv-secondary transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          </div>
        </div>

        {/* Filters */}
        <div className="bg-slate-800 rounded-lg p-4 mb-6 border border-slate-700">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <Filter className="w-4 h-4 text-slate-400" />
              <select
                value={filterRegion}
                onChange={(e) => setFilterRegion(e.target.value)}
                className="bg-slate-700 text-white rounded-lg px-3 py-2 border border-slate-600"
              >
                <option value="all">All Regions</option>
                <option value="US East">US East</option>
                <option value="EU West">EU West</option>
                <option value="Asia Pacific">Asia Pacific</option>
              </select>
            </div>
            <div className="flex items-center space-x-2">
              <Shield className="w-4 h-4 text-slate-400" />
              <select
                value={filterTeeType}
                onChange={(e) => setFilterTeeType(e.target.value)}
                className="bg-slate-700 text-white rounded-lg px-3 py-2 border border-slate-600"
              >
                <option value="all">All TEE Types</option>
                <option value="Intel SGX">Intel SGX</option>
                <option value="AMD SEV-SNP">AMD SEV-SNP</option>
                <option value="Intel TDX">Intel TDX</option>
              </select>
            </div>
            <div className="flex items-center space-x-2 flex-1">
              <Search className="w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="Search nodes by name or location..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="bg-slate-700 text-white rounded-lg px-3 py-2 border border-slate-600 flex-1"
              />
            </div>
          </div>
        </div>

        {/* Network Overview Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
            <div className="flex items-center space-x-3 mb-2">
              <Server className="w-6 h-6 text-knirv-primary" />
              <h3 className="text-lg font-semibold text-white">Total Nodes</h3>
            </div>
            <p className="text-3xl font-bold text-white">{nodes.length}</p>
            <p className="text-sm text-green-400">+2 this week</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
            <div className="flex items-center space-x-3 mb-2">
              <Activity className="w-6 h-6 text-green-400" />
              <h3 className="text-lg font-semibold text-white">Online Nodes</h3>
            </div>
            <p className="text-3xl font-bold text-white">{nodes.filter(n => n.status === 'online').length}</p>
            <p className="text-sm text-green-400">99.2% uptime</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
            <div className="flex items-center space-x-3 mb-2">
              <Zap className="w-6 h-6 text-yellow-400" />
              <h3 className="text-lg font-semibold text-white">Active Tasks</h3>
            </div>
            <p className="text-3xl font-bold text-white">{nodes.reduce((sum, n) => sum + n.activeTasks, 0)}</p>
            <p className="text-sm text-green-400">+15% from yesterday</p>
          </div>
          <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
            <div className="flex items-center space-x-3 mb-2">
              <Globe className="w-6 h-6 text-blue-400" />
              <h3 className="text-lg font-semibold text-white">Avg Latency</h3>
            </div>
            <p className="text-3xl font-bold text-white">15ms</p>
            <p className="text-sm text-green-400">-3ms improvement</p>
          </div>
        </div>

        {/* Nodes List/Map */}
        {viewMode === 'list' ? (
          <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
            {filteredNodes.map((node) => (
              <div
                key={node.id}
                className="bg-slate-800 rounded-lg border border-slate-700 hover:border-knirv-primary transition-colors cursor-pointer"
                onClick={() => setSelectedNode(node)}
              >
                <div className="p-6">
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(node.status)}
                      <h3 className="text-lg font-semibold text-white">{node.name}</h3>
                    </div>
                    <span className={`px-2 py-1 rounded text-xs ${getStatusColor(node.status)} bg-slate-700`}>
                      {node.status}
                    </span>
                  </div>
                  
                  <div className="space-y-2 mb-4">
                    <div className="flex items-center space-x-2 text-sm text-slate-300">
                      <MapPin className="w-4 h-4" />
                      <span>{node.location}</span>
                    </div>
                    <div className="flex items-center space-x-2 text-sm text-slate-300">
                      <Shield className="w-4 h-4" />
                      <span>{node.teeType}</span>
                    </div>
                    <div className="flex items-center space-x-2 text-sm text-slate-300">
                      <Activity className="w-4 h-4" />
                      <span>{node.activeTasks} active tasks</span>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span className="text-slate-300">Overall Load</span>
                      <span className="text-white">{node.load}%</span>
                    </div>
                    <div className="w-full bg-slate-600 rounded-full h-2">
                      <div
                        className={`h-2 rounded-full bg-gradient-to-r ${getLoadColor(node.load)}`}
                        style={{ width: `${node.load}%` }}
                      />
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
            <div className="h-96 flex items-center justify-center text-slate-400">
              <div className="text-center">
                <Globe className="w-16 h-16 mx-auto mb-4" />
                <p>Interactive network map coming soon</p>
                <p className="text-sm">Visualize global KNIRV-NEXUS node distribution</p>
              </div>
            </div>
          </div>
        )}

        {/* Node Detail Modal */}
        {selectedNode && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-slate-800 rounded-lg p-6 max-w-4xl w-full mx-4 border border-slate-700 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center space-x-3">
                  {getStatusIcon(selectedNode.status)}
                  <h3 className="text-2xl font-semibold text-white">{selectedNode.name}</h3>
                </div>
                <button
                  onClick={() => setSelectedNode(null)}
                  className="text-slate-400 hover:text-white text-2xl"
                >
                  ×
                </button>
              </div>

              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div>
                  <h4 className="text-lg font-semibold text-white mb-4">Node Information</h4>
                  <div className="space-y-3 text-sm">
                    <div className="flex justify-between">
                      <span className="text-slate-300">Location:</span>
                      <span className="text-white">{selectedNode.location}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-300">TEE Type:</span>
                      <span className="text-white">{selectedNode.teeType}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-300">Uptime:</span>
                      <span className="text-white">{selectedNode.uptime}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-300">Latency:</span>
                      <span className="text-white">{selectedNode.latency}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-300">Network Speed:</span>
                      <span className="text-white">{selectedNode.specs.networkSpeed}</span>
                    </div>
                  </div>
                </div>

                <div>
                  <h4 className="text-lg font-semibold text-white mb-4">Resource Utilization</h4>
                  <ResourceBar
                    label="CPU"
                    used={selectedNode.capacity.cpu.used}
                    total={selectedNode.capacity.cpu.total}
                    unit="%"
                    color="from-blue-500 to-cyan-500"
                  />
                  <ResourceBar
                    label="Memory"
                    used={selectedNode.capacity.memory.used}
                    total={selectedNode.capacity.memory.total}
                    unit=" GB"
                    color="from-green-500 to-emerald-500"
                  />
                  <ResourceBar
                    label="Storage"
                    used={selectedNode.capacity.storage.used}
                    total={selectedNode.capacity.storage.total}
                    unit=" GB"
                    color="from-yellow-500 to-orange-500"
                  />
                  <ResourceBar
                    label="TEE Slots"
                    used={selectedNode.capacity.teeSlots.used}
                    total={selectedNode.capacity.teeSlots.total}
                    color="from-knirv-primary to-knirv-secondary"
                  />
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default NetworkResourceExplorer;
