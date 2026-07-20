package types

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// AuthorSig is a single Network Author's signature over a checkpoint digest.
// PubKeyHex carries the signer's compressed secp256k1 public key (hex) so the
// quorum can be verified without an out-of-band key lookup.
type AuthorSig struct {
	Address   string `json:"address"`
	PubKeyHex string `json:"pubkey_hex"`
	Signature []byte `json:"signature"`
}

// RegisteredAuthor is one entry in a chain's registered author set.
type RegisteredAuthor struct {
	Address string `json:"address"`
	PubKey  []byte `json:"pubkey"`
	Weight  uint64 `json:"weight"`
}

// ChainRegistration is the Oracle's trust record for a single foreign chain.
type ChainRegistration struct {
	ChainID       string            `json:"chain_id"`
	Authors       []RegisteredAuthor `json:"authors"`
	QuorumNumer   uint64            `json:"quorum_numer"`
	QuorumDenom   uint64            `json:"quorum_denom"`
	LastHeight    uint64            `json:"last_height"`
	LastCheckHash [32]byte          `json:"last_check_hash"`
	Bond          uint64            `json:"bond"`
	ProofWindow   uint64            `json:"proof_window"`
	// RotationSigs authorizes a registry update; carried by POST /registry/rotate.
	RotationSigs []AuthorSig `json:"rotation_sigs,omitempty"`
}

// CheckpointStatus is the lifecycle state of a checkpoint record.
type CheckpointStatus string

const (
	CheckpointProvisional CheckpointStatus = "provisional"
	CheckpointFinal       CheckpointStatus = "final"
	CheckpointRejected    CheckpointStatus = "rejected"
)

// Checkpoint is the wire type KNIRVCHAIN posts; it mirrors KNIRVCHAIN's type and
// adds oracle-side fields downstream in CheckpointRecord.
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

// CheckpointRecord is the Oracle-side indexed view (replaces RollupRecord's
// role for the new pipeline). Status transitions live only here, never in the
// MMR log (Design Invariant #1).
type CheckpointRecord struct {
	Checkpoint    Checkpoint       `json:"checkpoint"`
	MMRPosition   uint64           `json:"mmr_position"`
	LeafHash      [32]byte         `json:"leaf_hash"`
	Status        CheckpointStatus `json:"status"`
	ReceivedAt    time.Time        `json:"received_at"`
	FinalByHeight uint64           `json:"final_by_height"`
	FinalityLeaf  *uint64          `json:"finality_leaf,omitempty"`
	RejectionLeaf *uint64          `json:"rejection_leaf,omitempty"`
	// PendingAttestations accumulates validation-chain sign-offs (Phase 4)
	// until quorum is reached, at which point the record is finalized.
	PendingAttestations []VerifierAttestation `json:"pending_attestations,omitempty"`
	// Source records how this leaf entered the MMR. Empty = direct KNIRVCHAIN
	// submission. "rollup:<id>" = projected from a legacy RollupRecord (Phase 3
	// bridge). It is informational only; it never changes admission semantics.
	Source string `json:"source,omitempty"`
}

// FinalityRecord is the phase-2 leaf — appended independently, never mutating
// the phase-1 leaf.
type FinalityRecord struct {
	SchemaVersion   string               `json:"schema_version"` // "knirv.finality.v1"
	CheckpointLeaf  uint64               `json:"checkpoint_leaf"`
	CheckpointHash  [32]byte             `json:"checkpoint_hash"`
	TransitionProof []byte               `json:"transition_proof"`
	ProofSystem     string               `json:"proof_system"`
	Attestations    []VerifierAttestation `json:"attestations"`
}

// VerifierAttestation is a validation-chain node's sign-off on a checkpoint.
type VerifierAttestation struct {
	VerifierID string `json:"verifier_id"`
	LeafIndex  uint64 `json:"leaf_index"`
	Approved   bool   `json:"approved"`
	Signature  []byte `json:"signature"` // ed25519 over (LeafIndex|CheckpointHash|Approved)
}

// LeafKind tags every MMR leaf so the append-only log is self-describing
// (merkle-math.md §3.1).
type LeafKind byte

const (
	LeafCheckpoint LeafKind = 0x01 // phase-1 provisional checkpoint / rollup projection
	LeafFinality   LeafKind = 0x02 // phase-2 finality record
	LeafRejection  LeafKind = 0x03 // window-miss / proof-fail tombstone
)

// FinalityLeaf is the canonical MMR payload for a LeafFinality leaf. The full
// transition proof is kept in the indexed FinalityRecord; the leaf carries only
// the binding references + attestation quorum so the log stays compact and
// self-describing.
type FinalityLeaf struct {
	Kind           byte                  `json:"kind"`
	CheckpointLeaf uint64                `json:"checkpoint_leaf"`
	CheckpointHash string                `json:"checkpoint_hash"`
	ProofSystem    string                `json:"proof_system"`
	Attestations   []VerifierAttestation `json:"attestations"`
}

// RejectionLeaf is the canonical MMR payload for a LeafRejection tombstone.
// Appended when a provisional record misses its proof window or fails
// verification; never mutates the original checkpoint leaf.
type RejectionLeaf struct {
	Kind          byte   `json:"kind"`
	CheckpointLeaf uint64 `json:"checkpoint_leaf"`
	Reason        string `json:"reason"`
}

// RotationSigs carries the author-set signatures authorizing a registry rotation.
// It reuses AuthorSig so the quorum machinery is shared with checkpoints.
type RotationSigs []AuthorSig

// CanonicalBytes returns the stable serialization of a checkpoint's body used as
// the MMR leaf payload. It omits signatures and JSON field ordering variance by
// constructing an explicit, ordered structure.
func (c *Checkpoint) CanonicalBytes() map[string]interface{} {
	return map[string]interface{}{
		"schema_version": c.SchemaVersion,
		"chain_id":       c.ChainID,
		"start_height":   c.StartHeight,
		"end_height":     c.EndHeight,
		"root":           fmt.Sprintf("%x", c.Root),
		"prev_check_hash": fmt.Sprintf("%x", c.PrevCheckHash),
		"proposer":       c.Proposer,
	}
}

// RegistrationDigest is the canonical 32-byte hash over a registration body that
// authors sign to authorize a rotation.
func RegistrationDigest(body []byte) [32]byte {
	return sha256.Sum256(body)
}

// VerifyRegistrationSig verifies one AuthorSig over a registration digest using
// the same secp256k1/keccak scheme as checkpoints.
func VerifyRegistrationSig(digest [32]byte, sig AuthorSig) bool {
	return VerifyAuthorSigDigest(digest, sig)
}
