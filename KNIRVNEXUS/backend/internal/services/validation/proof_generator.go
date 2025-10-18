package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"backend-server/internal/models"
)

// ProofGenerator generates cryptographic proofs for validation results
type ProofGenerator struct {
	nodeID string
}

// NewProofGenerator creates a new proof generator
func NewProofGenerator(nodeID string) *ProofGenerator {
	return &ProofGenerator{nodeID: nodeID}
}

// GenerateProof creates a cryptographic proof for a validation result
func (pg *ProofGenerator) GenerateProof(
	task *models.ValidationTask,
	result *models.ValidationResult,
) string {
	// Create proof data structure
	proofData := map[string]interface{}{
		"task_id":        task.ID,
		"result_id":      result.ID,
		"validator_node": pg.nodeID,
		"timestamp":      time.Now().Unix(),
		"score":          result.Score,
		"status":         result.Status,
		"execution_time": result.ExecutionTime.Milliseconds(),
		"test_results":   result.TestResults,
		"results":        result.Results,
	}

	// Serialize to JSON
	proofJSON, err := json.Marshal(proofData)
	if err != nil {
		return fmt.Sprintf("proof_error_%s", task.ID)
	}

	// Generate SHA-256 hash
	hash := sha256.Sum256(proofJSON)
	proofHash := hex.EncodeToString(hash[:])

	// Format proof (in production, this would be a proper cryptographic signature)
	proof := fmt.Sprintf("PROOF_V1:%s:%s", pg.nodeID, proofHash)

	return proof
}

// VerifyProof verifies a validation proof
func (pg *ProofGenerator) VerifyProof(proof string, task *models.ValidationTask, result *models.ValidationResult) bool {
	// In production, implement proper signature verification
	// For now, just check format
	if len(proof) < 10 || proof[:8] != "PROOF_V1" {
		return false
	}

	// Extract hash from proof
	parts := []string{}
	for i, part := range []rune(proof) {
		if i > 0 && part == ':' {
			break
		}
		if i >= 8 {
			parts = append(parts, string(part))
		}
	}

	if len(parts) < 2 {
		return false
	}

	expectedHash := parts[1]

	// Recreate proof data for verification
	proofData := map[string]interface{}{
		"task_id":        task.ID,
		"result_id":      result.ID,
		"validator_node": pg.nodeID,
		"timestamp":      time.Now().Unix(), // Note: timestamp mismatch in real verification
		"score":          result.Score,
		"status":         result.Status,
		"execution_time": result.ExecutionTime.Milliseconds(),
		"test_results":   result.TestResults,
		"results":        result.Results,
	}

	proofJSON, err := json.Marshal(proofData)
	if err != nil {
		return false
	}

	hash := sha256.Sum256(proofJSON)
	actualHash := hex.EncodeToString(hash[:])

	return actualHash == expectedHash
}
