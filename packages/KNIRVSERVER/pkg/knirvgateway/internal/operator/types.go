package operator

import (
	"time"
)

// OperatorSignature represents the signature information for an operator
type OperatorSignature struct {
	Type               string `json:"type"`                         // e.g., "RsaSignature2018", "EdDsaSAPublicKeySecp256k1"
	Created            string `json:"created"`                      // ISO date string for signature creation
	VerificationMethod string `json:"verificationMethod"`           // DID URL of the public key
	ProofPurpose       string `json:"proofPurpose"`                 // e.g., "assertionMethod", "capabilityDelegation"
	ProofValue         string `json:"proofValue"`                   // The signature value
	SimulatedIssuer    string `json:"simulatedIssuer,omitempty"`    // e.g., "NANDA+ANS Trusted CA G1"
	SimulatedPublicKey string `json:"simulatedPublicKey,omitempty"` // Hex representation of a public key
}

// EndpointSet represents the endpoint configuration for an operator
type EndpointSet struct {
	Type              string   `json:"@type"`                         // e.g., "nanda:EndpointSet"
	StaticEndpoint    []string `json:"static_endpoint,omitempty"`     // Static endpoints
	AdaptiveRouterURL string   `json:"adaptive_router_url,omitempty"` // Adaptive router URL
}

// ProtocolExtension represents protocol-specific extensions
type ProtocolExtension struct {
	Type    string                 `json:"@type"`             // e.g., "A2AAgentCard"
	Name    string                 `json:"name,omitempty"`    // Generic name for display
	Details map[string]interface{} `json:"details,omitempty"` // Protocol-specific fields
}

// ProtocolExtensionSet represents a set of protocol extensions
type ProtocolExtensionSet struct {
	Type string             `json:"@type"`         // e.g., "ans:ProtocolExtensionSet"
	A2A  *ProtocolExtension `json:"a2a,omitempty"` // A2A protocol extension
	MCP  *ProtocolExtension `json:"mcp,omitempty"` // MCP protocol extension
	ACP  *ProtocolExtension `json:"acp,omitempty"` // ACP protocol extension
}

// OperatorFacts represents the detailed information about an operator
type OperatorFacts struct {
	ID                 string                `json:"id"`                     // Operator's own DID, primary key
	AnsName            string                `json:"ansName,omitempty"`      // Full ANS name
	Name               string                `json:"name"`                   // User-friendly display name
	Capability         string                `json:"capability"`             // Primary capability for display
	Capabilities       []string              `json:"capabilities,omitempty"` // Full list of capabilities
	Provider           string                `json:"provider,omitempty"`
	Version            string                `json:"version,omitempty"`
	Extension          string                `json:"extension,omitempty"` // e.g., "certified"
	Description        string                `json:"description,omitempty"`
	Endpoints          *EndpointSet          `json:"endpoints,omitempty"`
	Attestations       []string              `json:"attestations,omitempty"` // Array of attestation DIDs
	ProtocolExtensions *ProtocolExtensionSet `json:"protocolExtensions,omitempty"`
	Signature          *OperatorSignature    `json:"signature,omitempty"`  // Signature by operator or publisher
	AvatarURL          string                `json:"avatarUrl,omitempty"`  // URL for operator's avatar
	DataAIHint         string                `json:"dataAiHint,omitempty"` // Keywords for AI placeholder generation
}

// OperatorAddr represents the pointer to OperatorFacts stored in the registry
type OperatorAddr struct {
	OperatorID        string            `json:"operator_id"` // DID, should match OperatorFacts.id
	FactsURL          string            `json:"facts_url"`   // NANDA pointer to the OperatorFacts URL
	PrivateFactsURL   string            `json:"private_facts_url,omitempty"`
	AdaptiveRouterURL string            `json:"adaptive_router_url,omitempty"`
	TTL               int               `json:"ttl"`       // Time-to-live for this record
	Signature         OperatorSignature `json:"signature"` // Signature from the registry shard
}

// Operator represents the combined operator information used for UI display
type Operator struct {
	OperatorFacts
	AddrTTL      int    `json:"addr_ttl,omitempty"`       // TTL from NANDA OperatorAddr
	AddrFactsURL string `json:"addr_facts_url,omitempty"` // facts_url from NANDA OperatorAddr
	AI_summary   string `json:"aiSummary,omitempty"`      // To be populated by GenAI

	// KNIRV-ORACLE integration fields
	TransactionHash string `json:"transactionHash,omitempty"` // Transaction hash from KNIRV-ORACLE
	CapabilityID    string `json:"capabilityId,omitempty"`    // Capability ID from KNIRV-ORACLE
	OracleStatus    string `json:"oracleStatus,omitempty"`    // Registration status

	// Internal fields
	RegisteredAt time.Time `json:"registeredAt,omitempty"` // Registration timestamp
}

// KnirvOracleRegistrationResponse represents the response from KNIRV-ORACLE registration
type KnirvOracleRegistrationResponse struct {
	TransactionHash string `json:"transactionHash"`
	CapabilityID    string `json:"capabilityId"`
	Status          string `json:"status"`
	Message         string `json:"message,omitempty"`
}

// RegisterOperatorRequest represents a request to register a new operator
type RegisterOperatorRequest struct {
	Name               string                `json:"name,omitempty"`
	AnsName            string                `json:"ansName,omitempty"`
	Capability         string                `json:"capability,omitempty"`
	Capabilities       []string              `json:"capabilities,omitempty"`
	Provider           string                `json:"provider,omitempty"`
	Version            string                `json:"version,omitempty"`
	Extension          string                `json:"extension,omitempty"`
	Description        string                `json:"description,omitempty"`
	Endpoints          *EndpointSet          `json:"endpoints,omitempty"`
	Attestations       []string              `json:"attestations,omitempty"`
	ProtocolExtensions *ProtocolExtensionSet `json:"protocolExtensions,omitempty"`
	AvatarURL          string                `json:"avatarUrl,omitempty"`
	DataAIHint         string                `json:"dataAiHint,omitempty"`
	AddrTTL            int                   `json:"addr_ttl,omitempty"`
	AddrFactsURL       string                `json:"addr_facts_url,omitempty"`
}

// RegisterOperatorResponse represents the response from registering a new operator
type RegisterOperatorResponse struct {
	Operator        Operator `json:"operator"`
	TransactionHash string   `json:"transactionHash"`
	CapabilityID    string   `json:"capabilityId"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string                 `json:"status"`
	Service     string                 `json:"service"`
	Version     string                 `json:"version"`
	Timestamp   string                 `json:"timestamp"`
	Uptime      float64                `json:"uptime"`
	Components  map[string]string      `json:"components"`
	Metrics     map[string]interface{} `json:"metrics"`
	Endpoints   map[string]string      `json:"endpoints"`
	Integration map[string]interface{} `json:"integration"`
}
