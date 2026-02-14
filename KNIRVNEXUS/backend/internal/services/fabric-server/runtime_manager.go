package fabricserver

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

// RuntimeManager manages live fabric runtime hosting
type RuntimeManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex

	activeFabrics map[string]*FabricInstance
	resourcePool  *ResourcePool
	scheduler     *FabricScheduler
	processMgr    *NativeProcessManager
	wasmRuntime   *WASMRuntime

	// Configuration
	fabricDir      string
	maxFabrics     int
	resourceLimits *ResourceLimits

	// Monitoring
	lastUpdate time.Time
	running    bool
}

// FabricInstance represents a running fabric instance
type FabricInstance struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Binary        string       `json:"binary"`
	Status        FabricStatus `json:"status"`
	PID           int          `json:"pid"`
	StartTime     time.Time    `json:"start_time"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`

	// Resource allocation
	Resources *ResourceAllocation `json:"resources"`
	Metrics   *FabricMetrics      `json:"metrics"`

	// Communication
	Communication *FabricComm `json:"communication"`

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

// FabricStatus represents the status of a fabric item
type FabricStatus string

const (
	FabricStatusStarting   FabricStatus = "starting"
	FabricStatusRunning    FabricStatus = "running"
	FabricStatusStopping   FabricStatus = "stopping"
	FabricStatusStopped    FabricStatus = "stopped"
	FabricStatusFailed     FabricStatus = "failed"
	FabricStatusRestarting FabricStatus = "restarting"
)

// ResourceAllocation represents allocated resources for a fabric unit
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

// FabricMetrics represents runtime metrics for a fabric item
type FabricMetrics struct {
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

// FabricComm represents communication settings for a fabric item
type FabricComm struct {
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

// FabricScheduler handles fabric scheduling and placement
type FabricScheduler struct {
	mu sync.RWMutex

	schedulingPolicy string // round-robin, resource-aware, priority
	queue            []*FabricScheduleRequest
	running          bool
}

// FabricScheduleRequest represents a request to schedule a fabric item
type FabricScheduleRequest struct {
	FabricName  string                 `json:"fabric_name"`
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
	FabricID  string    `json:"fabric_id"`
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
	MaxCPUPerFabric    float64 `json:"max_cpu_per_fabric"`
	MaxMemoryPerFabric uint64  `json:"max_memory_per_fabric"`
	MaxDiskPerFabric   uint64  `json:"max_disk_per_fabric"`

	MaxTotalCPU    float64 `json:"max_total_cpu"`
	MaxTotalMemory uint64  `json:"max_total_memory"`
	MaxTotalDisk   uint64  `json:"max_total_disk"`

	DefaultCPU    float64 `json:"default_cpu"`
	DefaultMemory uint64  `json:"default_memory"`
	DefaultDisk   uint64  `json:"default_disk"`
}

// NewRuntimeManager creates a new runtime manager
func NewRuntimeManager(ctx context.Context, fabricDir string, maxFabrics int) (*RuntimeManager, error) {
	runtimeCtx, cancel := context.WithCancel(ctx)

	rm := &RuntimeManager{
		ctx:           runtimeCtx,
		cancel:        cancel,
		activeFabrics: make(map[string]*FabricInstance),
		fabricDir:     fabricDir,
		maxFabrics:    maxFabrics,
		resourceLimits: &ResourceLimits{
			MaxCPUPerFabric:    2.0,
			MaxMemoryPerFabric: 1024 * 1024 * 1024,      // 1GB
			MaxDiskPerFabric:   10 * 1024 * 1024 * 1024, // 10GB
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

	rm.scheduler, err = NewFabricScheduler()
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
		ResourceLimits:   nil, // Will be set per fabric unit
	}
	rm.wasmRuntime, err = NewWASMRuntime(wasmConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create WASM runtime: %w", err)
	}

	// Set default resource limits for the WASM runtime
	defaultLimits := &objects.FabricResourceLimits{
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

	// Stop all fabrics
	for _, fabric := range rm.activeFabrics {
		if err := rm.stopFabricInternal(fabric); err != nil {
			fmt.Printf("Error stopping fabric item %s: %v\n", fabric.ID, err)
		}
	}

	// Stop components
	rm.scheduler.Stop()
	rm.processMgr.Stop()

	// Cancel context
	rm.cancel()

	return nil
}

// StartFabric starts a new fabric instance
func (rm *RuntimeManager) StartFabric(name, binary string, config map[string]interface{}) (*FabricInstance, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return nil, fmt.Errorf("runtime manager is not running")
	}

	// Check if fabric item already exists
	for _, fabric := range rm.activeFabrics {
		if fabric.Name == name {
			return nil, fmt.Errorf("fabric item %s is already running", name)
		}
	}

	// Check fabric limit
	if len(rm.activeFabrics) >= rm.maxFabrics {
		return nil, fmt.Errorf("maximum number of fabric units (%d) reached", rm.maxFabrics)
	}

	// Verify binary exists
	binaryPath := filepath.Join(rm.fabricDir, binary)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("fabric binary %s not found", binary)
	}

	// Create fabric instance
	fabric := &FabricInstance{
		ID:            fmt.Sprintf("%s-%d", name, time.Now().Unix()),
		Name:          name,
		Binary:        binary,
		Status:        FabricStatusStarting,
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
	fabric.Resources = resources

	// Setup communication
	fabric.Communication = &FabricComm{
		SocketPath:          fmt.Sprintf("/tmp/knirv-fabric-%s.sock", fabric.ID),
		Protocol:            "unix",
		Encrypted:           true,
		HealthCheckPath:     "/health",
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}

	// Start the fabric process
	if err := rm.startFabricProcess(fabric); err != nil {
		rm.resourcePool.ReleaseResources(resources)
		return nil, fmt.Errorf("failed to start fabric process: %w", err)
	}

	// Add to active fabrics
	rm.activeFabrics[fabric.ID] = fabric

	return fabric, nil
}

// StopFabric stops a fabric instance
func (rm *RuntimeManager) StopFabric(fabricID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	fabric, exists := rm.activeFabrics[fabricID]
	if !exists {
		return fmt.Errorf("fabric item %s not found", fabricID)
	}

	return rm.stopFabricInternal(fabric)
}

// GetFabricList returns list of active fabric units
func (rm *RuntimeManager) GetFabricList() []*FabricInstance {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var fabricUnits []*FabricInstance
	for _, fabric := range rm.activeFabrics {
		// Return a copy to prevent modification
		fabricCopy := *fabric
		fabricUnits = append(fabricUnits, &fabricCopy)
	}

	return fabricUnits
}

// GetFabric returns a specific fabric instance
func (rm *RuntimeManager) GetFabric(fabricID string) (*FabricInstance, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	fabric, exists := rm.activeFabrics[fabricID]
	if !exists {
		return nil, fmt.Errorf("fabric item %s not found", fabricID)
	}

	// Return a copy to prevent modification
	fabricCopy := *fabric
	return &fabricCopy, nil
}

// startFabricProcess starts the actual fabric process
func (rm *RuntimeManager) startFabricProcess(fabric *FabricInstance) error {
	binaryPath := filepath.Join(rm.fabricDir, fabric.Binary)

	// Create command
	cmd := exec.Command(binaryPath, fabric.Arguments...)

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range fabric.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set working directory
	cmd.Dir = rm.fabricDir

	// Setup resource isolation if available
	if rm.processMgr.cgroupManager.enabled {
		if err := rm.setupResourceIsolation(fabric); err != nil {
			return fmt.Errorf("failed to setup resource isolation: %w", err)
		}
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	fabric.Command = cmd
	fabric.Process = cmd.Process
	fabric.PID = cmd.Process.Pid
	fabric.Status = FabricStatusRunning
	fabric.LastHeartbeat = time.Now()

	// Register with process manager
	processInfo := &ProcessInfo{
		PID:       fabric.PID,
		FabricID:  fabric.ID,
		Command:   binaryPath,
		StartTime: fabric.StartTime,
		Status:    "running",
	}

	rm.processMgr.mu.Lock()
	rm.processMgr.processes[fabric.PID] = processInfo
	rm.processMgr.mu.Unlock()

	// Start monitoring the process
	go rm.monitorFabricProcess(fabric)

	return nil
}

// stopFabricInternal stops a fabric item (internal method, assumes lock is held)
func (rm *RuntimeManager) stopFabricInternal(fabric *FabricInstance) error {
	fabric.Status = FabricStatusStopping

	if fabric.Process != nil {
		// Send SIGTERM first
		if err := fabric.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			_, err := fabric.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(10 * time.Second):
			// Force kill after timeout
			if err := fabric.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	}

	// Clean up resources
	if fabric.Resources != nil {
		rm.resourcePool.ReleaseResources(fabric.Resources)
	}

	// Clean up communication socket
	if fabric.Communication != nil && fabric.Communication.SocketPath != "" {
		os.Remove(fabric.Communication.SocketPath)
	}

	// Remove from process manager
	if fabric.PID > 0 {
		rm.processMgr.mu.Lock()
		delete(rm.processMgr.processes, fabric.PID)
		rm.processMgr.mu.Unlock()
	}

	// Remove from active fabric units
	delete(rm.activeFabrics, fabric.ID)

	fabric.Status = FabricStatusStopped

	return nil
}

// setupResourceIsolation sets up cgroup-based resource isolation
func (rm *RuntimeManager) setupResourceIsolation(fabric *FabricInstance) error {
	// Create cgroup for the fabric item
	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/knirv-objects/%s", fabric.ID)
	fabric.Resources.CgroupPath = cgroupPath

	// This would implement actual cgroup setup
	// For now, just set the systemd slice
	fabric.Resources.SystemdSlice = fmt.Sprintf("knirv-fabric-%s.slice", fabric.ID)

	return nil
}

// monitorFabricProcess monitors a fabric process
func (rm *RuntimeManager) monitorFabricProcess(fabric *FabricInstance) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			// Check if process is still running
			if fabric.Process != nil {
				if err := fabric.Process.Signal(syscall.Signal(0)); err != nil {
					// Process is dead
					fabric.Status = FabricStatusFailed

					// Handle restart policy
					if fabric.RestartPolicy == "always" ||
						(fabric.RestartPolicy == "on-failure" && fabric.RestartCount < fabric.MaxRestarts) {
						rm.restartFabric(fabric)
					}
					return
				}
			}

			// Update metrics
			rm.updateFabricMetrics(fabric)
		}
	}
}

// restartFabric restarts a failed fabric unit
func (rm *RuntimeManager) restartFabric(fabric *FabricInstance) {
	fabric.RestartCount++
	fabric.Status = FabricStatusRestarting

	// Wait a bit before restarting
	time.Sleep(5 * time.Second)

	// Restart the process
	if err := rm.startFabricProcess(fabric); err != nil {
		fmt.Printf("Failed to restart fabric item %s: %v\n", fabric.ID, err)
		fabric.Status = FabricStatusFailed
	}
}

// updateFabricMetrics updates fabric metrics
func (rm *RuntimeManager) updateFabricMetrics(fabric *FabricInstance) {
	if fabric.Metrics == nil {
		fabric.Metrics = &FabricMetrics{}
	}

	// This would implement actual metrics collection
	// For now, just update the timestamp
	fabric.Metrics.CollectedAt = time.Now()
	fabric.LastHeartbeat = time.Now()
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

// healthCheckLoop runs health checks on fabric units
func (rm *RuntimeManager) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.mu.RLock()
			fabricUnits := make([]*FabricInstance, 0, len(rm.activeFabrics))
			for _, fabric := range rm.activeFabrics {
				fabricUnits = append(fabricUnits, fabric)
			}
			rm.mu.RUnlock()

			// Perform health checks
			for _, fabric := range fabricUnits {
				rm.performHealthCheck(fabric)
			}
		}
	}
}

// performHealthCheck performs a health check on a fabric item
func (rm *RuntimeManager) performHealthCheck(fabric *FabricInstance) {
	if fabric.Status != FabricStatusRunning {
		return
	}

	// This would implement actual health checking
	// For now, just check if the process is alive
	if fabric.Process != nil {
		if err := fabric.Process.Signal(syscall.Signal(0)); err != nil {
			fabric.Status = FabricStatusFailed
			if fabric.Metrics != nil {
				fabric.Metrics.HealthScore = 0.0
			}
		} else {
			if fabric.Metrics != nil {
				fabric.Metrics.HealthScore = 1.0
				fabric.Metrics.LastHealthCheck = time.Now()
			}
		}
	}
}
