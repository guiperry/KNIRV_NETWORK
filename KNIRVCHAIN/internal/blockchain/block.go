package blockchain

import (
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
