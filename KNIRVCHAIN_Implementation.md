# KNIRVCHAIN Implementation Plan

## Overview

KNIRVCHAIN is the Go-based skill and LLM registry chain. This document plans the full implementation of three node transformation flows that form the core intelligence mining mechanism.

## Node Transformation Flows

### 1. ErrorNode → SkillNode Mining (Result: LoRA Adapter Pointer)

**Purpose**: Transform AI errors into validated skills that solve them.

#### Data Structures

```go
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

// LoRAAdapterPointer points to the fine-tuned model weights
type LoRAAdapterPointer struct {
    AdapterID       string   `json:"adapter_id"`
    IPFSCID         string   `json:"ipfs_cid"`         // content address of LoRA weights
    CMU             string   `json:"cmu"`              // knirv://network/adapter_hash
    BaseModelRef    string   `json:"base_model_ref"`   // which base LLM this adapts
    Rank            int      `json:"rank"`             // LoRA rank parameter
    Alpha           float64  `json:"alpha"`            // LoRA alpha scaling
    TargetModules   []string `json:"target_modules"`   // q_proj, v_proj, etc.
}
```

#### Mining Process

1. **Error Detection**: ErrorNode created when NIM fails a task
2. **Mining Challenge**: Miners propose SkillNode with LoRA adapter that resolves the error
3. **Validation**: KNIRVNEXUS DVE runs the LoRA against test cases derived from ErrorNode
4. **Proof Generation**: DVE produces ValidationProof with accuracy metrics
5. **Consensus**: Network validates proof and confirms SkillNode
6. **Reward**: Miner receives NRN based on error severity and solution quality

#### Transaction Types

```go
TransactionTypeErrorNodeSubmit      // Submit new error to graph
TransactionTypeSkillMiningProposal  // Propose solution with LoRA
TransactionTypeSkillValidation      // Submit DVE validation proof
TransactionTypeSkillConfirmation    // Finalize skill in registry
```

---

### 2. ContextNode → CapabilityNode Minting (Result: MCP Server Pointer)

**Purpose**: Transform execution contexts into reusable MCP capabilities.

#### Data Structures

```go
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

// MCPServerPointer references the MCP server implementation
type MCPServerPointer struct {
    ServerID        string            `json:"server_id"`
    EndpointURI     string            `json:"endpoint_uri"`     // wss://server/mcp
    CMU             string            `json:"cmu"`              // knirv://network/mcp_hash
    ProtocolVersion string            `json:"protocol_version"` // MCP protocol version
    Capabilities    []string          `json:"capabilities"`     // tools, resources, prompts
    AuthMethod      string            `json:"auth_method"`      // none, api_key, oauth, udc
    MetadataCID     string            `json:"metadata_cid"`     // IPFS pointer to full spec
}
```

#### Minting Process

1. **Context Collection**: Track NIM interactions, identify repeating patterns
2. **Pattern Recognition**: ContextNode created when frequency threshold met
3. **Capability Proposal**: Developer proposes CapabilityNode wrapping the pattern as MCP server
4. **Standards Validation**: Verify MCP protocol compliance
5. **Minting**: CapabilityNode added to registry with NRN fee structure
6. **Discovery**: Available for other NIMs via MCP discovery protocol

#### Transaction Types

```go
TransactionTypeContextNodeCreate       // Record new context pattern
TransactionTypeCapabilityMintProposal  // Propose MCP capability
TransactionTypeCapabilityValidation    // MCP compliance check
TransactionTypeCapabilityMint          // Mint to registry
```

---

### 3. IdeaNode → PropertyNode Making (Result: Inference NFT Pointer)

**Purpose**: Transform novel ideas/inferences into tradeable intellectual property NFTs.

#### Data Structures

```go
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

// InferenceNFTPointer references the on-chain NFT
type InferenceNFTPointer struct {
    TokenID         string            `json:"token_id"`
    ContractAddress string            `json:"contract_address"`
    CMU             string            `json:"cmu"`              // knirv://network/nft_hash
    MetadataURI     string            `json:"metadata_uri"`     // IPFS/Arweave pointer
    ProvenanceChain []string          `json:"provenance_chain"` // history of ownership
    LicenseTerms    string            `json:"license_terms"`    // usage rights
}

// RoyaltyStructure defines payment distribution
type RoyaltyStructure struct {
    OriginNIMShare    uint8  `json:"origin_nim_share"`    // percentage to creator
    NetworkShare      uint8  `json:"network_share"`       // percentage to network
    DependencyShares  map[string]uint8 `json:"dependency_shares"` // to referenced ideas
}
```

#### Making Process

1. **Idea Generation**: NIM produces novel inference during task execution
2. **Novelty Assessment**: Network checks idea against existing PropertyNodes
3. **Provenance Tracking**: Link to all referenced/derived knowledge
4. **NFT Minting**: Create InferenceNFT on-chain with royalty structure
5. **PropertyNode Creation**: Register in graph with ownership record
6. **Marketplace**: NFT tradeable, royalties flow on derivative usage

#### Transaction Types

```go
TransactionTypeIdeaNodeSubmit       // Submit new idea
TransactionTypeNoveltyAssessment    // Run novelty check
TransactionTypePropertyMint         // Mint NFT and PropertyNode
TransactionTypePropertyTransfer     // Transfer ownership
TransactionTypeRoyaltyDistribution  // Distribute usage royalties
```

---

## File Structure

```
KNIRVCHAIN/internal/
├── types/
│   ├── node_types.go        # ErrorNode, SkillNode, ContextNode, etc.
│   ├── pointer_types.go     # LoRAAdapterPointer, MCPServerPointer, InferenceNFTPointer
│   └── mcp_types.go         # Existing MCP types
├── mining/
│   ├── skill_mining.go      # ErrorNode → SkillNode process
│   ├── capability_minting.go # ContextNode → CapabilityNode process
│   └── property_making.go   # IdeaNode → PropertyNode process
├── graph/
│   ├── node_store.go        # Node persistence
│   ├── relationships.go     # Node linking and traversal
│   └── queries.go           # Graph queries
├── validation/
│   ├── proof_verifier.go    # DVE proof verification
│   └── novelty_checker.go   # Idea novelty assessment
└── nft/
    ├── inference_nft.go     # NFT minting logic
    └── royalty_engine.go    # Royalty calculation and distribution
```

---

## Integration Points

### With KNIRVORACLE (Rust Governance)

- Model transition proposals for new base LLMs
- Governance votes on skill validation parameters
- Token economics for mining/minting rewards

### With KNIRVNEXUS (DVE)

- Validation proof generation for SkillNodes
- Secure execution environment for testing LoRA adapters
- Attestation for MCP server compliance

### With KNIRVGRAPH

- Knowledge graph storage and queries
- Node relationship management
- Cross-chain sync via IBC

### With KNIRVROUTER

- P2P distribution of LoRA adapters
- MCP server discovery and routing
- Proof-of-Connectivity for node propagation

---

## Implementation Phases

### Phase 1: Node Types & Storage
- Define all node types in Go
- Implement graph storage layer
- Create transaction types

### Phase 2: Mining Infrastructure
- ErrorNode → SkillNode mining logic
- Integration with KNIRVNEXUS for validation
- LoRA pointer generation and storage

### Phase 3: Minting Infrastructure
- ContextNode → CapabilityNode minting
- MCP protocol validation
- Server registration and discovery

### Phase 4: NFT Infrastructure
- IdeaNode → PropertyNode making
- NFT contract integration
- Royalty distribution engine

### Phase 5: Integration & Testing
- Cross-chain communication
- End-to-end flow testing
- Performance optimization
