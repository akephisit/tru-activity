const CACHE_NAME = 'tru-activity-v1';
const OFFLINE_URL = '/offline.html';

// Assets to cache for offline functionality
const ASSETS_TO_CACHE = [
  '/',
  '/dashboard',
  '/login',
  '/manifest.json',
  '/offline.html',
  // Add your main CSS and JS files here
];

// API routes that should be cached
const API_CACHE_ROUTES = [
  '/query', // GraphQL endpoint
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
  console.log('Service Worker: Installing...');
  
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('Service Worker: Caching assets');
        return cache.addAll(ASSETS_TO_CACHE);
      })
      .then(() => {
        console.log('Service Worker: Assets cached');
        return self.skipWaiting();
      })
      .catch((error) => {
        console.error('Service Worker: Cache failed', error);
      })
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('Service Worker: Activating...');
  
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            console.log('Service Worker: Deleting old cache', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => {
      console.log('Service Worker: Activated');
      return self.clients.claim();
    })
  );
});

// Fetch event - handle network requests
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Handle navigation requests
  if (request.mode === 'navigate') {
    event.respondWith(
      fetch(request)
        .then((response) => {
          // If online, return the response and cache it
          if (response.ok) {
            const responseClone = response.clone();
            caches.open(CACHE_NAME).then((cache) => {
              cache.put(request, responseClone);
            });
          }
          return response;
        })
        .catch(() => {
          // If offline, try to serve from cache
          return caches.match(request)
            .then((response) => {
              if (response) {
                return response;
              }
              // If no cache, serve offline page
              return caches.match(OFFLINE_URL);
            });
        })
    );
    return;
  }

  // Handle API requests with network-first strategy
  if (url.pathname.includes('/query') || API_CACHE_ROUTES.some(route => url.pathname.includes(route))) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          // Only cache successful GET requests
          if (response.ok && request.method === 'GET') {
            const responseClone = response.clone();
            caches.open(CACHE_NAME).then((cache) => {
              cache.put(request, responseClone);
            });
          }
          return response;
        })
        .catch(() => {
          // If offline, try to serve from cache
          if (request.method === 'GET') {
            return caches.match(request);
          }
          // For non-GET requests, return a custom offline response
          return new Response(
            JSON.stringify({
              error: 'Network unavailable',
              message: 'This action requires an internet connection'
            }),
            {
              status: 503,
              statusText: 'Service Unavailable',
              headers: { 'Content-Type': 'application/json' }
            }
          );
        })
    );
    return;
  }

  // Handle other assets with cache-first strategy
  event.respondWith(
    caches.match(request)
      .then((response) => {
        if (response) {
          return response;
        }

        return fetch(request)
          .then((response) => {
            // Cache successful responses
            if (response.ok) {
              const responseClone = response.clone();
              caches.open(CACHE_NAME).then((cache) => {
                cache.put(request, responseClone);
              });
            }
            return response;
          });
      })
  );
});

// Background sync for offline actions
self.addEventListener('sync', (event) => {
  console.log('Service Worker: Background sync', event.tag);
  
  if (event.tag === 'qr-scan-sync') {
    event.waitUntil(syncQRScans());
  }
  
  if (event.tag === 'participation-sync') {
    event.waitUntil(syncParticipations());
  }
});

// Handle push notifications
self.addEventListener('push', (event) => {
  console.log('Service Worker: Push notification received');
  
  const options = {
    body: 'You have a new notification from TRU Activity',
    icon: '/icons/icon-192x192.png',
    badge: '/icons/badge-72x72.png',
    vibrate: [200, 100, 200],
    data: {
      dateOfArrival: Date.now(),
      primaryKey: 1
    },
    actions: [
      {
        action: 'explore',
        title: 'View Details',
        icon: '/icons/action-explore.png'
      },
      {
        action: 'close',
        title: 'Close',
        icon: '/icons/action-close.png'
      }
    ]
  };

  if (event.data) {
    const data = event.data.json();
    options.body = data.message || options.body;
    options.data = { ...options.data, ...data };
  }

  event.waitUntil(
    self.registration.showNotification('TRU Activity', options)
  );
});

// Handle notification clicks
self.addEventListener('notificationclick', (event) => {
  console.log('Service Worker: Notification clicked', event);
  
  event.notification.close();

  if (event.action === 'explore') {
    // Open the app to a specific page
    event.waitUntil(
      clients.openWindow('/dashboard')
    );
  } else if (event.action !== 'close') {
    // Default action - open the app
    event.waitUntil(
      clients.matchAll({ type: 'window', includeUncontrolled: true })
        .then((clientList) => {
          // Try to focus existing window
          for (const client of clientList) {
            if (client.url.includes('/dashboard') && 'focus' in client) {
              return client.focus();
            }
          }
          // Open new window if none exists
          if (clients.openWindow) {
            return clients.openWindow('/dashboard');
          }
        })
    );
  }
});

// Sync functions for offline actions
async function syncQRScans() {
  try {
    // Get pending QR scans from IndexedDB
    const pendingScans = await getPendingQRScans();
    
    for (const scan of pendingScans) {
      try {
        // Attempt to sync with server
        const response = await fetch('/query', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${scan.token}`
          },
          body: JSON.stringify(scan.mutation)
        });

        if (response.ok) {
          // Remove from pending list
          await removePendingQRScan(scan.id);
          
          // Show success notification
          self.registration.showNotification('QR Scan Synced', {
            body: 'Your offline QR scan has been processed',
            icon: '/icons/icon-192x192.png'
          });
        }
      } catch (error) {
        console.error('Failed to sync QR scan:', error);
      }
    }
  } catch (error) {
    console.error('Background sync failed:', error);
  }
}

async function syncParticipations() {
  // Similar implementation for participation syncing
  console.log('Syncing participations...');
}

// IndexedDB helpers (simplified)
async function getPendingQRScans() {
  // This would interact with IndexedDB to get pending scans
  return [];
}

async function removePendingQRScan(id) {
  // This would remove the scan from IndexedDB
  console.log('Removing pending QR scan:', id);
}

// Message handling for communication with main thread
self.addEventListener('message', (event) => {
  console.log('Service Worker: Message received', event.data);
  
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
  
  if (event.data && event.data.type === 'CACHE_QR_SCAN') {
    // Cache QR scan for offline sync
    cacheOfflineAction('qr-scan', event.data.payload);
  }
});

async function cacheOfflineAction(type, payload) {
  // Store action in IndexedDB for later sync
  console.log('Caching offline action:', type, payload);
}

// Periodic background sync
self.addEventListener('periodicsync', (event) => {
  if (event.tag === 'content-sync') {
    event.waitUntil(syncContent());
  }
});

async function syncContent() {
  // Sync application content when network is available
  console.log('Syncing content...');
}