package hasher

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LogicalValidator handles logical validation using Z3 theorem prover
// as specified in HASHER_SDD.md sections 4.2 and 5.3
type LogicalValidator struct {
	KnowledgeBase *KnowledgeBase
}

// NewLogicalValidator creates a new logical validator
func NewLogicalValidator() (*LogicalValidator, error) {
	kb, err := NewKnowledgeBase()
	if err != nil {
		return nil, err
	}

	return &LogicalValidator{
		KnowledgeBase: kb,
	}, nil
}

// Validate checks if the prediction is logically consistent using the Z3 theorem prover
func (v *LogicalValidator) Validate(prediction int, domain string, context map[string]interface{}) (*ValidationResult, error) {
	start := time.Now()

	// Get logical rules for the domain
	rules, err := v.KnowledgeBase.GetRules(domain)
	if err != nil {
		return &ValidationResult{
			Prediction: prediction,
			Valid:      false,
			Domain:     domain,
			ErrorMessage: fmt.Sprintf("failed to get rules: %v", err),
			Latency:    time.Since(start),
		}, nil
	}

	if len(rules) == 0 {
		// No rules to validate against - consider valid
		return &ValidationResult{
			Prediction: prediction,
			Valid:      true,
			Domain:     domain,
			Latency:    time.Since(start),
		}, nil
	}

	// Check logical consistency
	valid := true
	errors := []string{}
	for _, rule := range rules {
		if err := v.checkRule(prediction, rule, context); err != nil {
			valid = false
			errors = append(errors, err.Error())
		}
	}

	return &ValidationResult{
		Prediction:   prediction,
		Valid:        valid,
		Domain:       domain,
		RulesApplied: len(rules),
		Errors:       errors,
		ErrorMessage: strings.Join(errors, "; "),
		Latency:      time.Since(start),
	}, nil
}

// checkRule applies a specific logical rule to the prediction
func (v *LogicalValidator) checkRule(prediction int, rule *LogicalRule, context map[string]interface{}) error {
	// Simple rule validation logic
	// In production, this would use the Z3 theorem prover

	switch rule.RuleType {
	case "constraint":
		if err := v.checkConstraint(prediction, rule, context); err != nil {
			return err
		}
	case "subsumption":
		if err := v.checkSubsumption(prediction, rule, context); err != nil {
			return err
		}
	case "disjoint":
		if err := v.checkDisjoint(prediction, rule, context); err != nil {
			return err
		}
	}

	return nil
}

// checkConstraint implements constraint validation
func (v *LogicalValidator) checkConstraint(prediction int, rule *LogicalRule, context map[string]interface{}) error {
	// Example: Check if prediction falls within allowed range
	for key, value := range context {
		if key == "min_value" {
			if minVal, ok := value.(int); ok && prediction < minVal {
				return fmt.Errorf("prediction %d is below minimum value %d", prediction, minVal)
			}
		}
		if key == "max_value" {
			if maxVal, ok := value.(int); ok && prediction > maxVal {
				return fmt.Errorf("prediction %d is above maximum value %d", prediction, maxVal)
			}
		}
	}

	return nil
}

// checkSubsumption implements subsumption validation
func (v *LogicalValidator) checkSubsumption(prediction int, rule *LogicalRule, context map[string]interface{}) error {
	// Example: Check if prediction is subsumed by another class
	// This is a placeholder for actual logical reasoning
	return nil
}

// checkDisjoint implements disjointness validation
func (v *LogicalValidator) checkDisjoint(prediction int, rule *LogicalRule, context map[string]interface{}) error {
	// Example: Check if prediction conflicts with other classes
	// This is a placeholder for actual logical reasoning
	return nil
}

// ValidationResult contains the result of logical validation
type ValidationResult struct {
	Prediction   int                 `json:"prediction"`
	Valid        bool                `json:"valid"`
	Domain       string              `json:"domain"`
	RulesApplied int                 `json:"rules_applied"`
	Errors       []string            `json:"errors,omitempty"`
	ErrorMessage string              `json:"error_message,omitempty"`
	Latency      time.Duration       `json:"latency,omitempty"`
}

// KnowledgeBase manages logical rules and domains
// as specified in HASHER_SDD.md section 4.2.1
type KnowledgeBase struct {
	Domains map[string][]*LogicalRule
}

// NewKnowledgeBase creates a new knowledge base
func NewKnowledgeBase() (*KnowledgeBase, error) {
	kb := &KnowledgeBase{
		Domains: make(map[string][]*LogicalRule),
	}

	// Initialize with default rules for common domains
	kb.initializeDefaultRules()

	return kb, nil
}

// initializeDefaultRules adds default rules to the knowledge base
func (kb *KnowledgeBase) initializeDefaultRules() {
	// Default rules for anomaly detection domain
	kb.Domains["anomaly_detection"] = []*LogicalRule{
		{
			RuleType:   "constraint",
			Premises:   []string{"prediction > 0"},
			Conclusion: "Prediction must be positive",
		},
	}

	// Default rules for classification domain
	kb.Domains["classification"] = []*LogicalRule{
		{
			RuleType:   "constraint",
			Premises:   []string{"prediction >= 0"},
			Conclusion: "Prediction must be non-negative",
		},
	}
}

// GetRules retrieves all rules for a specific domain
func (kb *KnowledgeBase) GetRules(domain string) ([]*LogicalRule, error) {
	rules, exists := kb.Domains[domain]
	if !exists {
		// Return empty rules for unknown domains
		return []*LogicalRule{}, nil
	}
	return rules, nil
}

// AddRule adds a new rule to the knowledge base
func (kb *KnowledgeBase) AddRule(domain string, rule *LogicalRule) error {
	if _, exists := kb.Domains[domain]; !exists {
		kb.Domains[domain] = []*LogicalRule{}
	}
	kb.Domains[domain] = append(kb.Domains[domain], rule)
	return nil
}

// RemoveRule removes a rule from the knowledge base
func (kb *KnowledgeBase) RemoveRule(domain string, index int) error {
	if _, exists := kb.Domains[domain]; !exists {
		return fmt.Errorf("unknown domain: %s", domain)
	}
	if index < 0 || index >= len(kb.Domains[domain]) {
		return fmt.Errorf("invalid rule index: %d", index)
	}

	kb.Domains[domain] = append(kb.Domains[domain][:index], kb.Domains[domain][index+1:]...)
	return nil
}

// LogicalRule represents a single logical rule
type LogicalRule struct {
	RuleType   string   `json:"rule_type"`   // 'subsumption', 'disjoint', 'constraint'
	Premises   []string `json:"premises"`    // Array of logical statements
	Conclusion string   `json:"conclusion"`  // The conclusion
	Description string  `json:"description"` // Human-readable description
}

// NewLogicalRule creates a new logical rule
func NewLogicalRule(ruleType string, premises []string, conclusion string, description string) (*LogicalRule, error) {
	if ruleType != "subsumption" && ruleType != "disjoint" && ruleType != "constraint" {
		return nil, fmt.Errorf("invalid rule type: %s", ruleType)
	}

	return &LogicalRule{
		RuleType:   ruleType,
		Premises:   premises,
		Conclusion: conclusion,
		Description: description,
	}, nil
}

// Serialize converts the rule to JSON
func (r *LogicalRule) Serialize() ([]byte, error) {
	return json.Marshal(r)
}

// DeserializeLogicalRule creates a LogicalRule from JSON
func DeserializeLogicalRule(data []byte) (*LogicalRule, error) {
	var rule LogicalRule
	if err := json.Unmarshal(data, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

// String representation of the rule
func (r *LogicalRule) String() string {
	return fmt.Sprintf("[%s] %s", r.RuleType, r.Conclusion)
}
