#!/usr/bin/env node

import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import compression from 'compression';
import helmet from 'helmet';
import fetch from 'node-fetch';
import fs from 'fs';
import http from 'http';
import net from 'net';
import tls from 'tls';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = process.env.PORT || 3000;
const DIST_PATH = path.join(__dirname, '..', 'dist');

// Upstream targets to keep browser requests same-origin (satisfies strict CSP / Render restrictions)
const API_TARGET = process.env.API_TARGET || 'http://localhost:3001'; // KNIRVCONTROLLER API server
const WALLET_TARGET = process.env.WALLET_TARGET || 'https://wallet.knirv.com'; // Wallet bridge/service
const ENABLE_PROXY_LOGS = (process.env.ENABLE_PROXY_LOGS || 'false').toLowerCase() === 'true';
const PROXY_TIMEOUT_MS = Number(process.env.PROXY_TIMEOUT_MS || 15000);

// Helper: build CSP connect-src dynamically (auto-add API/WALLET origins)
const baseConnectSrc = ["'self'", 'ws:', 'wss:'];
const extraConnect = (process.env.CSP_CONNECT || '')
  .split(',')
  .map(s => s.trim())
  .filter(Boolean);
// Derive origins from configured targets when absolute URLs are provided
const apiOrigin = (() => {
  try { return API_TARGET.startsWith('http') ? new URL(API_TARGET).origin : null; } catch { return null; }
})();
const walletOrigin = (() => {
  try { return WALLET_TARGET.startsWith('http') ? new URL(WALLET_TARGET).origin : null; } catch { return null; }
})();
const connectSrc = [...baseConnectSrc, apiOrigin, walletOrigin, ...extraConnect].filter(Boolean);

// Security middleware
app.use(
  helmet({
    contentSecurityPolicy: {
      directives: {
        defaultSrc: ["'self'"],
        scriptSrc: ["'self'", "'unsafe-inline'", "'unsafe-eval'"],
        styleSrc: ["'self'", "'unsafe-inline'"],
        imgSrc: ["'self'", 'data:', 'blob:'],
        connectSrc, // dynamically built; defaults to 'self' + ws/wss
        fontSrc: ["'self'"],
        objectSrc: ["'none'"],
        mediaSrc: ["'self'"],
        frameSrc: ["'none'"],
      },
    },
    crossOriginEmbedderPolicy: false,
  })
);

// Compression middleware
app.use(compression());

// Serve static files
app.use(
  express.static(DIST_PATH, {
    maxAge: '1d',
    etag: true,
    lastModified: true,
    index: 'index.html',
  })
);

// PWA Installation Routes - handle query parameters
app.get('/', (req, res, next) => {
  // Check for install parameters and handle accordingly
  if (req.query.install === 'android' || req.query.install === 'ios') {
    // Set headers for PWA installation
    res.set({
      'X-PWA-Install': req.query.install,
      'X-PWA-URL': 'https://beta-controller.knirv.com',
    });
  }
  next();
});

// API endpoint to check PWA installation status
app.get('/api/pwa/status', (req, res) => {
  res.json({
    pwa_url: 'https://beta-controller.knirv.com',
    pwa_install_android: 'https://beta-controller.knirv.com/?install=android',
    pwa_install_ios: 'https://beta-controller.knirv.com/?install=ios',
    manifest_url: '/manifest.json',
    service_worker_url: '/sw.js',
    status: 'available',
  });
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    pwa_enabled: true,
    endpoints: {
      pwa_url: 'https://beta-controller.knirv.com',
      pwa_install_android: 'https://beta-controller.knirv.com/?install=android',
      pwa_install_ios: 'https://beta-controller.knirv.com/?install=ios',
    },
  });
});

// --- Minimal same-origin proxy utilities (no extra deps) ---
const hopByHopHeaders = new Set([
  'connection',
  'transfer-encoding',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'upgrade',
]);

function logProxy(direction, info) {
  if (!ENABLE_PROXY_LOGS) return;
  console.log(`[proxy:${direction}]`, info);
}

function sanitizeHeaders(headersObj = {}) {
  const headers = { ...headersObj };
  // Remove hop-by-hop headers and let fetch set Host/Content-Length
  for (const key of Object.keys(headers)) {
    if (hopByHopHeaders.has(key.toLowerCase())) delete headers[key];
  }
  delete headers.host;
  delete headers['content-length'];
  return headers;
}

function createProxy(targetBase, options = {}) {
  const { stripPrefix = '', timeout = PROXY_TIMEOUT_MS } = options;
  const target = new URL(targetBase);

  return async (req, res) => {
    try {
      // Compute target URL, optionally stripping a base path prefix
      const original = req.originalUrl || req.url || '';
      const pathToAppend = stripPrefix && original.startsWith(stripPrefix)
        ? original.slice(stripPrefix.length) || '/'
        : original;
      const targetUrl = new URL(pathToAppend, target);

      const method = req.method || 'GET';
      const hasBody = !['GET', 'HEAD'].includes(method.toUpperCase());
      const headers = sanitizeHeaders(req.headers || {});

      logProxy('request', { method, from: original, to: targetUrl.toString() });

      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), timeout);

      const upstream = await fetch(targetUrl, {
        method,
        headers,
        // Pass through the raw stream for body when present
        body: hasBody ? req : undefined,
        redirect: 'manual',
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      // Copy status and a safe subset of headers
      res.status(upstream.status);
      upstream.headers.forEach((value, key) => {
        if (!hopByHopHeaders.has(key.toLowerCase())) {
          try { res.setHeader(key, value); } catch (_) { /* ignore */ }
        }
      });

      if (method.toUpperCase() === 'HEAD') return res.end();

      // Send response body
      const buf = Buffer.from(await upstream.arrayBuffer());
      logProxy('response', { status: upstream.status, bytes: buf.length });
      return res.send(buf);
    } catch (err) {
      const status = err?.name === 'AbortError' ? 504 : 502;
      logProxy('error', { error: err?.message || String(err), status });
      return res.status(status).json({ error: 'ProxyError', message: String(err) });
    }
  };
}

// --- Same-origin routes for internal APIs ---
// 1) Controller API: everything under /api/* (keeps browser calls same-origin)
// Place AFTER local /api routes (like /api/pwa/status) and BEFORE SPA fallback
app.use('/api', createProxy(API_TARGET));

// 2) Wallet UI and bridge under same-origin.
// If a local wallet UI build exists, serve it at /wallet (e.g., register.html), else proxy.
const BRIDGE_DIST = path.join(__dirname, '..', 'browser-bridge', 'packages', 'knirvwallet-extension', 'dist');
const WALLET_CHROME_STORE_URL = process.env.WALLET_CHROME_STORE_URL || '';
if (fs.existsSync(BRIDGE_DIST)) {
  app.use('/wallet', express.static(BRIDGE_DIST));
  // Redirect bare /wallet to a default page
  app.get('/wallet', (req, res, next) => {
    try {
      return res.sendFile(path.join(BRIDGE_DIST, 'register.html'));
    } catch (_) { return next(); }
  });
}
// Convenience route to initiate Chrome Extension install/register
app.get('/wallet/install', (req, res) => {
  if (WALLET_CHROME_STORE_URL) {
    return res.redirect(302, WALLET_CHROME_STORE_URL);
  }
  // Fallback to local UI instructions if available
  if (fs.existsSync(BRIDGE_DIST)) {
    return res.redirect(302, '/wallet/register.html');
  }
  // Final fallback to wallet base
  return res.redirect(302, '/wallet');
});
// Proxy both /wallet/* and /api/wallet/* to WALLET_TARGET for API bridge calls.
app.use('/wallet', createProxy(WALLET_TARGET, { stripPrefix: '/wallet' }));
app.use('/api/wallet', createProxy(WALLET_TARGET, { stripPrefix: '/api/wallet' }));

// Convenience health route to wallet
app.get('/api/wallet/health', async (req, res) => {
  try {
    const r = await fetch(new URL('/health', WALLET_TARGET));
    const data = await r.text();
    res.status(r.status).type(r.headers.get('content-type') || 'text/plain').send(data);
  } catch (e) {
    res.status(502).json({ error: 'Bad Gateway', target: WALLET_TARGET });
  }
});

// WebSocket upgrade proxy for API and Wallet same-origin WS (ws/wss)
const wsTargets = [API_TARGET, WALLET_TARGET].filter(t => t && t.startsWith('http'));
function selectWsTarget(req) {
  // Route by path prefix
  const url = req.url || '';
  if (url.startsWith('/api/')) return API_TARGET;
  if (url.startsWith('/wallet') || url.startsWith('/api/wallet')) return WALLET_TARGET;
  // Default to API
  return API_TARGET;
}

// Create HTTP server to handle upgrade
const server = http.createServer(app);

server.on('upgrade', (req, socket, head) => {
  try {
    const targetBase = selectWsTarget(req);
    const targetUrl = new URL(targetBase);
    const isTls = targetUrl.protocol === 'https:';
    const port = targetUrl.port || (isTls ? 443 : 80);

    const upstream = (isTls ? tls.connect : net.connect)({
      host: targetUrl.hostname,
      port: Number(port),
      servername: targetUrl.hostname
    }, () => {
      // Compose WebSocket handshake to upstream
      const pathWithQuery = (req.url?.startsWith('/wallet') ? req.url.replace(/^\/wallet/, '') : req.url) || '/';
      const requestHeaders = [
        `GET ${pathWithQuery} HTTP/1.1`,
        `Host: ${targetUrl.host}`,
        'Connection: Upgrade',
        'Upgrade: websocket',
        // Pass through headers that might be needed (Sec-WebSocket-*)
      ];
      for (const [k, v] of Object.entries(req.headers)) {
        const key = k.toLowerCase();
        if (key.startsWith('sec-websocket') || key === 'origin' || key === 'cookie') {
          requestHeaders.push(`${k}: ${v}`);
        }
      }
      requestHeaders.push('\r\n');
      upstream.write(requestHeaders.join('\r\n'));
      upstream.write(head);
      upstream.pipe(socket);
      socket.pipe(upstream);
    });

    upstream.on('error', () => {
      socket.destroy();
    });
  } catch (e) {
    socket.destroy();
  }
});

// Handle SPA routing - serve index.html for all routes
app.get('*', (req, res) => {
  res.sendFile(path.join(DIST_PATH, 'index.html'));
});

// Error handling
app.use((err, req, res, next) => {
  console.error('Server error:', err);
  res.status(500).send('Internal Server Error');
});

// Start server with WS upgrade support
server.listen(PORT, '0.0.0.0', () => {
  console.log(`KNIRV Controller static server running on port ${PORT}`);
  console.log(`Serving files from: ${DIST_PATH}`);
  console.log(`API_TARGET: ${API_TARGET}`);
  console.log(`WALLET_TARGET: ${WALLET_TARGET}`);
  if (extraConnect.length > 0) {
    console.log(`CSP connect-src extended with: ${extraConnect.join(', ')}`);
  }
  console.log('WebSocket upgrade proxy enabled');
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('Received SIGTERM, shutting down gracefully...');
  process.exit(0);
});

process.on('SIGINT', () => {
  console.log('Received SIGINT, shutting down gracefully...');
  process.exit(0);
});
