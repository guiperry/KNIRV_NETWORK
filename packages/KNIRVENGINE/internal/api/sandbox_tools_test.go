package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxToolsDirEnvOverride(t *testing.T) {
	dir := t.TempDir()
	old := os.Getenv("KNIRVENGINE_SANDBOX_TOOLS_DIR")
	defer os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", old)
	if err := os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", dir); err != nil {
		t.Fatal(err)
	}
	if got := sandboxToolsDir(); got != dir {
		t.Fatalf("sandboxToolsDir() = %q, want %q", got, dir)
	}
}

func TestResolveSandboxToolBundled(t *testing.T) {
	dir := t.TempDir()
	old := os.Getenv("KNIRVENGINE_SANDBOX_TOOLS_DIR")
	defer os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", old)
	if err := os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", dir); err != nil {
		t.Fatal(err)
	}

	// An executable file in the tools dir resolves to its absolute path.
	binPath := filepath.Join(dir, "bwrap")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := resolveSandboxTool("bwrap"); got != binPath {
		t.Fatalf("resolveSandboxTool(bwrap) = %q, want %q", got, binPath)
	}

	// A missing tool falls back to the bare name (PATH resolution).
	if got := resolveSandboxTool("x11vnc"); got != "x11vnc" {
		t.Fatalf("resolveSandboxTool(x11vnc) = %q, want %q", got, "x11vnc")
	}
}

func TestResolveSandboxToolNonExecutableNotBundled(t *testing.T) {
	dir := t.TempDir()
	old := os.Getenv("KNIRVENGINE_SANDBOX_TOOLS_DIR")
	defer os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", old)
	if err := os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", dir); err != nil {
		t.Fatal(err)
	}

	// A non-executable file must not be treated as a usable bundled binary.
	binPath := filepath.Join(dir, "Xvfb")
	if err := os.WriteFile(binPath, []byte("not a binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveSandboxTool("Xvfb"); got != "Xvfb" {
		t.Fatalf("resolveSandboxTool(Xvfb) = %q, want %q", got, "Xvfb")
	}
}

func TestResolveSandboxToolRejectsIncompleteBundledDotnet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", dir)
	dotnet := filepath.Join(dir, "dotnet")
	if err := os.WriteFile(dotnet, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := resolveSandboxTool("dotnet"); got != "dotnet" {
		t.Fatalf("incomplete bundled dotnet resolved to %q, want host fallback", got)
	}
	if err := os.MkdirAll(filepath.Join(dir, "host", "fxr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := resolveSandboxTool("dotnet"); got != dotnet {
		t.Fatalf("complete bundled dotnet resolved to %q, want %q", got, dotnet)
	}
}

func TestIsBundledTool(t *testing.T) {
	dir := t.TempDir()
	old := os.Getenv("KNIRVENGINE_SANDBOX_TOOLS_DIR")
	defer os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", old)
	if err := os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", dir); err != nil {
		t.Fatal(err)
	}

	if !isBundledTool(filepath.Join(dir, "bwrap")) {
		t.Error("expected absolute path inside tools dir to be bundled")
	}
	if isBundledTool("bwrap") {
		t.Error("expected bare name not to be bundled")
	}
	if isBundledTool("/usr/bin/bwrap") {
		t.Error("expected system path outside tools dir not to be bundled")
	}
}

func TestToolEnvKeepsManagedRuntimeLibrariesIsolated(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", dir)
	if err := os.MkdirAll(filepath.Join(dir, "pyenv", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	semgrep := filepath.Join(dir, "pyenv", "bin", "semgrep")
	if err := os.WriteFile(semgrep, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(dir, "bwrap")
	if err := os.WriteFile(native, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	session := &SandboxSession{}
	if got := environmentValue(session.toolEnv(semgrep), "LD_LIBRARY_PATH"); got != "" {
		t.Fatalf("managed Semgrep LD_LIBRARY_PATH = %q, want empty", got)
	}
	if got, want := environmentValue(session.toolEnv(native), "LD_LIBRARY_PATH"), filepath.Join(dir, "lib"); got != want {
		t.Fatalf("native tool LD_LIBRARY_PATH = %q, want %q", got, want)
	}
}

// TestResolveSandboxToolGeneratedBundle verifies that the engine resolves the
// actually-generated tools/ bundle shipped next to the package root.
func TestResolveSandboxToolGeneratedBundle(t *testing.T) {
	wd, _ := os.Getwd()
	bundle := filepath.Join(wd, "..", "tools")
	if _, err := os.Stat(filepath.Join(bundle, "Xvfb")); err != nil {
		t.Skip("generated tools/ bundle not present; run scripts/bundle-sandbox-tools.sh")
	}
	old := os.Getenv("KNIRVENGINE_SANDBOX_TOOLS_DIR")
	defer os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", old)
	if err := os.Setenv("KNIRVENGINE_SANDBOX_TOOLS_DIR", bundle); err != nil {
		t.Fatal(err)
	}

	if got := resolveSandboxTool("Xvfb"); got != filepath.Join(bundle, "Xvfb") {
		t.Fatalf("resolveSandboxTool(Xvfb) = %q, want %q", got, filepath.Join(bundle, "Xvfb"))
	}
	if !isBundledTool(filepath.Join(bundle, "Xvfb")) {
		t.Error("expected generated Xvfb to be reported as bundled")
	}
}
