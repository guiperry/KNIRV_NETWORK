package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestSharedKeyFileVector(t *testing.T) {
	data, err := os.ReadFile("../../../KNIRVSDK/testvectors/secrets/keyfile_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Password string `json:"password"`
		RootKey  string `json:"root_key"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(v.RootKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	root, err := loadEncryptedKeyFile(path, []byte(v.Password))
	if err != nil {
		t.Fatal(err)
	}
	if root.CloudflareAPIToken != "cf-token" || root.StripeSecretKey != "sk_test_stripe" {
		t.Fatal("shared vector fields differ")
	}
	t.Setenv("ORACLE_KEY_PATH", path)
	t.Setenv("KNIRV_ROOT_KEY_PATH", path)
	t.Setenv("ORACLE_KEY_PASSWORD", v.Password)
	if signer := replicationFileSigner(root, zap.NewNop()); signer == nil || signer.Address() == "" {
		t.Fatal("file provider did not supply the unlocked root-owner signer")
	}
	root.RootPrivateKeyHex = "0000000000000000000000000000000000000000000000000000000000000002"
	if signer := replicationFileSigner(root, zap.NewNop()); signer != nil {
		t.Fatal("file provider signer for a different root identity was accepted")
	}
}
