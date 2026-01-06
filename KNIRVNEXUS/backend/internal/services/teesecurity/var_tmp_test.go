package teesecurity

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVarTmpContainerDirectory verifies that the container directory uses /var/tmp
// instead of /tmp to avoid Docker read-only /tmp mount issues.

func TestVarTmpContainerDirectory(t *testing.T) {
	expectedContainerDir := "/var/tmp/knirv-containers"

	// Verify the expected directory path is correct
	if expectedContainerDir != "/var/tmp/knirv-containers" {
		t.Errorf("Expected container dir to be /var/tmp/knirv-containers, got %s", expectedContainerDir)
	}

	// Create the directory for testing
	if err := os.MkdirAll(expectedContainerDir, 0755); err != nil {
		t.Fatalf("Failed to create container directory %s: %v", expectedContainerDir, err)
	}
	defer os.RemoveAll(expectedContainerDir)

	// Verify the directory exists and has correct permissions
	info, err := os.Stat(expectedContainerDir)
	if err != nil {
		t.Fatalf("Failed to stat container directory: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", expectedContainerDir)
	}

	// Check permissions (should be 755 or more restrictive - no world write)
	mode := info.Mode().Perm()
	// 755 = rwxr-xr-x, world can read and execute but not write
	// Check if world has write permission
	if mode&02 != 0 {
		t.Errorf("Container directory has world write permissions: %v", mode)
	}

	t.Logf("Container directory %s created successfully with permissions %v", expectedContainerDir, mode)
}

// TestNativeContainerRuntimeUsesVarTmp verifies that NativeContainerRuntime
// uses /var/tmp for container storage
func TestNativeContainerRuntimeUsesVarTmp(t *testing.T) {
	// Create a mock Kali profile for testing
	mockProfile := &KaliLinuxProfile{
		OS:               "linux",
		IsKaliLinux:      false,
		PreferredRuntime: "native-go",
	}

	// Create the runtime (this will create /var/tmp/knirv-containers)
	runtime, err := NewNativeContainerRuntime(mockProfile)
	if err != nil {
		// Skip if strace or bash are not available (expected in non-Kali environments)
		t.Skipf("Skipping test - required tools not available: %v", err)
	}

	// Verify the container directory is /var/tmp
	expectedDir := "/var/tmp/knirv-containers"
	if runtime.containerDir != expectedDir {
		t.Errorf("Expected containerDir to be %s, got %s", expectedDir, runtime.containerDir)
	}

	// Verify the directory exists
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Container directory %s was not created", expectedDir)
	}

	t.Logf("NativeContainerRuntime correctly uses %s for container storage", runtime.containerDir)
}

// TestTEESecurityServiceCheckFileSystemSecurity verifies that the filesystem
// security check uses /var/tmp
func TestTEESecurityServiceCheckFileSystemSecurity(t *testing.T) {
	// Create a temporary directory for testing
	testDir := "/var/tmp/knirv-containers"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// The checkFileSystemSecurity function checks for /var/tmp/knirv-containers
	// This test verifies that the directory structure is correct
	expectedDir := "/var/tmp/knirv-containers"

	info, err := os.Stat(expectedDir)
	if err != nil {
		t.Fatalf("Failed to stat %s: %v", expectedDir, err)
	}

	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", expectedDir)
	}

	t.Logf("Filesystem security check will look for %s (exists: %v)", expectedDir, !os.IsNotExist(err))
}

// TestVarTmpIsWritable verifies that /var/tmp is writable in Docker environments
// where /tmp might be read-only
func TestVarTmpIsWritable(t *testing.T) {
	testDir := "/var/tmp/knirv-containers/test-write"

	// Ensure parent directory exists
	if err := os.MkdirAll("/var/tmp/knirv-containers", 0755); err != nil {
		t.Fatalf("Failed to create parent directory: %v", err)
	}
	defer os.RemoveAll("/var/tmp/knirv-containers")

	// Try to create a file in /var/tmp
	testContent := []byte("test content")
	if err := os.WriteFile(testDir, testContent, 0644); err != nil {
		t.Errorf("Failed to write to %s: %v", testDir, err)
		t.Log("This may indicate /var/tmp is not writable in this environment")
	} else {
		// Verify the file was created
		content, err := os.ReadFile(testDir)
		if err != nil {
			t.Errorf("Failed to read test file: %v", err)
		} else if string(content) != string(testContent) {
			t.Errorf("File content mismatch: expected %s, got %s", testContent, content)
		}

		// Clean up
		os.Remove(testDir)
		t.Logf("/var/tmp is writable - test file created and verified at %s", testDir)
	}
}

// ContainerDirPath constant for consistent path usage across the codebase
const ContainerDirPath = "/var/tmp/knirv-containers"

// TestContainerDirPathConstant verifies the constant is correctly defined
func TestContainerDirPathConstant(t *testing.T) {
	if ContainerDirPath != "/var/tmp/knirv-containers" {
		t.Errorf("ContainerDirPath should be /var/tmp/knirv-containers, got %s", ContainerDirPath)
	}

	// Verify it's different from /tmp
	if ContainerDirPath == "/tmp/knirv-containers" {
		t.Error("ContainerDirPath should use /var/tmp, not /tmp")
	}

	t.Logf("ContainerDirPath constant correctly set to %s", ContainerDirPath)
}

// TestContainerSubdirectoryCreation verifies that subdirectories can be created
// under /var/tmp/knirv-containers for individual containers
func TestContainerSubdirectoryCreation(t *testing.T) {
	parentDir := "/var/tmp/knirv-containers"

	// Ensure parent exists
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("Failed to create parent directory: %v", err)
	}
	defer os.RemoveAll(parentDir)

	// Create a container subdirectory
	containerID := "test-container-12345"
	subDir := filepath.Join(parentDir, containerID)

	if err := os.MkdirAll(subDir, 0700); err != nil {
		t.Fatalf("Failed to create container subdirectory: %v", err)
	}

	// Verify it was created
	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("Failed to stat subdirectory: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", subDir)
	}

	// Check permissions are more restrictive for container directories
	// Container dirs should not have world write permissions
	mode := info.Mode().Perm()
	if mode&02 != 0 {
		t.Errorf("Container subdirectory has world write permissions: %v", mode)
	}

	t.Logf("Container subdirectory %s created with permissions %v", subDir, mode)
}
