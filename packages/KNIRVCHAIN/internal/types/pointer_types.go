package types

import (
	"KNIRVCHAIN/internal/errors"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)
// LoRAAdapterPointer points to the fine-tuned model weights
type LoRAAdapterPointer struct {
	AdapterID       string   `json:"adapter_id"`
	IPFSCID         string   `json:"ipfs_cid"`         // content address of LoRA weights
	CMU             string   `json:"cmu"`              // knirv://network/adapter_hash
	BaseModelRef    string   `json:"base_model_ref"`   // which base LLM this adapts
	Rank            int      `json:"rank"`             // LoRA rank parameter
	Alpha           float64  `json:"alpha"`            // LoRA alpha scaling
	TargetModules   []string `json:"target_modules"`   // q_proj, v_proj, etc.
	CreatedAt       int64    `json:"created_at"`
}

// NewLoRAAdapterPointer creates a new LoRA adapter pointer with validation
func NewLoRAAdapterPointer(adapterID, ipfsCID, baseModelRef string, rank int, alpha float64, targetModules []string) (*LoRAAdapterPointer, error) {
	if adapterID == "" {
		return nil, errors.NewValidationError("adapter_id cannot be empty")
	}
	if ipfsCID == "" {
		return nil, errors.NewValidationError("ipfs_cid cannot be empty")
	}
	if baseModelRef == "" {
		return nil, errors.NewValidationError("base_model_ref cannot be empty")
	}
	if rank <= 0 {
		return nil, errors.NewValidationError("rank must be positive")
	}
	if alpha <= 0.0 {
		return nil, errors.NewValidationError("alpha must be positive")
	}
	if len(targetModules) == 0 {
		return nil, errors.NewValidationError("target_modules cannot be empty")
	}

	// Generate CMU (Content Management URI)
	cmuData := fmt.Sprintf("%s:%s:%s:%d:%.2f", adapterID, ipfsCID, baseModelRef, rank, alpha)
	hash := sha256.Sum256([]byte(cmuData))
	cmu := fmt.Sprintf("knirv://network/adapter_%s", hex.EncodeToString(hash[:]))

	pointer := &LoRAAdapterPointer{
		AdapterID:     adapterID,
		IPFSCID:       ipfsCID,
		CMU:           cmu,
		BaseModelRef:  baseModelRef,
		Rank:          rank,
		Alpha:         alpha,
		TargetModules: targetModules,
		CreatedAt:     time.Now().Unix(),
	}

	return pointer, nil
}

// Validate checks if the LoRA adapter pointer is valid
func (lap *LoRAAdapterPointer) Validate() error {
	if lap.AdapterID == "" {
		return errors.NewValidationError("adapter_id cannot be empty")
	}
	if lap.IPFSCID == "" {
		return errors.NewValidationError("ipfs_cid cannot be empty")
	}
	if lap.CMU == "" {
		return errors.NewValidationError("cmu cannot be empty")
	}
	if lap.BaseModelRef == "" {
		return errors.NewValidationError("base_model_ref cannot be empty")
	}
	if lap.Rank <= 0 {
		return errors.NewValidationError("rank must be positive")
	}
	if lap.Alpha <= 0.0 {
		return errors.NewValidationError("alpha must be positive")
	}
	if len(lap.TargetModules) == 0 {
		return errors.NewValidationError("target_modules cannot be empty")
	}
	return nil
}

// MCPServerPointer references the MCP server implementation
type MCPServerPointer struct {
	ServerID        string            `json:"server_id"`
	EndpointURI     string            `json:"endpoint_uri"`     // wss://server/mcp
	CMU             string            `json:"cmu"`              // knirv://network/mcp_hash
	ProtocolVersion string            `json:"protocol_version"` // MCP protocol version
	Capabilities    []string          `json:"capabilities"`     // tools, resources, prompts
	AuthMethod      string            `json:"auth_method"`      // none, api_key, oauth, udc
	MetadataCID     string            `json:"metadata_cid"`     // IPFS pointer to full spec
	CreatedAt       int64             `json:"created_at"`
}

// NewMCPServerPointer creates a new MCP server pointer with validation
func NewMCPServerPointer(serverID, endpointURI, protocolVersion string, capabilities []string, authMethod, metadataCID string) (*MCPServerPointer, error) {
	if serverID == "" {
		return nil, errors.NewValidationError("server_id cannot be empty")
	}
	if endpointURI == "" {
		return nil, errors.NewValidationError("endpoint_uri cannot be empty")
	}
	if protocolVersion == "" {
		return nil, errors.NewValidationError("protocol_version cannot be empty")
	}
	if len(capabilities) == 0 {
		return nil, errors.NewValidationError("capabilities cannot be empty")
	}
	if authMethod == "" {
		return nil, errors.NewValidationError("auth_method cannot be empty")
	}
	if metadataCID == "" {
		return nil, errors.NewValidationError("metadata_cid cannot be empty")
	}

	// Generate CMU
	cmuData := fmt.Sprintf("%s:%s:%s:%s", serverID, endpointURI, protocolVersion, metadataCID)
	hash := sha256.Sum256([]byte(cmuData))
	cmu := fmt.Sprintf("knirv://network/mcp_%s", hex.EncodeToString(hash[:]))

	pointer := &MCPServerPointer{
		ServerID:        serverID,
		EndpointURI:     endpointURI,
		CMU:             cmu,
		ProtocolVersion: protocolVersion,
		Capabilities:    capabilities,
		AuthMethod:      authMethod,
		MetadataCID:     metadataCID,
		CreatedAt:       time.Now().Unix(),
	}

	return pointer, nil
}

// Validate checks if the MCP server pointer is valid
func (msp *MCPServerPointer) Validate() error {
	if msp.ServerID == "" {
		return errors.NewValidationError("server_id cannot be empty")
	}
	if msp.EndpointURI == "" {
		return errors.NewValidationError("endpoint_uri cannot be empty")
	}
	if msp.CMU == "" {
		return errors.NewValidationError("cmu cannot be empty")
	}
	if msp.ProtocolVersion == "" {
		return errors.NewValidationError("protocol_version cannot be empty")
	}
	if len(msp.Capabilities) == 0 {
		return errors.NewValidationError("capabilities cannot be empty")
	}
	if msp.AuthMethod == "" {
		return errors.NewValidationError("auth_method cannot be empty")
	}
	if msp.MetadataCID == "" {
		return errors.NewValidationError("metadata_cid cannot be empty")
	}
	return nil
}

// InferenceNFTPointer references the on-chain NFT
type InferenceNFTPointer struct {
	TokenID         string            `json:"token_id"`
	ContractAddress string            `json:"contract_address"`
	CMU             string            `json:"cmu"`              // knirv://network/nft_hash
	MetadataURI     string            `json:"metadata_uri"`     // IPFS/Arweave pointer
	ProvenanceChain []string          `json:"provenance_chain"` // history of ownership
	LicenseTerms    string            `json:"license_terms"`    // usage rights
	CreatedAt       int64             `json:"created_at"`
}

// NewInferenceNFTPointer creates a new inference NFT pointer with validation
func NewInferenceNFTPointer(tokenID, contractAddress, metadataURI, licenseTerms string) (*InferenceNFTPointer, error) {
	if tokenID == "" {
		return nil, errors.NewValidationError("token_id cannot be empty")
	}
	if contractAddress == "" {
		return nil, errors.NewValidationError("contract_address cannot be empty")
	}
	if metadataURI == "" {
		return nil, errors.NewValidationError("metadata_uri cannot be empty")
	}
	if licenseTerms == "" {
		return nil, errors.NewValidationError("license_terms cannot be empty")
	}

	// Generate CMU
	cmuData := fmt.Sprintf("%s:%s:%s", tokenID, contractAddress, metadataURI)
	hash := sha256.Sum256([]byte(cmuData))
	cmu := fmt.Sprintf("knirv://network/nft_%s", hex.EncodeToString(hash[:]))

	pointer := &InferenceNFTPointer{
		TokenID:         tokenID,
		ContractAddress: contractAddress,
		CMU:             cmu,
		MetadataURI:     metadataURI,
		ProvenanceChain: []string{}, // Start empty, will be populated with ownership history
		LicenseTerms:    licenseTerms,
		CreatedAt:       time.Now().Unix(),
	}

	return pointer, nil
}

// Validate checks if the inference NFT pointer is valid
func (infp *InferenceNFTPointer) Validate() error {
	if infp.TokenID == "" {
		return errors.NewValidationError("token_id cannot be empty")
	}
	if infp.ContractAddress == "" {
		return errors.NewValidationError("contract_address cannot be empty")
	}
	if infp.CMU == "" {
		return errors.NewValidationError("cmu cannot be empty")
	}
	if infp.MetadataURI == "" {
		return errors.NewValidationError("metadata_uri cannot be empty")
	}
	if infp.LicenseTerms == "" {
		return errors.NewValidationError("license_terms cannot be empty")
	}
	return nil
}

// AddToProvenanceChain adds an address to the provenance chain
func (infp *InferenceNFTPointer) AddToProvenanceChain(address string) error {
	if address == "" {
		return errors.NewValidationError("address cannot be empty")
	}
	infp.ProvenanceChain = append(infp.ProvenanceChain, address)
	return nil
}

// RoyaltyStructure defines payment distribution
type RoyaltyStructure struct {
	OriginNIMShare    uint8             `json:"origin_nim_share"`    // percentage to creator
	NetworkShare      uint8             `json:"network_share"`       // percentage to network
	DependencyShares  map[string]uint8 `json:"dependency_shares"` // to referenced ideas
}

// NewRoyaltyStructure creates a new royalty structure with validation
func NewRoyaltyStructure(originNIMShare, networkShare uint8, dependencyShares map[string]uint8) (*RoyaltyStructure, error) {
	total := originNIMShare + networkShare
	for _, share := range dependencyShares {
		total += share
	}

	if total > 100 {
		return nil, errors.NewValidationError("total royalty shares cannot exceed 100%")
	}

	rs := &RoyaltyStructure{
		OriginNIMShare:   originNIMShare,
		NetworkShare:     networkShare,
		DependencyShares: dependencyShares,
	}

	return rs, nil
}

// Validate checks if the royalty structure is valid
func (rs *RoyaltyStructure) Validate() error {
	total := rs.OriginNIMShare + rs.NetworkShare
	for _, share := range rs.DependencyShares {
		total += share
	}

	if total > 100 {
		return errors.NewValidationError("total royalty shares cannot exceed 100%")
	}

	return nil
}

// CalculateDistribution calculates royalty distribution for a given amount
func (rs *RoyaltyStructure) CalculateDistribution(totalAmount uint64) map[string]uint64 {
	distribution := make(map[string]uint64)

	// Calculate shares
	originNIMAmount := (totalAmount * uint64(rs.OriginNIMShare)) / 100
	networkAmount := (totalAmount * uint64(rs.NetworkShare)) / 100

	distribution["origin_nim"] = originNIMAmount
	distribution["network"] = networkAmount

	// Calculate dependency shares
	for dep, share := range rs.DependencyShares {
		amount := (totalAmount * uint64(share)) / 100
		distribution[dep] = amount
	}

	return distribution
}