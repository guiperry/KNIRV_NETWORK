package fintech_validator

import (
	"testing"
	"time"

	"backend_server/internal/config"
	"backend_server/internal/fintech"
	"backend_server/internal/fintech/ontology"
	"backend_server/internal/services/p2p"
	"backend_server/internal/services/validation"
	"backend_server/internal/storage/mdstorage"
	"backend_server/internal/storage/pqc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/buntdb"
)

func TestNewFinTechValidatorService(t *testing.T) {
	// Setup test dependencies
	db, err := buntdb.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	p2pManager := &p2p.DVEP2PManager{}
	cfg := &config.Config{}

	validationCore, err := validation.NewValidationCore(db, p2pManager, cfg, nil)
	require.NoError(t, err)

	pqcManager := pqc.NewEncryptionManager()

	// Create temporary directory for storage
	storageDir := t.TempDir()
	masterKey, _ := pqc.GeneratePQCKeyPair("test-master", "master")
	pqcManager.SetMasterKey(masterKey)
	pqcManager.CacheKey(masterKey.ID, masterKey)

	mdStorage, err := mdstorage.NewMarkdownStorageDriver(storageDir, pqcManager, masterKey.ID)
	require.NoError(t, err)

	// Test service creation
	serviceConfig := &Config{
		EnableAMLChecks:       true,
		EnableKYCCheks:        true,
		EnableSECCheks:        true,
		EnableBaselCheks:      true,
		AutoSignEvidencePacks: false,
		MasterKeyID:           masterKey.ID,
	}

	service, err := NewFinTechValidatorService(validationCore, pqcManager, mdStorage, serviceConfig)
	require.NoError(t, err)
	assert.NotNil(t, service)

	// Test that ontologies are loaded
	ontologies := service.GetOntologies()
	assert.GreaterOrEqual(t, len(ontologies), 4, "Should have at least 4 ontologies loaded")
}

func TestEvidencePackCreation(t *testing.T) {
	// Test evidence pack creation
	pack := fintech.NewEvidencePack(
		fintech.EvidenceTypeValidation,
		"agent-001",
		"validation-001",
	)

	assert.NotEmpty(t, pack.ID)
	assert.Equal(t, fintech.EvidenceTypeValidation, pack.Type)
	assert.Equal(t, "agent-001", pack.AgentID)
	assert.Equal(t, "validation-001", pack.ValidationID)
	assert.Equal(t, fintech.EvidenceStatusPending, pack.Status)
	assert.NotNil(t, pack.Metadata)
	assert.NotNil(t, pack.ComplianceChecks)
}

func TestEvidencePackMarkdownGeneration(t *testing.T) {
	pack := fintech.NewEvidencePack(
		fintech.EvidenceTypeCompliance,
		"agent-002",
		"validation-002",
	)
	pack.AgentName = "Test Trading Bot"

	// Add validation result
	pack.ValidationResult = &fintech.ValidationEvidence{
		ValidatorID:   "test-validator",
		ValidatorType: "comprehensive",
		Status:        "valid",
		Confidence:    0.95,
		Score:         0.92,
		ExecutedAt:    time.Now(),
		Duration:      150,
	}

	// Add compliance check
	pack.AddComplianceCheck(fintech.ComplianceEvidence{
		RegulationID:   "aml-suspicious-amount",
		RegulationName: "Suspicious Transaction Amount",
		Category:       "AML",
		Status:         "compliant",
		Severity:       "high",
		Description:    "Transaction amount within normal range",
		CheckedAt:      time.Now(),
	})

	pack.MarkComplete()

	// Generate Markdown
	md, err := pack.GenerateMarkdown()
	require.NoError(t, err)
	assert.NotNil(t, md)

	// Check that Markdown contains expected sections
	mdStr := string(md)
	assert.Contains(t, mdStr, "# Evidence Pack:")
	assert.Contains(t, mdStr, "Test Trading Bot")
	assert.Contains(t, mdStr, "## Validation Result")
	assert.Contains(t, mdStr, "## Compliance Checks")
	assert.Contains(t, mdStr, "Suspicious Transaction Amount")
}

func TestOntologyRegistry(t *testing.T) {
	registry := ontology.NewOntologyRegistry()
	assert.NotNil(t, registry)

	// Register ontologies
	amlOntology := ontology.NewAMLOntology()
	err := registry.Register(amlOntology)
	require.NoError(t, err)

	kycOntology := ontology.NewKYCOntology()
	err = registry.Register(kycOntology)
	require.NoError(t, err)

	// Test Get
	retrieved, err := registry.Get(amlOntology.GetID())
	require.NoError(t, err)
	assert.Equal(t, amlOntology.GetID(), retrieved.GetID())

	// Test List
	ontologies := registry.List()
	assert.Len(t, ontologies, 2)

	// Test ListByCategory
	amlOntologies := registry.ListByCategory(ontology.CategoryAML)
	assert.Len(t, amlOntologies, 1)
}

func TestAMLRuleEvaluation(t *testing.T) {
	amlOntology := ontology.NewAMLOntology()
	require.NotNil(t, amlOntology)

	// Test with small amount (should be compliant)
	action1 := &ontology.FinancialAction{
		ID:      "trade-001",
		AgentID: "agent-001",
		Type:    ontology.ActionTrade,
		Amount: &ontology.MonetaryValue{
			Amount:   1000.00,
			Currency: "USD",
		},
	}

	result, err := amlOntology.Validate(nil, action1)
	require.NoError(t, err)
	assert.Equal(t, ontology.StatusCompliant, result.OverallStatus)
	assert.Equal(t, 1.0, result.Score)

	// Test with large amount (should be violated)
	action2 := &ontology.FinancialAction{
		ID:      "trade-002",
		AgentID: "agent-001",
		Type:    ontology.ActionTransfer,
		Amount: &ontology.MonetaryValue{
			Amount:   50000.00,
			Currency: "USD",
		},
	}

	result, err = amlOntology.Validate(nil, action2)
	require.NoError(t, err)
	assert.Equal(t, ontology.StatusViolated, result.OverallStatus)
	assert.Less(t, result.Score, 1.0)
}

func TestSanctionedEntityRule(t *testing.T) {
	amlOntology := ontology.NewAMLOntology()

	// Test with sanctioned entity
	action := &ontology.FinancialAction{
		ID:      "trade-003",
		AgentID: "agent-001",
		Type:    ontology.ActionTransfer,
		Counterparty: &ontology.CounterpartyInfo{
			ID:           "bad-actor-001",
			Type:         "corporate",
			IsSanctioned: true,
		},
	}

	result, err := amlOntology.Validate(nil, action)
	require.NoError(t, err)

	// Should have at least one violation
	foundSanctionedViolation := false
	for _, ruleResult := range result.RuleResults {
		if ruleResult.RuleID == "aml-sanctioned-entity" && ruleResult.Status == ontology.StatusViolated {
			foundSanctionedViolation = true
			break
		}
	}
	assert.True(t, foundSanctionedViolation, "Should detect sanctioned entity violation")
}

func TestBaselCapitalAdequacy(t *testing.T) {
	baselOntology := ontology.NewBaselOntology()
	require.NotNil(t, baselOntology)

	// Test with compliant ratios
	action1 := &ontology.FinancialAction{
		ID:      "risk-001",
		AgentID: "agent-001",
		Type:    ontology.ActionRiskAssessment,
		Context: map[string]interface{}{
			"cet1_ratio":          0.06, // 6% > 4.5% minimum
			"total_capital_ratio": 0.10, // 10% > 8% minimum
			"leverage_ratio":      0.05, // 5% > 3% minimum
			"stress_test_passed":  true,
		},
	}

	result, err := baselOntology.Validate(nil, action1)
	require.NoError(t, err)
	assert.Equal(t, ontology.StatusCompliant, result.OverallStatus)

	// Test with non-compliant CET1
	action2 := &ontology.FinancialAction{
		ID:      "risk-002",
		AgentID: "agent-001",
		Type:    ontology.ActionRiskAssessment,
		Context: map[string]interface{}{
			"cet1_ratio":          0.03, // 3% < 4.5% minimum
			"total_capital_ratio": 0.10,
		},
	}

	result, err = baselOntology.Validate(nil, action2)
	require.NoError(t, err)
	assert.Equal(t, ontology.StatusViolated, result.OverallStatus)
}

func TestEvidencePackBuilder(t *testing.T) {
	builder := fintech.NewEvidencePackBuilder(
		fintech.EvidenceTypeAudit,
		"agent-003",
		"validation-003",
	)

	pack, err := builder.
		WithAgentName("Audit Bot").
		WithMetadata("environment", "production").
		WithValidationResult(&fintech.ValidationEvidence{
			Status:     "valid",
			Confidence: 0.88,
			Score:      0.85,
		}).
		AddComplianceCheck(fintech.ComplianceEvidence{
			RegulationID:   "test-rule",
			RegulationName: "Test Rule",
			Category:       "TEST",
			Status:         "compliant",
			Severity:       "low",
			Description:    "Test compliance check",
		}).
		Build()

	require.NoError(t, err)
	assert.Equal(t, "Audit Bot", pack.AgentName)
	assert.Equal(t, "production", pack.Metadata["environment"])
	assert.NotNil(t, pack.ValidationResult)
	assert.Len(t, pack.ComplianceChecks, 1)
}

// Mock implementations for testing
type mockP2PManager struct{}

func (m *mockP2PManager) GetNodeID() string { return "test-node" }

type mockConfig struct{}

func (m *mockConfig) GetString(key string) string { return "" }
func (m *mockConfig) GetInt(key string) int       { return 0 }
func (m *mockConfig) GetBool(key string) bool     { return false }
