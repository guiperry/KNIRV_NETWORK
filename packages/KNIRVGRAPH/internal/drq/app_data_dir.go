package drq

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultSystemAppDataDir is the system-wide app data directory the KNIRVSERVER
// launcher prefers. It is a var so tests can point it at a temp directory.
var defaultSystemAppDataDir = "/var/lib/knirvserver"

// resolveAppDataDir returns the launcher's app data directory, in the same order
// as KNIRVSERVER's getAppDataDir: KNIRV_APP_DATA_DIR, then /var/lib/knirvserver
// when it is writable, then $XDG_DATA_HOME/knirvserver, then
// ~/.local/share/knirvserver. It is duplicated (not imported) because KNIRVGRAPH
// and the launcher are separate modules; ulorad (KNIRVULORA) keeps the same copy.
func resolveAppDataDir() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv(knirvAppDataDirEnv)); explicit != "" {
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
