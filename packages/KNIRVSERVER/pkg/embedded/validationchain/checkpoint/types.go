// Package checkpoint posts Validation Chain's merkle root to KNIRVORACLE.
//
// The signing scheme here must stay byte-identical to KNIRVORACLE's
// internal/oracle/types.Checkpoint.Digest / SignCheckpoint (and to
// KNIRVCHAIN's internal/checkpoint equivalents) so cross-service signatures
// verify. It is intentionally duplicated rather than imported — this repo's
// convention is no cross-package Go imports between packages/KNIRV* modules,
// only HTTP/gRPC.
package checkpoint

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/ethereum/go-ethereum/crypto"
)

const SchemaVersion = "knirv.checkpoint.v1"

// AuthorSig is a single author's signature over a checkpoint or registration
// digest, wire-compatible with KNIRVORACLE's types.AuthorSig.
type AuthorSig struct {
	Address   string `json:"address"`
	PubKeyHex string `json:"pubkey_hex"`
	Signature []byte `json:"signature"`
	Envelope  []byte `json:"envelope"`
}

// RegisteredAuthor is one entry in a chain's registered author set.
type RegisteredAuthor struct {
	Address string `json:"address"`
	PubKey  []byte `json:"pubkey"`
	Weight  uint64 `json:"weight"`
}

// ChainRegistration is the payload POSTed to /oracle/v3/registry/register.
// Field set mirrors KNIRVORACLE's types.ChainRegistration exactly (including
// fields this client never sets) — the server recomputes the registration
// digest from the deserialized struct, so an omitted field here becomes a
// zero value there, and registrationBody below must construct the identical
// zero-valued body the server will see, or the signature won't verify.
type ChainRegistration struct {
	ChainID              string             `json:"chain_id"`
	Authors              []RegisteredAuthor `json:"authors"`
	QuorumNumer          uint64             `json:"quorum_numer"`
	QuorumDenom          uint64             `json:"quorum_denom"`
	LastHeight           uint64             `json:"last_height"`
	LastCheckHash        [32]byte           `json:"last_check_hash"`
	Bond                 uint64             `json:"bond"`
	BondOwner            string             `json:"bond_owner,omitempty"`
	BondRemaining        uint64             `json:"bond_remaining,omitempty"`
	BondSlashed          uint64             `json:"bond_slashed,omitempty"`
	ProofWindow          uint64             `json:"proof_window"`
	VerificationKey      []byte             `json:"verification_key,omitempty"`
	PreferredProofSystem string             `json:"preferred_proof_system,omitempty"`
	RotationSigs         []AuthorSig        `json:"rotation_sigs,omitempty"`
}

// Checkpoint is the payload POSTed to /oracle/v3/checkpoints.
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
// authors sign. Must stay byte-identical to KNIRVORACLE's
// types.Checkpoint.Digest.
func (c *Checkpoint) Digest() [32]byte {
	parts := [][]byte{
		[]byte(c.SchemaVersion),
		{0x00},
		[]byte(c.ChainID),
		{0x00},
		u64Bytes(c.StartHeight),
		u64Bytes(c.EndHeight),
		c.Root[:],
		c.PrevCheckHash[:],
		[]byte(c.Proposer),
		{0x00},
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
	for i := range 8 {
		b[i] = byte(v >> (56 - i*8))
	}
	return b
}

// SignedMessage is the canonical string that gets signed: the hex encoding
// of the digest.
func (c *Checkpoint) SignedMessage() string { return fmt.Sprintf("%x", c.Digest()) }

// signMessage signs msg with keccak256 + secp256k1, returning the 64-byte
// raw signature (r||s) — same scheme as KNIRVORACLE/KNIRVCHAIN.
func signMessage(key *ecdsa.PrivateKey, msg, purpose, chainID string) (knirvsigning.SignedMessage, error) {
	now := time.Now().Unix()
	return knirvsigning.SignMessage(crypto.FromECDSA(key), knirvsigning.MessageEnvelope{
		Domain: "knirv.chain", Purpose: purpose, ChainID: chainID, Nonce: msg,
		IssuedAtUnix: now, ExpiresAtUnix: math.MaxInt64, Payload: []byte(msg),
	})
}

// SignCheckpoint signs the checkpoint with key and appends the AuthorSig.
func SignCheckpoint(cp *Checkpoint, key *ecdsa.PrivateKey) error {
	signed, err := signMessage(key, cp.SignedMessage(), "chain-checkpoint", cp.ChainID)
	if err != nil {
		return err
	}
	cp.Signatures = append(cp.Signatures, AuthorSig{
		Address: signed.Address, PubKeyHex: hex.EncodeToString(signed.PublicKey),
		Signature: signed.Signature, Envelope: signed.Envelope,
	})
	return nil
}

func oracleAddress(pub *ecdsa.PublicKey) string {
	address, err := knirvsigning.Address(crypto.CompressPubkey(pub), knirvsigning.DefaultAddressPrefix)
	if err != nil {
		panic(err)
	}
	return address
}

// registrationBody is the signature-free, deterministic registration payload
// authors sign — field set, JSON tags, and ordering must match KNIRVORACLE's
// types.RegistrationBody exactly, since the server recomputes this from the
// deserialized ChainRegistration it received, not from our request bytes.
func registrationBody(c *ChainRegistration) ([]byte, error) {
	type body struct {
		ChainID              string             `json:"chain_id"`
		Authors              []RegisteredAuthor `json:"authors"`
		QuorumNumer          uint64             `json:"quorum_numer"`
		QuorumDenom          uint64             `json:"quorum_denom"`
		LastHeight           uint64             `json:"last_height"`
		LastCheckHash        [32]byte           `json:"last_check_hash"`
		Bond                 uint64             `json:"bond"`
		BondOwner            string             `json:"bond_owner,omitempty"`
		BondRemaining        uint64             `json:"bond_remaining,omitempty"`
		BondSlashed          uint64             `json:"bond_slashed,omitempty"`
		ProofWindow          uint64             `json:"proof_window"`
		VerificationKey      []byte             `json:"verification_key,omitempty"`
		PreferredProofSystem string             `json:"preferred_proof_system,omitempty"`
	}
	return json.Marshal(body{
		ChainID: c.ChainID, Authors: c.Authors, QuorumNumer: c.QuorumNumer,
		QuorumDenom: c.QuorumDenom, LastHeight: c.LastHeight, LastCheckHash: c.LastCheckHash,
		Bond: c.Bond, BondOwner: c.BondOwner, BondRemaining: c.BondRemaining,
		BondSlashed: c.BondSlashed, ProofWindow: c.ProofWindow,
		VerificationKey: c.VerificationKey, PreferredProofSystem: c.PreferredProofSystem,
	})
}

// SignRegistration authorizes initial enrollment, appending to RotationSigs.
// The registration digest is sha256 (not keccak256) — must match
// KNIRVORACLE's types.RegistrationDigest exactly.
func SignRegistration(reg *ChainRegistration, key *ecdsa.PrivateKey) error {
	body, err := registrationBody(reg)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(body)
	signed, err := signMessage(key, fmt.Sprintf("%x", digest), "chain-registration", reg.ChainID)
	if err != nil {
		return err
	}
	reg.RotationSigs = append(reg.RotationSigs, AuthorSig{
		Address: signed.Address, PubKeyHex: hex.EncodeToString(signed.PublicKey),
		Signature: signed.Signature, Envelope: signed.Envelope,
	})
	return nil
}
