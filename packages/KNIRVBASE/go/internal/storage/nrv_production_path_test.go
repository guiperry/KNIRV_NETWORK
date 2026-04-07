package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/knirvcorp/knirvbase/go/internal/crypto/pqc"
)

func TestProductionAppDataDirectory(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory, skipping production path test")
	}

	// Standard production app data directory paths
	testPaths := []string{
		filepath.Join(homeDir, ".local", "share", "KNIRV", "test-storage"),
		filepath.Join(homeDir, "Library", "Application Support", "KNIRV", "test-storage"),
		filepath.Join(os.TempDir(), "KNIRV", "production-test"),
	}

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair failed: %v", err)
	}

	for _, appDataPath := range testPaths {
		t.Run(filepath.Base(appDataPath), func(t *testing.T) {
			// Clean up before and after
			os.RemoveAll(appDataPath)
			defer os.RemoveAll(appDataPath)

			storage := NewNRVStorage(appDataPath, keyPair)
			if storage == nil {
				t.Fatal("expected storage to be created with production path")
			}

			if storage.baseDir != appDataPath {
				t.Errorf("expected baseDir %s, got %s", appDataPath, storage.baseDir)
			}

			// Verify directory was created with correct permissions
			info, err := os.Stat(appDataPath)
			if err != nil {
				t.Fatalf("Failed to stat app data directory: %v", err)
			}

			if !info.IsDir() {
				t.Error("expected app data path to be directory")
			}

			// Verify permissions are secure (0700)
			mode := info.Mode().Perm()
			if mode != 0700 {
				t.Errorf("Expected directory permissions 0700, got %#o", mode)
			}

			// Test full operation in production path
			ctx := context.Background()
			testDoc := map[string]interface{}{
				"id": "prod-test-doc",
				"payload": map[string]interface{}{
					"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
					"seed":   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				},
				"verified":  true,
				"ergo_rank": 0.95,
			}

			if err := storage.Insert(ctx, "prod-collection", testDoc); err != nil {
				t.Fatalf("Insert failed in production path: %v", err)
			}

			foundDoc, err := storage.Find(ctx, "prod-collection", "prod-test-doc")
			if err != nil {
				t.Fatalf("Find failed in production path: %v", err)
			}

			if foundDoc == nil {
				t.Fatal("expected document to be found in production path")
			}

			if foundDoc["id"] != "prod-test-doc" {
				t.Errorf("expected document id to match")
			}

			if err := storage.Close(); err != nil {
				t.Errorf("Close failed: %v", err)
			}
		})
	}
}

func TestExistingProductionFiles(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory")
	}

	appDataPath := filepath.Join(homeDir, ".local", "share", "KNIRV", "existing-test")
	os.RemoveAll(appDataPath)
	defer os.RemoveAll(appDataPath)

	keyPair, err := pqc.GeneratePQCKeyPair("test-key", "test")
	if err != nil {
		t.Fatalf("GeneratePQCKeyPair failed: %v", err)
	}

	// First run: create file
	storage1 := NewNRVStorage(appDataPath, keyPair)
	ctx := context.Background()
	
	testDoc := map[string]interface{}{
		"id": "persistent-doc",
		"payload": map[string]interface{}{
			"vector": []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		"verified": true,
	}

	if err := storage1.Insert(ctx, "persistent-collection", testDoc); err != nil {
		t.Fatalf("Initial insert failed: %v", err)
	}

	if err := storage1.Close(); err != nil {
		t.Fatalf("First close failed: %v", err)
	}

	// Second run: open existing file
	storage2 := NewNRVStorage(appDataPath, keyPair)
	
	foundDoc, err := storage2.Find(ctx, "persistent-collection", "persistent-doc")
	if err != nil {
		t.Fatalf("Find existing document failed: %v", err)
	}

	if foundDoc == nil {
		t.Fatal("Expected to find existing document from previous run")
	}

	if err := storage2.Close(); err != nil {
		t.Fatalf("Second close failed: %v", err)
	}
}