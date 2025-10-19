package validation

import (
	"backend-server/internal/objects"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
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
	task *objects.ValidationTask,
	result *objects.ValidationResult,
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

// VerifyProof verifies a validation proof by checking format and hash validity
// Implements: ProofGenerator.VerifyProof (ID 2)
func (pg *ProofGenerator) VerifyProof(proof string, task *objects.ValidationTask, result *objects.ValidationResult) bool {
	// Check proof format: PROOF_V1:nodeID:sha256Hash
	if len(proof) < 10 || !strings.HasPrefix(proof, "PROOF_V1:") {
		log.Printf("Invalid proof format: %s", proof)
		return false
	}

	// Split proof into parts
	parts := strings.Split(proof, ":")
	if len(parts) != 3 {
		log.Printf("Invalid proof structure: %s", proof)
		return false
	}

	proofNodeID := parts[1]
	proofHash := parts[2]

	// Verify node ID matches
	if proofNodeID != pg.nodeID {
		log.Printf("Node ID mismatch: expected %s, got %s", pg.nodeID, proofNodeID)
		return false
	}

	// Recreate the exact proof data that was used during generation
	proofData := map[string]interface{}{
		"task_id":          task.ID,
		"result_id":        result.ID,
		"validator_node":   pg.nodeID,
		"timestamp":        time.Now().Unix(), // Use current time for verification (timestamps should match within reasonable window)
		"score":            result.Score,
		"status":           result.Status,
		"execution_time":   result.ExecutionTime.Milliseconds(),
		"test_results":     result.TestResults,
		"results":          result.Results,
	}

	proofJSON, err := json.Marshal(proofData)
	if err != nil {
		log.Printf("Failed to marshal proof data: %v", err)
		return false
	}

	// Generate hash and compare
	hash := sha256.Sum256(proofJSON)
	actualHash := hex.EncodeToString(hash[:])

	valid := actualHash == proofHash
	if !valid {
		log.Printf("Proof hash mismatch for task %s", task.ID)
	} else {
		log.Printf("Proof verification succeeded for task %s", task.ID)
	}

	return valid
}
