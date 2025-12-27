package blockchain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
)

type ChainNode struct {
	nodeID string
	blocks map[uuid.UUID]*Block
	mu     sync.RWMutex
}

type SearchRequest struct {
	Vector   []float32
	Limit    int
	Category *MemoryCategory
}

func NewChainNode(nodeURL string) (*ChainNode, error) {
	return &ChainNode{
		nodeID: nodeURL,
		blocks: make(map[uuid.UUID]*Block),
	}, nil
}

func (cn *ChainNode) CommitBlock(ctx context.Context, block *Block) error {
	cn.mu.Lock()
	defer cn.mu.Unlock()

	// Simple PoA: just append
	cn.blocks[block.BlockID] = block
	return nil
}

func (cn *ChainNode) SemanticSearch(ctx context.Context, req SearchRequest) ([]*Block, error) {
	cn.mu.RLock()
	defer cn.mu.RUnlock()

	var results []*Block
	for _, block := range cn.blocks {
		if req.Category != nil && block.Category != *req.Category {
			continue
		}
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

	block, ok := cn.blocks[blockID]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
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
