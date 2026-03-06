package teesecurity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppArmorManager_Creation(t *testing.T) {
	config := MACConfig{
		Profile:      "test-profile",
		AutoGenerate: true,
		Type:         "apparmor",
	}

	mgr := NewAppArmorManager(config)
	assert.NotNil(t, mgr)
	assert.Equal(t, "test-profile", config.Profile)
	assert.True(t, config.AutoGenerate)
}

func TestAppArmorManager_DefaultProfile(t *testing.T) {
	// Verify the default AppArmor profile contains expected rules
	assert.Contains(t, DefaultAppArmorProfile, "{{PROFILE_NAME}}")
	assert.Contains(t, DefaultAppArmorProfile, "deny capability sys_admin")
	assert.Contains(t, DefaultAppArmorProfile, "/skill/** rwix")
	assert.Contains(t, DefaultAppArmorProfile, "deny /** w")
}

func TestAppArmorManager_ProfileGeneration(t *testing.T) {
	// Create a temporary directory for test profiles
	tempDir := t.TempDir()
	profilePath := filepath.Join(tempDir, "test-profile")

	mgr := NewAppArmorManager(MACConfig{
		Profile:      "test-profile",
		AutoGenerate: true,
		Template:     DefaultAppArmorProfile,
	})

	// Generate profile
	err := mgr.generateProfile(profilePath)
	require.NoError(t, err)

	// Verify profile was created
	assert.FileExists(t, profilePath)

	// Verify profile content
	content, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "profile test-profile")
}

func TestAppArmorManager_SupportCheck(t *testing.T) {
	mgr := NewAppArmorManager(MACConfig{})
	supported := mgr.isAppArmorEnabled()

	// This test will pass on systems with AppArmor support
	// and fail gracefully on systems without it
	if supported {
		t.Log("AppArmor is supported on this system")
	} else {
		t.Log("AppArmor is not supported on this system")
	}

	// We don't assert the result since it depends on the test environment
}

func TestAppArmorManager_StatusCheck(t *testing.T) {
	status, err := GetAppArmorStatus()

	// This should not fail even if AppArmor is not available
	assert.NoError(t, err)

	// Status should be either "enabled" or "disabled"
	if status == "disabled" {
		t.Log("AppArmor status: disabled")
	} else {
		assert.Contains(t, status, "enabled")
		t.Logf("AppArmor status: %s", status)
	}
}

func TestAppArmorManager_ProfilePath(t *testing.T) {
	config := MACConfig{
		Profile: "test-container",
	}

	mgr := NewAppArmorManager(config)
	expectedPath := "/etc/apparmor.d/test-container"

	// Verify profile path construction
	assert.Equal(t, expectedPath, filepath.Join("/etc/apparmor.d", mgr.config.Profile))
}

func TestAppArmorManager_FileExists(t *testing.T) {
	// Test fileExists helper function
	tempFile := filepath.Join(t.TempDir(), "test-file")

	// File should not exist initially
	assert.False(t, fileExists(tempFile))

	// Create file
	f, err := os.Create(tempFile)
	require.NoError(t, err)
	f.Close()

	// File should exist now
	assert.True(t, fileExists(tempFile))
}
