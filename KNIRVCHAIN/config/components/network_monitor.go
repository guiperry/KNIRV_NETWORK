package components

import (
	"fmt"
	"strconv"
	"time"
)

// NetworkMonitorConfig represents network monitor component configuration
type NetworkMonitorConfig struct {
	Enabled           bool   `json:"enabled" mapstructure:"enabled"`
	Port              string `json:"port" mapstructure:"port"`
	Interval          int    `json:"interval" mapstructure:"interval"`
	LogLevel          string `json:"log_level" mapstructure:"log_level"`
	MetricsEnabled    bool   `json:"metrics_enabled" mapstructure:"metrics_enabled"`
	MetricsPort       string `json:"metrics_port" mapstructure:"metrics_port"`
	HealthCheckPath   string `json:"health_check_path" mapstructure:"health_check_path"`
	MaxRetries        int    `json:"max_retries" mapstructure:"max_retries"`
	TimeoutSeconds    int    `json:"timeout_seconds" mapstructure:"timeout_seconds"`
	AlertingEnabled   bool   `json:"alerting_enabled" mapstructure:"alerting_enabled"`
	AlertThresholds   AlertThresholds `json:"alert_thresholds" mapstructure:"alert_thresholds"`
	MonitoredServices []string `json:"monitored_services" mapstructure:"monitored_services"`
}

// AlertThresholds represents alerting thresholds for network monitoring
type AlertThresholds struct {
	CPUPercent    float64 `json:"cpu_percent" mapstructure:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent" mapstructure:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent" mapstructure:"disk_percent"`
	ResponseTimeMS int    `json:"response_time_ms" mapstructure:"response_time_ms"`
}

// GetConfigKey returns the configuration key prefix for network monitor
func (c *NetworkMonitorConfig) GetConfigKey() string {
	return "network_monitor"
}

// Validate validates the network monitor configuration
func (c *NetworkMonitorConfig) Validate() error {
	if c.Enabled {
		// Validate port
		if c.Port == "" {
			return fmt.Errorf("network monitor port cannot be empty when enabled")
		}
		
		if port, err := strconv.Atoi(c.Port); err != nil {
			return fmt.Errorf("invalid network monitor port format: %s", c.Port)
		} else if port <= 0 || port > 65535 {
			return fmt.Errorf("invalid network monitor port: %d (must be 1-65535)", port)
		}
		
		// Validate interval
		if c.Interval <= 0 {
			return fmt.Errorf("network monitor interval must be positive, got: %d", c.Interval)
		}
		
		if c.Interval < 5 {
			return fmt.Errorf("network monitor interval too low: %d (minimum 5 seconds)", c.Interval)
		}
		
		// Validate log level
		validLogLevels := map[string]bool{
			"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
		}
		if !validLogLevels[c.LogLevel] {
			return fmt.Errorf("invalid network monitor log level: %s (must be debug, info, warn, error, or fatal)", c.LogLevel)
		}
		
		// Validate metrics port if metrics are enabled
		if c.MetricsEnabled {
			if c.MetricsPort == "" {
				return fmt.Errorf("network monitor metrics port cannot be empty when metrics enabled")
			}
			
			if port, err := strconv.Atoi(c.MetricsPort); err != nil {
				return fmt.Errorf("invalid network monitor metrics port format: %s", c.MetricsPort)
			} else if port <= 0 || port > 65535 {
				return fmt.Errorf("invalid network monitor metrics port: %d (must be 1-65535)", port)
			}
			
			// Ensure metrics port is different from main port
			if c.MetricsPort == c.Port {
				return fmt.Errorf("network monitor metrics port cannot be the same as main port: %s", c.Port)
			}
		}
		
		// Validate health check path
		if c.HealthCheckPath == "" {
			return fmt.Errorf("network monitor health check path cannot be empty when enabled")
		}
		
		// Validate max retries
		if c.MaxRetries < 0 || c.MaxRetries > 10 {
			return fmt.Errorf("invalid network monitor max retries: %d (must be 0-10)", c.MaxRetries)
		}
		
		// Validate timeout
		if c.TimeoutSeconds <= 0 || c.TimeoutSeconds > 300 {
			return fmt.Errorf("invalid network monitor timeout: %d (must be 1-300 seconds)", c.TimeoutSeconds)
		}
		
		// Validate alert thresholds if alerting is enabled
		if c.AlertingEnabled {
			if err := c.validateAlertThresholds(); err != nil {
				return fmt.Errorf("network monitor alert thresholds validation failed: %w", err)
			}
		}
		
		// Validate monitored services list
		if len(c.MonitoredServices) == 0 {
			return fmt.Errorf("network monitor must have at least one monitored service when enabled")
		}
	}
	
	return nil
}

// validateAlertThresholds validates the alert threshold configuration
func (c *NetworkMonitorConfig) validateAlertThresholds() error {
	if c.AlertThresholds.CPUPercent < 0 || c.AlertThresholds.CPUPercent > 100 {
		return fmt.Errorf("invalid CPU alert threshold: %f (must be 0-100)", c.AlertThresholds.CPUPercent)
	}
	
	if c.AlertThresholds.MemoryPercent < 0 || c.AlertThresholds.MemoryPercent > 100 {
		return fmt.Errorf("invalid memory alert threshold: %f (must be 0-100)", c.AlertThresholds.MemoryPercent)
	}
	
	if c.AlertThresholds.DiskPercent < 0 || c.AlertThresholds.DiskPercent > 100 {
		return fmt.Errorf("invalid disk alert threshold: %f (must be 0-100)", c.AlertThresholds.DiskPercent)
	}
	
	if c.AlertThresholds.ResponseTimeMS < 0 || c.AlertThresholds.ResponseTimeMS > 60000 {
		return fmt.Errorf("invalid response time alert threshold: %d (must be 0-60000 ms)", c.AlertThresholds.ResponseTimeMS)
	}
	
	return nil
}

// GetDefaults returns default configuration values for network monitor
func (c *NetworkMonitorConfig) GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		"enabled":            false,
		"port":               "8091",
		"interval":           30,
		"log_level":          "info",
		"metrics_enabled":    true,
		"metrics_port":       "9091",
		"health_check_path":  "/health",
		"max_retries":        3,
		"timeout_seconds":    10,
		"alerting_enabled":   false,
		"alert_thresholds": map[string]interface{}{
			"cpu_percent":      80.0,
			"memory_percent":   85.0,
			"disk_percent":     90.0,
			"response_time_ms": 5000,
		},
		"monitored_services": []string{
			"economics-service",
			"operator-registry",
			"tunnel-registry",
			"payment-gateway",
		},
	}
}

// GetEnvironmentMappings returns environment variable mappings for network monitor
func (c *NetworkMonitorConfig) GetEnvironmentMappings() map[string]string {
	return map[string]string{
		"network_monitor.enabled":                        "KNIRV_NETWORK_MONITOR_ENABLED",
		"network_monitor.port":                           "KNIRV_NETWORK_MONITOR_PORT",
		"network_monitor.interval":                       "KNIRV_NETWORK_MONITOR_INTERVAL",
		"network_monitor.log_level":                      "KNIRV_NETWORK_MONITOR_LOG_LEVEL",
		"network_monitor.metrics_enabled":                "KNIRV_NETWORK_MONITOR_METRICS_ENABLED",
		"network_monitor.metrics_port":                   "KNIRV_NETWORK_MONITOR_METRICS_PORT",
		"network_monitor.health_check_path":              "KNIRV_NETWORK_MONITOR_HEALTH_CHECK_PATH",
		"network_monitor.max_retries":                    "KNIRV_NETWORK_MONITOR_MAX_RETRIES",
		"network_monitor.timeout_seconds":                "KNIRV_NETWORK_MONITOR_TIMEOUT_SECONDS",
		"network_monitor.alerting_enabled":               "KNIRV_NETWORK_MONITOR_ALERTING_ENABLED",
		"network_monitor.alert_thresholds.cpu_percent":   "KNIRV_NETWORK_MONITOR_CPU_THRESHOLD",
		"network_monitor.alert_thresholds.memory_percent": "KNIRV_NETWORK_MONITOR_MEMORY_THRESHOLD",
		"network_monitor.alert_thresholds.disk_percent":  "KNIRV_NETWORK_MONITOR_DISK_THRESHOLD",
		"network_monitor.alert_thresholds.response_time_ms": "KNIRV_NETWORK_MONITOR_RESPONSE_TIME_THRESHOLD",
	}
}

// NewNetworkMonitorConfig creates a new network monitor configuration with defaults
func NewNetworkMonitorConfig() *NetworkMonitorConfig {
	return &NetworkMonitorConfig{
		Enabled:         false,
		Port:            "8091",
		Interval:        30,
		LogLevel:        "info",
		MetricsEnabled:  true,
		MetricsPort:     "9091",
		HealthCheckPath: "/health",
		MaxRetries:      3,
		TimeoutSeconds:  10,
		AlertingEnabled: false,
		AlertThresholds: AlertThresholds{
			CPUPercent:     80.0,
			MemoryPercent:  85.0,
			DiskPercent:    90.0,
			ResponseTimeMS: 5000,
		},
		MonitoredServices: []string{
			"economics-service",
			"operator-registry",
			"tunnel-registry",
			"payment-gateway",
		},
	}
}

// GetPortInt returns the port as an integer
func (c *NetworkMonitorConfig) GetPortInt() (int, error) {
	return strconv.Atoi(c.Port)
}

// GetMetricsPortInt returns the metrics port as an integer
func (c *NetworkMonitorConfig) GetMetricsPortInt() (int, error) {
	return strconv.Atoi(c.MetricsPort)
}

// GetEndpoint returns the network monitor endpoint URL
func (c *NetworkMonitorConfig) GetEndpoint() string {
	return fmt.Sprintf("http://localhost:%s", c.Port)
}

// GetMetricsEndpoint returns the metrics endpoint URL
func (c *NetworkMonitorConfig) GetMetricsEndpoint() string {
	if c.MetricsEnabled {
		return fmt.Sprintf("http://localhost:%s/metrics", c.MetricsPort)
	}
	return ""
}

// GetHealthCheckURL returns the health check URL
func (c *NetworkMonitorConfig) GetHealthCheckURL() string {
	return fmt.Sprintf("http://localhost:%s%s", c.Port, c.HealthCheckPath)
}

// GetIntervalDuration returns the interval as a time.Duration
func (c *NetworkMonitorConfig) GetIntervalDuration() time.Duration {
	return time.Duration(c.Interval) * time.Second
}

// GetTimeoutDuration returns the timeout as a time.Duration
func (c *NetworkMonitorConfig) GetTimeoutDuration() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// IsServiceMonitored returns true if the specified service is being monitored
func (c *NetworkMonitorConfig) IsServiceMonitored(serviceName string) bool {
	for _, service := range c.MonitoredServices {
		if service == serviceName {
			return true
		}
	}
	return false
}

// AddMonitoredService adds a service to the monitored services list
func (c *NetworkMonitorConfig) AddMonitoredService(serviceName string) {
	if !c.IsServiceMonitored(serviceName) {
		c.MonitoredServices = append(c.MonitoredServices, serviceName)
	}
}

// RemoveMonitoredService removes a service from the monitored services list
func (c *NetworkMonitorConfig) RemoveMonitoredService(serviceName string) {
	for i, service := range c.MonitoredServices {
		if service == serviceName {
			c.MonitoredServices = append(c.MonitoredServices[:i], c.MonitoredServices[i+1:]...)
			break
		}
	}
}
