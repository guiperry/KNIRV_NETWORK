package knirvulora

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ManagerConfig configures the ulorad subprocess.
type ManagerConfig struct {
	// BinaryPath is the extracted ulorad (see ExtractEmbeddedBinary).
	BinaryPath string
	// SocketPath is the unix socket ulorad serves on (ULORA_SOCKET_PATH).
	SocketPath string
	// DataDir is ULORA_DATA_DIR.
	DataDir string
	// AuthToken is the shared secret callers present (ULORA_AUTH_TOKEN).
	AuthToken string
	// EnvOverrides are extra KEY=VALUE environment entries.
	EnvOverrides map[string]string
	StartTimeout time.Duration
	StopTimeout  time.Duration
}

// DefaultStartTimeout and DefaultStopTimeout apply when the config leaves them zero.
const (
	DefaultStartTimeout = 30 * time.Second
	DefaultStopTimeout  = 10 * time.Second
)

// Manager owns the ulorad subprocess.
type Manager struct {
	cfg     ManagerConfig
	mu      sync.Mutex
	cmd     *exec.Cmd
	done    chan struct{} // closed when cmd has exited and been reaped
	running bool
}

// NewManager applies defaults to cfg.
func NewManager(cfg ManagerConfig) *Manager {
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = DefaultStartTimeout
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = DefaultStopTimeout
	}
	return &Manager{cfg: cfg}
}

// killStale ends an ulorad left behind by a previous server that was killed
// uncleanly, so the new one can bind its socket.
func killStale(binaryPath string) {
	if binaryPath == "" {
		return
	}
	name := filepath.Base(binaryPath)
	if exec.Command("pkill", "-TERM", "-x", name).Run() == nil {
		time.Sleep(500 * time.Millisecond)
	}
	_ = exec.Command("pkill", "-KILL", "-x", name).Run()
}

// Start launches ulorad in its own process group and returns once its socket
// accepts connections.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return nil
	}
	if m.cfg.BinaryPath == "" {
		return fmt.Errorf("ulorad binary path is not set")
	}
	if _, err := os.Stat(m.cfg.BinaryPath); err != nil {
		return fmt.Errorf("ulorad binary not found: %w", err)
	}
	if m.cfg.SocketPath == "" || m.cfg.DataDir == "" {
		return fmt.Errorf("ulorad needs a socket path and a data directory")
	}
	killStale(m.cfg.BinaryPath)
	if err := os.MkdirAll(filepath.Dir(m.cfg.SocketPath), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(m.cfg.DataDir, 0755); err != nil {
		return err
	}
	_ = os.Remove(m.cfg.SocketPath)

	env := os.Environ()
	set := func(k, v string) { env = setEnv(env, k, v) }
	set("ULORA_SOCKET_PATH", m.cfg.SocketPath)
	set("ULORA_DATA_DIR", m.cfg.DataDir)
	if m.cfg.AuthToken != "" {
		set("ULORA_AUTH_TOKEN", m.cfg.AuthToken)
	}
	for k, v := range m.cfg.EnvOverrides {
		set(k, v)
	}

	cmd := exec.Command(m.cfg.BinaryPath)
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	// Own process group: a terminal Ctrl+C then reaches only the launcher, which
	// stops ulorad through Stop, and Stop can signal the whole group.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ulorad: %w", err)
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	m.cmd, m.done = cmd, done
	log.Printf("ulorad started (PID: %d)", cmd.Process.Pid)

	if err := m.waitForSocket(ctx, done); err != nil {
		m.killGroup(syscall.SIGKILL)
		<-done
		m.cmd, m.done = nil, nil
		return fmt.Errorf("ulorad did not become ready: %w", err)
	}
	m.running = true
	return nil
}

func (m *Manager) waitForSocket(ctx context.Context, exited <-chan struct{}) error {
	deadline := time.Now().Add(m.cfg.StartTimeout)
	for {
		if c, err := net.DialTimeout("unix", m.cfg.SocketPath, 500*time.Millisecond); err == nil {
			_ = c.Close()
			return nil
		}
		select {
		case <-exited:
			return fmt.Errorf("process exited during startup")
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("socket %s not accepting connections after %s", m.cfg.SocketPath, m.cfg.StartTimeout)
		}
	}
}

// killGroup signals ulorad's whole process group (it may run Python engines),
// falling back to the process alone.
func (m *Manager) killGroup(sig syscall.Signal) {
	if m.cmd == nil || m.cmd.Process == nil {
		return
	}
	if err := syscall.Kill(-m.cmd.Process.Pid, sig); err != nil {
		_ = m.cmd.Process.Signal(sig)
	}
}

// Stop sends SIGTERM to ulorad's process group, waits StopTimeout for it to exit
// and then SIGKILLs it. It is safe to call when ulorad never started.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	pid := m.cmd.Process.Pid
	m.killGroup(syscall.SIGTERM)
	select {
	case <-m.done:
		log.Printf("ulorad (PID %d) stopped", pid)
	case <-time.After(m.cfg.StopTimeout):
		log.Printf("ulorad (PID %d) did not stop within %s, sending SIGKILL", pid, m.cfg.StopTimeout)
		m.killGroup(syscall.SIGKILL)
		select {
		case <-m.done:
		case <-time.After(2 * time.Second):
			return fmt.Errorf("ulorad (PID %d) survived SIGKILL", pid)
		}
	}
	_ = os.Remove(m.cfg.SocketPath)
	m.cmd, m.done, m.running = nil, nil, false
	return nil
}

// IsRunning reports whether ulorad has been started and has not exited.
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running || m.done == nil {
		return false
	}
	select {
	case <-m.done:
		return false
	default:
		return true
	}
}

// PID returns ulorad's process id, or 0 when it is not running.
func (m *Manager) PID() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || m.cmd.Process == nil {
		return 0
	}
	return m.cmd.Process.Pid
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := env[:0:0]
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return append(out, prefix+value)
}
