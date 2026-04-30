package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/services/cognitiveengine"
	"backend_server/internal/services/guardrails"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// PipelineTriggerFunc is called after successful onboarding to trigger data pipeline processing
type PipelineTriggerFunc func(ctx context.Context, orgID string, config *OrganizationConfig) error

// OnboardingService handles intelligent onboarding and validation guardrails
type OnboardingService struct {
	db               *database.BuntDBManager
	cognitiveEngine  *cognitiveengine.CognitiveEngine
	guardrailManager *guardrails.DynamicGuardrailManager
	configurations   map[string]*OrganizationConfig
	configMu         sync.RWMutex
	pipelineTrigger  PipelineTriggerFunc
}

// OrganizationConfig represents an organization's value system and ontology
type OrganizationConfig struct {
	ID             string           `json:"id"`
	OrganizationID string           `json:"organization_id"`
	Name           string           `json:"name"`
	ValueSystem    *ValueSystem     `json:"value_system"`
	Ontology       *Ontology        `json:"ontology"`
	Guardrails     []*GuardrailRule `json:"guardrails"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	Status         string           `json:"status"`
}

// ValueSystem represents the organization's values and principles
type ValueSystem struct {
	Guidelines       []string               `json:"guidelines"`
	Customs          []string               `json:"customs"`
	Etiquette        []string               `json:"etiquette"`
	MissionStatement string                 `json:"mission_statement"`
	StatedValues     []string               `json:"stated_values"`
	Goals            []string               `json:"goals"`
	Objectives       []string               `json:"objectives"`
	Insights         []string               `json:"insights"`
	RiskAppetite     *RiskAppetite          `json:"risk_appetite"`
	CulturalContext  map[string]interface{} `json:"cultural_context"`
	RegionalNuances  map[string]interface{} `json:"regional_nuances"`
}

// RiskAppetite represents the organization's risk tolerance
type RiskAppetite struct {
	Level          string  `json:"level"`           // "low", "medium", "high"
	ToleranceScore float64 `json:"tolerance_score"` // 0.0 - 1.0
	MaxLossPercent float64 `json:"max_loss_percent"`
	RecoveryTime   string  `json:"recovery_time"`
}

// Ontology represents the organization's knowledge structure
type Ontology struct {
	TradeSecrets         []string               `json:"trade_secrets"`
	BusinessLogic        []string               `json:"business_logic"`
	UserData             []string               `json:"user_data"`
	Rules                []string               `json:"rules"`
	Regulations          []string               `json:"regulations"`
	Procedures           []string               `json:"procedures"`
	Policies             []string               `json:"policies"`
	FAQs                 []string               `json:"faqs"`
	CustomerService      []string               `json:"customer_service"`
	IndustryJargon       map[string]string      `json:"industry_jargon"`
	StakeholderHierarchy map[string]interface{} `json:"stakeholder_hierarchy"`
	DecisionRights       map[string]interface{} `json:"decision_rights"`
}

// GuardrailRule represents a validation guardrail rule
type GuardrailRule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // "value", "ontology", "policy"
	Category   string                 `json:"category"`
	Condition  string                 `json:"condition"`
	Action     string                 `json:"action"` // "allow", "deny", "warn", "log"
	Parameters map[string]interface{} `json:"parameters"`
	Severity   string                 `json:"severity"` // "low", "medium", "high", "critical"
	Enabled    bool                   `json:"enabled"`
	CreatedAt  time.Time              `json:"created_at"`
}

// OnboardingRequest represents a request to onboard an organization
type OnboardingRequest struct {
	OrganizationID string                 `json:"organization_id"`
	Name           string                 `json:"name"`
	ValueSystem    *ValueSystem           `json:"value_system"`
	Ontology       *Ontology              `json:"ontology"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// OnboardingResult represents the result of an onboarding operation
type OnboardingResult struct {
	ConfigID       string           `json:"config_id"`
	Guardrails     []*GuardrailRule `json:"guardrails_created"`
	Status         string           `json:"status"`
	Message        string           `json:"message"`
	ProcessingTime time.Duration    `json:"processing_time"`
}

// NewOnboardingService creates a new onboarding service
func NewOnboardingService(db *database.BuntDBManager, cognitiveEngine *cognitiveengine.CognitiveEngine, guardrailManager *guardrails.DynamicGuardrailManager) *OnboardingService {
	return &OnboardingService{
		db:               db,
		cognitiveEngine:  cognitiveEngine,
		guardrailManager: guardrailManager,
		configurations:   make(map[string]*OrganizationConfig),
	}
}

// SetPipelineTrigger registers a callback to invoke after successful onboarding data ingestion
func (s *OnboardingService) SetPipelineTrigger(fn PipelineTriggerFunc) {
	s.pipelineTrigger = fn
}

// OnboardOrganization processes an organization's value system and ontology
func (s *OnboardingService) OnboardOrganization(ctx context.Context, req *OnboardingRequest) (*OnboardingResult, error) {
	startTime := time.Now()

	// Generate configuration ID
	configID := uuid.New().String()[:12]

	// Create organization configuration
	config := &OrganizationConfig{
		ID:             configID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		ValueSystem:    req.ValueSystem,
		Ontology:       req.Ontology,
		Guardrails:     []*GuardrailRule{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Status:         "processing",
	}

	// Generate guardrails from value system
	valueGuardrails := s.generateValueGuardrails(req.ValueSystem)
	config.Guardrails = append(config.Guardrails, valueGuardrails...)

	// Generate guardrails from ontology
	ontologyGuardrails := s.generateOntologyGuardrails(req.Ontology)
	config.Guardrails = append(config.Guardrails, ontologyGuardrails...)

	// Register guardrails with the guardrail manager
	for _, rule := range config.Guardrails {
		if err := s.registerGuardrail(rule); err != nil {
			log.Printf("Warning: Failed to register guardrail %s: %v", rule.ID, err)
		}
	}

	// Feed value system into cognitive engine for contextual resonance
	if s.cognitiveEngine != nil {
		if err := s.feedValueSystemToCognitiveEngine(config); err != nil {
			log.Printf("Warning: Failed to feed value system to cognitive engine: %v", err)
		}
	}

	// Save configuration
	config.Status = "active"
	config.UpdatedAt = time.Now()

	s.configMu.Lock()
	s.configurations[configID] = config
	s.configMu.Unlock()

	// Persist to database
	if err := s.saveConfiguration(config); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	// Trigger data pipeline after successful ingestion
	if s.pipelineTrigger != nil {
		go func() {
			triggerCtx, triggerCancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer triggerCancel()
			if err := s.pipelineTrigger(triggerCtx, req.OrganizationID, config); err != nil {
				log.Printf("Warning: Pipeline trigger failed after onboarding %s: %v", configID, err)
			} else {
				log.Printf("Pipeline trigger succeeded for onboarding %s (org=%s)", configID, req.OrganizationID)
			}
		}()
	}

	processingTime := time.Since(startTime)

	return &OnboardingResult{
		ConfigID:       configID,
		Guardrails:     config.Guardrails,
		Status:         "completed",
		Message:        fmt.Sprintf("Organization onboarded successfully with %d guardrails", len(config.Guardrails)),
		ProcessingTime: processingTime,
	}, nil
}

// generateValueGuardrails generates guardrails from value system
func (s *OnboardingService) generateValueGuardrails(vs *ValueSystem) []*GuardrailRule {
	guardrails := []*GuardrailRule{}

	if vs == nil {
		return guardrails
	}

	// Generate guardrails from guidelines
	for i, guideline := range vs.Guidelines {
		rule := &GuardrailRule{
			ID:        uuid.New().String()[:12],
			Name:      fmt.Sprintf("Guideline: %s", guideline),
			Type:      "value",
			Category:  "guideline",
			Condition: fmt.Sprintf("action.violates_guideline('%s')", guideline),
			Action:    "warn",
			Parameters: map[string]interface{}{
				"guideline": guideline,
				"index":     i,
			},
			Severity:  "medium",
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		guardrails = append(guardrails, rule)
	}

	// Generate guardrails from stated values
	for _, value := range vs.StatedValues {
		rule := &GuardrailRule{
			ID:        uuid.New().String()[:12],
			Name:      fmt.Sprintf("Value: %s", value),
			Type:      "value",
			Category:  "stated_value",
			Condition: fmt.Sprintf("action.aligns_with_value('%s')", value),
			Action:    "allow",
			Parameters: map[string]interface{}{
				"value": value,
			},
			Severity:  "high",
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		guardrails = append(guardrails, rule)
	}

	// Generate guardrails from risk appetite
	if vs.RiskAppetite != nil {
		rule := &GuardrailRule{
			ID:        uuid.New().String()[:12],
			Name:      "Risk Appetite Control",
			Type:      "value",
			Category:  "risk_management",
			Condition: fmt.Sprintf("action.risk_score <= %f", vs.RiskAppetite.ToleranceScore),
			Action:    "deny",
			Parameters: map[string]interface{}{
				"max_risk_score": vs.RiskAppetite.ToleranceScore,
				"level":          vs.RiskAppetite.Level,
			},
			Severity:  "critical",
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		guardrails = append(guardrails, rule)
	}

	return guardrails
}

// generateOntologyGuardrails generates guardrails from ontology
func (s *OnboardingService) generateOntologyGuardrails(ontology *Ontology) []*GuardrailRule {
	guardrails := []*GuardrailRule{}

	if ontology == nil {
		return guardrails
	}

	// Generate guardrails from trade secrets
	for _, secret := range ontology.TradeSecrets {
		rule := &GuardrailRule{
			ID:        uuid.New().String()[:12],
			Name:      fmt.Sprintf("Trade Secret: %s", secret),
			Type:      "ontology",
			Category:  "trade_secret",
			Condition: fmt.Sprintf("data.contains_trade_secret('%s')", secret),
			Action:    "deny",
			Parameters: map[string]interface{}{
				"secret": secret,
			},
			Severity:  "critical",
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		guardrails = append(guardrails, rule)
	}

	// Generate guardrails from regulations
	for _, regulation := range ontology.Regulations {
		rule := &GuardrailRule{
			ID:        uuid.New().String()[:12],
			Name:      fmt.Sprintf("Regulation: %s", regulation),
			Type:      "ontology",
			Category:  "regulation",
			Condition: fmt.Sprintf("action.complies_with_regulation('%s')", regulation),
			Action:    "deny",
			Parameters: map[string]interface{}{
				"regulation": regulation,
			},
			Severity:  "critical",
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		guardrails = append(guardrails, rule)
	}

	// Generate guardrails from policies
	for _, policy := range ontology.Policies {
		rule := &GuardrailRule{
			ID:        uuid.New().String()[:12],
			Name:      fmt.Sprintf("Policy: %s", policy),
			Type:      "ontology",
			Category:  "policy",
			Condition: fmt.Sprintf("action.follows_policy('%s')", policy),
			Action:    "warn",
			Parameters: map[string]interface{}{
				"policy": policy,
			},
			Severity:  "high",
			Enabled:   true,
			CreatedAt: time.Now(),
		}
		guardrails = append(guardrails, rule)
	}

	return guardrails
}

// registerGuardrail registers a guardrail with the guardrail manager
func (s *OnboardingService) registerGuardrail(rule *GuardrailRule) error {
	if s.guardrailManager == nil {
		return fmt.Errorf("guardrail manager not available")
	}

	// Convert to guardrail manager format
	config := &guardrails.GuardrailConfig{
		Type:      guardrails.GuardrailType(rule.Type),
		Enabled:   rule.Enabled,
		MaxValue:  100,
		MinValue:  0,
		Action:    rule.Action,
		Threshold: 0.8,
		Cooldown:  5 * time.Minute,
	}

	return s.guardrailManager.ConfigureGuardrail("system", config)
}

// feedValueSystemToCognitiveEngine feeds value system into cognitive engine
func (s *OnboardingService) feedValueSystemToCognitiveEngine(config *OrganizationConfig) error {
	if s.cognitiveEngine == nil {
		return fmt.Errorf("cognitive engine not available")
	}

	riskLevel := "medium"
	var guidelines, values []string

	if config.ValueSystem != nil {
		guidelines = config.ValueSystem.Guidelines
		values = config.ValueSystem.StatedValues
		if config.ValueSystem.RiskAppetite != nil {
			riskLevel = config.ValueSystem.RiskAppetite.Level
		}
	}

	s.cognitiveEngine.InjectOrganizationContext(
		config.OrganizationID,
		config.Name,
		guidelines,
		values,
		riskLevel,
	)

	return nil
}

// GetConfiguration returns an organization configuration
func (s *OnboardingService) GetConfiguration(configID string) (*OrganizationConfig, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	config, ok := s.configurations[configID]
	if !ok {
		return nil, fmt.Errorf("configuration not found: %s", configID)
	}
	return config, nil
}

// ListConfigurations returns all organization configurations
func (s *OnboardingService) ListConfigurations() []*OrganizationConfig {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	configs := make([]*OrganizationConfig, 0, len(s.configurations))
	for _, config := range s.configurations {
		configs = append(configs, config)
	}
	return configs
}

// UpdateConfiguration updates an organization configuration
func (s *OnboardingService) UpdateConfiguration(configID string, req *OnboardingRequest) (*OrganizationConfig, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	config, ok := s.configurations[configID]
	if !ok {
		return nil, fmt.Errorf("configuration not found: %s", configID)
	}

	// Update fields
	if req.Name != "" {
		config.Name = req.Name
	}
	if req.ValueSystem != nil {
		config.ValueSystem = req.ValueSystem
	}
	if req.Ontology != nil {
		config.Ontology = req.Ontology
	}
	config.UpdatedAt = time.Now()

	// Regenerate guardrails
	config.Guardrails = []*GuardrailRule{}
	valueGuardrails := s.generateValueGuardrails(config.ValueSystem)
	config.Guardrails = append(config.Guardrails, valueGuardrails...)
	ontologyGuardrails := s.generateOntologyGuardrails(config.Ontology)
	config.Guardrails = append(config.Guardrails, ontologyGuardrails...)

	// Save to database
	if err := s.saveConfiguration(config); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	return config, nil
}

// DeleteConfiguration deletes an organization configuration
func (s *OnboardingService) DeleteConfiguration(configID string) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	if _, ok := s.configurations[configID]; !ok {
		return fmt.Errorf("configuration not found: %s", configID)
	}

	delete(s.configurations, configID)

	// Remove from database
	return s.db.Transaction(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(fmt.Sprintf("onboarding:config:%s", configID))
		return err
	})
}

// saveConfiguration saves configuration to database
func (s *OnboardingService) saveConfiguration(config *OrganizationConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err = tx.Set(fmt.Sprintf("onboarding:config:%s", config.ID), string(data), nil)
		return err
	})
}

// LoadConfigurations loads configurations from database
func (s *OnboardingService) LoadConfigurations() error {
	return s.db.GetObjectsByPrefix("onboarding:config:", func(key string, value []byte) bool {
		var config OrganizationConfig
		if err := json.Unmarshal(value, &config); err != nil {
			log.Printf("Warning: Failed to unmarshal onboarding config: %v", err)
			return true
		}

		s.configMu.Lock()
		s.configurations[config.ID] = &config
		s.configMu.Unlock()

		return true
	})
}

// ValidateAction validates an action against organization guardrails
func (s *OnboardingService) ValidateAction(ctx context.Context, configID string, action map[string]interface{}) (bool, string, error) {
	config, err := s.GetConfiguration(configID)
	if err != nil {
		return false, "", err
	}

	// Check each guardrail
	for _, rule := range config.Guardrails {
		if !rule.Enabled {
			continue
		}

		// Evaluate condition (simplified - in production would use a proper rule engine)
		if s.evaluateCondition(rule.Condition, action) {
			switch rule.Action {
			case "deny":
				return false, fmt.Sprintf("Action denied by guardrail: %s", rule.Name), nil
			case "warn":
				log.Printf("Warning: Action triggered guardrail: %s", rule.Name)
			case "log":
				log.Printf("Info: Action logged by guardrail: %s", rule.Name)
			}
		}
	}

	return true, "Action allowed", nil
}

// evaluateCondition evaluates a guardrail condition
func (s *OnboardingService) evaluateCondition(_ string, _ map[string]interface{}) bool {
	// Simplified condition evaluation
	// In production, this would use a proper rule engine like CEL or OPA
	return false
}

// Start starts the onboarding service
func (s *OnboardingService) Start() error {
	log.Println("Onboarding service started")
	return nil
}

// Stop stops the onboarding service
func (s *OnboardingService) Stop() error {
	log.Println("Onboarding service stopped")
	return nil
}
