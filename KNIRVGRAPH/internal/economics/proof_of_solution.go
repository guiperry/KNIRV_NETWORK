package economics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"blockchain-app/internal/nrv"
)

// ProofOfSolution implements the Proof-of-Solution consensus mechanism for KNIRVGRAPH
type ProofOfSolution struct {
	nrnIntegration *NRNIntegration
	nrvSystem      *nrv.NRVSystem
	rewardRates    *RewardRates
}

// RewardRates defines the reward structure for different activities
type RewardRates struct {
	ErrorNodeCreation    *big.Int `json:"error_node_creation"`
	SkillNodeCreation    *big.Int `json:"skill_node_creation"`
	SuccessfulResolution *big.Int `json:"successful_resolution"`
	NRVValidation        *big.Int `json:"nrv_validation"`
	NetworkParticipation *big.Int `json:"network_participation"`
}

// SolutionProof represents a cryptographic proof of solution
type SolutionProof struct {
	ID               string                 `json:"id"`
	NRVID            string                 `json:"nrv_id"`
	SkillID          string                 `json:"skill_id"`
	SolverID         string                 `json:"solver_id"`
	SolutionHash     string                 `json:"solution_hash"`
	ValidationProof  string                 `json:"validation_proof"`
	Timestamp        time.Time              `json:"timestamp"`
	RewardAmount     *big.Int               `json:"reward_amount"`
	Verified         bool                   `json:"verified"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ResolutionEvent represents a successful error resolution
type ResolutionEvent struct {
	ErrorNodeID      string    `json:"error_node_id"`
	SkillNodeID      string    `json:"skill_node_id"`
	SolverID         string    `json:"solver_id"`
	ResolutionTime   time.Time `json:"resolution_time"`
	EfficiencyScore  float64   `json:"efficiency_score"`
	QualityScore     float64   `json:"quality_score"`
	RewardEarned     *big.Int  `json:"reward_earned"`
}

// NewProofOfSolution creates a new Proof-of-Solution instance
func NewProofOfSolution(nrnIntegration *NRNIntegration, nrvSystem *nrv.NRVSystem) *ProofOfSolution {
	return &ProofOfSolution{
		nrnIntegration: nrnIntegration,
		nrvSystem:      nrvSystem,
		rewardRates: &RewardRates{
			ErrorNodeCreation:    big.NewInt(1000000),  // 0.001 NRN
			SkillNodeCreation:    big.NewInt(5000000),  // 0.005 NRN
			SuccessfulResolution: big.NewInt(10000000), // 0.01 NRN
			NRVValidation:        big.NewInt(2000000),  // 0.002 NRN
			NetworkParticipation: big.NewInt(500000),   // 0.0005 NRN
		},
	}
}

// ProcessErrorNodeCreation processes the creation of an error node and distributes rewards
func (pos *ProofOfSolution) ProcessErrorNodeCreation(errorNode *nrv.ErrorNode, observerID string) error {
	log.Printf("Processing error node creation: %s by observer %s", errorNode.ID, observerID)

	// Validate the error node
	if err := pos.validateErrorNode(errorNode); err != nil {
		return fmt.Errorf("invalid error node: %w", err)
	}

	// Calculate reward based on error complexity and network demand
	baseReward := new(big.Int).Set(pos.rewardRates.ErrorNodeCreation)
	complexityMultiplier := pos.calculateComplexityMultiplier(errorNode)
	demandMultiplier := pos.calculateDemandMultiplier(errorNode.ErrorType)
	
	finalReward := new(big.Int).Mul(baseReward, big.NewInt(int64(complexityMultiplier*100)))
	finalReward.Mul(finalReward, big.NewInt(int64(demandMultiplier*100)))
	finalReward.Div(finalReward, big.NewInt(10000)) // Normalize

	// Distribute reward to observer
	if err := pos.nrnIntegration.DistributeRewards(observerID, finalReward, "error_node_creation"); err != nil {
		return fmt.Errorf("failed to distribute error node creation reward: %w", err)
	}

	log.Printf("Distributed %s NRN to %s for error node creation", finalReward.String(), observerID)
	return nil
}

// ProcessSkillNodeCreation processes the creation of a skill node and distributes rewards
func (pos *ProofOfSolution) ProcessSkillNodeCreation(skillNode *nrv.SkillNode, creatorID string) error {
	log.Printf("Processing skill node creation: %s by creator %s", skillNode.ID, creatorID)

	// Validate the skill node
	if err := pos.validateSkillNode(skillNode); err != nil {
		return fmt.Errorf("invalid skill node: %w", err)
	}

	// Calculate reward based on skill utility and validation score
	baseReward := new(big.Int).Set(pos.rewardRates.SkillNodeCreation)
	utilityMultiplier := pos.calculateUtilityMultiplier(skillNode)
	validationMultiplier := skillNode.Validation.ValidationScore
	
	finalReward := new(big.Int).Mul(baseReward, big.NewInt(int64(utilityMultiplier*100)))
	finalReward.Mul(finalReward, big.NewInt(int64(validationMultiplier*100)))
	finalReward.Div(finalReward, big.NewInt(10000)) // Normalize

	// Distribute reward to creator
	if err := pos.nrnIntegration.DistributeRewards(creatorID, finalReward, "skill_node_creation"); err != nil {
		return fmt.Errorf("failed to distribute skill node creation reward: %w", err)
	}

	log.Printf("Distributed %s NRN to %s for skill node creation", finalReward.String(), creatorID)
	return nil
}

// ProcessSuccessfulResolution processes a successful error resolution and distributes rewards
func (pos *ProofOfSolution) ProcessSuccessfulResolution(event ResolutionEvent) error {
	log.Printf("Processing successful resolution: error=%s, skill=%s, solver=%s", 
		event.ErrorNodeID, event.SkillNodeID, event.SolverID)

	// Generate solution proof
	proof, err := pos.generateSolutionProof(event)
	if err != nil {
		return fmt.Errorf("failed to generate solution proof: %w", err)
	}

	// Verify the proof
	if err := pos.verifySolutionProof(proof); err != nil {
		return fmt.Errorf("solution proof verification failed: %w", err)
	}

	// Calculate reward based on efficiency and quality
	baseReward := new(big.Int).Set(pos.rewardRates.SuccessfulResolution)
	efficiencyBonus := big.NewInt(int64(event.EfficiencyScore * 100))
	qualityBonus := big.NewInt(int64(event.QualityScore * 100))
	
	finalReward := new(big.Int).Add(baseReward, efficiencyBonus)
	finalReward.Add(finalReward, qualityBonus)

	// Distribute reward to solver
	if err := pos.nrnIntegration.DistributeRewards(event.SolverID, finalReward, "successful_resolution"); err != nil {
		return fmt.Errorf("failed to distribute resolution reward: %w", err)
	}

	// Update skill node performance metrics
	if err := pos.updateSkillPerformance(event.SkillNodeID, event.EfficiencyScore, event.QualityScore); err != nil {
		log.Printf("Warning: failed to update skill performance: %v", err)
	}

	log.Printf("Distributed %s NRN to %s for successful resolution", finalReward.String(), event.SolverID)
	return nil
}

// generateSolutionProof generates a cryptographic proof of solution
func (pos *ProofOfSolution) generateSolutionProof(event ResolutionEvent) (*SolutionProof, error) {
	// Create proof data
	proofData := map[string]interface{}{
		"error_node_id":    event.ErrorNodeID,
		"skill_node_id":    event.SkillNodeID,
		"solver_id":        event.SolverID,
		"resolution_time":  event.ResolutionTime.Unix(),
		"efficiency_score": event.EfficiencyScore,
		"quality_score":    event.QualityScore,
	}

	// Serialize and hash
	jsonData, err := json.Marshal(proofData)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(jsonData)
	solutionHash := hex.EncodeToString(hash[:])

	// Generate validation proof (simplified)
	validationData := fmt.Sprintf("%s:%s:%f:%f", 
		event.ErrorNodeID, event.SkillNodeID, event.EfficiencyScore, event.QualityScore)
	validationHash := sha256.Sum256([]byte(validationData))
	validationProof := hex.EncodeToString(validationHash[:])

	proof := &SolutionProof{
		ID:              fmt.Sprintf("proof_%d", time.Now().UnixNano()),
		NRVID:           event.ErrorNodeID, // Assuming error node ID maps to NRV ID
		SkillID:         event.SkillNodeID,
		SolverID:        event.SolverID,
		SolutionHash:    solutionHash,
		ValidationProof: validationProof,
		Timestamp:       time.Now(),
		RewardAmount:    event.RewardEarned,
		Verified:        false,
		Metadata: map[string]interface{}{
			"efficiency_score": event.EfficiencyScore,
			"quality_score":    event.QualityScore,
			"resolution_time":  event.ResolutionTime.Unix(),
		},
	}

	return proof, nil
}

// verifySolutionProof verifies a solution proof
func (pos *ProofOfSolution) verifySolutionProof(proof *SolutionProof) error {
	// In a real implementation, this would verify the cryptographic proof
	// For now, we'll do basic validation
	
	if proof.SolutionHash == "" {
		return fmt.Errorf("missing solution hash")
	}
	
	if proof.ValidationProof == "" {
		return fmt.Errorf("missing validation proof")
	}
	
	if proof.SolverID == "" {
		return fmt.Errorf("missing solver ID")
	}

	// Mark as verified
	proof.Verified = true
	
	log.Printf("Solution proof verified: %s", proof.ID)
	return nil
}

// validateErrorNode validates an error node
func (pos *ProofOfSolution) validateErrorNode(errorNode *nrv.ErrorNode) error {
	if errorNode.ID == "" {
		return fmt.Errorf("missing error node ID")
	}
	
	if errorNode.ErrorType == "" {
		return fmt.Errorf("missing error type")
	}
	
	if len(errorNode.Context) == 0 {
		return fmt.Errorf("missing error context")
	}

	return nil
}

// validateSkillNode validates a skill node
func (pos *ProofOfSolution) validateSkillNode(skillNode *nrv.SkillNode) error {
	if skillNode.ID == "" {
		return fmt.Errorf("missing skill node ID")
	}
	
	if skillNode.SkillType == "" {
		return fmt.Errorf("missing skill type")
	}
	
	if len(skillNode.Capabilities) == 0 {
		return fmt.Errorf("missing skill capabilities")
	}

	return nil
}

// calculateComplexityMultiplier calculates a multiplier based on error complexity
func (pos *ProofOfSolution) calculateComplexityMultiplier(errorNode *nrv.ErrorNode) float64 {
	// Simple complexity calculation based on context size and error type
	baseComplexity := 1.0
	
	// Increase complexity based on context size
	contextComplexity := float64(len(errorNode.Context)) / 100.0
	if contextComplexity > 2.0 {
		contextComplexity = 2.0 // Cap at 2x
	}
	
	return baseComplexity + contextComplexity
}

// calculateDemandMultiplier calculates a multiplier based on network demand for error type
func (pos *ProofOfSolution) calculateDemandMultiplier(errorType string) float64 {
	// In a real implementation, this would analyze network demand
	// For now, return a base multiplier
	return 1.0
}

// calculateUtilityMultiplier calculates a multiplier based on skill utility
func (pos *ProofOfSolution) calculateUtilityMultiplier(skillNode *nrv.SkillNode) float64 {
	// Calculate utility based on capabilities and requirements
	baseUtility := 1.0
	
	// More capabilities = higher utility
	capabilityBonus := float64(len(skillNode.Capabilities)) * 0.1
	if capabilityBonus > 1.0 {
		capabilityBonus = 1.0 // Cap at 1x bonus
	}
	
	return baseUtility + capabilityBonus
}

// updateSkillPerformance updates skill node performance metrics
func (pos *ProofOfSolution) updateSkillPerformance(skillID string, efficiency, quality float64) error {
	// In a real implementation, this would update the skill node in the NRV system
	log.Printf("Updating skill %s performance: efficiency=%.2f, quality=%.2f", skillID, efficiency, quality)
	return nil
}

// GetRewardRates returns the current reward rates
func (pos *ProofOfSolution) GetRewardRates() *RewardRates {
	return pos.rewardRates
}

// UpdateRewardRates updates the reward rates
func (pos *ProofOfSolution) UpdateRewardRates(rates *RewardRates) {
	pos.rewardRates = rates
	log.Println("Reward rates updated")
}
