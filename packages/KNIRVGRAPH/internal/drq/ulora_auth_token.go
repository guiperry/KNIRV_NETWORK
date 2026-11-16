package drq

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Resolution of ulorad's bearer token, so KNIRVGRAPH can authenticate without the
// operator copying a secret into its environment.
//
// Order (first hit wins; a source that is set but unusable fails closed rather
// than falling through to a weaker one):
//
//  1. ULORA_AUTH_TOKEN: explicit value (same variable ulorad reads).
//  2. ULORA_AUTH_TOKEN_FILE: a handoff file a secrets provider can rewrite; it is
//     re-read on every compile, so rotation needs no restart. Mode 0600 or 0640.
//  3. ulorad's own token file: <ULORA_DATA_DIR>/auth_token, or when that is unset,
//     <KNIRV_APP_DATA_DIR>/ulora/auth_token, or else <launcher app data dir>/ulora/
//     auth_token (see resolveAppDataDir). ulorad writes this file (mode 0600) into
//     the same directory, using the same chain.
//
// Nothing here logs or returns the token value.

const (
	uloraAuthTokenFileEnv = "ULORA_AUTH_TOKEN_FILE"
	uloraDataDirEnv       = "ULORA_DATA_DIR"
	knirvAppDataDirEnv    = "KNIRV_APP_DATA_DIR"
	// uloraAuthTokenFileName must match KNIRVULORA internal/config.AuthTokenFileName.
	uloraAuthTokenFileName = "auth_token"
	uloraAuthTokenMaxBytes = 4096
)

// resolveULoRAAuthToken returns the bearer token ulorad will accept, or
// ErrULoRAAuthTokenUnavailable when no source provides one.
func resolveULoRAAuthToken() (string, error) {
	// A variable that is set but blank is unusable, so it fails closed instead of
	// falling through to a weaker source.
	if raw := os.Getenv(uloraAuthTokenEnv); raw != "" {
		return checkULoRAAuthToken(strings.TrimSpace(raw), uloraAuthTokenEnv)
	}

	if raw := os.Getenv(uloraAuthTokenFileEnv); raw != "" {
		path := strings.TrimSpace(raw)
		if path == "" {
			return "", fmt.Errorf("%s is set but blank", uloraAuthTokenFileEnv)
		}
		return readULoRAAuthTokenFile(path)
	}

	dir, err := uloraDataDirFromEnv()
	if err != nil {
		return "", fmt.Errorf("resolve ulorad data dir: %w", err)
	}
	// A missing file maps to ErrULoRAAuthTokenUnavailable inside the reader.
	return readULoRAAuthTokenFile(filepath.Join(dir, uloraAuthTokenFileName))
}

// uloraDataDirFromEnv names ulorad's data directory. It uses the same chain as
// ulorad (KNIRVULORA internal/config): ULORA_DATA_DIR, then
// <KNIRV_APP_DATA_DIR>/ulora, then <launcher app data dir>/ulora.
func uloraDataDirFromEnv() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(uloraDataDirEnv)); dir != "" {
		return dir, nil
	}
	appDataDir, err := resolveAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDataDir, "ulora"), nil
}

// readULoRAAuthTokenFile reads a token file that must not be readable by anyone
// other than its owner. A file that exists but is unusable is an error, not a
// silent fallback.
func readULoRAAuthTokenFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrULoRAAuthTokenUnavailable
		}
		return "", fmt.Errorf("stat ulorad auth token file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("ulorad auth token path %s is not a regular file", path)
	}
	// Accepted modes: 0600 (ulorad's own file) and 0640 (a handoff file that the
	// owner's group may read). Refused: any group write bit and any other-user bit.
	if info.Mode().Perm()&0o027 != 0 {
		return "", fmt.Errorf("ulorad auth token file %s has unsafe mode %04o (want 0600 or 0640); refusing to use it",
			path, info.Mode().Perm())
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open ulorad auth token file: %w", err)
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, uloraAuthTokenMaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read ulorad auth token file: %w", err)
	}
	if len(raw) > uloraAuthTokenMaxBytes {
		return "", fmt.Errorf("ulorad auth token file %s is too large", path)
	}
	return checkULoRAAuthToken(strings.TrimSpace(string(raw)), path)
}

// checkULoRAAuthToken rejects an empty or multi-line value. source names where the
// value came from; the value itself is never included in the error.
func checkULoRAAuthToken(tok, source string) (string, error) {
	if tok == "" {
		return "", fmt.Errorf("ulorad auth token from %s is empty", source)
	}
	if strings.ContainsAny(tok, " \t\r\n") {
		return "", fmt.Errorf("ulorad auth token from %s is malformed", source)
	}
	return tok, nil
}
