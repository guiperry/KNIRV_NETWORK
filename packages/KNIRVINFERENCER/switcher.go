package inferencer

import (
	"context"
	"log"
	"sync"
	"time"
)

// TrainingStateProvider reports whether the internal transformer model
// is trained enough for production use.
type TrainingStateProvider interface {
	IsReady() bool
	Progress() float64
}

// InternalTextGenerator generates text using the internal transformer
// (Gorgonite GPT or HasherTransformer via HEART).
type InternalTextGenerator interface {
	TrainingStateProvider
	GenerateText(ctx context.Context, prompt string) (string, error)
}

// InferenceSwitcher routes text generation between the internal transformer
// and external LLM providers. When the internal model is fully pre-trained,
// calls are routed internally; otherwise they fall through to the external
// LLM chain.
type InferenceSwitcher struct {
	internal  InternalTextGenerator
	external  TextGenerator
	force     int // 0=auto, 1=force internal, 2=force external
	ready     bool
	lastCheck time.Time
	mu        sync.RWMutex
	harness   *Harness
}

const switcherCheckInterval = 1 * time.Hour

// switcherRetryInterval is how soon a model that was not ready is asked again,
// so a daemon that was still starting (or a HEART that just finished training)
// is not ignored for a whole switcherCheckInterval.
const switcherRetryInterval = 1 * time.Minute

// NewInferenceSwitcher creates a switcher that auto-detects readiness.
func NewInferenceSwitcher(internal InternalTextGenerator, external TextGenerator) *InferenceSwitcher {
	s := &InferenceSwitcher{
		internal: internal,
		external: external,
	}
	// The switcher is a two-model view of the harness: HEART first, then the provider
	// chain. Readiness, breaker, fallback and traces are the harness's.
	s.harness = NewHarness(
		[]HarnessModel{&switcherInternal{s}, &switcherExternal{external}},
		WithRoutes(map[Kind][]string{KindGenerate: {"heart", "external"}}),
	)
	return s
}

// Harness exposes the switcher's harness (traces, stats).
func (s *InferenceSwitcher) Harness() *Harness { return s.harness }

type switcherInternal struct{ s *InferenceSwitcher }

func (m *switcherInternal) Name() string    { return "heart" }
func (m *switcherInternal) Info() ModelInfo { return ModelInfo{} }
func (m *switcherInternal) Ready(context.Context) Readiness {
	if m.s.internal == nil || !m.s.IsReady() {
		return Readiness{Detail: "internal model not ready"}
	}
	return Readiness{Ready: true, Version: "internal"}
}
func (m *switcherInternal) Do(ctx context.Context, t Task) (Result, error) {
	text, err := m.s.internal.GenerateText(ctx, t.Input.(string))
	if err != nil {
		// Look again after the retry interval instead of failing every call until then.
		m.s.mu.Lock()
		m.s.ready = false
		m.s.mu.Unlock()
		return Result{}, err
	}
	return Result{Text: text, Model: "heart", Version: "internal"}, nil
}

type switcherExternal struct{ ext TextGenerator }

func (m *switcherExternal) Name() string    { return "external" }
func (m *switcherExternal) Info() ModelInfo { return ModelInfo{External: true, Paid: true} }
func (m *switcherExternal) Ready(context.Context) Readiness {
	return Readiness{Ready: m.ext != nil, Version: "provider-chain"}
}
func (m *switcherExternal) Do(_ context.Context, t Task) (Result, error) {
	text, err := m.ext.GenerateText(t.Input.(string))
	if err != nil {
		return Result{}, err
	}
	return Result{Text: text, Model: "external", Version: "provider-chain"}, nil
}

// SetForce overrides the auto-detection mode.
//
//	0 = auto (default, checks internal readiness)
//	1 = force internal
//	2 = force external
func (s *InferenceSwitcher) SetForce(mode int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.force = mode
}

// IsReady returns true if the internal model should be used.
func (s *InferenceSwitcher) IsReady() bool {
	s.mu.RLock()
	force := s.force
	ready := s.ready
	lastCheck := s.lastCheck
	s.mu.RUnlock()

	if force == 1 {
		return true
	}
	if force == 2 {
		return false
	}

	// Re-check readiness periodically (sooner while not ready).
	recheck := switcherCheckInterval
	if !ready {
		recheck = switcherRetryInterval
	}
	if time.Since(lastCheck) > recheck {
		s.mu.Lock()
		if time.Since(s.lastCheck) > recheck {
			if s.internal != nil {
				s.ready = s.internal.IsReady()
			} else {
				s.ready = false
			}
			s.lastCheck = time.Now()
			if s.ready {
				log.Printf("[inference-switcher] internal model is READY — routing internally")
			}
		}
		ready = s.ready
		s.mu.Unlock()
	}

	return ready
}

// Progress returns the internal model's training progress.
func (s *InferenceSwitcher) Progress() float64 {
	s.mu.RLock()
	internal := s.internal
	s.mu.RUnlock()
	if internal == nil {
		return 1.0
	}
	return internal.Progress()
}

// GenerateText routes through the harness: HEART when it is ready, else (or on
// failure) the external chain. A forced mode pins the harness to one model.
func (s *InferenceSwitcher) GenerateText(ctx context.Context, prompt string) (string, error) {
	s.mu.RLock()
	force := s.force
	s.mu.RUnlock()
	var c Constraints
	switch force {
	case 1:
		c.Pin = "heart"
	case 2:
		c.Pin = "external"
	}
	res, tr, err := s.harness.Do(ctx, Task{Kind: KindGenerate, Input: prompt, Constraints: c})
	if err != nil {
		return "", err
	}
	log.Printf("[inference-switcher] routed to %s (%d fallback hop(s))", tr.Chosen, len(tr.Fallbacks))
	return res.Text, nil
}

// InferenceServiceAdapter wraps InferenceService to satisfy the TextGenerator
// interface used as the external fallback in NewInferenceSwitcher.
type InferenceServiceAdapter struct {
	Svc *InferenceService
}

func (a *InferenceServiceAdapter) GenerateText(prompt string) (string, error) {
	return a.Svc.GenerateTextWithContext(context.Background(), "", prompt, "")
}
