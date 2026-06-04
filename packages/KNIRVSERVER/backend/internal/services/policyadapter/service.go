package policyadapter

import (
	"fmt"
	"sync"
	"time"
)

type PolicySource string

const (
	PolicySourceAGT  PolicySource = "agt"
	PolicySourceOPA  PolicySource = "opa"
	PolicySourceKNIRV PolicySource = "knirv"
)

type NormalizedPolicyInput struct {
	NodeID      string                 `json:"node_id"`
	AgentID     string                 `json:"agent_id,omitempty"`
	Action      string                 `json:"action"`
	ActionType  string                 `json:"action_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Metrics     map[string]float64     `json:"metrics,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Source      PolicySource           `json:"source,omitempty"`
}

type PolicyEvaluation struct {
	PolicyID    string                 `json:"policy_id"`
	PolicyName  string                 `json:"policy_name"`
	Decision    string                 `json:"decision"`
	Reason      string                 `json:"reason,omitempty"`
	Confidence  float64                `json:"confidence"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
	MatchedRules []MatchedRule         `json:"matched_rules,omitempty"`
}

type MatchedRule struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Decision    string                 `json:"decision"`
	Condition   map[string]interface{} `json:"condition,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type NormalizedEnforcement struct {
	NodeID         string             `json:"node_id"`
	Action         string             `json:"action"`
	EnforcementID  string             `json:"enforcement_id"`
	Decision       string             `json:"decision"`
	Reason         string             `json:"reason,omitempty"`
	Input          NormalizedPolicyInput `json:"input"`
	Evaluations    []PolicyEvaluation `json:"evaluations,omitempty"`
	EnforcedAt     time.Time          `json:"enforced_at"`
	Source         PolicySource       `json:"source"`
}

type PortabilityContract struct {
	Version        string              `json:"version"`
	Schema         string              `json:"schema"`
	MetricNames    []string            `json:"metric_names"`
	ActionTypes    []string            `json:"action_types"`
	ContextFields  []string            `json:"context_fields"`
	OutputFormat   string              `json:"output_format"`
}

type PolicyAdapter struct {
	mu            sync.RWMutex
	inputs        map[string]*NormalizedPolicyInput
	enforcements  map[string]*NormalizedEnforcement
	contract      PortabilityContract
}

func NewPolicyAdapter() *PolicyAdapter {
	return &PolicyAdapter{
		inputs:       make(map[string]*NormalizedPolicyInput),
		enforcements: make(map[string]*NormalizedEnforcement),
		contract: PortabilityContract{
			Version: "1.0.0",
			Schema:  "https://knirv.network/schemas/policy-input-1.0.json",
			MetricNames: []string{
				"cpu_usage", "memory_usage", "network_bandwidth",
				"execution_time_ms", "token_consumption", "error_rate",
				"latency_p50", "latency_p99", "request_count",
			},
			ActionTypes: []string{
				"execute_tool", "read_file", "write_file",
				"network_request", "spawn_agent", "access_memory",
				"invoke_llm", "deploy_plugin",
			},
			ContextFields: []string{
				"environment", "node_type", "agent_role",
				"session_id", "request_id", "user_id",
			},
			OutputFormat: "application/json",
		},
	}
}

func (pa *PolicyAdapter) GetContract() PortabilityContract {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	return pa.contract
}

func (pa *PolicyAdapter) NormalizeInput(nodeID, action, actionType string, metrics map[string]float64, context map[string]interface{}) *NormalizedPolicyInput {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	input := &NormalizedPolicyInput{
		NodeID:     nodeID,
		Action:     action,
		ActionType: actionType,
		Timestamp:  time.Now().UTC(),
		Metrics:    metrics,
		Context:    context,
		Labels:     make(map[string]string),
		Source:     PolicySourceKNIRV,
	}

	pa.inputs[pa.inputKey(nodeID, action)] = input
	return input
}

func (pa *PolicyAdapter) RecordEnforcement(input NormalizedPolicyInput, decision, reason string, evaluations []PolicyEvaluation) *NormalizedEnforcement {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	enf := &NormalizedEnforcement{
		NodeID:        input.NodeID,
		Action:        input.Action,
		EnforcementID: fmt.Sprintf("enf-%d", time.Now().UnixNano()),
		Decision:      decision,
		Reason:        reason,
		Input:         input,
		Evaluations:   evaluations,
		EnforcedAt:    time.Now().UTC(),
		Source:        input.Source,
	}

	pa.enforcements[enf.EnforcementID] = enf
	return enf
}

func (pa *PolicyAdapter) GetEnforcement(enfID string) (*NormalizedEnforcement, bool) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	enf, ok := pa.enforcements[enfID]
	return enf, ok
}

func (pa *PolicyAdapter) ListEnforcements(nodeID string, limit int) []*NormalizedEnforcement {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	var result []*NormalizedEnforcement
	for _, enf := range pa.enforcements {
		if nodeID == "" || enf.NodeID == nodeID {
			result = append(result, enf)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

func (pa *PolicyAdapter) GetInput(nodeID, action string) (*NormalizedPolicyInput, bool) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	input, ok := pa.inputs[pa.inputKey(nodeID, action)]
	return input, ok
}

func (pa *PolicyAdapter) ListInputs(nodeID string, limit int) []*NormalizedPolicyInput {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	var result []*NormalizedPolicyInput
	for _, inp := range pa.inputs {
		if nodeID == "" || inp.NodeID == nodeID {
			result = append(result, inp)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

func (pa *PolicyAdapter) GetStatistics() map[string]interface{} {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	decisionCounts := map[string]int{}
	for _, enf := range pa.enforcements {
		decisionCounts[enf.Decision]++
	}

	return map[string]interface{}{
		"total_inputs":       len(pa.inputs),
		"total_enforcements": len(pa.enforcements),
		"enforcement_decisions": decisionCounts,
		"contract_version":   pa.contract.Version,
	}
}

func (pa *PolicyAdapter) inputKey(nodeID, action string) string {
	return nodeID + ":" + action
}
