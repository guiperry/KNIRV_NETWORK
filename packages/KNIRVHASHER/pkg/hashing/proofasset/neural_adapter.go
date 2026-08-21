package proofasset

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

// ProofCandidate is a theorem/proof pair proposed by the neural/search layer.
// It is advisory only and must never alter checker acceptance semantics.
type ProofCandidate struct {
	SchemaVersion       uint32
	TheoremSource       []byte
	ProofSource         []byte
	Imports             []ArtifactRef
	Diagnostics         []DiagnosticTaxonomy
	Confidence          float32
	CandidateProvenance CandidateProvenance
}

// CandidateMetrics tracks proposal success rates and costs.
type CandidateMetrics struct {
	TotalProposed   uint64
	Accepted        uint64
	Rejected        uint64
	RepairAttempts  uint64
	AvgRepairRounds float64
	mu              sync.RWMutex
}

func (m *CandidateMetrics) RecordProposal(accepted bool, repairRounds int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalProposed++
	if accepted {
		m.Accepted++
	} else {
		m.Rejected++
	}
	m.RepairAttempts += uint64(repairRounds)
	if m.TotalProposed > 0 {
		m.AvgRepairRounds = float64(m.RepairAttempts) / float64(m.TotalProposed)
	}
}

// ProofAssetFromCandidate converts a ProofCandidate to a ProofAsset for
// submission to the formal checker. The candidate provenance is preserved.
func ProofAssetFromCandidate(c *ProofCandidate) (*ProofAsset, error) {
	if c == nil {
		return nil, fmt.Errorf("proof candidate is nil")
	}
	if len(c.TheoremSource) == 0 {
		return nil, fmt.Errorf("theorem source is required")
	}
	if len(c.ProofSource) == 0 {
		return nil, fmt.Errorf("proof source is required")
	}

	depDigest := fmt.Sprintf("lean-%d-imports", len(c.Imports))
	toolchainDigest := "lean-dev"

	asset := &ProofAsset{
		SchemaVersion:        1,
		ProofSystem:          ProofSystemLean,
		ToolchainDigest:      toolchainDigest,
		DependencyLockDigest: depDigest,
		TheoremSource:        c.TheoremSource,
		ProofSource:          c.ProofSource,
		Imports:              c.Imports,
		CandidateProvenance:  c.CandidateProvenance,
	}

	if err := ValidateProofAsset(asset, importNames(c.Imports)); err != nil {
		return nil, fmt.Errorf("candidate validation failed: %w", err)
	}

	return asset, nil
}

func importNames(refs []ArtifactRef) []string {
	names := make([]string, len(refs))
	for i, r := range refs {
		names[i] = r.Name
	}
	return names
}

// CandidateDeduplicator prevents duplicate proof proposals from entering the
// repair queue. It is keyed by theorem content address.
type CandidateDeduplicator struct {
	seen map[string]struct{}
	mu   sync.RWMutex
}

func NewCandidateDeduplicator() *CandidateDeduplicator {
	return &CandidateDeduplicator{seen: make(map[string]struct{})}
}

func (d *CandidateDeduplicator) Seen(theoremSource []byte) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	id := fmt.Sprintf("%x", sha256.Sum256(theoremSource))
	_, ok := d.seen[id]
	return id, ok
}

func (d *CandidateDeduplicator) Mark(theoremSource []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := fmt.Sprintf("%x", sha256.Sum256(theoremSource))
	d.seen[id] = struct{}{}
}
