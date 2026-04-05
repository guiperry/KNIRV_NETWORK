package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

func setupTestWriter(t *testing.T) (*NRVWriter, string) {
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

	return writer, path
}

func createTestFrame(id string) *nrv.Frame {
	return &nrv.Frame{
		ID:     id,
		Vector: [12]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Seed:   [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		Thermo: nrv.ThermoData{TempCelsius: 50.5, VoltageV: 12.0, FreqMHz: 500, FanRPM: 3000},
		Proof:  []byte("test proof data"),
	}
}

func TestNewNRVWriterCreatesFile(t *testing.T) {
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
	defer writer.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected .nrv file to be created")
	}

	if writer.registry == nil {
		t.Error("expected registry to be initialized")
	}

	if writer.registry.Version != 1 {
		t.Errorf("expected registry version 1, got %d", writer.registry.Version)
	}
}

func TestAppendFrame(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frame := createTestFrame("test-frame-1")

	if err := writer.AppendFrame(frame, true, 0.85); err != nil {
		t.Fatalf("AppendFrame failed: %v", err)
	}

	if len(writer.registry.Frames) != 1 {
		t.Errorf("expected 1 frame in registry, got %d", len(writer.registry.Frames))
	}

	entry := writer.registry.Frames[0]
	if entry.ID != "test-frame-1" {
		t.Errorf("expected frame ID 'test-frame-1', got '%s'", entry.ID)
	}

	if !entry.Verified {
		t.Error("expected frame to be verified")
	}

	if entry.ERGORank != 0.85 {
		t.Errorf("expected ERGORank 0.85, got %f", entry.ERGORank)
	}
}

func TestAppendMultipleFrames(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frames := []string{"frame-1", "frame-2", "frame-3"}
	for _, id := range frames {
		frame := createTestFrame(id)
		if err := writer.AppendFrame(frame, true, 0.9); err != nil {
			t.Fatalf("AppendFrame failed for %s: %v", id, err)
		}
	}

	if len(writer.registry.Frames) != 3 {
		t.Errorf("expected 3 frames in registry, got %d", len(writer.registry.Frames))
	}

	if writer.registry.FrameCount != 3 {
		t.Errorf("expected FrameCount 3, got %d", writer.registry.FrameCount)
	}

	if writer.registry.GlobalMetrics.VerifiedFrameCount != 3 {
		t.Errorf("expected VerifiedFrameCount 3, got %d", writer.registry.GlobalMetrics.VerifiedFrameCount)
	}
}

func TestAppendFrameUpdatesMetrics(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frame := createTestFrame("test-frame")
	if err := writer.AppendFrame(frame, true, 0.75); err != nil {
		t.Fatalf("AppendFrame failed: %v", err)
	}

	if writer.registry.GlobalMetrics.ERGORankSum != 0.75 {
		t.Errorf("expected ERGORankSum 0.75, got %f", writer.registry.GlobalMetrics.ERGORankSum)
	}
}

func TestAppendFrameWithSignature(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frame := createTestFrame("signed-frame")
	if err := writer.AppendFrame(frame, true, 0.9); err != nil {
		t.Fatalf("AppendFrame failed: %v", err)
	}

	if _, ok := writer.registry.PQCManifest.FrameSignatures["signed-frame"]; !ok {
		t.Error("expected frame signature to be stored")
	}
}

func TestAppendFrameUnverified(t *testing.T) {
	writer, _ := setupTestWriter(t)
	defer writer.Close()

	frame := createTestFrame("unverified-frame")
	if err := writer.AppendFrame(frame, false, 0.5); err != nil {
		t.Fatalf("AppendFrame failed: %v", err)
	}

	if writer.registry.GlobalMetrics.VerifiedFrameCount != 0 {
		t.Errorf("expected VerifiedFrameCount 0 for unverified frame, got %d", writer.registry.GlobalMetrics.VerifiedFrameCount)
	}
}

func TestNRVWriterClose(t *testing.T) {
	writer, _ := setupTestWriter(t)

	if err := writer.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNewNRVWriterReopensExisting(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nrv")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair failed: %v", err)
	}

	writer1, err := NewNRVWriter(path, keyPair)
	if err != nil {
		t.Fatalf("NewNRVWriter failed: %v", err)
	}

	frame := createTestFrame("persisted-frame")
	if err := writer1.AppendFrame(frame, true, 0.9); err != nil {
		t.Fatalf("AppendFrame failed: %v", err)
	}
	writer1.Close()

	writer2, err := NewNRVWriter(path, keyPair)
	if err != nil {
		t.Fatalf("NewNRVWriter reopen failed: %v", err)
	}
	defer writer2.Close()

	if len(writer2.registry.Frames) != 1 {
		t.Errorf("expected 1 frame after reopen, got %d", len(writer2.registry.Frames))
	}
}
