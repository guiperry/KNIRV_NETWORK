package drq

import (
	"path/filepath"
	"testing"
)

// resolveAppDataDir must follow the launcher's order: KNIRV_APP_DATA_DIR, then the
// system dir when writable, then XDG_DATA_HOME, then ~/.local/share. Each test
// pins one step of that order.

func TestResolveAppDataDirExplicitEnvWins(t *testing.T) {
	isolateULoRAAuthEnv(t)
	writable := t.TempDir()
	origSystem := defaultSystemAppDataDir
	defaultSystemAppDataDir = writable
	t.Cleanup(func() { defaultSystemAppDataDir = origSystem })
	explicit := filepath.Join(t.TempDir(), "explicit")
	t.Setenv(knirvAppDataDirEnv, explicit)

	got, err := resolveAppDataDir()
	if err != nil || got != explicit {
		t.Fatalf("explicit KNIRV_APP_DATA_DIR must win: got %q, err %v", got, err)
	}
}

// The system dir is preferred over XDG when it is writable. The XDG dir is set
// too, so a fallback to XDG would fail this test.
func TestResolveAppDataDirPrefersWritableSystemDir(t *testing.T) {
	xdg := isolateULoRAAuthEnv(t)
	system := filepath.Join(t.TempDir(), "knirvserver")
	origSystem := defaultSystemAppDataDir
	defaultSystemAppDataDir = system
	t.Cleanup(func() { defaultSystemAppDataDir = origSystem })
	t.Setenv("XDG_DATA_HOME", xdg)

	got, err := resolveAppDataDir()
	if err != nil || got != system {
		t.Fatalf("writable system dir must be preferred over XDG: got %q, err %v", got, err)
	}
}

// When the system dir is unusable (isolateULoRAAuthEnv blocks it), XDG is used.
func TestResolveAppDataDirFallsBackToXDGWhenSystemUnusable(t *testing.T) {
	xdg := isolateULoRAAuthEnv(t)

	got, err := resolveAppDataDir()
	want := filepath.Join(xdg, "knirvserver")
	if err != nil || got != want {
		t.Fatalf("XDG fallback: got %q, want %q, err %v", got, want, err)
	}
}
