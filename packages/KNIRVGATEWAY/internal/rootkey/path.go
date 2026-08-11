// Package rootkey resolves and decrypts KNIRVSERVER's root.key file to
// extract the Cloudflare credentials (account ID, API token) needed to
// query D1 directly. This intentionally duplicates path-resolution logic
// that already exists in two other places — backend_server's
// internal/config/root_key_config.go (KNIRV_CORP) and this repo's
// packages/KNIRVSERVER/pkg/knirvoracle/rootkey.go — rather than importing
// either: backend_server is a separate Go module in a separate repo, and
// pkg/knirvoracle is scoped to KNIRVSERVER's own tree, not meant to be
// imported cross-package by KNIRVGATEWAY (see CLAUDE.md's "no cross-package
// Go imports" convention). If you change path resolution here, check
// whether the other two need the same change.
package rootkey

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveRootKeyPath returns the first existing candidate: an explicit
// KNIRV_ROOT_KEY_PATH override, the canonical <user config dir>/knirv-server/
// .key/root.key (and its non-dotfile sibling), and — since this process is
// typically a child of KNIRVSERVER, itself often invoked under sudo — the
// same two variants under the sudo-invoking user's home.
func ResolveRootKeyPath() (string, error) {
	candidates := make([]string, 0, 6)
	seen := make(map[string]struct{})
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}

	add(os.Getenv("KNIRV_ROOT_KEY_PATH"))

	if configDir, err := os.UserConfigDir(); err == nil {
		add(filepath.Join(configDir, "knirv-server", ".key", "root.key"))
		add(filepath.Join(configDir, "knirv-server", "root.key"))
	}

	if sudoUser := strings.TrimSpace(os.Getenv("SUDO_USER")); sudoUser != "" {
		add(filepath.Join("/home", sudoUser, ".config", "knirv-server", ".key", "root.key"))
		add(filepath.Join("/home", sudoUser, ".config", "knirv-server", "root.key"))
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return candidate, nil
	}

	return "", fmt.Errorf("root.key not found in any expected location")
}
