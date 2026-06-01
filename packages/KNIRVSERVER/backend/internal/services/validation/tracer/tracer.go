// Package tracer provides canonical types for NRV (Non-Repudiable Verification)
// reasoning traces. The generic tracer types are promoted from the fintech plugin
// package so they can be reused across different compliance validators.
package tracer

import (
	"backend_server/internal/services/plugins/fintech"
	"backend_server/internal/services/plugins/fintech/ontology"
)

// Re-export generic NRV types from the fintech package.

// NRVTraceStatus is an alias for fintech.NRVTraceStatus.
type NRVTraceStatus = fintech.NRVTraceStatus

// NRVTrace is an alias for fintech.NRVTrace.
type NRVTrace = fintech.NRVTrace

// NRVStep is an alias for fintech.NRVReasoningStep.
type NRVStep = fintech.NRVReasoningStep

// NRVTracer is an alias for fintech.NRVTracer.
type NRVTracer = fintech.NRVTracer

// Constants re-exported for convenience.
const (
	NRVTraceStatusCapturing = fintech.NRVTraceStatusCapturing
	NRVTraceStatusComplete  = fintech.NRVTraceStatusComplete
	NRVTraceStatusFailed    = fintech.NRVTraceStatusFailed
	NRVTraceStatusAnalyzed  = fintech.NRVTraceStatusAnalyzed
)

// NewNRVTracer creates a new NRVTracer using the provided ontology registry.
func NewNRVTracer(registry *ontology.OntologyRegistry) *NRVTracer {
	return fintech.NewNRVTracer(registry)
}
