// Package knirvsearch implements the flash metasearch manager (spec item 3.8).
//
// The manager owns the lifecycle of an optional, self-hosted metasearch backend
// (SearXNG-compatible). It is gated behind the "flash" flag (KNIRV_FLASH, default
// ON, owner decision O5).
//
// Hard rules:
//   - It NEVER starts a system service. It only ever launches a child process the
//     deployer explicitly points at via KNIRV_SEARCH_BINARY, or probes a URL the
//     deployer already runs (e.g. KNIRV_METASEARCH_URL).
//   - When neither a binary nor a URL is configured it reports StateDegraded and
//     leaves the local factslice verification fallback in use. It does not error
//     and it does not crash the daemon.
//
// The child handle is produced through Manager.ChildFactory so tests can inject a
// fake child with no real binary being executed.
package knirvsearch

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// State is the lifecycle state reported by Status().
type State string

const (
	// StateDisabled means flash is off; the manager does nothing.
	StateDisabled State = "Disabled"
	// StateDegraded means flash is on but no usable backend is available; the
	// factslice fallback stays in use.
	StateDegraded State = "Degraded"
	// StateStarting means a child is being launched / probed.
	StateStarting State = "Starting"
	// StateHealthy means a backend is up and passing health probes.
	StateHealthy State = "Healthy"
	// StateStopped means Start has not run or Stop has been called.
	StateStopped State = "Stopped"
)

// Config configures the manager. All values may be populated directly or via
// ConfigFromEnv().
type Config struct {
	// Enabled is the flash switch. When false the manager stays Disabled.
	Enabled bool
	// Binary is the path to a metasearch child process the deployer wants the
	// manager to launch and supervise. Empty means "do not launch anything".
	Binary string
	// Args are the arguments passed to Binary when launched.
	Args []string
	// URL is the base URL of a metasearch backend already running (loopback or
	// unix socket). Used for health probing and as the search target.
	URL string
	// Health is the path appended to URL for the health probe (e.g. "/healthz").
	Health string
	// MaxRetry is how many times Start probes health before giving up (Degraded).
	MaxRetry int
	// HealthProbe is the HTTP method used for the health probe (default GET).
	HealthProbe string
}

// Status is a point-in-time snapshot of the manager.
type Status struct {
	Enabled bool
	State   State
	Reason  string
	URL     string
	// Managed is true when the manager launched and supervises a child process.
	Managed bool
}

// Child is the handle the manager uses to control a search backend. The default
// production implementation (see child.go) launches a child process when a
// Binary is given, or probes an externally-running URL. Tests inject a fake via
// Manager.ChildFactory so no real binary is executed.
type Child interface {
	Start(ctx context.Context) error
	Stop() error
	Healthy(ctx context.Context) (bool, error)
}

// ChildFactory builds a Child from the config. The default is defaultChildFactory
// (production). Tests replace Manager.ChildFactory with a fake factory.
type ChildFactory func(ctx context.Context, cfg Config) (Child, error)

// Environment variable names read by ConfigFromEnv / EnabledFromEnv.
const (
	EnvFlash         = "KNIRV_FLASH"
	EnvNoFlash       = "KNIRV_NO_FLASH"
	EnvMetasearchURL = "KNIRV_METASEARCH_URL"
	EnvSearchBinary  = "KNIRV_SEARCH_BINARY"
	EnvSearchArgs    = "KNIRV_SEARCH_ARGS"
	EnvSearchURL     = "KNIRV_SEARCH_URL"
	EnvSearchHealth  = "KNIRV_SEARCH_HEALTH"
	EnvSearchProbe   = "KNIRV_SEARCH_HEALTH_PROBE"
	EnvSearchMaxRetry = "KNIRV_SEARCH_MAX_RETRY"
)

// EnabledFromEnv reports whether flash is on. Flash is ON by default (O5). It is
// turned off with KNIRV_FLASH=0|false|off|no or KNIRV_NO_FLASH=1|true|on|yes
// (the latter wins).
func EnabledFromEnv() bool {
	if truthy(os.Getenv(EnvNoFlash)) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvFlash))) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}

// ConfigFromEnv builds a Config from the environment. It mirrors the existing
// livesearch flash conventions and adds the dedicated search-backend knobs.
func ConfigFromEnv() Config {
	cfg := Config{Enabled: EnabledFromEnv()}

	if b := strings.TrimSpace(os.Getenv(EnvSearchBinary)); b != "" {
		cfg.Binary = b
	}
	if a := strings.TrimSpace(os.Getenv(EnvSearchArgs)); a != "" {
		cfg.Args = strings.Fields(a)
	}

	url := strings.TrimSpace(os.Getenv(EnvSearchURL))
	if url == "" {
		url = strings.TrimSpace(os.Getenv(EnvMetasearchURL))
	}
	cfg.URL = url

	if h := strings.TrimSpace(os.Getenv(EnvSearchHealth)); h != "" {
		cfg.Health = h
	}
	if p := strings.TrimSpace(os.Getenv(EnvSearchProbe)); p != "" {
		cfg.HealthProbe = p
	}
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(EnvSearchMaxRetry))); err == nil && n > 0 {
		cfg.MaxRetry = n
	}
	return cfg
}

// New constructs a Manager from cfg. MaxRetry defaults to 5 when unset.
func New(cfg Config) *Manager {
	if cfg.MaxRetry <= 0 {
		cfg.MaxRetry = 5
	}
	m := &Manager{
		cfg:    cfg,
		status: Status{Enabled: cfg.Enabled, State: StateStopped, URL: cfg.URL},
	}
	m.ChildFactory = defaultChildFactory
	return m
}

// NewFromEnv constructs a Manager from the environment.
func NewFromEnv() *Manager {
	return New(ConfigFromEnv())
}

// Manager owns the flash metasearch backend lifecycle.
type Manager struct {
	cfg         Config
	mu          sync.RWMutex
	status      Status
	child       Child
	started     bool
	ChildFactory ChildFactory
}

// Start brings the backend up according to the config. It never returns an error
// for the "nothing to do" cases (disabled, or no binary/URL); those are reported
// as StateDisabled / StateDegraded respectively. A failed child launch or health
// probe also degrades rather than erroring, so the daemon keeps running on the
// factslice fallback.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	cfg := m.cfg
	m.mu.RUnlock()

	if !cfg.Enabled {
		m.mu.Lock()
		m.status = Status{Enabled: false, State: StateDisabled, Reason: "flash disabled (KNIRV_FLASH=0 or KNIRV_NO_FLASH=1)"}
		m.mu.Unlock()
		log.Printf("[search] flash disabled; metasearch manager stays Disabled (factslice fallback in use)")
		return nil
	}

	if cfg.Binary == "" && cfg.URL == "" {
		m.mu.Lock()
		m.status = Status{
			Enabled: true,
			State:   StateDegraded,
			Reason:  "no SearXNG binary (KNIRV_SEARCH_BINARY) or URL (KNIRV_SEARCH_URL/KNIRV_METASEARCH_URL) configured; factslice fallback remains in use",
			Managed: false,
		}
		m.mu.Unlock()
		log.Printf("[search] no search binary/URL; staying Degraded (factslice fallback in use), not starting any service")
		return nil
	}

	m.mu.Lock()
	m.status.State = StateStarting
	m.mu.Unlock()

	child, err := m.ChildFactory(ctx, cfg)
	if err != nil {
		m.setDegraded(fmt.Sprintf("child factory error: %v", err))
		return nil
	}
	if err := child.Start(ctx); err != nil {
		m.setDegraded(fmt.Sprintf("child start error: %v", err))
		return nil
	}

	m.mu.Lock()
	m.child = child
	m.mu.Unlock()

	healthy := false
	for i := 0; i < cfg.MaxRetry; i++ {
		select {
		case <-ctx.Done():
			m.setDegraded("context cancelled during health probe")
			_ = child.Stop()
			m.mu.Lock()
			m.child = nil
			m.mu.Unlock()
			return ctx.Err()
		default:
		}
		ok, herr := child.Healthy(ctx)
		if ok {
			healthy = true
			break
		}
		log.Printf("[search] health probe attempt %d/%d failed: %v", i+1, cfg.MaxRetry, herr)
		time.Sleep(healthBackoff(i))
	}

	if healthy {
		m.mu.Lock()
		m.status = Status{
			Enabled: true,
			State:   StateHealthy,
			URL:     cfg.URL,
			Managed: cfg.Binary != "",
		}
		m.started = true
		m.mu.Unlock()
		log.Printf("[search] manager Healthy (managed=%t, url=%q)", cfg.Binary != "", cfg.URL)
		return nil
	}

	m.setDegraded("health probe failed after retries")
	_ = child.Stop()
	m.mu.Lock()
	m.child = nil
	m.mu.Unlock()
	return nil
}

// Status returns a snapshot of the manager.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// Stop stops a managed child (if any) and marks the manager Stopped. It is safe
// to call when nothing was started.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.child != nil {
		if err := m.child.Stop(); err != nil {
			log.Printf("[search] child stop error: %v", err)
		}
		m.child = nil
	}
	m.started = false
	m.status.State = StateStopped
	return nil
}

func (m *Manager) setDegraded(reason string) {
	m.mu.Lock()
	m.status = Status{Enabled: m.cfg.Enabled, State: StateDegraded, Reason: reason, URL: m.cfg.URL, Managed: false}
	m.mu.Unlock()
	log.Printf("[search] Degraded: %s", reason)
}

func healthBackoff(attempt int) time.Duration {
	d := 100 * time.Millisecond * time.Duration(attempt+1)
	if d > 2*time.Second {
		d = 2 * time.Second
	}
	return d
}

// truthy reports whether v is a positive truthy string.
func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}
