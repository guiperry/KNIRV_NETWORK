package keyfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRootKeyPathUsesConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	path, err := GetRootKeyPath()
	if err != nil {
		t.Fatalf("GetRootKeyPath returned error: %v", err)
	}

	want := filepath.Join(filepath.Clean(getenvOrFatal(t, "XDG_CONFIG_HOME")), "knirv-server", "root.key")
	if path != want {
		t.Fatalf("GetRootKeyPath = %q, want %q", path, want)
	}
}

func getenvOrFatal(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("%s is unexpectedly empty", key)
	}
	return value
}
