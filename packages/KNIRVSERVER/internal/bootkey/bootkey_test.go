package bootkey

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootKeyUsesCanonicalHiddenKeyDirectory(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("KNIRV_CONFIG_DIR", "")
	t.Setenv("ORACLE_KEY_PATH", "")

	want := filepath.Join(configHome, "knirv-server", ".key", "root.key")
	if err := os.MkdirAll(filepath.Dir(want), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("encrypted-root-key"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := FindRootKey(); got != want {
		t.Fatalf("FindRootKey() = %q, want %q", got, want)
	}
}

func TestFindRootKeyExplicitOverrideWins(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(explicit, []byte("encrypted-root-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORACLE_KEY_PATH", explicit)

	if got := FindRootKey(); got != explicit {
		t.Fatalf("FindRootKey() = %q, want %q", got, explicit)
	}
}
