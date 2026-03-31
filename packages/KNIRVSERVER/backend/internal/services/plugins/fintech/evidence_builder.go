package fintech

import (
	"context"
	"fmt"
	"time"
)

// EvidencePackBuilder builds EvidencePack instances step by step
type EvidencePackBuilder struct {
	pack *EvidencePack
	err  error
}

// NewEvidencePackBuilder creates a new builder for evidence packs
func NewEvidencePackBuilder(evidenceType EvidencePackType, agentID, validationID string) *EvidencePackBuilder {
	return &EvidencePackBuilder{
		pack: NewEvidencePack(evidenceType, agentID, validationID),
	}
}

// WithAgentName sets the agent name
func (b *EvidencePackBuilder) WithAgentName(name string) *EvidencePackBuilder {
	if b.err != nil {
		return b
	}
	b.pack.AgentName = name
	return b
}

// WithMetadata adds metadata
func (b *EvidencePackBuilder) WithMetadata(key string, value interface{}) *EvidencePackBuilder {
	if b.err != nil {
		return b
	}
	if b.pack.Metadata == nil {
		b.pack.Metadata = make(map[string]interface{})
	}
	b.pack.Metadata[key] = value
	return b
}

// WithEBPFTrace adds eBPF trace evidence
func (b *EvidencePackBuilder) WithEBPFTrace(trace *EBPFTraceEvidence) *EvidencePackBuilder {
	if b.err != nil {
		return b
	}
	b.pack.EBPFTrace = trace
	return b
}

// WithNRVTrace adds NRV trace evidence
func (b *EvidencePackBuilder) WithNRVTrace(trace *NRVTraceEvidence) *EvidencePackBuilder {
	if b.err != nil {
		return b
	}
	b.pack.NRVTrace = trace
	return b
}

// WithValidationResult adds validation result evidence
func (b *EvidencePackBuilder) WithValidationResult(result *ValidationEvidence) *EvidencePackBuilder {
	if b.err != nil {
		return b
	}
	b.pack.ValidationResult = result
	return b
}

// AddComplianceCheck adds a compliance check
func (b *EvidencePackBuilder) AddComplianceCheck(check ComplianceEvidence) *EvidencePackBuilder {
	if b.err != nil {
		return b
	}
	b.pack.AddComplianceCheck(check)
	return b
}

// WithSignature adds PQC signature
func (b *EvidencePackBuilder) WithSignature(signature *PQCSignature) *EvidencePackBuilder {
	if b.err != nil {
		return b
	}
	b.pack.Signature = signature
	return b
}

// Build finalizes and returns the EvidencePack
func (b *EvidencePackBuilder) Build() (*EvidencePack, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.pack, nil
}

// EvidenceCollector manages the collection of evidence during validation
type EvidenceCollector struct {
	builder    *EvidencePackBuilder
	startTime  time.Time
	ctx        context.Context
	cancel     context.CancelFunc
	onComplete func(*EvidencePack)
	onError    func(error)
}

// NewEvidenceCollector creates a new evidence collector
func NewEvidenceCollector(ctx context.Context, evidenceType EvidencePackType, agentID, validationID string) *EvidenceCollector {
	ctx, cancel := context.WithCancel(ctx)
	return &EvidenceCollector{
		builder:   NewEvidencePackBuilder(evidenceType, agentID, validationID),
		startTime: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// SetCallbacks sets completion and error callbacks
func (ec *EvidenceCollector) SetCallbacks(onComplete func(*EvidencePack), onError func(error)) {
	ec.onComplete = onComplete
	ec.onError = onError
}

// Start begins evidence collection
func (ec *EvidenceCollector) Start() error {
	ec.builder.pack.Status = EvidenceStatusCollecting
	return nil
}

// Stop stops evidence collection and finalizes the pack
func (ec *EvidenceCollector) Stop() (*EvidencePack, error) {
	defer ec.cancel()

	pack, err := ec.builder.Build()
	if err != nil {
		if ec.onError != nil {
			ec.onError(err)
		}
		return nil, err
	}

	pack.MarkComplete()

	if ec.onComplete != nil {
		ec.onComplete(pack)
	}

	return pack, nil
}

// StopWithError stops collection and marks as failed
func (ec *EvidenceCollector) StopWithError(reason string) (*EvidencePack, error) {
	defer ec.cancel()

	pack, err := ec.builder.Build()
	if err != nil {
		if ec.onError != nil {
			ec.onError(err)
		}
		return nil, err
	}

	pack.MarkFailed(reason)

	if ec.onComplete != nil {
		ec.onComplete(pack)
	}

	return pack, nil
}

// GetBuilder returns the underlying builder for incremental construction
func (ec *EvidenceCollector) GetBuilder() *EvidencePackBuilder {
	return ec.builder
}

// Context returns the collector's context
func (ec *EvidenceCollector) Context() context.Context {
	return ec.ctx
}

// EvidenceStore defines the interface for persisting evidence packs
type EvidenceStore interface {
	// Save persists an evidence pack
	Save(pack *EvidencePack) error

	// Load retrieves an evidence pack by ID
	Load(id string) (*EvidencePack, error)

	// List returns a list of evidence packs with optional filtering
	List(filter EvidenceFilter) ([]*EvidencePack, error)

	// Delete removes an evidence pack
	Delete(id string) error

	// ExportMarkdown exports an evidence pack as Markdown
	ExportMarkdown(id string) ([]byte, error)

	// Sign signs an evidence pack with PQC
	Sign(id string, keyID string) error
}

// EvidenceFilter defines filtering criteria for listing evidence packs
type EvidenceFilter struct {
	AgentID   string
	Type      EvidencePackType
	Status    EvidenceStatus
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// EvidenceExporter handles exporting evidence packs in various formats
type EvidenceExporter struct {
	store EvidenceStore
}

// NewEvidenceExporter creates a new evidence exporter
func NewEvidenceExporter(store EvidenceStore) *EvidenceExporter {
	return &EvidenceExporter{store: store}
}

// ExportToJSON exports an evidence pack as JSON
func (e *EvidenceExporter) ExportToJSON(id string) ([]byte, error) {
	pack, err := e.store.Load(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load evidence pack: %w", err)
	}
	return pack.ToJSON()
}

// ExportToMarkdown exports an evidence pack as Markdown
func (e *EvidenceExporter) ExportToMarkdown(id string) ([]byte, error) {
	return e.store.ExportMarkdown(id)
}

// ExportEvidencePack exports a complete evidence pack bundle
func (e *EvidenceExporter) ExportEvidencePack(id string) (*EvidencePackBundle, error) {
	pack, err := e.store.Load(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load evidence pack: %w", err)
	}

	jsonData, err := pack.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to JSON: %w", err)
	}

	mdData, err := pack.GenerateMarkdown()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Markdown: %w", err)
	}

	return &EvidencePackBundle{
		ID:           pack.ID,
		EvidenceJSON: jsonData,
		EvidenceMD:   mdData,
		Signature:    pack.Signature,
		CreatedAt:    pack.CreatedAt,
	}, nil
}

// EvidencePackBundle contains all evidence pack formats for export
type EvidencePackBundle struct {
	ID           string        `json:"id"`
	EvidenceJSON []byte        `json:"evidence_json"`
	EvidenceMD   []byte        `json:"evidence_md"`
	Signature    *PQCSignature `json:"signature,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
}
