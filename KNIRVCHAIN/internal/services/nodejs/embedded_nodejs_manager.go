package nodejs

import (
	"context"
	"fmt"
	"log"
	"sync"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/internal/services/interfaces"
)

// EmbeddedNodeJSManager manages all embedded Node.js services (stub implementation - services moved elsewhere)
type EmbeddedNodeJSManager struct {
	config  *config.NodeJSServicesConfig
	mutex   sync.RWMutex
	running bool
}

// NewEmbeddedNodeJSManager creates a new embedded Node.js service manager (stub)
func NewEmbeddedNodeJSManager(cfg *config.NodeJSServicesConfig) *EmbeddedNodeJSManager {
	return &EmbeddedNodeJSManager{
		config:  cfg,
		running: false,
	}
}

// RegisterAllServices registers all enabled services with the unified service manager (stub)
func (m *EmbeddedNodeJSManager) RegisterAllServices() error {
	log.Printf("Node.js services have been moved elsewhere - no services to register")
	return nil
}

// StartAllServices starts all registered embedded Node.js services (stub)
func (m *EmbeddedNodeJSManager) StartAllServices(ctx context.Context) error {
	log.Printf("Node.js services have been moved elsewhere - no services to start")
	m.running = true
	return nil
}

// StopAllServices stops all embedded Node.js services (stub)
func (m *EmbeddedNodeJSManager) StopAllServices(ctx context.Context) error {
	log.Printf("Node.js services have been moved elsewhere - no services to stop")
	m.running = false
	return nil
}

// GetServiceStatuses returns status information for all services (stub)
func (m *EmbeddedNodeJSManager) GetServiceStatuses() []interfaces.ServiceStatus {
	return []interfaces.ServiceStatus{}
}

// GetRunningServices returns all currently running services (stub)
func (m *EmbeddedNodeJSManager) GetRunningServices() []interfaces.Service {
	return []interfaces.Service{}
}

// StartService starts a specific service by name (stub)
func (m *EmbeddedNodeJSManager) StartService(name string) error {
	return fmt.Errorf("service %s management has been moved to separate components", name)
}

// StopService stops a specific service by name (stub)
func (m *EmbeddedNodeJSManager) StopService(name string) error {
	return fmt.Errorf("service %s management has been moved to separate components", name)
}

// RestartService restarts a specific service by name (stub)
func (m *EmbeddedNodeJSManager) RestartService(name string) error {
	return fmt.Errorf("service %s management has been moved to separate components", name)
}

// GetService returns a service by name (stub)
func (m *EmbeddedNodeJSManager) GetService(name string) (interfaces.Service, error) {
	return nil, fmt.Errorf("service %s not found - services have been moved elsewhere", name)
}

// IsRunning returns true if the manager is running
func (m *EmbeddedNodeJSManager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.running
}

// Shutdown gracefully shuts down all services (stub)
func (m *EmbeddedNodeJSManager) Shutdown() error {
	log.Printf("Embedded Node.js service manager shutdown complete (services moved elsewhere)")
	return nil
}

// SetServiceEnvironment sets environment variables for a specific service (stub)
func (m *EmbeddedNodeJSManager) SetServiceEnvironment(serviceName string, env map[string]string) error {
	return fmt.Errorf("service %s environment management has been moved to separate components", serviceName)
}

// GetServiceCount returns the number of registered services (stub)
func (m *EmbeddedNodeJSManager) GetServiceCount() int {
	return 0
}
