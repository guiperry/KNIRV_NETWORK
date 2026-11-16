package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

// When the system config dir exists but is not writable (a non-root user with a
// root-owned /etc/knirv-server), getConfigDir must fall back to the user config dir.
func TestGetConfigDirFallsBackWhenSystemDirIsNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write a 0555 directory")
	}
	oldSystemDir := defaultSystemConfigDir
	t.Cleanup(func() { defaultSystemConfigDir = oldSystemDir })

	systemDir := t.TempDir()
	if err := os.Chmod(systemDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(systemDir, 0755) })
	defaultSystemConfigDir = systemDir

	xdgConfig := t.TempDir()
	t.Setenv("KNIRV_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	got, err := getConfigDir()
	if err != nil {
		t.Fatalf("getConfigDir: %v", err)
	}
	if want := filepath.Join(xdgConfig, "knirv-server"); got != want {
		t.Fatalf("getConfigDir = %q, want the user fallback %q", got, want)
	}
}

// A writable system config dir is used as-is (the override is honoured).
func TestGetConfigDirUsesWritableSystemDir(t *testing.T) {
	oldSystemDir := defaultSystemConfigDir
	t.Cleanup(func() { defaultSystemConfigDir = oldSystemDir })

	systemDir := filepath.Join(t.TempDir(), "knirv-server")
	defaultSystemConfigDir = systemDir
	t.Setenv("KNIRV_CONFIG_DIR", "")

	got, err := getConfigDir()
	if err != nil {
		t.Fatalf("getConfigDir: %v", err)
	}
	if got != systemDir {
		t.Fatalf("getConfigDir = %q, want the writable system dir %q", got, systemDir)
	}
}

// ulorad, the launcher env, and the KNIRVGRAPH reader all use <app data>/ulora.
func TestUloraDataDirIsAppDataUlora(t *testing.T) {
	app := "/var/lib/knirvserver"
	if got, want := uloraDataDir(app), filepath.Join(app, "ulora"); got != want {
		t.Fatalf("uloraDataDir = %q, want %q", got, want)
	}
}
