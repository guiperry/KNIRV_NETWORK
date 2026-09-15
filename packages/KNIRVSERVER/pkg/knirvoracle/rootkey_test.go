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

// validEnvelope returns a minimal binary protobuf payload that passes
// validateEncryptedRootKeyEnvelope: field 1 (encrypted content) and
// field 2 (salt), both length-delimited and non-empty.
func validEnvelope() []byte {
	return []byte{0x0A, 0x04, 't', 'e', 's', 't', 0x12, 0x04, 's', 'a', 'l', 't'}
}

func TestValidateRootKeyFileHappyPath(t *testing.T) {
	f := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(f, validEnvelope(), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ValidateRootKeyFile(f); err != nil {
		t.Fatalf("ValidateRootKeyFile() = %v, want nil", err)
	}
}

func TestValidateRootKeyFileEmptyFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(f, []byte{}, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ValidateRootKeyFile(f); err == nil || err.Error() != "root.key is empty" {
		t.Fatalf("ValidateRootKeyFile() = %v, want 'root.key is empty'", err)
	}
}

func TestValidateRootKeyFileMissingEncryptedContent(t *testing.T) {
	// Field 2 (salt) only — field 1 (encrypted content) absent.
	f := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(f, []byte{0x12, 0x04, 's', 'a', 'l', 't'}, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ValidateRootKeyFile(f); err == nil || err.Error() != "root.key is missing encrypted content" {
		t.Fatalf("ValidateRootKeyFile() = %v, want missing encrypted content", err)
	}
}

func TestValidateRootKeyFileMissingSalt(t *testing.T) {
	// Field 1 (encrypted content) only — field 2 (salt) absent.
	f := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(f, []byte{0x0A, 0x04, 't', 'e', 's', 't'}, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ValidateRootKeyFile(f); err == nil || err.Error() != "root.key is missing salt" {
		t.Fatalf("ValidateRootKeyFile() = %v, want missing salt", err)
	}
}

func TestResolveAndValidateRootKeySkipsInvalidCanonicalPicksValidDotless(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	canonicalDir := filepath.Join(configHome, "knirv-server", ".key")
	if err := os.MkdirAll(canonicalDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	canonicalPath := filepath.Join(canonicalDir, "root.key")
	// Place an invalid file at the canonical .key/ location.
	if err := os.WriteFile(canonicalPath, []byte("invalid"), 0o600); err != nil {
		t.Fatalf("WriteFile(canonical): %v", err)
	}

	// Place a valid envelope at the non-dotkey sibling location.
	dotlessPath := filepath.Join(configHome, "knirv-server", "root.key")
	if err := os.WriteFile(dotlessPath, validEnvelope(), 0o600); err != nil {
		t.Fatalf("WriteFile(dotless): %v", err)
	}

	got, err := ResolveAndValidateRootKey("")
	if err != nil {
		t.Fatalf("ResolveAndValidateRootKey() error = %v", err)
	}
	if got != dotlessPath {
		t.Fatalf("ResolveAndValidateRootKey() = %q, want dotless path %q", got, dotlessPath)
	}
}

func TestResolveAndValidateRootKeyPrefersCanonicalOverDotless(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	canonicalPath := filepath.Join(configHome, "knirv-server", ".key", "root.key")
	if err := os.MkdirAll(filepath.Dir(canonicalPath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(canonicalPath, validEnvelope(), 0o600); err != nil {
		t.Fatalf("WriteFile(canonical): %v", err)
	}

	dotlessPath := filepath.Join(configHome, "knirv-server", "root.key")
	if err := os.WriteFile(dotlessPath, validEnvelope(), 0o600); err != nil {
		t.Fatalf("WriteFile(dotless): %v", err)
	}

	got, err := ResolveAndValidateRootKey("")
	if err != nil {
		t.Fatalf("ResolveAndValidateRootKey() error = %v", err)
	}
	if got != canonicalPath {
		t.Fatalf("ResolveAndValidateRootKey() = %q, want canonical path %q", got, canonicalPath)
	}
}

func TestValidateRootKeyFileTruncatedVarint(t *testing.T) {
	// A length-delimited tag followed by a varint length with the high bit
	// set and no continuation byte (truncated varint).
	f := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(f, []byte{0x0A, 0x80}, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ValidateRootKeyFile(f); err == nil {
		t.Fatalf("ValidateRootKeyFile() = nil, want error for truncated varint")
	}
}

func TestValidateRootKeyFileUnsupportedWireType(t *testing.T) {
	// Tag with wire type 3 (start group) — not supported by the parser.
	// field 1, wire type 3: tag = (1 << 3) | 3 = 0x0B.
	f := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(f, []byte{0x0B, 0x00}, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ValidateRootKeyFile(f); err == nil {
		t.Fatalf("ValidateRootKeyFile() = nil, want error for unsupported wire type")
	}
}

func TestValidateRootKeyFileTruncatedLengthDelimitedField(t *testing.T) {
	// Field 1 claims 10 bytes but only 2 are present.
	f := filepath.Join(t.TempDir(), "root.key")
	if err := os.WriteFile(f, []byte{0x0A, 0x0A, 0x01, 0x02}, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ValidateRootKeyFile(f); err == nil {
		t.Fatalf("ValidateRootKeyFile() = nil, want error for truncated length-delimited field")
	}
}
