package dns

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	dataengine "backend-server/internal/data-engine"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/buntdb"
)

func setupTestDNSService(t *testing.T) (*DynamicDNSService, *buntdb.DB) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_dns.db")

	// Create data engine config
	dataEngineConfig := dataengine.BuntDBDataEngineConfig{
		DatabasePath: dbPath,
	}

	// Create data engine
	dataEngine, err := dataengine.NewBuntDBDataEngine(dataEngineConfig)
	require.NoError(t, err)

	// Create DNS config
	config := DNSConfig{
		CloudFlareAPIToken:  "test-token",
		ZoneName:            "test.com",
		UpdateInterval:      30 * time.Second,
		ForceUpdateInterval: 5 * time.Minute,
		Records: []DNSRecordConfig{
			{
				Name:     "api.test.com",
				Type:     "A",
				Proxied:  false,
				Priority: 0,
			},
		},
	}

	service, err := NewDynamicDNSService(dataEngine, config)
	require.NoError(t, err)

	t.Cleanup(func() {
		if service.isRunning {
			service.Stop()
		}
		os.Remove(dbPath)
	})

	return service, nil
}

func TestNewDynamicDNSService(t *testing.T) {
	service, _ := setupTestDNSService(t)

	assert.NotNil(t, service)
	assert.NotNil(t, service.dataEngine)
	assert.False(t, service.isRunning)
	assert.Equal(t, "test.com", service.config.ZoneName)
	assert.Equal(t, 30*time.Second, service.config.UpdateInterval)
	assert.Len(t, service.config.Records, 1)
}

func TestDynamicDNSService_Start(t *testing.T) {
	service, _ := setupTestDNSService(t)

	err := service.Start()
	assert.NoError(t, err)
	assert.True(t, service.isRunning)

	// Test starting already running service
	err = service.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	err = service.Stop()
	assert.NoError(t, err)
}

func TestDynamicDNSService_Stop(t *testing.T) {
	service, _ := setupTestDNSService(t)

	// Test stopping non-running service (should be no-op, no error)
	err := service.Stop()
	assert.NoError(t, err)

	// Start and then stop
	err = service.Start()
	require.NoError(t, err)

	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.isRunning)
}

func TestDynamicDNSService_GetStatus(t *testing.T) {
	service, _ := setupTestDNSService(t)

	status := service.GetStatus()
	assert.NotNil(t, status)
	assert.False(t, status["running"].(bool))

	// Start service and check status
	err := service.Start()
	require.NoError(t, err)
	defer service.Stop()

	status = service.GetStatus()
	assert.True(t, status["running"].(bool))
}
