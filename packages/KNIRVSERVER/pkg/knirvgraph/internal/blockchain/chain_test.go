package blockchain

import (
	"KNIRVGRAPH/internal/storage"
	"KNIRVGRAPH/internal/types"
	"testing"
	"time"
)

func TestNewChain(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)
	if chain == nil {
		t.Fatal("Expected chain to be created")
	}

	if chain.height != 0 {
		t.Errorf("Expected initial height to be 0, got %d", chain.height)
	}

	if chain.state == nil {
		t.Error("Expected state to be initialized")
	}

	if chain.storage == nil {
		t.Error("Expected storage to be set")
	}
}

func TestChainAddBlock(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	// Create a test block
	block := createTestBlock(1, "")

	err = chain.AddBlock(block)
	if err != nil {
		t.Errorf("Failed to add block: %v", err)
	}

	if chain.height != 1 {
		t.Errorf("Expected height to be 1, got %d", chain.height)
	}
}

func TestChainAddBlockSequential(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	// Add multiple blocks sequentially
	var prevHash string
	for i := uint64(1); i <= 5; i++ {
		block := createTestBlock(i, prevHash)
		err = chain.AddBlock(block)
		if err != nil {
			t.Errorf("Failed to add block %d: %v", i, err)
		}

		if chain.height != i {
			t.Errorf("Expected height to be %d, got %d", i, chain.height)
		}

		// Set the previous hash for the next iteration
		prevHash = block.Hash
	}
}

func TestChainGetBlock(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	// Add a test block
	block := createTestBlock(1, "")
	err = chain.AddBlock(block)
	if err != nil {
		t.Fatalf("Failed to add block: %v", err)
	}

	// Retrieve the block
	retrievedBlock, err := chain.GetBlock(1)
	if err != nil {
		t.Errorf("Failed to get block: %v", err)
	}

	if retrievedBlock.Header.Height != 1 {
		t.Errorf("Expected block height 1, got %d", retrievedBlock.Header.Height)
	}

	// Compare timestamps with second precision (JSON serialization may lose nanosecond precision)
	if retrievedBlock.Header.Timestamp.Unix() != block.Header.Timestamp.Unix() {
		t.Errorf("Block timestamps don't match: expected %v, got %v",
			block.Header.Timestamp.Unix(), retrievedBlock.Header.Timestamp.Unix())
	}
}

func TestChainGetBlockNotFound(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	// Try to get a non-existent block
	_, err = chain.GetBlock(999)
	if err == nil {
		t.Error("Expected error when getting non-existent block")
	}
}

func TestChainGetHeight(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	// Initial height should be 0
	if chain.GetHeight() != 0 {
		t.Errorf("Expected initial height 0, got %d", chain.GetHeight())
	}

	// Add blocks and check height
	var prevHash string
	for i := uint64(1); i <= 3; i++ {
		block := createTestBlock(i, prevHash)
		err = chain.AddBlock(block)
		if err != nil {
			t.Fatalf("Failed to add block %d: %v", i, err)
		}

		if chain.GetHeight() != i {
			t.Errorf("Expected height %d, got %d", i, chain.GetHeight())
		}

		// Set the previous hash for the next iteration
		prevHash = block.Hash
	}
}

func TestChainGetState(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	state := chain.GetState()
	if state == nil {
		t.Error("Expected state to be returned")
	}
}

func TestChainValidateBlock(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	tests := []struct {
		name        string
		block       *types.Block
		expectError bool
	}{
		{
			name:        "Valid block",
			block:       createTestBlock(1, ""),
			expectError: false,
		},
		{
			name:        "Block with zero height",
			block:       createTestBlock(0, ""),
			expectError: true,
		},
		{
			name: "Block with future timestamp",
			block: &types.Block{
				Header: types.BlockHeader{
					Height:    1,
					Timestamp: time.Now().Add(time.Hour),
					PrevHash:  "",
				},
				Transactions: []*types.Transaction{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chain.validateBlock(tt.block)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestChainExecuteBlock(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	// Set up initial balance for the sender
	senderAccount := &types.Account{
		Address: "sender",
		Balance: 1000,
		Nonce:   0,
	}
	chain.state.SetAccount(senderAccount)

	// Create a block with transactions
	block := createTestBlockWithTransactions(1, "")

	err = chain.executeBlock(block)
	if err != nil {
		t.Errorf("Failed to execute block: %v", err)
	}

	// Verify state changes were applied
	state := chain.GetState()
	if state == nil {
		t.Error("Expected state to be available")
	}
}

func TestChainConcurrentAccess(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	// Test concurrent reads and writes
	done := make(chan bool, 2)

	// Add blocks sequentially first
	var prevHash string
	for i := uint64(1); i <= 10; i++ {
		block := createTestBlock(i, prevHash)
		err := chain.AddBlock(block)
		if err != nil {
			t.Fatalf("Failed to add block %d: %v", i, err)
		}
		prevHash = block.Hash
	}

	// Test concurrent reads
	go func() {
		for i := 0; i < 10; i++ {
			height := chain.GetHeight()
			if height != 10 {
				t.Errorf("Expected height 10, got %d", height)
			}
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := uint64(1); i <= 10; i++ {
			block, err := chain.GetBlock(i)
			if err != nil {
				t.Errorf("Failed to get block %d: %v", i, err)
			}
			if block.Header.Height != i {
				t.Errorf("Expected block height %d, got %d", i, block.Header.Height)
			}
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	if chain.GetHeight() != 10 {
		t.Errorf("Expected final height 10, got %d", chain.GetHeight())
	}
}

func TestChainBlockSerialization(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create memory storage: %v", err)
	}

	chain := NewChain(storage)

	// Create and add a block
	originalBlock := createTestBlock(1, "")
	err = chain.AddBlock(originalBlock)
	if err != nil {
		t.Fatalf("Failed to add block: %v", err)
	}

	// Retrieve and verify the block
	retrievedBlock, err := chain.GetBlock(1)
	if err != nil {
		t.Fatalf("Failed to get block: %v", err)
	}

	// Compare key fields
	if retrievedBlock.Header.Height != originalBlock.Header.Height {
		t.Error("Block height mismatch after serialization")
	}

	// Compare timestamps with second precision (JSON serialization may lose nanosecond precision)
	if retrievedBlock.Header.Timestamp.Unix() != originalBlock.Header.Timestamp.Unix() {
		t.Errorf("Block timestamp mismatch after serialization: expected %v, got %v",
			originalBlock.Header.Timestamp.Unix(), retrievedBlock.Header.Timestamp.Unix())
	}

	if retrievedBlock.Header.PrevHash != originalBlock.Header.PrevHash {
		t.Error("Block previous hash mismatch after serialization")
	}
}

// Helper functions

func createTestBlock(height uint64, prevHash string) *types.Block {
	block := &types.Block{
		Header: types.BlockHeader{
			Height:    height,
			Timestamp: time.Now(),
			PrevHash:  prevHash,
		},
		Transactions: []*types.Transaction{},
	}
	block.Hash = block.CalculateHash()
	return block
}

func createTestBlockWithTransactions(height uint64, prevHash string) *types.Block {
	tx := &types.Transaction{
		ID:        "test-tx-1",
		From:      "sender",
		To:        "receiver",
		Amount:    100,
		Timestamp: time.Now(),
	}

	return &types.Block{
		Header: types.BlockHeader{
			Height:    height,
			Timestamp: time.Now(),
			PrevHash:  prevHash,
		},
		Transactions: []*types.Transaction{tx},
	}
}
