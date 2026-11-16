package harness

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeLlama stands in for llama.cpp, the primary. up is whether it answers; phase is
// what its manager reports. permErr makes Supervise fail for good at once.
type fakeLlama struct {
	up      atomic.Bool
	phase   atomic.Value // string
	lastErr atomic.Value // string
	bootUp  bool         // comes up as soon as Supervise starts
	permErr error
	supers  atomic.Int32
	stops   atomic.Int32
}

func (f *fakeLlama) Supervise(ctx context.Context) error {
	f.supers.Add(1)
	if f.permErr != nil {
		return f.permErr
	}
	if f.bootUp {
		f.up.Store(true)
		f.phase.Store("running")
	}
	<-ctx.Done()
	return ctx.Err()
}
func (f *fakeLlama) Stop(context.Context) error {
	f.stops.Add(1)
	f.up.Store(false)
	return nil
}
func (f *fakeLlama) IsRunning() bool       { return f.up.Load() }
func (f *fakeLlama) GetSocketPath() string { return "/nonexistent.sock" }
func (f *fakeLlama) Phase() (string, string) {
	p, _ := f.phase.Load().(string)
	e, _ := f.lastErr.Load().(string)
	return p, e
}
func (f *fakeLlama) setPhase(phase, lastErr string) {
	f.phase.Store(phase)
	f.lastErr.Store(lastErr)
}

// fakeColibri stands in for the Colibri backup.
type fakeColibri struct {
	mu         sync.Mutex
	installErr []error // consumed one per Install call
	startErr   []error
	installs   atomic.Int32
	starts     atomic.Int32
	stops      atomic.Int32
	healthy    atomic.Bool
}

func (p *fakeColibri) pop(q *[]error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(*q) == 0 {
		return nil
	}
	e := (*q)[0]
	*q = (*q)[1:]
	return e
}
func (p *fakeColibri) Install(context.Context) error {
	p.installs.Add(1)
	return p.pop(&p.installErr)
}
func (p *fakeColibri) Start(context.Context) error {
	p.starts.Add(1)
	if err := p.pop(&p.startErr); err != nil {
		return err
	}
	p.healthy.Store(true)
	return nil
}
func (p *fakeColibri) Stop(context.Context) error {
	p.stops.Add(1)
	p.healthy.Store(false)
	return nil
}
func (p *fakeColibri) Healthy() bool       { return p.healthy.Load() }
func (p *fakeColibri) HostPort() string    { return "127.0.0.1:1" }
func (p *fakeColibri) ModelID() string     { return "m" }
func (p *fakeColibri) ChosenModel() string { return "small" }

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

func primaryNote(h *Harness) string { return h.primaryNote.Load().(string) }

func colibriRunning(h *Harness) bool { return h.BackupState() == StateServing }

// llama.cpp starts first. While it serves, Colibri is never installed or started.
func TestLlamaStartsFirstAndColibriStaysDownWhileLlamaServes(t *testing.T) {
	l := &fakeLlama{bootUp: true}
	c := &fakeColibri{}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c})

	eventually(t, "llama.cpp to serve", activeIs(h, "llama.cpp"))
	time.Sleep(300 * time.Millisecond)
	if c.installs.Load() != 0 || c.starts.Load() != 0 {
		t.Fatalf("colibri must not be installed or started while llama.cpp serves: installs=%d starts=%d", c.installs.Load(), c.starts.Load())
	}
	if h.PrimaryState() != StateServing || h.BackupState() != StateStopped {
		t.Fatalf("states primary=%q colibri=%q", h.PrimaryState(), h.BackupState())
	}
	if h.Router().teacher() == nil || h.Router().teacher().Name != "llama.cpp" {
		t.Fatal("llama.cpp must be the teacher while it is not out for good")
	}
}

// llama.cpp still loading is not a failure: Colibri stays down until it actually fails.
func TestColibriStaysDownWhileLlamaIsStillLoading(t *testing.T) {
	l := &fakeLlama{}
	l.setPhase("waiting", "")
	c := &fakeColibri{}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c})

	time.Sleep(300 * time.Millisecond)
	if c.installs.Load() != 0 {
		t.Fatal("a loading llama.cpp must not start colibri")
	}
	if h.PrimaryState() != StateStarting {
		t.Fatalf("primary state %q, want starting", h.PrimaryState())
	}
	l.up.Store(true)
	eventually(t, "llama.cpp to serve", activeIs(h, "llama.cpp"))
}

// A failed llama.cpp start brings Colibri up as the backup, and Colibri's transient
// install failure is retried.
func TestLlamaStartFailureStartsColibri(t *testing.T) {
	l := &fakeLlama{}
	l.setPhase("failed", "model file missing")
	c := &fakeColibri{installErr: []error{errors.New("still downloading")}}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c})

	eventually(t, "colibri to serve as the backup", activeIs(h, "colibri"))
	if c.installs.Load() < 2 {
		t.Fatalf("a failed colibri install must be retried, installs=%d", c.installs.Load())
	}
	if h.PrimaryState() != StateFailed || primaryNote(h) != "model file missing" {
		t.Fatalf("primary state %q note %q", h.PrimaryState(), primaryNote(h))
	}
	if h.BackupState() != StateServing {
		t.Fatalf("colibri state %q", h.BackupState())
	}
}

// llama.cpp recovers: it is preferred again and Colibri is released after StableAfter.
func TestLlamaRecoveryReleasesColibri(t *testing.T) {
	l := &fakeLlama{}
	l.setPhase("failed", "boom")
	c := &fakeColibri{}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c})
	eventually(t, "colibri serving", activeIs(h, "colibri"))

	l.up.Store(true)
	l.setPhase("running", "")
	eventually(t, "llama.cpp preferred again", activeIs(h, "llama.cpp"))
	eventually(t, "colibri released", func() bool { return h.BackupState() == StateStopped && c.stops.Load() >= 1 })
	if h.PrimaryState() != StateServing || primaryNote(h) != "" {
		t.Fatalf("primary state %q note %q", h.PrimaryState(), primaryNote(h))
	}
}

// llama.cpp that stops answering (after a serving run) counts as down: Colibri takes
// over, and is released again once llama.cpp is back and stable.
func TestLlamaStoppingStartsColibriAndRecoveryReleasesIt(t *testing.T) {
	l := &fakeLlama{bootUp: true}
	c := &fakeColibri{}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c})
	eventually(t, "llama.cpp serving", activeIs(h, "llama.cpp"))

	l.up.Store(false)
	l.setPhase("restarting", "")
	eventually(t, "colibri to serve", activeIs(h, "colibri"))
	if h.PrimaryState() != StateFailed {
		t.Fatalf("primary state %q, want failed", h.PrimaryState())
	}

	l.up.Store(true)
	l.setPhase("running", "")
	eventually(t, "colibri released", func() bool { return h.BackupState() == StateStopped })
	eventually(t, "llama.cpp preferred", activeIs(h, "llama.cpp"))
}

// A permanent llama.cpp failure leaves Colibri as the serving backup for good: it is
// never released, and it is the teacher.
func TestPermanentLlamaFailureKeepsColibriAsTheServingBackup(t *testing.T) {
	l := &fakeLlama{permErr: errors.New("binary missing")}
	c := &fakeColibri{}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c})
	eventually(t, "colibri serving", activeIs(h, "colibri"))
	time.Sleep(300 * time.Millisecond) // well past StableAfter
	if c.stops.Load() != 0 || !colibriRunning(h) {
		t.Fatalf("colibri must keep serving: stops=%d state=%q", c.stops.Load(), h.BackupState())
	}
	if h.PrimaryState() != StateUnavailable || primaryNote(h) != "binary missing" {
		t.Fatalf("primary state %q note %q", h.PrimaryState(), primaryNote(h))
	}
	if b := h.Router().teacher(); b == nil || b.Name != "colibri" {
		t.Fatalf("colibri must be the teacher once llama.cpp is out for good: %v", b)
	}
}

// KeepBackup leaves Colibri running after llama.cpp has recovered.
func TestKeepBackupLeavesColibriRunningAfterRecovery(t *testing.T) {
	l := &fakeLlama{}
	l.setPhase("failed", "boom")
	c := &fakeColibri{}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c, KeepBackup: true})
	eventually(t, "colibri serving", activeIs(h, "colibri"))

	l.up.Store(true)
	l.setPhase("running", "")
	eventually(t, "llama.cpp preferred", activeIs(h, "llama.cpp"))
	time.Sleep(300 * time.Millisecond)
	if c.stops.Load() != 0 || !colibriRunning(h) {
		t.Fatalf("KeepBackup must leave colibri running: stops=%d", c.stops.Load())
	}
}

// A permanent Colibri failure is not retried, and llama.cpp keeps serving.
func TestPermanentColibriFailureIsNotRetried(t *testing.T) {
	l := &fakeLlama{}
	l.setPhase("failed", "boom")
	c := &fakeColibri{installErr: []error{permanentError{ErrNoModelFits}}}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c})
	eventually(t, "colibri unavailable", func() bool { return h.BackupState() == StateUnavailable })
	time.Sleep(200 * time.Millisecond)
	if c.installs.Load() != 1 {
		t.Fatalf("a permanent failure must not be retried: installs=%d", c.installs.Load())
	}
	if h.Router().Active() != nil {
		t.Fatal("nothing should serve while llama.cpp is down and colibri is out")
	}
}

func TestPrimaryAloneAndBackupAlone(t *testing.T) {
	h, _, _ := runHarness(t, Config{Primary: &fakeLlama{bootUp: true}})
	eventually(t, "llama.cpp only", activeIs(h, "llama.cpp"))
	if h.BackupState() != StateUnavailable || h.Router().Status()["colibri_state"] != "unavailable" {
		t.Fatalf("colibri not configured: %q", h.BackupState())
	}

	// With llama.cpp not configured, Colibri is the serving backup for good.
	h2, _, _ := runHarness(t, Config{Backup: &fakeColibri{}})
	eventually(t, "colibri only", activeIs(h2, "colibri"))
	if h2.PrimaryState() != StateUnavailable {
		t.Fatalf("primary state %q", h2.PrimaryState())
	}
}

func TestShutdownStopsBothHarnesses(t *testing.T) {
	l := &fakeLlama{}
	l.setPhase("failed", "boom")
	c := &fakeColibri{}
	_, cancel, done := runHarness(t, Config{Primary: l, Backup: c})
	eventually(t, "colibri serving", func() bool { return c.healthy.Load() })
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return")
	}
	if l.stops.Load() == 0 || c.stops.Load() == 0 || l.up.Load() {
		t.Fatalf("both must be stopped: llama stops=%d colibri stops=%d", l.stops.Load(), c.stops.Load())
	}
}

// A colibri that fails its start (model did not load) is retried, and llama.cpp
// remains the primary once it is back.
func TestColibriStartFailureIsRetriedAsBackup(t *testing.T) {
	l := &fakeLlama{}
	l.setPhase("failed", "boom")
	c := &fakeColibri{startErr: []error{errors.New("model failed to load")}}
	h, _, _ := runHarness(t, Config{Primary: l, Backup: c})
	eventually(t, "colibri to serve after its retry", activeIs(h, "colibri"))
	if c.starts.Load() < 2 {
		t.Fatalf("starts=%d", c.starts.Load())
	}
}
