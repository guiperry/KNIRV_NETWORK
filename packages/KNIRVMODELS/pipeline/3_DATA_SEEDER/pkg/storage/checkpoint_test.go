package storage

import (
	"path/filepath"
	"testing"
)

func TestProcessedAssertionSurvivesRestartWithoutWinningCheckpoint(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "checkpoints.db")
	const assertionKey = "assertion-v2:test-record"

	manager := NewCheckpointManager(dbPath)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("initialize checkpoint manager: %v", err)
	}
	if err := manager.MarkAssertionProcessed(assertionKey, 42); err != nil {
		t.Fatalf("mark assertion processed: %v", err)
	}

	// Reopen from disk to exercise the next pipeline invocation. No winning
	// checkpoint is saved in this test: the completed-attempt ledger alone must
	// prevent the record from being mined again.
	restarted := NewCheckpointManager(dbPath)
	if err := restarted.Initialize(); err != nil {
		t.Fatalf("reinitialize checkpoint manager: %v", err)
	}
	processed, err := restarted.HasAssertionProcessed(assertionKey, 42)
	if err != nil {
		t.Fatalf("check processed assertion: %v", err)
	}
	if !processed {
		t.Fatal("processed assertion was not retained across restart")
	}

	hasCheckpoint, err := restarted.HasAssertionCheckpoint(assertionKey, 42)
	if err != nil {
		t.Fatalf("check winning checkpoint: %v", err)
	}
	if hasCheckpoint {
		t.Fatal("completed attempt unexpectedly created a winning checkpoint")
	}
}
