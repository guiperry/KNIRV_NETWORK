package transformer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestLedger(t *testing.T, dir string, lines []string) {
	t.Helper()
	path := filepath.Join(dir, seedWritesFile)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test ledger: %v", err)
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(l + "\n"); err != nil {
			t.Fatalf("write test ledger line: %v", err)
		}
	}
}

func TestLoadSeedStoreFromFrames_MissingLedger(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultUnifiedConfig()
	cfg.VocabSize = 10
	cfg.EmbedDim = 4
	_, _, err := LoadSeedStoreFromFrames(dir, cfg)
	if err == nil {
		t.Fatal("expected error for missing ledger")
	}
}

func TestLoadSeedStoreFromFrames_UsesRealMinedSeeds(t *testing.T) {
	dir := t.TempDir()
	writeTestLedger(t, dir, []string{
		`{"timestamp":"2026-01-01T00:00:00Z","source_file":"x","target_token_id":3,"asic_slots":[1,2,3,4,5,6,7,8,9,10,11,12],"best_seed":"KSX5x2mzKf23YCPjhEVNacOzrEx0aC10lqwavlVn8FU=","seed_bytes":32}`,
		`{"timestamp":"2026-01-01T00:00:01Z","source_file":"x","target_token_id":3,"asic_slots":[1,2,3,4,5,6,7,8,9,10,11,13],"best_seed":"TSlfKOB4dyLw/cGW5CNjfu//wWMgHuN1CLRiKuZ1u/A=","seed_bytes":32}`,
		`{"timestamp":"2026-01-01T00:00:02Z","source_file":"x","target_token_id":7,"asic_slots":[1,2,3,4,5,6,7,8,9,10,11,14],"best_seed":"5u00znIhG5uEAkJ+gqNOWZ6vFjYzBKxdm9hyhqVnyw4=","seed_bytes":32}`,
		`not json, should be skipped`,
		`{"timestamp":"2026-01-01T00:00:03Z","source_file":"x","target_token_id":999,"asic_slots":[1,2,3,4,5,6,7,8,9,10,11,15],"best_seed":"","seed_bytes":0}`,
	})

	cfg := DefaultUnifiedConfig()
	cfg.VocabSize = 10
	cfg.EmbedDim = 4

	store, stats, err := LoadSeedStoreFromFrames(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.LedgerRecords != 3 {
		t.Fatalf("expected 3 valid ledger records, got %d", stats.LedgerRecords)
	}
	if stats.TokensCovered != 2 {
		t.Fatalf("expected 2 tokens covered (token 999 is out of vocab range), got %d", stats.TokensCovered)
	}

	// Rebuilding from the same ledger must be fully deterministic.
	store2, _, err := LoadSeedStoreFromFrames(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error on reload: %v", err)
	}
	for j := range store.Embeddings[3] {
		if store.Embeddings[3][j] != store2.Embeddings[3][j] {
			t.Fatal("expected deterministic embedding derivation across reloads")
		}
	}

	// Token 3 was mined (appears twice in the ledger); token 5 was never
	// mined and should retain its random BuildDefaultSeedStore embedding
	// rather than an all-zero or otherwise degenerate value.
	allZero := true
	for _, s := range store.Embeddings[3] {
		if s != ([32]byte{}) {
			allZero = false
		}
	}
	if allZero {
		t.Fatal("expected non-trivial derived embedding for mined token")
	}
}

// TestLoadSeedStoreFromFrames_ProductionLedger is a smoke test against the
// real host directory, if present. It is skipped (not failed) when absent,
// since most dev/CI environments won't have /var/lib/knirvserver mounted.
func TestLoadSeedStoreFromFrames_ProductionLedger(t *testing.T) {
	if _, err := os.Stat(filepath.Join(DefaultFramesDir, seedWritesFile)); err != nil {
		t.Skipf("production frames dir not available: %v", err)
	}
	cfg := DefaultUnifiedConfig()
	store, stats, err := LoadSeedStoreFromFrames(DefaultFramesDir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	t.Logf("loaded %d ledger records, %d/%d tokens covered from real mining data",
		stats.LedgerRecords, stats.TokensCovered, stats.TokensTotal)
	if stats.TokensCovered == 0 {
		t.Fatal("expected at least one token covered from the real production ledger")
	}
}
