package models

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// The frame registry runs in the SDK's Rust core, loaded digest-pinned. A locally
// built core is not published, so seed a throwaway data directory from KNIRVSDK/dist,
// as the other packages' tests do. KNIRV_SDK_DATA_DIR, when set, wins.
func TestMain(m *testing.M) {
	cleanup := func() {}
	if os.Getenv("KNIRV_SDK_DATA_DIR") == "" {
		if matches, _ := filepath.Glob(filepath.Join("..", "..", "..", "KNIRVSDK", "dist", "*", "knirv_core.wasm")); len(matches) > 0 {
			sort.Slice(matches, func(i, j int) bool {
				a, _ := os.Stat(matches[i])
				b, _ := os.Stat(matches[j])
				return a.ModTime().After(b.ModTime())
			})
			if dir, err := os.MkdirTemp("", "models-core-"); err == nil {
				v := filepath.Base(filepath.Dir(matches[0]))
				if data, err := os.ReadFile(matches[0]); err == nil && os.MkdirAll(filepath.Join(dir, v), 0o700) == nil &&
					os.WriteFile(filepath.Join(dir, v, "knirv_core.wasm"), data, 0o600) == nil {
					os.Setenv("KNIRV_SDK_DATA_DIR", dir)
					cleanup = func() { os.RemoveAll(dir) }
				}
			}
		}
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
