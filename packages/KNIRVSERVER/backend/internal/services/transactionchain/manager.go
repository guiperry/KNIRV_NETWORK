package transactionchain

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
	ScriptPath   string
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
			BinaryPath:   "node",
			Port:         9190,
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

	binary := m.config.BinaryPath
	if binary == "" {
		binary = "node"
	}
	script := m.config.ScriptPath
	if script == "" {
		return fmt.Errorf("transaction chain script path not configured")
	}
	if !filepath.IsAbs(script) && m.config.WorkDir != "" {
		script = filepath.Join(m.config.WorkDir, script)
	}

	m.cmd = exec.Command(binary, script)
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
		fmt.Sprintf("PORT=%d", m.config.Port),
		fmt.Sprintf("DATA_PATH=%s", m.config.DataPath),
		fmt.Sprintf("CHAIN_ID=%s", m.config.ChainID),
	)
	m.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start transaction chain: %w", err)
	}
	if err := m.waitForHealth(ctx); err != nil {
		_ = m.cmd.Process.Kill()
		return err
	}

	m.running = true
	return nil
}

func (m *Manager) waitForHealth(ctx context.Context) error {
	deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for transaction chain health check")
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
		_ = m.cmd.Process.Kill()
	}

	done := make(chan error, 1)
	go func() { done <- m.cmd.Wait() }()

	select {
	case <-ctx.Done():
		_ = m.cmd.Process.Kill()
	case <-time.After(m.config.StopTimeout):
		_ = m.cmd.Process.Kill()
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
