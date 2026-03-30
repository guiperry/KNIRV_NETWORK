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
	DVEID             string // empty = applies to all DVEs
	Metric            string // "success_rate", "avg_processing_time", "resource_utilization", "violation_count"
	Operator          string // "lt", "gt", "eq", "lte", "gte"
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

// RemediationStatus tracks the state of remediation attempts.
type RemediationStatus struct {
	NodeID      string
	DVEID       string
	RuleID      string
	Attempt     int
	MaxAttempts int
	Cooldown    time.Time
	LastError   string
}

// GuardrailEngine enforces per-DVE policies, records violations, triggers
// automated remediation, and feeds learning data back to refine thresholds.
type GuardrailEngine struct {
	policies           map[string]*PolicyRule
	violations         []PolicyViolation
	remediators        map[string]RemediationFn
	eventBus           *EventBus
	mu                 sync.RWMutex
	remediationStatus  map[string]*RemediationStatus
	escalationPolicies map[string]string
	cooldownDuration   time.Duration
}

// NewGuardrailEngine creates a GuardrailEngine with the built-in default policies
// and remediators pre-registered.
func NewGuardrailEngine(bus *EventBus) *GuardrailEngine {
	ge := &GuardrailEngine{
		policies:           make(map[string]*PolicyRule),
		violations:         make([]PolicyViolation, 0, 64),
		remediators:        make(map[string]RemediationFn),
		eventBus:           bus,
		remediationStatus:  make(map[string]*RemediationStatus),
		escalationPolicies: make(map[string]string),
		cooldownDuration:   5 * time.Minute,
	}
	ge.registerDefaultPolicies()
	ge.registerDefaultRemediators()
	ge.registerEscalationPolicies()
	return ge
}

func (ge *GuardrailEngine) registerEscalationPolicies() {
	ge.escalationPolicies["quarantine_node"] = "drain_node"
	ge.escalationPolicies["redistribute_tasks"] = "scale_resources"
	ge.escalationPolicies["scale_resources"] = "alert_operators"
	ge.escalationPolicies["alert_operators"] = "kernel_isolation"
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
	ge.remediators["quarantine_node"] = ge.quarantineNodeRemediator
	ge.remediators["redistribute_tasks"] = ge.redistributeTasksRemediator
	ge.remediators["scale_resources"] = ge.scaleResourcesRemediator
	ge.remediators["kernel_isolation"] = ge.kernelIsolationRemediator
	ge.remediators["drain_node"] = ge.drainNodeRemediator
	ge.remediators["throttle_requests"] = ge.throttleRequestsRemediator
	ge.remediators["alert_operators"] = ge.alertOperatorsRemediator
	ge.remediators["restart_service"] = ge.restartServiceRemediator
}

func (ge *GuardrailEngine) quarantineNodeRemediator(ctx context.Context, v *PolicyViolation) error {
	log.Printf("[GUARDRAIL] Quarantining node %s (DVE: %s) – rule: %s metric=%.4f",
		v.NodeID, v.DVEID, v.RuleID, v.MetricValue)

	if ge.eventBus != nil {
		ge.eventBus.Publish(EngineEvent{
			Type:      EventNodeOverload,
			Source:    "guardrail_engine",
			Payload:   map[string]interface{}{"node_id": v.NodeID, "dve_id": v.DVEID, "action": "quarantine", "rule": v.RuleID},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (ge *GuardrailEngine) redistributeTasksRemediator(ctx context.Context, v *PolicyViolation) error {
	log.Printf("[GUARDRAIL] Redistributing tasks from slow node %s (DVE: %s) metric=%.2fs",
		v.NodeID, v.DVEID, v.MetricValue)

	if ge.eventBus != nil {
		ge.eventBus.Publish(EngineEvent{
			Type:      EventScalingDecision,
			Source:    "guardrail_engine",
			Payload:   map[string]interface{}{"node_id": v.NodeID, "dve_id": v.DVEID, "action": "redistribute", "metric": v.MetricValue},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (ge *GuardrailEngine) scaleResourcesRemediator(ctx context.Context, v *PolicyViolation) error {
	log.Printf("[GUARDRAIL] Requesting resource scale-up for node %s (DVE: %s) utilization=%.2f",
		v.NodeID, v.DVEID, v.MetricValue)

	if ge.eventBus != nil {
		ge.eventBus.Publish(EngineEvent{
			Type:      EventScalingDecision,
			Source:    "guardrail_engine",
			Payload:   map[string]interface{}{"node_id": v.NodeID, "dve_id": v.DVEID, "action": "scale_up", "utilization": v.MetricValue},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (ge *GuardrailEngine) kernelIsolationRemediator(ctx context.Context, v *PolicyViolation) error {
	log.Printf("[GUARDRAIL] PANIC: Kernel isolation triggered for node %s (DVE: %s) violations=%.0f",
		v.NodeID, v.DVEID, v.MetricValue)

	if ge.eventBus != nil {
		ge.eventBus.Publish(EngineEvent{
			Type:      EventEBPFSecurityAlert,
			Source:    "guardrail_engine",
			Payload:   map[string]interface{}{"node_id": v.NodeID, "dve_id": v.DVEID, "action": "kernel_isolation", "severity": "panic"},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (ge *GuardrailEngine) drainNodeRemediator(ctx context.Context, v *PolicyViolation) error {
	log.Printf("[GUARDRAIL] Draining node %s (DVE: %s) for maintenance – rule: %s",
		v.NodeID, v.DVEID, v.RuleID)

	if ge.eventBus != nil {
		ge.eventBus.Publish(EngineEvent{
			Type:      EventNodeOverload,
			Source:    "guardrail_engine",
			Payload:   map[string]interface{}{"node_id": v.NodeID, "dve_id": v.DVEID, "action": "drain"},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (ge *GuardrailEngine) throttleRequestsRemediator(ctx context.Context, v *PolicyViolation) error {
	log.Printf("[GUARDRAIL] Throttling requests for node %s (DVE: %s) due to high load",
		v.NodeID, v.DVEID)

	return nil
}

func (ge *GuardrailEngine) alertOperatorsRemediator(ctx context.Context, v *PolicyViolation) error {
	log.Printf("[GUARDRAIL] ALERT: Operators notified about violation on node %s (DVE: %s) – %s",
		v.NodeID, v.DVEID, v.RuleID)

	return nil
}

func (ge *GuardrailEngine) restartServiceRemediator(ctx context.Context, v *PolicyViolation) error {
	log.Printf("[GUARDRAIL] Initiating service restart for node %s (DVE: %s)",
		v.NodeID, v.DVEID)

	if ge.eventBus != nil {
		ge.eventBus.Publish(EngineEvent{
			Type:      EventAdaptationRequired,
			Source:    "guardrail_engine",
			Payload:   map[string]interface{}{"node_id": v.NodeID, "dve_id": v.DVEID, "action": "restart"},
			Timestamp: time.Now(),
		})
	}

	return nil
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

		// Apply remediation with retry logic
		if fn, exists := ge.remediators[policy.RemediationAction]; exists {
			statusKey := fmt.Sprintf("%s:%s:%s", nodeID, dveID, policy.ID)

			if status, exists := ge.remediationStatus[statusKey]; exists {
				if time.Now().Before(status.Cooldown) {
					continue
				}
				if status.Attempt >= 3 {
					if escalatedAction, hasEscalation := ge.escalationPolicies[policy.RemediationAction]; hasEscalation {
						log.Printf("[GUARDRAIL] Escalating remediation for node %s from %s to %s",
							nodeID, policy.RemediationAction, escalatedAction)
						if escalationFn, exists := ge.remediators[escalatedAction]; exists {
							if err := escalationFn(ctx, v); err != nil {
								log.Printf("[GUARDRAIL] Escalated remediation %s failed: %v", escalatedAction, err)
							} else {
								now := time.Now()
								v.Remediated = true
								v.RemediatedAt = &now
								v.RemediationResult = fmt.Sprintf("escalated_%s", escalatedAction)
							}
						}
					}
					delete(ge.remediationStatus, statusKey)
					continue
				}
			}

			if err := fn(ctx, v); err != nil {
				log.Printf("[GUARDRAIL] Remediation %s failed for node %s: %v",
					policy.RemediationAction, nodeID, err)
				ge.remediationStatus[statusKey] = &RemediationStatus{
					NodeID:      nodeID,
					DVEID:       dveID,
					RuleID:      policy.ID,
					Attempt:     1,
					MaxAttempts: 3,
					Cooldown:    time.Now().Add(ge.cooldownDuration),
					LastError:   err.Error(),
				}
			} else {
				now := time.Now()
				v.Remediated = true
				v.RemediatedAt = &now
				v.RemediationResult = "success"
				delete(ge.remediationStatus, statusKey)
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

func (ge *GuardrailEngine) GetRemediationStatus(nodeID, dveID, ruleID string) *RemediationStatus {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	statusKey := fmt.Sprintf("%s:%s:%s", nodeID, dveID, ruleID)
	return ge.remediationStatus[statusKey]
}

func (ge *GuardrailEngine) ClearRemediationStatus(nodeID, dveID, ruleID string) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	statusKey := fmt.Sprintf("%s:%s:%s", nodeID, dveID, ruleID)
	delete(ge.remediationStatus, statusKey)
}

func (ge *GuardrailEngine) GetStatistics() map[string]interface{} {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	stats := map[string]interface{}{
		"total_policies":      len(ge.policies),
		"total_violations":    len(ge.violations),
		"active_remediations": len(ge.remediationStatus),
	}

	violationsBySeverity := map[string]int{}
	violationsByDVE := map[string]int{}
	activeRemediations := 0

	for _, v := range ge.violations {
		violationsBySeverity[v.Severity]++
		violationsByDVE[v.DVEID]++
	}

	for _, status := range ge.remediationStatus {
		if time.Now().Before(status.Cooldown) {
			activeRemediations++
		}
	}

	stats["violations_by_severity"] = violationsBySeverity
	stats["violations_by_dve"] = violationsByDVE
	stats["active_remediations"] = activeRemediations

	return stats
}

func (ge *GuardrailEngine) GetActiveViolations() []PolicyViolation {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	var active []PolicyViolation
	for _, v := range ge.violations {
		if !v.Remediated {
			active = append(active, v)
		}
	}
	return active
}

func (ge *GuardrailEngine) ClearResolvedViolations() {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	var remaining []PolicyViolation
	for _, v := range ge.violations {
		if !v.Remediated {
			remaining = append(remaining, v)
		}
	}
	ge.violations = remaining
}

func (ge *GuardrailEngine) SetCooldownDuration(d time.Duration) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.cooldownDuration = d
}

func (ge *GuardrailEngine) RegisterEscalation(from, to string) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.escalationPolicies[from] = to
}

func (ge *GuardrailEngine) DisablePolicy(ruleID string) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	if policy, exists := ge.policies[ruleID]; exists {
		policy.Enabled = false
	}
}

func (ge *GuardrailEngine) EnablePolicy(ruleID string) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	if policy, exists := ge.policies[ruleID]; exists {
		policy.Enabled = true
	}
}
