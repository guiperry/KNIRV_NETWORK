const express = require('express');
const router = express.Router();

// Gateway health endpoint
router.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'KNIRVTESTNET Gateway',
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV || 'testnet',
    version: require('../../package.json').version
  });
});

// Gateway status endpoint
router.get('/status', (req, res) => {
  const { loadEndpoints } = require('../../scripts/load-endpoints');
  const { endpoints, config } = loadEndpoints(process.env.NODE_ENV || 'testnet');
  
  res.json({
    gateway: {
      status: 'operational',
      environment: process.env.NODE_ENV || 'testnet',
      uptime: process.uptime(),
      timestamp: new Date().toISOString()
    },
    endpoints,
    configuration: {
      testnetMode: config.TESTNET_MODE,
      debugMode: config.DEBUG_MODE,
      corsEnabled: config.ENABLE_CORS
    }
  });
});

// Server-Sent Events endpoint (replaces Netlify function)
router.get('/sse', (req, res) => {
  // Set SSE headers
  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Cache-Control'
  });

  // Send initial connection message
  res.write(`data: ${JSON.stringify({
    type: 'connection',
    message: 'Connected to KNIRVTESTNET Gateway SSE',
    timestamp: new Date().toISOString()
  })}\n\n`);

  // Send periodic heartbeat
  const heartbeat = setInterval(() => {
    res.write(`data: ${JSON.stringify({
      type: 'heartbeat',
      timestamp: new Date().toISOString(),
      uptime: process.uptime()
    })}\n\n`);
  }, 30000);

  // Send service status updates
  const statusUpdate = setInterval(async () => {
    try {
      const { loadEndpoints } = require('../../scripts/load-endpoints');
      const { endpoints } = loadEndpoints(process.env.NODE_ENV || 'testnet');
      
      res.write(`data: ${JSON.stringify({
        type: 'status_update',
        services: Object.keys(endpoints),
        timestamp: new Date().toISOString()
      })}\n\n`);
    } catch (error) {
      console.error('SSE status update error:', error);
    }
  }, 60000);

  // Clean up on client disconnect
  req.on('close', () => {
    clearInterval(heartbeat);
    clearInterval(statusUpdate);
  });

  req.on('end', () => {
    clearInterval(heartbeat);
    clearInterval(statusUpdate);
  });
});

// Configuration endpoint
router.get('/config', (req, res) => {
  const { loadEndpoints } = require('../../scripts/load-endpoints');
  const { endpoints, config } = loadEndpoints(process.env.NODE_ENV || 'testnet');
  
  res.json({
    endpoints,
    config: {
      environment: config.DEPLOYMENT_ENV,
      testnetMode: config.TESTNET_MODE,
      debugMode: config.DEBUG_MODE,
      corsEnabled: config.ENABLE_CORS
    },
    timestamp: new Date().toISOString()
  });
});

// Metrics endpoint
router.get('/metrics', (req, res) => {
  const os = require('os');
  
  res.json({
    system: {
      uptime: os.uptime(),
      loadAverage: os.loadavg(),
      memory: {
        total: os.totalmem(),
        free: os.freemem(),
        usage: (os.totalmem() - os.freemem()) / os.totalmem()
      },
      cpu: {
        cores: os.cpus().length,
        usage: os.loadavg()[0] / os.cpus().length
      }
    },
    application: {
      uptime: process.uptime(),
      memory: process.memoryUsage(),
      pid: process.pid,
      version: require('../../package.json').version
    },
    timestamp: new Date().toISOString()
  });
});

module.exports = router;
