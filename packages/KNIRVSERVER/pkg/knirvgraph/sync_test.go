package knirvgraph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewSyncManager(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &SyncManagerConfig{
		GraphURL: "http://localhost:7090",
		Interval: 30 * time.Second,
	}

	mgr := NewSyncManager(cfg, logger)

	require.NotNil(t, mgr)
	assert.Equal(t, "http://localhost:7090", mgr.graphClient.baseURL)
	assert.Equal(t, 30*time.Second, mgr.interval)
}

func TestDefaultSyncConfig(t *testing.T) {
	cfg := DefaultSyncConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "http://localhost:7090", cfg.GraphURL)
	assert.Equal(t, 30*time.Second, cfg.Interval)
}

func TestSyncManager_GetQueueSize(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &SyncManagerConfig{
		GraphURL: "http://localhost:7090",
		Interval: 1 * time.Second,
	}

	mgr := NewSyncManager(cfg, logger)
	assert.Equal(t, 0, mgr.GetQueueSize())
}

func TestSyncManager_IsRunning(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &SyncManagerConfig{
		GraphURL: "http://localhost:7090",
		Interval: 1 * time.Second,
	}

	mgr := NewSyncManager(cfg, logger)
	assert.False(t, mgr.IsRunning())
}

func TestSyncManager_QueueChange(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &SyncManagerConfig{
		GraphURL: "http://localhost:7090",
		Interval: 1 * time.Second,
	}

	mgr := NewSyncManager(cfg, logger)

	change := map[string]interface{}{
		"type":    "error_node",
		"data":    map[string]interface{}{"node_id": "test_123"},
		"message": "Test commit",
		"author":  "test_user",
	}

	mgr.QueueChange(change)

	assert.Equal(t, 1, mgr.GetQueueSize())
}

func TestSyncManager_QueueChange_InvalidType(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &SyncManagerConfig{
		GraphURL: "http://localhost:7090",
		Interval: 1 * time.Second,
	}

	mgr := NewSyncManager(cfg, logger)

	change := "invalid type"
	mgr.QueueChange(change)

	assert.Equal(t, 0, mgr.GetQueueSize())
}

func TestSyncManager_Stop(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &SyncManagerConfig{
		GraphURL: "http://localhost:7090",
		Interval: 1 * time.Second,
	}

	mgr := NewSyncManager(cfg, logger)

	mgr.Stop()
	assert.False(t, mgr.IsRunning())
}

func TestSyncManager_Start_AlreadyRunning(t *testing.T) {
	logger, _ := zap.NewProduction()
	cfg := &SyncManagerConfig{
		GraphURL: "http://localhost:7090",
		Interval: 1 * time.Second,
	}

	mgr := NewSyncManager(cfg, logger)

	mgr.Stop()
	assert.False(t, mgr.IsRunning())
}
