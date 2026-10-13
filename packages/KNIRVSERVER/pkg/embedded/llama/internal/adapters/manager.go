package adapters

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Errors a caller maps to HTTP statuses.
var (
	ErrInvalid      = errors.New("invalid adapter")
	ErrBaseMismatch = errors.New("adapter was bound to a different base model")
	ErrNotFound     = errors.New("adapter not found")
	ErrUnmanaged    = errors.New("llama-server is not managed by this process, so its adapter set cannot be changed")
)

// Base describes the model llama-server runs.
type Base struct {
	Model string
	Path  string
	// SHA256 is computed on first use (a 4 GB model takes seconds) and must equal
	// the base_sha256 ULoRA recorded in an adapter.
	SHA256 func() (string, error)
}

// Restarter restarts llama-server with exactly this adapter set, in order (index =
// llama adapter id), and returns once it is healthy with every adapter at scale 0.
type Restarter func(ctx context.Context, set []Entry) error

// Manager owns the adapter set and its relation to the running llama-server.
type Manager struct {
	store   *Store
	base    Base
	restart Restarter
	gate    *Gate

	// DrainTimeout bounds how long a change waits for in-flight requests.
	DrainTimeout time.Duration

	mu     sync.RWMutex
	loaded []Entry // what llama-server actually has; index = adapter id
	reload sync.Mutex
}

// NewManager returns a manager over the stored set. Call Restore once to start
// llama-server with it. A nil restart means llama-server is not ours to restart:
// reads work, changes are refused with ErrUnmanaged.
func NewManager(store *Store, base Base, restart Restarter, gate *Gate) *Manager {
	return &Manager{store: store, base: base, restart: restart, gate: gate, DrainTimeout: 60 * time.Second}
}

// Gate is what request handlers pass through.
func (m *Manager) Gate() *Gate { return m.gate }

// Base reports the base model.
func (m *Manager) Base() Base { return m.base }

// Loaded is the adapter set llama-server is running, with ids.
func (m *Manager) Loaded() []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Entry(nil), m.loaded...)
}

// SetLoaded records the set llama-server was started with (used at startup).
func (m *Manager) SetLoaded(set []Entry) {
	m.mu.Lock()
	m.loaded = append([]Entry(nil), set...)
	m.mu.Unlock()
}

// Resolve maps an adapter name to its llama id.
func (m *Manager) Resolve(name string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i, e := range m.loaded {
		if e.Name == name {
			return i, true
		}
	}
	return 0, false
}

// Registered returns the stored set (what will be loaded after the next reload).
func (m *Manager) Registered() []Entry { return m.store.List() }

// Register validates an adapter file against the base model and adds it.
//
// Refusals: not a LoRA adapter, wrong architecture, or bound to a different base
// model (sha256). An adapter with no recorded base hash is refused too: it may be
// shape-compatible with this model and still meaningless on it.
func (m *Manager) Register(ctx context.Context, name, path string) (Entry, error) {
	if !ValidName(name) {
		return Entry{}, fmt.Errorf("%w: name must match [a-z0-9][a-z0-9._-]{0,63}", ErrInvalid)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return Entry{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		return Entry{}, fmt.Errorf("%w: %s is not a readable file", ErrInvalid, path)
	}
	info, err := ReadInfo(path)
	if err != nil {
		return Entry{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if info.Type != "adapter" || info.AdapterType != "lora" {
		return Entry{}, fmt.Errorf("%w: %s is not a LoRA adapter (general.type=%q adapter.type=%q)", ErrInvalid, path, info.Type, info.AdapterType)
	}
	if info.BaseSHA256 == "" {
		return Entry{}, fmt.Errorf("%w: %s records no base model hash (knirv.base_sha256); write it with ULoRA so it can be matched to its base", ErrInvalid, path)
	}
	want, err := m.base.SHA256()
	if err != nil {
		return Entry{}, fmt.Errorf("hash base model: %w", err)
	}
	if !strings.EqualFold(info.BaseSHA256, want) {
		return Entry{}, fmt.Errorf("%w: adapter %s was bound to base %s…, but llama-server runs %s…", ErrBaseMismatch, name, short(info.BaseSHA256), short(want))
	}
	e := Entry{Name: name, Path: path, BaseSHA256: info.BaseSHA256, BundleHash: info.BundleHash, Skill: info.Skill, Alpha: info.Alpha}

	next := m.store.List()
	replaced := false
	for i := range next {
		if next[i].Name == name {
			next[i], replaced = e, true
		}
	}
	if !replaced {
		next = append(next, e)
	}
	if err := m.apply(ctx, next); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// Remove drops an adapter and restarts llama-server without it.
func (m *Manager) Remove(ctx context.Context, name string) error {
	var next []Entry
	found := false
	for _, e := range m.store.List() {
		if e.Name == name {
			found = true
			continue
		}
		next = append(next, e)
	}
	if !found {
		return fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	return m.apply(ctx, next)
}

// Restore starts llama-server with the stored set; call once at startup.
func (m *Manager) Restore(ctx context.Context) error {
	if m.restart == nil {
		return nil
	}
	var set []Entry
	for _, e := range m.store.List() {
		if _, err := os.Stat(e.Path); err != nil {
			log.Printf("[KNIRVLLAMA] adapter %q is registered but %s is gone; not loading it", e.Name, e.Path)
			continue
		}
		set = append(set, e)
	}
	if err := m.restart(ctx, set); err != nil {
		if len(set) == 0 {
			return err
		}
		// One bad adapter must not stop the base model from serving.
		log.Printf("[KNIRVLLAMA] llama-server would not start with the stored adapters (%v); starting without them", err)
		if err := m.restart(ctx, nil); err != nil {
			return err
		}
		set = nil
	}
	m.SetLoaded(set)
	return nil
}

// apply swaps the live set: drain, restart with the new set, and on failure restart
// with the previous one so a bad adapter cannot take the model down with it.
func (m *Manager) apply(ctx context.Context, next []Entry) error {
	if m.restart == nil {
		return ErrUnmanaged
	}
	m.reload.Lock()
	defer m.reload.Unlock()

	resume, err := m.gate.Pause(m.DrainTimeout)
	if err != nil {
		return err
	}
	defer resume()

	prev := m.Loaded()
	if err := m.restart(ctx, next); err != nil {
		log.Printf("[KNIRVLLAMA] adapter reload failed (%v); restoring %d previous adapter(s)", err, len(prev))
		if rerr := m.restart(ctx, prev); rerr != nil {
			return fmt.Errorf("reload failed (%v) and restoring the previous set also failed: %w", err, rerr)
		}
		return fmt.Errorf("llama-server rejected the adapter set (previous set restored): %w", err)
	}
	if err := m.store.Set(next); err != nil {
		return fmt.Errorf("adapters are loaded but the registry could not be saved: %w", err)
	}
	m.SetLoaded(next)
	return nil
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
