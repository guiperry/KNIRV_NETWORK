package wasm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type mockInfrastructure struct {
	mu               sync.Mutex
	nodeStatuses     map[string]string
	cooldowns        map[string]time.Time
	throttled        map[string]float64
	isolated         map[string]uint32
	detached         []string
	terminated       []string
	alerts           []string
}

func newMockInfrastructure() *mockInfrastructure {
	return &mockInfrastructure{
		nodeStatuses: make(map[string]string),
		cooldowns:    make(map[string]time.Time),
		throttled:    make(map[string]float64),
		isolated:     make(map[string]uint32),
	}
}

func (m *mockInfrastructure) SetNodeStatus(nodeID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeStatuses[nodeID] = status
	return nil
}

func (m *mockInfrastructure) TerminateContainer(containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminated = append(m.terminated, containerID)
	return nil
}

func (m *mockInfrastructure) SetNodeCooldown(nodeID string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cooldowns[nodeID] = time.Now().Add(duration)
	return nil
}

func (m *mockInfrastructure) ThrottleNode(nodeID string, rateLimit float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.throttled[nodeID] = rateLimit
	return nil
}

func (m *mockInfrastructure) IsolateProcess(nodeID string, pid uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isolated[nodeID] = pid
	return nil
}

func (m *mockInfrastructure) DetachFromNetwork(nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.detached = append(m.detached, nodeID)
	return nil
}

func (m *mockInfrastructure) EmitAlert(eventType, source, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = append(m.alerts, fmt.Sprintf("%s/%s: %s", eventType, source, message))
}

func TestProductionRemediationSink_ResourceExhaustion(t *testing.T) {
	infra := newMockInfrastructure()
	sink := NewProductionRemediationSink(infra)

	result := &ResolutionResult{
		NodeID:       "node-1",
		DVEID:        "dve-1",
		BadgeID:      "badge-001",
		ErrorClassID: ErrorClassResourceExhaustion,
		Resolved:     true,
	}

	err := sink.DispatchRemediation(context.Background(), result)
	if err != nil {
		t.Fatalf("DispatchRemediation failed: %v", err)
	}

	if status := infra.nodeStatuses["node-1"]; status != "maintenance" {
		t.Errorf("expected node status 'maintenance', got %q", status)
	}
	if _, ok := infra.cooldowns["node-1"]; !ok {
		t.Error("expected node cooldown to be set")
	}
	if len(infra.alerts) == 0 {
		t.Error("expected alert to be emitted")
	}
}

func TestProductionRemediationSink_Latency(t *testing.T) {
	infra := newMockInfrastructure()
	sink := NewProductionRemediationSink(infra)

	result := &ResolutionResult{
		NodeID:       "node-2",
		ErrorClassID: ErrorClassLatency,
		Resolved:     true,
	}

	err := sink.DispatchRemediation(context.Background(), result)
	if err != nil {
		t.Fatalf("DispatchRemediation failed: %v", err)
	}

	if rate, ok := infra.throttled["node-2"]; !ok || rate != 0.5 {
		t.Errorf("expected node throttled with rate 0.5, got rate=%v ok=%v", rate, ok)
	}
	if len(infra.detached) == 0 || infra.detached[0] != "node-2" {
		t.Error("expected node-2 to be detached from network")
	}
}

func TestProductionRemediationSink_Security(t *testing.T) {
	infra := newMockInfrastructure()
	sink := NewProductionRemediationSink(infra)

	result := &ResolutionResult{
		NodeID:       "node-3",
		ErrorClassID: ErrorClassSecurity,
		Resolved:     true,
	}

	err := sink.DispatchRemediation(context.Background(), result)
	if err != nil {
		t.Fatalf("DispatchRemediation failed: %v", err)
	}

	if status := infra.nodeStatuses["node-3"]; status != "isolated" {
		t.Errorf("expected node status 'isolated', got %q", status)
	}
	if len(infra.detached) == 0 || infra.detached[0] != "node-3" {
		t.Error("expected node-3 to be detached from network")
	}
}

func TestProductionRemediationSink_Crash(t *testing.T) {
	infra := newMockInfrastructure()
	sink := NewProductionRemediationSink(infra)

	result := &ResolutionResult{
		NodeID:       "node-4",
		ErrorClassID: ErrorClassCrash,
		Resolved:     true,
	}

	err := sink.DispatchRemediation(context.Background(), result)
	if err != nil {
		t.Fatalf("DispatchRemediation failed: %v", err)
	}

	if status := infra.nodeStatuses["node-4"]; status != "restarting" {
		t.Errorf("expected node status 'restarting', got %q", status)
	}
	if len(infra.terminated) == 0 || infra.terminated[0] != "node-4" {
		t.Error("expected node-4 to be terminated")
	}
}

func TestProductionRemediationSink_UnknownClass(t *testing.T) {
	infra := newMockInfrastructure()
	sink := NewProductionRemediationSink(infra)

	result := &ResolutionResult{
		NodeID:       "node-5",
		ErrorClassID: 99,
		Resolved:     true,
	}

	err := sink.DispatchRemediation(context.Background(), result)
	if err != nil {
		t.Fatalf("DispatchRemediation failed: %v", err)
	}

	if len(infra.alerts) == 0 {
		t.Error("expected alert for unknown error class")
	}
}

func TestProductionRemediationSink_NotResolved(t *testing.T) {
	infra := newMockInfrastructure()
	sink := NewProductionRemediationSink(infra)

	result := &ResolutionResult{
		NodeID:       "node-6",
		ErrorClassID: ErrorClassResourceExhaustion,
		Resolved:     false,
	}

	err := sink.DispatchRemediation(context.Background(), result)
	if err != nil {
		t.Fatalf("DispatchRemediation failed: %v", err)
	}

	if len(infra.alerts) != 0 {
		t.Error("expected no actions for unresolved result")
	}
}

func TestProductionRemediationSink_NilResult(t *testing.T) {
	infra := newMockInfrastructure()
	sink := NewProductionRemediationSink(infra)
	err := sink.DispatchRemediation(context.Background(), nil)
	if err != nil {
		t.Fatalf("DispatchRemediation with nil result should not error: %v", err)
	}
}
