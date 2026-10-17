package transactionchain

import (
	"os"
	"path/filepath"
	"testing"
)

func writeNodeVersionScript(t *testing.T, version string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "node")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho '"+version+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveNodeBinaryRejectsOldOverride(t *testing.T) {
	t.Setenv("KNIRV_NODE_BINARY_PATH", writeNodeVersionScript(t, "v12.22.12"))
	if _, err := resolveNodeBinary(); err == nil {
		t.Fatal("resolveNodeBinary() accepted Node 12")
	}
}

func TestResolveNodeBinaryAcceptsSupportedOverride(t *testing.T) {
	want := writeNodeVersionScript(t, "v22.22.0")
	t.Setenv("KNIRV_NODE_BINARY_PATH", want)
	got, err := resolveNodeBinary()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("resolveNodeBinary() = %q, want %q", got, want)
	}
}

func TestNpmInstallIsSkippedOnlyWhenDependenciesAreUnchanged(t *testing.T) {
	dir := t.TempDir()
	if npmInstallCurrent(dir) {
		t.Fatal("an empty directory is not installed")
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"a":"1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeNpmInstallStamp(dir); err != nil {
		t.Fatal(err)
	}
	if npmInstallCurrent(dir) {
		t.Fatal("no node_modules yet")
	}
	if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !npmInstallCurrent(dir) {
		t.Fatal("installed from the current manifest must be skipped")
	}
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"a":"2"}}`), 0o644)
	if npmInstallCurrent(dir) {
		t.Fatal("a changed dependency must reinstall")
	}
}
