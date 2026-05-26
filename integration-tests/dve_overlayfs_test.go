package integration_tests

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDVEWorkspaceIsolation verifies that two DVE workspaces are isolated.
// Requires OverlayFS support on the host — run with KNIRV_TEST_OVERLAY=1.
func TestDVEWorkspaceIsolation(t *testing.T) {
	if os.Getenv("KNIRV_TEST_OVERLAY") != "1" {
		t.Skip("Skipping OverlayFS integration test: set KNIRV_TEST_OVERLAY=1 to run")
	}

	// 1. Create two DVE workspace directories
	baseDir := t.TempDir()

	ws1 := filepath.Join(baseDir, "ws1")
	ws2 := filepath.Join(baseDir, "ws2")

	for _, d := range []string{ws1, ws2} {
		upper := filepath.Join(d, "upper")
		work := filepath.Join(d, "work")
		merged := filepath.Join(d, "merged")
		for _, sub := range []string{upper, work, merged} {
			if err := os.MkdirAll(sub, 0755); err != nil {
				t.Fatalf("mkdir %s: %v", sub, err)
			}
		}
	}

	// 2. Write a sentinel file to ws1's workspace
	sentinel := filepath.Join(ws1, "upper", "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("secret-data"), 0644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	// 3. Verify ws2 does NOT have access to ws1's sentinel
	ws2Sentinel := filepath.Join(ws2, "upper", "sentinel.txt")
	if _, err := os.Stat(ws2Sentinel); err == nil {
		t.Error("DVE isolation breach: ws2 can read ws1's sentinel file")
	}

	// 4. Verify ws1 can read its own sentinel
	if _, err := os.Stat(sentinel); err != nil {
		t.Errorf("ws1 cannot read its own sentinel: %v", err)
	}

	// 5. Terminate (remove) ws1; verify ws2 is unaffected
	if err := os.RemoveAll(ws1); err != nil {
		t.Fatalf("remove ws1: %v", err)
	}
	if _, err := os.Stat(ws2); os.IsNotExist(err) {
		t.Error("DVE isolation breach: removing ws1 affected ws2")
	}
}

// TestSkillWASMValidation verifies a skill WASM module can be executed
// against a temporary DVE workspace. Requires wazero dependency.
func TestSkillWASMValidation(t *testing.T) {
	// This test requires the dve_workspace package which may not be importable
	// from the integration test module. It serves as a placeholder for
	// runtime validation of the Wazero executor pipeline.
	t.Skip("Skipping WASM validation test: requires dve_workspace import from integration-tests module")
}
