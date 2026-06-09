package dvepod

import (
	_ "embed"
	"crypto/sha256"
	"encoding/hex"
)

//go:embed dvepod.wasm
var embeddedWASM []byte

func EmbeddedWASMHash() string {
	h := sha256.Sum256(embeddedWASM)
	return hex.EncodeToString(h[:])
}

func EmbeddedWASMSize() int {
	return len(embeddedWASM)
}

func HasEmbeddedWASM() bool {
	return len(embeddedWASM) > 0
}
