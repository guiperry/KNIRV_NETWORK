package modelserver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// RuntimeManager manages live model runtime hosting
type RuntimeManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex

	activeModels map[string]*ModelInstance
	resourcePool *ResourcePool
	scheduler    *ModelScheduler
	processMgr   *NativeProcessManager

	// Configuration
	modelDir       string
	maxModels      int
	resourceLimits *ResourceLimits

	// Monitoring
	lastUpdate time.Time
	running    bool
}

// ModelInstance represents a running model instance
type ModelInstance struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Binary        string      `json:"binary"`
	Status        ModelStatus `json:"status"`
	PID           int         `json:"pid"`
	StartTime     time.Time   `json:"start_time"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`

	// Resource allocation
	Resources *ResourceAllocation `json:"resources"`
	Metrics   *ModelMetrics       `json:"metrics"`

	// Communication
	Communication *ModelComm `json:"communication"`

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

// ModelStatus represents the status of an model
type ModelStatus string

const (
	ModelStatusStarting   ModelStatus = "starting"
	ModelStatusRunning    ModelStatus = "running"
	ModelStatusStopping   ModelStatus = "stopping"
	ModelStatusStopped    ModelStatus = "stopped"
	ModelStatusFailed     ModelStatus = "failed"
	ModelStatusRestarting ModelStatus = "restarting"
)

// ResourceAllocation represents allocated resources for an model
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

// ModelMetrics represents runtime metrics for an model
type ModelMetrics struct {
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

// ModelComm represents communication settings for an model
type ModelComm struct {
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

// ModelScheduler handles model scheduling and placement
type ModelScheduler struct {
	mu sync.RWMutex

	schedulingPolicy string // round-robin, resource-aware, priority
	queue            []*ModelScheduleRequest
	running          bool
}

// ModelScheduleRequest represents a request to schedule an model
type ModelScheduleRequest struct {
	ModelName   string                 `json:"model_name"`
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
	ModelID   string    `json:"model_id"`
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
	MaxCPUPerModel    float64 `json:"max_cpu_per_model"`
	MaxMemoryPerModel uint64  `json:"max_memory_per_model"`
	MaxDiskPerModel   uint64  `json:"max_disk_per_model"`

	MaxTotalCPU    float64 `json:"max_total_cpu"`
	MaxTotalMemory uint64  `json:"max_total_memory"`
	MaxTotalDisk   uint64  `json:"max_total_disk"`

	DefaultCPU    float64 `json:"default_cpu"`
	DefaultMemory uint64  `json:"default_memory"`
	DefaultDisk   uint64  `json:"default_disk"`
}

// NewRuntimeManager creates a new runtime manager
func NewRuntimeManager(ctx context.Context, modelDir string, maxModels int) (*RuntimeManager, error) {
	runtimeCtx, cancel := context.WithCancel(ctx)

	rm := &RuntimeManager{
		ctx:          runtimeCtx,
		cancel:       cancel,
		activeModels: make(map[string]*ModelInstance),
		modelDir:     modelDir,
		maxModels:    maxModels,
		resourceLimits: &ResourceLimits{
			MaxCPUPerModel:    2.0,
			MaxMemoryPerModel: 1024 * 1024 * 1024,      // 1GB
			MaxDiskPerModel:   10 * 1024 * 1024 * 1024, // 10GB
			DefaultCPU:        0.5,
			DefaultMemory:     256 * 1024 * 1024,  // 256MB
			DefaultDisk:       1024 * 1024 * 1024, // 1GB
		},
	}

	// Initialize components
	var err error

	rm.resourcePool, err = NewResourcePool()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create resource pool: %w", err)
	}

	rm.scheduler, err = NewModelScheduler()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	rm.processMgr, err = NewNativeProcessManager()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create process manager: %w", err)
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

	// Stop all models
	for _, model := range rm.activeModels {
		if err := rm.stopModelInternal(model); err != nil {
			fmt.Printf("Error stopping model %s: %v\n", model.ID, err)
		}
	}

	// Stop components
	rm.scheduler.Stop()
	rm.processMgr.Stop()

	// Cancel context
	rm.cancel()

	return nil
}

// StartModel starts a new model instance
func (rm *RuntimeManager) StartModel(name, binary string, config map[string]interface{}) (*ModelInstance, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return nil, fmt.Errorf("runtime manager is not running")
	}

	// Check if model already exists
	for _, model := range rm.activeModels {
		if model.Name == name {
			return nil, fmt.Errorf("model %s is already running", name)
		}
	}

	// Check model limit
	if len(rm.activeModels) >= rm.maxModels {
		return nil, fmt.Errorf("maximum number of models (%d) reached", rm.maxModels)
	}

	// Verify binary exists
	binaryPath := filepath.Join(rm.modelDir, binary)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model binary %s not found", binary)
	}

	// Create model instance
	model := &ModelInstance{
		ID:            fmt.Sprintf("%s-%d", name, time.Now().Unix()),
		Name:          name,
		Binary:        binary,
		Status:        ModelStatusStarting,
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
	model.Resources = resources

	// Setup communication
	model.Communication = &ModelComm{
		SocketPath:          fmt.Sprintf("/tmp/knirv-model-%s.sock", model.ID),
		Protocol:            "unix",
		Encrypted:           true,
		HealthCheckPath:     "/health",
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}

	// Start the model process
	if err := rm.startModelProcess(model); err != nil {
		rm.resourcePool.ReleaseResources(resources)
		return nil, fmt.Errorf("failed to start model process: %w", err)
	}

	// Add to active models
	rm.activeModels[model.ID] = model

	return model, nil
}

// StopModel stops an model instance
func (rm *RuntimeManager) StopModel(modelID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	model, exists := rm.activeModels[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	return rm.stopModelInternal(model)
}

// GetModelList returns list of active models
func (rm *RuntimeManager) GetModelList() []*ModelInstance {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var models []*ModelInstance
	for _, model := range rm.activeModels {
		// Return a copy to prevent modification
		modelCopy := *model
		models = append(models, &modelCopy)
	}

	return models
}

// GetModel returns a specific model instance
func (rm *RuntimeManager) GetModel(modelID string) (*ModelInstance, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	model, exists := rm.activeModels[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	// Return a copy to prevent modification
	modelCopy := *model
	return &modelCopy, nil
}

// startModelProcess starts the actual model process
func (rm *RuntimeManager) startModelProcess(model *ModelInstance) error {
	binaryPath := filepath.Join(rm.modelDir, model.Binary)

	// Create command
	cmd := exec.Command(binaryPath, model.Arguments...)

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range model.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set working directory
	cmd.Dir = rm.modelDir

	// Setup resource isolation if available
	if rm.processMgr.cgroupManager.enabled {
		if err := rm.setupResourceIsolation(model); err != nil {
			return fmt.Errorf("failed to setup resource isolation: %w", err)
		}
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	model.Command = cmd
	model.Process = cmd.Process
	model.PID = cmd.Process.Pid
	model.Status = ModelStatusRunning
	model.LastHeartbeat = time.Now()

	// Register with process manager
	processInfo := &ProcessInfo{
		PID:       model.PID,
		ModelID:   model.ID,
		Command:   binaryPath,
		StartTime: model.StartTime,
		Status:    "running",
	}

	rm.processMgr.mu.Lock()
	rm.processMgr.processes[model.PID] = processInfo
	rm.processMgr.mu.Unlock()

	// Start monitoring the process
	go rm.monitorModelProcess(model)

	return nil
}

// stopModelInternal stops an model (internal method, assumes lock is held)
func (rm *RuntimeManager) stopModelInternal(model *ModelInstance) error {
	model.Status = ModelStatusStopping

	if model.Process != nil {
		// Send SIGTERM first
		if err := model.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			_, err := model.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(10 * time.Second):
			// Force kill after timeout
			if err := model.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	}

	// Clean up resources
	if model.Resources != nil {
		rm.resourcePool.ReleaseResources(model.Resources)
	}

	// Clean up communication socket
	if model.Communication != nil && model.Communication.SocketPath != "" {
		os.Remove(model.Communication.SocketPath)
	}

	// Remove from process manager
	if model.PID > 0 {
		rm.processMgr.mu.Lock()
		delete(rm.processMgr.processes, model.PID)
		rm.processMgr.mu.Unlock()
	}

	// Remove from active models
	delete(rm.activeModels, model.ID)

	model.Status = ModelStatusStopped

	return nil
}

// setupResourceIsolation sets up cgroup-based resource isolation
func (rm *RuntimeManager) setupResourceIsolation(model *ModelInstance) error {
	// Create cgroup for the model
	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/knirv-models/%s", model.ID)
	model.Resources.CgroupPath = cgroupPath

	// This would implement actual cgroup setup
	// For now, just set the systemd slice
	model.Resources.SystemdSlice = fmt.Sprintf("knirv-model-%s.slice", model.ID)

	return nil
}

// monitorModelProcess monitors an model process
func (rm *RuntimeManager) monitorModelProcess(model *ModelInstance) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			// Check if process is still running
			if model.Process != nil {
				if err := model.Process.Signal(syscall.Signal(0)); err != nil {
					// Process is dead
					model.Status = ModelStatusFailed

					// Handle restart policy
					if model.RestartPolicy == "always" ||
						(model.RestartPolicy == "on-failure" && model.RestartCount < model.MaxRestarts) {
						rm.restartModel(model)
					}
					return
				}
			}

			// Update metrics
			rm.updateModelMetrics(model)
		}
	}
}

// restartModel restarts a failed model
func (rm *RuntimeManager) restartModel(model *ModelInstance) {
	model.RestartCount++
	model.Status = ModelStatusRestarting

	// Wait a bit before restarting
	time.Sleep(5 * time.Second)

	// Restart the process
	if err := rm.startModelProcess(model); err != nil {
		fmt.Printf("Failed to restart model %s: %v\n", model.ID, err)
		model.Status = ModelStatusFailed
	}
}

// updateModelMetrics updates model metrics
func (rm *RuntimeManager) updateModelMetrics(model *ModelInstance) {
	if model.Metrics == nil {
		model.Metrics = &ModelMetrics{}
	}

	// This would implement actual metrics collection
	// For now, just update the timestamp
	model.Metrics.CollectedAt = time.Now()
	model.LastHeartbeat = time.Now()
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

// healthCheckLoop runs health checks on models
func (rm *RuntimeManager) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.mu.RLock()
			models := make([]*ModelInstance, 0, len(rm.activeModels))
			for _, model := range rm.activeModels {
				models = append(models, model)
			}
			rm.mu.RUnlock()

			// Perform health checks
			for _, model := range models {
				rm.performHealthCheck(model)
			}
		}
	}
}

// performHealthCheck performs a health check on an model
func (rm *RuntimeManager) performHealthCheck(model *ModelInstance) {
	if model.Status != ModelStatusRunning {
		return
	}

	// This would implement actual health checking
	// For now, just check if the process is alive
	if model.Process != nil {
		if err := model.Process.Signal(syscall.Signal(0)); err != nil {
			model.Status = ModelStatusFailed
			if model.Metrics != nil {
				model.Metrics.HealthScore = 0.0
			}
		} else {
			if model.Metrics != nil {
				model.Metrics.HealthScore = 1.0
				model.Metrics.LastHealthCheck = time.Now()
			}
		}
	}
}
