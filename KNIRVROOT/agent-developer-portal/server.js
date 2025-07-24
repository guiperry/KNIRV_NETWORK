#!/usr/bin/env node
/**
 * Developer Portal Server
 * 
 * This Node.js server serves the Next.js static files and provides API endpoints
 * for the KNIRVROOT Developer Portal.
 */

const express = require('express');
const path = require('path');
const fs = require('fs');
const http = require('http');
const cors = require('cors');
const morgan = require('morgan');

// Configuration from environment variables
const HTTP_API_PORT = process.env.HTTP_API_PORT || 3000;
const API_KEY = process.env.API_KEY || 'default-api-key';
const NODE_ENV = process.env.NODE_ENV || 'development';
const CHAIN_ID = process.env.CHAIN_ID || 'unknown';

// Create Express app
const app = express();

// Middleware
app.use(cors());
app.use(express.json());
app.use(morgan('dev'));

// API key middleware for protected routes
const apiKeyAuth = (req, res, next) => {
  const providedKey = req.headers['x-api-key'];
  if (!providedKey || providedKey !== API_KEY) {
    return res.status(401).json({ error: 'Unauthorized: Invalid API key' });
  }
  next();
};

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'developer-portal' });
});

// Info endpoint
app.get('/info', (req, res) => {
  res.json({
    service: 'KNIRVROOT Developer Portal',
    version: '1.0.0',
    chainId: CHAIN_ID,
    environment: NODE_ENV
  });
});

// Protected API routes
app.get('/api/protected/config', apiKeyAuth, (req, res) => {
  res.json({
    chainId: CHAIN_ID,
    apiEndpoint: `http://localhost:${HTTP_API_PORT}`,
    features: {
      nftCapabilities: true,
      daos: true,
      settlement: true
    }
  });
});

// Proxy API requests to the blockchain server
app.use('/api/blockchain', apiKeyAuth, (req, res) => {
  // This would be implemented to proxy requests to the blockchain server
  // For now, return a placeholder response
  res.json({
    message: 'Blockchain API proxy not yet implemented'
  });
});

// Serve static files from the Next.js build
const staticDir = path.join(__dirname, 'static');

// Check if the static directory exists
if (fs.existsSync(staticDir)) {
  app.use(express.static(staticDir));
  
  // For any other routes, serve the index.html file
  app.get('*', (req, res) => {
    res.sendFile(path.join(staticDir, 'index.html'));
  });
} else {
  console.error(`Static directory not found: ${staticDir}`);
  app.get('*', (req, res) => {
    res.status(500).send('Static files not found. Please build the Next.js application first.');
  });
}

// Start the server
const server = http.createServer(app);
server.listen(HTTP_API_PORT, () => {
  console.log(`Developer Portal server running on port ${HTTP_API_PORT}`);
  console.log(`Environment: ${NODE_ENV}`);
  console.log(`Chain ID: ${CHAIN_ID}`);
});

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully');
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('SIGINT received, shutting down gracefully');
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});