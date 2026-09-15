package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockchainLLM_GetLLMTransactionByCMU(t *testing.T) {
	// Create a test blockchain
	bc := createTestBlockchain(t)

	// Create and add an LLM rooting transaction signed by the model owner
	signer := newTestSigner(t)
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  signer.address,
		APIEndpoint: "https://api.test.com",
		MetadataCID: "QmTest123",
	}

	tx := rootModel(t, bc, signer, llmData)

	// Now test GetLLMTransactionByCMU
	txData := llmRootingDataFromTx(t, tx)
	require.NotEmpty(t, txData.CMU)

	foundTx, err := bc.GetLLMTransactionByCMU(txData.CMU)
	require.NoError(t, err)
	assert.NotNil(t, foundTx)
	assert.Equal(t, tx.TransactionHash, foundTx.TransactionHash)

	// Test with non-existent CMU
	_, err = bc.GetLLMTransactionByCMU("knirv://mainnet/nonexistent")
	assert.Error(t, err)
}

func TestBlockchainLLM_GetLLMTransactionsByModelHash(t *testing.T) {
	// Create a test blockchain
	bc := createTestBlockchain(t)

	// Root a single model; the CMU contains the model hash.
	signer := newTestSigner(t)
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  signer.address,
		APIEndpoint: "https://api.test.com/v1",
		MetadataCID: "QmTest123",
	}

	tx := rootModel(t, bc, signer, llmData)

	txData := llmRootingDataFromTx(t, tx)
	modelHash := txData.CMU[len("knirv://mainnet/"):]

	// Test GetLLMTransactionsByModelHash
	transactions, err := bc.GetLLMTransactionsByModelHash(modelHash)
	require.NoError(t, err)
	assert.True(t, len(transactions) >= 1) // At least one transaction should be found
	found := false
	for _, candidate := range transactions {
		if candidate.TransactionHash == tx.TransactionHash {
			found = true
			break
		}
	}
	assert.True(t, found, "rooted transaction should be returned for its own model hash")

	// Non-existent model hash yields an empty result
	empty, err := bc.GetLLMTransactionsByModelHash("0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestBlockchainLLM_ResolveCMU(t *testing.T) {
	// Create a test blockchain
	bc := createTestBlockchain(t)

	// Create and add an LLM rooting transaction signed by the model owner
	signer := newTestSigner(t)
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  signer.address,
		APIEndpoint: "https://api.test.com/chat",
		MetadataCID: "QmTest123",
	}

	tx := rootModel(t, bc, signer, llmData)

	// Get CMU
	txData := llmRootingDataFromTx(t, tx)

	// Test ResolveCMU
	endpoint, err := bc.ResolveCMU(txData.CMU)
	require.NoError(t, err)
	assert.Equal(t, llmData.APIEndpoint, endpoint)

	// Test with invalid CMU
	_, err = bc.ResolveCMU("invalid-cmu")
	assert.Error(t, err)

	// Test with non-existent CMU
	_, err = bc.ResolveCMU("knirv://mainnet/nonexistent")
	assert.Error(t, err)
}

func TestBlockchainLLM_CheckCMUExists(t *testing.T) {
	// Create a test blockchain
	bc := createTestBlockchain(t)

	// Initially CMU should not exist
	assert.False(t, bc.CheckCMUExists("knirv://mainnet/test"))

	// Create and add an LLM rooting transaction signed by the model owner
	signer := newTestSigner(t)
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  signer.address,
		APIEndpoint: "https://api.test.com",
		MetadataCID: "QmTest123",
	}

	tx := rootModel(t, bc, signer, llmData)

	// Get CMU
	txData := llmRootingDataFromTx(t, tx)

	// Now CMU should exist
	assert.True(t, bc.CheckCMUExists(txData.CMU))
	assert.False(t, bc.CheckCMUExists("knirv://mainnet/nonexistent"))
}

func TestBlockchainLLM_CMUUniqueness(t *testing.T) {
	// Create a test blockchain
	bc := createTestBlockchain(t)
	signer := newTestSigner(t)

	// Create first LLM rooting transaction
	llmData1 := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  signer.address,
		APIEndpoint: "https://api.test.com",
		MetadataCID: "QmTest123",
	}

	firstTx := rootModel(t, bc, signer, llmData1)
	firstCMU := llmRootingDataFromTx(t, firstTx).CMU

	// Try to add another rooting for the same CMU (same model metadata, but a
	// different endpoint so the transaction bytes/hash differ).
	llmData2 := LLMRootingData{
		ModelName:   "TestModel",                // Same
		ModelOwner:  signer.address,             // Same
		APIEndpoint: "https://api.test.com/alt", // Different -> distinct tx
		MetadataCID: "QmTest123",                // Same -> same CMU
	}
	tx2 := signer.newSignedLLMRooting(t, llmData2, 0)
	require.NotEqual(t, firstTx.TransactionHash, tx2.TransactionHash)

	// This should fail because CMU already exists
	err := bc.AddTransactionToTransactionPool(tx2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CMU already exists")
	assert.Contains(t, err.Error(), firstCMU)
}
