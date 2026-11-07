package app

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeJWT builds an unsigned (but structurally valid) JWT whose payload carries
// the expected alg header and the mandatory iss/aud claims plus the given exp.
// readValidationToken does not verify the signature locally; it only enforces
// well-formedness, so this is enough to exercise the fail-closed path.
func fakeJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]interface{}{
		"alg":    "HS256",
		"exp":    exp.Unix(),
		"iss":    "knirv-server",
		"aud":    "knirv-validation-service",
		"sub":    "drq-validation",
		"tenant": "graph",
	})
	return strings.Join([]string{
		base64.RawURLEncoding.EncodeToString(header),
		base64.RawURLEncoding.EncodeToString(payload),
		"signature",
	}, ".")
}

func TestReadValidationTokenFailClosed(t *testing.T) {
	t.Setenv(validationServiceTokenFileEnv, "")
	t.Setenv(validationServiceTokenEnv, "")

	// No token configured at all -> error.
	if _, err := readValidationToken(); err == nil {
		t.Fatal("expected error when no token is configured")
	}

	// Expired token (env) -> error.
	t.Setenv(validationServiceTokenEnv, fakeJWT(t, time.Now().Add(-time.Hour)))
	if _, err := readValidationToken(); err == nil {
		t.Fatal("expected error for expired token")
	}

	// Malformed token -> error.
	t.Setenv(validationServiceTokenEnv, "not-a-jwt")
	if _, err := readValidationToken(); err == nil {
		t.Fatal("expected error for malformed token")
	}

	// Valid token (env) -> ok.
	t.Setenv(validationServiceTokenEnv, fakeJWT(t, time.Now().Add(time.Hour)))
	if tok, err := readValidationToken(); err != nil || tok == "" {
		t.Fatalf("expected valid token, got (%q, %v)", tok, err)
	}

	// File wins over env; empty file -> fail closed.
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty")
	if err := os.WriteFile(emptyFile, []byte("   "), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(validationServiceTokenFileEnv, emptyFile)
	if _, err := readValidationToken(); err == nil {
		t.Fatal("expected error for empty token file")
	}

	// File with a valid token -> ok, and re-reads each call (rotation).
	goodFile := filepath.Join(dir, "good")
	if err := os.WriteFile(goodFile, []byte(fakeJWT(t, time.Now().Add(time.Hour))), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(validationServiceTokenFileEnv, goodFile)
	if tok, err := readValidationToken(); err != nil || tok == "" {
		t.Fatalf("expected valid token from file, got (%q, %v)", tok, err)
	}
}
