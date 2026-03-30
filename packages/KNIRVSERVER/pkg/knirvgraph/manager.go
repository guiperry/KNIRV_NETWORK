package knirvgraph

import (
	"context"
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
	BinaryPath   string
	Port         int
	P2PPort      int
	APIPort      int
	DataPath     string
	StartTimeout time.Duration
	StopTimeout  time.Duration
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

	m := &Manager{
		binaryPath: cfg.BinaryPath,
		config:     cfg,
		logger:     logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: fmt.Sprintf("http://localhost:%d", cfg.Port),
	}

	return m
}

func DefaultConfig() *ManagerConfig {
	return &ManagerConfig{
		Port:         7090,
		P2PPort:      7091,
		APIPort:      7092,
		DataPath:     "./data",
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
	}
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

	if err := os.MkdirAll(m.config.DataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	killStaleGraph(m.binaryPath)
	if !waitForPortFree(m.config.Port, 5*time.Second) {
		m.logger.Warn("Port not free, proceeding anyway", zap.Int("port", m.config.Port))
	}

	m.cmd = exec.Command(m.binaryPath,
		"--rpc-port", fmt.Sprintf("%d", m.config.Port),
		"--p2p-port", fmt.Sprintf("%d", m.config.P2PPort),
		"--api-port", fmt.Sprintf("%d", m.config.APIPort),
		"--home", m.config.DataPath,
		"--headless",
	)

	m.cmd.Env = append(os.Environ(),
		fmt.Sprintf("KNIRVGRAPH_PORT=%d", m.config.Port),
		fmt.Sprintf("KNIRVGRAPH_P2P_PORT=%d", m.config.P2PPort),
		fmt.Sprintf("KNIRVGRAPH_API_PORT=%d", m.config.APIPort),
		fmt.Sprintf("KNIRVGRAPH_DATA_PATH=%s", m.config.DataPath),
	)

	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start KNIRVGRAPH: %w", err)
	}

	m.running = true
	m.logger.Info("KNIRVGRAPH started", zap.Int("pid", m.cmd.Process.Pid))

	if !m.waitForHealth(m.config.StartTimeout) {
		m.logger.Warn("KNIRVGRAPH did not become healthy within timeout")
	}

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
