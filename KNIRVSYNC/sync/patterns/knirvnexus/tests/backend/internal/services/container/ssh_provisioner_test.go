package container

import (
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestNewSSHProvisioner(t *testing.T) {
	sp := NewSSHProvisioner()

	if sp == nil {
		t.Fatal("SSH provisioner is nil")
	}
}

func TestSSHProvisioner_GenerateSSHKeypair(t *testing.T) {
	sp := NewSSHProvisioner()

	keypair, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate SSH keypair: %v", err)
	}

	if keypair == nil {
		t.Fatal("Keypair is nil")
	}

	if keypair.PublicKey == "" {
		t.Error("Public key is empty")
	}

	if keypair.PrivateKey == "" {
		t.Error("Private key is empty")
	}

	if keypair.KeyFingerprint == "" {
		t.Error("Key fingerprint is empty")
	}

	// Validate public key format
	if !strings.HasPrefix(keypair.PublicKey, "ssh-rsa ") {
		t.Error("Public key doesn't have correct format")
	}

	// Validate private key format
	if !strings.Contains(keypair.PrivateKey, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Error("Private key doesn't have correct format")
	}

	if !strings.Contains(keypair.PrivateKey, "-----END RSA PRIVATE KEY-----") {
		t.Error("Private key doesn't have correct format")
	}
}

func TestSSHProvisioner_ValidateSSHKey(t *testing.T) {
	sp := NewSSHProvisioner()

	// Generate a valid keypair for testing
	keypair, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate keypair for validation test: %v", err)
	}

	// Test valid key
	err = sp.ValidateSSHKey(keypair.PublicKey)
	if err != nil {
		t.Errorf("Valid SSH key was rejected: %v", err)
	}

	// Test invalid key
	err = sp.ValidateSSHKey("invalid-ssh-key")
	if err == nil {
		t.Error("Invalid SSH key was accepted")
	}

	// Test empty key
	err = sp.ValidateSSHKey("")
	if err == nil {
		t.Error("Empty SSH key was accepted")
	}
}

func TestSSHProvisioner_GetSSHConfig(t *testing.T) {
	sp := NewSSHProvisioner()

	// Generate a keypair for testing
	keypair, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate keypair for SSH config test: %v", err)
	}

	config, err := sp.GetSSHConfig(keypair.PrivateKey, "testuser", "localhost", 22)
	if err != nil {
		t.Fatalf("Failed to get SSH config: %v", err)
	}

	if config == nil {
		t.Fatal("SSH config is nil")
	}

	if config.User != "testuser" {
		t.Errorf("Expected user 'testuser', got '%s'", config.User)
	}

	if len(config.Auth) == 0 {
		t.Error("No authentication methods configured")
	}
}

func TestSSHProvisioner_GetSSHConfigInvalidKey(t *testing.T) {
	sp := NewSSHProvisioner()

	_, err := sp.GetSSHConfig("invalid-private-key", "testuser", "localhost", 22)
	if err == nil {
		t.Error("Expected error for invalid private key")
	}
}

func TestSSHProvisioner_TestSSHConnection(t *testing.T) {
	sp := NewSSHProvisioner()

	// Generate a valid keypair for testing
	keypair, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate keypair for connection test: %v", err)
	}

	config, err := sp.GetSSHConfig(keypair.PrivateKey, "testuser", "localhost", 22)
	if err != nil {
		t.Fatalf("Failed to create config for connection test: %v", err)
	}

	// This should not error in the current implementation (placeholder)
	err = sp.TestSSHConnection("localhost", 22, config)
	if err != nil {
		t.Errorf("SSH connection test failed: %v", err)
	}
}

func TestSSHProvisioner_InjectSSHKey(t *testing.T) {
	sp := NewSSHProvisioner()

	// Generate a keypair for testing
	keypair, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate keypair for injection test: %v", err)
	}

	err = sp.InjectSSHKey("test-container", keypair.PublicKey, "testuser")
	if err != nil {
		t.Errorf("SSH key injection failed: %v", err)
	}
}

func TestSSHProvisioner_RevokeSSHKey(t *testing.T) {
	sp := NewSSHProvisioner()

	err := sp.RevokeSSHKey("test-container", "testuser", "dummy-fingerprint")
	if err != nil {
		t.Errorf("SSH key revocation failed: %v", err)
	}
}

func TestSSHProvisioner_ListSSHKeys(t *testing.T) {
	sp := NewSSHProvisioner()

	keys, err := sp.ListSSHKeys("test-container", "testuser")
	if err != nil {
		t.Errorf("Failed to list SSH keys: %v", err)
	}

	// In the current implementation, this returns an empty slice
	if keys == nil {
		t.Error("Keys slice is nil")
	}
}

func TestSSHProvisioner_KeyFingerprint(t *testing.T) {
	sp := NewSSHProvisioner()

	keypair, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate keypair for fingerprint test: %v", err)
	}

	// Parse the public key to verify fingerprint
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keypair.PublicKey))
	if err != nil {
		t.Fatalf("Failed to parse generated public key: %v", err)
	}

	expectedFingerprint := ssh.FingerprintLegacyMD5(publicKey)
	if keypair.KeyFingerprint != expectedFingerprint {
		t.Errorf("Fingerprint mismatch: expected '%s', got '%s'", expectedFingerprint, keypair.KeyFingerprint)
	}
}

func TestSSHProvisioner_MultipleKeypairs(t *testing.T) {
	sp := NewSSHProvisioner()

	// Generate multiple keypairs to ensure they're unique
	keypair1, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate first keypair: %v", err)
	}

	keypair2, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate second keypair: %v", err)
	}

	// Public keys should be different
	if keypair1.PublicKey == keypair2.PublicKey {
		t.Error("Generated identical public keys")
	}

	// Private keys should be different
	if keypair1.PrivateKey == keypair2.PrivateKey {
		t.Error("Generated identical private keys")
	}

	// Fingerprints should be different
	if keypair1.KeyFingerprint == keypair2.KeyFingerprint {
		t.Error("Generated identical fingerprints")
	}
}

func TestSSHProvisioner_ValidateSSHKeyFormats(t *testing.T) {
	sp := NewSSHProvisioner()

	// Generate a valid RSA key for testing
	validKeypair, err := sp.GenerateSSHKeypair()
	if err != nil {
		t.Fatalf("Failed to generate valid key for format testing: %v", err)
	}

	testKeys := []struct {
		key       string
		shouldFail bool
	}{
		{validKeypair.PublicKey, false}, // Valid generated key
		{"invalid-key-format", true},
		{"", true},
		{"ssh-rsa", true},
	}

	for _, test := range testKeys {
		err := sp.ValidateSSHKey(test.key)
		if test.shouldFail && err == nil {
			t.Errorf("Expected validation to fail for key: %s", test.key)
		}
		if !test.shouldFail && err != nil {
			t.Errorf("Expected validation to pass for key: %s, got error: %v", test.key, err)
		}
	}
}