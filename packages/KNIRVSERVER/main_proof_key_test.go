package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateProofValidatorKeyPersistsStableSecret(t *testing.T) {
	t.Setenv("KNIRV_PROOF_VALIDATOR_X25519_PRIVATE_KEY", "")
	root := t.TempDir()
	first, err := loadOrCreateProofValidatorKey(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadOrCreateProofValidatorKey(root)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("validator key changed across loads")
	}
	raw, err := base64.StdEncoding.DecodeString(first)
	if err != nil || len(raw) != 32 {
		t.Fatal("persisted validator key has invalid encoding")
	}
	info, err := os.Stat(filepath.Join(root, "trust", "validator-x25519.key"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("validator key permissions = %o", info.Mode().Perm())
	}
}
