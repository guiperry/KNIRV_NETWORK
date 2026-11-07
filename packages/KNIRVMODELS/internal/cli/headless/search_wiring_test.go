package headless

import (
	"testing"
	"time"

	"knirvmodels/pkg/knirvsearch"
)

// TestServerWiresSearchManagerFlashOn verifies that, with flash on and no
// binary/URL configured, the daemon constructs a knirvsearch.Manager, starts it,
// and lands in Degraded (no system service started, factslice fallback in use).
func TestServerWiresSearchManagerFlashOn(t *testing.T) {
	t.Setenv("KNIRV_FLASH", "1")
	t.Setenv("KNIRV_SEARCH_BINARY", "")
	t.Setenv("KNIRV_SEARCH_URL", "")
	t.Setenv("KNIRV_METASEARCH_URL", "")

	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	if err := s.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if s.search == nil {
		t.Fatal("server did not create a knirvsearch.Manager")
	}
	if !s.search.Status().Enabled {
		t.Error("expected search manager Enabled=true when flash is on")
	}

	// Give the background Start() a moment to reach its terminal state.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st := s.search.Status()
		if st.State == knirvsearch.StateDegraded || st.State == knirvsearch.StateHealthy {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if st := s.search.Status(); st.State != knirvsearch.StateDegraded {
		t.Errorf("expected Degraded (no binary/URL), got %s", st.State)
	}
}

// TestServerWiresSearchManagerFlashOff verifies that with flash off the daemon
// leaves the manager Disabled and does not start anything.
func TestServerWiresSearchManagerFlashOff(t *testing.T) {
	t.Setenv("KNIRV_FLASH", "0")

	socketPath := testSocketPath(t)
	s := NewServer(socketPath, 0660, false)
	defer s.Stop()

	if err := s.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if s.search == nil {
		t.Fatal("server did not create a knirvsearch.Manager")
	}
	if s.search.Status().Enabled {
		t.Error("expected search manager Enabled=false when flash is off")
	}

	// Start() runs in a goroutine; poll briefly for it to set the terminal state.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.search.Status().State == knirvsearch.StateDisabled {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if s.search.Status().State != knirvsearch.StateDisabled {
		t.Errorf("expected StateDisabled, got %s", s.search.Status().State)
	}
}
