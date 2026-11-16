package config

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAuthTokenFileWritesModeZeroSixHundred(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ulora")
	if err := WriteAuthTokenFile(dir, "fake-token-test-only"); err != nil {
		t.Fatalf("WriteAuthTokenFile: %v", err)
	}
	info, err := os.Stat(AuthTokenPath(dir))
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("token file mode = %o, want 0600", perm)
	}
	raw, err := os.ReadFile(AuthTokenPath(dir))
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	if strings.TrimSpace(string(raw)) != "fake-token-test-only" {
		t.Fatalf("token file content mismatch")
	}
}

func TestWriteAuthTokenFileReplacesExistingAndLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	if err := WriteAuthTokenFile(dir, "first-test-only"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := WriteAuthTokenFile(dir, "second-test-only"); err != nil {
		t.Fatalf("second write: %v", err)
	}
	raw, _ := os.ReadFile(AuthTokenPath(dir))
	if strings.TrimSpace(string(raw)) != "second-test-only" {
		t.Fatalf("rotation did not replace the token")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected only the token file, found %d entries", len(entries))
	}
}

func TestWriteAuthTokenFileRefusesEmptyToken(t *testing.T) {
	dir := t.TempDir()
	if err := WriteAuthTokenFile(dir, ""); err == nil {
		t.Fatal("an empty token must not be written")
	}
	if _, err := os.Stat(AuthTokenPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("no token file should exist after refusing, stat err = %v", err)
	}
}

func TestLoadOrCreateAuthTokenGeneratesModeZeroSixHundred(t *testing.T) {
	dir := t.TempDir()
	tok, err := LoadOrCreateAuthToken(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateAuthToken: %v", err)
	}
	if len(tok) != 2*generatedTokenBytes {
		t.Fatalf("token length = %d, want %d", len(tok), 2*generatedTokenBytes)
	}
	info, err := os.Stat(AuthTokenPath(dir))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("token file mode = %o, want 0600", perm)
	}
}

func TestLoadOrCreateAuthTokenIsRandomPerDataDir(t *testing.T) {
	a, err := LoadOrCreateAuthToken(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadOrCreateAuthToken(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("two fresh data dirs produced the same token")
	}
}

func TestLoadOrCreateAuthTokenReusesExistingFile(t *testing.T) {
	dir := t.TempDir()
	if err := WriteAuthTokenFile(dir, "existing-token-test-only"); err != nil {
		t.Fatal(err)
	}
	tok, err := LoadOrCreateAuthToken(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateAuthToken: %v", err)
	}
	if tok != "existing-token-test-only" {
		t.Fatal("an existing 0600 token file must be reused, not replaced")
	}
}

func TestLoadOrCreateAuthTokenReplacesUnsafeFiles(t *testing.T) {
	cases := map[string]struct {
		content string
		mode    os.FileMode
	}{
		"world readable": {"unsafe-token-test-only", 0o644},
		"group readable": {"unsafe-token-test-only", 0o640},
		"empty":          {"   ", 0o600},
		"multi-word":     {"part-one part-two", 0o600},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := AuthTokenPath(dir)
			if err := os.WriteFile(path, []byte(c.content+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, c.mode); err != nil {
				t.Fatal(err)
			}
			tok, err := LoadOrCreateAuthToken(dir)
			if err != nil {
				t.Fatalf("LoadOrCreateAuthToken: %v", err)
			}
			if tok == c.content {
				t.Fatal("unusable token file was reused")
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o600 {
				t.Fatalf("replacement mode = %o, want 0600", info.Mode().Perm())
			}
		})
	}
}

func TestLoadOrCreateAuthTokenLeavesOnlyTokenFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadOrCreateAuthToken(dir); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != AuthTokenFileName {
		t.Fatalf("unexpected data dir contents: %v", entries)
	}
}

// Replacing an unusable token file logs the file path and the reason, and never
// the token value that was in the file.
func TestLoadOrCreateAuthTokenLogsReplacementWithoutToken(t *testing.T) {
	const secret = "secret-token-value-test-only"
	cases := map[string]struct {
		content string
		mode    os.FileMode
		reason  string
	}{
		"world readable": {secret, 0o644, "grants group or other access"},
		"group readable": {secret, 0o640, "grants group or other access"},
		// Content-based reason: a multi-word file with the secret inside must
		// not have its contents echoed into the log.
		"multi-word": {secret + " extra-words-test-only", 0o600, "file does not hold a single token word"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := AuthTokenPath(dir)
			if err := os.WriteFile(path, []byte(c.content+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, c.mode); err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			origOut := log.Writer()
			log.SetOutput(&buf)
			defer log.SetOutput(origOut)

			if _, err := LoadOrCreateAuthToken(dir); err != nil {
				t.Fatalf("LoadOrCreateAuthToken: %v", err)
			}
			logged := buf.String()
			if !strings.Contains(logged, path) {
				t.Fatalf("log does not name the file %s: %q", path, logged)
			}
			if !strings.Contains(logged, c.reason) {
				t.Fatalf("log does not give the reason %q: %q", c.reason, logged)
			}
			if strings.Contains(logged, secret) {
				t.Fatalf("log leaked the token value: %q", logged)
			}
		})
	}
}

// A first start (no file yet) is not a problem and logs nothing.
func TestLoadOrCreateAuthTokenDoesNotLogOnFirstStart(t *testing.T) {
	var buf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(origOut)

	if _, err := LoadOrCreateAuthToken(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("first start logged: %q", buf.String())
	}
}
