package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guiperry/KNIRVCHAIN-CLI/config"
	"github.com/sirupsen/logrus"
)

// ServiceStatus represents the status of a service
type ServiceStatus string

const (
	ServiceStatusHealthy   ServiceStatus = "healthy"
	ServiceStatusUnhealthy ServiceStatus = "unhealthy"
	ServiceStatusUnknown   ServiceStatus = "unknown"
)

// ServiceEndpoint represents a discovered service endpoint
type ServiceEndpoint struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Status       ServiceStatus     `json:"status"`
	Capabilities []string          `json:"capabilities"`
	Metadata     map[string]string `json:"metadata"`
	LastSeen     time.Time         `json:"last_seen"`
	Config       *config.ServiceConfig `json:"-"`
}

// ServiceDiscovery handles automatic discovery of KNIRV services
type ServiceDiscovery struct {
	registry *ServiceRegistry
	config   *config.Config
	logger   *logrus.Logger
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// ServiceRegistry manages discovered services and their health status
type ServiceRegistry struct {
	services map[string]*ServiceEndpoint
	config   *config.Config
	logger   *logrus.Logger
	mu       sync.RWMutex
	discovery *ServiceDiscovery
	healthMonitor *HealthMonitor
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(cfg *config.Config, logger *logrus.Logger) *ServiceRegistry {
	registry := &ServiceRegistry{
		services: make(map[string]*ServiceEndpoint),
		config:   cfg,
		logger:   logger,
	}

	// Initialize service discovery
	registry.discovery = &ServiceDiscovery{
		registry: registry,
		config:   cfg,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}

	// Initialize health monitor
	registry.healthMonitor = NewHealthMonitor(registry, logger)

	return registry
}

// Start starts the service registry and discovery
func (sr *ServiceRegistry) Start(ctx context.Context) error {
	sr.logger.Info("Starting service registry")

	// Register configured services
	if err := sr.registerConfiguredServices(); err != nil {
		return fmt.Errorf("failed to register configured services: %w", err)
	}

	// Start service discovery if enabled
	if sr.config.KNIRV.Network.Discovery.Enabled {
		sr.discovery.Start(ctx)
	}

	// Start health monitoring
	sr.healthMonitor.Start(ctx)

	return nil
}

// Stop stops the service registry and discovery
func (sr *ServiceRegistry) Stop() {
	sr.logger.Info("Stopping service registry")
	
	if sr.discovery != nil {
		sr.discovery.Stop()
	}
	
	if sr.healthMonitor != nil {
		sr.healthMonitor.Stop()
	}
}

// RegisterService registers a service endpoint
func (sr *ServiceRegistry) RegisterService(endpoint *ServiceEndpoint) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	endpoint.LastSeen = time.Now()
	sr.services[endpoint.Name] = endpoint
	sr.logger.Infof("Registered service: %s at %s", endpoint.Name, endpoint.URL)
}

// GetService retrieves a service endpoint by name
func (sr *ServiceRegistry) GetService(name string) (*ServiceEndpoint, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	service, exists := sr.services[name]
	return service, exists
}

// GetAllServices returns all registered services
func (sr *ServiceRegistry) GetAllServices() map[string]*ServiceEndpoint {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	services := make(map[string]*ServiceEndpoint)
	for name, service := range sr.services {
		services[name] = service
	}
	return services
}

// GetHealthyServices returns only healthy services
func (sr *ServiceRegistry) GetHealthyServices() map[string]*ServiceEndpoint {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	services := make(map[string]*ServiceEndpoint)
	for name, service := range sr.services {
		if service.Status == ServiceStatusHealthy {
			services[name] = service
		}
	}
	return services
}

// UpdateServiceStatus updates the status of a service
func (sr *ServiceRegistry) UpdateServiceStatus(name string, status ServiceStatus) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if service, exists := sr.services[name]; exists {
		oldStatus := service.Status
		service.Status = status
		service.LastSeen = time.Now()
		
		if oldStatus != status {
			sr.logger.Infof("Service %s status changed from %s to %s", name, oldStatus, status)
		}
	}
}

// registerConfiguredServices registers services from configuration
func (sr *ServiceRegistry) registerConfiguredServices() error {
	services := sr.config.KNIRV.Services

	// Register KNIRVROOT
	if services.KNIRVRoot.Enabled {
		endpoint := &ServiceEndpoint{
			Name:         "knirvroot",
			URL:          services.KNIRVRoot.URL,
			Status:       ServiceStatusUnknown,
			Capabilities: []string{"blockchain", "economics", "agent-management"},
			Metadata:     map[string]string{"type": "knirvroot"},
			Config:       &services.KNIRVRoot,
		}
		sr.RegisterService(endpoint)
	}

	// Register KNIRVGATEWAY
	if services.KNIRVGateway.Enabled {
		endpoint := &ServiceEndpoint{
			Name:         "knirvgateway",
			URL:          services.KNIRVGateway.URL,
			Status:       ServiceStatusUnknown,
			Capabilities: []string{"gateway", "proxy", "health-monitoring"},
			Metadata:     map[string]string{"type": "knirvgateway"},
			Config:       &services.KNIRVGateway,
		}
		sr.RegisterService(endpoint)
	}

	// Register KNIRVNEXUS
	if services.KNIRVNexus.Enabled {
		endpoint := &ServiceEndpoint{
			Name:         "knirvnexus",
			URL:          services.KNIRVNexus.URL,
			Status:       ServiceStatusUnknown,
			Capabilities: []string{"dve", "inference", "validation"},
			Metadata:     map[string]string{"type": "knirvnexus"},
			Config:       &services.KNIRVNexus,
		}
		sr.RegisterService(endpoint)
	}

	// Register KNIRVGRAPH
	if services.KNIRVGraph.Enabled {
		endpoint := &ServiceEndpoint{
			Name:         "knirvgraph",
			URL:          services.KNIRVGraph.URL,
			Status:       ServiceStatusUnknown,
			Capabilities: []string{"graph", "nrv", "skills", "errors"},
			Metadata:     map[string]string{"type": "knirvgraph"},
			Config:       &services.KNIRVGraph,
		}
		sr.RegisterService(endpoint)
	}

	return nil
}

// Start starts the service discovery process
func (sd *ServiceDiscovery) Start(ctx context.Context) {
	sd.logger.Info("Starting service discovery")
	
	sd.wg.Add(1)
	go sd.discoveryLoop(ctx)
}

// Stop stops the service discovery process
func (sd *ServiceDiscovery) Stop() {
	sd.logger.Info("Stopping service discovery")
	close(sd.stopCh)
	sd.wg.Wait()
}

// discoveryLoop runs the periodic service discovery
func (sd *ServiceDiscovery) discoveryLoop(ctx context.Context) {
	defer sd.wg.Done()

	ticker := time.NewTicker(sd.config.KNIRV.Network.Discovery.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sd.stopCh:
			return
		case <-ticker.C:
			sd.discoverServices(ctx)
		}
	}
}

// discoverServices performs service discovery
func (sd *ServiceDiscovery) discoverServices(ctx context.Context) {
	sd.logger.Debug("Performing service discovery")
	
	// For now, we'll just validate that configured services are reachable
	// In the future, this could be extended to discover services via DNS, consul, etc.
	
	services := sd.registry.GetAllServices()
	for name, service := range services {
		go sd.checkServiceHealth(ctx, name, service)
	}
}

// checkServiceHealth checks if a service is healthy
func (sd *ServiceDiscovery) checkServiceHealth(ctx context.Context, name string, service *ServiceEndpoint) {
	// Create a context with timeout for the health check
	checkCtx, cancel := context.WithTimeout(ctx, sd.config.KNIRV.Network.Discovery.Timeout)
	defer cancel()

	// Create API client for health check
	client := NewAPIClient(service.URL, WithTimeout(sd.config.KNIRV.Network.Discovery.Timeout))
	
	// Try to ping the service
	err := client.Get(checkCtx, "/health", nil)
	if err != nil {
		sd.logger.Debugf("Health check failed for %s: %v", name, err)
		sd.registry.UpdateServiceStatus(name, ServiceStatusUnhealthy)
	} else {
		sd.logger.Debugf("Health check passed for %s", name)
		sd.registry.UpdateServiceStatus(name, ServiceStatusHealthy)
	}
}
