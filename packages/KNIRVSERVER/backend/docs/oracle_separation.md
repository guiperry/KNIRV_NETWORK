# Oracle Separation Implementation Plan

## Overview

This document describes how to externalize the Oracle from KNIRVSERVER into its own standalone binary (`packages/KNIRVORACLE`), embedded into KNIRVSERVER, and managed via the Manager pattern consistent with other external programs (knirvchain, knirvgateway, knirvgraph, knirvhasher).

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         KNIRVSERVER                             │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │              internal/services/knirvoracle/               │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐  │  │
│  │  │ manager.go  │  │ client.go     │  │ config.go      │  │  │
│  │  │ (spawn/manage)│ (Unix Socket) │  │                │  │  │
│  │  └─────────────┘  └──────────────┘  └────────────────┘  │
│  └─────────────────────────────────────────────────────────┘  │
│                              │                                  │
│                              ▼                                  │
│                        bin/knirvoracle                          │
│                     (EMBEDDED BINARY)                          │
└─────────────────────────────────────────────────────────────────┘
                          │
                    Unix Socket
                          │
                          ▼
              ┌───────────────────────┐
              │   /var/run/knirv/     │
              │   oracle.sock         │
              │   (HTTP over socket)  │
              └───────────────────────┘
```

## Key Design Decisions

1. **Oracle moves to packages/KNIRVORACLE**: New standalone Go package with its own binary
2. **Embedded in KNIRVSERVER**: Binary embedded at build time via `//go:embed` in `bin/knirvoracle`
3. **Manager pattern**: Follows same architecture as knirvchain manager
4. **Unix sockets primary**: Uses Unix domain sockets for communication (not TCP)
5. **Activated by root.key**: Oracle binary only starts when a valid encrypted `root.key` is present

## New Package: packages/KNIRVORACLE

```
packages/KNIRVORACLE/
├── cmd/
│   └── oracle/
│       └── main.go           # New entry point - starts Oracle HTTP server
├── internal/
│   └── oracle/               # Migrated from backend/internal/oracle
│       ├── oracle.go
│       ├── config.go
│       ├── routes/routes.go
│       ├── token/
│       ├── governance/
│       ├── economics/
│       ├── consensus/
│       ├── crosschain/
│       ├── ibc/
│       ├── p2p/
│       └── types/
├── config/
│   ├── default.yaml
│   └── production.yaml.example
├── go.mod
├── go.sum
├── Dockerfile
└── Makefile
```

## Directory Structure (KNIRVSERVER)

```
packages/KNIRVSERVER/
├── backend/
│   ├── cmd/
│   │   └── backend_server/
│   │       └── main.go         # Updated to embed knirvoracle
│   ├── internal/
│   │   ├── services/
│   │   │   └── knirvoracle/
│   │   │       ├── manager.go   # Spawns and manages knirvoracle binary
│   │   │       ├── client.go    # Unix socket HTTP client
│   │   │       ├── config.go   # Manager configuration
│   │   │       └── types.go    # Health/status types
│   │   └── ...
│   └── ...
├── bin/
│   └── knirvoracle             # EMBEDDED from packages/KNIRVORACLE
├── Makefile                    # Updated to build and embed knirvoracle
└── ...
```

## Implementation

### Phase 1: Create packages/KNIRVORACLE

#### 1.1 Initial Structure

```bash
mkdir -p packages/KNIRVORACLE/cmd/oracle
mkdir -p packages/KNIRVORACLE/internal/oracle
mkdir -p packages/KNIRVORACLE/config
```

#### 1.2 Migrate Code

Move contents from `packages/KNIRVSERVER/backend/internal/oracle/` to `packages/KNIRVORACLE/internal/oracle/`

Files to move:
- `oracle.go`
- `config.go`
- `routes/routes.go`
- `token/*.go`
- `governance/*.go`
- `economics/*.go`
- `consensus/*.go`
- `crosschain/*.go`
- `ibc/*.go`
- `p2p/*.go`
- `crypto/*.go`
- `types/*.go`

#### 1.3 Create Entry Point

```go
// packages/KNIRVORACLE/cmd/oracle/main.go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"
    
    "github.com/knirvcorp/knirvoracle/internal/oracle"
    "github.com/knirvcorp/knirvoracle/internal/oracle/routes"
    "go.uber.org/zap"
)

var (
    socketPath = flag.String("socket", "/var/run/knirv/oracle.sock", "Unix socket path")
    dataDir    = flag.String("data-dir", "", "Oracle data directory")
    configFile = flag.String("config", "", "Config file path")
)

func main() {
    flag.Parse()
    
    // Load configuration
    cfg := oracle.LoadConfigFromEnv()
    if *configFile != "" {
        // Override from file if specified
    }
    if *dataDir != "" {
        cfg.DataDir = *dataDir
    }
    
    // Setup logger
    logger, err := zap.NewProduction()
    if err != nil {
        log.Fatalf("Failed to create logger: %v", err)
    }
    
    // Validate config
    if err := oracle.ValidateConfig(cfg); err != nil {
        log.Fatalf("Invalid config: %v", err)
    }
    
    // Create Oracle instance
    oracleInst, err := oracle.NewOracle(cfg, logger)
    if err != nil {
        log.Fatalf("Failed to create Oracle: %v", err)
    }
    
    // Ensure socket directory exists
    socketDir := filepath.Dir(*socketPath)
    if err := os.MkdirAll(socketDir, 0755); err != nil {
        log.Fatalf("Failed to create socket directory: %v", err)
    }
    
    // Remove existing socket
    os.Remove(*socketPath)
    
    // Create Unix socket listener
    listener, err := net.Listen("unix", *socketPath)
    if err != nil {
        log.Fatalf("Failed to create socket listener: %v", err)
    }
    os.Chmod(*socketPath, 0600)
    
    // Create HTTP handlers
    mux := http.NewServeMux()
    oracleRoutes := routes.NewOracleRoutes(oracleInst, logger)
    oracleRoutes.RegisterRoutes(mux)
    mux.HandleFunc("/health", handleHealth(oracleInst))
    
    server := &http.Server{Handler: mux}
    
    // Start Oracle
    if err := oracleInst.Start(); err != nil {
        log.Fatalf("Failed to start Oracle: %v", err)
    }
    
    logger.Info("Oracle started", zap.String("socket", *socketPath))
    
    // Serve in background
    go server.Serve(listener)
    
    // Wait for shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    
    logger.Info("Shutting down Oracle...")
    oracleInst.Stop()
    server.Close()
    listener.Close()
}

func handleHealth(o *oracle.Oracle) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"healthy","oracle":"active"}`))
    }
}
```

### Phase 2: KNIRVSERVER Integration

#### 2.1 Manager Implementation

```go
// internal/services/knirvoracle/manager.go
package knirvoracle

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "sync"
    "syscall"
    "time"
    
    "go.uber.org/zap"
)

func killStaleProcess(binaryPath string) {
    if binaryPath == "" { return }
    binaryName := filepath.Base(binaryPath)
    if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
        time.Sleep(1 * time.Second)
    }
    exec.Command("pkill", "-KILL", "-x", binaryName).Run()
}

func waitForSocketFree(socketPath string, deadline time.Duration) bool {
    end := time.Now().Add(deadline)
    for time.Now().Before(end) {
        if _, err := net.Dial("unix", socketPath); err != nil {
            return true
        }
        time.Sleep(200 * time.Millisecond)
    }
    return false
}

type Manager struct {
    binaryPath string
    config     *ManagerConfig
    cmd        *exec.Cmd
    logger     *zap.Logger
    client     *http.Client
    mu         sync.RWMutex
    running    bool
    baseURL    string
}

type ManagerConfig struct {
    BinaryPath   string
    SocketPath   string
    DataPath     string
    RootKeyPath  string
    StartTimeout time.Duration
    StopTimeout  time.Duration
    Stdout       io.Writer
    Stderr       io.Writer
}

type HealthStatus struct {
    Status    string    `json:"status"`
    Running   bool      `json:"running"`
    Socket    string    `json:"socket"`
    Timestamp time.Time `json:"timestamp"`
}

func DefaultManagerConfig() *ManagerConfig {
    return &ManagerConfig{
        SocketPath:   "/var/run/knirv/oracle.sock",
        DataPath:     "~/.knirvserver/oracle",
        StartTimeout: 30 * time.Second,
        StopTimeout:  10 * time.Second,
    }
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
    if cfg == nil {
        cfg = DefaultManagerConfig()
    }
    
    defaults := DefaultManagerConfig()
    if cfg.SocketPath == "" {
        cfg.SocketPath = defaults.SocketPath
    }
    if cfg.StartTimeout == 0 {
        cfg.StartTimeout = defaults.StartTimeout
    }
    if cfg.StopTimeout == 0 {
        cfg.StopTimeout = defaults.StopTimeout
    }
    
    client := &http.Client{
        Timeout: 10 * time.Second,
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                return (&net.Dialer{}).DialContext(ctx, "unix", cfg.SocketPath)
            },
        },
    }
    
    m := &Manager{
        binaryPath: cfg.BinaryPath,
        config:     cfg,
        logger:     logger,
        baseURL:    "http://unix",
        client:     client,
    }
    
    return m
}

func (m *Manager) Start(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if m.running {
        m.logger.Info("KNIRVORACLE already running")
        return nil
    }
    
    // Check root.key activation
    if !m.checkRootKeyActivation() {
        m.logger.Info("No root.key found - Oracle will not activate")
        return nil
    }
    
    // Resolve binary path
    resolved, err := m.resolveBinaryPath()
    if err != nil {
        return fmt.Errorf("KNIRVORACLE binary not found: %w", err)
    }
    m.binaryPath = resolved
    
    // Check if already running
    if m.client != nil {
        if resp, err := m.client.Get(m.baseURL + "/health"); err == nil {
            resp.Body.Close()
            if resp.StatusCode == http.StatusOK {
                m.logger.Info("Adopting existing healthy KNIRVORACLE process")
                m.running = true
                return nil
            }
        }
    }
    
    // Kill stale processes
    m.logger.Info("Killing any stale KNIRVORACLE processes", zap.String("binary", m.binaryPath))
    killStaleProcess(m.binaryPath)
    
    if waitForSocketFree(m.config.SocketPath, 5*time.Second) {
        m.logger.Debug("Socket freed after stale process kill")
    }
    
    m.logger.Info("Starting KNIRVORACLE",
        zap.String("binary", m.binaryPath),
        zap.String("socket", m.config.SocketPath),
        zap.String("data_dir", m.config.DataPath))
    
    // Build environment
    env := os.Environ()
    env = append(env,
        fmt.Sprintf("ORACLE_SOCKET_PATH=%s", m.config.SocketPath),
        fmt.Sprintf("ORACLE_DATA_DIR=%s", m.config.DataPath),
        fmt.Sprintf("ORACLE_ROOT_KEY_PATH=%s", m.config.RootKeyPath),
    )
    
    // Add oracle-specific env vars
    for k, v := range m.config.EnvOverrides {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    
    m.cmd = exec.Command(m.binaryPath)
    m.cmd.Env = env
    m.cmd.Stdout = m.config.Stdout
    m.cmd.Stderr = m.config.Stderr
    m.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    
    if err := m.cmd.Start(); err != nil {
        return fmt.Errorf("failed to start KNIRVORACLE: %w", err)
    }
    
    m.logger.Info("KNIRVORACLE subprocess started", zap.Int("pid", m.cmd.Process.Pid))
    m.running = true
    
    // Wait for health in background
    go func(parentCtx context.Context) {
        if err := m.waitForHealth(parentCtx); err != nil {
            m.logger.Warn("KNIRVORACLE health check did not pass", zap.Error(err))
            return
        }
        m.logger.Info("KNIRVORACLE health check passed")
    }(ctx)
    
    return nil
}

func (m *Manager) checkRootKeyActivation() bool {
    rootKeyPath := m.config.RootKeyPath
    if rootKeyPath == "" {
        // Default locations
        candidates := []string{
            filepath.Join(os.Getenv("HOME"), ".config", "knirv-server", "root.key"),
            filepath.Join(filepath.Dir(os.Args[0]), "bin", "root.key"),
            filepath.Join(filepath.Dir(os.Args[0]), "..", "bin", "root.key"),
        }
        for _, p := range candidates {
            if _, err := os.Stat(p); err == nil {
                rootKeyPath = p
                break
            }
        }
    }
    
    if rootKeyPath == "" {
        return false
    }
    
    if _, err := os.Stat(rootKeyPath); os.IsNotExist(err) {
        return false
    }
    
    m.logger.Info("Found root.key - Oracle will activate", zap.String("path", rootKeyPath))
    return true
}

func (m *Manager) resolveBinaryPath() (string, error) {
    var candidates []string
    
    // Check env override
    if envPath := os.Getenv("KNIRV_ORACLE_BINARY_PATH"); envPath != "" {
        candidates = append(candidates, envPath)
    }
    
    // Embedded in KNIRVSERVER
    if exe, err := os.Executable(); err == nil {
        exeDir := filepath.Dir(exe)
        candidates = append(candidates,
            filepath.Join(exeDir, "bin", "knirvoracle"),
            filepath.Join(exeDir, "..", "bin", "knirvoracle"),
        )
    }
    
    // Default to configured path
    candidates = append(candidates, m.binaryPath)
    
    for _, p := range candidates {
        if _, err := os.Stat(p); err == nil {
            return p, nil
        }
    }
    
    return "", fmt.Errorf("knirvoracle binary not found")
}

func (m *Manager) waitForHealth(ctx context.Context) error {
    deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
    defer cancel()
    
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-deadline.Done():
            return fmt.Errorf("timeout waiting for KNIRVORACLE health")
        case <-ticker.C:
            if err := m.client.Get(m.baseURL + "/health"); err == nil {
                return nil
            }
        }
    }
}

func (m *Manager) Stop(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if !m.running || m.cmd == nil || m.cmd.Process == nil {
        m.running = false
        return nil
    }
    
    m.logger.Info("Stopping KNIRVORACLE")
    
    if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
        m.logger.Warn("Failed to signal KNIRVORACLE", zap.Error(err))
    }
    
    done := make(chan error, 1)
    go func() {
        _, err := m.cmd.Process.Wait()
        done <- err
    }()
    
    select {
    case <-done:
    case <-time.After(m.config.StopTimeout):
        m.cmd.Process.Kill()
    }
    
    m.running = false
    m.logger.Info("KNIRVORACLE stopped")
    return nil
}

func (m *Manager) IsRunning() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.running
}

func (m *Manager) GetClient() *Client {
    return &Client{
        baseURL: m.baseURL,
        client:  m.client,
    }
}
```

#### 2.2 Client Implementation

```go
// internal/services/knirvoracle/client.go
package knirvoracle

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient(socketPath string) *Client {
    if socketPath == "" {
        socketPath = "/var/run/knirv/oracle.sock"
    }
    
    return &Client{
        baseURL: "http://unix",
        client: &http.Client{
            Timeout: 10 * time.Second,
            Transport: &http.Transport{
                DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
                    return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
                },
            },
        },
    }
}

func (c *Client) GetBalance(address string) (int64, error) {
    resp, err := c.client.Get(fmt.Sprintf("%s/oracle/v3/token/balance/%s", c.baseURL, address))
    if err != nil {
        return 0, fmt.Errorf("failed to get balance: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return 0, fmt.Errorf("oracle returned status %d", resp.StatusCode)
    }
    
    var result struct {
        Balance int64 `json:"balance"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return result.Balance, nil
}

func (c *Client) SubmitRollup(batchID, data string) (string, error) {
    // Implementation
    return "", nil
}

func (c *Client) Health() error {
    resp, err := c.client.Get(c.baseURL + "/health")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("oracle unhealthy: status %d", resp.StatusCode)
    }
    return nil
}
```

#### 2.3 Integration in backend_server/main.go

```go
// backend/cmd/backend_server/main.go

// Remove oracle import - no longer embedded
// import "backend_server/internal/oracle"

// Add knirvoracle manager import
import "backend_server/internal/services/knirvoracle"

var oracleManager *knirvoracle.Manager

// In server struct, replace:
// oracleService *oracle.Oracle
// with:
// oracleManager *knirvoracle.Manager

// In initialization:
func initOracleManager(logger *zap.Logger) *knirvoracle.Manager {
    // Check if root.key exists - if so, create manager
    rootKeyPath := resolveRootKeyPath() // existing function
    
    cfg := &knirvoracle.ManagerConfig{
        BinaryPath:   "knirvoracle",
        SocketPath:   "/var/run/knirv/oracle.sock",
        DataPath:     filepath.Join(appDataDir, "oracle"),
        RootKeyPath: rootKeyPath,
        StartTimeout: 30 * time.Second,
        StopTimeout:  10 * time.Second,
    }
    
    return knirvoracle.NewManager(cfg, logger)
}

// In main():
oracleManager := initOracleManager(logger)
if err := oracleManager.Start(ctx); err != nil {
    logger.Error("Failed to start Oracle manager", zap.Error(err))
    // Continue without oracle - not fatal
}

// Register routes
if oracleManager != nil && oracleManager.IsRunning() {
    oracleMux := http.NewServeMux()
    // Route through to socket
    oracleMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Proxy to unix socket
        client := oracleManager.GetClient()
        // Use http.ReverseProxy to unix socket
    })
    s.router.PathPrefix("/oracle/").Handler(oracleMux)
}

// In balance resolution:
func (s *Server) getOracleBalance(address string) (int64, error) {
    if s.oracleManager == nil || !s.oracleManager.IsRunning() {
        return 0, fmt.Errorf("oracle not available")
    }
    return s.oracleManager.GetClient().GetBalance(address)
}

// In shutdown:
if oracleManager != nil {
    oracleManager.Stop(ctx)
}
```

### Phase 3: Build & Embed

#### 3.1 Makefile Updates

```makefile
# packages/KNIRVSERVER/Makefile

ORACLE_DIR := $(WORKSPACE)/packages/KNIRVORACLE
ORACLE_BINARY := $(BACKEND_DIR)/bin/knirvoracle

.PHONY: oracle-build oracle-embed

oracle-build: deps-go ## Build KNIRVORACLE binary
	@echo "$(BLUE)Building KNIRVORACLE...$(NC)"
	cd $(ORACLE_DIR) && CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o knirvoracle ./cmd/oracle
	@echo "$(GREEN)KNIRVORACLE build completed$(NC)"

oracle-embed: oracle-build ## Copy KNIRVORACLE to bin for embedding
	@echo "$(BLUE)Embedding KNIRVORACLE...$(NC)"
	mkdir -p $(BACKEND_DIR)/bin
	cp $(ORACLE_DIR)/knirvoracle $(ORACLE_BINARY)
	@echo "$(GREEN)KNIRVORACLE embedded$(NC)"

build-full: ... oracle-embed ...
```

#### 3.2 Embed Directive (TOP LEVEL)

The embed directive must be at `packages/KNIRVSERVER/main.go` (the top-level application wrapper), not in `backend_server/main.go`. Only the top-level package can embed files from subdirectories.

```go
// packages/KNIRVSERVER/main.go

// (existing embeds)
var embeddedFiles embed.FS
var backendBinary []byte
var knirvgatewayBinary []byte
var knirvgraphBinary []byte
var knirvchainBinary []byte

// ADD: Embed the KNIRVORACLE binary for Oracle services
//
//go:embed bin/knirvoracle
var knirvoracleBinary []byte
```

**Note**: The Manager at `internal/services/knirvoracle/manager.go` resolves and spawns the embedded binary from `bin/knirvoracle` (relative to the embedded KNIRVSERVER binary location).

## Configuration

### Environment Variables (Oracle)

```bash
# Oracle binary (via Manager)
KNIRV_ORACLE_BINARY_PATH=/path/to/knirvoracle
ORACLE_SOCKET_PATH=/var/run/knirv/oracle.sock
ORACLE_DATA_DIR=~/.knirvserver/oracle

# Oracle runtime (read by knirvoracle binary)
ORACLE_CHAIN_ID=knirvoracle-1
ORACLE_OWNER_KEY=<hex key>
ORACLE_KEY_PASSWORD=<password>
ORACLE_P2P_ADDR=/ip4/0.0.0.0/tcp/26656
ORACLE_RPC_ADDR=127.0.0.1:26657
ORACLE_API_ADDR=0.0.0.0:8080
```

### YAML Config

```yaml
oracle:
  socket_path: "/var/run/knirv/oracle.sock"
  data_dir: "~/.knirvserver/oracle"
  start_timeout: 30s
  stop_timeout: 10s
```

## API Endpoints (via Unix Socket)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Oracle health status |
| `/oracle/v3/token/info` | GET | NRN token info |
| `/oracle/v3/token/balance/{address}` | GET | Account balance |
| `/oracle/v3/token/transfer` | POST | Transfer NRN |
| `/oracle/v3/token/mint` | POST | Mint NRN (owner) |
| `/oracle/v3/token/burn` | POST | Burn NRN |
| `/oracle/v3/governance/proposals` | GET | List proposals |
| `/oracle/v3/governance/vote` | POST | Vote on proposal |
| `/oracle/v3/economics/metrics` | GET | Economic metrics |
| `/oracle/v3/rollups/submit` | POST | Submit rollup batch |
| `/oracle/v3/rollups/{id}` | GET | Get rollup status |
| `/oracle/v3/status` | GET | Full oracle status |

## Lifecycle

1. **Build**: `make build-full` builds knirvoracle and embeds in KNIRVSERVER bin/
2. **Startup**: KNIRVSERVER starts → Manager checks for root.key
3. **Activation**: If root.key found → Manager spawns knirvoracle binary → starts socket server
4. **Runtime**: KNIRVSERVER services communicate with Oracle via Unix socket client
5. **Shutdown**: Manager stops knirvoracle gracefully → closes socket

## Error Handling

- No root.key: Log info, don't start oracle (not an error)
- Binary missing: Log error, continue without oracle (degraded)
- Oracle init fails: Log error, continue without oracle
- Socket communication fails: Return error to caller

## Migration Steps

1. **Phase 1**: Create `packages/KNIRVORACLE` structure
2. **Phase 2**: Migrate `backend/internal/oracle/` to `packages/KNIRVORACLE/internal/oracle/`
3. **Phase 3**: Create `packages/KNIRVORACLE/cmd/oracle/main.go`
4. **Phase 4**: Update KNIRVSERVER Makefile to build and embed knirvoracle
5. **Phase 5**: Create `internal/services/knirvoracle/manager.go`
6. **Phase 6**: Update `backend_server/main.go` to use Manager
7. **Phase 7**: Test integration

## Testing

```bash
# Build Oracle
cd packages/KNIRVORACLE && go build -o knirvoracle ./cmd/oracle

# Run Oracle standalone
./knirvoracle --socket /tmp/test-oracle.sock --data-dir /tmp/oracle-data

# Test with client
curl --unix-socket /tmp/test-oracle.sock http://localhost/health
```