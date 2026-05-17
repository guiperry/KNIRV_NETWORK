package validationchain

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

type ManagerConfig struct {
	BinaryPath   string
	WorkDir      string
	Port         int
	DataPath     string
	ChainID      string
	StartTimeout time.Duration
	StopTimeout  time.Duration
	Stdout       io.Writer
	Stderr       io.Writer
}

type Manager struct {
	config  *ManagerConfig
	cmd     *exec.Cmd
	logger  *zap.Logger
	client  *http.Client
	baseURL string
	mu      sync.RWMutex
	running bool
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg == nil {
		cfg = &ManagerConfig{
			Port:         9290,
			StartTimeout: 30 * time.Second,
			StopTimeout:  10 * time.Second,
		}
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &Manager{
		config:  cfg,
		logger:  logger,
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: fmt.Sprintf("http://localhost:%d", cfg.Port),
	}
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}
	if m.config.BinaryPath == "" {
		// Try common binary locations
		if path, err := m.resolveBinary(); err == nil {
			m.config.BinaryPath = path
			m.logger.Info("Validation chain binary resolved", zap.String("path", path))
		} else {
			return fmt.Errorf("validation chain binary path not configured and auto-resolution failed: %w", err)
		}
	}

	m.cmd = exec.Command(m.config.BinaryPath)
	if m.config.WorkDir != "" {
		m.cmd.Dir = m.config.WorkDir
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
	m.cmd.Env = append(os.Environ(),
		fmt.Sprintf("KNIRVCHAIN_RPC_ENDPOINT=127.0.0.1:%d", m.config.Port),
		fmt.Sprintf("CHAIN_ID=%s", m.config.ChainID),
		fmt.Sprintf("DATA_PATH=%s", m.config.DataPath),
	)
	m.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start validation chain: %w", err)
	}
	if err := m.waitForHealth(ctx); err != nil {
		_ = syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
		return err
	}

	m.running = true
	return nil
}

// resolveBinary checks standard paths for the validation chain binary.
// Used as a fallback when config.BinaryPath is empty.
func (m *Manager) resolveBinary() (string, error) {
	candidates := []string{
		"/var/lib/knirvserver/bin/validationchain",
		"/usr/local/bin/validationchain",
		"/usr/bin/validationchain",
	}
	// Check if there's a locally-built binary relative to the process
	if exe, err := os.Executable(); err == nil {
		localPath := filepath.Join(filepath.Dir(exe), "validationchain")
		candidates = append([]string{localPath}, candidates...)
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("validation chain binary not found at any standard location")
}

func (m *Manager) waitForHealth(ctx context.Context) error {
	deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for validation chain health check")
		case <-ticker.C:
			resp, err := m.client.Get(m.baseURL + "/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
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
	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
	}

	done := make(chan error, 1)
	go func() { done <- m.cmd.Wait() }()

	select {
	case <-ctx.Done():
		_ = syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
	case <-time.After(m.config.StopTimeout):
		_ = syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
	case <-done:
	}

	m.running = false
	return nil
}

func (m *Manager) GetBaseURL() string        { return m.baseURL }
func (m *Manager) GetConfig() *ManagerConfig { return m.config }
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}
