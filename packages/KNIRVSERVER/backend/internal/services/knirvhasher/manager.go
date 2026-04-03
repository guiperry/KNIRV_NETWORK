package knirvhasher

import (
	"context"
	"fmt"
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
	BinaryPath   string
	SocketPath   string
	DataPath     string
	StartTimeout time.Duration
	StopTimeout  time.Duration
	Stdout       interface{}
	Stderr       interface{}
}

type HasherStatus struct {
	Status     string    `json:"status"`
	Running    bool      `json:"running"`
	SocketPath string    `json:"socket_path"`
	Uptime     string    `json:"uptime"`
	Timestamp  time.Time `json:"timestamp"`
}

func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		SocketPath:   "/var/run/hasher.sock",
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
	}
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 30 * time.Second
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 10 * time.Second
	}
	if cfg.SocketPath == "" {
		cfg.SocketPath = "/var/run/hasher.sock"
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

	m.cmd = exec.Command(m.config.BinaryPath)
	m.cmd.Env = env
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr
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

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvhasher"),
			filepath.Join(exeDir, "bin", "knirvhasher"),
		)
	}

	embeddedDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "knirvserver", "bin")
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
