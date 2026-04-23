package knirvoracle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRootKeyPathFallsBackToLegacyLocation(t *testing.T) {
	configHome := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", homeDir)

	legacyPath := filepath.Join(homeDir, ".knirv", "root.key")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := ResolveRootKeyPath("")
	if err != nil {
		t.Fatalf("ResolveRootKeyPath() error = %v", err)
	}
	if got != legacyPath {
		t.Fatalf("ResolveRootKeyPath() = %q, want %q", got, legacyPath)
	}
}

func TestResolveAndValidateRootKeyFallsBackWhenCanonicalIsInvalid(t *testing.T) {
	configHome := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", homeDir)

	canonicalPath := filepath.Join(configHome, "knirv-server", "root.key")
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(canonical) error = %v", err)
	}
	if err := os.WriteFile(canonicalPath, []byte("{legacy-json}"), 0o600); err != nil {
		t.Fatalf("WriteFile(canonical) error = %v", err)
	}

	legacyPath := filepath.Join(homeDir, ".knirv", "root.key")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(legacy) error = %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte{0x0a, 0x01, 0x78, 0x12, 0x01, 0x79}, 0o600); err != nil {
		t.Fatalf("WriteFile(legacy) error = %v", err)
	}

	got, err := ResolveAndValidateRootKey("")
	if err != nil {
		t.Fatalf("ResolveAndValidateRootKey() error = %v", err)
	}
	if got != legacyPath {
		t.Fatalf("ResolveAndValidateRootKey() = %q, want %q", got, legacyPath)
	}
}
