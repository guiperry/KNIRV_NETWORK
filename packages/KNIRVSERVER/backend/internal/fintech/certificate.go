package fintech

import (
	"encoding/json"
	"fmt"
	"time"

	"backend_server/internal/storage/pqc"
)

// CertificateOfCorrectness is a PQC-signed certificate proving regulatory compliance
type CertificateOfCorrectness struct {
	// Certificate metadata
	ID           string    `json:"id"`
	Version      string    `json:"version"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	IssuerNodeID string    `json:"issuer_node_id"`

	// Subject (what was validated)
	Subject SubjectInfo `json:"subject"`

	// Validation results
	Validation ValidationResult `json:"validation"`

	// Evidence references
	EvidencePackID string   `json:"evidence_pack_id"`
	TraceID        string   `json:"trace_id,omitempty"`
	ScenarioIDs    []string `json:"scenario_ids"`

	// Compliance status
	OverallScore    float64 `json:"overall_score"`
	Status          string  `json:"status"`           // COMPLIANT, NON_COMPLIANT, PROVISIONAL
	ComplianceLevel string  `json:"compliance_level"` // FULL, PARTIAL, MINIMAL

	// Regulatory scope
	ApplicableRegulations []string `json:"applicable_regulations"`
	RegulatoryFramework   string   `json:"regulatory_framework"` // AML, KYC, SEC, BaselIII, etc.

	// PQC Signature
	Signature *PQCSignature `json:"signature"`

	// Revocation info
	Revocable    bool       `json:"revocable"`
	Revoked      bool       `json:"revoked"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokedBy    string     `json:"revoked_by,omitempty"`
	RevokeReason string     `json:"revoke_reason,omitempty"`
}

// SubjectInfo describes what was validated
type SubjectInfo struct {
	Type        string `json:"type"` // AGENT, MODEL, ALGORITHM, SYSTEM
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Hash        string `json:"hash,omitempty"` // Cryptographic hash of subject
}

// ValidationResult contains the validation outcome
type ValidationResult struct {
	Passed           bool     `json:"passed"`
	Score            float64  `json:"score"`
	TestsPassed      int      `json:"tests_passed"`
	TestsFailed      int      `json:"tests_failed"`
	TestsTotal       int      `json:"tests_total"`
	ScenariosPassed  int      `json:"scenarios_passed"`
	ScenariosFailed  int      `json:"scenarios_failed"`
	CriticalFailures int      `json:"critical_failures"`
	Findings         []string `json:"findings,omitempty"`
}

// CertificateTemplate defines parameters for certificate generation
type CertificateTemplate struct {
	Name                  string
	Description           string
	RegulatoryFramework   string
	ApplicableRegulations []string
	ValidDuration         time.Duration
	ComplianceLevel       string
	Revocable             bool
}

// DefaultTemplates provides standard certificate templates
var DefaultTemplates = map[string]*CertificateTemplate{
	"fintech_basic": {
		Name:                  "Basic FinTech Compliance",
		Description:           "Basic regulatory compliance for financial AI agents",
		RegulatoryFramework:   "Multi-Regulation",
		ApplicableRegulations: []string{"AML", "KYC"},
		ValidDuration:         30 * 24 * time.Hour, // 30 days
		ComplianceLevel:       "MINIMAL",
		Revocable:             true,
	},
	"fintech_standard": {
		Name:                  "Standard FinTech Compliance",
		Description:           "Standard regulatory compliance for financial AI agents",
		RegulatoryFramework:   "Multi-Regulation",
		ApplicableRegulations: []string{"AML", "KYC", "SEC"},
		ValidDuration:         90 * 24 * time.Hour, // 90 days
		ComplianceLevel:       "PARTIAL",
		Revocable:             true,
	},
	"fintech_full": {
		Name:                  "Full FinTech Compliance",
		Description:           "Comprehensive regulatory compliance for financial AI agents",
		RegulatoryFramework:   "Multi-Regulation",
		ApplicableRegulations: []string{"AML", "KYC", "SEC", "BaselIII"},
		ValidDuration:         365 * 24 * time.Hour, // 1 year
		ComplianceLevel:       "FULL",
		Revocable:             true,
	},
	"banking_basel": {
		Name:                  "Basel III Compliance",
		Description:           "Basel III capital adequacy compliance certificate",
		RegulatoryFramework:   "BaselIII",
		ApplicableRegulations: []string{"BaselIII"},
		ValidDuration:         180 * 24 * time.Hour, // 180 days
		ComplianceLevel:       "FULL",
		Revocable:             true,
	},
}

// CertificateBuilder builds certificates with a fluent interface
type CertificateBuilder struct {
	cert *CertificateOfCorrectness
}

// NewCertificateBuilder creates a new certificate builder
func NewCertificateBuilder() *CertificateBuilder {
	return &CertificateBuilder{
		cert: &CertificateOfCorrectness{
			ID:        generateCertID(),
			Version:   "1.0",
			IssuedAt:  time.Now(),
			Status:    "PROVISIONAL",
			Revocable: true,
		},
	}
}

// FromTemplate initializes the builder from a template
func (b *CertificateBuilder) FromTemplate(templateName string) *CertificateBuilder {
	template, ok := DefaultTemplates[templateName]
	if !ok {
		return b
	}

	b.cert.RegulatoryFramework = template.RegulatoryFramework
	b.cert.ApplicableRegulations = template.ApplicableRegulations
	b.cert.ComplianceLevel = template.ComplianceLevel
	b.cert.Revocable = template.Revocable
	b.cert.ExpiresAt = b.cert.IssuedAt.Add(template.ValidDuration)

	return b
}

// WithSubject sets the subject information
func (b *CertificateBuilder) WithSubject(subject SubjectInfo) *CertificateBuilder {
	b.cert.Subject = subject
	return b
}

// WithValidation sets the validation results
func (b *CertificateBuilder) WithValidation(validation ValidationResult) *CertificateBuilder {
	b.cert.Validation = validation

	// Determine status from validation
	if validation.Passed && validation.Score >= 80 {
		b.cert.Status = "COMPLIANT"
	} else if validation.Passed && validation.Score >= 60 {
		b.cert.Status = "PROVISIONAL"
	} else {
		b.cert.Status = "NON_COMPLIANT"
	}

	b.cert.OverallScore = validation.Score
	return b
}

// WithEvidence sets evidence references
func (b *CertificateBuilder) WithEvidence(evidencePackID, traceID string) *CertificateBuilder {
	b.cert.EvidencePackID = evidencePackID
	b.cert.TraceID = traceID
	return b
}

// WithScenarios sets the scenario IDs
func (b *CertificateBuilder) WithScenarios(scenarioIDs []string) *CertificateBuilder {
	b.cert.ScenarioIDs = scenarioIDs
	return b
}

// WithIssuer sets the issuer information
func (b *CertificateBuilder) WithIssuer(nodeID string) *CertificateBuilder {
	b.cert.IssuerNodeID = nodeID
	return b
}

// WithExpiry sets the expiration date
func (b *CertificateBuilder) WithExpiry(expiresAt time.Time) *CertificateBuilder {
	b.cert.ExpiresAt = expiresAt
	return b
}

// WithComplianceLevel sets the compliance level
func (b *CertificateBuilder) WithComplianceLevel(level string) *CertificateBuilder {
	b.cert.ComplianceLevel = level
	return b
}

// Build returns the constructed certificate
func (b *CertificateBuilder) Build() *CertificateOfCorrectness {
	return b.cert
}

// CertificateManager manages certificate lifecycle and PQC signing
type CertificateManager struct {
	pqcManager *pqc.EncryptionManager
	keyID      string
	templates  map[string]*CertificateTemplate
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(pqcManager *pqc.EncryptionManager, keyID string) *CertificateManager {
	return &CertificateManager{
		pqcManager: pqcManager,
		keyID:      keyID,
		templates:  DefaultTemplates,
	}
}

// IssueCertificate creates and signs a new certificate
func (cm *CertificateManager) IssueCertificate(
	builder *CertificateBuilder,
) (*CertificateOfCorrectness, error) {
	cert := builder.Build()

	// Validate certificate
	if err := cm.validateCertificate(cert); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	// Sign the certificate
	if err := cm.signCertificate(cert); err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	return cert, nil
}

// signCertificate creates a PQC signature for the certificate
func (cm *CertificateManager) signCertificate(cert *CertificateOfCorrectness) error {
	// Get the signing key
	keyPair, err := cm.pqcManager.GetKey(cm.keyID)
	if err != nil {
		return fmt.Errorf("signing key not found: %w", err)
	}

	// Create canonical JSON representation of certificate (excluding signature)
	certCopy := *cert
	certCopy.Signature = nil

	data, err := json.Marshal(certCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal certificate: %w", err)
	}

	// Sign with Dilithium
	signature, err := keyPair.Sign(data)
	if err != nil {
		return fmt.Errorf("signing failed: %w", err)
	}

	// Add signature to certificate
	cert.Signature = &PQCSignature{
		Algorithm:   "Dilithium-3",
		PublicKeyID: cm.keyID,
		Signature:   string(signature),
		SignedAt:    time.Now(),
		SignedData:  data,
	}

	return nil
}

// VerifyCertificate verifies the PQC signature of a certificate
func (cm *CertificateManager) VerifyCertificate(cert *CertificateOfCorrectness) (bool, error) {
	if cert.Signature == nil {
		return false, fmt.Errorf("certificate has no signature")
	}

	// Get the verification key
	keyPair, err := cm.pqcManager.GetKey(cert.Signature.PublicKeyID)
	if err != nil {
		return false, fmt.Errorf("verification key not found: %w", err)
	}

	// Create canonical representation (excluding signature)
	certCopy := *cert
	certCopy.Signature = nil

	data, err := json.Marshal(certCopy)
	if err != nil {
		return false, fmt.Errorf("failed to marshal certificate: %w", err)
	}

	// Verify signature
	valid := keyPair.Verify(data, []byte(cert.Signature.Signature))
	return valid, nil
}

// validateCertificate checks if certificate is valid
func (cm *CertificateManager) validateCertificate(cert *CertificateOfCorrectness) error {
	if cert.Subject.ID == "" {
		return fmt.Errorf("subject ID is required")
	}
	if cert.IssuerNodeID == "" {
		return fmt.Errorf("issuer node ID is required")
	}
	if cert.ExpiresAt.IsZero() {
		return fmt.Errorf("expiration date is required")
	}
	if cert.ExpiresAt.Before(cert.IssuedAt) {
		return fmt.Errorf("expiration date must be after issue date")
	}
	return nil
}

// IsCertificateValid checks if a certificate is currently valid (not expired or revoked)
func (cm *CertificateManager) IsCertificateValid(cert *CertificateOfCorrectness) (bool, string) {
	// Check if revoked
	if cert.Revoked {
		return false, "REVOKED"
	}

	// Check expiration
	if time.Now().After(cert.ExpiresAt) {
		return false, "EXPIRED"
	}

	// Check status
	if cert.Status == "NON_COMPLIANT" {
		return false, "NON_COMPLIANT"
	}

	return true, "VALID"
}

// RevokeCertificate revokes a certificate
func (cm *CertificateManager) RevokeCertificate(
	cert *CertificateOfCorrectness,
	revokedBy string,
	reason string,
) error {
	if !cert.Revocable {
		return fmt.Errorf("certificate is not revocable")
	}

	now := time.Now()
	cert.Revoked = true
	cert.RevokedAt = &now
	cert.RevokedBy = revokedBy
	cert.RevokeReason = reason

	return nil
}

// Export exports certificate to JSON
func (cert *CertificateOfCorrectness) Export() ([]byte, error) {
	return json.MarshalIndent(cert, "", "  ")
}

// ExportCompact exports certificate to compact JSON (for storage/transfer)
func (cert *CertificateOfCorrectness) ExportCompact() ([]byte, error) {
	return json.Marshal(cert)
}

// ImportCertificate imports a certificate from JSON
func ImportCertificate(data []byte) (*CertificateOfCorrectness, error) {
	var cert CertificateOfCorrectness
	if err := json.Unmarshal(data, &cert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certificate: %w", err)
	}
	return &cert, nil
}

// ToMarkdown exports certificate as Markdown document
func (cert *CertificateOfCorrectness) ToMarkdown() string {
	md := fmt.Sprintf("# Certificate of Correctness\n\n")
	md += fmt.Sprintf("**Certificate ID:** %s  \n", cert.ID)
	md += fmt.Sprintf("**Version:** %s  \n", cert.Version)
	md += fmt.Sprintf("**Status:** %s  \n", cert.Status)
	md += fmt.Sprintf("**Compliance Level:** %s  \n\n", cert.ComplianceLevel)

	md += "## Subject\n\n"
	md += fmt.Sprintf("- **Type:** %s\n", cert.Subject.Type)
	md += fmt.Sprintf("- **ID:** %s\n", cert.Subject.ID)
	md += fmt.Sprintf("- **Name:** %s\n", cert.Subject.Name)
	md += fmt.Sprintf("- **Version:** %s\n\n", cert.Subject.Version)

	md += "## Validation Results\n\n"
	md += fmt.Sprintf("- **Overall Score:** %.2f%%\n", cert.OverallScore)
	md += fmt.Sprintf("- **Passed:** %v\n", cert.Validation.Passed)
	md += fmt.Sprintf("- **Tests:** %d/%d passed (%d failed)\n",
		cert.Validation.TestsPassed, cert.Validation.TestsTotal, cert.Validation.TestsFailed)
	md += fmt.Sprintf("- **Scenarios:** %d passed, %d failed\n\n",
		cert.Validation.ScenariosPassed, cert.Validation.ScenariosFailed)

	md += "## Regulatory Scope\n\n"
	md += fmt.Sprintf("- **Framework:** %s\n", cert.RegulatoryFramework)
	md += fmt.Sprintf("- **Regulations:** %s\n", formatStringSlice(cert.ApplicableRegulations))
	md += fmt.Sprintf("- **Evidence Pack:** %s\n\n", cert.EvidencePackID)

	md += "## Validity\n\n"
	md += fmt.Sprintf("- **Issued:** %s\n", cert.IssuedAt.Format(time.RFC3339))
	md += fmt.Sprintf("- **Expires:** %s\n", cert.ExpiresAt.Format(time.RFC3339))
	md += fmt.Sprintf("- **Issuer:** %s\n\n", cert.IssuerNodeID)

	if cert.Signature != nil {
		md += "## PQC Signature\n\n"
		md += fmt.Sprintf("- **Algorithm:** %s\n", cert.Signature.Algorithm)
		md += fmt.Sprintf("- **Public Key ID:** %s\n", cert.Signature.PublicKeyID)
		md += fmt.Sprintf("- **Signed At:** %s\n\n", cert.Signature.SignedAt.Format(time.RFC3339))
	}

	if cert.Revoked {
		md += "## ⚠️ REVOKED\n\n"
		md += fmt.Sprintf("- **Revoked At:** %s\n", cert.RevokedAt.Format(time.RFC3339))
		md += fmt.Sprintf("- **Revoked By:** %s\n", cert.RevokedBy)
		md += fmt.Sprintf("- **Reason:** %s\n\n", cert.RevokeReason)
	}

	return md
}

// Helper functions
func generateCertID() string {
	return fmt.Sprintf("cert-%d-%s", time.Now().Unix(), randomCertString(8))
}

func randomCertString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func formatStringSlice(slice []string) string {
	if len(slice) == 0 {
		return "None"
	}
	result := ""
	for i, s := range slice {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
