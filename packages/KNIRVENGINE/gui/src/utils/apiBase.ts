/**
 * Runtime origin for engine API calls.
 *
 * The launcher provides VITE_API_BASE_URL after selecting an available API
 * port. Production builds leave it empty and use the GUI server's /api proxy.
 */
export const getApiBaseUrl = (): string =>
  (import.meta.env.VITE_API_BASE_URL ||
    (window.location.protocol === 'file:' ? 'http://localhost:8081' : '')).replace(/\/$/, '');

export const getApiUrl = (path: string): string => `${getApiBaseUrl()}${path}`;

export const getWebSocketUrl = (path: string): string => {
  const endpoint = new URL(getApiUrl(path), window.location.origin);
  endpoint.protocol = endpoint.protocol === 'https:' ? 'wss:' : 'ws:';
  return endpoint.toString();
};
