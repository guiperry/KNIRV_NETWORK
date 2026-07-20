package registry

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// signCheckpoint signs cp with key using the KNIRVCONTROLLER scheme
// (keccak256(utf8(hex digest)) + secp256k1 raw 64-byte sig) and returns an
// AuthorSig carrying the controller oracle address.
func signCheckpoint(t *testing.T, cp *types.Checkpoint, key *ecdsa.PrivateKey, addr, pubHex string) types.AuthorSig {
	t.Helper()
	msg := cp.SignedMessage()
	hash := crypto.Keccak256([]byte(msg))
	full, err := crypto.Sign(hash, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return types.AuthorSig{Address: addr, PubKeyHex: pubHex, Signature: full[:64]}
}

// genAuthor returns a registered author + its secp256k1 private key, with the
// controller oracle address and compressed-pubkey hex.
func genAuthor(t *testing.T) (types.RegisteredAuthor, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	cpk := crypto.CompressPubkey(&key.PublicKey)
	addr := types.OracleAddress(&key.PublicKey)
	return types.RegisteredAuthor{Address: addr, PubKey: cpk, Weight: 1}, key
}

func hexOf(b []byte) string {
	const hext = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hext[v>>4]
		out[i*2+1] = hext[v&0x0f]
	}
	return string(out)
}
