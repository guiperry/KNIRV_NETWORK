package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/utils"
)

// Use real ConsensusManager for testing
func newTestConsensusManager(bc *BlockchainStruct) *ConsensusManager {
	return &ConsensusManager{
		syncState:      make(chan bool),
		LongestChain:   []*Block{},
		Blockchain:     bc,
		MiningLocked:   false,
		UpdateRequired: false,
		reflectionURLs: make(map[string]bool),   // Initialize empty map for tests
		selfURL:        "http://localhost:test", // Test URL
		// mu is a value type (sync.Mutex), so it's automatically initialized to its zero value
	}
}

func TestTransactionFlow(t *testing.T) {
	// Create test DB connection with unique test path
	testDBPath := fmt.Sprintf("test_db/transaction_test_%d.db", time.Now().UnixNano())
	db, err := NewLevelDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("Warning: error closing test database: %v", err)
		}
		// Clean up test database files
		if err := os.RemoveAll(filepath.Dir(testDBPath)); err != nil {
			t.Logf("Warning: error cleaning up test database files: %v", err)
		}
	}()

	// Create test wallets
	senderWallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create sender wallet: %v", err)
	}
	receiverWallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create receiver wallet: %v", err)
	}

	// Create genesis block - NO NEED TO MINE IT HERE
	genesisBlock := NewBlock(nil, 0, 0)
	// genesisBlock.MineBlock() // <--- REMOVE THIS LINE

	// Create blockchain with test chain ID
	chainID := fmt.Sprintf("test_chain_%d", time.Now().UnixNano())
	// Provide a dummy CerebrasConfig for testing
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	// Pass nil for ChromemManager in tests - it will create a new one internally
	bc, err := NewBlockchain(genesisBlock, chainID, senderWallet.GetAddress(), db, nil, "/tmp/test_searchable_db", dummyCerebrasConfig) // Use sender's address as a placeholder miner
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}

	// --- Fund the sender wallet (Requires mining to work) ---
	// Add a temporary funding transaction directly for the test setup
	// This simulates the faucet but ensures funds exist before the main test txn
	fundingAmount := uint64(10000 * utils.DECIMAL) // Give sender enough funds
	fundingTxn := NewTransaction(utils.BLOCKCHAIN_ADDRESS, senderWallet.GetAddress(), fundingAmount, []byte("test setup funding"))

	// Create a temporary wallet for the blockchain address to sign the funding transaction
	blockchainWallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create blockchain wallet: %v", err)
	}

	// Sign the funding transaction with the blockchain wallet
	signedFundingTxn, err := blockchainWallet.GetSignedTxn(*fundingTxn)
	if err != nil {
		t.Fatalf("Failed to sign funding transaction: %v", err)
	}

	// Mark as verified since it's properly signed
	signedFundingTxn.Status = TXN_VERIFICATION_SUCCESS

	// Add funding txn to pool and signal miner
	bc.addVerifiedTxnToPoolAndSignal(signedFundingTxn)
	t.Logf("Added initial funding transaction %s for sender %s", signedFundingTxn.TransactionHash, senderWallet.GetAddress())

	// --- Start mining to process the funding transaction ---
	cm := newTestConsensusManager(bc)
	bc.Lock()
	bc.StopMining = false // Ensure mining is not stopped initially
	bc.Unlock()
	miningDone := make(chan struct{}) // Channel to signal mining completion
	go func() {
		bc.ProofOfWorkMining(context.Background(), receiverWallet.GetAddress(), cm)
		close(miningDone) // Signal when mining loop exits (e.g., on StopMining=true)
	}()

	// --- Wait for the funding transaction to be mined ---
	t.Log("Waiting for initial funding transaction to be mined...")
	fundingTimeout := time.After(15 * time.Second) // Adjust timeout as needed
	fundingTicker := time.NewTicker(500 * time.Millisecond)
	defer fundingTicker.Stop()
	fundingMined := false
	for !fundingMined {
		select {
		case <-fundingTicker.C:
			bc.Lock()               // Lock for safe access
			if len(bc.Blocks) > 1 { // Check if at least one block (after genesis) exists
				// Check if the funding transaction is in the latest block
				latestBlock := bc.Blocks[len(bc.Blocks)-1]
				for _, txn := range latestBlock.Transactions {
					if txn.TransactionHash == signedFundingTxn.TransactionHash {
						fundingMined = true
						t.Logf("Funding transaction %s mined in block %d", signedFundingTxn.TransactionHash, latestBlock.BlockNumber)
						break
					}
				}
			}
			bc.Unlock() // Unlock after checking
		case <-fundingTimeout:
			bc.Lock()
			bc.StopMining = true // Stop the miner on timeout
			bc.Unlock()
			<-miningDone // Wait for miner goroutine to finish
			t.Fatalf("Timeout waiting for initial funding transaction %s to be mined", signedFundingTxn.TransactionHash)
		}
	}
	// --- Funding transaction mined ---

	// --- Now proceed with the actual test transaction ---
	t.Log("Creating and signing the main test transaction...")
	// Create and sign a transaction
	txn := NewTransaction(
		senderWallet.GetAddress(),
		receiverWallet.GetAddress(),
		1000, // 10.00 nrn (DECIMAL=100)
		[]byte("test transaction"),
	)

	signedTxn, err := senderWallet.GetSignedTxn(*txn)
	if err != nil {
		bc.StopMining = true
		<-miningDone
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Add transaction to pool (miner is already running)
	t.Logf("Adding main test transaction %s to pool...", signedTxn.TransactionHash)
	bc.AddTransactionToTransactionPool(signedTxn)

	// Verify transaction is in pool
	foundInPool := false
	bc.Lock()
	for _, poolTxn := range bc.TransactionPool {
		if poolTxn.TransactionHash == signedTxn.TransactionHash {
			foundInPool = true
			if poolTxn.Status != TXN_VERIFICATION_SUCCESS {
				bc.Unlock()
				bc.StopMining = true
				<-miningDone
				t.Fatalf("Transaction %s not marked as verified in pool", signedTxn.TransactionHash)
			}
			break
		}
	}
	bc.Unlock()
	if !foundInPool {
		// It might take a moment to appear, add a small wait/retry if needed, but fail if consistently absent
		time.Sleep(100 * time.Millisecond) // Small delay
		bc.Lock()
		for _, poolTxn := range bc.TransactionPool {
			if poolTxn.TransactionHash == signedTxn.TransactionHash {
				foundInPool = true
				break
			}
		}
		bc.Unlock()
		if !foundInPool {
			bc.StopMining = true
			<-miningDone
			t.Fatalf("Transaction %s not found in pool", signedTxn.TransactionHash)
		}
	}
	t.Logf("Transaction %s found in pool and verified.", signedTxn.TransactionHash)

	// Wait for the main transaction block to be mined or timeout
	t.Logf("Waiting for main transaction %s to be mined...", signedTxn.TransactionHash)
	timeout := time.After(10 * time.Second) // Timeout for the main transaction
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	minedBlockIndex := -1
	expectedBlockCount := 3 // Genesis + Funding Block + Main Txn Block

	for minedBlockIndex == -1 {
		select {
		case <-ticker.C:
			bc.Lock()
			currentBlockCount := len(bc.Blocks)
			if currentBlockCount >= expectedBlockCount {
				// Check the latest block(s) for the transaction
				for i := currentBlockCount - 1; i > 0; i-- { // Check recent blocks (skip genesis)
					block := bc.Blocks[i]
					for _, blockTxn := range block.Transactions {
						if blockTxn.TransactionHash == signedTxn.TransactionHash {
							minedBlockIndex = i
							t.Logf("Main transaction %s found in block %d", signedTxn.TransactionHash, block.BlockNumber)
							break
						}
					}
					if minedBlockIndex != -1 {
						break
					}
				}
			}
			bc.Unlock()
		case <-timeout:
			bc.Lock()
			bc.StopMining = true // Stop the miner on timeout
			bc.Unlock()
			<-miningDone // Wait for miner goroutine to finish
			t.Fatalf("Mining timeout reached waiting for transaction %s", signedTxn.TransactionHash)
		}
	}

	// Stop mining now that we've found the block
	bc.Lock()
	bc.StopMining = true
	bc.Unlock()
	<-miningDone // Wait for miner goroutine to exit cleanly

	//verification: // Keep label for clarity, though goto is removed
	// Verify block count
	bc.Lock()
	finalBlockCount := len(bc.Blocks)
	bc.Unlock()
	if finalBlockCount < expectedBlockCount { // Use >= check if more blocks could be mined
		t.Fatalf("Expected at least %d blocks, got %d", expectedBlockCount, finalBlockCount)
	}
	if minedBlockIndex == -1 {
		t.Fatalf("Test logic error: Transaction was marked found but block index wasn't set.")
	}

	// Verify transaction is in the correct block and status is SUCCESS
	bc.Lock()
	minedBlock := bc.Blocks[minedBlockIndex]
	txnFoundInBlock := false
	for _, blockTxn := range minedBlock.Transactions {
		if blockTxn.TransactionHash == signedTxn.TransactionHash {
			txnFoundInBlock = true
			if blockTxn.Status != utils.SUCCESS {
				t.Errorf("Transaction status not marked as SUCCESS in block, got %s", blockTxn.Status)
			}
			break
		}
	}
	bc.Unlock()
	if !txnFoundInBlock {
		t.Errorf("Transaction %s not found in expected mined block %d", signedTxn.TransactionHash, minedBlock.BlockNumber)
	}

	// Verify block is valid
	bc.Lock()
	if !minedBlock.VerifyBlock() {
		t.Errorf("Mined block %d failed verification", minedBlock.BlockNumber)
	}
	bc.Unlock()

	t.Log("TestTransactionFlow completed successfully.")

	// Cleanup test chain (already deferred)
	// db.Client.Delete([]byte(chainID), nil) // Be careful with direct DB manipulation in tests
}
