// KNIRV TESTNET Gateway Health Check Function
// Provides health status for the gateway and all connected services

const { loadConfig } = require('./config-loader');

// Service endpoints configuration
const SERVICES = {
  'knirvoracle': { port: 1317, path: '/health' },
  'knirvchain': { port: 8090, path: '/health' },
  'knirvgraph': { port: 8082, path: '/health' },
  'knirvserver': { port: 8084, path: '/health' },
  'knirvrouter': { port: 8086, path: '/health' },
  'knirvcontroller': { port: 8089, path: '/health' }
};

// Check service health
async function checkServiceHealth(serviceName, config) {
  try {
    const response = await fetch(`http://localhost:${config.port}${config.path}`, {
      method: 'GET',
      timeout: 5000
    });
    
    if (response.ok) {
      const data = await response.json();
      return {
        name: serviceName,
        status: 'healthy',
        port: config.port,
        response: data,
        timestamp: new Date().toISOString()
      };
    } else {
      return {
        name: serviceName,
        status: 'unhealthy',
        port: config.port,
        error: `HTTP ${response.status}`,
        timestamp: new Date().toISOString()
      };
    }
  } catch (error) {
    return {
      name: serviceName,
      status: 'unreachable',
      port: config.port,
      error: error.message,
      timestamp: new Date().toISOString()
    };
  }
}

// Main handler
exports.handler = async (event, context) => {
  console.log('Gateway health check requested:', event.path);
  
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
    
    // Check all services in parallel
    const serviceChecks = await Promise.all(
      Object.entries(SERVICES).map(([name, serviceConfig]) => 
        checkServiceHealth(name, serviceConfig)
      )
    );

    // Calculate overall health
    const healthyServices = serviceChecks.filter(s => s.status === 'healthy').length;
    const totalServices = serviceChecks.length;
    const healthPercentage = Math.round((healthyServices / totalServices) * 100);
    
    const overallStatus = healthPercentage >= 80 ? 'healthy' : 
                         healthPercentage >= 50 ? 'degraded' : 'unhealthy';

    // Gateway health response
    const healthResponse = {
      gateway: {
        status: overallStatus,
        timestamp: new Date().toISOString(),
        version: '1.0.0',
        environment: 'testnet',
        uptime: process.uptime ? Math.floor(process.uptime()) : 'unknown'
      },
      services: {
        total: totalServices,
        healthy: healthyServices,
        unhealthy: totalServices - healthyServices,
        health_percentage: healthPercentage,
        details: serviceChecks
      },
      testnet: {
        mode: 'active',
        features: [
          'service_discovery',
          'health_monitoring',
          'authentication',
          'proxy_routing',
          'sse_support'
        ]
      },
      endpoints: {
        health: '/gateway/health',
        services: '/gateway/services',
        status: '/gateway/testnet/status',
        auth: '/auth/testnet-tokens'
      }
    };

    return {
      statusCode: 200,
      headers,
      body: JSON.stringify(healthResponse, null, 2)
    };

  } catch (error) {
    console.error('Gateway health check error:', error);
    
    return {
      statusCode: 500,
      headers,
      body: JSON.stringify({
        gateway: {
          status: 'error',
          timestamp: new Date().toISOString(),
          error: error.message
        },
        services: {
          total: 0,
          healthy: 0,
          unhealthy: 0,
          health_percentage: 0,
          details: []
        }
      }, null, 2)
    };
  }
};
