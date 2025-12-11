package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"go.uber.org/zap"
)

// Service represents a Node.js service process
type Service struct {
	Name       string
	Dir        string
	Port       int
	Cmd        *exec.Cmd
	mu         sync.RWMutex
	Running    bool
	StartTime  time.Time
	restartCount int
}

// Manager manages Node.js service processes
type Manager struct {
	config   *config.Config
	logger   *zap.Logger
	services map[string]*Service
	mu       sync.RWMutex
}

// NewManager creates a new service manager
func NewManager(cfg *config.Config, logger *zap.Logger, servicesDir string) (*Manager, error) {
	m := &Manager{
		config:   cfg,
		logger:   logger,
		services: make(map[string]*Service),
	}

	// Initialize service definitions
	if err := m.initializeServices(servicesDir); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	return m, nil
}

// initializeServices sets up service definitions based on configuration
func (m *Manager) initializeServices(servicesDir string) error {
	if !m.config.NodeJSServicesEnabled {
		m.logger.Info("Node.js services disabled")
		return nil
	}

	// Validate services directory exists
	if _, err := os.Stat(servicesDir); os.IsNotExist(err) {
		return fmt.Errorf("services directory does not exist: %s", servicesDir)
	}

	// Payment Gateway - migrated to Go, no longer a Node.js service
	// if m.config.PaymentGatewayEnabled {
	// 	m.services["payment-gateway"] = &Service{
	// 		Name: "payment-gateway",
	// 		Dir:  filepath.Join(servicesDir, "payment-gateway"),
	// 		Port: m.config.PaymentGatewayPort,
	// 	}
	// }

	// Tunnel Registry
	if m.config.TunnelRegistryEnabled {
		m.services["tunnel-registry"] = &Service{
			Name: "tunnel-registry",
			Dir:  filepath.Join(servicesDir, "tunnel-registry"),
			Port: m.config.TunnelRegistryHTTPPort,
		}
	}

	// Operator Registry (disabled by default)
	if m.config.OperatorRegistryEnabled {
		m.services["operator-registry"] = &Service{
			Name: "operator-registry",
			Dir:  filepath.Join(servicesDir, "operator-registry"),
			Port: m.config.OperatorRegistryPort,
		}
	}

	// WebGUI
	if m.config.WebGUIEnabled {
		m.services["webgui"] = &Service{
			Name: "webgui",
			Dir:  filepath.Join(servicesDir, "webgui"),
			Port: m.config.WebGUIPort,
		}
	}

	m.logger.Info("Initialized services",
		zap.Int("count", len(m.services)),
		zap.Strings("services", m.getServiceNames()),
	)

	return nil
}

// StartAll starts all enabled services
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Starting all services")

	var wg sync.WaitGroup
	errChan := make(chan error, len(m.services))

	for name, svc := range m.services {
		wg.Add(1)
		go func(n string, s *Service) {
			defer wg.Done()
			if err := m.startService(ctx, s); err != nil {
				m.logger.Error("Failed to start service",
					zap.String("service", n),
					zap.Error(err),
				)
				errChan <- fmt.Errorf("%s: %w", n, err)
			}
		}(name, svc)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to start %d services", len(errs))
	}

	return nil
}

// StopAll stops all running services
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Stopping all services")

	for name, svc := range m.services {
		if err := m.stopService(ctx, svc); err != nil {
			m.logger.Error("Failed to stop service",
				zap.String("service", name),
				zap.Error(err),
			)
		}
	}

	return nil
}

// StartService starts a specific service
func (m *Manager) StartService(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	svc, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service not found: %s", name)
	}

	return m.startService(ctx, svc)
}

// StopService stops a specific service
func (m *Manager) StopService(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	svc, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service not found: %s", name)
	}

	return m.stopService(ctx, svc)
}

// startService starts a service process (internal, must be called with lock held)
func (m *Manager) startService(ctx context.Context, svc *Service) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.Running {
		return fmt.Errorf("service already running")
	}

	// Check if server.js exists
	serverPath := filepath.Join(svc.Dir, "server.js")
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		return fmt.Errorf("server.js not found in %s", svc.Dir)
	}

	// Create command
	cmd := exec.CommandContext(ctx, "node", "server.js")
	cmd.Dir = svc.Dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", svc.Port),
		fmt.Sprintf("KNIRV_CHAIN_ID=%s", m.config.ChainID),
	)

	// Set up output pipes
	cmd.Stdout = &serviceLogger{logger: m.logger, service: svc.Name, level: "info"}
	cmd.Stderr = &serviceLogger{logger: m.logger, service: svc.Name, level: "error"}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	svc.Cmd = cmd
	svc.Running = true
	svc.StartTime = time.Now()
	svc.restartCount++

	m.logger.Info("Service started",
		zap.String("service", svc.Name),
		zap.Int("port", svc.Port),
		zap.Int("pid", cmd.Process.Pid),
	)

	// Monitor the process in a goroutine
	go func() {
		err := cmd.Wait()
		svc.mu.Lock()
		svc.Running = false
		svc.Cmd = nil
		svc.mu.Unlock()

		if err != nil {
			m.logger.Error("Service exited with error",
				zap.String("service", svc.Name),
				zap.Error(err),
			)
		} else {
			m.logger.Info("Service exited",
				zap.String("service", svc.Name),
			)
		}
	}()

	return nil
}

// stopService stops a service process (internal, must be called with lock held)
func (m *Manager) stopService(ctx context.Context, svc *Service) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if !svc.Running || svc.Cmd == nil || svc.Cmd.Process == nil {
		return nil
	}

	m.logger.Info("Stopping service",
		zap.String("service", svc.Name),
		zap.Int("pid", svc.Cmd.Process.Pid),
	)

	// Send SIGTERM
	if err := svc.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		m.logger.Warn("SIGTERM failed, using SIGKILL",
			zap.String("service", svc.Name),
			zap.Error(err),
		)
		if killErr := svc.Cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := svc.Cmd.Process.Wait()
		done <- err
	}()

	select {
	case <-ctx.Done():
		// Timeout - force kill
		_ = svc.Cmd.Process.Kill()
		return ctx.Err()
	case err := <-done:
		svc.Running = false
		svc.Cmd = nil
		if err != nil {
			return err
		}
	}

	return nil
}

// GetStatus returns the status of all services
func (m *Manager) GetStatus() map[string]ServiceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]ServiceStatus)
	for name, svc := range m.services {
		svc.mu.RLock()
		svcStatus := ServiceStatus{
			Name:         svc.Name,
			Running:      svc.Running,
			Port:         svc.Port,
			StartTime:    svc.StartTime,
			RestartCount: svc.restartCount,
		}
		if svc.Cmd != nil && svc.Cmd.Process != nil {
			svcStatus.PID = svc.Cmd.Process.Pid
		}
		status[name] = svcStatus
		svc.mu.RUnlock()
	}

	return status
}

// GetServicePort returns the port for a specific service
func (m *Manager) GetServicePort(name string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	svc, exists := m.services[name]
	if !exists {
		return 0, false
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	return svc.Port, svc.Running
}

// getServiceNames returns a list of service names
func (m *Manager) getServiceNames() []string {
	names := make([]string, 0, len(m.services))
	for name := range m.services {
		names = append(names, name)
	}
	return names
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name         string    `json:"name"`
	Running      bool      `json:"running"`
	Port         int       `json:"port"`
	PID          int       `json:"pid,omitempty"`
	StartTime    time.Time `json:"startTime,omitempty"`
	RestartCount int       `json:"restartCount"`
}

// serviceLogger adapts service output to zap logger
type serviceLogger struct {
	logger  *zap.Logger
	service string
	level   string
}

func (l *serviceLogger) Write(p []byte) (n int, err error) {
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	if l.level == "error" {
		l.logger.Error(msg, zap.String("service", l.service))
	} else {
		l.logger.Info(msg, zap.String("service", l.service))
	}

	return len(p), nil
}
