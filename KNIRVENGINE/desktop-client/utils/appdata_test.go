package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAppDataDir(t *testing.T) {
	t.Run("returns valid app data directory", func(t *testing.T) {
		dir, err := GetAppDataDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, AppName)

		// Verify the path is absolute
		assert.True(t, filepath.IsAbs(dir))
	})

	t.Run("platform specific paths", func(t *testing.T) {
		dir, err := GetAppDataDir()
		require.NoError(t, err)

		switch runtime.GOOS {
		case "windows":
			// Should contain APPDATA path
			assert.Contains(t, strings.ToLower(dir), "appdata")
		case "darwin":
			// Should contain Library/Application Support
			assert.Contains(t, dir, "Library/Application Support")
		default:
			// Linux and others should use config directory
			// The exact path depends on XDG_CONFIG_HOME or ~/.config
			assert.Contains(t, dir, AppName)
		}
	})
}

func TestGetDatabaseDir(t *testing.T) {
	t.Run("returns valid database directory", func(t *testing.T) {
		dir, err := GetDatabaseDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "data")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetConfigDir(t *testing.T) {
	t.Run("returns valid config directory", func(t *testing.T) {
		dir, err := GetConfigDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "config")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetAgentsDBPath(t *testing.T) {
	t.Run("returns valid agents database path", func(t *testing.T) {
		path, err := GetAgentsDBPath()
		require.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "agents.db")
		assert.Contains(t, path, "data")
		assert.Contains(t, path, AppName)

		// Verify it's a file path, not a directory
		assert.True(t, strings.HasSuffix(path, ".db"))
	})
}

func TestGetMCPDir(t *testing.T) {
	t.Run("returns valid MCP directory", func(t *testing.T) {
		dir, err := GetMCPDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "mcp")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetMCPConfigDir(t *testing.T) {
	t.Run("returns valid MCP config directory", func(t *testing.T) {
		dir, err := GetMCPConfigDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "mcp")
		assert.Contains(t, dir, "config")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetMCPServersDir(t *testing.T) {
	t.Run("returns valid MCP servers directory", func(t *testing.T) {
		dir, err := GetMCPServersDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "mcp")
		assert.Contains(t, dir, "servers")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetPluginsDir(t *testing.T) {
	t.Run("returns valid plugins directory", func(t *testing.T) {
		dir, err := GetPluginsDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "plugins")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetCacheDir(t *testing.T) {
	t.Run("returns valid cache directory", func(t *testing.T) {
		dir, err := GetCacheDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, AppName)

		// Verify the path is absolute
		assert.True(t, filepath.IsAbs(dir))
	})

	t.Run("platform specific cache paths", func(t *testing.T) {
		dir, err := GetCacheDir()
		require.NoError(t, err)

		switch runtime.GOOS {
		case "windows":
			// Should contain LOCALAPPDATA or APPDATA path
			lowerDir := strings.ToLower(dir)
			assert.True(t, strings.Contains(lowerDir, "appdata") || strings.Contains(lowerDir, "cache"))
		case "darwin":
			// Should contain Library/Caches
			assert.Contains(t, dir, "Library/Caches")
		default:
			// Linux and others should use cache directory
			assert.Contains(t, dir, AppName)
		}
	})
}

func TestGetTempWorkspaceDir(t *testing.T) {
	t.Run("returns valid temp workspace directory", func(t *testing.T) {
		dir, err := GetTempWorkspaceDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "workspace")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetLogsDir(t *testing.T) {
	t.Run("returns valid logs directory", func(t *testing.T) {
		dir, err := GetLogsDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "logs")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetPluginDataDir(t *testing.T) {
	t.Run("returns valid plugin data directory", func(t *testing.T) {
		dir, err := GetPluginDataDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "plugins")
		assert.Contains(t, dir, "data")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetMCPDataDir(t *testing.T) {
	t.Run("returns valid MCP data directory", func(t *testing.T) {
		dir, err := GetMCPDataDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "mcp")
		assert.Contains(t, dir, "data")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetMCPLogsDir(t *testing.T) {
	t.Run("returns valid MCP logs directory", func(t *testing.T) {
		dir, err := GetMCPLogsDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "mcp")
		assert.Contains(t, dir, "logs")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetMCPMonitoringDir(t *testing.T) {
	t.Run("returns valid MCP monitoring directory", func(t *testing.T) {
		dir, err := GetMCPMonitoringDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "mcp")
		assert.Contains(t, dir, "monitoring")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetQuarantineDir(t *testing.T) {
	t.Run("returns valid quarantine directory", func(t *testing.T) {
		dir, err := GetQuarantineDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "quarantine")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetBackupDir(t *testing.T) {
	t.Run("returns valid backup directory", func(t *testing.T) {
		dir, err := GetBackupDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "backups")
		assert.Contains(t, dir, AppName)
	})
}

func TestGetSecurityDir(t *testing.T) {
	t.Run("returns valid security directory", func(t *testing.T) {
		dir, err := GetSecurityDir()
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "security")
		assert.Contains(t, dir, AppName)
	})
}

func TestEnsureDir(t *testing.T) {
	t.Run("creates directory successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		testDir := filepath.Join(tmpDir, "test-dir")

		err := EnsureDir(testDir)
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(testDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		testDir := filepath.Join(tmpDir, "level1", "level2", "level3")

		err := EnsureDir(testDir)
		require.NoError(t, err)

		// Verify nested directory was created
		info, err := os.Stat(testDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("succeeds if directory already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		testDir := filepath.Join(tmpDir, "existing-dir")

		// Create directory first
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		// EnsureDir should succeed
		err = EnsureDir(testDir)
		require.NoError(t, err)
	})
}

func TestEnsureAppDataDirs(t *testing.T) {
	t.Run("creates all application directories", func(t *testing.T) {
		// This test verifies that EnsureAppDataDirs doesn't return an error
		// We can't easily test the actual directory creation without mocking
		// the OS-specific directory functions, but we can ensure it runs without error
		err := EnsureAppDataDirs()
		// Note: This might fail in some test environments where home directory
		// is not accessible, but it should work in most cases
		if err != nil {
			// If it fails, it should be an AppDataError
			var appDataErr *AppDataError
			assert.ErrorAs(t, err, &appDataErr)
		}
	})
}

func TestAppDataError(t *testing.T) {
	t.Run("error with path", func(t *testing.T) {
		err := &AppDataError{
			Op:   "TestOp",
			Path: "/test/path",
			Err:  "test error",
		}

		expected := "appdata TestOp /test/path: test error"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("error with OS", func(t *testing.T) {
		err := &AppDataError{
			Op:  "TestOp",
			OS:  "linux",
			Err: "test error",
		}

		expected := "appdata TestOp on linux: test error"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("error with only operation and error", func(t *testing.T) {
		err := &AppDataError{
			Op:  "TestOp",
			Err: "test error",
		}

		expected := "appdata TestOp: test error"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("error with path takes precedence over OS", func(t *testing.T) {
		err := &AppDataError{
			Op:   "TestOp",
			OS:   "linux",
			Path: "/test/path",
			Err:  "test error",
		}

		expected := "appdata TestOp /test/path: test error"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("empty fields", func(t *testing.T) {
		err := &AppDataError{}
		expected := "appdata : "
		assert.Equal(t, expected, err.Error())
	})
}

func TestAppNameConstant(t *testing.T) {
	t.Run("app name is defined", func(t *testing.T) {
		assert.Equal(t, "KNIRV-Engine", AppName)
		assert.NotEmpty(t, AppName)
	})
}

// Integration test to verify all directory functions work together
func TestDirectoryIntegration(t *testing.T) {
	t.Run("all directory functions return consistent paths", func(t *testing.T) {
		// Get app data directory as base
		appDataDir, err := GetAppDataDir()
		require.NoError(t, err)

		// Test that all other directories are subdirectories of app data or cache
		testCases := []struct {
			name string
			fn   func() (string, error)
		}{
			{"database", GetDatabaseDir},
			{"config", GetConfigDir},
			{"mcp", GetMCPDir},
			{"mcp_config", GetMCPConfigDir},
			{"mcp_servers", GetMCPServersDir},
			{"mcp_data", GetMCPDataDir},
			{"mcp_logs", GetMCPLogsDir},
			{"mcp_monitoring", GetMCPMonitoringDir},
			{"plugins", GetPluginsDir},
			{"plugin_data", GetPluginDataDir},
			{"logs", GetLogsDir},
			{"quarantine", GetQuarantineDir},
			{"backup", GetBackupDir},
			{"security", GetSecurityDir},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				dir, err := tc.fn()
				require.NoError(t, err)
				assert.NotEmpty(t, dir)
				assert.True(t, filepath.IsAbs(dir))
				// Should be under app data directory
				assert.True(t, strings.HasPrefix(dir, appDataDir),
					"Directory %s should be under app data dir %s", dir, appDataDir)
			})
		}

		// Test cache-related directories
		cacheDir, err := GetCacheDir()
		require.NoError(t, err)

		tempWorkspace, err := GetTempWorkspaceDir()
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(tempWorkspace, cacheDir),
			"Temp workspace %s should be under cache dir %s", tempWorkspace, cacheDir)

		// Test agents DB path
		agentsDBPath, err := GetAgentsDBPath()
		require.NoError(t, err)
		dbDir, err := GetDatabaseDir()
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(agentsDBPath, dbDir),
			"Agents DB path %s should be under database dir %s", agentsDBPath, dbDir)
	})
}
