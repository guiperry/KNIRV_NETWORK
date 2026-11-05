package authority

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// The SDK runs its Rust core as a WASM module that Go loads, digest-pinned, from
// a data directory or a release URL. A locally built core is not published, so
// these tests seed a throwaway data directory from KNIRVSDK/dist the same way the
// SDK's own `make test-go` does. KNIRV_SDK_DATA_DIR, when already set, wins.
func TestMain(m *testing.M) {
	cleanup := func() {}
	if os.Getenv("KNIRV_SDK_DATA_DIR") == "" {
		if dir, ok := seedCore(); ok {
			os.Setenv("KNIRV_SDK_DATA_DIR", dir)
			cleanup = func() { os.RemoveAll(dir) }
		}
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func seedCore() (string, bool) {
	matches, _ := filepath.Glob(filepath.Join("..", "..", "..", "KNIRVSDK", "dist", "*", "knirv_core.wasm"))
	if len(matches) == 0 {
		return "", false
	}
	sort.Slice(matches, func(i, j int) bool {
		a, _ := os.Stat(matches[i])
		b, _ := os.Stat(matches[j])
		return a.ModTime().After(b.ModTime())
	})
	src := matches[0]
	version := filepath.Base(filepath.Dir(src))
	dir, err := os.MkdirTemp("", "authority-core-")
	if err != nil {
		return "", false
	}
	if err := os.MkdirAll(filepath.Join(dir, version), 0o700); err != nil {
		return "", false
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", false
	}
	if err := os.WriteFile(filepath.Join(dir, version, "knirv_core.wasm"), data, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "seed core:", err)
		return "", false
	}
	return dir, true
}
