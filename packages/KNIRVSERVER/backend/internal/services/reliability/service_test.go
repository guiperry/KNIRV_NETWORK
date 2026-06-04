package reliability

import (
	"testing"
	"time"
)

func TestNewReliabilityController(t *testing.T) {
	rc := NewReliabilityController()
	if rc == nil {
		t.Fatal("Expected non-nil ReliabilityController")
	}
	if rc.breakers == nil {
		t.Error("Expected breakers map to be initialized")
	}
	if rc.budgets == nil {
		t.Error("Expected budgets map to be initialized")
	}
	if rc.killSwitches == nil {
		t.Error("Expected killSwitches map to be initialized")
	}
}

func TestCircuitBreakerClosedState(t *testing.T) {
	rc := NewReliabilityController()
	if !rc.IsRequestAllowed("test-breaker") {
		t.Error("Expected request allowed when breaker is closed")
	}
}

func TestCircuitBreakerTripsOnFailures(t *testing.T) {
	rc := NewReliabilityController()
	breaker := rc.GetOrCreateBreaker("test-breaker")
	breaker.Threshold = 3

	for i := 0; i < 3; i++ {
		rc.RecordFailure("test-breaker")
	}

	state := rc.GetBreakerState("test-breaker")
	if state != CircuitStateOpen {
		t.Errorf("Expected breaker to be open after 3 failures, got %s", state)
	}

	if rc.IsRequestAllowed("test-breaker") {
		t.Error("Expected request blocked when breaker is open")
	}
}

func TestCircuitBreakerHalfOpenRecovery(t *testing.T) {
	rc := NewReliabilityController()
	breaker := rc.GetOrCreateBreaker("test-breaker")
	breaker.Threshold = 2
	breaker.RecoveryTimeout = 1 * time.Millisecond

	rc.RecordFailure("test-breaker")
	rc.RecordFailure("test-breaker")

	if rc.IsRequestAllowed("test-breaker") {
		t.Error("Expected request blocked when breaker is open")
	}

	time.Sleep(2 * time.Millisecond)

	state := rc.GetBreakerState("test-breaker")
	if state != CircuitStateHalfOpen {
		t.Errorf("Expected breaker to be half-open after timeout, got %s", state)
	}
}

func TestCircuitBreakerClosesAfterHalfOpenSuccesses(t *testing.T) {
	rc := NewReliabilityController()
	breaker := rc.GetOrCreateBreaker("test-breaker")
	breaker.Threshold = 2
	breaker.HalfOpenMax = 2
	breaker.RecoveryTimeout = 1 * time.Millisecond

	rc.RecordFailure("test-breaker")
	rc.RecordFailure("test-breaker")

	time.Sleep(2 * time.Millisecond)

	rc.RecordSuccess("test-breaker")
	rc.RecordSuccess("test-breaker")

	state := rc.GetBreakerState("test-breaker")
	if state != CircuitStateClosed {
		t.Errorf("Expected breaker to be closed after half-open successes, got %s", state)
	}
}

func TestCircuitBreakerReopensOnHalfOpenFailure(t *testing.T) {
	rc := NewReliabilityController()
	breaker := rc.GetOrCreateBreaker("test-breaker")
	breaker.Threshold = 2
	breaker.RecoveryTimeout = 1 * time.Millisecond

	rc.RecordFailure("test-breaker")
	rc.RecordFailure("test-breaker")

	time.Sleep(2 * time.Millisecond)

	rc.RecordFailure("test-breaker")

	state := rc.GetBreakerState("test-breaker")
	if state != CircuitStateOpen {
		t.Errorf("Expected breaker to be open after half-open failure, got %s", state)
	}
}

func TestResetBreaker(t *testing.T) {
	rc := NewReliabilityController()
	breaker := rc.GetOrCreateBreaker("test-breaker")
	breaker.Threshold = 1
	rc.RecordFailure("test-breaker")

	rc.ResetBreaker("test-breaker")

	state := rc.GetBreakerState("test-breaker")
	if state != CircuitStateClosed {
		t.Errorf("Expected breaker to be closed after reset, got %s", state)
	}
	if !rc.IsRequestAllowed("test-breaker") {
		t.Error("Expected request allowed after reset")
	}
}

func TestErrorBudgetConsume(t *testing.T) {
	rc := NewReliabilityController()
	budget := rc.GetOrCreateBudget("test-budget")
	budget.TotalBudget = 0.5

	remaining, ok := rc.ConsumeBudget("test-budget")
	if !ok {
		t.Error("Expected budget to allow requests")
	}
	if remaining != 0.5 {
		t.Errorf("Expected remaining 0.5, got %f", remaining)
	}
}

func TestErrorBudgetRecordsErrors(t *testing.T) {
	rc := NewReliabilityController()
	budget := rc.GetOrCreateBudget("test-budget")
	budget.TotalBudget = 0.1

	rc.ConsumeBudget("test-budget")
	rc.ConsumeBudget("test-budget")
	rc.ConsumeBudget("test-budget")

	rc.RecordBudgetError("test-budget")

	status := rc.GetBudgetStatus("test-budget")
	if status["errors"].(int) != 1 {
		t.Errorf("Expected 1 error, got %d", status["errors"])
	}
}

func TestErrorBudgetExhaustion(t *testing.T) {
	rc := NewReliabilityController()
	budget := rc.GetOrCreateBudget("test-budget")
	budget.TotalBudget = 0.01

	rc.ConsumeBudget("test-budget")
	rc.RecordBudgetError("test-budget")
	rc.ConsumeBudget("test-budget")
	rc.RecordBudgetError("test-budget")

	status := rc.GetBudgetStatus("test-budget")
	remaining := status["remaining"].(float64)
	if remaining > 0.01 {
		t.Errorf("Expected remaining <= 0.01 after errors, got %f", remaining)
	}
}

func TestErrorBudgetResetsAfterPeriod(t *testing.T) {
	rc := NewReliabilityController()
	budget := rc.GetOrCreateBudget("test-budget")
	budget.TotalBudget = 0.1
	budget.Period = 1 * time.Millisecond

	rc.RecordBudgetError("test-budget")
	rc.RecordBudgetError("test-budget")

	time.Sleep(2 * time.Millisecond)

	remaining, ok := rc.ConsumeBudget("test-budget")
	if !ok {
		t.Error("Expected budget to allow after period reset")
	}
	if remaining != 0.1 {
		t.Errorf("Expected remaining 0.1 after reset, got %f", remaining)
	}
}

func TestKillSwitchArmTripDisarm(t *testing.T) {
	rc := NewReliabilityController()
	rc.ArmKillSwitch("agent-1", "node-1", 0)

	ks, ok := rc.GetKillSwitch("agent-1", "node-1")
	if !ok {
		t.Fatal("Expected to find kill switch")
	}
	if ks.State != KillSwitchStateArmed {
		t.Errorf("Expected state armed, got %s", ks.State)
	}

	event, err := rc.TripKillSwitch("agent-1", "node-1", "resource exhaustion", "guardrail-manager")
	if err != nil {
		t.Errorf("TripKillSwitch failed: %v", err)
	}
	if event == nil {
		t.Fatal("Expected non-nil breach event")
	}
	if event.Level != EscalationLevelShutdown {
		t.Errorf("Expected level shutdown, got %s", event.Level)
	}

	ks, _ = rc.GetKillSwitch("agent-1", "node-1")
	if ks.State != KillSwitchStateTripped {
		t.Errorf("Expected state tripped, got %s", ks.State)
	}

	err = rc.DisarmKillSwitch("agent-1", "node-1")
	if err != nil {
		t.Errorf("DisarmKillSwitch failed: %v", err)
	}

	ks, _ = rc.GetKillSwitch("agent-1", "node-1")
	if ks.State != KillSwitchStateDisarmed {
		t.Errorf("Expected state disarmed, got %s", ks.State)
	}
}

func TestKillSwitchDoubleTrip(t *testing.T) {
	rc := NewReliabilityController()
	rc.ArmKillSwitch("agent-1", "node-1", 0)

	rc.TripKillSwitch("agent-1", "node-1", "first", "system")
	_, err := rc.TripKillSwitch("agent-1", "node-1", "second", "system")
	if err == nil {
		t.Error("Expected error on double trip")
	}
}

func TestKillSwitchNotFound(t *testing.T) {
	rc := NewReliabilityController()
	_, err := rc.TripKillSwitch("no-agent", "no-node", "reason", "system")
	if err == nil {
		t.Error("Expected error for nonexistent kill switch")
	}
}

func TestEscalationEvaluation(t *testing.T) {
	rc := NewReliabilityController()
	rc.SetupEscalation("node-1", EscalationPolicy{
		WarningThreshold:  0.5,
		CriticalThreshold: 0.75,
		ShutdownThreshold: 0.95,
	})

	tests := []struct {
		value    float64
		expected EscalationLevel
	}{
		{0.3, EscalationLevelNone},
		{0.6, EscalationLevelWarning},
		{0.8, EscalationLevelCritical},
		{0.98, EscalationLevelShutdown},
	}
	for _, tc := range tests {
		level := rc.EvaluateEscalation("node-1", "error_rate", tc.value)
		if level != tc.expected {
			t.Errorf("Expected %s for value %f, got %s", tc.expected, tc.value, level)
		}
	}
}

func TestEscalationDefaultPolicy(t *testing.T) {
	rc := NewReliabilityController()
	level := rc.EvaluateEscalation("unknown-node", "error_rate", 0.3)
	if level != EscalationLevelNone {
		t.Errorf("Expected none for low value with default policy, got %s", level)
	}

	level = rc.EvaluateEscalation("unknown-node", "error_rate", 0.6)
	if level != EscalationLevelWarning {
		t.Errorf("Expected warning for 0.6 with default policy, got %s", level)
	}

	level = rc.EvaluateEscalation("unknown-node", "error_rate", 0.96)
	if level != EscalationLevelShutdown {
		t.Errorf("Expected shutdown for 0.96 with default policy, got %s", level)
	}
}

func TestBreachEvents(t *testing.T) {
	rc := NewReliabilityController()
	rc.RecordBreachEvent(&BreachEvent{
		ID:      "b1",
		AgentID: "agent-1",
		NodeID:  "node-1",
		Level:   EscalationLevelCritical,
		Reason:  "memory leak",
	})
	rc.RecordBreachEvent(&BreachEvent{
		ID:      "b2",
		AgentID: "agent-2",
		NodeID:  "node-1",
		Level:   EscalationLevelWarning,
		Reason:  "high latency",
	})

	events := rc.ListBreachEvents("agent-1", 0)
	if len(events) != 1 {
		t.Errorf("Expected 1 breach event for agent-1, got %d", len(events))
	}

	events = rc.ListBreachEvents("", 0)
	if len(events) != 2 {
		t.Errorf("Expected 2 breach events total, got %d", len(events))
	}

	err := rc.ResolveBreachEvent("b1")
	if err != nil {
		t.Errorf("ResolveBreachEvent failed: %v", err)
	}
	if !events[0].Resolved {
		t.Error("Expected event b1 to be resolved")
	}
}

func TestResolveBreachEventNotFound(t *testing.T) {
	rc := NewReliabilityController()
	err := rc.ResolveBreachEvent("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent breach event")
	}
}

func TestGetStatistics(t *testing.T) {
	rc := NewReliabilityController()
	rc.GetOrCreateBreaker("breaker-1")
	rc.GetOrCreateBreaker("breaker-2")
	rc.GetOrCreateBudget("budget-1")
	rc.ArmKillSwitch("agent-1", "node-1", 0)

	stats := rc.GetStatistics()
	if stats["total_circuit_breakers"].(int) != 2 {
		t.Errorf("Expected 2 breakers, got %d", stats["total_circuit_breakers"])
	}
	if stats["total_error_budgets"].(int) != 1 {
		t.Errorf("Expected 1 budget, got %d", stats["total_error_budgets"])
	}
	if stats["total_kill_switches"].(int) != 1 {
		t.Errorf("Expected 1 kill switch, got %d", stats["total_kill_switches"])
	}
}

func TestListBreakers(t *testing.T) {
	rc := NewReliabilityController()
	rc.GetOrCreateBreaker("breaker-1")
	rc.GetOrCreateBreaker("breaker-2")

	breakers := rc.ListBreakers()
	if len(breakers) != 2 {
		t.Errorf("Expected 2 breakers, got %d", len(breakers))
	}
}

func TestListKillSwitches(t *testing.T) {
	rc := NewReliabilityController()
	rc.ArmKillSwitch("agent-1", "node-1", 0)
	rc.ArmKillSwitch("agent-2", "node-1", time.Hour)

	switches := rc.ListKillSwitches()
	if len(switches) != 2 {
		t.Errorf("Expected 2 kill switches, got %d", len(switches))
	}
}

func TestDefaultConfigs(t *testing.T) {
	rc := NewReliabilityController()
	if rc.defaultBreakerConfig.Threshold != 5 {
		t.Errorf("Expected default threshold 5, got %d", rc.defaultBreakerConfig.Threshold)
	}
	if rc.defaultBreakerConfig.RecoveryTimeout != 30*time.Second {
		t.Errorf("Expected default recovery timeout 30s, got %v", rc.defaultBreakerConfig.RecoveryTimeout)
	}
	if rc.defaultBudgetConfig.TotalBudget != 0.1 {
		t.Errorf("Expected default budget 0.1, got %f", rc.defaultBudgetConfig.TotalBudget)
	}
	if rc.defaultEscalation.WarningThreshold != 0.5 {
		t.Errorf("Expected default warning threshold 0.5, got %f", rc.defaultEscalation.WarningThreshold)
	}
}
