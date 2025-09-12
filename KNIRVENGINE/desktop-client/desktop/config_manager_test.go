// config_manager_test.go
package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions
func createTestConfigManager(t *testing.T) (*ConfigManager, string) {
	tempDir, err := os.MkdirTemp("", "test_config")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	configPath := filepath.Join(tempDir, "config.json")

	// Create a config manager that uses the temp path
	manager, err := NewDesktopConfigManager(configPath)
	require.NoError(t, err)

	return manager, configPath
}

func createTestConfig() *DesktopConfig {
	return &DesktopConfig{
		AppVersion:     "1.0.0",
		FirstRun:       true,
		LastStartup:    "2024-01-01T00:00:00Z",
		AutoStart:      false,
		MinimizeToTray: true,

		WindowWidth:     1200,
		WindowHeight:    800,
		WindowX:         100,
		WindowY:         100,
		WindowMaximized: false,

		EnableTEE:                   true,
		EnableSignatureVerification: true,
		TrustedSigners:              []string{"signer1", "signer2"},
		RequireAuthentication:       true,
		SessionTimeout:              30,

		PluginAutoLoad:       true,
		PluginScanInterval:   5,
		AllowUnsignedPlugins: false,
		MaxPluginMemory:      1024,
		MaxPluginCPU:         50,

		ServerPort:             8080,
		GUIPort:                3000,
		EnableNetworkIsolation: true,
		AllowedHosts:           []string{"localhost", "127.0.0.1"},

		LogLevel:         "info",
		LogRetentionDays: 30,
		EnableDebugMode:  false,

		EnableAutoBackup:    true,
		BackupInterval:      24,
		BackupRetentionDays: 7,
		BackupLocation:      "/backup",

		MCPAutoInstall:    true,
		MCPUpdateInterval: 24,
		MCPRegistryURL:    "https://registry.mcp.com",
	}
}

// TestDesktopConfigManager_NewDesktopConfigManager tests the constructor
func TestDesktopConfigManager_NewDesktopConfigManager(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	assert.NotNil(t, manager)
}

// TestDesktopConfigManager_LoadConfig tests loading configuration
func TestDesktopConfigManager_LoadConfig(t *testing.T) {
	manager, configPath := createTestConfigManager(t)

	// Create a test config file
	testConfig := createTestConfig()
	configData, err := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// Load the config
	config, err := manager.LoadConfig()

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, testConfig.AppVersion, config.AppVersion)
	assert.Equal(t, testConfig.WindowWidth, config.WindowWidth)
	assert.Equal(t, testConfig.EnableTEE, config.EnableTEE)
	assert.Equal(t, testConfig.TrustedSigners, config.TrustedSigners)
}

// TestDesktopConfigManager_LoadConfig_NonexistentFile tests loading nonexistent config
func TestDesktopConfigManager_LoadConfig_NonexistentFile(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	// Try to load config from nonexistent file
	config, err := manager.LoadConfig()

	// Should return default config, not error
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.True(t, config.FirstRun) // Default value
}

// TestDesktopConfigManager_SaveConfig tests saving configuration
func TestDesktopConfigManager_SaveConfig(t *testing.T) {
	manager, configPath := createTestConfigManager(t)

	testConfig := createTestConfig()

	err := manager.SaveConfig(testConfig)

	assert.NoError(t, err)

	// Verify file was created and contains correct data
	assert.FileExists(t, configPath)

	// Read and verify content
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var savedConfig DesktopConfig
	err = json.Unmarshal(data, &savedConfig)
	require.NoError(t, err)

	assert.Equal(t, testConfig.AppVersion, savedConfig.AppVersion)
	assert.Equal(t, testConfig.WindowWidth, savedConfig.WindowWidth)
}

// TestDesktopConfigManager_GetDefaultConfig tests default configuration
func TestDesktopConfigManager_GetDefaultConfig(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	defaultConfig := manager.GetDefaultConfig()

	assert.NotNil(t, defaultConfig)
	assert.True(t, defaultConfig.FirstRun)
	assert.Equal(t, 1200, defaultConfig.WindowWidth)
	assert.Equal(t, 800, defaultConfig.WindowHeight)
	assert.True(t, defaultConfig.EnableTEE)
	assert.False(t, defaultConfig.AllowUnsignedPlugins)
}

// TestDesktopConfigManager_ValidateConfig tests configuration validation
func TestDesktopConfigManager_ValidateConfig(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	// Test valid config
	validConfig := createTestConfig()
	err := manager.ValidateConfig(validConfig)
	assert.NoError(t, err)

	// Test invalid config - negative window dimensions
	invalidConfig := createTestConfig()
	invalidConfig.WindowWidth = -100
	invalidConfig.WindowHeight = -50

	err = manager.ValidateConfig(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "window dimensions")

	// Test invalid config - invalid port numbers
	invalidConfig = createTestConfig()
	invalidConfig.ServerPort = -1
	invalidConfig.GUIPort = 70000

	err = manager.ValidateConfig(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port")
}

// TestDesktopConfigManager_UpdateConfig tests configuration updates
func TestDesktopConfigManager_UpdateConfig(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	// Load initial config
	_, err := manager.LoadConfig()
	require.NoError(t, err)

	// Update some values
	updates := map[string]interface{}{
		"window_width":  1600,
		"window_height": 1000,
		"theme":         "light",
		"auto_start":    true,
	}

	err = manager.UpdateConfig(updates)
	assert.NoError(t, err)

	// Reload and verify updates
	updatedConfig, err := manager.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 1600, updatedConfig.WindowWidth)
	assert.Equal(t, 1000, updatedConfig.WindowHeight)
	assert.Equal(t, "light", updatedConfig.Theme)
	assert.True(t, updatedConfig.AutoStart)
}

// TestDesktopConfigManager_ResetConfig tests configuration reset
func TestDesktopConfigManager_ResetConfig(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	// Save a custom config first
	customConfig := createTestConfig()
	customConfig.WindowWidth = 1600
	customConfig.Theme = "light"

	err := manager.SaveConfig(customConfig)
	require.NoError(t, err)

	// Reset to defaults
	err = manager.ResetConfig()
	assert.NoError(t, err)

	// Verify reset
	config, err := manager.LoadConfig()
	require.NoError(t, err)

	defaultConfig := manager.GetDefaultConfig()
	assert.Equal(t, defaultConfig.WindowWidth, config.WindowWidth)
	assert.Equal(t, defaultConfig.Theme, config.Theme)
}

// TestDesktopConfigManager_BackupConfig tests configuration backup
func TestDesktopConfigManager_BackupConfig(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	// Save a config first
	testConfig := createTestConfig()
	err := manager.SaveConfig(testConfig)
	require.NoError(t, err)

	// Create backup
	backupPath, err := manager.BackupConfig()

	assert.NoError(t, err)
	assert.NotEmpty(t, backupPath)
	assert.FileExists(t, backupPath)

	// Verify backup content
	data, err := os.ReadFile(backupPath)
	require.NoError(t, err)

	var backupConfig DesktopConfig
	err = json.Unmarshal(data, &backupConfig)
	require.NoError(t, err)

	assert.Equal(t, testConfig.AppVersion, backupConfig.AppVersion)
}

// TestDesktopConfigManager_RestoreConfig tests configuration restore
func TestDesktopConfigManager_RestoreConfig(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	// Save original config
	originalConfig := createTestConfig()
	originalConfig.WindowWidth = 1200
	err := manager.SaveConfig(originalConfig)
	require.NoError(t, err)

	// Create backup
	backupPath, err := manager.BackupConfig()
	require.NoError(t, err)

	// Modify config
	modifiedConfig := createTestConfig()
	modifiedConfig.WindowWidth = 1600
	err = manager.SaveConfig(modifiedConfig)
	require.NoError(t, err)

	// Restore from backup
	err = manager.RestoreConfig(backupPath)
	assert.NoError(t, err)

	// Verify restoration
	restoredConfig, err := manager.LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, originalConfig.WindowWidth, restoredConfig.WindowWidth)
}

// TestDesktopConfigManager_ConcurrentAccess tests thread safety
func TestDesktopConfigManager_ConcurrentAccess(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent config loading
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config, err := manager.LoadConfig()
			assert.NoError(t, err)
			assert.NotNil(t, config)
		}()
	}

	wg.Wait()
}

// TestDesktopConfigManager_InvalidJSON tests handling invalid JSON
func TestDesktopConfigManager_InvalidJSON(t *testing.T) {
	manager, configPath := createTestConfigManager(t)

	// Write invalid JSON to config file
	invalidJSON := `{"invalid": json content}`
	err := os.WriteFile(configPath, []byte(invalidJSON), 0644)
	require.NoError(t, err)

	// Try to load config
	config, err := manager.LoadConfig()

	// Should return default config when JSON is invalid
	assert.NoError(t, err) // Manager should handle gracefully
	assert.NotNil(t, config)
	assert.True(t, config.FirstRun) // Should be default
}

// TestDesktopConfigManager_PermissionErrors tests handling permission errors
func TestDesktopConfigManager_PermissionErrors(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	// Create config in read-only directory
	tempDir, err := os.MkdirTemp("", "test_readonly")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Make directory read-only
	err = os.Chmod(tempDir, 0444)
	require.NoError(t, err)
	defer os.Chmod(tempDir, 0755) // Restore permissions for cleanup

	configPath := filepath.Join(tempDir, "config.json")
	manager, _ := NewDesktopConfigManager(configPath)

	testConfig := createTestConfig()

	// Try to save config (should fail due to permissions)
	err = manager.SaveConfig(testConfig)
	assert.Error(t, err)
}

// TestDesktopConfigManager_EdgeCases tests various edge cases
func TestDesktopConfigManager_EdgeCases(t *testing.T) {
	manager, _ := createTestConfigManager(t)

	// Test with nil config
	err := manager.SaveConfig(nil)
	assert.Error(t, err)

	// Test validation with nil config
	err = manager.ValidateConfig(nil)
	assert.Error(t, err)

	// Test update with nil updates
	err = manager.UpdateConfig(nil)
	assert.Error(t, err)

	// Test update with empty updates
	err = manager.UpdateConfig(map[string]interface{}{})
	assert.NoError(t, err) // Should be no-op

	// Test restore with nonexistent backup
	err = manager.RestoreConfig("/nonexistent/backup.json")
	assert.Error(t, err)
}
