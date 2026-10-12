package loader

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFramesInputUsesVerifiedLatestBatch(t *testing.T) {
	dir := t.TempDir()
	batchID := "batch-test"
	batchDir := filepath.Join(dir, "batches", batchID)
	if err := os.MkdirAll(batchDir, 0755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`[{"source_file":"a","token_sequence":[1],"target_token_id":2}]`)
	if err := os.WriteFile(filepath.Join(batchDir, "training_frames.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	manifest := `{"version":1,"batch_id":"batch-test","artifacts":{"json":"training_frames.json"},"sha256":{"json":"` + fmt.Sprintf("%x", sum) + `"}}`
	if err := os.WriteFile(filepath.Join(dir, "latest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveFramesInput(filepath.Join(dir, "training_frames.json"))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.BatchID != batchID || resolved.Path != filepath.Join(batchDir, "training_frames.json") {
		t.Fatalf("unexpected resolution: %#v", resolved)
	}
}

func writeTestFrames(t *testing.T, dir string, frames []map[string]interface{}) string {
	t.Helper()
	path := filepath.Join(dir, "training_frames.json")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test frames: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(frames); err != nil {
		t.Fatalf("encode test frames: %v", err)
	}
	return path
}

func TestLoadFrames_EmptySequenceFiltered(t *testing.T) {
	dir := t.TempDir()
	writeTestFrames(t, dir, []map[string]interface{}{
		{"source_file": "a.txt", "token_sequence": []int32{1, 2, 3}, "target_token_id": int32(4)},
		{"source_file": "b.txt", "token_sequence": []int32{}, "target_token_id": int32(5)},
		{"source_file": "c.txt", "token_sequence": []int32{6, 7}, "target_token_id": int32(8)},
	})

	frames, err := LoadFrames(filepath.Join(dir, "training_frames.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	if frames[0].SourceFile != "a.txt" {
		t.Fatalf("expected first frame source_file=a.txt, got %s", frames[0].SourceFile)
	}
	if frames[1].SourceFile != "c.txt" {
		t.Fatalf("expected second frame source_file=c.txt, got %s", frames[1].SourceFile)
	}
}

func TestLoadFrames_MissingFile(t *testing.T) {
	_, err := LoadFrames(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFrames_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	_, err := LoadFrames(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadFrames_AllEmptyFiltered(t *testing.T) {
	dir := t.TempDir()
	writeTestFrames(t, dir, []map[string]interface{}{
		{"source_file": "x.txt", "token_sequence": []int32{}, "target_token_id": int32(1)},
	})

	frames, err := LoadFrames(filepath.Join(dir, "training_frames.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 0 {
		t.Fatalf("expected 0 frames after filtering, got %d", len(frames))
	}
}
