package drq

import (
	"errors"
	"time"
)

// This file holds the data contracts for the external services DRQ's call
// graph depends on (see interfaces.go). They are declared here rather than
// imported from the platform packages on purpose: drq is a leaf package inside
// KNIRVGRAPH, and importing the backend server's validation/certificate
// packages from here would invert the dependency (and drag a Go module cycle in
// with it). The production clients translate the platform's payloads into these
// shapes; test doubles construct them directly.

// ValidationTask is the handle the platform validation service returns when a
// cluster is submitted for validation.
type ValidationTask struct {
	// ID is the platform-side task identifier, used later to fetch the proof.
	ID string
	// ClusterID is the DRQ cluster this task validates.
	ClusterID string
	// Status is the platform's task status ("pending", "in_progress",
	// "completed", "failed").
	Status string
	// Degraded is set by the platform when the validator fell back to a
	// default response instead of running a real model. A degraded task must
	// never be treated as validated.
	Degraded  bool
	CreatedAt time.Time
}

// ValidationProof is the platform's validation certificate for a task.
//
// Valid is the gate DRQ trusts: DRQ does not re-implement validation policy
// locally, it reads the platform's verdict. A proof with Valid=false, or with
// an empty Proof payload, must block minting.
type ValidationProof struct {
	TaskID    string
	ClusterID string
	Valid     bool
	// Proof is the platform's cryptographic proof payload. Empty means the
	// platform did not actually produce a proof.
	Proof []byte
	// Certificate is the platform's CertificateOfCorrectness payload, when the
	// platform emits one alongside the proof.
	Certificate []byte
	IssuedAt    time.Time
	Degraded    bool
}

// RoyaltyRequest asks KNIRVCHAIN's royalty engine to distribute an invocation
// royalty for a skill held under perpetual ownership rights.
type RoyaltyRequest struct {
	SkillID    string
	OwnerAgent string
	// Fee is the invocation fee that triggered this royalty, in whole NRN.
	Fee float64
	// AmountBaseUnits is the NRN amount to distribute, in the token's
	// 18-decimal base units, as a decimal string.
	AmountBaseUnits string
	// RefID identifies the triggering invocation so a retried royalty call is
	// deduplicated rather than paid twice.
	RefID string
}

// RoyaltyReceipt is the chain's record of a royalty distribution.
type RoyaltyReceipt struct {
	SkillID string
	// TxHash is the chain transaction that paid the royalty.
	TxHash string
	// AmountBaseUnits is what was actually paid, in base units.
	AmountBaseUnits string
	PaidAt          time.Time
}

// ResolvedErrorIDs returns the sorted, de-duplicated ids of the error nodes a
// cluster resolves. Exported for callers outside this package (the app's DRQ
// loop, the validation submission payload) that need to describe a cluster.
func ResolvedErrorIDs(cluster *ErrorCluster) []string {
	if cluster == nil {
		return nil
	}
	return extractErrorIDs(cluster.Errors)
}

// Sentinel errors. These exist so a caller can distinguish "the backend is not
// there yet" from "the backend said no", and — critically — so that an absent
// backend surfaces as a real error instead of a silent no-op.
var (
	// ErrRoyaltyBackendUnavailable is returned when no KNIRVCHAIN royalty
	// backend is reachable or configured.
	ErrRoyaltyBackendUnavailable = errors.New("drq: royalty backend unavailable")
	// ErrLoRATrainingBackendNotImplemented is returned when LoRA training is
	// requested but no real training backend is configured. The trainer never
	// reports a manufactured success in this case.
	ErrLoRATrainingBackendNotImplemented = errors.New("drq: LoRA training backend is not implemented")
	// ErrValidationProofMissing is returned when a validation task reports
	// completion without a usable proof.
	ErrValidationProofMissing = errors.New("drq: validation proof missing or not valid")
)
