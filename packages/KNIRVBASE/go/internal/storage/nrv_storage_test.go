package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

func setupTestStorage(t *testing.T) (*NRVStorage, string) {
	t.Helper()
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "storage")

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair failed: %v", err)
	}

	storage := NewNRVStorage(baseDir, keyPair)
	_ = baseDir // used in tests
	return storage, baseDir
}

func TestNewNRVStorage(t *testing.T) {
	storage, _ := setupTestStorage(t)
	if storage == nil {
		t.Fatal("expected storage to be created")
	}

	if storage.baseDir == "" {
		t.Error("expected baseDir to be set")
	}

	if storage.keyPair == nil {
		t.Error("expected keyPair to be set")
	}

	if storage.fileStore == nil {
		t.Error("expected fileStore to be initialized")
	}
}

func TestNRVStorageInsertAndFind(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	doc := map[string]interface{}{
		"id": "test-doc-1",
		"payload": map[string]interface{}{
			"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			"seed":   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			"thermo": map[string]float64{
				"temp_celsius": 50.5,
				"voltage_v":    12.0,
				"freq_mhz":     500,
				"fan_rpm":      3000,
			},
			"proof": "test proof string",
		},
		"verified":  true,
		"ergo_rank": 0.85,
	}

	if err := storage.Insert(ctx, "test-collection", doc); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	foundDoc, err := storage.Find(ctx, "test-collection", "test-doc-1")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}

	if foundDoc == nil {
		t.Fatal("expected document to be found")
	}

	if foundDoc["id"] != "test-doc-1" {
		t.Errorf("expected id 'test-doc-1', got '%v'", foundDoc["id"])
	}

	if foundDoc["verified"] != true {
		t.Error("expected verified to be true")
	}

	if foundDoc["ergo_rank"] != 0.85 {
		t.Errorf("expected ergo_rank 0.85, got %v", foundDoc["ergo_rank"])
	}
}

func TestNRVStorageFindAll(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	docs := []map[string]interface{}{
		{
			"id": "findall-doc-1",
			"payload": map[string]interface{}{
				"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				"seed":   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				"thermo": map[string]float64{
					"temp_celsius": 50.0,
					"voltage_v":    12.0,
					"freq_mhz":     500,
					"fan_rpm":      3000,
				},
				"proof": "test proof 1",
			},
			"verified":  true,
			"ergo_rank": 0.8,
		},
		{
			"id": "findall-doc-2",
			"payload": map[string]interface{}{
				"vector": []float64{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
				"seed":   []byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
				"thermo": map[string]float64{
					"temp_celsius": 51.0,
					"voltage_v":    12.1,
					"freq_mhz":     501,
					"fan_rpm":      3001,
				},
				"proof": "test proof 2",
			},
			"verified":  false,
			"ergo_rank": 0.7,
		},
	}

	for _, doc := range docs {
		if err := storage.Insert(ctx, "test-collection", doc); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	allDocs, err := storage.FindAll(ctx, "test-collection")
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(allDocs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(allDocs))
	}

	for _, doc := range allDocs {
		switch doc["id"] {
		case "findall-doc-1":
			if doc["verified"] != true {
				t.Error("expected first document verified to be true")
			}
		case "findall-doc-2":
			if doc["verified"] != false {
				t.Error("expected second document verified to be false")
			}
		}
	}
}

func TestNRVStorageDelete(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	doc := map[string]interface{}{
		"id": "delete-test-doc",
		"payload": map[string]interface{}{
			"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			"seed":   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			"thermo": map[string]float64{
				"temp_celsius": 50.5,
				"voltage_v":    12.0,
				"freq_mhz":     500,
				"fan_rpm":      3000,
			},
			"proof": "test proof for deletion",
		},
		"verified":  true,
		"ergo_rank": 0.9,
	}

	if err := storage.Insert(ctx, "delete-collection", doc); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if err := storage.Delete(ctx, "delete-collection", "delete-test-doc"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	foundDoc, err := storage.Find(ctx, "delete-collection", "delete-test-doc")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}

	if foundDoc != nil {
		t.Error("expected deleted document to return nil")
	}
}

func TestNRVStorageGetModality(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	doc := map[string]interface{}{
		"id": "modality-test-doc",
		"payload": map[string]interface{}{
			"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			"seed":   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			"thermo": map[string]float64{
				"temp_celsius": 50.5,
				"voltage_v":    12.0,
				"freq_mhz":     500,
				"fan_rpm":      3000,
			},
			"proof": []byte("test proof bytes"),
		},
		"verified":  true,
		"ergo_rank": 0.85,
	}

	if err := storage.Insert(ctx, "modality-collection", doc); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	vectorBytes, err := storage.GetModality(ctx, "modality-collection", "modality-test-doc", nrv.ModalityVector)
	if err != nil {
		t.Fatalf("GetModality failed: %v", err)
	}

	if len(vectorBytes) != 48 {
		t.Errorf("expected vector modality length 48, got %d", len(vectorBytes))
	}

	seedBytes, err := storage.GetModality(ctx, "modality-collection", "modality-test-doc", nrv.ModalitySeed)
	if err != nil {
		t.Fatalf("GetModality failed: %v", err)
	}

	if len(seedBytes) != 32 {
		t.Errorf("expected seed modality length 32, got %d", len(seedBytes))
	}

	thermoBytes, err := storage.GetModality(ctx, "modality-collection", "modality-test-doc", nrv.ModalityThermo)
	if err != nil {
		t.Fatalf("GetModality failed: %v", err)
	}

	if len(thermoBytes) != 16 {
		t.Errorf("expected thermo modality length 16, got %d", len(thermoBytes))
	}

	proofBytes, err := storage.GetModality(ctx, "modality-collection", "modality-test-doc", nrv.ModalityProof)
	if err != nil {
		t.Fatalf("GetModality failed: %v", err)
	}

	if len(proofBytes) != 16 {
		t.Errorf("expected proof modality length 16, got %d", len(proofBytes))
	}
}

func TestNRVStorageStreamFrames(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	frames := []string{"stream-doc-1", "stream-doc-2"}
	for i, id := range frames {
		doc := map[string]interface{}{
			"id": id,
			"payload": map[string]interface{}{
				"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				"seed":   []byte{byte(i), byte(i + 1), byte(i + 2), byte(i + 3), byte(i + 4), byte(i + 5), byte(i + 6), byte(i + 7), byte(i + 8), byte(i + 9), byte(i + 10), byte(i + 11), byte(i + 12), byte(i + 13), byte(i + 14), byte(i + 15), byte(i + 16), byte(i + 17), byte(i + 18), byte(i + 19), byte(i + 20), byte(i + 21), byte(i + 22), byte(i + 23), byte(i + 24), byte(i + 25), byte(i + 26), byte(i + 27), byte(i + 28), byte(i + 29), byte(i + 30), byte(i + 31)},
				"thermo": map[string]float64{
					"temp_celsius": 50.5,
					"voltage_v":    12.0,
					"freq_mhz":     500,
					"fan_rpm":      3000,
				},
				"proof": []byte("stream proof"),
			},
			"verified":  true,
			"ergo_rank": 0.9,
		}

		if err := storage.Insert(ctx, "stream-collection", doc); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
	}

	frameChan, err := storage.StreamFrames(ctx, "stream-collection", nrv.ModalityVector)
	if err != nil {
		t.Fatalf("StreamFrames failed: %v", err)
	}

	count := 0
	for range frameChan {
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 frames from stream, got %d", count)
	}
}

func TestNRVStorageClose(t *testing.T) {
	storage, _ := setupTestStorage(t)

	if err := storage.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
