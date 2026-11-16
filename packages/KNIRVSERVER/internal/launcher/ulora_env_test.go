package launcher

import (
	"path/filepath"
	"strings"
	"testing"
)

// uloraServiceEnv is the env line source for ulorad's data directory. The launcher
// passes it to backend_server; the value must be <app data>/ulora, the directory
// KNIRVGRAPH and ulorad both read the auth_token file from.
func TestUloraServiceEnvPinsDataDirUnderAppData(t *testing.T) {
	app := filepath.Join(t.TempDir(), "knirvserver")
	env := uloraServiceEnv(app, "fake-token-test-only")

	want := map[string]string{
		"ULORA_AUTH_TOKEN":  "fake-token-test-only",
		"ULORA_SOCKET_PATH": filepath.Join(app, "sockets", "ulora.sock"),
		"ULORA_DATA_DIR":    filepath.Join(app, "ulora"),
		"ULORA_BINARY_PATH": filepath.Join(app, "bin", "ulorad"),
	}
	if len(env) != len(want) {
		t.Fatalf("env = %v, want %d entries", env, len(want))
	}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		if exp, ok := want[k]; !ok || exp != v {
			t.Fatalf("unexpected env entry %q", k)
		}
	}
}

// With a writable system app data dir, getAppDataDir must return it rather than
// the XDG or home fallbacks. XDG is set to a different path so a fallback fails.
func TestGetAppDataDirPrefersWritableSystemDir(t *testing.T) {
	oldSystemDir := defaultSystemAppDataDir
	t.Cleanup(func() { defaultSystemAppDataDir = oldSystemDir })

	system := filepath.Join(t.TempDir(), "knirvserver")
	defaultSystemAppDataDir = system
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	got, err := getAppDataDir()
	if err != nil {
		t.Fatalf("getAppDataDir: %v", err)
	}
	if got != system {
		t.Fatalf("getAppDataDir = %q, want writable system dir %q", got, system)
	}
}
