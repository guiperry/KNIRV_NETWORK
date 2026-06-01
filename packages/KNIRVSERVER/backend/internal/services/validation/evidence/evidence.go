// Package evidence provides canonical types for evidence packs and their components.
// These types are promoted from the fintech plugin package to a platform-level
// package so they can be shared across compliance validators.
package evidence

import (
	"time"

	"backend_server/internal/services/plugins/fintech"
)

// Re-export canonical types from the fintech package so downstream packages
// can import from this platform-level location without creating circular deps.

// EvidencePackType is an alias for fintech.EvidencePackType.
type EvidencePackType = fintech.EvidencePackType

// EvidenceStatus is an alias for fintech.EvidenceStatus.
type EvidenceStatus = fintech.EvidenceStatus

// EvidencePack is an alias for fintech.EvidencePack.
type EvidencePack = fintech.EvidencePack

// EBPFTraceEvidence is an alias for fintech.EBPFTraceEvidence.
type EBPFTraceEvidence = fintech.EBPFTraceEvidence

// SyscallEvent is an alias for fintech.SyscallEvent.
type SyscallEvent = fintech.SyscallEvent

// NetworkEvent is an alias for fintech.NetworkEvent.
type NetworkEvent = fintech.NetworkEvent

// FileAccessEvent is an alias for fintech.FileAccessEvent.
type FileAccessEvent = fintech.FileAccessEvent

// ProcessTree is an alias for fintech.ProcessTree.
type ProcessTree = fintech.ProcessTree

// ProcessNode is an alias for fintech.ProcessNode.
type ProcessNode = fintech.ProcessNode

// NRVTraceEvidence is an alias for fintech.NRVTraceEvidence.
type NRVTraceEvidence = fintech.NRVTraceEvidence

// ValidationEvidence is an alias for fintech.ValidationEvidence.
type ValidationEvidence = fintech.ValidationEvidence

// ComplianceEvidence is an alias for fintech.ComplianceEvidence.
type ComplianceEvidence = fintech.ComplianceEvidence

// PQCSignature is an alias for fintech.PQCSignature.
type PQCSignature = fintech.PQCSignature

// Store is an alias for fintech.EvidenceStore.
type Store = fintech.EvidenceStore

// Filter is an alias for fintech.EvidenceFilter.
type Filter = fintech.EvidenceFilter

// Bundle is an alias for fintech.EvidencePackBundle.
type Bundle = fintech.EvidencePackBundle

// Constants re-exported for convenience.
const (
	EvidenceTypeValidation = fintech.EvidenceTypeValidation
	EvidenceTypeCompliance = fintech.EvidenceTypeCompliance
	EvidenceTypeAudit      = fintech.EvidenceTypeAudit
	EvidenceTypeReplay     = fintech.EvidenceTypeReplay

	EvidenceStatusPending    = fintech.EvidenceStatusPending
	EvidenceStatusCollecting = fintech.EvidenceStatusCollecting
	EvidenceStatusComplete   = fintech.EvidenceStatusComplete
	EvidenceStatusFailed     = fintech.EvidenceStatusFailed
)

// NewEvidencePack creates a new evidence pack.
// This is a convenience wrapper so callers can import only the evidence package.
func NewEvidencePack(packType EvidencePackType, agentID, validationID string) *EvidencePack {
	return fintech.NewEvidencePack(packType, agentID, validationID)
}

// Builder provides a fluent API for constructing evidence packs.
type Builder struct {
	pack *EvidencePack
}

// NewBuilder creates a new Builder for the given agent and validation IDs.
func NewBuilder(packType EvidencePackType, agentID, validationID string) *Builder {
	return &Builder{pack: NewEvidencePack(packType, agentID, validationID)}
}

// WithAgentName sets the agent name.
func (b *Builder) WithAgentName(name string) *Builder {
	b.pack.AgentName = name
	return b
}

// AddComplianceCheck appends a compliance check to the evidence pack.
func (b *Builder) AddComplianceCheck(ce ComplianceEvidence) *Builder {
	b.pack.AddComplianceCheck(ce)
	return b
}

// MarkComplete finalises the evidence pack as completed.
func (b *Builder) MarkComplete() *Builder {
	b.pack.MarkComplete()
	return b
}

// MarkFailed finalises the evidence pack as failed with a reason.
func (b *Builder) MarkFailed(reason string) *Builder {
	b.pack.MarkFailed(reason)
	return b
}

// Build returns the constructed EvidencePack.
func (b *Builder) Build() *EvidencePack {
	return b.pack
}

// ReasoningStep is a platform alias for fintech.ReasoningStep where available.
// Because fintech doesn't export ReasoningStep separately we define a thin
// struct here so callers don't need to import the fintech package.
type ReasoningStep struct {
	StepNumber  int       `json:"step_number"`
	Timestamp   time.Time `json:"timestamp"`
	StepType    string    `json:"step_type"`
	Intent      string    `json:"intent"`
	Observation string    `json:"observation"`
	Inference   string    `json:"inference"`
	Confidence  float64   `json:"confidence"`
}
