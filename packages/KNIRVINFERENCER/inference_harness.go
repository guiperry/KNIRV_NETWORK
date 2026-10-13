package inferencer

import (
	"context"
	"errors"
)

const llamaProviderName = "llama"

// SetHarness routes the service's generation, llama server tasks and planning through
// h. Callers keep the same methods; only the path behind them changes.
func (s *InferenceService) SetHarness(h *Harness) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.harness = h
}

// Harness returns the attached harness, or nil.
func (s *InferenceService) Harness() *Harness {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.harness
}

// ExternalModel is the service's provider chain as a harness model. Its calls go to the
// delegator/orchestrator directly, never back through the harness.
func (s *InferenceService) ExternalModel() *ExternalModel {
	return &ExternalModel{GenerateFn: s.generateExternal, PlanFn: s.executeComplexTask}
}

// LlamaServerTask is what the harness's llama model calls for server tasks, so dream
// passes keep the llama provider's system prompt and JSON response format.
func (s *InferenceService) LlamaServerTask(_ context.Context, prompt string) (string, error) {
	return s.generateWithProvider(llamaProviderName, prompt)
}

func (s *InferenceService) generateExternal(ctx context.Context, prompt string) (string, error) {
	s.mutex.Lock()
	d := s.delegator
	running := s.isRunning
	s.mutex.Unlock()
	if !running {
		return "", errors.New("inference service is not running")
	}
	if d == nil {
		return "", errors.New("inference service: delegator not configured")
	}
	return d.GenerateSimple(ctx, "", prompt, "")
}
