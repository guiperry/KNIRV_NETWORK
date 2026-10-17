package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// Supervisor owns the llama-server child process so its adapter set can change:
// llama-server loads adapters only at startup, so a change is a restart.
type Supervisor struct {
	Path, Model, Address string
	Base                 Options // adapters are supplied per Start
	// HealthTimeout bounds how long a start waits for /health. Zero means
	// KNIRV_LLAMA_HEALTH_TIMEOUT, else DefaultHealthTimeout. A child that exits is
	// reported at once and never waits for it.
	HealthTimeout time.Duration
	// Logf receives progress lines ("loading model, 45s elapsed"). Nil is silent.
	Logf func(format string, args ...any)
	// NoWarm disables pre-reading the model into the page cache while it starts.
	NoWarm bool

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
		timeout = HealthTimeoutFromEnv()
	}
	if !s.NoWarm && os.Getenv("KNIRV_LLAMA_WARM") != "0" {
		warmCtx, stopWarm := context.WithCancel(ctx)
		defer stopWarm()
		go func() {
			if n, err := WarmFile(warmCtx, s.Model); err == nil && n > 0 && s.Logf != nil {
				s.Logf("model pre-read into page cache (%d MB)", n>>20)
			}
		}()
	}
	if err := WaitHealthy(ctx, s.Address, child.Exited(), WaitOptions{Timeout: timeout, Logf: s.Logf}); err != nil {
		child.Stop()
		s.child = nil
		return err
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
