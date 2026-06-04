package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/services/identitybridge"
	"backend_server/internal/services/mcphardening"
	"backend_server/internal/services/policyadapter"
	"backend_server/internal/services/reliability"

	"github.com/gorilla/mux"
)

type GovernanceHandlers struct {
	identityBridge  *identitybridge.IdentityBridge
	policyAdapter   *policyadapter.PolicyAdapter
	reliabilityCtrl *reliability.ReliabilityController
	mcpGateway      *mcphardening.MCPGateway
}

func NewGovernanceHandlers(
	ib *identitybridge.IdentityBridge,
	pa *policyadapter.PolicyAdapter,
	rc *reliability.ReliabilityController,
	mcp *mcphardening.MCPGateway,
) *GovernanceHandlers {
	return &GovernanceHandlers{
		identityBridge:  ib,
		policyAdapter:   pa,
		reliabilityCtrl: rc,
		mcpGateway:      mcp,
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
