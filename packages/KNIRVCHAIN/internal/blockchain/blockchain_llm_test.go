package blockchain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockchainLLM_GetLLMTransactionByCMU(t *testing.T) {
	// Create a test blockchain
	bc := createTestBlockchain()

	// Create and add an LLM rooting transaction
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  "TestOwner",
		APIEndpoint: "https://api.test.com",
		MetadataCID: "QmTest123",
	}

	// Use "blockchain" as sender to bypass signature verification in tests
	tx, err := NewLLMRootingTransaction("blockchain", llmData, 100)
	require.NoError(t, err)

	// Add transaction to pool
	err = bc.AddTransactionToTransactionPool(tx)
	require.NoError(t, err)

	// Create a block with the transaction using ProposePoAuDBlock
	block, err := bc.ProposePoAuDBlock("validator1")
	require.NoError(t, err)
	require.NotNil(t, block)

	// Add block to chain
	err = bc.AddBlock(block)
	require.NoError(t, err)

	// Now test GetLLMTransactionByCMU
	var txData LLMRootingData
	err = json.Unmarshal(tx.Data, &txData)
	require.NoError(t, err)

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
	bc := createTestBlockchain()

	// Create multiple transactions with same model hash
	llmData1 := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  "TestOwner",
		APIEndpoint: "https://api.test.com/v1",
		MetadataCID: "QmTest123",
	}

	llmData2 := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  "TestOwner",
		APIEndpoint: "https://api.test.com/v2", // Different endpoint
		MetadataCID: "QmTest123",               // Same metadata
	}

	tx1, err := NewLLMRootingTransaction(llmData1.ModelOwner, llmData1, 100)
	require.NoError(t, err)

	tx2, err := NewLLMRootingTransaction(llmData2.ModelOwner, llmData2, 100)
	require.NoError(t, err)

	// Add transactions to pool
	err = bc.AddTransactionToTransactionPool(tx1)
	require.NoError(t, err)
	err = bc.AddTransactionToTransactionPool(tx2)
	require.NoError(t, err)

	// Create blocks (ProposePoAuDBlock takes all pending transactions)
	block1, err := bc.ProposePoAuDBlock("validator1")
	require.NoError(t, err)
	require.NotNil(t, block1)

	// Add first block
	err = bc.AddBlock(block1)
	require.NoError(t, err)

	// Add second transaction and create another block
	err = bc.AddTransactionToTransactionPool(tx2)
	require.NoError(t, err)

	block2, err := bc.ProposePoAuDBlock("validator1")
	require.NoError(t, err)
	require.NotNil(t, block2)

	err = bc.AddBlock(block2)
	require.NoError(t, err)

	// Get model hash from first transaction
	var txData1 LLMRootingData
	err = json.Unmarshal(tx1.Data, &txData1)
	require.NoError(t, err)

	modelHash := txData1.CMU[len("knirv://mainnet/"):]

	// Test GetLLMTransactionsByModelHash
	transactions, err := bc.GetLLMTransactionsByModelHash(modelHash)
	require.NoError(t, err)
	assert.True(t, len(transactions) >= 1) // At least one transaction should be found
}

func TestBlockchainLLM_ResolveCMU(t *testing.T) {
	// Create a test blockchain
	bc := createTestBlockchain()

	// Create and add an LLM rooting transaction
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  "TestOwner",
		APIEndpoint: "https://api.test.com/chat",
		MetadataCID: "QmTest123",
	}

	tx, err := NewLLMRootingTransaction(llmData.ModelOwner, llmData, 100)
	require.NoError(t, err)

	// Add transaction to pool
	err = bc.AddTransactionToTransactionPool(tx)
	require.NoError(t, err)

	// Create and add block
	block, err := bc.ProposePoAuDBlock("validator1")
	require.NoError(t, err)
	require.NotNil(t, block)
	err = bc.AddBlock(block)
	require.NoError(t, err)

	// Get CMU
	var txData LLMRootingData
	err = json.Unmarshal(tx.Data, &txData)
	require.NoError(t, err)

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
	bc := createTestBlockchain()

	// Initially CMU should not exist
	assert.False(t, bc.CheckCMUExists("knirv://mainnet/test"))

	// Create and add an LLM rooting transaction
	llmData := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  "TestOwner",
		APIEndpoint: "https://api.test.com",
		MetadataCID: "QmTest123",
	}

	tx, err := NewLLMRootingTransaction(llmData.ModelOwner, llmData, 100)
	require.NoError(t, err)

	// Add transaction to pool
	err = bc.AddTransactionToTransactionPool(tx)
	require.NoError(t, err)

	// Create and add block
	block, err := bc.ProposePoAuDBlock("validator1")
	require.NoError(t, err)
	require.NotNil(t, block)
	err = bc.AddBlock(block)
	require.NoError(t, err)

	// Get CMU
	var txData LLMRootingData
	err = json.Unmarshal(tx.Data, &txData)
	require.NoError(t, err)

	// Now CMU should exist
	assert.True(t, bc.CheckCMUExists(txData.CMU))
	assert.False(t, bc.CheckCMUExists("knirv://mainnet/nonexistent"))
}

func TestBlockchainLLM_CMUUniqueness(t *testing.T) {
	// Create a test blockchain
	bc := createTestBlockchain()

	// Create first LLM rooting transaction
	llmData1 := LLMRootingData{
		ModelName:   "TestModel",
		ModelOwner:  "TestOwner",
		APIEndpoint: "https://api.test.com",
		MetadataCID: "QmTest123",
	}

	tx1, err := NewLLMRootingTransaction(llmData1.ModelOwner, llmData1, 100)
	require.NoError(t, err)

	// Add first transaction
	err = bc.AddTransactionToTransactionPool(tx1)
	require.NoError(t, err)

	block1, err := bc.ProposePoAuDBlock("validator1")
	require.NoError(t, err)
	require.NotNil(t, block1)
	err = bc.AddBlock(block1)
	require.NoError(t, err)

	// Try to add another transaction with same CMU (same data)
	llmData2 := LLMRootingData{
		ModelName:   "TestModel",            // Same
		ModelOwner:  "TestOwner",            // Same
		APIEndpoint: "https://api.test.com", // Same
		MetadataCID: "QmTest123",            // Same
	}

	tx2, err := NewLLMRootingTransaction(llmData2.ModelOwner, llmData2, 100)
	require.NoError(t, err)

	// This should fail because CMU already exists
	err = bc.AddTransactionToTransactionPool(tx2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CMU already exists")
}

// Helper function to create a test blockchain
func createTestBlockchain() *BlockchainStruct {
	// This is a simplified test blockchain creation
	// In a real test, you'd use proper initialization
	bc := &BlockchainStruct{
		TransactionPool:        []*Transaction{},
		Blocks:                 []*Block{},
		ChainAddress:           "test-chain",
		Reflections:            make(map[string]bool),
		MiningLocked:           false,
		OwnerAddress:           "test-owner",
		WalletAddress:          "test-wallet",
		txnSignal:              make(chan struct{}, 1),
		isActivelyMining:       false,
		NetworkAuthors:         make(map[string]bool),
		PoAuDEnabled:           false,
		TransactionPoolManager: NewTransactionPoolManager(nil), // Will be set to bc later
	}

	// Add genesis block
	genesisBlock := NewBlock([]byte{}, 0, 0)
	genesisBlock.ProposerAddress = "genesis"
	bc.Blocks = append(bc.Blocks, genesisBlock)

	// Set self-reference for transaction pool manager
	bc.TransactionPoolManager.blockchain = bc

	// Add test validator
	bc.AddNetworkAuthor("validator1")

	return bc
}
