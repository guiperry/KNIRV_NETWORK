const CACHE_NAME = 'knirv-pwa-v1';
const STATIC_CACHE_NAME = 'knirv-static-v1';
const API_CACHE_NAME = 'knirv-api-v1';

const urlsToCache = [
  '/',
  '/index.html',
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
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== STATIC_CACHE_NAME && cacheName !== API_CACHE_NAME) {
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
  self.clients.claim();
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

  // Let the browser handle these routes directly — do NOT call
  // event.respondWith().  This avoids:
  //   1. "promise was rejected" when an iframe navigation aborts the fetch
  //   2. 503 fallback from networkFirst() returning JSON for an HTML page
  if (url.pathname === '/login' || url.pathname === '/menu' || url.pathname.startsWith('/menu')) {
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

  return cachedResponse || fetchPromise;
}

self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});