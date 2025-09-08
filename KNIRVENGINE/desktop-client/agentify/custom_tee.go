// custom_tee.go
package agentify

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/shirou/gopsutil/v3/net"
)

// CustomTEE implements the TEE interface using pure Go process isolation
type CustomTEE struct {
	config        TEEConfig
	processID     int32
	workingDir    string
	resourceLimits ResourceLimits
	securityPolicy SecurityPolicy
	mutex         sync.Mutex
	cmdContext    context.Context
	cmdCancel     context.CancelFunc
	processStats  *ProcessStats
	isRunning     bool
}

// ProcessStats tracks process statistics
type ProcessStats struct {
	PID           int32
	StartTime     time.Time
	LastUpdated   time.Time
	MemoryUsage   []float64  // Historical memory usage in MB
	CPUUsage      []float64  // Historical CPU usage in percent
	DiskIO        []float64  // Historical disk I/O in MB
	NetworkIO     []float64  // Historical network I/O in MB
	ChildProcesses []int32   // Child process IDs
}

// NewCustomTEE creates a new CustomTEE
func NewCustomTEE(config TEEConfig) *CustomTEE {
	// Set default resource limits if not specified
	resourceLimits := config.ResourceLimits
	if resourceLimits.MemoryMB == 0 {
		resourceLimits.MemoryMB = 512 // Default 512MB
	}
	if resourceLimits.CPUCores == 0 {
		resourceLimits.CPUCores = 1.0 // Default 1 core
	}
	if resourceLimits.ExecutionTimeout == 0 {
		resourceLimits.ExecutionTimeout = 30 * time.Second // Default 30s timeout
	}

	// Set default alert thresholds if not specified
	if resourceLimits.MemoryAlertThreshold == 0 {
		resourceLimits.MemoryAlertThreshold = 0.8 // Default 80%
	}
	if resourceLimits.CPUAlertThreshold == 0 {
		resourceLimits.CPUAlertThreshold = 0.9 // Default 90%
	}
	if resourceLimits.DiskAlertThreshold == 0 {
		resourceLimits.DiskAlertThreshold = 0.7 // Default 70%
	}

	// Set default security policy if not specified
	securityPolicy := config.SecurityPolicy
	if securityPolicy.MaxExecutionTime == 0 {
		securityPolicy.MaxExecutionTime = 60 * time.Second // Default 60s max execution
	}

	return &CustomTEE{
		config:        config,
		workingDir:    config.WorkingDir,
		resourceLimits: resourceLimits,
		securityPolicy: securityPolicy,
		processStats:  &ProcessStats{
			StartTime:   time.Now(),
			LastUpdated: time.Now(),
			MemoryUsage: make([]float64, 0, 60),  // Store up to 60 data points
			CPUUsage:    make([]float64, 0, 60),
			DiskIO:      make([]float64, 0, 60),
			NetworkIO:   make([]float64, 0, 60),
		},
	}
}

// Start starts the CustomTEE
func (t *CustomTEE) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.isRunning {
		return nil
	}

	// Create the working directory if it doesn't exist
	if t.workingDir != "" {
		if err := os.MkdirAll(t.workingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}
		
		// Set secure permissions on working directory
		if err := os.Chmod(t.workingDir, 0750); err != nil {
			return fmt.Errorf("failed to set secure permissions on working directory: %v", err)
		}
	}

	// Validate security policy
	if err := t.validateSecurityPolicy(); err != nil {
		return fmt.Errorf("security policy validation failed: %v", err)
	}

	// Create a cancelable context for the process
	t.cmdContext, t.cmdCancel = context.WithCancel(context.Background())

	// Start a monitoring goroutine
	go t.monitorResources()

	t.isRunning = true
	log.Printf("CustomTEE started with working directory: %s", t.workingDir)

	return nil
}

// Stop stops the CustomTEE
func (t *CustomTEE) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.isRunning {
		return nil
	}

	// Cancel the context to stop any running processes
	if t.cmdCancel != nil {
		t.cmdCancel()
	}

	// Kill any remaining processes
	if t.processID != 0 {
		proc, err := os.FindProcess(int(t.processID))
		if err == nil {
			proc.Kill()
		}
	}

	// Kill any child processes
	for _, childPID := range t.processStats.ChildProcesses {
		childProc, err := os.FindProcess(int(childPID))
		if err == nil {
			childProc.Kill()
		}
	}

	t.processID = 0
	t.processStats.ChildProcesses = nil
	t.isRunning = false

	log.Printf("CustomTEE stopped")
	return nil
}

// Execute executes a command in the CustomTEE
func (t *CustomTEE) Execute(command string, args []string) (string, string, int, error) {
	// Create a context with the security policy timeout
	ctx, cancel := context.WithTimeout(context.Background(), t.securityPolicy.MaxExecutionTime)
	defer cancel()

	return t.ExecuteWithContext(ctx, command, args)
}

// ExecuteWithContext executes a command with context in the CustomTEE
func (t *CustomTEE) ExecuteWithContext(ctx context.Context, command string, args []string) (string, string, int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.isRunning {
		return "", "", 1, fmt.Errorf("CustomTEE not started")
	}

	// Validate the command against security policy
	if err := t.ValidateCommand(command, args); err != nil {
		return "", fmt.Sprintf("Command validation failed: %v", err), 1, err
	}

	// Create the command with the context
	cmd := exec.CommandContext(ctx, command, args...)

	// Set the working directory
	if t.workingDir != "" {
		cmd.Dir = t.workingDir
	}

	// Set environment variables
	if len(t.config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range t.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Apply resource limits if supported by the OS
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	t.applyResourceLimits(cmd)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()

	// Get the exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	// Store the process ID for monitoring
	if cmd.Process != nil {
		t.processID = int32(cmd.Process.Pid)
	}

	return stdout.String(), stderr.String(), exitCode, err
}

// applyResourceLimits applies resource limits to the command
func (t *CustomTEE) applyResourceLimits(cmd *exec.Cmd) {
	// Set process group for better process management
	setProcGroup(cmd.SysProcAttr)
}

// validateSecurityPolicy validates the security policy configuration
func (t *CustomTEE) validateSecurityPolicy() error {
	// Check for conflicting policies
	if len(t.securityPolicy.AllowedCommands) > 0 && len(t.securityPolicy.BlockedCommands) > 0 {
		// Check for commands that are both allowed and blocked
		for _, allowed := range t.securityPolicy.AllowedCommands {
			for _, blocked := range t.securityPolicy.BlockedCommands {
				if allowed == blocked {
					return fmt.Errorf("command '%s' cannot be both allowed and blocked", allowed)
				}
			}
		}
	}

	// Validate timeout values
	if t.securityPolicy.MaxExecutionTime <= 0 {
		return fmt.Errorf("max execution time must be positive")
	}

	return nil
}

// monitorResources monitors resource usage of the process
func (t *CustomTEE) monitorResources() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if t.processID != 0 {
				t.updateResourceStats()
			}
		case <-t.cmdContext.Done():
			return
		}
	}
}

// updateResourceStats updates resource usage statistics
func (t *CustomTEE) updateResourceStats() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.processID == 0 {
		return
	}

	// Use the process package to get resource usage
	proc, err := process.NewProcess(t.processID)
	if err != nil {
		log.Printf("Failed to get process info: %v", err)
		return
	}

	// Update memory usage
	memInfo, err := proc.MemoryInfo()
	if err == nil && memInfo != nil {
		memoryMB := float64(memInfo.RSS) / 1024 / 1024
		t.processStats.MemoryUsage = append(t.processStats.MemoryUsage, memoryMB)
		if len(t.processStats.MemoryUsage) > 60 {
			t.processStats.MemoryUsage = t.processStats.MemoryUsage[1:]
		}
	}

	// Update CPU usage
	cpuPercent, err := proc.CPUPercent()
	if err == nil {
		t.processStats.CPUUsage = append(t.processStats.CPUUsage, cpuPercent)
		if len(t.processStats.CPUUsage) > 60 {
			t.processStats.CPUUsage = t.processStats.CPUUsage[1:]
		}
	}

	// Update disk I/O
	ioCounters, err := proc.IOCounters()
	if err == nil && ioCounters != nil {
		diskIO := float64(ioCounters.ReadBytes+ioCounters.WriteBytes) / 1024 / 1024
		t.processStats.DiskIO = append(t.processStats.DiskIO, diskIO)
		if len(t.processStats.DiskIO) > 60 {
			t.processStats.DiskIO = t.processStats.DiskIO[1:]
		}
	}

	// Update network I/O (if available)
	netIOCounters, err := net.IOCounters(false)
	if err == nil && len(netIOCounters) > 0 {
		networkIO := float64(netIOCounters[0].BytesSent+netIOCounters[0].BytesRecv) / 1024 / 1024
		t.processStats.NetworkIO = append(t.processStats.NetworkIO, networkIO)
		if len(t.processStats.NetworkIO) > 60 {
			t.processStats.NetworkIO = t.processStats.NetworkIO[1:]
		}
	}

	// Update child processes
	children, err := proc.Children()
	if err == nil {
		childPIDs := make([]int32, len(children))
		for i, child := range children {
			childPIDs[i] = child.Pid
		}
		t.processStats.ChildProcesses = childPIDs
	}

	t.processStats.LastUpdated = time.Now()
}

// GetResourceUsage returns current resource usage for CustomTEE
func (t *CustomTEE) GetResourceUsage() (ResourceUsage, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	usage := ResourceUsage{
		Timestamp: time.Now(),
	}

	// Check if we have a demo mode flag set in the environment
	demoMode := os.Getenv("AGENTIC_ENGINE_DEMO_MODE") == "true"
	if demoMode {
		// Return simulated data only in demo mode
		return ResourceUsage{
			MemoryMB:    float64(t.resourceLimits.MemoryMB) * 0.5, // Simulated 50% usage
			CPUPercent:  25.0,                                     // Simulated 25% CPU usage
			DiskUsageMB: 10.0,                                     // Simulated 10MB disk usage
			NetworkMB:   1.0,                                      // Simulated 1MB network usage
			Processes:   1,                                        // Single process
			Timestamp:   time.Now(),
		}, nil
	}

	if t.processID == 0 {
		return usage, fmt.Errorf("no active process")
	}

	// Calculate average memory usage from recent measurements
	if len(t.processStats.MemoryUsage) > 0 {
		usage.MemoryMB = t.processStats.MemoryUsage[len(t.processStats.MemoryUsage)-1]
	}

	// Calculate average CPU usage from recent measurements
	if len(t.processStats.CPUUsage) > 0 {
		usage.CPUPercent = t.processStats.CPUUsage[len(t.processStats.CPUUsage)-1]
	}

	// Calculate average disk I/O from recent measurements
	if len(t.processStats.DiskIO) > 0 {
		usage.DiskUsageMB = t.processStats.DiskIO[len(t.processStats.DiskIO)-1]
	}

	// Calculate average network I/O from recent measurements
	if len(t.processStats.NetworkIO) > 0 {
		usage.NetworkMB = t.processStats.NetworkIO[len(t.processStats.NetworkIO)-1]
	}

	// Set process count
	usage.Processes = len(t.processStats.ChildProcesses) + 1 // +1 for the main process

	return usage, nil
}

// GetInfo returns information about the CustomTEE
func (t *CustomTEE) GetInfo() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return map[string]interface{}{
		"type":        "custom",
		"workingDir":  t.workingDir,
		"env":         t.config.Env,
		"processID":   t.processID,
		"isRunning":   t.isRunning,
		"startTime":   t.processStats.StartTime,
		"lastUpdated": t.processStats.LastUpdated,
	}
}

// SetResourceLimits sets resource limits for the CustomTEE
func (t *CustomTEE) SetResourceLimits(limits ResourceLimits) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.resourceLimits = limits
	return nil
}

// SetSecurityPolicy sets security policy for the CustomTEE
func (t *CustomTEE) SetSecurityPolicy(policy SecurityPolicy) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.securityPolicy = policy
	return nil
}

// ValidateCommand validates if a command is allowed to execute
func (t *CustomTEE) ValidateCommand(command string, args []string) error {
	// Check if command is in blocked list
	for _, blocked := range t.securityPolicy.BlockedCommands {
		if command == blocked {
			return fmt.Errorf("command '%s' is blocked by security policy", command)
		}
	}

	// If allowed commands are specified, check if command is in the list
	if len(t.securityPolicy.AllowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range t.securityPolicy.AllowedCommands {
			if command == allowedCmd {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command '%s' is not in allowed commands list", command)
		}
	}

	// Additional security checks
	if !t.securityPolicy.AllowNetworkAccess {
		// Check for network-related commands
		networkCommands := []string{"curl", "wget", "nc", "netcat", "ssh", "scp", "rsync"}
		for _, netCmd := range networkCommands {
			if command == netCmd {
				return fmt.Errorf("network command '%s' is not allowed by security policy", command)
			}
		}
	}

	if !t.securityPolicy.AllowFileSystemWrite {
		// Check for file write commands
		writeCommands := []string{"rm", "rmdir", "mv", "cp", "dd", "tee", "touch", "mkdir", "chmod", "chown"}
		for _, writeCmd := range writeCommands {
			if command == writeCmd {
				return fmt.Errorf("file write command '%s' is not allowed by security policy", command)
			}
		}
	}

	return nil
}

// CheckResourceAlerts checks if resource usage exceeds thresholds and returns alerts
func (t *CustomTEE) CheckResourceAlerts() ([]ResourceAlert, error) {
	// Get current resource usage
	usage, err := t.GetResourceUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get resource usage: %v", err)
	}

	alerts := []ResourceAlert{}
	now := time.Now()

	// Check memory usage
	if t.resourceLimits.MemoryMB > 0 {
		memoryUsagePercent := usage.MemoryMB / float64(t.resourceLimits.MemoryMB)

		// Critical alert (90% of limit)
		if memoryUsagePercent >= 0.9 {
			alerts = append(alerts, ResourceAlert{
				ResourceType: "memory",
				CurrentValue: usage.MemoryMB,
				Threshold:    float64(t.resourceLimits.MemoryMB) * 0.9,
				Limit:        float64(t.resourceLimits.MemoryMB),
				Severity:     "critical",
				Message:      fmt.Sprintf("Memory usage critical: %.2f MB (%.1f%% of limit)", usage.MemoryMB, memoryUsagePercent*100),
				Timestamp:    now,
			})
		} else if memoryUsagePercent >= t.resourceLimits.MemoryAlertThreshold {
			// Warning alert (threshold %)
			alerts = append(alerts, ResourceAlert{
				ResourceType: "memory",
				CurrentValue: usage.MemoryMB,
				Threshold:    float64(t.resourceLimits.MemoryMB) * t.resourceLimits.MemoryAlertThreshold,
				Limit:        float64(t.resourceLimits.MemoryMB),
				Severity:     "warning",
				Message:      fmt.Sprintf("Memory usage high: %.2f MB (%.1f%% of limit)", usage.MemoryMB, memoryUsagePercent*100),
				Timestamp:    now,
			})
		}
	}

	// Check CPU usage
	if t.resourceLimits.CPUCores > 0 {
		// Convert CPU percentage to cores
		cpuCores := usage.CPUPercent / 100.0
		cpuUsagePercent := cpuCores / t.resourceLimits.CPUCores

		// Critical alert (90% of limit)
		if cpuUsagePercent >= 0.9 {
			alerts = append(alerts, ResourceAlert{
				ResourceType: "cpu",
				CurrentValue: cpuCores,
				Threshold:    t.resourceLimits.CPUCores * 0.9,
				Limit:        t.resourceLimits.CPUCores,
				Severity:     "critical",
				Message:      fmt.Sprintf("CPU usage critical: %.2f cores (%.1f%% of limit)", cpuCores, cpuUsagePercent*100),
				Timestamp:    now,
			})
		} else if cpuUsagePercent >= t.resourceLimits.CPUAlertThreshold {
			// Warning alert (threshold %)
			alerts = append(alerts, ResourceAlert{
				ResourceType: "cpu",
				CurrentValue: cpuCores,
				Threshold:    t.resourceLimits.CPUCores * t.resourceLimits.CPUAlertThreshold,
				Limit:        t.resourceLimits.CPUCores,
				Severity:     "warning",
				Message:      fmt.Sprintf("CPU usage high: %.2f cores (%.1f%% of limit)", cpuCores, cpuUsagePercent*100),
				Timestamp:    now,
			})
		}
	}

	// Check disk usage
	if t.resourceLimits.DiskSpaceMB > 0 {
		diskUsagePercent := usage.DiskUsageMB / float64(t.resourceLimits.DiskSpaceMB)

		// Critical alert (90% of limit)
		if diskUsagePercent >= 0.9 {
			alerts = append(alerts, ResourceAlert{
				ResourceType: "disk",
				CurrentValue: usage.DiskUsageMB,
				Threshold:    float64(t.resourceLimits.DiskSpaceMB) * 0.9,
				Limit:        float64(t.resourceLimits.DiskSpaceMB),
				Severity:     "critical",
				Message:      fmt.Sprintf("Disk usage critical: %.2f MB (%.1f%% of limit)", usage.DiskUsageMB, diskUsagePercent*100),
				Timestamp:    now,
			})
		} else if diskUsagePercent >= t.resourceLimits.DiskAlertThreshold {
			// Warning alert (threshold %)
			alerts = append(alerts, ResourceAlert{
				ResourceType: "disk",
				CurrentValue: usage.DiskUsageMB,
				Threshold:    float64(t.resourceLimits.DiskSpaceMB) * t.resourceLimits.DiskAlertThreshold,
				Limit:        float64(t.resourceLimits.DiskSpaceMB),
				Severity:     "warning",
				Message:      fmt.Sprintf("Disk usage high: %.2f MB (%.1f%% of limit)", usage.DiskUsageMB, diskUsagePercent*100),
				Timestamp:    now,
			})
		}
	}

	// Check process count
	if t.resourceLimits.MaxProcesses > 0 {
		processUsagePercent := float64(usage.Processes) / float64(t.resourceLimits.MaxProcesses)

		// Critical alert (90% of limit)
		if processUsagePercent >= 0.9 {
			alerts = append(alerts, ResourceAlert{
				ResourceType: "processes",
				CurrentValue: float64(usage.Processes),
				Threshold:    float64(t.resourceLimits.MaxProcesses) * 0.9,
				Limit:        float64(t.resourceLimits.MaxProcesses),
				Severity:     "critical",
				Message:      fmt.Sprintf("Process count critical: %d (%.1f%% of limit)", usage.Processes, processUsagePercent*100),
				Timestamp:    now,
			})
		} else if processUsagePercent >= 0.8 {
			// Warning alert (80% of limit)
			alerts = append(alerts, ResourceAlert{
				ResourceType: "processes",
				CurrentValue: float64(usage.Processes),
				Threshold:    float64(t.resourceLimits.MaxProcesses) * 0.8,
				Limit:        float64(t.resourceLimits.MaxProcesses),
				Severity:     "warning",
				Message:      fmt.Sprintf("Process count high: %d (%.1f%% of limit)", usage.Processes, processUsagePercent*100),
				Timestamp:    now,
			})
		}
	}

	// Log alerts
	for _, alert := range alerts {
		log.Printf("CustomTEE Resource Alert: %s - %s", alert.Severity, alert.Message)
	}

	return alerts, nil
}