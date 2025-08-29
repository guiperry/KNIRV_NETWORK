import React, { useState, useEffect } from 'react';
import { useAuthKit } from '@picahq/authkit';
import { handleApiError } from '../utils/errorHandler';
import {
  Search,
  Filter,
  Plus,
  Globe,
  Database,
  Cloud,
  Mail,
  FileText,
  Calendar,
  MessageSquare,
  ShoppingCart,
  Settings,
  RefreshCw,
  Trash2,
  CheckCircle,
  AlertCircle,
  Clock,
  ExternalLink
} from 'lucide-react';

// API configuration - detect if running in Electron
const getApiBaseUrl = () => {
  // Check if running in Electron (file:// protocol)
  if (window.location.protocol === 'file:') {
    return 'http://localhost:8081'; // Backend server port in Electron
  }
  return ''; // Use relative URLs for web version
};

export const WebConnections = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [connections, setConnections] = useState([]);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [testingConnectionId, setTestingConnectionId] = useState(null);

  // AuthKit configuration
  const apiBaseUrl = getApiBaseUrl();
  const { open } = useAuthKit({
    token: {
      url: `${apiBaseUrl}/api/v1/authkit-token`,
      headers: {},
    },
    onSuccess: (connection) => {
      console.log('AuthKit connection successful:', connection);
      // Add the new connection to the list
      setConnections(prev => [...prev, {
        id: connection.id || Date.now().toString(),
        name: connection.name || 'New Connection',
        provider: connection.provider || 'Unknown',
        category: connection.category || 'other',
        status: 'connected',
        authType: connection.authType || 'oauth',
        lastConnected: 'just now',
        scopes: connection.scopes || [],
        createdAt: new Date().toISOString()
      }]);
    },
    onError: (error) => {
      console.error('AuthKit error:', error);
      console.error('AuthKit error details:', {
        message: error.message,
        stack: error.stack,
        tokenUrl: `${apiBaseUrl}/api/v1/authkit-token`
      });

      // Handle error with AI Error Engine
      handleApiError(error, {
        operation: 'connect_service',
        component: 'WebConnections',
        service: 'AuthKit',
        tokenUrl: `${apiBaseUrl}/api/v1/authkit-token`,
        timestamp: new Date().toISOString(),
        context: 'User attempted to connect a web service via AuthKit'
      });
    },
    onClose: () => {
      console.log('AuthKit modal closed');
    },
  });

  const categories = [
    { id: 'all', name: 'All Connections', icon: Globe },
    { id: 'crm', name: 'CRM', icon: Database },
    { id: 'cloud', name: 'Cloud Storage', icon: Cloud },
    { id: 'email', name: 'Email', icon: Mail },
    { id: 'document', name: 'Document', icon: FileText },
    { id: 'calendar', name: 'Calendar', icon: Calendar },
    { id: 'chat', name: 'Chat', icon: MessageSquare },
    { id: 'ecommerce', name: 'E-Commerce', icon: ShoppingCart },
  ];

  // Fetch connections on component mount
  useEffect(() => {
    fetchConnections();
  }, []);

  const fetchConnections = async () => {
    // In a real implementation, this would be an API call
    // For now, we'll use sample data
    setConnections([
      {
        id: '1',
        name: 'Salesforce CRM',
        provider: 'Salesforce',
        category: 'crm',
        status: 'connected',
        authType: 'oauth',
        lastConnected: '2 hours ago',
        scopes: ['read', 'write', 'contacts', 'opportunities'],
        createdAt: '2023-01-15T10:30:00Z'
      },
      {
        id: '2',
        name: 'Google Drive',
        provider: 'Google',
        category: 'cloud',
        status: 'connected',
        authType: 'oauth',
        lastConnected: '5 minutes ago',
        scopes: ['drive.readonly', 'drive.file'],
        createdAt: '2023-02-20T14:45:00Z'
      },
      {
        id: '3',
        name: 'Microsoft 365',
        provider: 'Microsoft',
        category: 'email',
        status: 'error',
        authType: 'oauth',
        lastConnected: '3 days ago',
        scopes: ['mail.read', 'mail.send', 'calendars.read'],
        createdAt: '2023-03-10T09:15:00Z'
      },
      {
        id: '4',
        name: 'Dropbox',
        provider: 'Dropbox',
        category: 'cloud',
        status: 'disconnected',
        authType: 'oauth',
        lastConnected: 'Never',
        scopes: ['files.content.read', 'files.content.write'],
        createdAt: '2023-04-05T16:20:00Z'
      },
      {
        id: '5',
        name: 'Slack',
        provider: 'Slack',
        category: 'chat',
        status: 'connected',
        authType: 'oauth',
        lastConnected: '1 day ago',
        scopes: ['channels:read', 'chat:write', 'users:read'],
        createdAt: '2023-05-12T11:10:00Z'
      },
      {
        id: '6',
        name: 'Shopify Store',
        provider: 'Shopify',
        category: 'ecommerce',
        status: 'connected',
        authType: 'api_key',
        lastConnected: '4 hours ago',
        scopes: ['read_products', 'write_orders', 'read_customers'],
        createdAt: '2023-06-18T08:30:00Z'
      }
    ]);
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'connected':
        return <CheckCircle className="w-4 h-4 text-green-400" />;
      case 'disconnected':
        return <Clock className="w-4 h-4 text-slate-400" />;
      case 'error':
        return <AlertCircle className="w-4 h-4 text-red-400" />;
      default:
        return <Clock className="w-4 h-4 text-slate-400" />;
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'connected':
        return 'text-green-400 bg-green-500/20 border-green-500/30';
      case 'disconnected':
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
      case 'error':
        return 'text-red-400 bg-red-500/20 border-red-500/30';
      default:
        return 'text-slate-400 bg-slate-500/20 border-slate-500/30';
    }
  };

  const getCategoryIcon = (category) => {
    const foundCategory = categories.find(cat => cat.id === category);
    return foundCategory ? foundCategory.icon : Globe;
  };

  const filteredConnections = connections.filter(connection => {
    const matchesSearch = connection.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         connection.provider.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || connection.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const handleTestConnection = async (connectionId) => {
    setIsTestingConnection(true);
    setTestingConnectionId(connectionId);

    try {
      console.log('Testing connection:', connectionId);

      // Make API call to test the connection
      const response = await fetch(`${apiBaseUrl}/api/v1/web-connections/${connectionId}/test`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      console.log('Test connection response status:', response.status);

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Test connection error:', errorText);
        throw new Error(`Failed to test connection: ${response.status} ${response.statusText}`);
      }

      const result = await response.json();
      console.log('Test connection result:', result);

      // Update the connection status based on the result
      setConnections(prev => prev.map(conn => {
        if (conn.id === connectionId) {
          return {
            ...conn,
            status: result.success ? 'connected' : 'error',
            lastConnected: result.success ? 'just now' : conn.lastConnected,
            error: result.success ? null : result.error
          };
        }
        return conn;
      }));

      // Show success/error message
      if (result.success) {
        console.log('Connection test successful');
      } else {
        console.error('Connection test failed:', result.error);
        throw new Error(result.error || 'Connection test failed');
      }

    } catch (error) {
      console.error('Error testing connection:', error);

      // Update connection status to error
      setConnections(prev => prev.map(conn => {
        if (conn.id === connectionId) {
          return {
            ...conn,
            status: 'error',
            error: error.message
          };
        }
        return conn;
      }));

      // Handle error with AI Error Engine
      handleApiError(error, {
        operation: 'test_connection',
        component: 'WebConnections',
        connectionId: connectionId,
        apiUrl: `${apiBaseUrl}/api/v1/web-connections/${connectionId}/test`,
        timestamp: new Date().toISOString(),
        context: 'User attempted to test a web connection'
      });
    } finally {
      setIsTestingConnection(false);
      setTestingConnectionId(null);
    }
  };

  const handleDeleteConnection = async (connectionId) => {
    if (!confirm('Are you sure you want to delete this connection?')) {
      return;
    }
    
    try {
      // In a real implementation, this would be an API call
      // For now, we'll simulate it
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // Remove the connection from the list
      setConnections(prev => prev.filter(conn => conn.id !== connectionId));
    } catch (error) {
      console.error('Error deleting connection:', error);
      // Handle error with AI Error Engine
      handleApiError(error, {
        operation: 'delete_connection',
        component: 'WebConnections',
        connectionId: connectionId,
        timestamp: new Date().toISOString(),
        context: 'User attempted to delete a web connection'
      });
    }
  };

  const handleAddConnection = () => {
    try {
      console.log('Opening AuthKit modal...');
      console.log('AuthKit token URL:', `${apiBaseUrl}/api/v1/authkit-token`);

      // Open AuthKit modal for secure authentication
      open();
    } catch (error) {
      console.error('Error opening AuthKit modal:', error);

      // Handle error with AI Error Engine
      handleApiError(error, {
        operation: 'open_authkit_modal',
        component: 'WebConnections',
        tokenUrl: `${apiBaseUrl}/api/v1/authkit-token`,
        timestamp: new Date().toISOString(),
        context: 'User attempted to open AuthKit modal for connecting a service'
      });

      // Show a fallback error message
      alert('Unable to open connection dialog. Please check the console for details.');
    }
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-bold text-white mb-2">Web Connections</h1>
          <p className="text-slate-400">Connect your agents to third-party services and APIs.</p>
        </div>
        <div className="mt-4 lg:mt-0">
          <button 
            onClick={handleAddConnection}
            className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 flex items-center space-x-2"
          >
            <Plus className="w-5 h-5" />
            <span>Connect Service</span>
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
            className="w-full bg-slate-800/50 border border-slate-700/50 rounded-lg pl-10 pr-4 py-3 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          />
        </div>
        <button className="bg-slate-800/50 border border-slate-700/50 rounded-lg px-4 py-3 text-slate-300 hover:text-white hover:border-slate-600/50 transition-all duration-200 flex items-center space-x-2">
          <Filter className="w-5 h-5" />
          <span>Filter</span>
        </button>
      </div>

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

      {/* Connections Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {filteredConnections.length > 0 ? (
          filteredConnections.map((connection) => {
            const CategoryIcon = getCategoryIcon(connection.category);
            return (
              <div key={connection.id} className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50 hover:border-slate-600/50 transition-all duration-300">
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center space-x-3">
                    <div className="p-3 bg-gradient-to-r from-purple-500/20 to-blue-500/20 rounded-lg border border-purple-500/30">
                      <CategoryIcon className="w-6 h-6 text-purple-400" />
                    </div>
                    <div>
                      <h3 className="text-lg font-bold text-white">{connection.name}</h3>
                      <p className="text-slate-400 text-sm">{connection.provider}</p>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className={`px-3 py-1 rounded-full text-xs font-medium border flex items-center space-x-1 ${getStatusColor(connection.status)}`}>
                      {getStatusIcon(connection.status)}
                      <span className="capitalize">{connection.status}</span>
                    </span>
                  </div>
                </div>
                
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Auth Type:</span>
                      <span className="text-white font-medium capitalize">{connection.authType}</span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Category:</span>
                      <span className="text-white font-medium capitalize">{connection.category}</span>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Last Connected:</span>
                      <span className="text-white font-medium">{connection.lastConnected}</span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-slate-400">Created:</span>
                      <span className="text-white font-medium">{new Date(connection.createdAt).toLocaleDateString()}</span>
                    </div>
                  </div>
                </div>
                
                <div className="space-y-3 mb-4">
                  <div>
                    <p className="text-slate-400 text-sm mb-2">Scopes:</p>
                    <div className="flex flex-wrap gap-1">
                      {connection.scopes.map((scope, index) => (
                        <span key={index} className="px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-full">
                          {scope}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
                
                <div className="flex items-center space-x-3">
                  <button
                    onClick={() => handleTestConnection(connection.id)}
                    disabled={isTestingConnection && testingConnectionId === connection.id}
                    className="flex-1 bg-gradient-to-r from-blue-500/20 to-cyan-500/20 hover:from-blue-500/30 hover:to-cyan-500/30 border border-blue-500/30 text-white py-2 px-4 rounded-lg transition-all duration-200 flex items-center justify-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {isTestingConnection && testingConnectionId === connection.id ? (
                      <>
                        <RefreshCw className="w-4 h-4 animate-spin" />
                        <span>Testing...</span>
                      </>
                    ) : (
                      <>
                        <RefreshCw className="w-4 h-4" />
                        <span>Test Connection</span>
                      </>
                    )}
                  </button>
                  <button
                    onClick={() => handleDeleteConnection(connection.id)}
                    className="bg-slate-700/50 border border-slate-600/50 text-red-400 hover:text-red-300 hover:border-red-500/30 p-2 rounded-lg transition-all duration-200"
                    title="Delete connection"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                  <button
                    className="bg-slate-700/50 border border-slate-600/50 text-slate-300 hover:text-white hover:border-slate-500/50 p-2 rounded-lg transition-all duration-200"
                    title="Settings"
                  >
                    <Settings className="w-4 h-4" />
                  </button>
                </div>
              </div>
            );
          })
        ) : (
          <div className="col-span-2 bg-slate-800/50 backdrop-blur-sm rounded-xl p-8 border border-slate-700/50 text-center">
            <Globe className="w-16 h-16 text-slate-600 mx-auto mb-4" />
            <h3 className="text-xl font-bold text-white mb-2">No Connections Found</h3>
            <p className="text-slate-400 mb-6">
              {searchTerm || selectedCategory !== 'all'
                ? "No connections match your search criteria. Try adjusting your filters."
                : "You haven't added any web connections yet. Add a connection to get started."}
            </p>
            <button
              onClick={handleAddConnection}
              className="bg-gradient-to-r from-purple-500 to-blue-500 text-white px-6 py-3 rounded-lg font-medium hover:from-purple-600 hover:to-blue-600 transition-all duration-200 inline-flex items-center space-x-2"
            >
              <Plus className="w-5 h-5" />
              <span>Add Connection</span>
            </button>
          </div>
        )}
      </div>

      {/* Add Connection Modal */}
      {isAddModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          {/* Backdrop */}
          <div 
            className="absolute inset-0 bg-black bg-opacity-70 backdrop-blur-sm"
            onClick={() => setIsAddModalOpen(false)}
          ></div>
          
          {/* Modal */}
          <div className="relative bg-slate-800 rounded-xl border border-slate-700 w-full max-w-2xl max-h-[90vh] overflow-hidden">
            {/* Header */}
            <div className="flex items-center justify-between p-6 border-b border-slate-700">
              <div className="flex items-center space-x-3">
                <div className="p-2 bg-gradient-to-r from-blue-500/20 to-cyan-500/20 rounded-lg border border-blue-500/30">
                  <Globe className="w-5 h-5 text-blue-400" />
                </div>
                <h2 className="text-xl font-bold text-white">Add Web Connection</h2>
              </div>
              <button
                onClick={() => setIsAddModalOpen(false)}
                aria-label="Close"
                className="p-2 text-slate-400 hover:text-white transition-colors duration-200"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            
            {/* Content */}
            <div className="p-6 overflow-y-auto max-h-[calc(90vh-140px)]">
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Connection Name
                  </label>
                  <input
                    type="text"
                    placeholder="Enter a name for this connection"
                    className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Service Provider
                  </label>
                  <select className="w-full bg-slate-700/50 border border-slate-600/50 rounded-lg px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-blue-500">
                    <option value="">Select a provider</option>
                    <option value="google">Google</option>
                    <option value="microsoft">Microsoft</option>
                    <option value="salesforce">Salesforce</option>
                    <option value="dropbox">Dropbox</option>
                    <option value="slack">Slack</option>
                    <option value="shopify">Shopify</option>
                    <option value="custom">Custom API</option>
                  </select>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Authentication Type
                  </label>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30 cursor-pointer hover:border-blue-500/30 transition-colors duration-200">
                      <div className="flex items-center space-x-3 mb-2">
                        <input type="radio" name="auth_type" value="oauth" className="text-blue-500" />
                        <h4 className="text-white font-medium">OAuth 2.0</h4>
                      </div>
                      <p className="text-slate-400 text-sm">Connect using secure OAuth authentication flow.</p>
                    </div>
                    <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30 cursor-pointer hover:border-blue-500/30 transition-colors duration-200">
                      <div className="flex items-center space-x-3 mb-2">
                        <input type="radio" name="auth_type" value="api_key" className="text-blue-500" />
                        <h4 className="text-white font-medium">API Key</h4>
                      </div>
                      <p className="text-slate-400 text-sm">Connect using an API key or token.</p>
                    </div>
                  </div>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Scopes
                  </label>
                  <div className="p-4 bg-slate-700/30 rounded-lg border border-slate-600/30">
                    <p className="text-slate-400 text-sm mb-3">Select the permissions this connection will have:</p>
                    <div className="space-y-2">
                      <div className="flex items-center">
                        <input type="checkbox" id="scope_read" className="mr-2" />
                        <label htmlFor="scope_read" className="text-white text-sm">Read Data</label>
                      </div>
                      <div className="flex items-center">
                        <input type="checkbox" id="scope_write" className="mr-2" />
                        <label htmlFor="scope_write" className="text-white text-sm">Write Data</label>
                      </div>
                      <div className="flex items-center">
                        <input type="checkbox" id="scope_delete" className="mr-2" />
                        <label htmlFor="scope_delete" className="text-white text-sm">Delete Data</label>
                      </div>
                    </div>
                  </div>
                </div>
                
                <div className="p-4 bg-blue-500/10 border border-blue-500/20 rounded-lg">
                  <div className="flex items-start space-x-3">
                    <ExternalLink className="w-5 h-5 text-blue-400 mt-0.5" />
                    <div>
                      <p className="text-blue-400 font-medium">Secure Authentication</p>
                      <p className="text-blue-300/70 text-sm">
                        Authentication is handled securely through picaos.com AuthKit. Your credentials are never stored directly in this application.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            
            {/* Footer */}
            <div className="p-6 border-t border-slate-700 bg-slate-800/80">
              <div className="flex items-center justify-between">
                <button
                  onClick={() => setIsAddModalOpen(false)}
                  className="px-4 py-2 bg-slate-700/50 text-slate-300 rounded-lg hover:bg-slate-700 transition-colors duration-200"
                >
                  Cancel
                </button>
                
                <button
                  className="px-6 py-2 bg-gradient-to-r from-blue-500 to-cyan-500 text-white rounded-lg font-medium hover:from-blue-600 hover:to-cyan-600 transition-all duration-200 flex items-center space-x-2"
                >
                  <ExternalLink className="w-4 h-4" />
                  <span>Connect Service</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default WebConnections;