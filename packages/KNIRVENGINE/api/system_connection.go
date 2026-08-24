package api

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// SystemConnection implements TargetSystemConnection for system-level access
type SystemConnection struct {
	target    *TargetSystem
	connected bool
	startTime time.Time
	mutex     sync.RWMutex
}

// NewSystemConnection creates a new system connection
func NewSystemConnection(target *TargetSystem) (TargetSystemConnection, error) {
	return &SystemConnection{
		target: target,
	}, nil
}

// Connect establishes the system connection
func (c *SystemConnection) Connect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return nil
	}

	// Basic system access test
	if _, err := os.Getwd(); err != nil {
		return fmt.Errorf("system access test failed: %v", err)
	}

	c.connected = true
	c.startTime = time.Now()
	return nil
}

// Disconnect closes the system connection
func (c *SystemConnection) Disconnect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.connected = false
	return nil
}

// IsConnected returns the connection status
func (c *SystemConnection) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// GetCapabilities returns available system capabilities
func (c *SystemConnection) GetCapabilities() []string {
	return []string{
		"execute_command",
		"get_system_info",
		"get_environment",
		"list_processes",
		"get_memory_info",
		"get_disk_info",
	}
}

// Execute executes a system operation
func (c *SystemConnection) Execute(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("system not connected")
	}

	switch operation {
	case "execute_command":
		return c.executeCommand(ctx, params)
	case "get_system_info":
		return c.getSystemInfo(ctx, params)
	case "get_environment":
		return c.getEnvironment(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// GetStatus returns detailed system status
func (c *SystemConnection) GetStatus() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"connected": c.connected,
		"type":      "system",
		"uptime":    time.Since(c.startTime).String(),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
	}
}

// GetType returns the target system type
func (c *SystemConnection) GetType() TargetSystemType {
	return TargetTypeSystem
}

// System operations (simplified implementations)

func (c *SystemConnection) executeCommand(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command := getStringParam(params, "command", "")
	if command == "" {
		return nil, fmt.Errorf("command parameter is required")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	output, err := cmd.Output()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return map[string]interface{}{
		"success":  err == nil,
		"command":  command,
		"output":   string(output),
		"exitCode": exitCode,
		"error":    fmt.Sprintf("%v", err),
	}, nil
}

func (c *SystemConnection) getSystemInfo(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx    // Context not used in current implementation
	_ = params // Parameters not used in current implementation
	hostname, _ := os.Hostname()
	wd, _ := os.Getwd()

	return map[string]interface{}{
		"success":      true,
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"hostname":     hostname,
		"workingDir":   wd,
		"goVersion":    runtime.Version(),
		"numCPU":       runtime.NumCPU(),
		"numGoroutine": runtime.NumGoroutine(),
	}, nil
}

func (c *SystemConnection) getEnvironment(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx    // Context not used in current implementation
	_ = params // Parameters not used in current implementation
	env := os.Environ()
	envMap := make(map[string]string)

	for _, e := range env {
		// Split on first '=' to handle values that contain '='
		if idx := findFirstEqual(e); idx != -1 {
			key := e[:idx]
			value := e[idx+1:]
			envMap[key] = value
		}
	}

	return map[string]interface{}{
		"success":     true,
		"environment": envMap,
		"count":       len(envMap),
	}, nil
}

func findFirstEqual(s string) int {
	for i, c := range s {
		if c == '=' {
			return i
		}
	}
	return -1
}
