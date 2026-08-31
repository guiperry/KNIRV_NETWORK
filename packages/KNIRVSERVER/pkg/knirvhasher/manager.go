package knirvhasher

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
	BinaryPath            string
	SocketPath            string
	GRPCSocketPath        string // backend gRPC socket the data-connector dials into
	DataPath              string
	SocketPerm            uint32
	HeadlessMode          bool
	ArxivEnabled          bool
	PipelineType          string
	StartTimeout          time.Duration
	StopTimeout           time.Duration
	Stdout                interface{}
	Stderr                interface{}
	EnvOverrides          map[string]string
	KnirvserverDeployed   bool // indicates KNIRVHASHER is initialized by KNIRVSERVER
	DirectMode            bool // skip CGMiner, drive ASIC directly over raw USB
	FormalVerifierEnabled bool // enables formal proof verification via Lean worker
}

type HasherStatus struct {
	Status     string             `json:"status"`
	Running    bool               `json:"running"`
	SocketPath string             `json:"socket_path"`
	Uptime     string             `json:"uptime"`
	Training   *TrainingRunStatus `json:"training,omitempty"`
	Timestamp  time.Time          `json:"timestamp"`
}

type TrainingRunStatus struct {
	RunID            string    `json:"run_id"`
	Status           string    `json:"status"`
	Message          string    `json:"message,omitempty"`
	DataPath         string    `json:"data_path"`
	TrainingDataPath string    `json:"training_data_path,omitempty"`
	RecordsLoaded    int       `json:"records_loaded"`
	CheckpointTokens int       `json:"checkpoint_tokens"`
	AverageFitness   float64   `json:"average_fitness"`
	CurrentBatch     int       `json:"current_batch"`
	BatchesDone      int       `json:"batches_done"`
	TotalBatches     int       `json:"total_batches"`
	LastError        string    `json:"last_error,omitempty"`
	Errors           []string  `json:"errors,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

const (
	DefaultSocketPath = "/var/run/knirvhasher.sock"
	DefaultSocketPerm = 0660
)

func DefaultManagerConfig() *ManagerConfig {
	appDataDir := getHasherAppDataDir()
	return &ManagerConfig{
		SocketPath:   filepath.Join(appDataDir, "sockets", "knirvhasher.sock"),
		DataPath:     filepath.Join(appDataDir, "knirvhasher"),
		SocketPerm:   DefaultSocketPerm,
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
	}
}

func getHasherAppDataDir() string {
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
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 30 * time.Second
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 10 * time.Second
	}
	if cfg.SocketPath == "" {
		defaults := DefaultManagerConfig()
		cfg.SocketPath = defaults.SocketPath
	}
	if cfg.SocketPerm == 0 {
		cfg.SocketPerm = DefaultSocketPerm
	}
	if cfg.PipelineType == "" {
		cfg.PipelineType = "goat"
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
	if hasherDir != "." && hasherDir != "/" {
		if err := os.MkdirAll(hasherDir, 0755); err != nil {
			return fmt.Errorf("failed to create socket directory: %w", err)
		}
	}

	if _, err := os.Stat(m.socketPath); err == nil {
		os.Remove(m.socketPath)
	}

	m.logger.Info("Starting KNIRVHASHER",
		zap.String("binary", m.config.BinaryPath),
		zap.String("socket", m.socketPath),
		zap.Bool("knirvserver_managed", m.config.KnirvserverDeployed),
		zap.Bool("asic_driver_autostart", m.config.KnirvserverDeployed && strings.TrimSpace(m.config.EnvOverrides["DEVICE_IP"]) != ""))

	env := os.Environ()
	env = append(env,
		fmt.Sprintf("HASHER_SOCKET_PATH=%s", m.socketPath),
		fmt.Sprintf("HASHER_DATA_PATH=%s", m.config.DataPath),
	)
	if m.config.GRPCSocketPath != "" {
		env = append(env, fmt.Sprintf("HASHER_GRPC_SOCKET_PATH=%s", m.config.GRPCSocketPath))
	}

	if appDataDir := os.Getenv("KNIRV_APP_DATA_DIR"); appDataDir != "" {
		env = append(env, fmt.Sprintf("KNIRV_APP_DATA_DIR=%s", appDataDir))
	} else if _, err := os.Stat("/var/lib/knirvserver"); err == nil {
		env = append(env, "KNIRV_APP_DATA_DIR=/var/lib/knirvserver")
	}

	if m.config.ArxivEnabled {
		env = append(env, "DATAMINER_MODE=production", "ARXIV_ENABLED=true")
	}

	// When running as root (e.g. via sudo), propagate the original user's Python
	// site-packages so the data-mapper's embedded Python can find en_core_web_sm.
	if pyPath := sudoUserPythonPath(); pyPath != "" {
		env = append(env, "PYTHONPATH="+pyPath)
	}

	for k, v := range m.config.EnvOverrides {
		env = setEnvOverride(env, k, v)
	}

	args := []string{}
	if m.config.HeadlessMode {
		socketPerm := fmt.Sprintf("%04o", m.config.SocketPerm)
		args = append(args,
			"--headless",
			fmt.Sprintf("--socket-path=%s", m.socketPath),
			fmt.Sprintf("--socket-perm=%s", socketPerm),
		)
	}
	// Default pipeline: MAPPER mode via Hugging Face API.
	// Data-connector (CONNECTION source) only runs after full user onboarding.
	if m.config.PipelineType == "goat" {
		args = append(args, "-goat")
	}

	if m.config.KnirvserverDeployed {
		args = append(args, "--knirvserver-deployed")
	}
	if m.config.DirectMode {
		args = append(args, "--direct")
	}
	if m.config.FormalVerifierEnabled {
		args = append(args, "--formal-verifier")
	}

	m.cmd = exec.Command(m.config.BinaryPath, args...)
	m.cmd.Env = env
	if w, ok := m.config.Stdout.(io.Writer); ok {
		m.cmd.Stdout = w
	} else {
		m.cmd.Stdout = os.Stdout
	}
	if w, ok := m.config.Stderr.(io.Writer); ok {
		m.cmd.Stderr = w
	} else {
		m.cmd.Stderr = os.Stderr
	}
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

	if m.config.HeadlessMode {
		m.logger.Info("KNIRVHASHER Unix socket ready")
		// Pipeline is NOT auto-started here.  The user triggers it via the
		// frontend "Start Training" button, which calls hasherStart → Start()
		// followed by RunPipeline() on the Manager, which POSTs to the
		// headless server's /api/v1/pipeline/run endpoint.
	}

	m.running = true
	m.logger.Info("KNIRVHASHER started successfully",
		zap.String("socket", m.socketPath))

	return nil
}

// setEnvOverride replaces rather than appends an environment variable. A child
// process can receive duplicate keys when a root.key override is appended to
// the launcher's inherited environment; libc then commonly reads the stale
// first value. This is especially important for DEVICE_IP, which selects the
// attached ASIC endpoint.
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

func resolveBinaryPath(configured string) (string, error) {
	var candidates []string

	if envPath := os.Getenv("KNIRV_HASHER_BINARY_PATH"); envPath != "" {
		candidates = append(candidates, envPath)
	}

	if extractedPath, err := ExtractEmbeddedBinary(""); err == nil {
		candidates = append(candidates, extractedPath)
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "knirvhasher"),
			filepath.Join(exeDir, "bin", "knirvhasher"),
		)
	}

	embeddedDir := filepath.Join(getHasherAppDataDir(), "bin")
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
			return fmt.Errorf("timeout waiting for KNIRVHASHER socket at %s", m.socketPath)
		case <-ticker.C:
			if _, err := os.Stat(m.socketPath); err == nil {
				m.logger.Debug("KNIRVHASHER socket created at", zap.String("socket", m.socketPath))
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
			m.logger.Info("KNIRVHASHER stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVHASHER stopped gracefully")
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
		Training:   m.loadTrainingRunStatus(),
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

func (m *Manager) loadTrainingRunStatus() *TrainingRunStatus {
	for _, statusPath := range m.trainingRunStatusPaths() {
		data, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}

		var status TrainingRunStatus
		if err := json.Unmarshal(data, &status); err != nil {
			if m.logger != nil {
				m.logger.Warn("Failed to decode KNIRVHASHER training status",
					zap.String("path", statusPath),
					zap.Error(err))
			}
			continue
		}
		return &status
	}
	return nil
}

func (m *Manager) trainingRunStatusPaths() []string {
	paths := make([]string, 0, 3)
	add := func(path string) {
		if path == "" {
			return
		}
		for _, existing := range paths {
			if existing == path {
				return
			}
		}
		paths = append(paths, path)
	}

	if m.config != nil && m.config.DataPath != "" {
		add(filepath.Join(m.config.DataPath, "data", "runs", "latest_training_run.json"))
		add(filepath.Join(m.config.DataPath, "runs", "latest_training_run.json"))
	}
	add(filepath.Join(getHasherAppDataDir(), "knirvhasher", "data", "runs", "latest_training_run.json"))
	return paths
}

func (m *Manager) triggerPipeline() {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", m.socketPath)
			},
		},
	}

	pipelineURL := "http://localhost/api/v1/pipeline/run"
	body := fmt.Sprintf(`{"type":"%s"}`, m.config.PipelineType)
	req, err := http.NewRequest("POST", pipelineURL, strings.NewReader(body))
	if err != nil {
		m.logger.Warn("Failed to create pipeline request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		m.logger.Warn("Failed to trigger KNIRVHASHER data pipeline",
			zap.String("socket", m.socketPath), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		m.logger.Info("KNIRVHASHER data pipeline triggered successfully",
			zap.String("socket", m.socketPath),
			zap.String("pipeline", m.config.PipelineType),
			zap.Int("status", resp.StatusCode))
	} else {
		m.logger.Warn("KNIRVHASHER data pipeline trigger returned unexpected status",
			zap.String("socket", m.socketPath),
			zap.Int("status", resp.StatusCode))
	}
}

func (m *Manager) RunPipeline(pipelineType string) error {
	if !m.running {
		return fmt.Errorf("KNIRVHASHER not running")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "unix", m.socketPath)
			},
		},
	}

	pipelineURL := "http://localhost/api/v1/pipeline/run"
	body := fmt.Sprintf(`{"type":"%s"}`, pipelineType)
	req, err := http.NewRequest("POST", pipelineURL, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to trigger pipeline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("pipeline request failed with status %d", resp.StatusCode)
	}

	return nil
}

func (m *Manager) GetHealth() (bool, error) {
	if !m.running {
		return false, fmt.Errorf("KNIRVHASHER not running")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "unix", m.socketPath)
			},
		},
	}

	resp, err := client.Get("http://localhost/api/v1/health")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (m *Manager) StopPipeline() error {
	if !m.running {
		return fmt.Errorf("KNIRVHASHER not running")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "unix", m.socketPath)
			},
		},
	}

	resp, err := client.Post("http://localhost/api/v1/pipeline/stop", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to stop pipeline: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func isSocketReady(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// sudoUserPythonPath returns the Python user site-packages directory for the
// original non-root user when knirvserver is launched via sudo. The data-mapper
// binary runs as root but needs to reach user-installed spaCy models (e.g.
// en_core_web_sm) that live under the invoking user's ~/.local, not /root/.local.
func sudoUserPythonPath() string {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		return ""
	}
	out, err := exec.Command("python3", "-c",
		"import sys; print(f'python{sys.version_info.major}.{sys.version_info.minor}', end='')").Output()
	if err != nil {
		return ""
	}
	pyVer := strings.TrimSpace(string(out))
	if pyVer == "" {
		return ""
	}
	userSite := filepath.Join("/home", sudoUser, ".local", "lib", pyVer, "site-packages")
	if _, err := os.Stat(userSite); err != nil {
		return ""
	}
	return userSite
}
