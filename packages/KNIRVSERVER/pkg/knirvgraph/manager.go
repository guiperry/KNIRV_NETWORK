package knirvgraph

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func killStaleGraph(binaryPath string) {
	if binaryPath == "" {
		return
	}
	binaryName := filepath.Base(binaryPath)
	if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
		time.Sleep(1 * time.Second)
	}
	exec.Command("pkill", "-KILL", "-x", binaryName).Run()
}

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
	BinaryPath    string
	SocketPath    string
	P2PSocketPath string
	Port          int
	P2PPort       int
	APIPort       int
	DataPath      string
	StartTimeout  time.Duration
	StopTimeout   time.Duration
	Stdout        io.Writer
	Stderr        io.Writer
	EnvOverrides  map[string]string
}

type HealthStatus struct {
	Status    string    `json:"status"`
	Running   bool      `json:"running"`
	Port      int       `json:"port"`
	Timestamp time.Time `json:"timestamp"`
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	var baseURL string
	var client *http.Client
	if cfg.SocketPath != "" {
		baseURL = "http://unix/" + cfg.SocketPath
		client = &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", cfg.SocketPath)
				},
			},
		}
	} else {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	m := &Manager{
		binaryPath: cfg.BinaryPath,
		config:     cfg,
		logger:     logger,
		baseURL:    baseURL,
		client:     client,
	}

	return m
}

func getAppDataDir() string {
	if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
		return explicit
	}
	if err := os.MkdirAll("/var/lib/knirvserver", 0755); err == nil {
		return "/var/lib/knirvserver"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "knirvserver")
	}
	if usr, err := user.Current(); err == nil {
		return filepath.Join(usr.HomeDir, ".local", "share", "knirvserver")
	}
	return filepath.Join(os.TempDir(), "knirvserver", "data")
}

func DefaultConfig() *ManagerConfig {
	appDataDir := getAppDataDir()
	return &ManagerConfig{
		SocketPath:    filepath.Join(appDataDir, "sockets", "graph.sock"),
		P2PSocketPath: filepath.Join(appDataDir, "sockets", "graph-p2p.sock"),
		Port:          7090,
		P2PPort:       7091,
		APIPort:       7092,
		DataPath:      filepath.Join(appDataDir, "knirvgraph"),
		StartTimeout:  30 * time.Second,
		StopTimeout:   10 * time.Second,
	}
}

func resolveBinaryPath(configured string) (string, error) {
	var candidates []string

	if envPath := os.Getenv("KNIRV_GRAPH_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvgraph"),
			filepath.Join(exeDir, "bin", "knirvgraph"),
			filepath.Join(exeDir, "..", "bin", "knirvgraph"),
		)
	}

	embeddedDir := filepath.Join(getAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "knirvgraph"),
		filepath.Join(embeddedDir, "..", "bin", "knirvgraph"),
	)

	candidates = append(candidates, configured)

	dir, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(dir, "bin", "knirvgraph"),
		filepath.Join(dir, "..", "bin", "knirvgraph"),
		filepath.Join(filepath.Dir(configured), "bin", "knirvgraph"),
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("binary not found in any candidate location")
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVGRAPH already running")
		return nil
	}

	m.logger.Info("Starting embedded KNIRVGRAPH",
		zap.String("binary", m.binaryPath),
		zap.Int("port", m.config.Port),
		zap.Int("p2p_port", m.config.P2PPort),
		zap.Int("api_port", m.config.APIPort),
		zap.String("data_path", m.config.DataPath),
	)

	// Resolve binary path (same logic as knirvgateway/knirvchain)
	resolved, err := resolveBinaryPath(m.binaryPath)
	if err != nil {
		return fmt.Errorf("KNIRVGRAPH binary not found: %w", err)
	}
	m.binaryPath = resolved
	m.logger.Info("Resolved KNIRVGRAPH binary", zap.String("binary", m.binaryPath))

	if err := os.MkdirAll(m.config.DataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	killStaleGraph(m.binaryPath)
	if !waitForPortFree(m.config.Port, 5*time.Second) {
		m.logger.Warn("Port not free, proceeding anyway", zap.Int("port", m.config.Port))
	}

	m.cmd = exec.Command(m.binaryPath,
		"--rpc-port", fmt.Sprintf("%d", m.config.Port),
		"--port", fmt.Sprintf("%d", m.config.P2PPort),
		"--home", m.config.DataPath,
		"--headless",
	)

	// Pass --socket if a socket path is configured (creates a Unix socket for the gateway to proxy to)
	if m.config.SocketPath != "" {
		m.cmd.Args = append(m.cmd.Args, "--socket", m.config.SocketPath)
	}

	m.cmd.Env = append(os.Environ(),
		fmt.Sprintf("KNIRVGRAPH_PORT=%d", m.config.Port),
		fmt.Sprintf("KNIRVGRAPH_P2P_PORT=%d", m.config.P2PPort),
		fmt.Sprintf("KNIRVGRAPH_API_PORT=%d", m.config.APIPort),
		fmt.Sprintf("KNIRVGRAPH_DATA_PATH=%s", m.config.DataPath),
	)
	for k, v := range m.config.EnvOverrides {
		m.cmd.Env = append(m.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

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

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start KNIRVGRAPH: %w", err)
	}

	m.running = true
	m.logger.Info("KNIRVGRAPH started", zap.Int("pid", m.cmd.Process.Pid))

	go func(timeout time.Duration) {
		if !m.waitForHealth(timeout) {
			m.logger.Warn("KNIRVGRAPH did not become healthy within timeout")
			return
		}
		m.logger.Info("KNIRVGRAPH health check passed")
	}(m.config.StartTimeout)

	return nil
}

func (m *Manager) waitForHealth(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if m.IsHealthy() {
				return true
			}
		}
	}
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.cmd == nil || m.cmd.Process == nil {
		m.logger.Info("KNIRVGRAPH not running")
		return nil
	}

	m.logger.Info("Stopping KNIRVGRAPH", zap.Int("pid", m.cmd.Process.Pid))

	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Warn("Failed to send SIGTERM", zap.Error(err))
	}

	done := make(chan error, 1)
	go func() {
		done <- m.cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(m.config.StopTimeout):
		m.logger.Warn("KNIRVGRAPH did not stop gracefully, sending SIGKILL")
		m.cmd.Process.Signal(syscall.SIGKILL)
		m.cmd.Wait()
	}

	m.running = false
	m.logger.Info("KNIRVGRAPH stopped")
	return nil
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) IsHealthy() bool {
	if !m.IsRunning() {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 500
}

func (m *Manager) GetStatus() (*HealthStatus, error) {
	status := &HealthStatus{
		Status:    "stopped",
		Running:   false,
		Port:      m.config.Port,
		Timestamp: time.Now(),
	}

	if !m.IsRunning() {
		return status, nil
	}

	status.Running = true

	if m.IsHealthy() {
		status.Status = "healthy"
	} else {
		status.Status = "unhealthy"
	}

	return status, nil
}

func (m *Manager) GetPort() int {
	return m.config.Port
}

func (m *Manager) GetP2PPort() int {
	return m.config.P2PPort
}

func (m *Manager) GetAPIPort() int {
	return m.config.APIPort
}

func (m *Manager) GetBaseURL() string {
	return m.baseURL
}

func (m *Manager) GetConfig() *ManagerConfig {
	return m.config
}
