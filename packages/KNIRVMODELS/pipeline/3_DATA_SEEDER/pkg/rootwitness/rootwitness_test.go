package rootwitness

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func tempQueue(t *testing.T) (*RootWitnessQueue, func()) {
	t.Helper()
	dir := t.TempDir()
	q := OpenRootWitnessQueue(filepath.Join(dir, "root_queue.jsonl"))
	return q, func() { _ = os.Remove(filepath.Join(dir, "root_queue.jsonl")) }
}

func TestRootWitnessQueueDurable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "root_queue.jsonl")
	q := OpenRootWitnessQueue(path)
	id1 := q.Submit([]byte("batch-root-A"), "seed-1")
	id2 := q.Submit([]byte("batch-root-B"), "seed-2")
	if id1 == id2 {
		t.Fatalf("distinct roots yielded identical ids: %s", id1)
	}
	if got := q.Submit([]byte("batch-root-A"), "seed-1"); got != id1 {
		t.Fatalf("re-submit changed id: %s != %s", got, id1)
	}
	pend := q.Pending()
	if len(pend) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pend))
	}
	q.Remove(id1)
	if len(q.Pending()) != 1 {
		t.Fatalf("expected 1 pending after remove, got %d", len(q.Pending()))
	}

	// Reload from disk: pending state (minus removed) and tombstones survive.
	re := OpenRootWitnessQueue(path)
	repend := re.Pending()
	if len(repend) != 1 {
		t.Fatalf("reload lost pending: expected 1, got %d", len(repend))
	}
	if re.Submit([]byte("batch-root-A"), "seed-1") != id1 {
		t.Fatalf("reload resurrected a removed root")
	}
}

func TestMineRootSeed(t *testing.T) {
	root := []byte("some-batch-root")
	seed, err := MineRootSeed(root)
	if err != nil {
		t.Fatalf("MineRootSeed failed: %v", err)
	}
	if len(seed) != 32 {
		t.Fatalf("seed length = %d, want 32", len(seed))
	}
	if !VerifyRootPoW(root, seed) {
		t.Fatalf("mined seed does not verify as a valid root PoW")
	}
	seed2, err := MineRootSeed(root)
	if err != nil {
		t.Fatalf("second MineRootSeed failed: %v", err)
	}
	if !bytes.Equal(seed, seed2) {
		t.Fatalf("MineRootSeed is not deterministic for the same root")
	}
	if _, err := MineRootSeed([]byte("some-batch-root")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMineRootSeedExhaustion(t *testing.T) {
	// Impossible difficulty: no 32-byte seed can produce 256 leading zero bits.
	if _, err := mineRootSeed([]byte("hostile-root"), 1<<20, 256); err == nil {
		t.Fatalf("expected exhaustion error for impossible difficulty, got nil")
	}
	// Tiny budget before any valid seed is found at the configured difficulty.
	if _, err := mineRootSeed([]byte("hostile-root"), 1<<10, 200); err == nil {
		t.Fatalf("expected exhaustion error for impossible budget, got nil")
	}
	// Sanity: a budget that should succeed still succeeds.
	if _, err := mineRootSeed([]byte("ok-root"), rootPoWMaxIter, rootPoWBits); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestVerifyRootPoW(t *testing.T) {
	root := []byte("verify-root")
	seed, err := MineRootSeed(root)
	if err != nil {
		t.Fatalf("mine failed: %v", err)
	}
	if !VerifyRootPoW(root, seed) {
		t.Fatalf("valid seed rejected")
	}
	// Wrong length is rejected.
	if VerifyRootPoW(root, seed[:31]) {
		t.Fatalf("31-byte seed accepted")
	}
	// Tampered seed is rejected.
	bad := append([]byte(nil), seed...)
	bad[0] ^= 0xff
	if VerifyRootPoW(root, bad) {
		t.Fatalf("tampered seed accepted")
	}
	// Seed for a different root is rejected.
	other, _ := MineRootSeed([]byte("different-root"))
	if VerifyRootPoW(root, other) {
		t.Fatalf("seed committed to a different root accepted")
	}
}

func TestRootAssertionKeyStable(t *testing.T) {
	a := RootAssertionKey([]byte("k"))
	b := RootAssertionKey([]byte("k"))
	c := RootAssertionKey([]byte("k2"))
	if a != b {
		t.Fatalf("RootAssertionKey not stable for same root")
	}
	if a == c {
		t.Fatalf("distinct roots collided")
	}
}
