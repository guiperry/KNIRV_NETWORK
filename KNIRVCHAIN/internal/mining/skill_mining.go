package mining

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"KNIRVCHAIN/internal/blockchain"
	"KNIRVCHAIN/internal/errors"
	"KNIRVCHAIN/internal/graph"
	"KNIRVCHAIN/internal/types"
)

// SkillMiner handles the mining of skills from error nodes
type SkillMiner struct {
	graphQueries *graph.GraphQueries
	nodeStore    *graph.NodeStore
	relManager   *graph.RelationshipManager
	blockchain   *blockchain.BlockchainStruct
}

// NewSkillMiner creates a new skill miner
func NewSkillMiner(graphQueries *graph.GraphQueries, nodeStore *graph.NodeStore, relManager *graph.RelationshipManager, bc *blockchain.BlockchainStruct) *SkillMiner {
	return &SkillMiner{
		graphQueries: graphQueries,
		nodeStore:    nodeStore,
		relManager:   relManager,
		blockchain:   bc,
	}
}

// MiningProposal represents a proposal to mine a skill from an error
type MiningProposal struct {
	ErrorNodeID     string                    `json:"error_node_id"`
	MinerAddress    string                    `json:"miner_address"`
	LoRAAdapter     *types.LoRAAdapterPointer `json:"lora_adapter"`
	ProposedSkillID string                    `json:"proposed_skill_id"`
	ProposedAt      int64                     `json:"proposed_at"`
	Status          MiningStatus              `json:"status"`
}

// MiningStatus represents the status of a mining proposal
type MiningStatus string

const (
	MiningStatusProposed   MiningStatus = "proposed"
	MiningStatusValidating MiningStatus = "validating"
	MiningStatusValidated  MiningStatus = "validated"
	MiningStatusRejected   MiningStatus = "rejected"
	MiningStatusConfirmed  MiningStatus = "confirmed"
)

// SubmitMiningProposal submits a proposal to mine a skill from an error
func (sm *SkillMiner) SubmitMiningProposal(errorNodeID, minerAddress string, loraAdapter *types.LoRAAdapterPointer) (*MiningProposal, error) {
	// Validate inputs
	if errorNodeID == "" {
		return nil, errors.NewValidationError("error_node_id cannot be empty")
	}
	if minerAddress == "" {
		return nil, errors.NewValidationError("miner_address cannot be empty")
	}
	if loraAdapter == nil {
		return nil, errors.NewValidationError("lora_adapter cannot be nil")
	}

	// Verify the error node exists
	errorNode, err := sm.nodeStore.GetErrorNode(errorNodeID)
	if err != nil {
		return nil, fmt.Errorf("error node not found: %w", err)
	}

	// Check if error is still open for mining
	if errorNode.Status != types.NodeStatusOpen && errorNode.Status != types.NodeStatusInProgress {
		return nil, fmt.Errorf("error node is not available for mining (status: %s)", errorNode.Status)
	}

	// Generate proposed skill ID
	proposedSkillID := sm.generateSkillID(errorNodeID, minerAddress, loraAdapter)

	proposal := &MiningProposal{
		ErrorNodeID:     errorNodeID,
		MinerAddress:    minerAddress,
		LoRAAdapter:     loraAdapter,
		ProposedSkillID: proposedSkillID,
		ProposedAt:      time.Now().Unix(),
		Status:          MiningStatusProposed,
	}

	// Create transaction for the mining proposal
	tx, err := sm.createMiningProposalTransaction(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to create mining proposal transaction: %w", err)
	}

	// Submit transaction to blockchain
	err = sm.blockchain.AddTransactionToTransactionPool(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to submit mining proposal transaction: %w", err)
	}

	log.Printf("Mining proposal submitted: %s for error %s by miner %s", proposedSkillID, errorNodeID, minerAddress)

	return proposal, nil
}

// generateSkillID generates a unique skill ID
func (sm *SkillMiner) generateSkillID(errorNodeID, minerAddress string, loraAdapter *types.LoRAAdapterPointer) string {
	data := fmt.Sprintf("%s:%s:%s:%d", errorNodeID, minerAddress, loraAdapter.AdapterID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// createMiningProposalTransaction creates a blockchain transaction for the mining proposal
func (sm *SkillMiner) createMiningProposalTransaction(proposal *MiningProposal) (*blockchain.Transaction, error) {
	data, err := json.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proposal data: %w", err)
	}

	tx, err := blockchain.NewMCPTransaction(
		proposal.MinerAddress,
		"", // No specific recipient
		0,  // No value transfer
		data,
		blockchain.TransactionTypeSkillMiningProposal,
		100, // Fee for proposal submission
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return tx, nil
}

// ValidateSkillProposal validates a skill proposal using KNIRVNEXUS DVE
func (sm *SkillMiner) ValidateSkillProposal(proposal *MiningProposal) (*types.SkillNode, error) {
	// Update proposal status
	proposal.Status = MiningStatusValidating

	// Get the error node
	errorNode, err := sm.nodeStore.GetErrorNode(proposal.ErrorNodeID)
	if err != nil {
		return nil, fmt.Errorf("error node not found: %w", err)
	}

	// Call KNIRVNEXUS DVE for validation
	validationResult, err := sm.validateWithKNIRVNEXUS(errorNode, proposal.LoRAAdapter)
	if err != nil {
		proposal.Status = MiningStatusRejected
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create the skill node if validation passed
	if validationResult.IsValid {
		skillNode, err := sm.createValidatedSkillNode(errorNode, proposal, validationResult)
		if err != nil {
			proposal.Status = MiningStatusRejected
			return nil, fmt.Errorf("failed to create skill node: %w", err)
		}

		proposal.Status = MiningStatusValidated
		return skillNode, nil
	}

	proposal.Status = MiningStatusRejected
	return nil, fmt.Errorf("skill validation failed: %s", validationResult.Reason)
}

// ValidationResult represents the result of skill validation
type ValidationResult struct {
	IsValid       bool                   `json:"is_valid"`
	Performance   types.SkillPerformance `json:"performance"`
	ValidationProof string               `json:"validation_proof"`
	Reason        string                 `json:"reason,omitempty"`
}

// validateWithKNIRVNEXUS simulates validation with KNIRVNEXUS DVE
// In a real implementation, this would make an actual call to KNIRVNEXUS
func (sm *SkillMiner) validateWithKNIRVNEXUS(errorNode *types.ErrorNode, loraAdapter *types.LoRAAdapterPointer) (*ValidationResult, error) {
	// This is a placeholder implementation
	// In the real system, this would:
	// 1. Send the error node and LoRA adapter to KNIRVNEXUS DVE
	// 2. DVE would run the LoRA adapter against test cases derived from the error
	// 3. DVE would return performance metrics and validation proof

	// Simulate validation logic
	if loraAdapter.Rank <= 0 || loraAdapter.Alpha <= 0 {
		return &ValidationResult{
			IsValid: false,
			Reason:  "Invalid LoRA parameters",
		}, nil
	}

	// Generate mock performance metrics
	performance := types.SkillPerformance{
		Accuracy:       0.85 + float64(loraAdapter.Rank%10)*0.01, // 0.85-0.94
		Precision:      0.82 + float64(loraAdapter.Rank%8)*0.01,  // 0.82-0.89
		Recall:         0.80 + float64(loraAdapter.Rank%12)*0.01, // 0.80-0.91
		F1Score:        0.83 + float64(loraAdapter.Rank%9)*0.01,  // 0.83-0.91
		TestCasesRun:   100,
		TestCasesPass:  uint64(85 + loraAdapter.Rank%10), // 85-94
		AvgResponseTime: int64(150 - loraAdapter.Rank%50), // 100-149ms
	}

	// Validate performance
	if err := performance.Validate(); err != nil {
		return &ValidationResult{
			IsValid: false,
			Reason:  fmt.Sprintf("Performance validation failed: %v", err),
		}, nil
	}

	// Generate validation proof
	proofData := fmt.Sprintf("validated:%s:%s:%d", errorNode.ID, loraAdapter.AdapterID, time.Now().Unix())
	proofHash := sha256.Sum256([]byte(proofData))
	validationProof := hex.EncodeToString(proofHash[:])

	return &ValidationResult{
		IsValid:         true,
		Performance:     performance,
		ValidationProof: validationProof,
	}, nil
}

// createValidatedSkillNode creates a skill node from a validated proposal
func (sm *SkillMiner) createValidatedSkillNode(errorNode *types.ErrorNode, proposal *MiningProposal, validationResult *ValidationResult) (*types.SkillNode, error) {
	skillNode, err := types.NewSkillNode(
		proposal.ProposedSkillID,
		fmt.Sprintf("Skill for %s error", errorNode.ErrorType),
		proposal.ErrorNodeID,
		proposal.LoRAAdapter,
		validationResult.ValidationProof,
		validationResult.Performance,
		proposal.MinerAddress,
		sm.calculateNRNReward(validationResult.Performance, errorNode.FailureCount),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create skill node: %w", err)
	}

	// Store the skill node
	err = sm.nodeStore.StoreSkillNode(skillNode)
	if err != nil {
		return nil, fmt.Errorf("failed to store skill node: %w", err)
	}

	// Create relationship between error node and skill node
	_, err = sm.relManager.CreateRelationship(
		"error_node", errorNode.ID,
		"skill_node", skillNode.ID,
		graph.RelationshipTypeErrorToSkill,
		map[string]interface{}{
			"validation_proof": validationResult.ValidationProof,
			"miner_address":    proposal.MinerAddress,
			"created_at":       skillNode.CreatedAt,
		},
	)
	if err != nil {
		log.Printf("Warning: failed to create relationship between error and skill node: %v", err)
	}

	return skillNode, nil
}

// calculateNRNReward calculates the NRN reward based on performance and error severity
func (sm *SkillMiner) calculateNRNReward(performance types.SkillPerformance, failureCount uint64) uint64 {
	// Base reward
	baseReward := uint64(1000)

	// Performance multiplier (0.5 to 2.0 based on F1 score)
	performanceMultiplier := 0.5 + performance.F1Score*1.5

	// Error severity multiplier (1.0 to 3.0 based on failure count)
	severityMultiplier := 1.0 + float64(failureCount)/100.0
	if severityMultiplier > 3.0 {
		severityMultiplier = 3.0
	}

	reward := uint64(float64(baseReward) * performanceMultiplier * severityMultiplier)

	// Minimum reward
	if reward < 500 {
		reward = 500
	}

	// Maximum reward
	if reward > 10000 {
		reward = 10000
	}

	return reward
}

// ConfirmSkillNode confirms a validated skill node in the registry
func (sm *SkillMiner) ConfirmSkillNode(skillNodeID string) error {
	// Get the skill node
	skillNode, err := sm.nodeStore.GetSkillNode(skillNodeID)
	if err != nil {
		return fmt.Errorf("skill node not found: %w", err)
	}

	// Create confirmation transaction
	confirmationData := map[string]interface{}{
		"skill_node_id": skillNodeID,
		"confirmed_at":  time.Now().Unix(),
	}

	data, err := json.Marshal(confirmationData)
	if err != nil {
		return fmt.Errorf("failed to marshal confirmation data: %w", err)
	}

	tx, err := blockchain.NewMCPTransaction(
		skillNode.MinerAddress,
		"", // No specific recipient
		0,  // No value transfer
		data,
		blockchain.TransactionTypeSkillConfirmation,
		50, // Fee for confirmation
	)
	if err != nil {
		return fmt.Errorf("failed to create confirmation transaction: %w", err)
	}

	// Submit transaction
	err = sm.blockchain.AddTransactionToTransactionPool(tx)
	if err != nil {
		return fmt.Errorf("failed to submit confirmation transaction: %w", err)
	}

	// Update error node status if this is the first confirmed skill
	errorNode, err := sm.nodeStore.GetErrorNode(skillNode.ErrorNodeID)
	if err == nil {
		skills, _ := sm.graphQueries.GetSkillsForError(skillNode.ErrorNodeID)
		if len(skills) == 1 { // This is the first skill
			errorNode.Status = types.NodeStatusResolved
			sm.nodeStore.StoreErrorNode(errorNode)
		}
	}

	log.Printf("Skill node confirmed: %s", skillNodeID)
	return nil
}

// GetMiningProposalsForError gets all mining proposals for a specific error
func (sm *SkillMiner) GetMiningProposalsForError(errorNodeID string) ([]*MiningProposal, error) {
	// This would typically query a database of proposals
	// For now, return empty slice as proposals are handled via transactions
	return []*MiningProposal{}, nil
}

// GetSkillsByMiner gets all confirmed skills by a specific miner
func (sm *SkillMiner) GetSkillsByMiner(minerAddress string) ([]*types.SkillNode, error) {
	return sm.graphQueries.FindSkillNodesByMiner(minerAddress, 0)
}

// GetTopSkills gets the highest performing skills
func (sm *SkillMiner) GetTopSkills(limit int) ([]*graph.SkillPerformanceResult, error) {
	return sm.graphQueries.GetTopPerformingSkills(limit)
}