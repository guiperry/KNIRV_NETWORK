package storage

import (
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

func setupTestReader(t *testing.T) (*NRVReader, *NRVWriter, string) {
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

	frame := createTestFrame("reader-test-frame")
	if err := writer.AppendFrame(frame, true, 0.9); err != nil {
		t.Fatalf("AppendFrame failed: %v", err)
	}
	writer.Close()

	reader, err := NewNRVReader(path)
	if err != nil {
		t.Fatalf("NewNRVReader failed: %v", err)
	}

	return reader, writer, path
}

func TestNewNRVReader(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	if reader.registry == nil {
		t.Error("expected registry to be loaded")
	}

	if len(reader.registry.Frames) != 1 {
		t.Errorf("expected 1 frame in registry, got %d", len(reader.registry.Frames))
	}
}

func TestNRVReaderGetFrame(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	frame, err := reader.GetFrame("reader-test-frame")
	if err != nil {
		t.Fatalf("GetFrame failed: %v", err)
	}

	if frame == nil {
		t.Fatal("expected frame to be found")
	}

	if frame.ID != "reader-test-frame" {
		t.Errorf("expected frame ID 'reader-test-frame', got '%s'", frame.ID)
	}

	for i, v := range frame.Vector {
		expected := float32(i + 1)
		if v != expected {
			t.Errorf("vector[%d]: expected %f, got %f", i, expected, v)
		}
	}
}

func TestNRVReaderGetFrameNotFound(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	frame, err := reader.GetFrame("nonexistent-frame")
	if err == nil {
		t.Error("expected error for nonexistent frame")
	}

	if frame != nil {
		t.Error("expected nil frame for nonexistent ID")
	}
}

func TestNRVReaderGetModality(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	vectorBytes, err := reader.GetModality("reader-test-frame", nrv.ModalityVector)
	if err != nil {
		t.Fatalf("GetModality failed: %v", err)
	}

	if len(vectorBytes) != 48 {
		t.Errorf("expected vector modality length 48, got %d", len(vectorBytes))
	}

	seedBytes, err := reader.GetModality("reader-test-frame", nrv.ModalitySeed)
	if err != nil {
		t.Fatalf("GetModality failed: %v", err)
	}

	if len(seedBytes) != 32 {
		t.Errorf("expected seed modality length 32, got %d", len(seedBytes))
	}
}

func TestNRVReaderGetModalityNotFound(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	_, err := reader.GetModality("nonexistent-frame", nrv.ModalityVector)
	if err == nil {
		t.Error("expected error for nonexistent frame")
	}
}

func TestNRVReaderStreamFrames(t *testing.T) {
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

	frames := []string{"stream-frame-1", "stream-frame-2", "stream-frame-3"}
	for _, id := range frames {
		frame := createTestFrame(id)
		if err := writer.AppendFrame(frame, true, 0.9); err != nil {
			t.Fatalf("AppendFrame failed: %v", err)
		}
	}
	writer.Close()

	reader, err := NewNRVReader(path)
	if err != nil {
		t.Fatalf("NewNRVReader failed: %v", err)
	}
	defer reader.Close()

	ch := reader.StreamFrames("")
	count := 0
	for frame := range ch {
		count++
		if frame == nil {
			t.Error("expected non-nil frame from stream")
		}
	}

	if count != 3 {
		t.Errorf("expected 3 frames from stream, got %d", count)
	}
}

func TestNRVReaderStreamFramesWithModalityFilter(t *testing.T) {
	reader, _, _ := setupTestReader(t)
	defer reader.Close()

	ch := reader.StreamFrames(nrv.ModalityVector)
	count := 0
	for frame := range ch {
		count++
		if frame.Vector[0] != 1 {
			t.Error("expected vector[0] to be 1")
		}
	}

	if count != 1 {
		t.Errorf("expected 1 frame from filtered stream, got %d", count)
	}
}

func TestNRVReaderVerifyFrame(t *testing.T) {
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

	frame := createTestFrame("verify-frame")
	if err := writer.AppendFrame(frame, true, 0.9); err != nil {
		t.Fatalf("AppendFrame failed: %v", err)
	}
	writer.Close()

	reader, err := NewNRVReader(path)
	if err != nil {
		t.Fatalf("NewNRVReader failed: %v", err)
	}
	defer reader.Close()

	valid, err := reader.VerifyFrame("verify-frame", keyPair)
	if err != nil {
		t.Fatalf("VerifyFrame failed: %v", err)
	}

	if !valid {
		t.Error("expected frame signature to be valid")
	}
}

func TestNRVReaderClose(t *testing.T) {
	reader, _, _ := setupTestReader(t)

	if err := reader.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
