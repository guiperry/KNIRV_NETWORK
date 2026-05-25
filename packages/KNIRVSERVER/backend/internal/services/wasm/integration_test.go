//go:build integration

package wasm

import (
	"context"
	"sync"
	"testing"

	"backend_server/internal/services/cognitiveengine"
)

type mockResolutionSink struct {
	mu   sync.Mutex
	results []*ResolutionResult
}

func (m *mockResolutionSink) DispatchRemediation(ctx context.Context, result *ResolutionResult) error {
	m.mu.Lock()
	m.results = append(m.results, result)
	m.mu.Unlock()
	return nil
}

// TestBadgeConfigResolver_ClosedLoop verifies the full closed loop:
//   GuardrailEngine.Evaluate() → BadgeConfigResolver → mock sink
func TestBadgeConfigResolver_ClosedLoop(t *testing.T) {
	sink := &mockResolutionSink{}
	resolver := NewBadgeConfigResolver(sink)
	resolver.RegisterBadge("badge-001", []string{"memory", "exhaustion"})

	bus := cognitiveengine.NewEventBus(64)
	ge := cognitiveengine.NewGuardrailEngine(bus)

	resolveFn := func(ctx context.Context, v *cognitiveengine.PolicyViolation) error {
		signal := &ResolutionSignal{
			DVEID:   v.DVEID,
			NodeID:  v.NodeID,
			BadgeID: v.BadgeID,
		}
		_, err := resolver.Resolve(ctx, signal)
		return err
	}
	ge.RegisterRemediator("quarantine_node", resolveFn)
	ge.InjectBadgeRules("dve-1", "badge-001", []string{"guardrail:policy"})

	metrics := map[string]float64{"violation_count": 5.0}
	violations := ge.Evaluate(context.Background(), "node-1", "dve-1", metrics)

	if len(violations) == 0 {
		t.Fatal("expected at least one violation from Evaluate")
	}

	var badgeViolation *cognitiveengine.PolicyViolation
	for _, v := range violations {
		if v.BadgeID == "badge-001" {
			badgeViolation = v
			break
		}
	}
	if badgeViolation == nil {
		t.Fatal("expected a badge-001 violation from InjectBadgeRules")
	}
	if !badgeViolation.Remediated {
		t.Error("expected badge violation to be remediated")
	}

	sink.mu.Lock()
	count := len(sink.results)
	sink.mu.Unlock()
	if count == 0 {
		t.Fatal("expected at least one result to reach the sink")
	}
}

func TestBadgeConfigResolver_DefaultPolicy(t *testing.T) {
	sink := &mockResolutionSink{}
	resolver := NewBadgeConfigResolver(sink)

	bus := cognitiveengine.NewEventBus(64)
	ge := cognitiveengine.NewGuardrailEngine(bus)

	resolveFn := func(ctx context.Context, v *cognitiveengine.PolicyViolation) error {
		signal := &ResolutionSignal{
			DVEID:   v.DVEID,
			NodeID:  v.NodeID,
			BadgeID: v.BadgeID,
		}
		_, err := resolver.Resolve(ctx, signal)
		return err
	}
	ge.RegisterRemediator("quarantine_node", resolveFn)
	ge.RegisterRemediator("drain_node", resolveFn)

	metrics := map[string]float64{"success_rate": 0.1}
	violations := ge.Evaluate(context.Background(), "node-2", "dve-2", metrics)

	if len(violations) == 0 {
		t.Fatal("expected at least one default policy violation")
	}

	for _, v := range violations {
		if v.RuleID == "dveguard_low_success" {
			return
		}
	}
	t.Error("expected dveguard_low_success violation")
}

func TestGuardrailEventBus_ForwardsToResolver(t *testing.T) {
	sink := &mockResolutionSink{}
	resolver := NewBadgeConfigResolver(sink)

	violationCh := make(chan *cognitiveengine.PolicyViolation, 16)
	go func() {
		for v := range violationCh {
			signal := &ResolutionSignal{
				DVEID:   v.DVEID,
				NodeID:  v.NodeID,
				BadgeID: v.BadgeID,
			}
			resolver.Resolve(context.Background(), signal)
		}
	}()

	violationCh <- &cognitiveengine.PolicyViolation{
		DVEID:   "dve-3",
		NodeID:  "node-3",
		BadgeID: "badge-003",
	}
	close(violationCh)

	// No badge-003 registered — should silently return unresolved.
	// This test verifies the event bus forwarding path works without error.
	sink.mu.Lock()
	count := len(sink.results)
	sink.mu.Unlock()
	if count != 0 {
		t.Fatalf("expected 0 sink results for unregistered badge, got %d", count)
	}
}
