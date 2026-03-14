package cognitiveengine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// PolicyRule defines a single per-DVE guardrail constraint.
type PolicyRule struct {
	ID                string
	Description       string
	DVEID             string  // empty = applies to all DVEs
	Metric            string  // "success_rate", "avg_processing_time", "resource_utilization", "violation_count"
	Operator          string  // "lt", "gt", "eq", "lte", "gte"
	Threshold         float64
	Severity          string // "warning", "critical", "panic"
	RemediationAction string
	Enabled           bool
	CreatedAt         time.Time
	LastTriggered     time.Time
	TriggerCount      int
}

// PolicyViolation records a single detected policy breach and its remediation status.
type PolicyViolation struct {
	ID                string
	RuleID            string
	DVEID             string
	NodeID            string
	MetricValue       float64
	Severity          string
	DetectedAt        time.Time
	Remediated        bool
	RemediatedAt      *time.Time
	RemediationResult string
}

// RemediationFn is a function that handles a policy violation.
type RemediationFn func(ctx context.Context, violation *PolicyViolation) error

// GuardrailEngine enforces per-DVE policies, records violations, triggers
// automated remediation, and feeds learning data back to refine thresholds.
type GuardrailEngine struct {
	policies    map[string]*PolicyRule
	violations  []PolicyViolation
	remediators map[string]RemediationFn
	eventBus    *EventBus
	mu          sync.RWMutex
}

// NewGuardrailEngine creates a GuardrailEngine with the built-in default policies
// and remediators pre-registered.
func NewGuardrailEngine(bus *EventBus) *GuardrailEngine {
	ge := &GuardrailEngine{
		policies:    make(map[string]*PolicyRule),
		violations:  make([]PolicyViolation, 0, 64),
		remediators: make(map[string]RemediationFn),
		eventBus:    bus,
	}
	ge.registerDefaultPolicies()
	ge.registerDefaultRemediators()
	return ge
}

func (ge *GuardrailEngine) registerDefaultPolicies() {
	defaults := []*PolicyRule{
		{
			ID:                "dveguard_low_success",
			Description:       "DVE node has critically low task success rate",
			Metric:            "success_rate",
			Operator:          "lt",
			Threshold:         0.4,
			Severity:          "critical",
			RemediationAction: "quarantine_node",
			Enabled:           true,
			CreatedAt:         time.Now(),
		},
		{
			ID:                "dveguard_slow_response",
			Description:       "DVE node average response time exceeds safety threshold",
			Metric:            "avg_processing_time",
			Operator:          "gt",
			Threshold:         300.0, // seconds
			Severity:          "warning",
			RemediationAction: "redistribute_tasks",
			Enabled:           true,
			CreatedAt:         time.Now(),
		},
		{
			ID:                "dveguard_high_resource",
			Description:       "DVE node resource utilization is critically high",
			Metric:            "resource_utilization",
			Operator:          "gt",
			Threshold:         0.95,
			Severity:          "critical",
			RemediationAction: "scale_resources",
			Enabled:           true,
			CreatedAt:         time.Now(),
		},
		{
			ID:                "dveguard_panic_trigger",
			Description:       "DVE node has breached multiple critical policies – kernel isolation required",
			Metric:            "violation_count",
			Operator:          "gt",
			Threshold:         5.0,
			Severity:          "panic",
			RemediationAction: "kernel_isolation",
			Enabled:           true,
			CreatedAt:         time.Now(),
		},
	}
	for _, p := range defaults {
		ge.policies[p.ID] = p
	}
}

func (ge *GuardrailEngine) registerDefaultRemediators() {
	ge.remediators["quarantine_node"] = func(ctx context.Context, v *PolicyViolation) error {
		log.Printf("[GUARDRAIL] Quarantining node %s (DVE: %s) – rule: %s metric=%.4f",
			v.NodeID, v.DVEID, v.RuleID, v.MetricValue)
		return nil
	}
	ge.remediators["redistribute_tasks"] = func(ctx context.Context, v *PolicyViolation) error {
		log.Printf("[GUARDRAIL] Redistributing tasks from slow node %s (DVE: %s) metric=%.2fs",
			v.NodeID, v.DVEID, v.MetricValue)
		return nil
	}
	ge.remediators["scale_resources"] = func(ctx context.Context, v *PolicyViolation) error {
		log.Printf("[GUARDRAIL] Requesting resource scale-up for node %s (DVE: %s) utilization=%.2f",
			v.NodeID, v.DVEID, v.MetricValue)
		return nil
	}
	ge.remediators["kernel_isolation"] = func(ctx context.Context, v *PolicyViolation) error {
		log.Printf("[GUARDRAIL] PANIC: Kernel isolation triggered for node %s (DVE: %s) violations=%.0f",
			v.NodeID, v.DVEID, v.MetricValue)
		return nil
	}
}

// AddPolicy registers (or replaces) a policy rule.
func (ge *GuardrailEngine) AddPolicy(rule *PolicyRule) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.policies[rule.ID] = rule
}

// GetPolicy retrieves a policy rule by ID.
func (ge *GuardrailEngine) GetPolicy(id string) (*PolicyRule, bool) {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	p, ok := ge.policies[id]
	return p, ok
}

// RegisterRemediator wires a custom remediation function to an action name.
func (ge *GuardrailEngine) RegisterRemediator(action string, fn RemediationFn) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.remediators[action] = fn
}

// Evaluate checks the supplied metrics map against all active policies for the
// given (nodeID, dveID) pair.  It executes remediation for each triggered rule
// and returns the full list of new violations.
func (ge *GuardrailEngine) Evaluate(
	ctx context.Context,
	nodeID, dveID string,
	metrics map[string]float64,
) []*PolicyViolation {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	var triggered []*PolicyViolation

	for _, policy := range ge.policies {
		if !policy.Enabled {
			continue
		}
		if policy.DVEID != "" && policy.DVEID != dveID {
			continue
		}
		value, ok := metrics[policy.Metric]
		if !ok {
			continue
		}
		if !ge.checkCondition(value, policy.Operator, policy.Threshold) {
			continue
		}

		v := &PolicyViolation{
			ID:          fmt.Sprintf("viol_%d_%s", time.Now().UnixNano(), policy.ID),
			RuleID:      policy.ID,
			DVEID:       dveID,
			NodeID:      nodeID,
			MetricValue: value,
			Severity:    policy.Severity,
			DetectedAt:  time.Now(),
		}
		ge.violations = append(ge.violations, *v)
		policy.TriggerCount++
		policy.LastTriggered = time.Now()
		triggered = append(triggered, v)

		// Emit event so other subsystems can react
		if ge.eventBus != nil {
			ge.eventBus.Publish(EngineEvent{
				Type:      EventGuardrailViolation,
				Source:    "guardrail_engine",
				Payload:   v,
				Timestamp: time.Now(),
			})
		}

		// Apply remediation
		if fn, exists := ge.remediators[policy.RemediationAction]; exists {
			if err := fn(ctx, v); err != nil {
				log.Printf("[GUARDRAIL] Remediation %s failed for node %s: %v",
					policy.RemediationAction, nodeID, err)
			} else {
				now := time.Now()
				v.Remediated = true
				v.RemediatedAt = &now
				v.RemediationResult = "success"
				// Update the stored copy
				ge.violations[len(ge.violations)-1] = *v
			}
		}
	}

	// Cap violation log to last 1000 entries
	if len(ge.violations) > 1000 {
		ge.violations = ge.violations[len(ge.violations)-1000:]
	}

	return triggered
}

func (ge *GuardrailEngine) checkCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case "lt":
		return value < threshold
	case "gt":
		return value > threshold
	case "eq":
		return value == threshold
	case "lte":
		return value <= threshold
	case "gte":
		return value >= threshold
	}
	return false
}

// GetViolations returns the most recent `limit` violations.
func (ge *GuardrailEngine) GetViolations(limit int) []PolicyViolation {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	if len(ge.violations) <= limit {
		result := make([]PolicyViolation, len(ge.violations))
		copy(result, ge.violations)
		return result
	}
	src := ge.violations[len(ge.violations)-limit:]
	result := make([]PolicyViolation, len(src))
	copy(result, src)
	return result
}

// ViolationCountForNode returns how many un-remediated violations exist for a node.
func (ge *GuardrailEngine) ViolationCountForNode(nodeID string) int {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	count := 0
	for _, v := range ge.violations {
		if v.NodeID == nodeID && !v.Remediated {
			count++
		}
	}
	return count
}

// RefinePolicy uses observed metric values to auto-tune a rule's threshold.
// A rule that fires more than 50% of observations is considered too sensitive;
// its threshold is moved 10% away from the mean.
func (ge *GuardrailEngine) RefinePolicy(ruleID string, observedValues []float64) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	rule, exists := ge.policies[ruleID]
	if !exists || len(observedValues) < 10 {
		return
	}

	sum := 0.0
	for _, v := range observedValues {
		sum += v
	}
	mean := sum / float64(len(observedValues))
	triggerRate := float64(rule.TriggerCount) / float64(len(observedValues))

	if triggerRate > 0.5 {
		oldThreshold := rule.Threshold
		switch rule.Operator {
		case "lt":
			rule.Threshold = mean * 0.9
		case "gt":
			rule.Threshold = mean * 1.1
		}
		log.Printf("[GUARDRAIL] Auto-refined policy %s: threshold %.4f → %.4f (trigger rate %.2f)",
			ruleID, oldThreshold, rule.Threshold, triggerRate)
	}
}
