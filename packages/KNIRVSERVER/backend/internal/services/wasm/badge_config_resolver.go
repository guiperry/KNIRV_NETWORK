package wasm

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

// ErrorClass constants classify the type of error a badge's ontology triggers.
const (
	ErrorClassNone               uint32 = 0
	ErrorClassResourceExhaustion uint32 = 1
	ErrorClassLatency            uint32 = 2
	ErrorClassSecurity           uint32 = 3
	ErrorClassCrash              uint32 = 4
)

// ResolutionSignal is produced when a policy violation needs resolution.
type ResolutionSignal struct {
	DVEID     string            `json:"dve_id"`
	NodeID    string            `json:"node_id"`
	BadgeID   string            `json:"badge_id"`
	Tag       string            `json:"tag"`
	ErrorType string            `json:"error_type"`
	SyscallID uint32            `json:"syscall_id"`
	PID       uint32            `json:"pid"`
	Context   map[string]string `json:"context,omitempty"`
}

// ResolutionResult captures the outcome of a resolution attempt.
type ResolutionResult struct {
	BadgeID      string        `json:"badge_id"`
	DVEID        string        `json:"dve_id"`
	NodeID       string        `json:"node_id"`
	Tag          string        `json:"tag"`
	Resolved     bool          `json:"resolved"`
	ErrorClassID uint32        `json:"error_class_id"`
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration_ms"`
	Timestamp    time.Time     `json:"timestamp"`
}

// RemediationSink receives resolution results and dispatches remediation actions.
type RemediationSink interface {
	DispatchRemediation(ctx context.Context, result *ResolutionResult) error
}

// BadgeResolutionConfig defines how a badge's ontology tags map to a
// resolution error class and whether it auto-resolves.
type BadgeResolutionConfig struct {
	BadgeID      string   `json:"badge_id"`
	ErrorClassID uint32   `json:"error_class_id"`
	AutoResolve  bool     `json:"auto_resolve"`
	OntologyTags []string `json:"ontology_tags,omitempty"`
}

// BadgeConfigResolver resolves policy violations into resolution results
// using a simple in-memory badge config map — no WASM modules needed.
type BadgeConfigResolver struct {
	configs map[string]*BadgeResolutionConfig
	sink    RemediationSink
	mu      sync.RWMutex
}

// NewBadgeConfigResolver creates a resolver with the given remediation sink.
func NewBadgeConfigResolver(sink RemediationSink) *BadgeConfigResolver {
	return &BadgeConfigResolver{
		configs: make(map[string]*BadgeResolutionConfig),
		sink:    sink,
	}
}

// Resolve looks up the badge config and returns a resolution result.
// If the badge is configured for auto-resolve, the result is dispatched
// to the remediation sink.
func (r *BadgeConfigResolver) Resolve(ctx context.Context, signal *ResolutionSignal) (*ResolutionResult, error) {
	start := time.Now()

	r.mu.RLock()
	config, exists := r.configs[signal.BadgeID]
	r.mu.RUnlock()

	result := &ResolutionResult{
		BadgeID:   signal.BadgeID,
		DVEID:     signal.DVEID,
		NodeID:    signal.NodeID,
		Tag:       signal.Tag,
		Timestamp: time.Now(),
	}

	if !exists {
		result.Error = "no badge config registered for " + signal.BadgeID
		result.Duration = time.Since(start)
		return result, nil
	}

	result.ErrorClassID = config.ErrorClassID
	result.Resolved = config.AutoResolve
	result.Duration = time.Since(start)

	if result.Resolved && r.sink != nil {
		if err := r.sink.DispatchRemediation(ctx, result); err != nil {
			log.Printf("[badge-config-resolver] remediation dispatch error: %v", err)
		}
	}

	return result, nil
}

// RegisterBadge creates or updates a badge config from ontology tags.
func (r *BadgeConfigResolver) RegisterBadge(badgeID string, ontologyTags []string) error {
	classID := classifyOntologyTags(ontologyTags)

	r.mu.Lock()
	r.configs[badgeID] = &BadgeResolutionConfig{
		BadgeID:      badgeID,
		ErrorClassID: classID,
		AutoResolve:  true,
		OntologyTags: ontologyTags,
	}
	r.mu.Unlock()

	log.Printf("[badge-config-resolver] registered badge %s: class=%d tags=%v",
		badgeID, classID, ontologyTags)
	return nil
}

// RemoveBadge deletes a badge config.
func (r *BadgeConfigResolver) RemoveBadge(badgeID string) {
	r.mu.Lock()
	delete(r.configs, badgeID)
	r.mu.Unlock()
	log.Printf("[badge-config-resolver] removed badge %s", badgeID)
}

// GetConfig returns the config for a badge, or nil if not registered.
func (r *BadgeConfigResolver) GetConfig(badgeID string) *BadgeResolutionConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.configs[badgeID]
}

// classifyOntologyTags maps keyword-matched ontology tags to ErrorClassIDs.
func classifyOntologyTags(tags []string) uint32 {
	for _, tag := range tags {
		lower := strings.ToLower(tag)
		if containsAny(lower, "memory", "exhaustion", "cpu", "oom", "disk", "resource") {
			return ErrorClassResourceExhaustion
		}
		if containsAny(lower, "latency", "timeout", "slow", "response") {
			return ErrorClassLatency
		}
		if containsAny(lower, "security", "isolate", "quarantine", "attack", "unauthorized") {
			return ErrorClassSecurity
		}
		if containsAny(lower, "crash", "panic", "fatal", "restart", "oom_kill") {
			return ErrorClassCrash
		}
	}
	return ErrorClassNone
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
