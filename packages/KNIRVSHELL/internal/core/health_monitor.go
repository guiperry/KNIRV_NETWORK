package core

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	ServiceName  string        `json:"service_name"`
	Status       ServiceStatus `json:"status"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// HealthMonitor monitors the health of registered services
type HealthMonitor struct {
	registry    *ServiceRegistry
	logger      *logrus.Logger
	stopCh      chan struct{}
	wg          sync.WaitGroup
	results     map[string]*HealthCheckResult
	resultsMu   sync.RWMutex
	subscribers []chan *HealthCheckResult
	subsMu      sync.RWMutex
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(registry *ServiceRegistry, logger *logrus.Logger) *HealthMonitor {
	return &HealthMonitor{
		registry:    registry,
		logger:      logger,
		stopCh:      make(chan struct{}),
		results:     make(map[string]*HealthCheckResult),
		subscribers: make([]chan *HealthCheckResult, 0),
	}
}

// Start starts the health monitoring
func (hm *HealthMonitor) Start(ctx context.Context) {
	hm.logger.Info("Starting health monitor")

	hm.wg.Add(1)
	go hm.monitorLoop(ctx)
}

// Stop stops the health monitoring
func (hm *HealthMonitor) Stop() {
	hm.logger.Info("Stopping health monitor")
	close(hm.stopCh)
	hm.wg.Wait()

	// Close all subscriber channels
	hm.subsMu.Lock()
	for _, ch := range hm.subscribers {
		close(ch)
	}
	hm.subscribers = nil
	hm.subsMu.Unlock()
}

// Subscribe subscribes to health check results
func (hm *HealthMonitor) Subscribe() <-chan *HealthCheckResult {
	hm.subsMu.Lock()
	defer hm.subsMu.Unlock()

	ch := make(chan *HealthCheckResult, 10)
	hm.subscribers = append(hm.subscribers, ch)
	return ch
}

// GetHealthStatus returns the current health status of all services
func (hm *HealthMonitor) GetHealthStatus() map[string]*HealthCheckResult {
	hm.resultsMu.RLock()
	defer hm.resultsMu.RUnlock()

	status := make(map[string]*HealthCheckResult)
	for name, result := range hm.results {
		status[name] = result
	}
	return status
}

// GetServiceHealth returns the health status of a specific service
func (hm *HealthMonitor) GetServiceHealth(serviceName string) (*HealthCheckResult, bool) {
	hm.resultsMu.RLock()
	defer hm.resultsMu.RUnlock()

	result, exists := hm.results[serviceName]
	return result, exists
}

// monitorLoop runs the health monitoring loop
func (hm *HealthMonitor) monitorLoop(ctx context.Context) {
	defer hm.wg.Done()

	// Initial health check
	hm.performHealthChecks(ctx)

	// Set up periodic health checks every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.stopCh:
			return
		case <-ticker.C:
			hm.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks performs health checks on all registered services
func (hm *HealthMonitor) performHealthChecks(ctx context.Context) {
	services := hm.registry.GetAllServices()

	for name, service := range services {
		go hm.checkServiceHealth(ctx, name, service)
	}
}

// checkServiceHealth performs a health check on a specific service
func (hm *HealthMonitor) checkServiceHealth(ctx context.Context, name string, service *ServiceEndpoint) {
	startTime := time.Now()

	result := &HealthCheckResult{
		ServiceName: name,
		Timestamp:   startTime,
	}

	// Create a context with timeout for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Determine health endpoint
	healthEndpoint := "/health"
	if service.Config != nil && service.Config.Endpoints.Health != "" {
		healthEndpoint = service.Config.Endpoints.Health
	}

	// Create API client for health check
	timeout := 10 * time.Second
	if service.Config != nil && service.Config.Timeout > 0 {
		timeout = service.Config.Timeout
	}

	client := NewAPIClient(service.URL, WithTimeout(timeout))

	// Perform health check
	var healthResponse interface{}
	err := client.Get(checkCtx, healthEndpoint, &healthResponse)

	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Status = ServiceStatusUnhealthy
		result.Error = err.Error()
		hm.logger.Debugf("Health check failed for %s: %v (response time: %v)", name, err, result.ResponseTime)
	} else {
		result.Status = ServiceStatusHealthy
		hm.logger.Debugf("Health check passed for %s (response time: %v)", name, result.ResponseTime)
	}

	// Update service status in registry
	hm.registry.UpdateServiceStatus(name, result.Status)

	// Store result
	hm.resultsMu.Lock()
	hm.results[name] = result
	hm.resultsMu.Unlock()

	// Notify subscribers
	hm.notifySubscribers(result)
}

// notifySubscribers notifies all subscribers of a health check result
func (hm *HealthMonitor) notifySubscribers(result *HealthCheckResult) {
	hm.subsMu.RLock()
	defer hm.subsMu.RUnlock()

	for _, ch := range hm.subscribers {
		select {
		case ch <- result:
		default:
			// Channel is full, skip this subscriber
			hm.logger.Warn("Health check subscriber channel is full, skipping notification")
		}
	}
}

// GetOverallHealth returns the overall health status of the system
func (hm *HealthMonitor) GetOverallHealth() ServiceStatus {
	hm.resultsMu.RLock()
	defer hm.resultsMu.RUnlock()

	if len(hm.results) == 0 {
		return ServiceStatusUnknown
	}

	healthyCount := 0
	totalCount := len(hm.results)

	for _, result := range hm.results {
		if result.Status == ServiceStatusHealthy {
			healthyCount++
		}
	}

	// If more than 50% of services are healthy, consider overall status healthy
	if float64(healthyCount)/float64(totalCount) > 0.5 {
		return ServiceStatusHealthy
	}

	return ServiceStatusUnhealthy
}

// GetHealthSummary returns a summary of service health
func (hm *HealthMonitor) GetHealthSummary() map[string]interface{} {
	hm.resultsMu.RLock()
	defer hm.resultsMu.RUnlock()

	summary := map[string]interface{}{
		"overall_status":     hm.GetOverallHealth(),
		"total_services":     len(hm.results),
		"healthy_services":   0,
		"unhealthy_services": 0,
		"unknown_services":   0,
		"services":           make(map[string]interface{}),
	}

	services := make(map[string]interface{})

	for name, result := range hm.results {
		switch result.Status {
		case ServiceStatusHealthy:
			summary["healthy_services"] = summary["healthy_services"].(int) + 1
		case ServiceStatusUnhealthy:
			summary["unhealthy_services"] = summary["unhealthy_services"].(int) + 1
		case ServiceStatusUnknown:
			summary["unknown_services"] = summary["unknown_services"].(int) + 1
		}

		services[name] = map[string]interface{}{
			"status":        result.Status,
			"response_time": result.ResponseTime.String(),
			"last_check":    result.Timestamp.Format(time.RFC3339),
			"error":         result.Error,
		}
	}

	summary["services"] = services
	return summary
}
