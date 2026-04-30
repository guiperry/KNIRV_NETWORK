# KNIRVHASHER CLI Refactor Plan: Headless Mode

## Overview

Add a `--headless` flag to `packages/KNIRVHASHER/cmd/cli/main.go` that starts the CLI without the Bubble Tea UI, instead launching an HTTP server that exposes endpoints to control all UI-menu options programmatically.

This refactor will:
1. Add headless mode flag and HTTP server
2. Extract UI action logic into a shared controller
3. Enable programmatic control of all CLI features via REST API
4. Maintain backward compatibility with existing Bubble Tea UI

---

## 1. Add `--headless` Flag to `main.go`

**File:** `packages/KNIRVHASHER/cmd/cli/main.go`

Add new flag at the existing flag section (around line 37-39):

```go
// CLI configuration flags
var (
	monitorLogs  = flag.Bool("monitor-logs", true, "enable server log monitoring")
	headlessMode = flag.Bool("headless", false, "run without UI; expose HTTP API for headless control")
	headlessPort = flag.Int("headless-port", 9090, "HTTP port for headless API server")
)
```

---

## 2. Create Headless Server Package

**New file:** `packages/KNIRVHASHER/internal/cli/headless/server.go`

Create an HTTP server that wraps the same functionality currently triggered by the Bubble Tea UI. The server will expose endpoints corresponding to each UI menu option.

### Endpoints to implement:

| Endpoint | Method | Description | UI Equivalent |
|----------|--------|-------------|----------------|
| `/api/v1/start-driver` | POST | Start hasher-host subprocess | Menu: "4. Start Driver" |
| `/api/v1/stop-driver` | POST | Stop hasher-host subprocess | Signal handler |
| `/api/v1/driver/status` | GET | Get hasher-host status | Header status display |
| `/api/v1/pipeline` | POST | Run data pipeline (accepts `type`: goat, arxiv, demo) | Menu: "1. Data Pipeline" |
| `/api/v1/verify` | POST | Run data verification (accepts `mode`: semantic, mathematical) | Menu: "2. Data Verification Mode" |
| `/api/v1/asic/discover` | POST | Run ASIC discovery | ASIC Menu: "1. Discovery" |
| `/api/v1/asic/probe` | POST | Probe ASIC device | ASIC Menu: "2. Probe" |
| `/api/v1/asic/protocol` | POST | Detect ASIC protocol | ASIC Menu: "3. Protocol" |
| `/api/v1/asic/provision` | POST | Provision ASIC device | ASIC Menu: "4. Provision" |
| `/api/v1/asic/troubleshoot` | POST | Run ASIC troubleshooting | ASIC Menu: "5. Troubleshoot" |
| `/api/v1/asic/configure` | POST | Configure ASIC | ASIC Menu: "6. Configure" |
| `/api/v1/asic/rules` | GET/POST/DELETE | Manage rules (GET list, POST add, DELETE by ID) | ASIC Menu: "7. Rules" |
| `/api/v1/asic/test` | POST | Test ASIC communication | ASIC Menu: "8. Test" |
| `/api/v1/chat` | POST | Send chat message to hasher-host | Chat View |
| `/api/v1/status` | GET | Get overall CLI status | Header info |
| `/api/v1/shutdown` | POST | Shutdown headless server | Menu: "0. Quit" |

---

## 3. Refactor UI Actions into Shared Controller

**Files to modify:**
- `packages/KNIRVHASHER/internal/cli/ui/ui.go`

Extract the action logic from the Bubble Tea `Update()` method into a shared controller/service that can be called from both:
- The Bubble Tea UI (existing behavior)
- The headless HTTP server (new behavior)

**New file:** `packages/KNIRVHASHER/internal/cli/controller/controller.go`

### Controller Interface

```go
package controller

import (
	"context"
	"sync"
	
	"knirvhasher/internal/analyzer"
	"knirvhasher/internal/client"
)

type Controller struct {
	// State
	model          *ui.Model // Reuse existing model or simplified state
	deployer       *analyzer.Deployer
	apiClient      *client.APIClient
	
	// Operation tracking
	activeOps      map[string]OperationStatus
	opsMu          sync.RWMutex
	
	// Channels for async operations
	pipelineLogChan chan PipelineLogMsg
	logChan        chan string
}

type OperationStatus struct {
	ID        string
	Type      string
	Status    string // "running", "complete", "error"
	Progress  float64
	Message   string
	StartTime time.Time
}

func NewController() *Controller {
	config := analyzer.DefaultDeployerConfig()
	deployer, _ := analyzer.NewDeployer(config)
	
	return &Controller{
		deployer:       deployer,
		activeOps:      make(map[string]OperationStatus),
		pipelineLogChan: make(chan PipelineLogMsg, 100),
		logChan:        make(chan string, 100),
	}
}
```

### Methods to Extract from UI

| UI Action | Controller Method | Description |
|-----------|-------------------|-------------|
| `startHasherHost()` | `StartDriver(ctx) error` | Start hasher-host subprocess |
| `runDiscovery()` | `DiscoverASIC(ctx) (*DiscoveryResult, error)` | Run ASIC discovery |
| `runProbe()` | `ProbeASIC(ctx) error` | Probe ASIC device |
| `runProtocol()` | `DetectProtocol(ctx) error` | Detect ASIC protocol |
| `runProvision()` | `ProvisionASIC(ctx, deviceIP) error` | Provision ASIC device |
| `runTroubleshoot()` | `TroubleshootASIC(ctx) error` | Run ASIC troubleshooting |
| `runConfigure()` | `ConfigureASIC(ctx) error` | Configure ASIC settings |
| `runRulesManager()` | `ManageRules(ctx, action string) error` | Manage logical rules |
| `runTest()` | `TestASIC(ctx) error` | Test ASIC communication |
| `runDataPipeline()` | `RunPipeline(ctx, pipelineType string) error` | Run data pipeline |
| `runMathVerifier()` | `RunMathVerifier(ctx) error` | Run mathematical verification |
| `handleInput()` | `SendChat(ctx, message string) (string, error)` | Send chat message |

### Progress Tracking

For long-running operations (pipelines, verification), the controller should provide progress updates:

```go
type PipelineLogMsg struct {
	StageIndex int
	Log        string
	Complete   bool
	Error      bool
}

// GetProgress returns channel for streaming operation progress
func (c *Controller) GetProgressChan() <-chan PipelineLogMsg {
	return c.pipelineLogChan
}
```

---

## 4. Modify `main.go` for Headless Mode

**File:** `packages/KNIRVHASHER/cmd/cli/main.go`

Update `main()` to branch based on the `--headless` flag:

```go
func main() {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\n", r)
		}
	}()

	flag.Parse()

	// Branch for headless mode
	if *headlessMode {
		runHeadlessMode()
		return
	}

	// Existing Bubble Tea UI code...
	initEmbeddedBinaries()
	
	// Set up signal handler for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Create UI model
	model := ui.NewModel()
	
	// Check if hasher-host is already running and update model state
	model.CheckExistingHasherHost()
	
	// Start the Bubble Tea UI with alternate screen and mouse support
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseAllMotion(), tea.WithInputTTY())
	
	// Handle server shutdown with ASIC cleanup
	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal.")
		cleanupASICDevice(model.Deployer)
		if model.ServerCmd != nil && model.ServerCmd.Process != nil {
			shutdownHasherHost(model.ServerCmd, true, 8080)
		}
		pipelineState.Mu.Lock()
		shutdownPipelineProcess(pipelineState.Cmd)
		pipelineState.Running = false
		pipelineState.Mu.Unlock()
		os.Exit(0)
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		cleanupASICDevice(model.Deployer)
		pipelineState.Mu.Lock()
		shutdownPipelineProcess(pipelineState.Cmd)
		pipelineState.Running = false
		pipelineState.Mu.Unlock()
		os.Exit(1)
	}

	// Ensure cleanup when exiting normally
	cleanupASICDevice(model.Deployer)
	pipelineState.Mu.Lock()
	shutdownPipelineProcess(pipelineState.Cmd)
	pipelineState.Running = false
	pipelineState.Mu.Unlock()
}

func runHeadlessMode() {
	// Initialize embedded binaries
	initEmbeddedBinaries()

	// Create controller
	ctrl := controller.NewController()

	// Start HTTP server
	srv := headless.NewServer(ctrl, *headlessPort)
	
	log.Printf("KNIRVHASHER CLI running in headless mode on :%d", *headlessPort)
	
	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Println("Shutting down headless server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		ctrl.Cleanup()
		log.Println("Cleanup complete")
		os.Exit(0)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Headless server error: %v", err)
	}
}
```

---

## 5. Implement Headless Server

**New file:** `packages/KNIRVHASHER/internal/cli/headless/server.go`

```go
package headless

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	
	"knirvhasher/internal/cli/controller"
)

type Server struct {
	ctrl   *controller.Controller
	port   int
	server *http.Server
}

func NewServer(ctrl *controller.Controller, port int) *Server {
	return &Server{ctrl: ctrl, port: port}
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	
	// Driver endpoints
	mux.HandleFunc("/api/v1/start-driver", s.handleStartDriver)
	mux.HandleFunc("/api/v1/stop-driver", s.handleStopDriver)
	mux.HandleFunc("/api/v1/driver/status", s.handleDriverStatus)
	
	// Pipeline endpoints
	mux.HandleFunc("/api/v1/pipeline", s.handlePipeline)
	mux.HandleFunc("/api/v1/verify", s.handleVerify)
	
	// ASIC endpoints
	mux.HandleFunc("/api/v1/asic/discover", s.handleDiscover)
	mux.HandleFunc("/api/v1/asic/probe", s.handleProbe)
	mux.HandleFunc("/api/v1/asic/protocol", s.handleProtocol)
	mux.HandleFunc("/api/v1/asic/provision", s.handleProvision)
	mux.HandleFunc("/api/v1/asic/troubleshoot", s.handleTroubleshoot)
	mux.HandleFunc("/api/v1/asic/configure", s.handleConfigure)
	mux.HandleFunc("/api/v1/asic/rules", s.handleRules)
	mux.HandleFunc("/api/v1/asic/test", s.handleTest)
	
	// Chat endpoint
	mux.HandleFunc("/api/v1/chat", s.handleChat)
	
	// Status and shutdown
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/shutdown", s.handleShutdown)
	
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}
	
	log.Printf("Headless API server listening on :%d", s.port)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Handler implementations

func (s *Server) handleStartDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	err := s.ctrl.StartDriver(r.Context())
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	jsonResponse(w, http.StatusOK, gin.H{"message": "Driver started successfully"})
}

func (s *Server) handlePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Type string `json:"type"` // goat, arxiv, demo
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Start pipeline asynchronously
	go func() {
		if err := s.ctrl.RunPipeline(r.Context(), req.Type); err != nil {
			log.Printf("Pipeline error: %v", err)
		}
	}()
	
	jsonResponse(w, http.StatusAccepted, gin.H{"message": "Pipeline started", "type": req.Type})
}

// ... implement other handlers similarly

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
```

**New file:** `packages/KNIRVHASHER/internal/cli/headless/types.go`

```go
package headless

// Request/response types for headless API

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

type StatusResponse struct {
	DriverRunning bool    `json:"driver_running"`
	ServerReady   bool    `json:"server_ready"`
	DeviceIP      string  `json:"device_ip,omitempty"`
	DeviceType    string  `json:"device_type,omitempty"`
	Uptime        string  `json:"uptime"`
}

type PipelineRequest struct {
	Type string `json:"type"` // goat, arxiv, demo
}

type VerifyRequest struct {
	Mode string `json:"mode"` // semantic, mathematical
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response  string  `json:"response"`
	Timestamp string  `json:"timestamp"`
}
```

---

## 6. Update Controller to Support Both Modes

**File:** `packages/KNIRVHASHER/internal/cli/ui/ui.go`

### Key Changes:

1. **Import the controller:**
```go
import (
	"knirvhasher/internal/cli/controller"
)
```

2. **Add controller to Model:**
```go
type Model struct {
	// ... existing fields ...
	Controller *controller.Controller
}
```

3. **Update `NewModel()` to initialize controller:**
```go
func NewModel() Model {
	// ... existing initialization ...
	
	model := Model{
		// ... existing fields ...
		Controller: controller.NewController(),
	}
	
	return model
}
```

4. **Refactor `Update()` to use controller methods:**

Instead of inline logic like:
```go
case "4. Start Driver":
    m.ServerStarting = true
    m.ShowingInitLogs = true
    cmds = append(cmds, m.startHasherHost())
```

Use controller:
```go
case "4. Start Driver":
    m.ServerStarting = true
    m.ShowingInitLogs = true
    cmds = append(cmds, func() tea.Msg {
        err := m.Controller.StartDriver(context.Background())
        return DriverStartMsg{Err: err}
    })
```

---

## 7. Signal Handling & Cleanup

Both headless and UI modes need proper cleanup. Centralize this in the controller:

**File:** `packages/KNIRVHASHER/internal/cli/controller/controller.go`

```go
func (c *Controller) Cleanup() {
	// Stop hasher-host if running
	if c.model != nil && c.model.ServerCmd != nil {
		shutdownHasherHost(c.model.ServerCmd, true, 8080)
	}
	
	// Cleanup ASIC device
	if c.deployer != nil {
		c.deployer.Cleanup()
	}
	
	// Stop pipeline if running
	// ... shutdown pipeline process ...
	
	log.Println("Controller cleanup complete")
}
```

---

## Files to Create/Modify Summary

| File | Action | Description |
|------|--------|-------------|
| `packages/KNIRVHASHER/cmd/cli/main.go` | Modify | Add `--headless` and `--headless-port` flags; branch to `runHeadlessMode()` |
| `packages/KNIRVHASHER/internal/cli/controller/controller.go` | Create | Shared controller wrapping all UI actions |
| `packages/KNIRVHASHER/internal/cli/controller/types.go` | Create | Types for controller requests/responses |
| `packages/KNIRVHASHER/internal/cli/headless/server.go` | Create | HTTP server for headless mode |
| `packages/KNIRVHASHER/internal/cli/headless/types.go` | Create | Request/response types for headless API |
| `packages/KNIRVHASHER/internal/cli/ui/ui.go` | Modify | Refactor action logic to use controller |

---

## Example Usage After Implementation

### Current UI Mode (Unchanged)
```bash
./hasher-cli
```

### New Headless Mode
```bash
# Start in headless mode
./hasher-cli --headless --headless-port 9090

# Check status
curl http://localhost:9090/api/v1/status

# Start hasher-host driver
curl -X POST http://localhost:9090/api/v1/start-driver

# Run data pipeline (GOAT dataset)
curl -X POST http://localhost:9090/api/v1/pipeline -H "Content-Type: application/json" -d '{"type":"goat"}'

# Run data verification (semantic mode)
curl -X POST http://localhost:9090/api/v1/verify -H "Content-Type: application/json" -d '{"mode":"semantic"}'

# Chat with hasher-host
curl -X POST http://localhost:9090/api/v1/chat -H "Content-Type: application/json" -d '{"message":"Hello"}'

# ASIC discovery
curl -X POST http://localhost:9090/api/v1/asic/discover

# Check driver status
curl http://localhost:9090/api/v1/driver/status

# Shutdown
curl -X POST http://localhost:9090/api/v1/shutdown
```

---

## Considerations

### 1. Authentication
The headless API has no auth by default. Consider adding:
```go
headlessToken = flag.String("headless-token", "", "Bearer token for headless API authentication")
```

Then wrap handlers with auth middleware:
```go
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if *headlessToken != "" {
			token := r.Header.Get("Authorization")
			if token != "Bearer "+*headlessToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}
```

### 2. Logging
Headless mode should log to a file (same as UI mode's `GetLogger()`):
```go
// In runHeadlessMode()
logger := ui.GetLogger()
logger.Write("Headless mode started\n")
```

### 3. Streaming Output
For long-running operations (pipelines), consider implementing:

**Option A: Polling Endpoint**
```go
// GET /api/v1/operations/{id}
func (s *Server) handleGetOperation(w http.ResponseWriter, r *http.Request) {
	opID := chi.URLParam(r, "id")
	status := s.ctrl.GetOperationStatus(opID)
	jsonResponse(w, http.StatusOK, status)
}
```

**Option B: Server-Sent Events (SSE)**
```go
// GET /api/v1/stream/pipeline
func (s *Server) handleStreamPipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	ch := s.ctrl.GetProgressChan()
	for msg := range ch {
		data, _ := json.Marshal(msg)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}
```

### 4. Port Conflicts
The headless server (`:9090`) and hasher-host (`:8080`) use different ports. Document this clearly in the help text.

### 5. Graceful Shutdown
Ensure the headless server properly propagates shutdown to hasher-host and cleans up ASIC devices:

```go
func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, gin.H{"message": "Shutdown initiated"})
	
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(syscall.SIGTERM)
	}()
}
```

### 6. Testing
Add tests for the headless API:

**New file:** `packages/KNIRVHASHER/internal/cli/headless/server_test.go`

```go
func TestHeadlessServer(t *testing.T) {
	ctrl := controller.NewController()
	srv := headless.NewServer(ctrl, 0) // random port
	
	// Test status endpoint
	resp, err := http.Get("http://localhost:0/api/v1/status")
	// ... assertions ...
}
```

---

## Implementation Order

1. **Phase 1: Controller Extraction**
   - Create `controller.go` with extracted logic
   - Modify `ui.go` to use controller
   - Verify Bubble Tea UI still works

2. **Phase 2: Headless Server**
   - Create `headless/server.go` and `headless/types.go`
   - Implement core endpoints
   - Test with curl/httpie

3. **Phase 3: Main Integration**
   - Add flags to `main.go`
   - Implement `runHeadlessMode()`
   - Test headless mode

4. **Phase 4: Polish**
   - Add authentication (optional)
   - Implement streaming output (SSE)
   - Add comprehensive tests
   - Update documentation/README
