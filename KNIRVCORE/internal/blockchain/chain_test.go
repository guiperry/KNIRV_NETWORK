package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knirvchain/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestChainNode(t *testing.T) (*ChainNode, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	stor, err := storage.NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)

	node, err := NewChainNode("test-node", stor)
	require.NoError(t, err)

	cleanup := func() {
		stor.Close()
	}

	return node, cleanup
}

// createTestBlock creates a valid block for testing
func createTestBlock(_ *testing.T, data []byte, category MemoryCategory, prevBlock *Block) *Block {
	blockID := uuid.New()
	timestamp := time.Now().Unix()

	// Ensure timestamp is after previous block
	if prevBlock != nil && timestamp <= prevBlock.Timestamp {
		timestamp = prevBlock.Timestamp + 1
	}

	// Calculate payload hash
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
		Category:       category,
		NRNCost:        nrnCost,
		UserID:         "test-node",
		SemanticVector: vector,
	}

	if prevBlock != nil {
		block.PrevHash = CreateBlockHash(prevBlock)
	}

	return block
}

func TestNewChainNode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "testdb")

	stor, err := storage.NewKNIRVBASEStorage(dbPath)
	require.NoError(t, err)
	defer stor.Close()

	node, err := NewChainNode("localhost:8080", stor)
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.Equal(t, "localhost:8080", node.nodeID)
	assert.NotNil(t, node.blockStore)
}

func TestChainNode_CommitBlock(t *testing.T) {
	node, cleanup := setupTestChainNode(t)
	defer cleanup()

	ctx := context.Background()

	block := createTestBlock(t, []byte("test data"), CategoryGeneral, nil)

	err := node.CommitBlock(ctx, block)
	assert.NoError(t, err)

	// Verify block was stored
	stored, err := node.GetBlock(ctx, block.BlockID)
	assert.NoError(t, err)
	assert.Equal(t, block.BlockID, stored.BlockID)
	assert.Equal(t, block.Timestamp, stored.Timestamp)
	assert.Equal(t, block.Category, stored.Category)
}

func TestChainNode_GetBlock(t *testing.T) {
	node, cleanup := setupTestChainNode(t)
	defer cleanup()

	ctx := context.Background()

	block := createTestBlock(t, []byte("test data"), CategoryGeneral, nil)
	blockID := block.BlockID

	// Test getting non-existent block
	_, err := node.GetBlock(ctx, blockID)
	assert.Error(t, err)

	// Commit block and test getting it
	err = node.CommitBlock(ctx, block)
	assert.NoError(t, err)

	stored, err := node.GetBlock(ctx, blockID)
	assert.NoError(t, err)
	assert.Equal(t, block.BlockID, stored.BlockID)
	assert.Equal(t, block.Category, stored.Category)
}

func TestChainNode_SemanticSearch(t *testing.T) {
	node, cleanup := setupTestChainNode(t)
	defer cleanup()

	ctx := context.Background()

	// Create test blocks with different categories
	block1 := createTestBlock(t, []byte("test data 1"), CategoryGeneral, nil)
	block2 := createTestBlock(t, []byte("test data 2"), CategoryError, block1)
	block3 := createTestBlock(t, []byte("test data 3"), CategoryGeneral, block2)

	// Commit blocks
	require.NoError(t, node.CommitBlock(ctx, block1))
	require.NoError(t, node.CommitBlock(ctx, block2))
	require.NoError(t, node.CommitBlock(ctx, block3))

	// Test search without category filter
	req := SearchRequest{
		Vector: block1.SemanticVector,
		Limit:  10,
	}

	results, err := node.SemanticSearch(ctx, req)
	assert.NoError(t, err)
	assert.Len(t, results, 3)

	// Test search with category filter
	category := CategoryError
	req = SearchRequest{
		Vector:   block2.SemanticVector,
		Limit:    10,
		Category: &category,
	}

	results, err = node.SemanticSearch(ctx, req)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, block2.BlockID, results[0].BlockID)

	// Test search with limit
	req = SearchRequest{
		Vector: block1.SemanticVector,
		Limit:  2,
	}

	results, err = node.SemanticSearch(ctx, req)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestSHA256Hash(t *testing.T) {
	data := []byte("test data")
	hash1 := SHA256Hash(data)
	hash2 := SHA256Hash(data)
	assert.Equal(t, hash1, hash2)
	assert.NotEmpty(t, hash1)

	// Different data should produce different hash
	hash3 := SHA256Hash([]byte("different data"))
	assert.NotEqual(t, hash1, hash3)
}

func TestCosineSimilarity(t *testing.T) {
	// Identical vectors
	v1 := []float32{1.0, 2.0, 3.0}
	v2 := []float32{1.0, 2.0, 3.0}
	sim := cosineSimilarity(v1, v2)
	assert.InDelta(t, 1.0, sim, 0.001)

	// Orthogonal vectors
	v1 = []float32{1.0, 0.0}
	v2 = []float32{0.0, 1.0}
	sim = cosineSimilarity(v1, v2)
	assert.InDelta(t, 0.0, sim, 0.001)

	// Opposite vectors
	v1 = []float32{1.0, 2.0}
	v2 = []float32{-1.0, -2.0}
	sim = cosineSimilarity(v1, v2)
	assert.InDelta(t, -1.0, sim, 0.001)

	// Different lengths
	v1 = []float32{1.0, 2.0}
	v2 = []float32{1.0}
	sim = cosineSimilarity(v1, v2)
	assert.Equal(t, 0.0, sim)

	// Zero vector
	v1 = []float32{0.0, 0.0}
	v2 = []float32{1.0, 2.0}
	sim = cosineSimilarity(v1, v2)
	assert.Equal(t, 0.0, sim)
}

func TestSqrt(t *testing.T) {
	assert.InDelta(t, 0.0, sqrt(0.0), 0.001)
	assert.InDelta(t, 1.0, sqrt(1.0), 0.001)
	assert.InDelta(t, 2.0, sqrt(4.0), 0.001)
	assert.InDelta(t, 1.414, sqrt(2.0), 0.001)

	// Negative should return 0
	assert.Equal(t, 0.0, sqrt(-1.0))
}
