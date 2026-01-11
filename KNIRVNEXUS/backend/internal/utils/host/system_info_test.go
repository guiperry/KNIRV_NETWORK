package host

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystemInfoCollector(t *testing.T) {
	ctx := context.Background()
	interval := 30 * time.Second

	collector, err := NewSystemInfoCollector(ctx, interval)
	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Equal(t, interval, collector.interval)
	assert.NotNil(t, collector.currentInfo)
	assert.True(t, collector.lastUpdate.After(time.Now().Add(-time.Second)))
}

func TestSystemInfoCollector_Start_Stop(t *testing.T) {
	ctx := context.Background()
	collector, err := NewSystemInfoCollector(ctx, time.Second)
	require.NoError(t, err)

	// Test Start
	err = collector.Start()
	assert.NoError(t, err)
	assert.True(t, collector.running)

	// Test Stop
	err = collector.Stop()
	assert.NoError(t, err)
	assert.False(t, collector.running)
}

func TestSystemInfoCollector_Start_AlreadyRunning(t *testing.T) {
	ctx := context.Background()
	collector, err := NewSystemInfoCollector(ctx, time.Second)
	require.NoError(t, err)

	// Start once
	err = collector.Start()
	assert.NoError(t, err)

	// Try to start again
	err = collector.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	collector.Stop()
}

func TestSystemInfoCollector_GetCurrentInfo(t *testing.T) {
	ctx := context.Background()
	collector, err := NewSystemInfoCollector(ctx, time.Second)
	require.NoError(t, err)

	info, err := collector.GetCurrentInfo()
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.NotEmpty(t, info.Hostname)
	assert.NotEmpty(t, info.OS)
	assert.NotEmpty(t, info.Architecture)
}

func TestSystemInfoCollector_HealthCheck(t *testing.T) {
	ctx := context.Background()
	collector, err := NewSystemInfoCollector(ctx, time.Second)
	require.NoError(t, err)

	// Test when not running
	err = collector.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start and test healthy
	err = collector.Start()
	assert.NoError(t, err)

	err = collector.HealthCheck()
	assert.NoError(t, err)

	// Test stale data
	collector.mu.Lock()
	collector.lastUpdate = time.Now().Add(-10 * time.Minute)
	collector.mu.Unlock()

	err = collector.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stale")

	collector.Stop()
}

func TestSystemInfoCollector_parseSize(t *testing.T) {
	ctx := context.Background()
	collector, err := NewSystemInfoCollector(ctx, time.Second)
	require.NoError(t, err)

	tests := []struct {
		input    string
		expected uint64
		hasError bool
	}{
		{"1024", 1024, false},
		{"1K", 1024, false},
		{"1M", 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"1.5G", 1536 * 1024 * 1024, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"1X", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := collector.parseSize(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSystemInfoCollector_getSecurityTools(t *testing.T) {
	ctx := context.Background()
	collector, err := NewSystemInfoCollector(ctx, time.Second)
	require.NoError(t, err)

	tools := collector.getSecurityTools()
	assert.IsType(t, []string{}, tools)
	// Note: This will vary based on what's installed in the test environment
}
