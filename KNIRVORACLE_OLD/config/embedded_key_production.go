//go:build !testnet
// +build !testnet

package config

import (
	_ "embed"
	"log"
)

//go:embed embedded/default_root.key
var EmbeddedRootKeyData []byte

func init() {
	log.Println("Using embedded root key from file (production build)")
}
