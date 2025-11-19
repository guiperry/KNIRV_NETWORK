package manager

import (
	"context"
	"fmt"
	"log"
	"sync"

	"KNIRVCHAIN/internal/services/interfaces"
)

// UnifiedServiceManager manages all services (Node.js, binary, embedded)
type UnifiedServiceManager struct {
	services map[string]interfaces.Service
	mutex    sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewUnifiedServiceManager creates a new unified service manager
func NewUnifiedServiceManager() *UnifiedServiceManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &UnifiedServiceManager{
		services: make(map[string]interfaces.Service),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterService registers a service with the manager
func (m *UnifiedServiceManager) RegisterService(service interfaces.Service) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := service.Name()
	if _, exists := m.services[name]; exists {
		return fmt.Errorf("service %s is already registered", name)
	}

	m.services[name] = service
	log.Printf("Service %s registered successfully", name)
	return nil
}

// UnregisterService unregisters a service from the manager
func (m *UnifiedServiceManager) UnregisterService(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	service, exists := m.services[name]
	if !exists {
		return fmt.Errorf("service %s is not registered", name)
	}

	// Stop the service if it's running
	if service.IsRunning() {
		if err := service.Stop(m.ctx); err != nil {
			log.Printf("Warning: Failed to stop service %s during unregistration: %v", name, err)
		}
	}

	delete(m.services, name)
	log.Printf("Service %s unregistered successfully", name)
	return nil
}

// StartService starts a specific service
func (m *UnifiedServiceManager) StartService(name string) error {
	m.mutex.RLock()
	service, exists := m.services[name]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("service %s is not registered", name)
	}

	if service.IsRunning() {
		return fmt.Errorf("service %s is already running", name)
	}

	log.Printf("Starting service %s...", name)
	if err := service.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}

	log.Printf("Service %s started successfully", name)
	return nil
}

// StopService stops a specific service
func (m *UnifiedServiceManager) StopService(name string) error {
	m.mutex.RLock()
	service, exists := m.services[name]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("service %s is not registered", name)
	}

	if !service.IsRunning() {
		return fmt.Errorf("service %s is not running", name)
	}

	log.Printf("Stopping service %s...", name)
	if err := service.Stop(m.ctx); err != nil {
		return fmt.Errorf("failed to stop service %s: %w", name, err)
	}

	log.Printf("Service %s stopped successfully", name)
	return nil
}

// StartAllServices starts all registered services
func (m *UnifiedServiceManager) StartAllServices() error {
	m.mutex.RLock()
	services := make([]interfaces.Service, 0, len(m.services))
	for _, service := range m.services {
		services = append(services, service)
	}
	m.mutex.RUnlock()

	var errors []error
	for _, service := range services {
		if !service.IsRunning() {
			if err := service.Start(m.ctx); err != nil {
				errors = append(errors, fmt.Errorf("failed to start service %s: %w", service.Name(), err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start some services: %v", errors)
	}

	log.Printf("All services started successfully")
	return nil
}

// StopAllServices stops all registered services
func (m *UnifiedServiceManager) StopAllServices() error {
	m.mutex.RLock()
	services := make([]interfaces.Service, 0, len(m.services))
	for _, service := range m.services {
		services = append(services, service)
	}
	m.mutex.RUnlock()

	var errors []error
	for _, service := range services {
		if service.IsRunning() {
			if err := service.Stop(m.ctx); err != nil {
				errors = append(errors, fmt.Errorf("failed to stop service %s: %w", service.Name(), err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some services: %v", errors)
	}

	log.Printf("All services stopped successfully")
	return nil
}

// GetService returns a service by name
func (m *UnifiedServiceManager) GetService(name string) (interfaces.Service, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	service, exists := m.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s is not registered", name)
	}

	return service, nil
}

// ListServices returns all registered services
func (m *UnifiedServiceManager) ListServices() []interfaces.Service {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	services := make([]interfaces.Service, 0, len(m.services))
	for _, service := range m.services {
		services = append(services, service)
	}

	return services
}

// GetRunningServices returns all currently running services
func (m *UnifiedServiceManager) GetRunningServices() []interfaces.Service {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var running []interfaces.Service
	for _, service := range m.services {
		if service.IsRunning() {
			running = append(running, service)
		}
	}

	return running
}

// Shutdown gracefully shuts down the service manager
func (m *UnifiedServiceManager) Shutdown() error {
	log.Printf("Shutting down unified service manager...")

	// Stop all services
	if err := m.StopAllServices(); err != nil {
		log.Printf("Warning: Error stopping services during shutdown: %v", err)
	}

	// Cancel context
	m.cancel()

	log.Printf("Unified service manager shutdown complete")
	return nil
}

// GetServiceStatuses returns status information for all services
func (m *UnifiedServiceManager) GetServiceStatuses() []interfaces.ServiceStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	statuses := make([]interfaces.ServiceStatus, 0, len(m.services))
	for _, service := range m.services {
		statuses = append(statuses, service.Status())
	}

	return statuses
}
