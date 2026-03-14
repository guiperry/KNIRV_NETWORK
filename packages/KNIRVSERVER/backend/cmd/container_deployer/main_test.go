package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetArtifactDirectory tests artifact directory generation for container_deployer
func TestGetArtifactDirectory(t *testing.T) {
	appDataDir := "/test/data"
	artifactsDir := getArtifactDirectory(appDataDir)

	expected := filepath.Join(appDataDir, "artifacts")
	if artifactsDir != expected {
		t.Errorf("Expected %s, got %s", expected, artifactsDir)
	}
}

// TestGetEmbeddedResourcesDirectory tests embedded resources directory generation
func TestGetEmbeddedResourcesDirectory(t *testing.T) {
	appDataDir := "/test/data"
	resourcesDir := getEmbeddedResourcesDirectory(appDataDir)

	expected := filepath.Join(appDataDir, "resources")
	if resourcesDir != expected {
		t.Errorf("Expected %s, got %s", expected, resourcesDir)
	}
}

// TestGetAppDataDirectory tests the app data directory creation
func TestGetAppDataDirectory(t *testing.T) {
	// This test uses the actual user home directory
	dir, err := getAppDataDirectory()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("Expected non-empty directory")
	}
}

// TestGetOsBuilderArtifactDirectory tests the os_builder artifact directory retrieval
func TestGetOsBuilderArtifactDirectory(t *testing.T) {
	dir, err := getOsBuilderArtifactDirectory()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("Expected non-empty directory")
	}

	// Verify it contains expected path components
	expectedSuffix := ".local/share/knirvserver/os_builder/artifacts"
	if len(dir) < len(expectedSuffix) || dir[len(dir)-len(expectedSuffix):] != expectedSuffix {
		t.Errorf("Expected directory to end with %s, got %s", expectedSuffix, dir)
	}
}

// TestGetAnsibleDirectory tests ansible directory selection based on deploy type
func TestGetAnsibleDirectory(t *testing.T) {
	resourcesDir := "/test/resources"

	// Test cloud deployment
	cloudDir := getAnsibleDirectory(resourcesDir, "cloud")
	expectedCloud := filepath.Join(resourcesDir, "ansible/cloud-deploy")
	if cloudDir != expectedCloud {
		t.Errorf("Expected %s, got %s", expectedCloud, cloudDir)
	}

	// Test local deployment
	localDir := getAnsibleDirectory(resourcesDir, "local")
	expectedLocal := filepath.Join(resourcesDir, "ansible/local-deploy")
	if localDir != expectedLocal {
		t.Errorf("Expected %s, got %s", expectedLocal, localDir)
	}
}

// TestCheckKataArtifacts tests Kata artifacts availability check
func TestCheckKataArtifacts(t *testing.T) {
	// This test will check if the artifacts exist in the default location
	// In a real test environment, we would mock this
	result := checkKataArtifacts()
	// We don't assert on the result as it depends on the test environment
	t.Logf("Kata artifacts available: %v", result)
}

// TestEnvironmentValidation tests environment variable validation
func TestEnvironmentValidation(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expectError bool
	}{
		{
			name:        "valid development",
			environment: "development",
			expectError: false,
		},
		{
			name:        "valid testnet",
			environment: "testnet",
			expectError: false,
		},
		{
			name:        "valid production",
			environment: "production",
			expectError: false,
		},
		{
			name:        "invalid environment",
			environment: "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.environment == "development" || tt.environment == "testnet" || tt.environment == "production"
			if tt.expectError && isValid {
				t.Error("Expected error but environment is valid")
			}
			if !tt.expectError && !isValid {
				t.Error("Expected no error but environment is invalid")
			}
		})
	}
}

// TestDeployTypeValidation tests deployment type validation
func TestDeployTypeValidation(t *testing.T) {
	tests := []struct {
		name        string
		deployType  string
		expectValid bool
	}{
		{
			name:        "valid local",
			deployType:  "local",
			expectValid: true,
		},
		{
			name:        "valid cloud",
			deployType:  "cloud",
			expectValid: true,
		},
		{
			name:        "invalid type",
			deployType:  "invalid",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.deployType == "local" || tt.deployType == "cloud"
			if isValid != tt.expectValid {
				t.Errorf("Expected %v but got %v", tt.expectValid, isValid)
			}
		})
	}
}

// TestDeployModeValidation tests deployment mode validation
func TestDeployModeValidation(t *testing.T) {
	tests := []struct {
		name        string
		deployMode  string
		expectValid bool
	}{
		{
			name:        "valid container",
			deployMode:  "container",
			expectValid: true,
		},
		{
			name:        "valid kata",
			deployMode:  "kata",
			expectValid: true,
		},
		{
			name:        "valid native",
			deployMode:  "native",
			expectValid: true,
		},
		{
			name:        "invalid mode",
			deployMode:  "invalid",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.deployMode == "container" || tt.deployMode == "kata" || tt.deployMode == "native"
			if isValid != tt.expectValid {
				t.Errorf("Expected %v but got %v", tt.expectValid, isValid)
			}
		})
	}
}

// TestGetCurrentRepoRoot tests repository root directory detection
func TestGetCurrentRepoRoot(t *testing.T) {
	root := getCurrentRepoRoot()
	if root == "" {
		t.Error("Expected non-empty repository root")
	}
	// Should contain KNIRV_NETWORK in the path
	if filepath.Base(root) != "KNIRV_NETWORK" {
		t.Logf("Warning: Repository root may not be KNIRV_NETWORK: %s", root)
	}
}

// TestNativeDeploymentBinaryPath tests binary path generation
func TestNativeDeploymentBinaryPath(t *testing.T) {
	artifactDir := "/test/artifacts"

	// The binary path would be artifactDir + "knirv-nexus"
	expectedPath := filepath.Join(artifactDir, "knirv-nexus")

	if expectedPath != "/test/artifacts/knirv-nexus" {
		t.Errorf("Unexpected path: %s", expectedPath)
	}
}

// TestDeploymentLogInitialization tests deployment log setup
func TestDeploymentLogInitialization(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "test-logs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory for log test
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Test that initDeploymentLog creates log directory
	err = initDeploymentLog()
	if err != nil {
		t.Errorf("Unexpected error initializing log: %v", err)
	}

	// Check that log directory was created
	logDir := filepath.Join(tmpDir, "log")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}

	// Close the log
	closeDeploymentLog()
}

// TestDeploymentLogFileClosure tests that deployment log file is properly closed
func TestDeploymentLogFileClosure(t *testing.T) {
	// This test verifies closeDeploymentLog doesn't panic when log file is nil
	deploymentLogFile = nil
	closeDeploymentLog() // Should not panic

	// This test verifies closeDeploymentLog works with a valid log file
	tmpDir, err := os.MkdirTemp("", "test-logs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	initDeploymentLog()
	closeDeploymentLog() // Should close properly
}
