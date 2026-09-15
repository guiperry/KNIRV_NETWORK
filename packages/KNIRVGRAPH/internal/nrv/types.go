package nrv

import (
	"time"

	"KNIRVGRAPH/internal/protocol/proto"

	"google.golang.org/protobuf/types/known/structpb"
)

// NetworkResolutionVector represents a vector in the NRV system
type NetworkResolutionVector struct {
	ID          string                 `json:"id"`
	SourcePeer  string                 `json:"source_peer"`
	TargetHash  string                 `json:"target_hash"`
	Coordinates []float64              `json:"coordinates"`
	Confidence  float64                `json:"confidence"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	Signatures  []VectorSignature      `json:"signatures"`
}

// VectorSignature represents a signature on a vector
type VectorSignature struct {
	PeerID    string    `json:"peer_id"`
	Signature []byte    `json:"signature"`
	Timestamp time.Time `json:"timestamp"`
}

// VectorUpdate represents an update to a vector
type VectorUpdate struct {
	Vector    *NetworkResolutionVector `json:"vector"`
	Operation string                   `json:"operation"` // "create", "update", "validate"
}

// ErrorNode is the canonical error node, generated from
// shared-proto/graph/v1/graph.proto into internal/protocol/proto.
//
// This package previously declared its own ErrorNode (and its own
// ResolutionPath/ResolutionStep), as did KNIRVCHAIN's types and three proto
// files — four divergent shapes for one concept. There is now exactly one
// definition, and these are aliases to it.
type ErrorNode = proto.ErrorNode

// SkillNode represents a skill that can resolve errors.
//
// Deprecated: KNIRVGRAPH's SkillNode is superseded by KNIRVCHAIN's own
// types.SkillNode (packages/KNIRVCHAIN/internal/types/node_types.go), which
// is the canonical on-chain skill-mining record (chain_refactor.md Open
// Decision #3 — "keep the ErrorNode on KNIRVGRAPH but deprecate the
// SkillNode"). ErrorNode remains KNIRVGRAPH's record; new skill
// registrations should go through KNIRVCHAIN's mining pipeline instead of
// creating a second, disconnected SkillNode here.
type SkillNode struct {
	ID           string                 `json:"id"`
	SkillType    string                 `json:"skill_type"`
	Capabilities []string               `json:"capabilities"`
	Requirements map[string]interface{} `json:"requirements"`
	Performance  *PerformanceMetrics    `json:"performance"`
	Validation   *ValidationStatus      `json:"validation"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ResolutionPath and ResolutionStep are the canonical definitions from the
// shared proto, aliased for the same reason as ErrorNode.
type ResolutionPath = proto.ResolutionPath

type ResolutionStep = proto.ResolutionStep

// NewErrorContextStruct converts a Go map into the protobuf Struct the shared
// contract carries. A nil or empty map yields a nil Struct, so an absent
// context stays absent rather than becoming an empty object.
func NewErrorContextStruct(values map[string]interface{}) (*structpb.Struct, error) {
	if len(values) == 0 {
		return nil, nil
	}
	return structpb.NewStruct(values)
}

// ErrorContextMap returns a node's structured context as a Go map, tolerating a
// nil node or an absent context.
func ErrorContextMap(node *ErrorNode) map[string]interface{} {
	if node == nil {
		return nil
	}
	fields := node.GetContext()
	if fields == nil {
		return nil
	}
	return fields.AsMap()
}

// NewResolutionStep builds a canonical resolution step, converting the
// parameters map into the protobuf Struct the contract uses.
func NewResolutionStep(action string, parameters map[string]interface{}, skillID string, confidence float64) (ResolutionStep, error) {
	params, err := NewErrorContextStruct(parameters)
	if err != nil {
		return ResolutionStep{}, err
	}
	return ResolutionStep{
		Action:     action,
		Parameters: params,
		SkillId:    skillID,
		Confidence: float32(confidence),
	}, nil
}

// PerformanceMetrics tracks skill performance
type PerformanceMetrics struct {
	SuccessRate      float64   `json:"success_rate"`
	AverageLatency   float64   `json:"average_latency"`
	TotalInvocations int64     `json:"total_invocations"`
	LastUpdated      time.Time `json:"last_updated"`
}

// ValidationStatus tracks validation state
type ValidationStatus struct {
	IsValidated     bool      `json:"is_validated"`
	ValidatedBy     []string  `json:"validated_by"`
	ValidationScore float64   `json:"validation_score"`
	LastValidated   time.Time `json:"last_validated"`
}

// ContextNode represents context data (MCP servers) that becomes capabilities
type ContextNode struct {
	ID            string                 `json:"id"`
	ContextType   string                 `json:"context_type"` // "mcp_server", "api_endpoint", "tool"
	Description   string                 `json:"description"`
	Schema        map[string]interface{} `json:"schema"`
	LocationHints []string               `json:"location_hints"`
	GasFeeNRN     uint64                 `json:"gas_fee_nrn,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Status        string                 `json:"status"` // "pending", "processing", "capability_created"
}

// IdeaNode represents ideas that become properties through collaboration
type IdeaNode struct {
	ID              string                 `json:"id"`
	IdeaType        string                 `json:"idea_type"` // "asset", "characteristic", "attribute"
	Description     string                 `json:"description"`
	FeasibilityData map[string]interface{} `json:"feasibility_data"`
	ExistenceCheck  *ExistenceReport       `json:"existence_check,omitempty"`
	Collaborators   []string               `json:"collaborators"` // Agent IDs working on this idea
	Stakes          map[string]float64     `json:"stakes"`        // Agent stakes in resulting property
	Timestamp       time.Time              `json:"timestamp"`
	Status          string                 `json:"status"` // "pending", "collaborative", "property_created"
}

// CapabilityNode represents capabilities created from context nodes
type CapabilityNode struct {
	ID             string                 `json:"id"`
	SourceContext  string                 `json:"source_context"` // ContextNode ID
	Name           string                 `json:"name"`
	CapabilityType string                 `json:"capability_type"`
	Schema         map[string]interface{} `json:"schema"`
	LocationHints  []string               `json:"location_hints"`
	GasFeeNRN      uint64                 `json:"gas_fee_nrn"`
	Performance    *PerformanceMetrics    `json:"performance"`
	Timestamp      time.Time              `json:"timestamp"`
}

// PropertyNode represents properties created from idea nodes
type PropertyNode struct {
	ID           string                 `json:"id"`
	SourceIdea   string                 `json:"source_idea"` // IdeaNode ID
	Name         string                 `json:"name"`
	PropertyType string                 `json:"property_type"`
	ValueType    string                 `json:"value_type"`
	Constraints  map[string]interface{} `json:"constraints"`
	Immutable    bool                   `json:"immutable"`
	Category     string                 `json:"category,omitempty"`
	Owners       map[string]float64     `json:"owners"` // Agent ownership stakes
	Timestamp    time.Time              `json:"timestamp"`
}

// ExistenceReport tracks whether an idea already exists
type ExistenceReport struct {
	Exists       bool                   `json:"exists"`
	ExistingRefs []string               `json:"existing_refs,omitempty"`
	Similarity   float64                `json:"similarity"`
	Analysis     map[string]interface{} `json:"analysis"`
	CheckedAt    time.Time              `json:"checked_at"`
}

// NRVConfig holds configuration for the NRV system
type NRVConfig struct {
	MaxVectors        int           `json:"max_vectors"`
	VectorTTL         time.Duration `json:"vector_ttl"`
	ConfidenceDecay   float64       `json:"confidence_decay"`
	ValidationTimeout time.Duration `json:"validation_timeout"`
	DHTBootstrapPeers []string      `json:"dht_bootstrap_peers"`
}

// DefaultNRVConfig returns default configuration
func DefaultNRVConfig() *NRVConfig {
	return &NRVConfig{
		MaxVectors:        10000,
		VectorTTL:         24 * time.Hour,
		ConfidenceDecay:   0.95,
		ValidationTimeout: 30 * time.Second,
		DHTBootstrapPeers: []string{},
	}
}
