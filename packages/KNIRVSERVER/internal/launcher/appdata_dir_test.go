package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

// mkdirIfUsable must not treat an existing but unwritable directory as usable.
// os.MkdirAll returns nil for an existing root-owned /var/lib/knirvserver, which
// used to win over the user fallback and break provisioning on the first write.
func TestMkdirIfUsableProbesWritability(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write a 0555 directory")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
	if mkdirIfUsable(dir) {
		t.Fatal("reported a read-only directory as usable")
	}
}

// When the system data dir exists but is not writable, getAppDataDir must fall back
// to XDG_DATA_HOME instead of returning the unusable system path.
func TestGetAppDataDirFallsBackWhenSystemDirIsNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write a 0555 directory")
	}
	oldSystemDir := defaultSystemAppDataDir
	t.Cleanup(func() { defaultSystemAppDataDir = oldSystemDir })

	systemDir := t.TempDir()
	if err := os.Chmod(systemDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(systemDir, 0755) })
	defaultSystemAppDataDir = systemDir

	xdg := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", xdg)

	got, err := getAppDataDir()
	if err != nil {
		t.Fatalf("getAppDataDir: %v", err)
	}
	want := filepath.Join(xdg, "knirvserver")
	if got != want {
		t.Fatalf("getAppDataDir = %q, want the XDG fallback %q", got, want)
	}
	if !mkdirIfUsable(got) {
		t.Fatalf("fallback %q is not writable", got)
	}
}

func TestGetAppDataDirHonorsExplicitOverride(t *testing.T) {
	explicit := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", explicit)
	got, err := getAppDataDir()
	if err != nil {
		t.Fatalf("getAppDataDir: %v", err)
	}
	if got != explicit {
		t.Fatalf("getAppDataDir = %q, want KNIRV_APP_DATA_DIR %q", got, explicit)
	}
}
