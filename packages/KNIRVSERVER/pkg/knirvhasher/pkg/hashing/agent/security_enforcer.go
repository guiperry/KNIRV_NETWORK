package agent

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"knirvhasher/pkg/storage"
)

type SecurityEnforcer struct {
	kvbaseClient *storage.KNIRVBASEClient
	userGates    map[string]*UserGates
	mu           sync.RWMutex
}

type UserGates struct {
	UserID   string
	Seeds    []storage.Seed
	Rules    []storage.LogicalRule
	Fitness  float64
	LoadedAt int64
}

type SecurityDecision struct {
	Allowed      bool
	Confidence   float64
	Violations   []string
	AppliedRules []string
	SeedID       string
	DecisionID   string
}

func NewSecurityEnforcer(kvbase *storage.KNIRVBASEClient) *SecurityEnforcer {
	return &SecurityEnforcer{
		kvbaseClient: kvbase,
		userGates:    make(map[string]*UserGates),
	}
}

func (se *SecurityEnforcer) ValidateAction(userID, action string, ctx map[string]string) (*SecurityDecision, error) {
	gates, err := se.loadUserGates(userID)
	if err != nil {
		return &SecurityDecision{
			Allowed:    false,
			Confidence: 0,
			Violations: []string{"failed to load user gates"},
		}, err
	}

	prediction := se.encodeAction(action, ctx)

	violations := se.validateRules(prediction, gates.Rules)

	decision := &SecurityDecision{
		Allowed:      len(violations) == 0,
		Confidence:   gates.Fitness,
		Violations:   violations,
		AppliedRules: se.getAppliedRules(gates.Rules),
		SeedID:       se.getBestSeedID(gates),
		DecisionID:   fmt.Sprintf("dec_%s_%d", userID, nowMillis()),
	}

	return decision, nil
}

func (se *SecurityEnforcer) loadUserGates(userID string) (*UserGates, error) {
	se.mu.RLock()
	if gates, ok := se.userGates[userID]; ok {
		se.mu.RUnlock()
		return gates, nil
	}
	se.mu.RUnlock()

	seeds, err := se.kvbaseClient.GetUserSeeds(userID)
	if err != nil {
		return nil, err
	}

	rules, err := se.kvbaseClient.GetUserRules(userID)
	if err != nil {
		return nil, err
	}

	gates := &UserGates{
		UserID:   userID,
		Seeds:    seeds,
		Rules:    rules,
		Fitness:  0.9,
		LoadedAt: nowMillis(),
	}

	se.mu.Lock()
	se.userGates[userID] = gates
	se.mu.Unlock()

	return gates, nil
}

func (se *SecurityEnforcer) encodeAction(action string, ctx map[string]string) int {
	hash := 0
	for i, c := range action {
		hash ^= int(c) << (i % 8)
	}
	for k, v := range ctx {
		for i, c := range k + v {
			hash ^= int(c) << (i % 8)
		}
	}
	return hash
}

func (se *SecurityEnforcer) validateRules(prediction int, rules []storage.LogicalRule) []string {
	violations := make([]string, 0)

	for _, rule := range rules {
		if rule.Conclusion == "deny" {
			// Check if prediction violates any premise in deny rules
			for _, premise := range rule.Premises {
				// Parse premise to check if prediction matches forbidden values
				// Example premises: "prediction == 42", "prediction in [1,2,3]", "prediction >= 100"
				if se.predictionViolatesPremise(prediction, premise) {
					violations = append(violations, fmt.Sprintf("rule violated: %s (prediction %d)", premise, prediction))
				}
			}
		}
	}

	return violations
}

// predictionViolatesPremise checks if a prediction violates a given premise
func (se *SecurityEnforcer) predictionViolatesPremise(prediction int, premise string) bool {
	// Simple premise parsing logic
	// Check for equality: "prediction == 42"
	if strings.HasPrefix(premise, "prediction == ") {
		valStr := strings.TrimPrefix(premise, "prediction == ")
		if val, err := strconv.Atoi(valStr); err == nil {
			return prediction == val
		}
	}

	// Check for inequality: "prediction != 42"
	if strings.HasPrefix(premise, "prediction != ") {
		valStr := strings.TrimPrefix(premise, "prediction != ")
		if val, err := strconv.Atoi(valStr); err == nil {
			return prediction != val
		}
	}

	// Check for list membership: "prediction in [1,2,3]"
	if strings.HasPrefix(premise, "prediction in [") && strings.HasSuffix(premise, "]") {
		listStr := strings.TrimPrefix(premise, "prediction in [")
		listStr = strings.TrimSuffix(listStr, "]")
		items := strings.Split(listStr, ",")
		for _, item := range items {
			if val, err := strconv.Atoi(strings.TrimSpace(item)); err == nil && prediction == val {
				return true
			}
		}
	}

	// Default: check if premise contains prediction as substring (fallback)
	return strings.Contains(premise, fmt.Sprintf("%d", prediction))
}

func (se *SecurityEnforcer) getAppliedRules(rules []storage.LogicalRule) []string {
	applied := make([]string, 0, len(rules))
	for _, r := range rules {
		applied = append(applied, r.RuleType)
	}
	return applied
}

func (se *SecurityEnforcer) getBestSeedID(gates *UserGates) string {
	if len(gates.Seeds) == 0 {
		return ""
	}
	return fmt.Sprintf("user_%s_seed_0", gates.UserID)
}

func (se *SecurityEnforcer) RefreshUserGates(userID string) error {
	se.mu.Lock()
	delete(se.userGates, userID)
	se.mu.Unlock()

	_, err := se.loadUserGates(userID)
	return err
}

func (se *SecurityEnforcer) ClearCache() {
	se.mu.Lock()
	se.userGates = make(map[string]*UserGates)
	se.mu.Unlock()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func nowMillis() int64 {
	return 0
}
