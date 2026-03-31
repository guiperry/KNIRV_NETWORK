import axios from 'axios';

// Use env var when set; otherwise use a relative URL so requests go to
// whichever host is serving the WebGUI (works both embedded and standalone).
const API_URL = process.env.NEXT_PUBLIC_BACKEND_URL || '';

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