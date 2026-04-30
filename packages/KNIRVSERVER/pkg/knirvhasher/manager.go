package knirvhasher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func killStaleHasher(binaryPath string) {
	if binaryPath == "" {
		return
	}
	binaryName := filepath.Base(binaryPath)
	if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
		time.Sleep(1 * time.Second)
	}
	exec.Command("pkill", "-KILL", "-x", binaryName).Run()
}

type Manager struct {
	binaryPath string
	config     *ManagerConfig
	cmd        *exec.Cmd
	logger     *zap.Logger
	mu         sync.RWMutex
	running    bool
	socketPath string
	startTime  time.Time
}

type ManagerConfig struct {
	BinaryPath    string
	SocketPath    string
	DataPath      string
	HeadlessPort  int
	HeadlessMode  bool
	ArxivEnabled  bool
	StartTimeout  time.Duration
	StopTimeout   time.Duration
	Stdout        interface{}
	Stderr        interface{}
}

type HasherStatus struct {
	Status     string    `json:"status"`
	Running    bool      `json:"running"`
	SocketPath string    `json:"socket_path"`
	Uptime     string    `json:"uptime"`
	Timestamp  time.Time `json:"timestamp"`
}

func DefaultManagerConfig() *ManagerConfig {
	appDataDir := getHasherAppDataDir()
	return &ManagerConfig{
		SocketPath:   filepath.Join(appDataDir, "sockets", "hasher.sock"),
		DataPath:     filepath.Join(appDataDir, "hasher"),
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
	}
}

func getHasherAppDataDir() string {
	if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
		return explicit
	}
	if err := os.MkdirAll("/var/lib/knirvserver", 0755); err == nil {
		return "/var/lib/knirvserver"
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".local", "share", "knirvserver")
	}
	return "data"
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 30 * time.Second
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 10 * time.Second
	}
	if cfg.SocketPath == "" {
		defaults := DefaultManagerConfig()
		cfg.SocketPath = defaults.SocketPath
	}
	if cfg.DataPath == "" {
		defaults := DefaultManagerConfig()
		cfg.DataPath = defaults.DataPath
	}

	return &Manager{
		config:     cfg,
		logger:     logger,
		socketPath: cfg.SocketPath,
	}
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVHASHER already running")
		return nil
	}

	if m.config.BinaryPath == "" {
		m.config.BinaryPath = "./knirvhasher"
	}

	resolved, err := resolveBinaryPath(m.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("KNIRVHASHER binary not found: %w", err)
	}
	m.config.BinaryPath = resolved

	hasherDir := filepath.Dir(m.socketPath)
	if err := os.MkdirAll(hasherDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	if _, err := os.Stat(m.socketPath); err == nil {
		os.Remove(m.socketPath)
	}

	m.logger.Info("Starting KNIRVHASHER",
		zap.String("binary", m.config.BinaryPath),
		zap.String("socket", m.socketPath))

	env := os.Environ()
	env = append(env,
		fmt.Sprintf("HASHER_SOCKET_PATH=%s", m.socketPath),
		fmt.Sprintf("HASHER_DATA_PATH=%s", m.config.DataPath),
	)

	if m.config.ArxivEnabled {
		env = append(env, "DATAMINER_MODE=production", "ARXIV_ENABLED=true")
	}

	args := []string{}
	if m.config.HeadlessMode {
		port := m.config.HeadlessPort
		if port == 0 {
			port = 8088
		}
		args = append(args, "-headless", fmt.Sprintf("-headless-port=%d", port))
	}

	m.cmd = exec.Command(m.config.BinaryPath, args...)
	m.cmd.Env = env
	if w, ok := m.config.Stdout.(io.Writer); ok {
		m.cmd.Stdout = w
	} else {
		m.cmd.Stdout = os.Stdout
	}
	if w, ok := m.config.Stderr.(io.Writer); ok {
		m.cmd.Stderr = w
	} else {
		m.cmd.Stderr = os.Stderr
	}
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start KNIRVHASHER: %w", err)
	}
	m.startTime = time.Now()
	m.logger.Info("KNIRVHASHER subprocess started",
		zap.Int("pid", m.cmd.Process.Pid))

	if err := m.waitForSocket(ctx); err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("socket creation failed: %w", err)
	}

	// In headless mode, also wait for the HTTP API then trigger the data pipeline
	if m.config.HeadlessMode {
		httpPort := m.config.HeadlessPort
		if httpPort == 0 {
			httpPort = 8088
		}
		if err := m.waitForHTTPServer(ctx, httpPort); err != nil {
			m.logger.Warn("KNIRVHASHER HTTP API not ready, pipeline will not auto-start",
				zap.Error(err))
		} else {
			m.logger.Info("KNIRVHASHER HTTP API ready, triggering data pipeline")
			m.triggerPipeline(httpPort)
		}
	}

	m.running = true
	m.logger.Info("KNIRVHASHER started successfully",
		zap.String("socket", m.socketPath))

	return nil
}

func resolveBinaryPath(configured string) (string, error) {
	var candidates []string

	if envPath := os.Getenv("KNIRV_HASHER_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvhasher"),
			filepath.Join(exeDir, "bin", "knirvhasher"),
		)
	}

	embeddedDir := filepath.Join(getHasherAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "knirvhasher"),
		filepath.Join(embeddedDir, "..", "bin", "knirvhasher"),
	)

	candidates = append(candidates, configured)

	dir, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(dir, "bin", "knirvhasher"),
		filepath.Join(dir, "..", "bin", "knirvhasher"),
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("binary not found in any candidate location")
}

func (m *Manager) waitForSocket(ctx context.Context) error {
	deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for KNIRVHASHER socket")
		case <-ticker.C:
			if _, err := os.Stat(m.socketPath); err == nil {
				m.logger.Debug("KNIRVHASHER socket created")
				return nil
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

	m.logger.Info("Stopping KNIRVHASHER")

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
		m.startTime = time.Time{}
		if err != nil {
			m.logger.Info("KNIRVHASHER stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVHASHER stopped gracefully")
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
		m.startTime = time.Time{}
		return fmt.Errorf("forced shutdown after timeout")
	}
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) GetSocketPath() string {
	return m.socketPath
}

func (m *Manager) GetConfig() *ManagerConfig {
	return m.config
}

func (m *Manager) GetPID() int {
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

func (m *Manager) GetStatus() *HasherStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := &HasherStatus{
		Status:     "stopped",
		Running:    m.running,
		SocketPath: m.socketPath,
		Timestamp:  time.Now(),
	}
	if m.running {
		status.Status = "running"
		if !m.startTime.IsZero() {
			uptime := time.Since(m.startTime)
			status.Uptime = uptime.Round(time.Second).String()
		}
	}

	return status
}

// waitForHTTPServer polls the headless HTTP API health endpoint until it responds.
func (m *Manager) waitForHTTPServer(ctx context.Context, port int) error {
	deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	healthURL := fmt.Sprintf("http://localhost:%d/api/v1/health", port)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for KNIRVHASHER HTTP API on port %d", port)
		case <-ticker.C:
			resp, err := client.Get(healthURL)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				m.logger.Info("KNIRVHASHER HTTP API health check passed",
					zap.Int("port", port))
				return nil
			}
		}
	}
}

// triggerPipeline sends a POST request to the headless API to start the data pipeline.
func (m *Manager) triggerPipeline(port int) {
	client := &http.Client{Timeout: 10 * time.Second}
	pipelineURL := fmt.Sprintf("http://localhost:%d/api/v1/pipeline/run", port)

	resp, err := client.Post(pipelineURL, "application/json", nil)
	if err != nil {
		m.logger.Warn("Failed to trigger KNIRVHASHER data pipeline",
			zap.Int("port", port), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		m.logger.Info("KNIRVHASHER data pipeline triggered successfully",
			zap.Int("port", port), zap.Int("status", resp.StatusCode))
	} else {
		m.logger.Warn("KNIRVHASHER data pipeline trigger returned unexpected status",
			zap.Int("port", port), zap.Int("status", resp.StatusCode))
	}
}
