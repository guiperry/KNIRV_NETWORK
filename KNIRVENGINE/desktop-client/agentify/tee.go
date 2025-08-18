// tee.go
package agentify

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	MemoryMB    float64   `json:"memory_mb"`
	CPUPercent  float64   `json:"cpu_percent"`
	DiskUsageMB float64   `json:"disk_usage_mb"`
	NetworkMB   float64   `json:"network_mb"`
	Processes   int       `json:"processes"`
	Timestamp   time.Time `json:"timestamp"`
}

// ResourceAlert represents a resource usage alert
type ResourceAlert struct {
	ResourceType string    `json:"resource_type"` // "memory", "cpu", "disk", "network", "processes"
	CurrentValue float64   `json:"current_value"`
	Threshold    float64   `json:"threshold"`
	Limit        float64   `json:"limit"`
	Severity     string    `json:"severity"` // "warning", "critical"
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	MemoryMB             int           `json:"memory_mb"`
	CPUCores             float64       `json:"cpu_cores"`
	DiskSpaceMB          int           `json:"disk_space_mb"`
	NetworkBandwidthMBps int           `json:"network_bandwidth_mbps"`
	MaxProcesses         int           `json:"max_processes"`
	ExecutionTimeout     time.Duration `json:"execution_timeout"`

	// Alert thresholds (percentage of limit)
	MemoryAlertThreshold float64 `json:"memory_alert_threshold"` // e.g., 0.8 for 80%
	CPUAlertThreshold    float64 `json:"cpu_alert_threshold"`    // e.g., 0.9 for 90%
	DiskAlertThreshold   float64 `json:"disk_alert_threshold"`   // e.g., 0.7 for 70%
}

// SecurityPolicy defines security constraints for TEE
type SecurityPolicy struct {
	AllowNetworkAccess   bool          `json:"allow_network_access"`
	AllowFileSystemWrite bool          `json:"allow_filesystem_write"`
	AllowedCommands      []string      `json:"allowed_commands"`
	BlockedCommands      []string      `json:"blocked_commands"`
	RequireSignature     bool          `json:"require_signature"`
	MaxExecutionTime     time.Duration `json:"max_execution_time"`
}

// TEE (Trusted Execution Environment) defines the interface for executing code in a secure environment
type TEE interface {
	// Start starts the TEE
	Start() error

	// Stop stops the TEE
	Stop() error

	// Execute executes a command in the TEE and returns the stdout, stderr, exit code, and error
	Execute(command string, args []string) (stdout string, stderr string, exitCode int, err error)

	// ExecuteWithContext executes a command with context and timeout
	ExecuteWithContext(ctx context.Context, command string, args []string) (stdout string, stderr string, exitCode int, err error)

	// GetInfo returns information about the TEE
	GetInfo() map[string]interface{}

	// GetResourceUsage returns current resource usage
	GetResourceUsage() (ResourceUsage, error)

	// SetResourceLimits sets resource limits for the TEE
	SetResourceLimits(limits ResourceLimits) error

	// SetSecurityPolicy sets security policy for the TEE
	SetSecurityPolicy(policy SecurityPolicy) error

	// ValidateCommand validates if a command is allowed to execute
	ValidateCommand(command string, args []string) error

	// CheckResourceAlerts checks if resource usage exceeds thresholds and returns alerts
	CheckResourceAlerts() ([]ResourceAlert, error)
}

// TEEConfig defines the configuration for a TEE
type TEEConfig struct {
	// The working directory for the TEE
	WorkingDir string

	// Environment variables for the TEE
	Env map[string]string

	// For container and VM TEEs
	Image string
	Tag   string

	// For VM TEEs
	Memory int
	CPU    int

	// Security and resource configuration
	ResourceLimits ResourceLimits `json:"resource_limits"`
	SecurityPolicy SecurityPolicy `json:"security_policy"`
}

// ProcessTEE implements the TEE interface using a process
type ProcessTEE struct {
	config         TEEConfig
	resourceLimits ResourceLimits
	securityPolicy SecurityPolicy
	mutex          sync.Mutex
	isRunning      bool
	startTime      time.Time
	processID      int32     // Process ID for resource monitoring
	lastCmd        *exec.Cmd // Last executed command for resource monitoring
}

// NewProcessTEE creates a new ProcessTEE
func NewProcessTEE(config TEEConfig) *ProcessTEE {
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

	return &ProcessTEE{
		config:         config,
		resourceLimits: resourceLimits,
		securityPolicy: securityPolicy,
		isRunning:      false,
	}
}

// Start starts the ProcessTEE
func (t *ProcessTEE) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.isRunning {
		return nil
	}

	// Create the working directory if it doesn't exist
	if t.config.WorkingDir != "" {
		if err := os.MkdirAll(t.config.WorkingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}

		// Set secure permissions on working directory
		if err := os.Chmod(t.config.WorkingDir, 0750); err != nil {
			return fmt.Errorf("failed to set secure permissions on working directory: %v", err)
		}
	}

	// Validate security policy
	if err := t.validateSecurityPolicy(); err != nil {
		return fmt.Errorf("security policy validation failed: %v", err)
	}

	t.isRunning = true
	t.startTime = time.Now()

	return nil
}

// Stop stops the ProcessTEE
func (t *ProcessTEE) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.isRunning = false
	return nil
}

// Execute executes a command in the ProcessTEE
func (t *ProcessTEE) Execute(command string, args []string) (string, string, int, error) {
	// Use ExecuteWithContext with default timeout
	ctx, cancel := context.WithTimeout(context.Background(), t.resourceLimits.ExecutionTimeout)
	defer cancel()

	return t.ExecuteWithContext(ctx, command, args)
}

// ExecuteWithContext executes a command with context and timeout
func (t *ProcessTEE) ExecuteWithContext(ctx context.Context, command string, args []string) (string, string, int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.isRunning {
		return "", "", 1, fmt.Errorf("TEE is not running")
	}

	// Validate command against security policy
	if err := t.ValidateCommand(command, args); err != nil {
		return "", "", 1, fmt.Errorf("command validation failed: %v", err)
	}

	// Create the command with context
	cmd := exec.CommandContext(ctx, command, args...)
	t.lastCmd = cmd // Store the command for resource monitoring

	// Set the working directory
	if t.config.WorkingDir != "" {
		cmd.Dir = t.config.WorkingDir
	}

	// Set environment variables
	if len(t.config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range t.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Apply resource limits (Unix-specific)
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Set process group for better resource management (Unix only)
	setProcGroup(cmd.SysProcAttr)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	startTime := time.Now()
	err := cmd.Run()
	executionTime := time.Since(startTime)

	// Store the process ID for resource monitoring if available
	if cmd.Process != nil {
		t.processID = int32(cmd.Process.Pid)
	}

	// Check if execution exceeded time limits
	if executionTime > t.securityPolicy.MaxExecutionTime {
		return "", "", 1, fmt.Errorf("execution time exceeded limit: %v > %v", executionTime, t.securityPolicy.MaxExecutionTime)
	}

	// Get the exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}

// GetInfo returns information about the ProcessTEE
func (t *ProcessTEE) GetInfo() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return map[string]interface{}{
		"type":           "process",
		"workingDir":     t.config.WorkingDir,
		"env":            t.config.Env,
		"isRunning":      t.isRunning,
		"startTime":      t.startTime,
		"resourceLimits": t.resourceLimits,
		"securityPolicy": t.securityPolicy,
	}
}

// GetResourceUsage returns current resource usage for ProcessTEE
func (t *ProcessTEE) GetResourceUsage() (ResourceUsage, error) {
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

	// If no process is running, get overall system metrics
	if t.processID == 0 {
		// Get memory information
		memInfo, err := mem.VirtualMemory()
		if err != nil {
			return usage, fmt.Errorf("failed to get memory info: %v", err)
		}
		usage.MemoryMB = float64(memInfo.Used) / 1024 / 1024

		// Get CPU usage
		cpuPercent, err := cpu.Percent(0, false)
		if err != nil {
			return usage, fmt.Errorf("failed to get CPU usage: %v", err)
		}
		if len(cpuPercent) > 0 {
			usage.CPUPercent = cpuPercent[0]
		}

		// Get disk usage for the working directory or current directory
		path := t.config.WorkingDir
		if path == "" {
			path, _ = os.Getwd()
		}
		diskUsage, err := disk.Usage(path)
		if err != nil {
			log.Printf("Warning: failed to get disk usage: %v", err)
		} else {
			usage.DiskUsageMB = float64(diskUsage.Used) / 1024 / 1024
		}

		// Get network usage
		netIOCounters, err := net.IOCounters(false)
		if err == nil && len(netIOCounters) > 0 {
			usage.NetworkMB = float64(netIOCounters[0].BytesSent+netIOCounters[0].BytesRecv) / 1024 / 1024
		}

		// Get process count
		processes, err := process.Processes()
		if err == nil {
			usage.Processes = len(processes)
		} else {
			usage.Processes = 1 // Default if we can't get the actual count
		}

		return usage, nil
	}

	// Get process information using process ID
	proc, err := process.NewProcess(t.processID)
	if err != nil {
		// Process might have ended, return system-wide metrics instead
		log.Printf("Warning: failed to get process info (PID %d): %v", t.processID, err)
		t.processID = 0             // Reset process ID
		return t.GetResourceUsage() // Recursively call to get system metrics
	}

	// Get memory information
	memInfo, err := proc.MemoryInfo()
	if err != nil {
		return usage, fmt.Errorf("failed to get memory info: %v", err)
	}
	usage.MemoryMB = float64(memInfo.RSS) / 1024 / 1024

	// Get CPU usage
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		return usage, fmt.Errorf("failed to get CPU usage: %v", err)
	}
	usage.CPUPercent = cpuPercent

	// Get disk usage
	ioCounters, err := proc.IOCounters()
	if err != nil {
		log.Printf("Warning: failed to get IO counters: %v", err)
	} else {
		usage.DiskUsageMB = float64(ioCounters.ReadBytes+ioCounters.WriteBytes) / 1024 / 1024
	}

	// Get network usage (if available)
	netIOCounters, err := net.IOCounters(false)
	if err == nil && len(netIOCounters) > 0 {
		usage.NetworkMB = float64(netIOCounters[0].BytesSent+netIOCounters[0].BytesRecv) / 1024 / 1024
	}

	// Get child processes count
	children, err := proc.Children()
	if err == nil {
		usage.Processes = len(children) + 1 // +1 for the main process
	} else {
		usage.Processes = 1 // At least the main process
	}

	return usage, nil
}

// SetResourceLimits sets resource limits for the ProcessTEE
func (t *ProcessTEE) SetResourceLimits(limits ResourceLimits) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.resourceLimits = limits
	return nil
}

// SetSecurityPolicy sets security policy for the ProcessTEE
func (t *ProcessTEE) SetSecurityPolicy(policy SecurityPolicy) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.securityPolicy = policy
	return nil
}

// CheckResourceAlerts checks if resource usage exceeds thresholds and returns alerts
func (t *ProcessTEE) CheckResourceAlerts() ([]ResourceAlert, error) {
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
		log.Printf("TEE Resource Alert: %s - %s", alert.Severity, alert.Message)
	}

	return alerts, nil
}

// ValidateCommand validates if a command is allowed to execute
func (t *ProcessTEE) ValidateCommand(command string, args []string) error {
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

	return nil
}

// validateSecurityPolicy validates the security policy configuration
func (t *ProcessTEE) validateSecurityPolicy() error {
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

// ContainerTEE implements the TEE interface using a container
type ContainerTEE struct {
	config      TEEConfig
	containerID string
	mutex       sync.Mutex
}

// NewContainerTEE creates a new ContainerTEE
func NewContainerTEE(config TEEConfig) *ContainerTEE {
	return &ContainerTEE{
		config: config,
	}
}

// Start starts the ContainerTEE
func (t *ContainerTEE) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Create the working directory if it doesn't exist
	if t.config.WorkingDir != "" {
		if err := os.MkdirAll(t.config.WorkingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}
	}

	// Pull the container image
	pullCmd := exec.Command("docker", "pull", fmt.Sprintf("%s:%s", t.config.Image, t.config.Tag))
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull container image: %v", err)
	}

	// Create the container
	createArgs := []string{
		"create",
		"--rm",
		"-v", fmt.Sprintf("%s:/workspace", t.config.WorkingDir),
	}

	// Add environment variables
	for k, v := range t.config.Env {
		createArgs = append(createArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add the image
	createArgs = append(createArgs, fmt.Sprintf("%s:%s", t.config.Image, t.config.Tag))

	// Add a command to keep the container running
	createArgs = append(createArgs, "tail", "-f", "/dev/null")

	// Create the container
	createCmd := exec.Command("docker", createArgs...)
	var stdout bytes.Buffer
	createCmd.Stdout = &stdout
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Get the container ID
	t.containerID = stdout.String()

	// Start the container
	startCmd := exec.Command("docker", "start", t.containerID)
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	return nil
}

// Stop stops the ContainerTEE
func (t *ContainerTEE) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.containerID == "" {
		return nil
	}

	// Stop the container
	stopCmd := exec.Command("docker", "stop", t.containerID)
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %v", err)
	}

	t.containerID = ""
	return nil
}

// Execute executes a command in the ContainerTEE
func (t *ContainerTEE) Execute(command string, args []string) (string, string, int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.containerID == "" {
		return "", "", 1, fmt.Errorf("container not started")
	}

	// Create the command
	execArgs := []string{
		"exec",
		"-w", "/workspace",
		t.containerID,
		command,
	}
	execArgs = append(execArgs, args...)

	// Execute the command
	cmd := exec.Command("docker", execArgs...)

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

	return stdout.String(), stderr.String(), exitCode, err
}

// GetInfo returns information about the ContainerTEE
func (t *ContainerTEE) GetInfo() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return map[string]interface{}{
		"type":        "container",
		"image":       t.config.Image,
		"tag":         t.config.Tag,
		"containerID": t.containerID,
		"workingDir":  t.config.WorkingDir,
		"env":         t.config.Env,
	}
}

// ExecuteWithContext executes a command with context for ContainerTEE
func (t *ContainerTEE) ExecuteWithContext(ctx context.Context, command string, args []string) (string, string, int, error) {
	// For now, delegate to the regular Execute method
	// In a full implementation, this would use docker exec with context
	return t.Execute(command, args)
}

// GetResourceUsage returns current resource usage for ContainerTEE
func (t *ContainerTEE) GetResourceUsage() (ResourceUsage, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Check if we have a demo mode flag set in the environment
	demoMode := os.Getenv("AGENTIC_ENGINE_DEMO_MODE") == "true"
	if demoMode {
		// Return simulated data only in demo mode
		return ResourceUsage{
			MemoryMB:    256.0, // Simulated container usage
			CPUPercent:  30.0,
			DiskUsageMB: 50.0,
			NetworkMB:   5.0,
			Processes:   3,
			Timestamp:   time.Now(),
		}, nil
	}

	// Initialize default usage with timestamp
	usage := ResourceUsage{
		Timestamp: time.Now(),
	}

	// If container is not running, return empty stats
	if t.containerID == "" {
		return usage, fmt.Errorf("container not running")
	}

	// Use docker stats to get container resource usage
	// docker stats --no-stream --format "{{.MemUsage}},{{.CPUPerc}},{{.NetIO}},{{.BlockIO}},{{.PIDs}}" containerID
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.MemUsage}},{{.CPUPerc}},{{.NetIO}},{{.BlockIO}},{{.PIDs}}", t.containerID)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return usage, fmt.Errorf("failed to get container stats: %v", err)
	}

	// Parse the output
	statsOutput := stdout.String()
	if statsOutput == "" {
		return usage, fmt.Errorf("empty stats output for container %s", t.containerID)
	}

	// Parse the comma-separated values
	parts := bytes.Split(bytes.TrimSpace(stdout.Bytes()), []byte(","))
	if len(parts) < 5 {
		return usage, fmt.Errorf("invalid stats output format: %s", statsOutput)
	}

	// Parse memory usage (format: "100MiB / 16GiB")
	memParts := bytes.Split(parts[0], []byte(" / "))
	if len(memParts) > 0 {
		memStr := string(memParts[0])
		// Extract numeric part and convert to MB
		var memValue float64
		var memUnit string
		fmt.Sscanf(memStr, "%f%s", &memValue, &memUnit)

		// Convert to MB based on unit
		switch memUnit {
		case "B":
			usage.MemoryMB = memValue / 1024 / 1024
		case "KiB", "KB":
			usage.MemoryMB = memValue / 1024
		case "MiB", "MB":
			usage.MemoryMB = memValue
		case "GiB", "GB":
			usage.MemoryMB = memValue * 1024
		}
	}

	// Parse CPU usage (format: "0.00%")
	cpuStr := string(parts[1])
	cpuStr = cpuStr[:len(cpuStr)-1] // Remove % sign
	cpuValue, err := parseFloat(cpuStr)
	if err == nil {
		usage.CPUPercent = cpuValue
	}

	// Parse network I/O (format: "100MB / 200MB")
	netParts := bytes.Split(parts[2], []byte(" / "))
	if len(netParts) > 1 {
		// Sum of received and sent
		netIn := parseSize(string(netParts[0]))
		netOut := parseSize(string(netParts[1]))
		usage.NetworkMB = netIn + netOut
	}

	// Parse block I/O (format: "100MB / 200MB")
	diskParts := bytes.Split(parts[3], []byte(" / "))
	if len(diskParts) > 1 {
		// Sum of read and write
		diskRead := parseSize(string(diskParts[0]))
		diskWrite := parseSize(string(diskParts[1]))
		usage.DiskUsageMB = diskRead + diskWrite
	}

	// Parse PIDs (number of processes)
	pidStr := string(parts[4])
	pidValue, err := parseInt(pidStr)
	if err == nil {
		usage.Processes = pidValue
	}

	return usage, nil
}

// Helper function to parse float values
func parseFloat(s string) (float64, error) {
	var value float64
	_, err := fmt.Sscanf(s, "%f", &value)
	return value, err
}

// Helper function to parse integer values
func parseInt(s string) (int, error) {
	var value int
	_, err := fmt.Sscanf(s, "%d", &value)
	return value, err
}

// Helper function to parse size strings like "10MB", "1.5GB" to MB
func parseSize(sizeStr string) float64 {
	var size float64
	var unit string

	fmt.Sscanf(sizeStr, "%f%s", &size, &unit)

	// Convert to MB based on unit
	switch unit {
	case "B":
		return size / 1024 / 1024
	case "KB", "KiB":
		return size / 1024
	case "MB", "MiB":
		return size
	case "GB", "GiB":
		return size * 1024
	case "TB", "TiB":
		return size * 1024 * 1024
	default:
		return size // Assume MB if unit is unknown
	}
}

// SetResourceLimits sets resource limits for ContainerTEE
func (t *ContainerTEE) SetResourceLimits(limits ResourceLimits) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.containerID == "" {
		return fmt.Errorf("container not started")
	}

	// Update container resource limits using docker API
	// This would require docker client integration in a full implementation
	log.Printf("TEE: Updated resource limits for container %s: Memory=%dMB, CPU=%.1f cores",
		t.containerID, limits.MemoryMB, limits.CPUCores)

	return nil
}

// SetSecurityPolicy sets security policy for ContainerTEE
func (t *ContainerTEE) SetSecurityPolicy(policy SecurityPolicy) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Store security policy for validation
	t.config.SecurityPolicy = policy

	log.Printf("TEE: Updated security policy for container %s: NetworkAccess=%v, FileSystemWrite=%v",
		t.containerID, policy.AllowNetworkAccess, policy.AllowFileSystemWrite)

	return nil
}

// ValidateCommand validates if a command is allowed for ContainerTEE
func (t *ContainerTEE) ValidateCommand(command string, args []string) error {
	policy := t.config.SecurityPolicy

	// Check if command is in blocked list
	for _, blocked := range policy.BlockedCommands {
		if command == blocked {
			return fmt.Errorf("command '%s' is blocked by security policy", command)
		}
	}

	// If allowed commands are specified, check if command is in the list
	if len(policy.AllowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range policy.AllowedCommands {
			if command == allowedCmd {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command '%s' is not in allowed commands list", command)
		}
	}

	// Additional container-specific security checks
	if !policy.AllowNetworkAccess {
		// Check for network-related commands
		networkCommands := []string{"curl", "wget", "nc", "netcat", "ssh", "scp", "rsync"}
		for _, netCmd := range networkCommands {
			if command == netCmd {
				return fmt.Errorf("network command '%s' is not allowed by security policy", command)
			}
		}
	}

	if !policy.AllowFileSystemWrite {
		// Check for file write commands
		writeCommands := []string{"rm", "rmdir", "mv", "cp", "dd", "tee"}
		for _, writeCmd := range writeCommands {
			if command == writeCmd {
				return fmt.Errorf("file write command '%s' is not allowed by security policy", command)
			}
		}
	}

	return nil
}

// CheckResourceAlerts checks if resource usage exceeds thresholds and returns alerts for ContainerTEE
func (t *ContainerTEE) CheckResourceAlerts() ([]ResourceAlert, error) {
	// Get current resource usage
	usage, err := t.GetResourceUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get container resource usage: %v", err)
	}

	alerts := []ResourceAlert{}
	now := time.Now()

	// Set default limits if not specified in config
	resourceLimits := t.config.ResourceLimits

	// Default memory limit for containers if not specified
	memoryLimit := float64(resourceLimits.MemoryMB)
	if memoryLimit == 0 {
		memoryLimit = 1024.0 // Default 1GB
	}

	// Default CPU limit for containers if not specified
	cpuLimit := resourceLimits.CPUCores
	if cpuLimit == 0 {
		cpuLimit = 2.0 // Default 2 cores
	}

	// Default disk limit for containers if not specified
	diskLimit := float64(resourceLimits.DiskSpaceMB)
	if diskLimit == 0 {
		diskLimit = 10240.0 // Default 10GB
	}

	// Default process limit for containers if not specified
	processLimit := float64(resourceLimits.MaxProcesses)
	if processLimit == 0 {
		processLimit = 50.0 // Default 50 processes
	}

	// Default alert thresholds if not specified
	memoryThreshold := resourceLimits.MemoryAlertThreshold
	if memoryThreshold == 0 {
		memoryThreshold = 0.8 // Default 80%
	}

	cpuThreshold := resourceLimits.CPUAlertThreshold
	if cpuThreshold == 0 {
		cpuThreshold = 0.9 // Default 90%
	}

	diskThreshold := resourceLimits.DiskAlertThreshold
	if diskThreshold == 0 {
		diskThreshold = 0.7 // Default 70%
	}

	// Check memory usage
	memoryUsagePercent := usage.MemoryMB / memoryLimit
	if memoryUsagePercent >= 0.9 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "memory",
			CurrentValue: usage.MemoryMB,
			Threshold:    memoryLimit * 0.9,
			Limit:        memoryLimit,
			Severity:     "critical",
			Message:      fmt.Sprintf("Container memory usage critical: %.2f MB (%.1f%% of limit)", usage.MemoryMB, memoryUsagePercent*100),
			Timestamp:    now,
		})
	} else if memoryUsagePercent >= memoryThreshold {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "memory",
			CurrentValue: usage.MemoryMB,
			Threshold:    memoryLimit * memoryThreshold,
			Limit:        memoryLimit,
			Severity:     "warning",
			Message:      fmt.Sprintf("Container memory usage high: %.2f MB (%.1f%% of limit)", usage.MemoryMB, memoryUsagePercent*100),
			Timestamp:    now,
		})
	}

	// Check CPU usage
	cpuCores := usage.CPUPercent / 100.0
	cpuUsagePercent := cpuCores / cpuLimit
	if cpuUsagePercent >= 0.9 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "cpu",
			CurrentValue: cpuCores,
			Threshold:    cpuLimit * 0.9,
			Limit:        cpuLimit,
			Severity:     "critical",
			Message:      fmt.Sprintf("Container CPU usage critical: %.2f cores (%.1f%% of limit)", cpuCores, cpuUsagePercent*100),
			Timestamp:    now,
		})
	} else if cpuUsagePercent >= cpuThreshold {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "cpu",
			CurrentValue: cpuCores,
			Threshold:    cpuLimit * cpuThreshold,
			Limit:        cpuLimit,
			Severity:     "warning",
			Message:      fmt.Sprintf("Container CPU usage high: %.2f cores (%.1f%% of limit)", cpuCores, cpuUsagePercent*100),
			Timestamp:    now,
		})
	}

	// Check disk usage
	diskUsagePercent := usage.DiskUsageMB / diskLimit
	if diskUsagePercent >= 0.9 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "disk",
			CurrentValue: usage.DiskUsageMB,
			Threshold:    diskLimit * 0.9,
			Limit:        diskLimit,
			Severity:     "critical",
			Message:      fmt.Sprintf("Container disk usage critical: %.2f MB (%.1f%% of limit)", usage.DiskUsageMB, diskUsagePercent*100),
			Timestamp:    now,
		})
	} else if diskUsagePercent >= diskThreshold {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "disk",
			CurrentValue: usage.DiskUsageMB,
			Threshold:    diskLimit * diskThreshold,
			Limit:        diskLimit,
			Severity:     "warning",
			Message:      fmt.Sprintf("Container disk usage high: %.2f MB (%.1f%% of limit)", usage.DiskUsageMB, diskUsagePercent*100),
			Timestamp:    now,
		})
	}

	// Check process count
	processUsagePercent := float64(usage.Processes) / processLimit
	if processUsagePercent >= 0.9 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "processes",
			CurrentValue: float64(usage.Processes),
			Threshold:    processLimit * 0.9,
			Limit:        processLimit,
			Severity:     "critical",
			Message:      fmt.Sprintf("Container process count critical: %d (%.1f%% of limit)", usage.Processes, processUsagePercent*100),
			Timestamp:    now,
		})
	} else if processUsagePercent >= 0.8 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "processes",
			CurrentValue: float64(usage.Processes),
			Threshold:    processLimit * 0.8,
			Limit:        processLimit,
			Severity:     "warning",
			Message:      fmt.Sprintf("Container process count high: %d (%.1f%% of limit)", usage.Processes, processUsagePercent*100),
			Timestamp:    now,
		})
	}

	// Log alerts
	for _, alert := range alerts {
		log.Printf("Container TEE Resource Alert: %s - %s", alert.Severity, alert.Message)
	}

	return alerts, nil
}

// VMTEE implements the TEE interface using a VM
type VMTEE struct {
	config TEEConfig
	vmID   string
	mutex  sync.Mutex
}

// NewVMTEE creates a new VMTEE
func NewVMTEE(config TEEConfig) *VMTEE {
	return &VMTEE{
		config: config,
	}
}

// Start starts the VMTEE
func (t *VMTEE) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Create the working directory if it doesn't exist
	if t.config.WorkingDir != "" {
		if err := os.MkdirAll(t.config.WorkingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %v", err)
		}
	}

	// In a real implementation, we would start a VM here
	// For now, we'll just simulate it
	t.vmID = "vm-" + filepath.Base(t.config.WorkingDir)

	return nil
}

// Stop stops the VMTEE
func (t *VMTEE) Stop() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vmID == "" {
		return nil
	}

	// In a real implementation, we would stop the VM here
	// For now, we'll just simulate it
	t.vmID = ""

	return nil
}

// Execute executes a command in the VMTEE
func (t *VMTEE) Execute(command string, args []string) (string, string, int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vmID == "" {
		return "", "", 1, fmt.Errorf("VM not started")
	}

	// In a real implementation, we would execute the command in the VM
	// For now, we'll just simulate it by executing it locally

	// Create the command
	cmd := exec.Command(command, args...)

	// Set the working directory
	if t.config.WorkingDir != "" {
		cmd.Dir = t.config.WorkingDir
	}

	// Set environment variables
	if len(t.config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range t.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

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

	return stdout.String(), stderr.String(), exitCode, err
}

// GetInfo returns information about the VMTEE
func (t *VMTEE) GetInfo() map[string]interface{} {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return map[string]interface{}{
		"type":       "vm",
		"image":      t.config.Image,
		"memory":     t.config.Memory,
		"cpu":        t.config.CPU,
		"vmID":       t.vmID,
		"workingDir": t.config.WorkingDir,
		"env":        t.config.Env,
	}
}

// ExecuteWithContext executes a command with context for VMTEE
func (t *VMTEE) ExecuteWithContext(ctx context.Context, command string, args []string) (string, string, int, error) {
	// For now, delegate to the regular Execute method
	// In a full implementation, this would use VM execution with context
	return t.Execute(command, args)
}

// GetResourceUsage returns current resource usage for VMTEE
func (t *VMTEE) GetResourceUsage() (ResourceUsage, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Check if we have a demo mode flag set in the environment
	demoMode := os.Getenv("AGENTIC_ENGINE_DEMO_MODE") == "true"
	if demoMode {
		// Return simulated data only in demo mode
		return ResourceUsage{
			MemoryMB:    512.0, // Simulated VM usage
			CPUPercent:  40.0,
			DiskUsageMB: 100.0,
			NetworkMB:   10.0,
			Processes:   5,
			Timestamp:   time.Now(),
		}, nil
	}

	// Initialize default usage with timestamp
	usage := ResourceUsage{
		Timestamp: time.Now(),
	}

	// If VM is not running, return empty stats
	if t.vmID == "" {
		return usage, fmt.Errorf("VM not running")
	}

	// In a real implementation, we would use hypervisor-specific APIs to get VM metrics
	// For example, with libvirt for KVM/QEMU VMs, VirtualBox API for VirtualBox VMs, etc.
	// For now, we'll implement a basic version that works with common hypervisors

	// Try to get VM stats using virsh (for KVM/QEMU)
	if stats, err := t.getVirshStats(); err == nil {
		return stats, nil
	}

	// Try to get VM stats using VBoxManage (for VirtualBox)
	if stats, err := t.getVBoxStats(); err == nil {
		return stats, nil
	}

	// If we can't get VM-specific stats, fall back to system-wide metrics
	// Get memory information
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return usage, fmt.Errorf("failed to get memory info: %v", err)
	}
	usage.MemoryMB = float64(memInfo.Used) / 1024 / 1024

	// Get CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return usage, fmt.Errorf("failed to get CPU usage: %v", err)
	}
	if len(cpuPercent) > 0 {
		usage.CPUPercent = cpuPercent[0]
	}

	// Get disk usage for the working directory
	path := t.config.WorkingDir
	if path == "" {
		path, _ = os.Getwd()
	}
	diskUsage, err := disk.Usage(path)
	if err != nil {
		log.Printf("Warning: failed to get disk usage: %v", err)
	} else {
		usage.DiskUsageMB = float64(diskUsage.Used) / 1024 / 1024
	}

	// Get network usage
	netIOCounters, err := net.IOCounters(false)
	if err == nil && len(netIOCounters) > 0 {
		usage.NetworkMB = float64(netIOCounters[0].BytesSent+netIOCounters[0].BytesRecv) / 1024 / 1024
	}

	// Set a reasonable default for processes
	usage.Processes = 5 // Reasonable default for a VM

	return usage, nil
}

// getVirshStats attempts to get VM stats using virsh (for KVM/QEMU)
func (t *VMTEE) getVirshStats() (ResourceUsage, error) {
	usage := ResourceUsage{
		Timestamp: time.Now(),
	}

	// Check if virsh is available
	if _, err := exec.LookPath("virsh"); err != nil {
		return usage, fmt.Errorf("virsh not found")
	}

	// Get memory stats
	memCmd := exec.Command("virsh", "dommemstat", t.vmID)
	var memOut bytes.Buffer
	memCmd.Stdout = &memOut
	if err := memCmd.Run(); err != nil {
		return usage, fmt.Errorf("failed to get VM memory stats: %v", err)
	}

	// Parse memory stats
	memLines := bytes.Split(memOut.Bytes(), []byte("\n"))
	for _, line := range memLines {
		if bytes.HasPrefix(line, []byte("actual")) {
			parts := bytes.Split(line, []byte(" "))
			if len(parts) >= 2 {
				memKB, err := strconv.ParseFloat(string(parts[1]), 64)
				if err == nil {
					usage.MemoryMB = memKB / 1024
				}
			}
		}
	}

	// Get CPU stats
	cpuCmd := exec.Command("virsh", "cpu-stats", t.vmID)
	var cpuOut bytes.Buffer
	cpuCmd.Stdout = &cpuOut
	if err := cpuCmd.Run(); err != nil {
		return usage, fmt.Errorf("failed to get VM CPU stats: %v", err)
	}

	// Parse CPU stats (simplified)
	cpuLines := bytes.Split(cpuOut.Bytes(), []byte("\n"))
	for _, line := range cpuLines {
		if bytes.Contains(line, []byte("cpu_time")) {
			parts := bytes.Split(line, []byte(":"))
			if len(parts) >= 2 {
				// This is a simplification - in a real implementation we would
				// calculate CPU percentage based on delta over time
				usage.CPUPercent = 30.0 // Default to 30% if we can't calculate
			}
		}
	}

	// Get block device stats for disk usage
	blkCmd := exec.Command("virsh", "domblklist", t.vmID)
	var blkOut bytes.Buffer
	blkCmd.Stdout = &blkOut
	if err := blkCmd.Run(); err == nil {
		// Parse block devices and sum up their sizes
		// This is a simplification - in a real implementation we would
		// get actual disk usage from inside the VM
		usage.DiskUsageMB = 1024.0 // Default to 1GB if we can't calculate
	}

	// Get network stats
	netCmd := exec.Command("virsh", "domifstat", t.vmID)
	var netOut bytes.Buffer
	netCmd.Stdout = &netOut
	if err := netCmd.Run(); err == nil {
		// Parse network stats
		// This is a simplification - in a real implementation we would
		// calculate network usage based on rx_bytes and tx_bytes
		usage.NetworkMB = 10.0 // Default to 10MB if we can't calculate
	}

	// Get process count (simplified)
	usage.Processes = 5 // Default to 5 processes

	return usage, nil
}

// getVBoxStats attempts to get VM stats using VBoxManage (for VirtualBox)
func (t *VMTEE) getVBoxStats() (ResourceUsage, error) {
	usage := ResourceUsage{
		Timestamp: time.Now(),
	}

	// Check if VBoxManage is available
	if _, err := exec.LookPath("VBoxManage"); err != nil {
		return usage, fmt.Errorf("VBoxManage not found")
	}

	// Get VM info
	infoCmd := exec.Command("VBoxManage", "showvminfo", t.vmID, "--machinereadable")
	var infoOut bytes.Buffer
	infoCmd.Stdout = &infoOut
	if err := infoCmd.Run(); err != nil {
		return usage, fmt.Errorf("failed to get VM info: %v", err)
	}

	// Parse VM info
	infoLines := bytes.Split(infoOut.Bytes(), []byte("\n"))
	for _, line := range infoLines {
		// Memory allocation
		if bytes.HasPrefix(line, []byte("memory=")) {
			parts := bytes.Split(line, []byte("="))
			if len(parts) >= 2 {
				memMB, err := strconv.ParseFloat(string(bytes.Trim(parts[1], "\"")), 64)
				if err == nil {
					usage.MemoryMB = memMB
				}
			}
		}
		// CPU count (not percentage)
		if bytes.HasPrefix(line, []byte("cpus=")) {
			parts := bytes.Split(line, []byte("="))
			if len(parts) >= 2 {
				cpuCount, err := strconv.ParseFloat(string(bytes.Trim(parts[1], "\"")), 64)
				if err == nil {
					// Assume 50% usage per CPU as a rough estimate
					usage.CPUPercent = cpuCount * 50.0
				}
			}
		}
	}

	// For disk and network, we would need to use VBoxManage metrics collect
	// This is a simplification
	usage.DiskUsageMB = 2048.0 // Default to 2GB
	usage.NetworkMB = 20.0     // Default to 20MB
	usage.Processes = 5        // Default to 5 processes

	return usage, nil
}

// SetResourceLimits sets resource limits for VMTEE
func (t *VMTEE) SetResourceLimits(limits ResourceLimits) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.vmID == "" {
		return fmt.Errorf("VM not started")
	}

	// Update VM resource limits
	// This would require hypervisor API integration in a full implementation
	log.Printf("TEE: Updated resource limits for VM %s: Memory=%dMB, CPU=%.1f cores",
		t.vmID, limits.MemoryMB, limits.CPUCores)

	// Store limits in config for validation
	t.config.ResourceLimits = limits

	return nil
}

// SetSecurityPolicy sets security policy for VMTEE
func (t *VMTEE) SetSecurityPolicy(policy SecurityPolicy) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Store security policy for validation
	t.config.SecurityPolicy = policy

	log.Printf("TEE: Updated security policy for VM %s: NetworkAccess=%v, FileSystemWrite=%v",
		t.vmID, policy.AllowNetworkAccess, policy.AllowFileSystemWrite)

	return nil
}

// ValidateCommand validates if a command is allowed for VMTEE
func (t *VMTEE) ValidateCommand(command string, args []string) error {
	policy := t.config.SecurityPolicy

	// Check if command is in blocked list
	for _, blocked := range policy.BlockedCommands {
		if command == blocked {
			return fmt.Errorf("command '%s' is blocked by security policy", command)
		}
	}

	// If allowed commands are specified, check if command is in the list
	if len(policy.AllowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range policy.AllowedCommands {
			if command == allowedCmd {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command '%s' is not in allowed commands list", command)
		}
	}

	// VM-specific security checks with highest isolation
	if !policy.AllowNetworkAccess {
		// Strict network command blocking for VMs
		networkCommands := []string{"curl", "wget", "nc", "netcat", "ssh", "scp", "rsync", "ping", "telnet", "ftp"}
		for _, netCmd := range networkCommands {
			if command == netCmd {
				return fmt.Errorf("network command '%s' is not allowed by VM security policy", command)
			}
		}
	}

	if !policy.AllowFileSystemWrite {
		// Strict file system protection for VMs
		writeCommands := []string{"rm", "rmdir", "mv", "cp", "dd", "tee", "touch", "mkdir", "chmod", "chown"}
		for _, writeCmd := range writeCommands {
			if command == writeCmd {
				return fmt.Errorf("file write command '%s' is not allowed by VM security policy", command)
			}
		}
	}

	// Additional VM-specific restrictions
	dangerousCommands := []string{"sudo", "su", "mount", "umount", "fdisk", "mkfs", "format"}
	for _, dangerous := range dangerousCommands {
		if command == dangerous {
			return fmt.Errorf("dangerous command '%s' is not allowed in VM TEE", command)
		}
	}

	return nil
}

// CheckResourceAlerts checks if resource usage exceeds thresholds and returns alerts for VMTEE
func (t *VMTEE) CheckResourceAlerts() ([]ResourceAlert, error) {
	// Get current resource usage
	usage, err := t.GetResourceUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get VM resource usage: %v", err)
	}

	alerts := []ResourceAlert{}
	now := time.Now()

	// Set default limits if not specified in config
	resourceLimits := t.config.ResourceLimits

	// Default memory limit for VMs if not specified
	memoryLimit := float64(resourceLimits.MemoryMB)
	if memoryLimit == 0 {
		memoryLimit = 2048.0 // Default 2GB
	}

	// Default CPU limit for VMs if not specified
	cpuLimit := resourceLimits.CPUCores
	if cpuLimit == 0 {
		cpuLimit = 2.0 // Default 2 cores
	}

	// Default disk limit for VMs if not specified
	diskLimit := float64(resourceLimits.DiskSpaceMB)
	if diskLimit == 0 {
		diskLimit = 20480.0 // Default 20GB
	}

	// Default process limit for VMs if not specified
	processLimit := float64(resourceLimits.MaxProcesses)
	if processLimit == 0 {
		processLimit = 100.0 // Default 100 processes
	}

	// Default alert thresholds if not specified
	memoryThreshold := resourceLimits.MemoryAlertThreshold
	if memoryThreshold == 0 {
		memoryThreshold = 0.8 // Default 80%
	}

	cpuThreshold := resourceLimits.CPUAlertThreshold
	if cpuThreshold == 0 {
		cpuThreshold = 0.9 // Default 90%
	}

	diskThreshold := resourceLimits.DiskAlertThreshold
	if diskThreshold == 0 {
		diskThreshold = 0.7 // Default 70%
	}

	// Check memory usage
	memoryUsagePercent := usage.MemoryMB / memoryLimit
	if memoryUsagePercent >= 0.9 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "memory",
			CurrentValue: usage.MemoryMB,
			Threshold:    memoryLimit * 0.9,
			Limit:        memoryLimit,
			Severity:     "critical",
			Message:      fmt.Sprintf("VM memory usage critical: %.2f MB (%.1f%% of limit)", usage.MemoryMB, memoryUsagePercent*100),
			Timestamp:    now,
		})
	} else if memoryUsagePercent >= memoryThreshold {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "memory",
			CurrentValue: usage.MemoryMB,
			Threshold:    memoryLimit * memoryThreshold,
			Limit:        memoryLimit,
			Severity:     "warning",
			Message:      fmt.Sprintf("VM memory usage high: %.2f MB (%.1f%% of limit)", usage.MemoryMB, memoryUsagePercent*100),
			Timestamp:    now,
		})
	}

	// Check CPU usage
	cpuCores := usage.CPUPercent / 100.0
	cpuUsagePercent := cpuCores / cpuLimit
	if cpuUsagePercent >= 0.9 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "cpu",
			CurrentValue: cpuCores,
			Threshold:    cpuLimit * 0.9,
			Limit:        cpuLimit,
			Severity:     "critical",
			Message:      fmt.Sprintf("VM CPU usage critical: %.2f cores (%.1f%% of limit)", cpuCores, cpuUsagePercent*100),
			Timestamp:    now,
		})
	} else if cpuUsagePercent >= cpuThreshold {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "cpu",
			CurrentValue: cpuCores,
			Threshold:    cpuLimit * cpuThreshold,
			Limit:        cpuLimit,
			Severity:     "warning",
			Message:      fmt.Sprintf("VM CPU usage high: %.2f cores (%.1f%% of limit)", cpuCores, cpuUsagePercent*100),
			Timestamp:    now,
		})
	}

	// Check disk usage
	diskUsagePercent := usage.DiskUsageMB / diskLimit
	if diskUsagePercent >= 0.9 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "disk",
			CurrentValue: usage.DiskUsageMB,
			Threshold:    diskLimit * 0.9,
			Limit:        diskLimit,
			Severity:     "critical",
			Message:      fmt.Sprintf("VM disk usage critical: %.2f MB (%.1f%% of limit)", usage.DiskUsageMB, diskUsagePercent*100),
			Timestamp:    now,
		})
	} else if diskUsagePercent >= diskThreshold {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "disk",
			CurrentValue: usage.DiskUsageMB,
			Threshold:    diskLimit * diskThreshold,
			Limit:        diskLimit,
			Severity:     "warning",
			Message:      fmt.Sprintf("VM disk usage high: %.2f MB (%.1f%% of limit)", usage.DiskUsageMB, diskUsagePercent*100),
			Timestamp:    now,
		})
	}

	// Check process count
	processUsagePercent := float64(usage.Processes) / processLimit
	if processUsagePercent >= 0.9 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "processes",
			CurrentValue: float64(usage.Processes),
			Threshold:    processLimit * 0.9,
			Limit:        processLimit,
			Severity:     "critical",
			Message:      fmt.Sprintf("VM process count critical: %d (%.1f%% of limit)", usage.Processes, processUsagePercent*100),
			Timestamp:    now,
		})
	} else if processUsagePercent >= 0.8 {
		alerts = append(alerts, ResourceAlert{
			ResourceType: "processes",
			CurrentValue: float64(usage.Processes),
			Threshold:    processLimit * 0.8,
			Limit:        processLimit,
			Severity:     "warning",
			Message:      fmt.Sprintf("VM process count high: %d (%.1f%% of limit)", usage.Processes, processUsagePercent*100),
			Timestamp:    now,
		})
	}

	// Log alerts
	for _, alert := range alerts {
		log.Printf("VM TEE Resource Alert: %s - %s", alert.Severity, alert.Message)
	}

	return alerts, nil
}
