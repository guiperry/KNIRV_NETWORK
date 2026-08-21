package operator

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"go.uber.org/zap"
)

// Service represents the operator registry service
type Service struct {
	logger         *zap.Logger
	operators      map[string]*Operator
	mutex          sync.RWMutex
	knirvOracleURL string
	startedAt      time.Time
}

// NewService creates a new operator registry service
func NewService(logger *zap.Logger, knirvOracleURL string) *Service {
	return &Service{
		logger:         logger,
		operators:      make(map[string]*Operator),
		knirvOracleURL: knirvOracleURL,
		startedAt:      time.Now(),
	}
}

// generateDemoSignature creates non-production catalog metadata.
func (s *Service) generateDemoSignature(operatorID string, createdDate *string, issuer, sigType string) *OperatorSignature {
	creationDate := time.Now().Format(time.RFC3339)
	if createdDate != nil {
		creationDate = *createdDate
	}

	if sigType == "" {
		sigType = "RsaSignature2018"
	}

	// Generate random proof value (long hex string)
	proofBytes := make([]byte, 64)
	rand.Read(proofBytes)
	proofValue := hex.EncodeToString(proofBytes)

	// Generate random public key
	pubKeyBytes := make([]byte, 32)
	rand.Read(pubKeyBytes)
	publicKey := "04:" + hex.EncodeToString(pubKeyBytes)

	simulatedIssuer := "NANDA+ANS Trusted CA G1"
	if issuer != "" {
		simulatedIssuer = issuer
	}

	return &OperatorSignature{
		Type:               sigType,
		Created:            creationDate,
		VerificationMethod: fmt.Sprintf("%s#key-pki-1", operatorID),
		ProofPurpose:       "assertionMethod",
		ProofValue:         proofValue,
		SimulatedIssuer:    simulatedIssuer,
		SimulatedPublicKey: publicKey,
	}
}

// getDemoOperators returns the explicit demo catalog.
func (s *Service) getDemoOperators() []*Operator {
	return []*Operator{
		{
			OperatorFacts: OperatorFacts{
				ID:           "did:nanda:operator-translator-001",
				Name:         "LinguaBot Pro",
				AnsName:      "a2a://LinguaBotPro.DocumentTranslation.PolyglotAI.v2.1.certified",
				Capability:   "Document Translation",
				Capabilities: []string{"Document Translation", "Language Identification", "Text Summarization"},
				Provider:     "PolyglotAI",
				Version:      "2.1",
				Extension:    "certified",
				Description:  "Advanced AI for translating documents in over 100 languages with high accuracy and context preservation. Supports various file formats and offers secure processing. Includes simulated PKI signature for trust demonstration.",
				Endpoints: &EndpointSet{
					Type:              "nanda:EndpointSet",
					StaticEndpoint:    []string{"https://api.polyglotai.com/translate/v2.1"},
					AdaptiveRouterURL: "https://router.polyglotai.com/translate",
				},
				Attestations: []string{"did:nanda:attestation-polyglot-hipaa-v1", "did:nanda:attestation-polyglot-iso27001-v1"},
				ProtocolExtensions: &ProtocolExtensionSet{
					Type: "ans:ProtocolExtensionSet",
					A2A: &ProtocolExtension{
						Type:    "A2AAgentCard",
						Name:    "LinguaBot A2A Interface",
						Details: map[string]interface{}{"supportedFormats": []string{"pdf", "docx", "txt"}},
					},
					MCP: &ProtocolExtension{
						Type:    "MCPToolDescription",
						Name:    "LinguaBot MCP Tool",
						Details: map[string]interface{}{"maxFileSize": "10MB"},
					},
				},
				AvatarURL:  "https://placehold.co/100x100.png",
				DataAIHint: "robot language",
				Signature:  s.generateDemoSignature("did:nanda:operator-translator-001", stringPtr("2024-01-14T00:00:00Z"), "PolyglotAI CA", ""),
			},
			AddrTTL:      3600,
			AddrFactsURL: "https://polyglotai.com/.well-known/operator-facts-linguabot.jsonld",
			RegisteredAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			OperatorFacts: OperatorFacts{
				ID:           "did:nanda:operator-dataprotector-002",
				Name:         "GuardianShield",
				AnsName:      "acp://GuardianShield.DataAnonymization.SecureDataCorp.v1.0.enterprise",
				Capability:   "Data Anonymization",
				Capabilities: []string{"Data Anonymization", "PII Detection", "Privacy Policy Enforcement"},
				Provider:     "SecureDataCorp",
				Version:      "1.0",
				Extension:    "enterprise",
				Description:  "Enterprise-grade data anonymization agent ensuring GDPR and CCPA compliance. Uses advanced techniques to protect sensitive information while preserving data utility. Includes simulated PKI signature for trust demonstration.",
				Endpoints: &EndpointSet{
					Type:           "nanda:EndpointSet",
					StaticEndpoint: []string{"https://api.securedatacorp.com/anonymize/v1.0"},
				},
				Attestations: []string{"did:nanda:attestation-securedata-gdpr-v2", "did:nanda:attestation-securedata-soc2-v1"},
				ProtocolExtensions: &ProtocolExtensionSet{
					Type: "ans:ProtocolExtensionSet",
					ACP: &ProtocolExtension{
						Type:    "ACPProfile",
						Name:    "GuardianShield ACP",
						Details: map[string]interface{}{"complianceLevel": "Strict"},
					},
				},
				AvatarURL:  "https://placehold.co/100x100.png",
				DataAIHint: "shield security",
				Signature:  s.generateDemoSignature("did:nanda:operator-dataprotector-002", stringPtr("2023-11-19T00:00:00Z"), "SecureDataCorp CA", "EcdsaSecp256k1Signature2019"),
			},
			AddrTTL:      7200,
			AddrFactsURL: "https://securedatacorp.com/.well-known/operator-facts-guardian.jsonld",
			RegisteredAt: time.Date(2023, 11, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			OperatorFacts: OperatorFacts{
				ID:           "did:nanda:operator-artcreator-003",
				Name:         "PixelDreamer",
				AnsName:      "mcp://PixelDreamer.ImageGeneration.ArtifyInc.v3.beta",
				Capability:   "Image Generation",
				Capabilities: []string{"Image Generation from Text", "Style Transfer", "Image Upscaling"},
				Provider:     "ArtifyInc",
				Version:      "3.0",
				Extension:    "beta",
				Description:  "A creative AI agent that generates stunning and unique images from textual descriptions. Explore various artistic styles and resolutions. Includes simulated PKI signature for trust demonstration.",
				Endpoints: &EndpointSet{
					Type:              "nanda:EndpointSet",
					StaticEndpoint:    []string{"https://api.artifyinc.com/imagegen/v3"},
					AdaptiveRouterURL: "https://router.artifyinc.com/imagegen",
				},
				ProtocolExtensions: &ProtocolExtensionSet{
					Type: "ans:ProtocolExtensionSet",
					MCP: &ProtocolExtension{
						Type:    "MCPToolDescription",
						Name:    "PixelDreamer MCP",
						Details: map[string]interface{}{"outputResolutions": []string{"512x512", "1024x1024", "2048x2048"}},
					},
				},
				AvatarURL:  "https://placehold.co/100x100.png",
				DataAIHint: "abstract art",
				Signature:  s.generateDemoSignature("did:nanda:operator-artcreator-003", stringPtr("2024-02-28T00:00:00Z"), "ArtifyInc CA", ""),
			},
			AddrTTL:      1800,
			AddrFactsURL: "https://artifyinc.com/.well-known/operator-facts-pixel.jsonld",
			RegisteredAt: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			OperatorFacts: OperatorFacts{
				ID:           "did:nanda:operator-codehelper-004",
				Name:         "DevMentor",
				AnsName:      "a2a://DevMentor.CodeGeneration.CodeGenius.v1.5",
				Capability:   "Code Generation",
				Capabilities: []string{"Code Generation", "Debugging Assistance", "API Documentation Lookup"},
				Provider:     "CodeGenius",
				Version:      "1.5",
				Description:  "An AI assistant for developers, providing code snippets, debugging help, and quick access to API documentation. Supports Python, JavaScript, and Java. Includes simulated PKI signature for trust demonstration.",
				Endpoints: &EndpointSet{
					Type:           "nanda:EndpointSet",
					StaticEndpoint: []string{"https://api.codegenius.com/devmentor/v1.5"},
				},
				Attestations: []string{"did:nanda:attestation-codegenius-opensource-contrib-v1"},
				ProtocolExtensions: &ProtocolExtensionSet{
					Type: "ans:ProtocolExtensionSet",
					A2A: &ProtocolExtension{
						Type:    "A2AAgentCard",
						Name:    "DevMentor A2A",
						Details: map[string]interface{}{"supportedLanguages": []string{"Python", "JavaScript", "Java"}},
					},
				},
				AvatarURL:  "https://placehold.co/100x100.png",
				DataAIHint: "code computer",
				Signature:  s.generateDemoSignature("did:nanda:operator-codehelper-004", stringPtr("2024-02-09T00:00:00Z"), "CodeGenius Community CA", ""),
			},
			AddrTTL:      3600,
			AddrFactsURL: "https://codegenius.com/.well-known/operator-facts-devmentor.jsonld",
			RegisteredAt: time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			OperatorFacts: OperatorFacts{
				ID:           "did:nanda:operator-dbquery-005",
				Name:         "DataOracle",
				AnsName:      "acp://DataOracle.DatabaseQuery.QueryWorks.v1.2.trusted",
				Capability:   "Database Query",
				Capabilities: []string{"SQL Query Execution", "NoSQL Data Retrieval", "Data Aggregation"},
				Provider:     "QueryWorks Inc.",
				Version:      "1.2",
				Extension:    "trusted",
				Description:  "A powerful AI agent for securely querying various types of databases and retrieving structured data. Supports natural language queries and returns results in multiple formats. Includes simulated PKI signature for trust demonstration.",
				Endpoints: &EndpointSet{
					Type:              "nanda:EndpointSet",
					StaticEndpoint:    []string{"https://api.queryworks.com/query/v1.2"},
					AdaptiveRouterURL: "https://router.queryworks.com/query",
				},
				Attestations: []string{"did:nanda:attestation-queryworks-pci-dss-v1"},
				ProtocolExtensions: &ProtocolExtensionSet{
					Type: "ans:ProtocolExtensionSet",
					ACP: &ProtocolExtension{
						Type:    "ACPProfile",
						Name:    "DataOracle ACP",
						Details: map[string]interface{}{"supportedDatabases": []string{"PostgreSQL", "MongoDB", "MySQL"}},
					},
				},
				AvatarURL:  "https://placehold.co/100x100.png",
				DataAIHint: "database tech",
				Signature:  s.generateDemoSignature("did:nanda:operator-dbquery-005", stringPtr("2023-09-04T00:00:00Z"), "QueryWorks Trusted CA", "JsonWebSignature2020"),
			},
			AddrTTL:      3000,
			AddrFactsURL: "https://queryworks.com/.well-known/operator-facts-dataoracle.jsonld",
			RegisteredAt: time.Date(2023, 9, 5, 0, 0, 0, 0, time.UTC),
		},
	}
}

// Initialize loads demo data only when explicitly enabled.
func (s *Service) Initialize() {
	demo := strings.TrimSpace(os.Getenv("KNIRV_ENABLE_DEMO"))
	if demo != "1" && !strings.EqualFold(demo, "true") {
		s.logger.Info("Operator registry initialized without demo records")
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()

	demoOperators := s.getDemoOperators()
	for _, operator := range demoOperators {
		s.operators[operator.ID] = operator
	}

	s.logger.Info("Operator registry initialized with explicit demo data", zap.Int("count", len(demoOperators)))
}

// GetOperatorByID retrieves an operator by ID
func (s *Service) GetOperatorByID(id string) (*Operator, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	operator, exists := s.operators[id]
	if !exists {
		return nil, fmt.Errorf("operator with ID %s not found", id)
	}

	// Return a copy to prevent external modification
	operatorCopy := *operator
	return &operatorCopy, nil
}

// GetAllOperators returns all operators
func (s *Service) GetAllOperators() ([]*Operator, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	operators := make([]*Operator, 0, len(s.operators))
	for _, operator := range s.operators {
		// Return copies to prevent external modification
		operatorCopy := *operator
		operators = append(operators, &operatorCopy)
	}

	return operators, nil
}

// registerOperatorWithKnirvOracle registers an operator with KNIRV-ORACLE
func (s *Service) registerOperatorWithKnirvOracle(operator *Operator) (*KnirvOracleRegistrationResponse, error) {
	registrationData := map[string]interface{}{
		"operatorId":   operator.ID,
		"name":         operator.Name,
		"capability":   operator.Capability,
		"capabilities": operator.Capabilities,
		"provider":     operator.Provider,
		"version":      operator.Version,
		"description":  operator.Description,
		"factsUrl":     operator.AddrFactsURL,
		"ansName":      operator.AnsName,
		"registeredAt": operator.RegisteredAt.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(registrationData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal registration data: %w", err)
	}

	url := fmt.Sprintf("%s/register-capability", s.knirvOracleURL)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NANDA-ANS-Registry/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KNIRV-ORACLE registration failed: %d %s", resp.StatusCode, resp.Status)
	}

	var result KnirvOracleRegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// RegisterNewOperator registers a new operator
func (s *Service) RegisterNewOperator(req *RegisterOperatorRequest) (*RegisterOperatorResponse, error) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, fmt.Errorf("operator id is required")
	}
	signature, err := verifyOperatorRegistration(req)
	if err != nil {
		return nil, err
	}
	operatorID := req.ID

	// Create the operator
	operator := &Operator{
		OperatorFacts: OperatorFacts{
			ID:                 operatorID,
			Name:               req.Name,
			AnsName:            req.AnsName,
			Capability:         req.Capability,
			Capabilities:       req.Capabilities,
			Provider:           req.Provider,
			Version:            req.Version,
			Extension:          req.Extension,
			Description:        req.Description,
			Endpoints:          req.Endpoints,
			Attestations:       req.Attestations,
			ProtocolExtensions: req.ProtocolExtensions,
			AvatarURL:          req.AvatarURL,
			DataAIHint:         req.DataAIHint,
			Signature: &OperatorSignature{
				Type: "CosmosSignModeDirectEnvelope", Created: time.Now().UTC().Format(time.RFC3339),
				VerificationMethod: signature.Address, ProofPurpose: "operator-registration",
				ProofValue: req.ControllerSignature,
			},
		},
		AddrTTL:      req.AddrTTL,
		AddrFactsURL: req.AddrFactsURL,
		RegisteredAt: time.Now(),
	}

	// Set defaults
	if operator.AddrTTL == 0 {
		operator.AddrTTL = 3600
	}
	if operator.AddrFactsURL == "" {
		return nil, fmt.Errorf("operator facts URL is required")
	}

	// Register with KNIRV-ORACLE
	s.logger.Info("Registering operator with KNIRV-ORACLE", zap.String("name", operator.Name))
	oracleResponse, err := s.registerOperatorWithKnirvOracle(operator)
	if err != nil {
		s.logger.Error("Failed to register with KNIRV-ORACLE", zap.Error(err))
		// Continue without oracle registration for now
		operator.OracleStatus = "failed"
	} else {
		operator.TransactionHash = oracleResponse.TransactionHash
		operator.CapabilityID = oracleResponse.CapabilityID
		operator.OracleStatus = oracleResponse.Status
	}

	// Store the operator
	s.mutex.Lock()
	s.operators[operatorID] = operator
	s.mutex.Unlock()

	s.logger.Info("Operator registered successfully",
		zap.String("id", operatorID),
		zap.String("transactionHash", operator.TransactionHash),
		zap.String("capabilityId", operator.CapabilityID))

	return &RegisterOperatorResponse{
		Operator:        *operator,
		TransactionHash: operator.TransactionHash,
		CapabilityID:    operator.CapabilityID,
	}, nil
}

func verifyOperatorRegistration(req *RegisterOperatorRequest) (knirvsigning.SignedMessage, error) {
	var signed knirvsigning.SignedMessage
	if err := json.Unmarshal([]byte(req.ControllerSignature), &signed); err != nil {
		return signed, fmt.Errorf("decode controller signature: %w", err)
	}
	copyReq := *req
	copyReq.ControllerSignature = ""
	payload, err := json.Marshal(copyReq)
	if err != nil {
		return signed, err
	}
	digest := sha256.Sum256(payload)
	expected := []byte(fmt.Sprintf("operator-registration:%s:%x", req.ID, digest))
	if err := knirvsigning.VerifyMessagePayload(signed, "knirv.gateway", "operator-registration", "knirv-1", expected, time.Now()); err != nil {
		return signed, fmt.Errorf("verify controller signature: %w", err)
	}
	return signed, nil
}

// GetHealth returns health information
func (s *Service) GetHealth() *HealthResponse {
	s.mutex.RLock()
	operatorCount := len(s.operators)
	s.mutex.RUnlock()

	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	uptime := time.Since(s.startedAt).Seconds()
	oracleState := "not_configured"
	if s.knirvOracleURL != "" {
		oracleState = "configured"
	}
	return &HealthResponse{
		Status:    "healthy",
		Service:   "NANDA ANS - Operator Registry",
		Version:   "1.0.0",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    uptime,
		Components: map[string]string{
			"api":                      "operational",
			"database":                 "operational",
			"operator_registry":        "operational",
			"knirv_oracle_integration": oracleState,
		},
		Metrics: map[string]interface{}{
			"memory_usage": map[string]interface{}{
				"heap_alloc": memory.HeapAlloc,
				"heap_sys":   memory.HeapSys,
				"sys":        memory.Sys,
			},
			"uptime_seconds": uptime,
			"operator_count": operatorCount,
		},
		Endpoints: map[string]string{
			"operator_discovery":    "/",
			"operator_registration": "/register",
			"operator_search":       "/api/operators",
			"health_check":          "/api/health",
		},
		Integration: map[string]interface{}{
			"knirv_oracle": map[string]interface{}{
				"endpoint": s.knirvOracleURL,
				"status":   "connected",
			},
		},
	}
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
