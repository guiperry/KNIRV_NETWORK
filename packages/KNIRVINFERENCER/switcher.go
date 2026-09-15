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
	internal InternalTextGenerator
	external TextGenerator
	force    int // 0=auto, 1=force internal, 2=force external
	ready    bool
	lastCheck time.Time
	mu       sync.RWMutex
}

const switcherCheckInterval = 1 * time.Hour

// NewInferenceSwitcher creates a switcher that auto-detects readiness.
func NewInferenceSwitcher(internal InternalTextGenerator, external TextGenerator) *InferenceSwitcher {
	return &InferenceSwitcher{
		internal: internal,
		external: external,
	}
}

// SetForce overrides the auto-detection mode.
//   0 = auto (default, checks internal readiness)
//   1 = force internal
//   2 = force external
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

	// Re-check readiness periodically
	if time.Since(lastCheck) > switcherCheckInterval {
		s.mu.Lock()
		if time.Since(s.lastCheck) > switcherCheckInterval {
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

// GenerateText routes to the internal or external generator.
func (s *InferenceSwitcher) GenerateText(ctx context.Context, prompt string) (string, error) {
	if s.IsReady() {
		log.Println("[inference-switcher] routed to internal transformer")
		return s.internal.GenerateText(ctx, prompt)
	}
	log.Println("[inference-switcher] routed to external LLM")
	return s.external.GenerateText(prompt)
}

// InferenceServiceAdapter wraps InferenceService to satisfy the TextGenerator
// interface used as the external fallback in NewInferenceSwitcher.
type InferenceServiceAdapter struct {
	Svc *InferenceService
}

func (a *InferenceServiceAdapter) GenerateText(prompt string) (string, error) {
	return a.Svc.GenerateTextWithContext(context.Background(), "", prompt, "")
}
