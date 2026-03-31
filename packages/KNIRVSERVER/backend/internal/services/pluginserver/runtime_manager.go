package pluginserver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"backend_server/internal/objects"
)

// RuntimeManager manages live plugin runtime hosting
type RuntimeManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex

	activePlugins map[string]*PluginInstance
	resourcePool  *ResourcePool
	scheduler     *PluginScheduler
	processMgr    *NativeProcessManager
	wasmRuntime   *WASMRuntime

	// Configuration
	pluginDir      string
	maxPlugins     int
	resourceLimits *ResourceLimits

	// Monitoring
	lastUpdate time.Time
	running    bool
}

// PluginInstance represents a running plugin instance
type PluginInstance struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Binary        string       `json:"binary"`
	Status        PluginStatus `json:"status"`
	PID           int          `json:"pid"`
	StartTime     time.Time    `json:"start_time"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`

	// Resource allocation
	Resources *ResourceAllocation `json:"resources"`
	Metrics   *PluginMetrics      `json:"metrics"`

	// Communication
	Communication *PluginComm `json:"communication"`

	// Process management
	Process *os.Process `json:"-"`
	Command *exec.Cmd   `json:"-"`

	// Configuration
	Config      map[string]interface{} `json:"config"`
	Environment map[string]string      `json:"environment"`
	Arguments   []string               `json:"arguments"`

	// Lifecycle
	RestartCount  int    `json:"restart_count"`
	MaxRestarts   int    `json:"max_restarts"`
	RestartPolicy string `json:"restart_policy"` // always, on-failure, never

	// Security
	UserID       int      `json:"user_id"`
	GroupID      int      `json:"group_id"`
	Capabilities []string `json:"capabilities"`
	Chroot       string   `json:"chroot,omitempty"`
}

// PluginStatus represents the status of a plugin item
type PluginStatus string

const (
	PluginStatusStarting   PluginStatus = "starting"
	PluginStatusRunning    PluginStatus = "running"
	PluginStatusStopping   PluginStatus = "stopping"
	PluginStatusStopped    PluginStatus = "stopped"
	PluginStatusFailed     PluginStatus = "failed"
	PluginStatusRestarting PluginStatus = "restarting"
)

// ResourceAllocation represents allocated resources for a plugin unit
type ResourceAllocation struct {
	CPUCores         float64 `json:"cpu_cores"`
	MemoryBytes      uint64  `json:"memory_bytes"`
	DiskBytes        uint64  `json:"disk_bytes"`
	NetworkBandwidth uint64  `json:"network_bandwidth"`

	// Limits
	CPULimit    float64 `json:"cpu_limit"`
	MemoryLimit uint64  `json:"memory_limit"`
	DiskLimit   uint64  `json:"disk_limit"`

	// Cgroup settings
	CgroupPath   string `json:"cgroup_path"`
	SystemdSlice string `json:"systemd_slice"`
}

// PluginMetrics represents runtime metrics for a plugin item
type PluginMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage uint64  `json:"memory_usage"`
	DiskUsage   uint64  `json:"disk_usage"`
	NetworkRx   uint64  `json:"network_rx"`
	NetworkTx   uint64  `json:"network_tx"`

	// Performance metrics
	RequestCount uint64  `json:"request_count"`
	ErrorCount   uint64  `json:"error_count"`
	ResponseTime float64 `json:"response_time_ms"`

	// Health metrics
	HealthScore     float64   `json:"health_score"`
	LastHealthCheck time.Time `json:"last_health_check"`

	// Collection timestamp
	CollectedAt time.Time `json:"collected_at"`
}

// PluginComm represents communication settings for a plugin item
type PluginComm struct {
	SocketPath string `json:"socket_path"`
	Port       int    `json:"port,omitempty"`
	Protocol   string `json:"protocol"` // unix, tcp, http
	Encrypted  bool   `json:"encrypted"`

	// Health check settings
	HealthCheckPath     string        `json:"health_check_path"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
}

// ResourcePool manages available system resources
type ResourcePool struct {
	mu sync.RWMutex

	TotalCPU    float64 `json:"total_cpu"`
	TotalMemory uint64  `json:"total_memory"`
	TotalDisk   uint64  `json:"total_disk"`

	AvailableCPU    float64 `json:"available_cpu"`
	AvailableMemory uint64  `json:"available_memory"`
	AvailableDisk   uint64  `json:"available_disk"`

	AllocatedCPU    float64 `json:"allocated_cpu"`
	AllocatedMemory uint64  `json:"allocated_memory"`
	AllocatedDisk   uint64  `json:"allocated_disk"`
}

// PluginScheduler handles plugin scheduling and placement
type PluginScheduler struct {
	mu sync.RWMutex

	schedulingPolicy string // round-robin, resource-aware, priority
	queue            []*PluginScheduleRequest
	running          bool
}

// PluginScheduleRequest represents a request to schedule a plugin item
type PluginScheduleRequest struct {
	PluginName  string                 `json:"plugin_name"`
	Binary      string                 `json:"binary"`
	Resources   *ResourceAllocation    `json:"resources"`
	Config      map[string]interface{} `json:"config"`
	Priority    int                    `json:"priority"`
	RequestTime time.Time              `json:"request_time"`
}

// NativeProcessManager manages native process execution
type NativeProcessManager struct {
	mu sync.RWMutex

	processes     map[int]*ProcessInfo
	cgroupManager *CgroupManager
}

// ProcessInfo contains information about a managed process
type ProcessInfo struct {
	PID       int       `json:"pid"`
	PluginID  string    `json:"plugin_id"`
	Command   string    `json:"command"`
	StartTime time.Time `json:"start_time"`
	Status    string    `json:"status"`
	ExitCode  int       `json:"exit_code,omitempty"`

	// Resource isolation
	CgroupPath   string `json:"cgroup_path"`
	SystemdSlice string `json:"systemd_slice"`
	Namespace    string `json:"namespace,omitempty"`
}

// CgroupManager manages cgroup-based resource isolation
type CgroupManager struct {
	cgroupRoot    string
	cgroupVersion int // 1 or 2
	enabled       bool
}

// ResourceLimits defines system-wide resource limits
type ResourceLimits struct {
	MaxCPUPerPlugin    float64 `json:"max_cpu_per_plugin"`
	MaxMemoryPerPlugin uint64  `json:"max_memory_per_plugin"`
	MaxDiskPerPlugin   uint64  `json:"max_disk_per_plugin"`

	MaxTotalCPU    float64 `json:"max_total_cpu"`
	MaxTotalMemory uint64  `json:"max_total_memory"`
	MaxTotalDisk   uint64  `json:"max_total_disk"`

	DefaultCPU    float64 `json:"default_cpu"`
	DefaultMemory uint64  `json:"default_memory"`
	DefaultDisk   uint64  `json:"default_disk"`
}

// NewRuntimeManager creates a new runtime manager
func NewRuntimeManager(ctx context.Context, pluginDir string, maxPlugins int) (*RuntimeManager, error) {
	runtimeCtx, cancel := context.WithCancel(ctx)

	rm := &RuntimeManager{
		ctx:           runtimeCtx,
		cancel:        cancel,
		activePlugins: make(map[string]*PluginInstance),
		pluginDir:     pluginDir,
		maxPlugins:    maxPlugins,
		resourceLimits: &ResourceLimits{
			MaxCPUPerPlugin:    2.0,
			MaxMemoryPerPlugin: 1024 * 1024 * 1024,      // 1GB
			MaxDiskPerPlugin:   10 * 1024 * 1024 * 1024, // 10GB
			DefaultCPU:         0.5,
			DefaultMemory:      256 * 1024 * 1024,  // 256MB
			DefaultDisk:        1024 * 1024 * 1024, // 1GB
		},
	}

	// Initialize components
	var err error

	rm.resourcePool, err = NewResourcePool()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create resource pool: %w", err)
	}

	rm.scheduler, err = NewPluginScheduler()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	rm.processMgr, err = NewNativeProcessManager()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create process manager: %w", err)
	}

	// Initialize WASM runtime
	wasmConfig := &WASMConfig{
		MaxMemoryPages:   1024, // 64MB
		MaxExecutionTime: 30 * time.Second,
		MaxInstances:     10,
		EnableProfiling:  true,
		EnableDebugging:  false,
		ResourceLimits:   nil, // Will be set per plugin unit
	}
	rm.wasmRuntime, err = NewWASMRuntime(wasmConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create WASM runtime: %w", err)
	}

	// Set default resource limits for the WASM runtime
	defaultLimits := &objects.PluginResourceLimits{
		MaxCPUPercent:    50.0, // 50% CPU limit
		MaxMemoryMB:      256,  // 256MB memory limit
		MaxExecutionTime: 30,   // 30 seconds execution time
	}

	if err := rm.wasmRuntime.setupResourceLimits(defaultLimits); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to set default resource limits: %w", err)
	}

	return rm, nil
}

// Start begins the runtime manager
func (rm *RuntimeManager) Start() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.running {
		return fmt.Errorf("runtime manager is already running")
	}

	rm.running = true

	// Start scheduler
	if err := rm.scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// Start process manager
	if err := rm.processMgr.Start(); err != nil {
		return fmt.Errorf("failed to start process manager: %w", err)
	}

	// Start monitoring loop
	go rm.monitorLoop()

	// Start health check loop
	go rm.healthCheckLoop()

	return nil
}

// Stop stops the runtime manager
func (rm *RuntimeManager) Stop() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return fmt.Errorf("runtime manager is not running")
	}

	rm.running = false

	// Stop all plugins
	for _, plugin := range rm.activePlugins {
		if err := rm.stopPluginInternal(plugin); err != nil {
			fmt.Printf("Error stopping plugin item %s: %v\n", plugin.ID, err)
		}
	}

	// Stop components
	rm.scheduler.Stop()
	rm.processMgr.Stop()

	// Cancel context
	rm.cancel()

	return nil
}

// StartPlugin starts a new plugin instance
func (rm *RuntimeManager) StartPlugin(name, binary string, config map[string]interface{}) (*PluginInstance, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return nil, fmt.Errorf("runtime manager is not running")
	}

	// Check if plugin item already exists
	for _, plugin := range rm.activePlugins {
		if plugin.Name == name {
			return nil, fmt.Errorf("plugin item %s is already running", name)
		}
	}

	// Check plugin limit
	if len(rm.activePlugins) >= rm.maxPlugins {
		return nil, fmt.Errorf("maximum number of plugin units (%d) reached", rm.maxPlugins)
	}

	// Verify binary exists
	binaryPath := filepath.Join(rm.pluginDir, binary)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin binary %s not found", binary)
	}

	// Create plugin instance
	plugin := &PluginInstance{
		ID:            fmt.Sprintf("%s-%d", name, time.Now().Unix()),
		Name:          name,
		Binary:        binary,
		Status:        PluginStatusStarting,
		StartTime:     time.Now(),
		Config:        config,
		Environment:   make(map[string]string),
		MaxRestarts:   3,
		RestartPolicy: "on-failure",
		UserID:        os.Getuid(),
		GroupID:       os.Getgid(),
	}

	// Allocate resources
	resources, err := rm.resourcePool.AllocateResources(&ResourceAllocation{
		CPUCores:    rm.resourceLimits.DefaultCPU,
		MemoryBytes: rm.resourceLimits.DefaultMemory,
		DiskBytes:   rm.resourceLimits.DefaultDisk,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to allocate resources: %w", err)
	}
	plugin.Resources = resources

	// Setup communication
	plugin.Communication = &PluginComm{
		SocketPath:          fmt.Sprintf("/tmp/knirv-plugin-%s.sock", plugin.ID),
		Protocol:            "unix",
		Encrypted:           true,
		HealthCheckPath:     "/health",
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}

	// Start the plugin process
	if err := rm.startPluginProcess(plugin); err != nil {
		rm.resourcePool.ReleaseResources(resources)
		return nil, fmt.Errorf("failed to start plugin process: %w", err)
	}

	// Add to active plugins
	rm.activePlugins[plugin.ID] = plugin

	return plugin, nil
}

// StopPlugin stops a plugin instance
func (rm *RuntimeManager) StopPlugin(pluginID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	plugin, exists := rm.activePlugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin item %s not found", pluginID)
	}

	return rm.stopPluginInternal(plugin)
}

// GetPluginList returns list of active plugin units
func (rm *RuntimeManager) GetPluginList() []*PluginInstance {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var pluginUnits []*PluginInstance
	for _, plugin := range rm.activePlugins {
		// Return a copy to prevent modification
		pluginCopy := *plugin
		pluginUnits = append(pluginUnits, &pluginCopy)
	}

	return pluginUnits
}

// GetPlugin returns a specific plugin instance
func (rm *RuntimeManager) GetPlugin(pluginID string) (*PluginInstance, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	plugin, exists := rm.activePlugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin item %s not found", pluginID)
	}

	// Return a copy to prevent modification
	pluginCopy := *plugin
	return &pluginCopy, nil
}

// startPluginProcess starts the actual plugin process
func (rm *RuntimeManager) startPluginProcess(plugin *PluginInstance) error {
	binaryPath := filepath.Join(rm.pluginDir, plugin.Binary)

	// Create command
	cmd := exec.Command(binaryPath, plugin.Arguments...)

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range plugin.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set working directory
	cmd.Dir = rm.pluginDir

	// Setup resource isolation if available
	if rm.processMgr.cgroupManager.enabled {
		if err := rm.setupResourceIsolation(plugin); err != nil {
			return fmt.Errorf("failed to setup resource isolation: %w", err)
		}
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	plugin.Command = cmd
	plugin.Process = cmd.Process
	plugin.PID = cmd.Process.Pid
	plugin.Status = PluginStatusRunning
	plugin.LastHeartbeat = time.Now()

	// Register with process manager
	processInfo := &ProcessInfo{
		PID:       plugin.PID,
		PluginID:  plugin.ID,
		Command:   binaryPath,
		StartTime: plugin.StartTime,
		Status:    "running",
	}

	rm.processMgr.mu.Lock()
	rm.processMgr.processes[plugin.PID] = processInfo
	rm.processMgr.mu.Unlock()

	// Start monitoring the process
	go rm.monitorPluginProcess(plugin)

	return nil
}

// stopPluginInternal stops a plugin item (internal method, assumes lock is held)
func (rm *RuntimeManager) stopPluginInternal(plugin *PluginInstance) error {
	plugin.Status = PluginStatusStopping

	if plugin.Process != nil {
		// Send SIGTERM first
		if err := plugin.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			_, err := plugin.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(10 * time.Second):
			// Force kill after timeout
			if err := plugin.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	}

	// Clean up resources
	if plugin.Resources != nil {
		rm.resourcePool.ReleaseResources(plugin.Resources)
	}

	// Clean up communication socket
	if plugin.Communication != nil && plugin.Communication.SocketPath != "" {
		os.Remove(plugin.Communication.SocketPath)
	}

	// Remove from process manager
	if plugin.PID > 0 {
		rm.processMgr.mu.Lock()
		delete(rm.processMgr.processes, plugin.PID)
		rm.processMgr.mu.Unlock()
	}

	// Remove from active plugin units
	delete(rm.activePlugins, plugin.ID)

	plugin.Status = PluginStatusStopped

	return nil
}

// setupResourceIsolation sets up cgroup-based resource isolation
func (rm *RuntimeManager) setupResourceIsolation(plugin *PluginInstance) error {
	// Create cgroup for the plugin item
	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/knirv-objects/%s", plugin.ID)
	plugin.Resources.CgroupPath = cgroupPath

	// This would implement actual cgroup setup
	// For now, just set the systemd slice
	plugin.Resources.SystemdSlice = fmt.Sprintf("knirv-plugin-%s.slice", plugin.ID)

	return nil
}

// monitorPluginProcess monitors a plugin process
func (rm *RuntimeManager) monitorPluginProcess(plugin *PluginInstance) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			// Check if process is still running
			if plugin.Process != nil {
				if err := plugin.Process.Signal(syscall.Signal(0)); err != nil {
					// Process is dead
					plugin.Status = PluginStatusFailed

					// Handle restart policy
					if plugin.RestartPolicy == "always" ||
						(plugin.RestartPolicy == "on-failure" && plugin.RestartCount < plugin.MaxRestarts) {
						rm.restartPlugin(plugin)
					}
					return
				}
			}

			// Update metrics
			rm.updatePluginMetrics(plugin)
		}
	}
}

// restartPlugin restarts a failed plugin unit
func (rm *RuntimeManager) restartPlugin(plugin *PluginInstance) {
	plugin.RestartCount++
	plugin.Status = PluginStatusRestarting

	// Wait a bit before restarting
	time.Sleep(5 * time.Second)

	// Restart the process
	if err := rm.startPluginProcess(plugin); err != nil {
		fmt.Printf("Failed to restart plugin item %s: %v\n", plugin.ID, err)
		plugin.Status = PluginStatusFailed
	}
}

// updatePluginMetrics updates plugin metrics
func (rm *RuntimeManager) updatePluginMetrics(plugin *PluginInstance) {
	if plugin.Metrics == nil {
		plugin.Metrics = &PluginMetrics{}
	}

	// This would implement actual metrics collection
	// For now, just update the timestamp
	plugin.Metrics.CollectedAt = time.Now()
	plugin.LastHeartbeat = time.Now()
}

// monitorLoop runs the main monitoring loop
func (rm *RuntimeManager) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.mu.RLock()
			running := rm.running
			rm.mu.RUnlock()

			if !running {
				return
			}

			// Update resource pool
			rm.resourcePool.UpdateAvailableResources()

			// Process scheduling queue
			rm.scheduler.ProcessQueue()

			rm.lastUpdate = time.Now()
		}
	}
}

// healthCheckLoop runs health checks on plugin units
func (rm *RuntimeManager) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.mu.RLock()
			pluginUnits := make([]*PluginInstance, 0, len(rm.activePlugins))
			for _, plugin := range rm.activePlugins {
				pluginUnits = append(pluginUnits, plugin)
			}
			rm.mu.RUnlock()

			// Perform health checks
			for _, plugin := range pluginUnits {
				rm.performHealthCheck(plugin)
			}
		}
	}
}

// performHealthCheck performs a health check on a plugin item
func (rm *RuntimeManager) performHealthCheck(plugin *PluginInstance) {
	if plugin.Status != PluginStatusRunning {
		return
	}

	// This would implement actual health checking
	// For now, just check if the process is alive
	if plugin.Process != nil {
		if err := plugin.Process.Signal(syscall.Signal(0)); err != nil {
			plugin.Status = PluginStatusFailed
			if plugin.Metrics != nil {
				plugin.Metrics.HealthScore = 0.0
			}
		} else {
			if plugin.Metrics != nil {
				plugin.Metrics.HealthScore = 1.0
				plugin.Metrics.LastHealthCheck = time.Now()
			}
		}
	}
}
