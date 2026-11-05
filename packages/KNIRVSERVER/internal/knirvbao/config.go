// Package knirvbao owns the lifecycle of the node-local OpenBao process.
package knirvbao

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config controls the optional node-local enclave. DataDir is the existing
// application data directory, not a path to an existing OpenBao installation.
// ConfigDir is the node's KNIRV config directory (where the key files live);
// the one-time first-boot recovery credentials are written there.
type Config struct {
	Enabled   bool
	DataDir   string
	ConfigDir string
}

func (c Config) root() (string, error) {
	if !safePath(c.DataDir) || filepath.Clean(c.DataDir) == string(filepath.Separator) {
		return "", errors.New("OpenBao requires an absolute application data directory")
	}
	return filepath.Join(c.DataDir, "openbao"), nil
}

func loopbackAddress(address string) bool {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n > 0 && n <= 65535
}

func safePath(path string) bool {
	if !filepath.IsAbs(path) || strings.Contains(path, "${") || strings.Contains(path, "%{") {
		return false
	}
	for _, r := range path {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// renderConfig binds the extra Raft listener only to loopback. All service
// clients use the owner-only Unix socket; the TCP listener is Raft's local
// cluster requirement and must never bind a non-loopback interface.
func renderConfig(root, pluginDir, pluginSHA, api, cluster string, seal []byte) (string, error) {
	if !safePath(root) || !safePath(pluginDir) || len(filepath.Join(root, "bao.sock")) > 103 || !loopbackAddress(api) || !loopbackAddress(cluster) || api == cluster {
		return "", errors.New("invalid OpenBao paths or loopback addresses")
	}
	decoded, err := hex.DecodeString(pluginSHA)
	if err != nil || len(decoded) != sha256.Size || len(seal) != 32 || os.Getuid() < 0 || os.Getgid() < 0 {
		return "", errors.New("invalid plugin digest or seal key")
	}
	sum := sha256.Sum256(seal)
	sealID := hex.EncodeToString(sum[:])[:16]
	q := strconv.Quote
	return fmt.Sprintf(`storage "raft" {
  path = %s
  node_id = "knirv"
}
listener "tcp" {
  address = %s
  cluster_address = %s
  tls_disable = true
}
listener "unix" {
  address = %s
  socket_mode = "0600"
  socket_user = %s
  socket_group = %s
}
seal "static" {
  current_key = "env://KNIRV_BAO_SEAL_KEY"
  current_key_id = %s
}
api_addr = %s
cluster_addr = %s
plugin_directory = %s
plugin "secret" "knirv" {
  command = "openbao-plugin-secrets-knirv"
  version = "v0.1.0"
  sha256sum = %s
}
audit "file" "knirv" {
  options { file_path = %s }
}
`, q(filepath.Join(root, "data")), q(api), q(cluster), q(filepath.Join(root, "bao.sock")), q(strconv.Itoa(os.Getuid())), q(strconv.Itoa(os.Getgid())), q(sealID), q("http://"+api), q("https://"+cluster), q(pluginDir), q(strings.ToLower(pluginSHA)), q(filepath.Join(root, "audit.log"))), nil
}
