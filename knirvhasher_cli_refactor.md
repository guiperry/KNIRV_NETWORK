# KNIRVHASHER CLI Refactor Plan: Headless Mode

## Overview

Add a `--headless` flag to `packages/KNIRVHASHER/cmd/cli/main.go` that starts the CLI without the Bubble Tea UI, instead launching an HTTP server over a Unix socket that exposes endpoints to control all UI-menu options programmatically.

This refactor will:
1. Add headless mode flag and HTTP server over Unix socket
2. Extract UI action logic into a shared controller
3. Enable programmatic control of all CLI features via REST API
4. Maintain backward compatibility with existing Bubble Tea UI
5. Allow KNIRVGATEWAY to proxy endpoints publicly as needed

---

## Architecture

```
┌─────────────────┐      Unix Socket       ┌──────────────────┐      HTTP Proxy      ┌─────────────────┐
│  KNIRVHASHER    │◄───────────────────────►│   KNIRVGATEWAY   │◄────────────────────►│    Public       │
│  (Headless CLI) │   /var/run/knirvhasher  │   (proxy_pass)   │                      │    Clients      │
│                 │                         │                  │                      │                 │
│  --headless     │                         │  Location /api/  │                      │                 │
│  --socket-path  │                         │  knirvhasher/    │                      │                 │
└─────────────────┘                         └──────────────────┘                      └─────────────────┘
```

**Socket Path**: `/var/run/knirvhasher.sock` (configurable via `--socket-path`)

---

## 1. Add `--headless` Flag to `main.go`

**File:** `packages/KNIRVHASHER/cmd/cli/main.go`

### Implementation (Complete)

```go
var (
	monitorLogs  = flag.Bool("monitor-logs", true, "enable server log monitoring")
	headlessMode = flag.Bool("headless", false, "run in headless mode with HTTP API server over Unix socket")
	socketPath   = flag.String("socket-path", "/var/run/knirvhasher.sock", "Unix socket path for headless API server")
	socketPerm   = flag.String("socket-perm", "0660", "Unix socket permissions (octal)")
)
```

---

## 2. Headless Server Package

**File:** `packages/KNIRVHASHER/internal/cli/headless/server.go`

### Implementation (Complete)

Unix socket-based server with all endpoints implemented.

### Updated Server Structure

```go
type Server struct {
	controller  *controller.Controller
	mux         *http.ServeMux
	socketPath  string
	socketPerm  os.FileMode
	ln          net.Listener
	shutdownCh  chan struct{}
	mu          sync.RWMutex
}
```

### Unix Socket Server Implementation

```go
func (s *Server) Start() error {
	// Remove existing socket file
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Set socket permissions
	perm, err := strconv.ParseUint(*socketPerm, 8, 32)
	if err != nil {
		perm = 0660
	}

	// Create Unix socket listener
	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket at %s: %w", s.socketPath, err)
	}

	// Set socket permissions
	if err := os.Chmod(s.socketPath, os.FileMode(perm)); err != nil {
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.ln = ln
	s.server = &http.Server{Handler: s.mux}

	log.Printf("Headless server listening on unix://%s", s.socketPath)

	go func() {
		if err := s.server.Serve(s.ln); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	return nil
}
```

---

## 3. API Endpoints

### Endpoint Implementation Status

| Endpoint | Method | Status | Notes |
|----------|--------|--------|-------|
| `/api/v1/health` | GET | ✅ Implemented | Health check |
| `/api/v1/status` | GET | ✅ Implemented | Overall status |
| `/api/v1/driver/start` | POST | ✅ Implemented | Start hasher-host |
| `/api/v1/driver/stop` | POST | ✅ Implemented | Stop hasher-host |
| `/api/v1/driver/status` | GET | ✅ Implemented | Driver status |
| `/api/v1/pipeline/run` | POST | ✅ Implemented | Run pipeline |
| `/api/v1/pipeline/status` | GET | ✅ Implemented | Pipeline status |
| `/api/v1/pipeline/logs` | GET | ✅ Implemented | SSE stream |
| `/api/v1/verify` | POST/GET | ✅ Implemented | Semantic/mathematical verification |
| `/api/v1/asic/discover` | POST | ✅ Implemented | ASIC discovery |
| `/api/v1/asic/probe` | POST | ✅ Implemented | Probe device |
| `/api/v1/asic/protocol` | POST | ✅ Implemented | Detect protocol |
| `/api/v1/asic/provision` | POST | ✅ Implemented | Provision device |
| `/api/v1/asic/troubleshoot` | POST | ✅ Implemented | Troubleshooting |
| `/api/v1/asic/configure` | GET/POST | ✅ Implemented | Get configuration |
| `/api/v1/asic/test` | POST | ✅ Implemented | Test ASIC |
| `/api/v1/rules` | GET/POST | ✅ Implemented | Rules management |
| `/api/v1/chat` | POST | ✅ Implemented | Chat with hasher-host |
| `/api/v1/shutdown` | POST | ✅ Implemented | Shutdown server |

**All endpoints implemented**

### Discrepancies from Original Spec

| Original Spec | Current Implementation | Action |
|---------------|----------------------|--------|
| `--headless-port 9090` | `--headless-port 8088` | Update docs or change default |
| `/api/v1/start-driver` | `/api/v1/driver/start` | Rename endpoint |
| `/api/v1/pipeline` | `/api/v1/pipeline/run` | Rename endpoint |
| `/api/v1/asic/rules` | `/api/v1/rules` | Flatten path |
| TCP listener | Unix Socket | **Update required** |

---

## 4. KNIRVGATEWAY Proxy Configuration

**File:** `packages/KNIRVGATEWAY/internal/server/server.go`

### Implementation (Complete)

Unix socket proxy registered at `/api/knirvhasher/` prefix.

```go
// Hasher proxy — KNIRVHASHER via Unix socket (Phase 8)
if s.config.HasherSocketPath != "" {
	hasherProxy := newSocketProxy(s.config.HasherSocketPath, "http://knirvhasher")
	r.PathPrefix("/api/knirvhasher/").Handler(http.StripPrefix("/api/knirvhasher", hasherProxy))

	s.logger.Info("Hasher proxy registered", zap.String("socket", s.config.HasherSocketPath))
} else {
	s.logger.Warn("Hasher socket path not configured — /api/knirvhasher/* will not be proxied")
}
```

### Configuration (config.go)

```go
HasherSocketPath string // /var/run/knirvhasher.sock
// Environment variable: HASHER_SOCKET_PATH
```

---

## 5. Controller Extraction

**File:** `packages/KNIRVHASHER/internal/cli/controller/controller.go`

### Current Status: ✅ Complete

All UI action logic extracted to controller:

| UI Action | Controller Method | Status |
|-----------|-------------------|--------|
| `startHasherHost()` | `StartDriver(ctx)` | ✅ |
| `runDiscovery()` | `DiscoverASIC(ctx)` | ✅ |
| `runProbe()` | `ProbeASIC(ctx)` | ✅ |
| `runProtocol()` | `DetectProtocol(ctx)` | ✅ |
| `runProvision()` | `ProvisionASIC(ctx, deviceIP)` | ✅ |
| `runTroubleshoot()` | `TroubleshootASIC(ctx)` | ✅ |
| `runConfigure()` | `ConfigureASIC(ctx)` | ✅ |
| `ManageRules()` | `ManageRules(ctx, action, args...)` | ✅ |
| `runTest()` | `TestASIC(ctx)` | ✅ |
| `runDataPipeline()` | `RunPipeline(ctx, pipelineType)` | ✅ |
| `runMathVerifier()` | `RunMathVerifier(ctx)` | ✅ |
| `handleInput()` | `SendChat(ctx, message)` | ✅ |

---

## 6. Files Summary

| File | Action | Status | Notes |
|------|--------|--------|-------|
| `cmd/cli/main.go` | Modify | ✅ Complete | Unix socket-based flags |
| `internal/cli/controller/controller.go` | Create | ✅ Complete | All logic extracted |
| `internal/cli/controller/types.go` | Create | ✅ Inline | Types in controller file |
| `internal/cli/headless/server.go` | Create | ✅ Complete | Unix socket server |
| `internal/cli/headless/types.go` | Create | ✅ Complete | Request/response types |
| `internal/cli/headless/server_test.go` | Create | ✅ Complete | Comprehensive tests |
| `internal/cli/ui/ui.go` | Modify | ✅ Complete | Uses controller |

---

## 7. Implementation Complete

All items from the specification have been implemented:

### High Priority (Complete)
- [x] **Unix Socket Server**: TCP converted to Unix socket listener
- [x] **Socket Permissions**: Configurable via `--socket-perm` flag
- [x] **Verify Endpoint**: `/api/v1/verify` supports semantic and mathematical modes

### Medium Priority (Complete)
- [x] **KNIRVGATEWAY Integration**: Socket proxy at `/api/knirvhasher/` with `HASHER_SOCKET_PATH` env
- [x] **Documentation Alignment**: Updated endpoint names and usage examples

### Low Priority (Future Work)
- [ ] **Authentication middleware**: Add `--headless-token` flag for API authentication
- [ ] **WebSocket support**: For real-time streaming in addition to SSE

---

## 8. Usage After Implementation

### Start Headless Mode

```bash
# With Unix socket (default)
./hasher-cli --headless

# With custom socket path
./hasher-cli --headless --socket-path /tmp/knirvhasher.sock

# With custom permissions
./hasher-cli --headless --socket-perm 0777
```

### Direct Socket Access

```bash
# Using curl with Unix socket
curl --unix-socket /var/run/knirvhasher.sock http://localhost/api/v1/health

# Check status
curl --unix-socket /var/run/knirvhasher.sock http://localhost/api/v1/status

# Start driver
curl --unix-socket /var/run/knirvhasher.sock -X POST http://localhost/api/v1/driver/start

# Run pipeline
curl --unix-socket /var/run/knirvhasher.sock -X POST http://localhost/api/v1/pipeline/run \
  -H "Content-Type: application/json" \
  -d '{"type":"goat"}'

# Verify (semantic mode)
curl --unix-socket /var/run/knirvhasher.sock -X POST http://localhost/api/v1/verify \
  -H "Content-Type: application/json" \
  -d '{"mode":"semantic"}'

# Verify (mathematical mode)
curl --unix-socket /var/run/knirvhasher.sock -X POST http://localhost/api/v1/verify \
  -H "Content-Type: application/json" \
  -d '{"mode":"mathematical"}'

# Chat with hasher-host
curl --unix-socket /var/run/knirvhasher.sock -X POST http://localhost/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello"}'

# ASIC discovery
curl --unix-socket /var/run/knirvhasher.sock -X POST http://localhost/api/v1/asic/discover

# Shutdown
curl --unix-socket /var/run/knirvhasher.sock -X POST http://localhost/api/v1/shutdown
```

### Via KNIRVGATEWAY

```bash
# When KNIRVGATEWAY proxies /api/knirvhasher/*
curl http://localhost:8080/api/knirvhasher/health
curl -X POST http://localhost:8080/api/knirvhasher/driver/start
curl -X POST http://localhost:8080/api/knirvhasher/pipeline/run -d '{"type":"goat"}'
```

**Environment variable for KNIRVGATEWAY**: `HASHER_SOCKET_PATH=/var/run/knirvhasher.sock`

---

## 9. Implementation Order

### Phase 1: Controller Extraction
- [x] Create `controller.go` with extracted logic
- [x] Modify `ui.go` to use controller
- [x] Verify Bubble Tea UI still works

### Phase 2: Headless Server (TCP)
- [x] Create `headless/server.go` and `headless/types.go`
- [x] Implement core endpoints
- [x] Test with curl/httpie

### Phase 3: Main Integration
- [x] Add flags to `main.go`
- [x] Implement `runHeadlessMode()`
- [x] Test headless mode

### Phase 4: Unix Socket Migration
- [x] Convert headless server from TCP to Unix socket
- [x] Update flag handling in main.go
- [x] Test socket-based connections
- [x] Document socket permissions

### Phase 5: KNIRVGATEWAY Integration
- [x] Create Unix socket transport (using existing newSocketProxy)
- [x] Add proxy routes
- [x] Configure public/internal endpoint splits
- [x] Test end-to-end proxying

### Phase 6: Verify Endpoint
- [x] Add `/api/v1/verify` endpoint
- [x] Support semantic and mathematical modes

### Phase 7: Polish
- [x] Comprehensive tests for headless server
- [x] Update documentation

---

## 10. Socket Lifecycle Management

```go
// On startup
func setupSocket(path string, perm os.FileMode) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove existing socket: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	return nil
}

// On shutdown
func cleanupSocket(path string) {
	os.RemoveAll(path)
}

// Register with systemd (optional)
func registerSocket() {
	// For socket activation
	os.Setenv("LISTEN_FDS", "1")
	os.Setenv("LISTEN_PID", strconv.Itoa(os.Getpid()))
}
```

---

## 11. Error Handling

### Socket Errors

```go
// Common socket errors and handling
switch {
case os.IsNotExist(err):
	// Socket dir doesn't exist - create it
case strings.Contains(err.Error(), "address already in use"):
	// Clean up stale socket and retry
case os.IsPermission(err):
	// Check socket permissions or run with elevated privileges
}
```

### Health Checks for Socket Availability

```go
func IsSocketReady(path string) bool {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
```

---

## Appendix: Original Spec Endpoints

For reference, the original specification defined:

| Original Endpoint | Current Endpoint | Notes |
|-------------------|------------------|-------|
| `/api/v1/start-driver` | `/api/v1/driver/start` | Prefixed with driver/ |
| `/api/v1/stop-driver` | `/api/v1/driver/stop` | Prefixed with driver/ |
| `/api/v1/driver/status` | `/api/v1/driver/status` | Unchanged |
| `/api/v1/pipeline` | `/api/v1/pipeline/run` | Added /run suffix |
| `/api/v1/verify` | ❌ | **Missing - needs implementation** |
| `/api/v1/asic/discover` | `/api/v1/asic/discover` | Unchanged |
| `/api/v1/asic/probe` | `/api/v1/asic/probe` | Unchanged |
| `/api/v1/asic/protocol` | `/api/v1/asic/protocol` | Unchanged |
| `/api/v1/asic/provision` | `/api/v1/asic/provision` | Unchanged |
| `/api/v1/asic/troubleshoot` | `/api/v1/asic/troubleshoot` | Unchanged |
| `/api/v1/asic/configure` | `/api/v1/asic/configure` | Unchanged |
| `/api/v1/asic/rules` | `/api/v1/rules` | Flattened path |
| `/api/v1/asic/test` | `/api/v1/asic/test` | Unchanged |
| `/api/v1/chat` | `/api/v1/chat` | Unchanged |
| `/api/v1/status` | `/api/v1/status` | Unchanged |
| `/api/v1/shutdown` | `/api/v1/shutdown` | Unchanged |