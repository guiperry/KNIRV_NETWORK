package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/services/compliance"
	"backend_server/internal/services/identitybridge"
	"backend_server/internal/services/mcphardening"
	"backend_server/internal/services/policyadapter"
	"backend_server/internal/services/reliability"

	"github.com/gorilla/mux"
)

type GovernanceHandlers struct {
	identityBridge    *identitybridge.IdentityBridge
	policyAdapter     *policyadapter.PolicyAdapter
	reliabilityCtrl   *reliability.ReliabilityController
	mcpGateway        *mcphardening.MCPGateway
	schemaValidator   *mcphardening.SchemaValidator
	injectionDetector *mcphardening.InjectionDetector
	responseSanitizer *mcphardening.ResponseSanitizer
	sloBudget         *reliability.SLOBudget
	dveBindingManager *reliability.DVEBindingManager
	escalationManager *reliability.EscalationManager
	correlator        *compliance.Correlator
	complianceFm      *compliance.FrameworkManager
}

func NewGovernanceHandlers(
	ib *identitybridge.IdentityBridge,
	pa *policyadapter.PolicyAdapter,
	rc *reliability.ReliabilityController,
	mcp *mcphardening.MCPGateway,
	sv *mcphardening.SchemaValidator,
	id *mcphardening.InjectionDetector,
	rs *mcphardening.ResponseSanitizer,
	sb *reliability.SLOBudget,
	dbm *reliability.DVEBindingManager,
	em *reliability.EscalationManager,
	corr *compliance.Correlator,
	cf *compliance.FrameworkManager,
) *GovernanceHandlers {
	return &GovernanceHandlers{
		identityBridge:    ib,
		policyAdapter:     pa,
		reliabilityCtrl:   rc,
		mcpGateway:        mcp,
		schemaValidator:   sv,
		injectionDetector: id,
		responseSanitizer: rs,
		sloBudget:         sb,
		dveBindingManager: dbm,
		escalationManager: em,
		correlator:        corr,
		complianceFm:      cf,
	}
}

func (gh *GovernanceHandlers) RegisterRoutes(apiV1 *mux.Router) {
	govRouter := apiV1.PathPrefix("/governance").Subrouter()

	identityRouter := govRouter.PathPrefix("/identity").Subrouter()
	identityRouter.HandleFunc("/envelopes", gh.CreateEnvelope).Methods("POST", "OPTIONS")
	identityRouter.HandleFunc("/envelopes/{id}", gh.GetEnvelope).Methods("GET", "OPTIONS")
	identityRouter.HandleFunc("/envelopes/{id}", gh.RevokeEnvelope).Methods("DELETE", "OPTIONS")
	identityRouter.HandleFunc("/envelopes/{id}/refresh", gh.RefreshEnvelope).Methods("POST", "OPTIONS")
	identityRouter.HandleFunc("/envelopes/{id}/trust", gh.UpdateTrustScore).Methods("PUT", "OPTIONS")
	identityRouter.HandleFunc("/envelopes/{id}/attributes", gh.UpdateAttributes).Methods("PUT", "OPTIONS")
	identityRouter.HandleFunc("/envelopes/{id}/federate", gh.FederateIdentity).Methods("POST", "OPTIONS")
	identityRouter.HandleFunc("/mappings", gh.CreateMapping).Methods("POST", "OPTIONS")
	identityRouter.HandleFunc("/mappings", gh.ListMappings).Methods("GET", "OPTIONS")
	identityRouter.HandleFunc("/mappings/{id}", gh.GetMapping).Methods("GET", "OPTIONS")
	identityRouter.HandleFunc("/stats", gh.IdentityStats).Methods("GET", "OPTIONS")

	policyRouter := govRouter.PathPrefix("/policy").Subrouter()
	policyRouter.HandleFunc("/contract", gh.GetContract).Methods("GET", "OPTIONS")
	policyRouter.HandleFunc("/inputs", gh.NormalizeInput).Methods("POST", "OPTIONS")
	policyRouter.HandleFunc("/inputs", gh.ListInputs).Methods("GET", "OPTIONS")
	policyRouter.HandleFunc("/enforcements", gh.ListEnforcements).Methods("GET", "OPTIONS")
	policyRouter.HandleFunc("/enforcements/{id}", gh.GetEnforcement).Methods("GET", "OPTIONS")
	policyRouter.HandleFunc("/stats", gh.PolicyStats).Methods("GET", "OPTIONS")

	reliabilityRouter := govRouter.PathPrefix("/reliability").Subrouter()
	reliabilityRouter.HandleFunc("/breakers", gh.ListBreakers).Methods("GET", "OPTIONS")
	reliabilityRouter.HandleFunc("/breakers/{id}/success", gh.RecordBreakerSuccess).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/breakers/{id}/failure", gh.RecordBreakerFailure).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/breakers/{id}/allow", gh.IsBreakerAllowed).Methods("GET", "OPTIONS")
	reliabilityRouter.HandleFunc("/breakers/{id}/reset", gh.ResetBreaker).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/budgets", gh.GetBudget).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/budgets/{id}/consume", gh.ConsumeBudget).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/budgets/{id}/error", gh.RecordBudgetError).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/kill-switches", gh.ArmKillSwitch).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/kill-switches/list", gh.ListKillSwitches).Methods("GET", "OPTIONS")
	reliabilityRouter.HandleFunc("/kill-switches/{agent}/{node}/trip", gh.TripKillSwitch).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/kill-switches/{agent}/{node}/disarm", gh.DisarmKillSwitch).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/escalation", gh.SetupEscalation).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/breaches", gh.ListBreaches).Methods("GET", "OPTIONS")
	reliabilityRouter.HandleFunc("/breaches/{id}/resolve", gh.ResolveBreach).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/stats", gh.ReliabilityStats).Methods("GET", "OPTIONS")

	mcpRouter := govRouter.PathPrefix("/mcp").Subrouter()
	mcpRouter.HandleFunc("/tools", gh.RegisterTool).Methods("POST", "OPTIONS")
	mcpRouter.HandleFunc("/tools", gh.ListTools).Methods("GET", "OPTIONS")
	mcpRouter.HandleFunc("/tools/{name}", gh.GetTool).Methods("GET", "OPTIONS")
	mcpRouter.HandleFunc("/endpoints", gh.RegisterEndpoint).Methods("POST", "OPTIONS")
	mcpRouter.HandleFunc("/endpoints", gh.ListEndpoints).Methods("GET", "OPTIONS")
	mcpRouter.HandleFunc("/call", gh.ProcessToolCall).Methods("POST", "OPTIONS")
	mcpRouter.HandleFunc("/audit", gh.GetAuditLog).Methods("GET", "OPTIONS")
	mcpRouter.HandleFunc("/signatures", gh.RegisterSignature).Methods("POST", "OPTIONS")
	mcpRouter.HandleFunc("/stats", gh.MCPStats).Methods("GET", "OPTIONS")
	mcpRouter.HandleFunc("/schema/validate", gh.ValidateToolSchema).Methods("POST", "OPTIONS")
	mcpRouter.HandleFunc("/injection/scan", gh.ScanInjection).Methods("POST", "OPTIONS")
	mcpRouter.HandleFunc("/response/sanitize", gh.SanitizeResponse).Methods("POST", "OPTIONS")

	reliabilityRouter.HandleFunc("/slos", gh.ListSLOs).Methods("GET", "OPTIONS")
	reliabilityRouter.HandleFunc("/slos/define", gh.DefineSLO).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/slos/{name}/status", gh.GetSLOStatus).Methods("GET", "OPTIONS")
	reliabilityRouter.HandleFunc("/slos/{name}/metric", gh.RecordSLO).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/dve-bindings", gh.ListDVEBindings).Methods("GET", "OPTIONS")
	reliabilityRouter.HandleFunc("/dve-bindings/bind", gh.BindDVE).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/dve-bindings/{breaker}/unbind", gh.UnbindDVE).Methods("DELETE", "OPTIONS")
	reliabilityRouter.HandleFunc("/escalation/routes", gh.ListEscalationRoutes).Methods("GET", "OPTIONS")
	reliabilityRouter.HandleFunc("/escalation/routes/add", gh.AddEscalationRoute).Methods("POST", "OPTIONS")
	reliabilityRouter.HandleFunc("/escalation/history", gh.GetEscalationHistory).Methods("GET", "OPTIONS")

	identityRouter.HandleFunc("/revoke", gh.RevokeNode).Methods("POST", "OPTIONS")
	identityRouter.HandleFunc("/revoked/{nodeID}", gh.CheckRevoked).Methods("GET", "OPTIONS")

	complianceRouter := govRouter.PathPrefix("/compliance").Subrouter()
	complianceRouter.HandleFunc("/events", gh.RecordComplianceEvent).Methods("POST", "OPTIONS")
	complianceRouter.HandleFunc("/events", gh.ListComplianceEvents).Methods("GET", "OPTIONS")
	complianceRouter.HandleFunc("/events/{nodeID}", gh.ListNodeComplianceEvents).Methods("GET", "OPTIONS")
	complianceRouter.HandleFunc("/frameworks", gh.ListComplianceFrameworks).Methods("GET", "OPTIONS")
	complianceRouter.HandleFunc("/frameworks/{id}", gh.GetComplianceFramework).Methods("GET", "OPTIONS")
	complianceRouter.HandleFunc("/status", gh.GetComplianceStatus).Methods("GET", "OPTIONS")
	complianceRouter.HandleFunc("/chain/verify", gh.VerifyComplianceChain).Methods("GET", "OPTIONS")
}

func (gh *GovernanceHandlers) CreateEnvelope(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID   string `json:"node_id"`
		AgentID  string `json:"agent_id,omitempty"`
		Source   string `json:"source"`
		NodeName string `json:"node_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.NodeID == "" || req.Source == "" {
		http.Error(w, `{"error":"node_id and source required"}`, http.StatusBadRequest)
		return
	}
	attrs := &identitybridge.IdentityAttributes{
		NodeID:      req.NodeID,
		AgentID:     req.AgentID,
		Environment: "production",
		Labels:      map[string]string{"name": req.NodeName},
	}
	env := gh.identityBridge.CreateEnvelope(req.NodeID, req.AgentID, identitybridge.IdentitySource(req.Source), attrs)
	json.NewEncoder(w).Encode(env)
}

func (gh *GovernanceHandlers) GetEnvelope(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	env, ok := gh.identityBridge.GetEnvelope(vars["id"])
	if !ok {
		http.Error(w, `{"error":"envelope not found or expired"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(env)
}

func (gh *GovernanceHandlers) RevokeEnvelope(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := gh.identityBridge.RevokeEnvelope(vars["id"]); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}

func (gh *GovernanceHandlers) RefreshEnvelope(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	env, err := gh.identityBridge.RefreshEnvelope(vars["id"])
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(env)
}

func (gh *GovernanceHandlers) UpdateTrustScore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req struct {
		Delta float64 `json:"delta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	env, err := gh.identityBridge.UpdateTrustScore(vars["id"], req.Delta)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(env)
}

func (gh *GovernanceHandlers) UpdateAttributes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var attrs identitybridge.IdentityAttributes
	if err := json.NewDecoder(r.Body).Decode(&attrs); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if err := gh.identityBridge.UpdateAttributes(vars["id"], &attrs); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (gh *GovernanceHandlers) FederateIdentity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req struct {
		FederationID string `json:"federation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if err := gh.identityBridge.FederateIdentity(vars["id"], req.FederationID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "federated"})
}

func (gh *GovernanceHandlers) CreateMapping(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InternalID     string            `json:"internal_id"`
		ExternalID     string            `json:"external_id"`
		ExternalSource string            `json:"external_source"`
		MappingType    string            `json:"mapping_type"`
		Metadata       map[string]string `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.InternalID == "" || req.ExternalID == "" || req.ExternalSource == "" {
		http.Error(w, `{"error":"internal_id, external_id, and external_source required"}`, http.StatusBadRequest)
		return
	}
	mapping := gh.identityBridge.CreateMapping(req.InternalID, req.ExternalID, req.ExternalSource, req.MappingType, req.Metadata)
	json.NewEncoder(w).Encode(mapping)
}

func (gh *GovernanceHandlers) ListMappings(w http.ResponseWriter, r *http.Request) {
	internalID := r.URL.Query().Get("internal_id")
	mappings := gh.identityBridge.ListMappings(internalID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mappings": mappings,
		"count":    len(mappings),
	})
}

func (gh *GovernanceHandlers) GetMapping(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	externalSource := r.URL.Query().Get("source")
	mapping, ok := gh.identityBridge.GetMapping(vars["id"], externalSource)
	if !ok {
		http.Error(w, `{"error":"mapping not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(mapping)
}

func (gh *GovernanceHandlers) IdentityStats(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(gh.identityBridge.GetStatistics())
}

func (gh *GovernanceHandlers) GetContract(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(gh.policyAdapter.GetContract())
}

func (gh *GovernanceHandlers) NormalizeInput(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID     string                 `json:"node_id"`
		Action     string                 `json:"action"`
		ActionType string                 `json:"action_type"`
		Metrics    map[string]float64     `json:"metrics,omitempty"`
		Context    map[string]interface{} `json:"context,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.NodeID == "" || req.Action == "" || req.ActionType == "" {
		http.Error(w, `{"error":"node_id, action, and action_type required"}`, http.StatusBadRequest)
		return
	}
	input := gh.policyAdapter.NormalizeInput(req.NodeID, req.Action, req.ActionType, req.Metrics, req.Context)
	json.NewEncoder(w).Encode(input)
}

func (gh *GovernanceHandlers) ListInputs(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	inputs := gh.policyAdapter.ListInputs(nodeID, 100)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"inputs": inputs,
		"count":  len(inputs),
	})
}

func (gh *GovernanceHandlers) ListEnforcements(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	enforcements := gh.policyAdapter.ListEnforcements(nodeID, 100)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enforcements": enforcements,
		"count":        len(enforcements),
	})
}

func (gh *GovernanceHandlers) GetEnforcement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	enf, ok := gh.policyAdapter.GetEnforcement(vars["id"])
	if !ok {
		http.Error(w, `{"error":"enforcement not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(enf)
}

func (gh *GovernanceHandlers) PolicyStats(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(gh.policyAdapter.GetStatistics())
}

func (gh *GovernanceHandlers) ListBreakers(w http.ResponseWriter, r *http.Request) {
	breakers := gh.reliabilityCtrl.ListBreakers()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"breakers": breakers,
		"count":    len(breakers),
	})
}

func (gh *GovernanceHandlers) RecordBreakerSuccess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gh.reliabilityCtrl.RecordSuccess(vars["id"])
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded", "type": "success"})
}

func (gh *GovernanceHandlers) RecordBreakerFailure(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gh.reliabilityCtrl.RecordFailure(vars["id"])
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded", "type": "failure"})
}

func (gh *GovernanceHandlers) IsBreakerAllowed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	allowed := gh.reliabilityCtrl.IsRequestAllowed(vars["id"])
	json.NewEncoder(w).Encode(map[string]interface{}{
		"breaker_id": vars["id"],
		"allowed":    allowed,
		"state":      gh.reliabilityCtrl.GetBreakerState(vars["id"]),
	})
}

func (gh *GovernanceHandlers) ResetBreaker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gh.reliabilityCtrl.ResetBreaker(vars["id"])
	json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
}

func (gh *GovernanceHandlers) GetBudget(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BudgetID string `json:"budget_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	status := gh.reliabilityCtrl.GetBudgetStatus(req.BudgetID)
	json.NewEncoder(w).Encode(status)
}

func (gh *GovernanceHandlers) ConsumeBudget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	remaining, ok := gh.reliabilityCtrl.ConsumeBudget(vars["id"])
	json.NewEncoder(w).Encode(map[string]interface{}{
		"remaining": remaining,
		"allowed":   ok,
	})
}

func (gh *GovernanceHandlers) RecordBudgetError(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gh.reliabilityCtrl.RecordBudgetError(vars["id"])
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

func (gh *GovernanceHandlers) ArmKillSwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID  string `json:"agent_id"`
		NodeID   string `json:"node_id"`
		AutoReset string `json:"auto_reset,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.AgentID == "" || req.NodeID == "" {
		http.Error(w, `{"error":"agent_id and node_id required"}`, http.StatusBadRequest)
		return
	}
	autoReset := 0 * time.Second
	if req.AutoReset != "" {
		d, err := time.ParseDuration(req.AutoReset)
		if err == nil {
			autoReset = d
		}
	}
	ks := gh.reliabilityCtrl.ArmKillSwitch(req.AgentID, req.NodeID, autoReset)
	json.NewEncoder(w).Encode(ks)
}

func (gh *GovernanceHandlers) ListKillSwitches(w http.ResponseWriter, r *http.Request) {
	switches := gh.reliabilityCtrl.ListKillSwitches()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"kill_switches": switches,
		"count":         len(switches),
	})
}

func (gh *GovernanceHandlers) TripKillSwitch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req struct {
		Reason    string `json:"reason"`
		TrippedBy string `json:"tripped_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	event, err := gh.reliabilityCtrl.TripKillSwitch(vars["agent"], vars["node"], req.Reason, req.TrippedBy)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(event)
}

func (gh *GovernanceHandlers) DisarmKillSwitch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := gh.reliabilityCtrl.DisarmKillSwitch(vars["agent"], vars["node"]); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "disarmed"})
}

func (gh *GovernanceHandlers) SetupEscalation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID            string  `json:"node_id"`
		WarningThreshold  float64 `json:"warning_threshold"`
		CriticalThreshold float64 `json:"critical_threshold"`
		ShutdownThreshold float64 `json:"shutdown_threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	policy := reliability.EscalationPolicy{
		WarningThreshold:  req.WarningThreshold,
		CriticalThreshold: req.CriticalThreshold,
		ShutdownThreshold: req.ShutdownThreshold,
		CooldownPeriod:    5 * time.Minute,
		AutoReset:         true,
	}
	if policy.WarningThreshold == 0 {
		policy.WarningThreshold = 0.5
	}
	if policy.CriticalThreshold == 0 {
		policy.CriticalThreshold = 0.75
	}
	if policy.ShutdownThreshold == 0 {
		policy.ShutdownThreshold = 0.95
	}
	gh.reliabilityCtrl.SetupEscalation(req.NodeID, policy)
	json.NewEncoder(w).Encode(map[string]string{"status": "configured"})
}

func (gh *GovernanceHandlers) ListBreaches(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	events := gh.reliabilityCtrl.ListBreachEvents(agentID, 100)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (gh *GovernanceHandlers) ResolveBreach(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := gh.reliabilityCtrl.ResolveBreachEvent(vars["id"]); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "resolved"})
}

func (gh *GovernanceHandlers) ReliabilityStats(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(gh.reliabilityCtrl.GetStatistics())
}

func (gh *GovernanceHandlers) RegisterTool(w http.ResponseWriter, r *http.Request) {
	var tool mcphardening.ToolDefinition
	if err := json.NewDecoder(r.Body).Decode(&tool); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if tool.Name == "" {
		http.Error(w, `{"error":"tool name required"}`, http.StatusBadRequest)
		return
	}
	gh.mcpGateway.Validator().RegisterTool(&tool)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered", "name": tool.Name})
}

func (gh *GovernanceHandlers) ListTools(w http.ResponseWriter, r *http.Request) {
	tools := gh.mcpGateway.Validator().ListTools()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	})
}

func (gh *GovernanceHandlers) GetTool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tool, ok := gh.mcpGateway.Validator().GetTool(vars["name"])
	if !ok {
		http.Error(w, `{"error":"tool not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(tool)
}

func (gh *GovernanceHandlers) RegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string              `json:"name"`
		EP   mcphardening.MCPEndpoint `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}
	gh.mcpGateway.RegisterEndpoint(req.Name, &req.EP)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func (gh *GovernanceHandlers) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	eps := gh.mcpGateway.ListEndpoints()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"endpoints": eps,
		"count":     len(eps),
	})
}

func (gh *GovernanceHandlers) ProcessToolCall(w http.ResponseWriter, r *http.Request) {
	var record mcphardening.ToolCallRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if record.AgentID == "" || record.ToolName == "" {
		http.Error(w, `{"error":"agent_id and tool_name required"}`, http.StatusBadRequest)
		return
	}
	if record.ID == "" {
		record.ID = r.Header.Get("X-Request-ID")
	}
	status := gh.mcpGateway.ProcessToolCall(&record)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"reason":  record.Reason,
		"record":  record,
	})
}

func (gh *GovernanceHandlers) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	log := gh.mcpGateway.GetAuditLog(agentID, 100)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"audit_log": log,
		"count":     len(log),
	})
}

func (gh *GovernanceHandlers) RegisterSignature(w http.ResponseWriter, r *http.Request) {
	var sig mcphardening.PoisoningSignature
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if sig.Pattern == "" {
		http.Error(w, `{"error":"pattern required"}`, http.StatusBadRequest)
		return
	}
	gh.mcpGateway.Detector().RegisterSignature(sig)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
}

func (gh *GovernanceHandlers) MCPStats(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(gh.mcpGateway.GetStatistics())
}

func (gh *GovernanceHandlers) ValidateToolSchema(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToolName string                 `json:"tool_name"`
		Args     map[string]interface{} `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	valid, reason := gh.schemaValidator.ValidateArgs(req.ToolName, req.Args)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":  valid,
		"reason": reason,
	})
}

func (gh *GovernanceHandlers) ScanInjection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Args map[string]interface{} `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	flagged := gh.injectionDetector.ScanArguments(req.Args)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"flagged": flagged,
		"count":   len(flagged),
	})
}

func (gh *GovernanceHandlers) SanitizeResponse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	sanitized, modified := gh.responseSanitizer.Sanitize(req.Content)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sanitized": sanitized,
		"modified":  modified,
	})
}

func (gh *GovernanceHandlers) DefineSLO(w http.ResponseWriter, r *http.Request) {
	var config reliability.SLOConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	gh.sloBudget.DefineSLO(config)
	json.NewEncoder(w).Encode(map[string]string{"status": "defined"})
}

func (gh *GovernanceHandlers) ListSLOs(w http.ResponseWriter, r *http.Request) {
	slos := gh.sloBudget.ListSLOs()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"slos":  slos,
		"count": len(slos),
	})
}

func (gh *GovernanceHandlers) GetSLOStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	status := gh.sloBudget.GetStatus(vars["name"])
	if status == nil {
		http.Error(w, `{"error":"SLO not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(status)
}

func (gh *GovernanceHandlers) RecordSLO(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req struct {
		Value float64 `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	gh.sloBudget.RecordMetric(vars["name"], req.Value)
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

func (gh *GovernanceHandlers) BindDVE(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DVEID     string `json:"dve_id"`
		NodeID    string `json:"node_id"`
		AgentID   string `json:"agent_id"`
		BreakerID string `json:"breaker_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	binding := gh.dveBindingManager.Bind(req.DVEID, req.NodeID, req.AgentID, req.BreakerID)
	json.NewEncoder(w).Encode(binding)
}

func (gh *GovernanceHandlers) ListDVEBindings(w http.ResponseWriter, r *http.Request) {
	bindings := gh.dveBindingManager.ListBindings()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bindings": bindings,
		"count":    len(bindings),
	})
}

func (gh *GovernanceHandlers) UnbindDVE(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gh.dveBindingManager.Unbind(vars["breaker"])
	json.NewEncoder(w).Encode(map[string]string{"status": "unbound"})
}

func (gh *GovernanceHandlers) AddEscalationRoute(w http.ResponseWriter, r *http.Request) {
	var route reliability.EscalationRoute
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	gh.escalationManager.AddRoute(route)
	json.NewEncoder(w).Encode(map[string]string{"status": "added"})
}

func (gh *GovernanceHandlers) ListEscalationRoutes(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	if level != "" {
		routes := gh.escalationManager.GetRoutes(reliability.EscalationLevel(level))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"routes": routes,
			"count":  len(routes),
		})
		return
	}
	var all []reliability.EscalationRoute
	for _, l := range []reliability.EscalationLevel{
		reliability.EscalationLevelWarning,
		reliability.EscalationLevelCritical,
		reliability.EscalationLevelShutdown,
	} {
		all = append(all, gh.escalationManager.GetRoutes(l)...)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"routes": all,
		"count":  len(all),
	})
}

func (gh *GovernanceHandlers) GetEscalationHistory(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	history := gh.escalationManager.GetHistory(nodeID, 100)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

func (gh *GovernanceHandlers) RevokeNode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IdentityID string `json:"identity_id"`
		NodeID     string `json:"node_id"`
		Reason     string `json:"reason"`
		RevokedBy  string `json:"revoked_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	entry := gh.identityBridge.RevocationList().Revoke(req.IdentityID, req.NodeID, req.Reason, req.RevokedBy)
	json.NewEncoder(w).Encode(entry)
}

func (gh *GovernanceHandlers) CheckRevoked(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	revoked := gh.identityBridge.RevocationList().IsRevoked(vars["nodeID"])
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id": vars["nodeID"],
		"revoked": revoked,
	})
}

func (gh *GovernanceHandlers) RecordComplianceEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID    string                 `json:"node_id"`
		AgentID   string                 `json:"agent_id"`
		EventType string                 `json:"event_type"`
		Severity  string                 `json:"severity"`
		Details   map[string]interface{} `json:"details"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	event := gh.correlator.RecordEvent(req.NodeID, req.AgentID, req.EventType, req.Severity, req.Details)
	json.NewEncoder(w).Encode(event)
}

func (gh *GovernanceHandlers) ListComplianceEvents(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	events := gh.correlator.GetEvents(nodeID, 100)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (gh *GovernanceHandlers) ListNodeComplianceEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	events := gh.correlator.GetEvents(vars["nodeID"], 100)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (gh *GovernanceHandlers) ListComplianceFrameworks(w http.ResponseWriter, r *http.Request) {
	frameworks := gh.complianceFm.ListFrameworks()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"frameworks": frameworks,
		"count":      len(frameworks),
	})
}

func (gh *GovernanceHandlers) GetComplianceFramework(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fw, ok := gh.complianceFm.GetFramework(compliance.FrameworkID(vars["id"]))
	if !ok {
		http.Error(w, `{"error":"framework not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(fw)
}

func (gh *GovernanceHandlers) GetComplianceStatus(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		http.Error(w, `{"error":"node_id query param required"}`, http.StatusBadRequest)
		return
	}
	status := gh.complianceFm.GetNodeCompliance(nodeID)
	json.NewEncoder(w).Encode(status)
}

func (gh *GovernanceHandlers) VerifyComplianceChain(w http.ResponseWriter, r *http.Request) {
	valid := gh.correlator.VerifyChain()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": valid,
		"tip":   gh.correlator.TipHash(),
	})
}
