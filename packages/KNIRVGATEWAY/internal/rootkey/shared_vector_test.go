package rootkey

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
	t.Setenv("KNIRV_ROOT_KEY_PATH", path)
	t.Setenv("ORACLE_KEY_PASSWORD", v.Password)
	creds, err := LoadCloudflareCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds.AccountID != "cf-account" || creds.APIToken != "cf-token" {
		t.Fatal("shared vector fields differ")
	}
}
