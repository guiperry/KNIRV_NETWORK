package guardrails

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/database"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

type GuardrailType string

const (
	GuardrailTypeMemory     GuardrailType = "memory"
	GuardrailTypeCPU        GuardrailType = "cpu"
	GuardrailTypeNetwork    GuardrailType = "network"
	GuardrailTypeExecution  GuardrailType = "execution"
	GuardrailTypeFileSystem GuardrailType = "filesystem"
	GuardrailTypeOntology   GuardrailType = "ontology"
)

type GuardrailConfig struct {
	Type      GuardrailType `json:"type"`
	Enabled   bool          `json:"enabled"`
	MaxValue  float64       `json:"max_value"`
	MinValue  float64       `json:"min_value"`
	Action    string        `json:"action"` // "warn", "block", "isolate"
	Threshold float64       `json:"threshold"`
	Cooldown  time.Duration `json:"cooldown"`
}

type OntologyGuardrail struct {
	Domain      string   `json:"domain"`
	Concepts    []string `json:"concepts"`
	Constraints []string `json:"constraints"`
}

type GuardrailViolation struct {
	ID            string                 `json:"id"`
	NodeID        string                 `json:"node_id"`
	GuardrailType GuardrailType          `json:"guardrail_type"`
	ViolationType string                 `json:"violation_type"`
	Message       string                 `json:"message"`
	Severity      string                 `json:"severity"` // "low", "medium", "high", "critical"
	Context       map[string]interface{} `json:"context"`
	Timestamp     time.Time              `json:"timestamp"`
	Resolved      bool                   `json:"resolved"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
}

type DynamicGuardrailManager struct {
	db               *database.BuntDBManager
	mu               sync.RWMutex
	configurations   map[string]*GuardrailConfig
	ontologyRules    map[string]*OntologyGuardrail
	violations       map[string]*GuardrailViolation
	eventBroadcaster interface {
		EmitGuardrailTrigger(nodeID, source, guardrailType, trigger string, context map[string]interface{})
		EmitPolicyViolation(nodeID, source, violationType, details string)
	}
	cooldowns  map[string]time.Time
	cooldownMu sync.RWMutex
}

func NewDynamicGuardrailManager(db *database.BuntDBManager) *DynamicGuardrailManager {
	return &DynamicGuardrailManager{
		db:             db,
		configurations: make(map[string]*GuardrailConfig),
		ontologyRules:  make(map[string]*OntologyGuardrail),
		violations:     make(map[string]*GuardrailViolation),
		cooldowns:      make(map[string]time.Time),
	}
}

func (gm *DynamicGuardrailManager) SetEventBroadcaster(eb interface {
	EmitGuardrailTrigger(nodeID, source, guardrailType, trigger string, context map[string]interface{})
	EmitPolicyViolation(nodeID, source, violationType, details string)
}) {
	gm.eventBroadcaster = eb
}

func (gm *DynamicGuardrailManager) ConfigureGuardrail(nodeID string, config *GuardrailConfig) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	key := fmt.Sprintf("%s:%s", nodeID, config.Type)
	gm.configurations[key] = config

	if err := gm.saveConfiguration(key, config); err != nil {
		return fmt.Errorf("failed to persist guardrail config: %w", err)
	}

	log.Printf("Configured guardrail %s for node %s: max=%v, action=%s",
		config.Type, nodeID, config.MaxValue, config.Action)
	return nil
}

func (gm *DynamicGuardrailManager) GetGuardrailConfig(nodeID string, guardrailType GuardrailType) (*GuardrailConfig, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", nodeID, guardrailType)
	config, ok := gm.configurations[key]
	return config, ok
}

func (gm *DynamicGuardrailManager) ValidateResourceUsage(nodeID string, metrics map[string]interface{}) (*GuardrailViolation, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	violations := make([]*GuardrailViolation, 0)

	for guardrailType, config := range gm.configurations {
		if !config.Enabled {
			continue
		}

		if !gm.isNodeScope(guardrailType, nodeID) {
			continue
		}

		if gm.isInCooldown(guardrailType) {
			continue
		}

		var value float64
		switch config.Type {
		case GuardrailTypeMemory:
			if mem, ok := metrics["memory_usage"].(float64); ok {
				value = mem
			}
		case GuardrailTypeCPU:
			if cpu, ok := metrics["cpu_usage"].(float64); ok {
				value = cpu
			}
		case GuardrailTypeNetwork:
			if net, ok := metrics["network_bandwidth"].(float64); ok {
				value = net
			}
		}

		if value > config.MaxValue && config.MaxValue > 0 {
			violation := &GuardrailViolation{
				ID:            uuid.New().String(),
				NodeID:        nodeID,
				GuardrailType: config.Type,
				ViolationType: "resource_exceeded",
				Message:       fmt.Sprintf("%s usage %.2f exceeds max %.2f", config.Type, value, config.MaxValue),
				Severity:      gm.determineSeverity(config.Type, value, config.MaxValue),
				Context:       metrics,
				Timestamp:     time.Now(),
			}
			violations = append(violations, violation)
			gm.setCooldown(guardrailType, config.Cooldown)
		}
	}

	if len(violations) > 0 {
		highest := violations[0]
		for _, v := range violations[1:] {
			if gm.severityOrder(v.Severity) > gm.severityOrder(highest.Severity) {
				highest = v
			}
		}
		return highest, true
	}

	return nil, false
}

func (gm *DynamicGuardrailManager) ValidateOntologyConstraint(nodeID, domain string, concept string) (bool, string) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	rule, ok := gm.ontologyRules[domain]
	if !ok {
		return true, ""
	}

	for _, allowedConcept := range rule.Concepts {
		if allowedConcept == concept {
			return true, ""
		}
	}

	return false, fmt.Sprintf("concept '%s' not allowed in domain '%s'", concept, domain)
}

func (gm *DynamicGuardrailManager) ValidateExecutionTime(nodeID string, duration time.Duration) (bool, *GuardrailViolation) {
	config, ok := gm.GetGuardrailConfig(nodeID, GuardrailTypeExecution)
	if !ok || !config.Enabled {
		return true, nil
	}

	maxDuration := time.Duration(config.MaxValue) * time.Second
	if duration > maxDuration {
		violation := &GuardrailViolation{
			ID:            uuid.New().String(),
			NodeID:        nodeID,
			GuardrailType: GuardrailTypeExecution,
			ViolationType: "execution_timeout",
			Message:       fmt.Sprintf("execution took %v, exceeds max %v", duration, maxDuration),
			Severity:      "high",
			Context: map[string]interface{}{
				"duration":     duration.String(),
				"max_duration": maxDuration.String(),
			},
			Timestamp: time.Now(),
		}
		return false, violation
	}

	return true, nil
}

func (gm *DynamicGuardrailManager) ValidateIntentObjective(nodeID, objectiveID string, context map[string]interface{}) (bool, *GuardrailViolation) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	for guardrailType, config := range gm.configurations {
		if config.Type != GuardrailTypeOntology || !config.Enabled {
			continue
		}

		if !gm.isNodeScope(guardrailType, nodeID) {
			continue
		}

		if context == nil {
			continue
		}

		if intentValue, ok := context["intent_score"].(float64); ok {
			if intentValue < config.MinValue && config.MinValue > 0 {
				violation := &GuardrailViolation{
					ID:            uuid.New().String(),
					NodeID:        nodeID,
					GuardrailType: GuardrailTypeOntology,
					ViolationType: "intent_threshold_breach",
					Message:       fmt.Sprintf("intent score %.2f below minimum %.2f", intentValue, config.MinValue),
					Severity:      gm.determineSeverity(GuardrailTypeOntology, intentValue, config.MinValue),
					Context:       context,
					Timestamp:     time.Now(),
				}
				return false, violation
			}
		}
	}

	return true, nil
}

func (gm *DynamicGuardrailManager) RecordViolation(violation *GuardrailViolation) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.violations[violation.ID] = violation

	if gm.db == nil {
		return nil
	}
	return gm.db.Transaction(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(violation)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf("guardrail:violation:%s", violation.ID), string(data), nil)
		return err
	})
}

func (gm *DynamicGuardrailManager) GetViolations(nodeID string, limit int) []*GuardrailViolation {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	var result []*GuardrailViolation
	for _, v := range gm.violations {
		if nodeID == "" || v.NodeID == nodeID {
			result = append(result, v)
		}
	}

	if limit > 0 && len(result) > limit {
		return result[len(result)-limit:]
	}
	return result
}

func (gm *DynamicGuardrailManager) ResolveViolation(violationID string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	violation, ok := gm.violations[violationID]
	if !ok {
		return fmt.Errorf("violation not found: %s", violationID)
	}

	violation.Resolved = true
	now := time.Now()
	violation.ResolvedAt = &now

	if gm.db == nil {
		return nil
	}
	return gm.db.Transaction(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(violation)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf("guardrail:violation:%s", violation.ID), string(data), nil)
		return err
	})
}

func (gm *DynamicGuardrailManager) ConfigureOntologyGuardrail(domain string, rule *OntologyGuardrail) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.ontologyRules[domain] = rule

	if gm.db == nil {
		return nil
	}
	return gm.db.Transaction(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(rule)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(fmt.Sprintf("guardrail:ontology:%s", domain), string(data), nil)
		return err
	})
}

func (gm *DynamicGuardrailManager) isNodeScope(key, nodeID string) bool {
	for _, gt := range []GuardrailType{GuardrailTypeMemory, GuardrailTypeCPU, GuardrailTypeNetwork, GuardrailTypeExecution, GuardrailTypeFileSystem, GuardrailTypeOntology} {
		suffix := ":" + string(gt)
		if len(key) > len(suffix) && key[len(key)-len(suffix):] == suffix {
			configuredNodeID := key[:len(key)-len(suffix)]
			return configuredNodeID == nodeID || configuredNodeID == "global"
		}
	}
	return false
}

func (gm *DynamicGuardrailManager) isInCooldown(key string) bool {
	gm.cooldownMu.RLock()
	defer gm.cooldownMu.RUnlock()

	if cooldown, ok := gm.cooldowns[key]; ok {
		return time.Now().Before(cooldown)
	}
	return false
}

func (gm *DynamicGuardrailManager) setCooldown(key string, duration time.Duration) {
	gm.cooldownMu.Lock()
	defer gm.cooldownMu.Unlock()
	gm.cooldowns[key] = time.Now().Add(duration)
}

func (gm *DynamicGuardrailManager) severityOrder(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func (gm *DynamicGuardrailManager) determineSeverity(gType GuardrailType, value, max float64) string {
	ratio := value / max
	if ratio > 1.5 {
		return "critical"
	} else if ratio > 1.2 {
		return "high"
	} else if ratio > 1.0 {
		return "medium"
	}
	return "low"
}

func (gm *DynamicGuardrailManager) saveConfiguration(key string, config *GuardrailConfig) error {
	if gm.db == nil {
		return nil
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return gm.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(fmt.Sprintf("guardrail:config:%s", key), string(data), nil)
		return err
	})
}

func (gm *DynamicGuardrailManager) LoadConfigurations() error {
	if gm.db == nil {
		return nil
	}
	return gm.db.GetObjectsByPrefix("guardrail:config:", func(key string, value []byte) bool {
		var config GuardrailConfig
		if err := json.Unmarshal(value, &config); err != nil {
			return true
		}
		gm.mu.Lock()
		gm.configurations[key] = &config
		gm.mu.Unlock()
		return true
	})
}

func (gm *DynamicGuardrailManager) LoadOntologyRules() error {
	if gm.db == nil {
		return nil
	}
	return gm.db.GetObjectsByPrefix("guardrail:ontology:", func(key string, value []byte) bool {
		var rule OntologyGuardrail
		if err := json.Unmarshal(value, &rule); err != nil {
			return true
		}
		gm.mu.Lock()
		gm.ontologyRules[rule.Domain] = &rule
		gm.mu.Unlock()
		return true
	})
}

func (gm *DynamicGuardrailManager) GetAllConfigurations(nodeID string) []*GuardrailConfig {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	var configs []*GuardrailConfig
	for key, config := range gm.configurations {
		if nodeID == "" || gm.isNodeScope(key, nodeID) {
			configs = append(configs, config)
		}
	}
	return configs
}

func (gm *DynamicGuardrailManager) GetStatistics() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	totalViolations := len(gm.violations)
	unresolvedViolations := 0
	severityCounts := map[string]int{"low": 0, "medium": 0, "high": 0, "critical": 0}

	for _, v := range gm.violations {
		if !v.Resolved {
			unresolvedViolations++
		}
		severityCounts[v.Severity]++
	}

	return map[string]interface{}{
		"total_configurations":   len(gm.configurations),
		"total_ontology_rules":   len(gm.ontologyRules),
		"total_violations":       totalViolations,
		"unresolved_violations":  unresolvedViolations,
		"violations_by_severity": severityCounts,
		"active_cooldowns":       len(gm.cooldowns),
	}
}

type PolicyEngine struct {
	db               *database.BuntDBManager
	guardrailManager *DynamicGuardrailManager
	icmeService      interface {
		RegisterObjective(obj *IntentObjective) error
		GetObjectiveForAgent(agentID, dveID string) *IntentObjective
	}
	activePolicies map[string]*Policy
	mu             sync.RWMutex
}

type Policy struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Enabled     bool          `json:"enabled"`
	Priority    int           `json:"priority"`
	Rules       []*PolicyRule `json:"rules"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type PolicyRule struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // "guardrail", "ontology", "execution"
	Condition  map[string]interface{} `json:"condition"`
	Action     string                 `json:"action"` // "allow", "deny", "warn"
	Parameters map[string]interface{} `json:"parameters"`
}

type IntentObjective struct {
	Name           string             `json:"name"`
	Scope          string             `json:"scope"` // "global" or DVE ID
	HardBoundaries map[string]float64 `json:"hard_boundaries"`
	SoftLimits     map[string]float64 `json:"soft_limits"`
	TradeOffs      map[string]float64 `json:"trade_offs"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	Version        int                `json:"version"`
}

func NewPolicyEngine(db *database.BuntDBManager, gm *DynamicGuardrailManager) *PolicyEngine {
	return &PolicyEngine{
		db:               db,
		guardrailManager: gm,
		activePolicies:   make(map[string]*Policy),
	}
}

func (pe *PolicyEngine) SetICMEService(icme interface {
	RegisterObjective(obj *IntentObjective) error
	GetObjectiveForAgent(agentID, dveID string) *IntentObjective
}) {
	pe.icmeService = icme
}

func (pe *PolicyEngine) CreatePolicy(policy *Policy) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	policy.ID = uuid.New().String()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	pe.activePolicies[policy.ID] = policy

	for _, rule := range policy.Rules {
		rule.ID = uuid.New().String()
		if rule.Type == "guardrail" {
			if err := pe.applyGuardrailRule(policy.ID, rule); err != nil {
				log.Printf("Warning: Failed to apply guardrail rule: %v", err)
			}
		} else if rule.Type == "ontology" {
			if err := pe.applyOntologyRule(policy.ID, rule); err != nil {
				log.Printf("Warning: Failed to apply ontology rule: %v", err)
			}
		}
	}

	return pe.savePolicy(policy)
}

func (pe *PolicyEngine) applyGuardrailRule(policyID string, rule *PolicyRule) error {
	nodeID, _ := rule.Parameters["node_id"].(string)
	if nodeID == "" {
		nodeID = "global"
	}

	guardrailType := GuardrailType(rule.Parameters["guardrail_type"].(string))
	config := &GuardrailConfig{
		Type:     guardrailType,
		Enabled:  rule.Action != "deny",
		Action:   rule.Action,
		MaxValue: rule.Condition["max_value"].(float64),
	}

	if minValue, ok := rule.Condition["min_value"].(float64); ok {
		config.MinValue = minValue
	}

	return pe.guardrailManager.ConfigureGuardrail(nodeID, config)
}

func (pe *PolicyEngine) applyOntologyRule(policyID string, rule *PolicyRule) error {
	domain := rule.Parameters["domain"].(string)
	ruleGuardrail := &OntologyGuardrail{
		Domain: domain,
	}

	if concepts, ok := rule.Parameters["concepts"].([]interface{}); ok {
		for _, c := range concepts {
			if concept, ok := c.(string); ok {
				ruleGuardrail.Concepts = append(ruleGuardrail.Concepts, concept)
			}
		}
	}

	return pe.guardrailManager.ConfigureOntologyGuardrail(domain, ruleGuardrail)
}

func (pe *PolicyEngine) EvaluateAction(nodeID, action string, context map[string]interface{}) (bool, string) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var applicablePolicies []*Policy
	for _, policy := range pe.activePolicies {
		if !policy.Enabled {
			continue
		}
		applicablePolicies = append(applicablePolicies, policy)
	}

	for _, policy := range applicablePolicies {
		for _, rule := range policy.Rules {
			if pe.matchesCondition(rule.Condition, context) {
				if rule.Action == "deny" {
					return false, fmt.Sprintf("denied by policy '%s' rule '%s'", policy.Name, rule.ID)
				}
				if rule.Action == "warn" {
					return true, fmt.Sprintf("warning from policy '%s'", policy.Name)
				}
			}
		}
	}

	return true, "allowed"
}

func (pe *PolicyEngine) matchesCondition(condition map[string]interface{}, context map[string]interface{}) bool {
	if condition == nil {
		return true
	}

	for key, expected := range condition {
		actual, ok := context[key]
		if !ok {
			return false
		}

		switch exp := expected.(type) {
		case float64:
			if act, ok := actual.(float64); ok {
				if act < exp {
					return false
				}
			}
		case string:
			if act, ok := actual.(string); ok {
				if act != exp {
					return false
				}
			}
		case bool:
			if act, ok := actual.(bool); ok {
				if act != exp {
					return false
				}
			}
		}
	}

	return true
}

func (pe *PolicyEngine) CommitPolicyToBlockchain(policyID string) (string, error) {
	pe.mu.RLock()
	policy, ok := pe.activePolicies[policyID]
	pe.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("policy not found: %s", policyID)
	}

	policyData, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal policy: %w", err)
	}

	txHash := fmt.Sprintf("pol_%s_%d", policyID, time.Now().UnixNano())

	if err := pe.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(fmt.Sprintf("policy:committed:%s", policyID), string(policyData), nil)
		return err
	}); err != nil {
		return "", fmt.Errorf("failed to commit policy: %w", err)
	}

	log.Printf("Policy %s committed to KNIRVCHAIN with tx hash: %s", policyID, txHash)
	return txHash, nil
}

func (pe *PolicyEngine) savePolicy(policy *Policy) error {
	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	return pe.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(fmt.Sprintf("policy:%s", policy.ID), string(data), nil)
		return err
	})
}

func (pe *PolicyEngine) LoadPolicies() error {
	return pe.db.GetObjectsByPrefix("policy:", func(key string, value []byte) bool {
		var policy Policy
		if err := json.Unmarshal(value, &policy); err != nil {
			return true
		}
		pe.mu.Lock()
		pe.activePolicies[policy.ID] = &policy
		pe.mu.Unlock()
		return true
	})
}

func (pe *PolicyEngine) GetPolicy(policyID string) (*Policy, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	policy, ok := pe.activePolicies[policyID]
	return policy, ok
}

func (pe *PolicyEngine) ListPolicies() []*Policy {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	policies := make([]*Policy, 0, len(pe.activePolicies))
	for _, p := range pe.activePolicies {
		policies = append(policies, p)
	}
	return policies
}

func (pe *PolicyEngine) DeletePolicy(policyID string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if _, ok := pe.activePolicies[policyID]; !ok {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	delete(pe.activePolicies, policyID)

	return pe.db.Transaction(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(fmt.Sprintf("policy:%s", policyID))
		return err
	})
}
