package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/services/guardrails"

	"github.com/gorilla/mux"
)

// policyEventEmitter is the minimal interface needed to broadcast policy events.
type policyEventEmitter interface {
	EmitPolicyUpdate(policyID, source, policyType string, rules map[string]interface{})
}

type GuardrailHandlers struct {
	guardrailManager *guardrails.DynamicGuardrailManager
	policyEngine     *guardrails.PolicyEngine
	eventBroadcaster policyEventEmitter
}

func NewGuardrailHandlers(gm *guardrails.DynamicGuardrailManager, pe *guardrails.PolicyEngine) *GuardrailHandlers {
	return &GuardrailHandlers{
		guardrailManager: gm,
		policyEngine:     pe,
	}
}

// SetEventBroadcaster wires the broadcaster used to push policy:update events
// to connected WebSocket clients after a policy is saved or committed.
func (h *GuardrailHandlers) SetEventBroadcaster(eb policyEventEmitter) {
	h.eventBroadcaster = eb
}

func (h *GuardrailHandlers) RegisterRoutes(r *mux.Router) {
	guardrailRouter := r.PathPrefix("/api/guardrails").Subrouter()

	guardrailRouter.HandleFunc("/configure", h.ConfigureGuardrail).Methods("POST", "OPTIONS")
	guardrailRouter.HandleFunc("/config/{nodeID}/{guardrailType}", h.GetGuardrailConfig).Methods("GET", "OPTIONS")
	guardrailRouter.HandleFunc("/validate", h.ValidateResourceUsage).Methods("POST", "OPTIONS")
	guardrailRouter.HandleFunc("/validate/ontology", h.ValidateOntology).Methods("POST", "OPTIONS")
	guardrailRouter.HandleFunc("/violations", h.ListViolations).Methods("GET", "OPTIONS")
	guardrailRouter.HandleFunc("/violations/{id}/resolve", h.ResolveViolation).Methods("POST", "OPTIONS")
	guardrailRouter.HandleFunc("/statistics", h.GetStatistics).Methods("GET", "OPTIONS")

	guardrailRouter.HandleFunc("/ontology/configure", h.ConfigureOntology).Methods("POST", "OPTIONS")

	if h.policyEngine != nil {
		guardrailRouter.HandleFunc("/policies", h.ListPolicies).Methods("GET", "OPTIONS")
		guardrailRouter.HandleFunc("/policies", h.CreatePolicy).Methods("POST", "OPTIONS")
		guardrailRouter.HandleFunc("/policies/{id}", h.GetPolicy).Methods("GET", "OPTIONS")
		guardrailRouter.HandleFunc("/policies/{id}", h.UpdatePolicy).Methods("PUT", "OPTIONS")
		guardrailRouter.HandleFunc("/policies/{id}", h.DeletePolicy).Methods("DELETE", "OPTIONS")
		guardrailRouter.HandleFunc("/policies/{id}/commit", h.CommitPolicy).Methods("POST", "OPTIONS")
		guardrailRouter.HandleFunc("/policies/{id}/evaluate", h.EvaluatePolicy).Methods("POST", "OPTIONS")
	}
}

func (h *GuardrailHandlers) ConfigureGuardrail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID string                      `json:"node_id"`
		Config *guardrails.GuardrailConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Config == nil || req.NodeID == "" {
		http.Error(w, `{"error":"node_id and config required"}`, http.StatusBadRequest)
		return
	}

	if err := h.guardrailManager.ConfigureGuardrail(req.NodeID, req.Config); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "configured",
		"node_id": req.NodeID,
		"config":  req.Config,
	})
}

func (h *GuardrailHandlers) GetGuardrailConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeID"]
	guardrailType := guardrails.GuardrailType(vars["guardrailType"])

	config, ok := h.guardrailManager.GetGuardrailConfig(nodeID, guardrailType)
	if !ok {
		http.Error(w, `{"error":"configuration not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(config)
}

func (h *GuardrailHandlers) ValidateResourceUsage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID  string                 `json:"node_id"`
		Metrics map[string]interface{} `json:"metrics"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.NodeID == "" || req.Metrics == nil {
		http.Error(w, `{"error":"node_id and metrics required"}`, http.StatusBadRequest)
		return
	}

	violation, isViolated := h.guardrailManager.ValidateResourceUsage(req.NodeID, req.Metrics)
	if isViolated {
		h.guardrailManager.RecordViolation(violation)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"violated":  true,
			"violation": violation,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"violated": false,
	})
}

func (h *GuardrailHandlers) ValidateOntology(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID  string `json:"node_id"`
		Domain  string `json:"domain"`
		Concept string `json:"concept"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.NodeID == "" || req.Domain == "" || req.Concept == "" {
		http.Error(w, `{"error":"node_id, domain, and concept required"}`, http.StatusBadRequest)
		return
	}

	allowed, reason := h.guardrailManager.ValidateOntologyConstraint(req.NodeID, req.Domain, req.Concept)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed": allowed,
		"reason":  reason,
	})
}

func (h *GuardrailHandlers) ConfigureOntology(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string                        `json:"domain"`
		Rule   *guardrails.OntologyGuardrail `json:"rule"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Domain == "" || req.Rule == nil {
		http.Error(w, `{"error":"domain and rule required"}`, http.StatusBadRequest)
		return
	}

	if err := h.guardrailManager.ConfigureOntologyGuardrail(req.Domain, req.Rule); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "configured",
		"domain": req.Domain,
	})
}

func (h *GuardrailHandlers) ListViolations(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	limit := 100

	violations := h.guardrailManager.GetViolations(nodeID, limit)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"violations": violations,
		"count":      len(violations),
	})
}

func (h *GuardrailHandlers) ResolveViolation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	violationID := vars["id"]

	if err := h.guardrailManager.ResolveViolation(violationID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "resolved",
		"violation_id": violationID,
		"resolved_at":  time.Now().Format(time.RFC3339),
	})
}

func (h *GuardrailHandlers) GetStatistics(w http.ResponseWriter, r *http.Request) {
	stats := h.guardrailManager.GetStatistics()
	json.NewEncoder(w).Encode(stats)
}

func (h *GuardrailHandlers) ListPolicies(w http.ResponseWriter, r *http.Request) {
	if h.policyEngine == nil {
		http.Error(w, `{"error":"policy engine not available"}`, http.StatusServiceUnavailable)
		return
	}

	policies := h.policyEngine.ListPolicies()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"policies": policies,
		"count":    len(policies),
	})
}

func (h *GuardrailHandlers) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	if h.policyEngine == nil {
		http.Error(w, `{"error":"policy engine not available"}`, http.StatusServiceUnavailable)
		return
	}

	var policy guardrails.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.policyEngine.CreatePolicy(&policy); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if h.eventBroadcaster != nil {
		h.eventBroadcaster.EmitPolicyUpdate(policy.ID, "guardrail_handler", "created", map[string]interface{}{
			"name":  policy.Name,
			"rules": len(policy.Rules),
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "created",
		"policy": &policy,
	})
}

func (h *GuardrailHandlers) GetPolicy(w http.ResponseWriter, r *http.Request) {
	if h.policyEngine == nil {
		http.Error(w, `{"error":"policy engine not available"}`, http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	policyID := vars["id"]

	policy, ok := h.policyEngine.GetPolicy(policyID)
	if !ok {
		http.Error(w, `{"error":"policy not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(policy)
}

func (h *GuardrailHandlers) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	if h.policyEngine == nil {
		http.Error(w, `{"error":"policy engine not available"}`, http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	policyID := vars["id"]

	var policy guardrails.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	policy.ID = policyID
	policy.UpdatedAt = time.Now()

	if err := h.policyEngine.CreatePolicy(&policy); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "updated",
		"policy": &policy,
	})
}

func (h *GuardrailHandlers) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	if h.policyEngine == nil {
		http.Error(w, `{"error":"policy engine not available"}`, http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	policyID := vars["id"]

	if err := h.policyEngine.DeletePolicy(policyID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "deleted",
		"policy_id": policyID,
	})
}

func (h *GuardrailHandlers) CommitPolicy(w http.ResponseWriter, r *http.Request) {
	if h.policyEngine == nil {
		http.Error(w, `{"error":"policy engine not available"}`, http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	policyID := vars["id"]

	txHash, err := h.policyEngine.CommitPolicyToBlockchain(policyID)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	if h.eventBroadcaster != nil {
		h.eventBroadcaster.EmitPolicyUpdate(policyID, "guardrail_handler", "committed", map[string]interface{}{
			"tx_hash": txHash,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "committed",
		"policy_id": policyID,
		"tx_hash":   txHash,
	})
}

func (h *GuardrailHandlers) EvaluatePolicy(w http.ResponseWriter, r *http.Request) {
	if h.policyEngine == nil {
		http.Error(w, `{"error":"policy engine not available"}`, http.StatusServiceUnavailable)
		return
	}

	var req struct {
		NodeID  string                 `json:"node_id"`
		Action  string                 `json:"action"`
		Context map[string]interface{} `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.NodeID == "" || req.Action == "" {
		http.Error(w, `{"error":"node_id and action required"}`, http.StatusBadRequest)
		return
	}

	allowed, reason := h.policyEngine.EvaluateAction(req.NodeID, req.Action, req.Context)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed": allowed,
		"reason":  reason,
	})
}
