package knirvgraph

import (
	"path/filepath"
	"strings"
	"testing"
)

// lastValue returns the value a child process sees for key. On duplicate keys the
// last entry wins, which is how exec's environment is interpreted.
func lastValue(env []string, key string) (string, bool) {
	val, found := "", false
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if ok && k == key {
			val, found = v, true
		}
	}
	return val, found
}

func testGraphConfig() *ManagerConfig {
	return &ManagerConfig{
		Port:     7001,
		P2PPort:  7002,
		APIPort:  7003,
		DataPath: "/data/graph",
	}
}

// Start hands the child the ulorad data directory for the launcher's app data dir,
// even when the inherited environment carries a stale copy.
func TestGraphChildEnvSetsUloraDataDirOverInheritedCopy(t *testing.T) {
	app := filepath.Join(t.TempDir(), "app")
	inherited := []string{"ULORA_DATA_DIR=/stale/ulora", "PATH=/usr/bin"}

	env := graphChildEnv(testGraphConfig(), inherited, app)

	if got, _ := lastValue(env, "ULORA_DATA_DIR"); got != filepath.Join(app, "ulora") {
		t.Fatalf("ULORA_DATA_DIR = %q, want %q", got, filepath.Join(app, "ulora"))
	}
	if got, _ := lastValue(env, "KNIRV_APP_DATA_DIR"); got != app {
		t.Fatalf("KNIRV_APP_DATA_DIR = %q, want %q", got, app)
	}
	if got, _ := lastValue(env, "PATH"); got != "/usr/bin" {
		t.Fatalf("inherited PATH lost: %q", got)
	}
}

func TestGraphChildEnvCarriesPortsAndDataPath(t *testing.T) {
	env := graphChildEnv(testGraphConfig(), nil, t.TempDir())
	want := map[string]string{
		"KNIRVGRAPH_PORT":      "7001",
		"KNIRVGRAPH_P2P_PORT":  "7002",
		"KNIRVGRAPH_API_PORT":  "7003",
		"KNIRVGRAPH_DATA_PATH": "/data/graph",
	}
	for k, v := range want {
		if got, ok := lastValue(env, k); !ok || got != v {
			t.Fatalf("%s = %q (present %v), want %q", k, got, ok, v)
		}
	}
}

// EnvOverrides come last and can override the ulorad data directory.
func TestGraphChildEnvEnvOverridesWinLast(t *testing.T) {
	cfg := testGraphConfig()
	cfg.EnvOverrides = map[string]string{"ULORA_DATA_DIR": "/operator/ulora"}

	env := graphChildEnv(cfg, nil, filepath.Join(t.TempDir(), "app"))

	if got, _ := lastValue(env, "ULORA_DATA_DIR"); got != "/operator/ulora" {
		t.Fatalf("ULORA_DATA_DIR = %q, want operator override", got)
	}
}
