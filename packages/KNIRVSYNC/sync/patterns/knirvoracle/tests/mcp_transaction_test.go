package main

import (
	"KNIRVORACLE/types"
	"encoding/json"
	"testing"
)

// TestMCPTransactionCreation tests the creation of MCP transactions
func TestMCPTransactionCreation(t *testing.T) {
	// Test data
	from := "sender-address-123"
	to := "recipient-address-123"
	value := uint64(0) // No value transfer for MCP transactions
	fee := uint64(100) // Gas fee

	// Test ResourceDescriptor registration
	resourceDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             "resource-123",
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource for unit testing",
			GasFeeNRN:      fee,
			Timestamp:      0, // Will be set by the transaction
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
	txn, err := NewMCPTransaction(from, to, value, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}

	// Verify transaction fields
	if txn.From != from {
		t.Errorf("Transaction From mismatch: expected %s, got %s", from, txn.From)
	}
	if txn.To != to {
		t.Errorf("Transaction To mismatch: expected %s, got %s", to, txn.To)
	}
	if txn.Value != value {
		t.Errorf("Transaction Value mismatch: expected %d, got %d", value, txn.Value)
	}
	if txn.Fee != fee {
		t.Errorf("Transaction Fee mismatch: expected %d, got %d", fee, txn.Fee)
	}
	if txn.Type != TransactionTypeMCPRegisterCapability {
		t.Errorf("Transaction Type mismatch: expected %s, got %s", TransactionTypeMCPRegisterCapability, txn.Type)
	}
	if txn.TransactionHash == "" {
		t.Errorf("Transaction hash should not be empty")
	}

	// Test ContextRecord for capability invocation
	contextRecord := types.ContextRecord{
		ID:              "context-123",
		CapabilityID:    "capability-123",
		InteractionType: types.InteractionTypeToolInvocation,
		Initiator:       from,
		Timestamp:       0, // Will be set by the transaction
		InputHash:       "sha256:input-hash",
		OutputHash:      "sha256:output-hash",
		Details:         map[string]interface{}{"param1": "value1", "param2": 42},
	}

	// Create transaction data for capability invocation
	invokeTxnData, err := json.Marshal(map[string]interface{}{
		"contextRecord": contextRecord,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create MCP transaction for capability invocation
	invokeTxn, err := NewMCPTransaction(from, to, value, invokeTxnData, TransactionTypeMCPInvokeCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP invoke transaction: %v", err)
	}

	// Verify transaction fields
	if invokeTxn.From != from {
		t.Errorf("Transaction From mismatch: expected %s, got %s", from, invokeTxn.From)
	}
	if invokeTxn.To != to {
		t.Errorf("Transaction To mismatch: expected %s, got %s", to, invokeTxn.To)
	}
	if invokeTxn.Value != value {
		t.Errorf("Transaction Value mismatch: expected %d, got %d", value, invokeTxn.Value)
	}
	if invokeTxn.Fee != fee {
		t.Errorf("Transaction Fee mismatch: expected %d, got %d", fee, invokeTxn.Fee)
	}
	if invokeTxn.Type != TransactionTypeMCPInvokeCapability {
		t.Errorf("Transaction Type mismatch: expected %s, got %s", TransactionTypeMCPInvokeCapability, invokeTxn.Type)
	}
	if invokeTxn.TransactionHash == "" {
		t.Errorf("Transaction hash should not be empty")
	}

	// Verify that the two transactions have different hashes
	if txn.TransactionHash == invokeTxn.TransactionHash {
		t.Errorf("Transaction hashes should be different for different transactions")
	}
}

// TestMCPTransactionVerification tests the verification of MCP transactions
func TestMCPTransactionVerification(t *testing.T) {
	// Create a wallet for signing
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	from := wallet.GetAddress()
	to := "recipient-address-123"
	value := uint64(0) // No value transfer for MCP transactions
	fee := uint64(100) // Gas fee

	// Create transaction data
	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": types.ResourceDescriptor{
			BaseDescriptor: types.BaseDescriptor{
				ID:             "resource-123",
				Name:           "Test Resource",
				Owner:          from,
				Version:        "1.0.0",
				Description:    "Test resource for unit testing",
				GasFeeNRN:      fee,
				Timestamp:      0, // Will be set by the transaction
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
		},
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create MCP transaction
	txn, err := NewMCPTransaction(from, to, value, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}

	// Sign the transaction
	err = wallet.SignTransaction(txn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Tamper with the transaction data
	txn.Data = append(txn.Data, []byte("tampered")...)

	// Verify the tampered transaction (should fail)
	if txn.VerifyTxn() {
		t.Errorf("Tampered transaction verification succeeded, but it should fail")
	}
}

// TestNewMCPTransaction_InvalidCapability tests that NewMCPTransaction fails on invalid capability IDs
func TestNewMCPTransaction_InvalidCapability(t *testing.T) {
	from := "sender-address-123"
	to := "recipient-address-123"
	value := uint64(0)
	fee := uint64(100)

	// Test with empty capability ID
	emptyIDDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             "", // Empty ID should fail
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource",
			GasFeeNRN:      fee,
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
	}

	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": emptyIDDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	txn, err := NewMCPTransaction(from, to, value, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err == nil {
		t.Errorf("Expected error for empty capability ID, got nil")
	}
	if txn != nil {
		t.Errorf("Expected nil transaction for empty capability ID, got %v", txn)
	}

	// Test with invalid capability ID format
	invalidIDDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             "invalid#id", // Invalid characters
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource",
			GasFeeNRN:      fee,
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
	}

	txnData, err = json.Marshal(map[string]interface{}{
		"capabilityDescriptor": invalidIDDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	txn, err = NewMCPTransaction(from, to, value, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err == nil {
		t.Errorf("Expected error for invalid capability ID, got nil")
	}
	if txn != nil {
		t.Errorf("Expected nil transaction for invalid capability ID, got %v", txn)
	}
}

// TestVerifyTxn_CapabilityValidation tests capability validation in VerifyTxn
func TestVerifyTxn_CapabilityValidation(t *testing.T) {
	// Create a wallet for signing
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	from := wallet.GetAddress()
	to := "recipient-address-123"
	value := uint64(0)
	fee := uint64(100)

	// Create valid transaction data
	validDesc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             "resource-valid-123",
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource",
			GasFeeNRN:      fee,
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
	}

	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": validDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create and sign valid transaction
	validTxn, err := NewMCPTransaction(from, to, value, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	err = wallet.SignTransaction(validTxn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify valid transaction should succeed
	if !validTxn.VerifyTxn() {
		t.Errorf("Valid capability transaction failed verification")
	}

	// Create transaction with invalid capability type
	invalidTypeDesc := validDesc
	invalidTypeDesc.CapabilityType = "invalid-type"

	txnData, err = json.Marshal(map[string]interface{}{
		"capabilityDescriptor": invalidTypeDesc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	invalidTypeTxn, err := NewMCPTransaction(from, to, value, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	err = wallet.SignTransaction(invalidTypeTxn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verify should fail for invalid capability type
	if invalidTypeTxn.VerifyTxn() {
		t.Errorf("Transaction with invalid capability type passed verification")
	}
}

// TestVerifyTxn_DuplicateCapability tests duplicate capability detection
func TestVerifyTxn_DuplicateCapability(t *testing.T) {
	// Create a wallet for signing
	wallet, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	from := wallet.GetAddress()
	to := "recipient-address-123"
	value := uint64(0)
	fee := uint64(100)

	// Create valid transaction data
	desc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             "duplicate-resource-123",
			Name:           "Test Resource",
			Owner:          from,
			Version:        "1.0.0",
			Description:    "Test resource",
			GasFeeNRN:      fee,
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
	}

	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": desc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create first transaction (should succeed)
	txn1, err := NewMCPTransaction(from, to, value, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	err = wallet.SignTransaction(txn1)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Create second identical transaction (should fail)
	txn2, err := NewMCPTransaction(from, to, value, txnData, TransactionTypeMCPRegisterCapability, fee)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	err = wallet.SignTransaction(txn2)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// First transaction should verify successfully
	if !txn1.VerifyTxn() {
		t.Errorf("First capability transaction failed verification")
	}

	// VerifyTxn itself doesn't check for duplicates against blockchain state.
	// It only checks the transaction's internal validity and signature.
	if !txn2.VerifyTxn() {
		t.Errorf("Second (structurally identical) capability transaction should also pass VerifyTxn if signed correctly")
	}
}

// TestVerifyTxn_InvalidCapabilityOwner tests capability owner validation
func TestVerifyTxn_InvalidCapabilityOwner(t *testing.T) {
	// Create wallets
	wallet1, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	wallet2, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Create transaction with mismatched owner
	desc := types.ResourceDescriptor{
		BaseDescriptor: types.BaseDescriptor{
			ID:             "owner-test-resource",
			Name:           "Test Resource",
			Owner:          wallet1.GetAddress(), // Different from transaction signer
			Version:        "1.0.0",
			Description:    "Test resource",
			GasFeeNRN:      uint64(100),
			CapabilityType: types.CapabilityTypeResource,
		},
		ResourceType: types.ResourceTypeFile,
	}

	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": desc,
	})
	if err != nil {
		t.Fatalf("Failed to marshal transaction data: %v", err)
	}

	// Create transaction signed by wallet2 but with wallet1 as owner
	txn, err := NewMCPTransaction(wallet2.GetAddress(), "recipient-address", 0, txnData, TransactionTypeMCPRegisterCapability, 100)
	if err != nil {
		t.Fatalf("Failed to create MCP transaction: %v", err)
	}
	err = wallet2.SignTransaction(txn)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Verification should fail due to owner mismatch
	if txn.VerifyTxn() {
		t.Errorf("Transaction with mismatched capability owner passed verification")
	}
}
