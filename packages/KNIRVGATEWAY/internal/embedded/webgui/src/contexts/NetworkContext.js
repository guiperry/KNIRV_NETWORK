import React, { createContext, useContext, useState, useEffect } from 'react';

// Network configurations
const NETWORK_CONFIGS = {
  mainnet: {
    id: 'mainnet',
    name: 'KNIRV Mainnet',
    description: 'Production KNIRV network',
    endpoints: {
      api: 'https://api.knirv.network',
      websocket: 'wss://ws.knirv.network',
      graphchain: 'https://graphchain.knirv.network',
      oracle: 'https://gateway-oracle.knirv.network'
    },
    chainId: 'knirv-1',
    currency: 'NRN',
    explorerUrl: 'https://explorer.knirv.network',
    status: 'active',
    color: '#007bff'
  },
  testnet: {
    id: 'testnet',
    name: 'KNIRV Testnet',
    description: 'Testing network for development',
    endpoints: {
      api: 'https://testnet-api.knirv.network',
      websocket: 'wss://testnet-ws.knirv.network',
      graphchain: 'https://testnet-graphchain.knirv.network',
      oracle: 'https://testnet-gateway-oracle.knirv.network'
    },
    chainId: 'knirv-testnet-1',
    currency: 'tNRN',
    explorerUrl: 'https://testnet-explorer.knirv.network',
    status: 'active',
    color: '#ffc107'
  },
  local: {
    id: 'local',
    name: 'Local Development',
    description: 'Local development environment',
    endpoints: {
      api: 'http://localhost:8080',
      websocket: 'ws://localhost:8081',
      graphchain: 'http://localhost:3001',
      oracle: 'http://localhost:3002'
    },
    chainId: 'knirv-local',
    currency: 'lNRN',
    explorerUrl: 'http://localhost:3000',
    status: 'development',
    color: '#28a745'
  },
  private: {
    id: 'private',
    name: 'Private Testnet',
    description: 'Private testing environment',
    endpoints: {
      api: 'http://localhost:8080',
      websocket: 'ws://localhost:8081',
      graphchain: 'http://localhost:3001',
      oracle: 'http://localhost:3002'
    },
    chainId: 'knirv-private',
    currency: 'pNRN',
    explorerUrl: 'http://localhost:3000',
    status: 'private',
    color: '#6c757d'
  }
};

const NetworkContext = createContext();

export const useNetwork = () => {
  const context = useContext(NetworkContext);
  if (!context) {
    throw new Error('useNetwork must be used within a NetworkProvider');
  }
  return context;
};

export const NetworkProvider = ({ children }) => {
  const [currentNetwork, setCurrentNetwork] = useState(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState('disconnected');
  const [networkHealth, setNetworkHealth] = useState({});

  // Initialize network from localStorage or default to testnet
  useEffect(() => {
    const savedNetworkId = localStorage.getItem('knirv-selected-network');
    const networkId = savedNetworkId || 'testnet';
    const network = NETWORK_CONFIGS[networkId];
    
    if (network) {
      setCurrentNetwork(network);
      checkNetworkHealth(network);
    }
  }, []);

  // Check network health and connectivity
  const checkNetworkHealth = async (network) => {
    try {
      setIsConnecting(true);
      
      // Check API endpoint
      const apiResponse = await fetch(`${network.endpoints.api}/health`, {
        method: 'GET',
        timeout: 5000
      }).catch(() => null);
      
      const apiHealthy = apiResponse?.ok || false;
      
      // Check WebSocket (simplified check)
      let wsHealthy = false;
      try {
        const ws = new WebSocket(network.endpoints.websocket);
        ws.onopen = () => {
          wsHealthy = true;
          ws.close();
        };
        ws.onerror = () => {
          wsHealthy = false;
        };
      } catch (error) {
        wsHealthy = false;
      }
      
      const health = {
        api: apiHealthy,
        websocket: wsHealthy,
        overall: apiHealthy, // Simplified overall health
        lastChecked: new Date().toISOString()
      };
      
      setNetworkHealth(health);
      setConnectionStatus(health.overall ? 'connected' : 'error');
      
    } catch (error) {
      console.error('Network health check failed:', error);
      setNetworkHealth({
        api: false,
        websocket: false,
        overall: false,
        lastChecked: new Date().toISOString(),
        error: error.message
      });
      setConnectionStatus('error');
    } finally {
      setIsConnecting(false);
    }
  };

  // Switch to a different network
  const switchNetwork = async (networkId) => {
    const network = NETWORK_CONFIGS[networkId];
    if (!network) {
      throw new Error(`Network ${networkId} not found`);
    }
    
    setIsConnecting(true);
    setConnectionStatus('connecting');
    
    try {
      // Save to localStorage
      localStorage.setItem('knirv-selected-network', networkId);
      
      // Update current network
      setCurrentNetwork(network);
      
      // Check health of new network
      await checkNetworkHealth(network);
      
      // Trigger any necessary re-initialization
      window.dispatchEvent(new CustomEvent('networkChanged', {
        detail: { network, previousNetwork: currentNetwork }
      }));
      
    } catch (error) {
      console.error('Failed to switch network:', error);
      setConnectionStatus('error');
      throw error;
    }
  };

  // Get API client configured for current network
  const getApiClient = () => {
    if (!currentNetwork) {
      throw new Error('No network selected');
    }
    
    return {
      baseURL: currentNetwork.endpoints.api,
      websocketURL: currentNetwork.endpoints.websocket,
      graphchainURL: currentNetwork.endpoints.graphchain,
      oracleURL: currentNetwork.endpoints.oracle,
      chainId: currentNetwork.chainId,
      currency: currentNetwork.currency,
      
      // Helper methods
      get: async (path, options = {}) => {
        const response = await fetch(`${currentNetwork.endpoints.api}${path}`, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            ...options.headers
          },
          ...options
        });
        
        if (!response.ok) {
          throw new Error(`API request failed: ${response.statusText}`);
        }
        
        return response.json();
      },
      
      post: async (path, data, options = {}) => {
        const response = await fetch(`${currentNetwork.endpoints.api}${path}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...options.headers
          },
          body: JSON.stringify(data),
          ...options
        });
        
        if (!response.ok) {
          throw new Error(`API request failed: ${response.statusText}`);
        }
        
        return response.json();
      }
    };
  };

  // Refresh network health
  const refreshHealth = () => {
    if (currentNetwork) {
      checkNetworkHealth(currentNetwork);
    }
  };

  const value = {
    // Current state
    currentNetwork,
    isConnecting,
    connectionStatus,
    networkHealth,
    
    // Available networks
    networks: NETWORK_CONFIGS,
    
    // Actions
    switchNetwork,
    refreshHealth,
    getApiClient,
    
    // Utilities
    isNetworkAvailable: (networkId) => !!NETWORK_CONFIGS[networkId],
    getNetworkConfig: (networkId) => NETWORK_CONFIGS[networkId],
    getAllNetworks: () => Object.values(NETWORK_CONFIGS)
  };

  return (
    <NetworkContext.Provider value={value}>
      {children}
    </NetworkContext.Provider>
  );
};

export default NetworkContext;
