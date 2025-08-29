import React, { useState, useEffect } from 'react';
import {
  Search,
  Filter,
  Plus,
  Zap,
  Server,
  Download,
  Settings,
  RefreshCw,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Eye,
  Code,
  Database,
  Brain,
  Globe,
  Shield,
  Terminal,
  Cpu
} from 'lucide-react';
import MCPServerModal from './modals/MCPServerModal';
import MCPCapabilityModal from './modals/MCPCapabilityModal';
import AdvancedFilterModal from './modals/AdvancedFilterModal';
import { fetchMCPServers } from '../utils/api';

export const MCPCapabilityManager = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [isServerModalOpen, setIsServerModalOpen] = useState(false);
  const [isCapabilityModalOpen, setIsCapabilityModalOpen] = useState(false);
  const [isFilterModalOpen, setIsFilterModalOpen] = useState(false);
  const [selectedServer, setSelectedServer] = useState(null);
  const [selectedCapability, setSelectedCapability] = useState(null);
  const [refreshing, setRefreshing] = useState(false);
  const [activeFilters, setActiveFilters] = useState([]);

  const categories = [
    { id: 'all', name: 'All Servers', icon: Server },
    { id: 'llm', name: 'LLM Providers', icon: Brain },
    { id: 'tool', name: 'Tool Providers', icon: Terminal },
    { id: 'data', name: 'Data Providers', icon: Database },
    { id: 'web', name: 'Web Services', icon: Globe },
    { id: 'security', name: 'Security Services', icon: Shield },
    { id: 'compute', name: 'Compute Services', icon: Cpu }
  ];

  const [mcpServers, setMcpServers] = useState([]);
  const [loading, setLoading] = useState(true);

  const [capabilities, setCapabilities] = useState([]);
  const [transformingServers, setTransformingServers] = useState(new Set());
  const [recentlyTransformed, setRecentlyTransformed] = useState([]);

  // Fetch installed MCP servers on component mount
  useEffect(() => {
    const fetchInstalledServers = async () => {
      try {
        setLoading(true);
        const data = await fetchMCPServers();

        // Filter only installed servers and transform them for the capability manager
        const installedServers = data.servers
          .filter(server => server.status === 'installed')
          .map(server => ({
            id: server.id,
            name: server.name,
            type: server.category || 'general',
            status: 'disconnected', // Initially disconnected until activated
            version: server.version || '1.0.0',
            lastSync: 'Not synced',
            description: server.description,
            endpoint: `http://localhost:8081/mcp/${server.id}`,
            capabilities: [
              // Default capability based on server type
              {
                id: `${server.id}-main`,
                name: `${server.name} Main`,
                type: server.type === 'typescript' ? 'tool' : 'resource',
                status: 'inactive'
              }
            ],
            health: {
              status: 'offline',
              uptime: '0%',
              latency: 'N/A',
              lastChecked: 'Never'
            },
            icon: getCategoryIcon(server.category || 'general')
          }));

        setMcpServers(installedServers);
      } catch (error) {
        console.error('Failed to fetch installed MCP servers:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchInstalledServers();
  }, []);

  // Function to simulate MCP server installation and transformation
  const simulateServerInstallation = async (serverId, serverName) => {
    // Add server to transforming state
    setTransformingServers(prev => new Set([...prev, serverId]));

    try {
      // Simulate installation process
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Create capability from MCP server
      const newCapability = {
        id: `mcp-${serverId}`,
        name: serverName,
        provider: 'MCP Server',
        type: 'mcp_capability',
        status: 'available',
        serverId: serverId,
        description: `${serverName} capability via MCP Server`,
        category: 'mcp',
        transformedAt: Date.now()
      };

      // Add to recently transformed for animation
      setRecentlyTransformed(prev => [...prev, newCapability]);

      // Add to capabilities list
      setCapabilities(prev => [...prev, newCapability]);

      // Remove from transforming state
      setTransformingServers(prev => {
        const newSet = new Set(prev);
        newSet.delete(serverId);
        return newSet;
      });

      // Remove from recently transformed after animation
      setTimeout(() => {
        setRecentlyTransformed(prev => prev.filter(cap => cap.id !== newCapability.id));
      }, 3000);

    } catch (error) {
      console.error('Failed to install server:', error);
      setTransformingServers(prev => {
        const newSet = new Set(prev);
        newSet.delete(serverId);
        return newSet;
      });
    }
  };

  // Filter options for the advanced filter modal
  const filterOptions = {
    fields: [
      { id: 'name', name: 'Name' },
      { id: 'type', name: 'Type' },
      { id: 'status', name: 'Status' },
      { id: 'version', name: 'Version' },
      { id: 'endpoint', name: 'Endpoint' },
      { id: 'health.status', name: 'Health Status' },
      { id: 'health.uptime', name: 'Uptime' },
      { id: 'health.latency', name: 'Latency' }
    ],
    operators: [
      { id: 'equals', name: 'Equals' },
      { id: 'not_equals', name: 'Not Equals' },
      { id: 'contains', name: 'Contains' },
      { id: 'not_contains', name: 'Not Contains' },
      { id: 'greater_than', name: 'Greater Than' },
      { id: 'less_than', name: 'Less Than' },
      { id: 'starts_with', name: 'Starts With' },
      { id: 'ends_with', name: 'Ends With' }
    ]
  };

  // Apply advanced filters
  const applyAdvancedFilters = (filters) => {
    setActiveFilters(filters);
  };

  // Check if a server matches the advanced filters
  const matchesAdvancedFilters = (server) => {
    if (activeFilters.length === 0) return true;
    
    return activeFilters.every(filter => {
      const { field, operator, value } = filter;
      
      // Handle nested fields like health.status
      let serverValue;
      if (field.includes('.')) {
        const [parent, child] = field.split('.');
        serverValue = server[parent] ? server[parent][child] : undefined;
      } else {
        serverValue = server[field];
      }
      
      // Handle null/undefined values
      if (serverValue === null || serverValue === undefined) {
        return operator === 'not_equals' || operator === 'not_contains';
      }
      
      // Convert to string for comparison
      const serverValueStr = String(serverValue).toLowerCase();
      const filterValueStr = String(value).toLowerCase();
      
      switch (operator) {
        case 'equals':
          return serverValueStr === filterValueStr;
        case 'not_equals':
          return serverValueStr !== filterValueStr;
        case 'contains':
          return serverValueStr.includes(filterValueStr);
        case 'not_contains':
          return !serverValueStr.includes(filterValueStr);
        case 'greater_than':
          return parseFloat(serverValueStr) > parseFloat(filterValueStr);
        case 'less_than':
          return parseFloat(serverValueStr) < parseFloat(filterValueStr);
        case 'starts_with':
          return serverValueStr.startsWith(filterValueStr);
        case 'ends_with':
          return serverValueStr.endsWith(filterValueStr);
        default:
          return true;
      }
    });
  };

  // Flatten all capabilities from all servers
  useEffect(() => {
    const allCapabilities = mcpServers.flatMap(server => 
      server.capabilities.map(cap => ({
        ...cap,
        serverId: server.id,
        serverName: server.name,
        serverStatus: server.status,
        serverType: server.type
      }))
    );
    setCapabilities(allCapabilities);
  }, [mcpServers]);

  const getStatusColor = (status) => {
    switch (status) {
      case 'connected':
        return 'text-green-400 bg-green-500/20 border-green-500/30';
      case 'warning':
        return 'text-yellow-400 bg-yellow-500/20 border-yellow-500/30';
      case 'disconnected':
        return 'text-red-400 bg-red-500/20 border-red-500/30';
      default:
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
    }
  };

  const getCapabilityStatusColor = (status) => {
    switch (status) {
      case 'active':
        return 'text-green-400 bg-green-500/20 border-green-500/30';
      case 'warning':
        return 'text-yellow-400 bg-yellow-500/20 border-yellow-500/30';
      case 'inactive':
        return 'text-red-400 bg-red-500/20 border-red-500/30';
      default:
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
    }
  };

  const getHealthStatusIcon = (health) => {
    switch (health.status) {
      case 'healthy':
        return <CheckCircle className="w-4 h-4 text-green-400" />;
      case 'degraded':
        return <AlertTriangle className="w-4 h-4 text-yellow-400" />;
      case 'offline':
        return <XCircle className="w-4 h-4 text-red-400" />;
      default:
        return <AlertTriangle className="w-4 h-4 text-slate-400" />;
    }
  };

  const getCategoryIcon = (categoryId) => {
    const category = categories.find(cat => cat.id === categoryId);
    return category ? category.icon : Server;
  };

  const filteredServers = mcpServers.filter(server => {
    const matchesSearch = server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         server.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || server.type === selectedCategory;
    const matchesAdvanced = matchesAdvancedFilters(server);
    return matchesSearch && matchesCategory && matchesAdvanced;
  });

  const handleAddServer = () => {
    setSelectedServer(null);
    setIsServerModalOpen(true);
  };

  const handleEditServer = (server) => {
    setSelectedServer(server);
    setIsServerModalOpen(true);
  };

  const handleServerSaved = (serverData) => {
    if (selectedServer) {
      // Update existing server
      setMcpServers(prevServers => 
        prevServers.map(server => 
          server.id === serverData.id ? serverData : server
        )
      );
    } else {
      // Add new server
      setMcpServers(prevServers => [...prevServers, serverData]);
    }
    setIsServerModalOpen(false);
    setSelectedServer(null);
  };

  const handleAddCapability = (serverId) => {
    const server = mcpServers.find(s => s.id === serverId);
    setSelectedServer(server);
    setSelectedCapability(null);
    setIsCapabilityModalOpen(true);
  };

  const handleEditCapability = (capability) => {
    const server = mcpServers.find(s => s.id === capability.serverId);
    setSelectedServer(server);
    setSelectedCapability(capability);
    setIsCapabilityModalOpen(true);
  };

  const handleCapabilitySaved = (capabilityData) => {
    if (selectedServer) {
      if (selectedCapability) {
        // Update existing capability
        setMcpServers(prevServers => 
          prevServers.map(server => 
            server.id === selectedServer.id 
              ? {
                  ...server,
                  capabilities: server.capabilities.map(cap => 
                    cap.id === capabilityData.id ? capabilityData : cap
                  )
                }
              : server
          )
        );
      } else {
        // Add new capability
        setMcpServers(prevServers => 
          prevServers.map(server => 
            server.id === selectedServer.id 
              ? {
                  ...server,
                  capabilities: [...server.capabilities, capabilityData]
                }
              : server
          )
        );
      }
    }
    setIsCapabilityModalOpen(false);
    setSelectedServer(null);
    setSelectedCapability(null);
  };

  const handleActivateDeactivate = async (server) => {
    const newStatus = server.status === 'connected' || server.status === 'warning'
      ? 'disconnected'
      : 'connected';

    setRefreshing(true);

    try {
      // Make real API call to activate/deactivate server
      const endpoint = newStatus === 'connected'
        ? `/api/v1/mcp/servers/${server.id}/start`
        : `/api/v1/mcp/servers/${server.id}/stop`;

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({})
      });

      if (!response.ok) {
        throw new Error(`Failed to ${newStatus === 'connected' ? 'activate' : 'deactivate'} server`);
      }

      // Update server status on success
      setMcpServers(prevServers =>
        prevServers.map(s =>
          s.id === server.id
            ? {
                ...s,
                status: newStatus,
                lastSync: 'just now',
                health: {
                  ...s.health,
                  status: newStatus === 'connected' ? 'healthy' : 'offline',
                  lastChecked: 'just now'
                },
                capabilities: s.capabilities.map(cap => ({
                  ...cap,
                  status: newStatus === 'connected' ? 'active' : 'inactive'
                }))
              }
            : s
        )
      );
    } catch (error) {
      console.error('Failed to update server status:', error);
      // Show error message to user
      alert(`Failed to ${newStatus === 'connected' ? 'activate' : 'deactivate'} server: ${error.message}`);
    } finally {
      setRefreshing(false);
    }
  };

  const handleRefreshServer = async (server) => {
    setRefreshing(true);

    try {
      // Make real API call to refresh server status
      const response = await fetch(`/api/v1/mcp/servers/${server.id}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        throw new Error('Failed to refresh server status');
      }

      const data = await response.json();
      const updatedServer = data.server;

      // Update server with fresh data
      setMcpServers(prevServers =>
        prevServers.map(s =>
          s.id === server.id
            ? {
                ...s,
                ...updatedServer,
                lastSync: 'just now',
                health: {
                  ...s.health,
                  lastChecked: 'just now'
                }
              }
            : s
        )
      );
    } catch (error) {
      console.error('Failed to refresh server:', error);
      // Fallback to just updating timestamps
      setMcpServers(prevServers =>
        prevServers.map(s =>
          s.id === server.id
            ? {
                ...s,
                lastSync: 'just now',
                health: {
                  ...s.health,
                  lastChecked: 'just now'
                }
              }
            : s
        )
      );
    } finally {
      setRefreshing(false);
    }
  };

  // Format the active filters for display
  const getActiveFiltersDisplay = () => {
    if (activeFilters.length === 0) return null;
    
    return (
      <div className="flex items-center space-x-2 text-sm text-slate-400">
        <span>Active filters:</span>
        <div className="flex flex-wrap gap-2">
          {activeFilters.map((filter, index) => {
            const fieldName = filterOptions.fields.find(f => f.id === filter.field)?.name || filter.field;
            const operatorName = filterOptions.operators.find(o => o.id === filter.operator)?.name || filter.operator;
            
            return (
              <span 
                key={filter.id} 
                className="px-2 py-1 bg-purple-500/20 text-purple-300 text-xs rounded-full border border-purple-500/30"
              >
                {fieldName} {operatorName} {filter.value}
              </span>
            );
          })}
          <button 
            onClick={() => setActiveFilters([])}
            className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full hover:bg-slate-700 transition-colors duration-200"
          >
            Clear
          </button>
        </div>
      </div>
    );
  };

  return (
    <div className="p-6 space-y-6">
      {/* Transformation Animation Overlay */}
      {recentlyTransformed.length > 0 && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <div className="bg-slate-800 rounded-xl p-8 border border-slate-700 max-w-md w-full mx-4">
            <div className="text-center">
              <div className="mb-6">
                <div className="relative">
                  {/* MCP Server Icon */}
                  <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-r from-blue-500/20 to-purple-500/20 rounded-lg border border-blue-500/30 mb-4">
                    <Server className="w-8 h-8 text-blue-400" />
                  </div>

                  {/* Transformation Arrow */}
                  <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2">
                    <div className="flex items-center space-x-2 animate-pulse">
                      <div className="w-8 h-0.5 bg-gradient-to-r from-blue-400 to-purple-400"></div>
                      <div className="w-0 h-0 border-l-4 border-l-purple-400 border-t-2 border-t-transparent border-b-2 border-b-transparent"></div>
                    </div>
                  </div>

                  {/* Capability Icon */}
                  <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-r from-purple-500/20 to-pink-500/20 rounded-lg border border-purple-500/30 ml-8 animate-bounce">
                    <Zap className="w-8 h-8 text-purple-400" />
                  </div>
                </div>
              </div>

              <h3 className="text-xl font-bold text-white mb-2">MCP Server Transformed!</h3>
              <p className="text-slate-400 mb-4">
                {recentlyTransformed[0]?.name} has been successfully transformed into a capability.
              </p>

              <div className="flex items-center justify-center space-x-2 text-sm text-green-400">
                <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                <span>Ready to use in your agents</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">MCP Server Manager</h1>
          <p className="text-slate-400">Manage Model Context Protocol servers and their capabilities.</p>
        </div>
        <div className="mt-4 lg:mt-0 flex items-center space-x-3">
          <button
            onClick={() => simulateServerInstallation('demo-server', 'Demo MCP Server')}
            className="bg-gradient-to-r from-green-500 to-emerald-500 text-white px-4 py-3 rounded-lg font-medium hover:from-green-600 hover:to-emerald-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Zap className="w-5 h-5" />
            <span>Demo Transform</span>
          </button>
          <button
            onClick={handleAddServer}
            className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Plus className="w-5 h-5" />
            <span>Add MCP Server</span>
          </button>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col lg:flex-row lg:items-center space-y-4 lg:space-y-0 lg:space-x-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
          <input
            type="text"
            placeholder="Search MCP servers..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full bg-slate-800/50 border border-slate-700/50 rounded-lg pl-10 pr-4 py-3 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
        </div>
        <button 
          onClick={() => setIsFilterModalOpen(true)}
          className="bg-slate-800/50 border border-slate-700/50 rounded-lg px-4 py-3 text-slate-300 hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2"
        >
          <Filter className="w-5 h-5" />
          <span>Filter</span>
        </button>
      </div>

      {/* Active Filters Display */}
      {getActiveFiltersDisplay()}

      {/* Categories */}
      <div className="flex flex-wrap gap-3">
        {categories.map((category) => {
          const Icon = category.icon;
          return (
            <button
              key={category.id}
              onClick={() => setSelectedCategory(category.id)}
              className={`flex items-center space-x-2 px-4 py-2 rounded-lg transition-all duration-200 ${
                selectedCategory === category.id
                  ? 'bg-gradient-to-r from-purple-500/20 to-blue-500/20 text-white border border-purple-500/30'
                  : 'bg-slate-800/50 text-slate-300 border border-slate-700/50 hover:text-white hover:border-slate-600/50'
              }`}
            >
              <Icon className="w-4 h-4" />
              <span className="font-medium">{category.name}</span>
            </button>
          );
        })}
      </div>

      {/* MCP Servers Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {filteredServers.map((server) => {
          const ServerIcon = server.icon;
          const isTransforming = transformingServers.has(server.id);
          return (
            <div key={server.id} data-testid="server-card" className={`bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border transition-all duration-300 ${
              isTransforming
                ? 'border-green-500/50 bg-green-500/10 animate-pulse'
                : 'border-slate-700/50 hover:border-slate-600/50'
            }`} role="article">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-3">
                  <div className="p-3 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
                    <ServerIcon className="w-6 h-6 text-purple-400" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-white">{server.name}</h3>
                    <div className="flex items-center space-x-2">
                      <span className={`px-3 py-1 rounded-full text-xs font-medium border flex items-center space-x-1 ${getStatusColor(server.status)}`}>
                        {server.status === 'connected' ? (
                          <CheckCircle className="w-3 h-3" />
                        ) : server.status === 'warning' ? (
                          <AlertTriangle className="w-3 h-3" />
                        ) : (
                          <XCircle className="w-3 h-3" />
                        )}
                        <span className="capitalize">{server.status}</span>
                      </span>
                      <span className="text-slate-400 text-xs">v{server.version}</span>
                      {isTransforming && (
                        <span className="px-2 py-1 rounded-full text-xs font-medium bg-green-500/20 text-green-400 border border-green-500/30 flex items-center space-x-1">
                          <Zap className="w-3 h-3 animate-pulse" />
                          <span>Transforming...</span>
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  {server.status === 'connected' && (
                    <button 
                      onClick={() => handleRefreshServer(server)}
                      disabled={refreshing}
                      className="p-2 text-slate-400 hover:text-white transition-colors duration-200 disabled:opacity-50"
                    >
                      <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
                    </button>
                  )}
                  <button 
                    onClick={() => handleEditServer(server)}
                    className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
                  >
                    <Settings className="w-4 h-4" />
                  </button>
                </div>
              </div>
              
              <p className="text-slate-300 text-sm mb-4">{server.description}</p>
              
              <div className="grid grid-cols-2 gap-4 mb-4">
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate-400">Endpoint:</span>
                    <span className="text-white font-mono text-xs">{server.endpoint}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate-400">Last Sync:</span>
                    <span className="text-white">{server.lastSync}</span>
                  </div>
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate-400">Health:</span>
                    <div className="flex items-center space-x-1">
                      {getHealthStatusIcon(server.health)}
                      <span className={`font-medium capitalize ${
                        server.health.status === 'healthy' ? 'text-green-400' :
                        server.health.status === 'degraded' ? 'text-yellow-400' :
                        'text-red-400'
                      }`}>
                        {server.health.status}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate-400">Latency:</span>
                    <span className={`text-white ${
                      server.health.latencyStatus === 'high' ? 'text-yellow-400' :
                      server.health.status === 'offline' ? 'text-slate-500' :
                      'text-white'
                    }`}>
                      {server.health.latency}
                    </span>
                  </div>
                </div>
              </div>
              
              <div className="mb-4">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="text-white font-medium">Capabilities</h4>
                  {server.status === 'connected' && (
                    <button 
                      onClick={() => handleAddCapability(server.id)}
                      className="text-purple-400 hover:text-purple-300 text-xs font-medium flex items-center space-x-1"
                    >
                      <Plus className="w-3 h-3" />
                      <span>Add</span>
                    </button>
                  )}
                </div>
                <div className="space-y-2">
                  {server.capabilities.length > 0 ? (
                    server.capabilities.map((capability) => (
                      <div 
                        key={capability.id}
                        className="flex items-center justify-between p-2 bg-slate-700/30 rounded-lg border border-slate-600/30 hover:bg-slate-700/50 transition-colors duration-200 cursor-pointer"
                        onClick={() => handleEditCapability(capability)}
                      >
                        <div className="flex items-center space-x-2">
                          <Zap className="w-4 h-4 text-purple-400" />
                          <span className="text-white text-sm">{capability.name}</span>
                        </div>
                        <div className="flex items-center space-x-2">
                          <span className={`px-2 py-0.5 rounded-full text-xs ${getCapabilityStatusColor(capability.status)}`}>
                            {capability.status}
                          </span>
                          <Eye className="w-4 h-4 text-slate-400 hover:text-white" />
                        </div>
                      </div>
                    ))
                  ) : (
                    <div className="text-center py-2 text-slate-500 text-sm">
                      No capabilities available
                    </div>
                  )}
                </div>
              </div>
              
              <div className="flex items-center space-x-3">
                {server.status === 'disconnected' ? (
                  <button
                    onClick={() => handleActivateDeactivate(server)}
                    disabled={refreshing}
                    className="flex-1 bg-gradient-to-r from-green-500/20 to-emerald-500/20 hover:from-green-500/30 hover:to-emerald-500/30 border border-green-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2 disabled:opacity-50"
                  >
                    {refreshing ? (
                      <>
                        <RefreshCw className="w-4 h-4 animate-spin" />
                        <span>Activating...</span>
                      </>
                    ) : (
                      <>
                        <CheckCircle className="w-4 h-4" />
                        <span>Activate</span>
                      </>
                    )}
                  </button>
                ) : (
                  <button
                    onClick={() => handleActivateDeactivate(server)}
                    disabled={refreshing}
                    className="flex-1 bg-gradient-to-r from-red-500/20 to-orange-500/20 hover:from-red-500/30 hover:to-orange-500/30 border border-red-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2 disabled:opacity-50"
                  >
                    {refreshing ? (
                      <>
                        <RefreshCw className="w-4 h-4 animate-spin" />
                        <span>Deactivating...</span>
                      </>
                    ) : (
                      <>
                        <XCircle className="w-4 h-4" />
                        <span>Deactivate</span>
                      </>
                    )}
                  </button>
                )}
                <button 
                  onClick={() => handleEditServer(server)}
                  className="bg-slate-700/50 border border-slate-600/50 text-slate-300 hover:text-white hover:border-slate-500/50 p-2 rounded-lg transition-all duration-200"
                >
                  <Settings className="w-4 h-4" />
                </button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Capabilities Section */}
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-bold text-white">Available Capabilities</h2>
          <div className="flex items-center space-x-2">
            <button 
              onClick={() => setRefreshing(true)}
              disabled={refreshing}
              className="text-purple-400 hover:text-purple-300 text-sm font-medium flex items-center space-x-1 disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          </div>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {capabilities.map((capability) => {
            const server = mcpServers.find(s => s.id === capability.serverId);
            const ServerIcon = server?.icon || Server;
            
            return (
              <div 
                key={`${capability.serverId}-${capability.id}`}
                className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30 hover:bg-slate-700/50 transition-colors duration-200 cursor-pointer"
                onClick={() => handleEditCapability(capability)}
              >
                <div className="flex items-start space-x-3">
                  <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
                    <Zap className="w-5 h-5 text-purple-400" />
                  </div>
                  <div>
                    <h3 className="text-white font-medium">{capability.name}</h3>
                    <div className="flex items-center space-x-2 mt-1">
                      <span className={`px-2 py-0.5 rounded-full text-xs ${getCapabilityStatusColor(capability.status)}`}>
                        {capability.status}
                      </span>
                      <span className="text-slate-400 text-xs flex items-center space-x-1">
                        <ServerIcon className="w-3 h-3" />
                        <span>{server?.name || 'Unknown Server'}</span>
                      </span>
                    </div>
                    <div className="mt-2 flex items-center space-x-2">
                      <button className="text-slate-400 hover:text-white text-xs flex items-center space-x-1">
                        <Eye className="w-3 h-3" />
                        <span>View</span>
                      </button>
                      <button className="text-slate-400 hover:text-white text-xs flex items-center space-x-1">
                        <Code className="w-3 h-3" />
                        <span>Schema</span>
                      </button>
                      {capability.status === 'inactive' && (
                        <button className="text-green-400 hover:text-green-300 text-xs flex items-center space-x-1">
                          <Download className="w-3 h-3" />
                          <span>Install</span>
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* MCP Server Modal */}
      <MCPServerModal
        isOpen={isServerModalOpen}
        onClose={() => {
          setIsServerModalOpen(false);
          setSelectedServer(null);
        }}
        server={selectedServer}
        onServerSaved={handleServerSaved}
      />

      {/* MCP Capability Modal */}
      <MCPCapabilityModal
        isOpen={isCapabilityModalOpen}
        onClose={() => {
          setIsCapabilityModalOpen(false);
          setSelectedServer(null);
          setSelectedCapability(null);
        }}
        server={selectedServer}
        capability={selectedCapability}
        onCapabilitySaved={handleCapabilitySaved}
      />

      {/* Advanced Filter Modal */}
      <AdvancedFilterModal
        isOpen={isFilterModalOpen}
        onClose={() => setIsFilterModalOpen(false)}
        onApplyFilters={applyAdvancedFilters}
        initialFilters={activeFilters}
        filterOptions={filterOptions}
        entityType="mcp-server"
      />
    </div>
  );
};

export default MCPCapabilityManager;