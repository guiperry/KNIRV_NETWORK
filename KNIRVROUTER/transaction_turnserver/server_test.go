package transaction_turnserver

import (
	"testing"
	"time"
)

func TestMockTxPoolAdapter(t *testing.T) {
	// Create a mock adapter
	adapter := NewMockTxPoolAdapter("KNIRVCHAIN-test-miner-address")
	
	// Create test session data
	sessionData := map[string]interface{}{
		"type":        "TURN_SESSION_START",
		"client_addr": "192.168.1.100:12345",
		"username":    "testuser",
		"realm":       "knirvchain.local",
	}
	
	// Submit the transaction
	err := adapter.SubmitTurnSessionTx(sessionData)
	if err != nil {
		t.Fatalf("Failed to submit transaction: %v", err)
	}
	
	// Verify that recorded_at and mock fields were added
	if _, ok := sessionData["recorded_at"]; !ok {
		t.Error("recorded_at field was not added to session data")
	}
	
	if mock, ok := sessionData["mock"]; !ok || mock != true {
		t.Error("mock field was not set to true in session data")
	}
}

func TestServerCreation(t *testing.T) {
	// Skip if running in CI environment
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	
	// Create a mock adapter
	adapter := NewMockTxPoolAdapter("KNIRVCHAIN-test-miner-address")
	
	// Create a server with test ports
	// Use high port numbers to avoid conflicts
	server, err := NewServer(34780, 34781, adapter)
	if err != nil {
		t.Fatalf("Failed to create TURN server: %v", err)
	}
	
	// Start the server
	server.Start()
	
	// Verify the server is running
	if !server.IsRunning() {
		t.Error("Server should be running after Start() is called")
	}
	
	// Wait a moment to ensure server is fully started
	time.Sleep(100 * time.Millisecond)
	
	// Stop the server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop TURN server: %v", err)
	}
	
	// Verify the server is stopped
	if server.IsRunning() {
		t.Error("Server should not be running after Stop() is called")
	}
}