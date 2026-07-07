package knirvoracle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalRootKeyPathIsHiddenUnderUserConfigDir(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	got, err := CanonicalRootKeyPath()
	if err != nil {
		t.Fatalf("CanonicalRootKeyPath() error = %v", err)
	}

	want := filepath.Join(configHome, "knirv-server", ".key", "root.key")
	if got != want {
		t.Fatalf("CanonicalRootKeyPath() = %q, want %q", got, want)
	}
}

func TestResolveRootKeyPathFindsCanonicalLocation(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	canonicalPath := filepath.Join(configHome, "knirv-server", ".key", "root.key")
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0o700); err != nil {
		t.Fatalf("MkdirAll(canonical) error = %v", err)
	}
	if err := os.WriteFile(canonicalPath, []byte("root-key-contents"), 0o600); err != nil {
		t.Fatalf("WriteFile(canonical) error = %v", err)
	}

	got, err := ResolveRootKeyPath("")
	if err != nil {
		t.Fatalf("ResolveRootKeyPath() error = %v", err)
	}
	if got != canonicalPath {
		t.Fatalf("ResolveRootKeyPath() = %q, want %q", got, canonicalPath)
	}
}

func TestResolveRootKeyPathNoLongerFallsBackToLegacyLocations(t *testing.T) {
	configHome := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", homeDir)

	// Legacy location from before consolidation. It must NOT be found
	// anymore — root.key lives in exactly one place now.
	legacyPath := filepath.Join(homeDir, ".knirv", "root.key")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(legacy) error = %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o600); err != nil {
		t.Fatalf("WriteFile(legacy) error = %v", err)
	}

	if _, err := ResolveRootKeyPath(""); err == nil {
		t.Fatalf("ResolveRootKeyPath() succeeded using a legacy location; want error")
	}
}

func TestResolveRootKeyPathHonorsExplicitConfiguredOverride(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	overrideDir := t.TempDir()
	overridePath := filepath.Join(overrideDir, "custom-root.key")
	if err := os.WriteFile(overridePath, []byte("override-contents"), 0o600); err != nil {
		t.Fatalf("WriteFile(override) error = %v", err)
	}

	// Also place a file at the canonical location to confirm the explicit
	// override takes priority over it rather than the other way around.
	canonicalPath := filepath.Join(configHome, "knirv-server", ".key", "root.key")
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0o700); err != nil {
		t.Fatalf("MkdirAll(canonical) error = %v", err)
	}
	if err := os.WriteFile(canonicalPath, []byte("canonical-contents"), 0o600); err != nil {
		t.Fatalf("WriteFile(canonical) error = %v", err)
	}

	got, err := ResolveRootKeyPath(overridePath)
	if err != nil {
		t.Fatalf("ResolveRootKeyPath() error = %v", err)
	}
	if got != overridePath {
		t.Fatalf("ResolveRootKeyPath() = %q, want override path %q", got, overridePath)
	}
}

func TestResolveAndValidateRootKeyRejectsInvalidCanonicalFile(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	canonicalPath := filepath.Join(configHome, "knirv-server", ".key", "root.key")
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0o700); err != nil {
		t.Fatalf("MkdirAll(canonical) error = %v", err)
	}
	if err := os.WriteFile(canonicalPath, []byte("{not-a-valid-protobuf-envelope}"), 0o600); err != nil {
		t.Fatalf("WriteFile(canonical) error = %v", err)
	}

	// With no fallback location left to try, an invalid canonical file must
	// surface its validation error rather than silently succeed elsewhere.
	if _, err := ResolveAndValidateRootKey(""); err == nil {
		t.Fatalf("ResolveAndValidateRootKey() succeeded with an invalid canonical file; want error")
	}
}
