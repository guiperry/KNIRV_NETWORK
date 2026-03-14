package host

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// HostController is the main Kali Linux host interface
type HostController struct {
	SystemInfo   *SystemInfoCollector
	ProcessMgr   *ProcessManager
	NetworkMgr   *NetworkManager
	SecurityMgr  *SecurityManager
	ContainerMgr *ContainerManager
	
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	
	// Configuration
	config *HostConfig
	
	// Status tracking
	status     HostStatus
	lastUpdate time.Time
}

// HostConfig holds configuration for the host controller
type HostConfig struct {
	// System monitoring intervals
	SystemInfoInterval time.Duration `yaml:"system_info_interval" default:"30s"`
	ProcessInterval    time.Duration `yaml:"process_interval" default:"10s"`
	NetworkInterval    time.Duration `yaml:"network_interval" default:"15s"`
	SecurityInterval   time.Duration `yaml:"security_interval" default:"60s"`
	
	// Resource limits
	MaxProcesses     int     `yaml:"max_processes" default:"1000"`
	MaxMemoryPercent float64 `yaml:"max_memory_percent" default:"80.0"`
	MaxCPUPercent    float64 `yaml:"max_cpu_percent" default:"90.0"`
	
	// Security settings
	EnableTEE        bool     `yaml:"enable_tee" default:"true"`
	EnableAuditLog   bool     `yaml:"enable_audit_log" default:"true"`
	ThreatMonitoring bool     `yaml:"threat_monitoring" default:"true"`
	AllowedPorts     []int    `yaml:"allowed_ports"`
	BlockedIPs       []string `yaml:"blocked_ips"`
	
	// Container settings
	ContainerRuntime string `yaml:"container_runtime" default:"podman"`
	EnableContainers bool   `yaml:"enable_containers" default:"true"`
}

// HostStatus represents the current status of the host
type HostStatus string

const (
	HostStatusStarting HostStatus = "starting"
	HostStatusRunning  HostStatus = "running"
	HostStatusStopping HostStatus = "stopping"
	HostStatusStopped  HostStatus = "stopped"
	HostStatusError    HostStatus = "error"
)

// NewHostController creates a new host controller instance
func NewHostController(config *HostConfig) (*HostController, error) {
	if config == nil {
		config = &HostConfig{
			SystemInfoInterval: 30 * time.Second,
			ProcessInterval:    10 * time.Second,
			NetworkInterval:    15 * time.Second,
			SecurityInterval:   60 * time.Second,
			MaxProcesses:       1000,
			MaxMemoryPercent:   80.0,
			MaxCPUPercent:      90.0,
			EnableTEE:          true,
			EnableAuditLog:     true,
			ThreatMonitoring:   true,
			ContainerRuntime:   "podman",
			EnableContainers:   true,
		}
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	hc := &HostController{
		ctx:        ctx,
		cancel:     cancel,
		config:     config,
		status:     HostStatusStarting,
		lastUpdate: time.Now(),
	}
	
	// Initialize components
	var err error
	
	hc.SystemInfo, err = NewSystemInfoCollector(ctx, config.SystemInfoInterval)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create system info collector: %w", err)
	}
	
	hc.ProcessMgr, err = NewProcessManager(ctx, config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create process manager: %w", err)
	}
	
	hc.NetworkMgr, err = NewNetworkManager(ctx, config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create network manager: %w", err)
	}
	
	hc.SecurityMgr, err = NewSecurityManager(ctx, config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create security manager: %w", err)
	}
	
	if config.EnableContainers {
		hc.ContainerMgr, err = NewContainerManager(ctx, config)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create container manager: %w", err)
		}
	}
	
	return hc, nil
}

// Start begins all host monitoring and management services
func (hc *HostController) Start() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	if hc.status != HostStatusStarting {
		return fmt.Errorf("host controller is not in starting state")
	}
	
	log.Println("Starting KNIRV-SERVER Host Controller...")
	
	// Start all components
	if err := hc.SystemInfo.Start(); err != nil {
		return fmt.Errorf("failed to start system info collector: %w", err)
	}
	
	if err := hc.ProcessMgr.Start(); err != nil {
		return fmt.Errorf("failed to start process manager: %w", err)
	}
	
	if err := hc.NetworkMgr.Start(); err != nil {
		return fmt.Errorf("failed to start network manager: %w", err)
	}
	
	if err := hc.SecurityMgr.Start(); err != nil {
		return fmt.Errorf("failed to start security manager: %w", err)
	}
	
	if hc.ContainerMgr != nil {
		if err := hc.ContainerMgr.Start(); err != nil {
			return fmt.Errorf("failed to start container manager: %w", err)
		}
	}
	
	hc.status = HostStatusRunning
	hc.lastUpdate = time.Now()
	
	log.Println("KNIRV-SERVER Host Controller started successfully")
	return nil
}

// Stop gracefully shuts down all host services
func (hc *HostController) Stop() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	if hc.status != HostStatusRunning {
		return fmt.Errorf("host controller is not running")
	}
	
	log.Println("Stopping KNIRV-SERVER Host Controller...")
	hc.status = HostStatusStopping
	
	// Stop all components in reverse order
	var errors []error
	
	if hc.ContainerMgr != nil {
		if err := hc.ContainerMgr.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("container manager stop error: %w", err))
		}
	}
	
	if err := hc.SecurityMgr.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("security manager stop error: %w", err))
	}
	
	if err := hc.NetworkMgr.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("network manager stop error: %w", err))
	}
	
	if err := hc.ProcessMgr.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("process manager stop error: %w", err))
	}
	
	if err := hc.SystemInfo.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("system info collector stop error: %w", err))
	}
	
	// Cancel context
	hc.cancel()
	
	hc.status = HostStatusStopped
	hc.lastUpdate = time.Now()
	
	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}
	
	log.Println("KNIRV-SERVER Host Controller stopped successfully")
	return nil
}

// GetStatus returns the current status of the host controller
func (hc *HostController) GetStatus() (HostStatus, time.Time) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.status, hc.lastUpdate
}

// GetSystemInfo returns current system information
func (hc *HostController) GetSystemInfo() (*SystemInfo, error) {
	if hc.SystemInfo == nil {
		return nil, fmt.Errorf("system info collector not initialized")
	}
	return hc.SystemInfo.GetCurrentInfo()
}

// GetProcessList returns current process information
func (hc *HostController) GetProcessList() ([]*Process, error) {
	if hc.ProcessMgr == nil {
		return nil, fmt.Errorf("process manager not initialized")
	}
	return hc.ProcessMgr.GetProcessList()
}

// GetNetworkInfo returns current network information
func (hc *HostController) GetNetworkInfo() (*NetworkInfo, error) {
	if hc.NetworkMgr == nil {
		return nil, fmt.Errorf("network manager not initialized")
	}
	return hc.NetworkMgr.GetNetworkInfo()
}

// GetSecurityStatus returns current security status
func (hc *HostController) GetSecurityStatus() (*SecurityStatus, error) {
	if hc.SecurityMgr == nil {
		return nil, fmt.Errorf("security manager not initialized")
	}
	return hc.SecurityMgr.GetSecurityStatus()
}

// GetContainerList returns current container information
func (hc *HostController) GetContainerList() ([]*Container, error) {
	if hc.ContainerMgr == nil {
		return nil, fmt.Errorf("container manager not initialized or disabled")
	}
	return hc.ContainerMgr.GetContainerList()
}

// HealthCheck performs a comprehensive health check of all components
func (hc *HostController) HealthCheck() error {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	if hc.status != HostStatusRunning {
		return fmt.Errorf("host controller is not running (status: %s)", hc.status)
	}
	
	// Check all components
	if err := hc.SystemInfo.HealthCheck(); err != nil {
		return fmt.Errorf("system info collector health check failed: %w", err)
	}
	
	if err := hc.ProcessMgr.HealthCheck(); err != nil {
		return fmt.Errorf("process manager health check failed: %w", err)
	}
	
	if err := hc.NetworkMgr.HealthCheck(); err != nil {
		return fmt.Errorf("network manager health check failed: %w", err)
	}
	
	if err := hc.SecurityMgr.HealthCheck(); err != nil {
		return fmt.Errorf("security manager health check failed: %w", err)
	}
	
	if hc.ContainerMgr != nil {
		if err := hc.ContainerMgr.HealthCheck(); err != nil {
			return fmt.Errorf("container manager health check failed: %w", err)
		}
	}
	
	return nil
}
