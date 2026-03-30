package embedded

import (
	"embed"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

var (
	Binary embed.FS
)

type Config struct {
	Port         int
	P2PPort      int
	APIPort      int
	DataPath     string
	StartTimeout time.Duration
	StopTimeout  time.Duration
}

type Manager struct {
	cfg     *Config
	logger  *zap.Logger
	stopCh  chan struct{}
	running bool
}

func NewManager(cfg *Config, logger *zap.Logger) (*Manager, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}

	return &Manager{
		cfg:     cfg,
		logger:  logger,
		stopCh:  make(chan struct{}),
		running: false,
	}, nil
}

func DefaultConfig() *Config {
	return &Config{
		Port:         7090,
		P2PPort:      7091,
		APIPort:      7092,
		DataPath:     "./data",
		StartTimeout: 30 * time.Second,
		StopTimeout:  10 * time.Second,
	}
}

func (m *Manager) Start() error {
	m.logger.Info("Starting embedded KNIRVGRAPH",
		zap.Int("port", m.cfg.Port),
		zap.Int("p2p_port", m.cfg.P2PPort),
		zap.Int("api_port", m.cfg.APIPort),
	)

	m.running = true
	m.logger.Info("Embedded KNIRVGRAPH started successfully")

	go m.handleSignals()

	return nil
}

func (m *Manager) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		m.logger.Info("Received shutdown signal")
		m.Stop()
	case <-m.stopCh:
		m.logger.Info("Received stop signal")
	}
}

func (m *Manager) Stop() error {
	m.logger.Info("Stopping embedded KNIRVGRAPH")

	close(m.stopCh)
	m.running = false

	m.logger.Info("Embedded KNIRVGRAPH stopped")
	return nil
}

func (m *Manager) GetPort() int {
	return m.cfg.Port
}

func (m *Manager) GetP2PPort() int {
	return m.cfg.P2PPort
}

func (m *Manager) GetAPIPort() int {
	return m.cfg.APIPort
}

func (m *Manager) IsRunning() bool {
	return m.running
}
