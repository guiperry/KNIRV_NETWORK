package knirvsearch

import (
	"context"
	"os"
	"testing"
)

// fakeChild is a no-real-binary Child used by the tests. It records lifecycle
// calls and reports a configurable health result, exercising the manager
// behavior without executing any system binary.
type fakeChild struct {
	healthy bool
	started bool
	stopped bool
}

func (f *fakeChild) Start(ctx context.Context) error {
	f.started = true
	return nil
}

func (f *fakeChild) Stop() error {
	f.stopped = true
	return nil
}

func (f *fakeChild) Healthy(ctx context.Context) (bool, error) {
	return f.healthy, nil
}

// fakeFactory returns a fakeChild (and records that the factory was invoked).
func fakeFactory(healthy bool) (ChildFactory, *fakeChild, *bool) {
	called := false
	child := &fakeChild{healthy: healthy}
	return func(ctx context.Context, cfg Config) (Child, error) {
		called = true
		// Return a fresh copy each call so Start/Stop state is per-child.
		return &fakeChild{healthy: child.healthy}, nil
	}, child, &called
}

// TestManagerStartsHealthyWithFakeBinary: flash ON + a provided binary -> the
// manager starts a (fake) child and reports Healthy.
func TestManagerStartsHealthyWithFakeBinary(t *testing.T) {
	cfg := Config{Enabled: true, Binary: "/fake/searxng", MaxRetry: 3}
	m := New(cfg)

	factory, _, called := fakeFactory(true)
	m.ChildFactory = factory

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if !*called {
		t.Error("expected ChildFactory to be invoked when a binary is provided")
	}

	st := m.Status()
	if st.State != StateHealthy {
		t.Errorf("expected StateHealthy, got %s (%s)", st.State, st.Reason)
	}
	if !st.Managed {
		t.Error("expected Managed=true when a binary is provided and healthy")
	}
	if st.Enabled != true {
		t.Error("expected Enabled=true")
	}
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

// TestManagerDegradedWithoutBinary: flash ON but no binary/URL -> Degraded, no
// error, and crucially the child factory is NEVER called (no service started).
func TestManagerDegradedWithoutBinary(t *testing.T) {
	cfg := Config{Enabled: true} // no Binary, no URL
	m := New(cfg)

	factory, _, called := fakeFactory(true)
	m.ChildFactory = factory

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start should not return an error in the degraded case: %v", err)
	}
	if *called {
		t.Error("ChildFactory must NOT be called when no binary/URL is configured")
	}

	st := m.Status()
	if st.State != StateDegraded {
		t.Errorf("expected StateDegraded, got %s", st.State)
	}
	if st.Managed {
		t.Error("expected Managed=false in degraded case")
	}
	if st.Reason == "" {
		t.Error("expected a Reason explaining the degraded state")
	}
}

// TestManagerDisabledWhenFlashOff: flash OFF -> Disabled, factory not called.
func TestManagerDisabledWhenFlashOff(t *testing.T) {
	cfg := Config{Enabled: false, Binary: "/fake/searxng"}
	m := New(cfg)

	factory, _, called := fakeFactory(true)
	m.ChildFactory = factory

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if *called {
		t.Error("ChildFactory must NOT be called when flash is disabled")
	}

	st := m.Status()
	if st.State != StateDisabled {
		t.Errorf("expected StateDisabled, got %s", st.State)
	}
	if st.Enabled {
		t.Error("expected Enabled=false")
	}
}

// TestManagerDegradesOnUnhealthyChild: flash ON + binary provided, but the child
// never becomes healthy -> Degraded, no error, child stopped.
func TestManagerDegradesOnUnhealthyChild(t *testing.T) {
	cfg := Config{Enabled: true, Binary: "/fake/searxng", MaxRetry: 2}
	m := New(cfg)

	factory, _, called := fakeFactory(false) // unhealthy
	m.ChildFactory = factory

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start should not error on unhealthy child: %v", err)
	}
	if !*called {
		t.Error("expected ChildFactory to be invoked")
	}

	st := m.Status()
	if st.State != StateDegraded {
		t.Errorf("expected StateDegraded, got %s (%s)", st.State, st.Reason)
	}
}

// TestConfigFromEnvFlashDefaultsOn: flash defaults ON; KNIRV_FLASH=0 and
// KNIRV_NO_FLASH=1 both turn it off (NO_FLASH wins).
func TestConfigFromEnvFlashDefaultsOn(t *testing.T) {
	// Default (no env): ON.
	os.Unsetenv(EnvFlash)
	os.Unsetenv(EnvNoFlash)
	if !EnabledFromEnv() {
		t.Error("flash should default to ON when no env is set")
	}

	// KNIRV_FLASH=0 -> OFF.
	t.Setenv(EnvFlash, "0")
	if EnabledFromEnv() {
		t.Error("KNIRV_FLASH=0 should disable flash")
	}

	// KNIRV_NO_FLASH=1 wins even if KNIRV_FLASH=1.
	t.Setenv(EnvFlash, "1")
	t.Setenv(EnvNoFlash, "1")
	if EnabledFromEnv() {
		t.Error("KNIRV_NO_FLASH=1 should win over KNIRV_FLASH=1")
	}

	os.Unsetenv(EnvFlash)
	os.Unsetenv(EnvNoFlash)
}

// TestConfigFromEnvReadsSearchURL: KNIRV_SEARCH_URL and KNIRV_METASEARCH_URL both
// populate Config.URL (SEARCH_URL takes precedence).
func TestConfigFromEnvReadsSearchURL(t *testing.T) {
	os.Unsetenv(EnvSearchURL)
	os.Unsetenv(EnvMetasearchURL)
	os.Unsetenv(EnvSearchBinary)

	t.Setenv(EnvMetasearchURL, "http://127.0.0.1:8080")
	cfg := ConfigFromEnv()
	if cfg.URL != "http://127.0.0.1:8080" {
		t.Errorf("expected URL from KNIRV_METASEARCH_URL, got %q", cfg.URL)
	}

	t.Setenv(EnvSearchURL, "http://10.0.0.1:9090")
	cfg = ConfigFromEnv()
	if cfg.URL != "http://10.0.0.1:9090" {
		t.Errorf("expected KNIRV_SEARCH_URL to take precedence, got %q", cfg.URL)
	}

	t.Setenv(EnvSearchBinary, "/usr/bin/searxng")
	cfg = ConfigFromEnv()
	if cfg.Binary != "/usr/bin/searxng" {
		t.Errorf("expected Binary from KNIRV_SEARCH_BINARY, got %q", cfg.Binary)
	}

	os.Unsetenv(EnvSearchURL)
	os.Unsetenv(EnvMetasearchURL)
	os.Unsetenv(EnvSearchBinary)
}
