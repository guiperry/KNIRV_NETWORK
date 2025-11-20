//go:build !testnet
// +build !testnet

package config

import "log"

// EmbeddedRootKeyData holds the default root key data for production builds
// Since the embedded key file has been moved, using a mock key for now
var EmbeddedRootKeyData = []byte(`-----BEGIN PRODUCTION MOCK KEY-----
KNIRV-ORACLE-PRODUCTION-MOCK-ROOT-KEY
This is a mock key used for production builds.
The original embedded key file has been moved.
Generated automatically as placeholder.
UUID: production-mock-key-67890
-----END PRODUCTION MOCK KEY-----`)

func init() {
	log.Println("Using production mock root key (original embedded file moved)")
}
