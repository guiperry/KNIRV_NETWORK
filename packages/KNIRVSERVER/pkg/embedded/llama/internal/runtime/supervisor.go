package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Supervisor owns the llama-server child process so its adapter set can change:
// llama-server loads adapters only at startup, so a change is a restart.
type Supervisor struct {
	Path, Model, Address string
	Base                 Options // adapters are supplied per Start
	// HealthTimeout bounds how long a start waits for /health (a large model takes a while).
	HealthTimeout time.Duration

	mu    sync.Mutex
	child *Server
}

// Start (re)starts llama-server with exactly these adapters. Every adapter is
// loaded inert and then zeroed server-wide, so a request without a `lora` field
// runs the plain base model. It returns once llama-server is healthy and reports
// each adapter.
func (s *Supervisor) Start(ctx context.Context, adapters []LoRAAdapter) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.child != nil {
		s.child.Stop()
		s.child = nil
	}
	opts := s.Base
	opts.LoRA = adapters
	opts.LoRAInitWithoutApply = len(adapters) > 0
	if err := opts.Validate(); err != nil {
		return err
	}
	child, err := Start(s.Path, s.Model, s.Address, opts)
	if err != nil {
		return err
	}
	s.child = child
	timeout := s.HealthTimeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for !Healthy(ctx, s.Address) {
		select {
		case <-child.Exited():
			s.child = nil
			return fmt.Errorf("llama-server exited during startup (bad model or adapter?)")
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			child.Stop()
			s.child = nil
			return fmt.Errorf("llama-server did not become healthy within %s", timeout)
		}
	}
	if len(adapters) == 0 {
		return nil
	}
	got, err := GetLoRAAdapters(ctx, s.Address, s.Base.APIKey)
	if err != nil {
		return err
	}
	if len(got) != len(adapters) {
		return fmt.Errorf("llama-server loaded %d adapter(s), expected %d", len(got), len(adapters))
	}
	if _, err := ZeroLoRAAdapters(ctx, s.Address, s.Base.APIKey); err != nil {
		return fmt.Errorf("zero adapters: %w", err)
	}
	return nil
}

// Stop stops the child.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.child != nil {
		s.child.Stop()
		s.child = nil
	}
}
