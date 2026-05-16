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

// ──────────────────────────────────────────────
// Utility helpers (preserved from original)
// ──────────────────────────────────────────────

// killStaleAgent sends SIGTERM (then SIGKILL after 2 s) to any running
// processes whose executable matches the given binary name.
func killStaleAgent(binaryPath string) {
	if binaryPath == "" {
		return
	}
	binaryName := filepath.Base(binaryPath)
	if cmd := exec.Command("pkill", "-TERM", "-x", binaryName); cmd.Run() == nil {
		time.Sleep(1 * time.Second)
	}
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

// resolveBinaryPath returns the first path where the knirvagent binary exists.
func resolveBinaryPath(configured string) (string, error) {
	var candidates []string

	if envPath := os.Getenv("KNIRV_AGENT_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvagent"),
			filepath.Join(exeDir, "bin", "knirvagent"),
		)
	}

	embeddedDir := filepath.Join(getAgentAppDataDir(), "bin")
	candidates = append(candidates,
		filepath.Join(embeddedDir, "knirvagent"),
		filepath.Join(embeddedDir, "..", "bin", "knirvagent"),
	)

	candidates = append(candidates, configured)

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

// waitForSocket blocks until the given Unix socket file exists or the
// deadline is reached. Returns true when the socket is ready.
func waitForSocket(socketPath string, deadline time.Duration) bool {
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		if _, err := os.Stat(socketPath); err == nil {
			// Socket file exists — verify it's actually listening
			conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// waitForTCPPort blocks until the given TCP port is accepting connections or
// the deadline is reached. Returns true when the port is ready.
func waitForTCPPort(port int, deadline time.Duration) bool {
	end := time.Now().Add(deadline)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(end) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// ──────────────────────────────────────────────
// AgentProcess — a single agent subprocess
// ──────────────────────────────────────────────

// AgentProcess represents a single KNIRVAGENT subprocess for one DVE.
type AgentProcess struct {
	DVEID      string
	Cmd        *exec.Cmd
	SocketPath string
	PID        int
	StartedAt  time.Time
	Healthy    bool
	mu         sync.RWMutex
}

func (ap *AgentProcess) setHealthy(h bool) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.Healthy = h
}

func (ap *AgentProcess) isHealthy() bool {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.Healthy
}

// ──────────────────────────────────────────────
// AgentManager — orchestrates N concurrent agents
// ──────────────────────────────────────────────

// AgentManager manages multiple KNIRVAGENT subprocesses, one per DVE.
type AgentManager struct {
	mu              sync.RWMutex
	agents          map[string]*AgentProcess // keyed by DVEID
	socketDir       string
	binaryPath      string
	maxConcurrent   int
	logger          *zap.Logger
	stopHealthCheck context.CancelFunc
	healthInterval  time.Duration
	wg              sync.WaitGroup
	extraEnv        []string // extra environment variables for subprocesses
}

// AgentManagerConfig configures the AgentManager.
type AgentManagerConfig struct {
	BinaryPath      string
	SocketDir       string
	MaxConcurrent   int
	StartTimeout    time.Duration
	StopTimeout     time.Duration
	HealthInterval  time.Duration
	ExtraEnv        []string
}

// HealthStatus represents the health status of an agent.
type HealthStatus struct {
	Status    string                 `json:"status"`
	Services  map[string]interface{} `json:"services"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewAgentManager creates a new AgentManager.
func NewAgentManager(cfg *AgentManagerConfig, logger *zap.Logger) *AgentManager {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 32
	}
	if cfg.HealthInterval <= 0 {
		cfg.HealthInterval = 30 * time.Second
	}

	return &AgentManager{
		agents:         make(map[string]*AgentProcess),
		socketDir:      cfg.SocketDir,
		binaryPath:     cfg.BinaryPath,
		maxConcurrent:  cfg.MaxConcurrent,
		logger:         logger,
		healthInterval: cfg.HealthInterval,
		extraEnv:       cfg.ExtraEnv,
	}
}

// StartAgent spawns a new KNIRVAGENT subprocess for the given DVE ID.
// It blocks until the agent's socket is ready (up to StartTimeout).
func (am *AgentManager) StartAgent(ctx context.Context, dveID string, startTimeout time.Duration) (*AgentProcess, error) {
	am.mu.Lock()

	// Check if already running
	if existing, ok := am.agents[dveID]; ok && existing.isHealthy() {
		am.mu.Unlock()
		am.logger.Info("Agent already running for DVE", zap.String("dveID", dveID))
		return existing, nil
	}

	// Check max concurrent limit
	if len(am.agents) >= am.maxConcurrent {
		am.mu.Unlock()
		return nil, fmt.Errorf("max concurrent agents reached (%d/%d)",
			len(am.agents), am.maxConcurrent)
	}

	resolvedPath := am.binaryPath
	if resolvedPath == "" {
		var err error
		resolvedPath, err = resolveBinaryPath("./knirvagent")
		if err != nil {
			am.mu.Unlock()
			return nil, fmt.Errorf("KNIRVAGENT binary not found: %w", err)
		}
	}

	if am.socketDir != "" {
		if err := os.MkdirAll(am.socketDir, 0755); err != nil {
			am.mu.Unlock()
			return nil, fmt.Errorf("failed to create socket dir %s: %w", am.socketDir, err)
		}
	}

	socketPath := filepath.Join(am.socketDir, fmt.Sprintf("agent-%s.sock", dveID))

	// Kill any stale instance for this DVE
	killStaleAgent(resolvedPath)
	os.Remove(socketPath)

	am.logger.Info("Starting KNIRVAGENT for DVE",
		zap.String("dveID", dveID),
		zap.String("socket", socketPath),
		zap.String("binary", resolvedPath))

	// Build command — the agent runs the `server` subcommand which listens
	// on a Unix socket (agent-{dveID}.sock) and exposes POST /api/execute.
	args := []string{
		"server",
		"--dve-id", dveID,
		"--socket-path", socketPath,
	}

	cmd := exec.Command(resolvedPath, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, am.extraEnv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		am.mu.Unlock()
		return nil, fmt.Errorf("failed to start KNIRVAGENT for DVE %s: %w", dveID, err)
	}

	pid := cmd.Process.Pid

	ap := &AgentProcess{
		DVEID:      dveID,
		Cmd:        cmd,
		SocketPath: socketPath,
		PID:        pid,
		StartedAt:  time.Now(),
		Healthy:    false,
	}

	am.agents[dveID] = ap
	am.mu.Unlock()

	// Wait for the socket to appear (outside the lock)
	if startTimeout <= 0 {
		startTimeout = 30 * time.Second
	}
	if !waitForSocket(socketPath, startTimeout) {
		// Socket didn't appear — kill the process group
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		am.mu.Lock()
		delete(am.agents, dveID)
		am.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for agent socket %s for DVE %s", socketPath, dveID)
	}

	ap.setHealthy(true)

	am.logger.Info("KNIRVAGENT started for DVE",
		zap.String("dveID", dveID),
		zap.Int("pid", pid),
		zap.String("socket", socketPath))

	return ap, nil
}

// StopAgent stops the agent subprocess for the given DVE ID.
func (am *AgentManager) StopAgent(dveID string, stopTimeout time.Duration) error {
	am.mu.Lock()
	ap, ok := am.agents[dveID]
	if !ok {
		am.mu.Unlock()
		return nil // idempotent — already stopped
	}
	delete(am.agents, dveID)
	am.mu.Unlock()

	if ap.Cmd == nil || ap.Cmd.Process == nil {
		os.Remove(ap.SocketPath)
		return nil
	}

	am.logger.Info("Stopping KNIRVAGENT for DVE",
		zap.String("dveID", dveID),
		zap.Int("pid", ap.PID))

	// Send SIGTERM
	if err := ap.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		am.logger.Warn("Failed to send SIGTERM to agent, forcing kill",
			zap.String("dveID", dveID),
			zap.Error(err))
		syscall.Kill(-ap.Cmd.Process.Pid, syscall.SIGKILL)
	}

	done := make(chan error, 1)
	go func() {
		done <- ap.Cmd.Wait()
	}()

	if stopTimeout <= 0 {
		stopTimeout = 10 * time.Second
	}

	select {
	case <-done:
		ap.setHealthy(false)
		os.Remove(ap.SocketPath)
		am.logger.Info("KNIRVAGENT stopped for DVE", zap.String("dveID", dveID))
		return nil
	case <-time.After(stopTimeout):
		am.logger.Warn("Timeout waiting for agent shutdown, forcing kill",
			zap.String("dveID", dveID))
		syscall.Kill(-ap.Cmd.Process.Pid, syscall.SIGKILL)
		ap.setHealthy(false)
		os.Remove(ap.SocketPath)
		return fmt.Errorf("forced shutdown of agent for DVE %s after timeout", dveID)
	}
}

// GetAgent returns the agent process for a DVE, or an error if not running.
func (am *AgentManager) GetAgent(dveID string) (*AgentProcess, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	ap, ok := am.agents[dveID]
	if !ok {
		return nil, fmt.Errorf("no agent running for DVE %s", dveID)
	}
	return ap, nil
}

// ListAgents returns all running agent processes.
func (am *AgentManager) ListAgents() []*AgentProcess {
	am.mu.RLock()
	defer am.mu.RUnlock()

	result := make([]*AgentProcess, 0, len(am.agents))
	for _, ap := range am.agents {
		result = append(result, ap)
	}
	return result
}

// RunningCount returns the number of running agents.
func (am *AgentManager) RunningCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.agents)
}

// MaxConcurrent returns the max concurrent agents limit.
func (am *AgentManager) MaxConcurrent() int {
	return am.maxConcurrent
}

// StopAll stops every agent process in parallel with a global timeout.
func (am *AgentManager) StopAll(ctx context.Context) error {
	am.mu.RLock()
	dveIDs := make([]string, 0, len(am.agents))
	for dveID := range am.agents {
		dveIDs = append(dveIDs, dveID)
	}
	am.mu.RUnlock()

	if len(dveIDs) == 0 {
		return nil
	}

	am.logger.Info("Stopping all KNIRVAGENT processes", zap.Int("count", len(dveIDs)))

	var wg sync.WaitGroup
	errCh := make(chan error, len(dveIDs))

	for _, dveID := range dveIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if err := am.StopAgent(id, 10*time.Second); err != nil {
				errCh <- err
			}
		}(dveID)
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		close(errCh)
		// Collect any errors
		var errs []error
		for err := range errCh {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			return fmt.Errorf("errors stopping agents: %v", errs)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout stopping all agents: %w", ctx.Err())
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timeout stopping all agents after 15s")
	}
}

// HealthCheck pings the agent's socket for the given DVE.
func (am *AgentManager) HealthCheck(dveID string) bool {
	ap, err := am.GetAgent(dveID)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", ap.SocketPath, 2*time.Second)
			},
		},
	}

	resp, err := client.Get("http://localhost/health")
	if err != nil {
		ap.setHealthy(false)
		return false
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == http.StatusOK
	ap.setHealthy(healthy)
	return healthy
}

// GetHealthStatus returns the full health status payload from an agent.
func (am *AgentManager) GetHealthStatus(ctx context.Context, dveID string) (*HealthStatus, error) {
	ap, err := am.GetAgent(dveID)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", ap.SocketPath, 2*time.Second)
			},
		},
	}

	resp, err := client.Get("http://localhost/health")
	if err != nil {
		return nil, fmt.Errorf("agent %s health check failed: %w", dveID, err)
	}
	defer resp.Body.Close()

	var status HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode health status: %w", err)
	}

	ap.setHealthy(resp.StatusCode == http.StatusOK)
	return &status, nil
}

// GetSocketPath returns the Unix socket path for a given DVE's agent.
func (am *AgentManager) GetSocketPath(dveID string) (string, error) {
	ap, err := am.GetAgent(dveID)
	if err != nil {
		return "", err
	}
	return ap.SocketPath, nil
}

// InnerAgentClient creates an HTTP client that dials the given DVE's agent
// Unix socket, suitable for forwarding requests to the inner agent API.
func (am *AgentManager) InnerAgentClient(dveID string) (*http.Client, string, error) {
	socketPath, err := am.GetSocketPath(dveID)
	if err != nil {
		return nil, "", err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", socketPath, 5*time.Second)
			},
		},
	}
	return client, socketPath, nil
}

// ForwardToInnerAgent forwards an HTTP request to the inner agent API on the
// given DVE's agent Unix socket and returns the response.
func (am *AgentManager) ForwardToInnerAgent(dveID, method, path string, body io.Reader) (*http.Response, error) {
	client, socketPath, err := am.InnerAgentClient(dveID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, "http://unix"+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build inner request for DVE %s (socket %s): %w", dveID, socketPath, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inner agent request failed for DVE %s (socket %s): %w", dveID, socketPath, err)
	}

	return resp, nil
}

// StartPeriodicHealthCheck starts a goroutine that periodically checks
// all agents and reaps any that are unhealthy.
func (am *AgentManager) StartPeriodicHealthCheck(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	am.stopHealthCheck = cancel

	am.wg.Add(1)
	go func() {
		defer am.wg.Done()
		ticker := time.NewTicker(am.healthInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				am.reapUnhealthy()
			}
		}
	}()

	am.logger.Info("Periodic agent health check started",
		zap.Duration("interval", am.healthInterval))
}

// StopPeriodicHealthCheck stops the periodic health check goroutine.
func (am *AgentManager) StopPeriodicHealthCheck() {
	if am.stopHealthCheck != nil {
		am.stopHealthCheck()
	}
	am.wg.Wait()
}

// reapUnhealthy checks all agents and removes any whose process has exited
// or whose socket is gone.
func (am *AgentManager) reapUnhealthy() {
	am.mu.RLock()
	snapshot := make([]*AgentProcess, 0, len(am.agents))
	for _, ap := range am.agents {
		snapshot = append(snapshot, ap)
	}
	am.mu.RUnlock()

	for _, ap := range snapshot {
		// Check if socket still exists
		if _, err := os.Stat(ap.SocketPath); os.IsNotExist(err) {
			am.logger.Warn("Agent socket missing, reaping",
				zap.String("dveID", ap.DVEID),
				zap.String("socket", ap.SocketPath))
			am.mu.Lock()
			// Re-check inside lock to avoid race
			if existing, ok := am.agents[ap.DVEID]; ok && existing.SocketPath == ap.SocketPath {
				delete(am.agents, ap.DVEID)
			}
			am.mu.Unlock()
			continue
		}

		// Check if process is still alive
		if ap.Cmd != nil && ap.Cmd.Process != nil {
			if err := ap.Cmd.Process.Signal(syscall.Signal(0)); err != nil {
				// Process is gone
				am.logger.Warn("Agent process exited, reaping",
					zap.String("dveID", ap.DVEID),
					zap.Int("pid", ap.PID))
				ap.setHealthy(false)
				os.Remove(ap.SocketPath)
				am.mu.Lock()
				if existing, ok := am.agents[ap.DVEID]; ok && existing.PID == ap.PID {
					delete(am.agents, ap.DVEID)
				}
				am.mu.Unlock()
			}
		}
	}
}

// ──────────────────────────────────────────────
// Legacy backward-compatible types (for main.go)
// ──────────────────────────────────────────────

// ManagerConfig is the legacy config type. New code should use AgentManagerConfig.
type ManagerConfig struct {
	BinaryPath     string
	SocketPath     string
	Port           int
	BackendAPIPort int
	StartTimeout   time.Duration
	StopTimeout    time.Duration
	Stdout         io.Writer
	Stderr         io.Writer
	ExtraEnv       []string
}

// Manager is the legacy manager type. New code should use AgentManager.
type Manager struct {
	inner *AgentManager
	logger *zap.Logger
	cfg    *ManagerConfig
}

// DefaultManagerConfig returns a legacy ManagerConfig with sensible defaults.
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		SocketPath:   "",
		Port:         8080,
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
	}
}

// NewManager creates a legacy Manager wrapping the new AgentManager.
func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	socketDir := ""
	if cfg.SocketPath != "" {
		socketDir = filepath.Dir(cfg.SocketPath)
	}
	if socketDir == "" {
		// Fall back to a stable app-data directory so that socket files are
		// never created in the process CWD, which may not be writable.
		socketDir = filepath.Join(getAgentAppDataDir(), "sockets")
	}

	innerCfg := &AgentManagerConfig{
		BinaryPath:     cfg.BinaryPath,
		SocketDir:      socketDir,
		MaxConcurrent:  32,
		StartTimeout:   cfg.StartTimeout,
		StopTimeout:    cfg.StopTimeout,
		HealthInterval: 30 * time.Second,
		ExtraEnv:       cfg.ExtraEnv,
	}

	inner := NewAgentManager(innerCfg, logger)
	return &Manager{inner: inner, logger: logger, cfg: cfg}
}

// StartAgent delegates per-DVE agent start to the inner AgentManager.
func (m *Manager) StartAgent(ctx context.Context, dveID string, startTimeout time.Duration) error {
	_, err := m.inner.StartAgent(ctx, dveID, startTimeout)
	return err
}

// RunningCount returns the number of running agents.
func (m *Manager) RunningCount() int {
	return m.inner.RunningCount()
}

// GetSocketPathForDVE returns the Unix socket path for the agent of a specific DVE.
func (m *Manager) GetSocketPathForDVE(dveID string) (string, error) {
	ap, err := m.inner.GetAgent(dveID)
	if err != nil {
		return "", err
	}
	return ap.SocketPath, nil
}

// Start starts the agent manager. In the new multi-DVE model, this is a no-op
// that just logs — agents are started per-DVE via StartAgent.
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("KNIRVAGENT legacy Start called (delegating to per-DVE StartAgent)")
	return nil
}

// Stop stops all running agents.
func (m *Manager) Stop(ctx context.Context) error {
	return m.inner.StopAll(ctx)
}

// IsRunning returns true if any agents are running.
func (m *Manager) IsRunning() bool {
	return m.inner.RunningCount() > 0
}

// GetBaseURL returns a legacy base URL stub.
func (m *Manager) GetBaseURL() string {
	return "http://localhost"
}

// GetConfig returns the legacy config.
func (m *Manager) GetConfig() *ManagerConfig {
	return m.cfg
}

// HealthCheck checks if any agent is healthy.
func (m *Manager) HealthCheck(ctx context.Context) error {
	agents := m.inner.ListAgents()
	if len(agents) == 0 {
		return fmt.Errorf("no agents running")
	}
	for _, ap := range agents {
		if m.inner.HealthCheck(ap.DVEID) {
			return nil
		}
	}
	return fmt.Errorf("no healthy agents")
}

// GetHealthStatus returns health status of a specific agent.
func (m *Manager) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	agents := m.inner.ListAgents()
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents running")
	}
	return m.inner.GetHealthStatus(ctx, agents[0].DVEID)
}

// GetPID returns the PID of the first running agent.
func (m *Manager) GetPID() int {
	agents := m.inner.ListAgents()
	if len(agents) == 0 {
		return 0
	}
	return agents[0].PID
}

// GetSocketPath returns the Unix socket path for a specific DVE's agent.
func (m *Manager) GetSocketPath(dveID string) (string, error) {
	return m.inner.GetSocketPath(dveID)
}

// InnerAgentClient creates an HTTP client dialing a specific DVE's agent socket.
func (m *Manager) InnerAgentClient(dveID string) (*http.Client, string, error) {
	return m.inner.InnerAgentClient(dveID)
}

// ForwardToInnerAgent forwards an HTTP request to a specific DVE's inner agent API.
func (m *Manager) ForwardToInnerAgent(dveID, method, path string, body io.Reader) (*http.Response, error) {
	return m.inner.ForwardToInnerAgent(dveID, method, path, body)
}