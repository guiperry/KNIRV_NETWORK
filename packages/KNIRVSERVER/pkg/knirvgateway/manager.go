package knirvgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func getGatewayAppDataDir() string {
	if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
		return explicit
	}
	if err := os.MkdirAll("/var/lib/knirvserver", 0755); err == nil {
		return "/var/lib/knirvserver"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "knirvserver")
	}
	return filepath.Join(os.TempDir(), "knirvserver", "data")
}

// killStaleGateway sends SIGTERM (then SIGKILL after 2 s) to any running
// processes whose executable matches the given binary name. This prevents
// port-conflict crashes when the parent process was previously killed
// without triggering the normal Stop() path.
func killStaleGateway(binaryPath string) {
	if binaryPath == "" {
		return
	}
	binaryName := filepath.Base(binaryPath)
	// SIGTERM first
	if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
		time.Sleep(1 * time.Second)
	}
	// SIGKILL any survivors
	exec.Command("pkill", "-KILL", "-x", binaryName).Run() //nolint:errcheck
}

// waitForPortFree blocks until the given TCP port is no longer occupied or
// the deadline is reached. Returns true if the port is free.
func waitForPortFree(port int, deadline time.Duration) bool {
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// releaseTCPPort terminates any listener currently owning port. KNIRVGATEWAY is
// the public ingress for the node, so the supervisor must make the configured
// port available before spawning the child instead of accepting a different one.
func releaseTCPPort(port int, logger *zap.Logger) {
	if port <= 0 || waitForPortFree(port, 200*time.Millisecond) {
		return
	}

	target := fmt.Sprintf("%d/tcp", port)
	if fuserPath, err := exec.LookPath("fuser"); err == nil {
		logger.Warn("Releasing occupied KNIRVGATEWAY port", zap.Int("port", port))
		_ = exec.Command(fuserPath, "-k", "-TERM", target).Run()
		if waitForPortFree(port, 2*time.Second) {
			return
		}
		_ = exec.Command(fuserPath, "-k", "-KILL", target).Run()
		_ = waitForPortFree(port, 2*time.Second)
		return
	}

	if lsofPath, err := exec.LookPath("lsof"); err == nil {
		out, err := exec.Command(lsofPath, "-tiTCP:"+fmt.Sprint(port), "-sTCP:LISTEN").Output()
		if err == nil {
			pids := strings.Fields(string(out))
			if len(pids) > 0 {
				logger.Warn("Releasing occupied KNIRVGATEWAY port", zap.Int("port", port), zap.Strings("pids", pids))
				_ = exec.Command("kill", append([]string{"-TERM"}, pids...)...).Run()
				if waitForPortFree(port, 2*time.Second) {
					return
				}
				_ = exec.Command("kill", append([]string{"-KILL"}, pids...)...).Run()
				_ = waitForPortFree(port, 2*time.Second)
			}
		}
	}
}

// withEnvOverrides returns a child environment with each override replacing
// the inherited value. execve permits duplicate environment keys, but lookup
// order for duplicates is platform/runtime-dependent. That matters here:
// backend_server supplies the key-derived CHAIN_NODE_ROLE and Cloudflare
// credentials as overrides, while KNIRVSERVER may already have values for
// those names in its own environment. Leaving both entries can make the
// gateway see the stale value and reject tunnel ownership during startup.
func withEnvOverrides(env []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return env
	}

	keys := make(map[string]struct{}, len(overrides))
	for key := range overrides {
		keys[key] = struct{}{}
	}

	filtered := make([]string, 0, len(env)+len(overrides))
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, replaced := keys[key]; replaced {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	for key, value := range overrides {
		filtered = append(filtered, key+"="+value)
	}
	return filtered
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
	BinaryPath             string
	Port                   int
	BackendAPIPort         int
	Ports                  *PortConfig
	DBPath                 string
	AuthSecret             string
	MinerAddress           string
	BackendSocketPath      string
	MonitorSocketPath      string
	AgentControlSocketPath string
	ChainSocketPath        string
	GraphSocketPath        string
	OracleSocketPath       string
	StartTimeout           time.Duration
	StopTimeout            time.Duration
	ChainID                string
	Stdout                 io.Writer
	Stderr                 io.Writer
	EnvOverrides           map[string]string // extra env vars (e.g. Cloudflare creds from root.key)
}

type PortConfig struct {
	TurnUDP     int
	TurnTCP     int
	TurnAPI     int
	TunnelHTTP  int
	TunnelCtrl  int
	TunnelRelay int
	TunnelSTUN  int
}

type TurnServerStatus struct {
	Status       string    `json:"status"`
	Running      bool      `json:"running"`
	UDPPort      int       `json:"udp_port"`
	TCPPort      int       `json:"tcp_port"`
	APIPort      int       `json:"api_port"`
	Realm        string    `json:"realm"`
	SessionCount int64     `json:"session_count"`
	ActiveRelays int64     `json:"active_relays"`
	Uptime       string    `json:"uptime"`
	Timestamp    time.Time `json:"timestamp"`
}

type HealthStatus struct {
	Status     string                 `json:"status"`
	TurnServer bool                   `json:"turn_server"`
	Services   map[string]interface{} `json:"services"`
	Timestamp  time.Time              `json:"timestamp"`
}

func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		Port: 8080,
		Ports: &PortConfig{
			TurnUDP:     3478,
			TurnTCP:     3479,
			TurnAPI:     3476,
			TunnelHTTP:  13002,
			TunnelCtrl:  13003,
			TunnelRelay: 13004,
			TunnelSTUN:  13005,
		},
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
		ChainID:      "testnet",
	}
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 30 * time.Second
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 10 * time.Second
	}

	port := cfg.Port
	if port == 0 {
		port = 8080
	}
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	m := &Manager{
		config:  cfg,
		logger:  logger,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	return m
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVGATEWAY already running")
		return nil
	}

	if m.config.BinaryPath == "" {
		m.config.BinaryPath = "./knirvgateway"
	}

	// Resolve the binary path, trying several candidate locations.
	resolved, err := resolveBinaryPath(m.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("KNIRVGATEWAY binary not found (tried %s and fallbacks): %w", m.config.BinaryPath, err)
	}
	m.config.BinaryPath = resolved

	// A healthy process already listening on this port is never adopted: it may
	// be a stale instance from a previous run (e.g. a prior -prod launch) that
	// was started with different CHAIN_NODE_ROLE / KNIRV_NETWORK_MODE / Cloudflare
	// credential env vars. Adopting it would silently keep that old configuration
	// active — including skipping Cloudflare Tunnel setup — regardless of what
	// the current run intends. It would also become unmanaged: Stop() below only
	// signals m.cmd, which is never set on the adopt path, so an adopted process
	// survives every future stop/restart cycle.
	//
	// Always kill any stale instance and spawn fresh so the new process reflects
	// this run's environment exactly.
	m.logger.Info("Killing any stale KNIRVGATEWAY processes", zap.String("binary", m.config.BinaryPath))
	killStaleGateway(m.config.BinaryPath)

	gatewayPort := m.config.Port
	if gatewayPort <= 0 {
		gatewayPort = 8080
	}
	releaseTCPPort(gatewayPort, m.logger)
	if !waitForPortFree(gatewayPort, 5*time.Second) {
		m.logger.Warn("KNIRVGATEWAY port still occupied after release attempt",
			zap.Int("port", gatewayPort))
	}

	// Wait for the tunnel control port to be released before spawning the new
	// binary. This closes the race window between kill and exec.
	ctrlPort := 13003
	if m.config.Ports != nil && m.config.Ports.TunnelCtrl != 0 {
		ctrlPort = m.config.Ports.TunnelCtrl
	}
	if !waitForPortFree(ctrlPort, 5*time.Second) {
		m.logger.Warn("Tunnel control port still occupied after kill — proceeding anyway",
			zap.Int("port", ctrlPort))
	}

	m.logger.Info("Starting KNIRVGATEWAY",
		zap.String("binary", m.config.BinaryPath),
		zap.Int("port", m.config.Port))

	env := os.Environ()
	backendPort := m.config.BackendAPIPort
	if backendPort <= 0 {
		backendPort = 9081
	}
	env = append(env,
		fmt.Sprintf("KNIRV_BACKEND_API_URL=http://127.0.0.1:%d", backendPort),
		fmt.Sprintf("KNIRV_CHAIN_ID=%s", m.config.ChainID),
		"AUTO_OPEN_BROWSER=false",
		"TURN_SERVER_ENABLED=true",
		fmt.Sprintf("TURN_SERVER_UDP_PORT=%d", m.config.Ports.TurnUDP),
		fmt.Sprintf("TURN_SERVER_TCP_PORT=%d", m.config.Ports.TurnTCP),
		fmt.Sprintf("TURN_SERVER_API_PORT=%d", m.config.Ports.TurnAPI),
		fmt.Sprintf("TURN_SERVER_AUTH_SECRET=%s", m.config.AuthSecret),
		"TURN_SERVER_REALM=knirvgateway.local",
		fmt.Sprintf("TURN_SERVER_MINER_ADDRESS=%s", m.config.MinerAddress),
		"TUNNEL_REGISTRY_ENABLED=true",
		fmt.Sprintf("TUNNEL_REGISTRY_HTTP_PORT=%d", m.config.Ports.TunnelHTTP),
		fmt.Sprintf("TUNNEL_REGISTRY_CONTROL_PORT=%d", m.config.Ports.TunnelCtrl),
		fmt.Sprintf("TUNNEL_REGISTRY_PUBLIC_RELAY_PORT=%d", m.config.Ports.TunnelRelay),
		fmt.Sprintf("TUNNEL_REGISTRY_STUN_PORT=%d", m.config.Ports.TunnelSTUN),
		fmt.Sprintf("DATABASE_PATH=%s", m.config.DBPath),
	)

	if m.config.BackendSocketPath != "" {
		env = append(env, fmt.Sprintf("BACKEND_SOCKET_PATH=%s", m.config.BackendSocketPath))
	}
	if m.config.MonitorSocketPath != "" {
		env = append(env, fmt.Sprintf("MONITOR_SOCKET_PATH=%s", m.config.MonitorSocketPath))
	}
	if m.config.AgentControlSocketPath != "" {
		env = append(env, fmt.Sprintf("AGENT_CONTROL_SOCKET_PATH=%s", m.config.AgentControlSocketPath))
	}
	if m.config.ChainSocketPath != "" {
		env = append(env, fmt.Sprintf("CHAIN_SOCKET_PATH=%s", m.config.ChainSocketPath))
	}
	if m.config.GraphSocketPath != "" {
		env = append(env, fmt.Sprintf("GRAPH_SOCKET_PATH=%s", m.config.GraphSocketPath))
	}
	if m.config.OracleSocketPath != "" {
		env = append(env, fmt.Sprintf("ORACLE_SOCKET_PATH=%s", m.config.OracleSocketPath))
	}

	// Apply env overrides (e.g. Cloudflare credentials from root.key).
	// Replace inherited entries so the child cannot observe a stale role or
	// credential before the authoritative override.
	env = withEnvOverrides(env, m.config.EnvOverrides)

	env = append(env, fmt.Sprintf("PORT=%d", gatewayPort))

	args := []string{}

	m.cmd = exec.Command(m.config.BinaryPath, args...)
	m.cmd.Env = env
	if m.config.Stdout != nil {
		m.cmd.Stdout = m.config.Stdout
	} else {
		m.cmd.Stdout = os.Stdout
	}
	if m.config.Stderr != nil {
		m.cmd.Stderr = m.config.Stderr
	} else {
		m.cmd.Stderr = os.Stderr
	}
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start KNIRVGATEWAY: %w", err)
	}

	m.logger.Info("KNIRVGATEWAY subprocess started",
		zap.Int("pid", m.cmd.Process.Pid))

	if err := m.waitForHealth(ctx); err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("health check failed: %w", err)
	}

	m.running = true
	m.logger.Info("KNIRVGATEWAY started successfully",
		zap.String("url", m.baseURL))

	return nil
}

// resolveBinaryPath returns the first path where the knirvgateway binary exists.
// Priority: KNIRV_GATEWAY_BINARY_PATH env var → configured path → standard fallbacks.
func resolveBinaryPath(configured string) (string, error) {
	var candidates []string

	// Highest priority: explicit env var set by the launcher (main.go).
	if envPath := os.Getenv("KNIRV_GATEWAY_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	// Try executable directory first for embedded binaries
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvgateway"),
			filepath.Join(exeDir, "bin", "knirvgateway"),
		)
	}

	// Embedded directory (extracted binaries)
	embeddedDir := filepath.Join(getGatewayAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "knirvgateway"),
		filepath.Join(embeddedDir, "..", "bin", "knirvgateway"),
	)

	candidates = append(candidates, configured)

	// Common fallback locations relative to the working directory.
	dir, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(dir, "bin", "knirvgateway"),
		filepath.Join(dir, "..", "bin", "knirvgateway"),
		filepath.Join(filepath.Dir(configured), "bin", "knirvgateway"),
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("binary not found in any candidate location")
}

func (m *Manager) waitForHealth(ctx context.Context) error {
	deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	healthURL := fmt.Sprintf("%s/health", m.baseURL)

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for KNIRVGATEWAY health check")
		case <-ticker.C:
			resp, err := m.client.Get(healthURL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					m.logger.Debug("KNIRVGATEWAY health check passed")
					return nil
				}
			}
		}
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	m.logger.Info("Stopping KNIRVGATEWAY")

	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Warn("Failed to send SIGTERM, forcing kill",
			zap.Error(err))
		syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
	}

	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	case err := <-done:
		m.running = false
		if err != nil {
			m.logger.Info("KNIRVGATEWAY stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVGATEWAY stopped gracefully")
		}
		return nil
	case <-time.After(m.config.StopTimeout):
		m.logger.Warn("Timeout waiting for graceful shutdown, forcing kill")
		syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			m.logger.Warn("Wait() did not complete after Kill() — zombie possible")
		}
		m.running = false
		return fmt.Errorf("forced shutdown after timeout")
	}
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) GetBaseURL() string {
	return m.baseURL
}

func (m *Manager) GetConfig() *ManagerConfig {
	return m.config
}

func (m *Manager) HealthCheck(ctx context.Context) error {
	resp, err := m.client.Get(fmt.Sprintf("%s/health", m.baseURL))
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

func (m *Manager) GetTurnStatus(ctx context.Context) (*TurnServerStatus, error) {
	url := fmt.Sprintf("http://localhost:%d/api/turn/status", m.config.Ports.TurnAPI)
	resp, err := m.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get TURN status: %w", err)
	}
	defer resp.Body.Close()

	var status TurnServerStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode TURN status: %w", err)
	}

	return &status, nil
}

func (m *Manager) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	url := fmt.Sprintf("%s/health", m.baseURL)
	resp, err := m.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}
	defer resp.Body.Close()

	var status HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode health status: %w", err)
	}

	return &status, nil
}

func (m *Manager) GetPID() int {
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

func (m *Manager) GetPorts() *PortConfig {
	return m.config.Ports
}
