# KNIRVSERVER PWA Refactor: Feasibility Report & Implementation Plan

## 1. Executive Summary
This report analyzes the feasibility and provides a detailed plan for refactoring the **KNIRVSERVER Desktop Application** (currently an Electron-based HUD) into a **Progressive Web Application (PWA)**. 

The goal is to allow users to "download" and install the client version of the application directly from their browser, providing a native-like experience with lower overhead, easier updates, and broader platform compatibility.

## 2. Current Architecture Overview (Aligned with Gateway Proxy Strategy)

The KNIRVSERVER PWA refactor operates within the established Gateway Proxy Architecture:

1.  **KNIRVSERVER PWA Wrapper** (public entrypoint on configured user-facing port, e.g., `8090`)
   - Serves the static Next.js frontend and PWA manifest
   - Proxies frontend API requests to internal backend Unix socket
   - Maintains HUD overlay as React component (no Electron runtime)
   - Acts as the single public-facing layer for the client

2.  **Next.js Frontend** (static export, served from wrapper)
   - Progressive Web Application with Service Worker
   - Browser-installable via PWA manifest
   - Makes internal API calls via the wrapper proxy layer
   - Includes React-based HUD overlay (replaces Electron HTML/CSS)

3.  **Backend Services** (internal Unix sockets, NOT publicly exposed)
   - System info/metrics service (socket-backed)
   - All internal APIs bound to Unix sockets, never direct TCP ports
   - Accessed exclusively through KNIRVSERVER's proxy layer

This architecture aligns with the Gateway Proxy Strategy documented in `Gateway_Proxy_Strategy.md`:
- All internal services bind to Unix sockets for security
- KNIRVSERVER acts as the public wrapper layer
- Browser/client traffic flows through KNIRVSERVER → internal sockets
- No direct browser access to internal service ports

## 3. Feasibility Analysis

### 3.1 Advantages of PWA Refactor
-   **Platform Agnostic**: Works on Windows, macOS, Linux, and even mobile without platform-specific builds.
-   **Reduced Overhead**: Eliminates the Electron runtime, significantly reducing memory and disk usage.
-   **Streamlined Updates**: Updates are delivered automatically when the user refreshes or the service worker detects a new version.
-   **Installability**: Users can still "install" the app to their desktop/dock via the browser's "Install" prompt.

### 3.2 Challenges & Mitigations
| Challenge | Impact | Mitigation Strategy |
| :--- | :--- | :--- |
| **System Metrics** | The current HUD uses Node.js `os` module. | Expose system metrics endpoint on Unix socket backend; KNIRVSERVER proxies to `/api/v1/system/info` |
| **Socket-Based Architecture** | Backend runs on Unix sockets per Gateway Strategy. | KNIRVSERVER acts as HTTP wrapper that proxies requests to internal socket-backed services |
| **Window Controls** | PWA doesn't have native "Minimize" or "Close" buttons. | Use standard browser/OS window controls. The HUD's "Close" button can be repurposed to "Log Out" or "Close Connection". |
| **Local File Access** | Limited in browsers compared to Electron. | Use the File System Access API where needed, or proxy file operations through KNIRVSERVER backend proxy. |
| **Offline Support** | Essential for a "client" feel. | Implement a robust Service Worker strategy (Cache First) to ensure the UI is always available. |
| **Direct Port Exposure** | Gateway Strategy forbids raw internal service ports. | All API endpoints served through KNIRVSERVER wrapper; internal backend never directly exposed to browser. |

## 4. Implementation Plan

### Phase 1: HUD & UI Consolidation
-   **Port HUD to React**: Extract the HUD layout from `packages/KNIRVSERVER/desktop/index.html` and `styles.css` and create a `HudLayout` component in Next.js.
-   **Unified Login/Menu**: Integrate the login and constellation menu logic directly into the Next.js app routes (`/login`, `/menu`).
-   **Theme Synchronization**: Ensure the HUD styling matches the existing Next.js frontend design language.

### Phase 2: Backend API Expansion (Socket-Backed)
-   **System Info Endpoint**: Implement a new handler in `packages/KNIRVSERVER/backend/internal/api` (running on Unix socket) that utilizes the existing `SystemInfoCollector` in `internal/utils/host/system_info.go`.
-   **KNIRVSERVER Proxy Route**: Add reverse proxy route in KNIRVSERVER wrapper that maps `/api/v1/system/info` to the backend Unix socket endpoint.
-   **WebSocket Metrics**: Implement a WebSocket stream on the Unix socket backend; expose through KNIRVSERVER wrapper for real-time performance metrics (CPU, Memory, Network RX/TX) to power the HUD charts.
-   **Proxy Configuration**: Update KNIRVSERVER config to specify internal socket paths and their corresponding public proxy routes (e.g., socket path `/var/run/knirvserver.sock` → public route `/api/v1/`).

### Phase 3: PWA Integration
-   **Manifest Generation**: Create `public/manifest.json` with appropriate icons, theme colors, and display modes (`standalone`).
-   **Service Worker**: Implement a service worker (using `next-pwa` or a custom script) to cache the static export files.
-   **Icons**: Generate a set of KNIRV-branded icons (192x192, 512x512) for the PWA.

### Phase 4: Build & Deployment Workflow (Gateway-Aware)
-   **Static Export**: Continue using Next.js `output: 'export'` to generate a static site.
-   **KNIRVSERVER Embedding**: The KNIRVSERVER wrapper will embed the `frontend/out` directory and serve it on the public user-facing port.
-   **Socket Configuration**: Backend services run on Unix sockets (e.g., `/var/run/knirvserver-backend.sock`), never on direct TCP ports.
-   **Proxy Middleware**: KNIRVSERVER includes reverse proxy middleware to route `/api/v1/*` requests to the internal backend Unix socket.
-   **MIME Types**: KNIRVSERVER serves `manifest.json` and `service-worker.js` with correct MIME types via proxy middleware.
-   **Deployment**: The user "downloads" the client by navigating to the KNIRVSERVER wrapper (single public port) and clicking "Install". No direct backend port exposure.

## 5. Gateway Proxy Strategy Compliance

This PWA implementation fully complies with the established **Gateway Proxy Strategy** (`docs/Gateway_Proxy_Strategy.md`). The following architectural decisions ensure alignment:

### 5.1 Socket-Based Internal Communication
- **Backend Service**: Runs on a Unix socket (e.g., `/var/run/knirvserver-backend.sock`), never on direct TCP ports
- **Security Benefit**: Unix sockets are local-only and file-permission controlled; internal services are firewall-protected by default
- **Compliance**: All system metrics endpoints, WebSocket streams, and API routes bound to sockets per Gateway Strategy

### 5.2 KNIRVSERVER as Single Public Layer
- **Public Entry Point**: KNIRVSERVER wrapper exposes a single user-facing port (e.g., `:8090`)
- **No Backend Port Exposure**: The Go backend never accepts external connections; all client traffic routes through the wrapper
- **Reverse Proxy**: KNIRVSERVER's reverse proxy middleware (`setupProxyRoutes`) transparently routes `/api/v1/*` requests to the internal Unix socket backend
- **Compliance**: Browser config generation and client code always reference wrapper routes (`/api/v1/*`), never internal service ports

### 5.3 Frontend URL Generation
The React frontend and Service Worker are configured to use **only wrapper-relative URLs**:

```typescript
// ✅ Correct - uses wrapper proxy route
const response = await fetch('/api/v1/system/info');

// ❌ Never - hardcoded internal port
// const response = await fetch('http://localhost:9000/api/v1/system/info');
```

Service Worker caching strategies also respect this:
- **API routes** (`/api/v1/*`): Network First via wrapper proxy
- **Static assets** (`/`, `/icons/*`, `/screenshots/*`): Cache First from wrapper

### 5.4 Architecture Diagram

```
┌─────────────────────────────────────────┐
│         Browser / PWA Client            │
├─────────────────────────────────────────┤
│ - React Frontend (Next.js static export)│
│ - Service Worker (offline support)      │
│ - HUD React Component                   │
└──────────────────┬──────────────────────┘
                   │ HTTP/HTTPS
                   │ All requests use relative /api/* paths
                   ▼
┌─────────────────────────────────────────────┐
│      KNIRVSERVER Wrapper (Port 8090)        │
├─────────────────────────────────────────────┤
│ - Static frontend server (/frontend/out)    │
│ - PWA manifest & service worker handler     │
│ - Reverse proxy middleware                  │
└──────┬──────────────────────────┬───────────┘
       │                          │
       │ /api/v1/* routes         │ Static files
       │ (to Unix socket)         │ (manifest, icons)
       ▼                          ▼
┌────────────────────────────────────────┐
│ Backend Services (Unix Socket)         │
│ /var/run/knirvserver-backend.sock      │
├────────────────────────────────────────┤
│ - GET /api/v1/system/info              │
│ - WS /api/v1/system/metrics/stream     │
│ - Other internal API routes            │
└────────────────────────────────────────┘
```

### 5.5 Configuration Example

**KNIRVSERVER Config** (`packages/KNIRVSERVER/config.yaml`):

```yaml
server:
  # Public-facing port (single entry point)
  port: 8090
  host: "0.0.0.0"
  
frontend:
  # Static export served by wrapper
  path: "./frontend/out"
  
backend:
  # Internal Unix socket (never exposed)
  socket: "/var/run/knirvserver-backend.sock"
  
proxy:
  # Map public routes to Unix socket backend
  routes:
    - path: "/api/v1"
      target: "unix:///var/run/knirvserver-backend.sock"
      protocol: "http"
```

### 5.6 Compliance Checklist

- [x] Backend runs on Unix socket, not TCP port
- [x] KNIRVSERVER is the single public HTTP entry point
- [x] Frontend never hardcodes internal service ports
- [x] Service Worker cache strategies use wrapper-relative paths
- [x] Reverse proxy transparently routes `/api/v1/*` to Unix socket backend
- [x] Static assets (manifest, icons) served through wrapper with correct MIME types
- [x] No browser config generation that emits raw internal service URLs
- [x] WebSocket endpoints proxied through KNIRVSERVER wrapper
- [x] All API calls respect wrapper proxy layer

## 6. Detailed Implementation Guide

### 5.1 System Info API Endpoint

**Backend Implementation** (`packages/KNIRVSERVER/backend/internal/api/system_handler.go`):

```go
package api

import (
	"net/http"
	"runtime"
	
	"github.com/gin-gonic/gin"
	"knirvserver/backend/internal/utils/host"
)

type SystemMetrics struct {
	CPU    float64 `json:"cpu"`
	Memory MemoryInfo `json:"memory"`
	Uptime int64 `json:"uptime_seconds"`
	OS     string `json:"os"`
	Arch   string `json:"arch"`
}

type MemoryInfo struct {
	Total       uint64 `json:"total_mb"`
	Used        uint64 `json:"used_mb"`
	Available   uint64 `json:"available_mb"`
	Percentage  float64 `json:"percentage"`
}

func (h *Handler) GetSystemInfo(c *gin.Context) {
	metrics, err := host.CollectSystemMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SystemMetrics{
		CPU:    metrics.CPUPercent,
		Memory: MemoryInfo{
			Total:      metrics.MemoryTotal / 1024 / 1024,
			Used:       metrics.MemoryUsed / 1024 / 1024,
			Available:  metrics.MemoryAvailable / 1024 / 1024,
			Percentage: (float64(metrics.MemoryUsed) / float64(metrics.MemoryTotal)) * 100,
		},
		Uptime: metrics.Uptime,
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
	})
}

// WebSocket handler for real-time metrics streaming
func (h *Handler) StreamSystemMetrics(c *gin.Context) {
	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade connection"})
		return
	}
	defer ws.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics, err := host.CollectSystemMetrics()
		if err != nil {
			continue
		}

		ws.WriteJSON(metrics)
	}
}
```

**Register routes on Unix socket backend** in `backend/cmd/backend_server/main.go`:

```go
// Backend runs on Unix socket, NOT HTTP
listener, err := net.Listen("unix", "/var/run/knirvserver-backend.sock")
if err != nil {
  log.Fatal(err)
}
defer listener.Close()

router := gin.Default()
router.GET("/api/v1/system/info", apiHandler.GetSystemInfo)
router.GET("/api/v1/system/metrics/stream", apiHandler.StreamSystemMetrics)
router.RunListener(listener)
```

**KNIRVSERVER wrapper proxies requests** in `packages/KNIRVSERVER/main.go`:

```go
import "net/http/httputil"

func setupProxyRoutes(router *gin.Engine) {
  // Backend Unix socket proxy
  backendURL := "unix:///var/run/knirvserver-backend.sock"
  
  // Create reverse proxy director for Unix socket
  director := func(req *http.Request) {
    req.URL.Scheme = "http"
    req.URL.Host = "unix"
    // Request path remains /api/v1/system/info, etc.
  }
  
  // Use httputil.ReverseProxy with Unix socket dialer
  proxy := &httputil.ReverseProxy{
    Director: director,
    Transport: &http.Transport{
      Dial: func(network, addr string) (net.Conn, error) {
        return net.Dial("unix", "/var/run/knirvserver-backend.sock")
      },
    },
  }
  
  router.Any("/api/v1/*path", gin.WrapH(proxy))
}
```

### 5.2 PWA Manifest Configuration

**File**: `packages/KNIRVSERVER/frontend/public/manifest.json`

```json
{
  "name": "KNIRVSERVER Client",
  "short_name": "KNIRV",
  "description": "Decentralized Trusted Execution Network Client",
  "start_url": "/",
  "scope": "/",
  "display": "standalone",
  "theme_color": "#000000",
  "background_color": "#ffffff",
  "orientation": "portrait-primary",
  "icons": [
    {
      "src": "/icons/knirv-192x192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/knirv-512x512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/knirv-192x192-maskable.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "maskable"
    }
  ],
  "screenshots": [
    {
      "src": "/screenshots/screenshot-540x720.png",
      "sizes": "540x720",
      "type": "image/png",
      "form_factor": "narrow"
    },
    {
      "src": "/screenshots/screenshot-1280x720.png",
      "sizes": "1280x720",
      "type": "image/png",
      "form_factor": "wide"
    }
  ],
  "shortcuts": [
    {
      "name": "Dashboard",
      "short_name": "Dashboard",
      "description": "View your KNIRV dashboard",
      "url": "/dashboard",
      "icons": [
        {
          "src": "/icons/dashboard-192x192.png",
          "sizes": "192x192"
        }
      ]
    },
    {
      "name": "Metrics",
      "short_name": "Metrics",
      "description": "View system metrics",
      "url": "/metrics",
      "icons": [
        {
          "src": "/icons/metrics-192x192.png",
          "sizes": "192x192"
        }
      ]
    }
  ]
}
```

**Update** `packages/KNIRVSERVER/frontend/next.config.js`:

```javascript
const withPWA = require('next-pwa')({
  dest: 'public',
  register: true,
  skipWaiting: false,
  runtimeCaching: [
    {
      urlPattern: /^https:\/\/fonts\.googleapis\.com\/.*/i,
      handler: 'CacheFirst',
      options: {
        cacheName: 'google-fonts-cache',
        expiration: {
          maxEntries: 20,
        },
      },
    },
    {
      urlPattern: /^https:\/\/fonts\.gstatic\.com\/.*/i,
      handler: 'CacheFirst',
      options: {
        cacheName: 'google-fonts-webfont-cache',
        expiration: {
          maxEntries: 30,
        },
      },
    },
    {
      urlPattern: /^\/api\/.*/i,
      handler: 'NetworkFirst',
      options: {
        cacheName: 'api-cache',
        networkTimeoutSeconds: 5,
        expiration: {
          maxEntries: 50,
          maxAgeSeconds: 300,
        },
      },
    },
  ],
});

module.exports = withPWA({
  reactStrictMode: true,
  output: 'export',
  basePath: '',
  images: {
    unoptimized: true,
  },
});
```

### 5.3 Service Worker Implementation

**File**: `packages/KNIRVSERVER/frontend/public/service-worker.js` (custom implementation):

```javascript
const CACHE_NAME = 'knirv-pwa-v1';
const urlsToCache = [
  '/',
  '/index.html',
  '/manifest.json',
  '/styles/globals.css',
  '/icons/knirv-192x192.png',
  '/icons/knirv-512x512.png',
];

// Install event - cache resources
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(urlsToCache);
    })
  );
});

// Activate event - cleanup old caches
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
});

// Fetch event - implement cache strategy
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // API calls - Network First
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          if (!response || response.status !== 200) {
            return response;
          }
          const responseClone = response.clone();
          caches.open(CACHE_NAME).then((cache) => {
            cache.put(request, responseClone);
          });
          return response;
        })
        .catch(() => {
          return caches.match(request).then((response) => {
            return response || new Response('Offline - API unavailable', {
              status: 503,
              statusText: 'Service Unavailable',
            });
          });
        })
    );
    return;
  }

  // Static assets - Cache First
  event.respondWith(
    caches.match(request).then((response) => {
      return response || fetch(request).then((response) => {
        if (!response || response.status !== 200 || response.type === 'error') {
          return response;
        }
        const responseClone = response.clone();
        caches.open(CACHE_NAME).then((cache) => {
          cache.put(request, responseClone);
        });
        return response;
      });
    })
  );
});
```

### 5.4 React HUD Component

**File**: `packages/KNIRVSERVER/frontend/src/components/HudOverlay.tsx`

```typescript
import React, { useEffect, useState, useCallback } from 'react';
import styles from './HudOverlay.module.css';

interface SystemMetrics {
  cpu: number;
  memory: {
    total_mb: number;
    used_mb: number;
    available_mb: number;
    percentage: number;
  };
  uptime_seconds: number;
  os: string;
  arch: string;
}

export const HudOverlay: React.FC = () => {
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [isMinimized, setIsMinimized] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState('connected');

  const fetchMetrics = useCallback(async () => {
    try {
      const response = await fetch('/api/v1/system/info');
      if (!response.ok) {
        setConnectionStatus('error');
        return;
      }
      const data = await response.json();
      setMetrics(data);
      setConnectionStatus('connected');
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
      setConnectionStatus('error');
    }
  }, []);

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 2000);
    return () => clearInterval(interval);
  }, [fetchMetrics]);

  const formatUptime = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
  };

  if (isMinimized) {
    return (
      <div className={styles.minimized}>
        <button onClick={() => setIsMinimized(false)} title="Restore HUD">
          ▲
        </button>
      </div>
    );
  }

  return (
    <div className={styles.hudContainer}>
      <div className={styles.hudHeader}>
        <span className={styles.title}>KNIRV System Monitor</span>
        <div className={styles.controls}>
          <span className={`${styles.status} ${styles[connectionStatus]}`}>
            {connectionStatus}
          </span>
          <button onClick={() => setIsMinimized(true)} title="Minimize">
            ▼
          </button>
        </div>
      </div>

      <div className={styles.metrics}>
        {metrics ? (
          <>
            <div className={styles.metric}>
              <label>CPU</label>
              <div className={styles.bar}>
                <div
                  className={styles.fill}
                  style={{ width: `${Math.min(metrics.cpu, 100)}%` }}
                />
              </div>
              <span>{metrics.cpu.toFixed(1)}%</span>
            </div>

            <div className={styles.metric}>
              <label>Memory</label>
              <div className={styles.bar}>
                <div
                  className={styles.fill}
                  style={{
                    width: `${Math.min(metrics.memory.percentage, 100)}%`,
                  }}
                />
              </div>
              <span>
                {metrics.memory.used_mb.toFixed(0)}MB /
                {metrics.memory.total_mb.toFixed(0)}MB
              </span>
            </div>

            <div className={styles.info}>
              <span>Uptime: {formatUptime(metrics.uptime_seconds)}</span>
              <span>
                {metrics.os} {metrics.arch}
              </span>
            </div>
          </>
        ) : (
          <div className={styles.loading}>Loading metrics...</div>
        )}
      </div>
    </div>
  );
};
```

**Styles**: `packages/KNIRVSERVER/frontend/src/components/HudOverlay.module.css`

```css
.hudContainer {
  position: fixed;
  top: 12px;
  right: 12px;
  width: 320px;
  background: rgba(0, 0, 0, 0.85);
  border: 1px solid #00ff00;
  border-radius: 4px;
  box-shadow: 0 0 20px rgba(0, 255, 0, 0.3);
  color: #00ff00;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  z-index: 9999;
  user-select: none;
}

.hudHeader {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px;
  border-bottom: 1px solid #00ff00;
  background: rgba(0, 20, 0, 0.5);
}

.title {
  font-weight: bold;
  font-size: 11px;
  letter-spacing: 1px;
}

.controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.status {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 2px;
  background: rgba(0, 255, 0, 0.1);
  border: 1px solid #00ff00;
}

.status.error {
  color: #ff0000;
  border-color: #ff0000;
  background: rgba(255, 0, 0, 0.1);
}

button {
  background: none;
  border: none;
  color: #00ff00;
  cursor: pointer;
  font-size: 10px;
  padding: 2px 6px;
  transition: all 0.2s;
}

button:hover {
  background: rgba(0, 255, 0, 0.2);
  border-radius: 2px;
}

.metrics {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.metric {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric label {
  font-size: 11px;
  font-weight: bold;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.bar {
  height: 12px;
  background: rgba(0, 255, 0, 0.1);
  border: 1px solid #00ff00;
  border-radius: 2px;
  overflow: hidden;
}

.fill {
  height: 100%;
  background: linear-gradient(90deg, #00ff00, #00aa00);
  transition: width 0.3s ease;
  box-shadow: 0 0 8px rgba(0, 255, 0, 0.8);
}

.info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid rgba(0, 255, 0, 0.3);
  font-size: 10px;
}

.minimized {
  position: fixed;
  top: 12px;
  right: 12px;
  width: 32px;
  height: 32px;
  background: rgba(0, 0, 0, 0.85);
  border: 1px solid #00ff00;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.minimized button {
  width: 100%;
  height: 100%;
  font-size: 16px;
  padding: 0;
}

.loading {
  padding: 12px;
  text-align: center;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
```

### 5.5 Installation Prompt Component

**File**: `packages/KNIRVSERVER/frontend/src/hooks/usePWAInstall.ts`

```typescript
import { useEffect, useState, useCallback } from 'react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

export const usePWAInstall = () => {
  const [installPrompt, setInstallPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [isInstalled, setIsInstalled] = useState(false);

  useEffect(() => {
    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault();
      setInstallPrompt(e as BeforeInstallPromptEvent);
    };

    const handleAppInstalled = () => {
      setIsInstalled(true);
      setInstallPrompt(null);
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    window.addEventListener('appinstalled', handleAppInstalled);

    // Check if already installed
    if (window.matchMedia('(display-mode: standalone)').matches) {
      setIsInstalled(true);
    }

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
      window.removeEventListener('appinstalled', handleAppInstalled);
    };
  }, []);

  const install = useCallback(async () => {
    if (!installPrompt) return;

    installPrompt.prompt();
    const choice = await installPrompt.userChoice;
    
    if (choice.outcome === 'accepted') {
      console.log('PWA installation accepted');
    }
    
    setInstallPrompt(null);
  }, [installPrompt]);

  return { installPrompt, isInstalled, install };
};
```

**Usage in component**:

```typescript
import { usePWAInstall } from '@/hooks/usePWAInstall';

export const InstallPrompt = () => {
  const { installPrompt, install } = usePWAInstall();

  if (!installPrompt) return null;

  return (
    <div className="install-banner">
      <p>Install KNIRVSERVER Client for a native experience</p>
      <button onClick={install}>Install</button>
    </div>
  );
};
```

### 5.6 KNIRVSERVER Wrapper Middleware for PWA & Proxy

**Go Middleware** (`packages/KNIRVSERVER/internal/middleware/pwa.go`):

```go
package middleware

import (
	"net/http"
	"strings"
	
	"github.com/gin-gonic/gin"
)

func PWAHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Service Worker must always be fresh (not cached internally)
		if strings.HasSuffix(c.Request.URL.Path, "service-worker.js") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Content-Type", "application/javascript; charset=utf-8")
		}

		// Manifest file configuration
		if strings.HasSuffix(c.Request.URL.Path, "manifest.json") {
			c.Header("Content-Type", "application/manifest+json; charset=utf-8")
			c.Header("Cache-Control", "public, max-age=3600")
		}

		c.Next()
	}
}

// ProxySocketPath returns the Unix socket path for backend services
func ProxySocketPath() string {
	return "/var/run/knirvserver-backend.sock"
}
```

**Register middleware and proxy routes in KNIRVSERVER wrapper main** (`packages/KNIRVSERVER/main.go`):

```go
router.Use(middleware.PWAHeaders())
// Static assets and frontend routes
router.NoRoute(gin.WrapH(http.FileServer(http.Dir("frontend/out"))))
// Backend API proxy (routes /api/v1/* to Unix socket)
setupProxyRoutes(router)
```

## 7. Browser Compatibility & Requirements

| Browser | Minimum Version | PWA Support | Notes |
|---------|-----------------|-------------|-------|
| Chrome/Edge | 51+ | ✅ Full | Install prompt, offline support |
| Firefox | 44+ | ⚠️ Partial | Service Workers only; install prompt in Firefox 58+ |
| Safari | 11.1+ | ⚠️ Limited | Service Workers; no install prompt until iOS 15.4 |
| Opera | 38+ | ✅ Full | Full PWA support |
| Samsung Internet | 5+ | ✅ Full | Full PWA support |

## 8. Security Considerations

### 8.1 HTTPS Requirement
- PWAs require HTTPS (or localhost for development)
- Use valid SSL certificates in production
- Service Workers will not register over HTTP

### 8.2 Content Security Policy
```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' wss:
```

### 8.3 CORS Headers (Socket-Aware)
Ensure proper CORS configuration for API endpoints:

```go
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}
```

## 9. Testing Strategy

### 9.1 Unit Tests - Service Worker

```javascript
// public/__tests__/service-worker.test.js
describe('Service Worker', () => {
  it('should cache static assets on install', async () => {
    const cache = await caches.open('test-cache');
    await cache.add('/');
    const cached = await cache.match('/');
    expect(cached).toBeDefined();
  });

  it('should return cached response on fetch', async () => {
    // Mock fetch event
  });
});
```

### 9.2 E2E Tests - PWA Installation

```typescript
// tests/pwa-install.e2e.ts
describe('PWA Installation', () => {
  it('should display install banner on install-prompt event', () => {
    // Simulate beforeinstallprompt event
    // Assert banner appears
  });

  it('should persist offline after installation', async () => {
    // Go offline
    // Assert app remains functional
  });
});
```

### 9.3 Performance Audits
```bash
# Use Lighthouse in CI/CD
lighthouse https://localhost:8084 --chrome-flags="--headless" --output-path=./reports/lighthouse.html
```

## 10. Migration Checklist

- [ ] Set up next-pwa or custom service worker
- [ ] Create PWA manifest.json with icons
- [ ] Implement system metrics API endpoint (`/api/v1/system/info`)
- [ ] Port HUD from Electron to React component
- [ ] Update Next.js config for static export and PWA settings
- [ ] Add PWA headers middleware to Go backend
- [ ] Add HTTPS/SSL certificates for development and production
- [ ] Test offline functionality with service worker
- [ ] Update documentation for users on how to install PWA
- [ ] Set up pre-commit hook with Caliber
- [ ] Verify installation prompt works across browsers
- [ ] Run Lighthouse audit and achieve 90+ score
- [ ] Test on multiple devices (desktop, tablet, mobile)
- [ ] Update CI/CD pipeline for PWA build process
- [ ] Monitor service worker errors in production

## 11. Performance Considerations

### 11.1 Bundle Size Optimization
- Use dynamic imports for code splitting
- Tree-shake unused dependencies
- Minify and compress service worker
- Target asset size under 1MB for fast installation

### 11.2 Caching Strategy
- Static assets: Cache First (with 30-day expiration)
- API calls: Network First (5s timeout, fall back to cache)
- HTML pages: Stale While Revalidate
- Images: Cache First (with 7-day expiration)

### 11.3 Updating Strategy
```typescript
// Detect new service worker version
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.ready.then((registration) => {
    registration.addEventListener('updatefound', () => {
      const newWorker = registration.installing;
      newWorker?.addEventListener('statechange', () => {
        if (newWorker.state === 'installed') {
          // Show "Update available" notification
          showUpdateBanner();
        }
      });
    });
  });
}
```

## 12. Troubleshooting Guide

### Service Worker Registration Fails
- **Check**: Browser console for errors
- **Verify**: HTTPS is enabled (or localhost)
- **Ensure**: Service worker file exists at `/service-worker.js`
- **Validate**: Manifest.json path is correct in `<link>` tag

### Offline Mode Not Working
- **Verify**: Service worker fetch event is implemented
- **Check**: Cache APIs are being called correctly
- **Test**: Go offline in DevTools and refresh

### Install Prompt Not Showing
- **Check**: beforeinstallprompt event on Android/desktop Chrome
- **Note**: Not supported in Safari without icon configuration
- **Verify**: Installation criteria met (HTTPS, manifest.json, icons)
- **Clear**: Browser cache and try reinstalling

## 13. Timeline & Milestones
1.  **Milestone 1 (Week 1)**: HUD componentized in Next.js and backend System API active.
2.  **Milestone 2 (Week 2)**: Unified Login/Menu flow and PWA manifest implementation.
3.  **Milestone 3 (Week 2)**: Service worker testing and offline mode validation.
4.  **Milestone 4 (Week 3)**: Final UX polish and documentation update.

## 14. Conclusion
Refactoring KNIRVSERVER to a PWA is not only feasible but highly recommended to reduce complexity and improve user accessibility. The existing Go-based system monitoring capabilities are already sufficient to replace the Node.js-specific logic in the current Electron HUD.

With the detailed implementation guide, code snippets, and comprehensive testing strategy provided above, the team can execute this migration with confidence and deliver a more performant, maintainable, and user-friendly client platform.
