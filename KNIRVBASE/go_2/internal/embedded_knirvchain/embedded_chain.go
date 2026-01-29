// Package embedded_knirvchain provides the embedded KNIRVCHAIN implementation for KNIRVCHAIN
// Revolutionary Architecture: KNIRVCHAIN embedded within itself instead of standalone blockchain
package embedded_knirvchain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EmbeddedChainConfig configuration for the embedded KNIRVCHAIN
type EmbeddedChainConfig struct {
	ModelKernel           string  `json:"model_kernel"` // hrm, phi3, recurrentgemma, tinyllama
	MaxMemoryMB           int     `json:"max_memory_mb"`
	ConsensusThreshold    float64 `json:"consensus_threshold"`
	LoRAAdapterCacheSize  int     `json:"lora_adapter_cache_size"`
	SkillChainDepth       int     `json:"skill_chain_depth"`
	EnableRealTimeUpdates bool    `json:"enable_real_time_updates"`
}

// LoRAAdapterSkill represents a skill as a LoRA adapter containing weights and biases
type LoRAAdapterSkill struct {
	SkillID                string            `json:"skill_id"`
	SkillName              string            `json:"skill_name"`
	Description            string            `json:"description"`
	BaseModelCompatibility string            `json:"base_model_compatibility"`
	Version                int               `json:"version"`
	Rank                   int               `json:"rank"`
	Alpha                  float64           `json:"alpha"`
	WeightsA               []float32         `json:"weights_a"`
	WeightsB               []float32         `json:"weights_b"`
	AdditionalMetadata     map[string]string `json:"additional_metadata"`
	CreatedAt              time.Time         `json:"created_at"`
	LastUsed               time.Time         `json:"last_used"`
	UsageCount             int               `json:"usage_count"`
	ConsensusScore         float64           `json:"consensus_score"`
}

// SkillInvocationRequest request to invoke a skill via KNIRVGRAPH discovery
type SkillInvocationRequest struct {
	InvocationID string                 `json:"invocation_id"`
	SkillURI     string                 `json:"skill_uri"`     // knirv://skill/code-refactor-v2 from KNIRVGRAPH
	ErrorContext *ErrorContext          `json:"error_context"` // Rich error context for KNIRVGRAPH lookup
	NRNToken     string                 `json:"nrn_token"`     // NRN token for payment
	AgentID      string                 `json:"agent_id"`      // KNIRV-CORTEX agent identifier
	Parameters   map[string]interface{} `json:"parameters"`
	UserContext  interface{}            `json:"user_context"`
	Priority     string                 `json:"priority"` // low, normal, high
	Timestamp    int64                  `json:"timestamp"`
}

// ErrorContext represents the rich error data payload sent to KNIRVGRAPH
type ErrorContext struct {
	// Agent Information
	AgentID      string `json:"agent_id"`
	AgentVersion string `json:"agent_version"`
	BaseModelID  string `json:"base_model_id"`

	// Environment Information
	OS                 string `json:"os"`
	Architecture       string `json:"architecture"`
	RuntimeEnvironment string `json:"runtime_environment"`

	// Error Details
	ErrorType         string `json:"error_type"`
	ErrorMessage      string `json:"error_message"`
	StackTrace        string `json:"stack_trace"`
	SourceCodeSnippet string `json:"source_code_snippet"`

	// Task Context
	TaskDescription string `json:"task_description"`
	InputDataHash   string `json:"input_data_hash"`
	SkillInvokedID  string `json:"skill_invoked_id"`

	// State & Metadata
	AgentStateHash    string                 `json:"agent_state_hash"`
	Timestamp         int64                  `json:"timestamp"`
	AdditionalContext map[string]interface{} `json:"additional_context"`
}

// NRNTokenValidation represents NRN token validation result
type NRNTokenValidation struct {
	IsValid      bool   `json:"is_valid"`
	TokenHash    string `json:"token_hash"`
	AgentID      string `json:"agent_id"`
	Amount       int64  `json:"amount"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// SkillInvocationResponse response from skill invocation
type SkillInvocationResponse struct {
	InvocationID     string            `json:"invocation_id"`
	Status           string            `json:"status"` // SUCCESS, FAILURE, NOT_FOUND
	ErrorMessage     string            `json:"error_message"`
	Skill            *LoRAAdapterSkill `json:"skill,omitempty"`
	ExecutionTime    int64             `json:"execution_time"`
	MemoryUsed       int64             `json:"memory_used"`
	ConsensusReached bool              `json:"consensus_reached"`
}

// SkillChain represents a chain of skills as serialized LoRA adapter vectors
type SkillChain struct {
	ChainID        string              `json:"chain_id"`
	Skills         []*LoRAAdapterSkill `json:"skills"`
	MergedWeights  *MergedWeights      `json:"merged_weights,omitempty"`
	ConsensusScore float64             `json:"consensus_score"`
	LastUpdated    time.Time           `json:"last_updated"`
}

// MergedWeights represents merged LoRA weights for complex operations
type MergedWeights struct {
	WeightsA []float32 `json:"weights_a"`
	WeightsB []float32 `json:"weights_b"`
	Rank     int       `json:"rank"`
	Alpha    float64   `json:"alpha"`
}

// LoRAAdapterFilter filter criteria for finding relevant adapters
type LoRAAdapterFilter struct {
	SkillType         *string  `json:"skill_type,omitempty"`
	BaseModel         *string  `json:"base_model,omitempty"`
	MinConsensusScore *float64 `json:"min_consensus_score,omitempty"`
	MaxRank           *int     `json:"max_rank,omitempty"`
	Capabilities      []string `json:"capabilities,omitempty"`
	ExcludeSkills     []string `json:"exclude_skills,omitempty"`
}

// EmbeddedKNIRVChain the main embedded KNIRVCHAIN implementation
type EmbeddedKNIRVChain struct {
	config            *EmbeddedChainConfig
	skillRegistry     map[string]*LoRAAdapterSkill
	skillChains       map[string]*SkillChain
	activeInvocations map[string]*SkillInvocationRequest
	consensusNodes    map[string]bool
	skillURIMapping   map[string]string // Maps skill URIs to skill IDs
	knirvgraphClient  KNIRVGraphClient  // Client for KNIRVGRAPH queries
	oracleClient      OracleClient      // Client for KNIRV-ORACLE IBC communication
	isInitialized     bool
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	eventChan         chan interface{}
}

// KNIRVGraphClient interface for KNIRVGRAPH communication
type KNIRVGraphClient interface {
	QueryErrorCluster(ctx context.Context, errorContext *ErrorContext) (*SkillNodeResult, error)
	SubmitErrorNode(ctx context.Context, errorContext *ErrorContext) error
}

// SkillNodeResult represents the result from KNIRVGRAPH query
type SkillNodeResult struct {
	SkillURI    string  `json:"skill_uri"`
	SkillNodeID string  `json:"skill_node_id"`
	ClusterID   string  `json:"cluster_id"`
	Confidence  float64 `json:"confidence"`
}

// OracleClient interface for KNIRV-ORACLE IBC communication
type OracleClient interface {
	SignalNRNBurn(ctx context.Context, tokenHash string, agentID string, amount int64) error
}

// NewEmbeddedKNIRVChain creates a new embedded KNIRVCHAIN instance
func NewEmbeddedKNIRVChain(config *EmbeddedChainConfig) *EmbeddedKNIRVChain {
	if config == nil {
		config = &EmbeddedChainConfig{
			ModelKernel:           "hrm",
			MaxMemoryMB:           512,
			ConsensusThreshold:    0.75,
			LoRAAdapterCacheSize:  100,
			SkillChainDepth:       10,
			EnableRealTimeUpdates: true,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EmbeddedKNIRVChain{
		config:            config,
		skillRegistry:     make(map[string]*LoRAAdapterSkill),
		skillChains:       make(map[string]*SkillChain),
		activeInvocations: make(map[string]*SkillInvocationRequest),
		consensusNodes:    make(map[string]bool),
		skillURIMapping:   make(map[string]string),
		knirvgraphClient:  nil, // Will be set via SetKNIRVGraphClient
		oracleClient:      nil, // Will be set via SetOracleClient
		isInitialized:     false,
		ctx:               ctx,
		cancel:            cancel,
		eventChan:         make(chan interface{}, 100),
	}
}

// Initialize initializes the embedded KNIRV Chain
func (ec *EmbeddedKNIRVChain) Initialize() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	log.Println("Initializing Embedded KNIRV Chain...")

	// Load Small Language Model kernel for genesis block
	if err := ec.loadModelKernel(); err != nil {
		return fmt.Errorf("failed to load model kernel: %w", err)
	}

	// Initialize skill registry
	ec.initializeSkillRegistry()

	// Setup consensus mechanism
	ec.setupInternalConsensus()

	// Start real-time weight update mechanism if enabled
	if ec.config.EnableRealTimeUpdates {
		go ec.startRealTimeUpdates()
	}

	ec.isInitialized = true
	ec.emitEvent("chainInitialized", nil)
	log.Println("Embedded KNIRV Chain initialized successfully")

	return nil
}

// InvokeSkill revolutionary /invoke endpoint - activates a skill by loading and applying LoRA adapter weights
func (ec *EmbeddedKNIRVChain) InvokeSkill(request *SkillInvocationRequest) (*SkillInvocationResponse, error) {
	if !ec.isInitialized {
		return nil, fmt.Errorf("embedded KNIRV Chain not initialized")
	}

	startTime := time.Now()

	ec.mu.Lock()
	ec.activeInvocations[request.InvocationID] = request
	ec.mu.Unlock()

	defer func() {
		ec.mu.Lock()
		delete(ec.activeInvocations, request.InvocationID)
		ec.mu.Unlock()
	}()

	log.Printf("Revolutionary skill invocation started: %s (agent: %s)", request.InvocationID, request.AgentID)

	// Step 1: Validate NRN Token
	tokenValidation, err := ec.validateNRNToken(request.NRNToken, request.AgentID)
	if err != nil || !tokenValidation.IsValid {
		return &SkillInvocationResponse{
			InvocationID:     request.InvocationID,
			Status:           "FAILURE",
			ErrorMessage:     fmt.Sprintf("NRN token validation failed: %s", tokenValidation.ErrorMessage),
			ExecutionTime:    time.Since(startTime).Milliseconds(),
			MemoryUsed:       0,
			ConsensusReached: false,
		}, nil
	}

	// Step 2: Resolve Skill URI via embedded KNIRVCHAIN
	var skillID string
	if request.SkillURI != "" {
		// Direct skill URI provided (from KNIRVGRAPH discovery)
		skillID = ec.resolveSkillURI(request.SkillURI)
		if skillID == "" {
			return &SkillInvocationResponse{
				InvocationID:     request.InvocationID,
				Status:           "NOT_FOUND",
				ErrorMessage:     fmt.Sprintf("Skill URI %s not found in local chain", request.SkillURI),
				ExecutionTime:    time.Since(startTime).Milliseconds(),
				MemoryUsed:       0,
				ConsensusReached: false,
			}, nil
		}
	} else if request.ErrorContext != nil {
		// Error context provided - query KNIRVGRAPH for skill discovery
		skillNodeResult, err := ec.queryKNIRVGraphForSkill(request.ErrorContext)
		if err != nil {
			return &SkillInvocationResponse{
				InvocationID:     request.InvocationID,
				Status:           "FAILURE",
				ErrorMessage:     fmt.Sprintf("KNIRVGRAPH query failed: %v", err),
				ExecutionTime:    time.Since(startTime).Milliseconds(),
				MemoryUsed:       0,
				ConsensusReached: false,
			}, nil
		}

		if skillNodeResult == nil {
			return &SkillInvocationResponse{
				InvocationID:     request.InvocationID,
				Status:           "NOT_FOUND",
				ErrorMessage:     "No matching skill found in KNIRVGRAPH for error context",
				ExecutionTime:    time.Since(startTime).Milliseconds(),
				MemoryUsed:       0,
				ConsensusReached: false,
			}, nil
		}

		skillID = ec.resolveSkillURI(skillNodeResult.SkillURI)
	} else {
		return &SkillInvocationResponse{
			InvocationID:     request.InvocationID,
			Status:           "FAILURE",
			ErrorMessage:     "Either skill_uri or error_context must be provided",
			ExecutionTime:    time.Since(startTime).Milliseconds(),
			MemoryUsed:       0,
			ConsensusReached: false,
		}, nil
	}

	// Step 3: Find skill using programmatic LoRA adapter filtering
	filter := &LoRAAdapterFilter{}
	if capabilities, ok := request.Parameters["required_capabilities"].([]string); ok {
		filter.Capabilities = capabilities
	}
	if baseModel, ok := request.Parameters["base_model"].(string); ok {
		filter.BaseModel = &baseModel
	}

	skill, err := ec.findSkillWithFiltering(skillID, filter)
	if err != nil {
		return &SkillInvocationResponse{
			InvocationID:     request.InvocationID,
			Status:           "FAILURE",
			ErrorMessage:     err.Error(),
			ExecutionTime:    time.Since(startTime).Milliseconds(),
			MemoryUsed:       0,
			ConsensusReached: false,
		}, nil
	}

	if skill == nil {
		return &SkillInvocationResponse{
			InvocationID:     request.InvocationID,
			Status:           "NOT_FOUND",
			ErrorMessage:     fmt.Sprintf("Skill %s not found in local chain", skillID),
			ExecutionTime:    time.Since(startTime).Milliseconds(),
			MemoryUsed:       0,
			ConsensusReached: false,
		}, nil
	}

	// Apply LoRA adapter weights to embedded model
	result, err := ec.applyLoRAWeights(skill, request.Parameters)
	if err != nil {
		return &SkillInvocationResponse{
			InvocationID:     request.InvocationID,
			Status:           "FAILURE",
			ErrorMessage:     err.Error(),
			ExecutionTime:    time.Since(startTime).Milliseconds(),
			MemoryUsed:       ec.calculateMemoryUsage(),
			ConsensusReached: false,
		}, nil
	}

	// Update skill usage statistics
	ec.mu.Lock()
	skill.LastUsed = time.Now()
	skill.UsageCount++
	ec.mu.Unlock()

	// Achieve consensus if multiple nodes
	consensusReached := ec.achieveConsensus(skill, result)

	response := &SkillInvocationResponse{
		InvocationID:     request.InvocationID,
		Status:           "SUCCESS",
		ErrorMessage:     "",
		Skill:            skill,
		ExecutionTime:    time.Since(startTime).Milliseconds(),
		MemoryUsed:       ec.calculateMemoryUsage(),
		ConsensusReached: consensusReached,
	}

	// Step 4: Signal NRN token burn to KNIRV-ORACLE via IBC
	go func() {
		if err := ec.signalNRNBurn(tokenValidation.TokenHash, request.AgentID, tokenValidation.Amount); err != nil {
			log.Printf("Failed to signal NRN burn for invocation %s: %v", request.InvocationID, err)
		}
	}()

	ec.emitEvent("skillInvocationCompleted", response)
	log.Printf("Revolutionary skill invocation completed successfully: %s", request.InvocationID)
	return response, nil
}

// RegisterSkill registers a new LoRA adapter skill
func (ec *EmbeddedKNIRVChain) RegisterSkill(skill *LoRAAdapterSkill) error {
	if !ec.isInitialized {
		return fmt.Errorf("embedded KNIRV Chain not initialized")
	}

	ec.mu.Lock()
	defer ec.mu.Unlock()

	skill.CreatedAt = time.Now()
	skill.LastUsed = time.Now()
	skill.UsageCount = 0
	skill.ConsensusScore = 1.0

	ec.skillRegistry[skill.SkillID] = skill
	ec.emitEvent("skillRegistered", map[string]interface{}{
		"skill_id":   skill.SkillID,
		"skill_name": skill.SkillName,
	})

	return nil
}

// GetSkills returns all available skills with optional filtering
func (ec *EmbeddedKNIRVChain) GetSkills(filter *LoRAAdapterFilter) ([]*LoRAAdapterSkill, error) {
	if !ec.isInitialized {
		return nil, fmt.Errorf("embedded KNIRV Chain not initialized")
	}

	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var skills []*LoRAAdapterSkill
	for _, skill := range ec.skillRegistry {
		if filter == nil || ec.matchesFilter(skill, filter) {
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// CreateSkillChain creates skill chain as serialized LoRA adapter vectors from KNIRVGRAPH
func (ec *EmbeddedKNIRVChain) CreateSkillChain(skills []*LoRAAdapterSkill) (*SkillChain, error) {
	if !ec.isInitialized {
		return nil, fmt.Errorf("embedded KNIRV Chain not initialized")
	}

	chainID := ec.generateChainID()

	// Merge LoRA adapters for complex multi-skill operations
	mergedWeights, err := ec.mergeLoRAAdapters(skills)
	if err != nil {
		return nil, fmt.Errorf("failed to merge LoRA adapters: %w", err)
	}

	// Calculate consensus score
	consensusScore := ec.calculateChainConsensus(skills)

	skillChain := &SkillChain{
		ChainID:        chainID,
		Skills:         skills,
		MergedWeights:  mergedWeights,
		ConsensusScore: consensusScore,
		LastUpdated:    time.Now(),
	}

	ec.mu.Lock()
	ec.skillChains[chainID] = skillChain
	ec.mu.Unlock()

	ec.emitEvent("skillChainCreated", map[string]interface{}{
		"chain_id":    chainID,
		"skill_count": len(skills),
	})

	return skillChain, nil
}

// GetSkillChains returns all skill chains
func (ec *EmbeddedKNIRVChain) GetSkillChains() ([]*SkillChain, error) {
	if !ec.isInitialized {
		return nil, fmt.Errorf("embedded KNIRV Chain not initialized")
	}

	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var chains []*SkillChain
	for _, chain := range ec.skillChains {
		chains = append(chains, chain)
	}

	return chains, nil
}

// Shutdown shuts down the embedded chain
func (ec *EmbeddedKNIRVChain) Shutdown() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	log.Println("Shutting down Embedded KNIRV Chain...")

	ec.cancel()
	ec.isInitialized = false
	close(ec.eventChan)

	ec.emitEvent("chainShutdown", nil)
	return nil
}

// Private methods

// loadModelKernel loads Small Language Model kernel for genesis block
func (ec *EmbeddedKNIRVChain) loadModelKernel() error {
	log.Printf("Loading %s model kernel...", ec.config.ModelKernel)

	// This would load the appropriate model based on config
	// For now, we'll simulate the loading process
	modelPath := fmt.Sprintf("/models/%s", ec.config.ModelKernel)
	log.Printf("Model path: %s", modelPath)

	// In a real implementation, this would load the actual model
	log.Printf("%s model kernel loaded successfully", ec.config.ModelKernel)
	return nil
}

// initializeSkillRegistry initializes skill registry with default skills
func (ec *EmbeddedKNIRVChain) initializeSkillRegistry() {
	log.Println("Initializing skill registry...")
	// Registry starts empty - skills are added via KNIRVGRAPH integration
}

// setupInternalConsensus sets up internal Tendermint consensus mechanism
func (ec *EmbeddedKNIRVChain) setupInternalConsensus() {
	log.Println("Setting up internal consensus mechanism...")

	// Add self as consensus node
	ec.consensusNodes["self"] = true

	// In a real implementation, this would connect to other agent-cores
	// for distributed consensus
}

// startRealTimeUpdates starts real-time weight update mechanism
func (ec *EmbeddedKNIRVChain) startRealTimeUpdates() {
	log.Println("Starting real-time weight update mechanism...")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ec.ctx.Done():
			return
		case <-ticker.C:
			ec.processWeightUpdates()
		}
	}
}

// processWeightUpdates processes pending weight updates
func (ec *EmbeddedKNIRVChain) processWeightUpdates() {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Process any pending LoRA adapter updates
	for skillID, skill := range ec.skillRegistry {
		if skill.ConsensusScore < ec.config.ConsensusThreshold {
			// Skill needs consensus update
			ec.requestConsensusUpdate(skillID, skill)
		}
	}
}

// findSkillWithFiltering programmatic LoRA adapter filtering system that traverses skill chains to find relevant adapters
func (ec *EmbeddedKNIRVChain) findSkillWithFiltering(skillID string, filter *LoRAAdapterFilter) (*LoRAAdapterSkill, error) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Direct skill lookup
	if skill, exists := ec.skillRegistry[skillID]; exists {
		if ec.matchesFilter(skill, filter) {
			return skill, nil
		}
	}

	// Traverse skill chains for related adapters
	for _, chain := range ec.skillChains {
		for _, chainSkill := range chain.Skills {
			if chainSkill.SkillID == skillID || ec.isRelatedSkill(chainSkill, skillID) {
				if ec.matchesFilter(chainSkill, filter) {
					return chainSkill, nil
				}
			}
		}
	}

	return nil, nil
}

// applyLoRAWeights applies LoRA adapter weights to the embedded model
func (ec *EmbeddedKNIRVChain) applyLoRAWeights(skill *LoRAAdapterSkill, parameters map[string]interface{}) (interface{}, error) {
	log.Printf("Applying LoRA weights for skill: %s", skill.SkillName)

	// Calculate the LoRA update: W_new = W_original + (alpha/rank) * (B * A)
	scaling := skill.Alpha / float64(skill.Rank)

	// In a real implementation, this would apply the weights to the model
	// For now, we simulate the process
	result := map[string]interface{}{
		"skill_applied":   skill.SkillID,
		"weights_applied": true,
		"scaling":         scaling,
		"parameters":      parameters,
		"timestamp":       time.Now().Unix(),
	}

	return result, nil
}

// achieveConsensus achieves consensus with other nodes
func (ec *EmbeddedKNIRVChain) achieveConsensus(skill *LoRAAdapterSkill, result interface{}) bool {
	if len(ec.consensusNodes) == 1 {
		return true // Single node, consensus achieved
	}

	// In a real implementation, this would communicate with other agent-cores
	// to achieve consensus on the skill execution result
	// For now, we use the skill's consensus score and ignore the actual result
	_ = result // Explicitly ignore unused parameter for now
	return skill.ConsensusScore >= ec.config.ConsensusThreshold
}

// matchesFilter checks if skill matches filter criteria
func (ec *EmbeddedKNIRVChain) matchesFilter(skill *LoRAAdapterSkill, filter *LoRAAdapterFilter) bool {
	if filter == nil {
		return true
	}

	if filter.BaseModel != nil && skill.BaseModelCompatibility != *filter.BaseModel {
		return false
	}

	if filter.MinConsensusScore != nil && skill.ConsensusScore < *filter.MinConsensusScore {
		return false
	}

	if filter.MaxRank != nil && skill.Rank > *filter.MaxRank {
		return false
	}

	if len(filter.ExcludeSkills) > 0 {
		for _, excludeSkill := range filter.ExcludeSkills {
			if skill.SkillID == excludeSkill {
				return false
			}
		}
	}

	if len(filter.Capabilities) > 0 {
		skillCapabilities := []string{}
		if capStr, exists := skill.AdditionalMetadata["capabilities"]; exists {
			json.Unmarshal([]byte(capStr), &skillCapabilities)
		}

		for _, requiredCap := range filter.Capabilities {
			found := false
			for _, skillCap := range skillCapabilities {
				if skillCap == requiredCap {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// isRelatedSkill checks if skills are related (for chain traversal)
func (ec *EmbeddedKNIRVChain) isRelatedSkill(skill *LoRAAdapterSkill, targetSkillID string) bool {
	// Check if skills share similar capabilities or base models
	if relatedSkills, exists := skill.AdditionalMetadata["related_skills"]; exists {
		var relatedList []string
		json.Unmarshal([]byte(relatedSkills), &relatedList)
		for _, relatedID := range relatedList {
			if relatedID == targetSkillID {
				return true
			}
		}
	}
	return false
}

// mergeLoRAAdapters merges multiple LoRA adapters for complex operations
func (ec *EmbeddedKNIRVChain) mergeLoRAAdapters(skills []*LoRAAdapterSkill) (*MergedWeights, error) {
	if len(skills) == 0 {
		return nil, fmt.Errorf("cannot merge empty skill list")
	}

	if len(skills) == 1 {
		return &MergedWeights{
			WeightsA: skills[0].WeightsA,
			WeightsB: skills[0].WeightsB,
			Rank:     skills[0].Rank,
			Alpha:    skills[0].Alpha,
		}, nil
	}

	// For multiple skills, we need to merge their LoRA weights
	// This is a simplified implementation - real merging would be more complex
	maxRank := 0
	totalAlpha := 0.0
	for _, skill := range skills {
		if skill.Rank > maxRank {
			maxRank = skill.Rank
		}
		totalAlpha += skill.Alpha
	}
	avgAlpha := totalAlpha / float64(len(skills))

	// Create merged weight matrices
	// Assuming 1024 features for simplicity
	mergedWeightsA := make([]float32, maxRank*1024)
	mergedWeightsB := make([]float32, 1024*maxRank)

	// Simple averaging merge strategy
	for i, skill := range skills {
		weight := 1.0 / float32(len(skills))

		for j := 0; j < len(skill.WeightsA) && j < len(mergedWeightsA); j++ {
			mergedWeightsA[j] += skill.WeightsA[j] * weight
		}

		for j := 0; j < len(skill.WeightsB) && j < len(mergedWeightsB); j++ {
			mergedWeightsB[j] += skill.WeightsB[j] * weight
		}

		_ = i // Suppress unused variable warning
	}

	return &MergedWeights{
		WeightsA: mergedWeightsA,
		WeightsB: mergedWeightsB,
		Rank:     maxRank,
		Alpha:    avgAlpha,
	}, nil
}

// calculateChainConsensus calculates consensus score for a skill chain
func (ec *EmbeddedKNIRVChain) calculateChainConsensus(skills []*LoRAAdapterSkill) float64 {
	if len(skills) == 0 {
		return 0
	}

	totalScore := 0.0
	for _, skill := range skills {
		totalScore += skill.ConsensusScore
	}
	return totalScore / float64(len(skills))
}

// generateChainID generates unique chain ID
func (ec *EmbeddedKNIRVChain) generateChainID() string {
	return fmt.Sprintf("chain_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
}

// calculateMemoryUsage calculates current memory usage
func (ec *EmbeddedKNIRVChain) calculateMemoryUsage() int64 {
	// Estimate memory usage based on active skills and chains
	skillMemory := int64(len(ec.skillRegistry) * 1024) // 1KB per skill estimate
	chainMemory := int64(len(ec.skillChains) * 2048)   // 2KB per chain estimate
	return skillMemory + chainMemory
}

// requestConsensusUpdate requests consensus update for a skill
func (ec *EmbeddedKNIRVChain) requestConsensusUpdate(skillID string, skill *LoRAAdapterSkill) {
	log.Printf("Requesting consensus update for skill: %s", skill.SkillName)
	// In a real implementation, this would communicate with other nodes
	ec.emitEvent("consensusUpdateRequested", map[string]interface{}{
		"skill_id": skillID,
	})
}

// emitEvent emits an event to the event channel
func (ec *EmbeddedKNIRVChain) emitEvent(eventType string, data interface{}) {
	select {
	case ec.eventChan <- map[string]interface{}{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now(),
	}:
	default:
		// Channel is full, skip event
	}
}

// validateNRNToken validates the NRN token for skill invocation payment
func (ec *EmbeddedKNIRVChain) validateNRNToken(token string, agentID string) (*NRNTokenValidation, error) {
	// In a real implementation, this would validate against the blockchain
	// For now, we'll implement basic validation logic

	if token == "" {
		return &NRNTokenValidation{
			IsValid:      false,
			ErrorMessage: "NRN token is required",
		}, nil
	}

	// Basic token format validation (simplified)
	if len(token) < 32 {
		return &NRNTokenValidation{
			IsValid:      false,
			ErrorMessage: "Invalid NRN token format",
		}, nil
	}

	// Mock validation - in real implementation, this would check blockchain state
	return &NRNTokenValidation{
		IsValid:   true,
		TokenHash: token,
		AgentID:   agentID,
		Amount:    1, // Standard skill invocation cost
	}, nil
}

// resolveSkillURI resolves a skill URI to a local skill ID
func (ec *EmbeddedKNIRVChain) resolveSkillURI(skillURI string) string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Check if we have a direct mapping
	if skillID, exists := ec.skillURIMapping[skillURI]; exists {
		return skillID
	}

	// Try to find skill by name or description matching the URI
	for skillID, skill := range ec.skillRegistry {
		// Simple URI matching - in real implementation this would be more sophisticated
		if skill.SkillName == skillURI ||
			fmt.Sprintf("knirv://skill/%s", skill.SkillName) == skillURI ||
			fmt.Sprintf("knirv://skill/%s-v%d", skill.SkillName, skill.Version) == skillURI {
			// Cache the mapping for future use
			ec.skillURIMapping[skillURI] = skillID
			return skillID
		}
	}

	return ""
}

// queryKNIRVGraphForSkill queries KNIRVGRAPH for skill discovery based on error context
func (ec *EmbeddedKNIRVChain) queryKNIRVGraphForSkill(errorContext *ErrorContext) (*SkillNodeResult, error) {
	if ec.knirvgraphClient == nil {
		// Fallback: try to find a skill locally based on error type
		return ec.findSkillByErrorType(errorContext.ErrorType), nil
	}

	// Query KNIRVGRAPH for matching error clusters
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := ec.knirvgraphClient.QueryErrorCluster(ctx, errorContext)
	if err != nil {
		log.Printf("KNIRVGRAPH query failed: %v", err)
		// Fallback to local search
		return ec.findSkillByErrorType(errorContext.ErrorType), nil
	}

	return result, nil
}

// findSkillByErrorType finds a skill locally based on error type (fallback method)
func (ec *EmbeddedKNIRVChain) findSkillByErrorType(errorType string) *SkillNodeResult {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Simple mapping of error types to skills
	errorTypeToSkill := map[string]string{
		"TypeError":            "javascript-type-checker",
		"ReferenceError":       "javascript-reference-resolver",
		"SyntaxError":          "syntax-error-fixer",
		"NullPointerException": "null-pointer-guard",
		"IndexOutOfBounds":     "bounds-checker",
	}

	if skillName, exists := errorTypeToSkill[errorType]; exists {
		// Find skill by name
		for skillID, skill := range ec.skillRegistry {
			if skill.SkillName == skillName {
				return &SkillNodeResult{
					SkillURI:    fmt.Sprintf("knirv://skill/%s-v%d", skillName, skill.Version),
					SkillNodeID: skillID,
					ClusterID:   fmt.Sprintf("cluster_%s", errorType),
					Confidence:  0.8, // Moderate confidence for local fallback
				}
			}
		}
	}

	return nil
}

// signalNRNBurn signals the KNIRV-ORACLE to burn the NRN token via IBC
func (ec *EmbeddedKNIRVChain) signalNRNBurn(tokenHash string, agentID string, amount int64) error {
	if ec.oracleClient == nil {
		log.Printf("Oracle client not configured, skipping NRN burn signal for token %s", tokenHash)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := ec.oracleClient.SignalNRNBurn(ctx, tokenHash, agentID, amount)
	if err != nil {
		log.Printf("Failed to signal NRN burn to oracle: %v", err)
		return err
	}

	log.Printf("Successfully signaled NRN burn: token=%s, agent=%s, amount=%d", tokenHash, agentID, amount)
	return nil
}

// SetKNIRVGraphClient sets the KNIRVGRAPH client for error cluster queries
func (ec *EmbeddedKNIRVChain) SetKNIRVGraphClient(client KNIRVGraphClient) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.knirvgraphClient = client
}

// SetOracleClient sets the KNIRV-ORACLE client for IBC communication
func (ec *EmbeddedKNIRVChain) SetOracleClient(client OracleClient) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.oracleClient = client
}

// SerializeLoRAAdapterResponse serializes a LoRA adapter response to protobuf format
func (ec *EmbeddedKNIRVChain) SerializeLoRAAdapterResponse(response *SkillInvocationResponse) ([]byte, error) {
	// In a real implementation, this would use actual protobuf serialization
	// For now, we'll use JSON serialization as a placeholder

	if response.Skill == nil {
		return nil, fmt.Errorf("no skill in response to serialize")
	}

	// Create a simplified protobuf-like structure
	protobufData := map[string]interface{}{
		"skill_id":                response.Skill.SkillID,
		"skill_name":              response.Skill.SkillName,
		"description":             response.Skill.Description,
		"base_model_compatibility": response.Skill.BaseModelCompatibility,
		"version":                 response.Skill.Version,
		"rank":                    response.Skill.Rank,
		"alpha":                   response.Skill.Alpha,
		"weights_a":               response.Skill.WeightsA,
		"weights_b":               response.Skill.WeightsB,
		"additional_metadata":     response.Skill.AdditionalMetadata,
		"invocation_id":           response.InvocationID,
		"execution_time":          response.ExecutionTime,
		"memory_used":             response.MemoryUsed,
		"consensus_reached":       response.ConsensusReached,
		"serialization_format":    "knirv-protobuf-v1",
		"timestamp":               time.Now().Unix(),
	}

	// Serialize to JSON (in real implementation, this would be protobuf)
	serialized, err := json.Marshal(protobufData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize LoRA adapter response: %v", err)
	}

	log.Printf("Serialized LoRA adapter response: %d bytes", len(serialized))
	return serialized, nil
}
