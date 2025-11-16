package cde

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCDEService_Start(t *testing.T) {
	// Create a minimal config for testing
	config := CDEConfig{
		WorkspaceRoot:       "/tmp/test-workspaces",
		ProjectStoragePath:  "/tmp/test-projects",
		MaxEnvironments:     5,
		MaxSessionsPerUser:  3,
		MaxProjectsPerUser:  5,
		SessionTimeout:      30 * time.Minute,
		DefaultTimeout:      10 * time.Minute,
		MaxCPUPerEnv:        2.0,
		MaxMemoryPerEnv:     1024 * 1024 * 1024, // 1GB
		MaxDiskPerEnv:       10 * 1024 * 1024 * 1024, // 10GB
		EnableSandboxing:    false,
		EnableNetworkIsolation: false,
	}

	// Create service with nil dependencies for basic testing
	service, err := NewCDEService(nil, nil, config)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Test Start
	err = service.Start()
	assert.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Test Stop
	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.IsRunning())
}

func TestCDEService_Stop(t *testing.T) {
	// Create a minimal config for testing
	config := CDEConfig{
		WorkspaceRoot:       "/tmp/test-workspaces",
		ProjectStoragePath:  "/tmp/test-projects",
		MaxEnvironments:     5,
		MaxSessionsPerUser:  3,
		MaxProjectsPerUser:  5,
		SessionTimeout:      30 * time.Minute,
		DefaultTimeout:      10 * time.Minute,
		MaxCPUPerEnv:        2.0,
		MaxMemoryPerEnv:     1024 * 1024 * 1024, // 1GB
		MaxDiskPerEnv:       10 * 1024 * 1024 * 1024, // 10GB
		EnableSandboxing:    false,
		EnableNetworkIsolation: false,
	}

	// Create service with nil dependencies for basic testing
	service, err := NewCDEService(nil, nil, config)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Test Stop without starting (should not error)
	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.IsRunning())
}

func TestCDEService_Start_AlreadyRunning(t *testing.T) {
	// Create a minimal config for testing
	config := CDEConfig{
		WorkspaceRoot:       "/tmp/test-workspaces",
		ProjectStoragePath:  "/tmp/test-projects",
		MaxEnvironments:     5,
		MaxSessionsPerUser:  3,
		MaxProjectsPerUser:  5,
		SessionTimeout:      30 * time.Minute,
		DefaultTimeout:      10 * time.Minute,
		MaxCPUPerEnv:        2.0,
		MaxMemoryPerEnv:     1024 * 1024 * 1024, // 1GB
		MaxDiskPerEnv:       10 * 1024 * 1024 * 1024, // 10GB
		EnableSandboxing:    false,
		EnableNetworkIsolation: false,
	}

	// Create service with nil dependencies for basic testing
	service, err := NewCDEService(nil, nil, config)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Start service
	err = service.Start()
	assert.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Try to start again (should fail)
	err = service.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Clean up
	service.Stop()
}

func TestCDEService_Stop_NotRunning(t *testing.T) {
	// Create a minimal config for testing
	config := CDEConfig{
		WorkspaceRoot:       "/tmp/test-workspaces",
		ProjectStoragePath:  "/tmp/test-projects",
		MaxEnvironments:     5,
		MaxSessionsPerUser:  3,
		MaxProjectsPerUser:  5,
		SessionTimeout:      30 * time.Minute,
		DefaultTimeout:      10 * time.Minute,
		MaxCPUPerEnv:        2.0,
		MaxMemoryPerEnv:     1024 * 1024 * 1024, // 1GB
		MaxDiskPerEnv:       10 * 1024 * 1024 * 1024, // 10GB
		EnableSandboxing:    false,
		EnableNetworkIsolation: false,
	}

	// Create service with nil dependencies for basic testing
	service, err := NewCDEService(nil, nil, config)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Stop without starting (should not error)
	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.IsRunning())
}

func TestCDEService_GetStatus(t *testing.T) {
	// Create a minimal config for testing
	config := CDEConfig{
		WorkspaceRoot:       "/tmp/test-workspaces",
		ProjectStoragePath:  "/tmp/test-projects",
		MaxEnvironments:     5,
		MaxSessionsPerUser:  3,
		MaxProjectsPerUser:  5,
		SessionTimeout:      30 * time.Minute,
		DefaultTimeout:      10 * time.Minute,
		MaxCPUPerEnv:        2.0,
		MaxMemoryPerEnv:     1024 * 1024 * 1024, // 1GB
		MaxDiskPerEnv:       10 * 1024 * 1024 * 1024, // 10GB
		EnableSandboxing:    false,
		EnableNetworkIsolation: false,
	}

	// Create service with nil dependencies for basic testing
	service, err := NewCDEService(nil, nil, config)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Test GetStatus when not running
	status := service.GetStatus()
	assert.False(t, status["running"].(bool))
	assert.Equal(t, config, status["config"].(CDEConfig))
	assert.Equal(t, 0, status["projects"].(int))

	// Start service and test again
	err = service.Start()
	require.NoError(t, err)

	status = service.GetStatus()
	assert.True(t, status["running"].(bool))
	assert.Equal(t, config, status["config"].(CDEConfig))
	assert.Equal(t, 0, status["projects"].(int))

	// Clean up
	service.Stop()
}

func TestCDEService_IsRunning(t *testing.T) {
	// Create a minimal config for testing
	config := CDEConfig{
		WorkspaceRoot:       "/tmp/test-workspaces",
		ProjectStoragePath:  "/tmp/test-projects",
		MaxEnvironments:     5,
		MaxSessionsPerUser:  3,
		MaxProjectsPerUser:  5,
		SessionTimeout:      30 * time.Minute,
		DefaultTimeout:      10 * time.Minute,
		MaxCPUPerEnv:        2.0,
		MaxMemoryPerEnv:     1024 * 1024 * 1024, // 1GB
		MaxDiskPerEnv:       10 * 1024 * 1024 * 1024, // 10GB
		EnableSandboxing:    false,
		EnableNetworkIsolation: false,
	}

	// Create service with nil dependencies for basic testing
	service, err := NewCDEService(nil, nil, config)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Initially not running
	assert.False(t, service.IsRunning())

	// Start and check
	err = service.Start()
	require.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Stop and check
	err = service.Stop()
	require.NoError(t, err)
	assert.False(t, service.IsRunning())
}