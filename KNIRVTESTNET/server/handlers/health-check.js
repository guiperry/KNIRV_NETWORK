/**
 * Health Check Handler
 * Provides comprehensive health checking for all KNIRV services
 */

const axios = require('axios');

const getHealth = async (req, res) => {
  try {
    const { loadEndpoints } = require('../../scripts/load-endpoints');
    const { endpoints } = loadEndpoints(process.env.NODE_ENV || 'testnet');
    
    const healthStatus = {
      status: 'healthy',
      timestamp: new Date().toISOString(),
      environment: process.env.NODE_ENV || 'testnet',
      version: require('../../package.json').version,
      services: {},
      summary: {
        total: 0,
        healthy: 0,
        unhealthy: 0,
        unknown: 0
      }
    };

    // Check faucet service first
    const faucetStartTime = Date.now();
    try {
      const faucetResponse = await axios.get('http://localhost:10000/api/faucet/health', {
        timeout: 5000,
        validateStatus: (status) => status < 500
      });

      const faucetResponseTime = Date.now() - faucetStartTime;

      healthStatus.services['testnet-faucet'] = {
        status: faucetResponse.status === 200 ? 'healthy' : 'degraded',
        url: 'http://localhost:10000/api/faucet',
        responseTime: `${faucetResponseTime}ms`,
        httpStatus: faucetResponse.status,
        lastCheck: new Date().toISOString(),
        details: faucetResponse.data || {}
      };

      if (faucetResponse.status === 200) {
        healthStatus.summary.healthy++;
      } else {
        healthStatus.summary.unhealthy++;
      }
    } catch (error) {
      const faucetResponseTime = Date.now() - faucetStartTime;

      healthStatus.services['testnet-faucet'] = {
        status: 'unhealthy',
        url: 'http://localhost:10000/api/faucet',
        responseTime: `${faucetResponseTime}ms`,
        error: error.message,
        lastCheck: new Date().toISOString()
      };

      healthStatus.summary.unhealthy++;
    }

    healthStatus.summary.total++;

    // Check each service endpoint
    const serviceChecks = Object.entries(endpoints).map(async ([serviceName, serviceUrl]) => {
      const startTime = Date.now();

      try {
        const response = await axios.get(`${serviceUrl}/health`, {
          timeout: 5000,
          validateStatus: (status) => status < 500
        });

        const responseTime = Date.now() - startTime;

        healthStatus.services[serviceName] = {
          status: 'healthy',
          url: serviceUrl,
          responseTime: `${responseTime}ms`,
          httpStatus: response.status,
          lastCheck: new Date().toISOString()
        };

        healthStatus.summary.healthy++;
      } catch (error) {
        const responseTime = Date.now() - startTime;
        
        healthStatus.services[serviceName] = {
          status: 'unhealthy',
          url: serviceUrl,
          error: error.message,
          responseTime: `${responseTime}ms`,
          lastCheck: new Date().toISOString()
        };
        
        healthStatus.summary.unhealthy++;
      }
      
      healthStatus.summary.total++;
    });

    await Promise.all(serviceChecks);

    // Determine overall status
    if (healthStatus.summary.unhealthy > 0) {
      healthStatus.status = healthStatus.summary.healthy > 0 ? 'degraded' : 'unhealthy';
    }

    const statusCode = healthStatus.status === 'healthy' ? 200 :
                      healthStatus.status === 'degraded' ? 207 : 503;

    res.status(statusCode).json(healthStatus);
  } catch (error) {
    res.status(500).json({
      status: 'error',
      error: 'Health check failed',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
};

const getServiceHealth = async (req, res) => {
  const serviceName = req.params.service.toUpperCase();
  const { loadEndpoints } = require('../../scripts/load-endpoints');
  const { endpoints } = loadEndpoints(process.env.NODE_ENV || 'testnet');
  
  const serviceKey = `${serviceName}_API`;
  const serviceUrl = endpoints[serviceKey];

  if (!serviceUrl) {
    return res.status(404).json({
      error: 'Service not found',
      service: serviceName,
      availableServices: Object.keys(endpoints)
        .map(key => key.replace('_API', '').toLowerCase())
    });
  }

  const startTime = Date.now();
  
  try {
    const response = await axios.get(`${serviceUrl}/health`, {
      timeout: 10000,
      validateStatus: (status) => status < 500
    });
    
    const responseTime = Date.now() - startTime;
    
    res.json({
      service: serviceName,
      status: 'healthy',
      url: serviceUrl,
      responseTime: `${responseTime}ms`,
      httpStatus: response.status,
      response: response.data,
      lastCheck: new Date().toISOString()
    });
  } catch (error) {
    const responseTime = Date.now() - startTime;
    
    res.status(503).json({
      service: serviceName,
      status: 'unhealthy',
      url: serviceUrl,
      error: error.message,
      responseTime: `${responseTime}ms`,
      lastCheck: new Date().toISOString()
    });
  }
};

module.exports = {
  getHealth,
  getServiceHealth
};
