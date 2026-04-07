// KNIRV TESTNET Gateway Testnet Status Function
// Provides comprehensive testnet status and metrics

const { loadConfig } = require('./config-loader');

// Get testnet metrics from services
async function getServiceMetrics(serviceName, endpoint) {
  try {
    const response = await fetch(endpoint, {
      method: 'GET',
      timeout: 5000
    });
    
    if (response.ok) {
      const data = await response.json();
      return {
        service: serviceName,
        status: 'responsive',
        data: data,
        timestamp: new Date().toISOString()
      };
    } else {
      return {
        service: serviceName,
        status: 'error',
        error: `HTTP ${response.status}`,
        timestamp: new Date().toISOString()
      };
    }
  } catch (error) {
    return {
      service: serviceName,
      status: 'unreachable',
      error: error.message,
      timestamp: new Date().toISOString()
    };
  }
}

// Main handler
exports.handler = async (event, context) => {
  console.log('Gateway testnet status requested:', event.path);
  
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
    
    // Collect metrics from all services
    const metricsPromises = [
      getServiceMetrics('knirvoracle', 'http://localhost:1317/health'),
      getServiceMetrics('knirvchain', 'http://localhost:8090/health'),
      getServiceMetrics('knirvgraph', 'http://localhost:8082/health'),
      getServiceMetrics('knirvserver', 'http://localhost:8084/health'),
      getServiceMetrics('knirvrouter', 'http://localhost:8086/health'),
      getServiceMetrics('knirvcontroller', 'http://localhost:8089/health')
    ];

    const serviceMetrics = await Promise.all(metricsPromises);
    
    // Get blockchain status from KNIRVROUTER
    let blockchainStatus = null;
    try {
      const blockchainResponse = await fetch('http://localhost:8086/blockchain', {
        method: 'GET',
        timeout: 5000
      });
      if (blockchainResponse.ok) {
        blockchainStatus = await blockchainResponse.json();
      }
    } catch (error) {
      console.log('Could not fetch blockchain status:', error.message);
    }

    // Calculate testnet health
    const responsiveServices = serviceMetrics.filter(s => s.status === 'responsive').length;
    const totalServices = serviceMetrics.length;
    const healthPercentage = Math.round((responsiveServices / totalServices) * 100);
    
    const testnetStatus = healthPercentage >= 80 ? 'healthy' : 
                         healthPercentage >= 50 ? 'degraded' : 'critical';

    // Build comprehensive testnet status
    const statusResponse = {
      testnet: {
        status: testnetStatus,
        health_percentage: healthPercentage,
        timestamp: new Date().toISOString(),
        uptime: process.uptime ? Math.floor(process.uptime()) : 'unknown',
        environment: 'development',
        version: '1.0.0'
      },
      services: {
        total: totalServices,
        responsive: responsiveServices,
        unreachable: totalServices - responsiveServices,
        details: serviceMetrics
      },
      blockchain: blockchainStatus ? {
        status: 'active',
        blocks: blockchainStatus.block_chain ? blockchainStatus.block_chain.length : 0,
        latest_block: blockchainStatus.block_chain ? blockchainStatus.block_chain[blockchainStatus.block_chain.length - 1] : null,
        transaction_pool: blockchainStatus.transaction_pool ? blockchainStatus.transaction_pool.length : 0,
        mining_status: blockchainStatus.mining_locked ? 'locked' : 'active'
      } : {
        status: 'unavailable',
        error: 'Could not connect to blockchain service'
      },
      features: {
        testnet_mode: true,
        service_discovery: true,
        health_monitoring: true,
        authentication: true,
        proxy_routing: true,
        sse_support: true,
        unified_binary_architecture: true,
        demo_controller_support: true
      },
      endpoints: {
        gateway_health: '/gateway/health',
        service_discovery: '/gateway/services',
        testnet_status: '/gateway/testnet/status',
        authentication: '/auth/testnet-tokens',
        sse_events: '/testnet-sse'
      },
      testnet_configuration: {
        resource_optimizations: true,
        simplified_security: true,
        mock_validations: true,
        in_memory_storage: true,
        reduced_mining_difficulty: true,
        p2p_disabled: true
      }
    };

    return {
      statusCode: 200,
      headers,
      body: JSON.stringify(statusResponse, null, 2)
    };

  } catch (error) {
    console.error('Gateway testnet status error:', error);
    
    return {
      statusCode: 500,
      headers,
      body: JSON.stringify({
        testnet: {
          status: 'error',
          timestamp: new Date().toISOString(),
          error: error.message
        },
        services: {
          total: 0,
          responsive: 0,
          unreachable: 0,
          details: []
        },
        blockchain: {
          status: 'unavailable',
          error: 'Service error'
        }
      }, null, 2)
    };
  }
};
