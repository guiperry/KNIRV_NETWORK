package checkpoint

import (
	"path/filepath"
	"testing"
)

// newTestManager builds a Manager whose checkpoint file lives in a temp dir.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewManager(filepath.Join(t.TempDir(), "out.jsonl"))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return mgr
}

// TestGetCheckpointReturnsAllFields verifies that the snapshot carries every
// field of the live checkpoint, so a field dropped from the copy is caught.
func TestGetCheckpointReturnsAllFields(t *testing.T) {
	mgr := newTestManager(t)
	mgr.SetQuotaLimit(42)
	mgr.UseQuota(7)
	mgr.UseQuota(100) // exceeds limit 42: sets QuotaExceeded, leaves QuotaUsed at 7
	mgr.UpdateProgress("file-a.txt", 3, 9)
	mgr.IncrementStats(5, 11)
	mgr.MarkRecordProcessed("file-a.txt", 3, 9)

	cp := mgr.GetCheckpoint()

	if cp.QuotaLimit != 42 {
		t.Errorf("QuotaLimit = %d, want 42", cp.QuotaLimit)
	}
	if cp.QuotaUsed != 7 {
		t.Errorf("QuotaUsed = %d, want 7", cp.QuotaUsed)
	}
	if !cp.QuotaExceeded {
		t.Errorf("QuotaExceeded = false, want true")
	}
	if cp.LastProcessedFile != "file-a.txt" {
		t.Errorf("LastProcessedFile = %q, want file-a.txt", cp.LastProcessedFile)
	}
	if cp.LastProcessedChunk != 3 {
		t.Errorf("LastProcessedChunk = %d, want 3", cp.LastProcessedChunk)
	}
	if cp.LastProcessedWindow != 9 {
		t.Errorf("LastProcessedWindow = %d, want 9", cp.LastProcessedWindow)
	}
	if cp.RecordsProcessed != 5 {
		t.Errorf("RecordsProcessed = %d, want 5", cp.RecordsProcessed)
	}
	if cp.FramesGenerated != 11 {
		t.Errorf("FramesGenerated = %d, want 11", cp.FramesGenerated)
	}
	if cp.LastUpdated == 0 {
		t.Errorf("LastUpdated = 0, want a non-zero timestamp")
	}
	if !cp.ProcessedKeys["file-a.txt#3#9"] {
		t.Errorf("ProcessedKeys missing file-a.txt#3#9: %v", cp.ProcessedKeys)
	}
}

// TestGetCheckpointScalarsAreSnapshot verifies that changing the live
// checkpoint after taking a snapshot does not change the snapshot's scalars.
func TestGetCheckpointScalarsAreSnapshot(t *testing.T) {
	mgr := newTestManager(t)
	mgr.IncrementStats(1, 2)
	snap := mgr.GetCheckpoint()

	mgr.IncrementStats(10, 20)

	if snap.RecordsProcessed != 1 || snap.FramesGenerated != 2 {
		t.Errorf("snapshot changed after live update: records=%d frames=%d, want 1 and 2",
			snap.RecordsProcessed, snap.FramesGenerated)
	}
}
