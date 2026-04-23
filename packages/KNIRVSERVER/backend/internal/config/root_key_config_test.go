package config

import (
	pb "backend_server/internal/proto"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestGetRootKeyPathPrefersCanonicalPath(t *testing.T) {
	configHome := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", homeDir)

	canonicalPath := filepath.Join(configHome, "knirv-server", "root.key")
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(canonicalPath, []byte("canonical"), 0o600); err != nil {
		t.Fatalf("WriteFile(canonical) error = %v", err)
	}

	legacyPath := filepath.Join(homeDir, ".knirv", "root.key")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() legacy error = %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o600); err != nil {
		t.Fatalf("WriteFile(legacy) error = %v", err)
	}

	got, err := GetRootKeyPath()
	if err != nil {
		t.Fatalf("GetRootKeyPath() error = %v", err)
	}
	if got != canonicalPath {
		t.Fatalf("GetRootKeyPath() = %q, want %q", got, canonicalPath)
	}
}

func TestGetRootKeyPathFallsBackToLegacyPath(t *testing.T) {
	configHome := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", homeDir)

	legacyPath := filepath.Join(homeDir, ".knirv", "root.key")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(legacyPath, mustMarshalEncryptedRootKeyFile(t), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := GetRootKeyPath()
	if err != nil {
		t.Fatalf("GetRootKeyPath() error = %v", err)
	}
	if got != legacyPath {
		t.Fatalf("GetRootKeyPath() = %q, want legacy path %q", got, legacyPath)
	}
}

func TestGetRootKeyPathFallsBackWhenCanonicalIsInvalid(t *testing.T) {
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
	if err := os.WriteFile(legacyPath, mustMarshalEncryptedRootKeyFile(t), 0o600); err != nil {
		t.Fatalf("WriteFile(legacy) error = %v", err)
	}

	got, err := GetRootKeyPath()
	if err != nil {
		t.Fatalf("GetRootKeyPath() error = %v", err)
	}
	if got != legacyPath {
		t.Fatalf("GetRootKeyPath() = %q, want legacy path %q", got, legacyPath)
	}
}

func mustMarshalEncryptedRootKeyFile(t *testing.T) []byte {
	t.Helper()

	data, err := proto.Marshal(&pb.EncryptedRootKeyFile{
		EncryptedContent: []byte("ciphertext"),
		Salt:             []byte("0123456789abcdef"),
	})
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	return data
}
