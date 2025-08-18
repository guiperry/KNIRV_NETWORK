import React, { useState, useEffect } from 'react';
import { 
  Search, 
  Filter, 
  Plus,
  Globe,
  Settings,
  Trash2,
  RefreshCw,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Key,
  Lock,
  Eye,
  EyeOff,
  ExternalLink,
  Clock,
  Shield,
  Database
} from 'lucide-react';
import WebConnectionModal from './modals/WebConnectionModal';
import AdvancedFilterModal from './modals/AdvancedFilterModal';

export const WebConnectionsManager = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isFilterModalOpen, setIsFilterModalOpen] = useState(false);
  const [selectedConnection, setSelectedConnection] = useState(null);
  const [refreshing, setRefreshing] = useState(false);
  const [activeFilters, setActiveFilters] = useState([]);
  const [showSecrets, setShowSecrets] = useState({});
  const [isAuthorizing, setIsAuthorizing] = useState(false);
  const [authWindow, setAuthWindow] = useState(null);

  const categories = [
    { id: 'all', name: 'All Connections' },
    { id: 'active', name: 'Active' },
    { id: 'inactive', name: 'Inactive' },
    { id: 'oauth', name: 'OAuth' },
    { id: 'apikey', name: 'API Key' }
  ];

  const [connections, setConnections] = useState([
    {
      id: 1,
      name: 'Google Drive',
      type: 'oauth',
      status: 'active',
      provider: 'Google',
      description: 'Access to Google Drive files and folders',
      icon: 'https://upload.wikimedia.org/wikipedia/commons/1/12/Google_Drive_icon_%282020%29.svg',
      lastUsed: '5 minutes ago',
      expiresAt: '2024-06-15T10:30:00Z',
      scopes: ['https://www.googleapis.com/auth/drive.readonly', 'https://www.googleapis.com/auth/drive.file'],
      createdAt: '2024-01-15T10:30:00Z',
      createdBy: 'Admin User',
      connectionDetails: {
        clientId: 'google-client-id-123',
        clientSecret: 'google-client-secret-456',
        redirectUri: 'https://app.example.com/oauth/callback',
        tokenEndpoint: 'https://oauth2.googleapis.com/token',
        authEndpoint: 'https://accounts.google.com/o/oauth2/auth'
      }
    },
    {
      id: 2,
      name: 'GitHub',
      type: 'oauth',
      status: 'active',
      provider: 'GitHub',
      description: 'Access to GitHub repositories and issues',
      icon: 'https://github.githubassets.com/assets/GitHub-Mark-ea2971cee799.png',
      lastUsed: '1 hour ago',
      expiresAt: '2024-05-20T14:45:00Z',
      scopes: ['repo', 'user'],
      createdAt: '2024-01-10T14:45:00Z',
      createdBy: 'Admin User',
      connectionDetails: {
        clientId: 'github-client-id-123',
        clientSecret: 'github-client-secret-456',
        redirectUri: 'https://app.example.com/oauth/callback',
        tokenEndpoint: 'https://github.com/login/oauth/access_token',
        authEndpoint: 'https://github.com/login/oauth/authorize'
      }
    },
    {
      id: 3,
      name: 'OpenAI API',
      type: 'apikey',
      status: 'active',
      provider: 'OpenAI',
      description: 'Access to OpenAI API services',
      icon: 'https://upload.wikimedia.org/wikipedia/commons/0/04/ChatGPT_logo.svg',
      lastUsed: '10 minutes ago',
      expiresAt: null, // No expiration
      scopes: ['*'],
      createdAt: '2023-12-05T09:15:00Z',
      createdBy: 'Admin User',
      connectionDetails: {
        apiKey: 'sk-openai-api-key-123456',
        baseUrl: 'https://api.openai.com/v1',
        organizationId: 'org-123'
      }
    },
    {
      id: 4,
      name: 'Slack Workspace',
      type: 'oauth',
      status: 'inactive',
      provider: 'Slack',
      description: 'Access to Slack channels and messages',
      icon: 'https://upload.wikimedia.org/wikipedia/commons/d/d5/Slack_icon_2019.svg',
      lastUsed: '5 days ago',
      expiresAt: '2024-01-10T16:20:00Z', // Expired
      scopes: ['channels:read', 'chat:write'],
      createdAt: '2023-11-14T16:20:00Z',
      createdBy: 'Admin User',
      connectionDetails: {
        clientId: 'slack-client-id-123',
        clientSecret: 'slack-client-secret-456',
        redirectUri: 'https://app.example.com/oauth/callback',
        tokenEndpoint: 'https://slack.com/api/oauth.v2.access',
        authEndpoint: 'https://slack.com/oauth/v2/authorize'
      }
    },
    {
      id: 5,
      name: 'AWS S3',
      type: 'apikey',
      status: 'active',
      provider: 'AWS',
      description: 'Access to AWS S3 storage',
      icon: 'https://upload.wikimedia.org/wikipedia/commons/9/93/Amazon_Web_Services_Logo.svg',
      lastUsed: '1 day ago',
      expiresAt: null, // No expiration
      scopes: ['s3:GetObject', 's3:PutObject', 's3:ListBucket'],
      createdAt: '2023-10-18T08:45:00Z',
      createdBy: 'Admin User',
      connectionDetails: {
        accessKeyId: 'AKIAIOSFODNN7EXAMPLE',
        secretAccessKey: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY',
        region: 'us-west-2',
        bucket: 'example-bucket'
      }
    },
    {
      id: 6,
      name: 'Salesforce',
      type: 'oauth',
      status: 'inactive',
      provider: 'Salesforce',
      description: 'Access to Salesforce CRM data',
      icon: 'https://upload.wikimedia.org/wikipedia/commons/f/f9/Salesforce.com_logo.svg',
      lastUsed: '30 days ago',
      expiresAt: '2023-12-20T11:30:00Z', // Expired
      scopes: ['api', 'refresh_token'],
      createdAt: '2023-09-20T11:30:00Z',
      createdBy: 'Admin User',
      connectionDetails: {
        clientId: 'salesforce-client-id-123',
        clientSecret: 'salesforce-client-secret-456',
        redirectUri: 'https://app.example.com/oauth/callback',
        tokenEndpoint: 'https://login.salesforce.com/services/oauth2/token',
        authEndpoint: 'https://login.salesforce.com/services/oauth2/authorize',
        instanceUrl: 'https://example.my.salesforce.com'
      }
    }
  ]);

  // Filter options for the advanced filter modal
  const filterOptions = {
    fields: [
      { id: 'name', name: 'Name' },
      { id: 'type', name: 'Type' },
      { id: 'status', name: 'Status' },
      { id: 'provider', name: 'Provider' },
      { id: 'lastUsed', name: 'Last Used' },
      { id: 'createdAt', name: 'Created Date' },
      { id: 'createdBy', name: 'Created By' }
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

  // Check if a connection matches the advanced filters
  const matchesAdvancedFilters = (connection) => {
    if (activeFilters.length === 0) return true;
    
    return activeFilters.every(filter => {
      const { field, operator, value } = filter;
      const connectionValue = connection[field];
      
      // Handle null/undefined values
      if (connectionValue === null || connectionValue === undefined) {
        return operator === 'not_equals' || operator === 'not_contains';
      }
      
      // Convert to string for comparison
      const connectionValueStr = String(connectionValue).toLowerCase();
      const filterValueStr = String(value).toLowerCase();
      
      switch (operator) {
        case 'equals':
          return connectionValueStr === filterValueStr;
        case 'not_equals':
          return connectionValueStr !== filterValueStr;
        case 'contains':
          return connectionValueStr.includes(filterValueStr);
        case 'not_contains':
          return !connectionValueStr.includes(filterValueStr);
        case 'greater_than':
          return parseFloat(connectionValueStr) > parseFloat(filterValueStr);
        case 'less_than':
          return parseFloat(connectionValueStr) < parseFloat(filterValueStr);
        case 'starts_with':
          return connectionValueStr.startsWith(filterValueStr);
        case 'ends_with':
          return connectionValueStr.endsWith(filterValueStr);
        default:
          return true;
      }
    });
  };

  const filteredConnections = connections.filter(connection => {
    const matchesSearch = connection.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         connection.provider.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         connection.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || 
                           (selectedCategory === 'oauth' && connection.type === 'oauth') ||
                           (selectedCategory === 'apikey' && connection.type === 'apikey') ||
                           (selectedCategory === 'active' && connection.status === 'active') ||
                           (selectedCategory === 'inactive' && connection.status === 'inactive');
    const matchesAdvanced = matchesAdvancedFilters(connection);
    return matchesSearch && matchesCategory && matchesAdvanced;
  });

  const handleAddConnection = () => {
    setSelectedConnection(null);
    setIsModalOpen(true);
  };

  const handleEditConnection = (connection) => {
    setSelectedConnection(connection);
    setIsModalOpen(true);
  };

  const handleConnectionSaved = (connectionData) => {
    if (selectedConnection) {
      // Update existing connection
      setConnections(prevConnections => 
        prevConnections.map(connection => 
          connection.id === connectionData.id ? connectionData : connection
        )
      );
    } else {
      // Add new connection
      setConnections(prevConnections => [...prevConnections, connectionData]);
    }
    setIsModalOpen(false);
    setSelectedConnection(null);
  };

  const handleDeleteConnection = (connectionId) => {
    // In a real implementation, this would be an API call
    // For now, we'll just update the state
    setConnections(prevConnections => 
      prevConnections.filter(connection => connection.id !== connectionId)
    );
  };

  const handleToggleConnectionStatus = (connection) => {
    // In a real implementation, this would be an API call
    // For now, we'll just update the state
    setRefreshing(true);
    setTimeout(() => {
      setConnections(prevConnections => 
        prevConnections.map(conn => 
          conn.id === connection.id 
            ? { 
                ...conn, 
                status: conn.status === 'active' ? 'inactive' : 'active',
                lastUsed: conn.status === 'inactive' ? 'just now' : conn.lastUsed
              } 
            : conn
        )
      );
      setRefreshing(false);
    }, 1000);
  };

  const handleRefreshConnection = async (connection) => {
    setRefreshing(true);

    try {
      // Make real API call to refresh the connection
      const response = await fetch(`/api/v1/web-connections/${connection.id}/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        }
      });

      if (!response.ok) {
        throw new Error('Failed to refresh connection');
      }

      const data = await response.json();
      const updatedConnection = data.connection;

      // Update connection with fresh data
      setConnections(prevConnections =>
        prevConnections.map(conn =>
          conn.id === connection.id
            ? {
                ...conn,
                ...updatedConnection,
                lastUsed: 'just now'
              }
            : conn
        )
      );
    } catch (error) {
      console.error('Failed to refresh connection:', error);
      // Fallback to just updating timestamps
      setConnections(prevConnections =>
        prevConnections.map(conn =>
          conn.id === connection.id
            ? {
                ...conn,
                lastUsed: 'just now',
                expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString() // 30 days from now
              }
            : conn
        )
      );
    } finally {
      setRefreshing(false);
    }
  };

  const handleToggleShowSecret = (connectionId) => {
    setShowSecrets(prev => ({
      ...prev,
      [connectionId]: !prev[connectionId]
    }));
  };

  const handleAuthorizeOAuth = (connection) => {
    setIsAuthorizing(true);
    
    // In a real implementation, this would use picaos.com/authkit
    // For now, we'll simulate the OAuth flow
    
    // Construct the authorization URL
    const authUrl = new URL(connection.connectionDetails.authEndpoint);
    authUrl.searchParams.append('client_id', connection.connectionDetails.clientId);
    authUrl.searchParams.append('redirect_uri', connection.connectionDetails.redirectUri);
    authUrl.searchParams.append('response_type', 'code');
    authUrl.searchParams.append('scope', connection.scopes.join(' '));
    authUrl.searchParams.append('state', `connection_${connection.id}`);
    
    // Open a new window for authorization
    const authWindowRef = window.open(authUrl.toString(), 'oauth_window', 'width=600,height=700');
    setAuthWindow(authWindowRef);
    
    // Set up message listener for the OAuth callback
    const messageHandler = (event) => {
      // In a real implementation, verify the origin
      if (event.data && event.data.type === 'oauth_callback') {
        // Process the OAuth callback
        const { code, state } = event.data;
        
        if (state === `connection_${connection.id}`) {
          // Exchange the code for tokens (would be done server-side in a real implementation)
          console.log(`Received authorization code: ${code} for connection ${connection.id}`);
          
          // Update the connection status
          setConnections(prevConnections => 
            prevConnections.map(conn => 
              conn.id === connection.id 
                ? { 
                    ...conn, 
                    status: 'active',
                    lastUsed: 'just now',
                    expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString() // 30 days from now
                  } 
                : conn
            )
          );
          
          // Close the auth window
          if (authWindowRef && !authWindowRef.closed) {
            authWindowRef.close();
          }
          
          // Clean up
          window.removeEventListener('message', messageHandler);
          setAuthWindow(null);
          setIsAuthorizing(false);
        }
      }
    };
    
    window.addEventListener('message', messageHandler);
    
    // Note: In a real implementation, the OAuth flow would be handled by the backend
    // and the authorization window would redirect to the actual OAuth provider.
    // For now, we'll keep the simulation for demo purposes, but in production
    // this would be replaced with real OAuth URLs and backend handling.
  };

  const isExpired = (expiresAt) => {
    if (!expiresAt) return false; // No expiration
    return new Date(expiresAt) < new Date();
  };

  const formatDate = (dateString) => {
    if (!dateString) return 'Never';
    const date = new Date(dateString);
    return date.toLocaleString();
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
                className="px-2 py-1 bg-blue-500/20 text-blue-300 text-xs rounded-full border border-blue-500/30"
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

  const getStatusColor = (status) => {
    switch (status) {
      case 'active':
        return 'text-green-400 bg-green-500/20 border-green-500/30';
      case 'inactive':
        return 'text-red-400 bg-red-500/20 border-red-500/30';
      default:
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
    }
  };

  const getTypeColor = (type) => {
    switch (type) {
      case 'oauth':
        return 'text-purple-400 bg-purple-500/20 border-purple-500/30';
      case 'apikey':
        return 'text-blue-400 bg-blue-500/20 border-blue-500/30';
      default:
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
    }
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">Web Connections</h1>
          <p className="text-slate-400">Manage connections to third-party services and APIs.</p>
        </div>
        <div className="mt-4 lg:mt-0">
          <button 
            onClick={handleAddConnection}
            className="bg-gradient-to-r from-blue-500 to-cyan-500 text-white px-6 py-3 rounded-lg font-medium hover:from-blue-600 hover:to-cyan-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Plus className="w-5 h-5" />
            <span>Add Connection</span>
          </button>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col lg:flex-row lg:items-center space-y-4 lg:space-y-0 lg:space-x-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
          <input
            type="text"
            placeholder="Search connections..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full bg-slate-800/50 border border-slate-700/50 rounded-lg pl-10 pr-4 py-3 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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
        {categories.map((category) => (
          <button
            key={category.id}
            onClick={() => setSelectedCategory(category.id)}
            className={`flex items-center space-x-2 px-4 py-2 rounded-lg transition-all duration-200 ${
              selectedCategory === category.id
                ? 'bg-gradient-to-r from-blue-500/20 to-cyan-500/20 text-white border border-blue-500/30'
                : 'bg-slate-800/50 text-slate-300 border border-slate-700/50 hover:text-white hover:border-slate-600/50'
            }`}
          >
            <span className="font-medium">{category.name}</span>
          </button>
        ))}
      </div>

      {/* Connections Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {filteredConnections.map((connection) => (
          <div key={connection.id} className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50 hover:border-slate-600/50 transition-all duration-300">
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center space-x-3">
                <img 
                  src={connection.icon} 
                  alt={connection.provider} 
                  className="w-10 h-10 rounded-lg object-contain bg-white p-1"
                />
                <div>
                  <h3 className="text-lg font-bold text-white">{connection.name}</h3>
                  <div className="flex items-center space-x-2">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${getTypeColor(connection.type)}`}>
                      {connection.type === 'oauth' ? 'OAuth' : 'API Key'}
                    </span>
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${getStatusColor(connection.status)}`}>
                      {connection.status}
                    </span>
                    {isExpired(connection.expiresAt) && (
                      <span className="px-2 py-0.5 bg-red-500/20 text-red-400 text-xs rounded-full border border-red-500/30">
                        Expired
                      </span>
                    )}
                  </div>
                </div>
              </div>
              <div className="flex items-center space-x-2">
                <button 
                  onClick={() => handleEditConnection(connection)}
                  className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
                >
                  <Settings className="w-4 h-4" />
                </button>
                <button 
                  onClick={() => handleDeleteConnection(connection.id)}
                  className="p-2 text-slate-400 hover:text-red-400 transition-colors duration-200"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
            
            <p className="text-slate-300 text-sm mb-4">{connection.description}</p>
            
            <div className="grid grid-cols-2 gap-4 mb-4">
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-slate-400">Provider:</span>
                  <span className="text-white font-medium">{connection.provider}</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-slate-400">Last Used:</span>
                  <span className="text-white">{connection.lastUsed}</span>
                </div>
                {connection.expiresAt && (
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate-400">Expires:</span>
                    <span className={`text-white ${isExpired(connection.expiresAt) ? 'text-red-400' : ''}`}>
                      {formatDate(connection.expiresAt)}
                    </span>
                  </div>
                )}
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-slate-400">Created:</span>
                  <span className="text-white">{formatDate(connection.createdAt)}</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-slate-400">Created By:</span>
                  <span className="text-white">{connection.createdBy}</span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-slate-400">Scopes:</span>
                  <span className="text-white">{connection.scopes.length}</span>
                </div>
              </div>
            </div>
            
            {/* Connection Details */}
            <div className="mb-4">
              <div className="flex items-center justify-between mb-2">
                <h4 className="text-white font-medium">Connection Details</h4>
                <button 
                  onClick={() => handleToggleShowSecret(connection.id)}
                  className="text-slate-400 hover:text-white text-xs flex items-center space-x-1"
                >
                  {showSecrets[connection.id] ? (
                    <>
                      <EyeOff className="w-3 h-3" />
                      <span>Hide</span>
                    </>
                  ) : (
                    <>
                      <Eye className="w-3 h-3" />
                      <span>Show</span>
                    </>
                  )}
                </button>
              </div>
              
              <div className="space-y-2">
                {connection.type === 'oauth' ? (
                  <>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Client ID:</span>
                      <span className="text-white font-mono text-xs">
                        {showSecrets[connection.id] 
                          ? connection.connectionDetails.clientId 
                          : connection.connectionDetails.clientId.substring(0, 8) + '...'
                        }
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Client Secret:</span>
                      <span className="text-white font-mono text-xs">
                        {showSecrets[connection.id] 
                          ? connection.connectionDetails.clientSecret 
                          : '••••••••••••••••'
                        }
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Redirect URI:</span>
                      <span className="text-white font-mono text-xs">{connection.connectionDetails.redirectUri}</span>
                    </div>
                  </>
                ) : (
                  <>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">API Key:</span>
                      <span className="text-white font-mono text-xs">
                        {showSecrets[connection.id] 
                          ? connection.connectionDetails.apiKey 
                          : '••••••••••••••••'
                        }
                      </span>
                    </div>
                    {connection.connectionDetails.accessKeyId && (
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-400">Access Key ID:</span>
                        <span className="text-white font-mono text-xs">
                          {showSecrets[connection.id] 
                            ? connection.connectionDetails.accessKeyId 
                            : connection.connectionDetails.accessKeyId.substring(0, 8) + '...'
                          }
                        </span>
                      </div>
                    )}
                    {connection.connectionDetails.secretAccessKey && (
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-400">Secret Access Key:</span>
                        <span className="text-white font-mono text-xs">
                          {showSecrets[connection.id] 
                            ? connection.connectionDetails.secretAccessKey 
                            : '••••••••••••••••'
                          }
                        </span>
                      </div>
                    )}
                    {connection.connectionDetails.baseUrl && (
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-slate-400">Base URL:</span>
                        <span className="text-white font-mono text-xs">{connection.connectionDetails.baseUrl}</span>
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
            
            {/* Scopes */}
            <div className="mb-4">
              <h4 className="text-white font-medium mb-2">Scopes</h4>
              <div className="flex flex-wrap gap-2">
                {connection.scopes.map((scope, index) => (
                  <span 
                    key={index}
                    className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full border border-slate-600/50"
                    title={scope}
                  >
                    {scope.length > 20 ? scope.substring(0, 18) + '...' : scope}
                  </span>
                ))}
              </div>
            </div>
            
            <div className="flex items-center space-x-3">
              {connection.status === 'active' ? (
                <button 
                  onClick={() => handleToggleConnectionStatus(connection)}
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
              ) : (
                <button 
                  onClick={() => connection.type === 'oauth' 
                    ? handleAuthorizeOAuth(connection) 
                    : handleToggleConnectionStatus(connection)
                  }
                  disabled={refreshing || isAuthorizing}
                  className="flex-1 bg-gradient-to-r from-green-500/20 to-emerald-500/20 hover:from-green-500/30 hover:to-emerald-500/30 border border-green-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2 disabled:opacity-50"
                >
                  {refreshing || isAuthorizing ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      <span>{connection.type === 'oauth' ? 'Authorizing...' : 'Activating...'}</span>
                    </>
                  ) : (
                    <>
                      <CheckCircle className="w-4 h-4" />
                      <span>{connection.type === 'oauth' ? 'Authorize' : 'Activate'}</span>
                    </>
                  )}
                </button>
              )}
              
              {connection.status === 'active' && connection.type === 'oauth' && (
                <button 
                  onClick={() => handleRefreshConnection(connection)}
                  disabled={refreshing}
                  className="bg-slate-700/50 border border-slate-600/50 text-slate-300 hover:text-white hover:border-slate-500/50 p-2 rounded-lg transition-all duration-200 disabled:opacity-50"
                  title="Refresh Token"
                >
                  <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
                </button>
              )}
              
              <button 
                onClick={() => window.open(`https://app.picaos.com/authkit/connections/${connection.id}`, '_blank')}
                className="bg-slate-700/50 border border-slate-600/50 text-slate-300 hover:text-white hover:border-slate-500/50 p-2 rounded-lg transition-all duration-200"
                title="View in AuthKit"
              >
                <ExternalLink className="w-4 h-4" />
              </button>
            </div>
          </div>
        ))}
        
        {/* Add Connection Card */}
        <div 
          className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50 border-dashed hover:border-blue-500/50 transition-all duration-300 flex flex-col items-center justify-center text-center cursor-pointer h-[400px]"
          onClick={handleAddConnection}
        >
          <div className="p-4 bg-blue-500/20 rounded-full border border-blue-500/30 mb-4">
            <Plus className="w-8 h-8 text-blue-400" />
          </div>
          <h3 className="text-lg font-bold text-white mb-2">Add New Connection</h3>
          <p className="text-slate-400 max-w-xs">Connect to third-party services and APIs to extend your agent capabilities.</p>
        </div>
      </div>

      {/* Empty State */}
      {filteredConnections.length === 0 && (
        <div className="text-center py-12 bg-slate-800/50 rounded-xl border border-slate-700/50">
          <Globe className="w-16 h-16 text-slate-700 mx-auto mb-4" />
          <h3 className="text-xl font-semibold text-white mb-2">No connections found</h3>
          <p className="text-slate-400 max-w-md mx-auto mb-6">
            {searchTerm || selectedCategory !== 'all' || activeFilters.length > 0
              ? 'No connections match your current filters. Try adjusting your search or filters.'
              : 'You haven\'t added any web connections yet. Add a connection to get started.'}
          </p>
          <button 
            onClick={handleAddConnection}
            className="bg-gradient-to-r from-blue-500 to-cyan-500 text-white px-6 py-3 rounded-lg font-medium hover:from-blue-600 hover:to-cyan-600 transition-all duration-200 flex items-center space-x-2 mx-auto"
          >
            <Plus className="w-5 h-5" />
            <span>Add Connection</span>
          </button>
        </div>
      )}

      {/* Web Connection Modal */}
      <WebConnectionModal
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setSelectedConnection(null);
        }}
        connection={selectedConnection}
        onConnectionSaved={handleConnectionSaved}
      />

      {/* Advanced Filter Modal */}
      <AdvancedFilterModal
        isOpen={isFilterModalOpen}
        onClose={() => setIsFilterModalOpen(false)}
        onApplyFilters={applyAdvancedFilters}
        initialFilters={activeFilters}
        filterOptions={filterOptions}
        entityType="connection"
      />
    </div>
  );
};

export default WebConnectionsManager;