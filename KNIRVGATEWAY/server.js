/**
 * KNIRVGATEWAY Unified Server
 * 
 * Supports multiple deployment modes:
 * - persistent: Full DHT node for Render deployment
 * - serverless: Lightweight proxy for Netlify/Vercel
 * 
 * Environment Variables:
 * - GATEWAY_MODE: 'persistent' | 'serverless' (default: 'persistent')
 * - PORT: Server port (default: 8080)
 * - KNIRV_CHAIN_ID: Chain ID (default: 'testnet')
 * - KNIRV_BOOTSTRAP_PEERS: Comma-separated bootstrap peers
 * - RENDER_GATEWAY_INTERNAL_API: Internal API endpoint for serverless mode
 * - INTERNAL_API_KEY: API key for internal communication
 */

const express = require('express');
const path = require('path');
const cors = require('cors');
const { PrivateDHTManager } = require('./lib/p2p/private_dht_manager');
const axios = require('axios');
const NodeCache = require('node-cache');

// Configuration
const GATEWAY_MODE = process.env.GATEWAY_MODE || 'persistent';
const PORT = process.env.PORT || 8080;
const CHAIN_ID = process.env.KNIRV_CHAIN_ID || 'testnet';
const BOOTSTRAP_PEERS = process.env.KNIRV_BOOTSTRAP_PEERS ? 
  process.env.KNIRV_BOOTSTRAP_PEERS.split(',').map(peer => peer.trim()) : [];

// Cache for serverless mode
const cache = new NodeCache({ stdTTL: 60, checkperiod: 10 });

// Global state
let dhtManager = null;
let app = null;

/**
 * Initialize Express application
 */
function createApp() {
  const app = express();
  
  // Middleware
  app.use(cors());
  app.use(express.json());
  app.use(express.static('.'));
  
  // Health check endpoint
  app.get('/health', (req, res) => {
    const status = {
      status: 'healthy',
      mode: GATEWAY_MODE,
      timestamp: Date.now(),
      chainId: CHAIN_ID
    };
    
    if (dhtManager) {
      status.dht = dhtManager.getNetworkStatus();
    }
    
    res.json(status);
  });
  
  // Provision endpoint - core functionality
  app.get('/provision', async (req, res) => {
    try {
      if (GATEWAY_MODE === 'persistent') {
        // Persistent mode: return peers from local DHT
        if (!dhtManager || !dhtManager.isStarted) {
          return res.status(503).json({ 
            error: 'DHT not available',
            message: 'DHT manager not started or not available'
          });
        }
        
        const peers = dhtManager.getProvisionPeers();
        console.log(`[Gateway] Provisioning ${peers.length} peers`);
        
        res.json(peers);
      } else {
        // Serverless mode: proxy to persistent gateway
        await handleServerlessProvision(req, res);
      }
    } catch (error) {
      console.error('[Gateway] Provision endpoint error:', error);
      res.status(500).json({ 
        error: 'Failed to fetch DHT peers',
        details: error.message 
      });
    }
  });
  
  // Internal API endpoint for serverless gateways to query
  app.get('/internal/peers', authenticateInternal, (req, res) => {
    if (GATEWAY_MODE !== 'persistent') {
      return res.status(403).json({ error: 'Internal API only available in persistent mode' });
    }
    
    if (!dhtManager || !dhtManager.isStarted) {
      return res.status(503).json({ error: 'DHT not available' });
    }
    
    const peers = dhtManager.getProvisionPeers();
    res.json(peers);
  });
  
  // DHT status endpoint
  app.get('/dht/status', (req, res) => {
    if (!dhtManager) {
      return res.json({ status: 'not_initialized', mode: GATEWAY_MODE });
    }
    
    res.json(dhtManager.getNetworkStatus());
  });
  
  // Service discovery endpoint
  app.get('/services', (req, res) => {
    if (!dhtManager || !dhtManager.isStarted) {
      return res.status(503).json({ error: 'DHT not available' });
    }
    
    const status = dhtManager.getNetworkStatus();
    res.json({
      discoveredServices: status.discoveredServices,
      connectedPeers: status.connectedPeers,
      chainId: CHAIN_ID
    });
  });
  
  return app;
}

/**
 * Handle provision requests in serverless mode
 */
async function handleServerlessProvision(req, res) {
  try {
    // Check cache first
    const cachedPeers = cache.get('dht_peers');
    if (cachedPeers) {
      console.log('[Gateway] Returning cached DHT peers (serverless mode)');
      return res.json(cachedPeers);
    }
    
    // Fetch from persistent gateway
    const RENDER_GATEWAY_INTERNAL_API = process.env.RENDER_GATEWAY_INTERNAL_API;
    const INTERNAL_API_KEY = process.env.INTERNAL_API_KEY;
    
    if (!RENDER_GATEWAY_INTERNAL_API || !INTERNAL_API_KEY) {
      return res.status(500).json({
        error: 'Gateway internal API endpoint or key not configured',
        message: 'Serverless mode requires RENDER_GATEWAY_INTERNAL_API and INTERNAL_API_KEY'
      });
    }
    
    const response = await axios.get(RENDER_GATEWAY_INTERNAL_API, {
      headers: {
        'Authorization': `Bearer ${INTERNAL_API_KEY}`
      },
      timeout: 5000
    });
    
    const dhtPeers = response.data;
    
    // Cache the result
    cache.set('dht_peers', dhtPeers);
    
    console.log(`[Gateway] Fetched ${dhtPeers.length} peers from persistent gateway`);
    res.json(dhtPeers);
    
  } catch (error) {
    console.error('[Gateway] Serverless provision error:', error);
    
    // Return empty array as fallback
    res.json([]);
  }
}

/**
 * Middleware to authenticate internal API requests
 */
function authenticateInternal(req, res, next) {
  const authHeader = req.headers.authorization;
  const expectedKey = process.env.INTERNAL_API_KEY;
  
  if (!expectedKey) {
    return res.status(500).json({ error: 'Internal API key not configured' });
  }
  
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Missing or invalid authorization header' });
  }
  
  const token = authHeader.substring(7);
  if (token !== expectedKey) {
    return res.status(401).json({ error: 'Invalid API key' });
  }
  
  next();
}

/**
 * Start persistent gateway mode
 */
async function startPersistentGateway() {
  console.log('[Gateway] Starting in persistent mode...');
  
  try {
    // Initialize DHT manager
    dhtManager = new PrivateDHTManager(CHAIN_ID, 'knirvgateway', {
      clientMode: false, // This is a bootstrap node
      enableBootstrap: true,
      bootstrapPeers: BOOTSTRAP_PEERS,
      listenPort: process.env.DHT_PORT || 0,
      capabilities: ['gateway', 'bootstrap', 'provision']
    });
    
    // Set up DHT event listeners
    dhtManager.on('initialized', (data) => {
      console.log('[Gateway] DHT initialized:', data);
    });
    
    dhtManager.on('peer:connect', (data) => {
      console.log('[Gateway] Peer connected:', data.peerId);
    });
    
    dhtManager.on('service:discovered', (data) => {
      console.log('[Gateway] Service discovered:', data.serviceType);
    });
    
    dhtManager.on('error', (error) => {
      console.error('[Gateway] DHT error:', error);
    });
    
    // Initialize DHT
    const success = await dhtManager.initialize();
    if (!success) {
      throw new Error('Failed to initialize DHT manager');
    }
    
    console.log('[Gateway] Persistent gateway DHT started successfully');
    
  } catch (error) {
    console.error('[Gateway] Failed to start persistent gateway:', error);
    process.exit(1);
  }
}

/**
 * Start serverless gateway mode
 */
async function startServerlessGateway() {
  console.log('[Gateway] Starting in serverless mode...');
  console.log('[Gateway] Will proxy provision requests to persistent gateway');
}

/**
 * Main startup function
 */
async function main() {
  console.log(`[Gateway] KNIRVGATEWAY starting in ${GATEWAY_MODE} mode`);
  console.log(`[Gateway] Chain ID: ${CHAIN_ID}`);
  console.log(`[Gateway] Port: ${PORT}`);
  
  // Create Express app
  app = createApp();
  
  // Initialize based on mode
  if (GATEWAY_MODE === 'persistent') {
    await startPersistentGateway();
  } else {
    await startServerlessGateway();
  }
  
  // Start HTTP server
  const server = app.listen(PORT, () => {
    console.log(`[Gateway] Server listening on port ${PORT}`);
    console.log(`[Gateway] Health check: http://localhost:${PORT}/health`);
    console.log(`[Gateway] Provision endpoint: http://localhost:${PORT}/provision`);
  });
  
  // Graceful shutdown
  process.on('SIGINT', async () => {
    console.log('[Gateway] Shutting down gracefully...');
    
    if (dhtManager) {
      await dhtManager.stop();
    }
    
    server.close(() => {
      console.log('[Gateway] Server closed');
      process.exit(0);
    });
  });
  
  process.on('SIGTERM', async () => {
    console.log('[Gateway] Received SIGTERM, shutting down...');
    
    if (dhtManager) {
      await dhtManager.stop();
    }
    
    server.close(() => {
      console.log('[Gateway] Server closed');
      process.exit(0);
    });
  });
}

// Start the server if this file is run directly
if (require.main === module) {
  main().catch(error => {
    console.error('[Gateway] Startup error:', error);
    process.exit(1);
  });
}

module.exports = { createApp, main };
