package knirvllama

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	running    bool
	listenAddr string
	startTime  time.Time
}

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
}

type LlamaStatus struct {
	Status     string    `json:"status"`
	Running    bool      `json:"running"`
	ListenAddr string    `json:"listen_addr"`
	Uptime     string    `json:"uptime"`
	Timestamp  time.Time `json:"timestamp"`
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

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVLLAMA already running")
		return nil
	}

	if m.config.BinaryPath == "" {
		m.config.BinaryPath = "./knirvllama"
	}

	resolved, err := resolveLlamaBinaryPath(m.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("KNIRVLLAMA binary not found: %w", err)
	}
	m.config.BinaryPath = resolved

	killStaleLlama(m.config.BinaryPath)

	if err := os.MkdirAll(m.config.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	if m.config.SocketPath != "" {
		if err := os.MkdirAll(filepath.Dir(m.config.SocketPath), 0750); err != nil {
			return fmt.Errorf("failed to create llama socket directory: %w", err)
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

	m.cmd = exec.Command(m.config.BinaryPath, args...)
	m.cmd.Env = env
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start KNIRVLLAMA: %w", err)
	}
	m.startTime = time.Now()
	m.logger.Info("KNIRVLLAMA subprocess started",
		zap.Int("pid", m.cmd.Process.Pid))

	if err := m.waitForHealth(ctx); err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("health check failed: %w", err)
	}

	m.running = true
	m.logger.Info("KNIRVLLAMA started successfully",
		zap.String("listen", m.listenAddr))

	return nil
}

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

func (m *Manager) waitForHealth(ctx context.Context) error {
	deadline, cancel := context.WithTimeout(ctx, m.config.StartTimeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for KNIRVLLAMA health at %s", m.listenAddr)
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

	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.Warn("Failed to send SIGTERM, forcing kill",
			zap.Error(err))
		syscall.Kill(-m.cmd.Process.Pid, syscall.SIGKILL)
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
			m.logger.Info("KNIRVLLAMA stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVLLAMA stopped gracefully")
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
		m.startTime = time.Time{}
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
	cmd := m.cmd
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
