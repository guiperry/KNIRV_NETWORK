package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPoAValidator(t *testing.T) {
	validator := NewPoAValidator("node1", "node2", "node3")

	assert.NotNil(t, validator)
	assert.Equal(t, "node1", validator.GetNodeID())

	// Should include self and initial validators
	assert.True(t, validator.IsAuthorizedValidator("node1"))
	assert.True(t, validator.IsAuthorizedValidator("node2"))
	assert.True(t, validator.IsAuthorizedValidator("node3"))
	assert.False(t, validator.IsAuthorizedValidator("node4"))

	validators := validator.GetValidators()
	assert.Len(t, validators, 3)
}

func TestPoAValidator_AddRemoveValidator(t *testing.T) {
	validator := NewPoAValidator("node1")

	// Add new validator
	err := validator.AddValidator("node2")
	require.NoError(t, err)
	assert.True(t, validator.IsAuthorizedValidator("node2"))

	// Try to add duplicate
	err = validator.AddValidator("node2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already authorized")

	// Try to add empty ID
	err = validator.AddValidator("")
	assert.Error(t, err)

	// Remove validator
	err = validator.RemoveValidator("node2")
	require.NoError(t, err)
	assert.False(t, validator.IsAuthorizedValidator("node2"))

	// Try to remove self
	err = validator.RemoveValidator("node1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove self")

	// Try to remove non-existent
	err = validator.RemoveValidator("node3")
	assert.Error(t, err)
}

func createValidBlock(_ *testing.T, prevBlock *Block) *Block {
	blockID := uuid.New()
	timestamp := time.Now().Unix()

	// Ensure timestamp is after previous block
	if prevBlock != nil && timestamp <= prevBlock.Timestamp {
		timestamp = prevBlock.Timestamp + 1
	}

	// Create sample payload
	data := []byte("test memory data")
	payloadHash := sha256.Sum256(data)
	payloadHashStr := hex.EncodeToString(payloadHash[:])

	// Calculate NRN cost
	nrnCost := CalculateNRNCost(len(data))

	// Create normalized vector
	vector := make([]float32, 768)
	for i := range vector {
		vector[i] = 1.0 / 27.712812921102035 // Normalized to L2 norm = 1
	}

	block := &Block{
		BlockID:        blockID,
		Timestamp:      timestamp,
		PrevHash:       "",
		PayloadHash:    payloadHashStr,
		Data:           data,
		Category:       CategoryGeneral,
		NRNCost:        nrnCost,
		UserID:         "node1",
		SemanticVector: vector,
	}

	if prevBlock != nil {
		block.PrevHash = CreateBlockHash(prevBlock)
	}

	return block
}

func TestPoAValidator_ValidateBlock_Success(t *testing.T) {
	validator := NewPoAValidator("node1")

	// Create valid genesis block
	block := createValidBlock(t, nil)

	err := validator.ValidateBlock(block, nil)
	require.NoError(t, err)
}

func TestPoAValidator_ValidateBlock_InvalidBlockID(t *testing.T) {
	validator := NewPoAValidator("node1")

	block := createValidBlock(t, nil)
	block.BlockID = uuid.Nil

	err := validator.ValidateBlock(block, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid block ID")
}

func TestPoAValidator_ValidateTimestamp(t *testing.T) {
	validator := NewPoAValidator("node1")

	// Test timestamp in future
	t.Run("future timestamp", func(t *testing.T) {
		block := createValidBlock(t, nil)
		block.Timestamp = time.Now().Unix() + 400 // 6 minutes in future

		err := validator.ValidateBlock(block, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too far in the future")
	})

	// Test timestamp in past
	t.Run("past timestamp", func(t *testing.T) {
		block := createValidBlock(t, nil)
		block.Timestamp = time.Now().Unix() - 90000 // > 24 hours ago

		err := validator.ValidateBlock(block, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too far in the past")
	})

	// Test timestamp before previous block
	t.Run("before previous block", func(t *testing.T) {
		prevBlock := createValidBlock(t, nil)
		time.Sleep(10 * time.Millisecond)

		block := createValidBlock(t, prevBlock)
		block.Timestamp = prevBlock.Timestamp - 1

		err := validator.ValidateBlock(block, prevBlock)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be after previous block")
	})
}

func TestPoAValidator_ValidatePreviousHash(t *testing.T) {
	validator := NewPoAValidator("node1")

	// Test genesis block with non-empty previous hash
	t.Run("genesis with previous hash", func(t *testing.T) {
		block := createValidBlock(t, nil)
		block.PrevHash = "shouldbeempty"

		err := validator.ValidateBlock(block, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "genesis block must have empty previous hash")
	})

	// Test incorrect previous hash
	t.Run("incorrect previous hash", func(t *testing.T) {
		prevBlock := createValidBlock(t, nil)
		time.Sleep(10 * time.Millisecond)

		block := createValidBlock(t, prevBlock)
		block.PrevHash = "incorrecthash"

		err := validator.ValidateBlock(block, prevBlock)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "previous hash mismatch")
	})
}

func TestPoAValidator_ValidatePayloadHash(t *testing.T) {
	validator := NewPoAValidator("node1")

	block := createValidBlock(t, nil)
	block.PayloadHash = "incorrecthash"

	err := validator.ValidateBlock(block, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payload hash mismatch")
}

func TestPoAValidator_ValidateNRNCost(t *testing.T) {
	validator := NewPoAValidator("node1")

	// Test incorrect NRN cost
	t.Run("incorrect cost", func(t *testing.T) {
		block := createValidBlock(t, nil)
		block.NRNCost = 999

		err := validator.ValidateBlock(block, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "NRN cost mismatch")
	})

	// Test cost calculation for various sizes
	t.Run("cost calculation", func(t *testing.T) {
		testCases := []struct {
			size         int
			expectedCost uint64
		}{
			{100, 1},    // < 1 KB
			{1024, 1},   // = 1 KB
			{1025, 2},   // > 1 KB
			{2048, 2},   // = 2 KB
			{10240, 10}, // = 10 KB
			{10241, 11}, // > 10 KB
		}

		for _, tc := range testCases {
			cost := CalculateNRNCost(tc.size)
			assert.Equal(t, tc.expectedCost, cost,
				"Size %d bytes should cost %d NRN, got %d", tc.size, tc.expectedCost, cost)
		}
	})
}

func TestPoAValidator_ValidateCategory(t *testing.T) {
	validator := NewPoAValidator("node1")

	validCategories := []MemoryCategory{
		CategoryError, CategoryContext, CategoryIdea, CategoryTask, CategoryGeneral,
	}

	// Test valid categories
	for _, category := range validCategories {
		block := createValidBlock(t, nil)
		block.Category = category

		err := validator.ValidateBlock(block, nil)
		require.NoError(t, err, "Category %s should be valid", category)
	}

	// Test invalid category
	block := createValidBlock(t, nil)
	block.Category = "INVALID_CATEGORY"

	err := validator.ValidateBlock(block, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid category")
}

func TestPoAValidator_ValidateSemanticVector(t *testing.T) {
	validator := NewPoAValidator("node1")

	// Test nil vector
	t.Run("nil vector", func(t *testing.T) {
		block := createValidBlock(t, nil)
		block.SemanticVector = nil

		err := validator.ValidateBlock(block, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "semantic vector is nil")
	})

	// Test wrong dimension
	t.Run("wrong dimension", func(t *testing.T) {
		block := createValidBlock(t, nil)
		block.SemanticVector = make([]float32, 100) // Wrong size

		err := validator.ValidateBlock(block, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dimension mismatch")
	})

	// Test non-normalized vector
	t.Run("non-normalized vector", func(t *testing.T) {
		block := createValidBlock(t, nil)
		// Create vector with L2 norm != 1
		block.SemanticVector = make([]float32, 768)
		for i := range block.SemanticVector {
			block.SemanticVector[i] = 0.5
		}

		err := validator.ValidateBlock(block, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not normalized")
	})
}

func TestPoAValidator_ValidateUnauthorizedValidator(t *testing.T) {
	validator := NewPoAValidator("node1", "node2")

	block := createValidBlock(t, nil)
	block.UserID = "node3" // Not authorized

	err := validator.ValidateBlock(block, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized validator")
}

func TestPoAValidator_ValidateBlockChain(t *testing.T) {
	validator := NewPoAValidator("node1")

	// Create a chain of 5 blocks
	blocks := make([]*Block, 5)
	blocks[0] = createValidBlock(t, nil)

	for i := 1; i < 5; i++ {
		time.Sleep(10 * time.Millisecond)
		blocks[i] = createValidBlock(t, blocks[i-1])
	}

	// Validate the entire chain
	err := validator.ValidateBlockChain(blocks)
	require.NoError(t, err)

	// Test with invalid block in middle
	blocks[2].PayloadHash = "corrupted"
	err = validator.ValidateBlockChain(blocks)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block 2 validation failed")
}

func TestPoAValidator_ValidateBlockChain_EmptyChain(t *testing.T) {
	validator := NewPoAValidator("node1")

	err := validator.ValidateBlockChain([]*Block{})
	require.NoError(t, err) // Empty chain is valid
}

func TestCreateBlockHash(t *testing.T) {
	block := createValidBlock(t, nil)

	hash1 := CreateBlockHash(block)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA256 hex = 64 chars

	// Same block should produce same hash
	hash2 := CreateBlockHash(block)
	assert.Equal(t, hash1, hash2)

	// Different block should produce different hash
	block2 := createValidBlock(t, nil)
	hash3 := CreateBlockHash(block2)
	assert.NotEqual(t, hash1, hash3)
}

func TestPoAValidator_ComputeL2Norm(t *testing.T) {
	validator := NewPoAValidator("node1")

	// Test normalized vector
	vector := make([]float32, 768)
	for i := range vector {
		vector[i] = 1.0 / 27.712812921102035 // L2 norm = 1
	}

	norm := validator.computeL2Norm(vector)
	assert.InDelta(t, 1.0, norm, 0.01) // Should be ~1.0

	// Test unnormalized vector
	vector2 := make([]float32, 768)
	for i := range vector2 {
		vector2[i] = 0.5
	}

	norm2 := validator.computeL2Norm(vector2)
	assert.Greater(t, norm2, 1.01) // Should be > 1
}

func TestPoAValidator_Integration(t *testing.T) {
	// Create validator with multiple nodes
	validator := NewPoAValidator("node1", "node2", "node3")

	// Create and validate a chain of blocks from different validators
	blocks := make([]*Block, 6)

	// Genesis block from node1
	blocks[0] = createValidBlock(t, nil)
	blocks[0].UserID = "node1"
	err := validator.ValidateBlock(blocks[0], nil)
	require.NoError(t, err)

	// Subsequent blocks from different validators
	userIDs := []string{"node2", "node3", "node1", "node2", "node3"}
	for i := 1; i < 6; i++ {
		time.Sleep(10 * time.Millisecond)
		blocks[i] = createValidBlock(t, blocks[i-1])
		blocks[i].UserID = userIDs[i-1]

		err := validator.ValidateBlock(blocks[i], blocks[i-1])
		require.NoError(t, err, "Block %d from validator %s should be valid",
			i, userIDs[i-1])
	}

	// Validate entire chain
	err = validator.ValidateBlockChain(blocks)
	require.NoError(t, err)

	// Try to add block from unauthorized validator
	unauthorizedBlock := createValidBlock(t, blocks[5])
	unauthorizedBlock.UserID = "node4"
	err = validator.ValidateBlock(unauthorizedBlock, blocks[5])
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized validator")

	// Add node4 as validator and try again
	err = validator.AddValidator("node4")
	require.NoError(t, err)

	err = validator.ValidateBlock(unauthorizedBlock, blocks[5])
	require.NoError(t, err) // Should now be valid

	// Remove node4 and validation should fail again
	err = validator.RemoveValidator("node4")
	require.NoError(t, err)

	err = validator.ValidateBlock(unauthorizedBlock, blocks[5])
	assert.Error(t, err)
}
