package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"KNIRVORACLE/config"
)

// TestNewFailoverManager tests the creation of a new FailoverManager
func TestNewFailoverManager(t *testing.T) {
	cfg := &config.Config{
		ChainID: "test-chain",
	}

	// Test with valid URL
	fm := NewFailoverManager("http://localhost:9999", cfg, "/tmp/config.json", nil, nil, nil, func() {})
	if fm == nil {
		t.Fatal("Expected FailoverManager to be created, got nil")
	}

	if fm.GetCurrentOracleAPIURL() != "http://localhost:9999" {
		t.Errorf("Expected URL to be 'http://localhost:9999', got '%s'", fm.GetCurrentOracleAPIURL())
	}

	// Test with empty URL
	fm2 := NewFailoverManager("", cfg, "/tmp/config.json", nil, nil, nil, func() {})
	if fm2 != nil {
		t.Error("Expected nil FailoverManager for empty URL")
	}
}

// TestFailoverManagerGlobalFunctions tests the global failover manager functions
func TestFailoverManagerGlobalFunctions(t *testing.T) {
	cfg := &config.Config{
		ChainID: "test-chain",
	}

	// Test setting and getting global failover manager
	fm := NewFailoverManager("http://localhost:9999", cfg, "/tmp/config.json", nil, nil, nil, func() {})

	SetGlobalFailoverManager(fm)
	retrieved := GetGlobalFailoverManager()

	if retrieved != fm {
		t.Error("Global failover manager not set/retrieved correctly")
	}

	// Test with nil
	SetGlobalFailoverManager(nil)
	retrieved = GetGlobalFailoverManager()
	if retrieved != nil {
		t.Error("Expected nil global failover manager")
	}
}

// TestFailoverManagerHealthCheck tests the health check functionality
func TestFailoverManagerHealthCheck(t *testing.T) {
	// Create a test server that responds to health checks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		ChainID: "test-chain",
	}

	fm := NewFailoverManager(server.URL, cfg, "/tmp/config.json", nil, nil, nil, func() {})
	if fm == nil {
		t.Fatal("Failed to create FailoverManager")
	}

	// Test health check
	fm.CheckOracleStatus()

	// Should be online after successful health check
	if !fm.IsOracleOnline() {
		t.Error("Expected oracle to be online after successful health check")
	}
}

// TestFailoverManagerOfflineDetection tests offline detection
func TestFailoverManagerOfflineDetection(t *testing.T) {
	// Create a test server that will be closed to simulate offline
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cfg := &config.Config{
		ChainID: "test-chain",
	}

	fm := NewFailoverManager(server.URL, cfg, "/tmp/config.json", nil, nil, nil, func() {})
	if fm == nil {
		t.Fatal("Failed to create FailoverManager")
	}

	// First check should succeed
	fm.CheckOracleStatus()

	// Close server to simulate offline
	server.Close()

	// Second check should detect offline
	fm.CheckOracleStatus()

	if fm.IsOracleOnline() {
		t.Error("Expected oracle to be detected as offline")
	}
}

// TestNetworkControlMessage tests network control message creation
func TestNetworkControlMessage(t *testing.T) {
	pausePayload := NetworkPausePayload{
		InitiatorPeerID: "test-peer-123",
		Reason:          "Test failover",
		Timestamp:       time.Now().Unix(),
	}

	msg := NetworkControlMessage{
		Type:    "NetworkPause",
		Payload: pausePayload,
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal NetworkControlMessage: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledMsg NetworkControlMessage
	err = json.Unmarshal(data, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal NetworkControlMessage: %v", err)
	}

	if unmarshaledMsg.Type != "NetworkPause" {
		t.Errorf("Expected Type 'NetworkPause', got '%s'", unmarshaledMsg.Type)
	}
}

// TestHandleFailoverPromotion tests the failover promotion handling
func TestHandleFailoverPromotion(t *testing.T) {
	// Test with no global failover manager
	err := HandleFailoverPromotion("/tmp/config.json", &config.Config{}, nil)
	if err != nil {
		t.Errorf("Expected no error with nil failover manager, got: %v", err)
	}

	// Test with failover manager but no failover in progress
	cfg := &config.Config{
		ChainID: "test-chain",
	}
	fm := NewFailoverManager("http://localhost:9999", cfg, "/tmp/config.json", nil, nil, nil, func() {})
	SetGlobalFailoverManager(fm)

	err = HandleFailoverPromotion("/tmp/config.json", cfg, nil)
	if err != nil {
		t.Errorf("Expected no error with no failover in progress, got: %v", err)
	}

	// Clean up
	SetGlobalFailoverManager(nil)
}

// TestFailoverManagerStartStop tests starting and stopping the failover manager
func TestFailoverManagerStartStop(t *testing.T) {
	cfg := &config.Config{
		ChainID: "test-chain",
	}

	fm := NewFailoverManager("http://localhost:9999", cfg, "/tmp/config.json", nil, nil, nil, func() {})
	if fm == nil {
		t.Fatal("Failed to create FailoverManager")
	}

	// Test starting monitoring
	fm.StartMonitoring()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test stopping monitoring
	fm.StopMonitoring()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)

	// Should not panic or cause issues
}

// TestFailoverManagerElection tests the election logic
func TestFailoverManagerElection(t *testing.T) {
	cfg := &config.Config{
		ChainID: "test-chain",
	}

	fm := NewFailoverManager("http://localhost:9999", cfg, "/tmp/config.json", nil, nil, nil, func() {})
	if fm == nil {
		t.Fatal("Failed to create FailoverManager")
	}

	// Test election (currently always returns true as placeholder)
	elected := fm.AmIElectedToBecomeOracle()
	if !elected {
		t.Error("Expected to be elected (placeholder implementation)")
	}
}
