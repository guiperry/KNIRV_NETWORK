package host

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHostController(t *testing.T) {
	config := HostConfig{
		EnableMonitoring:        true,
		MonitoringInterval:      30 * time.Second,
		EnableP2P:              false,
		EnableTEE:              false,
		ContainerRuntime:       "podman",
		EnableSecurityMonitoring: true,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)
	require.NotNil(t, controller)

	assert.Equal(t, config.EnableMonitoring, controller.config.EnableMonitoring)
	assert.Equal(t, config.ContainerRuntime, controller.config.ContainerRuntime)
	assert.False(t, controller.isRunning)
}

func TestHostControllerStartStop(t *testing.T) {
	config := HostConfig{
		EnableMonitoring:   true,
		MonitoringInterval: 100 * time.Millisecond, // Short interval for testing
		EnableP2P:         false,
		EnableTEE:         false,
		ContainerRuntime:  "podman",
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	// Test start
	err = controller.Start()
	assert.NoError(t, err)
	assert.True(t, controller.IsRunning())

	// Wait a bit for monitoring to run
	time.Sleep(200 * time.Millisecond)

	// Test stop
	err = controller.Stop()
	assert.NoError(t, err)
	assert.False(t, controller.IsRunning())

	// Test double start should return error
	err = controller.Start()
	assert.NoError(t, err)
	err = controller.Start()
	assert.Error(t, err)

	controller.Stop()
}

func TestHostControllerGetSystemInfo(t *testing.T) {
	config := HostConfig{
		EnableMonitoring: false,
		EnableP2P:       false,
		EnableTEE:       false,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	info := controller.GetSystemInfo()
	assert.NotNil(t, info)
	assert.NotEmpty(t, info["hostname"])
	assert.NotEmpty(t, info["os"])
	assert.NotEmpty(t, info["architecture"])
	assert.Contains(t, info, "cpu_count")
	assert.Contains(t, info, "memory_total")
}

func TestHostControllerGetStatus(t *testing.T) {
	config := HostConfig{
		EnableMonitoring: true,
		EnableP2P:       false,
		EnableTEE:       false,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	status := controller.GetStatus()
	assert.NotNil(t, status)
	assert.Contains(t, status, "running")
	assert.Contains(t, status, "uptime")
	assert.Contains(t, status, "services")
	assert.Equal(t, false, status["running"])

	// Start controller and check status again
	err = controller.Start()
	require.NoError(t, err)
	defer controller.Stop()

	status = controller.GetStatus()
	assert.Equal(t, true, status["running"])
}

func TestHostControllerGetMetrics(t *testing.T) {
	config := HostConfig{
		EnableMonitoring: true,
		EnableP2P:       false,
		EnableTEE:       false,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	err = controller.Start()
	require.NoError(t, err)
	defer controller.Stop()

	// Wait for at least one monitoring cycle
	time.Sleep(200 * time.Millisecond)

	metrics := controller.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "cpu_usage")
	assert.Contains(t, metrics, "memory_usage")
	assert.Contains(t, metrics, "disk_usage")
	assert.Contains(t, metrics, "network_stats")
}

func TestHostControllerExecuteCommand(t *testing.T) {
	config := HostConfig{
		EnableMonitoring: false,
		EnableP2P:       false,
		EnableTEE:       false,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	// Test simple command
	result, err := controller.ExecuteCommand("echo", []string{"hello", "world"})
	assert.NoError(t, err)
	assert.Contains(t, result, "hello world")

	// Test command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = controller.ExecuteCommandWithContext(ctx, "sleep", []string{"1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestHostControllerListProcesses(t *testing.T) {
	config := HostConfig{
		EnableMonitoring: false,
		EnableP2P:       false,
		EnableTEE:       false,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	processes, err := controller.ListProcesses()
	assert.NoError(t, err)
	assert.NotEmpty(t, processes)

	// Check that we have at least the current process
	found := false
	for _, proc := range processes {
		if proc["name"] == "go" || proc["name"] == "test" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find at least the test process")
}

func TestHostControllerNetworkInterfaces(t *testing.T) {
	config := HostConfig{
		EnableMonitoring: false,
		EnableP2P:       false,
		EnableTEE:       false,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	interfaces, err := controller.GetNetworkInterfaces()
	assert.NoError(t, err)
	assert.NotEmpty(t, interfaces)

	// Should have at least loopback interface
	found := false
	for _, iface := range interfaces {
		if iface["name"] == "lo" || iface["name"] == "loopback" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find loopback interface")
}

func TestHostControllerContainerOperations(t *testing.T) {
	config := HostConfig{
		EnableMonitoring: false,
		EnableP2P:       false,
		EnableTEE:       false,
		ContainerRuntime: "podman",
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	// Test listing containers (should not error even if no containers)
	containers, err := controller.ListContainers()
	assert.NoError(t, err)
	assert.NotNil(t, containers)
}

func TestHostControllerSecurityFeatures(t *testing.T) {
	config := HostConfig{
		EnableMonitoring:        false,
		EnableP2P:              false,
		EnableTEE:              false,
		EnableSecurityMonitoring: true,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	// Test security status
	status := controller.GetSecurityStatus()
	assert.NotNil(t, status)
	assert.Contains(t, status, "tee_enabled")
	assert.Contains(t, status, "security_monitoring")
}

func TestHostControllerConcurrency(t *testing.T) {
	config := HostConfig{
		EnableMonitoring:   true,
		MonitoringInterval: 50 * time.Millisecond,
		EnableP2P:         false,
		EnableTEE:         false,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	err = controller.Start()
	require.NoError(t, err)
	defer controller.Stop()

	// Test concurrent access to metrics
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 5; j++ {
				metrics := controller.GetMetrics()
				assert.NotNil(t, metrics)
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHostControllerConfigValidation(t *testing.T) {
	// Test invalid monitoring interval
	config := HostConfig{
		EnableMonitoring:   true,
		MonitoringInterval: 0, // Invalid
		EnableP2P:         false,
		EnableTEE:         false,
	}

	controller, err := NewHostController(config)
	assert.NoError(t, err) // Should create with default values

	// Verify default was applied
	assert.Equal(t, 30*time.Second, controller.config.MonitoringInterval)
}

func TestHostControllerResourceLimits(t *testing.T) {
	config := HostConfig{
		EnableMonitoring: false,
		EnableP2P:       false,
		EnableTEE:       false,
	}

	controller, err := NewHostController(config)
	require.NoError(t, err)

	// Test resource usage calculation
	usage := controller.GetResourceUsage()
	assert.NotNil(t, usage)
	assert.Contains(t, usage, "cpu_percent")
	assert.Contains(t, usage, "memory_percent")
	assert.Contains(t, usage, "disk_percent")

	// Verify values are reasonable
	cpuPercent, ok := usage["cpu_percent"].(float64)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, cpuPercent, 0.0)
	assert.LessOrEqual(t, cpuPercent, 100.0)
}
