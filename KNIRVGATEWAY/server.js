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

// Load polyfills first for libp2p compatibility
import './lib/polyfills.js';

import express from 'express';
import path from 'path';
import cors from 'cors';
import { PrivateDHTManager } from './lib/p2p/private_dht_manager.js';
import { NodeJSServiceManager } from './lib/services/nodejs_service_manager.js';
import axios from 'axios';
import NodeCache from 'node-cache';
import { createProxyMiddleware } from 'http-proxy-middleware';
import { fileURLToPath } from 'url';
import crypto from 'crypto';

// ES module compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

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
let nodeJSServiceManager = null;
let app = null;
let dhtInitialized = false;
let dhtStartupInProgress = false;

/**
 * Initialize Express application
 */
function createApp() {
  const app = express();
  
  // Middleware
  app.use(cors());
  app.use(express.json());

  // --- Minimal cookie/session handling for per-user controller URL ---
  const sessions = new Map(); // In-memory session store: sid -> { controllerUrl?: string }

  function parseCookies(cookieHeader = '') {
    return cookieHeader.split(';').reduce((acc, part) => {
      const [k, v] = part.split('=');
      if (!k) return acc;
      acc[k.trim()] = decodeURIComponent((v || '').trim());
      return acc;
    }, {});
  }

  function getOrInitSession(req, res) {
    const cookies = parseCookies(req.headers.cookie || '');
    let sid = cookies.knirv_sid;
    if (!sid) {
      sid = crypto.randomBytes(16).toString('hex');
      const cookie = `knirv_sid=${encodeURIComponent(sid)}; Path=/; HttpOnly; SameSite=Lax`;
      res.setHeader('Set-Cookie', cookie);
    }
    if (!sessions.has(sid)) sessions.set(sid, {});
    return { sid, data: sessions.get(sid) };
  }

  // Expose simple session endpoints for controller URL
  app.get('/session/controller', (req, res) => {
    const { data } = getOrInitSession(req, res);
    res.json({ controllerUrl: data.controllerUrl || null });
  });

  app.post('/session/controller', (req, res) => {
    const { data } = getOrInitSession(req, res);
    const { controllerUrl } = req.body || {};
    try {
      if (!controllerUrl || typeof controllerUrl !== 'string') {
        return res.status(400).json({ error: 'controllerUrl (string) required' });
      }
      // Basic validation and normalization
      const u = new URL(controllerUrl);
      if (!['http:', 'https:'].includes(u.protocol)) {
        return res.status(400).json({ error: 'controllerUrl must be http(s)' });
      }
      // Trim trailing slash for consistency
      u.pathname = '/';
      data.controllerUrl = u.toString().replace(/\/$/, '');
      res.json({ ok: true, controllerUrl: data.controllerUrl });
    } catch (e) {
      res.status(400).json({ error: 'Invalid controllerUrl', message: e.message });
    }
  });

  // Route: /dashboard -> WebGUI (Next.js) running on localhost:3007 (configurable via WEBGUI_PORT)
  const WEBGUI_PORT = parseInt(process.env.WEBGUI_PORT) || 3007;

  // Proxy the WebGUI app itself (mounted at /dashboard)
  app.use('/dashboard', createProxyMiddleware({
    target: `http://localhost:${WEBGUI_PORT}`,
    changeOrigin: true,
    ws: true,
    pathRewrite: { '^/dashboard': '' },
    onProxyReq: (proxyReq, req) => {
      proxyReq.setHeader('X-Forwarded-Host', req.headers.host || '');
      proxyReq.setHeader('X-Forwarded-Proto', req.protocol || 'http');
    }
  }));

  // Also proxy Next.js static asset routes that are referenced with absolute paths (e.g., /_next/*)
  // Without this, requests like /_next/static/... would hit the gateway directly and return HTML/404, causing MIME errors
  app.use('/_next', createProxyMiddleware({
    target: `http://localhost:${WEBGUI_PORT}`,
    changeOrigin: true,
    ws: true,
  }));

  // Optional: common root assets served by Next.js
  app.use(['/favicon.ico', '/favicon.png', '/manifest.json', '/robots.txt'], createProxyMiddleware({
    target: `http://localhost:${WEBGUI_PORT}`,
    changeOrigin: true,
  }));

  // Serve landing page and static assets (after proxy mounts to avoid intercepting Next.js assets)
  app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
  });
  app.use(express.static(__dirname));

  // Helper to resolve service URL dynamically from the NodeJSServiceManager
  function resolveServiceUrl(serviceName, preferred = 'main') {
    try {
      if (!nodeJSServiceManager) return null;
      const svc = nodeJSServiceManager.processes?.get(serviceName);
      if (!svc || !svc.ports) return null;
      const { port, httpPort } = svc.ports;
      const chosenPort = preferred === 'api' ? (httpPort || port) : (port || httpPort);
      if (!chosenPort) return null;
      return `http://localhost:${chosenPort}`;
    } catch (e) {
      return null;
    }
  }

  // Primary route: Payment Gateway
  app.use('/payment', (req, res, next) => {
    const target = resolveServiceUrl('paymentGateway', 'main');
    if (!target) {
      return res.status(503).json({ error: 'Payment Gateway unavailable', hint: 'Service not started yet' });
    }
    return createProxyMiddleware({
      target,
      changeOrigin: true,
      ws: true,
      pathRewrite: { '^/payment': '' }
    })(req, res, next);
  });

  // Dynamic Controller proxy: use per-session controllerUrl
  app.use('/controller', (req, res, next) => {
    const { data } = getOrInitSession(req, res);
    const target = data.controllerUrl;
    if (!target) {
      return res.status(428).json({
        error: 'ControllerNotConnected',
        message: 'No controller URL set in session. Use QR Connect or POST /session/controller to set it.'
      });
    }
    return createProxyMiddleware({
      target,
      changeOrigin: true,
      ws: true,
      pathRewrite: { '^/controller': '' }
    })(req, res, next);
  });

  // Primary route: Operator Registry (Next.js + API)
  app.use('/operator-registry', (req, res, next) => {
    const target = resolveServiceUrl('operatorRegistry', 'main');
    if (!target) {
      return res.status(503).json({ error: 'Operator Registry unavailable', hint: 'Service not started yet' });
    }
    return createProxyMiddleware({
      target,
      changeOrigin: true,
      ws: true,
      pathRewrite: { '^/operator-registry': '' }
    })(req, res, next);
  });

  // Primary route: Tunnel Registry (HTTP API + Status)
  app.use('/tunnel-registry', (req, res, next) => {
    const target = resolveServiceUrl('tunnelRegistry', 'api');
    if (!target) {
      return res.status(503).json({ error: 'Tunnel Registry unavailable', hint: 'Service not started yet' });
    }
    return createProxyMiddleware({
      target,
      changeOrigin: true,
      ws: true,
      pathRewrite: { '^/tunnel-registry': '' }
    })(req, res, next);
  });

  // Mock central API gateway. Replace with real delegations to services when ready.
  app.use('/api', (req, res) => {
    if (req.method === 'GET' && (req.path === '/' || req.path === '/health')) {
      return res.json({
        status: 'ok',
        message: 'Mock KNIRV central API gateway',
        chainId: CHAIN_ID,
        timestamp: Date.now(),
      });
    }

    return res.status(501).json({
      error: 'Not Implemented',
      message: 'Central API routing is not yet implemented. This is a mock endpoint.',
      method: req.method,
      route: req.originalUrl
    });
  });
  
  // Health check endpoint
  app.get('/health', (req, res) => {
    const status = {
      status: 'healthy',
      mode: GATEWAY_MODE,
      timestamp: Date.now(),
      chainId: CHAIN_ID,
      dhtInitialized,
      dhtStartupInProgress
    };

    if (dhtManager) {
      status.dht = dhtManager.getNetworkStatus();
    } else {
      status.dht = { status: 'not_initialized' };
    }

    if (nodeJSServiceManager) {
      status.nodeJSServices = nodeJSServiceManager.getServicesStatus();
    } else {
      status.nodeJSServices = { status: 'not_initialized' };
    }

    res.json(status);
  });

  // Node.js Services Management Endpoints
  app.get('/services/status', (req, res) => {
    if (!nodeJSServiceManager) {
      return res.status(503).json({ error: 'Node.js Service Manager not initialized' });
    }
    res.json(nodeJSServiceManager.getServicesStatus());
  });

  app.get('/services/endpoints', (req, res) => {
    if (!nodeJSServiceManager) {
      return res.status(503).json({ error: 'Node.js Service Manager not initialized' });
    }
    res.json(nodeJSServiceManager.getServiceEndpoints());
  });

  app.post('/services/start', async (req, res) => {
    if (!nodeJSServiceManager) {
      return res.status(503).json({ error: 'Node.js Service Manager not initialized' });
    }

    try {
      const result = await nodeJSServiceManager.startAllServices();
      res.json({ success: result, message: result ? 'Services started' : 'Failed to start services' });
    } catch (error) {
      res.status(500).json({ error: error.message });
    }
  });

  app.post('/services/stop', async (req, res) => {
    if (!nodeJSServiceManager) {
      return res.status(503).json({ error: 'Node.js Service Manager not initialized' });
    }

    try {
      await nodeJSServiceManager.stopAllServices();
      res.json({ success: true, message: 'Services stopped' });
    } catch (error) {
      res.status(500).json({ error: error.message });
    }
  });

  app.post('/services/:serviceName/start', async (req, res) => {
    if (!nodeJSServiceManager) {
      return res.status(503).json({ error: 'Node.js Service Manager not initialized' });
    }

    const { serviceName } = req.params;
    const serviceConfig = nodeJSServiceManager.config.services[serviceName];

    if (!serviceConfig) {
      return res.status(404).json({ error: `Service ${serviceName} not found` });
    }

    try {
      const result = await nodeJSServiceManager.startService(serviceName, serviceConfig);
      res.json({ success: result, message: result ? `Service ${serviceName} started` : `Failed to start ${serviceName}` });
    } catch (error) {
      res.status(500).json({ error: error.message });
    }
  });

  app.post('/services/:serviceName/stop', async (req, res) => {
    if (!nodeJSServiceManager) {
      return res.status(503).json({ error: 'Node.js Service Manager not initialized' });
    }

    const { serviceName } = req.params;

    try {
      const result = await nodeJSServiceManager.stopService(serviceName);
      res.json({ success: result, message: result ? `Service ${serviceName} stopped` : `Service ${serviceName} was not running` });
    } catch (error) {
      res.status(500).json({ error: error.message });
    }
  });

  // Provision endpoint - core functionality
  app.get('/provision', async (req, res) => {
    try {
      if (GATEWAY_MODE === 'persistent') {
        // Persistent mode: return peers from local DHT
        if (!dhtManager || !dhtManager.isStarted) {
          console.log('[Gateway] DHT not available, returning empty peer list');
          return res.json([]); // Return empty array instead of error for graceful degradation
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

  // DHT control endpoints
  app.post('/dht/start', async (req, res) => {
    try {
      if (GATEWAY_MODE !== 'persistent') {
        return res.status(400).json({
          error: 'DHT can only be started in persistent mode',
          currentMode: GATEWAY_MODE
        });
      }

      if (dhtInitialized) {
        return res.json({
          message: 'DHT already initialized',
          status: dhtManager ? dhtManager.getNetworkStatus() : { status: 'unknown' }
        });
      }

      if (dhtStartupInProgress) {
        return res.json({
          message: 'DHT startup already in progress',
          status: 'starting'
        });
      }

      console.log('[Gateway] DHT start requested via API');
      dhtStartupInProgress = true;

      // Start DHT initialization
      await startPersistentGateway();

      dhtStartupInProgress = false;
      dhtInitialized = true;

      res.json({
        message: 'DHT started successfully',
        status: dhtManager ? dhtManager.getNetworkStatus() : { status: 'started' }
      });

    } catch (error) {
      console.error('[Gateway] DHT start failed:', error);
      dhtStartupInProgress = false;

      res.status(500).json({
        error: 'Failed to start DHT',
        details: error.message
      });
    }
  });

  app.post('/dht/stop', async (req, res) => {
    try {
      if (!dhtManager) {
        return res.json({ message: 'DHT not running' });
      }

      console.log('[Gateway] DHT stop requested via API');
      await dhtManager.stop();
      dhtManager = null;
      dhtInitialized = false;
      dhtStartupInProgress = false;

      res.json({ message: 'DHT stopped successfully' });

    } catch (error) {
      console.error('[Gateway] DHT stop failed:', error);
      res.status(500).json({
        error: 'Failed to stop DHT',
        details: error.message
      });
    }
  });

  app.get('/dht/restart', async (req, res) => {
    try {
      if (GATEWAY_MODE !== 'persistent') {
        return res.status(400).json({
          error: 'DHT can only be restarted in persistent mode',
          currentMode: GATEWAY_MODE
        });
      }

      console.log('[Gateway] DHT restart requested via API');

      // Stop if running
      if (dhtManager) {
        await dhtManager.stop();
        dhtManager = null;
      }

      dhtInitialized = false;
      dhtStartupInProgress = true;

      // Start again
      await startPersistentGateway();

      dhtStartupInProgress = false;
      dhtInitialized = true;

      res.json({
        message: 'DHT restarted successfully',
        status: dhtManager ? dhtManager.getNetworkStatus() : { status: 'restarted' }
      });

    } catch (error) {
      console.error('[Gateway] DHT restart failed:', error);
      dhtStartupInProgress = false;

      res.status(500).json({
        error: 'Failed to restart DHT',
        details: error.message
      });
    }
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
  console.log('[Gateway] Starting DHT initialization...');

  // Check if DHT should be disabled for debugging
  const DISABLE_DHT = process.env.DISABLE_DHT === 'true';
  if (DISABLE_DHT) {
    console.log('[Gateway] ⚠️  DHT disabled via DISABLE_DHT environment variable');
    console.log('[Gateway] ✅ DHT initialization skipped');
    return;
  }

  try {
    console.log('[Gateway] Bootstrap peers:', BOOTSTRAP_PEERS);
    console.log('[Gateway] DHT port:', process.env.DHT_PORT || 'auto');

    // Initialize DHT manager with timeout
    console.log('[Gateway] Creating DHT manager...');
    dhtManager = new PrivateDHTManager(CHAIN_ID, 'knirvgateway', {
      clientMode: false, // This is a bootstrap node
      enableBootstrap: true,
      bootstrapPeers: BOOTSTRAP_PEERS,
      listenPort: process.env.DHT_PORT || 0,
      capabilities: ['gateway', 'bootstrap', 'provision']
    });
    console.log('[Gateway] DHT manager created successfully');

    // Set up DHT event listeners
    dhtManager.on('initialized', (data) => {
      console.log('[Gateway] ✅ DHT initialized:', data);
    });

    dhtManager.on('peer:connect', (data) => {
      console.log('[Gateway] ✅ Peer connected:', data.peerId);
    });

    dhtManager.on('service:discovered', (data) => {
      console.log('[Gateway] ✅ Service discovered:', data.serviceType);
    });

    dhtManager.on('error', (error) => {
      console.error('[Gateway] ❌ DHT error:', error);
    });

    // Initialize DHT with timeout
    console.log('[Gateway] Initializing DHT (with 30s timeout)...');
    const initPromise = dhtManager.initialize();
    const timeoutPromise = new Promise((_, reject) => {
      setTimeout(() => reject(new Error('DHT initialization timeout after 30 seconds')), 30000);
    });

    const success = await Promise.race([initPromise, timeoutPromise]);
    if (!success) {
      throw new Error('Failed to initialize DHT manager');
    }

    console.log('[Gateway] ✅ Persistent gateway DHT started successfully');

  } catch (error) {
    console.error('[Gateway] ❌ Failed to start persistent gateway:', error);
    console.error('[Gateway] ❌ Error details:', error.message);
    console.error('[Gateway] ❌ Stack trace:', error.stack);

    // In production, continue without DHT rather than failing completely
    if (process.env.NODE_ENV === 'production') {
      console.log('[Gateway] ⚠️  Continuing without DHT in production mode');
      console.log('[Gateway] ⚠️  Gateway will operate in degraded mode (no P2P functionality)');
      dhtManager = null;
      return;
    }

    // In development, also continue without DHT for easier debugging
    console.log('[Gateway] ⚠️  Continuing without DHT for debugging');
    dhtManager = null;
    return;
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
  try {
    console.log(`[Gateway] KNIRVGATEWAY starting in ${GATEWAY_MODE} mode`);
    console.log(`[Gateway] Chain ID: ${CHAIN_ID}`);
    console.log(`[Gateway] Port: ${PORT}`);
    console.log(`[Gateway] Node.js version: ${process.version}`);
    console.log(`[Gateway] Platform: ${process.platform}`);
    console.log(`[Gateway] Architecture: ${process.arch}`);

    // Log environment variables for debugging
    console.log(`[Gateway] Environment variables:`);
    console.log(`[Gateway]   NODE_ENV: ${process.env.NODE_ENV}`);
    console.log(`[Gateway]   GATEWAY_MODE: ${process.env.GATEWAY_MODE}`);
    console.log(`[Gateway]   DISABLE_DHT: ${process.env.DISABLE_DHT}`);
    console.log(`[Gateway]   DHT_PORT: ${process.env.DHT_PORT}`);
    console.log(`[Gateway]   KNIRV_BOOTSTRAP_PEERS: ${process.env.KNIRV_BOOTSTRAP_PEERS ? 'SET' : 'NOT SET'}`);
    console.log(`[Gateway]   INTERNAL_API_KEY: ${process.env.INTERNAL_API_KEY ? 'SET' : 'NOT SET'}`);

    // Create Express app
    console.log(`[Gateway] Creating Express application...`);
    app = createApp();
    console.log(`[Gateway] Express application created successfully`);

    // Initialize Node.js Service Manager
    console.log(`[Gateway] Initializing Node.js Service Manager...`);
    nodeJSServiceManager = new NodeJSServiceManager({
      enabled: process.env.NODEJS_SERVICES_ENABLED !== 'false',
      chainId: CHAIN_ID,
      nodeId: `knirvgateway-${Date.now()}`,
      publicHost: process.env.PUBLIC_HOST || 'localhost',
      paymentGateway: {
        enabled: process.env.PAYMENT_GATEWAY_ENABLED !== 'false',
        port: parseInt(process.env.PAYMENT_GATEWAY_PORT) || 3001
      },
      tunnelRegistry: {
        enabled: process.env.TUNNEL_REGISTRY_ENABLED !== 'false',
        httpPort: parseInt(process.env.TUNNEL_REGISTRY_HTTP_PORT) || 3002,
        controlPort: parseInt(process.env.TUNNEL_REGISTRY_CONTROL_PORT) || 3003,
        publicRelayPort: parseInt(process.env.TUNNEL_REGISTRY_PUBLIC_RELAY_PORT) || 3004,
        stunPort: parseInt(process.env.TUNNEL_REGISTRY_STUN_PORT) || 3005
      },
      operatorRegistry: {
        enabled: process.env.OPERATOR_REGISTRY_ENABLED !== 'false',
        port: parseInt(process.env.OPERATOR_REGISTRY_PORT) || 3006
      },
      webgui: {
        enabled: process.env.WEBGUI_ENABLED !== 'false',
        port: parseInt(process.env.WEBGUI_PORT) || 3007
      }
    });
    console.log(`[Gateway] Node.js Service Manager initialized`);

    // Auto-start Node.js services if enabled
    if (process.env.NODEJS_SERVICES_AUTOSTART !== 'false') {
      console.log(`[Gateway] Auto-starting Node.js services...`);
      try {
        const servicesStarted = await nodeJSServiceManager.startAllServices();
        if (servicesStarted) {
          console.log(`[Gateway] ✅ Node.js services started successfully`);
          const endpoints = nodeJSServiceManager.getServiceEndpoints();
          console.log(`[Gateway] Service endpoints:`, endpoints);
        } else {
          console.log(`[Gateway] ⚠️  Some Node.js services failed to start`);
        }
      } catch (error) {
        console.error(`[Gateway] ❌ Failed to start Node.js services:`, error);
      }
    }

    // Initialize based on mode
    if (GATEWAY_MODE === 'persistent') {
      console.log(`[Gateway] Persistent mode - DHT will be started on demand`);
      console.log(`[Gateway] Use POST /dht/start to initialize DHT when ready`);
      console.log(`[Gateway] Persistent gateway mode initialized successfully (DHT disabled by default)`);
    } else {
      console.log(`[Gateway] Initializing serverless gateway mode...`);
      await startServerlessGateway();
      console.log(`[Gateway] Serverless gateway mode initialized successfully`);
    }

    // Start HTTP server
    console.log(`[Gateway] Starting HTTP server on port ${PORT}...`);
    const server = app.listen(PORT, '0.0.0.0', () => {
      console.log(`[Gateway] ✅ Server listening on port ${PORT}`);
      console.log(`[Gateway] ✅ Health check: http://localhost:${PORT}/health`);
      console.log(`[Gateway] ✅ Provision endpoint: http://localhost:${PORT}/provision`);
      console.log(`[Gateway] ✅ KNIRVGATEWAY startup completed successfully`);
    });

    // Add error handling for server
    server.on('error', (error) => {
      console.error(`[Gateway] ❌ Server error:`, error);
      if (error.code === 'EADDRINUSE') {
        console.error(`[Gateway] ❌ Port ${PORT} is already in use`);
      } else if (error.code === 'EACCES') {
        console.error(`[Gateway] ❌ Permission denied to bind to port ${PORT}`);
      }
      process.exit(1);
    });

    // Graceful shutdown
    process.on('SIGINT', async () => {
      console.log('[Gateway] Shutting down gracefully...');

      if (nodeJSServiceManager) {
        await nodeJSServiceManager.stopAllServices();
      }

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

      if (nodeJSServiceManager) {
        await nodeJSServiceManager.stopAllServices();
      }

      if (dhtManager) {
        await dhtManager.stop();
      }

      server.close(() => {
        console.log('[Gateway] Server closed');
        process.exit(0);
      });
    });

  } catch (error) {
    console.error(`[Gateway] ❌ Fatal startup error:`, error);
    console.error(`[Gateway] ❌ Stack trace:`, error.stack);
    process.exit(1);
  }
}

// Start the server if this file is run directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(error => {
    console.error('[Gateway] Startup error:', error);
    process.exit(1);
  });
}

export { createApp, main };
