package harness

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakePrimary struct {
	mu         sync.Mutex
	installErr []error // consumed one per Install call
	startErr   []error
	installs   atomic.Int32
	starts     atomic.Int32
	stops      atomic.Int32
	healthy    atomic.Bool
}

func (p *fakePrimary) pop(q *[]error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(*q) == 0 {
		return nil
	}
	e := (*q)[0]
	*q = (*q)[1:]
	return e
}
func (p *fakePrimary) Install(context.Context) error { p.installs.Add(1); return p.pop(&p.installErr) }
func (p *fakePrimary) Start(context.Context) error {
	p.starts.Add(1)
	if err := p.pop(&p.startErr); err != nil {
		return err
	}
	p.healthy.Store(true)
	return nil
}
func (p *fakePrimary) Stop(context.Context) error { p.stops.Add(1); p.healthy.Store(false); return nil }
func (p *fakePrimary) Healthy() bool              { return p.healthy.Load() }
func (p *fakePrimary) HostPort() string           { return "127.0.0.1:1" }
func (p *fakePrimary) ModelID() string            { return "m" }
func (p *fakePrimary) ChosenModel() string        { return "small" }

type fakeFallback struct {
	running atomic.Bool
	supers  atomic.Int32
	stops   atomic.Int32
}

func (f *fakeFallback) Supervise(ctx context.Context) error {
	f.supers.Add(1)
	f.running.Store(true)
	<-ctx.Done()
	return ctx.Err()
}
func (f *fakeFallback) Stop(context.Context) error {
	f.stops.Add(1)
	f.running.Store(false)
	return nil
}
func (f *fakeFallback) IsRunning() bool       { return f.running.Load() }
func (f *fakeFallback) GetSocketPath() string { return "/nonexistent.sock" }

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func runHarness(t *testing.T, cfg Config) (*Harness, context.CancelFunc, chan struct{}) {
	t.Helper()
	cfg.FrontSocket = t.TempDir() + "/llama.sock"
	cfg.StableAfter = 50 * time.Millisecond
	cfg.MonitorEvery = 10 * time.Millisecond
	cfg.FailuresBeforeDown = 2
	cfg.RetryBackoff = []time.Duration{20 * time.Millisecond}
	logs := &safeBuf{}
	cfg.Logf = logs.logf
	h := New(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { h.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return h, cancel, done
}

func activeIs(h *Harness, name string) func() bool {
	return func() bool { a := h.Router().Active(); return a != nil && a.Name == name }
}

// While Colibri is still installing, llama.cpp is what serves.
func TestFallbackStartsImmediatelyAndPrimaryTakesOver(t *testing.T) {
	p, f := &fakePrimary{}, &fakeFallback{}
	p.installErr = []error{errors.New("still downloading")}
	h, _, _ := runHarness(t, Config{Primary: p, Fallback: f})

	eventually(t, "fallback serving while the primary is not ready", activeIs(h, "llama.cpp"))
	eventually(t, "primary to serve", activeIs(h, "colibri"))
	eventually(t, "fallback released after the primary is stable", func() bool { return !f.IsRunning() })
	if p.installs.Load() < 2 {
		t.Fatalf("a failed install must be retried, installs=%d", p.installs.Load())
	}
}

func TestPrimaryFailureBringsFallbackBackAndRestartsPrimary(t *testing.T) {
	p, f := &fakePrimary{}, &fakeFallback{}
	h, _, _ := runHarness(t, Config{Primary: p, Fallback: f})
	eventually(t, "primary serving", activeIs(h, "colibri"))
	eventually(t, "fallback released", func() bool { return !f.IsRunning() })

	p.healthy.Store(false) // Colibri dies
	eventually(t, "primary to be restarted", func() bool { return p.starts.Load() >= 2 })
	eventually(t, "primary to serve again", activeIs(h, "colibri"))
	if f.supers.Load() < 2 {
		t.Fatalf("the fallback must be started again when the primary fails, supervise calls=%d", f.supers.Load())
	}
}

func TestPermanentPrimaryFailureLeavesFallbackServing(t *testing.T) {
	p, f := &fakePrimary{}, &fakeFallback{}
	p.installErr = []error{permanentError{ErrNoModelFits}}
	h, _, _ := runHarness(t, Config{Primary: p, Fallback: f})
	eventually(t, "fallback serving", activeIs(h, "llama.cpp"))
	time.Sleep(200 * time.Millisecond)
	if p.installs.Load() != 1 || !f.IsRunning() {
		t.Fatalf("a permanent failure must not be retried and the fallback must stay: installs=%d running=%v", p.installs.Load(), f.IsRunning())
	}
	if h.Router().Status()["primary_note"] == "" {
		t.Fatal("the reason must be visible in the status")
	}
}

func TestKeepFallbackLeavesItRunning(t *testing.T) {
	p, f := &fakePrimary{}, &fakeFallback{}
	h, _, _ := runHarness(t, Config{Primary: p, Fallback: f, KeepFallback: true})
	eventually(t, "primary serving", activeIs(h, "colibri"))
	time.Sleep(200 * time.Millisecond)
	if !f.IsRunning() {
		t.Fatal("KeepFallback must leave llama.cpp running")
	}
}

func TestFallbackAloneAndPrimaryAlone(t *testing.T) {
	h, _, _ := runHarness(t, Config{Fallback: &fakeFallback{}})
	eventually(t, "fallback only", activeIs(h, "llama.cpp"))

	h2, _, _ := runHarness(t, Config{Primary: &fakePrimary{}})
	eventually(t, "primary only", activeIs(h2, "colibri"))
}

func TestShutdownStopsBothHarnesses(t *testing.T) {
	p, f := &fakePrimary{}, &fakeFallback{}
	_, cancel, done := runHarness(t, Config{Primary: p, Fallback: f, KeepFallback: true})
	eventually(t, "up", func() bool { return f.IsRunning() && p.healthy.Load() })
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return")
	}
	if p.stops.Load() == 0 || f.stops.Load() == 0 || f.IsRunning() {
		t.Fatalf("both must be stopped: primary stops=%d fallback stops=%d", p.stops.Load(), f.stops.Load())
	}
}

type preparedPrimary struct{ fakePrimary }

func (*preparedPrimary) Prepared() bool        { return true }
func (*preparedPrimary) SupportsGrammar() bool { return false }

// An installed Colibri only has to load, so llama.cpp must not compete with it for the
// disk and CPUs unless Colibri turns out to be late.
func TestFallbackWaitsWhenColibriIsAlreadyInstalled(t *testing.T) {
	p, f := &preparedPrimary{}, &fakeFallback{}
	h, _, _ := runHarness(t, Config{Primary: p, Fallback: f, FallbackGrace: time.Hour})
	eventually(t, "primary serving", activeIs(h, "colibri"))
	time.Sleep(150 * time.Millisecond)
	if f.supers.Load() != 0 {
		t.Fatal("the fallback must not start while an installed colibri loads")
	}
}

func TestFallbackStartsWhenInstalledColibriIsLate(t *testing.T) {
	p, f := &preparedPrimary{}, &fakeFallback{}
	p.startErr = []error{errors.New("model failed to load")}
	h, _, _ := runHarness(t, Config{Primary: p, Fallback: f, FallbackGrace: 50 * time.Millisecond})
	eventually(t, "fallback to serve while colibri is down", activeIs(h, "llama.cpp"))
}

// A request colibri cannot serve brings llama.cpp up; it then stays while in use and
// is released once idle.
func TestStructuredOutputStartsFallbackOnDemandAndReleasesItWhenIdle(t *testing.T) {
	p, f := &preparedPrimary{}, &fakeFallback{}
	h, _, _ := runHarness(t, Config{Primary: p, Fallback: f, FallbackGrace: time.Hour, FallbackIdle: 300 * time.Millisecond})
	eventually(t, "primary serving", activeIs(h, "colibri"))

	fb := h.router.backends[1]
	fb.Dial = func() (string, string) { return "unix", "/nonexistent.sock" }
	if fb.Supports != nil || !h.router.backends[0].Supports(Needs{}) || h.router.backends[0].Supports(Needs{Grammar: true}) || h.router.backends[0].Supports(Needs{Adapter: true}) {
		t.Fatal("colibri must serve plain chat only here")
	}
	fb.Demand("structured output")
	eventually(t, "fallback started on demand", f.IsRunning)
	eventually(t, "fallback released once idle", func() bool { return !f.IsRunning() })
}

type unqualifiedPrimary struct{ fakePrimary }

func (*unqualifiedPrimary) Qualify(context.Context) error {
	return permanentError{ErrHostTooWeak}
}

// A host that fails the gate keeps llama.cpp serving, starts it immediately, and does
// not retry.
func TestUnqualifiedColibriLeavesLlamaServingAndIsNotRetried(t *testing.T) {
	p, f := &unqualifiedPrimary{}, &fakeFallback{}
	h, _, _ := runHarness(t, Config{Primary: p, Fallback: f, FallbackGrace: time.Hour})
	eventually(t, "llama.cpp to serve", activeIs(h, "llama.cpp"))
	time.Sleep(200 * time.Millisecond)
	if p.starts.Load() != 1 || h.primaryUp.Load() {
		t.Fatalf("starts=%d up=%v", p.starts.Load(), h.primaryUp.Load())
	}
	if h.Router().Status()["primary_note"] == "" {
		t.Fatal("the reason must be in the status")
	}
}
