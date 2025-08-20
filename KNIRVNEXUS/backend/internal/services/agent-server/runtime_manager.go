package main

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

// RuntimeManager manages live agent runtime hosting
type RuntimeManager struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex

	activeAgents map[string]*AgentInstance
	resourcePool *ResourcePool
	scheduler    *AgentScheduler
	processMgr   *NativeProcessManager

	// Configuration
	agentDir       string
	maxAgents      int
	resourceLimits *ResourceLimits

	// Monitoring
	lastUpdate time.Time
	running    bool
}

// AgentInstance represents a running agent instance
type AgentInstance struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Binary        string      `json:"binary"`
	Status        AgentStatus `json:"status"`
	PID           int         `json:"pid"`
	StartTime     time.Time   `json:"start_time"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`

	// Resource allocation
	Resources *ResourceAllocation `json:"resources"`
	Metrics   *AgentMetrics       `json:"metrics"`

	// Communication
	Communication *AgentComm `json:"communication"`

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

// AgentStatus represents the status of an agent
type AgentStatus string

const (
	AgentStatusStarting   AgentStatus = "starting"
	AgentStatusRunning    AgentStatus = "running"
	AgentStatusStopping   AgentStatus = "stopping"
	AgentStatusStopped    AgentStatus = "stopped"
	AgentStatusFailed     AgentStatus = "failed"
	AgentStatusRestarting AgentStatus = "restarting"
)

// ResourceAllocation represents allocated resources for an agent
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

// AgentMetrics represents runtime metrics for an agent
type AgentMetrics struct {
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

// AgentComm represents communication settings for an agent
type AgentComm struct {
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

// AgentScheduler handles agent scheduling and placement
type AgentScheduler struct {
	mu sync.RWMutex

	schedulingPolicy string // round-robin, resource-aware, priority
	queue            []*AgentScheduleRequest
	running          bool
}

// AgentScheduleRequest represents a request to schedule an agent
type AgentScheduleRequest struct {
	AgentName   string                 `json:"agent_name"`
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
	AgentID   string    `json:"agent_id"`
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
	mu sync.RWMutex

	cgroupRoot    string
	cgroupVersion int // 1 or 2
	enabled       bool
}

// ResourceLimits defines system-wide resource limits
type ResourceLimits struct {
	MaxCPUPerAgent    float64 `json:"max_cpu_per_agent"`
	MaxMemoryPerAgent uint64  `json:"max_memory_per_agent"`
	MaxDiskPerAgent   uint64  `json:"max_disk_per_agent"`

	MaxTotalCPU    float64 `json:"max_total_cpu"`
	MaxTotalMemory uint64  `json:"max_total_memory"`
	MaxTotalDisk   uint64  `json:"max_total_disk"`

	DefaultCPU    float64 `json:"default_cpu"`
	DefaultMemory uint64  `json:"default_memory"`
	DefaultDisk   uint64  `json:"default_disk"`
}

// NewRuntimeManager creates a new runtime manager
func NewRuntimeManager(ctx context.Context, agentDir string, maxAgents int) (*RuntimeManager, error) {
	runtimeCtx, cancel := context.WithCancel(ctx)

	rm := &RuntimeManager{
		ctx:          runtimeCtx,
		cancel:       cancel,
		activeAgents: make(map[string]*AgentInstance),
		agentDir:     agentDir,
		maxAgents:    maxAgents,
		resourceLimits: &ResourceLimits{
			MaxCPUPerAgent:    2.0,
			MaxMemoryPerAgent: 1024 * 1024 * 1024,      // 1GB
			MaxDiskPerAgent:   10 * 1024 * 1024 * 1024, // 10GB
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

	rm.scheduler, err = NewAgentScheduler()
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

	// Stop all agents
	for _, agent := range rm.activeAgents {
		if err := rm.stopAgentInternal(agent); err != nil {
			fmt.Printf("Error stopping agent %s: %v\n", agent.ID, err)
		}
	}

	// Stop components
	rm.scheduler.Stop()
	rm.processMgr.Stop()

	// Cancel context
	rm.cancel()

	return nil
}

// StartAgent starts a new agent instance
func (rm *RuntimeManager) StartAgent(name, binary string, config map[string]interface{}) (*AgentInstance, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return nil, fmt.Errorf("runtime manager is not running")
	}

	// Check if agent already exists
	for _, agent := range rm.activeAgents {
		if agent.Name == name {
			return nil, fmt.Errorf("agent %s is already running", name)
		}
	}

	// Check agent limit
	if len(rm.activeAgents) >= rm.maxAgents {
		return nil, fmt.Errorf("maximum number of agents (%d) reached", rm.maxAgents)
	}

	// Verify binary exists
	binaryPath := filepath.Join(rm.agentDir, binary)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("agent binary %s not found", binary)
	}

	// Create agent instance
	agent := &AgentInstance{
		ID:            fmt.Sprintf("%s-%d", name, time.Now().Unix()),
		Name:          name,
		Binary:        binary,
		Status:        AgentStatusStarting,
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
	agent.Resources = resources

	// Setup communication
	agent.Communication = &AgentComm{
		SocketPath:          fmt.Sprintf("/tmp/knirv-agent-%s.sock", agent.ID),
		Protocol:            "unix",
		Encrypted:           true,
		HealthCheckPath:     "/health",
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}

	// Start the agent process
	if err := rm.startAgentProcess(agent); err != nil {
		rm.resourcePool.ReleaseResources(resources)
		return nil, fmt.Errorf("failed to start agent process: %w", err)
	}

	// Add to active agents
	rm.activeAgents[agent.ID] = agent

	return agent, nil
}

// StopAgent stops an agent instance
func (rm *RuntimeManager) StopAgent(agentID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	agent, exists := rm.activeAgents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	return rm.stopAgentInternal(agent)
}

// GetAgentList returns list of active agents
func (rm *RuntimeManager) GetAgentList() []*AgentInstance {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var agents []*AgentInstance
	for _, agent := range rm.activeAgents {
		// Return a copy to prevent modification
		agentCopy := *agent
		agents = append(agents, &agentCopy)
	}

	return agents
}

// GetAgent returns a specific agent instance
func (rm *RuntimeManager) GetAgent(agentID string) (*AgentInstance, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	agent, exists := rm.activeAgents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	// Return a copy to prevent modification
	agentCopy := *agent
	return &agentCopy, nil
}

// startAgentProcess starts the actual agent process
func (rm *RuntimeManager) startAgentProcess(agent *AgentInstance) error {
	binaryPath := filepath.Join(rm.agentDir, agent.Binary)

	// Create command
	cmd := exec.Command(binaryPath, agent.Arguments...)

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range agent.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set working directory
	cmd.Dir = rm.agentDir

	// Setup resource isolation if available
	if rm.processMgr.cgroupManager.enabled {
		if err := rm.setupResourceIsolation(agent, cmd); err != nil {
			return fmt.Errorf("failed to setup resource isolation: %w", err)
		}
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	agent.Command = cmd
	agent.Process = cmd.Process
	agent.PID = cmd.Process.Pid
	agent.Status = AgentStatusRunning
	agent.LastHeartbeat = time.Now()

	// Register with process manager
	processInfo := &ProcessInfo{
		PID:       agent.PID,
		AgentID:   agent.ID,
		Command:   binaryPath,
		StartTime: agent.StartTime,
		Status:    "running",
	}

	rm.processMgr.mu.Lock()
	rm.processMgr.processes[agent.PID] = processInfo
	rm.processMgr.mu.Unlock()

	// Start monitoring the process
	go rm.monitorAgentProcess(agent)

	return nil
}

// stopAgentInternal stops an agent (internal method, assumes lock is held)
func (rm *RuntimeManager) stopAgentInternal(agent *AgentInstance) error {
	agent.Status = AgentStatusStopping

	if agent.Process != nil {
		// Send SIGTERM first
		if err := agent.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			_, err := agent.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(10 * time.Second):
			// Force kill after timeout
			if err := agent.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	}

	// Clean up resources
	if agent.Resources != nil {
		rm.resourcePool.ReleaseResources(agent.Resources)
	}

	// Clean up communication socket
	if agent.Communication != nil && agent.Communication.SocketPath != "" {
		os.Remove(agent.Communication.SocketPath)
	}

	// Remove from process manager
	if agent.PID > 0 {
		rm.processMgr.mu.Lock()
		delete(rm.processMgr.processes, agent.PID)
		rm.processMgr.mu.Unlock()
	}

	// Remove from active agents
	delete(rm.activeAgents, agent.ID)

	agent.Status = AgentStatusStopped

	return nil
}

// setupResourceIsolation sets up cgroup-based resource isolation
func (rm *RuntimeManager) setupResourceIsolation(agent *AgentInstance, cmd *exec.Cmd) error {
	// Create cgroup for the agent
	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/knirv-agents/%s", agent.ID)
	agent.Resources.CgroupPath = cgroupPath

	// This would implement actual cgroup setup
	// For now, just set the systemd slice
	agent.Resources.SystemdSlice = fmt.Sprintf("knirv-agent-%s.slice", agent.ID)

	return nil
}

// monitorAgentProcess monitors an agent process
func (rm *RuntimeManager) monitorAgentProcess(agent *AgentInstance) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			// Check if process is still running
			if agent.Process != nil {
				if err := agent.Process.Signal(syscall.Signal(0)); err != nil {
					// Process is dead
					agent.Status = AgentStatusFailed

					// Handle restart policy
					if agent.RestartPolicy == "always" ||
						(agent.RestartPolicy == "on-failure" && agent.RestartCount < agent.MaxRestarts) {
						rm.restartAgent(agent)
					}
					return
				}
			}

			// Update metrics
			rm.updateAgentMetrics(agent)
		}
	}
}

// restartAgent restarts a failed agent
func (rm *RuntimeManager) restartAgent(agent *AgentInstance) {
	agent.RestartCount++
	agent.Status = AgentStatusRestarting

	// Wait a bit before restarting
	time.Sleep(5 * time.Second)

	// Restart the process
	if err := rm.startAgentProcess(agent); err != nil {
		fmt.Printf("Failed to restart agent %s: %v\n", agent.ID, err)
		agent.Status = AgentStatusFailed
	}
}

// updateAgentMetrics updates agent metrics
func (rm *RuntimeManager) updateAgentMetrics(agent *AgentInstance) {
	if agent.Metrics == nil {
		agent.Metrics = &AgentMetrics{}
	}

	// This would implement actual metrics collection
	// For now, just update the timestamp
	agent.Metrics.CollectedAt = time.Now()
	agent.LastHeartbeat = time.Now()
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

// healthCheckLoop runs health checks on agents
func (rm *RuntimeManager) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.mu.RLock()
			agents := make([]*AgentInstance, 0, len(rm.activeAgents))
			for _, agent := range rm.activeAgents {
				agents = append(agents, agent)
			}
			rm.mu.RUnlock()

			// Perform health checks
			for _, agent := range agents {
				rm.performHealthCheck(agent)
			}
		}
	}
}

// performHealthCheck performs a health check on an agent
func (rm *RuntimeManager) performHealthCheck(agent *AgentInstance) {
	if agent.Status != AgentStatusRunning {
		return
	}

	// This would implement actual health checking
	// For now, just check if the process is alive
	if agent.Process != nil {
		if err := agent.Process.Signal(syscall.Signal(0)); err != nil {
			agent.Status = AgentStatusFailed
			if agent.Metrics != nil {
				agent.Metrics.HealthScore = 0.0
			}
		} else {
			if agent.Metrics != nil {
				agent.Metrics.HealthScore = 1.0
				agent.Metrics.LastHealthCheck = time.Now()
			}
		}
	}
}
