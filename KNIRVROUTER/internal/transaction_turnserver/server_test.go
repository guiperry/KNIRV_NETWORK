package transaction_turnserver

import (
	"testing"
	"time"

	"KNIRVROUTER/internal/types"
)

// MockBlockchain implements BlockchainInterface for testing
type MockBlockchain struct {
	transactionPool []*types.Transaction
}

func NewMockBlockchain() *MockBlockchain {
	return &MockBlockchain{
		transactionPool: make([]*types.Transaction, 0),
	}
}

func (mb *MockBlockchain) AddTransactionToTransactionPool(transaction *types.Transaction) error {
	mb.transactionPool = append(mb.transactionPool, transaction)
	return nil
}

func (mb *MockBlockchain) GetTransactionPool() []*types.Transaction {
	return mb.transactionPool
}

func TestMockTxPoolAdapter(t *testing.T) {
	// Create a mock adapter
	adapter := NewMockTxPoolAdapter("KNIRVROUTER-test-miner-address")

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

func TestBlockchainAdapter(t *testing.T) {
	// Create a mock blockchain
	mockBlockchain := NewMockBlockchain()

	// Create a blockchain adapter
	adapter := NewBlockchainAdapterWithBlockchain(mockBlockchain, "KNIRVROUTER-test-miner-address")

	// Test TURN session transaction
	sessionData := map[string]interface{}{
		"type":        "TURN_SESSION_START",
		"client_addr": "192.168.1.100:12345",
		"username":    "testuser",
		"realm":       "knirvchain.local",
	}

	err := adapter.SubmitTurnSessionTx(sessionData)
	if err != nil {
		t.Fatalf("Failed to submit TURN session transaction: %v", err)
	}

	// Verify transaction was added to blockchain
	txPool := mockBlockchain.GetTransactionPool()
	if len(txPool) != 1 {
		t.Fatalf("Expected 1 transaction in pool, got %d", len(txPool))
	}

	tx := txPool[0]
	if tx.From != "TURN_SERVER" {
		t.Errorf("Expected transaction from TURN_SERVER, got %s", tx.From)
	}
	if tx.To != "KNIRVROUTER-test-miner-address" {
		t.Errorf("Expected transaction to miner address, got %s", tx.To)
	}

	// Test NRN mint transaction
	err = adapter.SubmitNRNMintTx("test-recipient", "1000000000000000000", "test_reward", "proof123")
	if err != nil {
		t.Fatalf("Failed to submit NRN mint transaction: %v", err)
	}

	// Verify second transaction was added
	txPool = mockBlockchain.GetTransactionPool()
	if len(txPool) != 2 {
		t.Fatalf("Expected 2 transactions in pool, got %d", len(txPool))
	}

	mintTx := txPool[1]
	if mintTx.From != "NRN_MINTER" {
		t.Errorf("Expected mint transaction from NRN_MINTER, got %s", mintTx.From)
	}
	if mintTx.To != "test-recipient" {
		t.Errorf("Expected mint transaction to test-recipient, got %s", mintTx.To)
	}

	// Test connectivity proof reward
	err = adapter.SubmitConnectivityProofReward("node123", "proof456", 85.5, "500000000000000000")
	if err != nil {
		t.Fatalf("Failed to submit connectivity proof reward: %v", err)
	}

	// Test participation reward
	err = adapter.SubmitParticipationReward("node456", "validator", "250000000000000000")
	if err != nil {
		t.Fatalf("Failed to submit participation reward: %v", err)
	}

	// Test minting stats
	stats := adapter.GetMintingStats()
	if stats["data_source"] != "blockchain" {
		t.Error("Expected data_source to be 'blockchain'")
	}
	if stats["transaction_pool_size"] != 4 {
		t.Errorf("Expected 4 transactions in pool, got %v", stats["transaction_pool_size"])
	}
	if stats["total_turn_sessions"] != 1 {
		t.Errorf("Expected 1 TURN session, got %v", stats["total_turn_sessions"])
	}
	if stats["total_nrn_mints"] != 3 {
		t.Errorf("Expected 3 NRN mints, got %v", stats["total_nrn_mints"])
	}
}

func TestServerCreation(t *testing.T) {
	// Skip if running in CI environment
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a mock blockchain and real adapter
	mockBlockchain := NewMockBlockchain()
	adapter := NewBlockchainAdapterWithBlockchain(mockBlockchain, "KNIRVROUTER-test-miner-address")

	// Create a server with test ports
	// Use high port numbers to avoid conflicts
	server, err := NewServer(34780, 34781, 34782, adapter)
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

func TestServerCreationWithMockAdapter(t *testing.T) {
	// Skip if running in CI environment
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a mock adapter (deprecated but still supported)
	adapter := NewMockTxPoolAdapter("KNIRVROUTER-test-miner-address")

	// Create a server with test ports
	// Use high port numbers to avoid conflicts
	server, err := NewServer(34782, 34783, 34784, adapter)
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
