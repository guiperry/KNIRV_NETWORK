package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateTestDatabasePath(t *testing.T) {
	// Test creating a test database path
	testName := "sample_test"
	dbPath := CreateTestDatabasePath(testName)
	
	// Verify the path is in test-reports directory
	expectedPrefix := filepath.Join("test-reports", "test_"+testName+"_")
	if !filepath.HasPrefix(dbPath, expectedPrefix) {
		t.Errorf("Expected path to start with %s, got %s", expectedPrefix, dbPath)
	}
}

func TestCleanupTestDatabases(t *testing.T) {
	// Create some test database directories
	testDirs := []string{
		"test_db_123456789",
		"testdb_987654321",
		"test_chromem_111222333",
	}
	
	// Create the test directories
	for _, dir := range testDirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}
	
	// Verify they exist
	for _, dir := range testDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Fatalf("Test directory %s was not created", dir)
		}
	}
	
	// Run cleanup
	err := CleanupTestDatabases()
	if err != nil {
		t.Fatalf("CleanupTestDatabases failed: %v", err)
	}
	
	// Verify they are removed
	for _, dir := range testDirs {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("Test directory %s was not cleaned up", dir)
		}
	}
}

func TestStartEndTestSession(t *testing.T) {
	// Test starting a test session
	session := StartTestSession("test_session", "KNIRVORACLE")
	
	if session == nil {
		t.Fatal("StartTestSession returned nil")
	}
	
	if session.Name != "test_session" {
		t.Errorf("Expected session name 'test_session', got '%s'", session.Name)
	}
	
	if session.Component != "KNIRVORACLE" {
		t.Errorf("Expected component 'KNIRVORACLE', got '%s'", session.Component)
	}
	
	if session.StartTime.IsZero() {
		t.Error("Session start time should not be zero")
	}
	
	// Test ending the session
	EndTestSession(session, "passed", nil)
	
	// Test ending with error
	EndTestSession(session, "failed", os.ErrNotExist)
	
	// Test ending nil session (should not panic)
	EndTestSession(nil, "passed", nil)
}

func TestStringifyMetadataTestHelper(t *testing.T) {
	// Test metadata conversion
	metadata := map[string]interface{}{
		"string_key": "string_value",
		"int_key":    42,
		"float_key":  3.14,
		"bool_key":   true,
	}
	
	result := StringifyMetadataTestHelper(metadata)
	
	expected := map[string]string{
		"string_key": "string_value",
		"int_key":    "42",
		"float_key":  "3.14",
		"bool_key":   "true",
	}
	
	for key, expectedValue := range expected {
		if result[key] != expectedValue {
			t.Errorf("For key %s, expected %s, got %s", key, expectedValue, result[key])
		}
	}
}

func TestWaitForChromemDB(t *testing.T) {
	// Test successful condition
	callCount := 0
	err := WaitForChromemDB(1*time.Second, func() (bool, error) {
		callCount++
		if callCount >= 3 {
			return true, nil
		}
		return false, nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if callCount < 3 {
		t.Errorf("Expected at least 3 calls, got %d", callCount)
	}
	
	// Test timeout
	err = WaitForChromemDB(100*time.Millisecond, func() (bool, error) {
		return false, nil
	})
	
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}
