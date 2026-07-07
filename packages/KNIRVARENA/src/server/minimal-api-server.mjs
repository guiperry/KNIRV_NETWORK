import { createServer } from 'http';

const server = createServer((req, res) => {
  // Enable CORS
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-API-Key');

  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  // Basic routing
  const url = new URL(req.url || '', `http://localhost:3001`);
  const path = url.pathname;

  try {
    if (path === '/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ 
        status: 'ok', 
        timestamp: new Date().toISOString() 
      }));
      return;
    }

    if (path === '/api/status') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ 
        status: 'running',
        version: '1.0.0',
        services: {
          api: 'running',
          database: 'initializing'
        }
      }));
      return;
    }

    if (path === '/api/cognitive/hrm/init' && req.method === 'POST') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ 
        status: 'initialized',
        message: 'KNIRVARENA cognitive system initialized'
      }));
      return;
    }

    if (path === '/api/knirvbase/initialize' && req.method === 'POST') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ 
        success: true, 
        message: 'KNIRVBASE database initialized successfully',
        dataDir: '/tmp/knirv-data'
      }));
      return;
    }

    // Default 404
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      error: 'Not Found',
      path: path
    }));

  } catch (error) {
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      error: 'Internal Server Error',
      message: error.message
    }));
  }
});

const PORT = process.env.PORT || 3001;
server.listen(PORT, '0.0.0.0', () => {
  console.log(`KNIRVARENA API Server running on port ${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
  console.log(`API status: http://localhost:${PORT}/api/status`);
  console.log(`Cognitive HRM: http://localhost:${PORT}/api/cognitive/hrm/init`);
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