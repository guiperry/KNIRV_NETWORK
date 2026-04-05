package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
)

func setupTestCompactor(t *testing.T) (*Compactor, *NRVWriter, string) {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair failed: %v", err)
	}

	writer, err := NewNRVWriter(path, keyPair)
	if err != nil {
		t.Fatalf("NewNRVWriter failed: %v", err)
	}

	compactor := NewCompactor(path, keyPair)

	return compactor, writer, path
}

func TestCompactorMaybeCompactBelowThreshold(t *testing.T) {
	compactor, writer, _ := setupTestCompactor(t)
	defer writer.Close()

	for i := 0; i < 10; i++ {
		frame := createTestFrame("frame-below-threshold")
		if err := writer.AppendFrame(frame, true, 0.9); err != nil {
			t.Fatalf("AppendFrame failed: %v", err)
		}
	}

	compactor.MaybeCompact(writer.registry)

	time.Sleep(100 * time.Millisecond)

	if compactor.running {
		t.Error("expected compaction not to start below threshold")
	}
}

func TestCompactorMaybeCompactAboveThreshold(t *testing.T) {
	compactor, writer, path := setupTestCompactor(t)
	defer writer.Close()

	totalFrames := 10
	deleteCount := 3

	for i := 0; i < totalFrames; i++ {
		frame := createTestFrame("frame-above-threshold")
		if err := writer.AppendFrame(frame, true, 0.9); err != nil {
			t.Fatalf("AppendFrame failed: %v", err)
		}
	}

	for i := 0; i < deleteCount; i++ {
		nowNano := time.Now().UnixNano()
		writer.registry.Frames[i].Tombstone = &nowNano
		writer.registry.TombstoneCount++
	}

	if err := writer.saveRegistry(); err != nil {
		t.Fatalf("saveRegistry failed: %v", err)
	}

	ratio := float64(writer.registry.TombstoneCount) / float64(writer.registry.FrameCount)
	if ratio < compactionThreshold {
		t.Fatalf("test setup error: ratio %f is below threshold %f", ratio, compactionThreshold)
	}

	compactor.MaybeCompact(writer.registry)

	time.Sleep(500 * time.Millisecond)

	reader, err := NewNRVReader(path)
	if err != nil {
		t.Fatalf("NewNRVReader after compaction failed: %v", err)
	}
	defer reader.Close()

	liveCount := 0
	for _, entry := range reader.registry.Frames {
		if entry.Tombstone == nil {
			liveCount++
		}
	}

	expectedLive := totalFrames - deleteCount
	if liveCount != expectedLive {
		t.Errorf("expected %d live frames after compaction, got %d", expectedLive, liveCount)
	}
}

func TestCompactorCompactPreservesData(t *testing.T) {
	compactor, writer, path := setupTestCompactor(t)
	defer writer.Close()

	frameIDs := []string{"compact-frame-1", "compact-frame-2", "compact-frame-3"}
	for _, id := range frameIDs {
		frame := createTestFrame(id)
		if err := writer.AppendFrame(frame, true, 0.85); err != nil {
			t.Fatalf("AppendFrame failed: %v", err)
		}
	}

	nowNano := time.Now().UnixNano()
	writer.registry.Frames[0].Tombstone = &nowNano
	writer.registry.TombstoneCount++

	if err := writer.saveRegistry(); err != nil {
		t.Fatalf("saveRegistry failed: %v", err)
	}

	if err := compactor.compact(); err != nil {
		t.Fatalf("compact failed: %v", err)
	}

	reader, err := NewNRVReader(path)
	if err != nil {
		t.Fatalf("NewNRVReader after compaction failed: %v", err)
	}
	defer reader.Close()

	if len(reader.registry.Frames) != 2 {
		t.Errorf("expected 2 frames after compaction, got %d", len(reader.registry.Frames))
	}

	for _, entry := range reader.registry.Frames {
		if entry.Tombstone != nil {
			t.Error("expected no tombstoned frames after compaction")
		}
	}
}

func TestCompactorCompactUpdatesMetrics(t *testing.T) {
	compactor, writer, path := setupTestCompactor(t)
	defer writer.Close()

	for i := 0; i < 5; i++ {
		frame := createTestFrame("metrics-frame")
		if err := writer.AppendFrame(frame, true, 0.8); err != nil {
			t.Fatalf("AppendFrame failed: %v", err)
		}
	}

	nowNano := time.Now().UnixNano()
	writer.registry.Frames[0].Tombstone = &nowNano
	writer.registry.TombstoneCount++

	if err := writer.saveRegistry(); err != nil {
		t.Fatalf("saveRegistry failed: %v", err)
	}

	if err := compactor.compact(); err != nil {
		t.Fatalf("compact failed: %v", err)
	}

	reader, err := NewNRVReader(path)
	if err != nil {
		t.Fatalf("NewNRVReader after compaction failed: %v", err)
	}
	defer reader.Close()

	if reader.registry.TombstoneCount != 0 {
		t.Errorf("expected TombstoneCount 0 after compaction, got %d", reader.registry.TombstoneCount)
	}

	if reader.registry.GlobalMetrics.CompactedAt == nil {
		t.Error("expected CompactedAt to be set after compaction")
	}
}

func TestCompactorStartStop(t *testing.T) {
	compactor, _, _ := setupTestCompactor(t)

	compactor.Start()

	time.Sleep(50 * time.Millisecond)

	compactor.Stop()

	if compactor.running {
		t.Error("expected compactor to be stopped")
	}
}
