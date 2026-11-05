package storage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lab/hasher/data-seeder/pkg/rootwitness"
)

func TestJSONSeedWriterUsesTrainingFramesAliasAndExactMatch(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "frames"), 0755); err != nil {
		t.Fatal(err)
	}

	frames := []map[string]interface{}{
		{
			"chunk_id":     1,
			"context_hash": 0,
			"feature_vector": []interface{}{
				286331153.0, 572662306.0, 858993459.0, 1145324612.0,
				1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 4096.0, 0.0,
			},
			"source_file":    "demo.txt",
			"target_token":   917.0,
			"token_sequence": []interface{}{906.0},
			"window_start":   0.0,
		},
	}

	sourceFile := "training_frames.json"
	sourcePath := filepath.Join(root, "frames", sourceFile)
	data, _ := json.MarshalIndent(frames, "", "  ")
	if err := os.WriteFile(sourcePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	sw := NewJSONSeedWriter(root)

	slots := [12]uint32{
		286331153, 572662306, 858993459, 1145324612,
		1, 0, 0, 0, 0, 0, 4096, 0,
	}
	targetTokenID := int32(917)
	bestSeed := []byte("GOLDEN_SEED_12345")

	if err := sw.AddSeedWrite("demo.txt", slots, targetTokenID, bestSeed); err != nil {
		t.Fatalf("AddSeedWrite failed: %v", err)
	}

	if err := sw.WriteBack(); err != nil {
		t.Fatalf("WriteBack failed: %v", err)
	}

	outputPath := filepath.Join(root, "frames", "training_frames_with_seeds.json")
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	var updated []map[string]interface{}
	if err := json.Unmarshal(outputData, &updated); err != nil {
		t.Fatal(err)
	}
	if got := updated[0]["best_seed"]; got != base64.StdEncoding.EncodeToString(bestSeed) {
		t.Fatalf("best_seed = %v, want base64 encoded seed", got)
	}
}

func TestJSONSeedWriterDoesNotFailWhenNoRecordsUpdated(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "frames"), 0755); err != nil {
		t.Fatal(err)
	}

	frames := []map[string]interface{}{
		{
			"feature_vector": []interface{}{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0},
			"target_token":   100.0,
		},
	}
	data, _ := json.MarshalIndent(frames, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "frames", "training_frames.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	sw := NewJSONSeedWriter(root)
	mismatchedSlots := [12]uint32{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9}
	if err := sw.AddSeedWrite("training_frames.json", mismatchedSlots, 100, []byte("seed")); err != nil {
		t.Fatal(err)
	}

	if err := sw.WriteBack(); err != nil {
		t.Fatalf("WriteBack error = %v, want non-fatal materialization warning", err)
	}
}

func TestDualSeedWriterAppendsLedgerBeforeBestEffortMaterialization(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "frames"), 0755); err != nil {
		t.Fatal(err)
	}

	frames := []map[string]interface{}{
		{
			"feature_vector": []interface{}{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0},
			"target_token":   100.0,
		},
	}
	data, _ := json.MarshalIndent(frames, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "frames", "training_frames.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	sw := NewDualSeedWriter(root)
	seed := []byte("ledgered-seed")
	mismatchedSlots := [12]uint32{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9}
	if err := sw.AddSeedWrite("training_frames.json", mismatchedSlots, 100, seed); err != nil {
		t.Fatal(err)
	}
	if err := sw.WriteBack(); err != nil {
		t.Fatalf("WriteBack error = %v, want best-effort success", err)
	}

	ledgerData, err := os.ReadFile(filepath.Join(root, "frames", "seed_writes.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var entry SeedWriteLedgerEntry
	if err := json.Unmarshal(ledgerData[:len(ledgerData)-1], &entry); err != nil {
		t.Fatal(err)
	}
	if entry.TargetTokenID != 100 {
		t.Fatalf("ledger target = %d, want 100", entry.TargetTokenID)
	}
	if entry.BestSeed != base64.StdEncoding.EncodeToString(seed) {
		t.Fatalf("ledger seed = %q, want base64 seed", entry.BestSeed)
	}
}

func TestDualSeedWriterPersistsVersionedAssertionIdentity(t *testing.T) {
	root := t.TempDir()
	sw := NewDualSeedWriter(root)
	slots := [12]uint32{1, 2, 3}
	context := []int32{10, 20, 30}
	span := []int32{40, 50}
	if err := sw.AddAssertionWrite("frames.json", slots, 40, context, span, 123, 456, []byte("seed")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "frames", "seed_writes.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var entry SeedWriteLedgerEntry
	if err := json.Unmarshal(data[:len(data)-1], &entry); err != nil {
		t.Fatal(err)
	}
	if entry.SchemaVersion != 2 || entry.ContextHash != 123 || entry.CommitmentTarget != 456 {
		t.Fatalf("unexpected assertion metadata: %+v", entry)
	}
	if entry.AssertionKey == "" {
		t.Fatal("versioned assertion ledger entry is missing its canonical key")
	}
	if len(entry.ContextTokens) != 3 || len(entry.AssertionSpan) != 2 {
		t.Fatalf("assertion identity was not persisted: %+v", entry)
	}
}

func TestDualSeedWriterAcceptsRootAsAssertion(t *testing.T) {
	root := t.TempDir()
	sw := NewDualSeedWriter(root)
	rootHash := make([]byte, 32)
	for i := range rootHash {
		rootHash[i] = byte(i)
	}

	id, err := sw.AddRootAssertion(rootHash, "evidence-batch-1", []byte("witness-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty root assertion id")
	}
	if got, ok := sw.RootAssertion(id); !ok {
		t.Fatal("root assertion not recorded")
	} else if got.Kind != AssertionKindRoot || !bytes.Equal(got.Root, rootHash) {
		t.Fatalf("unexpected root assertion: %+v", got)
	}

	// The same root maps to the same id (idempotent key space).
	id2, _ := sw.AddRootAssertion(rootHash, "other-ref", []byte("seed2"))
	if id2 != id {
		t.Fatalf("root id not stable: %s != %s", id, id2)
	}

	// Span (non-root) entries must not populate the backfill map under a zero key.
	seeds := ledgerSeedsByKey(root)
	if _, ok := seeds["0:0:0:0:0:0:0:0:0:0:0:0:0"]; ok {
		t.Fatal("root assertion leaked a zero asic/token key into the backfill map")
	}
}

func TestRootWitnessQueueSubmitAndCommit(t *testing.T) {
	root := t.TempDir()
	sw := NewDualSeedWriter(root)
	rootHash := make([]byte, 32)
	rootHash[0] = 7

	id := sw.SubmitRoot(rootHash, "evidence-batch-2")
	if got := sw.Pending(); len(got) != 1 || got[0].ID != id {
		t.Fatalf("pending queue mismatch: %+v", got)
	}

	// The seed must be a real 32-byte root-PoW commitment (audit finding 2): the
	// committed seed can no longer be an arbitrary string.
	seed, err := rootwitness.MineRootSeed(rootHash)
	if err != nil {
		t.Fatalf("mine failed: %v", err)
	}
	committed, err := sw.CommitRootAssertion(id, seed)
	if err != nil {
		t.Fatal(err)
	}
	if committed != id {
		t.Fatalf("committed id %s != %s", committed, id)
	}
	if _, ok := sw.RootAssertion(id); !ok {
		t.Fatal("committed root assertion not found")
	}
	if got := sw.Pending(); len(got) != 0 {
		t.Fatalf("queue not drained: %+v", got)
	}

	if _, err := sw.CommitRootAssertion(id, []byte("x")); err == nil {
		t.Fatal("commit of a non-32-byte seed should fail")
	}
	if _, err := sw.CommitRootAssertion(id, make([]byte, 32)); err == nil {
		t.Fatal("commit of a non-PoW 32-byte seed should fail")
	}
}
