package fintech

import (
	"encoding/json"
	"fmt"
	"time"
)

// EvidencePackType defines the type of evidence being collected
type EvidencePackType string

const (
	EvidenceTypeValidation EvidencePackType = "VALIDATION"
	EvidenceTypeCompliance EvidencePackType = "COMPLIANCE"
	EvidenceTypeAudit      EvidencePackType = "AUDIT"
	EvidenceTypeReplay     EvidencePackType = "REPLAY"
)

// EvidenceStatus represents the status of an evidence pack
type EvidenceStatus string

const (
	EvidenceStatusPending    EvidenceStatus = "pending"
	EvidenceStatusCollecting EvidenceStatus = "collecting"
	EvidenceStatusComplete   EvidenceStatus = "complete"
	EvidenceStatusFailed     EvidenceStatus = "failed"
)

// EvidencePack represents a comprehensive audit trail for financial AI agent validation
type EvidencePack struct {
	ID           string           `json:"id"`
	Type         EvidencePackType `json:"type"`
	Status       EvidenceStatus   `json:"status"`
	AgentID      string           `json:"agent_id"`
	AgentName    string           `json:"agent_name,omitempty"`
	ValidationID string           `json:"validation_id"`
	CreatedAt    time.Time        `json:"created_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`

	// Core Evidence Components
	EBPFTrace        *EBPFTraceEvidence   `json:"ebpf_trace,omitempty"`
	NRVTrace         *NRVTraceEvidence    `json:"nrv_trace,omitempty"`
	ValidationResult *ValidationEvidence  `json:"validation_result,omitempty"`
	ComplianceChecks []ComplianceEvidence `json:"compliance_checks,omitempty"`

	// PQC Signature
	Signature *PQCSignature `json:"signature,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EBPFTraceEvidence contains syscall-level execution traces
type EBPFTraceEvidence struct {
	TraceID       string            `json:"trace_id"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       time.Time         `json:"end_time"`
	Syscalls      []SyscallEvent    `json:"syscalls"`
	NetworkEvents []NetworkEvent    `json:"network_events,omitempty"`
	FileAccesses  []FileAccessEvent `json:"file_accesses,omitempty"`
	ProcessTree   *ProcessTree      `json:"process_tree,omitempty"`
	RawTrace      []byte            `json:"-"` // Raw trace data, not serialized
}

// SyscallEvent represents a single syscall capture
type SyscallEvent struct {
	Timestamp  int64             `json:"timestamp"`
	PID        uint32            `json:"pid"`
	TGID       uint32            `json:"tgid"`
	SyscallID  uint64            `json:"syscall_id"`
	Arguments  map[string]string `json:"args,omitempty"`
	ReturnCode int64             `json:"return_code"`
	Duration   int64             `json:"duration_ns"`
}

// NetworkEvent represents a network operation
type NetworkEvent struct {
	Timestamp  int64  `json:"timestamp"`
	PID        uint32 `json:"pid"`
	Operation  string `json:"operation"` // connect, bind, send, recv
	SourceIP   string `json:"source_ip,omitempty"`
	SourcePort uint16 `json:"source_port,omitempty"`
	DestIP     string `json:"dest_ip,omitempty"`
	DestPort   uint16 `json:"dest_port,omitempty"`
	Bytes      uint64 `json:"bytes,omitempty"`
}

// FileAccessEvent represents a file system operation
type FileAccessEvent struct {
	Timestamp int64  `json:"timestamp"`
	PID       uint32 `json:"pid"`
	Operation string `json:"operation"` // open, read, write, close
	Path      string `json:"path"`
	Flags     string `json:"flags,omitempty"`
	Bytes     int64  `json:"bytes,omitempty"`
}

// ProcessTree represents the process hierarchy
type ProcessTree struct {
	RootPID  uint32                  `json:"root_pid"`
	RootComm string                  `json:"root_comm"`
	Children map[uint32]*ProcessNode `json:"children"`
}

// ProcessNode represents a process in the tree
type ProcessNode struct {
	PID      uint32                  `json:"pid"`
	PPID     uint32                  `json:"ppid"`
	Comm     string                  `json:"comm"`
	Children map[uint32]*ProcessNode `json:"children,omitempty"`
}

// NRVTraceEvidence contains reasoning and semantic traces
type NRVTraceEvidence struct {
	TraceID         string                 `json:"trace_id"`
	AgentID         string                 `json:"agent_id"`
	ReasoningSteps  []ReasoningStep        `json:"reasoning_steps"`
	SemanticContext map[string]interface{} `json:"semantic_context,omitempty"`
	FidelityScore   float64                `json:"fidelity_score,omitempty"`
	OntologyRefs    []string               `json:"ontology_refs,omitempty"`
}

// ReasoningStep represents a single step in the reasoning process
type ReasoningStep struct {
	StepNumber  int                    `json:"step_number"`
	Timestamp   time.Time              `json:"timestamp"`
	Intent      string                 `json:"intent"`
	Observation string                 `json:"observation"`
	Action      string                 `json:"action,omitempty"`
	Confidence  float64                `json:"confidence"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationEvidence contains the validation outcome
type ValidationEvidence struct {
	ValidatorID   string                 `json:"validator_id"`
	ValidatorType string                 `json:"validator_type"`
	Status        string                 `json:"status"` // valid, invalid, inconclusive
	Confidence    float64                `json:"confidence"`
	Score         float64                `json:"score"`
	Details       map[string]interface{} `json:"details,omitempty"`
	TestCases     []TestCaseResult       `json:"test_cases,omitempty"`
	ExecutedAt    time.Time              `json:"executed_at"`
	Duration      int64                  `json:"duration_ms"`
}

// TestCaseResult represents a single test case outcome
type TestCaseResult struct {
	Name     string                 `json:"name"`
	Passed   bool                   `json:"passed"`
	Input    map[string]interface{} `json:"input,omitempty"`
	Expected interface{}            `json:"expected,omitempty"`
	Actual   interface{}            `json:"actual,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Duration int64                  `json:"duration_ms"`
}

// ComplianceEvidence contains regulatory compliance check results
type ComplianceEvidence struct {
	RegulationID   string                 `json:"regulation_id"`
	RegulationName string                 `json:"regulation_name"`
	Category       string                 `json:"category"` // AML, KYC, SEC, etc.
	Status         string                 `json:"status"`   // compliant, violated, warning
	Severity       string                 `json:"severity"` // critical, high, medium, low
	Description    string                 `json:"description"`
	Details        map[string]interface{} `json:"details,omitempty"`
	CheckedAt      time.Time              `json:"checked_at"`
}

// PQCSignature contains post-quantum cryptographic signatures
type PQCSignature struct {
	Algorithm   string    `json:"algorithm"`
	PublicKeyID string    `json:"public_key_id"`
	Signature   string    `json:"signature"`
	SignedAt    time.Time `json:"signed_at"`
	SignedData  []byte    `json:"-"` // Original signed data
}

// NewEvidencePack creates a new evidence pack with a unique ID
func NewEvidencePack(evidenceType EvidencePackType, agentID, validationID string) *EvidencePack {
	return &EvidencePack{
		ID:               generateEvidenceID(),
		Type:             evidenceType,
		Status:           EvidenceStatusPending,
		AgentID:          agentID,
		ValidationID:     validationID,
		CreatedAt:        time.Now(),
		Metadata:         make(map[string]interface{}),
		ComplianceChecks: make([]ComplianceEvidence, 0),
	}
}

// GenerateMarkdown generates a human-readable Markdown representation
func (ep *EvidencePack) GenerateMarkdown() ([]byte, error) {
	md := fmt.Sprintf("# Evidence Pack: %s\n\n", ep.ID)
	md += fmt.Sprintf("**Type:** %s  \n", ep.Type)
	md += fmt.Sprintf("**Status:** %s  \n", ep.Status)
	md += fmt.Sprintf("**Agent ID:** %s  \n", ep.AgentID)
	md += fmt.Sprintf("**Validation ID:** %s  \n", ep.ValidationID)
	md += fmt.Sprintf("**Created:** %s  \n", ep.CreatedAt.Format(time.RFC3339))
	if ep.CompletedAt != nil {
		md += fmt.Sprintf("**Completed:** %s  \n", ep.CompletedAt.Format(time.RFC3339))
	}
	md += "\n"

	// Validation Result Section
	if ep.ValidationResult != nil {
		md += "## Validation Result\n\n"
		md += fmt.Sprintf("- **Status:** %s\n", ep.ValidationResult.Status)
		md += fmt.Sprintf("- **Confidence:** %.2f%%\n", ep.ValidationResult.Confidence*100)
		md += fmt.Sprintf("- **Score:** %.2f\n", ep.ValidationResult.Score)
		md += fmt.Sprintf("- **Duration:** %dms\n", ep.ValidationResult.Duration)
		md += "\n"
	}

	// NRV Trace Section
	if ep.NRVTrace != nil {
		md += "## Reasoning Trace (NRV)\n\n"
		md += fmt.Sprintf("**Fidelity Score:** %.2f  \n\n", ep.NRVTrace.FidelityScore)
		for _, step := range ep.NRVTrace.ReasoningSteps {
			md += fmt.Sprintf("### Step %d\n", step.StepNumber)
			md += fmt.Sprintf("- **Intent:** %s\n", step.Intent)
			md += fmt.Sprintf("- **Observation:** %s\n", step.Observation)
			if step.Action != "" {
				md += fmt.Sprintf("- **Action:** %s\n", step.Action)
			}
			md += fmt.Sprintf("- **Confidence:** %.2f%%\n", step.Confidence*100)
			md += "\n"
		}
	}

	// eBPF Trace Section
	if ep.EBPFTrace != nil {
		md += "## Execution Trace (eBPF)\n\n"
		md += fmt.Sprintf("**Trace ID:** %s  \n", ep.EBPFTrace.TraceID)
		md += fmt.Sprintf("**Start Time:** %s  \n", ep.EBPFTrace.StartTime.Format(time.RFC3339))
		md += fmt.Sprintf("**End Time:** %s  \n", ep.EBPFTrace.EndTime.Format(time.RFC3339))
		md += fmt.Sprintf("**Syscalls Captured:** %d  \n", len(ep.EBPFTrace.Syscalls))
		md += fmt.Sprintf("**Network Events:** %d  \n", len(ep.EBPFTrace.NetworkEvents))
		md += fmt.Sprintf("**File Accesses:** %d  \n\n", len(ep.EBPFTrace.FileAccesses))
	}

	// Compliance Section
	if len(ep.ComplianceChecks) > 0 {
		md += "## Compliance Checks\n\n"
		for _, check := range ep.ComplianceChecks {
			statusEmoji := "✅"
			if check.Status == "violated" {
				statusEmoji = "❌"
			} else if check.Status == "warning" {
				statusEmoji = "⚠️"
			}
			md += fmt.Sprintf("### %s %s\n", statusEmoji, check.RegulationName)
			md += fmt.Sprintf("- **Category:** %s\n", check.Category)
			md += fmt.Sprintf("- **Status:** %s\n", check.Status)
			md += fmt.Sprintf("- **Severity:** %s\n", check.Severity)
			md += fmt.Sprintf("- **Description:** %s\n", check.Description)
			md += "\n"
		}
	}

	// Signature Section
	if ep.Signature != nil {
		md += "## PQC Signature\n\n"
		md += fmt.Sprintf("**Algorithm:** %s  \n", ep.Signature.Algorithm)
		md += fmt.Sprintf("**Public Key ID:** %s  \n", ep.Signature.PublicKeyID)
		md += fmt.Sprintf("**Signed At:** %s  \n", ep.Signature.SignedAt.Format(time.RFC3339))
		md += "\n"
	}

	return []byte(md), nil
}

// ToJSON returns the evidence pack as JSON
func (ep *EvidencePack) ToJSON() ([]byte, error) {
	return json.MarshalIndent(ep, "", "  ")
}

// AddComplianceCheck adds a compliance check to the evidence pack
func (ep *EvidencePack) AddComplianceCheck(check ComplianceEvidence) {
	ep.ComplianceChecks = append(ep.ComplianceChecks, check)
}

// MarkComplete marks the evidence pack as complete
func (ep *EvidencePack) MarkComplete() {
	now := time.Now()
	ep.CompletedAt = &now
	ep.Status = EvidenceStatusComplete
}

// MarkFailed marks the evidence pack as failed
func (ep *EvidencePack) MarkFailed(reason string) {
	now := time.Now()
	ep.CompletedAt = &now
	ep.Status = EvidenceStatusFailed
	ep.Metadata["failure_reason"] = reason
}

// helper function to generate unique IDs
func generateEvidenceID() string {
	return fmt.Sprintf("evidence_%d_%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
