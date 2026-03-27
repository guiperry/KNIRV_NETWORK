package knirvgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

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
	BinaryPath   string
	Port         int
	Ports        *PortConfig
	DBPath       string
	AuthSecret   string
	MinerAddress string
	StartTimeout time.Duration
	StopTimeout  time.Duration
	ChainID      string
}

type PortConfig struct {
	TurnUDP     int
	TurnTCP     int
	TurnAPI     int
	TunnelHTTP  int
	TunnelCtrl  int
	TunnelRelay int
	TunnelSTUN  int
}

type TurnServerStatus struct {
	Status       string    `json:"status"`
	Running      bool      `json:"running"`
	UDPPort      int       `json:"udp_port"`
	TCPPort      int       `json:"tcp_port"`
	APIPort      int       `json:"api_port"`
	Realm        string    `json:"realm"`
	SessionCount int64     `json:"session_count"`
	ActiveRelays int64     `json:"active_relays"`
	Uptime       string    `json:"uptime"`
	Timestamp    time.Time `json:"timestamp"`
}

type HealthStatus struct {
	Status     string                 `json:"status"`
	TurnServer bool                   `json:"turn_server"`
	Services   map[string]interface{} `json:"services"`
	Timestamp  time.Time              `json:"timestamp"`
}

func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		Port: 8081,
		Ports: &PortConfig{
			TurnUDP:     3478,
			TurnTCP:     3479,
			TurnAPI:     3476,
			TunnelHTTP:  3002,
			TunnelCtrl:  3003,
			TunnelRelay: 3004,
			TunnelSTUN:  3005,
		},
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
		ChainID:      "testnet",
	}
}

func NewManager(cfg *ManagerConfig, logger *zap.Logger) *Manager {
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 30 * time.Second
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 10 * time.Second
	}

	baseURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
	if cfg.Port == 0 {
		baseURL = "http://localhost:8081"
	}

	return &Manager{
		config:  cfg,
		logger:  logger,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Info("KNIRVGATEWAY already running")
		return nil
	}

	if m.config.BinaryPath == "" {
		m.config.BinaryPath = "./knirvgateway"
	}

	if _, err := os.Stat(m.config.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("KNIRVGATEWAY binary not found at %s", m.config.BinaryPath)
	}

	m.logger.Info("Starting KNIRVGATEWAY",
		zap.String("binary", m.config.BinaryPath),
		zap.Int("port", m.config.Port))

	env := os.Environ()
	env = append(env,
		fmt.Sprintf("PORT=%d", m.config.Port),
		fmt.Sprintf("KNIRV_CHAIN_ID=%s", m.config.ChainID),
		fmt.Sprintf("TURN_SERVER_ENABLED=true"),
		fmt.Sprintf("TURN_SERVER_UDP_PORT=%d", m.config.Ports.TurnUDP),
		fmt.Sprintf("TURN_SERVER_TCP_PORT=%d", m.config.Ports.TurnTCP),
		fmt.Sprintf("TURN_SERVER_API_PORT=%d", m.config.Ports.TurnAPI),
		fmt.Sprintf("TURN_SERVER_AUTH_SECRET=%s", m.config.AuthSecret),
		fmt.Sprintf("TURN_SERVER_REALM=knirvgateway.local"),
		fmt.Sprintf("TURN_SERVER_MINER_ADDRESS=%s", m.config.MinerAddress),
		fmt.Sprintf("TUNNEL_REGISTRY_ENABLED=true"),
		fmt.Sprintf("TUNNEL_REGISTRY_HTTP_PORT=%d", m.config.Ports.TunnelHTTP),
		fmt.Sprintf("TUNNEL_REGISTRY_CONTROL_PORT=%d", m.config.Ports.TunnelCtrl),
		fmt.Sprintf("TUNNEL_REGISTRY_PUBLIC_RELAY_PORT=%d", m.config.Ports.TunnelRelay),
		fmt.Sprintf("TUNNEL_REGISTRY_STUN_PORT=%d", m.config.Ports.TunnelSTUN),
		fmt.Sprintf("DATABASE_PATH=%s", m.config.DBPath),
	)

	m.cmd = exec.Command(m.config.BinaryPath)
	m.cmd.Env = env
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start KNIRVGATEWAY: %w", err)
	}

	m.logger.Info("KNIRVGATEWAY subprocess started",
		zap.Int("pid", m.cmd.Process.Pid))

	if err := m.waitForHealth(ctx); err != nil {
		m.cmd.Process.Kill()
		return fmt.Errorf("health check failed: %w", err)
	}

	m.running = true
	m.logger.Info("KNIRVGATEWAY started successfully",
		zap.String("url", m.baseURL))

	return nil
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
			return fmt.Errorf("timeout waiting for KNIRVGATEWAY health check")
		case <-ticker.C:
			resp, err := m.client.Get(healthURL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					m.logger.Debug("KNIRVGATEWAY health check passed")
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

	m.logger.Info("Stopping KNIRVGATEWAY")

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
			m.logger.Info("KNIRVGATEWAY stopped", zap.Error(err))
		} else {
			m.logger.Info("KNIRVGATEWAY stopped gracefully")
		}
		return nil
	case <-time.After(m.config.StopTimeout):
		m.logger.Warn("Timeout waiting for graceful shutdown, forcing kill")
		m.cmd.Process.Kill()
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

func (m *Manager) GetTurnStatus(ctx context.Context) (*TurnServerStatus, error) {
	url := fmt.Sprintf("http://localhost:%d/api/turn/status", m.config.Ports.TurnAPI)
	resp, err := m.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get TURN status: %w", err)
	}
	defer resp.Body.Close()

	var status TurnServerStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode TURN status: %w", err)
	}

	return &status, nil
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

func (m *Manager) GetPorts() *PortConfig {
	return m.config.Ports
}
