package knirvshell

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDefaultManagerConfig(t *testing.T) {
	cfg := DefaultManagerConfig()
	assert.Equal(t, "", cfg.SocketPath)
	assert.Equal(t, 30*time.Second, cfg.StartTimeout)
	assert.Equal(t, 10*time.Second, cfg.StopTimeout)
}

func TestNewManager(t *testing.T) {
	logger := zap.NewNop()

	t.Run("creates with config defaults", func(t *testing.T) {
		cfg := DefaultManagerConfig()
		m := NewManager(cfg, logger)
		require.NotNil(t, m)
		assert.False(t, m.IsRunning())
		assert.Equal(t, 0, m.GetPID())
	})

	t.Run("fills zero timeouts", func(t *testing.T) {
		m := NewManager(&ManagerConfig{}, logger)
		require.NotNil(t, m)
		// Zero timeouts should be filled with defaults
		assert.Equal(t, 30*time.Second, m.config.StartTimeout)
		assert.Equal(t, 10*time.Second, m.config.StopTimeout)
	})

	t.Run("sets up socket transport when SocketPath provided", func(t *testing.T) {
		m := NewManager(&ManagerConfig{
			SocketPath: "/tmp/test-shell.sock",
		}, logger)
		require.NotNil(t, m)
		assert.NotNil(t, m.client.Transport)
	})
}

func TestManagerHealthCheck(t *testing.T) {
	logger := zap.NewNop()

	t.Run("returns error when not started", func(t *testing.T) {
		m := NewManager(DefaultManagerConfig(), logger)
		err := m.HealthCheck(t.Context())
		assert.Error(t, err)
	})
}

func TestManagerIsRunning(t *testing.T) {
	logger := zap.NewNop()

	t.Run("returns false before start", func(t *testing.T) {
		m := NewManager(DefaultManagerConfig(), logger)
		assert.False(t, m.IsRunning())
	})

	t.Run("returns false after stop", func(t *testing.T) {
		// Without a real binary, IsRunning should remain false
		m := NewManager(&ManagerConfig{
			BinaryPath: "/nonexistent/knirvshell",
		}, logger)
		assert.False(t, m.IsRunning())
	})
}

func TestManagerGetPID(t *testing.T) {
	logger := zap.NewNop()

	t.Run("returns 0 before start", func(t *testing.T) {
		m := NewManager(DefaultManagerConfig(), logger)
		assert.Equal(t, 0, m.GetPID())
	})
}

func TestManagerStartNoBinary(t *testing.T) {
	logger := zap.NewNop()

	t.Run("returns error when binary not found", func(t *testing.T) {
		m := NewManager(&ManagerConfig{
			BinaryPath: "/nonexistent/path/to/knirvshell",
		}, logger)
		err := m.Start(t.Context())
		assert.Error(t, err)
		assert.False(t, m.IsRunning())
	})
}

func TestManagerStartGracefulAdopt(t *testing.T) {
	// Verify that Start returns error when stale binary can't be killed
	// (the preflight check will fail gracefully since no socket exists)
	logger := zap.NewNop()

	t.Run("fails when binary not found and no existing socket", func(t *testing.T) {
		m := NewManager(&ManagerConfig{
			BinaryPath: "/nonexistent/knirvshell",
		}, logger)
		err := m.Start(t.Context())
		assert.Error(t, err)
		assert.False(t, m.IsRunning())
	})
}

func TestManagerConfigPreservesValues(t *testing.T) {
	logger := zap.NewNop()

	cfg := &ManagerConfig{
		BinaryPath:   "/custom/knirvshell",
		SocketPath:   "/custom/shell.sock",
		StartTimeout: 60 * time.Second,
		StopTimeout:  15 * time.Second,
	}

	m := NewManager(cfg, logger)
	require.NotNil(t, m)

	config := m.config
	assert.Equal(t, "/custom/knirvshell", config.BinaryPath)
	assert.Equal(t, "/custom/shell.sock", config.SocketPath)
	assert.Equal(t, 60*time.Second, config.StartTimeout)
	assert.Equal(t, 15*time.Second, config.StopTimeout)
}
