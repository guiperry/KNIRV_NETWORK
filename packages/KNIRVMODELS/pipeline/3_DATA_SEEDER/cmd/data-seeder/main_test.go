package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lab/hasher/data-seeder/internal/logging"
)

func TestSelectTrainingDataPathPrefersBaseFramesOverSeededOutput(t *testing.T) {
	root := t.TempDir()
	framesDir := filepath.Join(root, "frames")
	if err := os.MkdirAll(framesDir, 0755); err != nil {
		t.Fatal(err)
	}

	basePath := filepath.Join(framesDir, "training_frames.json")
	seededPath := filepath.Join(framesDir, "training_frames_with_seeds.json")
	if err := os.WriteFile(seededPath, []byte(`[{"best_seed":"already-trained"}]`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(basePath, []byte(`[{"target_token_id":7}]`), 0644); err != nil {
		t.Fatal(err)
	}

	got, seededOnly, err := selectTrainingDataPath(root)
	if err != nil {
		t.Fatalf("selectTrainingDataPath returned error: %v", err)
	}
	if got != basePath {
		t.Fatalf("selected %q, want %q", got, basePath)
	}
	if seededOnly {
		t.Fatal("seededOnly = true, want false when base frames exist")
	}
}

func TestSelectTrainingDataPathAllowsSeededOutputOnlyAsCompleteInput(t *testing.T) {
	root := t.TempDir()
	framesDir := filepath.Join(root, "frames")
	if err := os.MkdirAll(framesDir, 0755); err != nil {
		t.Fatal(err)
	}

	seededPath := filepath.Join(framesDir, "training_frames_with_seeds.json")
	if err := os.WriteFile(seededPath, []byte(`[{"best_seed":"already-trained"}]`), 0644); err != nil {
		t.Fatal(err)
	}

	got, seededOnly, err := selectTrainingDataPath(root)
	if err != nil {
		t.Fatalf("selectTrainingDataPath returned error: %v", err)
	}
	if got != seededPath {
		t.Fatalf("selected %q, want %q", got, seededPath)
	}
	if !seededOnly {
		t.Fatal("seededOnly = false, want true when only seeded output exists")
	}
}

func TestFinalizeTrainingHandlesCompletedRecordShortCircuit(t *testing.T) {
	logger, err := logging.NewLogger(&logging.LoggingConfig{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	defer logger.Close()

	statusPath := filepath.Join(t.TempDir(), "runs", "latest_training_run.json")
	orchestrator := &TrainingOrchestrator{
		logger:           logger,
		allRecordsSeeded: true,
		runID:            "completed-records",
		runStatusPath:    statusPath,
	}

	if err := orchestrator.Run(context.Background(), 1, 1); err != nil {
		t.Fatalf("run completed-record short-circuit: %v", err)
	}
	if _, err := os.Stat(statusPath); err != nil {
		t.Fatalf("completed run status was not written: %v", err)
	}
}
