#!/usr/bin/env node

/**
 * KNIRVTESTNET Express Server
 * 
 * Replaces Netlify functions with Express.js routes for Render deployment
 * Serves static files and provides API endpoints for all applications
 */

const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const morgan = require('morgan');
const path = require('path');
const fs = require('fs');

// Import route handlers
const apiRoutes = require('./routes/api');
const healthRoutes = require('./routes/health');
const gatewayRoutes = require('./routes/gateway');
const forumRoutes = require('./routes/forum');
const supportRoutes = require('./routes/support');

// Load endpoints configuration
const { loadEndpoints } = require('../scripts/load-endpoints');

const app = express();
const PORT = process.env.PORT || 10000;
const NODE_ENV = process.env.NODE_ENV || 'staging-testnet';

// Load environment-specific endpoints
const { endpoints, config } = loadEndpoints(NODE_ENV);

// Security middleware
app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'", "https://fonts.googleapis.com"],
      fontSrc: ["'self'", "https://fonts.gstatic.com"],
      scriptSrc: ["'self'", "'unsafe-inline'", "'unsafe-eval'"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'", "https:", "wss:"]
    }
  },
  crossOriginEmbedderPolicy: false
}));

// CORS configuration
app.use(cors({
  origin: config.ENABLE_CORS ? '*' : ['https://knirvtestnet.onrender.com'],
  credentials: true,
  methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'Authorization', 'X-Requested-With']
}));

// Compression and logging
app.use(compression());
app.use(morgan('combined'));

// Body parsing middleware
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

// Cache middleware for static assets
const staticOptions = {
  maxAge: '1d', // 1 day cache
  etag: true,
  lastModified: true,
  setHeaders: (res, path) => {
    // Set cache headers based on file type
    if (path.match(/\.(css|js|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$/)) {
      res.setHeader('Cache-Control', 'public, max-age=86400'); // 24 hours
    }
  }
};

// Serve static files with caching
app.use('/assets', express.static(path.join(__dirname, '../assets'), staticOptions));
app.use('/images', express.static(path.join(__dirname, '../images'), staticOptions));
app.use('/js', express.static(path.join(__dirname, '../js'), staticOptions));

// Environment detection
const environment = {
  isLocal: !process.env.RENDER && !process.env.NETLIFY,
  isStaging: !!process.env.RENDER,
  isProduction: !!process.env.NETLIFY
};

// Serve application static files based on environment
if (environment.isLocal) {
  // Local development - serve from testnet-gateway
  app.use('/', express.static(path.join(__dirname, '../data/testnet-gateway')));
  app.use('/nexus-portal', express.static(path.join(__dirname, '../data/knirvnexus/portal')));
  app.use('/graphchain-explorer', express.static(path.join(__dirname, '../graphchain-explorer')));
  app.use('/agent-developer-portal', express.static(path.join(__dirname, '../agent-developer-portal')));
  app.use('/nanda-ans', express.static(path.join(__dirname, '../nanda_ans/.next')));
} else {
  // Staging/Production - serve from testnet-gateway with Netlify functions
  app.use('/', express.static(path.join(__dirname, '../data/testnet-gateway')));
  app.use('/nexus-portal', express.static(path.join(__dirname, '../data/knirvnexus/portal')));
}

// Simple health check for Render.com
app.get('/ping', (req, res) => {
  res.status(200).json({
    status: 'ok',
    timestamp: new Date().toISOString(),
    port: process.env.PORT,
    environment: process.env.NODE_ENV
  });
});

// Gateway routes for staging-testnet
app.get('/gateway/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'KNIRVTESTNET Gateway',
    timestamp: new Date().toISOString(),
    environment: NODE_ENV,
    version: '1.0.0',
    endpoints: Object.keys(endpoints).length
  });
});

app.get('/gateway/services', async (req, res) => {
  try {
    const services = {};
    for (const [name, url] of Object.entries(endpoints)) {
      try {
        const axios = require('axios');
        const response = await axios.get(`${url}/health`, { timeout: 3000 });
        services[name] = {
          url,
          status: 'healthy',
          response_time: response.headers['x-response-time'] || 'unknown'
        };
      } catch (error) {
        services[name] = {
          url,
          status: 'unhealthy',
          error: error.message
        };
      }
    }

    res.json({
      timestamp: new Date().toISOString(),
      environment: NODE_ENV,
      services
    });
  } catch (error) {
    res.status(500).json({
      error: 'Service discovery failed',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

app.get('/gateway/testnet/status', async (req, res) => {
  try {
    const status = {
      testnet: 'KNIRV TESTNET',
      environment: NODE_ENV,
      timestamp: new Date().toISOString(),
      services: {},
      summary: {
        total: 0,
        healthy: 0,
        unhealthy: 0
      }
    };

    // Check each service
    for (const [name, url] of Object.entries(endpoints)) {
      status.summary.total++;
      try {
        const axios = require('axios');
        await axios.get(`${url}/health`, { timeout: 3000 });
        status.services[name] = { status: 'healthy', url };
        status.summary.healthy++;
      } catch (error) {
        status.services[name] = { status: 'unhealthy', url, error: error.message };
        status.summary.unhealthy++;
      }
    }

    res.json(status);
  } catch (error) {
    res.status(500).json({
      error: 'Testnet status check failed',
      message: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

app.get('/auth/testnet-tokens', (req, res) => {
  res.json({
    message: 'Testnet authentication tokens',
    environment: NODE_ENV,
    timestamp: new Date().toISOString(),
    tokens: {
      testnet_access: 'testnet-access-token-placeholder',
      api_key: 'testnet-api-key-placeholder',
      session: 'testnet-session-token-placeholder'
    },
    note: 'These are mock tokens for testnet environment'
  });
});

// API Routes
app.use('/api', apiRoutes);
app.use('/health', healthRoutes);
app.use('/gateway', gatewayRoutes);
app.use('/forum', forumRoutes);
app.use('/support', supportRoutes);

// Application routes - Environment aware
app.get('/', (req, res) => {
  const primaryIndexPath = path.join(__dirname, '../data/testnet-gateway/index.html');
  const fallbackIndexPath = path.join(__dirname, '../index.html');

  // Try primary path first (testnet-gateway)
  if (fs.existsSync(primaryIndexPath)) {
    res.sendFile(primaryIndexPath);
  }
  // Try fallback path (KNIRVTESTNET root)
  else if (fs.existsSync(fallbackIndexPath)) {
    res.sendFile(fallbackIndexPath);
  }
  // Last resort: diagnostics page
  else {
    res.redirect('/diagnostics');
  }
});

// Diagnostics page
app.get('/diagnostics', (req, res) => {
  res.status(200).send(`
    <!DOCTYPE html>
    <html>
    <head>
      <title>KNIRV TESTNET - Diagnostics</title>
      <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #1a1a1a; color: #fff; }
        .container { max-width: 1000px; margin: 0 auto; }
        .header { text-align: center; margin-bottom: 30px; }
        .status { background: #2a2a2a; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .success { color: #4CAF50; }
        .info { color: #2196F3; }
        .warning { color: #FF9800; }
        .error { color: #f44336; }
        .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
        .nav-button {
          position: fixed; top: 20px; right: 20px;
          background: #4CAF50; color: white; padding: 10px 20px;
          border: none; border-radius: 5px; cursor: pointer; text-decoration: none;
        }
        .nav-button:hover { background: #45a049; }
        table { width: 100%; border-collapse: collapse; margin: 10px 0; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #444; }
        th { background: #333; }
        .code { background: #333; padding: 10px; border-radius: 4px; font-family: monospace; }
      </style>
    </head>
    <body>
      <a href="/" class="nav-button">← Back to Site</a>
      <div class="container">
        <div class="header">
          <h1>🔧 KNIRV TESTNET - Server Diagnostics</h1>
          <p class="info">Real-time server status and configuration information</p>
        </div>

        <div class="grid">
          <div class="status">
            <h2 class="success">✅ Server Status</h2>
            <table>
              <tr><td><strong>Status:</strong></td><td class="success">Running</td></tr>
              <tr><td><strong>Environment:</strong></td><td>${process.env.NODE_ENV || 'testnet'}</td></tr>
              <tr><td><strong>Port:</strong></td><td>${process.env.PORT || 'not set'}</td></tr>
              <tr><td><strong>Host:</strong></td><td>${req.get('host') || 'unknown'}</td></tr>
              <tr><td><strong>Protocol:</strong></td><td>${req.protocol}</td></tr>
              <tr><td><strong>URL:</strong></td><td>${req.protocol}://${req.get('host')}</td></tr>
              <tr><td><strong>Timestamp:</strong></td><td>${new Date().toISOString()}</td></tr>
            </table>
          </div>

          <div class="status">
            <h2 class="info">🌐 Render Environment</h2>
            <table>
              <tr><td><strong>Render Service:</strong></td><td>${process.env.RENDER_SERVICE_ID || 'not set'}</td></tr>
              <tr><td><strong>External URL:</strong></td><td>${process.env.RENDER_EXTERNAL_URL || 'not set'}</td></tr>
              <tr><td><strong>Service Name:</strong></td><td>${process.env.RENDER_SERVICE_NAME || 'not set'}</td></tr>
              <tr><td><strong>Git Commit:</strong></td><td>${process.env.RENDER_GIT_COMMIT || 'not set'}</td></tr>
              <tr><td><strong>Git Branch:</strong></td><td>${process.env.RENDER_GIT_BRANCH || 'not set'}</td></tr>
            </table>
          </div>
        </div>

        <div class="status">
          <h2 class="info">📁 File System Status</h2>
          <div class="grid">
            <div>
              <h3>Primary Index Path:</h3>
              <div class="code">${path.join(__dirname, '../data/testnet-gateway/index.html')}</div>
              <p class="${fs.existsSync(path.join(__dirname, '../data/testnet-gateway/index.html')) ? 'success' : 'error'}">
                ${fs.existsSync(path.join(__dirname, '../data/testnet-gateway/index.html')) ? '✅ File exists' : '❌ File not found'}
              </p>
            </div>
            <div>
              <h3>Fallback Index Path:</h3>
              <div class="code">${path.join(__dirname, '../index.html')}</div>
              <p class="${fs.existsSync(path.join(__dirname, '../index.html')) ? 'success' : 'error'}">
                ${fs.existsSync(path.join(__dirname, '../index.html')) ? '✅ File exists' : '❌ File not found'}
              </p>
            </div>
          </div>
        </div>

        <div class="status">
          <h2 class="info">🔗 Available Endpoints</h2>
          <div class="grid">
            <div>
              <h3>Health & Monitoring:</h3>
              <p><a href="/ping" style="color: #4CAF50;">/ping</a> - Simple health check</p>
              <p><a href="/health" style="color: #4CAF50;">/health</a> - Comprehensive health check</p>
              <p><a href="/health-monitor" style="color: #4CAF50;">/health-monitor</a> - Health monitor UI</p>
              <p><a href="/api/health-monitor/status" style="color: #4CAF50;">/api/health-monitor/status</a> - API status</p>
            </div>
            <div>
              <h3>Applications:</h3>
              <p><a href="/nexus-portal" style="color: #4CAF50;">/nexus-portal</a> - NEXUS Portal</p>
              <p><a href="/graphchain-explorer" style="color: #4CAF50;">/graphchain-explorer</a> - GraphChain Explorer</p>
              <p><a href="/agent-developer-portal" style="color: #4CAF50;">/agent-developer-portal</a> - Agent Developer Portal</p>
            </div>
          </div>
        </div>

        <div class="status">
          <h2 class="info">⚙️ System Information</h2>
          <table>
            <tr><td><strong>Node.js Version:</strong></td><td>${process.version}</td></tr>
            <tr><td><strong>Platform:</strong></td><td>${process.platform}</td></tr>
            <tr><td><strong>Architecture:</strong></td><td>${process.arch}</td></tr>
            <tr><td><strong>Working Directory:</strong></td><td>${process.cwd()}</td></tr>
            <tr><td><strong>Memory Usage:</strong></td><td>${Math.round(process.memoryUsage().heapUsed / 1024 / 1024)} MB</td></tr>
            <tr><td><strong>Uptime:</strong></td><td>${Math.round(process.uptime())} seconds</td></tr>
          </table>
        </div>
      </div>
    </body>
    </html>
  `);
});

app.get('/health-monitor', (req, res) => {
  const gatewayHealthMonitor = path.join(__dirname, '../data/testnet-gateway/health-monitor.html');
  const fallbackHealthMonitor = path.join(__dirname, '../health-monitor.html');

  // Try testnet-gateway version first, then fallback
  if (fs.existsSync(gatewayHealthMonitor)) {
    res.sendFile(gatewayHealthMonitor);
  } else if (fs.existsSync(fallbackHealthMonitor)) {
    res.sendFile(fallbackHealthMonitor);
  } else {
    res.status(404).send('Health monitor not found');
  }
});

// Health monitor API endpoint
app.get('/api/health-monitor/status', async (req, res) => {
  try {
    const healthResults = await checkAllServices();
    res.json({
      timestamp: Date.now(),
      services: healthResults,
      overall: Object.values(healthResults).every(s => s.healthy) ? 'healthy' : 'degraded'
    });
  } catch (error) {
    console.error('Health check error:', error);
    res.status(500).json({ error: 'Health check failed' });
  }
});

app.get('/test-navigation', (req, res) => {
  res.sendFile(path.join(__dirname, '../test-navigation.html'));
});

// Redirect old gateway paths to new structure
app.get('/gateway', (req, res) => {
  res.redirect('/');
});

app.get('/testnet', (req, res) => {
  res.redirect('/');
});

// GraphChain Explorer routes
app.get('/graphchain-explorer', (req, res) => {
  res.sendFile(path.join(__dirname, '../graphchain-explorer/index.html'));
});

app.get('/graphchain-explorer/*', (req, res) => {
  const filePath = path.join(__dirname, '../graphchain-explorer', req.params[0] || 'index.html');
  if (fs.existsSync(filePath)) {
    res.sendFile(filePath);
  } else {
    res.sendFile(path.join(__dirname, '../graphchain-explorer/index.html'));
  }
});

// Nexus Portal routes - Serve KNIRVNEXUS native frontend
app.get('/nexus-portal', (req, res) => {
  const nexusIndexPath = path.join(__dirname, '../data/knirvnexus/portal/index.html');
  const nexusNextPath = path.join(__dirname, '../data/knirvnexus/portal/.next/server/app/page.html');

  // Try to serve local NEXUS frontend files
  if (fs.existsSync(nexusIndexPath)) {
    res.sendFile(nexusIndexPath);
  } else if (fs.existsSync(nexusNextPath)) {
    res.sendFile(nexusNextPath);
  } else {
    // Fallback: show nexus portal status
    res.status(200).send(`
      <!DOCTYPE html>
      <html>
      <head>
        <title>KNIRV NEXUS Portal</title>
        <style>
          body { font-family: Arial, sans-serif; margin: 40px; background: #1a1a1a; color: #fff; text-align: center; }
          .container { max-width: 600px; margin: 0 auto; }
          .status { background: #2a2a2a; padding: 20px; border-radius: 8px; margin: 20px 0; }
          .warning { color: #FF9800; }
          .info { color: #2196F3; }
        </style>
      </head>
      <body>
        <div class="container">
          <h1>🔧 KNIRV NEXUS Portal</h1>
          <div class="status">
            <h2 class="warning">⚠️ Portal Not Available</h2>
            <p class="info">The NEXUS portal frontend is not built or deployed.</p>
            <p>Expected paths:</p>
            <p>• ${nexusIndexPath}</p>
            <p>• ${nexusNextPath}</p>
            <p><a href="/" style="color: #4CAF50;">← Back to Testnet</a></p>
          </div>
        </div>
      </body>
      </html>
    `);
  }
});

app.get('/nexus-portal/*', (req, res) => {
  // For SPA routing, redirect to main nexus-portal route
  res.redirect('/nexus-portal');
});

// Agent Developer Portal routes
app.get('/agent-developer-portal', (req, res) => {
  res.sendFile(path.join(__dirname, '../agent-developer-portal/index.html'));
});

app.get('/agent-developer-portal/*', (req, res) => {
  const filePath = path.join(__dirname, '../agent-developer-portal', req.params[0] || 'index.html');
  if (fs.existsSync(filePath)) {
    res.sendFile(filePath);
  } else {
    res.sendFile(path.join(__dirname, '../agent-developer-portal/index.html'));
  }
});

// NANDA ANS routes (Next.js)
app.get('/nanda-ans', (req, res) => {
  res.redirect('https://nanda-test.knirv.com');
});

app.get('/nanda-ans/*', (req, res) => {
  res.redirect('https://nanda-test.knirv.com');
});

app.get('/nanda-ans/*', (req, res) => {
  // For Next.js, we'll serve a placeholder or redirect to the Next.js server
  res.json({
    message: 'NANDA ANS - Agent Naming Service',
    status: 'Available',
    path: req.path,
    description: 'Next.js application for agent registration and naming'
  });
});

// Configuration endpoint
app.get('/config', (req, res) => {
  res.json({
    environment: NODE_ENV,
    endpoints,
    config: {
      ...config,
      // Don't expose sensitive config
      JWT_SECRET: undefined,
      DATABASE_URL: undefined
    }
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    error: 'Not Found',
    message: `Route ${req.originalUrl} not found`,
    timestamp: new Date().toISOString()
  });
});

// Error handler
app.use((err, req, res, next) => {
  console.error('Server Error:', err);
  res.status(500).json({
    error: 'Internal Server Error',
    message: config.DEBUG_MODE ? err.message : 'Something went wrong',
    timestamp: new Date().toISOString()
  });
});

// Function to check all services
async function checkAllServices() {
  // Subdomain to port mapping based on NEXUS config
  const services = environment.isLocal ? {
    knirvoracle: 'http://localhost:1317',
    knirvchain: 'http://localhost:8090',
    knirvgraph: 'http://localhost:8082',
    knirvnexus_gui: 'http://localhost:8082',      // gui_port from config
    knirvnexus_api: 'http://localhost:8083',      // api_port from config
    knirvnexus_tee: 'http://localhost:8182',      // tee_port from config
    knirvrouter: 'http://localhost:8086',
    knirvgateway: 'http://localhost:8888',
    nanda_ans: 'http://localhost:9002'
  } : {
    knirvoracle: 'https://oracle-test.knirv.com',
    knirvchain: 'https://chain-test.knirv.com',
    knirvgraph: 'https://graph-test.knirv.com',
    knirvnexus_gui: 'https://nexus-test.knirv.com',
    knirvnexus_api: 'https://nexus-dve-test.knirv.com',
    knirvnexus_tee: 'https://nexus-validation-test.knirv.com',
    knirvrouter: 'https://router-test.knirv.com',
    knirvgateway: 'https://testnet.knirv.com',
    nanda_ans: 'https://nanda-test.knirv.com'
  };

  const results = {};

  // Check each service in parallel
  const checks = Object.entries(services).map(async ([name, url]) => {
    const result = await checkServiceHealth(name, url);
    results[name] = result;
  });

  await Promise.all(checks);
  return results;
}

// Function to check individual service health
async function checkServiceHealth(serviceName, serviceUrl) {
  const startTime = Date.now();

  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 10000); // 10s timeout

    const response = await fetch(`${serviceUrl}/health`, {
      method: 'GET',
      signal: controller.signal,
      headers: {
        'User-Agent': 'KNIRV-Testnet-Health-Monitor/1.0'
      }
    });

    clearTimeout(timeoutId);

    const responseTime = Date.now() - startTime;
    const healthy = response.ok;

    return {
      name: serviceName,
      healthy,
      status: response.status,
      responseTime,
      lastCheck: new Date().toISOString(),
      url: serviceUrl
    };
  } catch (error) {
    const responseTime = Date.now() - startTime;

    return {
      name: serviceName,
      healthy: false,
      status: 0,
      responseTime,
      lastCheck: new Date().toISOString(),
      url: serviceUrl,
      error: error.message
    };
  }
}

// Start server
app.listen(PORT, '0.0.0.0', () => {
  console.log(`🚀 KNIRVTESTNET Server running on port ${PORT}`);
  console.log(`📍 Environment: ${NODE_ENV}`);
  console.log(`🌐 Endpoints loaded:`, Object.keys(endpoints).length);
  console.log(`🔧 Debug mode: ${config.DEBUG_MODE}`);
  console.log(`🔒 CORS enabled: ${config.ENABLE_CORS}`);
  
  if (config.DEBUG_MODE) {
    console.log('\n📋 Active Endpoints:');
    Object.entries(endpoints).forEach(([key, value]) => {
      console.log(`   ${key}: ${value}`);
    });
  }
});

module.exports = app;
