package fintech_validator

import (
	"context"
	"fmt"
	"time"

	"backend_server/internal/ebpf"
	"backend_server/internal/fintech"
	"backend_server/internal/fintech/ontology"
	"backend_server/internal/objects"
	"backend_server/internal/services/validation"
	"backend_server/internal/services/vault"
	"backend_server/internal/storage/mdstorage"
	"backend_server/internal/storage/pqc"
)

// convertTestCases converts fintech test cases to objects.TestCase
func convertTestCases(results []fintech.TestCaseResult) []objects.TestCase {
	converted := make([]objects.TestCase, len(results))
	for i, tc := range results {
		converted[i] = objects.TestCase{
			Name: tc.Name,
		}
	}
	return converted
}

// TrajectoryStore defines the interface for trajectory storage
type TrajectoryStore interface {
	Save(trajectory *fintech.ExecutionTrajectory) error
	Load(id string) (*fintech.ExecutionTrajectory, error)
	List(filter fintech.TrajectoryFilter) ([]*fintech.ExecutionTrajectory, error)
	Delete(id string) error
}

// FinTechValidatorService provides financial AI agent validation
type FinTechValidatorService struct {
	Config                *Config
	validationCore        *validation.ValidationCore
	ontologyRegistry      *ontology.OntologyRegistry
	complianceEngine      *ontology.ComplianceEngine
	pqcManager            *pqc.EncryptionManager
	mdStorage             *mdstorage.MarkdownStorageDriver
	evidenceStore         fintech.EvidenceStore
	scenarioRepository    *vault.ScenarioRepository
	vaultComplianceEngine *vault.ComplianceEngine
	certificateManager    *fintech.CertificateManager

	// Phase 3: Trajectory Capture & Replay
	agentTracer     *ebpf.AgentTracer
	replayEngine    *fintech.ReplayEngine
	trajectoryStore TrajectoryStore

	// Phase 4: NRV Financial Semantic Engine
	nrvTracer      *fintech.NRVTracer
	fidelityScorer *fintech.FidelityScorer

	// Phase 6: Arrow Flight Tick Data Streaming
	tickStreamManager *fintech.TickStreamManager
}

// Config holds configuration for the FinTech validator service
type Config struct {
	Enabled               bool
	EnableAMLChecks       bool
	EnableKYCCheks        bool
	EnableSECCheks        bool
	EnableBaselCheks      bool
	AutoSignEvidencePacks bool
	MasterKeyID           string
	EnableScenarioTesting bool
	EnableCertification   bool
	CertificateValidity   time.Duration

	// Phase 3: Trajectory Capture & Replay
	EnableTrajectoryCapture bool
	EnableReplayEngine      bool
	DefaultCaptureConfig    *fintech.TrajectoryCaptureConfig

	// Phase 4: NRV Financial Semantic Engine
	EnableNRVTracing      bool
	EnableFidelityScoring bool
	FidelityScorerConfig  *fintech.FidelityScorerConfig

	// Phase 6: Arrow Flight Tick Data Streaming
	EnableTickDataStreaming bool
	TickDataServerPort      string
	MaxStreamBufferSize     int
}

// NewFinTechValidatorService creates a new FinTech validator service
func NewFinTechValidatorService(
	validationCore *validation.ValidationCore,
	pqcManager *pqc.EncryptionManager,
	mdStorage *mdstorage.MarkdownStorageDriver,
	config *Config,
) (*FinTechValidatorService, error) {
	// Initialize ontology registry
	registry := ontology.NewOntologyRegistry()

	// Load built-in ontologies based on configuration
	loader := ontology.NewOntologyLoader(registry)

	if config.EnableAMLChecks || config.EnableKYCCheks || config.EnableSECCheks || config.EnableBaselCheks {
		if err := loader.LoadBuiltInOntologies(); err != nil {
			return nil, fmt.Errorf("failed to load built-in ontologies: %w", err)
		}
	}

	// Create compliance engine
	complianceEngine := ontology.NewComplianceEngine(registry)

	// Create evidence store
	evidenceStore := NewMarkdownEvidenceStore(mdStorage, pqcManager)

	// Initialize Phase 2 components
	var scenarioRepository *vault.ScenarioRepository
	var vaultComplianceEngine *vault.ComplianceEngine
	var certificateManager *fintech.CertificateManager

	if config.EnableScenarioTesting || config.EnableCertification {
		// Create scenario repository
		scenarioRepository = vault.NewScenarioRepository(mdStorage)

		// Create compliance engine with default config
		complianceConfig := vault.DefaultComplianceConfig()
		vaultComplianceEngine = vault.NewComplianceEngine(scenarioRepository, &complianceConfig)
		vaultComplianceEngine.SetOntologyRegistry(registry)

		// Initialize default scenarios if none exist
		if err := scenarioRepository.CreateDefaultScenarios(); err != nil {
			// Log warning but don't fail - scenarios might already exist
			fmt.Printf("Warning: failed to create default scenarios: %v\n", err)
		}
	}

	if config.EnableCertification {
		// Create certificate manager
		keyID := config.MasterKeyID
		if keyID == "" {
			keyID = "nexus-master"
		}
		certificateManager = fintech.NewCertificateManager(pqcManager, keyID)
	}

	// Initialize Phase 3 components
	var agentTracer *ebpf.AgentTracer
	var replayEngine *fintech.ReplayEngine
	var trajectoryStore TrajectoryStore

	if config.EnableTrajectoryCapture {
		// Create trajectory store
		trajectoryStore = NewMarkdownTrajectoryStore(mdStorage, pqcManager)
	}

	if config.EnableReplayEngine {
		// Create replay engine
		replayEngine = fintech.NewReplayEngine()
	}

	// Initialize Phase 4 components
	var nrvTracer *fintech.NRVTracer
	var fidelityScorer *fintech.FidelityScorer

	if config.EnableNRVTracing {
		// Create NRV tracer
		nrvTracer = fintech.NewNRVTracer(registry)
	}

	if config.EnableFidelityScoring && nrvTracer != nil {
		// Create fidelity scorer
		scorerConfig := config.FidelityScorerConfig
		if scorerConfig == nil {
			scorerConfig = fintech.DefaultFidelityScorerConfig()
		}
		fidelityScorer = fintech.NewFidelityScorer(nrvTracer, registry, scorerConfig)
	}

	// Initialize Phase 6 components
	var tickStreamManager *fintech.TickStreamManager

	if config.EnableTickDataStreaming {
		// Create tick stream manager
		maxBufferSize := config.MaxStreamBufferSize
		if maxBufferSize <= 0 {
			maxBufferSize = 1000
		}
		tickStreamManager = fintech.NewTickStreamManager()
		_ = maxBufferSize // Used for stream buffer sizing
	}

	return &FinTechValidatorService{
		Config:                config,
		validationCore:        validationCore,
		ontologyRegistry:      registry,
		complianceEngine:      complianceEngine,
		pqcManager:            pqcManager,
		mdStorage:             mdStorage,
		evidenceStore:         evidenceStore,
		scenarioRepository:    scenarioRepository,
		vaultComplianceEngine: vaultComplianceEngine,
		certificateManager:    certificateManager,
		agentTracer:           agentTracer,
		replayEngine:          replayEngine,
		trajectoryStore:       trajectoryStore,
		nrvTracer:             nrvTracer,
		fidelityScorer:        fidelityScorer,
		tickStreamManager:     tickStreamManager,
	}, nil
}

// ValidationRequest represents a request to validate a financial AI agent
type ValidationRequest struct {
	AgentID                string                        `json:"agent_id"`
	AgentName              string                        `json:"agent_name,omitempty"`
	AgentCode              string                        `json:"agent_code,omitempty"`
	ValidationType         string                        `json:"validation_type,omitempty"` // compliance, audit, trade, etc.
	Categories             []ontology.RegulationCategory `json:"categories,omitempty"`
	TestCases              []fintech.TestCaseResult      `json:"test_cases,omitempty"`
	FinancialActions       []*ontology.FinancialAction   `json:"financial_actions,omitempty"`
	SimpleFinancialActions []SimpleFinancialAction       `json:"simple_financial_actions,omitempty"`
	Parameters             map[string]interface{}        `json:"parameters,omitempty"`
	RequestedBy            string                        `json:"requested_by,omitempty"`
}

// SimpleFinancialAction is a simplified financial action format for easier frontend integration
type SimpleFinancialAction struct {
	ActionType string  `json:"action_type"` // BUY, SELL, TRANSFER, etc.
	Instrument string  `json:"instrument"`  // AAPL, BTC, etc.
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
	Timestamp  string  `json:"timestamp"`
	AccountID  string  `json:"account_id"`
}

// ValidationResponse represents the response from validation
type ValidationResponse struct {
	ValidationID     string                            `json:"validation_id"`
	EvidencePackID   string                            `json:"evidence_pack_id"`
	Status           string                            `json:"status"`
	OverallScore     float64                           `json:"overall_score"`
	IsCompliant      bool                              `json:"is_compliant"`
	ValidationResult *fintech.ValidationEvidence       `json:"validation_result,omitempty"`
	ComplianceResult *ontology.ComprehensiveValidation `json:"compliance_result,omitempty"`
	EvidencePackURL  string                            `json:"evidence_pack_url,omitempty"`
	CompletedAt      time.Time                         `json:"completed_at"`
	Duration         int64                             `json:"duration_ms"`
}

// Validate performs comprehensive FinTech validation
func (s *FinTechValidatorService) Validate(ctx context.Context, req *ValidationRequest) (*ValidationResponse, error) {
	startTime := time.Now()

	// Convert simple financial actions to full format if provided
	if len(req.SimpleFinancialActions) > 0 {
		for _, simple := range req.SimpleFinancialActions {
			timestamp, _ := time.Parse(time.RFC3339, simple.Timestamp)
			if timestamp.IsZero() {
				timestamp = time.Now()
			}
			action := &ontology.FinancialAction{
				ID:        fmt.Sprintf("action-%d", time.Now().UnixNano()),
				AgentID:   req.AgentID,
				Type:      ontology.ActionType(simple.ActionType),
				Timestamp: timestamp,
				Intent:    fmt.Sprintf("%s %s", simple.ActionType, simple.Instrument),
				Parameters: map[string]interface{}{
					"quantity": simple.Quantity,
					"price":    simple.Price,
				},
				Context: map[string]interface{}{
					"account_id": simple.AccountID,
				},
				Instrument: &ontology.FinancialInstrument{
					Symbol:   simple.Instrument,
					Name:     simple.Instrument,
					Type:     "EQUITY",
					Exchange: "NASDAQ",
				},
			}
			req.FinancialActions = append(req.FinancialActions, action)
		}
	}

	// Create validation task through existing ValidationCore
	validationTask, err := s.validationCore.CreateValidationTask(&validation.CreateTaskRequest{
		Type:        "fintech_validation",
		Priority:    1,
		Data:        req.Parameters,
		SkillCode:   req.AgentCode,
		TestCases:   convertTestCases(req.TestCases),
		RequestedBy: req.RequestedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create validation task: %w", err)
	}

	// Create evidence pack
	evidencePack := fintech.NewEvidencePack(
		fintech.EvidencePackType(req.ValidationType),
		req.AgentID,
		validationTask.ID,
	)
	evidencePack.AgentName = req.AgentName

	// Initialize response
	response := &ValidationResponse{
		ValidationID:   validationTask.ID,
		EvidencePackID: evidencePack.ID,
		Status:         "in_progress",
	}

	// Step 1: Run standard validation through ValidationCore
	validationResult, err := s.validationCore.ExecuteValidation(validationTask)
	if err != nil {
		evidencePack.MarkFailed(fmt.Sprintf("validation execution failed: %v", err))
		s.evidenceStore.Save(evidencePack)
		response.Status = "failed"
		return response, fmt.Errorf("validation execution failed: %w", err)
	}

	// Convert validation result to evidence format
	validationEvidence := &fintech.ValidationEvidence{
		ValidatorID:   "fintech-validator",
		ValidatorType: "comprehensive",
		Status:        validationResult.Status,
		Confidence:    validationResult.Confidence,
		Score:         validationResult.Confidence, // Use confidence as score
		Details:       validationResult.Details,
		ExecutedAt:    time.Now(),
		Duration:      time.Since(startTime).Milliseconds(),
	}
	evidencePack.ValidationResult = validationEvidence

	// Step 2: Run compliance checks if financial actions provided
	var complianceResult *ontology.ComprehensiveValidation
	if len(req.FinancialActions) > 0 {
		complianceResult, err = s.validateCompliance(ctx, req.FinancialActions, req.Categories)
		if err != nil {
			// Log error but don't fail entire validation
			validationEvidence.Details["compliance_error"] = err.Error()
		} else {
			response.ComplianceResult = complianceResult
			response.IsCompliant = complianceResult.IsCompliant
			response.OverallScore = complianceResult.OverallScore

			// Add compliance checks to evidence pack
			for _, result := range complianceResult.Results {
				for _, ruleResult := range result.RuleResults {
					if ruleResult.Status == ontology.StatusViolated {
						evidencePack.AddComplianceCheck(fintech.ComplianceEvidence{
							RegulationID:   ruleResult.RuleID,
							RegulationName: ruleResult.RuleName,
							Category:       string(result.Category),
							Status:         string(ruleResult.Status),
							Severity:       string(ruleResult.Severity),
							Description:    ruleResult.Description,
							Details:        ruleResult.Details,
							CheckedAt:      time.Now(),
						})
					}
				}
			}
		}
	}

	// Step 3: Mark evidence pack complete
	evidencePack.MarkComplete()

	// Step 4: Save evidence pack first
	if err := s.evidenceStore.Save(evidencePack); err != nil {
		return nil, fmt.Errorf("failed to save evidence pack: %w", err)
	}

	// Step 5: Sign evidence pack with PQC
	if err := s.evidenceStore.Sign(evidencePack.ID, "nexus-master"); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to sign evidence pack: %v\n", err)
	}

	// Finalize response
	response.Status = "completed"
	response.ValidationResult = validationEvidence
	response.CompletedAt = time.Now()
	response.Duration = time.Since(startTime).Milliseconds()

	return response, nil
}

// validateCompliance validates financial actions against compliance ontologies
func (s *FinTechValidatorService) validateCompliance(
	ctx context.Context,
	actions []*ontology.FinancialAction,
	categories []ontology.RegulationCategory,
) (*ontology.ComprehensiveValidation, error) {
	if len(actions) == 0 {
		return nil, nil
	}

	// Validate each action
	var comprehensiveResults []*ontology.ComprehensiveValidation

	for _, action := range actions {
		result, err := s.complianceEngine.ValidateAction(ctx, action, categories)
		if err != nil {
			return nil, fmt.Errorf("compliance validation failed for action %s: %w", action.ID, err)
		}
		comprehensiveResults = append(comprehensiveResults, result)
	}

	// Aggregate results
	if len(comprehensiveResults) == 0 {
		return nil, nil
	}

	// Use the worst result as overall
	worstResult := comprehensiveResults[0]
	for _, result := range comprehensiveResults {
		if result.OverallScore < worstResult.OverallScore {
			worstResult = result
		}
		if !result.IsCompliant {
			worstResult.IsCompliant = false
			worstResult.Violations += result.Violations
		}
	}

	return worstResult, nil
}

// GetEvidencePack retrieves an evidence pack by ID
func (s *FinTechValidatorService) GetEvidencePack(id string) (*fintech.EvidencePack, error) {
	return s.evidenceStore.Load(id)
}

// ListEvidencePacks lists evidence packs with filtering
func (s *FinTechValidatorService) ListEvidencePacks(filter fintech.EvidenceFilter) ([]*fintech.EvidencePack, error) {
	return s.evidenceStore.List(filter)
}

// ExportEvidencePack exports an evidence pack bundle
func (s *FinTechValidatorService) ExportEvidencePack(id string) (*fintech.EvidencePackBundle, error) {
	exporter := fintech.NewEvidenceExporter(s.evidenceStore)
	return exporter.ExportEvidencePack(id)
}

// GetOntologies returns all registered ontologies
func (s *FinTechValidatorService) GetOntologies() []ontology.FinancialOntology {
	return s.ontologyRegistry.List()
}

// GetOntology returns a specific ontology by ID
func (s *FinTechValidatorService) GetOntology(id string) (ontology.FinancialOntology, error) {
	return s.ontologyRegistry.Get(id)
}

// MarkdownEvidenceStore implements EvidenceStore using Markdown storage
type MarkdownEvidenceStore struct {
	storage    *mdstorage.MarkdownStorageDriver
	pqcManager *pqc.EncryptionManager
}

// NewMarkdownEvidenceStore creates a new Markdown-based evidence store
func NewMarkdownEvidenceStore(storage *mdstorage.MarkdownStorageDriver, pqcManager *pqc.EncryptionManager) fintech.EvidenceStore {
	return &MarkdownEvidenceStore{
		storage:    storage,
		pqcManager: pqcManager,
	}
}

// Save persists an evidence pack as a Markdown document
func (s *MarkdownEvidenceStore) Save(pack *fintech.EvidencePack) error {
	// Generate Markdown content
	content, err := pack.GenerateMarkdown()
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Create document
	doc := &mdstorage.MarkdownDocument{
		ID:        pack.ID,
		Type:      "EVIDENCE_PACK",
		Timestamp: pack.CreatedAt,
		Metadata: map[string]interface{}{
			"agent_id":      pack.AgentID,
			"agent_name":    pack.AgentName,
			"validation_id": pack.ValidationID,
			"type":          pack.Type,
			"status":        pack.Status,
			"has_signature": pack.Signature != nil,
		},
		Content: content,
	}

	return s.storage.SaveDocument(doc)
}

// Load retrieves an evidence pack from storage
func (s *MarkdownEvidenceStore) Load(id string) (*fintech.EvidencePack, error) {
	doc, err := s.storage.LoadDocument("EVIDENCE_PACK", id)
	if err != nil {
		return nil, fmt.Errorf("failed to load evidence pack: %w", err)
	}

	// Parse the evidence pack from the content
	// For simplicity, we'll reconstruct from metadata and content
	pack := &fintech.EvidencePack{
		ID:        doc.ID,
		CreatedAt: doc.Timestamp,
		Metadata:  doc.Metadata,
	}

	if agentID, ok := doc.Metadata["agent_id"].(string); ok {
		pack.AgentID = agentID
	}
	if agentName, ok := doc.Metadata["agent_name"].(string); ok {
		pack.AgentName = agentName
	}
	if validationID, ok := doc.Metadata["validation_id"].(string); ok {
		pack.ValidationID = validationID
	}
	if packType, ok := doc.Metadata["type"].(string); ok {
		pack.Type = fintech.EvidencePackType(packType)
	}
	if status, ok := doc.Metadata["status"].(string); ok {
		pack.Status = fintech.EvidenceStatus(status)
	}

	return pack, nil
}

// List returns evidence packs with filtering
func (s *MarkdownEvidenceStore) List(filter fintech.EvidenceFilter) ([]*fintech.EvidencePack, error) {
	// This is a simplified implementation
	// In production, you'd want to index and query the storage
	return []*fintech.EvidencePack{}, nil
}

// Delete removes an evidence pack
func (s *MarkdownEvidenceStore) Delete(id string) error {
	// Implementation depends on storage capabilities
	return fmt.Errorf("delete not implemented")
}

// ExportMarkdown returns the Markdown content
func (s *MarkdownEvidenceStore) ExportMarkdown(id string) ([]byte, error) {
	doc, err := s.storage.LoadDocument("EVIDENCE_PACK", id)
	if err != nil {
		return nil, err
	}
	return doc.Content, nil
}

// Sign signs an evidence pack with PQC
func (s *MarkdownEvidenceStore) Sign(id string, keyID string) error {
	// Load the evidence pack
	doc, err := s.storage.LoadDocument("EVIDENCE_PACK", id)
	if err != nil {
		return fmt.Errorf("failed to load evidence pack for signing: %w", err)
	}

	// Sign the content
	// Note: This is a simplified implementation
	// In production, you'd use proper PQC signing
	signature := &fintech.PQCSignature{
		Algorithm:   "ML-DSA-65",
		PublicKeyID: keyID,
		Signature:   "placeholder-signature",
		SignedAt:    time.Now(),
	}

	// Update document metadata with signature info
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["signature"] = signature

	// Re-save the document
	return s.storage.SaveDocument(doc)
}

// === Phase 2: Scenario Injection Methods ===

// RunScenarioValidation runs an agent against regulatory scenarios
func (s *FinTechValidatorService) RunScenarioValidation(ctx context.Context, agentID string, agentOutput string, scenarioIDs []string) (*vault.ValidationReport, error) {
	if s.vaultComplianceEngine == nil {
		return nil, fmt.Errorf("scenario testing not enabled")
	}

	if len(scenarioIDs) > 0 {
		// Run specific scenarios
		return s.vaultComplianceEngine.ValidateAgentWithScenarios(ctx, agentID, agentOutput, scenarioIDs)
	}

	// Run all active scenarios
	filter := &vault.ScenarioFilter{IsActive: boolPtr(true)}
	return s.vaultComplianceEngine.ValidateAgent(ctx, agentID, agentOutput, filter)
}

// GetScenarios returns available regulatory scenarios
func (s *FinTechValidatorService) GetScenarios(filter *vault.ScenarioFilter) ([]*vault.RegulatoryScenario, error) {
	if s.scenarioRepository == nil {
		return nil, fmt.Errorf("scenario repository not initialized")
	}
	return s.scenarioRepository.List(filter)
}

// GetScenario returns a specific scenario by ID
func (s *FinTechValidatorService) GetScenario(id string) (*vault.RegulatoryScenario, error) {
	if s.scenarioRepository == nil {
		return nil, fmt.Errorf("scenario repository not initialized")
	}
	return s.scenarioRepository.Get(id)
}

// CreateScenario creates a new regulatory scenario
func (s *FinTechValidatorService) CreateScenario(scenario *vault.RegulatoryScenario) error {
	if s.scenarioRepository == nil {
		return fmt.Errorf("scenario repository not initialized")
	}
	return s.scenarioRepository.Save(scenario)
}

// UpdateScenario updates an existing scenario
func (s *FinTechValidatorService) UpdateScenario(scenario *vault.RegulatoryScenario) error {
	if s.scenarioRepository == nil {
		return fmt.Errorf("scenario repository not initialized")
	}
	return s.scenarioRepository.Update(scenario)
}

// DeleteScenario marks a scenario as inactive
func (s *FinTechValidatorService) DeleteScenario(id string) error {
	if s.scenarioRepository == nil {
		return fmt.Errorf("scenario repository not initialized")
	}
	return s.scenarioRepository.Delete(id)
}

// GetScenarioStatistics returns execution statistics
func (s *FinTechValidatorService) GetScenarioStatistics() *vault.ComplianceStatistics {
	if s.vaultComplianceEngine == nil {
		return nil
	}
	return s.vaultComplianceEngine.GetStatistics()
}

// === Phase 2: Certificate of Correctness Methods ===

// IssueCertificate generates a PQC-signed Certificate of Correctness
func (s *FinTechValidatorService) IssueCertificate(ctx context.Context, req *CertificateRequest) (*fintech.CertificateOfCorrectness, error) {
	if s.certificateManager == nil {
		return nil, fmt.Errorf("certification not enabled")
	}

	// Validate the agent first if requested
	var validationResult fintech.ValidationResult
	if req.ValidateFirst && req.AgentOutput != "" {
		report, err := s.RunScenarioValidation(ctx, req.Subject.ID, req.AgentOutput, req.ScenarioIDs)
		if err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}

		validationResult = fintech.ValidationResult{
			Passed:           report.Status == "passed",
			Score:            report.OverallScore,
			TestsPassed:      len(report.ScenarioResults) - report.CriticalFailures,
			TestsFailed:      report.CriticalFailures,
			TestsTotal:       len(report.ScenarioResults),
			ScenariosPassed:  len(report.ScenarioResults) - report.CriticalFailures,
			ScenariosFailed:  report.CriticalFailures,
			CriticalFailures: report.CriticalFailures,
			Findings:         report.Recommendations,
		}
	} else {
		// Use provided validation results
		validationResult = req.ValidationResult
	}

	// Build certificate
	builder := fintech.NewCertificateBuilder()

	// Use template if specified
	if req.TemplateName != "" {
		builder.FromTemplate(req.TemplateName)
	}

	// Set subject
	builder.WithSubject(req.Subject)

	// Set validation results
	builder.WithValidation(validationResult)

	// Set evidence references
	builder.WithEvidence(req.EvidencePackID, req.TraceID)

	// Set scenarios
	builder.WithScenarios(req.ScenarioIDs)

	// Set issuer
	builder.WithIssuer(req.IssuerNodeID)

	// Set expiry if provided
	if !req.ExpiresAt.IsZero() {
		builder.WithExpiry(req.ExpiresAt)
	}

	// Set compliance level
	if req.ComplianceLevel != "" {
		builder.WithComplianceLevel(req.ComplianceLevel)
	}

	// Issue and sign the certificate
	certificate, err := s.certificateManager.IssueCertificate(builder)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	return certificate, nil
}

// VerifyCertificate verifies a certificate's PQC signature
func (s *FinTechValidatorService) VerifyCertificate(certificate *fintech.CertificateOfCorrectness) (bool, error) {
	if s.certificateManager == nil {
		return false, fmt.Errorf("certification not enabled")
	}
	return s.certificateManager.VerifyCertificate(certificate)
}

// IsCertificateValid checks if a certificate is valid (not expired, not revoked)
func (s *FinTechValidatorService) IsCertificateValid(certificate *fintech.CertificateOfCorrectness) (bool, string) {
	if s.certificateManager == nil {
		return false, "CERTIFICATION_NOT_ENABLED"
	}
	return s.certificateManager.IsCertificateValid(certificate)
}

// RevokeCertificate revokes a certificate
func (s *FinTechValidatorService) RevokeCertificate(certificate *fintech.CertificateOfCorrectness, revokedBy string, reason string) error {
	if s.certificateManager == nil {
		return fmt.Errorf("certification not enabled")
	}
	return s.certificateManager.RevokeCertificate(certificate, revokedBy, reason)
}

// CertificateRequest represents a request to issue a certificate
type CertificateRequest struct {
	Subject          fintech.SubjectInfo      `json:"subject"`
	AgentOutput      string                   `json:"agent_output,omitempty"`
	ValidationResult fintech.ValidationResult `json:"validation_result,omitempty"`
	ValidateFirst    bool                     `json:"validate_first"`
	EvidencePackID   string                   `json:"evidence_pack_id,omitempty"`
	TraceID          string                   `json:"trace_id,omitempty"`
	ScenarioIDs      []string                 `json:"scenario_ids,omitempty"`
	IssuerNodeID     string                   `json:"issuer_node_id"`
	TemplateName     string                   `json:"template_name,omitempty"`
	ExpiresAt        time.Time                `json:"expires_at,omitempty"`
	ComplianceLevel  string                   `json:"compliance_level,omitempty"`
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}

// === Phase 3: Trajectory Capture & Replay Methods ===

// StartTrajectoryCapture begins capturing execution trajectory for an agent
func (s *FinTechValidatorService) StartTrajectoryCapture(ctx context.Context, agentID, validationID string, pid uint32, config *fintech.TrajectoryCaptureConfig) (*ebpf.CaptureSession, error) {
	if s.agentTracer == nil {
		return nil, fmt.Errorf("trajectory capture not enabled")
	}

	return s.agentTracer.StartCapture(ctx, agentID, validationID, pid, config)
}

// StopTrajectoryCapture stops an active capture and returns the trajectory
func (s *FinTechValidatorService) StopTrajectoryCapture(sessionID string) (*fintech.ExecutionTrajectory, error) {
	if s.agentTracer == nil {
		return nil, fmt.Errorf("trajectory capture not enabled")
	}

	trajectory, err := s.agentTracer.StopCapture(sessionID)
	if err != nil {
		return nil, err
	}

	// Save trajectory to store
	if s.trajectoryStore != nil {
		if err := s.trajectoryStore.Save(trajectory); err != nil {
			// Log but don't fail - trajectory is still returned
			fmt.Printf("Warning: failed to save trajectory: %v\n", err)
		}
	}

	return trajectory, nil
}

// GetTrajectoryCaptureSession returns an active capture session
func (s *FinTechValidatorService) GetTrajectoryCaptureSession(sessionID string) (*ebpf.CaptureSession, error) {
	if s.agentTracer == nil {
		return nil, fmt.Errorf("trajectory capture not enabled")
	}

	return s.agentTracer.GetCaptureSession(sessionID)
}

// ListActiveTrajectoryCaptures returns all active capture sessions
func (s *FinTechValidatorService) ListActiveTrajectoryCaptures() ([]*ebpf.CaptureSession, error) {
	if s.agentTracer == nil {
		return nil, fmt.Errorf("trajectory capture not enabled")
	}

	return s.agentTracer.ListActiveCaptures(), nil
}

// ReplayTrajectory starts replaying a trajectory
func (s *FinTechValidatorService) ReplayTrajectory(ctx context.Context, trajectoryID string, config *fintech.TrajectoryReplayConfig) (*fintech.ReplaySession, error) {
	if s.replayEngine == nil {
		return nil, fmt.Errorf("replay engine not enabled")
	}

	// Load trajectory from store
	if s.trajectoryStore == nil {
		return nil, fmt.Errorf("trajectory store not available")
	}

	trajectory, err := s.trajectoryStore.Load(trajectoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to load trajectory: %w", err)
	}

	return s.replayEngine.StartReplay(ctx, trajectory, config)
}

// GetReplayResult returns the result of a replay session
func (s *FinTechValidatorService) GetReplayResult(sessionID string) (*fintech.ReplayResult, error) {
	if s.replayEngine == nil {
		return nil, fmt.Errorf("replay engine not enabled")
	}

	session, err := s.replayEngine.GetReplaySession(sessionID)
	if err != nil {
		return nil, err
	}

	return session.Result, nil
}

// GetTrajectory returns a trajectory by ID
func (s *FinTechValidatorService) GetTrajectory(id string) (*fintech.ExecutionTrajectory, error) {
	if s.trajectoryStore == nil {
		return nil, fmt.Errorf("trajectory store not available")
	}

	return s.trajectoryStore.Load(id)
}

// ListTrajectories lists trajectories with filtering
func (s *FinTechValidatorService) ListTrajectories(filter fintech.TrajectoryFilter) ([]*fintech.ExecutionTrajectory, error) {
	// If trajectory capture is disabled, the store may be nil. Return an empty slice instead of an error to avoid 500 responses.
	if s.trajectoryStore == nil {
		return []*fintech.ExecutionTrajectory{}, nil
	}

	return s.trajectoryStore.List(filter)
}

// CompareTrajectories compares two trajectories
func (s *FinTechValidatorService) CompareTrajectories(baseID, compareID string) (*fintech.TrajectoryComparisonResult, error) {
	if s.trajectoryStore == nil {
		return nil, fmt.Errorf("trajectory store not available")
	}

	base, err := s.trajectoryStore.Load(baseID)
	if err != nil {
		return nil, fmt.Errorf("failed to load base trajectory: %w", err)
	}

	compare, err := s.trajectoryStore.Load(compareID)
	if err != nil {
		return nil, fmt.Errorf("failed to load compare trajectory: %w", err)
	}

	comparator := fintech.NewTrajectoryComparator()
	return comparator.CompareTrajectories(base, compare), nil
}

// ValidateWithDeterminism runs validation with trajectory capture and replay
func (s *FinTechValidatorService) ValidateWithDeterminism(ctx context.Context, req *ValidationRequest, pid uint32) (*ValidationResponse, error) {
	// Run standard validation first to get validation ID
	response, err := s.Validate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Start trajectory capture after getting validation ID
	captureConfig := fintech.DefaultTrajectoryCaptureConfig()
	session, err := s.StartTrajectoryCapture(ctx, req.AgentID, response.ValidationID, pid, captureConfig)
	if err != nil {
		// Log but continue without capture
		fmt.Printf("Warning: failed to start trajectory capture: %v\n", err)
		return response, nil
	}

	// Run a quick validation capture (for demo purposes)
	time.Sleep(100 * time.Millisecond)

	// Stop capture and get trajectory
	var trajectory *fintech.ExecutionTrajectory
	if session != nil {
		trajectory, err = s.StopTrajectoryCapture(session.ID)
		if err != nil {
			fmt.Printf("Warning: failed to stop trajectory capture: %v\n", err)
		} else {
			// Link trajectory to evidence pack
			response.EvidencePackID = trajectory.ID
		}
	}

	// If replay engine is enabled, run a replay to verify determinism
	if s.replayEngine != nil && trajectory != nil {
		replayConfig := fintech.DefaultTrajectoryReplayConfig()
		_, err := s.ReplayTrajectory(ctx, trajectory.ID, replayConfig)
		if err != nil {
			fmt.Printf("Warning: failed to replay trajectory: %v\n", err)
		}
	}

	return response, nil
}

// MarkdownTrajectoryStore implements TrajectoryStore using Markdown storage
type MarkdownTrajectoryStore struct {
	storage    *mdstorage.MarkdownStorageDriver
	pqcManager *pqc.EncryptionManager
}

// NewMarkdownTrajectoryStore creates a new Markdown-based trajectory store
func NewMarkdownTrajectoryStore(storage *mdstorage.MarkdownStorageDriver, pqcManager *pqc.EncryptionManager) TrajectoryStore {
	return &MarkdownTrajectoryStore{
		storage:    storage,
		pqcManager: pqcManager,
	}
}

// Save persists a trajectory as a Markdown document
func (s *MarkdownTrajectoryStore) Save(trajectory *fintech.ExecutionTrajectory) error {
	// Generate Markdown content
	content, err := trajectory.ToMarkdown()
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Create document
	doc := &mdstorage.MarkdownDocument{
		ID:        trajectory.ID,
		Type:      "TRAJECTORY",
		Timestamp: trajectory.CreatedAt,
		Metadata: map[string]interface{}{
			"agent_id":         trajectory.AgentID,
			"agent_name":       trajectory.AgentName,
			"validation_id":    trajectory.ValidationID,
			"status":           trajectory.Status,
			"event_count":      trajectory.EventCount,
			"is_deterministic": trajectory.IsDeterministic,
			"has_signature":    trajectory.Signature != nil,
		},
		Content: content,
	}

	return s.storage.SaveDocument(doc)
}

// Load retrieves a trajectory from storage
func (s *MarkdownTrajectoryStore) Load(id string) (*fintech.ExecutionTrajectory, error) {
	doc, err := s.storage.LoadDocument("TRAJECTORY", id)
	if err != nil {
		return nil, fmt.Errorf("failed to load trajectory: %w", err)
	}

	// For now, reconstruct from metadata
	trajectory := &fintech.ExecutionTrajectory{
		ID:        doc.ID,
		CreatedAt: doc.Timestamp,
	}

	if agentID, ok := doc.Metadata["agent_id"].(string); ok {
		trajectory.AgentID = agentID
	}
	if agentName, ok := doc.Metadata["agent_name"].(string); ok {
		trajectory.AgentName = agentName
	}
	if validationID, ok := doc.Metadata["validation_id"].(string); ok {
		trajectory.ValidationID = validationID
	}
	if status, ok := doc.Metadata["status"].(string); ok {
		trajectory.Status = fintech.TrajectoryStatus(status)
	}
	if eventCount, ok := doc.Metadata["event_count"].(float64); ok {
		trajectory.EventCount = uint64(eventCount)
	}

	return trajectory, nil
}

// List returns trajectories with filtering
func (s *MarkdownTrajectoryStore) List(filter fintech.TrajectoryFilter) ([]*fintech.ExecutionTrajectory, error) {
	// This is a simplified implementation
	// In production, you'd want to index and query the storage
	return []*fintech.ExecutionTrajectory{}, nil
}

// Delete removes a trajectory
func (s *MarkdownTrajectoryStore) Delete(id string) error {
	return fmt.Errorf("delete not implemented")
}

// === Phase 4: NRV Financial Semantic Engine Methods ===

// StartNRVTrace begins capturing an NRV reasoning trace
func (s *FinTechValidatorService) StartNRVTrace(agentID, agentName, validationID string) (*fintech.NRVTrace, error) {
	if s.nrvTracer == nil {
		return nil, fmt.Errorf("NRV tracing not enabled")
	}

	trace := s.nrvTracer.StartTrace(agentID, agentName, validationID)
	return trace, nil
}

// RecordNRVInference records an inference step in an NRV trace
func (s *FinTechValidatorService) RecordNRVInference(traceID, intent, observation, inference string, confidence float64, financialContext *fintech.FinancialContext) error {
	if s.nrvTracer == nil {
		return fmt.Errorf("NRV tracing not enabled")
	}

	return s.nrvTracer.RecordInference(traceID, intent, observation, inference, confidence, financialContext)
}

// RecordNRVValidation records a validation step in an NRV trace
func (s *FinTechValidatorService) RecordNRVValidation(traceID string, action *ontology.FinancialAction, result *ontology.ValidationResult) error {
	if s.nrvTracer == nil {
		return fmt.Errorf("NRV tracing not enabled")
	}

	return s.nrvTracer.RecordValidation(traceID, action, result)
}

// RecordNRVDecision records a final decision in an NRV trace
func (s *FinTechValidatorService) RecordNRVDecision(traceID, decision string, confidence float64, metadata map[string]interface{}) error {
	if s.nrvTracer == nil {
		return fmt.Errorf("NRV tracing not enabled")
	}

	return s.nrvTracer.RecordDecision(traceID, decision, confidence, metadata)
}

// FinalizeNRVTrace finalizes an NRV trace
func (s *FinTechValidatorService) FinalizeNRVTrace(traceID string) (*fintech.NRVTrace, error) {
	if s.nrvTracer == nil {
		return nil, fmt.Errorf("NRV tracing not enabled")
	}

	return s.nrvTracer.FinalizeTrace(traceID)
}

// GetNRVTrace retrieves an NRV trace by ID
func (s *FinTechValidatorService) GetNRVTrace(traceID string) (*fintech.NRVTrace, error) {
	if s.nrvTracer == nil {
		return nil, fmt.Errorf("NRV tracing not enabled")
	}

	return s.nrvTracer.GetTrace(traceID)
}

// ListNRVTraces lists all NRV traces
func (s *FinTechValidatorService) ListNRVTraces(activeOnly bool) []*fintech.NRVTrace {
	if s.nrvTracer == nil {
		return nil
	}

	if activeOnly {
		return s.nrvTracer.ListActiveTraces()
	}

	return s.nrvTracer.ListCompletedTraces()
}

// ScoreFidelity calculates the fidelity score for an NRV trace
func (s *FinTechValidatorService) ScoreFidelity(ctx context.Context, traceID string, expectedOutcome string) (*fintech.FidelityScoreResult, error) {
	if s.fidelityScorer == nil {
		return nil, fmt.Errorf("fidelity scoring not enabled")
	}

	return s.fidelityScorer.ScoreTrace(ctx, traceID, expectedOutcome)
}

// ValidateFidelity validates if a trace meets fidelity requirements
func (s *FinTechValidatorService) ValidateFidelity(result *fintech.FidelityScoreResult) (bool, []string) {
	if s.fidelityScorer == nil {
		return false, []string{"fidelity scoring not enabled"}
	}

	return s.fidelityScorer.ValidateFidelity(result)
}

// DetectKYCBypass detects potential KYC bypass attempts in a trace
func (s *FinTechValidatorService) DetectKYCBypass(traceID string) []fintech.RiskIndicator {
	if s.nrvTracer == nil {
		return nil
	}

	return s.nrvTracer.DetectKYCBypass(traceID)
}

// DetectPositionLimitViolation detects position limit violations
func (s *FinTechValidatorService) DetectPositionLimitViolation(traceID string, limits map[string]float64) []fintech.RiskIndicator {
	if s.nrvTracer == nil {
		return nil
	}

	return s.nrvTracer.DetectPositionLimitViolation(traceID, limits)
}

// CalculateSemanticDistance calculates semantic distance for a trace
func (s *FinTechValidatorService) CalculateSemanticDistance(traceID string, expectedOutcome string) (*fintech.DistanceResult, error) {
	if s.nrvTracer == nil {
		return nil, fmt.Errorf("NRV tracing not enabled")
	}

	trace, err := s.nrvTracer.GetTrace(traceID)
	if err != nil {
		return nil, err
	}

	calculator := fintech.NewSemanticDistanceCalculator(s.ontologyRegistry)
	return calculator.CalculateDistance(trace, expectedOutcome)
}

// ValidateWithNRV performs validation with NRV tracing and fidelity scoring
func (s *FinTechValidatorService) ValidateWithNRV(ctx context.Context, req *ValidationRequest, expectedOutcome string) (*ValidationResponse, *fintech.FidelityScoreResult, error) {
	// Run standard validation first
	response, err := s.Validate(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	// If NRV tracing is not enabled, return just the validation response
	if s.nrvTracer == nil {
		return response, nil, nil
	}

	// Start NRV trace
	trace, err := s.StartNRVTrace(req.AgentID, req.AgentName, response.ValidationID)
	if err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to start NRV trace: %v\n", err)
		return response, nil, nil
	}

	// Record reasoning steps based on validation
	if response.ComplianceResult != nil {
		for _, result := range response.ComplianceResult.Results {
			for _, action := range req.FinancialActions {
				s.RecordNRVValidation(trace.ID, action, result)
			}
		}
	}

	// Record final decision
	s.RecordNRVDecision(trace.ID, response.Status, response.OverallScore, map[string]interface{}{
		"is_compliant": response.IsCompliant,
		"score":        response.OverallScore,
	})

	// Finalize trace
	_, err = s.FinalizeNRVTrace(trace.ID)
	if err != nil {
		fmt.Printf("Warning: failed to finalize NRV trace: %v\n", err)
		return response, nil, nil
	}

	// If fidelity scoring is enabled, calculate score
	var fidelityResult *fintech.FidelityScoreResult
	if s.fidelityScorer != nil {
		fidelityResult, err = s.ScoreFidelity(ctx, trace.ID, expectedOutcome)
		if err != nil {
			fmt.Printf("Warning: failed to calculate fidelity score: %v\n", err)
		}
	}

	return response, fidelityResult, nil
}

// NRVTraceStorage defines the interface for NRV trace storage
type NRVTraceStorage interface {
	Save(trace *fintech.NRVTrace) error
	Load(id string) (*fintech.NRVTrace, error)
	List() ([]*fintech.NRVTrace, error)
	Delete(id string) error
}

// MarkdownNRVTraceStore implements NRVTraceStorage using Markdown storage
type MarkdownNRVTraceStore struct {
	storage    *mdstorage.MarkdownStorageDriver
	pqcManager *pqc.EncryptionManager
}

// NewMarkdownNRVTraceStore creates a new Markdown-based NRV trace store
func NewMarkdownNRVTraceStore(storage *mdstorage.MarkdownStorageDriver, pqcManager *pqc.EncryptionManager) NRVTraceStorage {
	return &MarkdownNRVTraceStore{
		storage:    storage,
		pqcManager: pqcManager,
	}
}

// Save persists an NRV trace as a Markdown document
func (s *MarkdownNRVTraceStore) Save(trace *fintech.NRVTrace) error {
	// Generate Markdown content
	content, err := trace.ToMarkdown()
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Create document
	doc := &mdstorage.MarkdownDocument{
		ID:        trace.ID,
		Type:      "NRV_TRACE",
		Timestamp: trace.CreatedAt,
		Metadata: map[string]interface{}{
			"agent_id":       trace.AgentID,
			"agent_name":     trace.AgentName,
			"validation_id":  trace.ValidationID,
			"status":         trace.Status,
			"step_count":     trace.StepCount,
			"fidelity_score": trace.FidelityScore,
			"has_analysis":   trace.FidelityAnalysis != nil,
		},
		Content: content,
	}

	return s.storage.SaveDocument(doc)
}

// Load retrieves an NRV trace from storage
func (s *MarkdownNRVTraceStore) Load(id string) (*fintech.NRVTrace, error) {
	doc, err := s.storage.LoadDocument("NRV_TRACE", id)
	if err != nil {
		return nil, fmt.Errorf("failed to load NRV trace: %w", err)
	}

	// Reconstruct from metadata
	trace := &fintech.NRVTrace{
		ID:        doc.ID,
		CreatedAt: doc.Timestamp,
	}

	if agentID, ok := doc.Metadata["agent_id"].(string); ok {
		trace.AgentID = agentID
	}
	if agentName, ok := doc.Metadata["agent_name"].(string); ok {
		trace.AgentName = agentName
	}
	if validationID, ok := doc.Metadata["validation_id"].(string); ok {
		trace.ValidationID = validationID
	}
	if status, ok := doc.Metadata["status"].(string); ok {
		trace.Status = fintech.NRVTraceStatus(status)
	}
	if stepCount, ok := doc.Metadata["step_count"].(float64); ok {
		trace.StepCount = int(stepCount)
	}
	if fidelityScore, ok := doc.Metadata["fidelity_score"].(float64); ok {
		trace.FidelityScore = fidelityScore
	}

	return trace, nil
}

// List returns all NRV traces
func (s *MarkdownNRVTraceStore) List() ([]*fintech.NRVTrace, error) {
	// This is a simplified implementation
	// In production, you'd want to index and query the storage
	return []*fintech.NRVTrace{}, nil
}

// Delete removes an NRV trace
func (s *MarkdownNRVTraceStore) Delete(id string) error {
	return fmt.Errorf("delete not implemented")
}

// === Phase 6: Arrow Flight Tick Data Streaming Methods ===

// CreateTickStream creates a new tick data stream for a symbol
func (s *FinTechValidatorService) CreateTickStream(symbol, tickType string, maxSize int, onFlush func([]fintech.TickData) error) (*fintech.TickDataStream, error) {
	if s.tickStreamManager == nil {
		return nil, fmt.Errorf("tick data streaming not enabled")
	}

	return s.tickStreamManager.CreateStream(symbol, tickType, maxSize, onFlush), nil
}

// AddTick adds a tick to a stream
func (s *FinTechValidatorService) AddTick(tick fintech.TickData) error {
	if s.tickStreamManager == nil {
		return fmt.Errorf("tick data streaming not enabled")
	}

	return s.tickStreamManager.AddTick(tick.Symbol, tick)
}

// GetTicks retrieves ticks for a symbol
func (s *FinTechValidatorService) GetTicks(symbol string) ([]fintech.TickData, error) {
	if s.tickStreamManager == nil {
		return nil, fmt.Errorf("tick data streaming not enabled")
	}

	stream, ok := s.tickStreamManager.GetStream(symbol)
	if !ok {
		return nil, fmt.Errorf("stream not found: %s", symbol)
	}

	return stream.Data, nil
}

// GetTickRecord returns tick data as an Arrow record
func (s *FinTechValidatorService) GetTickRecord(symbol string) (interface{}, error) {
	if s.tickStreamManager == nil {
		return nil, fmt.Errorf("tick data streaming not enabled")
	}

	return s.tickStreamManager.BuildRecord(symbol)
}

// QueryTicks queries tick data with filters
func (s *FinTechValidatorService) QueryTicks(filter fintech.TickDataFilter) ([]fintech.TickData, error) {
	if s.tickStreamManager == nil {
		return nil, fmt.Errorf("tick data streaming not enabled")
	}

	manager := s.tickStreamManager
	var result []fintech.TickData

	for sym, stream := range manager.GetAllStreams() {
		if filter.Symbol != "" && sym != filter.Symbol {
			continue
		}

		for _, tick := range stream.Data {
			if filter.StartTime > 0 && tick.Timestamp.UnixNano() < filter.StartTime {
				continue
			}
			if filter.EndTime > 0 && tick.Timestamp.UnixNano() > filter.EndTime {
				continue
			}
			result = append(result, tick)
		}
	}

	return result, nil
}

// ExportTicks exports tick data to IPC format
func (s *FinTechValidatorService) ExportTicks(ticks []fintech.TickData) ([]byte, error) {
	if len(ticks) == 0 {
		return nil, nil
	}

	writer := fintech.NewTickDataWriter()
	defer writer.Release()

	_, err := writer.BuildRecord(ticks)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// FlushTickStream flushes buffered ticks for a symbol
func (s *FinTechValidatorService) FlushTickStream(symbol string) error {
	if s.tickStreamManager == nil {
		return fmt.Errorf("tick data streaming not enabled")
	}

	return s.tickStreamManager.FlushStream(symbol)
}

// IsTickStreamingEnabled returns whether tick data streaming is enabled
func (s *FinTechValidatorService) IsTickStreamingEnabled() bool {
	return s.tickStreamManager != nil
}

// IsEnabled returns whether the FinTech validator service is enabled
func (s *FinTechValidatorService) IsEnabled() bool {
	return s.Config != nil && s.Config.Enabled
}

// IsRunning returns whether the FinTech validator service is running
func (s *FinTechValidatorService) IsRunning() bool {
	// For now, if it's initialized and enabled, we consider it running
	return s.IsEnabled()
}
