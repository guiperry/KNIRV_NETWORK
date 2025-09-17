package utils

import (
	"context"
	"fmt"
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
