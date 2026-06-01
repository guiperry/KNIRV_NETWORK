// Package fidelity provides canonical types for AI agent fidelity scoring.
// These types are promoted from the fintech plugin package to a platform-level
// package so they can be shared across different compliance validators.
package fidelity

import (
	"backend_server/internal/services/plugins/fintech"
	"backend_server/internal/services/plugins/fintech/ontology"
)

// Re-export canonical types from the fintech package.

// Scorer is an alias for fintech.FidelityScorer.
type Scorer = fintech.FidelityScorer

// ScoreResult is an alias for fintech.FidelityScoreResult.
type ScoreResult = fintech.FidelityScoreResult

// ScorerConfig is an alias for fintech.FidelityScorerConfig.
type ScorerConfig = fintech.FidelityScorerConfig

// ComplianceGap is an alias for fintech.ComplianceGap.
type ComplianceGap = fintech.ComplianceGap

// RequiredAction is an alias for fintech.RequiredAction.
type RequiredAction = fintech.RequiredAction

// DefaultScorerConfig returns the default fidelity scorer configuration.
func DefaultScorerConfig() *ScorerConfig {
	return fintech.DefaultFidelityScorerConfig()
}

// NewScorer creates a new FidelityScorer.
func NewScorer(tracer *fintech.NRVTracer, registry *ontology.OntologyRegistry, cfg *ScorerConfig) *Scorer {
	return fintech.NewFidelityScorer(tracer, registry, cfg)
}
