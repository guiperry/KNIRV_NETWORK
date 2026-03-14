const express = require('express');
const router = express.Router();

// Import API handlers (migrated from Netlify functions)
const gatewaySSE = require('../handlers/gateway-sse');
const configLoader = require('../handlers/config-loader');
const healthCheck = require('../handlers/health-check');

// Gateway SSE endpoint (replaces netlify/functions/gateway-sse.js)
router.get('/gateway/sse', gatewaySSE.handleSSE);
router.post('/gateway/sse', gatewaySSE.handleSSEPost);

// Configuration endpoints
router.get('/config', configLoader.getConfig);
router.post('/config', configLoader.updateConfig);

// Health check endpoints
router.get('/health', healthCheck.getHealth);
router.get('/health/:service', healthCheck.getServiceHealth);

// KNIRV Service Proxy endpoints
router.all('/chain/*', (req, res) => {
  // Proxy to KNIRVCHAIN
  const targetUrl = process.env.KNIRVCHAIN_API || 'http://localhost:8080';
  proxyRequest(req, res, targetUrl);
});

router.all('/graph/*', (req, res) => {
  // Proxy to KNIRVGRAPH
  const targetUrl = process.env.KNIRVGRAPH_API || 'http://localhost:8081';
  proxyRequest(req, res, targetUrl);
});

router.all('/nexus/*', (req, res) => {
  // Proxy to KNIRVSERVER
  const targetUrl = process.env.KNIRVNEXUS_API || 'http://localhost:8082';
  proxyRequest(req, res, targetUrl);
});

router.all('/oracle/*', (req, res) => {
  // Proxy to KNIRVORACLE
  const targetUrl = process.env.KNIRVORACLE_API || 'http://localhost:1317';
  proxyRequest(req, res, targetUrl);
});

router.all('/router/*', (req, res) => {
  // Proxy to KNIRVROUTER
  const targetUrl = process.env.KNIRVROUTER_API || 'http://localhost:8086';
  proxyRequest(req, res, targetUrl);
});

router.all('/knirvana/*', (req, res) => {
  // Proxy to KNIRVANA
  const targetUrl = process.env.KNIRVANA_API || 'http://localhost:3000';
  proxyRequest(req, res, targetUrl);
});

// Generic proxy function
function proxyRequest(req, res, targetUrl) {
  const axios = require('axios');
  const url = targetUrl + req.path.replace(/^\/api\/[^\/]+/, '');
  
  const config = {
    method: req.method,
    url: url,
    headers: {
      ...req.headers,
      host: undefined, // Remove host header
    },
    timeout: 30000,
  };

  if (req.body && Object.keys(req.body).length > 0) {
    config.data = req.body;
  }

  axios(config)
    .then(response => {
      res.status(response.status);
      Object.entries(response.headers).forEach(([key, value]) => {
        if (key.toLowerCase() !== 'content-encoding') {
          res.set(key, value);
        }
      });
      res.send(response.data);
    })
    .catch(error => {
      console.error(`Proxy error for ${url}:`, error.message);
      if (error.response) {
        res.status(error.response.status).json({
          error: 'Proxy Error',
          message: error.response.data || error.message,
          service: targetUrl
        });
      } else {
        res.status(503).json({
          error: 'Service Unavailable',
          message: `Unable to connect to ${targetUrl}`,
          service: targetUrl
        });
      }
    });
}

// Test endpoints for development
router.get('/test', (req, res) => {
  res.json({
    message: 'KNIRVTESTNET API is working',
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV || 'development'
  });
});

router.get('/endpoints', (req, res) => {
  const { loadEndpoints } = require('../../scripts/load-endpoints');
  const { endpoints, config } = loadEndpoints(process.env.NODE_ENV || 'testnet');
  
  res.json({
    endpoints,
    config: {
      ...config,
      // Don't expose sensitive information
      JWT_SECRET: undefined,
      DATABASE_URL: undefined
    }
  });
});

module.exports = router;
