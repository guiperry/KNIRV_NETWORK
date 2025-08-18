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

// Serve application static files
app.use('/graphchain-explorer', express.static(path.join(__dirname, '../graphchain-explorer')));
app.use('/nexus-portal', express.static(path.join(__dirname, '../nexus-portal/dist')));
app.use('/agent-developer-portal', express.static(path.join(__dirname, '../agent-developer-portal')));
app.use('/nanda-ans', express.static(path.join(__dirname, '../nanda_ans/.next')));

// API Routes
app.use('/api', apiRoutes);
app.use('/health', healthRoutes);
app.use('/gateway', gatewayRoutes);
app.use('/forum', forumRoutes);
app.use('/support', supportRoutes);

// Application routes
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, '../index.html'));
});

app.get('/health-monitor', (req, res) => {
  res.sendFile(path.join(__dirname, '../health-monitor.html'));
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

// Nexus Portal routes (SPA)
app.get('/nexus-portal', (req, res) => {
  res.sendFile(path.join(__dirname, '../nexus-portal/dist/index.html'));
});

app.get('/nexus-portal/*', (req, res) => {
  res.sendFile(path.join(__dirname, '../nexus-portal/dist/index.html'));
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
  // For Next.js, we'll serve a placeholder or redirect to the Next.js server
  res.json({
    message: 'NANDA ANS - Agent Naming Service',
    status: 'Available',
    description: 'Next.js application for agent registration and naming',
    note: 'This service runs on a separate port in production'
  });
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
