package guardrails

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGovernanceBridge(t *testing.T) {
	gb := NewGovernanceBridge(nil, nil)
	assert.NotNil(t, gb)
}

func TestValidateNodeActionNoModules(t *testing.T) {
	gb := NewGovernanceBridge(nil, nil)
	allowed, reason := gb.ValidateNodeAction("node-1", "read", nil)
	assert.True(t, allowed)
	assert.Equal(t, "allowed", reason)
}

func TestValidateNodeActionGuardrailViolation(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)
	gm.configurations["node-1:"+string(GuardrailTypeCPU)] = &GuardrailConfig{
		Type:     GuardrailTypeCPU,
		Enabled:  true,
		MaxValue: 80,
		Action:   "block",
	}

	gb := NewGovernanceBridge(gm, nil)
	allowed, reason := gb.ValidateNodeAction("node-1", "compute", map[string]interface{}{
		"metrics": map[string]interface{}{
			"cpu_usage": float64(95),
		},
	})
	assert.False(t, allowed)
	assert.Contains(t, reason, "exceeds max")
}

func TestValidateNodeActionNoViolation(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)
	gm.configurations["node-1:"+string(GuardrailTypeCPU)] = &GuardrailConfig{
		Type:     GuardrailTypeCPU,
		Enabled:  true,
		MaxValue: 80,
		Action:   "block",
	}

	gb := NewGovernanceBridge(gm, nil)
	allowed, reason := gb.ValidateNodeAction("node-1", "compute", map[string]interface{}{
		"metrics": map[string]interface{}{
			"cpu_usage": float64(50),
		},
	})
	assert.True(t, allowed)
	assert.Equal(t, "allowed", reason)
}

func TestValidateNodeActionPolicyDenied(t *testing.T) {
	pe := NewPolicyEngine(nil, nil)
	pe.activePolicies["pol-1"] = &Policy{
		ID:      "pol-1",
		Name:    "deny-all",
		Enabled: true,
		Rules: []*PolicyRule{
			{
				ID:        "rule-1",
				Type:      "execution",
				Condition: map[string]interface{}{"action": "delete"},
				Action:    "deny",
			},
		},
	}

	gb := NewGovernanceBridge(nil, pe)
	allowed, reason := gb.ValidateNodeAction("node-1", "delete", map[string]interface{}{
		"action": "delete",
	})
	assert.False(t, allowed)
	assert.Contains(t, reason, "denied by policy")
}

func TestValidateNodeActionPolicyWarning(t *testing.T) {
	pe := NewPolicyEngine(nil, nil)
	pe.activePolicies["pol-1"] = &Policy{
		ID:      "pol-1",
		Name:    "warn-policy",
		Enabled: true,
		Rules: []*PolicyRule{
			{
				ID:     "rule-1",
				Type:   "execution",
				Action: "warn",
			},
		},
	}

	gb := NewGovernanceBridge(nil, pe)
	allowed, reason := gb.ValidateNodeAction("node-1", "anything", nil)
	assert.True(t, allowed)
	assert.Contains(t, reason, "warning from policy")
}

func TestValidateNodeActionPolicyAllows(t *testing.T) {
	pe := NewPolicyEngine(nil, nil)
	pe.activePolicies["pol-1"] = &Policy{
		ID:      "pol-1",
		Name:    "allow-all",
		Enabled: true,
		Rules: []*PolicyRule{
			{
				ID:     "rule-1",
				Type:   "execution",
				Action: "allow",
			},
		},
	}

	gb := NewGovernanceBridge(nil, pe)
	allowed, reason := gb.ValidateNodeAction("node-1", "read", nil)
	assert.True(t, allowed)
	assert.Equal(t, "allowed", reason)
}

func TestRecordViolation(t *testing.T) {
	gm := NewDynamicGuardrailManager(nil)
	gb := NewGovernanceBridge(gm, nil)

	err := gb.RecordViolation("node-1", "unauthorized_access", "high", map[string]interface{}{
		"message": "unauthorized access attempt by agent-1",
	})
	assert.NoError(t, err)

	violations := gm.GetViolations("node-1", 0)
	assert.Len(t, violations, 1)
	assert.Equal(t, "unauthorized_access", violations[0].ViolationType)
}

func TestRecordViolationNilManager(t *testing.T) {
	gb := NewGovernanceBridge(nil, nil)
	err := gb.RecordViolation("node-1", "violation", "low", map[string]interface{}{
		"message": "test",
	})
	assert.NoError(t, err)
}
