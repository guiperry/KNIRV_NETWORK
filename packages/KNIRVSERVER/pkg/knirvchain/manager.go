package knirvchain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func killStaleChain(binaryPath string) {
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
	Role          string
	ChainID       string
	StartTimeout  time.Duration
	StopTimeout   time.Duration
	Stdout        io.Writer
	Stderr        io.Writer
}

type HealthStatus struct {
	Status    string    `json:"status"`
	Running   bool      `json:"running"`
	Port      int       `json:"port"`
	Timestamp time.Time `json:"timestamp"`
}

func getChainAppDataDir() string {
	if explicit := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); explicit != "" {
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
	return "data"
}

func DefaultManagerConfig() *ManagerConfig {
	appDataDir := getChainAppDataDir()
	return &ManagerConfig{
		SocketPath:    "chain.sock",
		P2PSocketPath: "chain-p2p.sock",
		Port:          8083,
		P2PPort:       4001,
		APIPort:       9090,
		DataPath:      filepath.Join(appDataDir, "knirvchain"),
		Role:          "client",
		ChainID:       "testnet",
		StartTimeout:  30 * time.Second,
		StopTimeout:   10 * time.Second,
	}
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	defaults := DefaultManagerConfig()
	if cfg.Port == 0 {
		cfg.Port = defaults.Port
	}
	if cfg.APIPort == 0 {
		cfg.APIPort = cfg.Port
	}
	if cfg.P2PPort == 0 {
		cfg.P2PPort = defaults.P2PPort
	}
	if cfg.Role == "" {
		cfg.Role = defaults.Role
	}
	if cfg.ChainID == "" {
		cfg.ChainID = defaults.ChainID
	}

	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	var baseURL string
	var client *http.Client
	if cfg.SocketPath != "" {
		baseURL = "http://localhost"
		client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", cfg.SocketPath)
				},
			},
		}
	} else {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
		client = &http.Client{
			Timeout: 10 * time.Second,
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

func DefaultConfig() *ManagerConfig {
	cfg := DefaultManagerConfig()
	return cfg
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVCHAIN already running")
		return nil
	}

	if m.config.BinaryPath == "" {
		m.config.BinaryPath = "./knirvchain"
	}

	resolved, err := resolveBinaryPath(m.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("KNIRVCHAIN binary not found (tried %s and fallbacks): %w", m.config.BinaryPath, err)
	}
	m.config.BinaryPath = resolved

	preflightClient := &http.Client{Timeout: 3 * time.Second}
	if m.config.SocketPath != "" {
		preflightClient = m.client
	}
	if resp, err := preflightClient.Get(fmt.Sprintf("%s/health", m.baseURL)); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			m.logger.Info("Adopting existing healthy KNIRVCHAIN process (skipping new spawn)")
			m.running = true
			return nil
		}
	}

	m.logger.Info("Killing any stale KNIRVCHAIN processes", zap.String("binary", m.config.BinaryPath))
	killStaleChain(m.config.BinaryPath)

	if m.config.SocketPath == "" && !waitForPortFree(m.config.Port, 5*time.Second) {
		m.logger.Warn("API port still occupied after kill — proceeding anyway",
			zap.Int("port", m.config.Port))
	}

	m.logger.Info("Starting KNIRVCHAIN",
		zap.String("binary", m.config.BinaryPath),
		zap.Int("port", m.config.Port),
		zap.Int("api_port", m.config.APIPort),
		zap.Int("p2p_port", m.config.P2PPort),
		zap.String("socket_path", m.config.SocketPath),
		zap.String("role", m.config.Role))

	env := os.Environ()
	env = append(env,
		fmt.Sprintf("KNIRV_HTTP_PORT=%d", m.config.Port),
		fmt.Sprintf("KNIRV_P2P_PORT=%d", m.config.P2PPort),
		fmt.Sprintf("KNIRV_CHAIN_ID=%s", m.config.ChainID),
		fmt.Sprintf("KNIRV_DATA_DIR=%s", m.config.DataPath),
		fmt.Sprintf("KNIRV_SOCKET_PATH=%s", m.config.SocketPath),
		// GATEWAY_SOCKET_PATH is no longer needed — KNIRVCHAIN now calls the
		// gateway directly via http://localhost:8080.
	)

	args := []string{
		"-role", normalizeRole(m.config.Role),
		"-p2p.port", fmt.Sprintf("%d", m.config.P2PPort),
		"-skip-install",
		"-non-interactive",
	}
	// Start the wallet server only in root mode — client mode doesn't need it
	// and the WebGUI's /wallet/info endpoint depends on it being available.
	if normalizeRole(m.config.Role) != "root" {
		args = append(args, "-no-wallet-server")
	}
	if !m.config.NoUI {
		args = append(args, "-gui=false")
	}
	if m.config.SocketPath != "" {
		args = append(args, "-socket", m.config.SocketPath)
	} else {
		args = append(args, "-port", fmt.Sprintf("%d", m.config.Port))
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
		return fmt.Errorf("failed to start KNIRVCHAIN: %w", err)
	}

	m.logger.Info("KNIRVCHAIN subprocess started",
		zap.Int("pid", m.cmd.Process.Pid))

	m.running = true
	m.logger.Info("KNIRVCHAIN started successfully",
		zap.String("url", m.baseURL))

	go func(parentCtx context.Context) {
		if err := m.waitForHealth(parentCtx); err != nil {
			m.logger.Warn("KNIRVCHAIN health check did not pass during startup window", zap.Error(err))
			return
		}
		m.logger.Info("KNIRVCHAIN health check passed", zap.String("url", m.baseURL))
	}(ctx)

	return nil
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "root":
		return "Root"
	case "bootnode":
		return "Bootnode"
	case "peer":
		return "Peer"
	case "client":
		return "Client"
	default:
		return "Client"
	}
}

func resolveBinaryPath(configured string) (string, error) {
	var candidates []string

	if envPath := os.Getenv("KNIRV_CHAIN_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvchain"),
			filepath.Join(exeDir, "bin", "knirvchain"),
			filepath.Join(exeDir, "..", "bin", "knirvchain"),
		)
	}

	embeddedDir := filepath.Join(getChainAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "knirvchain"),
		filepath.Join(embeddedDir, "..", "bin", "knirvchain"),
	)

	candidates = append(candidates, configured)

	dir, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(dir, "bin", "knirvchain"),
		filepath.Join(dir, "..", "bin", "knirvchain"),
		filepath.Join(filepath.Dir(configured), "bin", "knirvchain"),
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
			return fmt.Errorf("timeout waiting for KNIRVCHAIN health check")
		case <-ticker.C:
			resp, err := m.client.Get(healthURL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					m.logger.Debug("KNIRVCHAIN health check passed")
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

	m.logger.Info("Stopping KNIRVCHAIN")

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
			m.logger.Info("KNIRVCHAIN stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVCHAIN stopped gracefully")
		}
		return nil
	case <-time.After(m.config.StopTimeout):
		m.logger.Warn("Timeout waiting for graceful shutdown, forcing kill")
		m.cmd.Process.Kill()
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
