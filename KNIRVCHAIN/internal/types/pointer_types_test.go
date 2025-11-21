package types

import (
	"testing"
	"time"
)

func TestNewLoRAAdapterPointer(t *testing.T) {
	targetModules := []string{"q_proj", "v_proj", "k_proj"}

	pointer, err := NewLoRAAdapterPointer("adapter1", "ipfs123", "gpt-4", 8, 16.0, targetModules)
	if err != nil {
		t.Fatalf("Failed to create LoRA adapter pointer: %v", err)
	}

	if pointer.AdapterID != "adapter1" {
		t.Errorf("Expected adapter ID 'adapter1', got '%s'", pointer.AdapterID)
	}

	if pointer.Rank != 8 {
		t.Errorf("Expected rank 8, got %d", pointer.Rank)
	}

	if pointer.Alpha != 16.0 {
		t.Errorf("Expected alpha 16.0, got %.1f", pointer.Alpha)
	}

	if len(pointer.TargetModules) != 3 {
		t.Errorf("Expected 3 target modules, got %d", len(pointer.TargetModules))
	}

	if pointer.CMU == "" {
		t.Error("Expected non-empty CMU")
	}
}

func TestLoRAAdapterPointerValidate(t *testing.T) {
	pointer := &LoRAAdapterPointer{
		AdapterID:     "adapter1",
		IPFSCID:       "ipfs123",
		CMU:           "knirv://network/adapter_hash",
		BaseModelRef:  "gpt-4",
		Rank:          8,
		Alpha:         16.0,
		TargetModules: []string{"q_proj"},
		CreatedAt:     time.Now().Unix(),
	}

	err := pointer.Validate()
	if err != nil {
		t.Errorf("Expected valid LoRA pointer, got error: %v", err)
	}

	// Test invalid rank
	pointer.Rank = 0
	err = pointer.Validate()
	if err == nil {
		t.Error("Expected error for invalid rank")
	}
	pointer.Rank = 8 // Reset

	// Test invalid alpha
	pointer.Alpha = -1.0
	err = pointer.Validate()
	if err == nil {
		t.Error("Expected error for invalid alpha")
	}
	pointer.Alpha = 16.0 // Reset

	// Test empty target modules
	pointer.TargetModules = []string{}
	err = pointer.Validate()
	if err == nil {
		t.Error("Expected error for empty target modules")
	}
}

func TestNewMCPServerPointer(t *testing.T) {
	capabilities := []string{"tools", "resources"}

	pointer, err := NewMCPServerPointer("server1", "ws://server.com", "2024-11-05", capabilities, "none", "ipfs123")
	if err != nil {
		t.Fatalf("Failed to create MCP server pointer: %v", err)
	}

	if pointer.ServerID != "server1" {
		t.Errorf("Expected server ID 'server1', got '%s'", pointer.ServerID)
	}

	if pointer.ProtocolVersion != "2024-11-05" {
		t.Errorf("Expected protocol version '2024-11-05', got '%s'", pointer.ProtocolVersion)
	}

	if len(pointer.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(pointer.Capabilities))
	}

	if pointer.CMU == "" {
		t.Error("Expected non-empty CMU")
	}
}

func TestMCPServerPointerValidate(t *testing.T) {
	pointer := &MCPServerPointer{
		ServerID:        "server1",
		EndpointURI:     "ws://server.com",
		CMU:             "knirv://network/mcp_hash",
		ProtocolVersion: "2024-11-05",
		Capabilities:    []string{"tools"},
		AuthMethod:      "none",
		MetadataCID:     "ipfs123",
		CreatedAt:       time.Now().Unix(),
	}

	err := pointer.Validate()
	if err != nil {
		t.Errorf("Expected valid MCP pointer, got error: %v", err)
	}

	// Test empty server ID
	pointer.ServerID = ""
	err = pointer.Validate()
	if err == nil {
		t.Error("Expected error for empty server ID")
	}
	pointer.ServerID = "server1" // Reset

	// Test empty capabilities
	pointer.Capabilities = []string{}
	err = pointer.Validate()
	if err == nil {
		t.Error("Expected error for empty capabilities")
	}
}

func TestNewInferenceNFTPointer(t *testing.T) {
	pointer, err := NewInferenceNFTPointer("token1", "0x123456789", "ipfs://metadata.json", "MIT License")
	if err != nil {
		t.Fatalf("Failed to create inference NFT pointer: %v", err)
	}

	if pointer.TokenID != "token1" {
		t.Errorf("Expected token ID 'token1', got '%s'", pointer.TokenID)
	}

	if pointer.ContractAddress != "0x123456789" {
		t.Errorf("Expected contract address '0x123456789', got '%s'", pointer.ContractAddress)
	}

	if len(pointer.ProvenanceChain) != 0 {
		t.Errorf("Expected empty provenance chain, got %d entries", len(pointer.ProvenanceChain))
	}

	if pointer.CMU == "" {
		t.Error("Expected non-empty CMU")
	}
}

func TestInferenceNFTPointerValidate(t *testing.T) {
	pointer := &InferenceNFTPointer{
		TokenID:         "token1",
		ContractAddress: "0x123",
		CMU:             "knirv://network/nft_hash",
		MetadataURI:     "ipfs://metadata.json",
		ProvenanceChain: []string{},
		LicenseTerms:    "MIT",
		CreatedAt:       time.Now().Unix(),
	}

	err := pointer.Validate()
	if err != nil {
		t.Errorf("Expected valid NFT pointer, got error: %v", err)
	}

	// Test empty token ID
	pointer.TokenID = ""
	err = pointer.Validate()
	if err == nil {
		t.Error("Expected error for empty token ID")
	}
	pointer.TokenID = "token1" // Reset

	// Test empty license terms
	pointer.LicenseTerms = ""
	err = pointer.Validate()
	if err == nil {
		t.Error("Expected error for empty license terms")
	}
}

func TestInferenceNFTPointerAddToProvenanceChain(t *testing.T) {
	pointer := &InferenceNFTPointer{
		ProvenanceChain: []string{},
	}

	err := pointer.AddToProvenanceChain("owner1")
	if err != nil {
		t.Fatalf("Failed to add to provenance chain: %v", err)
	}

	if len(pointer.ProvenanceChain) != 1 {
		t.Errorf("Expected 1 entry in provenance chain, got %d", len(pointer.ProvenanceChain))
	}

	if pointer.ProvenanceChain[0] != "owner1" {
		t.Errorf("Expected 'owner1' in provenance chain, got '%s'", pointer.ProvenanceChain[0])
	}

	// Test empty address
	err = pointer.AddToProvenanceChain("")
	if err == nil {
		t.Error("Expected error for empty address")
	}
}

func TestNewRoyaltyStructure(t *testing.T) {
	dependencyShares := map[string]uint8{
		"dep1": 30,
		"dep2": 20,
	}

	rs, err := NewRoyaltyStructure(40, 10, dependencyShares)
	if err != nil {
		t.Fatalf("Failed to create royalty structure: %v", err)
	}

	if rs.OriginNIMShare != 40 {
		t.Errorf("Expected origin NIM share 40, got %d", rs.OriginNIMShare)
	}

	if rs.NetworkShare != 10 {
		t.Errorf("Expected network share 10, got %d", rs.NetworkShare)
	}

	if rs.DependencyShares["dep1"] != 30 {
		t.Errorf("Expected dep1 share 30, got %d", rs.DependencyShares["dep1"])
	}
}

func TestRoyaltyStructureValidate(t *testing.T) {
	rs := &RoyaltyStructure{
		OriginNIMShare: 40,
		NetworkShare:   10,
		DependencyShares: map[string]uint8{
			"dep1": 30,
			"dep2": 20,
		},
	}

	err := rs.Validate()
	if err != nil {
		t.Errorf("Expected valid royalty structure, got error: %v", err)
	}

	// Test total > 100
	rs.OriginNIMShare = 60
	err = rs.Validate()
	if err == nil {
		t.Error("Expected error for total shares > 100")
	}
	rs.OriginNIMShare = 40 // Reset
}

func TestRoyaltyStructureCalculateDistribution(t *testing.T) {
	rs := &RoyaltyStructure{
		OriginNIMShare: 50,
		NetworkShare:   20,
		DependencyShares: map[string]uint8{
			"dep1": 30,
		},
	}

	distribution := rs.CalculateDistribution(1000)

	expectedOrigin := uint64(500) // 50% of 1000
	expectedNetwork := uint64(200) // 20% of 1000
	expectedDep1 := uint64(300)   // 30% of 1000

	if distribution["origin_nim"] != expectedOrigin {
		t.Errorf("Expected origin_nim %d, got %d", expectedOrigin, distribution["origin_nim"])
	}

	if distribution["network"] != expectedNetwork {
		t.Errorf("Expected network %d, got %d", expectedNetwork, distribution["network"])
	}

	if distribution["dep1"] != expectedDep1 {
		t.Errorf("Expected dep1 %d, got %d", expectedDep1, distribution["dep1"])
	}
}