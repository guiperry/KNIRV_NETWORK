package utils

import (
	"testing"

	pb "knirv-nexus/proto"
	"google.golang.org/protobuf/proto"
)

func TestCryptoUtils(t *testing.T) {
	// Test salt generation
	salt, err := GenerateSalt(SaltLen)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	if len(salt) != SaltLen {
		t.Errorf("Expected salt length %d, got %d", SaltLen, len(salt))
	}

	// Test key derivation
	password := []byte("test_password")
	key, err := DeriveKeyFromPassword(password, salt, ScryptN, ScryptR, ScryptP, KeyLen)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}
	if len(key) != KeyLen {
		t.Errorf("Expected key length %d, got %d", KeyLen, len(key))
	}

	// Test encryption/decryption
	plaintext := []byte("Hello, World!")
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Error("Ciphertext is empty")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decryption failed: expected %s, got %s", plaintext, decrypted)
	}
}

func TestProtobufSerialization(t *testing.T) {
	// Create test data
	content := &pb.RootKeyFileContentProto{
		StripeSecretKey:       "sk_test_123",
		StripeWebhookSecret:   "whsec_test_456",
		CoinbaseApiKey:        "api_key_test",
		CoinbaseWebhookSecret: "webhook_secret_test",
		RootPrivateKeyHex:     "0x123456789abcdef",
	}

	// Marshal
	data, err := proto.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal protobuf: %v", err)
	}

	// Unmarshal
	var decoded pb.RootKeyFileContentProto
	if err := proto.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal protobuf: %v", err)
	}

	// Verify
	if decoded.StripeSecretKey != content.StripeSecretKey {
		t.Errorf("StripeSecretKey mismatch: expected %s, got %s", content.StripeSecretKey, decoded.StripeSecretKey)
	}
	if decoded.RootPrivateKeyHex != content.RootPrivateKeyHex {
		t.Errorf("RootPrivateKeyHex mismatch: expected %s, got %s", content.RootPrivateKeyHex, decoded.RootPrivateKeyHex)
	}
}