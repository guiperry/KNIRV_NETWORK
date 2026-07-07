/**
 * Simple API Server for KNIRVARENA
 * Basic Express server to resolve the immediate connection issue
 */

const express = require('express');
const cors = require('cors');
const { createServer } = require('http');
const { WebSocketServer } = 'ws';

const app = express();
const server = createServer(app);

// Middleware
app.use(cors());
app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ extended: true }));

// Basic health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Basic status endpoint
app.get('/api/status', (req, res) => {
  res.json({ 
    status: 'running',
    version: '1.0.0',
    services: {
      api: 'running',
      database: 'initializing'
    }
  });
});

// Basic cognitive endpoint
app.post('/api/cognitive/hrm/init', (req, res) => {
  res.json({ 
    status: 'initialized',
    message: 'KNIRVARENA cognitive system initialized'
  });
});

// Start server
const PORT = process.env.PORT || 3001;
server.listen(PORT, '0.0.0.0', () => {
  console.log(`KNIRVARENA API Server running on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
  console.log(`API status: http://localhost:${PORT}/api/status`);
});

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully');
  server.close(() => {
    console.log('Process terminated');
  });
});

process.on('SIGINT', () => {
  console.log('SIGINT received, shutting down gracefully');
  server.close(() => {
    console.log('Process terminated');
  });
});