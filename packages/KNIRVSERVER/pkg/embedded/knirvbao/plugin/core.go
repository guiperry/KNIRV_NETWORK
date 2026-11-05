package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
)

const embeddedCoreVersion = "0.2.0"
const embeddedCoreDigest = "cbbe06852d55d5835bc22606424fe54bab83e24d5f67144b0049ce42ebbaa634"

//go:embed assets/knirv_core.wasm
var embeddedCore []byte

// installEmbeddedCore gives this process an isolated SDK cache. The SDK also
// checks its own pinned digest on load; the plugin never needs a network fetch.
func installEmbeddedCore() (func(), error) {
	digest := sha256.Sum256(embeddedCore)
	if hex.EncodeToString(digest[:]) != embeddedCoreDigest {
		return nil, errors.New("embedded SDK core digest mismatch")
	}
	dir, err := os.MkdirTemp("", "knirvbao-core-")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	versionDir := filepath.Join(dir, embeddedCoreVersion)
	if err := os.Mkdir(versionDir, 0700); err != nil {
		cleanup()
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(versionDir, "knirv_core.wasm"), embeddedCore, 0600); err != nil {
		cleanup()
		return nil, err
	}
	if err := os.Setenv("KNIRV_SDK_DATA_DIR", dir); err != nil {
		cleanup()
		return nil, err
	}
	if err := os.Setenv("KNIRV_WASM_CACHE_DIR", filepath.Join(dir, "compiled")); err != nil {
		cleanup()
		return nil, err
	}
	// A mismatched SDK pin must fail closed without reaching a release server.
	if err := os.Setenv("KNIRV_SDK_BASE_URL", "disabled://offline"); err != nil {
		cleanup()
		return nil, err
	}
	return cleanup, nil
}
