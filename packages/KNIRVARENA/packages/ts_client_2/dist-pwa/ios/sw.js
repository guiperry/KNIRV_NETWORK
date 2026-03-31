// KNIRV Controller PWA Service Worker - IOS
const CACHE_NAME = 'knirv-controller-ios-v1.0.0';
const OFFLINE_URL = './index.html';

const ESSENTIAL_RESOURCES = [
  './',
  './index.html',
  './manifest.json',
  './assets/main-CLN7z47v.js',
  './install.js',
  './install.css',
  './assets/main-CrBmO3ty.css',
  './assets/vendor-BXttUi4B.js',
  './assets/ui-BklU9til.js',
  './assets/blockchain-DRtzGIWH.js',
  './assets/database-CeK0uOht.js',
  './icons/icon-152x152.png',
  './icons/icon-180x180.png',
  './icons/icon-32x32.png',
  './icons/icon-16x16.png'
];

self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => cache.addAll(ESSENTIAL_RESOURCES))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys()
      .then(cacheNames => Promise.all(cacheNames.filter(n => n !== CACHE_NAME).map(n => caches.delete(n))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', event => {
  if (event.request.mode === 'navigate') {
    event.respondWith(
      fetch(event.request).catch(() => caches.match('./index.html'))
    );
  } else {
    event.respondWith(
      caches.match(event.request)
        .then(response => response || fetch(event.request))
        .catch(() => {
          if (event.request.destination === 'document') {
            return caches.match('./index.html');
          }
        })
    );
  }
});