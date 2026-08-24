package utils

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlatformInfo(t *testing.T) {
	t.Run("returns valid platform info", func(t *testing.T) {
		info := GetPlatformInfo()
		
		assert.NotEmpty(t, info.Platform)
		assert.NotEmpty(t, info.Architecture)
		assert.NotEmpty(t, info.OS)
		assert.NotEmpty(t, info.Version)
		
		// Verify platform matches runtime
		assert.Equal(t, runtime.GOOS, info.Platform)
		assert.Equal(t, runtime.GOARCH, info.Architecture)
		assert.Equal(t, runtime.GOOS, info.OS)
		assert.Equal(t, runtime.Version(), info.Version)
	})
}

func TestIsProcessRunning(t *testing.T) {
	t.Run("unsupported platform", func(t *testing.T) {
		// This test is tricky since we can't easily mock runtime.GOOS
		// We'll test the actual platform-specific functions instead
		switch runtime.GOOS {
		case "windows", "darwin", "linux":
			// These are supported platforms, so we test with a non-existent process
			running, err := IsProcessRunning("non-existent-process-12345")
			require.NoError(t, err)
			assert.False(t, running)
		default:
			// For unsupported platforms, it should return an error
			_, err := IsProcessRunning("any-process")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported platform")
		}
	})

	t.Run("non-existent process", func(t *testing.T) {
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			running, err := IsProcessRunning("non-existent-process-12345")
			require.NoError(t, err)
			assert.False(t, running)
		}
	})

	// Note: Testing for existing processes is platform-dependent and may be flaky
	// We focus on testing the error cases and non-existent processes
}

func TestIsApplicationInstalled(t *testing.T) {
	t.Run("unsupported platform", func(t *testing.T) {
		switch runtime.GOOS {
		case "windows", "darwin", "linux":
			// These are supported platforms, test with non-existent app
			installed, err := IsApplicationInstalled("non-existent-app-12345")
			require.NoError(t, err)
			assert.False(t, installed)
		default:
			// For unsupported platforms, it should return an error
			_, err := IsApplicationInstalled("any-app")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported platform")
		}
	})

	t.Run("non-existent application", func(t *testing.T) {
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			installed, err := IsApplicationInstalled("non-existent-app-12345")
			require.NoError(t, err)
			assert.False(t, installed)
		}
	})

	// Test with common applications that are likely to be available
	t.Run("common applications", func(t *testing.T) {
		switch runtime.GOOS {
		case "linux":
			// Test with 'ls' which should be available on most Linux systems
			installed, err := IsApplicationInstalled("ls")
			require.NoError(t, err)
			assert.True(t, installed)
		case "darwin":
			// Test with 'ls' which should be available on macOS
			installed, err := IsApplicationInstalled("ls")
			require.NoError(t, err)
			assert.True(t, installed)
		case "windows":
			// Test with 'cmd' which should be available on Windows
			installed, err := IsApplicationInstalled("cmd")
			require.NoError(t, err)
			assert.True(t, installed)
		}
	})
}

func TestIsServiceRunning(t *testing.T) {
	t.Run("unsupported platform", func(t *testing.T) {
		switch runtime.GOOS {
		case "windows", "darwin", "linux":
			// These are supported platforms, test with non-existent service
			running, err := IsServiceRunning("non-existent-service-12345")
			// Note: This may return an error or false depending on the platform
			// We just ensure it doesn't panic
			_ = running
			_ = err
		default:
			// For unsupported platforms, it should return an error
			_, err := IsServiceRunning("any-service")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported platform")
		}
	})

	t.Run("non-existent service", func(t *testing.T) {
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			running, err := IsServiceRunning("non-existent-service-12345")
			// The behavior may vary by platform for non-existent services
			// We just ensure the function doesn't panic
			_ = running
			_ = err
		}
	})
}

func TestIsDatabaseRunning(t *testing.T) {
	tests := []struct {
		name   string
		dbType string
		valid  bool
	}{
		{
			name:   "mysql",
			dbType: "mysql",
			valid:  true,
		},
		{
			name:   "MySQL uppercase",
			dbType: "MySQL",
			valid:  true,
		},
		{
			name:   "postgresql",
			dbType: "postgresql",
			valid:  true,
		},
		{
			name:   "postgres",
			dbType: "postgres",
			valid:  true,
		},
		{
			name:   "PostgreSQL uppercase",
			dbType: "PostgreSQL",
			valid:  true,
		},
		{
			name:   "mongodb",
			dbType: "mongodb",
			valid:  true,
		},
		{
			name:   "mongo",
			dbType: "mongo",
			valid:  true,
		},
		{
			name:   "MongoDB uppercase",
			dbType: "MongoDB",
			valid:  true,
		},
		{
			name:   "redis",
			dbType: "redis",
			valid:  true,
		},
		{
			name:   "Redis uppercase",
			dbType: "Redis",
			valid:  true,
		},
		{
			name:   "unsupported database",
			dbType: "unsupported-db",
			valid:  false,
		},
		{
			name:   "empty string",
			dbType: "",
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			running, err := IsDatabaseRunning(tt.dbType)
			if tt.valid {
				// For valid database types, the function should not return an error
				// about unsupported database type (though it may return other errors)
				if err != nil {
					assert.NotContains(t, err.Error(), "unsupported database type")
				}
				_ = running // We don't assert on the running status as it depends on system state
			} else {
				// For invalid database types, it should return an error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported database type")
			}
		})
	}
}

// Test platform-specific functions with mocked behavior would require more complex setup
// For now, we focus on testing the public API and error conditions
func TestSystemInfoStruct(t *testing.T) {
	t.Run("SystemInfo struct fields", func(t *testing.T) {
		info := SystemInfo{
			Platform:     "test-platform",
			Architecture: "test-arch",
			OS:           "test-os",
			Version:      "test-version",
		}
		
		assert.Equal(t, "test-platform", info.Platform)
		assert.Equal(t, "test-arch", info.Architecture)
		assert.Equal(t, "test-os", info.OS)
		assert.Equal(t, "test-version", info.Version)
	})
}

func TestProcessInfoStruct(t *testing.T) {
	t.Run("ProcessInfo struct fields", func(t *testing.T) {
		info := ProcessInfo{
			PID:     1234,
			Name:    "test-process",
			Command: "test-command",
		}
		
		assert.Equal(t, 1234, info.PID)
		assert.Equal(t, "test-process", info.Name)
		assert.Equal(t, "test-command", info.Command)
	})
}

func TestNetworkInterfaceStruct(t *testing.T) {
	t.Run("NetworkInterface struct fields", func(t *testing.T) {
		iface := NetworkInterface{
			Name:      "eth0",
			Address:   "192.168.1.1",
			Type:      "ethernet",
			Status:    "up",
			Available: true,
		}
		
		assert.Equal(t, "eth0", iface.Name)
		assert.Equal(t, "192.168.1.1", iface.Address)
		assert.Equal(t, "ethernet", iface.Type)
		assert.Equal(t, "up", iface.Status)
		assert.True(t, iface.Available)
	})
}

func TestMountedDriveStruct(t *testing.T) {
	t.Run("MountedDrive struct fields", func(t *testing.T) {
		drive := MountedDrive{
			Name:       "C:",
			Path:       "/mnt/c",
			Type:       "ntfs",
			Size:       "500GB",
			Available:  "100GB",
			Accessible: true,
		}
		
		assert.Equal(t, "C:", drive.Name)
		assert.Equal(t, "/mnt/c", drive.Path)
		assert.Equal(t, "ntfs", drive.Type)
		assert.Equal(t, "500GB", drive.Size)
		assert.Equal(t, "100GB", drive.Available)
		assert.True(t, drive.Accessible)
	})
}
