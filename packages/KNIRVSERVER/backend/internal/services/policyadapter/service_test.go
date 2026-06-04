package policyadapter

import (
	"testing"
)

func TestNewPolicyAdapter(t *testing.T) {
	pa := NewPolicyAdapter()
	if pa == nil {
		t.Fatal("Expected non-nil PolicyAdapter")
	}
	if pa.inputs == nil {
		t.Error("Expected inputs map to be initialized")
	}
	if pa.enforcements == nil {
		t.Error("Expected enforcements map to be initialized")
	}
}

func TestGetContract(t *testing.T) {
	pa := NewPolicyAdapter()
	contract := pa.GetContract()
	if contract.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", contract.Version)
	}
	if len(contract.MetricNames) == 0 {
		t.Error("Expected non-empty MetricNames")
	}
	if len(contract.ActionTypes) == 0 {
		t.Error("Expected non-empty ActionTypes")
	}
	if contract.OutputFormat != "application/json" {
		t.Errorf("Expected application/json, got %s", contract.OutputFormat)
	}
}

func TestNormalizeInput(t *testing.T) {
	pa := NewPolicyAdapter()
	metrics := map[string]float64{
		"cpu_usage":    75.5,
		"memory_usage": 4096,
		"error_rate":   0.02,
	}
	context := map[string]interface{}{
		"environment": "production",
		"node_type":   "validator",
		"request_id":  "req-123",
	}
	input := pa.NormalizeInput("node-1", "execute_tool", "execute_tool", metrics, context)
	if input == nil {
		t.Fatal("Expected non-nil NormalizedPolicyInput")
	}
	if input.NodeID != "node-1" {
		t.Errorf("Expected NodeID node-1, got %s", input.NodeID)
	}
	if input.Action != "execute_tool" {
		t.Errorf("Expected Action execute_tool, got %s", input.Action)
	}
	if input.Source != PolicySourceKNIRV {
		t.Errorf("Expected Source knirv, got %s", input.Source)
	}
	if input.Metrics["cpu_usage"] != 75.5 {
		t.Errorf("Expected cpu_usage 75.5, got %f", input.Metrics["cpu_usage"])
	}
}

func TestGetInput(t *testing.T) {
	pa := NewPolicyAdapter()
	pa.NormalizeInput("node-1", "execute_tool", "execute_tool", nil, nil)

	input, ok := pa.GetInput("node-1", "execute_tool")
	if !ok {
		t.Error("Expected to retrieve input")
	}
	if input == nil {
		t.Fatal("Expected non-nil input")
	}
}

func TestRecordEnforcement(t *testing.T) {
	pa := NewPolicyAdapter()
	input := pa.NormalizeInput("node-1", "execute_tool", "execute_tool", nil, nil)

	evaluations := []PolicyEvaluation{
		{
			PolicyID:   "pol-1",
			PolicyName: "memory-guard",
			Decision:   "allow",
			Confidence: 0.95,
		},
	}

	enf := pa.RecordEnforcement(*input, "allow", "all good", evaluations)
	if enf == nil {
		t.Fatal("Expected non-nil NormalizedEnforcement")
	}
	if enf.Decision != "allow" {
		t.Errorf("Expected decision allow, got %s", enf.Decision)
	}
	if len(enf.Evaluations) != 1 {
		t.Errorf("Expected 1 evaluation, got %d", len(enf.Evaluations))
	}
}

func TestGetEnforcement(t *testing.T) {
	pa := NewPolicyAdapter()
	input := pa.NormalizeInput("node-1", "execute_tool", "execute_tool", nil, nil)
	enf := pa.RecordEnforcement(*input, "deny", "rate limit exceeded", nil)

	retrieved, ok := pa.GetEnforcement(enf.EnforcementID)
	if !ok {
		t.Error("Expected to retrieve enforcement")
	}
	if retrieved.Decision != "deny" {
		t.Errorf("Expected decision deny, got %s", retrieved.Decision)
	}
}

func TestListEnforcements(t *testing.T) {
	pa := NewPolicyAdapter()
	in1 := pa.NormalizeInput("node-1", "action-1", "type-1", nil, nil)
	in2 := pa.NormalizeInput("node-2", "action-2", "type-2", nil, nil)
	pa.RecordEnforcement(*in1, "allow", "", nil)
	pa.RecordEnforcement(*in2, "deny", "", nil)

	enfs := pa.ListEnforcements("node-1", 0)
	if len(enfs) != 1 {
		t.Errorf("Expected 1 enforcement for node-1, got %d", len(enfs))
	}

	enfs = pa.ListEnforcements("", 0)
	if len(enfs) != 2 {
		t.Errorf("Expected 2 enforcements for empty filter, got %d", len(enfs))
	}
}

func TestListEnforcementsLimit(t *testing.T) {
	pa := NewPolicyAdapter()
	for i := 0; i < 10; i++ {
		input := pa.NormalizeInput("node-1", "action", "type", nil, nil)
		pa.RecordEnforcement(*input, "allow", "", nil)
	}

	enfs := pa.ListEnforcements("node-1", 5)
	if len(enfs) > 5 {
		t.Errorf("Expected at most 5 enforcements with limit, got %d", len(enfs))
	}
}

func TestListInputs(t *testing.T) {
	pa := NewPolicyAdapter()
	pa.NormalizeInput("node-1", "action-1", "type-1", nil, nil)
	pa.NormalizeInput("node-2", "action-2", "type-2", nil, nil)
	pa.NormalizeInput("node-1", "action-3", "type-3", nil, nil)

	inputs := pa.ListInputs("node-1", 0)
	if len(inputs) != 2 {
		t.Errorf("Expected 2 inputs for node-1, got %d", len(inputs))
	}

	inputs = pa.ListInputs("", 0)
	if len(inputs) != 3 {
		t.Errorf("Expected 3 inputs for empty filter, got %d", len(inputs))
	}
}

func TestGetStatistics(t *testing.T) {
	pa := NewPolicyAdapter()
	input := pa.NormalizeInput("node-1", "action-1", "type-1", nil, nil)
	pa.RecordEnforcement(*input, "allow", "", nil)
	pa.RecordEnforcement(*input, "deny", "", nil)

	stats := pa.GetStatistics()
	if stats["total_inputs"].(int) != 1 {
		t.Errorf("Expected 1 input, got %d", stats["total_inputs"])
	}
	if stats["total_enforcements"].(int) != 2 {
		t.Errorf("Expected 2 enforcements, got %d", stats["total_enforcements"])
	}
}

func TestInputKey(t *testing.T) {
	pa := NewPolicyAdapter()
	key := pa.inputKey("node-1", "execute")
	if key != "node-1:execute" {
		t.Errorf("Expected 'node-1:execute', got '%s'", key)
	}
}
