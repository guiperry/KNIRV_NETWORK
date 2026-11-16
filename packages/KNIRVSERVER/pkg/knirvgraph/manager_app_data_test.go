package knirvgraph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Each test isolates the inputs of getAppDataDir so it never touches a real
// /var/lib/knirvserver or the developer's home directory.
func isolateAppDataEnv(t *testing.T) (xdg string) {
	t.Helper()
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	origSystem := defaultSystemAppDataDir
	defaultSystemAppDataDir = filepath.Join(blocker, "knirvserver")
	t.Cleanup(func() { defaultSystemAppDataDir = origSystem })
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	t.Setenv("HOME", filepath.Join(root, "home"))
	xdg = filepath.Join(root, "xdg")
	t.Setenv("XDG_DATA_HOME", xdg)
	return xdg
}

func TestGetAppDataDirFallsBackToXDGWhenSystemDirUnusable(t *testing.T) {
	xdg := isolateAppDataEnv(t)
	if got, want := getAppDataDir(), filepath.Join(xdg, "knirvserver"); got != want {
		t.Fatalf("getAppDataDir = %q, want XDG %q", got, want)
	}
}

func TestGetAppDataDirPrefersWritableSystemDir(t *testing.T) {
	isolateAppDataEnv(t)
	system := filepath.Join(t.TempDir(), "knirvserver")
	defaultSystemAppDataDir = system
	if got := getAppDataDir(); got != system {
		t.Fatalf("getAppDataDir = %q, want writable system dir %q", got, system)
	}
}

func TestGetAppDataDirHonoursExplicitEnv(t *testing.T) {
	isolateAppDataEnv(t)
	explicit := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", explicit)
	if got := getAppDataDir(); got != explicit {
		t.Fatalf("getAppDataDir = %q, want explicit %q", got, explicit)
	}
}

// The KNIRVGRAPH child is told its ulorad data directory explicitly, and that is
// the same <app data dir>/ulora the launcher gives ulorad.
func TestGraphChildUloraEnvNamesUloraDataDir(t *testing.T) {
	app := filepath.Join(t.TempDir(), "app")
	env := graphChildUloraEnv(app)
	want := map[string]string{
		"KNIRV_APP_DATA_DIR": app,
		"ULORA_DATA_DIR":     filepath.Join(app, "ulora"),
	}
	if len(env) != len(want) {
		t.Fatalf("env = %v, want exactly %d entries", env, len(want))
	}
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || want[k] != v {
			t.Fatalf("unexpected env entry %q", kv)
		}
	}
}
