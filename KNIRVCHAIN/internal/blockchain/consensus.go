package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BlockValidator defines the interface for block validation
type BlockValidator interface {
	ValidateBlock(block *Block, prevBlock *Block) error
	IsAuthorizedValidator(validatorID string) bool
	AddValidator(validatorID string) error
	RemoveValidator(validatorID string) error
}

// PoAValidator implements Proof-of-Authority consensus
// In PoA, only authorized validators can create blocks
type PoAValidator struct {
	validators map[string]bool // Set of authorized validator node IDs
	nodeID     string          // This node's ID
}

// NewPoAValidator creates a new PoA validator
// nodeID is the ID of this node
// initialValidators is the list of initially authorized validators
func NewPoAValidator(nodeID string, initialValidators ...string) *PoAValidator {
	validators := make(map[string]bool)

	// Add initial validators
	for _, v := range initialValidators {
		validators[v] = true
	}

	// Always add self as validator
	validators[nodeID] = true

	return &PoAValidator{
		validators: validators,
		nodeID:     nodeID,
	}
}

// ValidateBlock performs comprehensive block validation
func (v *PoAValidator) ValidateBlock(block *Block, prevBlock *Block) error {
	// 1. Validate block ID is properly formatted
	if block.BlockID == uuid.Nil {
		return fmt.Errorf("invalid block ID: cannot be nil UUID")
	}

	// 2. Validate timestamp
	if err := v.validateTimestamp(block, prevBlock); err != nil {
		return fmt.Errorf("timestamp validation failed: %w", err)
	}

	// 3. Validate previous hash
	if err := v.validatePreviousHash(block, prevBlock); err != nil {
		return fmt.Errorf("previous hash validation failed: %w", err)
	}

	// 4. Validate payload hash matches GLB data
	if err := v.validatePayloadHash(block); err != nil {
		return fmt.Errorf("payload hash validation failed: %w", err)
	}

	// 5. Validate NRN cost calculation
	if err := v.validateNRNCost(block); err != nil {
		return fmt.Errorf("NRN cost validation failed: %w", err)
	}

	// 6. Validate category is valid
	if err := v.validateCategory(block); err != nil {
		return fmt.Errorf("category validation failed: %w", err)
	}

	// 7. Validate user/validator is authorized
	if !v.IsAuthorizedValidator(block.UserID) {
		return fmt.Errorf("unauthorized validator: %s", block.UserID)
	}

	// 8. Validate semantic vector dimensions
	if err := v.validateSemanticVector(block); err != nil {
		return fmt.Errorf("semantic vector validation failed: %w", err)
	}

	return nil
}

// validateTimestamp ensures block timestamp is after previous block
func (v *PoAValidator) validateTimestamp(block *Block, prevBlock *Block) error {
	if prevBlock != nil {
		if block.Timestamp <= prevBlock.Timestamp {
			return fmt.Errorf("block timestamp (%d) must be after previous block timestamp (%d)",
				block.Timestamp, prevBlock.Timestamp)
		}
	}

	// Validate timestamp is not too far in the future (allow 5 minute drift)
	maxFuture := time.Now().Unix() + 300
	if block.Timestamp > maxFuture {
		return fmt.Errorf("block timestamp (%d) is too far in the future (max: %d)",
			block.Timestamp, maxFuture)
	}

	// Validate timestamp is not too far in the past (allow 24 hours)
	minPast := time.Now().Unix() - 86400
	if block.Timestamp < minPast {
		return fmt.Errorf("block timestamp (%d) is too far in the past (min: %d)",
			block.Timestamp, minPast)
	}

	return nil
}

// validatePreviousHash verifies the previous block hash matches
func (v *PoAValidator) validatePreviousHash(block *Block, prevBlock *Block) error {
	if prevBlock == nil {
		// Genesis block - previous hash should be empty
		if block.PrevHash != "" {
			return fmt.Errorf("genesis block must have empty previous hash")
		}
		return nil
	}

	// Compute hash of previous block
	expectedHash := v.computeBlockHash(prevBlock)

	if block.PrevHash != expectedHash {
		return fmt.Errorf("previous hash mismatch: got %s, expected %s",
			block.PrevHash, expectedHash)
	}

	return nil
}

// validatePayloadHash verifies the payload hash matches the GLB data
func (v *PoAValidator) validatePayloadHash(block *Block) error {
	// Compute hash of payload
	h := sha256.New()
	h.Write(block.Data)
	computedHash := hex.EncodeToString(h.Sum(nil))

	if block.PayloadHash != computedHash {
		return fmt.Errorf("payload hash mismatch: got %s, computed %s",
			block.PayloadHash, computedHash)
	}

	return nil
}

// validateNRNCost validates the NRN cost calculation
// Cost is based on payload size: 1 NRN per KB
func (v *PoAValidator) validateNRNCost(block *Block) error {
	// Calculate expected cost (1 NRN per KB, rounded up)
	payloadSizeKB := (len(block.Data) + 1023) / 1024
	expectedCost := uint64(payloadSizeKB)

	if block.NRNCost != expectedCost {
		return fmt.Errorf("NRN cost mismatch: got %d, expected %d (payload: %d bytes)",
			block.NRNCost, expectedCost, len(block.Data))
	}

	return nil
}

// validateCategory ensures the category is valid
func (v *PoAValidator) validateCategory(block *Block) error {
	switch block.Category {
	case CategoryError, CategoryContext, CategoryIdea, CategoryTask, CategoryGeneral:
		return nil
	default:
		return fmt.Errorf("invalid category: %s", block.Category)
	}
}

// validateSemanticVector ensures the vector has proper dimensions
func (v *PoAValidator) validateSemanticVector(block *Block) error {
	if block.SemanticVector == nil {
		return fmt.Errorf("semantic vector is nil")
	}

	// Vector should be 768 dimensions (TF-IDF + LSA output)
	expectedDim := 768
	if len(block.SemanticVector) != expectedDim {
		return fmt.Errorf("semantic vector dimension mismatch: got %d, expected %d",
			len(block.SemanticVector), expectedDim)
	}

	// Validate vector is normalized (L2 norm should be ~1.0)
	// Allow some floating point error
	norm := v.computeL2Norm(block.SemanticVector)
	if norm < 0.99 || norm > 1.01 {
		return fmt.Errorf("semantic vector not normalized: L2 norm = %f", norm)
	}

	return nil
}

// computeBlockHash computes the hash of a block
func (v *PoAValidator) computeBlockHash(block *Block) string {
	h := sha256.New()

	// Hash components in canonical order
	h.Write([]byte(block.BlockID.String()))
	h.Write([]byte(fmt.Sprintf("%d", block.Timestamp)))
	h.Write([]byte(block.PrevHash))
	h.Write([]byte(block.PayloadHash))
	h.Write([]byte(block.Category))
	h.Write([]byte(fmt.Sprintf("%d", block.NRNCost)))
	h.Write([]byte(block.UserID))

	return hex.EncodeToString(h.Sum(nil))
}

// computeL2Norm computes the L2 (Euclidean) norm of a vector
func (v *PoAValidator) computeL2Norm(vector []float32) float64 {
	var sum float64
	for _, val := range vector {
		sum += float64(val) * float64(val)
	}
	return sum // Return squared norm for efficiency (checking if ~1.0)
}

// IsAuthorizedValidator checks if a validator ID is authorized
func (v *PoAValidator) IsAuthorizedValidator(validatorID string) bool {
	return v.validators[validatorID]
}

// AddValidator adds a new authorized validator
func (v *PoAValidator) AddValidator(validatorID string) error {
	if validatorID == "" {
		return fmt.Errorf("validator ID cannot be empty")
	}

	if v.validators[validatorID] {
		return fmt.Errorf("validator %s is already authorized", validatorID)
	}

	v.validators[validatorID] = true
	return nil
}

// RemoveValidator removes an authorized validator
func (v *PoAValidator) RemoveValidator(validatorID string) error {
	if validatorID == "" {
		return fmt.Errorf("validator ID cannot be empty")
	}

	if !v.validators[validatorID] {
		return fmt.Errorf("validator %s is not authorized", validatorID)
	}

	// Don't allow removing self
	if validatorID == v.nodeID {
		return fmt.Errorf("cannot remove self as validator")
	}

	delete(v.validators, validatorID)
	return nil
}

// GetValidators returns the list of authorized validators
func (v *PoAValidator) GetValidators() []string {
	validators := make([]string, 0, len(v.validators))
	for id := range v.validators {
		validators = append(validators, id)
	}
	return validators
}

// GetNodeID returns this node's ID
func (v *PoAValidator) GetNodeID() string {
	return v.nodeID
}

// ValidateBlockChain validates a sequence of blocks
// This can be used to validate a chain segment or entire chain
func (v *PoAValidator) ValidateBlockChain(blocks []*Block) error {
	if len(blocks) == 0 {
		return nil
	}

	// Validate first block (genesis or chain start)
	if err := v.ValidateBlock(blocks[0], nil); err != nil {
		return fmt.Errorf("block 0 validation failed: %w", err)
	}

	// Validate remaining blocks in sequence
	for i := 1; i < len(blocks); i++ {
		if err := v.ValidateBlock(blocks[i], blocks[i-1]); err != nil {
			return fmt.Errorf("block %d validation failed: %w", i, err)
		}
	}

	return nil
}

// CreateBlockHash generates a hash for a new block
// This is a helper for block creation
func CreateBlockHash(block *Block) string {
	h := sha256.New()

	h.Write([]byte(block.BlockID.String()))
	h.Write([]byte(fmt.Sprintf("%d", block.Timestamp)))
	h.Write([]byte(block.PrevHash))
	h.Write([]byte(block.PayloadHash))
	h.Write([]byte(block.Category))
	h.Write([]byte(fmt.Sprintf("%d", block.NRNCost)))
	h.Write([]byte(block.UserID))

	return hex.EncodeToString(h.Sum(nil))
}

// CalculateNRNCost calculates the NRN cost for a payload
// Cost is 1 NRN per KB (rounded up)
func CalculateNRNCost(payloadSize int) uint64 {
	return uint64((payloadSize + 1023) / 1024)
}
