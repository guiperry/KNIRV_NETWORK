// Package harness starts every pre-trained model KNIRVMODELS serves for inference and
// puts them behind one endpoint. llama.cpp is the primary harness: it starts first and
// is the default serving backend whenever it is ready. Colibri is the backup: it is
// started only when llama.cpp fails to start or stops serving (and is the backup for
// good when llama.cpp is out for good), and it is released again once llama.cpp has
// been healthy for StableAfter, unless KeepBackup is set.
package harness

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Primary is the harness that serves by default (llama.cpp).
type Primary interface {
	// Supervise starts the harness and keeps it running until ctx ends. It returns an
	// error only when the harness has failed for good; transient failures are retried
	// by the harness itself.
	Supervise(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	GetSocketPath() string
}

// phaser is optionally implemented by a Primary. Phase reports the manager's own
// lifecycle ("starting", "waiting", "running", "restarting", "failed", "stopped") and
// the last error. A "failed" phase means the last start attempt failed.
type phaser interface {
	Phase() (phase, lastError string)
}

// Backup is the harness that serves only while the Primary cannot (Colibri).
type Backup interface {
	Install(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Healthy() bool
	HostPort() string
	ModelID() string
	ChosenModel() string
}

// Config wires a Harness. Either harness may be nil to run the other alone.
type Config struct {
	// FrontSocket is the single inference socket callers use.
	FrontSocket string
	SocketPerm  os.FileMode

	// Primary is llama.cpp. It is started first and serves whenever it is ready.
	Primary Primary
	// Backup is Colibri. It is started only while the Primary is down (failed to start,
	// or stopped serving) or out for good, and released after the Primary has been
	// healthy for StableAfter.
	Backup Backup

	// KeepBackup leaves the backup running while the primary serves. Off by default:
	// Colibri needs the RAM its model holds.
	KeepBackup bool
	// BackupIdle is how long a backup that a request has waited on stays up after its
	// last use (default 15m). It holds the backup past its release only while needed.
	BackupIdle time.Duration
	// StableAfter is how long the primary must stay healthy before the backup is
	// released (default 30s).
	StableAfter time.Duration
	// MonitorEvery and FailuresBeforeDown set how the primary and backup are watched
	// (default every 10s, down after 3 failed checks).
	MonitorEvery       time.Duration
	FailuresBeforeDown int
	// RetryBackoff is the wait before each retry of a failed backup install or start
	// (default 30s, 2m, 10m, 30m; the last repeats).
	RetryBackoff []time.Duration

	// Extra adds blocks to the /harness/status document (the Ollama manager's status
	// goes here, under "ollama"). Called on every status request; it must be cheap.
	Extra func() map[string]any
	// Embed is the daemon-owned embedding operation (Ollama). It is exposed only
	// through the front Unix socket, never as a caller-configured Ollama URL.
	Embed func(context.Context, []string) ([][]float32, error)

	Logf func(format string, args ...any)
}

// Harness runs the primary and the backup and routes inference between them.
type Harness struct {
	cfg    Config
	router *Router

	lastDemand atomic.Int64 // unix nanos of the last request that waited on a backend

	// primaryUpAt is when llama.cpp last came up (unix nanos), 0 while it is not
	// answering. The backup is released only once this is StableAfter old.
	primaryUpAt atomic.Int64
	// primaryFailed: llama.cpp is down now (failed to start or stopped). The backup is
	// wanted while it is.
	primaryFailed atomic.Bool
	// primaryOut: llama.cpp is out for good (its supervision failed, or it is not
	// configured). The backup is then the serving backup and the teacher for good.
	primaryOut   atomic.Bool
	primaryState atomic.Value // State
	primaryNote  atomic.Value // string

	backupUp    atomic.Bool  // Colibri is serving
	backupDead  atomic.Bool  // Colibri failed for good (never retried)
	backupState atomic.Value // State
	backupNote  atomic.Value // string
}

// New builds a Harness and fills defaults.
func New(cfg Config) *Harness {
	if cfg.SocketPerm == 0 {
		cfg.SocketPerm = 0o660
	}
	if cfg.StableAfter <= 0 {
		cfg.StableAfter = 30 * time.Second
	}
	if cfg.BackupIdle <= 0 {
		cfg.BackupIdle = 15 * time.Minute
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
	h.backupNote.Store("")
	h.primaryState.Store(StateStarting)
	h.backupState.Store(StateStopped)
	if cfg.Primary == nil {
		h.primaryState.Store(StateUnavailable)
		h.primaryNote.Store("llama.cpp is not configured")
		h.primaryOut.Store(true)
		h.primaryFailed.Store(true)
	}
	if cfg.Backup == nil {
		h.backupState.Store(StateUnavailable)
		h.backupNote.Store("colibri is not configured")
	}

	// Backends are listed in priority order: llama.cpp first, Colibri second.
	var backends []*Backend
	if f := cfg.Primary; f != nil {
		backends = append(backends, &Backend{
			Name:   "llama.cpp",
			Ready:  f.IsRunning,
			Dial:   func() (string, string) { return "unix", f.GetSocketPath() },
			Demand: h.noteDemand,
			// Teacher output comes from llama.cpp (see the teacher note below). It is the
			// teacher until it is out for good; a transient failure makes teacher requests
			// wait, it does not hand them to Colibri mid-run.
			Teacher: func() bool { return !h.primaryOut.Load() },
			// Out for good: a request that needs it is refused at once, not held.
			Gone: h.primaryOut.Load,
		})
	}
	if p := cfg.Backup; p != nil {
		backends = append(backends, &Backend{
			Name:   "colibri",
			Ready:  h.backupUp.Load,
			Dial:   func() (string, string) { return "tcp", p.HostPort() },
			Model:  p.ModelID,
			Demand: h.noteDemand,
			// Teacher output comes from Colibri only once llama.cpp is out for good, so one
			// data set is never a mix of two teachers from the same run.
			Teacher: h.primaryOut.Load,
			Supports: func(n Needs) bool {
				// Colibri serves plain chat only. Adapters, structured output, tools and
				// role frames are llama.cpp's: the primary supports them all.
				if n.Adapter || n.Frame {
					return false
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
	h.router = NewRouter(h.logf, backends...)
	h.router.Extra = func() map[string]any {
		doc := map[string]any{
			"primary_state": string(h.PrimaryState()),
			"primary_note":  h.primaryNote.Load().(string),
			"colibri_state": string(h.BackupState()),
			"colibri_note":  h.backupNote.Load().(string),
		}
		if p := cfg.Backup; p != nil {
			doc["colibri_model"] = p.ChosenModel()
			if q, ok := p.(interface{ Qualification() map[string]any }); ok {
				doc["colibri_qualification"] = q.Qualification()
			}
		}
		if cfg.Extra != nil {
			for k, v := range cfg.Extra() {
				doc[k] = v
			}
		}
		return doc
	}
	h.router.Embed = cfg.Embed
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

	if h.cfg.Primary != nil {
		// llama.cpp starts first and is not stopped while it serves.
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.runPrimary(ctx)
		}()
	}
	h.control(ctx)
	wg.Wait()
	return ctx.Err()
}

// runPrimary runs llama.cpp until ctx ends. It records whether llama.cpp is down
// (primaryFailed: the backup is wanted) or out for good (primaryOut), and when it came
// up (primaryUpAt: the backup is released after StableAfter of that).
//
// llama.cpp counts as down when its manager reports a failed start attempt, or when
// it has stopped answering for FailuresBeforeDown checks. Restarts that the manager
// is doing itself, inside that window, are not failures yet.
func (h *Harness) runPrimary(ctx context.Context) {
	f := h.cfg.Primary
	sctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := f.Stop(stopCtx); err != nil {
			h.logf("stopping llama.cpp: %v", err)
		}
		stopCancel()
	}()

	errc := make(chan error, 1)
	go func() { errc <- f.Supervise(sctx) }()

	tick := time.NewTicker(h.cfg.MonitorEvery)
	defer tick.Stop()
	wasUp := false // llama.cpp answered on the last check
	down := false  // llama.cpp is down now (failed to start, or stopped for good enough to count)
	bad := 0       // failed checks since it stopped answering
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errc:
			if ctx.Err() != nil {
				return
			}
			if err == nil {
				err = errors.New("llama.cpp supervision ended")
			}
			// Supervise returns an error only when llama.cpp will not start for good.
			h.primaryUpAt.Store(0)
			h.primaryOut.Store(true)
			h.primaryFailed.Store(true)
			h.primaryNote.Store(err.Error())
			h.setPrimaryState(StateUnavailable)
			h.logf("llama.cpp is unavailable and will not be retried: %v; colibri serves as the backup for good", err)
			return
		case <-tick.C:
		}

		if f.IsRunning() {
			bad, down = 0, false
			if !wasUp {
				wasUp = true
				h.primaryUpAt.Store(time.Now().UnixNano())
				h.primaryFailed.Store(false)
				h.primaryNote.Store("")
				h.setPrimaryState(StateServing)
				h.logf("llama.cpp is serving")
			}
			continue
		}

		// Not answering now.
		h.primaryUpAt.Store(0)
		note := h.primaryNote.Load().(string) // kept while it stays down
		if wasUp {
			bad++
			if bad < h.cfg.FailuresBeforeDown {
				h.setPrimaryState(StateStarting) // a restart the manager is doing itself
				continue
			}
			wasUp, bad = false, 0
			down, note = true, "llama.cpp stopped answering"
		} else if !down {
			phase, lastErr := "", ""
			if p, ok := f.(phaser); ok {
				phase, lastErr = p.Phase()
			}
			if phase != "failed" {
				h.setPrimaryState(StateStarting) // still loading its model
				continue
			}
			down = true
			note = lastErr
			if note == "" {
				note = "llama.cpp failed to start"
			}
		}
		if !h.primaryFailed.Load() {
			h.logf("llama.cpp is down (%s); colibri serves as the backup until it is back", note)
		}
		h.primaryFailed.Store(true)
		h.primaryNote.Store(note)
		h.setPrimaryState(StateFailed)
	}
}

// control runs the backup policy until ctx ends: Colibri is started while llama.cpp
// is down or out for good, and released once llama.cpp has been healthy for
// StableAfter (unless KeepBackup), when nothing is waiting on it.
func (h *Harness) control(ctx context.Context) {
	if h.cfg.Backup == nil {
		<-ctx.Done()
		return
	}
	var cancel context.CancelFunc
	var done chan struct{}
	start := func() {
		var bctx context.Context
		bctx, cancel = context.WithCancel(ctx)
		done = make(chan struct{})
		go func(d chan struct{}) {
			defer close(d)
			h.runBackup(bctx)
		}(done)
	}
	stop := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(60 * time.Second):
			h.logf("colibri did not stop in time")
		}
		cancel, done = nil, nil
	}
	defer func() {
		if cancel != nil {
			stop()
		}
	}()

	tick := time.NewTicker(h.cfg.MonitorEvery)
	defer tick.Stop()
	for {
		running := done != nil && isOpen(done)
		want := h.primaryFailed.Load() || h.primaryOut.Load()
		switch {
		case want && !running && !h.backupDead.Load():
			h.logf("starting colibri as the backup: llama.cpp is not serving")
			start()
		case !want && running && h.releasable():
			h.logf("llama.cpp has been healthy for %s; releasing colibri", h.cfg.StableAfter)
			stop()
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// releasable reports whether the backup can be stopped: llama.cpp has been up for
// StableAfter, KeepBackup is off, and no request is using or waiting on the backup.
func (h *Harness) releasable() bool {
	up := h.primaryUpAt.Load()
	if h.cfg.KeepBackup || up == 0 || time.Since(time.Unix(0, up)) < h.cfg.StableAfter {
		return false
	}
	for _, b := range h.router.backends {
		if b.Name == "colibri" && b.Busy() {
			return false
		}
	}
	last := h.lastDemand.Load()
	return last == 0 || time.Since(time.Unix(0, last)) >= h.cfg.BackupIdle
}

// runBackup runs Colibri as the backup until ctx ends: install, start, qualify, then
// watch it. A transient failure is retried with backoff; a permanent one (host too
// weak, no model fits, no python) ends it for good. On exit Colibri is stopped.
func (h *Harness) runBackup(ctx context.Context) {
	p := h.cfg.Backup
	failures := 0
	defer func() {
		h.backupUp.Store(false)
		if h.BackupState() != StateUnavailable {
			h.setBackupState(StateStopped)
		}
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := p.Stop(stopCtx); err != nil {
			h.logf("stopping colibri: %v", err)
		}
		cancel()
	}()

	for ctx.Err() == nil {
		h.setBackupState(StateInstalling)
		err := p.Install(ctx)
		if err == nil {
			h.setBackupState(StateStarting)
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
			h.backupNote.Store(err.Error())
			if IsPermanent(err) {
				h.backupDead.Store(true)
				h.setBackupState(StateUnavailable)
				if errors.Is(err, ErrHostTooWeak) {
					h.logf("%v", err)
				} else {
					h.logf("colibri is unavailable and will not be retried: %v", err)
				}
				return
			}
			failures++
			h.setBackupState(StateFailed)
			wait := backoffAt(h.cfg.RetryBackoff, failures)
			h.logf("colibri failed: %v; retrying in %s", err, wait)
			if !sleep(ctx, wait) {
				return
			}
			continue
		}
		failures = 0
		h.backupNote.Store("")
		h.backupUp.Store(true)
		h.setBackupState(StateServing)
		h.logf("colibri is serving (model %s)", p.ChosenModel())
		if !h.watchBackup(ctx) {
			return
		}
		// It went down: bring it back.
		h.backupUp.Store(false)
		h.backupNote.Store("colibri stopped answering")
		h.setBackupState(StateFailed)
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		p.Stop(stopCtx)
		cancel()
	}
}

// watchBackup returns true when Colibri went down, false when ctx ended.
func (h *Harness) watchBackup(ctx context.Context) bool {
	bad := 0
	tick := time.NewTicker(h.cfg.MonitorEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-tick.C:
		}
		if h.cfg.Backup.Healthy() {
			bad = 0
			continue
		}
		bad++
		if bad >= h.cfg.FailuresBeforeDown {
			h.logf("colibri failed %d health checks", bad)
			return true
		}
	}
}

// State is where a harness is in its lifecycle, as the status document reports it.
// llama.cpp (primary_state) uses starting, serving, failed and unavailable. Colibri
// (colibri_state) uses all of them, plus installing while it downloads and stopped
// once it has been released.
type State string

const (
	StateInstalling  State = "installing"  // Colibri only: Install is running (download, build)
	StateStarting    State = "starting"    // the server is loading its model, or restarting
	StateServing     State = "serving"     // answering requests
	StateFailed      State = "failed"      // the last attempt failed; the harness is retried
	StateUnavailable State = "unavailable" // out for good, or not configured
	StateStopped     State = "stopped"     // Colibri only: not running (released, or never needed)
)

func (h *Harness) setPrimaryState(s State) { h.primaryState.Store(s) }
func (h *Harness) setBackupState(s State)  { h.backupState.Store(s) }

// PrimaryState reports where llama.cpp, the primary, is right now.
func (h *Harness) PrimaryState() State { return loadState(&h.primaryState) }

// BackupState reports where Colibri, the backup, is right now.
func (h *Harness) BackupState() State { return loadState(&h.backupState) }

func loadState(v *atomic.Value) State {
	if s, ok := v.Load().(State); ok {
		return s
	}
	return ""
}

// noteDemand records that a request waited on a backend, which holds the backup up
// for BackupIdle after its last use.
func (h *Harness) noteDemand(string) { h.lastDemand.Store(time.Now().UnixNano()) }

func isOpen(c chan struct{}) bool {
	select {
	case <-c:
		return false
	default:
		return true
	}
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
