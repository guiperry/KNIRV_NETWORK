// Package harness starts every pre-trained model KNIRVMODELS serves for inference and
// puts them behind one endpoint. Colibri is the primary harness; llama.cpp is the
// fallback that serves whenever Colibri cannot: while it is still downloading and
// loading, when it has no model that fits the machine, and after it fails.
package harness

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Primary is the preferred model harness (Colibri).
type Primary interface {
	Install(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Healthy() bool
	HostPort() string
	ModelID() string
	ChosenModel() string
}

// Fallback is the harness that serves when the primary cannot (llama.cpp).
type Fallback interface {
	// Supervise starts the fallback and keeps it running until ctx ends.
	Supervise(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	GetSocketPath() string
}

// Config wires a Harness. Either harness may be nil to run the other alone.
type Config struct {
	// FrontSocket is the single inference socket callers use.
	FrontSocket string
	SocketPerm  os.FileMode

	Primary  Primary
	Fallback Fallback

	// KeepFallback leaves the fallback running while the primary serves. Off by
	// default: Colibri needs the RAM the fallback's model holds.
	KeepFallback bool
	// FallbackGrace is how long, once Colibri is installed, the fallback waits for it
	// to load before starting anyway (default 3m). A first run with nothing installed
	// starts the fallback at once, since the download can take hours.
	FallbackGrace time.Duration
	// FallbackIdle is how long a fallback that was started for a request Colibri
	// cannot serve (adapters, structured output) stays up after its last use
	// (default 15m).
	FallbackIdle time.Duration
	// StableAfter is how long the primary must stay healthy before the fallback is
	// released (default 30s).
	StableAfter time.Duration
	// MonitorEvery and FailuresBeforeDown set how the primary is watched (default
	// every 10s, down after 3 failed checks).
	MonitorEvery       time.Duration
	FailuresBeforeDown int
	// RetryBackoff is the wait before each retry of a failed primary install or
	// start (default 30s, 2m, 10m, 30m; the last repeats).
	RetryBackoff []time.Duration

	Logf func(format string, args ...any)
}

// Harness runs the primary and the fallback and routes inference between them.
type Harness struct {
	cfg    Config
	router *Router

	runCtx     atomic.Value // context.Context of Run
	lastDemand atomic.Int64 // unix nanos of the last request that needed the fallback
	primaryUp  atomic.Bool
	// teacherFallback: Colibri is not enabled on this run (no primary, or it was
	// skipped/disabled for good), so llama.cpp is the teacher.
	teacherFallback atomic.Bool
	primaryNote     atomic.Value // string: why the primary is not serving

	fbMu     sync.Mutex
	fbCancel context.CancelFunc
	fbDone   chan struct{}
}

// New builds a Harness and fills defaults.
func New(cfg Config) *Harness {
	if cfg.SocketPerm == 0 {
		cfg.SocketPerm = 0o660
	}
	if cfg.StableAfter <= 0 {
		cfg.StableAfter = 30 * time.Second
	}
	if cfg.FallbackGrace <= 0 {
		cfg.FallbackGrace = 3 * time.Minute
	}
	if cfg.FallbackIdle <= 0 {
		cfg.FallbackIdle = 15 * time.Minute
	}
	if cfg.MonitorEvery <= 0 {
		cfg.MonitorEvery = 10 * time.Second
	}
	if cfg.FailuresBeforeDown <= 0 {
		cfg.FailuresBeforeDown = 3
	}
	if cfg.RetryBackoff == nil {
		cfg.RetryBackoff = []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute}
	}
	h := &Harness{cfg: cfg}
	h.primaryNote.Store("")
	h.teacherFallback.Store(cfg.Primary == nil)

	var backends []*Backend
	if p := cfg.Primary; p != nil {
		backends = append(backends, &Backend{
			Name:  "colibri",
			Ready: func() bool { return h.primaryUp.Load() },
			Dial:  func() (string, string) { return "tcp", p.HostPort() },
			Model: p.ModelID,
			// Teacher jobs (synthetic data, suggestions, judging) are Colibri's work while
			// it is enabled, even when it is briefly down (they wait; llama.cpp does not
			// take over mid-run).
			Teacher: func() bool { return !h.teacherFallback.Load() },
			Supports: func(n Needs) bool {
				if n.Adapter {
					return false // skill adapters are GGUF LoRA files: llama.cpp only
				}
				if n.Frame {
					return false // role-framed calls are checked, prioritised and tool-gated by the llama wrapper
				}
				if g, ok := p.(interface{ SupportsGrammar() bool }); ok && n.Grammar && !g.SupportsGrammar() {
					return false
				}
				if t, ok := p.(interface{ SupportsTools() bool }); ok && n.Tools && !t.SupportsTools() {
					return false
				}
				return true
			},
		})
	}
	if f := cfg.Fallback; f != nil {
		backends = append(backends, &Backend{
			Name:   "llama.cpp",
			Ready:  f.IsRunning,
			Dial:   func() (string, string) { return "unix", f.GetSocketPath() },
			Demand: h.demandFallback,
			// llama.cpp teaches only when Colibri is not enabled on this host.
			Teacher: func() bool { return h.teacherFallback.Load() },
		})
	}
	h.router = NewRouter(h.logf, backends...)
	h.router.Extra = func() map[string]any {
		doc := map[string]any{"primary_note": h.primaryNote.Load().(string)}
		if cfg.Primary != nil {
			doc["colibri_model"] = cfg.Primary.ChosenModel()
			if q, ok := cfg.Primary.(interface{ Qualification() map[string]any }); ok {
				doc["colibri_qualification"] = q.Qualification()
			}
		}
		return doc
	}
	return h
}

func (h *Harness) logf(format string, args ...any) {
	if h.cfg.Logf != nil {
		h.cfg.Logf(format, args...)
	}
}

// Router exposes the routing table (tests, status).
func (h *Harness) Router() *Router { return h.router }

// Run starts everything and blocks until ctx ends.
func (h *Harness) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := h.router.Serve(ctx, h.cfg.FrontSocket, h.cfg.SocketPerm); err != nil && !errors.Is(err, context.Canceled) {
			h.logf("inference endpoint failed: %v", err)
		}
	}()

	h.runCtx.Store(ctx)
	if h.cfg.Primary != nil {
		if pp, ok := h.cfg.Primary.(interface{ Prepared() bool }); ok && pp.Prepared() {
			// Colibri only has to load its model. Starting llama.cpp too would slow
			// that load down (both read from the same disk and use the same CPUs), so it
			// is started only if Colibri is not serving after a grace period.
			go h.startFallbackIfPrimaryLate(ctx)
		} else {
			// The primary may spend hours downloading a model.
			h.setFallback(ctx, true)
		}
		h.runPrimary(ctx)
	} else {
		h.setFallback(ctx, true)
		<-ctx.Done()
	}

	h.setFallback(ctx, false)
	if p := h.cfg.Primary; p != nil {
		sctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := p.Stop(sctx); err != nil {
			h.logf("stopping colibri: %v", err)
		}
		cancel()
	}
	wg.Wait()
	return ctx.Err()
}

// runPrimary installs, starts and watches the primary until ctx ends.
func (h *Harness) runPrimary(ctx context.Context) {
	p := h.cfg.Primary
	failures := 0
	for ctx.Err() == nil {
		err := p.Install(ctx)
		if err == nil {
			err = p.Start(ctx)
		}
		if q, ok := p.(interface{ Qualify(context.Context) error }); ok && err == nil {
			// Colibri only counts as serving on a host where it is fast enough.
			err = q.Qualify(ctx)
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			h.primaryNote.Store(err.Error())
			if IsPermanent(err) {
				h.teacherFallback.Store(true) // Colibri is out for good: llama.cpp is the teacher
				if errors.Is(err, ErrHostTooWeak) {
					h.logf("%v", err)
				} else {
					h.logf("colibri is unavailable and will not be retried: %v; llama.cpp stays the model backend", err)
				}
				h.setFallback(ctx, true)
				<-ctx.Done()
				return
			}
			failures++
			h.setFallback(ctx, true)
			wait := backoffAt(h.cfg.RetryBackoff, failures)
			h.logf("colibri failed: %v; llama.cpp serves meanwhile, retrying in %s", err, wait)
			if !sleep(ctx, wait) {
				return
			}
			continue
		}
		failures = 0
		h.primaryNote.Store("")
		h.primaryUp.Store(true)
		h.logf("colibri is serving (model %s)", p.ChosenModel())
		if !h.watchPrimary(ctx) {
			return
		}
		// It went down: serve from the fallback and bring Colibri back.
		h.primaryUp.Store(false)
		h.primaryNote.Store("colibri stopped answering")
		h.setFallback(ctx, true)
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		p.Stop(stopCtx)
		cancel()
	}
}

// watchPrimary returns true when the primary went down, false when ctx ended. It
// releases the fallback once the primary has been healthy for StableAfter and no
// request has needed the fallback for FallbackIdle.
func (h *Harness) watchPrimary(ctx context.Context) bool {
	healthySince := time.Now()
	bad := 0
	tick := time.NewTicker(h.cfg.MonitorEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-tick.C:
		}
		if h.cfg.Primary.Healthy() {
			bad = 0
			if !h.cfg.KeepFallback && time.Since(healthySince) >= h.cfg.StableAfter && h.fallbackRunning() && h.fallbackIdle() {
				h.logf("colibri has been healthy for %s; releasing the llama.cpp fallback", h.cfg.StableAfter)
				h.setFallback(ctx, false)
			}
			continue
		}
		bad++
		healthySince = time.Now()
		if bad >= h.cfg.FailuresBeforeDown {
			h.logf("colibri failed %d health checks", bad)
			return true
		}
	}
}

// startFallbackIfPrimaryLate starts the fallback when the primary is still not
// serving after FallbackGrace.
func (h *Harness) startFallbackIfPrimaryLate(ctx context.Context) {
	if !sleep(ctx, h.cfg.FallbackGrace) {
		return
	}
	if !h.primaryUp.Load() {
		h.logf("colibri is not serving after %s; starting the llama.cpp fallback", h.cfg.FallbackGrace)
		h.setFallback(ctx, true)
	}
}

// demandFallback is called for a request that Colibri cannot serve. It starts
// llama.cpp if it is stopped and records the use so the idle release waits.
func (h *Harness) demandFallback(string) {
	h.lastDemand.Store(time.Now().UnixNano())
	if h.fallbackRunning() {
		return
	}
	if ctx, ok := h.runCtx.Load().(context.Context); ok && ctx.Err() == nil {
		h.setFallback(ctx, true)
	}
}

func (h *Harness) fallbackRunning() bool {
	h.fbMu.Lock()
	defer h.fbMu.Unlock()
	return h.fbCancel != nil
}

// fallbackIdle reports that nothing is using the fallback: no request needed it for
// FallbackIdle and none is being served by it.
func (h *Harness) fallbackIdle() bool {
	for _, b := range h.router.backends {
		if b.Name == "llama.cpp" && b.Busy() {
			return false
		}
	}
	last := h.lastDemand.Load()
	return last == 0 || time.Since(time.Unix(0, last)) >= h.cfg.FallbackIdle
}

// setFallback starts (want) or stops the fallback. Both are idempotent.
func (h *Harness) setFallback(ctx context.Context, want bool) {
	f := h.cfg.Fallback
	if f == nil {
		return
	}
	h.fbMu.Lock()
	defer h.fbMu.Unlock()
	if want {
		if h.fbCancel != nil {
			return
		}
		fctx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		h.fbCancel, h.fbDone = cancel, done
		go func() {
			defer close(done)
			if err := f.Supervise(fctx); err != nil && fctx.Err() == nil {
				h.logf("llama.cpp fallback ended: %v", err)
			}
		}()
		return
	}
	if h.fbCancel == nil {
		return
	}
	h.fbCancel()
	done := h.fbDone
	h.fbCancel, h.fbDone = nil, nil
	h.fbMu.Unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
	sctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := f.Stop(sctx); err != nil {
		h.logf("stopping llama.cpp: %v", err)
	}
	cancel()
	h.fbMu.Lock()
}

func backoffAt(schedule []time.Duration, n int) time.Duration {
	if len(schedule) == 0 {
		return 0
	}
	if n-1 < len(schedule) {
		return schedule[n-1]
	}
	return schedule[len(schedule)-1]
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}
