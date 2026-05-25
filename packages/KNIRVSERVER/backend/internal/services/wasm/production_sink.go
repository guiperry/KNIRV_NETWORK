package wasm

import (
	"context"
	"log"
	"time"
)

// RemediationInfrastructure provides the real system-level operations
// that ProductionRemediationSink dispatches after a successful resolution.
type RemediationInfrastructure interface {
	SetNodeStatus(nodeID, status string) error
	TerminateContainer(containerID string) error
	SetNodeCooldown(nodeID string, duration time.Duration) error
	ThrottleNode(nodeID string, rateLimit float64) error
	IsolateProcess(nodeID string, pid uint32) error
	DetachFromNetwork(nodeID string) error
	EmitAlert(eventType, source, message string)
}

// ProductionRemediationSink implements RemediationSink with real
// infrastructure actions. It dispatches the resolution result to
// the appropriate system-level operation based on the ErrorClassID
// returned by the resolution.wasm module.
type ProductionRemediationSink struct {
	infra RemediationInfrastructure
}

// NewProductionRemediationSink creates a sink backed by the given
// infrastructure adapter.
func NewProductionRemediationSink(infra RemediationInfrastructure) *ProductionRemediationSink {
	return &ProductionRemediationSink{infra: infra}
}

// DispatchRemediation executes the real remediation action determined
// by the resolution.wasm ErrorClassID.
func (s *ProductionRemediationSink) DispatchRemediation(ctx context.Context, result *ResolutionResult) error {
	if result == nil {
		return nil
	}

	log.Printf("[production-sink] dispatching remediation: node=%s dve=%s badge=%s class=%d resolved=%v",
		result.NodeID, result.DVEID, result.BadgeID, result.ErrorClassID, result.Resolved)

	if !result.Resolved {
		log.Printf("[production-sink] skipping unremediated result for node %s", result.NodeID)
		return nil
	}

	switch result.ErrorClassID {
	case ErrorClassResourceExhaustion:
		return s.handleResourceExhaustion(ctx, result)
	case ErrorClassLatency:
		return s.handleLatency(ctx, result)
	case ErrorClassSecurity:
		return s.handleSecurity(ctx, result)
	case ErrorClassCrash:
		return s.handleCrash(ctx, result)
	default:
		return s.handleUnknown(ctx, result)
	}
}

func (s *ProductionRemediationSink) handleResourceExhaustion(ctx context.Context, result *ResolutionResult) error {
	log.Printf("[production-sink] RESOURCE EXHAUSTION: quarantining node %s", result.NodeID)
	s.infra.EmitAlert("guardrail", "production_sink",
		"resource_exhaustion on node "+result.NodeID)
	if err := s.infra.SetNodeStatus(result.NodeID, "maintenance"); err != nil {
		return err
	}
	return s.infra.SetNodeCooldown(result.NodeID, 5*time.Minute)
}

func (s *ProductionRemediationSink) handleLatency(ctx context.Context, result *ResolutionResult) error {
	log.Printf("[production-sink] LATENCY: throttling node %s", result.NodeID)
	s.infra.EmitAlert("guardrail", "production_sink",
		"high_latency on node "+result.NodeID)
	if err := s.infra.ThrottleNode(result.NodeID, 0.5); err != nil {
		return err
	}
	return s.infra.DetachFromNetwork(result.NodeID)
}

func (s *ProductionRemediationSink) handleSecurity(ctx context.Context, result *ResolutionResult) error {
	log.Printf("[production-sink] SECURITY: kernel isolation for node %s (class=%d)",
		result.NodeID, result.ErrorClassID)
	s.infra.EmitAlert("guardrail", "production_sink",
		"kernel_isolation on node "+result.NodeID)
	if err := s.infra.SetNodeStatus(result.NodeID, "isolated"); err != nil {
		return err
	}
	return s.infra.DetachFromNetwork(result.NodeID)
}

func (s *ProductionRemediationSink) handleCrash(ctx context.Context, result *ResolutionResult) error {
	log.Printf("[production-sink] CRASH: restarting service for node %s", result.NodeID)
	s.infra.EmitAlert("guardrail", "production_sink",
		"service_crash on node "+result.NodeID)
	if err := s.infra.SetNodeStatus(result.NodeID, "restarting"); err != nil {
		return err
	}
	return s.infra.TerminateContainer(result.NodeID)
}

func (s *ProductionRemediationSink) handleUnknown(ctx context.Context, result *ResolutionResult) error {
	log.Printf("[production-sink] UNKNOWN class %d: alerting operators for node %s",
		result.ErrorClassID, result.NodeID)
	s.infra.EmitAlert("guardrail", "production_sink",
		"unknown_error_class node="+result.NodeID)
	return nil
}
