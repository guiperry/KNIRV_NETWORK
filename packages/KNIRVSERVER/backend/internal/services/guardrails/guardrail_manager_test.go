package guardrails

import (
	"testing"
	"time"
)

func TestNewDynamicGuardrailManager(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)
	if gm == nil {
		t.Fatal("Expected non-nil DynamicGuardrailManager")
	}

	if gm.configurations == nil {
		t.Error("Expected configurations map to be initialized")
	}
	if gm.ontologyRules == nil {
		t.Error("Expected ontologyRules map to be initialized")
	}
	if gm.violations == nil {
		t.Error("Expected violations map to be initialized")
	}
}

func TestConfigureGuardrail(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	config := &GuardrailConfig{
		Type:     GuardrailTypeMemory,
		Enabled:  true,
		MaxValue: 1024,
		Action:   "block",
	}

	err := gm.ConfigureGuardrail("test-node", config)
	if err != nil {
		t.Errorf("ConfigureGuardrail failed: %v", err)
	}

	retrieved, ok := gm.GetGuardrailConfig("test-node", GuardrailTypeMemory)
	if !ok {
		t.Error("Expected to retrieve guardrail config")
	}
	if retrieved.MaxValue != 1024 {
		t.Errorf("Expected MaxValue 1024, got %v", retrieved.MaxValue)
	}
}

func TestValidateResourceUsage_NoViolation(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	config := &GuardrailConfig{
		Type:     GuardrailTypeMemory,
		Enabled:  true,
		MaxValue: 2048,
	}
	gm.ConfigureGuardrail("test-node", config)

	metrics := map[string]interface{}{
		"memory_usage": 1024.0,
	}

	violation, isViolated := gm.ValidateResourceUsage("test-node", metrics)
	if isViolated {
		t.Errorf("Expected no violation, got: %v", violation)
	}
}

func TestValidateResourceUsage_Violation(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	config := &GuardrailConfig{
		Type:     GuardrailTypeMemory,
		Enabled:  true,
		MaxValue: 512,
		Cooldown: 1 * time.Second,
	}
	gm.ConfigureGuardrail("test-node", config)

	metrics := map[string]interface{}{
		"memory_usage": 1024.0,
	}

	violation, isViolated := gm.ValidateResourceUsage("test-node", metrics)
	if !isViolated {
		t.Error("Expected violation")
	}
	if violation == nil {
		t.Fatal("Expected violation object")
	}
	if violation.GuardrailType != GuardrailTypeMemory {
		t.Errorf("Expected GuardrailTypeMemory, got %v", violation.GuardrailType)
	}
}

func TestValidateOntologyConstraint_Allowed(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	rule := &OntologyGuardrail{
		Domain:   "finance",
		Concepts: []string{"stock", "bond", "option"},
	}
	gm.ConfigureOntologyGuardrail("finance", rule)

	allowed, reason := gm.ValidateOntologyConstraint("test-node", "finance", "stock")
	if !allowed {
		t.Errorf("Expected concept to be allowed, reason: %s", reason)
	}
}

func TestValidateOntologyConstraint_NotAllowed(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	rule := &OntologyGuardrail{
		Domain:   "finance",
		Concepts: []string{"stock", "bond"},
	}
	gm.ConfigureOntologyGuardrail("finance", rule)

	allowed, reason := gm.ValidateOntologyConstraint("test-node", "finance", "crypto")
	if allowed {
		t.Error("Expected concept to not be allowed")
	}
	if reason == "" {
		t.Error("Expected non-empty reason")
	}
}

func TestValidateExecutionTime_NoViolation(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	config := &GuardrailConfig{
		Type:     GuardrailTypeExecution,
		Enabled:  true,
		MaxValue: 60,
	}
	gm.ConfigureGuardrail("test-node", config)

	allowed, violation := gm.ValidateExecutionTime("test-node", 30*time.Second)
	if !allowed {
		t.Errorf("Expected no violation, got: %v", violation)
	}
}

func TestValidateExecutionTime_Violation(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	config := &GuardrailConfig{
		Type:     GuardrailTypeExecution,
		Enabled:  true,
		MaxValue: 30,
	}
	gm.ConfigureGuardrail("test-node", config)

	allowed, violation := gm.ValidateExecutionTime("test-node", 60*time.Second)
	if allowed {
		t.Error("Expected violation")
	}
	if violation == nil {
		t.Fatal("Expected violation object")
	}
}

func TestValidateIntentObjective(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	config := &GuardrailConfig{
		Type:     GuardrailTypeOntology,
		Enabled:  true,
		MinValue: 0.5,
	}
	gm.ConfigureGuardrail("global", config)

	allowed, _ := gm.ValidateIntentObjective("test-node", "obj-1", map[string]interface{}{
		"intent_score": 0.8,
	})
	if !allowed {
		t.Error("Expected no violation for high intent score")
	}

	allowed, _ = gm.ValidateIntentObjective("test-node", "obj-1", map[string]interface{}{
		"intent_score": 0.3,
	})
	if allowed {
		t.Error("Expected violation for low intent score")
	}
}

func TestRecordAndGetViolations(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	violation := &GuardrailViolation{
		ID:            "v1",
		NodeID:        "test-node",
		GuardrailType: GuardrailTypeMemory,
		Message:       "test violation",
		Severity:      "high",
		Timestamp:     time.Now(),
	}

	err := gm.RecordViolation(violation)
	if err != nil {
		t.Errorf("RecordViolation failed: %v", err)
	}

	violations := gm.GetViolations("test-node", 0)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(violations))
	}

	violations = gm.GetViolations("other-node", 0)
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for other-node, got %d", len(violations))
	}
}

func TestResolveViolation(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	violation := &GuardrailViolation{
		ID:            "v1",
		NodeID:        "test-node",
		GuardrailType: GuardrailTypeMemory,
		Message:       "test violation",
		Severity:      "high",
		Timestamp:     time.Now(),
	}
	gm.RecordViolation(violation)

	err := gm.ResolveViolation("v1")
	if err != nil {
		t.Errorf("ResolveViolation failed: %v", err)
	}

	violations := gm.GetViolations("test-node", 0)
	if len(violations) != 1 || !violations[0].Resolved {
		t.Error("Expected violation to be resolved")
	}
}

func TestGetAllConfigurations(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	gm.ConfigureGuardrail("node1", &GuardrailConfig{Type: GuardrailTypeMemory, Enabled: true})
	gm.ConfigureGuardrail("node2", &GuardrailConfig{Type: GuardrailTypeCPU, Enabled: true})
	gm.ConfigureGuardrail("global", &GuardrailConfig{Type: GuardrailTypeNetwork, Enabled: true})

	configs := gm.GetAllConfigurations("")
	if len(configs) != 3 {
		t.Errorf("Expected 3 configs, got %d", len(configs))
	}

	configs = gm.GetAllConfigurations("node1")
	if len(configs) != 2 {
		t.Errorf("Expected 2 configs for node1 (1 node-specific + 1 global), got %d", len(configs))
	}
}

func TestGetStatistics(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	gm.ConfigureGuardrail("node1", &GuardrailConfig{Type: GuardrailTypeMemory, Enabled: true})
	gm.ConfigureGuardrail("node2", &GuardrailConfig{Type: GuardrailTypeCPU, Enabled: true})

	gm.RecordViolation(&GuardrailViolation{
		ID:        "v1",
		NodeID:    "node1",
		Severity:  "high",
		Resolved:  false,
		Timestamp: time.Now(),
	})
	gm.RecordViolation(&GuardrailViolation{
		ID:        "v2",
		NodeID:    "node1",
		Severity:  "low",
		Resolved:  true,
		Timestamp: time.Now(),
	})

	stats := gm.GetStatistics()
	if stats["total_configurations"].(int) != 2 {
		t.Errorf("Expected 2 configurations, got %v", stats["total_configurations"])
	}
	if stats["total_violations"].(int) != 2 {
		t.Errorf("Expected 2 violations, got %v", stats["total_violations"])
	}
	if stats["unresolved_violations"].(int) != 1 {
		t.Errorf("Expected 1 unresolved violation, got %v", stats["unresolved_violations"])
	}
}

func TestSeverityOrder(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	tests := []struct {
		severity string
		expected int
	}{
		{"critical", 4},
		{"high", 3},
		{"medium", 2},
		{"low", 1},
		{"unknown", 0},
	}

	for _, tc := range tests {
		order := gm.severityOrder(tc.severity)
		if order != tc.expected {
			t.Errorf("Expected severity order %d for %s, got %d", tc.expected, tc.severity, order)
		}
	}
}

func TestDetermineSeverity(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	tests := []struct {
		value    float64
		max      float64
		expected string
	}{
		{1.6, 1.0, "critical"},
		{1.3, 1.0, "high"},
		{1.1, 1.0, "medium"},
		{0.8, 1.0, "low"},
	}

	for _, tc := range tests {
		severity := gm.determineSeverity(GuardrailTypeMemory, tc.value, tc.max)
		if severity != tc.expected {
			t.Errorf("Expected severity %s for value %.2f/%.2f, got %s", tc.expected, tc.value, tc.max, severity)
		}
	}
}

func TestIsInCooldown(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	key := "test:memory"
	if gm.isInCooldown(key) {
		t.Error("Expected false for non-existent cooldown")
	}

	gm.setCooldown(key, 100*time.Millisecond)
	if !gm.isInCooldown(key) {
		t.Error("Expected true during cooldown period")
	}

	time.Sleep(150 * time.Millisecond)
	if gm.isInCooldown(key) {
		t.Error("Expected false after cooldown period")
	}
}

func TestIsNodeScope(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)

	gm.ConfigureGuardrail("node1", &GuardrailConfig{Type: GuardrailTypeMemory, Enabled: true})
	gm.ConfigureGuardrail("global", &GuardrailConfig{Type: GuardrailTypeMemory, Enabled: true})

	if !gm.isNodeScope("node1:memory", "node1") {
		t.Error("Expected node1:memory to match node1")
	}
	if !gm.isNodeScope("global:memory", "node2") {
		t.Error("Expected global:memory to match any node")
	}
	if gm.isNodeScope("node1:memory", "node2") {
		t.Error("Expected node1:memory to not match node2")
	}
}
