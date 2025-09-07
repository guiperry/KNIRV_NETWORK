import React, { useState, useEffect } from 'react';
import {
  Wifi,
  Globe,
  Settings,
  CheckCircle,
  AlertTriangle,
  Loader,
  RefreshCw,
  Network,
  Server,
  Link
} from 'lucide-react';
import { apiService } from '../services/APIService';

export interface NetworkType {
  id: string;
  name: string;
  description: string;
  type: 'local-testnet' | 'local-network' | 'public-testnet' | 'public-network';
  endpoints: Record<string, string>;
  requiresDiscovery: boolean;
  priority: number;
}

export interface ServiceConnection {
  name: string;
  status: 'connected' | 'disconnected' | 'connecting' | 'error';
  url?: string;
  lastChecked: Date;
  error?: string;
}

interface NetworkSelectorProps {
  isOpen: boolean;
  onClose: () => void;
  onNetworkChange?: (network: NetworkType) => void;
  currentNetwork?: NetworkType;
}

const NetworkSelector: React.FC<NetworkSelectorProps> = ({
  isOpen,
  onClose,
  onNetworkChange,
  currentNetwork
}) => {
  // Pre-defined network configurations
  const preDefinedNetworks: NetworkType[] = [
    {
      id: 'local-testnet',
      name: 'Local Testnet',
      description: 'Connect to local KNIRV testnet services via IP scanning',
      type: 'local-testnet',
      endpoints: {},
      requiresDiscovery: true,
      priority: 1
    },
    {
      id: 'local-network',
      name: 'Local Network',
      description: 'Connect to services on your local network',
      type: 'local-network',
      endpoints: {},
      requiresDiscovery: true,
      priority: 2
    },
    {
      id: 'public-testnet',
      name: 'Public Testnet',
      description: 'Official KNIRV public testnet',
      type: 'public-testnet',
      endpoints: {
        KNIRVCHAIN_API: 'https://chain-test.knirv.com',
        KNIRVGRAPH_API: 'https://graph-test.knirv.com',
        KNIRVNEXUS_API: 'https://nexus-test.knirv.com',
        KNIRVROUTER_API: 'https://router-test.knirv.com',
        KNIRVANA_API: 'https://knirvana-test.knirv.com'
      },
      requiresDiscovery: false,
      priority: 3
    },
    {
      id: 'public-network',
      name: 'Public Network',
      description: 'KNIRV main public network',
      type: 'public-network',
      endpoints: {
        KNIRVCHAIN_API: 'https://chain.knirv.com',
        KNIRVGRAPH_API: 'https://graph.knirv.com',
        KNIRVNEXUS_API: 'https://nexus.knirv.com',
        KNIRVROUTER_API: 'https://router.knirv.com',
        KNIRVANA_API: 'https://knirvana.knirv.com'
      },
      requiresDiscovery: false,
      priority: 4
    }
  ];

  const [networks] = useState<NetworkType[]>(preDefinedNetworks);
  const [serviceConnections, setServiceConnections] = useState<Record<string, ServiceConnection>>({});
  const [selectedNetwork, setSelectedNetwork] = useState<NetworkType | null>(currentNetwork || null);
  const [isScanning, setIsScanning] = useState(false);
  const [isConnecting, setIsConnecting] = useState<string | null>(null);
  const [discoveredServices, setDiscoveredServices] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen && networks.length === 0) {
      checkCurrentConnections();
    }
  }, [isOpen, networks.length]);

  // Future enhancement: Add backend network config loading
  // const loadNetworksFromBackend = async () => {
  //   try {
  //     const response = await apiService.get('/network/config');
  //     if (response.success && response.data?.networks) {
  //       setNetworks(response.data.networks);
  //     } else {
  //       throw new Error('Failed to load network configurations');
  //     }
  //   } catch (error) {
  //     console.log('Using predefined networks:', error);
  //     setNetworks(preDefinedNetworks);
  //   }
  // };

  const checkCurrentConnections = async () => {
    const servicesToCheck = [
      'KNIRVCHAIN_API',
      'KNIRVGRAPH_API',
      'KNIRVNEXUS_API',
      'KNIRVROUTER_API',
      'KNIRVANA_API'
    ];

    const connections: Record<string, ServiceConnection> = {};

    for (const service of servicesToCheck) {
      try {
        const response = await apiService.get('/health', { timeout: 3000 });
        connections[service] = {
          name: service,
          status: response.success ? 'connected' : 'error',
          lastChecked: new Date()
        };
        } catch {
          connections[service] = {
            name: service,
            status: 'error',
            lastChecked: new Date(),
            error: 'Connection failed'
          };
        }
    }

    setServiceConnections(connections);
  };

  const scanLocalNetwork = async (networkType: 'testnet' | 'network') => {
    setIsScanning(true);
    setError(null);

    try {
      // For demo purposes - in real implementation this would use WebRTC or similar
      // to scan local network for KNIRV services
      const scannedServices: string[] = [];

      if (networkType === 'testnet') {
        // Common local testnet ports
        const testnetPorts = [8545, 30303, 4000, 5000, 8080];
        for (const port of testnetPorts) {
          scannedServices.push(`http://localhost:${port}`);
        }
      } else {
        // Common local network ports
        const networkPorts = [3000, 4000, 5000, 8080, 9000];
        // In real implementation, scan local subnet
        for (const port of networkPorts) {
          scannedServices.push(`http://192.168.1.100:${port}`);
        }
      }

      setDiscoveredServices(scannedServices);

      // Test discovered services
      const connections: Record<string, ServiceConnection> = {};
      for (const service of scannedServices) {
        try {
          const controller = new AbortController();
          const timeoutId = setTimeout(() => controller.abort(), 5000);

          const response = await fetch(service + '/health', {
            signal: controller.signal
          });

          clearTimeout(timeoutId);
          connections[service] = {
            name: service,
            status: response.ok ? 'connected' : 'disconnected',
            url: service,
            lastChecked: new Date()
          };
        } catch {
          connections[service] = {
            name: service,
            status: 'error',
            url: service,
            lastChecked: new Date(),
            error: 'Not reachable'
          };
        }
      }

      setServiceConnections(connections);
    } catch (_error) {
      setError('Network scanning failed');
      console.error('Network scan error:', _error);
    } finally {
      setIsScanning(false);
    }
  };

  const connectToNetwork = async (network: NetworkType) => {
    setIsConnecting(network.id);
    setError(null);

    try {
      let finalEndpoints = network.endpoints;

      // If network requires discovery, use discovered services
      if (network.requiresDiscovery && discoveredServices.length > 0) {
        const discoveredEndpoints: Record<string, string> = {};
        // Map discovered services to KNIRV service names
        discoveredServices.forEach((serviceEndpoint, index) => {
          const serviceNames = ['KNIRVCHAIN_API', 'KNIRVGRAPH_API', 'KNIRVNEXUS_API', 'KNIRVROUTER_API', 'KNIRVANA_API'];
          if (serviceNames[index]) {
            discoveredEndpoints[serviceNames[index]] = serviceEndpoint;
          }
        });
        finalEndpoints = discoveredEndpoints;
      }

      // Update API service with new endpoints
      for (const [, url] of Object.entries(finalEndpoints)) {
        apiService.setBaseURL(url);
        break; // For now, just use first endpoint as base
      }

      // Verify connection
      await checkCurrentConnections();

      // Update selected network
      setSelectedNetwork(network);

      // Notify parent component
      if (onNetworkChange) {
        onNetworkChange(network);
      }

      console.log(`Connected to ${network.name}`);

    } catch (error) {
      setError(`Failed to connect to ${network.name}`);
      console.error('Connection error:', error);
    } finally {
      setIsConnecting(null);
    }
  };

  const getNetworkIcon = (type: NetworkType['type']) => {
    switch (type) {
      case 'local-testnet':
        return <Wifi className="w-5 h-5" />;
      case 'local-network':
        return <Network className="w-5 h-5" />;
      case 'public-testnet':
        return <Globe className="w-5 h-5" />;
      case 'public-network':
        return <Server className="w-5 h-5" />;
      default:
        return <Settings className="w-5 h-5" />;
    }
  };

  const getConnectionStatusColor = (status: ServiceConnection['status']) => {
    switch (status) {
      case 'connected':
        return 'text-green-400';
      case 'connecting':
        return 'text-yellow-400';
      case 'error':
        return 'text-red-400';
      default:
        return 'text-gray-400';
    }
  };

  const getConnectionStatusIcon = (status: ServiceConnection['status']) => {
    switch (status) {
      case 'connected':
        return <CheckCircle className="w-4 h-4 text-green-400" />;
      case 'connecting':
        return <Loader className="w-4 h-4 text-yellow-400 animate-spin" />;
      case 'error':
        return <AlertTriangle className="w-4 h-4 text-red-400" />;
      default:
        return <AlertTriangle className="w-4 h-4 text-gray-400" />;
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-gray-900/95 border border-gray-700/50 rounded-xl w-full max-w-4xl h-[80vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700/50">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
              <Network className="w-5 h-5 text-blue-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">Network Selection</h2>
              <p className="text-sm text-gray-400">Choose your KNIRV network connection</p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={checkCurrentConnections}
              disabled={isScanning}
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all disabled:opacity-50"
              title="Refresh connections"
            >
              <RefreshCw className={`w-4 h-4 ${isScanning ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
            >
              ×
            </button>
          </div>
        </div>

        <div className="flex h-full">
          {/* Network List */}
          <div className="w-80 border-r border-gray-700/50 p-4">
            <h3 className="text-sm font-semibold text-gray-300 uppercase tracking-wide mb-4">Available Networks</h3>

            {error && (
              <div className="mb-4 p-3 bg-red-500/20 border border-red-500/30 rounded-lg">
                <p className="text-sm text-red-400">{error}</p>
              </div>
            )}

            <div className="space-y-2">
              {networks
                .sort((a, b) => a.priority - b.priority)
                .map((network) => (
                  <button
                    key={network.id}
                    onClick={() => setSelectedNetwork(network)}
                    disabled={isConnecting === network.id}
                    className={`w-full p-3 rounded-lg border transition-all text-left ${
                      selectedNetwork?.id === network.id
                        ? 'bg-blue-500/10 border-blue-500/30 text-blue-400'
                        : 'bg-gray-800/50 border-gray-700/50 text-gray-400 hover:text-white hover:bg-gray-700/30'
                    } disabled:opacity-50 disabled:cursor-not-allowed`}
                  >
                    <div className="flex items-center space-x-3 mb-2">
                      <div className={`p-1 rounded ${selectedNetwork?.id === network.id ? 'bg-blue-500/20' : 'bg-gray-700/50'}`}>
                        {getNetworkIcon(network.type)}
                      </div>
                      <div className="flex-1 min-w-0">
                        <h4 className="font-medium text-sm">{network.name}</h4>
                        <p className="text-xs opacity-80 line-clamp-2">{network.description}</p>
                      </div>
                      {isConnecting === network.id && (
                        <Loader className="w-4 h-4 animate-spin text-blue-400" />
                      )}
                    </div>
                  </button>
                ))}
            </div>

            {/* Network Actions */}
            <div className="mt-6 pt-6 border-t border-gray-700/50">
              {selectedNetwork?.requiresDiscovery && (
                <button
                  onClick={() => scanLocalNetwork(selectedNetwork.type === 'local-testnet' ? 'testnet' : 'network')}
                  disabled={isScanning}
                  className="w-full p-3 bg-purple-500/20 hover:bg-purple-500/30 border border-purple-500/30 text-purple-400 hover:text-purple-300 rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <div className="flex items-center justify-center space-x-2">
                    {isScanning ? (
                      <Loader className="w-4 h-4 animate-spin" />
                    ) : (
                      <Wifi className="w-4 h-4" />
                    )}
                    <span className="text-sm">
                      {isScanning ? 'Scanning...' : `Scan ${selectedNetwork.type === 'local-testnet' ? 'Testnet' : 'Network'}`}
                    </span>
                  </div>
                </button>
              )}

              {selectedNetwork && (
                <button
                  onClick={() => connectToNetwork(selectedNetwork)}
                  disabled={isConnecting === selectedNetwork.id}
                  className="w-full mt-2 p-3 bg-green-500/20 hover:bg-green-500/30 border border-green-500/30 text-green-400 hover:text-green-300 rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <div className="flex items-center justify-center space-x-2">
                    {isConnecting === selectedNetwork.id ? (
                      <Loader className="w-4 h-4 animate-spin" />
                    ) : (
                      <Link className="w-4 h-4" />
                    )}
                    <span className="text-sm">
                      {isConnecting === selectedNetwork.id ? 'Connecting...' : 'Connect'}
                    </span>
                  </div>
                </button>
              )}
            </div>
          </div>

          {/* Network Details and Service Status */}
          <div className="flex-1 p-6 overflow-y-auto">
            {selectedNetwork ? (
              <div className="space-y-6">
                <div>
                  <h3 className="text-lg font-semibold text-white mb-2">{selectedNetwork.name}</h3>
                  <p className="text-sm text-gray-400">{selectedNetwork.description}</p>
                </div>

                {/* Current Network Status */}
                {selectedNetwork.id === selectedNetwork?.id && (
                  <div className="p-4 bg-green-500/10 border border-green-500/20 rounded-lg">
                    <div className="flex items-center space-x-2">
                      <CheckCircle className="w-5 h-5 text-green-400" />
                      <span className="text-white font-medium">Connected</span>
                    </div>
                  </div>
                )}

                {/* Service Connections */}
                <div>
                  <h4 className="text-md font-semibold text-white mb-4">Service Connections</h4>
                  <div className="space-y-3">
                    {Object.entries(serviceConnections).length === 0 ? (
                      <div className="p-4 bg-gray-800/50 border border-gray-700/50 rounded-lg">
                        <p className="text-sm text-gray-400 text-center">No services checked yet</p>
                      </div>
                    ) : (
                      Object.entries(serviceConnections).map(([serviceName, connection]) => (
                        <div
                          key={serviceName}
                          className="flex items-center justify-between p-3 bg-gray-800/50 border border-gray-700/50 rounded-lg"
                        >
                          <div className="flex items-center space-x-3">
                            {getConnectionStatusIcon(connection.status)}
                            <div>
                              <p className="text-white font-medium">{serviceName.replace('_', ' ')}</p>
                              {connection.url && (
                                <p className="text-xs text-gray-400">{connection.url}</p>
                              )}
                              {connection.error && (
                                <p className="text-xs text-red-400">{connection.error}</p>
                              )}
                            </div>
                          </div>
                          <div className="text-right">
                            <p className={`text-sm font-medium ${getConnectionStatusColor(connection.status)}`}>
                              {connection.status.charAt(0).toUpperCase() + connection.status.slice(1)}
                            </p>
                            <p className="text-xs text-gray-500">
                              {connection.lastChecked.toLocaleTimeString()}
                            </p>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* Discovered Services (for local networks) */}
                {selectedNetwork.requiresDiscovery && discoveredServices.length > 0 && (
                  <div>
                    <h4 className="text-md font-semibold text-white mb-4">Discovered Services</h4>
                    <div className="space-y-2">
                      {discoveredServices.map((discoveredService, index) => (
                        <div
                          key={index}
                          className="p-3 bg-gray-800/50 border border-gray-700/50 rounded-lg"
                        >
                          <p className="text-white font-mono text-sm">{discoveredService}</p>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex items-center justify-center h-full">
                <div className="text-center">
                  <Network className="w-12 h-12 text-gray-600 mx-auto mb-4" />
                  <h3 className="text-lg font-semibold text-gray-400 mb-2">Select a Network</h3>
                  <p className="text-sm text-gray-500">Choose a network from the list to view details</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default NetworkSelector;
