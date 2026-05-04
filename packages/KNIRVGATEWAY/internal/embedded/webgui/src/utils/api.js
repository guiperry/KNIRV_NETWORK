import axios from 'axios';

// Use the gateway's base URL at runtime when served through the wrapper proxy.
// window.__GATEWAY_BASE__ is injected by the gateway server (injectGatewayBase)
// and points to http://localhost:8080 — all /api/v1/* calls go through the
// gateway's Unix socket proxy to the real backend.
const GATEWAY_BASE = typeof window !== 'undefined' && window.__GATEWAY_BASE__
  ? window.__GATEWAY_BASE__
  : '';

const API_URL = process.env.NEXT_PUBLIC_BACKEND_URL || GATEWAY_BASE || '';

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 5000, // 5 second timeout
});

// Inject API key when available (dev/local only)
api.interceptors.request.use((config) => {
  try {
    const envKey = process.env.NEXT_PUBLIC_CONTROLLER_API_KEY;
    const clientKey = typeof window !== 'undefined' ? localStorage.getItem('controller_api_key') : null;
    const key = envKey || clientKey;
    if (key) {
      config.headers = config.headers || {};
      config.headers['X-API-Key'] = key;
    }
  } catch (_) {
    // ignore
  }
  return config;
});

export default api;
