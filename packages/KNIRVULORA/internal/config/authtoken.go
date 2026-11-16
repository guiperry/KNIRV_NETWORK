package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// AuthTokenFileName is the file ulorad writes its bearer token to, inside its
// data directory. KNIRVGRAPH resolves the same name under the same directory
// (ULORA_DATA_DIR, or KNIRV_APP_DATA_DIR/ulora), so a client on the same host can
// obtain the token without the operator copying it around. The file is mode 0600
// and holds only the credential; it is never logged.
const AuthTokenFileName = "auth_token"

// AuthTokenPath returns where the bearer token file lives for a data directory.
func AuthTokenPath(dataDir string) string {
	return filepath.Join(dataDir, AuthTokenFileName)
}

// WriteAuthTokenFile atomically writes token to AuthTokenPath(dataDir) with mode
// 0600. It writes a temp file in the same directory and renames it into place, so
// a reader never sees a partial token. Errors never include the token value.
func WriteAuthTokenFile(dataDir, token string) error {
	if token == "" {
		return fmt.Errorf("refusing to write an empty auth token file")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir for auth token: %w", err)
	}
	tmp, err := os.CreateTemp(dataDir, "."+AuthTokenFileName+"-*")
	if err != nil {
		return fmt.Errorf("create auth token temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	// CreateTemp opens with 0600; set it explicitly so the guarantee does not
	// depend on that default.
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod auth token temp file: %w", err)
	}
	if _, err := tmp.WriteString(token + "\n"); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write auth token temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync auth token temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close auth token temp file: %w", err)
	}
	if err := os.Rename(tmpName, AuthTokenPath(dataDir)); err != nil {
		cleanup()
		return fmt.Errorf("install auth token file: %w", err)
	}
	return nil
}

// generatedTokenBytes is the entropy of a token ulorad generates (256 bits).
const generatedTokenBytes = 32

// maxAuthTokenFileBytes bounds how much of an existing token file is read.
const maxAuthTokenFileBytes = 4096

// LoadOrCreateAuthToken returns the bearer token ulorad uses when ULORA_AUTH_TOKEN
// is not set. It reuses the token in AuthTokenPath(dataDir) when that file is a
// regular file readable only by its owner and holds one non-empty word. Otherwise
// it generates a random token and installs it with WriteAuthTokenFile. The file is
// never deleted on exit, so the next start reuses the same token. The token value
// is never logged or included in errors.
func LoadOrCreateAuthToken(dataDir string) (string, error) {
	path := AuthTokenPath(dataDir)
	tok, reason := readReusableAuthToken(path)
	if tok != "" {
		return tok, nil
	}
	if reason != "" {
		// Names the file and the reason, never the token value.
		log.Printf("ulorad: replacing unusable auth token file %s: %s", path, reason)
	}
	buf := make([]byte, generatedTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate auth token: %w", err)
	}
	tok = hex.EncodeToString(buf)
	if err := WriteAuthTokenFile(dataDir, tok); err != nil {
		return "", err
	}
	return tok, nil
}

// readReusableAuthToken returns the token in path when the file is usable as-is.
// Otherwise it returns "" and a reason that names what is wrong with the file
// (never its contents). A missing file returns "" with no reason: that is a first
// start, not a problem. The caller replaces the file.
func readReusableAuthToken(path string) (tok string, reason string) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ""
		}
		return "", "cannot stat file"
	}
	if !info.Mode().IsRegular() {
		return "", "not a regular file (symlinks are never reused)"
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Sprintf("mode %04o grants group or other access (want 0600)", info.Mode().Perm())
	}
	f, err := os.Open(path)
	if err != nil {
		return "", "cannot open file"
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxAuthTokenFileBytes+1))
	if err != nil {
		return "", "cannot read file"
	}
	if len(raw) > maxAuthTokenFileBytes {
		return "", "file is too large"
	}
	tok = strings.TrimSpace(string(raw))
	if tok == "" {
		return "", "file is empty"
	}
	if strings.ContainsAny(tok, " \t\r\n") {
		return "", "file does not hold a single token word"
	}
	return tok, ""
}
