package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/knirv/network-monitor/internal/config"
	"github.com/knirv/network-monitor/internal/metrics"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sirupsen/logrus"
)

// Monitor represents the main monitoring system
type Monitor struct {
	config      *config.Config
	logger      *logrus.Logger
	services    map[string]*ServiceMonitor
	metrics     *metrics.Collector
	prometheus  v1.API
	testMonitor *TestMonitor
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// ServiceStatus represents the status of a monitored service
type ServiceStatus struct {
	Name         string                 `json:"name"`
	URL          string                 `json:"url"`
	Status       string                 `json:"status"` // "up", "down", "degraded", "unknown"
	LastCheck    time.Time              `json:"last_check"`
	ResponseTime time.Duration          `json:"response_time"`
	Error        string                 `json:"error,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Critical     bool                   `json:"critical"`
}

// NetworkStatus represents the overall network status
type NetworkStatus struct {
	Name          string                   `json:"name"`
	OverallStatus string                   `json:"overall_status"`
	ServicesUp    int                      `json:"services_up"`
	ServicesDown  int                      `json:"services_down"`
	ServicesTotal int                      `json:"services_total"`
	LastUpdate    time.Time                `json:"last_update"`
	Services      map[string]ServiceStatus `json:"services"`
}

// New creates a new monitoring system
func New(cfg *config.Config, logger *logrus.Logger) (*Monitor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &Monitor{
		config:      cfg,
		logger:      logger,
		services:    make(map[string]*ServiceMonitor),
		testMonitor: NewTestMonitor(logger),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize metrics collector
	metricsCollector, err := metrics.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics collector: %w", err)
	}
	monitor.metrics = metricsCollector

	// Initialize Prometheus client if configured
	if cfg.Monitoring.PrometheusConfig.URL != "" {
		client, err := api.NewClient(api.Config{
			Address: cfg.Monitoring.PrometheusConfig.URL,
		})
		if err != nil {
			logger.Warnf("Failed to initialize Prometheus client: %v", err)
		} else {
			monitor.prometheus = v1.NewAPI(client)
		}
	}

	return monitor, nil
}

// Start starts the monitoring system
func (m *Monitor) Start(ctx context.Context) error {
	m.logger.Info("Starting monitoring system...")

	// Get active network configuration
	network, err := m.config.GetActiveNetwork()
	if err != nil {
		return fmt.Errorf("failed to get active network: %w", err)
	}

	// Initialize service monitors
	for _, service := range network.Services {
		serviceMonitor := NewServiceMonitor(service, m.logger)
		m.services[service.Name] = serviceMonitor

		// Start monitoring this service
		go m.monitorService(ctx, serviceMonitor)
	}

	// Start metrics collection
	if err := m.metrics.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metrics collection: %w", err)
	}

	// Start test monitoring
	if err := m.testMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start test monitor: %w", err)
	}

	m.logger.Infof("Monitoring system started for network: %s", network.Name)
	return nil
}

// Stop stops the monitoring system
func (m *Monitor) Stop() {
	m.logger.Info("Stopping monitoring system...")
	m.cancel()

	if m.metrics != nil {
		m.metrics.Stop()
	}

	if m.testMonitor != nil {
		m.testMonitor.Stop()
	}

	m.logger.Info("Monitoring system stopped")
}

// GetTestMonitor returns the test monitor instance
func (m *Monitor) GetTestMonitor() *TestMonitor {
	return m.testMonitor
}

// GetNetworkStatus returns the current status of the monitored network
func (m *Monitor) GetNetworkStatus() *NetworkStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	network, _ := m.config.GetActiveNetwork()

	status := &NetworkStatus{
		Name:       network.Name,
		LastUpdate: time.Now(),
		Services:   make(map[string]ServiceStatus),
	}

	servicesUp := 0
	servicesDown := 0

	for name, monitor := range m.services {
		serviceStatus := monitor.GetStatus()
		status.Services[name] = serviceStatus

		if serviceStatus.Status == "up" {
			servicesUp++
		} else {
			servicesDown++
		}
	}

	status.ServicesUp = servicesUp
	status.ServicesDown = servicesDown
	status.ServicesTotal = len(m.services)

	// Determine overall status
	if servicesDown == 0 {
		status.OverallStatus = "healthy"
	} else if servicesUp > servicesDown {
		status.OverallStatus = "degraded"
	} else {
		status.OverallStatus = "critical"
	}

	return status
}

// GetServiceStatus returns the status of a specific service
func (m *Monitor) GetServiceStatus(serviceName string) (*ServiceStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	monitor, exists := m.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service '%s' not found", serviceName)
	}

	status := monitor.GetStatus()
	return &status, nil
}

// GetMetrics returns current metrics for the network
func (m *Monitor) GetMetrics() map[string]interface{} {
	if m.metrics == nil {
		return nil
	}
	return m.metrics.GetCurrentMetrics()
}

// QueryPrometheus executes a Prometheus query
func (m *Monitor) QueryPrometheus(query string) (interface{}, error) {
	if m.prometheus == nil {
		return nil, fmt.Errorf("Prometheus client not configured")
	}

	result, warnings, err := m.prometheus.Query(m.ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("Prometheus query failed: %w", err)
	}

	if len(warnings) > 0 {
		m.logger.Warnf("Prometheus query warnings: %v", warnings)
	}

	return result, nil
}

// monitorService continuously monitors a single service
func (m *Monitor) monitorService(ctx context.Context, monitor *ServiceMonitor) {
	ticker := time.NewTicker(monitor.service.CheckInterval)
	defer ticker.Stop()

	// Initial check
	monitor.Check()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			monitor.Check()
		}
	}
}

// ServiceMonitor monitors a single service
type ServiceMonitor struct {
	service config.Service
	logger  *logrus.Logger
	status  ServiceStatus
	client  *http.Client
	mu      sync.RWMutex
}

// NewServiceMonitor creates a new service monitor
func NewServiceMonitor(service config.Service, logger *logrus.Logger) *ServiceMonitor {
	return &ServiceMonitor{
		service: service,
		logger:  logger,
		client: &http.Client{
			Timeout: service.Timeout,
		},
		status: ServiceStatus{
			Name:     service.Name,
			URL:      service.URL,
			Status:   "unknown",
			Critical: service.Critical,
		},
	}
}

// Check performs a health check on the service
func (sm *ServiceMonitor) Check() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	start := time.Now()
	sm.status.LastCheck = start

	// Construct health check URL
	healthURL := sm.service.URL + sm.service.HealthEndpoint

	// Perform HTTP request
	resp, err := sm.client.Get(healthURL)
	responseTime := time.Since(start)
	sm.status.ResponseTime = responseTime

	if err != nil {
		sm.status.Status = "down"
		sm.status.Error = err.Error()
		sm.logger.Warnf("Service %s health check failed: %v", sm.service.Name, err)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		sm.status.Status = "up"
		sm.status.Error = ""
		sm.logger.Debugf("Service %s is healthy (response time: %v)", sm.service.Name, responseTime)
	} else {
		sm.status.Status = "degraded"
		sm.status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		sm.logger.Warnf("Service %s returned HTTP %d", sm.service.Name, resp.StatusCode)
	}

	// Collect additional metrics if endpoint is configured
	if sm.service.MetricsEndpoint != "" {
		sm.collectMetrics()
	}
}

// GetStatus returns the current status of the service
func (sm *ServiceMonitor) GetStatus() ServiceStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.status
}

// collectMetrics collects additional metrics from the service
func (sm *ServiceMonitor) collectMetrics() {
	metricsURL := sm.service.URL + sm.service.MetricsEndpoint

	resp, err := sm.client.Get(metricsURL)
	if err != nil {
		sm.logger.Debugf("Failed to collect metrics for %s: %v", sm.service.Name, err)
		return
	}
	defer resp.Body.Close()

	// Parse metrics response (implementation depends on metrics format)
	// This is a placeholder for metrics parsing logic
	sm.status.Metrics = map[string]interface{}{
		"metrics_collected_at": time.Now(),
		"metrics_endpoint":     metricsURL,
	}
}
