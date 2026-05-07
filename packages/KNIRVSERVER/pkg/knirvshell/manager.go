package knirvshell

import (
	"context"
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

// Manager manages the KNIRVSHELL binary subprocess lifecycle.
// It spawns the knirvshell binary as a daemon listening on a Unix socket
// and provides Start/Stop/HealthCheck methods, following the same pattern
// as knirvagent.Manager and knirvgraph.Manager.
type Manager struct {
	binaryPath string
	config     *ManagerConfig
	cmd        *exec.Cmd
	logger     *zap.Logger
	client     *http.Client
	mu         sync.RWMutex
	running    bool
}

// ManagerConfig configures the KNIRVSHELL Manager.
type ManagerConfig struct {
	BinaryPath   string
	SocketPath   string
	StartTimeout time.Duration
	StopTimeout  time.Duration
	Stdout       io.Writer
	Stderr       io.Writer
	ExtraEnv     []string
}

// DefaultManagerConfig returns a ManagerConfig with sensible defaults.
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		SocketPath:   "",
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
	}
}

// NewManager creates a new KNIRVSHELL Manager.
func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 30 * time.Second
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 10 * time.Second
	}

	m := &Manager{
		config: cfg,
		logger: logger,
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

// Start launches the KNIRVSHELL binary as a subprocess.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVSHELL already running")
		return nil
	}

	if m.config.BinaryPath == "" {
		// Try to resolve the binary path using existing utilities
		resolved, err := GetBinaryPath()
		if err != nil {
			return fmt.Errorf("KNIRVSHELL binary not found: %w", err)
		}
		m.config.BinaryPath = resolved
	}

	// Pre-flight: if a healthy shell is already listening, adopt it
	if m.config.SocketPath != "" {
		preflightClient := &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", m.config.SocketPath)
				},
			},
		}
		if resp, err := preflightClient.Get("http://localhost/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				m.logger.Info("Adopting existing healthy KNIRVSHELL process (skipping new spawn)")
				m.running = true
				return nil
			}
		}
	}

	// Kill any stale instance so the socket/port is free
	m.logger.Info("Killing any stale KNIRVSHELL processes", zap.String("binary", m.config.BinaryPath))
	killStaleShell(m.config.BinaryPath)

	// Clean up stale socket file if it exists
	if m.config.SocketPath != "" {
		os.Remove(m.config.SocketPath)
	}

	m.logger.Info("Starting KNIRVSHELL",
		zap.String("binary", m.config.BinaryPath),
		zap.String("socketPath", m.config.SocketPath))

	env := os.Environ()
	env = append(env, m.config.ExtraEnv...)

	args := []string{"server"}
	if m.config.SocketPath != "" {
		args = append(args, "--socket-path", m.config.SocketPath)
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
		return fmt.Errorf("failed to start KNIRVSHELL: %w", err)
	}

	m.logger.Info("KNIRVSHELL subprocess started",
		zap.Int("pid", m.cmd.Process.Pid))

	if err := m.waitForHealth(ctx); err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("health check failed: %w", err)
	}

	m.running = true
	m.logger.Info("KNIRVSHELL started successfully",
		zap.String("socket", m.config.SocketPath))

	return nil
}

// waitForHealth blocks until the shell socket responds to a health check
// or the timeout expires.
func (m *Manager) waitForHealth(ctx context.Context) error {
	deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for KNIRVSHELL health check")
		case <-ticker.C:
			resp, err := m.client.Get("http://localhost/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					m.logger.Debug("KNIRVSHELL health check passed")
					return nil
				}
			}
		}
	}
}

// Stop gracefully stops the KNIRVSHELL subprocess.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}

	m.logger.Info("Stopping KNIRVSHELL")

	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Warn("Failed to send SIGTERM, forcing kill", zap.Error(err))
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
			m.logger.Info("KNIRVSHELL stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVSHELL stopped gracefully")
		}
		// Clean up socket file
		if m.config.SocketPath != "" {
			os.Remove(m.config.SocketPath)
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
		if m.config.SocketPath != "" {
			os.Remove(m.config.SocketPath)
		}
		return fmt.Errorf("forced shutdown after timeout")
	}
}

// IsRunning returns whether the shell subprocess is running.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// HealthCheck performs a health check against the shell socket.
func (m *Manager) HealthCheck(ctx context.Context) error {
	resp, err := m.client.Get("http://localhost/health")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// GetPID returns the subprocess PID, or 0 if not running.
func (m *Manager) GetPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

// killStaleShell sends SIGTERM (then SIGKILL) to any running processes
// whose executable matches the given binary name.
func killStaleShell(binaryPath string) {
	if binaryPath == "" {
		return
	}
	binaryName := filepath.Base(binaryPath)
	if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
		time.Sleep(1 * time.Second)
	}
	exec.Command("pkill", "-KILL", "-x", binaryName).Run() //nolint:errcheck
}
