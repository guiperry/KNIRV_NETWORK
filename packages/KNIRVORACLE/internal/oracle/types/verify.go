package types

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	btcecsecp "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
)

// Digest returns the canonical 32-byte hash over the checkpoint body that
// authors sign. It excludes the Signatures field. It must stay byte-identical
// to KNIRVCHAIN's checkpoint.Checkpoint.Digest so cross-service signatures
// verify. The scheme matches KNIRVCONTROLLER's KnirvWallet: keccak256 over the
// canonical body (schema|0x00|chainID|0x00|start|end|root|prev|proposer|0x00).
func (c *Checkpoint) Digest() [32]byte {
	parts := [][]byte{
		[]byte(c.SchemaVersion),
		[]byte{0x00},
		[]byte(c.ChainID),
		[]byte{0x00},
		u64Bytes(c.StartHeight),
		u64Bytes(c.EndHeight),
		c.Root[:],
		c.PrevCheckHash[:],
		[]byte(c.Proposer),
		[]byte{0x00},
	}
	var data []byte
	for _, p := range parts {
		data = append(data, p...)
	}
	var out [32]byte
	copy(out[:], crypto.Keccak256(data))
	return out
}

func u64Bytes(v uint64) []byte {
	b := make([]byte, 8)
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> (56 - i*8))
	}
	return b
}

// SignedMessage is the canonical string the official KNIRVCONTROLLER wallet
// signs: the hex encoding of the digest.
func (c *Checkpoint) SignedMessage() string {
	return fmt.Sprintf("%x", c.Digest())
}

// VerifyAuthorSig verifies a single AuthorSig against the checkpoint digest,
// using the KNIRVCONTROLLER signing scheme: keccak256(utf8(hex digest)) signed
// with secp256k1, producing a 64-byte raw signature (r||s).
func VerifyAuthorSig(cp *Checkpoint, sig AuthorSig) bool {
	return VerifyAuthorSigDigest(cp.Digest(), sig)
}

// VerifyAuthorSigDigest verifies a single AuthorSig against an explicit digest.
// The signed message is the hex encoding of the digest (same as
// Checkpoint.SignedMessage), so a rotation over a registration body uses the
// body's sha256 digest encoded as hex.
func VerifyAuthorSigDigest(digest [32]byte, sig AuthorSig) bool {
	pubBytes, err := hex.DecodeString(stripHexPrefix(sig.PubKeyHex))
	if err != nil {
		return false
	}
	pub, err := btcec.ParsePubKey(pubBytes)
	if err != nil {
		return false
	}
	if OracleAddress(pub.ToECDSA()) != sig.Address {
		return false
	}
	return VerifyMessage(pub, fmt.Sprintf("%x", digest), sig.Signature)
}

// VerifyMessage verifies a 64-byte raw secp256k1 signature over keccak256(msg).
func VerifyMessage(pub *btcec.PublicKey, msg string, sig []byte) bool {
	if len(sig) != 64 {
		return false
	}
	hash := crypto.Keccak256([]byte(msg))
	var r, s btcec.ModNScalar
	r.SetByteSlice(sig[:32])
	s.SetByteSlice(sig[32:64])
	signature := btcecsecp.NewSignature(&r, &s)
	return signature.Verify(hash, pub)
}

// OracleAddress derives the controller's oracle address from a secp256k1 key:
// 0x + keccak256(uncompressed pubkey without 0x04 prefix)[12:].
func OracleAddress(pub *ecdsa.PublicKey) string {
	unc := crypto.FromECDSAPub(pub)
	sum := crypto.Keccak256(unc[1:])
	return "0x" + hex.EncodeToString(sum[12:])
}

func stripHexPrefix(s string) string {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return s[2:]
	}
	return s
}

// SignMessage signs msg with the KNIRVCONTROLLER scheme: keccak256(utf8(msg))
// with secp256k1, returning the 64-byte raw signature (r||s).
func SignMessage(key *ecdsa.PrivateKey, msg string) ([]byte, error) {
	hash := crypto.Keccak256([]byte(msg))
	full, err := crypto.Sign(hash, key) // 65 bytes: r||s||v
	if err != nil {
		return nil, err
	}
	return full[:64], nil
}

// SignCheckpoint signs the checkpoint with key and appends the AuthorSig, using
// the same keccak256 + secp256k1 raw-sig scheme as KNIRVCHAIN and the official
// KNIRVCONTROLLER wallet, so cross-service signatures verify.
func SignCheckpoint(cp *Checkpoint, key *ecdsa.PrivateKey) error {
	sig, err := SignMessage(key, cp.SignedMessage())
	if err != nil {
		return err
	}
	cpk := crypto.CompressPubkey(key.Public().(*ecdsa.PublicKey))
	addr := OracleAddress(key.Public().(*ecdsa.PublicKey))
	cp.Signatures = append(cp.Signatures, AuthorSig{
		Address:   addr,
		PubKeyHex: hex.EncodeToString(cpk),
		Signature: sig,
	})
	return nil
}
