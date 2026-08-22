// Bump this whenever routing or authentication behavior changes. Existing
// clients activate this worker immediately and discard the old app shell.
const CACHE_NAME = 'knirv-pwa-v3';
const STATIC_CACHE_NAME = 'knirv-static-v3';
const API_CACHE_NAME = 'knirv-api-v3';

const urlsToCache = [
  '/manifest.json',
  '/favicon.ico',
];

const API_CACHE_DURATION = 5 * 60 * 1000;
const STATIC_CACHE_DURATION = 30 * 24 * 60 * 60 * 1000;

self.addEventListener('install', (event) => {
  console.log('Service worker installing');
  event.waitUntil(
    caches.open(STATIC_CACHE_NAME).then((cache) => {
      return cache.addAll(urlsToCache);
    })
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then(async (cacheNames) => {
      await Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== STATIC_CACHE_NAME && cacheName !== API_CACHE_NAME) {
            return caches.delete(cacheName);
          }
        })
      );
      await self.clients.claim();

      // A client that was booted from the previous app shell still has its old
      // JavaScript in memory. Reload it once under this worker after a cache
      // version upgrade, preserving its current route.
      const openClients = await self.clients.matchAll({ type: 'window' });
      await Promise.all(openClients.map((client) => client.navigate(client.url)));
    })
  );
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Cache API only supports GET — skip all other methods entirely
  if (request.method !== 'GET') {
    return;
  }

  if (url.pathname.startsWith('/api/')) {
    event.respondWith(networkFirst(request));
    return;
  }

  if (url.pathname === '/service-worker.js') {
    return;
  }

  // Never cache app-router documents or React Server Component flight data.
  // Serving a stale `/` document or `index.txt?_rsc=…` response can combine
  // an old route tree with a new bundle and reintroduce a redirect loop.
  // Hashed static assets remain safe to cache below.
  if (request.mode === 'navigate' || url.searchParams.has('_rsc') || url.pathname.endsWith('/index.txt')) {
    return;
  }

  if (url.pathname.startsWith('/icons/') || 
      url.pathname.startsWith('/screenshots/') ||
      url.pathname.endsWith('.png') ||
      url.pathname.endsWith('.jpg') ||
      url.pathname.endsWith('.svg') ||
      url.pathname.endsWith('.woff2')) {
    event.respondWith(cacheFirst(request, STATIC_CACHE_NAME));
    return;
  }

  event.respondWith(staleWhileRevalidate(request, STATIC_CACHE_NAME));
});

async function networkFirst(request) {
  try {
    const networkResponse = await fetch(request);

    if (networkResponse.ok && request.method === 'GET') {
      const responseClone = networkResponse.clone();
      const cache = await caches.open(API_CACHE_NAME);
      cache.put(request, responseClone);
    }
    
    return networkResponse;
  } catch (error) {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }

    return new Response(JSON.stringify({ error: 'Offline - API unavailable' }), {
      status: 503,
      statusText: 'Service Unavailable',
      headers: { 'Content-Type': 'application/json' },
    });
  }
}

async function cacheFirst(request, cacheName) {
  const cachedResponse = await caches.match(request);

  if (cachedResponse) {
    return cachedResponse;
  }

  try {
    const networkResponse = await fetch(request);

    if (networkResponse.ok) {
      const responseClone = networkResponse.clone();
      const cache = await caches.open(cacheName);
      cache.put(request, responseClone);
    }

    return networkResponse;
  } catch (error) {
    return new Response('Resource not available offline', {
      status: 503,
      statusText: 'Service Unavailable',
    });
  }
}

async function staleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);

  const fetchPromise = fetch(request)
    .then((networkResponse) => {
      if (networkResponse.ok) {
        cache.put(request, networkResponse.clone());
      }
      return networkResponse;
    })
    .catch(() => cachedResponse);

  // Return cached response immediately, or wait for network.
  // If both are unavailable, serve a minimal HTML fallback so
  // event.respondWith() never receives undefined.
  const response = cachedResponse || await fetchPromise;
  if (response) {
    return response;
  }

  return new Response(
    '<!DOCTYPE html><title>KNIRV Network</title><p>Offline — please check your connection.</p>',
    {
      status: 503,
      statusText: 'Service Unavailable',
      headers: { 'Content-Type': 'text/html; charset=utf-8' },
    }
  );
}

self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});
