package knirvbao

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestEmbeddedBinaries(t *testing.T) {
	if len(BaoBinary()) == 0 {
		t.Fatal("OpenBao binary missing; run make knirvbao-build")
	}
	plugin := PluginBinary()
	if len(plugin) == 0 {
		t.Fatal("signing plugin missing; run make knirvbao-build")
	}
	digest := sha256.Sum256(plugin)
	if PluginSHA256() != hex.EncodeToString(digest[:]) {
		t.Fatal("plugin digest differs from embedded bytes")
	}
}
