package types

import (
	"crypto/sha256"
	"encoding/json"
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
	ChainID       string             `json:"chain_id"`
	Authors       []RegisteredAuthor `json:"authors"`
	QuorumNumer   uint64             `json:"quorum_numer"`
	QuorumDenom   uint64             `json:"quorum_denom"`
	LastHeight    uint64             `json:"last_height"`
	LastCheckHash [32]byte           `json:"last_check_hash"`
	Bond          uint64             `json:"bond"`
	BondOwner     string             `json:"bond_owner,omitempty"`
	BondRemaining uint64             `json:"bond_remaining,omitempty"`
	BondSlashed   uint64             `json:"bond_slashed,omitempty"`
	// ProofWindow is the dispute/finality window in seconds (wall-clock time),
	// not a block-height count: the Oracle's own block-commit cadence is a
	// heartbeat, not a security-relevant clock (see registry.DefaultProofWindow
	// and CheckpointRecord.FinalBy), so a chain's proof window must not depend
	// on how often the Oracle happens to be ticking.
	ProofWindow uint64 `json:"proof_window"`
	// VerificationKey carries the registered SNARK verification key for this
	// chain (merkle-math.md Phase 5). Empty until a verifier chain enrolls one.
	VerificationKey []byte `json:"verification_key,omitempty"`
	// PreferredProofSystem is the proof system the Oracle expects for this
	// chain's finality records ("hashchain-v0" | "groth16" | "plonk").
	PreferredProofSystem string `json:"preferred_proof_system,omitempty"`
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

// Checkpoint is the wire type KNIRVSERVER's embedded Validation Chain posts
// (pkg/embedded/validationchain/checkpoint in the KNIRV_NETWORK repo) — the
// merkle source of record. KNIRVCHAIN no longer submits checkpoints; it
// stays sovereign, minting NFT bundles only. This type mirrors the poster's
// wire type and adds oracle-side fields downstream in CheckpointRecord.
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
	Checkpoint  Checkpoint       `json:"checkpoint"`
	MMRPosition uint64           `json:"mmr_position"`
	LeafHash    [32]byte         `json:"leaf_hash"`
	Status      CheckpointStatus `json:"status"`
	ReceivedAt  time.Time        `json:"received_at"`
	// FinalBy is the wall-clock deadline (ReceivedAt + the chain's ProofWindow)
	// by which a finality record must land or the checkpoint is swept and
	// rejected as a window miss (see Oracle.sweepExpired). This used to be an
	// Oracle block-height deadline, which tied dispute-window correctness to
	// how often the Oracle happened to tick — including during idle periods
	// where nothing should need to depend on that at all.
	FinalBy       time.Time `json:"final_by"`
	FinalityLeaf  *uint64   `json:"finality_leaf,omitempty"`
	RejectionLeaf *uint64   `json:"rejection_leaf,omitempty"`
	// PendingAttestations accumulates validation-chain sign-offs (Phase 4)
	// until quorum is reached, at which point the record is finalized.
	PendingAttestations    []VerifierAttestation `json:"pending_attestations,omitempty"`
	PendingTransitionProof []byte                `json:"pending_transition_proof,omitempty"`
	PendingProofSystem     string                `json:"pending_proof_system,omitempty"`
	// Source records how this leaf entered the MMR. Empty = direct submission
	// by a registered checkpoint chain (Validation Chain). "rollup:<id>" =
	// projected from a RollupRecord (Phase 3 bridge, e.g. Transaction Chain).
	// Non-empty sources are audit-only and cannot enter finality.
	Source string `json:"source,omitempty"`
}

// FinalityRecord is the phase-2 leaf — appended independently, never mutating
// the phase-1 leaf.
type FinalityRecord struct {
	SchemaVersion   string                `json:"schema_version"` // "knirv.finality.v1"
	CheckpointLeaf  uint64                `json:"checkpoint_leaf"`
	CheckpointHash  [32]byte              `json:"checkpoint_hash"`
	TransitionProof []byte                `json:"transition_proof"`
	ProofSystem     string                `json:"proof_system"`
	Attestations    []VerifierAttestation `json:"attestations"`
}

// VerifierAttestation is a validation-chain node's sign-off on a checkpoint.
type VerifierAttestation struct {
	VerifierID string `json:"verifier_id"`
	LeafIndex  uint64 `json:"leaf_index"`
	Approved   bool   `json:"approved"`
	Signature  []byte `json:"signature"` // ed25519 over (LeafIndex|CheckpointHash|Approved)
}

// AttestationMessage is the canonical ed25519 message shared by Oracle and
// verifier chains: big-endian leaf index || checkpoint hash || approved byte.
func AttestationMessage(leafIndex uint64, checkpointHash [32]byte, approved bool) []byte {
	message := make([]byte, 8+32+1)
	for i := 0; i < 8; i++ {
		message[i] = byte(leafIndex >> (56 - i*8))
	}
	copy(message[8:40], checkpointHash[:])
	if approved {
		message[40] = 1
	}
	return message
}

// TransitionProof is the phase-2 validity proof accompanying a FinalityRecord
// (merkle-math.md §3.2c / Phase 5). For hashchain-v0 the verifier re-executes the
// deterministic replay (pre-root + batch → post-root); the SNARK systems carry a
// groth16/plonk proof plus public inputs. The Oracle runs a cheap binding check
// itself (PostRoot == checkpoint.Root, PreRoot == checkpoint.PrevCheckHash) and
// delegates full re-execution / SNARK verification to the verifier chains.
type TransitionProof struct {
	SchemaVersion string   `json:"schema_version"` // "knirv.transition-proof.v1"
	ChainID       string   `json:"chain_id"`
	StartHeight   uint64   `json:"start_height"`
	EndHeight     uint64   `json:"end_height"`
	PreRoot       [32]byte `json:"pre_root"`     // AccumRoot at StartHeight-1
	PostRoot      [32]byte `json:"post_root"`    // == Checkpoint.Root
	ProofSystem   string   `json:"proof_system"` // "hashchain-v0" | "groth16" | "plonk"
	Proof         []byte   `json:"proof"`
	PublicInputs  []byte   `json:"public_inputs"` // canonical (PreRoot, PostRoot, txBatchHash)
}

// CheckpointLeaf is the canonical phase-1 MMR payload. Kind is deliberately
// part of the serialized body so all three MMR record classes are
// self-describing. Leaf-kind constants live in the mmr package only.
type CheckpointLeaf struct {
	Kind          byte     `json:"kind"`
	SchemaVersion string   `json:"schema_version"`
	ChainID       string   `json:"chain_id"`
	StartHeight   uint64   `json:"start_height"`
	EndHeight     uint64   `json:"end_height"`
	Root          [32]byte `json:"root"`
	PrevCheckHash [32]byte `json:"prev_check_hash"`
	Proposer      string   `json:"proposer"`
}

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
	Kind             byte              `json:"kind"`
	CheckpointLeaf   uint64            `json:"checkpoint_leaf"`
	Reason           string            `json:"reason"`
	SlashingEvidence *SlashingEvidence `json:"slashing_evidence,omitempty"`
}

// SlashingEvidence is persisted in the rejection leaf and queued into the
// Oracle consensus block's EvidenceData. Amount is denominated in the token's
// smallest NRN unit.
type SlashingEvidence struct {
	SchemaVersion  string    `json:"schema_version"`
	ChainID        string    `json:"chain_id"`
	BondOwner      string    `json:"bond_owner"`
	CheckpointLeaf uint64    `json:"checkpoint_leaf"`
	Reason         string    `json:"reason"`
	Amount         uint64    `json:"amount"`
	OracleHeight   uint64    `json:"oracle_height"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// RotationSigs carries the author-set signatures authorizing a registry rotation.
// It reuses AuthorSig so the quorum machinery is shared with checkpoints.
type RotationSigs []AuthorSig

// CanonicalBytes returns the stable serialization of a checkpoint's body used as
// the MMR leaf payload. It omits signatures and JSON field ordering variance by
// constructing an explicit, ordered structure.
func (c *Checkpoint) CanonicalBytes() CheckpointLeaf {
	return CheckpointLeaf{
		Kind: 0x01, SchemaVersion: c.SchemaVersion, ChainID: c.ChainID,
		StartHeight: c.StartHeight, EndHeight: c.EndHeight, Root: c.Root,
		PrevCheckHash: c.PrevCheckHash, Proposer: c.Proposer,
	}
}

// RegistrationBody returns the signature-free, deterministic registration
// payload used for both initial enrollment and author-set rotations.
func RegistrationBody(c *ChainRegistration) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("registration required")
	}
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
		QuorumDenom: c.QuorumDenom, LastHeight: c.LastHeight,
		LastCheckHash: c.LastCheckHash, Bond: c.Bond, BondOwner: c.BondOwner,
		BondRemaining: c.BondRemaining, BondSlashed: c.BondSlashed,
		ProofWindow: c.ProofWindow, VerificationKey: c.VerificationKey,
		PreferredProofSystem: c.PreferredProofSystem,
	})
}

func ChainRegistrationDigest(c *ChainRegistration) ([32]byte, error) {
	body, err := RegistrationBody(c)
	if err != nil {
		return [32]byte{}, err
	}
	return RegistrationDigest(body), nil
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
