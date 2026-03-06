package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotKeyFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Test case 1: Valid key file with key=value pairs
	validKeyFile := filepath.Join(tempDir, "valid.key")
	validContent := `# Comment line
STRIPE_SECRET_KEY=____test_123
STRIPE_WEBHOOK_SECRET=whsec_456

COINBASE_API_KEY=api_789
ROOT_PRIVATE_KEY=0xabc123
`
	err := os.WriteFile(validKeyFile, []byte(validContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	values := loadDotKeyFile(validKeyFile)
	expected := map[string]string{
		"STRIPE_SECRET_KEY":     "____test_123",
		"STRIPE_WEBHOOK_SECRET": "whsec_456",
		"COINBASE_API_KEY":      "api_789",
		"ROOT_PRIVATE_KEY":      "0xabc123",
	}

	for key, expectedValue := range expected {
		if actualValue, exists := values[key]; !exists || actualValue != expectedValue {
			t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
		}
	}

	// Test case 2: File does not exist
	nonExistentFile := filepath.Join(tempDir, "nonexistent.key")
	values = loadDotKeyFile(nonExistentFile)
	if len(values) != 0 {
		t.Errorf("Expected empty map for non-existent file, got %v", values)
	}

	// Test case 3: Empty file
	emptyFile := filepath.Join(tempDir, "empty.key")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
	}
	values = loadDotKeyFile(emptyFile)
	if len(values) != 0 {
		t.Errorf("Expected empty map for empty file, got %v", values)
	}

	// Test case 4: File with only comments and empty lines
	commentOnlyFile := filepath.Join(tempDir, "comments.key")
	commentContent := "# This is a comment\n\n# Another comment\n"
	err = os.WriteFile(commentOnlyFile, []byte(commentContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create comment-only test file: %v", err)
	}
	values = loadDotKeyFile(commentOnlyFile)
	if len(values) != 0 {
		t.Errorf("Expected empty map for comment-only file, got %v", values)
	}

	// Test case 5: Malformed lines (no =)
	malformedFile := filepath.Join(tempDir, "malformed.key")
	malformedContent := "INVALID_LINE\nSTRIPE_SECRET_KEY=valid\nANOTHER_INVALID"
	err = os.WriteFile(malformedFile, []byte(malformedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create malformed test file: %v", err)
	}
	values = loadDotKeyFile(malformedFile)
	if len(values) != 1 || values["STRIPE_SECRET_KEY"] != "valid" {
		t.Errorf("Expected only one valid key-value pair, got %v", values)
	}

	// Test case 6: Project root .key file takes precedence
	projectRootKeyFile := ".key"
	projectRootContent := "ROOT_PRIVATE_KEY=from_root"
	err = os.WriteFile(projectRootKeyFile, []byte(projectRootContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create project root .key file: %v", err)
	}
	defer os.Remove(projectRootKeyFile) // Clean up

	values = loadDotKeyFile(validKeyFile) // Pass a different file, but .key exists
	if values["ROOT_PRIVATE_KEY"] != "from_root" {
		t.Errorf("Expected project root .key to take precedence, got %v", values)
	}
}

func TestLoadDotKeyFile_TrimSpaces(t *testing.T) {
	tempDir := t.TempDir()
	keyFile := filepath.Join(tempDir, "trim.key")
	content := "  KEY1  =  value1  \nKEY2=value2\n  KEY3 = value3  "
	err := os.WriteFile(keyFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	values := loadDotKeyFile(keyFile)
	expected := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	for key, expectedValue := range expected {
		if actualValue, exists := values[key]; !exists || actualValue != expectedValue {
			t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
		}
	}
}
