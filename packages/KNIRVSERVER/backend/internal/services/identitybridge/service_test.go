package identitybridge

import (
	"testing"
	"time"
)

func TestNewIdentityBridge(t *testing.T) {
	ib := NewIdentityBridge()
	if ib == nil {
		t.Fatal("Expected non-nil IdentityBridge")
	}
	if ib.envelopes == nil {
		t.Error("Expected envelopes map to be initialized")
	}
	if ib.mappings == nil {
		t.Error("Expected mappings map to be initialized")
	}
	if ib.attributes == nil {
		t.Error("Expected attributes map to be initialized")
	}
}

func TestCreateEnvelope(t *testing.T) {
	ib := NewIdentityBridge()
	attrs := &IdentityAttributes{
		NodeID:      "node-1",
		AgentID:     "agent-1",
		Roles:       []string{"validator"},
		Stake:       5000,
		Reputation:  0.9,
		TEEType:     "sgx",
		Environment: "production",
	}
	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, attrs)
	if env == nil {
		t.Fatal("Expected non-nil TrustEnvelope")
	}
	if env.IdentityID == "" {
		t.Error("Expected non-empty IdentityID")
	}
	if env.NodeID != "node-1" {
		t.Errorf("Expected NodeID node-1, got %s", env.NodeID)
	}
	if env.Source != IdentitySourceKNIRV {
		t.Errorf("Expected source knirv, got %s", env.Source)
	}
	if env.TrustLevel != TrustLevelMaximum {
		t.Errorf("Expected TrustLevel maximum for high reputation+stake, got %s", env.TrustLevel)
	}
	if env.TrustScore < 0.7 {
		t.Errorf("Expected trust score >= 0.7, got %f", env.TrustScore)
	}
}

func TestCreateEnvelopeDefaultAttrs(t *testing.T) {
	ib := NewIdentityBridge()
	env := ib.CreateEnvelope("node-1", "", IdentitySourceAGT, nil)
	if env == nil {
		t.Fatal("Expected non-nil TrustEnvelope")
	}
	if env.TrustLevel != TrustLevelMedium {
		t.Errorf("Expected TrustLevel medium for default attrs, got %s", env.TrustLevel)
	}
}

func TestGetEnvelope(t *testing.T) {
	ib := NewIdentityBridge()
	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)

	retrieved, ok := ib.GetEnvelope(env.IdentityID)
	if !ok {
		t.Error("Expected to retrieve envelope by ID")
	}
	if retrieved.IdentityID != env.IdentityID {
		t.Errorf("Expected ID %s, got %s", env.IdentityID, retrieved.IdentityID)
	}
}

func TestGetEnvelopeExpired(t *testing.T) {
	ib := NewIdentityBridge()
	ib.SetDefaultTTL(0)

	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)

	time.Sleep(1 * time.Millisecond)
	_, ok := ib.GetEnvelope(env.IdentityID)
	if ok {
		t.Error("Expected envelope to be expired")
	}
}

func TestRevokeEnvelope(t *testing.T) {
	ib := NewIdentityBridge()
	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)

	err := ib.RevokeEnvelope(env.IdentityID)
	if err != nil {
		t.Errorf("RevokeEnvelope failed: %v", err)
	}

	_, ok := ib.GetEnvelope(env.IdentityID)
	if ok {
		t.Error("Expected revoked envelope to not be retrievable")
	}
}

func TestRevokeEnvelopeNotFound(t *testing.T) {
	ib := NewIdentityBridge()
	err := ib.RevokeEnvelope("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent envelope")
	}
}

func TestRefreshEnvelope(t *testing.T) {
	ib := NewIdentityBridge()
	ib.SetDefaultTTL(0)
	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)

	ib.SetDefaultTTL(24 * time.Hour)
	refreshed, err := ib.RefreshEnvelope(env.IdentityID)
	if err != nil {
		t.Errorf("RefreshEnvelope failed: %v", err)
	}
	if refreshed.IdentityID != env.IdentityID {
		t.Errorf("Expected ID %s, got %s", env.IdentityID, refreshed.IdentityID)
	}

	_, ok := ib.GetEnvelope(env.IdentityID)
	if !ok {
		t.Error("Expected refreshed envelope to be retrievable")
	}
}

func TestUpdateTrustScore(t *testing.T) {
	ib := NewIdentityBridge()
	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)
	originalScore := env.TrustScore

	_, err := ib.UpdateTrustScore(env.IdentityID, 0.2)
	if err != nil {
		t.Errorf("UpdateTrustScore failed: %v", err)
	}

	updated, ok := ib.GetEnvelope(env.IdentityID)
	if !ok {
		t.Fatal("Expected to retrieve updated envelope")
	}
	if updated.TrustScore <= originalScore {
		t.Errorf("Expected trust score to increase from %f", originalScore)
	}
}

func TestUpdateTrustScoreClamps(t *testing.T) {
	ib := NewIdentityBridge()
	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)

	_, err := ib.UpdateTrustScore(env.IdentityID, -10.0)
	if err != nil {
		t.Errorf("UpdateTrustScore failed: %v", err)
	}

	updated, _ := ib.GetEnvelope(env.IdentityID)
	if updated.TrustScore < 0 {
		t.Errorf("Expected trust score clamped to >= 0, got %f", updated.TrustScore)
	}

	ib.UpdateTrustScore(env.IdentityID, 10.0)
	updated, _ = ib.GetEnvelope(env.IdentityID)
	if updated.TrustScore > 1.0 {
		t.Errorf("Expected trust score clamped to <= 1.0, got %f", updated.TrustScore)
	}
}

func TestCreateMapping(t *testing.T) {
	ib := NewIdentityBridge()
	mapping := ib.CreateMapping("internal-1", "did:example:123", "did", "did-to-knirv", map[string]string{
		"source": "federation-1",
	})
	if mapping == nil {
		t.Fatal("Expected non-nil mapping")
	}
	if mapping.InternalID != "internal-1" {
		t.Errorf("Expected InternalID internal-1, got %s", mapping.InternalID)
	}
	if mapping.ExternalID != "did:example:123" {
		t.Errorf("Expected ExternalID did:example:123, got %s", mapping.ExternalID)
	}
}

func TestGetMapping(t *testing.T) {
	ib := NewIdentityBridge()
	ib.CreateMapping("internal-1", "did:example:123", "did", "did-to-knirv", nil)

	mapping, ok := ib.GetMapping("internal-1", "did")
	if !ok {
		t.Error("Expected to retrieve mapping")
	}
	if mapping.ExternalID != "did:example:123" {
		t.Errorf("Expected ExternalID did:example:123, got %s", mapping.ExternalID)
	}
}

func TestListMappings(t *testing.T) {
	ib := NewIdentityBridge()
	ib.CreateMapping("internal-1", "ext-1", "did", "type-1", nil)
	ib.CreateMapping("internal-1", "ext-2", "oidc", "type-2", nil)
	ib.CreateMapping("internal-2", "ext-3", "did", "type-1", nil)

	mappings := ib.ListMappings("internal-1")
	if len(mappings) != 2 {
		t.Errorf("Expected 2 mappings for internal-1, got %d", len(mappings))
	}

	mappings = ib.ListMappings("internal-2")
	if len(mappings) != 1 {
		t.Errorf("Expected 1 mapping for internal-2, got %d", len(mappings))
	}
}

func TestListEnvelopes(t *testing.T) {
	ib := NewIdentityBridge()
	ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)
	ib.CreateEnvelope("node-2", "agent-2", IdentitySourceKNIRV, nil)
	ib.CreateEnvelope("node-1", "agent-3", IdentitySourceAGT, nil)

	envs := ib.ListEnvelopes("node-1")
	if len(envs) != 2 {
		t.Errorf("Expected 2 envelopes for node-1, got %d", len(envs))
	}

	envs = ib.ListEnvelopes("")
	if len(envs) != 3 {
		t.Errorf("Expected 3 envelopes for empty filter, got %d", len(envs))
	}
}

func TestUpdateAttributes(t *testing.T) {
	ib := NewIdentityBridge()
	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)

	newAttrs := &IdentityAttributes{
		NodeID:      "node-1",
		AgentID:     "agent-1",
		TEEType:     "nitro",
		Stake:       10000,
		Reputation:  0.95,
		Environment: "production",
		Roles:       []string{"validator", "executor"},
	}
	err := ib.UpdateAttributes(env.IdentityID, newAttrs)
	if err != nil {
		t.Errorf("UpdateAttributes failed: %v", err)
	}

	attrs, ok := ib.GetAttributes(env.IdentityID)
	if !ok {
		t.Fatal("Expected to retrieve attributes")
	}
	if attrs.Stake != 10000 {
		t.Errorf("Expected Stake 10000, got %f", attrs.Stake)
	}

	updated, _ := ib.GetEnvelope(env.IdentityID)
	if updated.TrustLevel != TrustLevelMaximum {
		t.Errorf("Expected TrustLevel maximum after high-value update, got %s", updated.TrustLevel)
	}
}

func TestFederateIdentity(t *testing.T) {
	ib := NewIdentityBridge()
	env := ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, nil)

	err := ib.FederateIdentity(env.IdentityID, "fed-1")
	if err != nil {
		t.Errorf("FederateIdentity failed: %v", err)
	}

	updated, ok := ib.GetEnvelope(env.IdentityID)
	if !ok {
		t.Fatal("Expected to retrieve federated envelope")
	}
	if !updated.Federated {
		t.Error("Expected Federated to be true")
	}
	if updated.FederationID != "fed-1" {
		t.Errorf("Expected FederationID fed-1, got %s", updated.FederationID)
	}
}

func TestGetStatistics(t *testing.T) {
	ib := NewIdentityBridge()
	ib.CreateEnvelope("node-1", "agent-1", IdentitySourceKNIRV, &IdentityAttributes{Stake: 1000, Reputation: 0.8})
	ib.CreateEnvelope("node-2", "agent-2", IdentitySourceAGT, &IdentityAttributes{Stake: 100, Reputation: 0.3})
	ib.CreateEnvelope("node-1", "agent-3", IdentitySourceDID, &IdentityAttributes{Stake: 5000, Reputation: 0.95})

	stats := ib.GetStatistics()
	if stats["total_envelopes"].(int) != 3 {
		t.Errorf("Expected 3 total envelopes, got %d", stats["total_envelopes"])
	}
	if stats["active_envelopes"].(int) != 3 {
		t.Errorf("Expected 3 active envelopes, got %d", stats["active_envelopes"])
	}
	dist := stats["trust_distribution"].(map[TrustLevel]int)
	highCount := dist[TrustLevelHigh]
	maximumCount := dist[TrustLevelMaximum]
	if highCount < 1 {
		t.Errorf("Expected at least 1 high trust envelope, got %d", highCount)
	}
	if maximumCount < 1 {
		t.Errorf("Expected at least 1 maximum trust envelope, got %d", maximumCount)
	}
}

func TestScoreToLevel(t *testing.T) {
	ib := NewIdentityBridge()
	tests := []struct {
		score    float64
		expected TrustLevel
	}{
		{0.95, TrustLevelMaximum},
		{0.8, TrustLevelHigh},
		{0.55, TrustLevelMedium},
		{0.2, TrustLevelLow},
		{0.0, TrustLevelNone},
	}
	for _, tc := range tests {
		level := ib.scoreToLevel(tc.score)
		if level != tc.expected {
			t.Errorf("Expected %s for score %f, got %s", tc.expected, tc.score, level)
		}
	}
}

func TestCalculateTrustScore(t *testing.T) {
	ib := NewIdentityBridge()
	tests := []struct {
		name     string
		attrs    *IdentityAttributes
		minScore float64
	}{
		{"nil attrs", nil, 0},
		{"minimal", &IdentityAttributes{NodeID: "n1"}, 0.5},
		{"with stake", &IdentityAttributes{NodeID: "n1", Stake: 500}, 0.65},
		{"high stake", &IdentityAttributes{NodeID: "n1", Stake: 5000}, 0.75},
		{"reputation", &IdentityAttributes{NodeID: "n1", Reputation: 0.8}, 0.66},
		{"tee and production", &IdentityAttributes{NodeID: "n1", TEEType: "sgx", Environment: "production"}, 0.65},
		{"with roles", &IdentityAttributes{NodeID: "n1", Roles: []string{"admin"}}, 0.55},
		{"all max", &IdentityAttributes{NodeID: "n1", Stake: 10000, Reputation: 1.0, TEEType: "sgx", Environment: "production", Roles: []string{"admin", "validator"}}, 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := ib.calculateTrustScore(tc.attrs)
			if score < tc.minScore {
				t.Errorf("Expected score >= %f, got %f", tc.minScore, score)
			}
			if score > 1.0 {
				t.Errorf("Expected score <= 1.0, got %f", score)
			}
		})
	}
}
