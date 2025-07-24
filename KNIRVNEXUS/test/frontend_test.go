package test // Or your appropriate package name

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunFrontendTests(t *testing.T) {
	// Determine the project root dynamically
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Could not determine current file path")
	}
	projectRoot := filepath.Dir(currentFile) // Assumes this test file is in the project root

	guiDir := filepath.Join(projectRoot, "gui")
	t.Logf("Running npm tests in: %s", guiDir)

	cmd := exec.Command("npm", "test")
	cmd.Dir = guiDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		// npm test likely failed
		t.Errorf("npm test failed: %v\nOutput:\n%s", err, string(output))
	} else {
		// npm test succeeded
		t.Logf("npm test succeeded.\nOutput:\n%s", string(output))
		// You might want to check for specific success messages in the output if needed
	}
}
