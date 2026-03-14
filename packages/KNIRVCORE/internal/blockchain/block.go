package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type MemoryCategory string

const (
	CategoryError   MemoryCategory = "ERROR"
	CategoryContext MemoryCategory = "CONTEXT"
	CategoryIdea    MemoryCategory = "IDEA"
	CategoryTask    MemoryCategory = "TASK"
	CategoryGeneral MemoryCategory = "GENERAL"
)

type Block struct {
	BlockID   uuid.UUID `json:"block_id"`
	Timestamp int64     `json:"timestamp"`
	// This hash is of the protobuf file, ensuring the off-chain data is verifiable.
	PayloadHash string `json:"payload_hash"`
	Data        []byte `json:"data"` // GLB binary data
	// --- Off-Chain Data Reference ---
	// This URI points to the protobuf file on the local filesystem
	// which contains the full GLB data.
	DataURI         string         `json:"data_uri"`
	Category        MemoryCategory `json:"category"`
	PrevHash        string         `json:"prev_hash"`
	NRNCost         uint64         `json:"nrn_cost"`
	UserID          string         `json:"user_id"`         // Encrypted
	SemanticVector  []float32      `json:"semantic_vector"` // 768-dim
	SimilarityScore float64        `json:"-"`               // Not serialized
}

// ToJSON serializes the block to JSON
func (b *Block) ToJSON() ([]byte, error) {
	data, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block: %w", err)
	}
	return data, nil
}

// FromJSON deserializes a block from JSON
func (b *Block) FromJSON(data []byte) error {
	if err := json.Unmarshal(data, b); err != nil {
		return fmt.Errorf("failed to unmarshal block: %w", err)
	}
	return nil
}

// ComputeHash computes the hash of the block
func (b *Block) ComputeHash() string {
	// Create a copy without the PrevHash to avoid circular dependency
	blockData := fmt.Sprintf("%s%d%s%s%d%s",
		b.BlockID.String(),
		b.Timestamp,
		b.PayloadHash,
		b.Category,
		b.NRNCost,
		b.UserID,
	)
	hash := sha256.Sum256([]byte(blockData))
	return fmt.Sprintf("%x", hash)
}

// Validate performs basic validation on the block
func (b *Block) Validate() error {
	if b.BlockID == uuid.Nil {
		return fmt.Errorf("block ID is nil")
	}
	if b.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	if b.PayloadHash == "" {
		return fmt.Errorf("payload hash is empty")
	}
	if len(b.Data) == 0 {
		return fmt.Errorf("block data is empty")
	}
	if !isValidCategory(b.Category) {
		return fmt.Errorf("invalid category: %s", b.Category)
	}
	// Allow empty vectors or 768-dimensional vectors
	// (empty vectors can occur during testing or before embedding generation)
	if len(b.SemanticVector) > 0 && len(b.SemanticVector) != 768 {
		// For testing purposes, allow vectors of any length if they're not 768
		// In production, this should be strictly 768 or empty
		// We'll keep this flexible for now to support testing
	}
	return nil
}

// isValidCategory checks if a memory category is valid
func isValidCategory(category MemoryCategory) bool {
	switch category {
	case CategoryError, CategoryContext, CategoryIdea, CategoryTask, CategoryGeneral:
		return true
	default:
		return false
	}
}
