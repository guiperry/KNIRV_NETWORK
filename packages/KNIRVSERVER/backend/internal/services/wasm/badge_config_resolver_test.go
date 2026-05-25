package wasm

import (
	"context"
	"sync"
	"testing"
	"time"
)

type recordingSink struct {
	mu      sync.Mutex
	results []*ResolutionResult
}

func (s *recordingSink) DispatchRemediation(ctx context.Context, result *ResolutionResult) error {
	s.mu.Lock()
	s.results = append(s.results, result)
	s.mu.Unlock()
	return nil
}

func TestBadgeConfigResolver_Resolve_Success(t *testing.T) {
	sink := &recordingSink{}
	r := NewBadgeConfigResolver(sink)
	r.RegisterBadge("badge-001", []string{"memory", "exhaustion"})

	signal := &ResolutionSignal{
		DVEID:   "dve-1",
		NodeID:  "node-1",
		BadgeID: "badge-001",
	}
	result, err := r.Resolve(context.Background(), signal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Resolved {
		t.Error("expected resolved=true for registered badge")
	}
	if result.ErrorClassID != ErrorClassResourceExhaustion {
		t.Errorf("expected class %d, got %d", ErrorClassResourceExhaustion, result.ErrorClassID)
	}

	sink.mu.Lock()
	if len(sink.results) != 1 {
		t.Fatalf("expected 1 sink dispatch, got %d", len(sink.results))
	}
	sink.mu.Unlock()
}

func TestBadgeConfigResolver_Resolve_Unregistered(t *testing.T) {
	r := NewBadgeConfigResolver(nil)
	signal := &ResolutionSignal{
		DVEID:   "dve-2",
		NodeID:  "node-2",
		BadgeID: "unknown-badge",
	}
	result, err := r.Resolve(context.Background(), signal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Resolved {
		t.Error("expected resolved=false for unregistered badge")
	}
	if result.Error == "" {
		t.Error("expected error message for unregistered badge")
	}
}

func TestBadgeConfigResolver_Resolve_NoSink(t *testing.T) {
	r := NewBadgeConfigResolver(nil)
	r.RegisterBadge("badge-002", []string{"latency"})

	signal := &ResolutionSignal{DVEID: "dve-3", NodeID: "node-3", BadgeID: "badge-002"}
	result, err := r.Resolve(context.Background(), signal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Resolved {
		t.Error("expected resolved=true")
	}
	if result.ErrorClassID != ErrorClassLatency {
		t.Errorf("expected class %d, got %d", ErrorClassLatency, result.ErrorClassID)
	}
}

func TestBadgeConfigResolver_RegisterBadge_Overwrite(t *testing.T) {
	r := NewBadgeConfigResolver(nil)
	r.RegisterBadge("badge-003", []string{"crash"})
	r.RegisterBadge("badge-003", []string{"latency"})

	config := r.GetConfig("badge-003")
	if config == nil {
		t.Fatal("expected config to exist")
	}
	if config.ErrorClassID != ErrorClassLatency {
		t.Errorf("expected class %d after overwrite, got %d", ErrorClassLatency, config.ErrorClassID)
	}
}

func TestBadgeConfigResolver_RemoveBadge(t *testing.T) {
	r := NewBadgeConfigResolver(nil)
	r.RegisterBadge("badge-004", []string{"security"})
	r.RemoveBadge("badge-004")

	if config := r.GetConfig("badge-004"); config != nil {
		t.Error("expected config to be nil after removal")
	}
}

func TestClassifyOntologyTags_ResourceExhaustion(t *testing.T) {
	tests := []struct {
		tags  []string
		class uint32
	}{
		{[]string{"memory"}, ErrorClassResourceExhaustion},
		{[]string{"cpu", "exhaustion"}, ErrorClassResourceExhaustion},
		{[]string{"disk_full"}, ErrorClassResourceExhaustion},
		{[]string{"OOM"}, ErrorClassResourceExhaustion},
		{[]string{"resource_pressure"}, ErrorClassResourceExhaustion},
		{[]string{"latency"}, ErrorClassLatency},
		{[]string{"slow_response"}, ErrorClassLatency},
		{[]string{"timeout"}, ErrorClassLatency},
		{[]string{"security_breach"}, ErrorClassSecurity},
		{[]string{"isolate"}, ErrorClassSecurity},
		{[]string{"unauthorized_access"}, ErrorClassSecurity},
		{[]string{"crash"}, ErrorClassCrash},
		{[]string{"fatal_error"}, ErrorClassCrash},
		{[]string{"panic"}, ErrorClassCrash},
		{[]string{"oom_kill"}, ErrorClassResourceExhaustion},
		{[]string{"info"}, ErrorClassNone},
		{[]string{}, ErrorClassNone},
	}
	for _, tt := range tests {
		got := classifyOntologyTags(tt.tags)
		if got != tt.class {
			t.Errorf("classifyOntologyTags(%v) = %d, want %d", tt.tags, got, tt.class)
		}
	}
}

func TestClassifyOntologyTags_Priority(t *testing.T) {
	// First matching tag wins: "latency" matched before "memory"
	class := classifyOntologyTags([]string{"latency", "memory"})
	if class != ErrorClassLatency {
		t.Errorf("expected latency (latency matched first), got %d", class)
	}
}

func TestBadgeConfigResolver_Duration(t *testing.T) {
	r := NewBadgeConfigResolver(&recordingSink{})
	r.RegisterBadge("badge-005", []string{"crash"})

	signal := &ResolutionSignal{BadgeID: "badge-005"}
	result, err := r.Resolve(context.Background(), signal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestBadgeConfigResolver_Timestamp(t *testing.T) {
	r := NewBadgeConfigResolver(&recordingSink{})
	r.RegisterBadge("badge-006", []string{"none"})

	before := time.Now()
	signal := &ResolutionSignal{BadgeID: "badge-006"}
	result, err := r.Resolve(context.Background(), signal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Timestamp.Before(before) {
		t.Error("expected timestamp after start of test")
	}
}
