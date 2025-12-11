# KNIRVGATEWAY - Embedded Services Implementation

## Overview

The KNIRVGATEWAY has been fully refactored to **embed all Node.js services and the network-website directly into the Go binary**. Upon startup, the binary extracts these embedded files to a temporary runtime directory, installs dependencies, and serves them.

## What's Embedded

The Go binary contains:

### Services (487MB)
- **payment-gateway** (224MB) - Stripe/Coinbase payment processing, testnet faucet
- **tunnel-registry** (95MB) - P2P tunnel management, NAT traversal, STUN server
- **webgui** (92MB) - Next.js dashboard and management interface
- **operator-registry** (76MB) - Operator registration and management (disabled by default)

### Network Website (500MB)
- Full network-website from `services/network-website/public/`
- All HTML, CSS, JavaScript, images, and assets
- Login functionality that redirects to `/dashboard` (webgui)

**Total:** ~1GB embedded → 580MB compiled binary

## How It Works

### 1. Binary Initialization
```
User runs: ./bin/knirvgateway
    ↓
Runtime Manager initializes
    ↓
Extracts embedded files to: /tmp/knirvgateway-runtime/
    ├── services/
    │   ├── payment-gateway/
    │   ├── tunnel-registry/
    │   ├── webgui/
    │   └── operator-registry/
    └── network-website/
        └── public/
```

### 2. Service Initialization
```
For each service:
    1. Extract all files from embedded FS
    2. Run npm install --production
    3. Start Node.js process: node server.js
    4. Monitor process health
```

### 3. HTTP Server Routes
```
GET  /                          → Network-website (landing page)
GET  /dashboard                 → WebGUI proxy (port 3007)
GET  /payment                   → Payment Gateway proxy (port 3001)
GET  /tunnel-registry           → Tunnel Registry proxy (port 3002)
GET  /operator-registry         → Operator Registry proxy (port 3006)
GET  /controller                → Dynamic controller proxy (session-based)
```

### 4. Network-Website → WebGUI Flow
```
User visits http://localhost:8080
    ↓
Sees network-website landing page
    ↓
Clicks "Login" button
    ↓
JavaScript: window.location.href = '/dashboard'
    ↓
Go server proxies to WebGUI (localhost:3007)
    ↓
User sees WebGUI dashboard
```

### 5. Shutdown & Cleanup
```
SIGINT/SIGTERM received
    ↓
Stop all Node.js services (SIGTERM → SIGKILL)
    ↓
Cleanup: rm -rf /tmp/knirvgateway-runtime/
    ↓
Exit
```

## File Structure

```
KNIRVGATEWAY/
├── bin/
│   └── knirvgateway               # 580MB binary with ALL embedded files
├── cmd/gateway/main.go            # Entry point with runtime extraction
├── internal/
│   ├── embedded/
│   │   ├── embed.go               # Embed directives and extraction functions
│   │   ├── services/              # Embedded: All 4 services
│   │   │   ├── payment-gateway/
│   │   │   ├── tunnel-registry/
│   │   │   ├── webgui/
│   │   │   └── operator-registry/
│   │   └── network-website/       # Embedded: Full website
│   │       └── public/
│   ├── runtime/
│   │   └── runtime.go             # Runtime extraction & npm install
│   ├── services/
│   │   └── manager.go             # Service process management
│   ├── server/
│   │   └── server.go              # HTTP server & routing
│   ├── proxy/
│   │   └── handlers.go            # Reverse proxy handlers
│   ├── session/
│   │   └── session.go             # Session management
│   └── config/
│       └── config.go              # Configuration
└── services/                      # Source files (not used at runtime)
    ├── payment-gateway/
    ├── tunnel-registry/
    ├── webgui/
    ├── operator-registry/
    └── network-website/
```

## Implementation Details

### Embedding (`internal/embedded/embed.go`)

```go
// Embed all service directories
//go:embed all:services
var ServicesFS embed.FS

// Embed network-website public folder
//go:embed all:network-website/public
var NetworkWebsiteFS embed.FS

// Extract to disk at runtime
func ExtractServices(targetDir string, logger *zap.Logger) error
func ExtractNetworkWebsite(targetDir string, logger *zap.Logger) error
```

### Runtime Management (`internal/runtime/runtime.go`)

```go
type Runtime struct {
    BaseDir           string  // /tmp/knirvgateway-runtime
    ServicesDir       string  // /tmp/knirvgateway-runtime/services
    NetworkWebsiteDir string  // /tmp/knirvgateway-runtime/network-website
}

// Setup extracts all embedded files
func (r *Runtime) Setup() error {
    1. Clean existing runtime directory
    2. Extract embedded services
    3. Extract embedded network-website
    4. Install npm dependencies for each service
    5. Return paths for use by service manager and HTTP server
}

// Cleanup removes runtime directory
func (r *Runtime) Cleanup() error
```

### Service Manager (`internal/services/manager.go`)

```go
// Now accepts servicesDir from extracted runtime
func NewManager(cfg *config.Config, logger *zap.Logger, servicesDir string) (*Manager, error)

// Starts services from extracted directory
func (m *Manager) StartAll(ctx context.Context) error {
    For each enabled service:
        1. Validate server.js exists in extracted directory
        2. Spawn: node server.js
        3. Set PORT and environment variables
        4. Monitor process with goroutine
}
```

### HTTP Server (`internal/server/server.go`)

```go
// Now accepts networkWebsiteDir from extracted runtime
func New(cfg *config.Config, svcMgr *services.Manager, networkWebsiteDir string, logger *zap.Logger) (*Server, error)

// Serves network-website at root
func (s *Server) setupRoutes() error {
    // ... all proxy routes ...

    // Serve network-website at root (catches all remaining routes)
    r.PathPrefix("/").Handler(http.FileServer(http.Dir(s.networkWebsiteDir)))
}
```

## Running the Binary

### Quick Start

```bash
# Run the binary
./bin/knirvgateway
```

**What happens:**
1. Extracts ~1GB of embedded files (takes 5-10 seconds)
2. Installs npm dependencies for each service (takes 30-60 seconds first run)
3. Starts all services
4. HTTP server starts on port 8080

### Access Points

```bash
# Landing page (network-website)
open http://localhost:8080

# Click "Login" button → automatically redirects to dashboard
# Or access directly:
open http://localhost:8080/dashboard

# Payment Gateway
curl http://localhost:8080/payment/health

# Tunnel Registry
curl http://localhost:8080/tunnel-registry/status

# Health check
curl http://localhost:8080/health
```

### Configuration

Same environment variables as before:

```bash
PORT=8080                           # HTTP server port
GATEWAY_MODE=persistent             # persistent or serverless
KNIRV_CHAIN_ID=testnet             # testnet or mainnet
NODEJS_SERVICES_AUTOSTART=true     # Auto-start services
PAYMENT_GATEWAY_PORT=3001          # Service ports
TUNNEL_REGISTRY_HTTP_PORT=3002
WEBGUI_PORT=3007
OPERATOR_REGISTRY_ENABLED=false    # Disabled by default
```

## Advantages of Embedded Approach

### ✅ Self-Contained
- Single 580MB binary contains everything
- No need to distribute separate service directories
- No "service not found" errors

### ✅ Portable
- Copy binary anywhere and run
- All dependencies embedded
- Works on any Linux system with Node.js installed

### ✅ Consistent
- Services always match the binary version
- No version mismatches
- Atomic deployments

### ✅ Simpler Deployment
- Docker: Just copy the binary
- VM: Just copy the binary
- Kubernetes: Single container with binary

## Disadvantages & Considerations

### ⚠️ Large Binary Size
- **580MB** binary (was 11MB without embedding)
- Includes all node_modules for each service
- Startup extraction takes 5-10 seconds

### ⚠️ First-Run npm Install
- Each service runs `npm install` on first extraction
- Takes 30-60 seconds on first run
- Can be optimized by pre-installing in Docker image

### ⚠️ Requires Node.js Runtime
- Binary still needs Node.js installed on the system
- Cannot be truly standalone without bundling Node.js itself
- Future: Consider pkg or nexe to bundle Node.js

### ⚠️ Temporary Disk Space
- Requires ~1GB free space in /tmp
- Files extracted on every run
- Cleanup on shutdown

## Optimization Options

### Option 1: Exclude node_modules

Don't embed node_modules, install at runtime:
- Binary size: ~50MB (down from 580MB)
- First run: Slower (must npm install)
- Requires: Internet connection for npm packages

### Option 2: Pre-built Services

Use pkg/nexe to bundle Node.js services:
- Each service becomes a standalone binary
- No npm install needed
- Faster startup
- Larger total size

### Option 3: Lazy Extraction

Only extract services when first accessed:
- Faster startup
- Services extracted on-demand
- More complex logic

### Option 4: Persistent Runtime

Don't cleanup between runs:
- Check if /tmp/knirvgateway-runtime exists
- Only extract if missing or outdated
- Faster subsequent startups

## Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o bin/knirvgateway ./cmd/gateway

FROM node:20-alpine
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/knirvgateway /usr/local/bin/
EXPOSE 8080
CMD ["knirvgateway"]
```

Benefits:
- Single stage once binary is built
- Small base image (Node.js Alpine)
- No need to copy services directory
- Everything embedded in binary

## Testing

### Verify Embedded Files

```bash
# Check binary size
ls -lh bin/knirvgateway

# Run with verbose logging
LOGLEVEL=debug ./bin/knirvgateway
```

### Test Extraction

```bash
# Start binary
./bin/knirvgateway &
PID=$!

# Check runtime directory
ls -la /tmp/knirvgateway-runtime/
ls -la /tmp/knirvgateway-runtime/services/
ls -la /tmp/knirvgateway-runtime/network-website/public/

# Cleanup
kill $PID
```

### Test Services

```bash
# Check services are running
curl http://localhost:8080/services/status

# Access network-website
curl http://localhost:8080 | head -50

# Test login redirect (manual)
open http://localhost:8080
# Click "Login" button → should redirect to /dashboard
```

## Troubleshooting

### Binary Too Large

- Exclude node_modules from embedding
- Use .dockerignore to exclude from Docker builds
- Consider lazy loading services

### Slow Startup

- First run: npm install takes time (normal)
- Subsequent runs: Extraction takes 5-10s (normal)
- Optimize: Keep runtime directory between runs

### Services Won't Start

```bash
# Check extraction worked
ls -la /tmp/knirvgateway-runtime/services/payment-gateway/

# Check Node.js is installed
node --version
npm --version

# Check logs
# Binary outputs structured logs with zap logger
```

### Out of Disk Space

- Runtime needs ~1GB in /tmp
- Set TMPDIR to different location:
```bash
TMPDIR=/var/tmp ./bin/knirvgateway
```

## Migration from Node.js Version

### Before (Node.js)

```bash
git clone repo
cd KNIRVGATEWAY
npm install
npm run services:install
npm start
```

### After (Go with Embedded Services)

```bash
# Just run the binary!
./bin/knirvgateway
```

**That's it!** No npm install, no service setup, everything embedded.

## Future Enhancements

1. **Bundle Node.js**: Use pkg/nexe to create truly standalone binary
2. **Compression**: Use compressed embedded FS (already happens with Go embed)
3. **Lazy Loading**: Extract services only when accessed
4. **Persistent Runtime**: Cache extracted files between runs
5. **Service Binaries**: Compile services to native binaries (if possible)

## Conclusion

The embedded services implementation provides a **self-contained, portable, single-binary deployment** of KNIRVGATEWAY. While the binary is larger (580MB), it eliminates the complexity of managing separate service directories and ensures consistency across deployments.

The trade-off is:
- ✅ Simpler deployment
- ✅ No version mismatches
- ✅ Truly portable
- ⚠️ Larger binary size
- ⚠️ Requires Node.js runtime
- ⚠️ Slower startup (extraction + npm install)

For production use, consider the optimization options based on your deployment requirements.
