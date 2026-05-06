package knirvagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// killStaleAgent sends SIGTERM (then SIGKILL after 2 s) to any running
// processes whose executable matches the given binary name. This prevents
// port-conflict crashes when the parent process was previously killed
// without triggering the normal Stop() path.
func killStaleAgent(binaryPath string) {
	if binaryPath == "" {
		return
	}
	binaryName := filepath.Base(binaryPath)
	// SIGTERM first
	if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
		time.Sleep(1 * time.Second)
	}
	// SIGKILL any survivors
	exec.Command("pkill", "-KILL", "-x", binaryName).Run() //nolint:errcheck
}

// waitForPortFree blocks until the given TCP port is no longer occupied or
// the deadline is reached. Returns true if the port is free.
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
	BinaryPath     string
	SocketPath     string
	Port           int
	BackendAPIPort int
	StartTimeout   time.Duration
	StopTimeout    time.Duration
	Stdout         io.Writer
	Stderr         io.Writer
	// ExtraEnv are additional environment variables propagated to the
	// KNIRVAGENT subprocess, e.g. "GEMINI_API_KEY=..." or "OPENAI_API_KEY=...".
	ExtraEnv []string
}

type HealthStatus struct {
	Status    string                 `json:"status"`
	Services  map[string]interface{} `json:"services"`
	Timestamp time.Time              `json:"timestamp"`
}

func getAgentAppDataDir() string {
	if explicit := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); explicit != "" {
		return explicit
	}
	if err := os.MkdirAll("/var/lib/knirvserver", 0755); err == nil {
		return "/var/lib/knirvserver"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "knirvserver")
	}
	return "data"
}

func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		SocketPath:   "",
		Port:         8080,
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

	var baseURL string
	if cfg.SocketPath != "" {
		baseURL = "http://localhost"
	} else {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Port)
		if cfg.Port == 0 {
			baseURL = "http://localhost:8080"
		}
	}

	m := &Manager{
		config:  cfg,
		logger:  logger,
		baseURL: baseURL,
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

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVAGENT already running")
		return nil
	}

	if m.config.BinaryPath == "" {
		m.config.BinaryPath = "./knirvagent"
	}

	// Resolve the binary path, trying several candidate locations.
	resolved, err := resolveBinaryPath(m.config.BinaryPath)
	if err != nil {
		return fmt.Errorf("KNIRVAGENT binary not found (tried %s and fallbacks): %w", m.config.BinaryPath, err)
	}
	m.config.BinaryPath = resolved

	// Pre-flight: if a healthy agent is already listening, adopt it rather than
	// starting a duplicate and hitting a port conflict.
	preflightClient := &http.Client{
		Timeout:   3 * time.Second,
		Transport: m.client.Transport,
	}
	if resp, err := preflightClient.Get(fmt.Sprintf("%s/health", m.baseURL)); err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			m.logger.Info("Adopting existing healthy KNIRVAGENT process (skipping new spawn)")
			m.running = true
			return nil
		}
	}

	// Existing process is unhealthy or absent — kill any stale instance so the
	// ports are free before we start.
	m.logger.Info("Killing any stale KNIRVAGENT processes", zap.String("binary", m.config.BinaryPath))
	killStaleAgent(m.config.BinaryPath)

	// Wait for the port to be released before spawning the new binary.
	gatewayPort := m.config.Port
	if gatewayPort <= 0 {
		gatewayPort = 8080
	}
	if !waitForPortFree(gatewayPort, 5*time.Second) {
		m.logger.Warn("Gateway port still occupied after kill — proceeding anyway",
			zap.Int("port", gatewayPort))
	}

	m.logger.Info("Starting KNIRVAGENT",
		zap.String("binary", m.config.BinaryPath),
		zap.String("socketPath", m.config.SocketPath))

	env := os.Environ()
	backendPort := m.config.BackendAPIPort
	if backendPort <= 0 {
		backendPort = 9081
	}
	env = append(env,
		fmt.Sprintf("KNIRV_BACKEND_API_URL=http://127.0.0.1:%d", backendPort),
	)

	if m.config.SocketPath != "" {
		env = append(env, fmt.Sprintf("SOCKET_PATH=%s", m.config.SocketPath))
	} else {
		env = append(env, fmt.Sprintf("PORT=%d", gatewayPort))
	}

	// Propagate any extra environment variables (e.g. API keys from root.key).
	env = append(env, m.config.ExtraEnv...)

	args := []string{"gateway"}
	if m.config.SocketPath != "" {
		args = append(args, "-socket", m.config.SocketPath)
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
		return fmt.Errorf("failed to start KNIRVAGENT: %w", err)
	}

	m.logger.Info("KNIRVAGENT subprocess started",
		zap.Int("pid", m.cmd.Process.Pid))

	if err := m.waitForHealth(ctx); err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("health check failed: %w", err)
	}

	m.running = true
	m.logger.Info("KNIRVAGENT started successfully",
		zap.String("url", m.baseURL))

	return nil
}

// resolveBinaryPath returns the first path where the knirvagent binary exists.
// Priority: KNIRV_AGENT_BINARY_PATH env var → configured path → standard fallbacks.
func resolveBinaryPath(configured string) (string, error) {
	var candidates []string

	// Highest priority: explicit env var set by the launcher (main.go).
	if envPath := os.Getenv("KNIRV_AGENT_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	// Try executable directory first for embedded binaries
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvagent"),
			filepath.Join(exeDir, "bin", "knirvagent"),
		)
	}

	// Embedded directory (extracted binaries)
	embeddedDir := filepath.Join(getAgentAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "knirvagent"),
		filepath.Join(embeddedDir, "..", "bin", "knirvagent"),
	)

	candidates = append(candidates, configured)

	// Common fallback locations relative to the working directory.
	dir, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(dir, "bin", "knirvagent"),
		filepath.Join(dir, "..", "bin", "knirvagent"),
		filepath.Join(filepath.Dir(configured), "bin", "knirvagent"),
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

	healthURL := fmt.Sprintf("%s/health", m.baseURL)

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for KNIRVAGENT health check")
		case <-ticker.C:
			resp, err := m.client.Get(healthURL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					m.logger.Debug("KNIRVAGENT health check passed")
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

	m.logger.Info("Stopping KNIRVAGENT")

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
		if err != nil {
			m.logger.Info("KNIRVAGENT stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVAGENT stopped gracefully")
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
