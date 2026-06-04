package identitybridge

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TrustLevel string

const (
	TrustLevelNone       TrustLevel = "none"
	TrustLevelLow        TrustLevel = "low"
	TrustLevelMedium     TrustLevel = "medium"
	TrustLevelHigh       TrustLevel = "high"
	TrustLevelMaximum    TrustLevel = "maximum"
)

type IdentitySource string

const (
	IdentitySourceKNIRV   IdentitySource = "knirv"
	IdentitySourceAGT     IdentitySource = "agt"
	IdentitySourceDID     IdentitySource = "did"
	IdentitySourceOIDC    IdentitySource = "oidc"
	IdentitySourceX509    IdentitySource = "x509"
)

type TrustEnvelope struct {
	IdentityID    string            `json:"identity_id"`
	Source        IdentitySource    `json:"source"`
	NodeID        string            `json:"node_id"`
	AgentID       string            `json:"agent_id,omitempty"`
	TrustLevel    TrustLevel        `json:"trust_level"`
	TrustScore    float64           `json:"trust_score"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Capabilities  []string          `json:"capabilities,omitempty"`
	IssuedAt      time.Time         `json:"issued_at"`
	ExpiresAt     time.Time         `json:"expires_at"`
	Signature     string            `json:"signature,omitempty"`
	Federated     bool              `json:"federated"`
	FederationID  string            `json:"federation_id,omitempty"`
}

type IdentityAttributes struct {
	NodeID       string            `json:"node_id"`
	AgentID      string            `json:"agent_id,omitempty"`
	Roles        []string          `json:"roles,omitempty"`
	Permissions  []string          `json:"permissions,omitempty"`
	Environment  string            `json:"environment"`
	Labels       map[string]string `json:"labels,omitempty"`
	Stake        float64           `json:"stake,omitempty"`
	Reputation   float64           `json:"reputation,omitempty"`
	TEEType      string            `json:"tee_type,omitempty"`
	GeoRegion    string            `json:"geo_region,omitempty"`
}

type TrustMapping struct {
	InternalID      string    `json:"internal_id"`
	ExternalID      string    `json:"external_id"`
	ExternalSource  string    `json:"external_source"`
	MappingType     string    `json:"mapping_type"`
	MappedAt        time.Time `json:"mapped_at"`
	LastVerifiedAt  time.Time `json:"last_verified_at"`
	MappingMetadata map[string]string `json:"mapping_metadata,omitempty"`
}

type IdentityBridge struct {
	mu           sync.RWMutex
	envelopes    map[string]*TrustEnvelope
	mappings     map[string]*TrustMapping
	attributes   map[string]*IdentityAttributes
	defaultTTL   time.Duration
}

func NewIdentityBridge() *IdentityBridge {
	return &IdentityBridge{
		envelopes:  make(map[string]*TrustEnvelope),
		mappings:   make(map[string]*TrustMapping),
		attributes: make(map[string]*IdentityAttributes),
		defaultTTL: 24 * time.Hour,
	}
}

func (ib *IdentityBridge) SetDefaultTTL(ttl time.Duration) {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	ib.defaultTTL = ttl
}

func (ib *IdentityBridge) CreateEnvelope(nodeID, agentID string, source IdentitySource, attrs *IdentityAttributes) *TrustEnvelope {
	ib.mu.Lock()
	defer ib.mu.Unlock()

	if attrs == nil {
		attrs = &IdentityAttributes{
			NodeID:      nodeID,
			Environment: "unknown",
		}
	}

	trustScore := ib.calculateTrustScore(attrs)
	trustLevel := ib.scoreToLevel(trustScore)

	env := &TrustEnvelope{
		IdentityID:   uuid.New().String(),
		Source:       source,
		NodeID:       nodeID,
		AgentID:      agentID,
		TrustLevel:   trustLevel,
		TrustScore:   trustScore,
		Metadata:     make(map[string]string),
		Capabilities: make([]string, 0),
		IssuedAt:     time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(ib.defaultTTL),
		Federated:    false,
	}

	ib.envelopes[env.IdentityID] = env
	ib.attributes[env.IdentityID] = attrs
	return env
}

func (ib *IdentityBridge) GetEnvelope(identityID string) (*TrustEnvelope, bool) {
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	env, ok := ib.envelopes[identityID]
	if !ok {
		return nil, false
	}
	if time.Now().UTC().After(env.ExpiresAt) {
		return nil, false
	}
	return env, true
}

func (ib *IdentityBridge) RevokeEnvelope(identityID string) error {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	env, ok := ib.envelopes[identityID]
	if !ok {
		return fmt.Errorf("envelope not found: %s", identityID)
	}
	env.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
	return nil
}

func (ib *IdentityBridge) RefreshEnvelope(identityID string) (*TrustEnvelope, error) {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	env, ok := ib.envelopes[identityID]
	if !ok {
		return nil, fmt.Errorf("envelope not found: %s", identityID)
	}
	env.IssuedAt = time.Now().UTC()
	env.ExpiresAt = time.Now().UTC().Add(ib.defaultTTL)
	return env, nil
}

func (ib *IdentityBridge) UpdateTrustScore(identityID string, delta float64) (*TrustEnvelope, error) {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	env, ok := ib.envelopes[identityID]
	if !ok {
		return nil, fmt.Errorf("envelope not found: %s", identityID)
	}
	newScore := env.TrustScore + delta
	if newScore < 0 {
		newScore = 0
	}
	if newScore > 1.0 {
		newScore = 1.0
	}
	env.TrustScore = newScore
	env.TrustLevel = ib.scoreToLevel(newScore)
	env.IssuedAt = time.Now().UTC()
	return env, nil
}

func (ib *IdentityBridge) CreateMapping(internalID, externalID, externalSource, mappingType string, metadata map[string]string) *TrustMapping {
	ib.mu.Lock()
	defer ib.mu.Unlock()

	mapping := &TrustMapping{
		InternalID:      internalID,
		ExternalID:      externalID,
		ExternalSource:  externalSource,
		MappingType:     mappingType,
		MappedAt:        time.Now().UTC(),
		LastVerifiedAt:  time.Now().UTC(),
		MappingMetadata: metadata,
	}

	mappingKey := ib.mappingKey(internalID, externalSource)
	ib.mappings[mappingKey] = mapping
	return mapping
}

func (ib *IdentityBridge) GetMapping(internalID, externalSource string) (*TrustMapping, bool) {
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	mapping, ok := ib.mappings[ib.mappingKey(internalID, externalSource)]
	return mapping, ok
}

func (ib *IdentityBridge) ListMappings(internalID string) []*TrustMapping {
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	var result []*TrustMapping
	for _, m := range ib.mappings {
		if m.InternalID == internalID {
			result = append(result, m)
		}
	}
	return result
}

func (ib *IdentityBridge) ListEnvelopes(nodeID string) []*TrustEnvelope {
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	var result []*TrustEnvelope
	for _, env := range ib.envelopes {
		if nodeID == "" || env.NodeID == nodeID {
			if time.Now().UTC().Before(env.ExpiresAt) {
				result = append(result, env)
			}
		}
	}
	return result
}

func (ib *IdentityBridge) UpdateAttributes(identityID string, attrs *IdentityAttributes) error {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	if _, ok := ib.envelopes[identityID]; !ok {
		return fmt.Errorf("envelope not found: %s", identityID)
	}
	ib.attributes[identityID] = attrs

	env := ib.envelopes[identityID]
	newScore := ib.calculateTrustScore(attrs)
	env.TrustScore = newScore
	env.TrustLevel = ib.scoreToLevel(newScore)
	env.IssuedAt = time.Now().UTC()
	return nil
}

func (ib *IdentityBridge) GetAttributes(identityID string) (*IdentityAttributes, bool) {
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	attrs, ok := ib.attributes[identityID]
	return attrs, ok
}

func (ib *IdentityBridge) calculateTrustScore(attrs *IdentityAttributes) float64 {
	if attrs == nil {
		return 0.0
	}
	score := 0.5

	if attrs.Stake > 0 {
		score += 0.15
		if attrs.Stake > 1000 {
			score += 0.1
		}
	}

	if attrs.Reputation > 0 {
		score += attrs.Reputation * 0.2
	}

	if attrs.TEEType != "" {
		score += 0.1
	}

	if attrs.Environment == "production" {
		score += 0.05
	}

	if len(attrs.Roles) > 0 {
		score += 0.05
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}

func (ib *IdentityBridge) scoreToLevel(score float64) TrustLevel {
	switch {
	case score >= 0.9:
		return TrustLevelMaximum
	case score >= 0.7:
		return TrustLevelHigh
	case score >= 0.4:
		return TrustLevelMedium
	case score >= 0.1:
		return TrustLevelLow
	default:
		return TrustLevelNone
	}
}

func (ib *IdentityBridge) mappingKey(internalID, externalSource string) string {
	return internalID + ":" + externalSource
}

func (ib *IdentityBridge) FederateIdentity(identityID, federationID string) error {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	env, ok := ib.envelopes[identityID]
	if !ok {
		return fmt.Errorf("envelope not found: %s", identityID)
	}
	env.Federated = true
	env.FederationID = federationID
	return nil
}

func (ib *IdentityBridge) GetStatistics() map[string]interface{} {
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	activeCount := 0
	for _, env := range ib.envelopes {
		if time.Now().UTC().Before(env.ExpiresAt) {
			activeCount++
		}
	}
	return map[string]interface{}{
		"total_envelopes":   len(ib.envelopes),
		"active_envelopes":  activeCount,
		"total_mappings":    len(ib.mappings),
		"trust_distribution": ib.computeTrustDistribution(),
	}
}

func (ib *IdentityBridge) computeTrustDistribution() map[TrustLevel]int {
	dist := map[TrustLevel]int{}
	for _, env := range ib.envelopes {
		if time.Now().UTC().Before(env.ExpiresAt) {
			dist[env.TrustLevel]++
		}
	}
	return dist
}
