package knirvoracle

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

func killStaleOracle(binaryPath string) {
	if binaryPath == "" {
		return
	}
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
	EnvOverrides map[string]string
}

type HealthStatus struct {
	Status    string    `json:"status"`
	Running   bool      `json:"running"`
	Socket    string    `json:"socket"`
	Timestamp time.Time `json:"timestamp"`
}

func getOracleAppDataDir() string {
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

func DefaultManagerConfig() *ManagerConfig {
	appDataDir := getOracleAppDataDir()
	return &ManagerConfig{
		SocketPath:   filepath.Join(appDataDir, "sockets", "oracle.sock"),
		DataPath:     filepath.Join(appDataDir, "oracle"),
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

	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	m := &Manager{
		binaryPath: cfg.BinaryPath,
		config:     cfg,
		logger:     logger,
		baseURL:    "http://unix",
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", cfg.SocketPath)
				},
			},
		},
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

	rootKeyPath, err := m.checkRootKeyActivation()
	if err != nil {
		m.logger.Info("No valid root.key found — Oracle will not activate", zap.Error(err))
		return nil
	}
	m.config.RootKeyPath = rootKeyPath

	resolved, err := m.resolveBinaryPath()
	if err != nil {
		return fmt.Errorf("KNIRVORACLE binary not found: %w", err)
	}
	m.binaryPath = resolved

	preflightClient := &http.Client{Timeout: 3 * time.Second}
	if m.config.SocketPath != "" {
		preflightClient = m.client
	}
	if resp, err := preflightClient.Get(fmt.Sprintf("%s/health", m.baseURL)); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			m.logger.Info("Adopting existing healthy KNIRVORACLE process")
			m.running = true
			return nil
		}
	}

	m.logger.Info("Killing any stale KNIRVORACLE processes", zap.String("binary", m.binaryPath))
	killStaleOracle(m.binaryPath)

	if m.config.SocketPath != "" {
		if !waitForSocketFree(m.config.SocketPath, 5*time.Second) {
			m.logger.Warn("Socket still in use after kill — proceeding anyway")
		}
	}

	m.logger.Info("Starting KNIRVORACLE",
		zap.String("binary", m.binaryPath),
		zap.String("socket", m.config.SocketPath),
		zap.String("data_dir", m.config.DataPath))

	env := os.Environ()
	env = append(env,
		fmt.Sprintf("ORACLE_SOCKET_PATH=%s", m.config.SocketPath),
		fmt.Sprintf("ORACLE_DATA_DIR=%s", m.config.DataPath),
		fmt.Sprintf("ORACLE_ROOT_KEY_PATH=%s", m.config.RootKeyPath),
	)

	for k, v := range m.config.EnvOverrides {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	m.cmd = exec.Command(m.binaryPath)
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
		return fmt.Errorf("failed to start KNIRVORACLE: %w", err)
	}

	m.logger.Info("KNIRVORACLE subprocess started", zap.Int("pid", m.cmd.Process.Pid))
	m.running = true
	m.logger.Info("KNIRVORACLE started successfully", zap.String("url", m.baseURL))

	go func(parentCtx context.Context) {
		if err := m.waitForHealth(parentCtx); err != nil {
			m.logger.Warn("KNIRVORACLE health check did not pass during startup window", zap.Error(err))
			return
		}
		m.logger.Info("KNIRVORACLE health check passed", zap.String("url", m.baseURL))
	}(ctx)

	return nil
}

func (m *Manager) checkRootKeyActivation() (string, error) {
	rootKeyPath, err := ResolveAndValidateRootKey(m.config.RootKeyPath)
	if err != nil {
		return "", err
	}

	m.logger.Info("Found valid root.key — Oracle will activate", zap.String("path", rootKeyPath))
	return rootKeyPath, nil
}

func (m *Manager) resolveBinaryPath() (string, error) {
	var candidates []string

	if envPath := os.Getenv("KNIRV_ORACLE_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	} else {
		m.logger.Debug("Failed to extract embedded KNIRVORACLE binary", zap.Error(err))
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvoracle"),
			filepath.Join(exeDir, "bin", "knirvoracle"),
			filepath.Join(exeDir, "..", "bin", "knirvoracle"),
		)
	}

	embeddedDir := filepath.Join(getOracleAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "knirvoracle"),
		filepath.Join(embeddedDir, "..", "bin", "knirvoracle"),
	)

	candidates = append(candidates, m.binaryPath)

	dir, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(dir, "bin", "knirvoracle"),
		filepath.Join(dir, "..", "bin", "knirvoracle"),
		filepath.Join(filepath.Dir(m.binaryPath), "bin", "knirvoracle"),
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("knirvoracle binary not found in any candidate location")
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
			return fmt.Errorf("timeout waiting for KNIRVORACLE health check")
		case <-ticker.C:
			resp, err := m.client.Get(healthURL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					m.logger.Debug("KNIRVORACLE health check passed")
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
		m.running = false
		return nil
	}

	m.logger.Info("Stopping KNIRVORACLE")

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
			m.logger.Info("KNIRVORACLE stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVORACLE stopped gracefully")
		}
		return nil
	case <-time.After(m.config.StopTimeout):
		m.logger.Warn("Timeout waiting for graceful shutdown, forcing kill")
		syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
		// Wait for the reap goroutine to finish so the zombie does not
		// linger.  After SIGKILL the child should die nearly instantly.
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

func (m *Manager) GetClient() *Client {
	return &Client{
		baseURL: m.baseURL,
		client:  m.client,
	}
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
