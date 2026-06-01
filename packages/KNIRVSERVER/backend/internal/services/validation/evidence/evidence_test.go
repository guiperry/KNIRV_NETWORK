package evidence

import (
	"testing"
	"time"
)

func TestNewEvidencePack(t *testing.T) {
	pack := NewEvidencePack(EvidenceTypeValidation, "agent-123", "validation-456")

	if pack.ID == "" {
		t.Error("expected non-empty ID")
	}
	if pack.AgentID != "agent-123" {
		t.Errorf("expected AgentID 'agent-123', got %q", pack.AgentID)
	}
	if pack.ValidationID != "validation-456" {
		t.Errorf("expected ValidationID 'validation-456', got %q", pack.ValidationID)
	}
	if pack.Type != EvidenceTypeValidation {
		t.Errorf("expected type %q, got %q", EvidenceTypeValidation, pack.Type)
	}
	if pack.Status != EvidenceStatusPending && pack.Status != EvidenceStatusCollecting {
		t.Errorf("unexpected initial status: %q", pack.Status)
	}
}

func TestMarkComplete(t *testing.T) {
	pack := NewEvidencePack(EvidenceTypeCompliance, "agent-1", "val-1")
	pack.MarkComplete()

	if pack.Status != EvidenceStatusComplete {
		t.Errorf("expected status 'complete', got %q", pack.Status)
	}
	if pack.CompletedAt == nil {
		t.Error("expected CompletedAt to be set after MarkComplete")
	}
}

func TestMarkFailed(t *testing.T) {
	pack := NewEvidencePack(EvidenceTypeAudit, "agent-2", "val-2")
	pack.MarkFailed("something went wrong")

	if pack.Status != EvidenceStatusFailed {
		t.Errorf("expected status 'failed', got %q", pack.Status)
	}
}

func TestAddComplianceCheck(t *testing.T) {
	pack := NewEvidencePack(EvidenceTypeCompliance, "agent-3", "val-3")

	check := ComplianceEvidence{
		RegulationID:   "AML-001",
		RegulationName: "AML Rule 1",
		Category:       "AML",
		Status:         "VIOLATED",
		Severity:       "HIGH",
		Description:    "Suspicious transaction pattern",
		CheckedAt:      time.Now(),
	}
	pack.AddComplianceCheck(check)

	if len(pack.ComplianceChecks) != 1 {
		t.Errorf("expected 1 compliance check, got %d", len(pack.ComplianceChecks))
	}
	if pack.ComplianceChecks[0].RegulationID != "AML-001" {
		t.Errorf("expected RegulationID 'AML-001', got %q", pack.ComplianceChecks[0].RegulationID)
	}
}

func TestBuilder(t *testing.T) {
	check := ComplianceEvidence{
		RegulationID: "KYC-001",
		Status:       "PASSED",
		CheckedAt:    time.Now(),
	}

	pack := NewBuilder(EvidenceTypeValidation, "agent-b", "val-b").
		WithAgentName("Test Agent").
		AddComplianceCheck(check).
		MarkComplete().
		Build()

	if pack.AgentName != "Test Agent" {
		t.Errorf("expected agent name 'Test Agent', got %q", pack.AgentName)
	}
	if len(pack.ComplianceChecks) != 1 {
		t.Errorf("expected 1 compliance check from builder, got %d", len(pack.ComplianceChecks))
	}
	if pack.Status != EvidenceStatusComplete {
		t.Errorf("expected status 'complete' from builder, got %q", pack.Status)
	}
}
