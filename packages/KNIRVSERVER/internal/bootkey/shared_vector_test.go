package bootkey

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedKeyFileVector(t *testing.T) {
	data, err := os.ReadFile("../../../KNIRVSDK/testvectors/secrets/keyfile_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct{ Password, RootKey, BootKey string }
	var wire struct {
		Password string `json:"password"`
		RootKey  string `json:"root_key"`
		BootKey  string `json:"boot_key"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	v.Password, v.RootKey, v.BootKey = wire.Password, wire.RootKey, wire.BootKey
	for name, encoded := range map[string]string{"root": v.RootKey, "boot": v.BootKey} {
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), name+".key")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("KNIRV_"+map[string]string{"root": "ROOT", "boot": "BOOT"}[name]+"_KEY_PATH", path)
	}
	t.Setenv("ORACLE_KEY_PASSWORD", v.Password)
	t.Setenv("BOOT_KEY_PASSWORD", v.Password)
	root, err := LoadRootKeyCloudflareCreds()
	if err != nil {
		t.Fatal(err)
	}
	boot, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if root.CloudflareAccountID != "cf-account" || boot.CloudflareAccountID != "cf-account-b" {
		t.Fatal("shared vector fields differ")
	}
}
