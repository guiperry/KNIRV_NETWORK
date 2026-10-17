package knirvllama

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func killStaleLlama(binaryPath string) {
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
	startMu    sync.Mutex // serializes Start so mu is never held across a long wait
	running    bool
	listenAddr string
	startTime  time.Time

	exited    chan struct{} // closed once the current child has been reaped (single Wait)
	waitMu    sync.Mutex    // guards waitErr; separate from mu so Stop can wait for exit while holding mu
	waitErr   error
	ring      *tailRing
	phase     string
	attempt   int
	lastError string
}

// killStale is replaced by tests: pkill -x on a test binary's name would kill the test.
var killStale = killStaleLlama

// resolveBinary is replaced by tests that need a binary that cannot be found.
var resolveBinary = resolveLlamaBinaryPath

type ManagerConfig struct {
	BinaryPath string
	ListenAddr string
	// SocketPath optionally exposes the chat API over a Unix domain socket.
	// When set it takes precedence over ListenAddr for the wrapper API.
	SocketPath string
	// LlamaAddress is the private address used by the wrapper to start
	// llama-server itself.
	LlamaAddress string
	DataDir      string
	StartTimeout time.Duration
	StopTimeout  time.Duration
	EnvOverrides map[string]string

	// Deterministic llama-server tunables (Phase A). See
	// llama_cognitive_engine.md §2.1, §4 Phase A.
	// Parallel defaults to 1 in DefaultManagerConfig so the embedded CPU
	// model never sees four concurrent slots competing for context + compute.
	Parallel int
	CtxSize  int
	Threads  int
	// APIKey is the shared-secret token forwarded to llama-server as
	// `--api-key`.  Treat as defense-in-depth only — the wrapper's
	// listen-address already restricts network access and traffic reaches
	// the model over a private Unix socket.
	APIKey string

	// MaxStartAttempts is how many times Start tries when the child exits during
	// startup (default 3). A health timeout or a missing binary is never retried.
	MaxStartAttempts int
	// RetryBackoff is the wait before attempt 2, 3, ... (default 5s, 30s; the last
	// value repeats).
	RetryBackoff []time.Duration
	// SuperviseBackoff is the wait before each restart that Supervise makes, 1st, 2nd,
	// ... (default 10s, 30s, 2m, 5m; the last value repeats).
	SuperviseBackoff []time.Duration
	// StableAfter is how long a child must stay up before Supervise forgets earlier
	// failures and starts the backoff over (default 2m).
	StableAfter time.Duration
	// Logf receives operator-facing progress and failure lines. Nil logs through the
	// standard logger, so a failed start is visible in the server log.
	Logf func(format string, args ...any)
	// Stdout and Stderr receive the child's output, prefixed "[KNIRVLLAMA] ".
	// Nil means the process's own.
	Stdout, Stderr io.Writer
}

type LlamaStatus struct {
	Status     string    `json:"status"`
	Running    bool      `json:"running"`
	ListenAddr string    `json:"listen_addr"`
	Uptime     string    `json:"uptime"`
	Timestamp  time.Time `json:"timestamp"`
	// Phase is starting, waiting, running, failed or stopped; Attempt and LastError
	// say how far a start got and why it last failed.
	Phase     string `json:"phase,omitempty"`
	Attempt   int    `json:"attempt,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

const (
	DefaultListenAddr   = "127.0.0.1:8081"
	DefaultLlamaAddress = "127.0.0.1:8000"
	// DefaultStartTimeout must accommodate a first run. The embedded llama
	// program can clone and build llama.cpp and download a GGUF model before it
	// starts its chat API; its provisioning context permits up to 30 minutes.
	// A short, service-style health timeout would kill that work partway through
	// and make every subsequent launch start again from scratch.
	DefaultStartTimeout = 35 * time.Minute
)

func DefaultManagerConfig() *ManagerConfig {
	appDataDir := getLlamaAppDataDir()
	return &ManagerConfig{
		ListenAddr:   DefaultListenAddr,
		LlamaAddress: DefaultLlamaAddress,
		DataDir:      filepath.Join(appDataDir, "knirvllama"),
		StartTimeout: DefaultStartTimeout,
		StopTimeout:  10 * time.Second,
		Parallel:     1,
	}
}

func getLlamaAppDataDir() string {
	if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
		return explicit
	}
	if err := os.MkdirAll("/var/lib/knirvserver", 0755); err == nil {
		return "/var/lib/knirvserver"
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".local", "share", "knirvserver")
	}
	return filepath.Join(os.TempDir(), "knirvserver", "data")
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg == nil {
		cfg = DefaultManagerConfig()
	}
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = DefaultStartTimeout
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 10 * time.Second
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = DefaultListenAddr
	}
	if cfg.LlamaAddress == "" {
		cfg.LlamaAddress = DefaultLlamaAddress
	}
	if cfg.DataDir == "" {
		defaults := DefaultManagerConfig()
		cfg.DataDir = defaults.DataDir
	}
	if cfg.Parallel == 0 {
		cfg.Parallel = 1
	}
	if cfg.MaxStartAttempts <= 0 {
		cfg.MaxStartAttempts = 3
	}
	if cfg.RetryBackoff == nil {
		cfg.RetryBackoff = []time.Duration{5 * time.Second, 30 * time.Second}
	}
	if cfg.SuperviseBackoff == nil {
		cfg.SuperviseBackoff = []time.Duration{10 * time.Second, 30 * time.Second, 2 * time.Minute, 5 * time.Minute}
	}
	if cfg.StableAfter <= 0 {
		cfg.StableAfter = 2 * time.Minute
	}
	if cfg.APIKey == "" {
		// Operators may supply their own token via env, but a fresh install
		// still needs *something* to put behind --api-key so the CORS-wildcard
		// warning llama-server emits at start-up isn't a defense-in-depth gap.
		if envKey := os.Getenv("KNIRV_LLAMA_API_KEY"); envKey != "" {
			cfg.APIKey = envKey
		} else {
			cfg.APIKey = generateAPIKey()
		}
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Manager{
		config:     cfg,
		logger:     logger,
		listenAddr: cfg.ListenAddr,
	}
}

// APIKey returns the shared-secret token this manager will forward to
// llama-server. Callers wiring LlamaProvider must read this and set it on the
// provider so requests carry `Authorization: Bearer <token>`.
func (m *Manager) APIKey() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config == nil {
		return ""
	}
	return m.config.APIKey
}

// generateAPIKey returns a 32-byte hex token.  Good enough for the
// "defense-in-depth, bind to 127.0.0.1, only reachable via Unix socket"
// posture llama-server has today; not a substitute for end-user auth.
func generateAPIKey() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("knirv-llama-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func (m *Manager) logf(format string, args ...any) {
	if m.config.Logf != nil {
		m.config.Logf(format, args...)
		return
	}
	log.Printf("KNIRVLLAMA: "+format, args...)
}

func (m *Manager) setWaitErr(err error) {
	m.waitMu.Lock()
	m.waitErr = err
	m.waitMu.Unlock()
}

func (m *Manager) getWaitErr() error {
	m.waitMu.Lock()
	defer m.waitMu.Unlock()
	return m.waitErr
}

func (m *Manager) setPhase(phase string, attempt int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.phase, m.attempt = phase, attempt
	if err != nil {
		m.lastError = err.Error()
	}
}

// Start launches KNIRVLLAMA and waits until it is healthy. If the process exits during
// startup it is retried with backoff, because a cold page cache or a busy host often
// kills the first attempt and a second one succeeds. A failure names the exit status
// and the last lines the child printed.
func (m *Manager) Start(ctx context.Context) error {
	m.startMu.Lock()
	defer m.startMu.Unlock()

	m.mu.RLock()
	already := m.running
	m.mu.RUnlock()
	if already {
		m.logger.Info("KNIRVLLAMA already running")
		return nil
	}

	var last error
	for attempt := 1; attempt <= m.config.MaxStartAttempts; attempt++ {
		m.setPhase("starting", attempt, nil)
		err := m.startOnce(ctx, attempt)
		if err == nil {
			m.setPhase("running", attempt, nil)
			m.logf("started on %s (attempt %d)", m.listenAddr, attempt)
			return nil
		}
		last = err
		if ctx.Err() != nil {
			// Stopped on purpose (the harness released it): not a failure to report.
			m.setPhase("stopped", attempt, nil)
			return ctx.Err()
		}
		m.setPhase("failed", attempt, err)
		m.logf("%v", err)
		var se *StartError
		if !errors.As(err, &se) || !se.retryable() || attempt == m.config.MaxStartAttempts || ctx.Err() != nil {
			break
		}
		wait := backoffFor(m.config.RetryBackoff, attempt)
		m.logf("retrying in %s (attempt %d of %d)", wait, attempt+1, m.config.MaxStartAttempts)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return last
}

// Supervise keeps KNIRVLLAMA running until ctx is cancelled. It starts it (Start's own
// quick retries included) and, whenever a start fails or the child later exits, waits
// out a growing backoff and starts it again. A health timeout is retried here too: the
// model may simply have needed longer than the timeout on a cold host. Only a failure
// no retry can fix (missing binary) ends the loop, and then the error is returned.
// Backoff restarts from the beginning once a child has stayed up for StableAfter.
func (m *Manager) Supervise(ctx context.Context) error {
	failures := 0
	for {
		began := time.Now()
		err := m.Start(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			var p permanent
			if errors.As(err, &p) {
				return err
			}
			m.logf("start failed: %v", err)
		} else {
			m.mu.RLock()
			exited := m.exited
			m.mu.RUnlock()
			select {
			case <-exited:
			case <-ctx.Done():
				return ctx.Err()
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		if time.Since(began) >= m.config.StableAfter {
			failures = 0
		}
		failures++
		wait := backoffFor(m.config.SuperviseBackoff, failures)
		m.setPhase("restarting", failures, nil)
		m.logf("restarting in %s (restart %d)", wait, failures)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *Manager) startOnce(ctx context.Context, attempt int) error {
	if m.config.BinaryPath == "" {
		m.config.BinaryPath = "./knirvllama"
	}
	resolved, err := resolveBinary(m.config.BinaryPath)
	if err != nil {
		return permanent{fmt.Errorf("KNIRVLLAMA binary not found: %w", err)}
	}
	m.config.BinaryPath = resolved

	killStale(m.config.BinaryPath)

	if err := os.MkdirAll(m.config.DataDir, 0755); err != nil {
		return permanent{fmt.Errorf("failed to create data directory: %w", err)}
	}
	if m.config.SocketPath != "" {
		if err := os.MkdirAll(filepath.Dir(m.config.SocketPath), 0750); err != nil {
			return permanent{fmt.Errorf("failed to create llama socket directory: %w", err)}
		}
		// A socket file left by a previous run (even one from before a reboot) makes
		// "the socket exists" look like "llama is up". Remove it before spawning.
		if err := removeStaleSocket(m.config.SocketPath); err != nil {
			return permanent{fmt.Errorf("llama socket: %w", err)}
		}
	}

	m.logger.Info("Starting KNIRVLLAMA",
		zap.String("binary", m.config.BinaryPath),
		zap.String("listen", m.listenAddr),
		zap.String("data_dir", m.config.DataDir))

	env := os.Environ()
	env = append(env,
		fmt.Sprintf("KNIRV_APP_DATA_DIR=%s", getLlamaAppDataDir()),
	)

	for k, v := range m.config.EnvOverrides {
		env = setEnvOverride(env, k, v)
	}

	args := []string{
		"-listen", m.listenAddr,
		"-llama-address", m.config.LlamaAddress,
		"-data-dir", m.config.DataDir,
		"-parallel", strconv.Itoa(m.config.Parallel),
	}
	if m.config.CtxSize > 0 {
		args = append(args, "-ctx-size", strconv.Itoa(m.config.CtxSize))
	}
	if m.config.Threads > 0 {
		args = append(args, "-threads", strconv.Itoa(m.config.Threads))
	}
	if m.config.APIKey != "" {
		args = append(args, "-api-key", m.config.APIKey)
	}
	if m.config.SocketPath != "" {
		args = append(args, "-unix-socket", m.config.SocketPath)
	}

	stdout, stderr := m.config.Stdout, m.config.Stderr
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	ring := newTailRing(40)
	outW := &lineWriter{prefix: "[KNIRVLLAMA] ", out: stdout, ring: ring}
	errW := &lineWriter{prefix: "[KNIRVLLAMA] ", out: stderr, ring: ring}

	cmd := exec.Command(m.config.BinaryPath, args...)
	cmd.Env = env
	cmd.Stdout = outW
	cmd.Stderr = errW
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return &StartError{Attempt: attempt, Underlay: fmt.Errorf("failed to start KNIRVLLAMA: %w", err)}
	}
	exited := make(chan struct{})
	started := time.Now()

	m.mu.Lock()
	m.cmd, m.exited, m.ring = cmd, exited, ring
	m.startTime = started
	m.mu.Unlock()
	m.setWaitErr(nil)

	// One goroutine owns Wait for the life of the child, so a crash is reaped at once
	// (no zombie) and both startup and Stop learn about it from the same channel.
	go func() {
		err := cmd.Wait()
		outW.Flush()
		errW.Flush()
		m.setWaitErr(err)
		close(exited) // before taking mu: Stop holds mu while waiting on this channel
		m.mu.Lock()
		wasRunning := m.running
		m.running = false
		m.mu.Unlock()
		if wasRunning {
			m.setPhase("failed", attempt, fmt.Errorf("KNIRVLLAMA exited after start: %v", err))
			m.logf("exited unexpectedly: %v; last output: %s", err, strings.Join(ring.tail(5), " | "))
		}
	}()
	m.logger.Info("KNIRVLLAMA subprocess started", zap.Int("pid", cmd.Process.Pid))
	m.logf("subprocess started (pid %d, attempt %d); waiting for the model to load", cmd.Process.Pid, attempt)

	if err := m.waitForHealth(ctx, exited); err != nil {
		se := &StartError{Attempt: attempt, Elapsed: time.Since(started), Tail: ring.tail(8)}
		select {
		case <-exited:
			se.Exited = true
			se.ExitErr = m.getWaitErr()
		default:
			se.Underlay = err
			se.Timeout = errors.Is(err, errHealthTimeout)
			// Still alive but not healthy (timeout or cancel): kill the group and reap.
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			select {
			case <-exited:
			case <-time.After(2 * time.Second):
			}
		}
		return se
	}

	m.mu.Lock()
	m.running = true
	m.mu.Unlock()
	return nil
}

var errHealthTimeout = errors.New("timeout waiting for KNIRVLLAMA health")

func setEnvOverride(env []string, key, value string) []string {
	prefix := key + "="
	filtered := env[:0]
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered, fmt.Sprintf("%s=%s", key, value))
}

func resolveLlamaBinaryPath(configured string) (string, error) {
	var candidates []string

	if envPath := os.Getenv("KNIRV_LLAMA_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvllama"),
			filepath.Join(exeDir, "bin", "knirvllama"),
		)
	}

	embeddedDir := filepath.Join(getLlamaAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "knirvllama"),
		filepath.Join(embeddedDir, "..", "bin", "knirvllama"),
	)

	candidates = append(candidates, configured)

	dir, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(dir, "bin", "knirvllama"),
		filepath.Join(dir, "..", "bin", "knirvllama"),
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("binary not found in any candidate location")
}

// waitForHealth returns when KNIRVLLAMA answers /health, the child exits (reported at
// once, not after the deadline), the deadline passes, or ctx is cancelled. It logs
// progress so a slow model load is distinguishable from a hang.
func (m *Manager) waitForHealth(ctx context.Context, exited <-chan struct{}) error {
	deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
	defer cancel()

	started := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	progress := time.NewTicker(time.Minute)
	defer progress.Stop()

	for {
		select {
		case <-exited:
			return fmt.Errorf("KNIRVLLAMA process exited before becoming healthy")
		case <-deadline.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%w at %s", errHealthTimeout, m.listenAddr)
		case <-progress.C:
			m.mu.RLock()
			attempt := m.attempt
			m.mu.RUnlock()
			m.setPhase("waiting", attempt, nil)
			m.logf("still starting, %s elapsed (the model may be loading from a cold disk cache)", time.Since(started).Round(time.Second))
		case <-ticker.C:
			if healthy, _ := m.GetHealth(); healthy {
				m.logger.Debug("KNIRVLLAMA health check passed", zap.String("listen", m.listenAddr))
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

	m.logger.Info("Stopping KNIRVLLAMA")

	cmd, exited := m.cmd, m.exited
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Warn("Failed to send SIGTERM, forcing kill",
			zap.Error(err))
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	select {
	case <-exited:
		m.running = false
		m.startTime = time.Time{}
		m.phase = "stopped"
		if err := m.getWaitErr(); err != nil {
			m.logger.Info("KNIRVLLAMA stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVLLAMA stopped gracefully")
		}
		return nil
	case <-time.After(m.config.StopTimeout):
		m.logger.Warn("Timeout waiting for graceful shutdown, forcing kill")
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		select {
		case <-exited:
		case <-time.After(2 * time.Second):
			m.logger.Warn("Wait() did not complete after Kill() — zombie possible")
		}
		m.running = false
		m.startTime = time.Time{}
		m.phase = "stopped"
		return fmt.Errorf("forced shutdown after timeout")
	}
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) GetListenAddr() string {
	return m.listenAddr
}

// GetSocketPath returns the Unix socket used by the chat API, if configured.
func (m *Manager) GetSocketPath() string {
	return m.config.SocketPath
}

func (m *Manager) GetConfig() *ManagerConfig {
	return m.config
}

func (m *Manager) GetPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

func (m *Manager) GetStatus() *LlamaStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := &LlamaStatus{
		Status:     "stopped",
		Running:    m.running,
		ListenAddr: m.listenAddr,
		Timestamp:  time.Now(),
		Phase:      m.phase,
		Attempt:    m.attempt,
		LastError:  m.lastError,
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

func (m *Manager) GetHealth() (bool, error) {
	socketPath := m.config.SocketPath
	listenAddr := m.listenAddr
	m.mu.RLock()
	cmd := m.cmd
	m.mu.RUnlock()
	if cmd == nil {
		return false, fmt.Errorf("KNIRVLLAMA not running")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	endpoint := "http://" + listenAddr + "/health"
	if socketPath != "" {
		transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}}
		defer transport.CloseIdleConnections()
		client.Transport = transport
		endpoint = "http://localhost/health"
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
