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
	return "data"
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
	BinaryPath     string
	SocketPath     string
	Port           int
	BackendAPIPort int
	Ports          *PortConfig
	DBPath         string
	AuthSecret     string
	MinerAddress   string
	StartTimeout   time.Duration
	StopTimeout    time.Duration
	ChainID        string
	Stdout         io.Writer
	Stderr         io.Writer
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
		SocketPath: "gateway.sock",
		Port:       8080,
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

	var baseURL string
	if cfg.SocketPath != "" {
		baseURL = "http://localhost"
	} else {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
		if cfg.Port == 0 {
			baseURL = "http://localhost:8080"
		}
	}

	m := &Manager{
		config:  cfg,
		logger:  logger,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if cfg.SocketPath != "" {
		m.client.Transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", cfg.SocketPath)
			},
		}
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

	// Pre-flight: if a healthy gateway is already listening (stale process from a
	// previous run), adopt it rather than starting a duplicate and hitting a port
	// conflict on the tunnel control port (3003).
	preflightClient := &http.Client{
		Timeout:   3 * time.Second,
		Transport: m.client.Transport,
	}
	if resp, err := preflightClient.Get(fmt.Sprintf("%s/health", m.baseURL)); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			m.logger.Info("Adopting existing healthy KNIRVGATEWAY process (skipping new spawn)")
			m.running = true
			return nil
		}
	}

	// Existing process is unhealthy or absent — kill any stale instance so the
	// ports (especially the tunnel control port 3003) are free before we start.
	m.logger.Info("Killing any stale KNIRVGATEWAY processes", zap.String("binary", m.config.BinaryPath))
	killStaleGateway(m.config.BinaryPath)

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
		zap.String("socketPath", m.config.SocketPath))

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

	if m.config.SocketPath != "" {
		env = append(env, fmt.Sprintf("SOCKET_PATH=%s", m.config.SocketPath))
	} else {
		gatewayPort := m.config.Port
		if gatewayPort <= 0 {
			gatewayPort = 8080
		}
		env = append(env, fmt.Sprintf("PORT=%d", gatewayPort))
	}

	args := []string{}
	if m.config.SocketPath != "" {
		args = append(args, "-socket", m.config.SocketPath)
	}

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
		m.cmd.Process.Kill()
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
		m.cmd.Process.Kill()
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
