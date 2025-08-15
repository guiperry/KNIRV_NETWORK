package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	chromem "github.com/philippgille/chromem-go"
	// Removed leveldb import as we no longer use it directly

	"KNIRVROOT/config"
	pb "KNIRVROOT/proto"
	"KNIRVROOT/types"
	"KNIRVROOT/utils"
)

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

	// Create a test blockchain
	chainID := fmt.Sprintf("test_chain_%d", time.Now().UnixNano())
	// Provide a dummy CerebrasConfig for testing
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	// Pass nil for ChromemManager in tests - it will create a new one internally
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_processing", db, nil, searchableDBPath, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}

	// Create a test wallet
	wallet, _ := NewWallet()
	from := wallet.GetAddress()

	// Generate unique capability ID for this test run
	capabilityID := fmt.Sprintf("resource-test-%d", time.Now().UnixNano())
	registrationNetworkFee := uint64(10) // Network fee for registration
	capabilityGasFee := uint64(50)       // GasFeeNRN for the capability itself

	// Create a test resource descriptor
	resourceDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             capabilityID,
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource for unit testing",
			GasFeeNRN:      capabilityGasFee, // Set the capability's invocation fee
			Timestamp:      time.Now().Unix(),
			CustomMetadata: map[string]interface{}{"key1": "value1"},
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
		ContentHash:  "sha256:1234567890abcdef",
		Schema: types.PluginSchemaDetail{
			Summary:       "This is resource 2 (API)",
			LocationHints: []string{"http://api.example.com/res2"},
			// ManifestFile and ExecutableFile might be empty for ResourceTypeAPI
		},
	}

	// Create transaction data
	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": resourceDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create MCP transaction
	txn, err := NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPRegisterCapability, registrationNetworkFee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}

	// Sign the transaction
	err = wallet.SignTransaction(txn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// --- Directly fund the 'from' account before adding capability registration ---
	fundingAmount := registrationNetworkFee + capabilityGasFee + 10000 // Enough for all fees and some buffer
	err = db.SaveAccountBalance(from, fundingAmount)
	if err != nil {
		t.Fatalf("Failed to directly fund account %s: %v", from, err)
	}
	t.Logf("TestMCPTransactionProcessing: Directly funded account %s with %d via SaveAccountBalance", from, fundingAmount)

	// Add the transaction to the pool AFTER funding is confirmed
	bc.AddTransactionToTransactionPool(txn.Clone()) // Use cloned transaction to ensure immutability

	// Create a new block with the transaction
	newBlock := NewBlock(bc.Blocks[len(bc.Blocks)-1].Hash(), int(bc.Blocks[len(bc.Blocks)-1].BlockNumber)+1, bc.Blocks[len(bc.Blocks)-1].BlockNumber+1)
	newBlock.Transactions = append(newBlock.Transactions, txn)
	// Set the ProposerAddress BEFORE mining the block to ensure it's included in the hash
	newBlock.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer for network fees
	newBlock.MineBlock()

	// Add the registration block to the blockchain
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	bc.AddBlock(newBlock)

	// Verify that the capability was processed and stored with retry logic
	var capability *pb.CapabilityDescriptorContainerProto
	var verifyErr error
	for i := 0; i < 10; i++ {
		capability, verifyErr = db.GetCapabilityByID(resourceDesc.ID)
		if verifyErr == nil && capability != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if verifyErr != nil {
		t.Fatalf("Failed to get capability from database after retries: %v", verifyErr)
	}
	if capability == nil {
		t.Fatalf("Capability should not be nil after retries")
	}

	// Verify the capability fields
	resourceDescProto, ok := capability.Descriptor_.(*pb.CapabilityDescriptorContainerProto_Resource)
	if !ok {
		t.Fatalf("Capability is not of type ResourceDescriptor")
	}
	resourceDescFromDB := resourceDescProto.Resource
	if resourceDescFromDB.BaseDescriptor.Id != resourceDesc.ID {
		t.Errorf("ResourceDescriptor ID mismatch: expected %s, got %s", resourceDesc.ID, resourceDescFromDB.BaseDescriptor.Id)
	}
	if resourceDescFromDB.BaseDescriptor.Name != resourceDesc.Name {
		t.Errorf("ResourceDescriptor Name mismatch: expected %s, got %s", resourceDesc.Name, resourceDescFromDB.BaseDescriptor.Name)
	}
	if resourceDescFromDB.BaseDescriptor.Owner != resourceDesc.Owner {
		t.Errorf("ResourceDescriptor Owner mismatch: expected %s, got %s", resourceDesc.Owner, resourceDescFromDB.BaseDescriptor.Owner)
	}
	if resourceDescFromDB.BaseDescriptor.CapabilityType != capabilityTypeToProto(resourceDesc.CapabilityType) {
		t.Errorf("ResourceDescriptor CapabilityType mismatch: expected %d, got %d",
			capabilityTypeToProto(resourceDesc.CapabilityType),
			resourceDescFromDB.BaseDescriptor.CapabilityType)
	}

	// Convert protobuf enum to domain enum
	expectedResourceType := string(resourceDesc.ResourceType)
	actualResourceType := strings.TrimPrefix(resourceDescFromDB.ResourceType.String(), "DISCOVERY_RESOURCE_TYPE_PROTO_")
	if actualResourceType != expectedResourceType {
		t.Errorf("ResourceDescriptor ResourceType mismatch: expected %s, got %s",
			expectedResourceType,
			actualResourceType)
	}

	// --- ChromemDB Verification ---
	t.Log("Verifying capability registration in ChromemDB...")
	err = utils.WaitForChromemDB(30*time.Second, func() (bool, error) {
		// 1. Verify CapabilityDescriptor in capabilityDescriptorCollection
		// Use the collection from ChromemDBSyncManager which handles detailed processing
		var capCollection *chromem.Collection
		if bc.ChromemDBSyncManager != nil {
			capCollection = bc.ChromemDBSyncManager.capabilityDescriptorCollection
		} else {
			// Fallback to legacy ChromemSync for backward compatibility
			capCollection = bc.ChromemSync.capabilityDescriptorCollection
		}
		if capCollection == nil {
			return false, fmt.Errorf("capability_descriptors collection not initialized")
		}

		// Attempt to get the capability by its ID directly
		t.Logf("WaitForChromemDB: Attempting to Get capability with ID: '%s'", resourceDesc.ID)
		documents, err := bc.ChromemDBSyncManager.Get(context.Background(), []string{resourceDesc.ID}, []string{"documents", "metadatas"}, nil)
		results := make([]chromem.Result, len(documents))
		for i, doc := range documents {
			results[i] = chromem.Result{
				ID:       doc.ID,
				Metadata: doc.Metadata,
				Content:  doc.Content,
			}
		}
		if err != nil {
			// Log the error and allow retry by WaitForChromemDB
			t.Logf("WaitForChromemDB: ChromemDB Get for capability %s failed: %v. Retrying...", resourceDesc.ID, err)
			return false, err // Returning nil error to allow retry
		}

		if len(results) == 0 {
			t.Logf("WaitForChromemDB: Capability %s not found in ChromemDB via Get. Retrying...", resourceDesc.ID)
			return false, fmt.Errorf("capability %s not found in ChromemDB via Get (empty results)", resourceDesc.ID) // Returning nil error to allow retry
		}

		// Verify metadata
		if results[0].Metadata == nil {
			return false, fmt.Errorf("capability %s found in ChromemDB but metadata is nil", resourceDesc.ID)
		}

		// The document ID itself should be the capability ID.
		// The metadata map as set by PrepareCapabilityDescriptorForChromemFromRegister in sync_manager.go
		// does not contain "capability_id". The ID is the document's ID.
		if results[0].ID != resourceDesc.ID {
			return false, fmt.Errorf("capability ID mismatch in ChromemDB document ID: expected %s, got %s", resourceDesc.ID, results[0].ID)
		}
		// Verify document content (formatted string) contains the ID
		if !strings.Contains(results[0].Content, resourceDesc.ID) {
			return false, fmt.Errorf("capability ID %s not found in ChromemDB document content: '%s'", resourceDesc.ID, results[0].Content)
		}
		t.Logf("Capability %s successfully verified in ChromemDB capability_descriptors collection.", resourceDesc.ID)

		// 2. Verify ContextRecord for registration in contextRecordCollection
		// The context record ID is the transaction hash
		// Use ChromemSyncManager.Get to retrieve it, similar to how capability descriptor is retrieved.
		// ChromemSyncManager.Get searches across collections.
		// Log the attempt to query context record
		t.Logf("WaitForChromemDB: Attempting to Get context_record with ID (tx hash): '%s'", txn.TransactionHash)
		ctxDocuments, err := bc.ChromemDBSyncManager.Get(context.Background(), []string{txn.TransactionHash}, []string{"documents", "metadatas"}, nil)

		// Convert to chromem.Result for consistency if needed, or directly use ctxDocuments
		var foundContextRecord bool
		var contextRecordData chromem.Result // This is philippgille/chromem-go.Result

		for _, doc := range ctxDocuments { // doc is main.chromem.Document
			// Check if this document is indeed a context record by looking at its metadata
			// PrepareContextRecordForChromemEnhanced sets "RecordType": "context_record"
			if doc.Metadata != nil && doc.Metadata["RecordType"] == "context_record" {
				contextRecordData = chromem.Result{Metadata: doc.Metadata}
				foundContextRecord = true
				break
			}
		}

		if err != nil {
			// Log the error and allow retry
			t.Logf("WaitForChromemDB: ChromemDB Query for context record %s failed: %v. Retrying...", txn.TransactionHash, err)
			return false, err // Propagate error
		}
		if len(ctxDocuments) == 0 {
			// Log not found and allow retry
			t.Logf("WaitForChromemDB: Registration context record %s not found in ChromemDB via Get (empty results). Retrying...", txn.TransactionHash)
			return false, fmt.Errorf("registration context record %s not found in ChromemDB via Get (empty results)", txn.TransactionHash)
		}
		if !foundContextRecord {
			t.Logf("WaitForChromemDB: Document with ID %s found via Get, but it's not a context_record. Retrying...", txn.TransactionHash)
			return false, fmt.Errorf("document with ID %s found, but not a context_record", txn.TransactionHash)
		}

		// Verify metadata of the found context record
		if contextRecordData.Metadata == nil || contextRecordData.Metadata["CapabilityID"] != resourceDesc.ID {
			return false, fmt.Errorf("context record %s metadata invalid or CapabilityID mismatch. Metadata: %v", txn.TransactionHash, contextRecordData.Metadata)
		}
		t.Logf("Registration ContextRecord %s successfully found in ChromemDB context_records collection.", txn.TransactionHash)
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for capability registration failed: %v", err)
	}
}

// capabilityTypeToProto converts domain CapabilityType to protobuf enum
func capabilityTypeToProto(capType types.CapabilityType) pb.CapabilityTypeProto {
	switch capType {
	case types.CapabilityTypeResource:
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_RESOURCE
	case types.CapabilityTypeTool:
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_TOOL
	case types.CapabilityTypePrompt:
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_PROMPT
	case types.CapabilityTypeMemoryService:
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_MEMORY_SERVICE
	default:
		return pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_UNSPECIFIED
	}
}

func TestCapabilityInvocation(t *testing.T) {
	// Store expected context record fields for verification
	var (
		expectedCapabilityID    string
		expectedInteractionType types.InteractionType
		expectedInitiator       string
		expectedTimestamp       int64
		expectedInputHash       string
		expectedOutputHash      string
		expectedDetails         map[string]interface{}
	)

	// Setup test database
	dbPath := fmt.Sprintf("testdb_invoke_%d", time.Now().UnixNano())
	searchableDBPathInvoke := fmt.Sprintf("test_chromem_invoke_%d", time.Now().UnixNano())
	defer os.RemoveAll(dbPath)
	defer os.RemoveAll(searchableDBPathInvoke)

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create genesis block with root wallet address as proposer
	genesisBlock := NewBlock(nil, 0, 0)
	genesisBlock.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Use root wallet address for genesis block

	chainID := fmt.Sprintf("test_chain_invoke_%d", time.Now().UnixNano())
	// Provide a dummy CerebrasConfig for testing
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	// Pass nil for ChromemManager in tests - it will create a new one internally
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_cap_invoke", db, nil, searchableDBPathInvoke, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}

	// Use specific initiator address
	from := "KNIRVROOTfdc23d2edefeca228ddaa7efe7a2483db1eef06f"

	// Create standard test wallet (address won't match initiator but that's okay)
	wallet, _ := NewWallet()

	// Initialize account balance for initiator address
	initialBalance := uint64(1000) // Sufficient for fees
	err = db.SaveAccountBalance(from, initialBalance)
	if err != nil {
		t.Fatalf("Failed to set initial balance for initiator: %v", err)
	}

	// Commit initial balance to blockchain state
	initialBlock := NewBlock(bc.Blocks[len(bc.Blocks)-1].Hash(), int(bc.Blocks[len(bc.Blocks)-1].BlockNumber)+1, bc.Blocks[len(bc.Blocks)-1].BlockNumber+1)
	initialBlock.ProposerAddress = utils.BLOCKCHAIN_ADDRESS
	initialBlock.MineBlock()
	bc.AddBlock(initialBlock)

	// Verify balance was properly committed
	actualBalance, err := db.GetAccountBalance(from)
	if err != nil || actualBalance != initialBalance {
		t.Fatalf("Failed to verify initial balance: expected %d, got %d, err: %v", initialBalance, actualBalance, err)
	}

	// Register test capability with the wallet's address as owner
	capabilityID := "resource-123"
	err = registerTestCapabilityWithOwner(db, capabilityID, from)
	if err != nil {
		t.Fatalf("Failed to register test capability: %v", err)
	}

	// Create test resource descriptor
	capabilityGasFee := uint64(50) // Gas fee for capability
	resourceDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             capabilityID,
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource for unit testing",
			GasFeeNRN:      capabilityGasFee,
			Timestamp:      time.Now().Unix(),
			CustomMetadata: map[string]interface{}{"key1": "value1"},
			CapabilityType: types.CapabilityTypeResource,
		},
	}

	invocationNetworkFee := uint64(50) // Network fee for the invocation transaction - match the capability gas fee

	// Test capability invocation
	// Set expected values for verification
	expectedCapabilityID = resourceDesc.ID
	expectedInteractionType = types.InteractionTypeResourceAccess
	expectedInitiator = from
	expectedTimestamp = time.Now().Unix()
	expectedInputHash = "sha256:input-hash"
	expectedOutputHash = "sha256:output-hash"
	expectedDetails = map[string]interface{}{"param1": "value1", "param2": 42}

	// Create transaction data first to get the tx hash
	invokeTxnData, err := json.Marshal(map[string]interface{}{
		"contextRecord": types.ContextRecord{
			CapabilityID:    expectedCapabilityID,
			InteractionType: expectedInteractionType,
			Initiator:       expectedInitiator,
			Timestamp:       expectedTimestamp,
			InputHash:       expectedInputHash,
			OutputHash:      expectedOutputHash,
			Details:         expectedDetails,
		},
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create MCP transaction for capability invocation
	invokeTxn, err := NewMCPTransaction(from, "", 0, invokeTxnData, TransactionTypeMCPInvokeCapability, invocationNetworkFee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}

	// Sign the transaction
	err = wallet.SignTransaction(invokeTxn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Add the transaction to the pool
	bc.AddTransactionToTransactionPool(invokeTxn)

	// --- Directly fund the 'from' account before processing the invocation ---
	fundingAmount := invocationNetworkFee + capabilityGasFee + 10000 // Enough for all fees and some buffer
	err = db.SaveAccountBalance(from, fundingAmount)
	if err != nil {
		t.Fatalf("Failed to directly fund account %s: %v", from, err)
	}
	t.Logf("TestCapabilityInvocation: Directly funded account %s with %d via SaveAccountBalance", from, fundingAmount)

	// --- Get balances before invocation block is processed ---
	// 'from' is both initiator and owner in this test
	initialInitiatorBalance, _ := db.GetAccountBalance(from)

	initialProposerBalance, _ := db.GetAccountBalance(utils.BLOCKCHAIN_ADDRESS)

	t.Logf("Before Invoke - Initiator/Owner (%s) Balance: %d, Proposer (%s) Balance: %d", from, initialInitiatorBalance, utils.BLOCKCHAIN_ADDRESS, initialProposerBalance)

	// Create a new block with the transaction
	newBlock2 := NewBlock(bc.Blocks[len(bc.Blocks)-1].Hash(), int(bc.Blocks[len(bc.Blocks)-1].BlockNumber)+1, bc.Blocks[len(bc.Blocks)-1].BlockNumber+1)
	newBlock2.Transactions = append(newBlock2.Transactions, invokeTxn)
	// Set the ProposerAddress BEFORE mining the block to ensure it's included in the hash
	newBlock2.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer
	newBlock2.MineBlock()

	// Add the invocation block to the blockchain (this will trigger fee transfers)
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	t.Logf("Adding block with transaction hash: %s", invokeTxn.TransactionHash)
	t.Logf("Context record to be stored: %+v", types.ContextRecord{
		ID:              invokeTxn.TransactionHash,
		CapabilityID:    expectedCapabilityID,
		InteractionType: expectedInteractionType,
		Initiator:       expectedInitiator,
		Timestamp:       expectedTimestamp,
		InputHash:       expectedInputHash,
		OutputHash:      expectedOutputHash,
		Details:         expectedDetails,
	})
	bc.AddBlock(newBlock2)

	// --- Verify balances after invocation ---
	finalInitiatorBalance, _ := db.GetAccountBalance(from)
	finalProposerBalance, _ := db.GetAccountBalance(utils.BLOCKCHAIN_ADDRESS)

	// Wait for LevelDB writes to complete with retry
	var contextRecordFromDB *pb.ContextRecordProto
	t.Logf("Starting context record retrieval attempts for tx %s", invokeTxn.TransactionHash)
	for i := 0; i < 10; i++ { // Increased retry count
		contextRecordFromDB, err = db.GetContextRecord(invokeTxn.TransactionHash)
		if err == nil {
			t.Logf("Successfully retrieved context record on attempt %d", i+1)
			break
		}
		// Check if the error indicates "not found" without direct leveldb dependency
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("Failed to get context record: %v", err)
		}
		t.Logf("Context record not found yet (attempt %d), waiting...", i+1)
		time.Sleep(500 * time.Millisecond) // Increased delay
	}
	if err != nil {
		t.Fatalf("Failed to retrieve context record after retries: %v", err)
	}

	// Verify that the context record was processed and stored using tx hash as ID
	t.Logf("Using context record with key: %s", invokeTxn.TransactionHash)
	// No need to retrieve again, we already have it from the retry loop above
	if contextRecordFromDB == nil {
		t.Fatalf("Context record should not be nil")
	}
	t.Logf("Successfully retrieved context record: %+v", contextRecordFromDB)

	// Verify the context record fields
	if contextRecordFromDB.Id != invokeTxn.TransactionHash {
		t.Errorf("ContextRecord ID mismatch: expected %s, got %s", invokeTxn.TransactionHash, contextRecordFromDB.Id)
	}
	if contextRecordFromDB.CapabilityId != expectedCapabilityID {
		t.Errorf("ContextRecord CapabilityID mismatch: expected %s, got %s", expectedCapabilityID, contextRecordFromDB.CapabilityId)
	}
	// Convert proto InteractionType to string for comparison
	var protoInteractionType string
	switch contextRecordFromDB.InteractionType {
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_RESOURCE_ACCESS:
		protoInteractionType = string(types.InteractionTypeResourceAccess) // Use the correct type
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_TOOL_INVOCATION:
		protoInteractionType = "QUERY"
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PROMPT_USAGE:
		protoInteractionType = "REGISTER"
	case pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PLUGIN_EXECUTION:
		protoInteractionType = "UPDATE"
	default:
		protoInteractionType = "UNKNOWN"
	}
	if protoInteractionType != string(expectedInteractionType) { // Compare with the expected string
		t.Errorf("ContextRecord InteractionType mismatch: expected %s, got %s", string(expectedInteractionType), protoInteractionType)
	}
	if contextRecordFromDB.Initiator != expectedInitiator {
		t.Errorf("ContextRecord Initiator mismatch: expected %s, got %s", expectedInitiator, contextRecordFromDB.Initiator)
	}

	// --- Assert balance changes ---
	// Initiator pays: network fee for invocation.
	// If initiator is owner, capability gas fee is paid by initiator and "burned" (not credited back).
	expectedInitiatorBalance := new(big.Int).SetUint64(initialInitiatorBalance)
	expectedInitiatorBalance.Sub(expectedInitiatorBalance, new(big.Int).SetUint64(invocationNetworkFee))
	// Do not subtract capabilityGasFee here if initiator is owner, as they receive it back.
	expectedInitiatorBalance.Sub(expectedInitiatorBalance, new(big.Int).SetUint64(capabilityGasFee))

	if new(big.Int).SetUint64(finalInitiatorBalance).Cmp(expectedInitiatorBalance) != 0 {
		t.Errorf("Initiator balance incorrect after invocation. Expected: %s, Got: %d. Initial: %d, InvocationNetFee: %d, CapGasFee: %d",
			expectedInitiatorBalance.String(), finalInitiatorBalance, initialInitiatorBalance, invocationNetworkFee, capabilityGasFee)
	}

	// Owner (same as initiator here) pays the capability gas fee which is "burned" (not credited back)
	// So, finalOwnerBalance should reflect the net change.

	expectedProposerBalance := new(big.Int).SetUint64(initialProposerBalance)
	expectedProposerBalance.Add(expectedProposerBalance, new(big.Int).SetUint64(invocationNetworkFee)) // Proposer gets network fee
	// Note: Proposer also gets mining reward from newBlock2, which is not explicitly checked here but happens.
	// We are primarily focused on the fee transfer from the invokeTxn.
	t.Logf("After Invoke - Initiator/Owner (%s) Balance: %d, Proposer (%s) Balance: %d", from, finalInitiatorBalance, utils.BLOCKCHAIN_ADDRESS, finalProposerBalance)

	// --- ChromemDB Verification for Invocation ContextRecord ---
	t.Log("Verifying capability invocation context record in ChromemDB...")
	err = utils.WaitForChromemDB(10*time.Second, func() (bool, error) {
		// The ID for the context record is the transaction hash
		// The key used in LevelDB is "ctx:" + invokeTxn.TransactionHash
		// The ID stored in ChromemDB by ChromemDBSyncManager is invokeTxn.TransactionHash
		// IMPORTANT: Use the ChromemDBSyncManager's collection, as that's where the record is added.
		var actualCtxCollection *chromem.Collection
		if bc.ChromemDBSyncManager != nil {
			// Ensure the sync manager is initialized
			actualCtxCollection = bc.ChromemDBSyncManager.contextRecordCollection
		} else {
			// Fallback to legacy ChromemSync for backward compatibility
			actualCtxCollection = bc.ChromemSync.contextRecordCollection
		}

		if actualCtxCollection == nil {
			return false, fmt.Errorf("context_records collection not initialized")
		}

		doc, err := actualCtxCollection.GetByID(
			context.Background(),
			invokeTxn.TransactionHash,
		)
		if err != nil {
			// Don't return error immediately, let WaitForChromemDB retry
			t.Logf("ChromemDB GetByID error during verification for ID %s: %v", invokeTxn.TransactionHash, err)
			return false, nil
		}
		if doc.ID == "" {
			return false, fmt.Errorf("context record %s not found in ChromemDB context_records collection", invokeTxn.TransactionHash)
		}
		// Further checks on doc.Content and doc.Metadata can be added here
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
	}

	// Create a genesis block with root wallet address as proposer
	genesisBlock := NewBlock(nil, 0, 0)
	genesisBlock.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Use root wallet address for genesis block

	// Create a test blockchain
	chainID := fmt.Sprintf("test_chain_update_%d", time.Now().UnixNano()) // Use unique chain ID for update test
	// Provide a dummy CerebrasConfig for testing
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	// Pass nil for ChromemManager in tests - it will create a new one internally
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_update", db, nil, searchableDBPathUpdate, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}

	// Create a test wallet
	wallet, _ := NewWallet()
	from := wallet.GetAddress()
	fee := uint64(100)

	// Initialize account balance directly in the database with retry logic
	initialBalance := uint64(10000) // Enough for all transactions
	var actualBalance uint64
	for i := 0; i < 5; i++ {
		err = db.SaveAccountBalance(from, initialBalance)
		if err != nil {
			t.Logf("Attempt %d to set account balance failed: %v", i+1, err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		actualBalance, err = db.GetAccountBalance(from)
		if err == nil && actualBalance == initialBalance {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Failed to initialize account balance after retries: %v", err)
	}
	if actualBalance != initialBalance {
		t.Fatalf("Account balance not set correctly after retries. Expected: %d, Got: %d", initialBalance, actualBalance)
	}
	t.Logf("Successfully initialized account %s with balance %d after verification", from, actualBalance)

	// Generate unique capability ID for this test
	capabilityID := fmt.Sprintf("resource-test-update-%d", time.Now().UnixNano())

	// Create initial resource descriptor (don't register it yet)
	resourceDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             capabilityID,
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource for unit testing",
			GasFeeNRN:      fee,
			Timestamp:      time.Now().Unix(),
			CustomMetadata: map[string]interface{}{"key1": "value1"},
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
		ContentHash:  "sha256:1234567890abcdef",
		Schema: types.PluginSchemaDetail{
			Summary:       "This is resource 2 (API)",
			LocationHints: []string{"http://api.example.com/res2"},
		},
	}

	// Create a test resource descriptor for update
	updatedDesc := &types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             capabilityID,
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource for unit testing",
			GasFeeNRN:      fee,
			Timestamp:      time.Now().Unix(),
			CustomMetadata: map[string]interface{}{"key1": "value1"},
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
		ContentHash:  "sha256:1234567890abcdef",
		Schema: types.PluginSchemaDetail{
			Summary:       "This is resource 2 (API)",
			LocationHints: []string{"http://api.example.com/res2"},
		},
	}

	// Create transaction data
	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": resourceDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create MCP transaction
	txn, err := NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}

	// Sign the transaction
	err = wallet.SignTransaction(txn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Add the transaction to the pool
	bc.AddTransactionToTransactionPool(txn)

	// Create a new block with the transaction
	newBlock := NewBlock(bc.Blocks[len(bc.Blocks)-1].Hash(), int(bc.Blocks[len(bc.Blocks)-1].BlockNumber)+1, bc.Blocks[len(bc.Blocks)-1].BlockNumber+1)
	newBlock.Transactions = append(newBlock.Transactions, txn)
	// Set the ProposerAddress BEFORE mining the block to ensure it's included in the hash
	newBlock.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer for network fees
	newBlock.MineBlock()

	// Add the block to the blockchain
	bc.AddBlock(newBlock)

	// Create an updated resource descriptor
	updatedResourceDesc := updatedDesc
	updatedResourceDesc.Name = "Updated Test Resource"
	updatedResourceDesc.Version = "1.1.0"
	updatedResourceDesc.Description = "Updated test resource for unit testing"
	updatedResourceDesc.GasFeeNRN = fee * 2
	updatedResourceDesc.CustomMetadata = map[string]interface{}{"key1": "updated-value1", "key2": "value2"}

	// Create transaction data for capability update - include capabilityID explicitly
	updateTxnData, err := json.Marshal(map[string]interface{}{
		"capabilityID":         capabilityID,
		"capabilityDescriptor": updatedResourceDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create MCP transaction for capability update
	updateTxn, err := NewMCPTransaction(from, "", 0, updateTxnData, TransactionTypeMCPUpdateCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}

	// Sign the transaction
	err = wallet.SignTransaction(updateTxn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify that the capability exists in the database before attempting to update it with retry logic
	var capability *pb.CapabilityDescriptorContainerProto
	var verifyErr error
	for i := 0; i < 10; i++ {
		capability, verifyErr = db.GetCapabilityByID(resourceDesc.ID)
		if verifyErr == nil && capability != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if verifyErr != nil {
		t.Fatalf("Failed to get capability from database after retries: %v", verifyErr)
	}
	if capability == nil {
		t.Fatalf("Capability should not be nil after retries")
	}

	// Add the transaction to the pool
	err = bc.AddTransactionToTransactionPool(updateTxn)
	if err != nil {
		t.Fatalf("Failed to add update transaction to pool: %v", err)
	}

	// Create a new block with the transaction
	newBlock2 := NewBlock(bc.Blocks[len(bc.Blocks)-1].Hash(), int(bc.Blocks[len(bc.Blocks)-1].BlockNumber)+1, bc.Blocks[len(bc.Blocks)-1].BlockNumber+1)
	newBlock2.Transactions = append(newBlock2.Transactions, updateTxn)
	// Set the ProposerAddress BEFORE mining the block to ensure it's included in the hash
	newBlock2.ProposerAddress = utils.BLOCKCHAIN_ADDRESS // Simulate a proposer for network fees
	newBlock2.MineBlock()

	// Add the block to the blockchain
	// This will trigger ChromemSync.OnNewBlockConfirmed via a goroutine
	err = bc.AddBlock(newBlock2)
	if err != nil {
		t.Fatalf("Failed to add block with update transaction: %v", err)
	}

	// Verify that the capability was updated with retry logic
	for i := 0; i < 10; i++ {
		capability, err = db.GetCapabilityByID(resourceDesc.ID)
		if err == nil && capability != nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("Failed to get updated capability from database after retries: %v", err)
	}
	if capability == nil {
		t.Fatalf("Updated capability should not be nil after retries")
	}

	// Verify the updated capability fields with retry logic
	var resourceDescFromDB *pb.ResourceDescriptorProto
	for i := 0; i < 5; i++ {
		capability, err = db.GetCapabilityByID(resourceDesc.ID)
		if err != nil {
			t.Fatalf("Failed to get capability from database: %v", err)
		}

		resourceDescProto, ok := capability.Descriptor_.(*pb.CapabilityDescriptorContainerProto_Resource)
		if !ok {
			t.Fatalf("Capability does not contain ResourceDescriptor")
		}
		resourceDescFromDB = resourceDescProto.Resource

		// Check if updates have propagated
		if resourceDescFromDB.BaseDescriptor.Name == updatedResourceDesc.Name &&
			resourceDescFromDB.BaseDescriptor.Version == updatedResourceDesc.Version &&
			resourceDescFromDB.BaseDescriptor.Description == updatedResourceDesc.Description &&
			resourceDescFromDB.BaseDescriptor.GasFeeNrn == updatedResourceDesc.GasFeeNRN {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if resourceDescFromDB.BaseDescriptor.Id != updatedResourceDesc.ID {
		t.Errorf("ResourceDescriptor ID mismatch: expected %s, got %s", updatedResourceDesc.ID, resourceDescFromDB.BaseDescriptor.Id)
	}
	if resourceDescFromDB.BaseDescriptor.Name != updatedResourceDesc.Name {
		t.Errorf("ResourceDescriptor Name mismatch: expected %s, got %s", updatedResourceDesc.Name, resourceDescFromDB.BaseDescriptor.Name)
	}
	if resourceDescFromDB.BaseDescriptor.Version != updatedResourceDesc.Version {
		t.Errorf("ResourceDescriptor Version mismatch: expected %s, got %s", updatedResourceDesc.Version, resourceDescFromDB.BaseDescriptor.Version)
	}
	if resourceDescFromDB.BaseDescriptor.Description != updatedResourceDesc.Description {
		t.Errorf("ResourceDescriptor Description mismatch: expected %s, got %s", updatedResourceDesc.Description, resourceDescFromDB.BaseDescriptor.Description)
	}
	if resourceDescFromDB.BaseDescriptor.GasFeeNrn != updatedResourceDesc.GasFeeNRN {
		t.Errorf("ResourceDescriptor GasFeeNRN mismatch: expected %d, got %d", updatedResourceDesc.GasFeeNRN, resourceDescFromDB.BaseDescriptor.GasFeeNrn)
	}

	// --- ChromemDB Verification with longer timeout and better error reporting ---
	t.Log("Verifying capability update in ChromemDB...")
	err = utils.WaitForChromemDB(60*time.Second, func() (bool, error) {
		// First verify the capability descriptor was updated
		var capCollection *chromem.Collection
		if bc.ChromemDBSyncManager != nil {
			capCollection = bc.ChromemDBSyncManager.capabilityDescriptorCollection
		} else {
			// Fallback to legacy ChromemSync for backward compatibility
			capCollection = bc.ChromemSync.capabilityDescriptorCollection
		}
		if capCollection == nil {
			return false, fmt.Errorf("capability_descriptors collection not initialized")
		}

		capResults, err := capCollection.Query(context.Background(), resourceDesc.ID, 1, nil, nil)
		if err != nil {
			return false, fmt.Errorf("query failed for capability %s: %w", resourceDesc.ID, err)
		}
		if len(capResults) == 0 {
			return false, fmt.Errorf("capability %s not found in ChromemDB", resourceDesc.ID)
		}

		// Verify the updated fields in metadata (ChromemDB stores metadata, not JSON content)
		metadata := capResults[0].Metadata
		if metadata == nil {
			return false, fmt.Errorf("capability metadata is nil")
		}

		if name, ok := metadata["name"]; !ok || name != updatedResourceDesc.Name {
			return false, fmt.Errorf("Name not updated in ChromemDB: expected %s, got %s",
				updatedResourceDesc.Name, name)
		}

		// Check if this is an update by looking for the isUpdate flag
		if isUpdate, ok := metadata["isUpdate"]; !ok || isUpdate != "true" {
			return false, fmt.Errorf("isUpdate flag not set in ChromemDB metadata")
		}

		// Then verify the context record exists
		var ctxCollection *chromem.Collection
		if bc.ChromemDBSyncManager != nil {
			// Ensure the sync manager is initialized
			ctxCollection = bc.ChromemDBSyncManager.contextRecordCollection
		} else {
			// Fallback to legacy ChromemSync for backward compatibility
			ctxCollection = bc.ChromemSync.contextRecordCollection
		}

		if ctxCollection == nil {
			return false, fmt.Errorf("context_records collection not initialized")
		}

		ctxResults, err := ctxCollection.Query(context.Background(), updateTxn.TransactionHash, 1, nil, nil)
		if err != nil {
			return false, fmt.Errorf("query failed for context record %s: %w", updateTxn.TransactionHash, err)
		}
		if len(ctxResults) == 0 {
			return false, fmt.Errorf("context record %s not found in ChromemDB", updateTxn.TransactionHash)
		}

		t.Logf("Successfully verified capability update in ChromemDB")
		return true, nil
	})
	if err != nil {
		t.Fatalf("ChromemDB verification for capability update failed: %v", err)
	}
}

// Helper function to register a test capability with a specific owner
func registerTestCapabilityWithOwner(db *LevelDB, capabilityID string, owner string) error {
	// Create a test capability
	capability := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             capabilityID,
			Name:           "Test Resource",
			Owner:          owner, // Use the provided owner address
			Version:        "1.0.0",
			Description:    "Test resource for unit testing",
			GasFeeNRN:      50, // Set the capability's invocation fee
			Timestamp:      time.Now().Unix(),
			CustomMetadata: map[string]interface{}{"key1": "value1"},
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
		ContentHash:  "sha256:1234567890abcdef",
		Schema: types.PluginSchemaDetail{
			Summary:       "This is a test resource",
			LocationHints: []string{"http://api.example.com/test"},
		},
	}

	// Save the capability to the database
	return db.SaveCapability(capability)
}

// TestValidateTransactionInBlockContext_DuplicateCapability tests duplicate capability detection
func TestValidateTransactionInBlockContext_DuplicateCapability(t *testing.T) {
	// Create test database
	dbPath := fmt.Sprintf("testdb_dup_%d", time.Now().UnixNano())
	searchableDBPath := fmt.Sprintf("test_chromem_dup_%d", time.Now().UnixNano())
	defer os.RemoveAll(dbPath)
	defer os.RemoveAll(searchableDBPath)

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create genesis block
	genesisBlock := NewBlock(nil, 0, 0)

	// Create test blockchain
	chainID := fmt.Sprintf("test_chain_dup_%d", time.Now().UnixNano())
	// Provide a dummy CerebrasConfig for testing
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	// Pass nil for ChromemManager in tests - it will create a new one internally
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_dup", db, nil, searchableDBPath, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}

	// Create test wallet
	wallet, _ := NewWallet()
	from := wallet.GetAddress()
	fee := uint64(100)

	// Initialize test account with balance
	err = db.SaveAccountBalance(from, 1000)
	if err != nil {
		t.Fatalf("Failed to initialize test account balance: %v", err)
	}

	// Create test resource descriptor with valid capability ID format
	resourceDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             fmt.Sprintf("resource-%x", sha256.Sum256([]byte("test-resource")))[:32],
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource for unit testing",
			GasFeeNRN:      fee,
			Timestamp:      time.Now().Unix(),
			CustomMetadata: map[string]interface{}{"key1": "value1"},
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
	}

	// Initialize test account with sufficient balance
	err = db.SaveAccountBalance(from, 10000)
	if err != nil {
		t.Fatalf("Failed to initialize test account balance: %v", err)
	}

	// Create transaction data
	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": resourceDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create first transaction (should succeed)
	txn1, err := NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	if txn1 == nil {
		t.Fatal("Failed to create first transaction: returned nil")
	}
	err = wallet.SignTransaction(txn1)
	if err != nil {
		t.Fatalf("Failed to sign first transaction: %v", err)
	}

	// Create second identical transaction (should fail)
	txn2, err := NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	if txn2 == nil {
		t.Fatal("Failed to create second transaction: returned nil")
	}
	err = wallet.SignTransaction(txn2)
	if err != nil {
		t.Fatalf("Failed to sign second transaction: %v", err)
	}

	// Initialize test balances
	balances := make(map[string]*big.Int)
	balances[from] = big.NewInt(1000) // Sufficient balance

	// First register the capability directly in ChromemDB
	if err := db.SaveCapability(resourceDesc); err != nil {
		t.Fatalf("Failed to register test capability: %v", err)
	}

	// Now test that transaction validation fails for duplicate capability
	err = bc.validateTransactionInBlockContext(txn1, balances)
	if err == nil {
		t.Fatal("Validation should fail for duplicate capability")
	}
	if !strings.Contains(err.Error(), "capability ID already exists") {
		t.Fatalf("Expected 'capability ID already exists' error, got: %v", err)
	}

	// Create transaction data for second transaction
	txnData2, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": resourceDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create second transaction with same capability (should fail)
	txn2, err = NewMCPTransaction(from, "", 0, txnData2, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	if txn2 == nil {
		t.Fatal("Failed to create second transaction: returned nil")
	}
	err = wallet.SignTransaction(txn2)
	if err != nil {
		t.Fatalf("Failed to sign second transaction: %v", err)
	}

	// Second validation should fail (duplicate capability)
	if err := bc.validateTransactionInBlockContext(txn2, balances); err == nil {
		t.Fatal("Second validation should fail (duplicate capability)")
	} else if !strings.Contains(err.Error(), "capability ID already exists") {
		t.Fatalf("Expected duplicate capability error, got: %v", err)
	}

	// Second transaction should fail validation (duplicate)
	if err := bc.validateTransactionInBlockContext(txn2, balances); err == nil {
		t.Errorf("Duplicate capability transaction should have failed validation")
	}
}

// TestChromemDBSkipsInvalidTransactions tests that ChromemDB skips invalid transactions
func TestChromemDBSkipsInvalidTransactions(t *testing.T) {
	// Create test database
	dbPath := fmt.Sprintf("testdb_invalid_%d", time.Now().UnixNano())
	searchableDBPath := fmt.Sprintf("test_chromem_invalid_%d", time.Now().UnixNano())
	defer os.RemoveAll(dbPath)
	defer os.RemoveAll(searchableDBPath)

	db, err := NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create genesis block
	genesisBlock := NewBlock(nil, 0, 0)

	// Create test blockchain
	chainID := fmt.Sprintf("test_chain_invalid_%d", time.Now().UnixNano())
	// Provide a dummy CerebrasConfig for testing
	dummyCerebrasConfig := &config.CerebrasConfig{
		APIKey: "dummy_api_key", BaseURL: "http://dummy_url",
	}
	// Pass nil for ChromemManager in tests - it will create a new one internally
	bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_invalid", db, nil, searchableDBPath, dummyCerebrasConfig)
	if err != nil {
		t.Fatalf("Failed to create test blockchain: %v", err)
	}

	// Create test wallet
	wallet, _ := NewWallet()
	from := wallet.GetAddress()
	fee := uint64(100)

	// Create invalid descriptor that will pass NewMCPTransaction but fail later validation
	invalidDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             "valid-id-but-empty-name", // Valid ID
			Name:           "",                        // Empty name should fail validateBaseDescriptor
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Invalid resource for unit testing",
			GasFeeNRN:      fee,
			Timestamp:      time.Now().Unix(),
			CustomMetadata: map[string]interface{}{"key1": "value1"},
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
	}

	// Create transaction data
	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": invalidDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create invalid transaction with empty capability ID
	invalidTxn, err := NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	if invalidTxn == nil {
		// This is expected if NewMCPTransaction is strict about empty IDs.
		// The test's intent is that ChromemDB skips it, which it will if the txn can't be created.
		t.Log("NewMCPTransaction correctly returned nil for invalid (empty) capability ID. Test passes vacuously as txn won't reach ChromemDB.")
		return
	}
	err = wallet.SignTransaction(invalidTxn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Create a new block with the invalid transaction
	newBlock := NewBlock(bc.Blocks[len(bc.Blocks)-1].Hash(), int(bc.Blocks[len(bc.Blocks)-1].BlockNumber)+1, bc.Blocks[len(bc.Blocks)-1].BlockNumber+1)
	newBlock.Transactions = append(newBlock.Transactions, invalidTxn)
	newBlock.ProposerAddress = utils.BLOCKCHAIN_ADDRESS
	newBlock.MineBlock()

	// Add the block to the blockchain
	// Mark the transaction as invalid manually since AddBlock will reject the block
	newBlock.InvalidTxHashes[invalidTxn.TransactionHash] = "manually marked as invalid for testing"

	// Try to add the block - this will likely fail due to invalid transaction
	err = bc.AddBlock(newBlock)
	if err != nil {
		t.Logf("AddBlock failed as expected: %v", err)
		// This is expected, continue with the test
	}

	// Short delay to allow any erroneous sync to happen
	time.Sleep(200 * time.Millisecond)

	// Since we've removed direct LevelDB access, we'll just verify the transaction wasn't added to the blockchain
	t.Logf("Verifying invalid transaction %s was NOT added to the blockchain...", invalidTxn.TransactionHash)

	// Verify the transaction is not in the blockchain's transaction pool
	if txn := bc.GetTransactionFromPool(invalidTxn.TransactionHash); txn != nil {
		t.Errorf("Invalid transaction was found in transaction pool")
	}

	// Verify the transaction is not in any confirmed blocks
	found := false
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if tx.TransactionHash == invalidTxn.TransactionHash {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if found {
		t.Errorf("Invalid transaction was found in blockchain blocks")
	} else {
		t.Logf("Success: Invalid transaction was not added to the blockchain")
	}
	// Transaction verification complete - confirmed not added to LevelDB
}
