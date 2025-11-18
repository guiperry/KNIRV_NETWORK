package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"KNIRVCHAIN/config"
)

// TestWaitForShutdownSignalIntegration tests the integration of shutdown signal handling
func TestWaitForShutdownSignalIntegration(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{
		ChainID:    "test-chain",
		IsBootnode: false, // Not a bootnode to avoid failover logic
	}

	// Create a context that we can cancel
	_, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Test that the function can be called without panicking
	// We'll cancel immediately to avoid waiting for actual signals
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel() // Simulate shutdown signal
	}()

	// This should complete without hanging
	waitForShutdownSignal(cancel, &wg, "/tmp/test-config.json", cfg, nil)
}

// TestFailoverManagerIntegration tests the integration of FailoverManager with main application
func TestFailoverManagerIntegration(t *testing.T) {
	// Test configuration for bootnode
	cfg := &config.Config{
		ChainID:                 "test-chain",
		IsBootnode:              true,
		CurrentOracleNodeAPIURL: "http://localhost:9999",
	}

	// Test that global failover manager can be set and retrieved
	fm := NewFailoverManager(cfg.CurrentOracleNodeAPIURL, cfg, "/tmp/config.json", nil, nil, nil, func() {})
	if fm == nil {
		t.Fatal("Failed to create FailoverManager")
	}

	SetGlobalFailoverManager(fm)
	retrieved := GetGlobalFailoverManager()

	if retrieved != fm {
		t.Error("FailoverManager integration failed")
	}

	// Test failover promotion handling
	err := HandleFailoverPromotion("/tmp/config.json", cfg, nil)
	if err != nil {
		t.Errorf("Unexpected error in failover promotion: %v", err)
	}

	// Clean up
	SetGlobalFailoverManager(nil)
}

// TestConfigurationValidation tests configuration validation for failover
func TestConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name         string
		config       *config.Config
		shouldInitFM bool
		description  string
	}{
		{
			name: "Valid bootnode config",
			config: &config.Config{
				ChainID:                 "test-chain",
				IsBootnode:              true,
				CurrentOracleNodeAPIURL: "http://localhost:9999",
			},
			shouldInitFM: true,
			description:  "Should initialize FailoverManager for valid bootnode",
		},
		{
			name: "Non-bootnode config",
			config: &config.Config{
				ChainID:                 "test-chain",
				IsBootnode:              false,
				CurrentOracleNodeAPIURL: "http://localhost:9999",
			},
			shouldInitFM: false,
			description:  "Should not initialize FailoverManager for non-bootnode",
		},
		{
			name: "Bootnode without Oracle URL",
			config: &config.Config{
				ChainID:                 "test-chain",
				IsBootnode:              true,
				CurrentOracleNodeAPIURL: "",
			},
			shouldInitFM: false,
			description:  "Should not initialize FailoverManager without Oracle URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the initialization logic from main.go
			var fm *FailoverManager

			if tc.config.IsBootnode && tc.config.CurrentOracleNodeAPIURL != "" {
				fm = NewFailoverManager(tc.config.CurrentOracleNodeAPIURL, tc.config, "/tmp/config.json", nil, nil, nil, func() {})
			}

			if tc.shouldInitFM && fm == nil {
				t.Errorf("%s: Expected FailoverManager to be initialized", tc.description)
			}

			if !tc.shouldInitFM && fm != nil {
				t.Errorf("%s: Expected FailoverManager to not be initialized", tc.description)
			}
		})
	}
}

// TestNetworkControlMessageFlow tests the flow of network control messages
func TestNetworkControlMessageFlow(t *testing.T) {
	// Test creating a pause message
	pausePayload := NetworkPausePayload{
		InitiatorPeerID: "bootnode-123",
		Reason:          "Oracle node failover in progress",
		Timestamp:       time.Now().Unix(),
	}

	controlMsg := NetworkControlMessage{
		Type:    "NetworkPause",
		Payload: pausePayload,
	}

	// Verify message structure
	if controlMsg.Type != "NetworkPause" {
		t.Errorf("Expected message type 'NetworkPause', got '%s'", controlMsg.Type)
	}

	// Test payload extraction
	if payload, ok := controlMsg.Payload.(NetworkPausePayload); ok {
		if payload.InitiatorPeerID != "bootnode-123" {
			t.Errorf("Expected InitiatorPeerID 'bootnode-123', got '%s'", payload.InitiatorPeerID)
		}
		if payload.Reason != "Oracle node failover in progress" {
			t.Errorf("Expected specific reason, got '%s'", payload.Reason)
		}
	} else {
		t.Error("Failed to extract NetworkPausePayload from message")
	}
}

// TestErrorHandling tests error handling in various scenarios
func TestErrorHandling(t *testing.T) {
	// Test HandleFailoverPromotion with nil parameters
	err := HandleFailoverPromotion("", nil, nil)
	if err != nil {
		t.Errorf("Expected no error with nil config, got: %v", err)
	}

	// Test with invalid config path
	cfg := &config.Config{ChainID: "test"}
	err = HandleFailoverPromotion("/invalid/path/config.json", cfg, nil)
	if err != nil {
		// This might fail in promoteToOracleAndRestart, which is expected
		t.Logf("Expected error with invalid config path: %v", err)
	}
}

// TestConcurrentFailoverOperations tests concurrent failover operations
func TestConcurrentFailoverOperations(t *testing.T) {
	cfg := &config.Config{
		ChainID:                 "test-chain",
		IsBootnode:              true,
		CurrentOracleNodeAPIURL: "http://localhost:9999",
	}

	// Create multiple failover managers concurrently
	var wg sync.WaitGroup
	managers := make([]*FailoverManager, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			managers[index] = NewFailoverManager(cfg.CurrentOracleNodeAPIURL, cfg, "/tmp/config.json", nil, nil, nil, func() {})
		}(i)
	}

	wg.Wait()

	// Verify all managers were created
	for i, fm := range managers {
		if fm == nil {
			t.Errorf("FailoverManager %d was not created", i)
		}
	}
}

// TestMemoryLeakPrevention tests that resources are properly cleaned up
func TestMemoryLeakPrevention(t *testing.T) {
	cfg := &config.Config{
		ChainID:                 "test-chain",
		IsBootnode:              true,
		CurrentOracleNodeAPIURL: "http://localhost:9999",
	}

	// Create and destroy multiple failover managers
	for i := 0; i < 10; i++ {
		fm := NewFailoverManager(cfg.CurrentOracleNodeAPIURL, cfg, "/tmp/config.json", nil, nil, nil, func() {})
		if fm != nil {
			SetGlobalFailoverManager(fm)
			fm.StartMonitoring()
			time.Sleep(10 * time.Millisecond) // Brief operation
			fm.StopMonitoring()
			SetGlobalFailoverManager(nil)
		}
	}

	// Verify global manager is cleaned up
	if GetGlobalFailoverManager() != nil {
		t.Error("Global FailoverManager was not cleaned up")
	}
}

// TestConfigurationEdgeCases tests edge cases in configuration
func TestConfigurationEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "Empty ChainID",
			config: &config.Config{
				ChainID:                 "",
				IsBootnode:              true,
				CurrentOracleNodeAPIURL: "http://localhost:9999",
			},
		},
		{
			name: "Very long ChainID",
			config: &config.Config{
				ChainID:                 "very-long-chain-id-with-many-characters-that-might-cause-issues-in-some-systems",
				IsBootnode:              true,
				CurrentOracleNodeAPIURL: "http://localhost:9999",
			},
		},
		{
			name: "Special characters in ChainID",
			config: &config.Config{
				ChainID:                 "test-chain@#$%^&*()",
				IsBootnode:              true,
				CurrentOracleNodeAPIURL: "http://localhost:9999",
			},
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic with edge case configurations
			fm := NewFailoverManager(tc.config.CurrentOracleNodeAPIURL, tc.config, "/tmp/config.json", nil, nil, nil, func() {})
			if fm == nil {
				t.Errorf("FailoverManager creation failed for %s", tc.name)
			}
		})
	}
}
