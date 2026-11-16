package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultSystemAppDataDir is the system-wide app data directory the KNIRVSERVER
// launcher prefers. It is a var so tests can point it at a temp directory.
var defaultSystemAppDataDir = "/var/lib/knirvserver"

// resolveAppDataDir returns the app data directory with the same order as the
// KNIRVSERVER launcher (getAppDataDir): KNIRV_APP_DATA_DIR, then
// /var/lib/knirvserver when it is writable, then $XDG_DATA_HOME/knirvserver, then
// ~/.local/share/knirvserver. ulorad must agree with the launcher on this path
// when neither ULORA_DATA_DIR nor KNIRV_APP_DATA_DIR is set, so the duplication is
// deliberate: this module is standalone and cannot import the launcher.
func resolveAppDataDir() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); explicit != "" {
		return explicit, nil
	}
	if mkdirIfUsable(defaultSystemAppDataDir) {
		return defaultSystemAppDataDir, nil
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		return filepath.Join(xdg, "knirvserver"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve app data dir: no home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "knirvserver"), nil
}

// mkdirIfUsable creates path and reports whether it is writable, matching the
// launcher's probe (a temp file is created and removed).
func mkdirIfUsable(path string) bool {
	if err := os.MkdirAll(path, 0755); err != nil {
		return false
	}
	f, err := os.CreateTemp(path, ".knirv-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	closeErr := f.Close()
	removeErr := os.Remove(name)
	return closeErr == nil && removeErr == nil
}
