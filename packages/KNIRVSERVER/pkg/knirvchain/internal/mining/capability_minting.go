package mining

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"KNIRVCHAIN/internal/blockchain"
	"KNIRVCHAIN/internal/graph"
	"KNIRVCHAIN/internal/monitoring"
	"KNIRVCHAIN/internal/pricing"
	"KNIRVCHAIN/internal/types"
	"KNIRVCHAIN/internal/validation"
)

// CapabilityMinter handles the minting of capabilities from context nodes
type CapabilityMinter struct {
	graphQueries  *graph.GraphQueries
	nodeStore     *graph.NodeStore
	relManager    *graph.RelationshipManager
	blockchain    *blockchain.BlockchainStruct
	proofVerifier *validation.ProofVerifier
	pricingEngine *pricing.PricingEngine
	metrics       *monitoring.Metrics
}

// NewCapabilityMinter creates a new capability minter
func NewCapabilityMinter(graphQueries *graph.GraphQueries, nodeStore *graph.NodeStore, relManager *graph.RelationshipManager, bc *blockchain.BlockchainStruct, proofVerifier *validation.ProofVerifier) *CapabilityMinter {
	return &CapabilityMinter{
		graphQueries:  graphQueries,
		nodeStore:     nodeStore,
		relManager:    relManager,
		blockchain:    bc,
		proofVerifier: proofVerifier,
		pricingEngine: pricing.NewPricingEngine(),
		metrics:       monitoring.NewMetrics(),
	}
}

// MintingProposal represents a proposal to mint a capability from a context
type MintingProposal struct {
	ContextNodeID        string                    `json:"context_node_id"`
	MinterAddress        string                    `json:"minter_address"`
	MCPServer            *types.MCPServerPointer   `json:"mcp_server"`
	CapabilityType       types.CapabilityType      `json:"capability_type"`
	AccessControl        types.AccessControlPolicy `json:"access_control"`
	ProposedCapabilityID string                    `json:"proposed_capability_id"`
	ProposedAt           int64                     `json:"proposed_at"`
	Status               MintingStatus             `json:"status"`
}

// MintingStatus represents the status of a minting proposal
type MintingStatus string

const (
	MintingStatusProposed   MintingStatus = "proposed"
	MintingStatusValidating MintingStatus = "validating"
	MintingStatusValidated  MintingStatus = "validated"
	MintingStatusRejected   MintingStatus = "rejected"
	MintingStatusMinted     MintingStatus = "minted"
)

// SubmitMintingProposal submits a proposal to mint a capability from a context
func (cm *CapabilityMinter) SubmitMintingProposal(contextNodeID, minterAddress string, mcpServer *types.MCPServerPointer, capabilityType types.CapabilityType, accessControl types.AccessControlPolicy) (*MintingProposal, error) {
	// Validate inputs
	if contextNodeID == "" {
		return nil, fmt.Errorf("context_node_id cannot be empty")
	}
	if minterAddress == "" {
		return nil, fmt.Errorf("minter_address cannot be empty")
	}
	if mcpServer == nil {
		return nil, fmt.Errorf("mcp_server cannot be nil")
	}

	// Verify the context node exists
	_, err := cm.nodeStore.GetContextNode(contextNodeID)
	if err != nil {
		return nil, fmt.Errorf("context node not found: %w", err)
	}

	// Generate proposed capability ID
	proposedCapabilityID := cm.generateCapabilityID(contextNodeID, minterAddress, mcpServer)

	proposal := &MintingProposal{
		ContextNodeID:        contextNodeID,
		MinterAddress:        minterAddress,
		MCPServer:            mcpServer,
		CapabilityType:       capabilityType,
		AccessControl:        accessControl,
		ProposedCapabilityID: proposedCapabilityID,
		ProposedAt:           time.Now().Unix(),
		Status:               MintingStatusProposed,
	}

	// Create transaction for the minting proposal
	tx, err := cm.createMintingProposalTransaction(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to create minting proposal transaction: %w", err)
	}

	// Submit transaction to blockchain
	err = cm.blockchain.AddTransactionToTransactionPool(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to submit minting proposal transaction: %w", err)
	}

	log.Printf("Capability minting proposal submitted: %s for context %s by minter %s", proposedCapabilityID, contextNodeID, minterAddress)

	if cm.metrics != nil {
		cm.metrics.CapabilityMintOps.Inc()
	}

	return proposal, nil
}

// generateCapabilityID generates a unique capability ID
func (cm *CapabilityMinter) generateCapabilityID(contextNodeID, minterAddress string, mcpServer *types.MCPServerPointer) string {
	data := fmt.Sprintf("%s:%s:%s:%d", contextNodeID, minterAddress, mcpServer.ServerID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// createMintingProposalTransaction creates a blockchain transaction for the minting proposal
func (cm *CapabilityMinter) createMintingProposalTransaction(proposal *MintingProposal) (*blockchain.Transaction, error) {
	data, err := json.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proposal data: %w", err)
	}

	tx, err := blockchain.NewMCPTransaction(
		proposal.MinterAddress,
		"", // No specific recipient
		0,  // No value transfer
		data,
		blockchain.TransactionTypeCapabilityMintProposal,
		100, // Fee for proposal submission
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return tx, nil
}

// ValidateCapabilityProposal validates a capability proposal
func (cm *CapabilityMinter) ValidateCapabilityProposal(proposal *MintingProposal) (*types.CapabilityNode, error) {
	startTime := time.Now()

	// Update proposal status
	proposal.Status = MintingStatusValidating

	// Get the context node
	contextNode, err := cm.nodeStore.GetContextNode(proposal.ContextNodeID)
	if err != nil {
		return nil, fmt.Errorf("context node not found: %w", err)
	}

	// Validate MCP server compliance
	validationResult, err := cm.validateMCPServerCompliance(proposal.MCPServer)
	if err != nil {
		proposal.Status = MintingStatusRejected
		if cm.metrics != nil {
			cm.metrics.ErrorCount.Inc()
		}
		return nil, fmt.Errorf("MCP validation failed: %w", err)
	}

	if !validationResult.IsValid {
		proposal.Status = MintingStatusRejected
		if cm.metrics != nil {
			cm.metrics.ErrorCount.Inc()
		}
		return nil, fmt.Errorf("capability validation failed: %s", validationResult.Notes)
	}

	// Create the capability node if validation passed
	capabilityNode, err := cm.createValidatedCapabilityNode(contextNode, proposal, validationResult)
	if err != nil {
		proposal.Status = MintingStatusRejected
		if cm.metrics != nil {
			cm.metrics.ErrorCount.Inc()
		}
		return nil, fmt.Errorf("failed to create capability node: %w", err)
	}

	proposal.Status = MintingStatusValidated

	if cm.metrics != nil {
		cm.metrics.ValidationDuration.Observe(time.Since(startTime).Seconds())
		cm.metrics.CapabilityNodesTotal.Inc()
	}

	return capabilityNode, nil
}

// validateMCPServerCompliance validates MCP server compliance
func (cm *CapabilityMinter) validateMCPServerCompliance(mcpServer *types.MCPServerPointer) (*validation.CapabilityValidationResult, error) {
	// Create a temporary capability node for validation
	tempCapabilityNode := &types.CapabilityNode{
		MCPPointer: mcpServer,
	}

	// Use the proof verifier to validate MCP compliance
	return cm.proofVerifier.ValidateCapabilityProposal(tempCapabilityNode)
}

// createValidatedCapabilityNode creates a capability node from a validated proposal
func (cm *CapabilityMinter) createValidatedCapabilityNode(contextNode *types.ContextNode, proposal *MintingProposal, validationResult *validation.CapabilityValidationResult) (*types.CapabilityNode, error) {
	// Calculate NRN cost based on capability type and validation score
	nrnCost := cm.calculateNRNCost(proposal.CapabilityType, validationResult.ComplianceScore, contextNode.UsageFrequency)

	capabilityNode, err := types.NewCapabilityNode(
		proposal.ProposedCapabilityID,
		proposal.ContextNodeID,
		proposal.MCPServer,
		proposal.CapabilityType,
		proposal.AccessControl,
		proposal.MinterAddress,
		nrnCost,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create capability node: %w", err)
	}

	// Store the capability node
	err = cm.nodeStore.StoreCapabilityNode(capabilityNode)
	if err != nil {
		return nil, fmt.Errorf("failed to store capability node: %w", err)
	}

	// Create relationship between context node and capability node
	_, err = cm.relManager.CreateRelationship(
		"context_node", contextNode.ID,
		"capability_node", capabilityNode.ID,
		graph.RelationshipTypeContextToCapability,
		map[string]interface{}{
			"validation_score": validationResult.ComplianceScore,
			"minter_address":   proposal.MinterAddress,
			"created_at":       capabilityNode.CreatedAt,
		},
	)
	if err != nil {
		log.Printf("Warning: failed to create relationship between context and capability node: %v", err)
	}

	return capabilityNode, nil
}

// calculateNRNCost calculates the NRN cost for using a capability
func (cm *CapabilityMinter) calculateNRNCost(capabilityType types.CapabilityType, complianceScore float64, usageFrequency uint64) uint64 {
	// Base cost depends on capability type
	baseCost := uint64(10) // Default base cost

	switch capabilityType {
	case types.CapabilityTypeTool:
		baseCost = 50
	case types.CapabilityTypeResource:
		baseCost = 20
	case types.CapabilityTypePrompt:
		baseCost = 5
	case types.CapabilityTypeMemoryService:
		baseCost = 30
	}

	// Adjust based on compliance score (higher compliance = lower cost)
	complianceMultiplier := 2.0 - complianceScore // Range: 1.0 to 2.0

	// Adjust based on usage frequency (higher frequency = lower cost per use)
	frequencyMultiplier := 1.0
	if usageFrequency > 100 {
		frequencyMultiplier = 0.8
	} else if usageFrequency > 1000 {
		frequencyMultiplier = 0.6
	}

	cost := uint64(float64(baseCost) * complianceMultiplier * frequencyMultiplier)

	// Minimum cost
	if cost < 1 {
		cost = 1
	}

	// Maximum cost
	if cost > 1000 {
		cost = 1000
	}

	return cost
}

// MintCapabilityNode mints a validated capability to the registry
func (cm *CapabilityMinter) MintCapabilityNode(capabilityNodeID string) error {
	// Get the capability node
	capabilityNode, err := cm.nodeStore.GetCapabilityNode(capabilityNodeID)
	if err != nil {
		return fmt.Errorf("capability node not found: %w", err)
	}

	// Create minting transaction
	mintingData := map[string]interface{}{
		"capability_node_id": capabilityNodeID,
		"minted_at":          time.Now().Unix(),
	}

	data, err := json.Marshal(mintingData)
	if err != nil {
		return fmt.Errorf("failed to marshal minting data: %w", err)
	}

	tx, err := blockchain.NewMCPTransaction(
		capabilityNode.MinterAddress,
		"", // No specific recipient
		0,  // No value transfer
		data,
		blockchain.TransactionTypeCapabilityMint,
		200, // Fee for minting
	)
	if err != nil {
		return fmt.Errorf("failed to create minting transaction: %w", err)
	}

	// Submit transaction
	err = cm.blockchain.AddTransactionToTransactionPool(tx)
	if err != nil {
		return fmt.Errorf("failed to submit minting transaction: %w", err)
	}

	log.Printf("Capability node minted: %s", capabilityNodeID)
	return nil
}

// GetMintingProposalsForContext gets all minting proposals for a specific context
func (cm *CapabilityMinter) GetMintingProposalsForContext(contextNodeID string) ([]*MintingProposal, error) {
	// This would typically query a database of proposals
	// For now, return empty slice as proposals are handled via transactions
	return []*MintingProposal{}, nil
}

// GetCapabilitiesByMinter gets all minted capabilities by a specific minter
func (cm *CapabilityMinter) GetCapabilitiesByMinter(minterAddress string) ([]*types.CapabilityNode, error) {
	return cm.graphQueries.FindCapabilityNodesByMinter(minterAddress, 0)
}

// GetCapabilitiesByType gets all capabilities of a specific type
func (cm *CapabilityMinter) GetCapabilitiesByType(capabilityType types.CapabilityType) ([]*types.CapabilityNode, error) {
	return cm.graphQueries.FindCapabilityNodesByType(capabilityType, 0)
}

// RegisterCapabilityServer registers a capability server for discovery
func (cm *CapabilityMinter) RegisterCapabilityServer(capabilityNodeID string) error {
	capabilityNode, err := cm.nodeStore.GetCapabilityNode(capabilityNodeID)
	if err != nil {
		return fmt.Errorf("capability node not found: %w", err)
	}

	// Register with discovery service (this would integrate with KNIRVROUTER)
	err = cm.registerWithDiscoveryService(capabilityNode)
	if err != nil {
		return fmt.Errorf("failed to register with discovery service: %w", err)
	}

	log.Printf("Capability server registered: %s", capabilityNodeID)
	return nil
}

// registerWithDiscoveryService simulates registration with a discovery service
func (cm *CapabilityMinter) registerWithDiscoveryService(capabilityNode *types.CapabilityNode) error {
	// This would make a call to KNIRVROUTER's discovery service
	// For now, just log the registration
	log.Printf("Registering capability %s (%s) at %s", capabilityNode.ID, capabilityNode.CapabilityType, capabilityNode.MCPPointer.EndpointURI)
	return nil
}

// UnregisterCapabilityServer unregisters a capability server
func (cm *CapabilityMinter) UnregisterCapabilityServer(capabilityNodeID string) error {
	// This would call KNIRVROUTER to remove from discovery
	log.Printf("Capability server unregistered: %s", capabilityNodeID)
	return nil
}

// DiscoverCapabilities discovers available capabilities by type
func (cm *CapabilityMinter) DiscoverCapabilities(capabilityType types.CapabilityType, limit int) ([]*types.CapabilityNode, error) {
	// This would query KNIRVROUTER's discovery service
	// For now, return capabilities from our local graph
	return cm.graphQueries.FindCapabilityNodesByType(capabilityType, limit)
}

// InvokeCapability invokes a capability (this would typically be done by NIMs)
func (cm *CapabilityMinter) InvokeCapability(ctx context.Context, capabilityNodeID, invokerAddress string, parameters map[string]interface{}) (interface{}, error) {
	capabilityNode, err := cm.nodeStore.GetCapabilityNode(capabilityNodeID)
	if err != nil {
		return nil, fmt.Errorf("capability node not found: %w", err)
	}

	// Check access control
	if err := cm.checkAccessControl(capabilityNode, invokerAddress); err != nil {
		return nil, fmt.Errorf("access control check failed: %w", err)
	}

	// Check if invoker has sufficient NRN balance
	if err := cm.checkNRNBalance(invokerAddress, capabilityNode.NRNCost); err != nil {
		return nil, fmt.Errorf("insufficient NRN balance: %w", err)
	}

	// Invoke the capability via MCP
	result, err := cm.invokeViaMCP(ctx, capabilityNode, parameters)
	if err != nil {
		return nil, fmt.Errorf("MCP invocation failed: %w", err)
	}

	// Deduct NRN cost
	if err := cm.deductNRNCost(invokerAddress, capabilityNode.NRNCost); err != nil {
		log.Printf("Warning: failed to deduct NRN cost: %v", err)
	}

	return result, nil
}

// checkAccessControl checks if the invoker has access to the capability
func (cm *CapabilityMinter) checkAccessControl(capabilityNode *types.CapabilityNode, invokerAddress string) error {
	// If no addresses specified, it's public
	if len(capabilityNode.AccessControl.AllowedAddresses) == 0 {
		return nil
	}

	// Check if invoker is in allowed addresses
	for _, addr := range capabilityNode.AccessControl.AllowedAddresses {
		if addr == invokerAddress {
			return nil
		}
	}

	return fmt.Errorf("invoker %s not in allowed addresses", invokerAddress)
}

// checkNRNBalance checks if the invoker has sufficient NRN balance
func (cm *CapabilityMinter) checkNRNBalance(invokerAddress string, requiredAmount uint64) error {
	// This would check the blockchain balance
	// For now, assume sufficient balance
	return nil
}

// deductNRNCost deducts the NRN cost from the invoker's balance
func (cm *CapabilityMinter) deductNRNCost(invokerAddress string, amount uint64) error {
	// This would create a transaction to deduct NRN
	// For now, just log
	log.Printf("Deducted %d NRN from %s for capability invocation", amount, invokerAddress)
	return nil
}

// invokeViaMCP invokes the capability via MCP protocol
func (cm *CapabilityMinter) invokeViaMCP(ctx context.Context, capabilityNode *types.CapabilityNode, parameters map[string]interface{}) (interface{}, error) {
	// This would make an actual MCP call to the server
	// For now, return a mock result
	result := map[string]interface{}{
		"result":        "capability invoked successfully",
		"output":        "mock output",
		"invocation_id": fmt.Sprintf("inv_%d", time.Now().Unix()),
	}

	return result, nil
}
