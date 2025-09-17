package binary

import (
	"context"
	"fmt"
	"log"
	"sync"

	"KNIRVORACLE/config"
	"KNIRVORACLE/pkg/embedded/binaries/economics"
	"KNIRVORACLE/pkg/embedded/binaries/network"
	"KNIRVORACLE/pkg/services/interfaces"
	"KNIRVORACLE/pkg/services/manager"
)

// EmbeddedBinaryManager manages all embedded binary services
type EmbeddedBinaryManager struct {
	config           *config.Config
	serviceManager   *manager.UnifiedServiceManager
	economicsService *economics.EmbeddedEconomicsService
	networkService   *network.EmbeddedNetworkMonitor
	mutex            sync.RWMutex
	running          bool
}

// NewEmbeddedBinaryManager creates a new embedded binary service manager
func NewEmbeddedBinaryManager(cfg *config.Config) *EmbeddedBinaryManager {
	manager := &EmbeddedBinaryManager{
		config:         cfg,
		serviceManager: manager.NewUnifiedServiceManager(),
		running:        false,
	}
	
	// Initialize economics service if enabled
	if cfg.IsRoot {
		manager.economicsService = economics.NewEmbeddedEconomicsService(
			8090, // Default economics port
			true, // Always enabled for root nodes
		)
	}
	
	// Initialize network monitor service if testnet is enabled
	if cfg.Testnet.Enabled {
		manager.networkService = network.NewEmbeddedNetworkMonitor(
			8091, // Default network monitor port
			true, // Always enabled for testnet
		)
	}
	
	return manager
}

// RegisterAllServices registers all enabled services with the unified service manager
func (m *EmbeddedBinaryManager) RegisterAllServices() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	var errors []error
	
	// Register economics service
	if m.economicsService != nil {
		if err := m.serviceManager.RegisterService(m.economicsService); err != nil {
			errors = append(errors, fmt.Errorf("failed to register economics service: %w", err))
		} else {
			log.Printf("Registered embedded economics service")
		}
	}
	
	// Register network monitor service
	if m.networkService != nil {
		if err := m.serviceManager.RegisterService(m.networkService); err != nil {
			errors = append(errors, fmt.Errorf("failed to register network monitor service: %w", err))
		} else {
			log.Printf("Registered embedded network monitor service")
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("failed to register some services: %v", errors)
	}
	
	log.Printf("All enabled embedded binary services registered successfully")
	return nil
}

// StartAllServices starts all registered embedded binary services
func (m *EmbeddedBinaryManager) StartAllServices(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.running {
		return fmt.Errorf("embedded binary services are already running")
	}
	
	// Register all services first
	if err := m.RegisterAllServices(); err != nil {
		return fmt.Errorf("failed to register services: %w", err)
	}
	
	// Start all services
	if err := m.serviceManager.StartAllServices(); err != nil {
		return fmt.Errorf("failed to start embedded binary services: %w", err)
	}
	
	m.running = true
	log.Printf("All embedded binary services started successfully")
	return nil
}

// StopAllServices stops all embedded binary services
func (m *EmbeddedBinaryManager) StopAllServices(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.running {
		return fmt.Errorf("embedded binary services are not running")
	}
	
	// Stop all services
	if err := m.serviceManager.StopAllServices(); err != nil {
		return fmt.Errorf("failed to stop embedded binary services: %w", err)
	}
	
	m.running = false
	log.Printf("All embedded binary services stopped successfully")
	return nil
}

// GetServiceStatuses returns status information for all services
func (m *EmbeddedBinaryManager) GetServiceStatuses() []interfaces.ServiceStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.serviceManager.GetServiceStatuses()
}

// GetRunningServices returns all currently running services
func (m *EmbeddedBinaryManager) GetRunningServices() []interfaces.Service {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.serviceManager.GetRunningServices()
}

// StartService starts a specific service by name
func (m *EmbeddedBinaryManager) StartService(name string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.serviceManager.StartService(name)
}

// StopService stops a specific service by name
func (m *EmbeddedBinaryManager) StopService(name string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.serviceManager.StopService(name)
}

// RestartService restarts a specific service by name
func (m *EmbeddedBinaryManager) RestartService(name string) error {
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
func (m *EmbeddedBinaryManager) GetService(name string) (interfaces.Service, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.serviceManager.GetService(name)
}

// IsRunning returns true if the manager is running
func (m *EmbeddedBinaryManager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.running
}

// Shutdown gracefully shuts down all services
func (m *EmbeddedBinaryManager) Shutdown() error {
	log.Printf("Shutting down embedded binary service manager...")
	
	// Stop all services
	if err := m.StopAllServices(context.Background()); err != nil {
		log.Printf("Warning: Error stopping services during shutdown: %v", err)
	}
	
	// Shutdown the unified service manager
	if err := m.serviceManager.Shutdown(); err != nil {
		log.Printf("Warning: Error shutting down service manager: %v", err)
	}
	
	log.Printf("Embedded binary service manager shutdown complete")
	return nil
}

// SetServiceEnvironment sets environment variables for a specific service
func (m *EmbeddedBinaryManager) SetServiceEnvironment(serviceName string, env map[string]string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	switch serviceName {
	case "economics-service":
		if m.economicsService != nil {
			m.economicsService.SetEnvironment(env)
			return nil
		}
	case "network-monitor":
		if m.networkService != nil {
			m.networkService.SetEnvironment(env)
			return nil
		}
	}
	
	return fmt.Errorf("service %s not found or not enabled", serviceName)
}

// GetServiceCount returns the number of registered services
func (m *EmbeddedBinaryManager) GetServiceCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	count := 0
	if m.economicsService != nil {
		count++
	}
	if m.networkService != nil {
		count++
	}
	
	return count
}

// GetEconomicsService returns the economics service if available
func (m *EmbeddedBinaryManager) GetEconomicsService() *economics.EmbeddedEconomicsService {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.economicsService
}

// GetNetworkMonitorService returns the network monitor service if available
func (m *EmbeddedBinaryManager) GetNetworkMonitorService() *network.EmbeddedNetworkMonitor {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.networkService
}

// EnableEconomicsService enables the economics service for the current configuration
func (m *EmbeddedBinaryManager) EnableEconomicsService(port uint64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.economicsService != nil {
		return fmt.Errorf("economics service is already enabled")
	}
	
	m.economicsService = economics.NewEmbeddedEconomicsService(port, true)
	log.Printf("Economics service enabled on port %d", port)
	return nil
}

// EnableNetworkMonitorService enables the network monitor service for the current configuration
func (m *EmbeddedBinaryManager) EnableNetworkMonitorService(port uint64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.networkService != nil {
		return fmt.Errorf("network monitor service is already enabled")
	}
	
	m.networkService = network.NewEmbeddedNetworkMonitor(port, true)
	log.Printf("Network monitor service enabled on port %d", port)
	return nil
}
