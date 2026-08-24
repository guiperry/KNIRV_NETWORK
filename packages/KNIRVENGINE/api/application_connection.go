package api

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// ApplicationConnection implements TargetSystemConnection for application integration
type ApplicationConnection struct {
	target    *TargetSystem
	connected bool
	startTime time.Time
	mutex     sync.RWMutex
}

// NewApplicationConnection creates a new application connection
func NewApplicationConnection(target *TargetSystem) (TargetSystemConnection, error) {
	return &ApplicationConnection{
		target: target,
	}, nil
}

// Connect establishes the application connection
func (c *ApplicationConnection) Connect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return nil
	}

	// Test if target application is available
	appName := getStringFromConfig(c.target.Config, "appName", "")
	if appName != "" {
		if _, err := exec.LookPath(appName); err != nil {
			return fmt.Errorf("application not found: %s", appName)
		}
	}

	c.connected = true
	c.startTime = time.Now()
	return nil
}

// Disconnect closes the application connection
func (c *ApplicationConnection) Disconnect(ctx context.Context) error {
	_ = ctx // Context not used in current implementation
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.connected = false
	return nil
}

// IsConnected returns the connection status
func (c *ApplicationConnection) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// GetCapabilities returns available application capabilities
func (c *ApplicationConnection) GetCapabilities() []string {
	return []string{
		"launch_app",
		"send_command",
		"get_app_info",
		"list_processes",
		"kill_process",
	}
}

// Execute executes an application operation
func (c *ApplicationConnection) Execute(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("application not connected")
	}

	switch operation {
	case "launch_app":
		return c.launchApp(ctx, params)
	case "send_command":
		return c.sendCommand(ctx, params)
	case "get_app_info":
		return c.getAppInfo(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// GetStatus returns detailed application status
func (c *ApplicationConnection) GetStatus() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"connected": c.connected,
		"type":      "application",
		"uptime":    time.Since(c.startTime).String(),
	}
}

// GetType returns the target system type
func (c *ApplicationConnection) GetType() TargetSystemType {
	return TargetTypeApplication
}

// Application operations (simplified implementations)

func (c *ApplicationConnection) launchApp(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	appName := getStringParam(params, "app", "")
	if appName == "" {
		return nil, fmt.Errorf("app parameter is required")
	}

	cmd := exec.CommandContext(ctx, appName)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to launch app: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"app":     appName,
		"pid":     cmd.Process.Pid,
	}, nil
}

func (c *ApplicationConnection) sendCommand(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command := getStringParam(params, "command", "")
	if command == "" {
		return nil, fmt.Errorf("command parameter is required")
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"command": command,
		"output":  string(output),
	}, nil
}

func (c *ApplicationConnection) getAppInfo(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context not used in current implementation
	appName := getStringParam(params, "app", "")
	if appName == "" {
		return nil, fmt.Errorf("app parameter is required")
	}

	path, err := exec.LookPath(appName)
	if err != nil {
		return nil, fmt.Errorf("app not found: %v", err)
	}

	return map[string]interface{}{
		"success": true,
		"app":     appName,
		"path":    path,
	}, nil
}
