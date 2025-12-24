package distributed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/knirv/knirvbase/internal/crypto/pqc"
)

func TestFileStorage_PQCEncryption(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "knirvbase_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage
	storage := NewFileStorage(tmpDir)

	// Generate master key
	masterKey, err := pqc.GeneratePQCKeyPair("master", "encryption")
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	storage.SetMasterKey(masterKey)
	storage.encryptionMgr.CacheKey(masterKey.ID, masterKey) // Explicitly cache the key
	t.Logf("Master key ID: %s", masterKey.ID)

	// Create a document with sensitive data
	doc := map[string]interface{}{
		"id":        "test-cred-1",
		"entryType": "CREDENTIAL",
		"payload": map[string]interface{}{
			"username": "alice@example.com",
			"hash":     "sensitive_hash_data",
			"salt":     "sensitive_salt_data",
			"email":    "alice@example.com", // not sensitive
		},
	}

	// Insert document (should be encrypted)
	err = storage.Insert("credentials", doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Read document back (should be decrypted)
	retrieved, err := storage.Find("credentials", "test-cred-1")
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Document not found")
	}

	// Check that sensitive fields are decrypted
	payload := retrieved["payload"].(map[string]interface{})
	if payload["hash"] != "sensitive_hash_data" {
		t.Errorf("Hash not decrypted correctly: got %v", payload["hash"])
	}

	if payload["salt"] != "sensitive_salt_data" {
		t.Errorf("Salt not decrypted correctly: got %v", payload["salt"])
	}

	if payload["email"] != "alice@example.com" {
		t.Errorf("Email not preserved: got %v", payload["email"])
	}

	// Check that the file on disk is encrypted
	docPath := filepath.Join(tmpDir, "credentials", "test-cred-1.json")
	encryptedData, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	// Parse the encrypted document
	var encryptedDoc map[string]interface{}
	if err := json.Unmarshal(encryptedData, &encryptedDoc); err != nil {
		t.Fatalf("Failed to parse encrypted document: %v", err)
	}

	// Should have encryption metadata
	if _, ok := encryptedDoc["encrypted"]; !ok {
		t.Error("Document should be marked as encrypted")
	}

	if keyID, ok := encryptedDoc["encryption_key_id"]; ok {
		t.Logf("Document encrypted with key ID: %v", keyID)
	} else {
		t.Error("Document should have encryption key ID")
	}

	// Payload should be encrypted
	if _, ok := encryptedDoc["payload"]; !ok {
		t.Error("Encrypted document should have payload")
	}
}
