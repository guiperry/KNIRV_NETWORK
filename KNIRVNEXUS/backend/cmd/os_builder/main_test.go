package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAWSConfigValidation tests the AWS configuration validation
func TestAWSConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *awsBuildConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &awsBuildConfig{
				Region:       "us-east-1",
				AMIName:      "test-ami",
				InstanceType: "t3.medium",
				WorkDir:      "/tmp/test",
			},
			expectError: false,
		},
		{
			name: "missing region",
			config: &awsBuildConfig{
				AMIName:      "test-ami",
				InstanceType: "t3.medium",
				WorkDir:      "/tmp/test",
			},
			expectError: true,
		},
		{
			name: "missing ami name",
			config: &awsBuildConfig{
				Region:       "us-east-1",
				InstanceType: "t3.medium",
				WorkDir:      "/tmp/test",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAWSConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestGetDefaultBuildConfig tests the default build configuration
func TestGetDefaultBuildConfig(t *testing.T) {
	workDir := "/test/work/dir"
	config := getDefaultBuildConfig(workDir)

	if config.Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", config.Region)
	}

	if config.InstanceType != "t3.medium" {
		t.Errorf("Expected instance type t3.medium, got %s", config.InstanceType)
	}

	if config.WorkDir != workDir {
		t.Errorf("Expected work dir %s, got %s", workDir, config.WorkDir)
	}
}

// TestAWSRegionFromEnv tests AWS region detection from environment
func TestAWSRegionFromEnv(t *testing.T) {
	// Test default region when env var is not set
	os.Unsetenv("AWS_DEFAULT_REGION")
	region := getAWSRegion()
	if region != "us-east-1" {
		t.Errorf("Expected default region us-east-1, got %s", region)
	}

	// Test custom region from env var
	os.Setenv("AWS_DEFAULT_REGION", "eu-west-1")
	defer os.Unsetenv("AWS_DEFAULT_REGION")
	region = getAWSRegion()
	if region != "eu-west-1" {
		t.Errorf("Expected region eu-west-1, got %s", region)
	}
}

// TestIsAWSConfigured tests AWS credential configuration check
func TestIsAWSConfigured(t *testing.T) {
	// Reset environment
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	if isAWSConfigured() {
		t.Error("Expected AWS not configured without credentials")
	}

	// Set credentials
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	if !isAWSConfigured() {
		t.Error("Expected AWS to be configured with credentials")
	}
}

// TestAWSCredentialsStatus tests the credentials status check
func TestAWSCredentialsStatus(t *testing.T) {
	// Reset environment
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SESSION_TOKEN")

	status, err := getAWSCredentialsStatus()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if status["AWS_ACCESS_KEY_ID"] {
		t.Error("Expected AWS_ACCESS_KEY_ID to be false")
	}

	if status["AWS_SECRET_ACCESS_KEY"] {
		t.Error("Expected AWS_SECRET_ACCESS_KEY to be false")
	}

	// Set credentials
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	status, err = getAWSCredentialsStatus()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !status["AWS_ACCESS_KEY_ID"] {
		t.Error("Expected AWS_ACCESS_KEY_ID to be true")
	}

	if !status["AWS_SECRET_ACCESS_KEY"] {
		t.Error("Expected AWS_SECRET_ACCESS_KEY to be true")
	}
}

// TestFindAMIID tests AMI ID extraction from output
func TestFindAMIID(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantAMI  bool
		expected string
	}{
		{
			name:     "valid ami id in output",
			output:   "Some text ami-1234567890abcdef0 more text",
			wantAMI:  true,
			expected: "ami-1234567890abcdef0",
		},
		{
			name:     "ami id with comma",
			output:   "ami-1234567890abcdef0, other stuff",
			wantAMI:  true,
			expected: "ami-1234567890abcdef0",
		},
		{
			name:     "no ami id in output",
			output:   "Some text without AMI ID",
			wantAMI:  false,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amiID, err := findAMIID(tt.output)
			if tt.wantAMI {
				if err != nil {
					t.Errorf("Expected AMI ID but got error: %v", err)
				}
				if amiID != tt.expected {
					t.Errorf("Expected AMI ID %s, got %s", tt.expected, amiID)
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}

// TestAMIBuildLog tests the AMI build log functionality
func TestAMIBuildLog(t *testing.T) {
	log := newAMIBuildLog()

	if len(log.Steps) != 0 {
		t.Error("Expected empty steps initially")
	}

	// Add some steps
	log.AddStep("Step 1: Initialize Packer")
	log.AddStep("Step 2: Build AMI")

	if len(log.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(log.Steps))
	}

	// Add an error
	log.AddError("Test error")

	if len(log.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(log.Errors))
	}

	// Test save (will fail because dir doesn't exist, but we can check the function runs)
	log.EndTime = log.StartTime.Add(1)
	err := log.Save("/tmp/nonexistent/directory/test.log")
	if err == nil {
		t.Error("Expected error when saving to nonexistent directory")
	}
}

// TestIsBaseKaliImageAvailable tests the base Kali image availability check
func TestIsBaseKaliImageAvailable(t *testing.T) {
	// Create a temporary directory and file for testing
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.ova")

	// Test with non-existent file
	if isBaseKaliImageAvailable(testFile) {
		t.Error("Expected false for non-existent file")
	}

	// Create the file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.Close()

	// Test with existing file
	if !isBaseKaliImageAvailable(testFile) {
		t.Error("Expected true for existing file")
	}
}

// TestGetArtifactDirectory tests artifact directory generation
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
	// We'll just verify it doesn't error
	dir, err := getAppDataDirectory()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("Expected non-empty directory")
	}
}
