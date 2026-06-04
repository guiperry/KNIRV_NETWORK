package reliability

import (
	"fmt"
	"sync"
	"time"
)

type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"
	CircuitStateOpen     CircuitState = "open"
	CircuitStateHalfOpen CircuitState = "half_open"
)

type KillSwitchState string

const (
	KillSwitchStateArmed    KillSwitchState = "armed"
	KillSwitchStateTripped  KillSwitchState = "tripped"
	KillSwitchStateDisarmed KillSwitchState = "disarmed"
)

type EscalationLevel string

const (
	EscalationLevelNone      EscalationLevel = "none"
	EscalationLevelWarning   EscalationLevel = "warning"
	EscalationLevelCritical  EscalationLevel = "critical"
	EscalationLevelShutdown  EscalationLevel = "shutdown"
)

type CircuitBreaker struct {
	State           CircuitState `json:"state"`
	FailureCount    int          `json:"failure_count"`
	SuccessCount    int          `json:"success_count"`
	Threshold       int          `json:"threshold"`
	HalfOpenMax     int          `json:"half_open_max"`
	RecoveryTimeout time.Duration `json:"recovery_timeout"`
	OpenedAt        time.Time    `json:"opened_at,omitempty"`
	LastStateChange time.Time    `json:"last_state_change"`
	mu              sync.RWMutex
}

type ErrorBudget struct {
	TotalBudget    float64       `json:"total_budget"`
	Remaining      float64       `json:"remaining"`
	Consumed       float64       `json:"consumed"`
	Period         time.Duration `json:"period"`
	PeriodStart    time.Time     `json:"period_start"`
	Errors         int           `json:"errors"`
	TotalRequests  int           `json:"total_requests"`
	mu             sync.RWMutex
}

type EscalationPolicy struct {
	WarningThreshold   float64 `json:"warning_threshold"`
	CriticalThreshold  float64 `json:"critical_threshold"`
	ShutdownThreshold  float64 `json:"shutdown_threshold"`
	CooldownPeriod     time.Duration `json:"cooldown_period"`
	AutoReset          bool    `json:"auto_reset"`
}

type KillSwitch struct {
	State          KillSwitchState `json:"state"`
	AgentID        string          `json:"agent_id"`
	NodeID         string          `json:"node_id"`
	Reason         string          `json:"reason,omitempty"`
	TriggeredAt    time.Time       `json:"triggered_at,omitempty"`
	TrippedBy      string          `json:"tripped_by,omitempty"`
	AutoResetAfter time.Duration   `json:"auto_reset_after,omitempty"`
	mu             sync.RWMutex
}

type BreachEvent struct {
	ID          string          `json:"id"`
	AgentID     string          `json:"agent_id"`
	NodeID      string          `json:"node_id"`
	Level       EscalationLevel `json:"level"`
	Reason      string          `json:"reason"`
	MetricName  string          `json:"metric_name,omitempty"`
	MetricValue float64         `json:"metric_value,omitempty"`
	Threshold   float64         `json:"threshold,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
	Resolved    bool            `json:"resolved"`
	ResolvedAt  *time.Time      `json:"resolved_at,omitempty"`
}

type ReliabilityController struct {
	mu            sync.RWMutex
	breakers      map[string]*CircuitBreaker
	budgets       map[string]*ErrorBudget
	killSwitches  map[string]*KillSwitch
	escalations   map[string]*EscalationPolicy
	breachEvents  []*BreachEvent
	defaultBreakerConfig CircuitBreaker
	defaultBudgetConfig  ErrorBudget
	defaultEscalation    EscalationPolicy
}

func NewReliabilityController() *ReliabilityController {
	return &ReliabilityController{
		breakers:     make(map[string]*CircuitBreaker),
		budgets:      make(map[string]*ErrorBudget),
		killSwitches: make(map[string]*KillSwitch),
		escalations:  make(map[string]*EscalationPolicy),
		breachEvents: make([]*BreachEvent, 0),
		defaultBreakerConfig: CircuitBreaker{
			State:           CircuitStateClosed,
			Threshold:       5,
			HalfOpenMax:     3,
			RecoveryTimeout: 30 * time.Second,
			LastStateChange: time.Now(),
		},
		defaultBudgetConfig: ErrorBudget{
			TotalBudget: 0.1,
			Period:     24 * time.Hour,
			PeriodStart: time.Now(),
		},
		defaultEscalation: EscalationPolicy{
			WarningThreshold:  0.5,
			CriticalThreshold: 0.75,
			ShutdownThreshold: 0.95,
			CooldownPeriod:    5 * time.Minute,
			AutoReset:         true,
		},
	}
}

func (rc *ReliabilityController) GetOrCreateBreaker(breakerID string) *CircuitBreaker {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	cb, ok := rc.breakers[breakerID]
	if !ok {
		cb = &CircuitBreaker{
			State:           CircuitStateClosed,
			Threshold:       rc.defaultBreakerConfig.Threshold,
			HalfOpenMax:     rc.defaultBreakerConfig.HalfOpenMax,
			RecoveryTimeout: rc.defaultBreakerConfig.RecoveryTimeout,
			LastStateChange: time.Now(),
		}
		rc.breakers[breakerID] = cb
	}
	return cb
}

func (rc *ReliabilityController) transitionToHalfOpenIfExpired(cb *CircuitBreaker) {
	if cb.State == CircuitStateOpen && time.Since(cb.OpenedAt) >= cb.RecoveryTimeout {
		cb.State = CircuitStateHalfOpen
		cb.SuccessCount = 0
		cb.FailureCount = 0
		cb.LastStateChange = time.Now()
	}
}

func (rc *ReliabilityController) RecordSuccess(breakerID string) {
	cb := rc.GetOrCreateBreaker(breakerID)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	rc.transitionToHalfOpenIfExpired(cb)

	switch cb.State {
	case CircuitStateClosed:
		cb.SuccessCount++
		cb.FailureCount = 0
	case CircuitStateHalfOpen:
		cb.SuccessCount++
		cb.FailureCount = 0
		if cb.SuccessCount >= cb.HalfOpenMax {
			cb.State = CircuitStateClosed
			cb.LastStateChange = time.Now()
			cb.SuccessCount = 0
		}
	}
}

func (rc *ReliabilityController) RecordFailure(breakerID string) {
	cb := rc.GetOrCreateBreaker(breakerID)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	rc.transitionToHalfOpenIfExpired(cb)

	switch cb.State {
	case CircuitStateClosed:
		cb.FailureCount++
		cb.SuccessCount = 0
		if cb.FailureCount >= cb.Threshold {
			cb.State = CircuitStateOpen
			cb.OpenedAt = time.Now()
			cb.LastStateChange = time.Now()
		}
	case CircuitStateHalfOpen:
		cb.State = CircuitStateOpen
		cb.OpenedAt = time.Now()
		cb.LastStateChange = time.Now()
		cb.FailureCount++
		cb.SuccessCount = 0
	}
}

func (rc *ReliabilityController) IsRequestAllowed(breakerID string) bool {
	cb := rc.GetOrCreateBreaker(breakerID)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	rc.transitionToHalfOpenIfExpired(cb)

	switch cb.State {
	case CircuitStateClosed:
		return true
	case CircuitStateOpen:
		return false
	case CircuitStateHalfOpen:
		return cb.SuccessCount < cb.HalfOpenMax
	default:
		return false
	}
}

func (rc *ReliabilityController) GetBreakerState(breakerID string) CircuitState {
	cb := rc.GetOrCreateBreaker(breakerID)
	cb.mu.Lock()
	defer cb.mu.Unlock()

	rc.transitionToHalfOpenIfExpired(cb)
	return cb.State
}

func (rc *ReliabilityController) ResetBreaker(breakerID string) {
	cb := rc.GetOrCreateBreaker(breakerID)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.State = CircuitStateClosed
	cb.FailureCount = 0
	cb.SuccessCount = 0
	cb.LastStateChange = time.Now()
}

func (rc *ReliabilityController) GetOrCreateBudget(budgetID string) *ErrorBudget {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	b, ok := rc.budgets[budgetID]
	if !ok {
		b = &ErrorBudget{
			TotalBudget: rc.defaultBudgetConfig.TotalBudget,
			Period:      rc.defaultBudgetConfig.Period,
			PeriodStart: time.Now(),
		}
		rc.budgets[budgetID] = b
	}
	return b
}

func (rc *ReliabilityController) ConsumeBudget(budgetID string) (float64, bool) {
	b := rc.GetOrCreateBudget(budgetID)
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Remaining == 0 && b.Consumed == 0 && b.Errors == 0 {
		b.Remaining = b.TotalBudget
	}

	if time.Since(b.PeriodStart) >= b.Period {
		b.Remaining = b.TotalBudget
		b.Consumed = 0
		b.Errors = 0
		b.TotalRequests = 0
		b.PeriodStart = time.Now()
	}

	b.TotalRequests++

	exhausted := b.Remaining <= 0
	return b.Remaining, !exhausted
}

func (rc *ReliabilityController) RecordBudgetError(budgetID string) {
	b := rc.GetOrCreateBudget(budgetID)
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Remaining == 0 && b.Consumed == 0 && b.Errors == 0 {
		b.Remaining = b.TotalBudget
	}

	b.Errors++
	b.Consumed = float64(b.Errors) / float64(b.TotalRequests)
	if b.TotalRequests == 0 {
		b.TotalRequests = 1
		b.Consumed = 1.0
	}
	b.Remaining = b.TotalBudget - b.Consumed
	if b.Remaining < 0 {
		b.Remaining = 0
	}
}

func (rc *ReliabilityController) GetBudgetStatus(budgetID string) map[string]interface{} {
	b := rc.GetOrCreateBudget(budgetID)
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Remaining == 0 && b.Consumed == 0 && b.Errors == 0 {
		b.Remaining = b.TotalBudget
	}

	return map[string]interface{}{
		"total_budget":   b.TotalBudget,
		"remaining":      b.Remaining,
		"consumed":       b.Consumed,
		"errors":         b.Errors,
		"total_requests": b.TotalRequests,
		"period_start":   b.PeriodStart,
		"period":         b.Period.String(),
	}
}

func (rc *ReliabilityController) SetupEscalation(nodeID string, policy EscalationPolicy) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.escalations[nodeID] = &policy
}

func (rc *ReliabilityController) EvaluateEscalation(nodeID string, metricName string, value float64) EscalationLevel {
	rc.mu.RLock()
	policy, ok := rc.escalations[nodeID]
	rc.mu.RUnlock()

	if !ok {
		policy = &rc.defaultEscalation
	}

	switch {
	case value >= policy.ShutdownThreshold:
		return EscalationLevelShutdown
	case value >= policy.CriticalThreshold:
		return EscalationLevelCritical
	case value >= policy.WarningThreshold:
		return EscalationLevelWarning
	default:
		return EscalationLevelNone
	}
}

func (rc *ReliabilityController) ArmKillSwitch(agentID, nodeID string, autoReset time.Duration) *KillSwitch {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	ks := &KillSwitch{
		State:          KillSwitchStateArmed,
		AgentID:        agentID,
		NodeID:         nodeID,
		AutoResetAfter: autoReset,
	}
	rc.killSwitches[rc.killKey(agentID, nodeID)] = ks
	return ks
}

func (rc *ReliabilityController) TripKillSwitch(agentID, nodeID, reason, trippedBy string) (*BreachEvent, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	key := rc.killKey(agentID, nodeID)
	ks, ok := rc.killSwitches[key]
	if !ok {
		return nil, fmt.Errorf("no kill switch found for agent %s on node %s", agentID, nodeID)
	}

	if ks.State == KillSwitchStateTripped {
		return nil, fmt.Errorf("kill switch already tripped for agent %s", agentID)
	}

	ks.State = KillSwitchStateTripped
	ks.Reason = reason
	ks.TrippedBy = trippedBy
	ks.TriggeredAt = time.Now()

	event := &BreachEvent{
		ID:        fmt.Sprintf("breach-%d", time.Now().UnixNano()),
		AgentID:   agentID,
		NodeID:    nodeID,
		Level:     EscalationLevelShutdown,
		Reason:    reason,
		Timestamp: time.Now(),
	}
	rc.breachEvents = append(rc.breachEvents, event)
	return event, nil
}

func (rc *ReliabilityController) DisarmKillSwitch(agentID, nodeID string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	key := rc.killKey(agentID, nodeID)
	ks, ok := rc.killSwitches[key]
	if !ok {
		return fmt.Errorf("no kill switch found for agent %s on node %s", agentID, nodeID)
	}

	ks.State = KillSwitchStateDisarmed
	return nil
}

func (rc *ReliabilityController) GetKillSwitch(agentID, nodeID string) (*KillSwitch, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	ks, ok := rc.killSwitches[rc.killKey(agentID, nodeID)]
	return ks, ok
}

func (rc *ReliabilityController) RecordBreachEvent(event *BreachEvent) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.breachEvents = append(rc.breachEvents, event)
}

func (rc *ReliabilityController) ListBreachEvents(agentID string, limit int) []*BreachEvent {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	var result []*BreachEvent
	for _, e := range rc.breachEvents {
		if agentID == "" || e.AgentID == agentID {
			result = append(result, e)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

func (rc *ReliabilityController) ResolveBreachEvent(eventID string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for _, e := range rc.breachEvents {
		if e.ID == eventID {
			e.Resolved = true
			now := time.Now()
			e.ResolvedAt = &now
			return nil
		}
	}
	return fmt.Errorf("breach event not found: %s", eventID)
}

func (rc *ReliabilityController) GetStatistics() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	openBreakers := 0
	totalBreakers := len(rc.breakers)
	for _, cb := range rc.breakers {
		cb.mu.RLock()
		if cb.State == CircuitStateOpen {
			openBreakers++
		}
		cb.mu.RUnlock()
	}

	totalKillSwitches := len(rc.killSwitches)
	tripped := 0
	for _, ks := range rc.killSwitches {
		ks.mu.RLock()
		if ks.State == KillSwitchStateTripped {
			tripped++
		}
		ks.mu.RUnlock()
	}

	unresolvedBreaches := 0
	for _, e := range rc.breachEvents {
		if !e.Resolved {
			unresolvedBreaches++
		}
	}

	return map[string]interface{}{
		"total_circuit_breakers": totalBreakers,
		"open_circuit_breakers":  openBreakers,
		"total_error_budgets":    len(rc.budgets),
		"total_kill_switches":    totalKillSwitches,
		"tripped_kill_switches":  tripped,
		"total_breach_events":    len(rc.breachEvents),
		"unresolved_breaches":    unresolvedBreaches,
	}
}

func (rc *ReliabilityController) killKey(agentID, nodeID string) string {
	return agentID + "@" + nodeID
}

func (rc *ReliabilityController) ListBreakers() map[string]CircuitState {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	result := make(map[string]CircuitState, len(rc.breakers))
	for id, cb := range rc.breakers {
		cb.mu.RLock()
		state := cb.State
		if state == CircuitStateOpen && time.Since(cb.OpenedAt) >= cb.RecoveryTimeout {
			state = CircuitStateHalfOpen
		}
		result[id] = state
		cb.mu.RUnlock()
	}
	return result
}

func (rc *ReliabilityController) ListKillSwitches() []*KillSwitch {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	result := make([]*KillSwitch, 0, len(rc.killSwitches))
	for _, ks := range rc.killSwitches {
		result = append(result, ks)
	}
	return result
}
