package main

import (
	"os"
	"path/filepath"
	"testing"

	pb "knirv-server/proto"
)

func TestPasswordPromptIntegration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "knirv_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	keyFilePath := filepath.Join(tempDir, "test_key.key")

	// Test data
	testContent := &pb.RootKeyFileContentProto{
		StripeSecretKey:       "sk_test_integration",
		StripeWebhookSecret:   "whsec_test_integration",
		CoinbaseApiKey:        "api_key_integration",
		CoinbaseWebhookSecret: "webhook_secret_integration",
		RootPrivateKeyHex:     "0xabcdef123456789",
	}

	password := []byte("test_password_123")

	// Test key file creation
	err = CreateEncryptedKeyFile(testContent, password, keyFilePath)
	if err != nil {
		t.Fatalf("Failed to create encrypted key file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {
		t.Error("Key file was not created")
	}

	// Test key file loading
	loadedContent, err := LoadEncryptedKeyFile(keyFilePath, password)
	if err != nil {
		t.Fatalf("Failed to load encrypted key file: %v", err)
	}

	// Verify content matches
	if loadedContent.StripeSecretKey != testContent.StripeSecretKey {
		t.Errorf("StripeSecretKey mismatch: expected %s, got %s", testContent.StripeSecretKey, loadedContent.StripeSecretKey)
	}
	if loadedContent.CoinbaseApiKey != testContent.CoinbaseApiKey {
		t.Errorf("CoinbaseApiKey mismatch: expected %s, got %s", testContent.CoinbaseApiKey, loadedContent.CoinbaseApiKey)
	}
	if loadedContent.RootPrivateKeyHex != testContent.RootPrivateKeyHex {
		t.Errorf("RootPrivateKeyHex mismatch: expected %s, got %s", testContent.RootPrivateKeyHex, loadedContent.RootPrivateKeyHex)
	}
}

func TestWrongPassword(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "knirv_test_wrong_pass")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	keyFilePath := filepath.Join(tempDir, "test_key.key")

	// Test data
	testContent := &pb.RootKeyFileContentProto{
		RootPrivateKeyHex: "0xabcdef123456789",
	}

	correctPassword := []byte("correct_password")
	wrongPassword := []byte("wrong_password")

	// Create key file
	err = CreateEncryptedKeyFile(testContent, correctPassword, keyFilePath)
	if err != nil {
		t.Fatalf("Failed to create encrypted key file: %v", err)
	}

	// Try to load with wrong password - should fail
	_, err = LoadEncryptedKeyFile(keyFilePath, wrongPassword)
	if err == nil {
		t.Error("Expected decryption to fail with wrong password, but it succeeded")
	}
}