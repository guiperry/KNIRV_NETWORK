package knirvgraph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewManager(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{
		BinaryPath:   "./knirvgraph",
		Port:         7090,
		P2PPort:      7091,
		APIPort:      7092,
		DataPath:     "/tmp/test_knirvgraph",
		StartTimeout: 10 * time.Second,
		StopTimeout:  5 * time.Second,
	}

	mgr := NewManager(cfg, logger)

	assert.NotNil(t, mgr)
	assert.Equal(t, 7090, mgr.config.Port)
	assert.Equal(t, 7091, mgr.config.P2PPort)
	assert.Equal(t, 7092, mgr.config.APIPort)
	assert.Equal(t, "/tmp/test_knirvgraph", mgr.config.DataPath)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, 7090, cfg.Port)
	assert.Equal(t, 7091, cfg.P2PPort)
	assert.Equal(t, 7092, cfg.APIPort)
	assert.Equal(t, "./data", cfg.DataPath)
	assert.Equal(t, 30*time.Second, cfg.StartTimeout)
	assert.Equal(t, 10*time.Second, cfg.StopTimeout)
}

func TestManager_GetPort(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{
		Port: 7090,
	}

	mgr := NewManager(cfg, logger)
	assert.Equal(t, 7090, mgr.GetPort())
}

func TestManager_GetP2PPort(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{
		P2PPort: 7091,
	}

	mgr := NewManager(cfg, logger)
	assert.Equal(t, 7091, mgr.GetP2PPort())
}

func TestManager_GetAPIPort(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{
		APIPort: 7092,
	}

	mgr := NewManager(cfg, logger)
	assert.Equal(t, 7092, mgr.GetAPIPort())
}

func TestManager_GetBaseURL(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{
		Port: 7090,
	}

	mgr := NewManager(cfg, logger)
	assert.Equal(t, "http://localhost:7090", mgr.GetBaseURL())
}

func TestManager_GetConfig(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{
		BinaryPath: "/usr/bin/knirvgraph",
		Port:       7090,
		P2PPort:    7091,
		APIPort:    7092,
		DataPath:   "/tmp/knirvgraph",
	}

	mgr := NewManager(cfg, logger)
	returnedCfg := mgr.GetConfig()

	assert.Equal(t, cfg, returnedCfg)
}

func TestManager_IsRunning(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{}

	mgr := NewManager(cfg, logger)
	assert.False(t, mgr.IsRunning())
}

func TestManager_GetStatus_NotRunning(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{
		Port: 7090,
	}

	mgr := NewManager(cfg, logger)
	status, err := mgr.GetStatus()

	require.NoError(t, err)
	assert.Equal(t, "stopped", status.Status)
	assert.False(t, status.Running)
	assert.Equal(t, 7090, status.Port)
}

func TestManager_IsHealthy_NotRunning(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &ManagerConfig{}

	mgr := NewManager(cfg, logger)
	assert.False(t, mgr.IsHealthy())
}
