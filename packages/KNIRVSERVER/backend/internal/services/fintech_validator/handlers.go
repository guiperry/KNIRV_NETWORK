package fintech_validator

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/fintech"
	"backend_server/internal/fintech/ontology"
	"backend_server/internal/services/vault"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// Handlers contains all HTTP handlers for FinTech validation
type Handlers struct {
	service *FinTechValidatorService
}

// NewHandlers creates new FinTech validation handlers
func NewHandlers(service *FinTechValidatorService) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers all FinTech validation routes
func (h *Handlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Skip registration if the service is not enabled
	if h.service.Config == nil || !h.service.Config.Enabled {
		return
	}

	// Create subrouter for FinTech endpoints
	fintechRouter := r.PathPrefix("/api/fintech").Subrouter()

	// Public status endpoint
	fintechRouter.HandleFunc("/status", h.HandleGetStatus).Methods("GET")

	// Protected routes
	protectedRouter := fintechRouter.PathPrefix("").Subrouter()
	if authMiddleware != nil {
		protectedRouter.Use(authMiddleware.RequireAuth)
	}

	// Validation routes
	protectedRouter.HandleFunc("/validate", h.HandleValidate).Methods("POST")
	protectedRouter.HandleFunc("/validate/quick", h.HandleQuickValidate).Methods("POST")

	// Evidence pack routes
	protectedRouter.HandleFunc("/evidence", h.HandleListEvidencePacks).Methods("GET")
	protectedRouter.HandleFunc("/evidence/{id}", h.HandleGetEvidencePack).Methods("GET")
	protectedRouter.HandleFunc("/evidence/{id}/export", h.HandleExportEvidencePack).Methods("GET")
	protectedRouter.HandleFunc("/evidence/{id}/download", h.HandleDownloadEvidencePack).Methods("GET")

	// Ontology routes
	protectedRouter.HandleFunc("/ontologies", h.HandleListOntologies).Methods("GET")
	protectedRouter.HandleFunc("/ontologies/{id}", h.HandleGetOntology).Methods("GET")
	protectedRouter.HandleFunc("/ontologies/{id}/rules", h.HandleGetOntologyRules).Methods("GET")

	// Compliance check route
	protectedRouter.HandleFunc("/compliance/check", h.HandleComplianceCheck).Methods("POST")

	// Phase 2: Scenario routes
	protectedRouter.HandleFunc("/scenarios", h.HandleListScenarios).Methods("GET")
	protectedRouter.HandleFunc("/scenarios/{id}", h.HandleGetScenario).Methods("GET")
	protectedRouter.HandleFunc("/scenarios/validate", h.HandleRunScenarioValidation).Methods("POST")
	protectedRouter.HandleFunc("/scenarios/statistics", h.HandleGetScenarioStatistics).Methods("GET")

	// Phase 2: Certificate routes
	protectedRouter.HandleFunc("/certificates/issue", h.HandleIssueCertificate).Methods("POST")
	protectedRouter.HandleFunc("/certificates/verify", h.HandleVerifyCertificate).Methods("POST")
	protectedRouter.HandleFunc("/certificates/check", h.HandleCheckCertificateValidity).Methods("POST")
	protectedRouter.HandleFunc("/certificates/templates", h.HandleGetCertificateTemplates).Methods("GET")

	// Phase 3: Trajectory routes
	protectedRouter.HandleFunc("/trajectories", h.HandleListTrajectories).Methods("GET")
	protectedRouter.HandleFunc("/trajectories/{id}", h.HandleGetTrajectory).Methods("GET")
	protectedRouter.HandleFunc("/trajectories/{id}/replay", h.HandleReplayTrajectory).Methods("POST")
	protectedRouter.HandleFunc("/trajectories/{id}/compare", h.HandleCompareTrajectories).Methods("POST")
	protectedRouter.HandleFunc("/trajectories/capture/start", h.HandleStartTrajectoryCapture).Methods("POST")
	protectedRouter.HandleFunc("/trajectories/capture/{id}/stop", h.HandleStopTrajectoryCapture).Methods("POST")
	protectedRouter.HandleFunc("/trajectories/capture/active", h.HandleListActiveCaptures).Methods("GET")

	// Phase 3: Deterministic validation route
	protectedRouter.HandleFunc("/validate/deterministic", h.HandleValidateWithDeterminism).Methods("POST")

	// Phase 4: NRV Trace routes
	protectedRouter.HandleFunc("/nrv/traces", h.HandleListNRVTraces).Methods("GET")
	protectedRouter.HandleFunc("/nrv/traces/{id}", h.HandleGetNRVTrace).Methods("GET")
	protectedRouter.HandleFunc("/nrv/traces/{id}/score", h.HandleScoreFidelity).Methods("POST")
	protectedRouter.HandleFunc("/nrv/traces/{id}/distance", h.HandleCalculateSemanticDistance).Methods("POST")

	// Phase 4: Risk detection routes
	protectedRouter.HandleFunc("/nrv/traces/{id}/detect/kyc-bypass", h.HandleDetectKYCBypass).Methods("GET")
	protectedRouter.HandleFunc("/nrv/traces/{id}/detect/position-limits", h.HandleDetectPositionLimitViolation).Methods("POST")

	// Phase 4: Validation with NRV and fidelity scoring
	protectedRouter.HandleFunc("/validate/with-nrv", h.HandleValidateWithNRV).Methods("POST")

	// Phase 6: Arrow Flight Tick Data Streaming
	protectedRouter.HandleFunc("/ticks/streams", h.HandleCreateTickStream).Methods("POST")
	protectedRouter.HandleFunc("/ticks/{symbol}", h.HandleGetTicks).Methods("GET")
	protectedRouter.HandleFunc("/ticks/{symbol}/add", h.HandleAddTick).Methods("POST")
	protectedRouter.HandleFunc("/ticks/{symbol}/flush", h.HandleFlushTicks).Methods("POST")
	protectedRouter.HandleFunc("/ticks/query", h.HandleQueryTicks).Methods("POST")
	protectedRouter.HandleFunc("/ticks/export", h.HandleExportTicks).Methods("GET")

	// Handle OPTIONS for CORS
	fintechRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// HandleGetStatus returns the status of the FinTech validator service
func (h *Handlers) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":     "operational",
		"version":    "1.0.0",
		"ontologies": len(h.service.GetOntologies()),
	}

	json.NewEncoder(w).Encode(response)
}

// HandleValidate handles comprehensive FinTech validation requests
func (h *Handlers) HandleValidate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Perform validation
	result, err := h.service.Validate(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// HandleQuickValidate handles quick validation requests (simplified)
func (h *Handlers) HandleQuickValidate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req QuickValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Convert to full validation request
	fullReq := &ValidationRequest{
		AgentID:        req.AgentID,
		AgentName:      req.AgentName,
		ValidationType: "QUICK",
		Categories:     req.Categories,
		Parameters: map[string]interface{}{
			"quick_check": true,
		},
	}

	// Perform validation
	result, err := h.service.Validate(r.Context(), fullReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Simplify response for quick validation
	quickResponse := map[string]interface{}{
		"validation_id":    result.ValidationID,
		"status":           result.Status,
		"is_compliant":     result.IsCompliant,
		"overall_score":    result.OverallScore,
		"evidence_pack_id": result.EvidencePackID,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(quickResponse)
}

// QuickValidationRequest represents a quick validation request
type QuickValidationRequest struct {
	AgentID    string                        `json:"agent_id"`
	AgentName  string                        `json:"agent_name,omitempty"`
	Categories []ontology.RegulationCategory `json:"categories,omitempty"`
}

// HandleListEvidencePacks lists all evidence packs
func (h *Handlers) HandleListEvidencePacks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse query parameters for filtering
	filter := fintech.EvidenceFilter{}

	if agentID := r.URL.Query().Get("agent_id"); agentID != "" {
		filter.AgentID = agentID
	}

	if packType := r.URL.Query().Get("type"); packType != "" {
		filter.Type = fintech.EvidencePackType(packType)
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = fintech.EvidenceStatus(status)
	}

	packs, err := h.service.ListEvidencePacks(filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to list evidence packs",
			"details": err.Error(),
		})
		return
	}

	response := map[string]interface{}{
		"evidence_packs": packs,
		"count":          len(packs),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleGetEvidencePack retrieves a specific evidence pack
func (h *Handlers) HandleGetEvidencePack(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	pack, err := h.service.GetEvidencePack(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Evidence pack not found",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pack)
}

// HandleExportEvidencePack exports an evidence pack as a bundle
func (h *Handlers) HandleExportEvidencePack(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	bundle, err := h.service.ExportEvidencePack(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to export evidence pack",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bundle)
}

// HandleDownloadEvidencePack downloads an evidence pack as Markdown
func (h *Handlers) HandleDownloadEvidencePack(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get the evidence pack
	pack, err := h.service.GetEvidencePack(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Evidence pack not found",
			"details": err.Error(),
		})
		return
	}

	// Generate Markdown
	content, err := pack.GenerateMarkdown()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to generate Markdown",
			"details": err.Error(),
		})
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "text/markdown")
	w.Header().Set("Content-Disposition", "attachment; filename=\"evidence_"+id+".md\"")
	w.Write(content)
}

// HandleListOntologies lists all available ontologies
func (h *Handlers) HandleListOntologies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ontologies := h.service.GetOntologies()

	// Convert to simplified response
	simplified := make([]map[string]interface{}, len(ontologies))
	for i, o := range ontologies {
		simplified[i] = map[string]interface{}{
			"id":          o.GetID(),
			"name":        o.GetName(),
			"category":    o.GetCategory(),
			"version":     o.GetVersion(),
			"description": o.GetDescription(),
			"rule_count":  len(o.GetRules()),
		}
	}

	response := map[string]interface{}{
		"ontologies": simplified,
		"count":      len(simplified),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleGetOntology retrieves details about a specific ontology
func (h *Handlers) HandleGetOntology(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	o, err := h.service.GetOntology(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Ontology not found",
			"details": err.Error(),
		})
		return
	}

	response := map[string]interface{}{
		"id":          o.GetID(),
		"name":        o.GetName(),
		"category":    o.GetCategory(),
		"version":     o.GetVersion(),
		"description": o.GetDescription(),
		"rule_count":  len(o.GetRules()),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleGetOntologyRules retrieves all rules for a specific ontology
func (h *Handlers) HandleGetOntologyRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	o, err := h.service.GetOntology(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Ontology not found",
			"details": err.Error(),
		})
		return
	}

	rules := o.GetRules()
	simplified := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		simplified[i] = map[string]interface{}{
			"id":          rule.GetID(),
			"name":        rule.GetName(),
			"description": rule.GetDescription(),
			"category":    rule.GetCategory(),
			"severity":    rule.GetSeverity(),
		}
	}

	response := map[string]interface{}{
		"ontology_id": id,
		"rules":       simplified,
		"count":       len(simplified),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleComplianceCheck performs a compliance check on a financial action
func (h *Handlers) HandleComplianceCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ComplianceCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate the action against compliance engine
	// Note: This is a simplified version - full implementation would use the compliance engine directly
	response := map[string]interface{}{
		"action_id": req.Action.ID,
		"status":    "checked",
		"message":   "Compliance check completed - full implementation requires direct compliance engine access",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ComplianceCheckRequest represents a compliance check request
type ComplianceCheckRequest struct {
	Action     *ontology.FinancialAction     `json:"action"`
	Categories []ontology.RegulationCategory `json:"categories,omitempty"`
}

// === Phase 2: Scenario Validation Handlers ===

// HandleListScenarios lists all regulatory scenarios
func (h *Handlers) HandleListScenarios(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse query parameters
	filter := &vault.ScenarioFilter{}

	if regulation := r.URL.Query().Get("regulation"); regulation != "" {
		filter.Regulation = regulation
	}

	if scenarioType := r.URL.Query().Get("type"); scenarioType != "" {
		filter.Type = vault.RegulatoryScenarioType(scenarioType)
	}

	if severity := r.URL.Query().Get("severity"); severity != "" {
		filter.Severity = severity
	}

	if active := r.URL.Query().Get("active"); active != "" {
		isActive := active == "true"
		filter.IsActive = &isActive
	}

	scenarios, err := h.service.GetScenarios(filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to list scenarios",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	})
}

// HandleGetScenario retrieves a specific scenario
func (h *Handlers) HandleGetScenario(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	scenario, err := h.service.GetScenario(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Scenario not found",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(scenario)
}

// HandleRunScenarioValidation runs scenario validation against an agent
func (h *Handlers) HandleRunScenarioValidation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ScenarioValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	report, err := h.service.RunScenarioValidation(r.Context(), req.AgentID, req.AgentOutput, req.ScenarioIDs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Scenario validation failed",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(report)
}

// HandleGetScenarioStatistics returns scenario execution statistics
func (h *Handlers) HandleGetScenarioStatistics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := h.service.GetScenarioStatistics()
	if stats == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Scenario testing not enabled",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// ScenarioValidationRequest represents a scenario validation request
type ScenarioValidationRequest struct {
	AgentID     string   `json:"agent_id"`
	AgentOutput string   `json:"agent_output"`
	ScenarioIDs []string `json:"scenario_ids,omitempty"`
}

// === Phase 2: Certificate Handlers ===

// HandleIssueCertificate issues a PQC-signed Certificate of Correctness
func (h *Handlers) HandleIssueCertificate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	certificate, err := h.service.IssueCertificate(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to issue certificate",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(certificate)
}

// HandleVerifyCertificate verifies a certificate's PQC signature
func (h *Handlers) HandleVerifyCertificate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var certificate fintech.CertificateOfCorrectness
	if err := json.NewDecoder(r.Body).Decode(&certificate); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid certificate data",
			"details": err.Error(),
		})
		return
	}

	valid, err := h.service.VerifyCertificate(&certificate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Certificate verification failed",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": valid,
	})
}

// HandleCheckCertificateValidity checks if a certificate is valid
func (h *Handlers) HandleCheckCertificateValidity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var certificate fintech.CertificateOfCorrectness
	if err := json.NewDecoder(r.Body).Decode(&certificate); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid certificate data",
			"details": err.Error(),
		})
		return
	}

	isValid, reason := h.service.IsCertificateValid(&certificate)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":  isValid,
		"reason": reason,
	})
}

// HandleGetCertificateTemplates returns available certificate templates
func (h *Handlers) HandleGetCertificateTemplates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	templates := make(map[string]interface{})
	for name, template := range fintech.DefaultTemplates {
		templates[name] = map[string]interface{}{
			"name":                   template.Name,
			"description":            template.Description,
			"regulatory_framework":   template.RegulatoryFramework,
			"applicable_regulations": template.ApplicableRegulations,
			"valid_duration_days":    template.ValidDuration.Hours() / 24,
			"compliance_level":       template.ComplianceLevel,
			"revocable":              template.Revocable,
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	})
}

// === Phase 3: Trajectory Handlers ===

// HandleListTrajectories returns all trajectories with optional filtering
func (h *Handlers) HandleListTrajectories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	filter := fintech.TrajectoryFilter{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if agentID := r.URL.Query().Get("agent_id"); agentID != "" {
		filter.AgentID = agentID
	}
	if validationID := r.URL.Query().Get("validation_id"); validationID != "" {
		filter.ValidationID = validationID
	}

	trajectories, err := h.service.ListTrajectories(filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to list trajectories",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"trajectories": trajectories,
		"count":        len(trajectories),
	})
}

// HandleGetTrajectory returns a specific trajectory
func (h *Handlers) HandleGetTrajectory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	trajectory, err := h.service.GetTrajectory(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Trajectory not found",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(trajectory)
}

// HandleReplayTrajectory starts a replay of a trajectory
func (h *Handlers) HandleReplayTrajectory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	var config fintech.TrajectoryReplayConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		// Use default config if no body provided
		config = *fintech.DefaultTrajectoryReplayConfig()
	}

	session, err := h.service.ReplayTrajectory(r.Context(), id, &config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to start replay",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"replay_id":     session.ID,
		"trajectory_id": id,
		"status":        session.Status,
		"message":       "Replay started",
	})
}

// HandleCompareTrajectories compares two trajectories
func (h *Handlers) HandleCompareTrajectories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	baseID := vars["id"]

	var req struct {
		CompareTrajectoryID string `json:"compare_trajectory_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	result, err := h.service.CompareTrajectories(baseID, req.CompareTrajectoryID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to compare trajectories",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// HandleStartTrajectoryCapture starts a new trajectory capture session
func (h *Handlers) HandleStartTrajectoryCapture(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		AgentID      string                           `json:"agent_id"`
		ValidationID string                           `json:"validation_id,omitempty"`
		PID          uint32                           `json:"pid,omitempty"`
		Config       *fintech.TrajectoryCaptureConfig `json:"config,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	session, err := h.service.StartTrajectoryCapture(r.Context(), req.AgentID, req.ValidationID, req.PID, req.Config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to start capture",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"capture_id": session.ID,
		"agent_id":   req.AgentID,
		"status":     session.Status,
		"message":    "Capture started",
		"started_at": session.StartTime,
	})
}

// HandleStopTrajectoryCapture stops an active capture and returns the trajectory
func (h *Handlers) HandleStopTrajectoryCapture(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	trajectory, err := h.service.StopTrajectoryCapture(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to stop capture",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"trajectory": trajectory,
		"message":    "Capture stopped successfully",
	})
}

// HandleListActiveCaptures returns all active trajectory capture sessions
func (h *Handlers) HandleListActiveCaptures(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sessions, err := h.service.ListActiveTrajectoryCaptures()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to list active captures",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"captures": sessions,
		"count":    len(sessions),
	})
}

// HandleValidateWithDeterminism runs validation with trajectory capture and replay
func (h *Handlers) HandleValidateWithDeterminism(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		ValidationRequest
		PID uint32 `json:"pid,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	response, err := h.service.ValidateWithDeterminism(r.Context(), &req.ValidationRequest, req.PID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// === Phase 4: NRV Financial Semantic Engine Handlers ===

// HandleListNRVTraces handles GET /api/fintech/nrv/traces
func (h *Handlers) HandleListNRVTraces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get active only filter
	activeOnly := r.URL.Query().Get("active") == "true"

	traces := h.service.ListNRVTraces(activeOnly)

	response := map[string]interface{}{
		"traces": traces,
		"count":  len(traces),
	}

	json.NewEncoder(w).Encode(response)
}

// HandleGetNRVTrace handles GET /api/fintech/nrv/traces/{id}
func (h *Handlers) HandleGetNRVTrace(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	traceID := vars["id"]

	trace, err := h.service.GetNRVTrace(traceID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "NRV trace not found",
		})
		return
	}

	json.NewEncoder(w).Encode(trace)
}

// HandleScoreFidelity handles POST /api/fintech/nrv/traces/{id}/score
func (h *Handlers) HandleScoreFidelity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	traceID := vars["id"]

	var req struct {
		ExpectedOutcome string `json:"expected_outcome,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	result, err := h.service.ScoreFidelity(r.Context(), traceID, req.ExpectedOutcome)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to calculate fidelity score",
			"details": err.Error(),
		})
		return
	}

	// Validate fidelity
	valid, issues := h.service.ValidateFidelity(result)

	response := map[string]interface{}{
		"fidelity": result,
		"valid":    valid,
	}

	if !valid {
		response["issues"] = issues
	}

	json.NewEncoder(w).Encode(response)
}

// HandleCalculateSemanticDistance handles POST /api/fintech/nrv/traces/{id}/distance
func (h *Handlers) HandleCalculateSemanticDistance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	traceID := vars["id"]

	var req struct {
		ExpectedOutcome string `json:"expected_outcome,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	result, err := h.service.CalculateSemanticDistance(traceID, req.ExpectedOutcome)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to calculate semantic distance",
			"details": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}

// HandleDetectKYCBypass handles GET /api/fintech/nrv/traces/{id}/detect/kyc-bypass
func (h *Handlers) HandleDetectKYCBypass(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	traceID := vars["id"]

	indicators := h.service.DetectKYCBypass(traceID)

	response := map[string]interface{}{
		"trace_id":         traceID,
		"indicators":       indicators,
		"indicators_found": len(indicators),
		"has_violations":   len(indicators) > 0,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleDetectPositionLimitViolation handles POST /api/fintech/nrv/traces/{id}/detect/position-limits
func (h *Handlers) HandleDetectPositionLimitViolation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	traceID := vars["id"]

	var req struct {
		Limits map[string]float64 `json:"limits"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	if req.Limits == nil {
		req.Limits = make(map[string]float64)
	}

	indicators := h.service.DetectPositionLimitViolation(traceID, req.Limits)

	response := map[string]interface{}{
		"trace_id":         traceID,
		"indicators":       indicators,
		"indicators_found": len(indicators),
		"has_violations":   len(indicators) > 0,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleValidateWithNRV handles POST /api/fintech/validate/with-nrv
func (h *Handlers) HandleValidateWithNRV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		ValidationRequest
		ExpectedOutcome string `json:"expected_outcome,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	validationResponse, fidelityResult, err := h.service.ValidateWithNRV(r.Context(), &req.ValidationRequest, req.ExpectedOutcome)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Validation with NRV failed",
			"details": err.Error(),
		})
		return
	}

	response := map[string]interface{}{
		"validation": validationResponse,
	}

	if fidelityResult != nil {
		response["fidelity"] = fidelityResult

		// Add validation status
		valid, issues := h.service.ValidateFidelity(fidelityResult)
		response["fidelity_valid"] = valid
		if !valid {
			response["fidelity_issues"] = issues
		}
	}

	json.NewEncoder(w).Encode(response)
}

// === Phase 6: Tick Data Streaming Handlers ===

// HandleCreateTickStream creates a new tick data stream
func (h *Handlers) HandleCreateTickStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Symbol   string `json:"symbol"`
		TickType string `json:"tick_type"`
		MaxSize  int    `json:"max_size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	if req.Symbol == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "symbol is required",
		})
		return
	}

	if req.MaxSize <= 0 {
		req.MaxSize = 1000
	}

	stream, err := h.service.CreateTickStream(req.Symbol, req.TickType, req.MaxSize, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"symbol":    stream.Symbol,
		"tick_type": stream.TickType,
		"max_size":  stream.MaxSize,
		"message":   "Stream created successfully",
	})
}

// HandleGetTicks retrieves ticks for a symbol
func (h *Handlers) HandleGetTicks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	symbol := vars["symbol"]

	ticks, err := h.service.GetTicks(symbol)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"symbol": symbol,
		"ticks":  ticks,
		"count":  len(ticks),
	})
}

// HandleAddTick adds a tick to a stream
func (h *Handlers) HandleAddTick(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	symbol := vars["symbol"]

	var req struct {
		Price      float64 `json:"price"`
		Volume     float64 `json:"volume"`
		TickType   string  `json:"tick_type"`
		Exchange   string  `json:"exchange"`
		Bid        float64 `json:"bid"`
		Ask        float64 `json:"ask"`
		BidSize    float64 `json:"bid_size"`
		AskSize    float64 `json:"ask_size"`
		Conditions string  `json:"conditions"`
		TradeID    string  `json:"trade_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	tick := fintech.TickData{
		Symbol:     symbol,
		Timestamp:  time.Now(),
		Price:      req.Price,
		Volume:     req.Volume,
		TickType:   fintech.TickDataType(req.TickType),
		Exchange:   req.Exchange,
		Bid:        req.Bid,
		Ask:        req.Ask,
		BidSize:    req.BidSize,
		AskSize:    req.AskSize,
		Conditions: req.Conditions,
		TradeID:    req.TradeID,
	}

	if err := h.service.AddTick(tick); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"symbol": symbol,
		"status": "added",
	})
}

// HandleFlushTicks flushes buffered ticks for a symbol
func (h *Handlers) HandleFlushTicks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	symbol := vars["symbol"]

	if err := h.service.FlushTickStream(symbol); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"symbol": symbol,
		"status": "flushed",
	})
}

// HandleQueryTicks queries tick data with filters
func (h *Handlers) HandleQueryTicks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var filter fintech.TickDataFilter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	ticks, err := h.service.QueryTicks(filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ticks": ticks,
		"count": len(ticks),
	})
}

// HandleExportTicks exports tick data
func (h *Handlers) HandleExportTicks(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "symbol is required",
		})
		return
	}

	ticks, err := h.service.GetTicks(symbol)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	export, err := h.service.ExportTicks(ticks)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\"ticks_"+symbol+".arrow\"")
	w.Write(export)
}
