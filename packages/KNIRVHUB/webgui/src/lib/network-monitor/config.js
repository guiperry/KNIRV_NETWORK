// Network Monitor Configuration
// This defines all services to monitor and their health check endpoints

const config = {
  activeNetwork: process.env.ACTIVE_NETWORK || 'production',

  networks: {
    production: {
      name: 'KNIRV Production Network',
      description: 'Main KNIRV production network',
      services: [
        {
          name: 'ipfs',
          url: process.env.IPFS_URL || 'http://localhost:5001',
          healthEndpoint: '/api/v0/version',
          metricsEndpoint: '/api/v0/stats/repo',
          checkInterval: 30000, // 30 seconds
          timeout: 10000, // 10 seconds
          critical: true,
          type: 'ipfs'
        },
        {
          name: 'knirvoracle',
          url: process.env.KNIRVORACLE_URL || 'http://localhost:8083',
          healthEndpoint: '/health',
          metricsEndpoint: '/metrics',
          checkInterval: 30000,
          timeout: 10000,
          critical: true,
          type: 'cosmos'
        },
        {
          name: 'knirvchain',
          url: process.env.KNIRVCHAIN_URL || 'http://localhost:8080',
          healthEndpoint: '/health',
          metricsEndpoint: '/metrics',
          checkInterval: 30000,
          timeout: 10000,
          critical: true,
          type: 'cosmos'
        },
        {
          name: 'knirvgraph',
          url: process.env.KNIRVGRAPH_URL || 'http://localhost:8081',
          healthEndpoint: '/height',
          metricsEndpoint: '/metrics',
          checkInterval: 30000,
          timeout: 10000,
          critical: true,
          type: 'cosmos'
        },
        {
          name: 'knirvnexus',
          url: process.env.KNIRVNEXUS_URL || 'http://localhost:8082',
          healthEndpoint: '/health',
          metricsEndpoint: '/metrics',
          checkInterval: 30000,
          timeout: 10000,
          critical: true,
          type: 'cosmos'
        },
        {
          name: 'knirvrouter',
          url: process.env.KNIRVROUTER_URL || 'http://localhost:5001',
          healthEndpoint: '/status',
          metricsEndpoint: '/metrics',
          checkInterval: 30000,
          timeout: 10000,
          critical: true,
          type: 'p2p'
        }
      ],
      endpoints: {
        prometheus: process.env.PROMETHEUS_URL || 'http://localhost:9090',
        grafana: process.env.GRAFANA_URL || 'http://localhost:3000',
        kibana: process.env.KIBANA_URL || 'http://localhost:5601',
        alertmanager: process.env.ALERTMANAGER_URL || 'http://localhost:9093'
      }
    },
    testnet: {
      name: 'KNIRV Testnet',
      description: 'KNIRV development and testing network',
      services: [
        {
          name: 'ipfs',
          url: process.env.IPFS_TESTNET_URL || 'http://localhost:5001',
          healthEndpoint: '/api/v0/version',
          checkInterval: 30000,
          timeout: 10000,
          critical: true,
          type: 'ipfs'
        },
        {
          name: 'knirvoracle',
          url: process.env.KNIRVORACLE_TESTNET_URL || 'http://localhost:1317',
          healthEndpoint: '/health',
          checkInterval: 30000,
          timeout: 10000,
          critical: true,
          type: 'cosmos'
        }
      ],
      endpoints: {
        prometheus: 'http://localhost:9091',
        grafana: 'http://localhost:3001',
        kibana: 'http://localhost:5602',
        alertmanager: 'http://localhost:9094'
      }
    }
  },

  monitoring: {
    enabled: true,
    checkInterval: 30000, // 30 seconds
    metricsRetention: 30 * 24 * 60 * 60 * 1000, // 30 days in ms
    maxLogEntries: 1000,
    collectSystemMetrics: true
  },

  logging: {
    enabled: true,
    level: process.env.LOG_LEVEL || 'info',
    retention: 7 * 24 * 60 * 60 * 1000, // 7 days in ms
    maxEntries: 10000
  }
};

// Get active network configuration
function getActiveNetwork() {
  const network = config.networks[config.activeNetwork];
  if (!network) {
    throw new Error(`Network '${config.activeNetwork}' not found`);
  }
  return network;
}

// Get service by name
function getService(serviceName) {
  const network = getActiveNetwork();
  const service = network.services.find(s => s.name === serviceName);
  if (!service) {
    throw new Error(`Service '${serviceName}' not found`);
  }
  return service;
}

module.exports = {
  config,
  getActiveNetwork,
  getService
};
