package api

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBinaryExists(t *testing.T) {
	if binaryExists("this-binary-should-never-exist-12345") {
		t.Fatal("expected false for nonexistent binary")
	}
	// `sh` exists on essentially every Linux host; if not, the env is exotic.
	if !binaryExists("sh") && runtime.GOOS == "linux" {
		t.Skip("sh not on PATH; skipping presence assertion")
	}
}

func TestDetectPackageManagerFromPath(t *testing.T) {
	dir := t.TempDir()
	pmDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(pmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dnfPath := filepath.Join(pmDir, "dnf")
	if err := os.WriteFile(dnfPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Isolate PATH so only the injected manager is discoverable.
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", pmDir)
	defer os.Setenv("PATH", oldPath)

	pm, err := detectPackageManager()
	if err != nil {
		t.Fatalf("expected to detect dnf, got err: %v", err)
	}
	if pm != "dnf" {
		t.Fatalf("expected dnf, got %s", pm)
	}
}

func TestDetectPackageManagerNone(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent-dir-xyz")
	defer os.Setenv("PATH", oldPath)

	if _, err := detectPackageManager(); err == nil {
		t.Fatal("expected error when no package manager present")
	}
}

func TestManualInstallCommand(t *testing.T) {
	cases := map[string]string{
		"apt-get":  "sudo apt-get update && sudo apt-get install -y x11vnc",
		"dnf":      "sudo dnf install -y x11vnc",
		"microdnf": "sudo microdnf install -y x11vnc",
		"yum":      "sudo yum install -y x11vnc",
		"pacman":   "sudo pacman -S --noconfirm x11vnc",
		"zypper":   "sudo zypper install -y x11vnc",
		"apk":      "sudo apk add --no-cache x11vnc",
	}
	for pm, want := range cases {
		got := manualInstallCommand(pm, []string{"x11vnc"})
		if got != want {
			t.Errorf("manualInstallCommand(%s) = %q, want %q", pm, got, want)
		}
	}
}

func TestEnsureSandboxDependenciesAllPresent(t *testing.T) {
	withFakeManagedTools(t)
	statuses, err := EnsureSandboxDependencies(func(string, ...string) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, st := range statuses {
		if !st.Present {
			t.Errorf("expected %s present, got missing", st.Binary)
		}
	}
}

func TestEnsureSandboxDependenciesRunnerInvoked(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	var attempted []string
	runner := func(name string, args ...string) error {
		attempted = append(attempted, name)
		return nil
	}
	st := EnsureToolDependency("bpftrace", runner)
	if st.Present {
		t.Skip("bpftrace is already present; cannot exercise package acquisition")
	}
	if len(attempted) == 0 {
		t.Fatal("expected a package-manager install attempt")
	}
}

func TestEnsureSandboxDependenciesRunnerNoOpReflectsReality(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	runner := func(string, ...string) error { return nil }
	withFakeManagedTools(t)
	statuses, err := EnsureSandboxDependencies(runner)
	if err != nil {
		t.Fatalf("unexpected dependency check error: %v", err)
	}
	for _, st := range statuses {
		if !st.Present {
			t.Errorf("expected managed fake %s to be present", st.Binary)
		}
	}
}

func TestEnsureSandboxDependenciesInstallErrorRecorded(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	failRunner := func(string, ...string) error { return os.ErrPermission }
	st := EnsureToolDependency("bpftrace", failRunner)
	if st.Present {
		t.Skip("bpftrace is already present; cannot exercise failure")
	}
	if st.Error == "" {
		t.Error("expected package acquisition error")
	}
}

func withFakeManagedTools(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old := os.Getenv("KNIRVENGINE_SANDBOX_TOOLS_DIR")
	t.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", dir)
	if old != "" {
		t.Cleanup(func() { _ = os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", old) })
	}
	for _, binary := range []string{"bwrap", "Xvfb", "x11vnc", "java", "dotnet", "jadx", "proxychains4", "bpftrace", "tshark", "zeek", "afl-fuzz", "rizin", "frida-server"} {
		writeFakeExecutable(t, filepath.Join(dir, binary))
	}
	for _, binary := range []string{"semgrep", "jwt_tool.py", "frida"} {
		writeFakeExecutable(t, filepath.Join(dir, "pyenv", "bin", binary))
	}
	writeFakeExecutable(t, filepath.Join(dir, "dotnettools", "ilspycmd"))
}

func writeFakeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
