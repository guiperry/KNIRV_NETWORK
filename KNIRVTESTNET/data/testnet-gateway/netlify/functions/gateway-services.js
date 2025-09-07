// KNIRV TESTNET Gateway Services Discovery Function
// Provides service discovery and registration for the testnet

const { loadConfig } = require('./config-loader');

// Service registry with testnet configurations
const SERVICE_REGISTRY = {
  'knirvoracle': {
    name: 'KNIRV-ORACLE',
    description: 'Blockchain oracle service for external data integration',
    port: 1317,
    rpc_port: 26657,
    endpoints: {
      api: 'http://localhost:1317',
      rpc: 'http://localhost:26657',
      health: 'http://localhost:1317/health'
    },
    features: ['blockchain', 'oracle', 'consensus'],
    status: 'unknown'
  },
  'knirvchain': {
    name: 'KNIRVCHAIN',
    description: 'Core blockchain service for skill execution and validation',
    port: 8090,
    endpoints: {
      api: 'http://localhost:8090',
      health: 'http://localhost:8090/health',
      testnet: 'http://localhost:8090/testnet/status'
    },
    features: ['blockchain', 'skills', 'validation', 'testnet'],
    status: 'unknown'
  },
  'knirvgraph': {
    name: 'KNIRVGRAPH',
    description: 'Distributed graph database for error context and skill mapping',
    port: 8082,
    endpoints: {
      api: 'http://localhost:8082',
      health: 'http://localhost:8082/health',
      graph: 'http://localhost:8082/graph'
    },
    features: ['graph', 'dht', 'error_context', 'skill_mapping'],
    status: 'unknown'
  },
  'knirvnexus': {
    name: 'KNIRV-NEXUS',
    description: 'Unified DVE (Decentralized Validation Environment) service',
    port: 8084,
    endpoints: {
      frontend: 'http://localhost:8084',
      api: 'http://localhost:8084/api',
      health: 'http://localhost:8084/health',
      version: 'http://localhost:8084/version'
    },
    features: ['dve', 'validation', 'tee_simulation', 'unified_binary'],
    status: 'unknown'
  },
  'knirvrouter': {
    name: 'KNIRV-ROUTER',
    description: 'Network routing and NRN token management service',
    port: 8086,
    endpoints: {
      api: 'http://localhost:8086',
      health: 'http://localhost:8086/health',
      blockchain: 'http://localhost:8086/blockchain'
    },
    features: ['routing', 'nrn_tokens', 'mining', 'testnet'],
    status: 'unknown'
  },
  'knirvcontroller': {
    name: 'KNIRVCONTROLLER',
    description: 'Agent and skill management controller (demo or real)',
    port: 8089,
    fallback_port: 8088,
    endpoints: {
      api: 'http://localhost:8089',
      health: 'http://localhost:8089/health',
      dashboard: 'http://localhost:8089/',
      agents: 'http://localhost:8089/api/agents',
      skills: 'http://localhost:8089/api/skills'
    },
    features: ['agents', 'skills', 'management', 'demo_mode'],
    status: 'unknown'
  }
};

// Check service availability
async function checkServiceAvailability(serviceId, serviceConfig) {
  try {
    const response = await fetch(serviceConfig.endpoints.health, {
      method: 'GET',
      timeout: 3000
    });
    
    if (response.ok) {
      return { ...serviceConfig, status: 'available', last_check: new Date().toISOString() };
    } else {
      return { ...serviceConfig, status: 'unhealthy', last_check: new Date().toISOString() };
    }
  } catch (error) {
    // For KNIRVCONTROLLER, try fallback port
    if (serviceId === 'knirvcontroller' && serviceConfig.fallback_port) {
      try {
        const fallbackUrl = `http://localhost:${serviceConfig.fallback_port}/health`;
        const fallbackResponse = await fetch(fallbackUrl, {
          method: 'GET',
          timeout: 3000
        });
        
        if (fallbackResponse.ok) {
          const updatedConfig = { ...serviceConfig };
          updatedConfig.port = serviceConfig.fallback_port;
          updatedConfig.endpoints.api = `http://localhost:${serviceConfig.fallback_port}`;
          updatedConfig.endpoints.health = fallbackUrl;
          updatedConfig.endpoints.dashboard = `http://localhost:${serviceConfig.fallback_port}/`;
          return { ...updatedConfig, status: 'available', last_check: new Date().toISOString() };
        }
      } catch (fallbackError) {
        // Continue to return unavailable
      }
    }
    
    return { ...serviceConfig, status: 'unavailable', last_check: new Date().toISOString(), error: error.message };
  }
}

// Main handler
exports.handler = async (event, context) => {
  console.log('Gateway services discovery requested:', event.path);
  
  // Set CORS headers
  const headers = {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'Content-Type': 'application/json'
  };

  // Handle preflight requests
  if (event.httpMethod === 'OPTIONS') {
    return {
      statusCode: 200,
      headers,
      body: ''
    };
  }

  try {
    // Load configuration
    const config = await loadConfig();
    
    // Check all services
    const serviceChecks = await Promise.all(
      Object.entries(SERVICE_REGISTRY).map(([id, serviceConfig]) => 
        checkServiceAvailability(id, serviceConfig).then(result => [id, result])
      )
    );

    // Build service registry response
    const services = {};
    serviceChecks.forEach(([id, result]) => {
      services[id] = result;
    });

    // Calculate statistics
    const totalServices = Object.keys(services).length;
    const availableServices = Object.values(services).filter(s => s.status === 'available').length;
    const unhealthyServices = Object.values(services).filter(s => s.status === 'unhealthy').length;
    const unavailableServices = Object.values(services).filter(s => s.status === 'unavailable').length;

    const servicesResponse = {
      discovery: {
        timestamp: new Date().toISOString(),
        environment: 'testnet',
        gateway_version: '1.0.0'
      },
      statistics: {
        total: totalServices,
        available: availableServices,
        unhealthy: unhealthyServices,
        unavailable: unavailableServices,
        availability_percentage: Math.round((availableServices / totalServices) * 100)
      },
      services: services,
      testnet_features: {
        service_discovery: true,
        health_monitoring: true,
        auto_registration: true,
        fallback_detection: true
      }
    };

    return {
      statusCode: 200,
      headers,
      body: JSON.stringify(servicesResponse, null, 2)
    };

  } catch (error) {
    console.error('Gateway services discovery error:', error);
    
    return {
      statusCode: 500,
      headers,
      body: JSON.stringify({
        discovery: {
          timestamp: new Date().toISOString(),
          environment: 'testnet',
          error: error.message
        },
        statistics: {
          total: 0,
          available: 0,
          unhealthy: 0,
          unavailable: 0,
          availability_percentage: 0
        },
        services: {}
      }, null, 2)
    };
  }
};
