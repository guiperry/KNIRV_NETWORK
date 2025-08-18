const express = require('express');
const router = express.Router();
const axios = require('axios');

// Main health check endpoint
router.get('/', async (req, res) => {
  const { loadEndpoints } = require('../../scripts/load-endpoints');
  const { endpoints } = loadEndpoints(process.env.NODE_ENV || 'testnet');
  
  const healthStatus = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV || 'testnet',
    services: {},
    summary: {
      total: 0,
      healthy: 0,
      unhealthy: 0,
      unknown: 0
    }
  };

  // Check each service
  const serviceChecks = Object.entries(endpoints).map(async ([serviceName, serviceUrl]) => {
    try {
      const response = await axios.get(`${serviceUrl}/health`, { 
        timeout: 5000,
        validateStatus: (status) => status < 500 // Accept 4xx as "reachable"
      });
      
      healthStatus.services[serviceName] = {
        status: 'healthy',
        url: serviceUrl,
        responseTime: response.headers['x-response-time'] || 'unknown',
        lastCheck: new Date().toISOString()
      };
      healthStatus.summary.healthy++;
    } catch (error) {
      healthStatus.services[serviceName] = {
        status: 'unhealthy',
        url: serviceUrl,
        error: error.message,
        lastCheck: new Date().toISOString()
      };
      healthStatus.summary.unhealthy++;
    }
    healthStatus.summary.total++;
  });

  await Promise.all(serviceChecks);

  // Determine overall status
  if (healthStatus.summary.unhealthy > 0) {
    healthStatus.status = 'degraded';
  }
  if (healthStatus.summary.healthy === 0) {
    healthStatus.status = 'unhealthy';
  }

  const statusCode = healthStatus.status === 'healthy' ? 200 : 
                    healthStatus.status === 'degraded' ? 207 : 503;

  res.status(statusCode).json(healthStatus);
});

// Individual service health check
router.get('/:service', async (req, res) => {
  const { loadEndpoints } = require('../../scripts/load-endpoints');
  const { endpoints } = loadEndpoints(process.env.NODE_ENV || 'testnet');
  
  const serviceName = req.params.service.toUpperCase();
  const serviceKey = `${serviceName}_API`;
  const serviceUrl = endpoints[serviceKey];

  if (!serviceUrl) {
    return res.status(404).json({
      error: 'Service not found',
      service: serviceName,
      availableServices: Object.keys(endpoints).map(key => key.replace('_API', '').toLowerCase())
    });
  }

  try {
    const response = await axios.get(`${serviceUrl}/health`, { 
      timeout: 10000,
      validateStatus: (status) => status < 500
    });
    
    res.json({
      service: serviceName,
      status: 'healthy',
      url: serviceUrl,
      response: response.data,
      responseTime: response.headers['x-response-time'] || 'unknown',
      lastCheck: new Date().toISOString()
    });
  } catch (error) {
    res.status(503).json({
      service: serviceName,
      status: 'unhealthy',
      url: serviceUrl,
      error: error.message,
      lastCheck: new Date().toISOString()
    });
  }
});

// Detailed system health
router.get('/system/detailed', (req, res) => {
  const os = require('os');
  const process = require('process');

  res.json({
    system: {
      platform: os.platform(),
      arch: os.arch(),
      nodeVersion: process.version,
      uptime: process.uptime(),
      memory: {
        total: os.totalmem(),
        free: os.freemem(),
        used: process.memoryUsage()
      },
      cpu: {
        cores: os.cpus().length,
        loadAverage: os.loadavg()
      }
    },
    application: {
      environment: process.env.NODE_ENV || 'testnet',
      pid: process.pid,
      uptime: process.uptime(),
      version: require('../../package.json').version
    },
    timestamp: new Date().toISOString()
  });
});

module.exports = router;
