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
const NODE_ENV = process.env.NODE_ENV || 'testnet';

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

// Serve static files
app.use('/assets', express.static(path.join(__dirname, '../assets')));
app.use('/images', express.static(path.join(__dirname, '../images')));
app.use('/js', express.static(path.join(__dirname, '../js')));

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
  app.use('/developer-portal', express.static(path.join(__dirname, '../developer-portal')));
  app.use('/nanda-ans', express.static(path.join(__dirname, '../nanda_ans/.next')));
} else {
  // Staging/Production - serve from testnet-gateway with Netlify functions
  app.use('/', express.static(path.join(__dirname, '../data/testnet-gateway')));
  app.use('/nexus-portal', express.static(path.join(__dirname, '../data/knirvnexus/portal')));
}

// API Routes
app.use('/api', apiRoutes);
app.use('/health', healthRoutes);
app.use('/gateway', gatewayRoutes);
app.use('/forum', forumRoutes);
app.use('/support', supportRoutes);

// Application routes - Environment aware
app.get('/', (req, res) => {
  if (environment.isLocal) {
    // Serve from testnet-gateway for local development
    res.sendFile(path.join(__dirname, '../data/testnet-gateway/index.html'));
  } else {
    // Serve from testnet-gateway for staging/production
    res.sendFile(path.join(__dirname, '../data/testnet-gateway/index.html'));
  }
});

app.get('/health-monitor', (req, res) => {
  res.sendFile(path.join(__dirname, '../health-monitor.html'));
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
  if (environment.isLocal) {
    // Serve local NEXUS frontend
    res.sendFile(path.join(__dirname, '../data/knirvnexus/portal/.next/server/app/page.html'));
  } else {
    // Redirect to testnet subdomain
    res.redirect('https://nexus-test.knirv.com');
  }
});

app.get('/nexus-portal/*', (req, res) => {
  if (environment.isLocal) {
    // Serve local NEXUS frontend (SPA routing)
    res.sendFile(path.join(__dirname, '../data/knirvnexus/portal/.next/server/app/page.html'));
  } else {
    // Redirect to testnet subdomain
    res.redirect('https://nexus-test.knirv.com');
  }
});

// Developer Portal routes
app.get('/developer-portal', (req, res) => {
  res.sendFile(path.join(__dirname, '../developer-portal/index.html'));
});

app.get('/developer-portal/*', (req, res) => {
  const filePath = path.join(__dirname, '../developer-portal', req.params[0] || 'index.html');
  if (fs.existsSync(filePath)) {
    res.sendFile(filePath);
  } else {
    res.sendFile(path.join(__dirname, '../developer-portal/index.html'));
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
