package main

import (
	"os"
	"path/filepath"
	"testing"
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
