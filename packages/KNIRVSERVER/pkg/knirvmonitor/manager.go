package knirvmonitor

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

func killStaleMonitor(binaryPath string) {
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

// Manager owns the lifecycle of the KNIRVMONITOR subprocess (the "monitor"
// binary). Unlike knirvchain/knirvgraph, KNIRVMONITOR has no Unix-socket
// mode — it always speaks plain HTTP on a TCP port.
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

// ManagerConfig mirrors the CLI flags KNIRVMONITOR's cmd/server/main.go
// accepts. The four upstream-URL fields are the same probe targets
// documented in KNIRV_CORP's metrics_alignment.md; leave any of them empty
// to skip registering that probe (KNIRVMONITOR only registers a probe when
// its URL flag is non-empty).
type ManagerConfig struct {
	BinaryPath        string
	SocketPath        string
	PrometheusURL     string
	GrafanaURL        string
	KNIRVBaseURL      string
	KNIRVChainURL     string
	KNIRVGraphURL     string
	KNIRVOracleURL    string
	GatewayURL        string
	BackendSocketPath string
	ScrapeInterval    time.Duration
	RequestTimeout    time.Duration
	StartTimeout      time.Duration
	StopTimeout       time.Duration
	Stdout            io.Writer
	Stderr            io.Writer
}

type HealthStatus struct {
	Status    string    `json:"status"`
	Running   bool      `json:"running"`
	Port      int       `json:"port"`
	Timestamp time.Time `json:"timestamp"`
}

func getMonitorAppDataDir() string {
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
	return filepath.Join(os.TempDir(), "knirvserver", "data")
}

// DefaultManagerConfig deliberately has no TCP listener. KNIRVGATEWAY owns
// public transport and reaches KNIRVMONITOR over its configured Unix socket.
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		ScrapeInterval: 15 * time.Second,
		RequestTimeout: 5 * time.Second,
		StartTimeout:   30 * time.Second,
		StopTimeout:    10 * time.Second,
	}
}

func DefaultConfig() *ManagerConfig {
	return DefaultManagerConfig()
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	defaults := DefaultManagerConfig()
	if cfg.ScrapeInterval == 0 {
		cfg.ScrapeInterval = defaults.ScrapeInterval
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = defaults.RequestTimeout
	}
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = defaults.StartTimeout
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = defaults.StopTimeout
	}

	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	baseURL := "http://knirvmonitor"
	transport := &http.Transport{}
	if strings.TrimSpace(cfg.SocketPath) != "" {
		socketPath := cfg.SocketPath
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: transport}

	return &Manager{
		binaryPath: cfg.BinaryPath,
		config:     cfg,
		logger:     logger,
		baseURL:    baseURL,
		client:     client,
	}
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVMONITOR already running")
		return nil
	}

	if m.config.BinaryPath == "" {
		m.config.BinaryPath = "./monitor"
	}

	resolved, err := resolveBinaryPath(m.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("KNIRVMONITOR binary not found (tried %s and fallbacks): %w", m.config.BinaryPath, err)
	}
	m.config.BinaryPath = resolved

	if strings.TrimSpace(m.config.SocketPath) == "" {
		return fmt.Errorf("KNIRVMONITOR socket path is required")
	}
	preflightClient := &http.Client{Timeout: 3 * time.Second, Transport: m.client.Transport}
	if resp, err := preflightClient.Get(fmt.Sprintf("%s/health", m.baseURL)); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			m.logger.Info("Adopting existing healthy KNIRVMONITOR process (skipping new spawn)")
			m.running = true
			return nil
		}
	}

	m.logger.Info("Killing any stale KNIRVMONITOR processes", zap.String("binary", m.config.BinaryPath))
	killStaleMonitor(m.config.BinaryPath)

	m.logger.Info("Starting KNIRVMONITOR",
		zap.String("binary", m.config.BinaryPath), zap.String("socket", m.config.SocketPath))

	env := os.Environ()
	env = append(env, "KNIRV_MANAGED_BY_SERVER=true")
	if m.config.BackendSocketPath != "" {
		env = append(env, "BACKEND_SOCKET_PATH="+m.config.BackendSocketPath)
	}

	args := []string{
		"-socket", m.config.SocketPath,
		"-scrape-interval", m.config.ScrapeInterval.String(),
		"-request-timeout", m.config.RequestTimeout.String(),
	}
	if m.config.PrometheusURL != "" {
		args = append(args, "-prometheus-url", m.config.PrometheusURL)
	}
	if m.config.GrafanaURL != "" {
		args = append(args, "-grafana-url", m.config.GrafanaURL)
	}
	if m.config.KNIRVBaseURL != "" {
		args = append(args, "-knirvbase-url", m.config.KNIRVBaseURL)
	}
	if m.config.KNIRVChainURL != "" {
		args = append(args, "-knirvchain-url", m.config.KNIRVChainURL)
	}
	if m.config.KNIRVGraphURL != "" {
		args = append(args, "-knirvgraph-url", m.config.KNIRVGraphURL)
	}
	if m.config.KNIRVOracleURL != "" {
		args = append(args, "-knirvoracle-url", m.config.KNIRVOracleURL)
	}
	if m.config.GatewayURL != "" {
		args = append(args, "-gateway-url", m.config.GatewayURL)
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
		return fmt.Errorf("failed to start KNIRVMONITOR: %w", err)
	}

	m.logger.Info("KNIRVMONITOR subprocess started", zap.Int("pid", m.cmd.Process.Pid))
	m.running = true

	go func(parentCtx context.Context) {
		if err := m.waitForHealth(parentCtx); err != nil {
			m.logger.Warn("KNIRVMONITOR health check did not pass during startup window", zap.Error(err))
			return
		}
		m.logger.Info("KNIRVMONITOR health check passed", zap.String("url", m.baseURL))
	}(ctx)

	return nil
}

func resolveBinaryPath(configured string) (string, error) {
	var candidates []string

	if envPath := os.Getenv("KNIRV_MONITOR_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "monitor"),
			filepath.Join(exeDir, "bin", "monitor"),
			filepath.Join(exeDir, "..", "bin", "monitor"),
		)
	}

	embeddedDir := filepath.Join(getMonitorAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "monitor"),
		filepath.Join(embeddedDir, "..", "bin", "monitor"),
	)

	candidates = append(candidates, configured)

	dir, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(dir, "bin", "monitor"),
		filepath.Join(dir, "..", "bin", "monitor"),
		filepath.Join(filepath.Dir(configured), "bin", "monitor"),
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
			return fmt.Errorf("timeout waiting for KNIRVMONITOR health check")
		case <-ticker.C:
			resp, err := m.client.Get(healthURL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					m.logger.Debug("KNIRVMONITOR health check passed")
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

	m.logger.Info("Stopping KNIRVMONITOR")

	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Warn("Failed to send SIGTERM, forcing kill", zap.Error(err))
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
			m.logger.Info("KNIRVMONITOR stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVMONITOR stopped gracefully")
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
