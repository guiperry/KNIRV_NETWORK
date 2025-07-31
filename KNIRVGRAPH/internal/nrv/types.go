package nrv

import (
	"time"
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

// ErrorNode represents an error in the system that needs resolution
type ErrorNode struct {
	ID          string                 `json:"id"`
	ErrorType   string                 `json:"error_type"`
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context"`
	Resolution  *ResolutionPath        `json:"resolution,omitempty"`
	Severity    int                    `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
}

// SkillNode represents a skill that can resolve errors
type SkillNode struct {
	ID           string                 `json:"id"`
	SkillType    string                 `json:"skill_type"`
	Capabilities []string               `json:"capabilities"`
	Requirements map[string]interface{} `json:"requirements"`
	Performance  *PerformanceMetrics    `json:"performance"`
	Validation   *ValidationStatus      `json:"validation"`
	Timestamp    time.Time              `json:"timestamp"`
}

// ResolutionPath represents a path to resolve an error
type ResolutionPath struct {
	Steps         []ResolutionStep `json:"steps"`
	Confidence    float64          `json:"confidence"`
	EstimatedCost float64          `json:"estimated_cost"`
}

// ResolutionStep represents a single step in error resolution
type ResolutionStep struct {
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	SkillID    string                 `json:"skill_id,omitempty"`
	Confidence float64                `json:"confidence"`
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
