

---

**Source**: KNIRVROOT/docs/completedImplementations/update_tests.md

Core Assumption for Test Updates:

The following test modifications assume that your ChromemManager.OnNewBlockConfirmed method (likely in sync_manager.go or chromemDB_manager.go) has been or will be enhanced to:

Index the transaction itself into the transactionCollection (which it currently does).
Parse transaction data (tx.Data) based on tx.Type.
For TransactionTypeMCPRegisterCapability, prepare and add the capability descriptor to the capabilityDescriptorCollection and a corresponding context record to contextRecordCollection.
For TransactionTypeMCPInvokeCapability, prepare and add the ContextRecord from the transaction data to the contextRecordCollection.
For TransactionTypeMCPUpdateCapability, prepare and update/add the capability descriptor in capabilityDescriptorCollection and add a corresponding context record to contextRecordCollection.
If ChromemManager.OnNewBlockConfirmed only indexes transactions, then the tests would only verify the transactionCollection, and checks for other collections would fail or be irrelevant.

General Changes to Tests:

ChromemDB Path Management: Tests creating BlockchainStruct instances (which initialize ChromemManager) will need to manage the searchableDBPath for ChromemDB. This path should be unique per test or cleaned up to avoid interference.
Waiting for ChromemDB Sync: Since ChromemManager.OnNewBlockConfirmed is called in a goroutine within BlockchainStruct.AddBlock, tests will need a mechanism (e.g., a short delay or a polling helper) to wait for ChromemDB to be updated before verification.
ChromemDB Verification: After blockchain operations, tests will query the relevant ChromemDB collections (transactionCollection, capabilityDescriptorCollection, contextRecordCollection) using the ChromemManager instance (bc.ChromemSync).
We'll use collection Get methods with known IDs or Query methods with filters.
Verification will involve comparing the retrieved document content and metadata with what's expected from the Prepare...ForChromem... functions in chromemDB_conversion.go.
Let's add a helper function for waiting for ChromemDB updates. This can be placed in a test utility file or directly in the test files if preferred.

go
// Helper for mcp_blockchain_test.go or a common test utility file
package main

import (
	"context"
	"fmt"
	"time"
	// ... other necessary imports from your project ...
)

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

// Helper to convert metadata map[string]interface{} to map[string]string for ChromemDB
// This should match the logic in your ChromemManager or conversion utilities
func stringifyMetadataTestHelper(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		result[k] = fmt.Sprintf("%v", v) // Simplified conversion for test comparison
	}
	return result
}
Now, let's update the specific test files:

1. /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN/mcp_blockchain_test.go
This file tests the processing of MCP transactions, capability invocation, and updates. We'll add ChromemDB verification steps.

mcp_blockchain_test.go
-10
+166
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"os"
	"testing"
	"time"

	"KNIRVCHAIN/utils"
)

// WaitForChromemDBTest is a local helper for this test file.
// In a real project, this might be in a shared test utility.
func WaitForChromemDBTest(timeout time.Duration, checkFunc func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond) // Poll every 250ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ChromemDB update: %w", ctx.Err())
		case <-ticker.C:
			done, err := checkFunc()
			if err != nil {
				// Log transient errors during polling but don't fail immediately
				// log.Printf("Polling ChromemDB (transient error): %v", err)
			}
			if done {
				return nil
			}
		}
	}
}

// TestMCPTransactionProcessing tests the processing of MCP transactions
func TestMCPTransactionProcessing(t *testing.T) {
	// Create a test database
	dbPath := fmt.Sprintf("testdb_%d", time.Now().UnixNano())
	searchableDBPath := fmt.Sprintf("test_chromem_mcp_proc_%d", time.Now().UnixNano())
	defer os.RemoveAll(dbPath)
	defer os.RemoveAll(searchableDBPath)

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create a genesis block
	genesisBlock := NewBlock(nil, 0, 0)

	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_processing", db, "/tmp/test_searchable_db", dummyCerebrasConfig)
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_processing", db, searchableDBPath, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	// Create a test wallet
	wallet, _ := NewWallet()
	from := wallet.GetAddress()

	// Register test capability directly in the database to ensure it exists
	err = registerTestCapabilityWithOwner(db, "resource-123", from)
	if err != nil {
		t.Fatalf("Failed to register test capability: %v", err)
	}
	registrationNetworkFee := uint64(10) // Network fee for registration
	capabilityGasFee := uint64(50)       // GasFeeNRN for the capability itself

	// Set the ProposerAddress BEFORE mining the block to ensure it's included in the hash
	newBlock.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer for network fees
	newBlock.MineBlock()

	blockNumberForChromem := newBlock.BlockNumber
	blockTimestampForChromem := newBlock.Timestamp
	// Add the registration block to the blockchain
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	bc.AddBlock(db, newBlock)

	// Verify that the capability was processed and stored
			expectedResourceType,
			actualResourceType)
	}

	// --- ChromemDB Verification ---
	t.Log("Verifying capability registration in ChromemDB...")
	err = WaitForChromemDBTest(10*time.Second, func() (bool, error) {
		// 1. Verify CapabilityDescriptor in capabilityDescriptorCollection
		capCollection, err := bc.ChromemSync.client.GetCollection("capability_descriptors")
		if err != nil {
			return false, fmt.Errorf("failed to get capability_descriptors collection: %w", err)
		}
		results, err := capCollection.Get(context.Background(), []string{resourceDesc.ID}, nil, nil)
		if err != nil {
			return false, fmt.Errorf("failed to get capability %s from ChromemDB: %w", resourceDesc.ID, err)
		}
		if len(results) == 0 {
			return false, fmt.Errorf("capability %s not found in ChromemDB capability_descriptors collection", resourceDesc.ID)
		}
		// Further checks on results[0].Content and results[0].Metadata can be added here
		// based on PrepareCapabilityDescriptorForChromemFromRegisterLocal
		t.Logf("Capability %s found in ChromemDB capability_descriptors collection.", resourceDesc.ID)

		// 2. Verify ContextRecord for registration in contextRecordCollection
		// The ID for this context record would be the registration transaction's hash.
		ctxCollection, err := bc.ChromemSync.client.GetCollection("context_records")
		if err != nil {
			return false, fmt.Errorf("failed to get context_records collection: %w", err)
		}
		// Assuming the context record ID for registration is the transaction hash
		regCtxRecordID := txn.TransactionHash
		ctxResults, err := ctxCollection.Get(context.Background(), []string{regCtxRecordID}, nil, nil)
		if err != nil {
			return false, fmt.Errorf("failed to get registration context record %s from ChromemDB: %w", regCtxRecordID, err)
		}
		if len(ctxResults) == 0 {
			return false, fmt.Errorf("registration context record %s not found in ChromemDB", regCtxRecordID)
		}
		t.Logf("Registration ContextRecord %s found in ChromemDB.", regCtxRecordID)

		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for capability registration failed: %v", err)
	}
}

// capabilityTypeToProto converts domain CapabilityType to protobuf enum

	// Setup test database
	dbPath := fmt.Sprintf("testdb_invoke_%d", time.Now().UnixNano())
	searchableDBPathInvoke := fmt.Sprintf("test_chromem_invoke_%d", time.Now().UnixNano())
	defer os.RemoveAll(dbPath)
	defer os.RemoveAll(searchableDBPathInvoke)

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_cap_invoke", db, "/tmp/test_searchable_db", dummyCerebrasConfig)
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_cap_invoke", db, searchableDBPathInvoke, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	newBlock2.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer
	newBlock2.MineBlock()

	invokeBlockNumberForChromem := newBlock2.BlockNumber
	invokeBlockTimestampForChromem := newBlock2.Timestamp

	// Add the invocation block to the blockchain (this will trigger fee transfers)
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	bc.AddBlock(db, newBlock2)

	// --- Verify balances after invocation ---
	// Note: Proposer also gets mining reward from newBlock2, which is not explicitly checked here but happens.
	// We are primarily focused on the fee transfer from the invokeTxn.
	t.Logf("After Invoke - Initiator/Owner (%s) Balance: %d, Proposer (%s) Balance: %d", from, finalInitiatorBalance, utils.BLOCKCHAIN_ADDRESS, finalProposerBalance)

	// --- ChromemDB Verification for Invocation ContextRecord ---
	t.Log("Verifying capability invocation context record in ChromemDB...")
	err = WaitForChromemDBTest(10*time.Second, func() (bool, error) {
		ctxCollection, err := bc.ChromemSync.client.GetCollection("context_records")
		if err != nil {
			return false, fmt.Errorf("failed to get context_records collection: %w", err)
		}
		// The ID for the context record is the transaction hash
		results, err := ctxCollection.Get(context.Background(), []string{invokeTxn.TransactionHash}, nil, nil)
		if err != nil {
			return false, fmt.Errorf("failed to get context record %s from ChromemDB: %w", invokeTxn.TransactionHash, err)
		}
		if len(results) == 0 {
			return false, fmt.Errorf("context record %s not found in ChromemDB context_records collection", invokeTxn.TransactionHash)
		}
		// Further checks on results[0].Content and results[0].Metadata can be added here
		// based on PrepareContextRecordForChromemEnhanced
		t.Logf("ContextRecord %s for invocation found in ChromemDB.", invokeTxn.TransactionHash)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for invocation context record failed: %v", err)
	}
}

// TestMCPCapabilityUpdate tests the update of an MCP capability
func TestMCPCapabilityUpdate(t *testing.T) {
	// Create a test database
	dbPath := fmt.Sprintf("testdb_%d", time.Now().UnixNano())
	searchableDBPathUpdate := fmt.Sprintf("test_chromem_update_%d", time.Now().UnixNano())
	defer os.RemoveAll(dbPath)
	defer os.RemoveAll(searchableDBPathUpdate)

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_update", db, "/tmp/test_searchable_db", dummyCerebrasConfig)
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_update", db, searchableDBPathUpdate, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	newBlock2.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer for network fees
	newBlock2.MineBlock()

	updateBlockNumberForChromem := newBlock2.BlockNumber
	updateBlockTimestampForChromem := newBlock2.Timestamp

	// Add the block to the blockchain
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	bc.AddBlock(db, newBlock2)

	// Verify that the capability was updated
	if resourceDescFromDB.BaseDescriptor.GasFeeNrn != updatedResourceDesc.GasFeeNRN {
		t.Errorf("ResourceDescriptor GasFeeNRN mismatch: expected %d, got %d", updatedResourceDesc.GasFeeNRN, resourceDescFromDB.BaseDescriptor.GasFeeNrn)
	}

	// --- ChromemDB Verification ---
	t.Log("Verifying capability update in ChromemDB...")
	err = WaitForChromemDBTest(10*time.Second, func() (bool, error) {
		// 1. Verify the update transaction is indexed
		txCollection, err := bc.ChromemSync.client.GetCollection("transactions")
		if err != nil {
			return false, fmt.Errorf("failed to get transactions collection: %w", err)
		}
		txResults, err := txCollection.Get(context.Background(), []string{updateTxn.TransactionHash}, nil, nil)
		if err != nil || len(txResults) == 0 {
			return false, fmt.Errorf("update transaction %s not found in ChromemDB: %w", updateTxn.TransactionHash, err)
		}
		t.Logf("Update transaction %s found in ChromemDB.", updateTxn.TransactionHash)

		// 2. Verify ContextRecord for update in contextRecordCollection
		// The ID for this context record would be the update transaction's hash.
		ctxCollection, err := bc.ChromemSync.client.GetCollection("context_records")
		if err != nil {
			return false, fmt.Errorf("failed to get context_records collection: %w", err)
		}
		updateCtxRecordID := updateTxn.TransactionHash
		ctxResults, err := ctxCollection.Get(context.Background(), []string{updateCtxRecordID}, nil, nil)
		if err != nil || len(ctxResults) == 0 {
			return false, fmt.Errorf("update context record %s not found in ChromemDB: %w", updateCtxRecordID, err)
		}
		t.Logf("Update ContextRecord %s found in ChromemDB.", updateCtxRecordID)

		// 3. Verify the updated CapabilityDescriptor in capabilityDescriptorCollection
		// (This assumes ChromemManager.OnNewBlockConfirmed is enhanced to handle updates to this collection)
		capCollection, err := bc.ChromemSync.client.GetCollection("capability_descriptors")
		if err != nil {
			return false, fmt.Errorf("failed to get capability_descriptors collection: %w", err)
		}
		capResults, err := capCollection.Get(context.Background(), []string{resourceDesc.ID}, nil, nil)
		if err != nil || len(capResults) == 0 {
			return false, fmt.Errorf("updated capability %s not found in ChromemDB: %w", resourceDesc.ID, err)
		}
		// TODO: Add detailed verification of capResults[0].Content and capResults[0].Metadata
		// to ensure it reflects 'updatedResourceDesc' based on PrepareCapabilityDescriptorForChromemFromUpdate.
		t.Logf("Updated capability %s found in ChromemDB capability_descriptors collection.", resourceDesc.ID)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for capability update failed: %v", err)
+	}
 }
 
 // Helper function to register a test capability with a specific owner

2. /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN/transaction_test.go
This file tests the basic transaction flow. We'll add ChromemDB verification for the transactions.

diff
-1
+75
The code change produced by Gemini could not be fully parsed. Can't automatically accept changes.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"KNIRVCHAIN/utils"
)

// WaitForChromemDBTxTest is a local helper for this test file.
func WaitForChromemDBTxTest(timeout time.Duration, checkFunc func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond) // Poll every 250ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ChromemDB update: %w", ctx.Err())
		case <-ticker.C:
			done, err := checkFunc()
			if err != nil {
				// log.Printf("Polling ChromemDB (transient error): %v", err)
			}
			if done {
				return nil
			}
		}
	}
}

// Use real ConsensusManager for testing
func newTestConsensusManager(bc *BlockchainStruct) *ConsensusManager {
	return &ConsensusManager{
func TestTransactionFlow(t *testing.T) {
	// Create test DB connection with unique test path
	testDBPath := fmt.Sprintf("test_db/transaction_test_%d.db", time.Now().UnixNano())
	searchableDBPathTx := fmt.Sprintf("test_chromem_tx_flow_%d", time.Now().UnixNano())
	db, err := NewLevelDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer func() {
		os.RemoveAll(searchableDBPathTx) // Cleanup ChromemDB path

		if err := db.Close(); err != nil {
			t.Logf("Warning: error closing test database: %v", err)
		}
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	bc, err := NewBlockchain(genesisBlock, chainID, senderWallet.GetAddress(), db, "/tmp/test_searchable_db", dummyCerebrasConfig) // Use sender's address as a placeholder miner
	bc, err := NewBlockchain(genesisBlock, chainID, senderWallet.GetAddress(), db, searchableDBPathTx, dummyCerebrasConfig) // Use sender's address as a placeholder miner
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}
	// Add a temporary funding transaction directly for the test setup
	// This simulates the faucet but ensures funds exist before the main test txn
	fundingAmount := uint64(10000 * utils.DECIMAL) // Give sender enough funds
	fundingBlockNumber := uint64(0) // Will be set after mining
	fundingBlockTimestamp := int64(0) // Will be set after mining
	fundingTxn := NewTransaction(utils.BLOCKCHAIN_ADDRESS, senderWallet.GetAddress(), fundingAmount, []byte("test setup funding"))
	fundingTxn.Status = TXN_VERIFICATION_SUCCESS // Mark as verified for pool

				for _, txn := range latestBlock.Transactions {
					if txn.TransactionHash == fundingTxn.TransactionHash {
						fundingMined = true
						fundingBlockNumber = latestBlock.BlockNumber
						fundingBlockTimestamp = latestBlock.Timestamp
						t.Logf("Funding transaction %s mined in block %d", fundingTxn.TransactionHash, latestBlock.BlockNumber)
						break
					}
	}
	// --- Funding transaction mined ---

	// --- ChromemDB Verification for Funding Transaction ---
	t.Logf("Verifying funding transaction %s in ChromemDB...", fundingTxn.TransactionHash)
	err = WaitForChromemDBTxTest(10*time.Second, func() (bool, error) {
		txCollection, errGet := bc.ChromemSync.client.GetCollection("transactions")
		if errGet != nil {
			return false, fmt.Errorf("failed to get transactions collection: %w", errGet)
		}
		results, errQuery := txCollection.Get(context.Background(), []string{fundingTxn.TransactionHash}, nil, nil)
		if errQuery != nil || len(results) == 0 {
			return false, fmt.Errorf("funding transaction %s not found in ChromemDB: %w", fundingTxn.TransactionHash, errQuery)
		}
		// TODO: Add more detailed verification of results[0].Content and results[0].Metadata
		// based on PrepareTransactionForChromem output.
		t.Logf("Funding transaction %s found in ChromemDB.", fundingTxn.TransactionHash)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for funding transaction failed: %v", err)
	}

	// --- Now proceed with the actual test transaction ---
	t.Log("Creating and signing the main test transaction...")
	// Create and sign a transaction
		1000, // 10.00 nrn (DECIMAL=100)
		[]byte("test transaction"),
	)
	mainTxBlockNumber := uint64(0)
	mainTxBlockTimestamp := int64(0)

	signedTxn, err := senderWallet.GetSignedTxn(*txn)
	if err != nil {
					for _, blockTxn := range block.Transactions {
						if blockTxn.TransactionHash == signedTxn.TransactionHash {
							minedBlockIndex = i
							mainTxBlockNumber = block.BlockNumber
							mainTxBlockTimestamp = block.Timestamp
							t.Logf("Main transaction %s found in block %d", signedTxn.TransactionHash, block.BlockNumber)
							break
						}
	}
	bc.Unlock()

	// --- ChromemDB Verification for Main Transaction ---
	t.Logf("Verifying main transaction %s in ChromemDB...", signedTxn.TransactionHash)
	err = WaitForChromemDBTxTest(10*time.Second, func() (bool, error) {
		txCollection, errGet := bc.ChromemSync.client.GetCollection("transactions")
		if errGet != nil {
			return false, fmt.Errorf("failed to get transactions collection: %w", errGet)
		}
		results, errQuery := txCollection.Get(context.Background(), []string{signedTxn.TransactionHash}, nil, nil)
		if errQuery != nil || len(results) == 0 {
			return false, fmt.Errorf("main transaction %s not found in ChromemDB: %w", signedTxn.TransactionHash, errQuery)
		}
		// TODO: Add more detailed verification of results[0].Content and results[0].Metadata
		t.Logf("Main transaction %s found in ChromemDB.", signedTxn.TransactionHash)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for main transaction failed: %v", err)
	}

	t.Log("TestTransactionFlow completed successfully.")

	// Cleanup test chain (already deferred)

Important Considerations and Potential ChromemManager Enhancements:

ChromemManager.OnNewBlockConfirmed Logic:
As highlighted, the current ChromemManager (from sync_manager.go / chromemDB_manager.go) primarily indexes the raw transaction into the transactionCollection.
For capabilityDescriptorCollection and contextRecordCollection to be populated correctly, OnNewBlockConfirmed needs to:
Identify transaction types (e.g., TransactionTypeMCPRegisterCapability, TransactionTypeMCPInvokeCapability).
Unmarshal tx.Data according to the transaction type.
Use the specific Prepare...ForChromem... functions from chromemDB_conversion.go (e.g., PrepareCapabilityDescriptorForChromemFromRegisterLocal, PrepareContextRecordForChromemEnhanced).
Add the resulting documents and metadata to their respective collections (capabilityDescriptorCollection, contextRecordCollection).
Handle updates to capabilities in capabilityDescriptorCollection (e.g., mark old versions, ensure only latest is easily queryable, or overwrite).
Error Handling in WaitForChromemDBTest: The provided helper is basic. In a production test suite, you might want more sophisticated error handling within the checkFunc or the poller itself to distinguish between "not yet found" and actual errors.
Metadata Verification: The diffs add basic checks for the existence of records in ChromemDB. You should expand these to verify the Content and Metadata of the chromem.Document against the expected values derived from the Prepare... functions. This involves comparing stringified metadata.
Test Data Uniqueness: Ensure dbPath and searchableDBPath (for ChromemDB) are unique for each test function or are thoroughly cleaned up to prevent state leakage between tests. The defer os.RemoveAll() pattern is good for this.
Idempotency: If tests are re-run, ensure ChromemDB state is clean or that operations are idempotent if the manager is designed that way.
Complexity of Prepare...ForChromem...: The conversion functions in chromemDB_conversion.go determine the structure of data in ChromemDB. Ensure these are robust and cover all necessary fields for your search and analytics use cases.
These changes should give you a good starting point for adapting your tests to include ChromemDB verification. Remember to adjust the detailed checks within the WaitForChromemDBTest's checkFunc to match the exact structure and content you expect in ChromemDB.Okay, let's get your tests updated to incorporate the new ChromemDB architecture!

It's great that you're thinking about how ChromemDB impacts your testing strategy. The key is to ensure that data not only makes it to your primary datastore (LevelDB) but is also correctly indexed and searchable in ChromemDB.

Core Assumption for Test Updates:

For the following test modifications to be fully effective, I'm assuming that your ChromemManager.OnNewBlockConfirmed method (likely in sync_manager.go or chromemDB_manager.go) is responsible for:

Indexing the transaction itself into the transactionCollection.
Parsing the tx.Data based on tx.Type.
For TransactionTypeMCPRegisterCapability, preparing and adding the capability descriptor to the capabilityDescriptorCollection and a corresponding context record (for the registration event) to contextRecordCollection.
For TransactionTypeMCPInvokeCapability, preparing and adding the ContextRecord from the transaction data to the contextRecordCollection.
For TransactionTypeMCPUpdateCapability, preparing and updating/adding the capability descriptor in capabilityDescriptorCollection and adding a corresponding context record (for the update event) to contextRecordCollection.
If ChromemManager.OnNewBlockConfirmed currently only indexes the raw transaction, the tests verifying specific capabilities or context records in ChromemDB would not pass as expected. The manager would need to be enhanced to populate these other collections. My proposed test changes will operate under the assumption that this enhanced behavior is in place or intended.

General Approach for Test Updates:

ChromemDB Path Management: Your tests that initialize a BlockchainStruct (which in turn initializes ChromemManager) will need to handle the searchableDBPath for ChromemDB. Ensure this path is unique for each test (e.g., by using t.TempDir() or a timestamped name) and cleaned up afterwards.
Waiting for ChromemDB Sync: Since ChromemManager.OnNewBlockConfirmed is called within a goroutine in BlockchainStruct.AddBlock, your tests will need a mechanism to wait for ChromemDB to process the data. A simple polling helper function is a good way to achieve this.
ChromemDB Verification:
After blockchain operations that should result in ChromemDB entries (like mining a block with relevant transactions), your tests will query the appropriate ChromemDB collections (transactionCollection, capabilityDescriptorCollection, contextRecordCollection).
You'll use the ChromemManager instance (e.g., bc.ChromemSync) to access the ChromemDB client and its collections.
Verification will involve using methods like Get (for specific IDs) or Query (with filters) on the collections.
The content and metadata of the retrieved ChromemDB documents will be compared against expected values, typically derived from your Prepare...ForChromem... functions (e.g., PrepareTransactionForChromem, PrepareCapabilityDescriptorForChromemFromRegisterLocal, PrepareContextRecordForChromemEnhanced).
Let's add a helper function for waiting for ChromemDB updates. You can place this in a test utility file or directly within the test files.

go
// This can be in mcp_blockchain_test.go or a shared test_utils.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	// ... other necessary imports from your project ...
)

// WaitForChromemDB polls a check function until it returns true or a timeout occurs.
func WaitForChromemDB(timeout time.Duration, checkFunc func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Shorten poll interval for faster test execution, but not too short to spam.
	ticker := time.NewTicker(250 * time.Millisecond) 
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ChromemDB update: %w", ctx.Err())
		case <-ticker.C:
			done, err := checkFunc()
			if err != nil {
				// It's common for "not found" errors to occur during polling before data is synced.
				// We can log these as debug/trace if needed, but don't fail the poll immediately.
				// log.Printf("Polling ChromemDB (transient error): %v", err)
			}
			if done {
				return nil // Condition met
			}
		}
	}
}

// stringifyMetadataTestHelper converts map[string]interface{} to map[string]string for ChromemDB metadata comparison.
// This should ideally mirror the logic in your actual stringifyMetadata function used by ChromemManager.
func stringifyMetadataTestHelper(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		switch val := v.(type) {
		case string:
			result[k] = val
		case json.Number: // Handle numbers that might have been unmarshalled as json.Number
			result[k] = val.String()
		default:
			result[k] = fmt.Sprintf("%v", val) // General case
		}
	}
	return result
}
Now, let's apply these principles to update your test files.

1. /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN/mcp_blockchain_test.go
This file is crucial as it tests MCP transaction processing, capability invocation, and updates, all of which should have corresponding ChromemDB entries.

mcp_blockchain_test.go
-10
+185
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
	"KNIRVCHAIN/utils"
)

// WaitForChromemDB is a local helper for this test file.
// In a real project, this might be in a shared test utility.
func WaitForChromemDB(timeout time.Duration, checkFunc func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond) // Poll every 250ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ChromemDB update: %w", ctx.Err())
		case <-ticker.C:
			done, err := checkFunc()
			if err != nil {
				// Log transient errors during polling but don't fail immediately
				// t.Logf("Polling ChromemDB (transient error): %v", err)
			}
			if done {
				return nil
			}
		}
	}
}

// TestMCPTransactionProcessing tests the processing of MCP transactions
func TestMCPTransactionProcessing(t *testing.T) {
	// Create a test database
	dbPath := fmt.Sprintf("testdb_%d", time.Now().UnixNano())
	searchableDBPath := fmt.Sprintf("%s%s", t.TempDir(), "/chromem_mcp_proc")
	// defer os.RemoveAll(dbPath) // Handled by t.TempDir() if dbPath is inside it, or manually if separate
	// defer os.RemoveAll(searchableDBPath) // Handled by t.TempDir()

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create a genesis block
	genesisBlock := NewBlock(nil, 0, 0)

	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_processing", db, "/tmp/test_searchable_db", dummyCerebrasConfig)
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_processing", db, searchableDBPath, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	// Create a test wallet
	wallet, _ := NewWallet()
	from := wallet.GetAddress()

	// Register test capability directly in the database to ensure it exists
	err = registerTestCapabilityWithOwner(db, "resource-123", from)
	if err != nil {
		t.Fatalf("Failed to register test capability: %v", err)
	}
	registrationNetworkFee := uint64(10) // Network fee for registration
	capabilityGasFee := uint64(50)       // GasFeeNRN for the capability itself

	// Set the ProposerAddress BEFORE mining the block to ensure it's included in the hash
	newBlock.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer for network fees
	newBlock.MineBlock()

	blockNumberForChromem := newBlock.BlockNumber
	blockTimestampForChromem := newBlock.Timestamp
	// Add the registration block to the blockchain
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	bc.AddBlock(db, newBlock)

	// Verify that the capability was processed and stored
			expectedResourceType,
			actualResourceType)
	}

	// --- ChromemDB Verification ---
	t.Log("Verifying capability registration in ChromemDB...")
	err = WaitForChromemDB(10*time.Second, func() (bool, error) {
		// 1. Verify CapabilityDescriptor in capabilityDescriptorCollection
		// Ensure ChromemSync and its client are not nil
		if bc.ChromemSync == nil || bc.ChromemSync.client == nil {
			return false, fmt.Errorf("ChromemSync or its client is nil")
		}
		capCollection, errGetCol := bc.ChromemSync.client.GetCollection("capability_descriptors")
		if errGetCol != nil {
			return false, fmt.Errorf("failed to get capability_descriptors collection: %w", errGetCol)
		}
		results, errQuery := capCollection.Get(context.Background(), []string{resourceDesc.ID}, nil, nil)
		if errQuery != nil {
			// t.Logf("Error getting capability %s from ChromemDB: %v", resourceDesc.ID, errQuery)
			return false, nil // Don't fail poll on transient "not found"
		}
		if len(results) == 0 {
			// t.Logf("Capability %s not yet found in ChromemDB capability_descriptors collection", resourceDesc.ID)
			return false, nil // Not found yet
		}
		// TODO: Add detailed verification of results[0].Content and results[0].Metadata
		// based on PrepareCapabilityDescriptorForChromemFromRegisterLocal output.
		t.Logf("Capability %s found in ChromemDB capability_descriptors collection.", resourceDesc.ID)

		// 2. Verify ContextRecord for registration in contextRecordCollection
		// The ID for this context record would be the registration transaction's hash.
		ctxCollection, errGetCol := bc.ChromemSync.client.GetCollection("context_records")
		if errGetCol != nil {
			return false, fmt.Errorf("failed to get context_records collection: %w", errGetCol)
		}
		regCtxRecordID := txn.TransactionHash // The transaction hash is the ID for the registration context record
		ctxResults, errQuery := ctxCollection.Get(context.Background(), []string{regCtxRecordID}, nil, nil)
		if errQuery != nil {
			// t.Logf("Error getting registration context record %s from ChromemDB: %v", regCtxRecordID, errQuery)
			return false, nil // Don't fail poll on transient "not found"
		}
		if len(ctxResults) == 0 {
			// t.Logf("Registration context record %s not yet found in ChromemDB", regCtxRecordID)
			return false, nil // Not found yet
		}
		t.Logf("Registration ContextRecord %s found in ChromemDB.", regCtxRecordID)

		return true, nil // Both found
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for capability registration failed: %v", err)
	}
}

// capabilityTypeToProto converts domain CapabilityType to protobuf enum

	// Setup test database
	dbPath := fmt.Sprintf("testdb_invoke_%d", time.Now().UnixNano())
	searchableDBPathInvoke := fmt.Sprintf("%s%s", t.TempDir(), "/chromem_invoke")
	// defer os.RemoveAll(dbPath) // Handled by t.TempDir()
	// defer os.RemoveAll(searchableDBPathInvoke) // Handled by t.TempDir()

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_cap_invoke", db, "/tmp/test_searchable_db", dummyCerebrasConfig)
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_cap_invoke", db, searchableDBPathInvoke, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	newBlock2.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer
	newBlock2.MineBlock()

	invokeBlockNumberForChromem := newBlock2.BlockNumber
	invokeBlockTimestampForChromem := newBlock2.Timestamp

	// Add the invocation block to the blockchain (this will trigger fee transfers)
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	bc.AddBlock(db, newBlock2)

	// --- Verify balances after invocation ---
	// Note: Proposer also gets mining reward from newBlock2, which is not explicitly checked here but happens.
	// We are primarily focused on the fee transfer from the invokeTxn.
	t.Logf("After Invoke - Initiator/Owner (%s) Balance: %d, Proposer (%s) Balance: %d", from, finalInitiatorBalance, utils.BLOCKCHAIN_ADDRESS, finalProposerBalance)

	// --- ChromemDB Verification for Invocation ContextRecord ---
	t.Log("Verifying capability invocation context record in ChromemDB...")
	err = WaitForChromemDB(10*time.Second, func() (bool, error) {
		if bc.ChromemSync == nil || bc.ChromemSync.client == nil {
			return false, fmt.Errorf("ChromemSync or its client is nil")
		}
		ctxCollection, errGetCol := bc.ChromemSync.client.GetCollection("context_records")
		if errGetCol != nil {
			return false, fmt.Errorf("failed to get context_records collection: %w", errGetCol)
		}
		// The ID for the context record is the transaction hash
		results, errQuery := ctxCollection.Get(context.Background(), []string{invokeTxn.TransactionHash}, nil, nil)
		if errQuery != nil {
			// t.Logf("Error getting context record %s from ChromemDB: %v", invokeTxn.TransactionHash, errQuery)
			return false, nil
		}
		if len(results) == 0 {
			// t.Logf("Context record %s not yet found in ChromemDB context_records collection", invokeTxn.TransactionHash)
			return false, nil
		}
		// TODO: Add detailed verification of results[0].Content and results[0].Metadata
		// based on PrepareContextRecordForChromemEnhanced output.
		t.Logf("ContextRecord %s for invocation found in ChromemDB.", invokeTxn.TransactionHash)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for invocation context record failed: %v", err)
	}
}

// TestMCPCapabilityUpdate tests the update of an MCP capability
func TestMCPCapabilityUpdate(t *testing.T) {
	// Create a test database
	dbPath := fmt.Sprintf("testdb_%d", time.Now().UnixNano())
	searchableDBPathUpdate := fmt.Sprintf("%s%s", t.TempDir(), "/chromem_update")
	// defer os.RemoveAll(dbPath)
	// defer os.RemoveAll(searchableDBPathUpdate)

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_update", db, "/tmp/test_searchable_db", dummyCerebrasConfig)
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_update", db, searchableDBPathUpdate, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}
	newBlock2.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer for network fees
	newBlock2.MineBlock()

	updateBlockNumberForChromem := newBlock2.BlockNumber
	updateBlockTimestampForChromem := newBlock2.Timestamp

	// Add the block to the blockchain
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	bc.AddBlock(db, newBlock2)

	// Verify that the capability was updated
	if resourceDescFromDB.BaseDescriptor.GasFeeNrn != updatedResourceDesc.GasFeeNRN {
		t.Errorf("ResourceDescriptor GasFeeNRN mismatch: expected %d, got %d", updatedResourceDesc.GasFeeNRN, resourceDescFromDB.BaseDescriptor.GasFeeNrn)
	}

	// --- ChromemDB Verification ---
	t.Log("Verifying capability update in ChromemDB...")
	err = WaitForChromemDB(10*time.Second, func() (bool, error) {
		if bc.ChromemSync == nil || bc.ChromemSync.client == nil {
			return false, fmt.Errorf("ChromemSync or its client is nil")
		}
		// 1. Verify the update transaction is indexed
		txCollection, errGetCol := bc.ChromemSync.client.GetCollection("transactions")
		if errGetCol != nil {
			return false, fmt.Errorf("failed to get transactions collection: %w", errGetCol)
		}
		txResults, errQuery := txCollection.Get(context.Background(), []string{updateTxn.TransactionHash}, nil, nil)
		if errQuery != nil || len(txResults) == 0 {
			// t.Logf("Update transaction %s not yet found in ChromemDB: %v", updateTxn.TransactionHash, errQuery)
			return false, nil
		}
		t.Logf("Update transaction %s found in ChromemDB.", updateTxn.TransactionHash)

		// 2. Verify ContextRecord for update in contextRecordCollection
		ctxCollection, errGetCol := bc.ChromemSync.client.GetCollection("context_records")
		if errGetCol != nil {
			return false, fmt.Errorf("failed to get context_records collection: %w", errGetCol)
		}
		updateCtxRecordID := updateTxn.TransactionHash // The transaction hash is the ID for the update context record
		ctxResults, errQuery := ctxCollection.Get(context.Background(), []string{updateCtxRecordID}, nil, nil)
		if errQuery != nil || len(ctxResults) == 0 {
			// t.Logf("Update context record %s not yet found in ChromemDB: %v", updateCtxRecordID, errQuery)
			return false, nil
		}
		t.Logf("Update ContextRecord %s found in ChromemDB.", updateCtxRecordID)

		// 3. Verify the updated CapabilityDescriptor in capabilityDescriptorCollection
		// (This assumes ChromemManager.OnNewBlockConfirmed is enhanced to handle updates to this collection)
		capCollection, errGetCol := bc.ChromemSync.client.GetCollection("capability_descriptors")
		if errGetCol != nil {
			return false, fmt.Errorf("failed to get capability_descriptors collection: %w", errGetCol)
		}
		capResults, errQuery := capCollection.Get(context.Background(), []string{resourceDesc.ID}, nil, nil)
		if errQuery != nil || len(capResults) == 0 {
			// t.Logf("Updated capability %s not yet found in ChromemDB: %v", resourceDesc.ID, errQuery)
			return false, nil
		}
		// TODO: Add detailed verification of capResults[0].Content and capResults[0].Metadata
		// to ensure it reflects 'updatedResourceDesc' based on PrepareCapabilityDescriptorForChromemFromUpdate.
		// For now, just check if the name matches.
		// This requires parsing capResults[0].Metadata["Name"] or similar.
		t.Logf("Updated capability %s found in ChromemDB capability_descriptors collection.", resourceDesc.ID)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for capability update failed: %v", err)
+	}
 }
 
 // Helper function to register a test capability with a specific owner

2. /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN/transaction_test.go
This file tests the basic transaction flow. We'll add ChromemDB verification for the transactions.

diff
-11
+84
The code change produced by Gemini could not be fully parsed. Can't automatically accept changes.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"KNIRVCHAIN/utils"
)

// WaitForChromemDBTxTest is a local helper for this test file.
func WaitForChromemDBTxTest(timeout time.Duration, checkFunc func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond) // Poll every 250ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ChromemDB update: %w", ctx.Err())
		case <-ticker.C:
			done, err := checkFunc()
			if err != nil {
				// t.Logf("Polling ChromemDB (transient error): %v", err)
			}
			if done {
				return nil
			}
		}
	}
}

// Use real ConsensusManager for testing
func newTestConsensusManager(bc *BlockchainStruct) *ConsensusManager {
	return &ConsensusManager{
func TestTransactionFlow(t *testing.T) {
	// Create test DB connection with unique test path
	testDBPath := fmt.Sprintf("test_db/transaction_test_%d.db", time.Now().UnixNano())
	searchableDBPathTx := fmt.Sprintf("%s%s", t.TempDir(), "/chromem_tx_flow")
	// defer os.RemoveAll(testDBPath) // Handled by t.TempDir()
	// defer os.RemoveAll(searchableDBPathTx) // Handled by t.TempDir()

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
		db.Close() // No need to remove files if using t.TempDir() for paths
	}()

	// Create test wallets
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	bc, err := NewBlockchain(genesisBlock, chainID, senderWallet.GetAddress(), db, "/tmp/test_searchable_db", dummyCerebrasConfig) // Use sender's address as a placeholder miner
	bc, err := NewBlockchain(genesisBlock, chainID, senderWallet.GetAddress(), db, searchableDBPathTx, dummyCerebrasConfig) // Use sender's address as a placeholder miner
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}
	// Add a temporary funding transaction directly for the test setup
	// This simulates the faucet but ensures funds exist before the main test txn
	fundingAmount := uint64(10000 * utils.DECIMAL) // Give sender enough funds
	fundingBlockNumber := uint64(0)                // Will be set after mining
	fundingBlockTimestamp := int64(0)              // Will be set after mining
	fundingTxn := NewTransaction(utils.BLOCKCHAIN_ADDRESS, senderWallet.GetAddress(), fundingAmount, []byte("test setup funding"))
	fundingTxn.Status = TXN_VERIFICATION_SUCCESS // Mark as verified for pool

				for _, txn := range latestBlock.Transactions {
					if txn.TransactionHash == fundingTxn.TransactionHash {
						fundingMined = true
						fundingBlockNumber = latestBlock.BlockNumber
						fundingBlockTimestamp = latestBlock.Timestamp
						t.Logf("Funding transaction %s mined in block %d", fundingTxn.TransactionHash, latestBlock.BlockNumber)
						break
					}
	}
	// --- Funding transaction mined ---

	// --- ChromemDB Verification for Funding Transaction ---
	t.Logf("Verifying funding transaction %s in ChromemDB...", fundingTxn.TransactionHash)
	err = WaitForChromemDBTxTest(10*time.Second, func() (bool, error) {
		if bc.ChromemSync == nil || bc.ChromemSync.client == nil {
			return false, fmt.Errorf("ChromemSync or its client is nil")
		}
		txCollection, errGetCol := bc.ChromemSync.client.GetCollection("transactions")
		if errGetCol != nil {
			return false, fmt.Errorf("failed to get transactions collection: %w", errGetCol)
		}
		results, errQuery := txCollection.Get(context.Background(), []string{fundingTxn.TransactionHash}, nil, nil)
		if errQuery != nil || len(results) == 0 {
			// t.Logf("Funding transaction %s not yet found in ChromemDB: %v", fundingTxn.TransactionHash, errQuery)
			return false, nil
		}
		// TODO: Add more detailed verification of results[0].Content and results[0].Metadata
		// based on PrepareTransactionForChromem output.
		t.Logf("Funding transaction %s found in ChromemDB.", fundingTxn.TransactionHash)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for funding transaction failed: %v", err)
	}

	// --- Now proceed with the actual test transaction ---
	t.Log("Creating and signing the main test transaction...")
	// Create and sign a transaction
		1000, // 10.00 nrn (DECIMAL=100)
		[]byte("test transaction"),
	)
	mainTxBlockNumber := uint64(0)
	mainTxBlockTimestamp := int64(0)

	signedTxn, err := senderWallet.GetSignedTxn(*txn)
	if err != nil {
					for _, blockTxn := range block.Transactions {
						if blockTxn.TransactionHash == signedTxn.TransactionHash {
							minedBlockIndex = i
							mainTxBlockNumber = block.BlockNumber
							mainTxBlockTimestamp = block.Timestamp
							t.Logf("Main transaction %s found in block %d", signedTxn.TransactionHash, block.BlockNumber)
							break
						}
	}
	bc.Unlock()

	// --- ChromemDB Verification for Main Transaction ---
	t.Logf("Verifying main transaction %s in ChromemDB...", signedTxn.TransactionHash)
	err = WaitForChromemDBTxTest(10*time.Second, func() (bool, error) {
		if bc.ChromemSync == nil || bc.ChromemSync.client == nil {
			return false, fmt.Errorf("ChromemSync or its client is nil")
		}
		txCollection, errGetCol := bc.ChromemSync.client.GetCollection("transactions")
		if errGetCol != nil {
			return false, fmt.Errorf("failed to get transactions collection: %w", errGetCol)
		}
		results, errQuery := txCollection.Get(context.Background(), []string{signedTxn.TransactionHash}, nil, nil)
		if errQuery != nil || len(results) == 0 {
			// t.Logf("Main transaction %s not yet found in ChromemDB: %v", signedTxn.TransactionHash, errQuery)
			return false, nil
		}
		t.Logf("Main transaction %s found in ChromemDB.", signedTxn.TransactionHash)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for main transaction failed: %v", err)
	}

	t.Log("TestTransactionFlow completed successfully.")

	// Cleanup test chain (already deferred)
	// db.Client.Delete([]byte(chainID), nil) // Be careful with direct DB manipulation in tests
 }

Summary of Changes and Next Steps:
ChromemDB Path: Tests now use t.TempDir() to create unique paths for ChromemDB, ensuring test isolation and cleanup.
WaitForChromemDB Helper: A polling helper is introduced to wait for asynchronous updates to ChromemDB.
Verification Logic:
After a block is added (bc.AddBlock(db, ...)), the tests now include a WaitForChromemDB call followed by queries to the relevant ChromemDB collections (capability_descriptors, context_records, transactions).
The basic check is for the existence of the record by its ID.
TODO for you: You'll need to add more detailed assertions within the checkFunc passed to WaitForChromemDB. This involves:
Fetching the document from ChromemDB using collection.Get(...).
Comparing results[0].Content (the document string) and results[0].Metadata (map[string]string) with the expected output from your Prepare...ForChromem... functions in chromemDB_conversion.go. Remember that metadata in ChromemDB will be stringified.
ChromemManager Assumption: These test changes heavily rely on the assumption that ChromemManager.OnNewBlockConfirmed is correctly processing different transaction types and populating all three collections (transactionCollection, capabilityDescriptorCollection, contextRecordCollection). If this is not the case, the ChromemDB verification steps for capabilities and context records will fail.
By implementing these changes, your tests will provide better coverage for your new ChromemDB architecture, ensuring that data is not only committed to LevelDB but also correctly indexed and available in ChromemDB.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
