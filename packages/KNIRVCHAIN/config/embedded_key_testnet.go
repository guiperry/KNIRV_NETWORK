//go:build testnet
// +build testnet

package config

import "log"

// EmbeddedRootKeyData holds the default root key data for testnet builds
// This is a mock key that doesn't require the embedded file
var EmbeddedRootKeyData = []byte(`-----BEGIN TESTNET MOCK KEY-----
KNIRV-ORACLE-TESTNET-MOCK-ROOT-KEY
This is a mock key used for testnet builds only.
Not suitable for production use.
Generated automatically for testing purposes.
UUID: testnet-mock-key-12345
-----END TESTNET MOCK KEY-----`)

func init() {
	log.Println("Using testnet mock root key (build tag: testnet)")
}
