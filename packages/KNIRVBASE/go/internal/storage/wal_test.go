package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestWAL(t *testing.T) (*WAL, string) {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv.wal")
	wal := NewWAL(path)
	return wal, path
}

func TestWALBeginCommit(t *testing.T) {
	wal, path := setupTestWAL(t)

	entry := WALEntry{
		FrameID:        "frame-1",
		LastGoodLength: 1024,
		Committed:      false,
	}

	if err := wal.Begin(entry); err != nil {
		t.Fatalf("WAL.Begin failed: %v", err)
	}

	if err := wal.Commit("frame-1"); err != nil {
		t.Fatalf("WAL.Commit failed: %v", err)
	}

	entries, err := wal.readEntries()
	if err != nil {
		t.Fatalf("WAL.readEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if !entries[0].Committed {
		t.Error("expected entry to be committed")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected WAL file to exist after commit")
	}
}

func TestWALRecoverNoUncommitted(t *testing.T) {
	wal, _ := setupTestWAL(t)

	entry := WALEntry{
		FrameID:        "frame-1",
		LastGoodLength: 1024,
		Committed:      false,
	}

	if err := wal.Begin(entry); err != nil {
		t.Fatalf("WAL.Begin failed: %v", err)
	}

	if err := wal.Commit("frame-1"); err != nil {
		t.Fatalf("WAL.Commit failed: %v", err)
	}

	recoverLen, err := wal.Recover()
	if err != nil {
		t.Fatalf("WAL.Recover failed: %v", err)
	}

	if recoverLen != -1 {
		t.Errorf("expected -1 for clean WAL, got %d", recoverLen)
	}
}

func TestWALRecoverWithUncommitted(t *testing.T) {
	wal, _ := setupTestWAL(t)

	entry1 := WALEntry{
		FrameID:        "frame-1",
		LastGoodLength: 1024,
		Committed:      true,
	}

	entry2 := WALEntry{
		FrameID:        "frame-2",
		LastGoodLength: 2048,
		Committed:      false,
	}

	if err := wal.Begin(entry1); err != nil {
		t.Fatalf("WAL.Begin failed: %v", err)
	}
	if err := wal.Commit("frame-1"); err != nil {
		t.Fatalf("WAL.Commit failed: %v", err)
	}

	if err := wal.Begin(entry2); err != nil {
		t.Fatalf("WAL.Begin failed: %v", err)
	}

	recoverLen, err := wal.Recover()
	if err != nil {
		t.Fatalf("WAL.Recover failed: %v", err)
	}

	if recoverLen != 2048 {
		t.Errorf("expected recovery length 2048, got %d", recoverLen)
	}
}

func TestWALRecoverEmptyWAL(t *testing.T) {
	wal, _ := setupTestWAL(t)

	recoverLen, err := wal.Recover()
	if err != nil {
		t.Fatalf("WAL.Recover failed: %v", err)
	}

	if recoverLen != -1 {
		t.Errorf("expected -1 for empty WAL, got %d", recoverLen)
	}
}

func TestWALTruncate(t *testing.T) {
	wal, path := setupTestWAL(t)

	entry := WALEntry{
		FrameID:        "frame-1",
		LastGoodLength: 1024,
		Committed:      false,
	}

	if err := wal.Begin(entry); err != nil {
		t.Fatalf("WAL.Begin failed: %v", err)
	}

	if err := wal.Truncate(); err != nil {
		t.Fatalf("WAL.Truncate failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected WAL file to be deleted after truncate")
	}
}

func TestWALMultipleEntries(t *testing.T) {
	wal, _ := setupTestWAL(t)

	entries := []WALEntry{
		{FrameID: "frame-1", LastGoodLength: 1024, Committed: false},
		{FrameID: "frame-2", LastGoodLength: 2048, Committed: false},
		{FrameID: "frame-3", LastGoodLength: 3072, Committed: false},
	}

	for _, entry := range entries {
		if err := wal.Begin(entry); err != nil {
			t.Fatalf("WAL.Begin failed for %s: %v", entry.FrameID, err)
		}
	}

	if err := wal.Commit("frame-2"); err != nil {
		t.Fatalf("WAL.Commit failed: %v", err)
	}

	recoverLen, err := wal.Recover()
	if err != nil {
		t.Fatalf("WAL.Recover failed: %v", err)
	}

	if recoverLen != 1024 {
		t.Errorf("expected minimum uncommitted length 1024, got %d", recoverLen)
	}
}
