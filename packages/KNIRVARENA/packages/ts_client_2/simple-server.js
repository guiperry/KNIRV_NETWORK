const express = require('express');
const cors = require('cors');

const app = express();
const PORT = 3001;

// Middleware
app.use(cors());
app.use(express.json());

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'healthy', timestamp: new Date().toISOString() });
});

// KNIRVBASE endpoints
app.post('/api/knirvbase/initialize', async (req, res) => {
  try {
    const { sessionId, dataDir, distributedEnabled, networkId, bootstrapPeers } = req.body;
    
    if (!sessionId) {
      return res.status(400).json({ error: 'sessionId is required' });
    }
    
    console.log('KNIRVBASE initialized for session:', sessionId);
    
    res.json({ 
      success: true, 
      message: 'KNIRVBASE database initialized successfully',
      sessionId,
      dataDir: dataDir || '/tmp/knirvbase'
    });
  } catch (error) {
    console.error('Failed to initialize KNIRVBASE:', error);
    res.status(500).json({ 
      error: 'Failed to initialize KNIRVBASE',
      details: error.message
    });
  }
});

app.get('/api/knirvbase/app-data-path', (req, res) => {
  res.json({ 
    success: true, 
    appDataDir: '/tmp/knirvbase',
    platform: process.platform,
    homedir: process.env.HOME || '/tmp'
  });
});

app.get('/api/knirvbase/session/:sessionId/info', (req, res) => {
  const { sessionId } = req.params;
  res.json({ 
    success: true, 
    sessionId,
    dataDir: '/tmp/knirvbase',
    initialized: true
  });
});

// HRM Bridge endpoint
app.post('/api/cognitive/hrm/init', (req, res) => {
  const { modelPath, config } = req.body;
  console.log('HRM Bridge initialized:', { modelPath, config });
  res.json({ 
    status: 'initialized',
    modelPath,
    config,
    bridgeId: `hrm_${Date.now()}`
  });
});

// Catch all other API routes
app.all('*', (req, res) => {
  console.log(`Unhandled ${req.method} ${req.path}`);
  res.status(404).json({ 
    error: 'Not found',
    path: req.path,
    method: req.method,
    timestamp: new Date().toISOString()
  });
});

app.listen(PORT, () => {
  console.log(`🚀 Simple API Server running on port ${PORT}`);
  console.log(`📡 Health check: http://localhost:${PORT}/health`);
});