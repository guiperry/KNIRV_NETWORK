package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WaitForChromemDB is a helper function that polls until a condition is met or timeout occurs
func WaitForChromemDB(timeout time.Duration, checkFunc func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(200 * time.Millisecond) // Poll every 200ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ChromemDB update: %w", ctx.Err())
		case <-ticker.C:
			done, err := checkFunc()
			if err != nil {
				// Don't immediately fail on "not found" type errors during polling
				// Allow a few retries for eventual consistency
				// Log the transient error if needed: log.Printf("Polling ChromemDB: %v", err)
			}
			if done {
				return nil
			}
		}
	}
}

// stringifyMetadataTestHelper converts metadata map[string]interface{} to map[string]string for ChromemDB test comparisons
// This is a test-only helper that should match the logic in ChromemManager or conversion utilities
func StringifyMetadataTestHelper(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		result[k] = fmt.Sprintf("%v", v) // Simplified conversion for test comparison
	}
	return result
}

// CleanupTestDatabases removes all test database files from the current directory
func CleanupTestDatabases() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find all test database files and directories
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var cleanupErrors []string
	for _, entry := range entries {
		name := entry.Name()
		// Match test database patterns
		if strings.HasPrefix(name, "test_db_") ||
			strings.HasPrefix(name, "testdb_") ||
			strings.HasPrefix(name, "test_chromem_") ||
			(strings.Contains(name, "test") && (strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".leveldb"))) {

			fullPath := filepath.Join(cwd, name)
			if err := os.RemoveAll(fullPath); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to remove %s: %v", name, err))
			}
		}
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(cleanupErrors, "; "))
	}

	return nil
}

// CreateTestDatabasePath creates a unique test database path in the test-reports directory
func CreateTestDatabasePath(testName string) string {
	timestamp := time.Now().UnixNano()
	dbName := fmt.Sprintf("test_%s_%d", testName, timestamp)
	return filepath.Join("test-reports", dbName)
}

// TestSession represents a test session for monitoring integration
type TestSession struct {
	ID        string
	Name      string
	Component string
	StartTime time.Time
}

// StartTestSession starts a new test session and reports to monitor if available
func StartTestSession(testName, component string) *TestSession {
	session := &TestSession{
		ID:        fmt.Sprintf("%s_%d", testName, time.Now().UnixNano()),
		Name:      testName,
		Component: component,
		StartTime: time.Now(),
	}

	// TODO: Integrate with network monitor when available
	// This would connect to the test monitor API to report test start

	return session
}

// EndTestSession ends a test session and reports results to monitor if available
func EndTestSession(session *TestSession, status string, err error) {
	if session == nil {
		return
	}

	// TODO: Integrate with network monitor when available
	// This would connect to the test monitor API to report test completion

	// For now, just log the completion
	duration := time.Since(session.StartTime)
	if err != nil {
		fmt.Printf("Test session '%s' completed with status '%s' in %v: %v\n",
			session.Name, status, duration, err)
	} else {
		fmt.Printf("Test session '%s' completed with status '%s' in %v\n",
			session.Name, status, duration)
	}
}
