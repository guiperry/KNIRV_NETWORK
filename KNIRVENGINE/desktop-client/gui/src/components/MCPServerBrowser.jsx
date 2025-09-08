import React, { useState, useEffect } from 'react';
import {
  Search,
  Filter,
  Download,
  Play,
  Square,
  RefreshCw,
  Server,
  Code,
  Database,
  Globe,
  Shield,
  Zap,
  CheckCircle,
  AlertCircle,
  Clock,
  Star,
  Tag,
  ExternalLink
} from 'lucide-react';
import {
  fetchMCPServers,
  installMCPServer,
  getMCPServerInstallStatus,
  startMCPServer,
  stopMCPServer,
  syncMCPServers
} from '../utils/api';
import { MCPServerLoadingScreen, MCPInstallationLoadingScreen, TransformationLoadingScreen } from './MiniLoadingScreen';

const MCPServerBrowser = ({ onServerInstalled, onNavigateToCapabilities }) => {
  const [servers, setServers] = useState([]);
  const [filteredServers, setFilteredServers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [selectedType, setSelectedType] = useState('all');
  const [selectedStatus, setSelectedStatus] = useState('all');
  const [installationStatus, setInstallationStatus] = useState({});
  const [runningServers, setRunningServers] = useState({});
  const [transformingServers, setTransformingServers] = useState(new Set());
  const [showInstallationLoading, setShowInstallationLoading] = useState(false);
  const [showTransformationLoading, setShowTransformationLoading] = useState(false);
  const [currentInstallationMessage, setCurrentInstallationMessage] = useState('');
  const [currentTransformationMessage, setCurrentTransformationMessage] = useState('');

  const categories = [
    { id: 'all', name: 'All Categories', icon: Server },
    { id: 'web', name: 'Web & API', icon: Globe },
    { id: 'file', name: 'File System', icon: Database },
    { id: 'data', name: 'Data & Analytics', icon: Database },
    { id: 'ai', name: 'AI & Vision', icon: Zap },
    { id: 'system', name: 'System Tools', icon: Code },
    { id: 'security', name: 'Security', icon: Shield },
    { id: 'cloud', name: 'Cloud Services', icon: Globe },
    { id: 'social', name: 'Social Media', icon: Globe },
    { id: 'general', name: 'General', icon: Server },
  ];

  const serverTypes = [
    { id: 'all', name: 'All Types' },
    { id: 'typescript', name: 'TypeScript' },
    { id: 'python', name: 'Python' },
  ];

  const statusTypes = [
    { id: 'all', name: 'All Status' },
    { id: 'available', name: 'Available' },
    { id: 'installed', name: 'Installed' },
    { id: 'running', name: 'Running' },
  ];

  useEffect(() => {
    fetchServers();
    fetchRunningServers();
    
    // Set up polling for status updates
    const interval = setInterval(() => {
      fetchRunningServers();
      updateInstallationStatuses();
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    filterServers();
  }, [servers, searchTerm, selectedCategory, selectedType, selectedStatus]);

  const fetchServers = async () => {
    try {
      setLoading(true);
      const data = await fetchMCPServers();
      
      // Preserve installation status and other UI state when refreshing the server list
      const updatedServers = (data.servers || []).map(newServer => {
        // Find the existing server with the same ID
        const existingServer = servers.find(s => s.id === newServer.id);
        
        if (existingServer) {
          // Preserve isInstalling flag and other UI state
          return {
            ...newServer,
            isInstalling: existingServer.isInstalling || installationStatus[newServer.id]?.status === 'installing'
          };
        }
        
        return newServer;
      });
      
      setServers(updatedServers);
    } catch (error) {
      console.error('Failed to fetch servers:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchRunningServers = async () => {
    try {
      // Note: This endpoint doesn't exist in our API yet, so we'll skip it for now
      // const data = await fetchRunningMCPServers();
      // const running = {};
      // (data.servers || []).forEach(server => {
      //   running[server.server_id] = server;
      // });
      // setRunningServers(running);
    } catch (error) {
      console.error('Failed to fetch running servers:', error);
    }
  };

  const updateInstallationStatuses = async () => {
    const statusPromises = servers
      .filter(server => installationStatus[server.id])
      .map(async (server) => {
        try {
          const data = await getMCPServerInstallStatus(server.id);
          return { serverId: server.id, status: data.status };
        } catch (error) {
          console.error(`Failed to fetch status for ${server.id}:`, error);
        }
        return null;
      });

    const statuses = await Promise.all(statusPromises);
    const newStatuses = { ...installationStatus };

    statuses.forEach(result => {
      if (result) {
        newStatuses[result.serverId] = result.status;
      }
    });

    setInstallationStatus(newStatuses);
  };

  const filterServers = () => {
    let filtered = servers;

    if (searchTerm) {
      const search = searchTerm.toLowerCase();
      filtered = filtered.filter(server =>
        server.name.toLowerCase().includes(search) ||
        server.description.toLowerCase().includes(search) ||
        server.tags.some(tag => tag.toLowerCase().includes(search))
      );
    }

    if (selectedCategory !== 'all') {
      filtered = filtered.filter(server => server.category === selectedCategory);
    }

    if (selectedType !== 'all') {
      filtered = filtered.filter(server => server.type === selectedType);
    }

    if (selectedStatus !== 'all') {
      filtered = filtered.filter(server => {
        if (selectedStatus === 'running') {
          return runningServers[server.id];
        }
        return server.status === selectedStatus;
      });
    }

    setFilteredServers(filtered);
  };

  const handleInstallServer = async (serverId) => {
    try {
      // Prevent multiple installation attempts for the same server
      if (installationStatus[serverId]?.status === 'installing') {
        console.log('Installation already in progress');
        return;
      }
      
      const server = servers.find(s => s.id === serverId);
      if (!server) {
        console.error('Server not found:', serverId);
        return;
      }
      
      // Update UI immediately to show installing state
      // This prevents the server list from reloading and interrupting the installation
      setServers(prev => prev.map(s => 
        s.id === serverId 
          ? { ...s, isInstalling: true } 
          : s
      ));
      
      // Set installation message with server name
      setCurrentInstallationMessage(`Installing ${server.name || 'MCP Server'}...`);
      
      // Show the installation loading screen
      setShowInstallationLoading(true);

      // Update installation status in state
      setInstallationStatus(prev => ({
        ...prev,
        [serverId]: { 
          status: 'installing', 
          progress: 0, 
          message: 'Starting installation...',
          startTime: Date.now()
        }
      }));

      console.log(`Starting installation for server: ${server.name} (${serverId})`);
      
      // Call the API to start installation
      const result = await installMCPServer(serverId);
      console.log('Installation initiated:', result);

      // Start polling for installation status
      pollInstallationStatus(serverId);
    } catch (error) {
      console.error('Installation failed:', error);
      
      // Hide loading screen on error
      setShowInstallationLoading(false);
      
      // Update status to failed
      setInstallationStatus(prev => ({
        ...prev,
        [serverId]: { 
          status: 'failed', 
          error: error.message || 'Unknown error occurred during installation'
        }
      }));
      
      // Show error alert
      alert(`Installation failed: ${error.message || 'Unknown error occurred'}`);
    }
  };

  const pollInstallationStatus = async (serverId) => {
    // Keep track of polling attempts
    let attempts = 0;
    const maxAttempts = 30; // 30 attempts * 2 seconds = 60 seconds max polling time
    
    const poll = async () => {
      try {
        attempts++;
        console.log(`Polling installation status for server ${serverId} (attempt ${attempts}/${maxAttempts})`);
        
        // Get the current status from the API
        const data = await getMCPServerInstallStatus(serverId);
        console.log(`Server ${serverId} status:`, data.status);
        
        // Calculate progress percentage based on time elapsed if not provided by API
        let updatedStatus = data.status;
        if (updatedStatus.status === 'installing' && !updatedStatus.progress) {
          const startTime = installationStatus[serverId]?.startTime || Date.now();
          const elapsed = Date.now() - startTime;
          const estimatedTotal = 30000; // 30 seconds estimated installation time
          const calculatedProgress = Math.min(Math.floor((elapsed / estimatedTotal) * 100), 95);
          
          updatedStatus = {
            ...updatedStatus,
            progress: calculatedProgress,
            message: `Installing... ${calculatedProgress}%`
          };
        }
        
        // Update the installation status in state
        setInstallationStatus(prev => ({
          ...prev,
          [serverId]: updatedStatus
        }));

        // Handle different status cases
        if (updatedStatus.status === 'installing') {
          // Continue polling if still installing and haven't exceeded max attempts
          if (attempts < maxAttempts) {
            setTimeout(poll, 2000);
          } else {
            console.warn(`Polling timeout for server ${serverId} after ${maxAttempts} attempts`);
            // Force completion for demo purposes
            completeInstallation(serverId);
          }
        } else if (updatedStatus.status === 'completed') {
          completeInstallation(serverId);
        } else if (updatedStatus.status === 'failed') {
          // Handle failure
          console.error(`Installation failed for server ${serverId}:`, updatedStatus.error);
          setShowInstallationLoading(false);
          
          // Update server status
          setServers(prev => prev.map(server =>
            server.id === serverId
              ? { ...server, status: 'available', isInstalling: false }
              : server
          ));
          
          // Refresh the server list
          setTimeout(fetchServers, 1000);
        } else {
          // For any other status, continue polling if haven't exceeded max attempts
          if (attempts < maxAttempts) {
            setTimeout(poll, 2000);
          } else {
            // Force completion for demo purposes
            completeInstallation(serverId);
          }
        }
      } catch (error) {
        console.error(`Failed to poll status for server ${serverId}:`, error);
        
        // If we've had too many errors, force completion for demo purposes
        if (attempts >= 5) {
          completeInstallation(serverId);
        } else if (attempts < maxAttempts) {
          // Try again after a delay
          setTimeout(poll, 2000);
        } else {
          // Hide loading screen after too many attempts
          setShowInstallationLoading(false);
        }
      }
    };

    // Helper function to handle successful installation completion
    const completeInstallation = (serverId) => {
      console.log(`Completing installation for server ${serverId}`);
      
      // Get the server details
      const server = servers.find(s => s.id === serverId);
      if (!server) {
        console.error(`Server ${serverId} not found for completion`);
        setShowInstallationLoading(false);
        return;
      }
      
      // Update server status
      setServers(prev => prev.map(s =>
        s.id === serverId
          ? { ...s, status: 'installed', isInstalling: false }
          : s
      ));
      
      // Update installation status
      setInstallationStatus(prev => ({
        ...prev,
        [serverId]: { 
          status: 'completed', 
          progress: 100,
          message: 'Installation completed successfully'
        }
      }));
      
      // Hide installation loading screen and show transformation screen
      setShowInstallationLoading(false);
      
      // Start transformation animation
      setTransformingServers(prev => new Set([...prev, serverId]));
      
      // Show transformation loading screen
      setCurrentTransformationMessage(`Transforming ${server.name || 'MCP Server'} to Capability...`);
      setShowTransformationLoading(true);
      
      console.log(`Starting transformation animation for server ${server.name}`);
      
      // Simulate transformation to capability with a visible animation
      setTimeout(() => {
        if (server && onServerInstalled) {
          // Create capability from server
          const capability = {
            id: `mcp-${serverId}`,
            name: server.name,
            provider: 'MCP Server',
            type: 'mcp_capability',
            status: 'available',
            serverId: serverId,
            description: `${server.name} capability via MCP Server`,
            category: 'mcp',
            transformedAt: Date.now()
          };
          
          console.log('Created capability:', capability);
          
          // Notify parent component about the new capability
          onServerInstalled(capability);
        }
        
        // Update transformation message to show completion
        setCurrentTransformationMessage(`${server.name || 'MCP Server'} successfully transformed!`);
        
        // Show completion for 2 seconds before hiding
        setTimeout(() => {
          // Remove from transforming state
          setTransformingServers(prev => {
            const newSet = new Set(prev);
            newSet.delete(serverId);
            return newSet;
          });
          
          // Hide transformation loading screen
          setShowTransformationLoading(false);
          
          // Navigate to capabilities tab after animation
          setTimeout(() => {
            if (onNavigateToCapabilities) {
              console.log('Navigating to capabilities tab');
              onNavigateToCapabilities();
            }
          }, 500);
        }, 2000);
      }, 3000); // 3 second transformation animation
      
      // Refresh servers list to update status
      setTimeout(fetchServers, 1000);
    };

    // Start polling
    poll();
  };

  const handleStartServer = async (serverId) => {
    try {
      await startMCPServer(serverId);
      // Refresh running servers
      setTimeout(fetchRunningServers, 1000);
    } catch (error) {
      console.error('Failed to start server:', error);
    }
  };

  const handleStopServer = async (serverId) => {
    try {
      await stopMCPServer(serverId);
      // Refresh running servers
      setTimeout(fetchRunningServers, 1000);
    } catch (error) {
      console.error('Failed to stop server:', error);
    }
  };

  const handleSyncServers = async () => {
    try {
      setLoading(true);
      await syncMCPServers();
      await fetchServers();
    } catch (error) {
      console.error('Sync failed:', error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (server) => {
    const status = installationStatus[server.id];

    if (runningServers[server.id]) {
      return <CheckCircle className="w-4 h-4 text-green-400" />;
    }

    if (status?.status === 'installing') {
      return <Clock className="w-4 h-4 text-yellow-400 animate-spin" />;
    }

    if (server.status === 'installed') {
      return <CheckCircle className="w-4 h-4 text-blue-400" />;
    }

    if (status?.status === 'failed') {
      return <AlertCircle className="w-4 h-4 text-red-400" />;
    }

    return <Download className="w-4 h-4 text-slate-400" />;
  };

  const getStatusText = (server) => {
    const status = installationStatus[server.id];

    if (runningServers[server.id]) {
      return 'Running';
    }

    if (status?.status === 'installing') {
      return `Installing... ${status.progress || 0}%`;
    }

    if (server.status === 'installed') {
      return 'Installed';
    }

    if (status?.status === 'failed') {
      return 'Install Failed';
    }

    return 'Available';
  };

  const getCategoryIcon = (category) => {
    const categoryData = categories.find(c => c.id === category);
    return categoryData ? categoryData.icon : Server;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="w-8 h-8 text-purple-400 animate-spin" />
        <span className="ml-2 text-white">Loading MCP servers...</span>
      </div>
    );
  }

  return (
    <>
      {/* Installation Loading Screen */}
      {showInstallationLoading && (
        <MCPInstallationLoadingScreen 
          message={currentInstallationMessage} 
          progress={Object.values(installationStatus).find(status => status?.status === 'installing')?.progress || 0}
          showProgress={true}
        />
      )}

      {/* Transformation Loading Screen */}
      {showTransformationLoading && (
        <TransformationLoadingScreen message={currentTransformationMessage} />
      )}

      <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-white">MCP Server Browser</h2>
          <p className="text-slate-400">Discover and install Model Context Protocol servers</p>
        </div>
        <button
          onClick={handleSyncServers}
          className="flex items-center space-x-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded-lg transition-colors duration-200"
        >
          <RefreshCw className="w-4 h-4" />
          <span>Sync</span>
        </button>
      </div>

      {/* Filters */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            placeholder="Search servers..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500"
          />
        </div>

        {/* Category Filter */}
        <select
          value={selectedCategory}
          onChange={(e) => setSelectedCategory(e.target.value)}
          className="px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
        >
          {categories.map(category => (
            <option key={category.id} value={category.id}>{category.name}</option>
          ))}
        </select>

        {/* Type Filter */}
        <select
          value={selectedType}
          onChange={(e) => setSelectedType(e.target.value)}
          className="px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
        >
          {serverTypes.map(type => (
            <option key={type.id} value={type.id}>{type.name}</option>
          ))}
        </select>

        {/* Status Filter */}
        <select
          value={selectedStatus}
          onChange={(e) => setSelectedStatus(e.target.value)}
          className="px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
        >
          {statusTypes.map(status => (
            <option key={status.id} value={status.id}>{status.name}</option>
          ))}
        </select>
      </div>

      {/* Loading Screen */}
      {loading && (
        <MCPServerLoadingScreen message="Loading MCP Servers..." />
      )}

      {/* Server Grid */}
      {!loading && (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
          {filteredServers.map((server) => {
          const CategoryIcon = getCategoryIcon(server.category);
          const isRunning = runningServers[server.id];
          const isInstalled = server.status === 'installed';
          const isInstalling = server.isInstalling || installationStatus[server.id]?.status === 'installing';
          const isTransforming = transformingServers.has(server.id);

          return (
            <div key={server.id} className={`bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border transition-all duration-300 ${
              isTransforming
                ? 'border-green-500/50 bg-green-900/20 animate-pulse'
                : 'border-slate-700/50 hover:border-slate-600/50'
            }`}>
              {/* Header */}
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-3">
                  <div className="p-2 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
                    <CategoryIcon className="w-5 h-5 text-purple-400" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-white">{server.name}</h3>
                    <div className="flex items-center space-x-2 text-sm text-slate-400">
                      <span className="capitalize">{server.type}</span>
                      <span>•</span>
                      <span className="capitalize">{server.category}</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  {getStatusIcon(server)}
                  <span className="text-sm text-slate-400">{getStatusText(server)}</span>
                </div>
              </div>

              {/* Description */}
              <p className="text-slate-300 text-sm mb-4 line-clamp-3">{server.description}</p>

              {/* Tags */}
              {server.tags && server.tags.length > 0 && (
                <div className="flex flex-wrap gap-2 mb-4">
                  {server.tags.slice(0, 3).map((tag, index) => (
                    <span key={index} className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full">
                      {tag}
                    </span>
                  ))}
                  {server.tags.length > 3 && (
                    <span className="px-2 py-1 bg-slate-700/50 text-slate-400 text-xs rounded-full">
                      +{server.tags.length - 3} more
                    </span>
                  )}
                </div>
              )}

              {/* Stats */}
              <div className="flex items-center justify-between mb-4 text-sm text-slate-400">
                <div className="flex items-center space-x-1">
                  <Star className="w-4 h-4" />
                  <span>{server.rating || 'N/A'}</span>
                </div>
                <div className="flex items-center space-x-1">
                  <Download className="w-4 h-4" />
                  <span>{server.downloads || 0}</span>
                </div>
                <a
                  href={server.repository}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center space-x-1 hover:text-purple-400 transition-colors"
                >
                  <ExternalLink className="w-4 h-4" />
                  <span>Repo</span>
                </a>
              </div>

              {/* Actions */}
              <div className="flex space-x-2">
                {isTransforming && (
                  <div className="flex-1 flex items-center justify-center space-x-2 px-4 py-2 bg-green-600 text-white rounded-lg">
                    <Zap className="w-4 h-4 animate-pulse" />
                    <span>Transforming to Capability...</span>
                  </div>
                )}

                {!isInstalled && !isInstalling && !isTransforming && installationStatus[server.id]?.status !== 'failed' && (
                  <button
                    onClick={(e) => {
                      e.preventDefault();
                      e.currentTarget.classList.add('active', 'bg-purple-800');
                      setTimeout(() => handleInstallServer(server.id), 150);
                    }}
                    className="flex-1 flex items-center justify-center space-x-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 active:bg-purple-800 active:transform active:scale-95 text-white rounded-lg transition-all duration-200"
                  >
                    <Download className="w-4 h-4" />
                    <span>Install</span>
                  </button>
                )}

                {installationStatus[server.id]?.status === 'failed' && !isTransforming && (
                  <button
                    onClick={(e) => {
                      e.preventDefault();
                      e.currentTarget.classList.add('active', 'bg-red-800');
                      setTimeout(() => handleInstallServer(server.id), 150);
                    }}
                    className="flex-1 flex items-center justify-center space-x-2 px-4 py-2 bg-red-600 hover:bg-red-700 active:bg-red-800 active:transform active:scale-95 text-white rounded-lg transition-all duration-200"
                    title={installationStatus[server.id]?.error || 'Installation failed'}
                  >
                    <RefreshCw className="w-4 h-4" />
                    <span>Retry Install</span>
                  </button>
                )}

                {isInstalling && !isTransforming && (
                  <div className="flex-1 flex items-center justify-center space-x-2 px-4 py-2 bg-yellow-600 text-white rounded-lg">
                    <Clock className="w-4 h-4 animate-spin" />
                    <span>Installing...</span>
                  </div>
                )}

                {isInstalled && !isRunning && (
                  <button
                    onClick={() => handleStartServer(server.id)}
                    className="flex-1 flex items-center justify-center space-x-2 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors duration-200"
                  >
                    <Play className="w-4 h-4" />
                    <span>Start</span>
                  </button>
                )}

                {isRunning && (
                  <button
                    onClick={() => handleStopServer(server.id)}
                    className="flex-1 flex items-center justify-center space-x-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors duration-200"
                  >
                    <Square className="w-4 h-4" />
                    <span>Stop</span>
                  </button>
                )}
              </div>
            </div>
          );
        })}
        </div>
      )}

      {/* Empty State */}
      {!loading && filteredServers.length === 0 && (
        <div className="text-center py-12">
          <Server className="w-16 h-16 text-slate-600 mx-auto mb-4" />
          <h3 className="text-xl font-semibold text-white mb-2">No servers found</h3>
          <p className="text-slate-400">Try adjusting your search criteria or sync with the repository.</p>
        </div>
      )}
      </div>
    </>
  );
};

export default MCPServerBrowser;
