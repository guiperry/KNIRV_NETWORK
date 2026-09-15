package compiler

import (
	"strings"
	"testing"
)

// A bundle must be byte-identical for identical input, because content_hash is
// what the mint route verifies and what a validation verdict is attributed to.
// path_b_sft.py encodes text with Python's builtin hash(), which CPython salts
// per process, so the seed must be pinned on every invocation.
func TestPythonEnvPinsHashSeed(t *testing.T) {
	env := pythonEnv()

	seedEntries := 0
	value := ""
	for _, entry := range env {
		if strings.HasPrefix(entry, "PYTHONHASHSEED=") {
			seedEntries++
			value = entry
		}
	}
	if seedEntries != 1 {
		t.Fatalf("expected exactly one PYTHONHASHSEED entry, got %d", seedEntries)
	}
	if value != "PYTHONHASHSEED=0" {
		t.Fatalf("PYTHONHASHSEED = %q, want 0 (0 disables hash randomisation)", value)
	}
}

// An inherited value must be replaced, not duplicated: with duplicate entries the
// effective value depends on how the child resolves them, which is exactly the
// kind of ambiguity that would make determinism unreliable.
func TestPythonEnvReplacesInheritedSeed(t *testing.T) {
	t.Setenv("PYTHONHASHSEED", "12345")

	env := pythonEnv()

	seedEntries := 0
	for _, entry := range env {
		if strings.HasPrefix(entry, "PYTHONHASHSEED=") {
			seedEntries++
			if entry != "PYTHONHASHSEED=0" {
				t.Fatalf("inherited seed survived: %q", entry)
			}
		}
	}
	if seedEntries != 1 {
		t.Fatalf("expected exactly one PYTHONHASHSEED entry after replacing, got %d", seedEntries)
	}
}

// The rest of the environment must pass through untouched — the engines may rely
// on PATH or the venv's own variables.
func TestPythonEnvPreservesOtherVariables(t *testing.T) {
	t.Setenv("KNIRV_ULORA_TEST_MARKER", "kept")

	found := false
	for _, entry := range pythonEnv() {
		if entry == "KNIRV_ULORA_TEST_MARKER=kept" {
			found = true
		}
	}
	if !found {
		t.Fatal("unrelated environment variables must be preserved")
	}
}
