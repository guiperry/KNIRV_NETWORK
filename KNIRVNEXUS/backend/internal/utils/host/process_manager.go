package host

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ProcessManager manages and monitors system processes
type ProcessManager struct {
	ctx    context.Context
	config *HostConfig
	mu     sync.RWMutex

	activeProcesses map[string]*Process
	resourceLimits  *ResourceConfig
	lastUpdate      time.Time
	running         bool
}

// Process represents a system process
type Process struct {
	PID           int       `json:"pid"`
	PPID          int       `json:"ppid"`
	Name          string    `json:"name"`
	Command       string    `json:"command"`
	User          string    `json:"user"`
	Status        string    `json:"status"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryBytes   uint64    `json:"memory_bytes"`
	MemoryPercent float64   `json:"memory_percent"`
	StartTime     time.Time `json:"start_time"`
	RunTime       string    `json:"run_time"`

	// KNIRV-specific fields
	IsKNIRVProcess bool   `json:"is_knirv_process"`
	ServiceType    string `json:"service_type,omitempty"`

	// Resource limits
	CPULimit    float64 `json:"cpu_limit,omitempty"`
	MemoryLimit uint64  `json:"memory_limit,omitempty"`
}

// ResourceConfig defines resource limits and policies
type ResourceConfig struct {
	MaxProcesses     int                      `json:"max_processes"`
	MaxMemoryPercent float64                  `json:"max_memory_percent"`
	MaxCPUPercent    float64                  `json:"max_cpu_percent"`
	ProcessLimits    map[string]*ProcessLimit `json:"process_limits"`

	// Enforcement policies
	KillOnExceed     bool `json:"kill_on_exceed"`
	AlertOnExceed    bool `json:"alert_on_exceed"`
	ThrottleOnExceed bool `json:"throttle_on_exceed"`
}

// ProcessLimit defines limits for specific processes
type ProcessLimit struct {
	MaxCPUPercent    float64 `json:"max_cpu_percent"`
	MaxMemoryBytes   uint64  `json:"max_memory_bytes"`
	MaxMemoryPercent float64 `json:"max_memory_percent"`
	MaxInstances     int     `json:"max_instances"`
}

// NewProcessManager creates a new process manager
func NewProcessManager(ctx context.Context, config *HostConfig) (*ProcessManager, error) {
	pm := &ProcessManager{
		ctx:             ctx,
		config:          config,
		activeProcesses: make(map[string]*Process),
		resourceLimits: &ResourceConfig{
			MaxProcesses:     config.MaxProcesses,
			MaxMemoryPercent: config.MaxMemoryPercent,
			MaxCPUPercent:    config.MaxCPUPercent,
			ProcessLimits:    make(map[string]*ProcessLimit),
			KillOnExceed:     false, // Conservative default
			AlertOnExceed:    true,
			ThrottleOnExceed: true,
		},
	}

	// Set default process limits for KNIRV services
	pm.setDefaultKNIRVLimits()

	return pm, nil
}

// Start begins process monitoring
func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return fmt.Errorf("process manager is already running")
	}

	pm.running = true

	// Initial process scan
	if err := pm.scanProcesses(); err != nil {
		return fmt.Errorf("initial process scan failed: %w", err)
	}

	// Start monitoring loop
	go pm.monitorLoop()

	return nil
}

// Stop stops process monitoring
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.running = false
	return nil
}

// GetProcessList returns current process list
func (pm *ProcessManager) GetProcessList() ([]*Process, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var processes []*Process
	for _, proc := range pm.activeProcesses {
		// Return a copy to prevent modification
		procCopy := *proc
		processes = append(processes, &procCopy)
	}

	return processes, nil
}

// GetKNIRVProcesses returns only KNIRV-related processes
func (pm *ProcessManager) GetKNIRVProcesses() ([]*Process, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var knirvProcesses []*Process
	for _, proc := range pm.activeProcesses {
		if proc.IsKNIRVProcess {
			procCopy := *proc
			knirvProcesses = append(knirvProcesses, &procCopy)
		}
	}

	return knirvProcesses, nil
}

// KillProcess kills a process by PID
func (pm *ProcessManager) KillProcess(pid int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	// Send SIGTERM first
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait a bit for graceful shutdown
	time.Sleep(5 * time.Second)

	// Check if process is still running
	if err := process.Signal(syscall.Signal(0)); err == nil {
		// Process still running, send SIGKILL
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Remove from active processes
	delete(pm.activeProcesses, strconv.Itoa(pid))

	return nil
}

// HealthCheck verifies the process manager is working properly
func (pm *ProcessManager) HealthCheck() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.running {
		return fmt.Errorf("process manager is not running")
	}

	// Check if data is stale
	if time.Since(pm.lastUpdate) > pm.config.ProcessInterval*2 {
		return fmt.Errorf("process data is stale (last update: %v)", pm.lastUpdate)
	}

	return nil
}

// monitorLoop runs the periodic monitoring loop
func (pm *ProcessManager) monitorLoop() {
	ticker := time.NewTicker(pm.config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.mu.RLock()
			running := pm.running
			pm.mu.RUnlock()

			if !running {
				return
			}

			if err := pm.scanProcesses(); err != nil {
				fmt.Printf("Error scanning processes: %v\n", err)
			}

			// Check resource limits
			pm.checkResourceLimits()
		}
	}
}

// scanProcesses scans and updates the process list
func (pm *ProcessManager) scanProcesses() error {
	// Use ps command to get process information
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run ps command: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("invalid ps output")
	}

	newProcesses := make(map[string]*Process)

	// Skip header line
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		process, err := pm.parseProcessLine(line)
		if err != nil {
			continue // Skip invalid lines
		}

		newProcesses[strconv.Itoa(process.PID)] = process
	}

	pm.mu.Lock()
	pm.activeProcesses = newProcesses
	pm.lastUpdate = time.Now()
	pm.mu.Unlock()

	return nil
}

// parseProcessLine parses a line from ps aux output
func (pm *ProcessManager) parseProcessLine(line string) (*Process, error) {
	fields := strings.Fields(line)
	if len(fields) < 11 {
		return nil, fmt.Errorf("invalid ps line format")
	}

	pid, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("invalid PID: %w", err)
	}

	cpuPercent, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		cpuPercent = 0
	}

	memoryPercent, err := strconv.ParseFloat(fields[3], 64)
	if err != nil {
		memoryPercent = 0
	}

	// Calculate memory in bytes (approximate)
	var memoryBytes uint64
	if memoryPercent > 0 {
		// This is a rough approximation - would need /proc/meminfo for accuracy
		memoryBytes = uint64(memoryPercent * 1024 * 1024 * 1024 / 100) // Assume 1GB total for now
	}

	command := strings.Join(fields[10:], " ")

	process := &Process{
		PID:           pid,
		Name:          fields[10],
		Command:       command,
		User:          fields[0],
		CPUPercent:    cpuPercent,
		MemoryBytes:   memoryBytes,
		MemoryPercent: memoryPercent,
		StartTime:     time.Now(), // Simplified - would need more parsing for actual start time
		RunTime:       fields[9],
	}

	// Check if this is a KNIRV process
	pm.identifyKNIRVProcess(process)

	return process, nil
}

// identifyKNIRVProcess identifies if a process is KNIRV-related
func (pm *ProcessManager) identifyKNIRVProcess(process *Process) {
	knirvKeywords := []string{
		"knirv", "nexus", "dve-manager", "validation-core",
		"model-server", "data_engine", "inference",
	}

	commandLower := strings.ToLower(process.Command)
	nameLower := strings.ToLower(process.Name)

	for _, keyword := range knirvKeywords {
		if strings.Contains(commandLower, keyword) || strings.Contains(nameLower, keyword) {
			process.IsKNIRVProcess = true

			// Determine service type
			if strings.Contains(commandLower, "dve-manager") {
				process.ServiceType = "dve-manager"
			} else if strings.Contains(commandLower, "validation-core") {
				process.ServiceType = "validation-core"
			} else if strings.Contains(commandLower, "model-server") {
				process.ServiceType = "model-server"
			} else if strings.Contains(commandLower, "data_engine") {
				process.ServiceType = "data_engine"
			} else if strings.Contains(commandLower, "inference") {
				process.ServiceType = "inference"
			} else {
				process.ServiceType = "knirv-other"
			}

			break
		}
	}
}

// setDefaultKNIRVLimits sets default resource limits for KNIRV services
func (pm *ProcessManager) setDefaultKNIRVLimits() {
	pm.resourceLimits.ProcessLimits["dve-manager"] = &ProcessLimit{
		MaxCPUPercent:    50.0,
		MaxMemoryPercent: 25.0,
		MaxInstances:     1,
	}

	pm.resourceLimits.ProcessLimits["validation-core"] = &ProcessLimit{
		MaxCPUPercent:    70.0,
		MaxMemoryPercent: 40.0,
		MaxInstances:     1,
	}

	pm.resourceLimits.ProcessLimits["model-server"] = &ProcessLimit{
		MaxCPUPercent:    30.0,
		MaxMemoryPercent: 15.0,
		MaxInstances:     1,
	}

	pm.resourceLimits.ProcessLimits["data_engine"] = &ProcessLimit{
		MaxCPUPercent:    40.0,
		MaxMemoryPercent: 20.0,
		MaxInstances:     1,
	}

	pm.resourceLimits.ProcessLimits["inference"] = &ProcessLimit{
		MaxCPUPercent:    80.0,
		MaxMemoryPercent: 50.0,
		MaxInstances:     1,
	}
}

// checkResourceLimits checks and enforces resource limits
func (pm *ProcessManager) checkResourceLimits() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, process := range pm.activeProcesses {
		if !process.IsKNIRVProcess {
			continue
		}

		limit, exists := pm.resourceLimits.ProcessLimits[process.ServiceType]
		if !exists {
			continue
		}

		// Check CPU limit
		if limit.MaxCPUPercent > 0 && process.CPUPercent > limit.MaxCPUPercent {
			pm.handleResourceViolation(process, "CPU", process.CPUPercent, limit.MaxCPUPercent)
		}

		// Check memory limit
		if limit.MaxMemoryPercent > 0 && process.MemoryPercent > limit.MaxMemoryPercent {
			pm.handleResourceViolation(process, "Memory", process.MemoryPercent, limit.MaxMemoryPercent)
		}
	}
}

// handleResourceViolation handles resource limit violations
func (pm *ProcessManager) handleResourceViolation(process *Process, resourceType string, current, limit float64) {
	if pm.resourceLimits.AlertOnExceed {
		fmt.Printf("ALERT: Process %s (PID %d) exceeds %s limit: %.2f%% > %.2f%%\n",
			process.Name, process.PID, resourceType, current, limit)
	}

	// Additional enforcement actions would go here
	// (throttling, killing, etc. - implemented conservatively)
}
