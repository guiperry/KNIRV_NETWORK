// Package knirvbao provides the embedded OpenBao daemon and signing plugin.
// Process management belongs to the launcher, not this package.
package knirvbao

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
)

//go:embed bin/*
var binaries embed.FS

// BaoBinary returns the embedded daemon, or nil if it has not been built.
func BaoBinary() []byte {
	data, _ := binaries.ReadFile("bin/bao")
	return data
}

// PluginBinary returns the embedded signing plugin, or nil if not built.
func PluginBinary() []byte {
	data, _ := binaries.ReadFile("bin/openbao-plugin-secrets-knirv")
	return data
}

// PluginSHA256 is the digest used by OpenBao's declarative plugin catalog.
func PluginSHA256() string {
	data := PluginBinary()
	if len(data) == 0 {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
