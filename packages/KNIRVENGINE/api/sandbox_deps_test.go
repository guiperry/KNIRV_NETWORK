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
	if !binaryExists("bwrap") || !binaryExists("Xvfb") || !binaryExists("x11vnc") {
		t.Skip("one or more sandbox binaries already missing; skipping no-op test")
	}
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
	statuses, err := EnsureSandboxDependencies(runner)
	if err != nil {
		t.Skipf("no supported package manager in test env: %v", err)
	}
	anyMissing := false
	for _, st := range statuses {
		if !st.Present {
			anyMissing = true
		}
	}
	if anyMissing && len(attempted) == 0 {
		t.Fatal("expected an install attempt when binaries are missing")
	}
}

func TestEnsureSandboxDependenciesRunnerNoOpReflectsReality(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	runner := func(string, ...string) error { return nil }
	statuses, err := EnsureSandboxDependencies(runner)
	if err != nil {
		t.Skipf("no supported package manager in test env: %v", err)
	}
	for _, st := range statuses {
		if binaryExists(st.Binary) != st.Present {
			t.Errorf("present mismatch for %s", st.Binary)
		}
	}
}

func TestEnsureSandboxDependenciesInstallErrorRecorded(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	failRunner := func(string, ...string) error { return os.ErrPermission }
	statuses, err := EnsureSandboxDependencies(failRunner)
	if err != nil {
		t.Skipf("no supported package manager in test env: %v", err)
	}
	anyMissing := false
	for _, st := range statuses {
		if !st.Present {
			anyMissing = true
			if st.Error == "" {
				t.Errorf("expected error recorded for missing %s", st.Binary)
			}
		}
	}
	if !anyMissing {
		t.Skip("all binaries present; cannot exercise error path")
	}
}
