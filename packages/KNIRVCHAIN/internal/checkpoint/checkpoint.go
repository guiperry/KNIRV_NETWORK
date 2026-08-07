package checkpoint

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"github.com/ethereum/go-ethereum/crypto"
)

// SchemaVersion is the wire/schema tag for KNIRVCHAIN checkpoints.
const SchemaVersion = "knirv.checkpoint.v1"

// CheckpointFinalityDepth is the default number of blocks a checkpoint must sit
// below the chain tip (Design Invariant #3: no checkpoint covers shallow blocks).
const CheckpointFinalityDepth = uint64(32)

// AuthorSig is a single Network Author's signature over a checkpoint's digest.
// PubKeyHex carries the signer's compressed secp256k1 public key (hex, no 0x
// prefix) so the quorum can be verified without an out-of-band key lookup.
type AuthorSig struct {
	Address   string `json:"address"`    // canonical knirv Bech32 address
	PubKeyHex string `json:"pubkey_hex"` // compressed secp256k1 pubkey, hex
	Signature []byte `json:"signature"`  // Cosmos-compatible 64-byte r||s
	Envelope  []byte `json:"envelope"`
}

// Checkpoint is the self-describing checkpoint object KNIRVCHAIN posts to the
// Oracle. It is signed by a quorum of Network Authors over hash(checkpoint).
type Checkpoint struct {
	SchemaVersion string      `json:"schema_version"`
	ChainID       string      `json:"chain_id"`
	StartHeight   uint64      `json:"start_height"`
	EndHeight     uint64      `json:"end_height"`
	Root          [32]byte    `json:"root"`
	PrevCheckHash [32]byte    `json:"prev_check_hash"`
	Proposer      string      `json:"proposer"`
	Signatures    []AuthorSig `json:"signatures"`
}

// Digest returns the canonical 32-byte hash over the checkpoint body that
// authors sign. It excludes the Signatures field. It must stay byte-identical
// to KNIRVORACLE's types.Checkpoint.Digest so cross-service signatures verify.
func (c *Checkpoint) Digest() [32]byte {
	buf := make([]byte, 8)
	appendU64 := func(v uint64) {
		for i := 0; i < 8; i++ {
			buf[i] = byte(v >> (56 - i*8))
		}
	}
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
	_ = appendU64
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
// signs: the hex encoding of the digest. KnirvWallet.signOracleMessage hashes
// this with keccak256(utf8(msg)) and signs with secp256k1, producing a 64-byte
// raw signature — the exact scheme this package mirrors.
func (c *Checkpoint) SignedMessage() string {
	return fmt.Sprintf("%x", c.Digest())
}

// SignCheckpoint signs the checkpoint with the given author key and appends the
// signature, matching KNIRVCONTROLLER's KnirvWallet.signOracleMessage.
func SignCheckpoint(c *Checkpoint, key *ecdsa.PrivateKey) error {
	if key == nil {
		return fmt.Errorf("nil private key")
	}
	signed, err := signEnvelope(key, c.SignedMessage(), "chain-checkpoint", c.ChainID)
	if err != nil {
		return err
	}
	c.Signatures = append(c.Signatures, AuthorSig{
		Address: signed.Address, PubKeyHex: hex.EncodeToString(signed.PublicKey),
		Signature: signed.Signature, Envelope: signed.Envelope,
	})
	return nil
}

// VerifySignature verifies one AuthorSig against the checkpoint digest.
func (c *Checkpoint) VerifySignature(sig AuthorSig) bool {
	pubBytes, err := hex.DecodeString(stripHexPrefix(sig.PubKeyHex))
	if err != nil {
		return false
	}
	signed := knirvsigning.SignedMessage{Envelope: sig.Envelope, Signature: sig.Signature, PublicKey: pubBytes, Address: sig.Address}
	return knirvsigning.VerifyMessagePayload(signed, "knirv.chain", "chain-checkpoint", c.ChainID, []byte(c.SignedMessage()), time.Now()) == nil
}

// VerifyQuorum checks that the checkpoint carries valid, unique signatures from
// enough members of the provided author set to satisfy the quorum by count.
func (c *Checkpoint) VerifyQuorum(authors map[string]bool, quorumNumer, quorumDenom uint64) bool {
	if quorumDenom == 0 {
		return false
	}
	seen := make(map[string]bool)
	valid := uint64(0)
	for _, sig := range c.Signatures {
		if seen[sig.Address] {
			continue
		}
		if !authors[sig.Address] {
			continue
		}
		if !c.VerifySignature(sig) {
			continue
		}
		seen[sig.Address] = true
		valid++
	}
	if valid == 0 {
		return false
	}
	return valid*quorumDenom >= quorumNumer
}

// SignMessage signs msg with the KNIRVCONTROLLER scheme: keccak256(utf8(msg))
// with secp256k1, returning the 64-byte raw signature (r||s).
func signEnvelope(key *ecdsa.PrivateKey, msg, purpose, chainID string) (knirvsigning.SignedMessage, error) {
	now := time.Now().Unix()
	return knirvsigning.SignMessage(crypto.FromECDSA(key), knirvsigning.MessageEnvelope{
		Domain: "knirv.chain", Purpose: purpose, ChainID: chainID, Nonce: msg,
		IssuedAtUnix: now, ExpiresAtUnix: math.MaxInt64, Payload: []byte(msg),
	})
}

func SignMessage(key *ecdsa.PrivateKey, msg string) ([]byte, error) {
	signed, err := signEnvelope(key, msg, "service-message", "knirv-1")
	if err != nil {
		return nil, err
	}
	return signed.Signature, nil
}

// OracleAddress derives the controller's oracle address from a secp256k1 key:
// 0x + keccak256(uncompressed pubkey without 0x04 prefix)[12:].
func OracleAddress(pub *ecdsa.PublicKey) string {
	address, err := knirvsigning.Address(crypto.CompressPubkey(pub), knirvsigning.DefaultAddressPrefix)
	if err != nil {
		panic(err)
	}
	return address
}

func stripHexPrefix(s string) string {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		return s[2:]
	}
	return s
}
