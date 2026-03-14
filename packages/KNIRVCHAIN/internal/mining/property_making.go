package mining

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"KNIRVCHAIN/internal/blockchain"
	"KNIRVCHAIN/internal/graph"
	"KNIRVCHAIN/internal/types"
	"KNIRVCHAIN/internal/validation"
)

// PropertyMaker handles the making of properties from idea nodes
type PropertyMaker struct {
	graphQueries   *graph.GraphQueries
	nodeStore      *graph.NodeStore
	relManager     *graph.RelationshipManager
	blockchain     *blockchain.BlockchainStruct
	noveltyChecker *validation.NoveltyChecker
	nftContract    *NFTContractClient
}

// NewPropertyMaker creates a new property maker
func NewPropertyMaker(graphQueries *graph.GraphQueries, nodeStore *graph.NodeStore, relManager *graph.RelationshipManager, bc *blockchain.BlockchainStruct, noveltyChecker *validation.NoveltyChecker, nftContract *NFTContractClient) *PropertyMaker {
	return &PropertyMaker{
		graphQueries:   graphQueries,
		nodeStore:      nodeStore,
		relManager:     relManager,
		blockchain:     bc,
		noveltyChecker: noveltyChecker,
		nftContract:    nftContract,
	}
}

// MakingProposal represents a proposal to make a property from an idea
type MakingProposal struct {
	IdeaNodeID       string                      `json:"idea_node_id"`
	MakerAddress     string                      `json:"maker_address"`
	IPType           types.IPType                `json:"ip_type"`
	Royalties        types.RoyaltyStructure      `json:"royalties"`
	ProposedPropertyID string                    `json:"proposed_property_id"`
	ProposedAt       int64                       `json:"proposed_at"`
	Status           MakingStatus                `json:"status"`
	NoveltyAssessment *validation.NoveltyResult  `json:"novelty_assessment,omitempty"`
}

// MakingStatus represents the status of a making proposal
type MakingStatus string

const (
	MakingStatusProposed     MakingStatus = "proposed"
	MakingStatusAssessing    MakingStatus = "assessing"
	MakingStatusAssessed     MakingStatus = "assessed"
	MakingStatusMinting      MakingStatus = "minting"
	MakingStatusMinted       MakingStatus = "minted"
	MakingStatusRejected     MakingStatus = "rejected"
)

// SubmitMakingProposal submits a proposal to make a property from an idea
func (pm *PropertyMaker) SubmitMakingProposal(ideaNodeID, makerAddress string, ipType types.IPType, royalties types.RoyaltyStructure) (*MakingProposal, error) {
	// Validate inputs
	if ideaNodeID == "" {
		return nil, fmt.Errorf("idea_node_id cannot be empty")
	}
	if makerAddress == "" {
		return nil, fmt.Errorf("maker_address cannot be empty")
	}

	// Verify the idea node exists
	_, err := pm.nodeStore.GetIdeaNode(ideaNodeID)
	if err != nil {
		return nil, fmt.Errorf("idea node not found: %w", err)
	}

	// Generate proposed property ID
	proposedPropertyID := pm.generatePropertyID(ideaNodeID, makerAddress, ipType)

	proposal := &MakingProposal{
		IdeaNodeID:         ideaNodeID,
		MakerAddress:       makerAddress,
		IPType:            ipType,
		Royalties:         royalties,
		ProposedPropertyID: proposedPropertyID,
		ProposedAt:        time.Now().Unix(),
		Status:            MakingStatusProposed,
	}

	// Create transaction for the making proposal
	tx, err := pm.createMakingProposalTransaction(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to create making proposal transaction: %w", err)
	}

	// Submit transaction to blockchain
	err = pm.blockchain.AddTransactionToTransactionPool(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to submit making proposal transaction: %w", err)
	}

	log.Printf("Property making proposal submitted: %s for idea %s by maker %s", proposedPropertyID, ideaNodeID, makerAddress)

	return proposal, nil
}

// generatePropertyID generates a unique property ID
func (pm *PropertyMaker) generatePropertyID(ideaNodeID, makerAddress string, ipType types.IPType) string {
	data := fmt.Sprintf("%s:%s:%s:%d", ideaNodeID, makerAddress, ipType, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// createMakingProposalTransaction creates a blockchain transaction for the making proposal
func (pm *PropertyMaker) createMakingProposalTransaction(proposal *MakingProposal) (*blockchain.Transaction, error) {
	data, err := json.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proposal data: %w", err)
	}

	tx, err := blockchain.NewMCPTransaction(
		proposal.MakerAddress,
		"", // No specific recipient
		0,  // No value transfer
		data,
		blockchain.TransactionTypePropertyMint,
		100, // Fee for proposal submission
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return tx, nil
}

// AssessNovelty assesses the novelty of an idea
func (pm *PropertyMaker) AssessNovelty(proposal *MakingProposal) (*validation.NoveltyResult, error) {
	// Update proposal status
	proposal.Status = MakingStatusAssessing

	// Get the idea node
	ideaNode, err := pm.nodeStore.GetIdeaNode(proposal.IdeaNodeID)
	if err != nil {
		return nil, fmt.Errorf("idea node not found: %w", err)
	}

	// Perform novelty assessment
	noveltyResult, err := pm.noveltyChecker.AssessNovelty(ideaNode)
	if err != nil {
		proposal.Status = MakingStatusRejected
		return nil, fmt.Errorf("novelty assessment failed: %w", err)
	}

	proposal.NoveltyAssessment = noveltyResult
	proposal.Status = MakingStatusAssessed

	return noveltyResult, nil
}

// MintProperty mints a property NFT from a validated idea
func (pm *PropertyMaker) MintProperty(proposal *MakingProposal) (*types.PropertyNode, error) {
	// Check if novelty assessment passed
	if proposal.NoveltyAssessment == nil || !proposal.NoveltyAssessment.IsNovel {
		proposal.Status = MakingStatusRejected
		return nil, fmt.Errorf("idea does not meet novelty requirements")
	}

	// Update proposal status
	proposal.Status = MakingStatusMinting

	// Get the idea node
	ideaNode, err := pm.nodeStore.GetIdeaNode(proposal.IdeaNodeID)
	if err != nil {
		return nil, fmt.Errorf("idea node not found: %w", err)
	}

	// Create NFT on-chain
	nftPointer, err := pm.mintNFT(ideaNode, proposal)
	if err != nil {
		proposal.Status = MakingStatusRejected
		return nil, fmt.Errorf("NFT minting failed: %w", err)
	}

	// Create the property node
	propertyNode, err := types.NewPropertyNode(
		proposal.ProposedPropertyID,
		proposal.IdeaNodeID,
		nftPointer,
		proposal.IPType,
		proposal.Royalties,
		proposal.MakerAddress,
	)

	if err != nil {
		proposal.Status = MakingStatusRejected
		return nil, fmt.Errorf("failed to create property node: %w", err)
	}

	// Store the property node
	err = pm.nodeStore.StorePropertyNode(propertyNode)
	if err != nil {
		proposal.Status = MakingStatusRejected
		return nil, fmt.Errorf("failed to store property node: %w", err)
	}

	// Create relationship between idea node and property node
	_, err = pm.relManager.CreateRelationship(
		"idea_node", ideaNode.ID,
		"property_node", propertyNode.ID,
		graph.RelationshipTypeIdeaToProperty,
		map[string]interface{}{
			"novelty_score": proposal.NoveltyAssessment.Score,
			"maker_address": proposal.MakerAddress,
			"created_at":    propertyNode.CreatedAt,
		},
	)
	if err != nil {
		log.Printf("Warning: failed to create relationship between idea and property node: %v", err)
	}

	proposal.Status = MakingStatusMinted

	log.Printf("Property minted: %s (NFT: %s)", propertyNode.ID, nftPointer.TokenID)
	return propertyNode, nil
}

// mintNFT mints an NFT on-chain
func (pm *PropertyMaker) mintNFT(ideaNode *types.IdeaNode, proposal *MakingProposal) (*types.InferenceNFTPointer, error) {
	// Generate metadata
	metadata := pm.generateNFTMetadata(ideaNode, proposal)

	// Mint NFT via contract
	tokenID, contractAddress, err := pm.nftContract.MintNFT(proposal.MakerAddress, metadata)
	if err != nil {
		return nil, fmt.Errorf("contract minting failed: %w", err)
	}

	// Create NFT pointer
	nftPointer, err := types.NewInferenceNFTPointer(
		tokenID,
		contractAddress,
		fmt.Sprintf("ipfs://%s", metadata["metadata_cid"]),
		"Standard IP license terms",
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create NFT pointer: %w", err)
	}

	// Add provenance information
	nftPointer.AddToProvenanceChain(proposal.MakerAddress)

	return nftPointer, nil
}

// generateNFTMetadata generates metadata for the NFT
func (pm *PropertyMaker) generateNFTMetadata(ideaNode *types.IdeaNode, proposal *MakingProposal) map[string]interface{} {
	metadata := map[string]interface{}{
		"name":        fmt.Sprintf("KNIRVCHAIN Property: %s", ideaNode.IdeaType),
		"description": fmt.Sprintf("Intellectual property derived from idea: %s", ideaNode.ContentHash),
		"idea_type":   ideaNode.IdeaType,
		"ip_type":     proposal.IPType,
		"novelty_score": proposal.NoveltyAssessment.Score,
		"origin_nim":  ideaNode.OriginNIM,
		"maker":       proposal.MakerAddress,
		"created_at":  time.Now().Unix(),
		"royalties":   proposal.Royalties,
	}

	// Generate a mock IPFS CID for metadata
	metadataData, _ := json.Marshal(metadata)
	hash := sha256.Sum256(metadataData)
	metadata["metadata_cid"] = fmt.Sprintf("Qm%s", hex.EncodeToString(hash[:16]))

	return metadata
}

// TransferProperty transfers ownership of a property
func (pm *PropertyMaker) TransferProperty(propertyNodeID, fromAddress, toAddress string, priceNRN uint64) error {
	// Get the property node
	propertyNode, err := pm.nodeStore.GetPropertyNode(propertyNodeID)
	if err != nil {
		return fmt.Errorf("property node not found: %w", err)
	}

	// Check ownership
	if propertyNode.Ownership.CurrentOwner != fromAddress {
		return fmt.Errorf("sender is not the current owner")
	}

	// Transfer NFT ownership on-chain
	err = pm.nftContract.TransferNFT(propertyNode.NFTPointer.TokenID, fromAddress, toAddress)
	if err != nil {
		return fmt.Errorf("NFT transfer failed: %w", err)
	}

	// Update property node ownership
	err = propertyNode.TransferOwnership(toAddress, priceNRN)
	if err != nil {
		return fmt.Errorf("ownership transfer failed: %w", err)
	}

	// Update the property node in storage
	err = pm.nodeStore.StorePropertyNode(propertyNode)
	if err != nil {
		return fmt.Errorf("failed to update property node: %w", err)
	}

	// Add to NFT provenance chain
	propertyNode.NFTPointer.AddToProvenanceChain(toAddress)

	// Create transfer transaction
	transferData := map[string]interface{}{
		"property_node_id": propertyNodeID,
		"from_address":     fromAddress,
		"to_address":       toAddress,
		"price_nrn":        priceNRN,
		"transferred_at":   time.Now().Unix(),
	}

	data, err := json.Marshal(transferData)
	if err != nil {
		return fmt.Errorf("failed to marshal transfer data: %w", err)
	}

	tx, err := blockchain.NewMCPTransaction(
		fromAddress,
		toAddress,
		priceNRN, // Transfer the NRN payment
		data,
		blockchain.TransactionTypePropertyTransfer,
		50, // Fee for transfer
	)
	if err != nil {
		return fmt.Errorf("failed to create transfer transaction: %w", err)
	}

	// Submit transaction
	err = pm.blockchain.AddTransactionToTransactionPool(tx)
	if err != nil {
		return fmt.Errorf("failed to submit transfer transaction: %w", err)
	}

	log.Printf("Property transferred: %s from %s to %s for %d NRN", propertyNodeID, fromAddress, toAddress, priceNRN)
	return nil
}

// DistributeRoyalties distributes royalties for property usage
func (pm *PropertyMaker) DistributeRoyalties(propertyNodeID string, usageAmount uint64) error {
	// Get the property node
	propertyNode, err := pm.nodeStore.GetPropertyNode(propertyNodeID)
	if err != nil {
		return fmt.Errorf("property node not found: %w", err)
	}

	// Calculate royalty distribution
	distribution := propertyNode.Royalties.CalculateDistribution(usageAmount)

	// Create royalty distribution transaction
	royaltyData := map[string]interface{}{
		"property_node_id": propertyNodeID,
		"usage_amount":     usageAmount,
		"distribution":     distribution,
		"distributed_at":   time.Now().Unix(),
	}

	data, err := json.Marshal(royaltyData)
	if err != nil {
		return fmt.Errorf("failed to marshal royalty data: %w", err)
	}

	// Use the current owner as the transaction sender
	tx, err := blockchain.NewMCPTransaction(
		propertyNode.Ownership.CurrentOwner,
		"", // No specific recipient - royalties go to multiple addresses
		0,  // Royalties are distributed, not transferred as a single amount
		data,
		blockchain.TransactionTypeRoyaltyDistribution,
		25, // Fee for distribution
	)
	if err != nil {
		return fmt.Errorf("failed to create royalty distribution transaction: %w", err)
	}

	// Submit transaction
	err = pm.blockchain.AddTransactionToTransactionPool(tx)
	if err != nil {
		return fmt.Errorf("failed to submit royalty distribution transaction: %w", err)
	}

	log.Printf("Royalties distributed for property %s: %d total", propertyNodeID, usageAmount)
	return nil
}

// GetMakingProposalsForIdea gets all making proposals for a specific idea
func (pm *PropertyMaker) GetMakingProposalsForIdea(ideaNodeID string) ([]*MakingProposal, error) {
	// This would typically query a database of proposals
	// For now, return empty slice as proposals are handled via transactions
	return []*MakingProposal{}, nil
}

// GetPropertiesByMaker gets all minted properties by a specific maker
func (pm *PropertyMaker) GetPropertiesByMaker(makerAddress string) ([]*types.PropertyNode, error) {
	return pm.graphQueries.FindPropertyNodesByOwner(makerAddress, 0)
}

// GetPropertiesByType gets all properties of a specific IP type
func (pm *PropertyMaker) GetPropertiesByType(ipType types.IPType) ([]*types.PropertyNode, error) {
	// This would require querying property nodes by IP type
	// For now, return empty slice
	return []*types.PropertyNode{}, nil
}

// NFTContractClient handles NFT contract interactions
type NFTContractClient struct {
	contractAddress string
	rpcEndpoint     string
}

// NewNFTContractClient creates a new NFT contract client
func NewNFTContractClient(contractAddress, rpcEndpoint string) *NFTContractClient {
	return &NFTContractClient{
		contractAddress: contractAddress,
		rpcEndpoint:     rpcEndpoint,
	}
}

// MintNFT mints a new NFT
func (ncc *NFTContractClient) MintNFT(ownerAddress string, metadata map[string]interface{}) (string, string, error) {
	// This would interact with the actual NFT contract
	// For now, simulate minting

	tokenID := fmt.Sprintf("nft_%d", time.Now().UnixNano())
	contractAddress := ncc.contractAddress

	log.Printf("NFT minted: token %s on contract %s for owner %s", tokenID, contractAddress, ownerAddress)
	return tokenID, contractAddress, nil
}

// TransferNFT transfers an NFT to a new owner
func (ncc *NFTContractClient) TransferNFT(tokenID, fromAddress, toAddress string) error {
	// This would interact with the actual NFT contract
	// For now, simulate transfer

	log.Printf("NFT transferred: token %s from %s to %s", tokenID, fromAddress, toAddress)
	return nil
}

// GetNFTOwner gets the current owner of an NFT
func (ncc *NFTContractClient) GetNFTOwner(tokenID string) (string, error) {
	// This would query the NFT contract
	// For now, return a mock owner
	return "mock_owner_address", nil
}