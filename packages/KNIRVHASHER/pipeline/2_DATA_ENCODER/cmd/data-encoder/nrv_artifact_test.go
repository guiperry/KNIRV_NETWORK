package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"data-encoder/pkg/schema"
)

func TestWriteNRVArtifactRetainsPriorArtifactsAndPublishesLatest(t *testing.T) {
	dir := t.TempDir()
	latestPath := filepath.Join(dir, "training_frames.nrv")
	frames := []schema.TrainingFrame{{TargetTokenID: 7}}

	first, err := writeNRVArtifact(latestPath, frames)
	if err != nil {
		t.Fatalf("write first NRV artifact: %v", err)
	}
	time.Sleep(time.Nanosecond)
	second, err := writeNRVArtifact(latestPath, frames)
	if err != nil {
		t.Fatalf("write second NRV artifact: %v", err)
	}
	if first == second {
		t.Fatalf("artifact paths must be unique: %s", first)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("first immutable artifact missing: %v", err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatalf("second immutable artifact missing: %v", err)
	}

	data, err := os.ReadFile(latestPath + ".latest.json")
	if err != nil {
		t.Fatalf("read latest manifest: %v", err)
	}
	var manifest nrvLatestManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode latest manifest: %v", err)
	}
	if manifest.Artifact != filepath.Base(second) || manifest.BracketCount != len(frames) {
		t.Fatalf("manifest = %#v, want latest %q with %d brackets", manifest, filepath.Base(second), len(frames))
	}
}

func TestWriteBatchArtifactsKeepsEveryFormatForEachRun(t *testing.T) {
	framesDir := t.TempDir()
	output := filepath.Join(framesDir, "training_frames.json")
	frames := []schema.TrainingFrame{{SourceFile: "first", TokenSequence: []int32{1, 2}, TargetTokenID: 3}}
	first, err := writeBatchArtifacts(output, frames)
	if err != nil {
		t.Fatalf("write first batch: %v", err)
	}
	frames[0].SourceFile = "second"
	second, err := writeBatchArtifacts(output, frames)
	if err != nil {
		t.Fatalf("write second batch: %v", err)
	}
	if first.BatchID == second.BatchID {
		t.Fatal("batch IDs must be unique")
	}
	for _, batch := range []batchManifest{first, second} {
		for kind, name := range batch.Artifacts {
			if _, err := os.Stat(filepath.Join(framesDir, "batches", batch.BatchID, name)); err != nil {
				t.Fatalf("%s artifact for %s missing: %v", kind, batch.BatchID, err)
			}
		}
	}
	latest, err := os.ReadFile(filepath.Join(framesDir, "latest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var published batchManifest
	if err := json.Unmarshal(latest, &published); err != nil {
		t.Fatal(err)
	}
	if published.BatchID != second.BatchID {
		t.Fatalf("latest batch = %s, want %s", published.BatchID, second.BatchID)
	}
}
