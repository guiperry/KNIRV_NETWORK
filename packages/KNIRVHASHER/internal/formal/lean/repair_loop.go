package lean

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"knirvhasher/pkg/hashing/proofasset"
)

// RepairLoop applies bounded neural-generated proof repairs to rejected proof
// assets. It never alters checker acceptance semantics: every repaired asset
// must still pass ValidateProofAsset and the formal checker.
type RepairLoop struct {
	worker         *Worker
	maxRounds      int
	repairInterval time.Duration
	metrics        *proofasset.CandidateMetrics
	dedup          *proofasset.CandidateDeduplicator
	mu             sync.Mutex
}

// NewRepairLoop creates a RepairLoop with bounded repair attempts.
func NewRepairLoop(worker *Worker, maxRounds int, interval time.Duration) *RepairLoop {
	if maxRounds <= 0 {
		maxRounds = 3
	}
	if interval <= 0 {
		interval = time.Second
	}
	return &RepairLoop{
		worker:         worker,
		maxRounds:      maxRounds,
		repairInterval: interval,
		metrics:        &proofasset.CandidateMetrics{},
		dedup:          proofasset.NewCandidateDeduplicator(),
	}
}

// AttemptRepair submits a proof asset to the checker, then applies up to
// maxRounds of neural-generated repairs if rejected. It returns the final
// receipt and the number of repair rounds performed.
func (r *RepairLoop) AttemptRepair(asset *proofasset.ProofAsset, repairFunc func(*proofasset.ProofAsset, []proofasset.DiagnosticTaxonomy) *proofasset.ProofAsset) (*proofasset.VerificationReceipt, int, error) {
	if asset == nil {
		return nil, 0, fmt.Errorf("proof asset is nil")
	}
	if repairFunc == nil {
		return nil, 0, fmt.Errorf("repair function is nil")
	}

	receipt, err := r.worker.SubmitProof(asset)
	if err != nil {
		return nil, 0, fmt.Errorf("initial submission failed: %w", err)
	}

	if receipt == nil || receipt.Receipt == nil {
		return nil, 0, fmt.Errorf("worker returned nil receipt")
	}

	if receipt.Receipt.Status == proofasset.StatusFormallyVerified {
		r.metrics.RecordProposal(true, 0)
		return receipt.Receipt, 0, nil
	}

	taxonomy := classifyDiagnostics(receipt.Diagnostic)
	current := asset
	rounds := 0

	for rounds < r.maxRounds {
		if _, seen := r.dedup.Seen(current.TheoremSource); seen {
			break
		}
		r.dedup.Mark(current.TheoremSource)

		repaired := repairFunc(current, taxonomy)
		if repaired == nil {
			break
		}

		repairedID, _ := proofasset.ComputeProofAssetID(repaired)
		originalID, _ := proofasset.ComputeProofAssetID(current)
		if repairedID == originalID {
			break
		}

		current = repaired
		rounds++

		receipt, err := r.worker.SubmitProof(current)
		if err != nil {
			continue
		}
		if receipt == nil || receipt.Receipt == nil {
			continue
		}

		if receipt.Receipt.Status == proofasset.StatusFormallyVerified {
			r.metrics.RecordProposal(true, rounds)
			return receipt.Receipt, rounds, nil
		}

		taxonomy = classifyDiagnostics(receipt.Diagnostic)
		time.Sleep(r.repairInterval)
	}

	r.metrics.RecordProposal(false, rounds)
	if receipt.Receipt != nil {
		return receipt.Receipt, rounds, nil
	}
	return nil, rounds, fmt.Errorf("repair loop exhausted without receipt")
}

func classifyDiagnostics(diagnostic string) []proofasset.DiagnosticTaxonomy {
	var tax []proofasset.DiagnosticTaxonomy
	if diagnostic == "" {
		return tax
	}
	lower := []byte(strings.ToLower(diagnostic))
	if bytes.Contains(lower, []byte("unknown")) {
		tax = append(tax, proofasset.DiagnosticUnknownIdentifier)
	}
	if bytes.Contains(lower, []byte("type")) {
		tax = append(tax, proofasset.DiagnosticTypeMismatch)
	}
	if bytes.Contains(lower, []byte("unsolved")) || bytes.Contains(lower, []byte("goal")) {
		tax = append(tax, proofasset.DiagnosticUnsolvedGoal)
	}
	if bytes.Contains(lower, []byte("import")) || bytes.Contains(lower, []byte("denied")) {
		tax = append(tax, proofasset.DiagnosticImportPolicyDenied)
	}
	if bytes.Contains(lower, []byte("memory")) || bytes.Contains(lower, []byte("time")) || bytes.Contains(lower, []byte("limit")) {
		tax = append(tax, proofasset.DiagnosticResourceLimit)
	}
	if bytes.Contains(lower, []byte("error")) || bytes.Contains(lower, []byte("fail")) {
		tax = append(tax, proofasset.DiagnosticCheckerFailure)
	}
	if len(tax) == 0 {
		tax = append(tax, proofasset.DiagnosticParseError)
	}
	return tax
}

// Metrics returns the candidate proposal metrics.
func (r *RepairLoop) Metrics() *proofasset.CandidateMetrics {
	return r.metrics
}
