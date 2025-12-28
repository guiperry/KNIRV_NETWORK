package blockchain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/knirvchain/internal/storage"
)

type ChainNode struct {
	nodeID     string
	blockStore *BlockStore
	validator  BlockValidator
	mu         sync.RWMutex
}

type SearchRequest struct {
	Vector   []float32
	Limit    int
	Category *MemoryCategory
}

func NewChainNode(nodeID string, storage storage.Storage) (*ChainNode, error) {
	blockStore := NewBlockStore(storage)
	validator := NewPoAValidator(nodeID) // Default PoA validator with only self authorized

	return &ChainNode{
		nodeID:     nodeID,
		blockStore: blockStore,
		validator:  validator,
	}, nil
}

// NewChainNodeWithValidator creates a chain node with a custom validator
func NewChainNodeWithValidator(nodeID string, storage storage.Storage, validator BlockValidator) (*ChainNode, error) {
	blockStore := NewBlockStore(storage)

	return &ChainNode{
		nodeID:     nodeID,
		blockStore: blockStore,
		validator:  validator,
	}, nil
}

func (cn *ChainNode) CommitBlock(ctx context.Context, block *Block) error {
	cn.mu.Lock()
	defer cn.mu.Unlock()

	// Get latest block for validation (nil if this is first block)
	var prevBlock *Block
	allBlocks, err := cn.blockStore.GetAllBlocks()
	if err == nil && len(allBlocks) > 0 {
		// Find most recent block by timestamp
		prevBlock = allBlocks[0]
		for _, b := range allBlocks {
			if b.Timestamp > prevBlock.Timestamp {
				prevBlock = b
			}
		}
	}

	// Validate block with consensus rules
	if err := cn.validator.ValidateBlock(block, prevBlock); err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}

	// Persist block to storage
	if err := cn.blockStore.PutBlock(block); err != nil {
		return fmt.Errorf("failed to persist block: %w", err)
	}

	return nil
}

func (cn *ChainNode) SemanticSearch(ctx context.Context, req SearchRequest) ([]*Block, error) {
	cn.mu.RLock()
	defer cn.mu.RUnlock()

	var blocks []*Block
	var err error

	// Get blocks by category if specified, otherwise get all
	if req.Category != nil {
		blocks, err = cn.blockStore.GetBlocksByCategory(*req.Category)
	} else {
		blocks, err = cn.blockStore.GetAllBlocks()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve blocks: %w", err)
	}

	// Calculate similarity scores
	var results []*Block
	for _, block := range blocks {
		// Simple similarity: cosine similarity
		similarity := cosineSimilarity(req.Vector, block.SemanticVector)
		block.SimilarityScore = similarity
		results = append(results, block)
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].SimilarityScore > results[j].SimilarityScore
	})

	if len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results, nil
}

func (cn *ChainNode) GetBlock(ctx context.Context, blockID uuid.UUID) (*Block, error) {
	cn.mu.RLock()
	defer cn.mu.RUnlock()

	block, err := cn.blockStore.GetBlock(blockID)
	if err != nil {
		return nil, fmt.Errorf("block not found: %w", err)
	}
	return block, nil
}

// GetLatestBlock returns the most recent block by timestamp
func (cn *ChainNode) GetLatestBlock(ctx context.Context) (*Block, error) {
	cn.mu.RLock()
	defer cn.mu.RUnlock()

	allBlocks, err := cn.blockStore.GetAllBlocks()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve blocks: %w", err)
	}

	if len(allBlocks) == 0 {
		return nil, fmt.Errorf("no blocks in chain")
	}

	// Find most recent block by timestamp
	latestBlock := allBlocks[0]
	for _, b := range allBlocks {
		if b.Timestamp > latestBlock.Timestamp {
			latestBlock = b
		}
	}

	return latestBlock, nil
}

// GetValidator returns the block validator for this chain node
func (cn *ChainNode) GetValidator() BlockValidator {
	return cn.validator
}

// GetNodeID returns the node ID
func (cn *ChainNode) GetNodeID() string {
	return cn.nodeID
}

func SHA256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func cosineSimilarity(v1, v2 []float32) float64 {
	if len(v1) != len(v2) {
		return 0.0
	}

	var dot, norm1, norm2 float64
	for i := range v1 {
		dot += float64(v1[i] * v2[i])
		norm1 += float64(v1[i] * v1[i])
		norm2 += float64(v2[i] * v2[i])
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dot / (sqrt(norm1) * sqrt(norm2))
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
