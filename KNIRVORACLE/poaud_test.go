package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"KNIRVORACLE/types"
)

// TestNetworkAuthorsManagement tests the Network Authors management functionality
func TestNetworkAuthorsManagement(t *testing.T) {
	// Create a test blockchain
	bc, err := CreateTestBlockchain()
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	defer CleanupTestBlockchain(bc)

	// Test adding network authors
	testAddress1 := "test_address_1"
	testAddress2 := "test_address_2"

	// Initially should be empty
	if bc.IsNetworkAuthor(testAddress1) {
		t.Error("Address should not be a network author initially")
	}

	// Add first network author
	err = bc.AddNetworkAuthor(testAddress1)
	if err != nil {
		t.Fatalf("Failed to add network author: %v", err)
	}

	// Verify it was added
	if !bc.IsNetworkAuthor(testAddress1) {
		t.Error("Address should be a network author after adding")
	}

	// Add second network author
	err = bc.AddNetworkAuthor(testAddress2)
	if err != nil {
		t.Fatalf("Failed to add second network author: %v", err)
	}

	// Verify both are present
	authors := bc.GetNetworkAuthors()
	if len(authors) != 2 {
		t.Errorf("Expected 2 network authors, got %d", len(authors))
	}

	// Remove first network author
	err = bc.RemoveNetworkAuthor(testAddress1)
	if err != nil {
		t.Fatalf("Failed to remove network author: %v", err)
	}

	// Verify it was removed
	if bc.IsNetworkAuthor(testAddress1) {
		t.Error("Address should not be a network author after removal")
	}

	// Verify second is still present
	if !bc.IsNetworkAuthor(testAddress2) {
		t.Error("Second address should still be a network author")
	}
}

// TestTransactionPoolManager tests the transaction pool manager functionality
func TestTransactionPoolManager(t *testing.T) {
	// Create a test blockchain
	bc, err := CreateTestBlockchain()
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	defer CleanupTestBlockchain(bc)

	tpm := bc.TransactionPoolManager
	if tpm == nil {
		t.Fatal("Transaction pool manager should not be nil")
	}

	// Create a test transaction
	testTx := CreateTestMCPTransaction()

	// Test PAS Pool operations
	tpm.AddToPASPool(testTx)

	if tpm.GetPASPoolSize() != 1 {
		t.Errorf("Expected PAS pool size 1, got %d", tpm.GetPASPoolSize())
	}

	if !tpm.IsTransactionInPASPool(testTx.TransactionHash) {
		t.Error("Transaction should be in PAS pool")
	}

	retrievedTx := tpm.GetPASPoolTransaction(testTx.TransactionHash)
	if retrievedTx == nil {
		t.Error("Should be able to retrieve transaction from PAS pool")
	}

	// Test delegation tracking
	tpm.MarkAsDelegated(testTx.TransactionHash)

	if !tpm.IsDelegated(testTx.TransactionHash) {
		t.Error("Transaction should be marked as delegated")
	}

	if tpm.GetDelegatedTransactionsCount() != 1 {
		t.Errorf("Expected 1 delegated transaction, got %d", tpm.GetDelegatedTransactionsCount())
	}

	// Test unmark delegation
	tpm.UnmarkAsDelegated(testTx.TransactionHash)

	if tpm.IsDelegated(testTx.TransactionHash) {
		t.Error("Transaction should not be marked as delegated after unmarking")
	}

	// Test removing from PAS pool
	tpm.RemoveFromPASPool(testTx.TransactionHash)

	if tpm.GetPASPoolSize() != 0 {
		t.Errorf("Expected PAS pool size 0, got %d", tpm.GetPASPoolSize())
	}
}

// TestPoAuDBlockProposal tests the PoAu-D block proposal functionality
func TestPoAuDBlockProposal(t *testing.T) {
	// Create a test blockchain
	bc, err := CreateTestBlockchain()
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	defer CleanupTestBlockchain(bc)

	// Enable PoAu-D
	bc.PoAuDEnabled = true

	// Create a test proposer address
	proposerAddress := "test_proposer"

	// Test 1: Non-NAP with no transactions should fail
	_, err = bc.ProposePoAuDBlock(proposerAddress)
	if err == nil {
		t.Error("Expected error when non-NAP has no transactions")
	}

	// Test 2: Add proposer as NAP
	err = bc.AddNetworkAuthor(proposerAddress)
	if err != nil {
		t.Fatalf("Failed to add proposer as network author: %v", err)
	}

	// Test 3: NAP with no transactions should return nil (no block to propose)
	block, err := bc.ProposePoAuDBlock(proposerAddress)
	if err != nil {
		t.Errorf("NAP should be able to propose even with no transactions: %v", err)
	}
	if block != nil {
		t.Error("Expected nil block when no transactions available")
	}

	// Test 4: Add a transaction and try again
	testTx := CreateTestMCPTransaction()
	bc.TransactionPool = append(bc.TransactionPool, testTx)

	block, err = bc.ProposePoAuDBlock(proposerAddress)
	if err != nil {
		t.Errorf("Failed to propose PoAu-D block: %v", err)
	}
	if block == nil {
		t.Error("Expected a valid block to be proposed")
	}

	// Verify block properties
	if block.ProposerAddress != proposerAddress {
		t.Errorf("Expected proposer address %s, got %s", proposerAddress, block.ProposerAddress)
	}

	if len(block.Transactions) < 1 {
		t.Error("Block should contain at least the mining reward transaction")
	}
}

// TestDelegationValidation tests transaction delegation validation
func TestDelegationValidation(t *testing.T) {
	// Test valid MCP transaction
	validTx := CreateTestMCPTransaction()
	if !ValidateTransactionForDelegation(validTx) {
		t.Error("Valid MCP transaction should pass delegation validation")
	}

	// Test invalid transaction type
	invalidTx := &Transaction{
		Type: "INVALID_TYPE",
		Data: []byte("{}"),
	}
	if ValidateTransactionForDelegation(invalidTx) {
		t.Error("Invalid transaction type should fail delegation validation")
	}

	// Test transaction with invalid data
	invalidDataTx := &Transaction{
		Type: TransactionTypeMCPInvokeCapability,
		Data: []byte("invalid json"),
	}
	if ValidateTransactionForDelegation(invalidDataTx) {
		t.Error("Transaction with invalid data should fail delegation validation")
	}
}

// TestPoAuDConfiguration tests PoAu-D configuration management
func TestPoAuDConfiguration(t *testing.T) {
	// Create a test blockchain
	bc, err := CreateTestBlockchain()
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	defer CleanupTestBlockchain(bc)

	// Test initial state (should be disabled by default)
	if bc.PoAuDEnabled {
		t.Error("PoAu-D should be disabled by default")
	}

	// Test enabling PoAu-D
	err = EnableDelegation(bc)
	if err != nil {
		t.Errorf("Failed to enable PoAu-D: %v", err)
	}

	if !bc.PoAuDEnabled {
		t.Error("PoAu-D should be enabled after calling EnableDelegation")
	}

	// Test disabling PoAu-D
	err = DisableDelegation(bc)
	if err != nil {
		t.Errorf("Failed to disable PoAu-D: %v", err)
	}

	if bc.PoAuDEnabled {
		t.Error("PoAu-D should be disabled after calling DisableDelegation")
	}
}

// Helper functions for testing

// CreateTestBlockchain creates a blockchain for testing
func CreateTestBlockchain() (*BlockchainStruct, error) {
	// Create a temporary database
	db, err := NewLevelDB("test_db_" + fmt.Sprintf("%d", time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}

	// Create blockchain with test configuration
	bc := &BlockchainStruct{
		TransactionPool: make([]*Transaction, 0),
		Blocks:          make([]*Block, 0),
		ChainAddress:    "test_chain",
		ChainID:         "test_chain_id",
		NetworkAuthors:  make(map[string]bool),
		PoAuDEnabled:    false,
		Reflections:     make(map[string]bool),
		db:              db,
		mcpProcessor:    NewMCPProcessor(db),
	}

	// Initialize transaction pool manager
	bc.TransactionPoolManager = NewTransactionPoolManager(bc)

	// Create genesis block
	genesisBlock := NewBlock([]byte{}, 0, 1)
	genesisBlock.BlockHash = genesisBlock.Hash()
	bc.Blocks = append(bc.Blocks, genesisBlock)

	return bc, nil
}

// CleanupTestBlockchain cleans up test blockchain resources
func CleanupTestBlockchain(bc *BlockchainStruct) {
	if bc.db != nil {
		bc.db.Close()
	}
}

// CreateTestMCPTransaction creates a test MCP transaction
func CreateTestMCPTransaction() *Transaction {
	mcpData := types.MCPInvokeCapabilityData{
		ContextRecord: types.ContextRecord{
			CapabilityID: "test_capability_id",
		},
	}

	data, _ := json.Marshal(mcpData)

	tx := &Transaction{
		Type:            TransactionTypeMCPInvokeCapability,
		From:            "test_from",
		To:              "test_to",
		Value:           0,
		Data:            data,
		Status:          TXN_VERIFICATION_SUCCESS,
		TransactionHash: "test_tx_hash_" + fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	return tx
}

// TestPoAuDIntegration tests the complete PoAu-D workflow
func TestPoAuDIntegration(t *testing.T) {
	// Create test blockchain
	bc, err := CreateTestBlockchain()
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	defer CleanupTestBlockchain(bc)

	// Enable PoAu-D
	bc.PoAuDEnabled = true

	// Create test addresses
	napAddress := "nap_address"
	papAddress := "pap_address"

	// Add NAP to network authors
	err = bc.AddNetworkAuthor(napAddress)
	if err != nil {
		t.Fatalf("Failed to add NAP: %v", err)
	}

	// Create test capability owned by PAP
	testCapability := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:    "test_capability",
			Owner: papAddress,
		},
	}

	// Store capability in database
	err = bc.mcpProcessor.db.SaveCapability(testCapability)
	if err != nil {
		t.Fatalf("Failed to store test capability: %v", err)
	}

	// Create MCP transaction that invokes the capability
	mcpData := types.MCPInvokeCapabilityData{
		ContextRecord: types.ContextRecord{
			CapabilityID: "test_capability",
		},
	}
	data, _ := json.Marshal(mcpData)

	testTx := &Transaction{
		Type:            TransactionTypeMCPInvokeCapability,
		From:            "invoker_address",
		To:              papAddress,
		Value:           0,
		Data:            data,
		Status:          TXN_VERIFICATION_SUCCESS,
		TransactionHash: "integration_test_tx",
	}

	// Add transaction to main pool
	bc.TransactionPool = append(bc.TransactionPool, testTx)

	// Test delegation logic (simulated)
	// In a real scenario, this would be handled by the delegator
	if ValidateTransactionForDelegation(testTx) {
		// Transaction should be delegated to PAP
		bc.TransactionPoolManager.AddToPASPool(testTx)
		bc.TransactionPoolManager.MarkAsDelegated(testTx.TransactionHash)
	}

	// Verify transaction is in PAS pool
	if !bc.TransactionPoolManager.IsTransactionInPASPool(testTx.TransactionHash) {
		t.Error("Transaction should be in PAS pool after delegation")
	}

	// Test PAP block proposal
	papBlock, err := bc.ProposePoAuDBlock(papAddress)
	if err != nil {
		t.Errorf("PAP should be able to propose block with delegated transactions: %v", err)
	}

	if papBlock == nil {
		t.Error("PAP should propose a block with delegated transactions")
	}

	// Test NAP block proposal
	napBlock, err := bc.ProposePoAuDBlock(napAddress)
	if err != nil {
		t.Errorf("NAP should be able to propose block: %v", err)
	}

	// NAP can propose blocks with any transactions
	if napBlock == nil && len(bc.TransactionPool) > 0 {
		t.Error("NAP should propose a block when transactions are available")
	}

	t.Logf("Integration test completed successfully")
}

// TestStaleTransactionReclaim tests the stale transaction reclaim functionality
func TestStaleTransactionReclaim(t *testing.T) {
	bc, err := CreateTestBlockchain()
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	defer CleanupTestBlockchain(bc)

	tpm := bc.TransactionPoolManager

	// Create test transaction
	testTx := CreateTestMCPTransaction()

	// Mark as delegated
	tpm.MarkAsDelegated(testTx.TransactionHash)

	// Verify it's delegated
	if !tpm.IsDelegated(testTx.TransactionHash) {
		t.Error("Transaction should be marked as delegated")
	}

	// Test reclaim with very short stale time (should reclaim immediately)
	tpm.ReclaimStaleTransactions(1 * time.Nanosecond)

	// Transaction should no longer be marked as delegated
	if tpm.IsDelegated(testTx.TransactionHash) {
		t.Error("Stale transaction should have been reclaimed")
	}
}

// TestPoolStats tests the pool statistics functionality
func TestPoolStats(t *testing.T) {
	bc, err := CreateTestBlockchain()
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	defer CleanupTestBlockchain(bc)

	tpm := bc.TransactionPoolManager

	// Get initial stats
	stats := tpm.GetPoolStats()
	if stats["main_pool_size"].(int) != 0 {
		t.Error("Main pool should be empty initially")
	}
	if stats["pas_pool_size"].(int) != 0 {
		t.Error("PAS pool should be empty initially")
	}

	// Add transactions
	testTx1 := CreateTestMCPTransaction()
	testTx2 := CreateTestMCPTransaction()

	bc.TransactionPool = append(bc.TransactionPool, testTx1)
	tpm.AddToPASPool(testTx2)
	tpm.MarkAsDelegated(testTx2.TransactionHash)

	// Get updated stats
	stats = tpm.GetPoolStats()
	if stats["main_pool_size"].(int) != 1 {
		t.Errorf("Expected main pool size 1, got %d", stats["main_pool_size"])
	}
	if stats["pas_pool_size"].(int) != 1 {
		t.Errorf("Expected PAS pool size 1, got %d", stats["pas_pool_size"])
	}
	if stats["delegated_transactions"].(int) != 1 {
		t.Errorf("Expected 1 delegated transaction, got %d", stats["delegated_transactions"])
	}
}
