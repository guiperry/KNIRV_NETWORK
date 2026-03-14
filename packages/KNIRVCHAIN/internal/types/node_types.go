package types

import (
	"KNIRVCHAIN/internal/errors"
	"time"
)

// NodeStatus represents the status of a node in the transformation flows
type NodeStatus string

const (
	NodeStatusOpen        NodeStatus = "Open"
	NodeStatusInProgress  NodeStatus = "InProgress"
	NodeStatusResolved    NodeStatus = "Resolved"
	NodeStatusValidated   NodeStatus = "Validated"
	NodeStatusRejected    NodeStatus = "Rejected"
)

// IsValidNodeStatus checks if the provided node status is valid
func IsValidNodeStatus(status NodeStatus) bool {
	switch status {
	case NodeStatusOpen, NodeStatusInProgress, NodeStatusResolved, NodeStatusValidated, NodeStatusRejected:
		return true
	default:
		return false
	}
}

// SkillPerformance represents the performance metrics of a validated skill
type SkillPerformance struct {
	Accuracy       float64 `json:"accuracy"`        // 0.0 to 1.0
	Precision      float64 `json:"precision"`       // 0.0 to 1.0
	Recall         float64 `json:"recall"`          // 0.0 to 1.0
	F1Score        float64 `json:"f1_score"`        // 0.0 to 1.0
	TestCasesRun   uint64  `json:"test_cases_run"`  // number of test cases executed
	TestCasesPass  uint64  `json:"test_cases_pass"` // number of test cases passed
	AvgResponseTime int64   `json:"avg_response_time"` // average response time in milliseconds
}

// Validate checks if the performance metrics are within valid ranges
func (sp *SkillPerformance) Validate() error {
	if sp.Accuracy < 0.0 || sp.Accuracy > 1.0 {
		return errors.NewValidationError("accuracy must be between 0.0 and 1.0")
	}
	if sp.Precision < 0.0 || sp.Precision > 1.0 {
		return errors.NewValidationError("precision must be between 0.0 and 1.0")
	}
	if sp.Recall < 0.0 || sp.Recall > 1.0 {
		return errors.NewValidationError("recall must be between 0.0 and 1.0")
	}
	if sp.F1Score < 0.0 || sp.F1Score > 1.0 {
		return errors.NewValidationError("f1_score must be between 0.0 and 1.0")
	}
	if sp.TestCasesRun == 0 {
		return errors.NewValidationError("test_cases_run must be greater than 0")
	}
	if sp.TestCasesPass > sp.TestCasesRun {
		return errors.NewValidationError("test_cases_pass cannot exceed test_cases_run")
	}
	return nil
}

// AccessControlPolicy defines access control for capabilities
type AccessControlPolicy struct {
	AllowedAddresses []string `json:"allowed_addresses"` // empty means public access
	RequiredNRN      uint64   `json:"required_nrn"`      // minimum NRN balance required
	RateLimit        uint64   `json:"rate_limit"`        // requests per hour
}

// NoveltyScore represents the uniqueness assessment of an idea
type NoveltyScore struct {
	Score         float64 `json:"score"`          // 0.0 to 1.0, higher means more novel
	SimilarIdeas  uint64  `json:"similar_ideas"`  // number of similar ideas found
	Uniqueness    float64 `json:"uniqueness"`     // uniqueness factor
	AssessmentBy  string  `json:"assessment_by"`  // address of assessor
	AssessedAt    int64   `json:"assessed_at"`    // timestamp of assessment
}

// Validate checks if the novelty score is within valid ranges
func (ns *NoveltyScore) Validate() error {
	if ns.Score < 0.0 || ns.Score > 1.0 {
		return errors.NewValidationError("score must be between 0.0 and 1.0")
	}
	if ns.Uniqueness < 0.0 || ns.Uniqueness > 1.0 {
		return errors.NewValidationError("uniqueness must be between 0.0 and 1.0")
	}
	if ns.AssessmentBy == "" {
		return errors.NewValidationError("assessment_by cannot be empty")
	}
	if ns.AssessedAt <= 0 {
		return errors.NewValidationError("assessed_at must be a valid timestamp")
	}
	return nil
}

// IPType defines the type of intellectual property
type IPType string

const (
	IPTypePatent      IPType = "Patent"
	IPTypeCopyright   IPType = "Copyright"
	IPTypeTradeSecret IPType = "TradeSecret"
)

// IsValidIPType checks if the provided IP type is valid
func IsValidIPType(ipType IPType) bool {
	switch ipType {
	case IPTypePatent, IPTypeCopyright, IPTypeTradeSecret:
		return true
	default:
		return false
	}
}

// OwnershipRecord tracks ownership of intellectual property
type OwnershipRecord struct {
	CurrentOwner string            `json:"current_owner"` // current owner's address
	History      []OwnershipEntry  `json:"history"`       // ownership transfer history
}

// OwnershipEntry represents a single ownership transfer
type OwnershipEntry struct {
	Owner     string `json:"owner"`      // owner's address
	Acquired  int64  `json:"acquired"`   // timestamp when acquired
	PriceNRN  uint64 `json:"price_nrn"`  // price paid in NRN (0 for initial minting)
}

// AddEntry adds a new ownership entry to the history
func (or *OwnershipRecord) AddEntry(owner string, priceNRN uint64) {
	entry := OwnershipEntry{
		Owner:    owner,
		Acquired: time.Now().Unix(),
		PriceNRN: priceNRN,
	}
	or.History = append(or.History, entry)
	or.CurrentOwner = owner
}

// ErrorNode represents a failure in the network
type ErrorNode struct {
	ID              string                 `json:"id"`
	ErrorType       string                 `json:"error_type"`        // classification, generation, reasoning, etc.
	ErrorSignature  string                 `json:"error_signature"`   // hash of error pattern
	ModelOrigin     string                 `json:"model_origin"`      // which LLM produced the error
	Context         map[string]interface{} `json:"context"`           // execution context
	FailureCount    uint64                 `json:"failure_count"`     // times this error occurred
	CreatedAt       int64                  `json:"created_at"`
	Status          NodeStatus             `json:"status"`            // Open, InProgress, Resolved
}

// NewErrorNode creates a new ErrorNode with validation
func NewErrorNode(id, errorType, errorSignature, modelOrigin string, context map[string]interface{}) (*ErrorNode, error) {
	if id == "" {
		return nil, errors.NewValidationError("id cannot be empty")
	}
	if errorType == "" {
		return nil, errors.NewValidationError("error_type cannot be empty")
	}
	if errorSignature == "" {
		return nil, errors.NewValidationError("error_signature cannot be empty")
	}
	if modelOrigin == "" {
		return nil, errors.NewValidationError("model_origin cannot be empty")
	}

	node := &ErrorNode{
		ID:             id,
		ErrorType:      errorType,
		ErrorSignature: errorSignature,
		ModelOrigin:    modelOrigin,
		Context:        context,
		FailureCount:   1,
		CreatedAt:      time.Now().Unix(),
		Status:         NodeStatusOpen,
	}

	return node, nil
}

// IncrementFailureCount increases the failure count
func (en *ErrorNode) IncrementFailureCount() {
	en.FailureCount++
}

// UpdateStatus updates the node status with validation
func (en *ErrorNode) UpdateStatus(status NodeStatus) error {
	if !IsValidNodeStatus(status) {
		return errors.NewValidationError("invalid node status")
	}
	en.Status = status
	return nil
}

// SkillNode represents a validated solution
type SkillNode struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	ErrorNodeID     string                 `json:"error_node_id"`     // link to source error
	LoRAPointer     *LoRAAdapterPointer    `json:"lora_pointer"`      // result artifact
	ValidationProof string                 `json:"validation_proof"`  // from KNIRVNEXUS DVE
	Performance     SkillPerformance       `json:"performance"`
	MinerAddress    string                 `json:"miner_address"`
	NRNReward       uint64                 `json:"nrn_reward"`
	CreatedAt       int64                  `json:"created_at"`
}

// NewSkillNode creates a new SkillNode with validation
func NewSkillNode(id, name, errorNodeID string, loraPointer *LoRAAdapterPointer, validationProof string, performance SkillPerformance, minerAddress string, nrnReward uint64) (*SkillNode, error) {
	if id == "" {
		return nil, errors.NewValidationError("id cannot be empty")
	}
	if name == "" {
		return nil, errors.NewValidationError("name cannot be empty")
	}
	if errorNodeID == "" {
		return nil, errors.NewValidationError("error_node_id cannot be empty")
	}
	if loraPointer == nil {
		return nil, errors.NewValidationError("lora_pointer cannot be nil")
	}
	if validationProof == "" {
		return nil, errors.NewValidationError("validation_proof cannot be empty")
	}
	if err := performance.Validate(); err != nil {
		return nil, err
	}
	if minerAddress == "" {
		return nil, errors.NewValidationError("miner_address cannot be empty")
	}

	node := &SkillNode{
		ID:              id,
		Name:            name,
		ErrorNodeID:     errorNodeID,
		LoRAPointer:     loraPointer,
		ValidationProof: validationProof,
		Performance:     performance,
		MinerAddress:    minerAddress,
		NRNReward:       nrnReward,
		CreatedAt:       time.Now().Unix(),
	}

	return node, nil
}

// ContextNode captures execution patterns
type ContextNode struct {
	ID                string                 `json:"id"`
	ContextType       string                 `json:"context_type"`      // tool_use, resource_access, workflow
	PatternSignature  string                 `json:"pattern_signature"` // hash of usage pattern
	UsageFrequency    uint64                 `json:"usage_frequency"`
	RequiredResources []string               `json:"required_resources"`
	Metadata          map[string]interface{} `json:"metadata"`
	CreatedAt         int64                  `json:"created_at"`
}

// NewContextNode creates a new ContextNode with validation
func NewContextNode(id, contextType, patternSignature string, requiredResources []string, metadata map[string]interface{}) (*ContextNode, error) {
	if id == "" {
		return nil, errors.NewValidationError("id cannot be empty")
	}
	if contextType == "" {
		return nil, errors.NewValidationError("context_type cannot be empty")
	}
	if patternSignature == "" {
		return nil, errors.NewValidationError("pattern_signature cannot be empty")
	}

	node := &ContextNode{
		ID:               id,
		ContextType:      contextType,
		PatternSignature: patternSignature,
		UsageFrequency:   1,
		RequiredResources: requiredResources,
		Metadata:         metadata,
		CreatedAt:        time.Now().Unix(),
	}

	return node, nil
}

// IncrementUsageFrequency increases the usage frequency
func (cn *ContextNode) IncrementUsageFrequency() {
	cn.UsageFrequency++
}

// CapabilityNode represents a minted MCP capability
type CapabilityNode struct {
	ID              string               `json:"id"`
	ContextNodeID   string               `json:"context_node_id"`
	MCPPointer      *MCPServerPointer    `json:"mcp_pointer"`
	CapabilityType  CapabilityType       `json:"capability_type"` // Tool, Resource, Prompt, Memory
	AccessControl   AccessControlPolicy  `json:"access_control"`
	MinterAddress   string               `json:"minter_address"`
	NRNCost         uint64               `json:"nrn_cost"`        // cost per invocation
	CreatedAt       int64                `json:"created_at"`
}

// NewCapabilityNode creates a new CapabilityNode with validation
func NewCapabilityNode(id, contextNodeID string, mcpPointer *MCPServerPointer, capabilityType CapabilityType, accessControl AccessControlPolicy, minterAddress string, nrnCost uint64) (*CapabilityNode, error) {
	if id == "" {
		return nil, errors.NewValidationError("id cannot be empty")
	}
	if contextNodeID == "" {
		return nil, errors.NewValidationError("context_node_id cannot be empty")
	}
	if mcpPointer == nil {
		return nil, errors.NewValidationError("mcp_pointer cannot be nil")
	}
	if !IsValidCapabilityType(capabilityType) {
		return nil, errors.NewValidationError("invalid capability type")
	}
	if minterAddress == "" {
		return nil, errors.NewValidationError("minter_address cannot be empty")
	}

	node := &CapabilityNode{
		ID:             id,
		ContextNodeID:  contextNodeID,
		MCPPointer:     mcpPointer,
		CapabilityType: capabilityType,
		AccessControl:  accessControl,
		MinterAddress:  minterAddress,
		NRNCost:        nrnCost,
		CreatedAt:      time.Now().Unix(),
	}

	return node, nil
}

// IdeaNode represents a novel inference or insight
type IdeaNode struct {
	ID              string                 `json:"id"`
	IdeaType        string                 `json:"idea_type"`       // insight, hypothesis, synthesis
	ContentHash     string                 `json:"content_hash"`    // hash of idea content
	OriginNIM       string                 `json:"origin_nim"`      // NIM that generated idea
	Novelty         NoveltyScore           `json:"novelty"`         // uniqueness assessment
	Dependencies    []string               `json:"dependencies"`    // referenced knowledge
	CreatedAt       int64                  `json:"created_at"`
}

// NewIdeaNode creates a new IdeaNode with validation
func NewIdeaNode(id, ideaType, contentHash, originNIM string, novelty NoveltyScore, dependencies []string) (*IdeaNode, error) {
	if id == "" {
		return nil, errors.NewValidationError("id cannot be empty")
	}
	if ideaType == "" {
		return nil, errors.NewValidationError("idea_type cannot be empty")
	}
	if contentHash == "" {
		return nil, errors.NewValidationError("content_hash cannot be empty")
	}
	if originNIM == "" {
		return nil, errors.NewValidationError("origin_nim cannot be empty")
	}
	if err := novelty.Validate(); err != nil {
		return nil, err
	}

	node := &IdeaNode{
		ID:           id,
		IdeaType:     ideaType,
		ContentHash:  contentHash,
		OriginNIM:    originNIM,
		Novelty:      novelty,
		Dependencies: dependencies,
		CreatedAt:    time.Now().Unix(),
	}

	return node, nil
}

// PropertyNode represents minted intellectual property
type PropertyNode struct {
	ID              string               `json:"id"`
	IdeaNodeID      string               `json:"idea_node_id"`
	NFTPointer      *InferenceNFTPointer `json:"nft_pointer"`
	IPType          IPType               `json:"ip_type"`          // Patent, Copyright, TradeSecret
	Ownership       OwnershipRecord      `json:"ownership"`
	Royalties       RoyaltyStructure     `json:"royalties"`
	MakerAddress    string               `json:"maker_address"`
	CreatedAt       int64                `json:"created_at"`
}

// NewPropertyNode creates a new PropertyNode with validation
func NewPropertyNode(id, ideaNodeID string, nftPointer *InferenceNFTPointer, ipType IPType, royalties RoyaltyStructure, makerAddress string) (*PropertyNode, error) {
	if id == "" {
		return nil, errors.NewValidationError("id cannot be empty")
	}
	if ideaNodeID == "" {
		return nil, errors.NewValidationError("idea_node_id cannot be empty")
	}
	if nftPointer == nil {
		return nil, errors.NewValidationError("nft_pointer cannot be nil")
	}
	if !IsValidIPType(ipType) {
		return nil, errors.NewValidationError("invalid IP type")
	}
	if makerAddress == "" {
		return nil, errors.NewValidationError("maker_address cannot be empty")
	}

	ownership := OwnershipRecord{
		CurrentOwner: makerAddress,
		History:      []OwnershipEntry{},
	}
	ownership.AddEntry(makerAddress, 0) // Initial minting

	node := &PropertyNode{
		ID:           id,
		IdeaNodeID:   ideaNodeID,
		NFTPointer:   nftPointer,
		IPType:       ipType,
		Ownership:    ownership,
		Royalties:    royalties,
		MakerAddress: makerAddress,
		CreatedAt:    time.Now().Unix(),
	}

	return node, nil
}

// TransferOwnership transfers ownership of the property
func (pn *PropertyNode) TransferOwnership(newOwner string, priceNRN uint64) error {
	if newOwner == "" {
		return errors.NewValidationError("new_owner cannot be empty")
	}
	if newOwner == pn.Ownership.CurrentOwner {
		return errors.NewValidationError("new_owner cannot be the same as current owner")
	}

	pn.Ownership.AddEntry(newOwner, priceNRN)
	return nil
}