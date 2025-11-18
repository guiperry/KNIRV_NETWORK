package nodejs

import (
	"context"
	"fmt"
	"log"
	"sync"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/pkg/embedded/nodejs/operator"
	"KNIRVCHAIN/pkg/embedded/nodejs/payment"
	"KNIRVCHAIN/pkg/embedded/nodejs/tunnel"
	"KNIRVCHAIN/pkg/embedded/nodejs/webgui"
	"KNIRVCHAIN/pkg/services/interfaces"
	"KNIRVCHAIN/pkg/services/manager"
)

// EmbeddedNodeJSManager manages all embedded Node.js services
type EmbeddedNodeJSManager struct {
	config          *config.NodeJSServicesConfig
	serviceManager  *manager.UnifiedServiceManager
	operatorService *operator.EmbeddedOperatorRegistry
	tunnelService   *tunnel.EmbeddedTunnelRegistry
	paymentService  *payment.EmbeddedPaymentGateway
	webguiService   *webgui.EmbeddedWebGUI
	mutex           sync.RWMutex
	running         bool
}

// NewEmbeddedNodeJSManager creates a new embedded Node.js service manager
func NewEmbeddedNodeJSManager(cfg *config.NodeJSServicesConfig) *EmbeddedNodeJSManager {
	manager := &EmbeddedNodeJSManager{
		config:         cfg,
		serviceManager: manager.NewUnifiedServiceManager(),
		running:        false,
	}

	// Initialize services based on configuration
	if cfg.OperatorRegistry.Enabled {
		manager.operatorService = operator.NewEmbeddedOperatorRegistry(
			uint64(cfg.OperatorRegistry.HTTPPort),
			cfg.OperatorRegistry.Enabled,
		)
	}

	if cfg.TunnelRegistry.Enabled {
		manager.tunnelService = tunnel.NewEmbeddedTunnelRegistry(
			uint64(cfg.TunnelRegistry.HTTPPort),
			cfg.TunnelRegistry.Enabled,
		)
	}

	if cfg.PaymentGateway.Enabled {
		manager.paymentService = payment.NewEmbeddedPaymentGateway(
			uint64(cfg.PaymentGateway.HTTPPort),
			cfg.PaymentGateway.Enabled,
		)
	}

	if cfg.WebGUI.Enabled {
		manager.webguiService = webgui.NewEmbeddedWebGUI(
			uint64(cfg.WebGUI.HTTPPort),
			cfg.WebGUI.Enabled,
		)
	}

	return manager
}

// RegisterAllServices registers all enabled services with the unified service manager
func (m *EmbeddedNodeJSManager) RegisterAllServices() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var errors []error

	// Register operator registry service
	if m.operatorService != nil {
		if err := m.serviceManager.RegisterService(m.operatorService); err != nil {
			errors = append(errors, fmt.Errorf("failed to register operator registry service: %w", err))
		} else {
			log.Printf("Registered embedded operator registry service")
		}
	}

	// Register tunnel registry service
	if m.tunnelService != nil {
		if err := m.serviceManager.RegisterService(m.tunnelService); err != nil {
			errors = append(errors, fmt.Errorf("failed to register tunnel registry service: %w", err))
		} else {
			log.Printf("Registered embedded tunnel registry service")
		}
	}

	// Register payment gateway service
	if m.paymentService != nil {
		if err := m.serviceManager.RegisterService(m.paymentService); err != nil {
			errors = append(errors, fmt.Errorf("failed to register payment gateway service: %w", err))
		} else {
			log.Printf("Registered embedded payment gateway service")
		}
	}

	// Register Web GUI service
	if m.webguiService != nil {
		if err := m.serviceManager.RegisterService(m.webguiService); err != nil {
			errors = append(errors, fmt.Errorf("failed to register Web GUI service: %w", err))
		} else {
			log.Printf("Registered embedded Web GUI service")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to register some services: %v", errors)
	}

	log.Printf("All enabled embedded Node.js services registered successfully")
	return nil
}

// StartAllServices starts all registered embedded Node.js services
func (m *EmbeddedNodeJSManager) StartAllServices(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("embedded Node.js services are already running")
	}

	// Register all services first
	if err := m.RegisterAllServices(); err != nil {
		return fmt.Errorf("failed to register services: %w", err)
	}

	// Start all services
	if err := m.serviceManager.StartAllServices(); err != nil {
		return fmt.Errorf("failed to start embedded Node.js services: %w", err)
	}

	m.running = true
	log.Printf("All embedded Node.js services started successfully")
	return nil
}

// StopAllServices stops all embedded Node.js services
func (m *EmbeddedNodeJSManager) StopAllServices(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return fmt.Errorf("embedded Node.js services are not running")
	}

	// Stop all services
	if err := m.serviceManager.StopAllServices(); err != nil {
		return fmt.Errorf("failed to stop embedded Node.js services: %w", err)
	}

	m.running = false
	log.Printf("All embedded Node.js services stopped successfully")
	return nil
}

// GetServiceStatuses returns status information for all services
func (m *EmbeddedNodeJSManager) GetServiceStatuses() []interfaces.ServiceStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.serviceManager.GetServiceStatuses()
}

// GetRunningServices returns all currently running services
func (m *EmbeddedNodeJSManager) GetRunningServices() []interfaces.Service {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.serviceManager.GetRunningServices()
}

// StartService starts a specific service by name
func (m *EmbeddedNodeJSManager) StartService(name string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.serviceManager.StartService(name)
}

// StopService stops a specific service by name
func (m *EmbeddedNodeJSManager) StopService(name string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.serviceManager.StopService(name)
}

// RestartService restarts a specific service by name
func (m *EmbeddedNodeJSManager) RestartService(name string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	service, err := m.serviceManager.GetService(name)
	if err != nil {
		return fmt.Errorf("failed to get service %s: %w", name, err)
	}

	embeddedService, ok := service.(interfaces.EmbeddedService)
	if !ok {
		return fmt.Errorf("service %s does not support restart", name)
	}

	return embeddedService.Restart(context.Background())
}

// GetService returns a service by name
func (m *EmbeddedNodeJSManager) GetService(name string) (interfaces.Service, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.serviceManager.GetService(name)
}

// IsRunning returns true if the manager is running
func (m *EmbeddedNodeJSManager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.running
}

// Shutdown gracefully shuts down all services
func (m *EmbeddedNodeJSManager) Shutdown() error {
	log.Printf("Shutting down embedded Node.js service manager...")

	// Stop all services
	if err := m.StopAllServices(context.Background()); err != nil {
		log.Printf("Warning: Error stopping services during shutdown: %v", err)
	}

	// Shutdown the unified service manager
	if err := m.serviceManager.Shutdown(); err != nil {
		log.Printf("Warning: Error shutting down service manager: %v", err)
	}

	log.Printf("Embedded Node.js service manager shutdown complete")
	return nil
}

// SetServiceEnvironment sets environment variables for a specific service
func (m *EmbeddedNodeJSManager) SetServiceEnvironment(serviceName string, env map[string]string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch serviceName {
	case "operator-registry":
		if m.operatorService != nil {
			m.operatorService.SetEnvironment(env)
			return nil
		}
	case "tunnel-registry":
		if m.tunnelService != nil {
			m.tunnelService.SetEnvironment(env)
			return nil
		}
	case "payment-gateway":
		if m.paymentService != nil {
			m.paymentService.SetEnvironment(env)
			return nil
		}
	case "webgui":
		if m.webguiService != nil {
			m.webguiService.SetEnvironment(env)
			return nil
		}
	}

	return fmt.Errorf("service %s not found or not enabled", serviceName)
}

// GetServiceCount returns the number of registered services
func (m *EmbeddedNodeJSManager) GetServiceCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	count := 0
	if m.operatorService != nil {
		count++
	}
	if m.tunnelService != nil {
		count++
	}
	if m.paymentService != nil {
		count++
	}
	if m.webguiService != nil {
		count++
	}

	return count
}
