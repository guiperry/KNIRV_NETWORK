package staging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "input.txt")
	if err := os.WriteFile(sourcePath, []byte("knirv-shell-stage"), 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	result, err := StageFile(sourcePath, filepath.Join(tempDir, "staging"))
	if err != nil {
		t.Fatalf("stage file: %v", err)
	}

	if result.Size == 0 {
		t.Fatalf("expected non-zero size")
	}
	if result.Digest == "" {
		t.Fatalf("expected digest")
	}
	if _, err := os.Stat(result.StagedPath); err != nil {
		t.Fatalf("expected staged file to exist: %v", err)
	}
}
