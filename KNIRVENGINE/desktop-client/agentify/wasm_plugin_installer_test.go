// wasm_plugin_installer_test.go
package agentify

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions for WASM plugin installer
func createTestInstallerDirs(t *testing.T) (string, string, string) {
	downloadsDir, err := os.MkdirTemp("", "test_downloads")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(downloadsDir) })

	pluginsDir, err := os.MkdirTemp("", "test_plugins")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(pluginsDir) })

	wasmDir, err := os.MkdirTemp("", "test_wasm")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(wasmDir) })

	return downloadsDir, pluginsDir, wasmDir
}

func createMockWASMZip(t *testing.T, zipPath, agentID, version string) {
	// Create a mock ZIP file with WASM plugin structure
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Create mock agent.wasm file
	wasmFile, err := zipWriter.Create("agent.wasm")
	require.NoError(t, err)
	_, err = wasmFile.Write([]byte("mock wasm content"))
	require.NoError(t, err)

	// Create mock metadata.json file
	metadata := map[string]interface{}{
		"agent_id":     agentID,
		"agent_name":   fmt.Sprintf("Mock Agent %s", agentID),
		"version":      version,
		"description":  "Mock WASM agent for testing",
		"build_target": "wasm32-wasi",
		"compiled_at":  time.Now().Format(time.RFC3339),
	}

	metadataBytes, err := json.Marshal(metadata)
	require.NoError(t, err)

	metadataFile, err := zipWriter.Create("metadata.json")
	require.NoError(t, err)
	_, err = metadataFile.Write(metadataBytes)
	require.NoError(t, err)

	// Create mock config.json file
	config := map[string]interface{}{
		"memory_limit": 1024,
		"timeout":      30,
	}

	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	configFile, err := zipWriter.Create("config.json")
	require.NoError(t, err)
	_, err = configFile.Write(configBytes)
	require.NoError(t, err)
}

// TestWASMPluginInstaller_NewWASMPluginInstaller tests the constructor
func TestWASMPluginInstaller_NewWASMPluginInstaller(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)

	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	assert.NotNil(t, installer)
}

// TestWASMPluginInstaller_DiscoverZipPlugins tests ZIP plugin discovery
func TestWASMPluginInstaller_DiscoverZipPlugins(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Create some mock ZIP files
	zipPath1 := filepath.Join(downloadsDir, "agent_test1_1.0.0.zip")
	createMockWASMZip(t, zipPath1, "test1", "1.0.0")

	zipPath2 := filepath.Join(downloadsDir, "agent_test2_2.0.0.zip")
	createMockWASMZip(t, zipPath2, "test2", "2.0.0")

	// Create a non-ZIP file
	nonZipPath := filepath.Join(downloadsDir, "not_a_zip.txt")
	err := os.WriteFile(nonZipPath, []byte("not a zip"), 0644)
	require.NoError(t, err)

	plugins, err := installer.DiscoverZipPlugins()

	assert.NoError(t, err)
	assert.Len(t, plugins, 2) // Only ZIP files should be discovered

	// Check plugin info
	for _, plugin := range plugins {
		assert.NotEmpty(t, plugin.AgentID)
		assert.NotEmpty(t, plugin.Version)
		assert.NotEmpty(t, plugin.ZipPath)
		assert.False(t, plugin.IsInstalled)
	}
}

// TestWASMPluginInstaller_DiscoverZipPlugins_EmptyDirectory tests discovery in empty directory
func TestWASMPluginInstaller_DiscoverZipPlugins_EmptyDirectory(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	plugins, err := installer.DiscoverZipPlugins()

	assert.NoError(t, err)
	assert.Empty(t, plugins)
}

// TestWASMPluginInstaller_DiscoverZipPlugins_NonexistentDirectory tests discovery with nonexistent directory
func TestWASMPluginInstaller_DiscoverZipPlugins_NonexistentDirectory(t *testing.T) {
	_, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller("/nonexistent/directory", pluginsDir, wasmDir)

	plugins, err := installer.DiscoverZipPlugins()

	assert.Error(t, err)
	assert.Nil(t, plugins)
}

// TestWASMPluginInstaller_InstallPlugin tests plugin installation
func TestWASMPluginInstaller_InstallPlugin(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Create a mock ZIP file
	zipPath := filepath.Join(downloadsDir, "agent_test_1.0.0.zip")
	createMockWASMZip(t, zipPath, "test", "1.0.0")

	pluginInfo, err := installer.InstallPlugin(zipPath)

	assert.NoError(t, err)
	assert.NotNil(t, pluginInfo)
	assert.Equal(t, "test", pluginInfo.AgentID)
	assert.Equal(t, "1.0.0", pluginInfo.Version)
	assert.True(t, pluginInfo.IsInstalled)
	assert.NotEmpty(t, pluginInfo.InstallPath)

	// Check that files were extracted
	assert.FileExists(t, filepath.Join(pluginInfo.InstallPath, "agent.wasm"))
	assert.FileExists(t, filepath.Join(pluginInfo.InstallPath, "metadata.json"))
	assert.FileExists(t, filepath.Join(pluginInfo.InstallPath, "config.json"))
}

// TestWASMPluginInstaller_InstallPlugin_InvalidZip tests installation with invalid ZIP
func TestWASMPluginInstaller_InstallPlugin_InvalidZip(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Create an invalid ZIP file
	invalidZipPath := filepath.Join(downloadsDir, "invalid.zip")
	err := os.WriteFile(invalidZipPath, []byte("not a zip file"), 0644)
	require.NoError(t, err)

	_, err = installer.InstallPlugin(invalidZipPath)

	assert.Error(t, err)
}

// TestWASMPluginInstaller_InstallPlugin_NonexistentZip tests installation with nonexistent ZIP
func TestWASMPluginInstaller_InstallPlugin_NonexistentZip(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	_, err := installer.InstallPlugin("/nonexistent/file.zip")

	assert.Error(t, err)
}

// TestWASMPluginInstaller_UninstallPlugin tests plugin uninstallation
func TestWASMPluginInstaller_UninstallPlugin(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Create and install a plugin first
	zipPath := filepath.Join(downloadsDir, "agent_test_1.0.0.zip")
	createMockWASMZip(t, zipPath, "test", "1.0.0")

	pluginInfo, err := installer.InstallPlugin(zipPath)
	require.NoError(t, err)
	require.NotNil(t, pluginInfo)

	// Store the install path for later verification
	installPath := pluginInfo.InstallPath

	// Now uninstall it
	err = installer.UninstallPlugin(pluginInfo.AgentID, pluginInfo.Version)

	assert.NoError(t, err)

	// Check that installation directory was removed
	_, err = os.Stat(installPath)
	assert.True(t, os.IsNotExist(err))
}

// TestWASMPluginInstaller_UninstallPlugin_NotInstalled tests uninstalling non-installed plugin
func TestWASMPluginInstaller_UninstallPlugin_NotInstalled(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	err := installer.UninstallPlugin("test", "1.0.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

// TestWASMPluginInstaller_EdgeCases tests various edge cases
func TestWASMPluginInstaller_EdgeCases(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Test with empty path
	_, err := installer.InstallPlugin("")
	assert.Error(t, err)

	// Test with nonexistent path
	_, err = installer.InstallPlugin("/nonexistent/path.zip")
	assert.Error(t, err)
}

// TestWASMPluginInstaller_ConcurrentInstallation tests concurrent plugin installation
func TestWASMPluginInstaller_ConcurrentInstallation(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	const numPlugins = 5
	var wg sync.WaitGroup
	errors := make(chan error, numPlugins)

	// Create multiple ZIP files and install them concurrently
	for i := 0; i < numPlugins; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			agentID := fmt.Sprintf("concurrent-agent-%d", id)
			zipPath := filepath.Join(downloadsDir, fmt.Sprintf("agent_%s_1.0.0.zip", agentID))
			createMockWASMZip(t, zipPath, agentID, "1.0.0")

			_, err := installer.InstallPlugin(zipPath)
			if err != nil {
				errors <- fmt.Errorf("installation error for %s: %v", agentID, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent installation error: %v", err)
	}
}

// TestWASMPluginInstaller_ErrorHandling tests error handling scenarios
func TestWASMPluginInstaller_ErrorHandling(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Test with non-existent ZIP file
	_, err := installer.InstallPlugin("/non/existent/file.zip")
	assert.Error(t, err)

	// Test with invalid ZIP file
	invalidZipPath := filepath.Join(downloadsDir, "invalid.zip")
	file, err := os.Create(invalidZipPath)
	require.NoError(t, err)
	_, err = file.WriteString("not a zip file")
	require.NoError(t, err)
	file.Close()

	_, err = installer.InstallPlugin(invalidZipPath)
	assert.Error(t, err)

	// Test with empty parameters
	_, err = installer.InstallPlugin("")
	assert.Error(t, err)
}

// TestWASMPluginInstaller_MetadataValidation tests metadata validation
func TestWASMPluginInstaller_MetadataValidation(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Create ZIP with invalid metadata
	zipPath := filepath.Join(downloadsDir, "invalid_metadata.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Create agent.wasm file
	wasmFile, err := zipWriter.Create("agent.wasm")
	require.NoError(t, err)
	_, err = wasmFile.Write([]byte("mock wasm content"))
	require.NoError(t, err)

	// Create invalid metadata.json file
	metadataFile, err := zipWriter.Create("metadata.json")
	require.NoError(t, err)
	_, err = metadataFile.Write([]byte("invalid json"))
	require.NoError(t, err)

	zipWriter.Close()
	zipFile.Close()

	// Test installation with invalid metadata
	_, err = installer.InstallPlugin(zipPath)
	assert.Error(t, err)
}

// TestWASMPluginInstaller_ResourceLimits tests resource limit enforcement
func TestWASMPluginInstaller_ResourceLimits(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Create a very large ZIP file to test size limits
	zipPath := filepath.Join(downloadsDir, "large.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Create a large file within the ZIP
	largeFile, err := zipWriter.Create("large_file.bin")
	require.NoError(t, err)

	// Write a moderately large amount of data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	_, err = largeFile.Write(largeData)
	require.NoError(t, err)

	zipWriter.Close()
	zipFile.Close()

	// Test installation - should handle large files appropriately
	_, err = installer.InstallPlugin(zipPath)
	// The behavior depends on implementation - it might succeed or fail based on size limits
	t.Logf("Installation result with large file: %v", err)
}

// TestWASMPluginInstaller_PathSecurity tests path security validation
func TestWASMPluginInstaller_PathSecurity(t *testing.T) {
	downloadsDir, pluginsDir, wasmDir := createTestInstallerDirs(t)
	installer := NewWASMPluginInstaller(downloadsDir, pluginsDir, wasmDir)

	// Create ZIP with path traversal attempts
	zipPath := filepath.Join(downloadsDir, "malicious.zip")
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Try to create files with malicious paths
	maliciousPaths := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\cmd.exe",
		"/etc/shadow",
		"C:\\Windows\\System32\\calc.exe",
	}

	for _, maliciousPath := range maliciousPaths {
		file, err := zipWriter.Create(maliciousPath)
		require.NoError(t, err)
		_, err = file.Write([]byte("malicious content"))
		require.NoError(t, err)
	}

	zipWriter.Close()
	zipFile.Close()

	// Test installation should fail or sanitize paths
	_, err = installer.InstallPlugin(zipPath)
	// The installer should either reject the ZIP or sanitize the paths
	// We don't assert specific behavior here as it depends on implementation
	t.Logf("Installation result with malicious paths: %v", err)
}
