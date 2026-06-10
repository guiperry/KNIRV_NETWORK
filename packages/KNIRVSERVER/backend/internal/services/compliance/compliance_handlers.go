package compliance

import (
	"encoding/json"
	"net/http"
)

type ComplianceHandlers struct {
	correlator       *Correlator
	frameworkManager *FrameworkManager
}

func NewComplianceHandlers(c *Correlator, fm *FrameworkManager) *ComplianceHandlers {
	return &ComplianceHandlers{
		correlator:       c,
		frameworkManager: fm,
	}
}

func (ch *ComplianceHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/compliance/v1/events", ch.handleEvents)
	mux.HandleFunc("/compliance/v1/events/", ch.handleNodeEvents)
	mux.HandleFunc("/compliance/v1/frameworks", ch.handleFrameworks)
	mux.HandleFunc("/compliance/v1/frameworks/", ch.handleFramework)
	mux.HandleFunc("/compliance/v1/status", ch.handleStatus)
	mux.HandleFunc("/compliance/v1/chain/verify", ch.handleChainVerify)
}

func (ch *ComplianceHandlers) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			NodeID    string                 `json:"node_id"`
			AgentID   string                 `json:"agent_id"`
			EventType string                 `json:"event_type"`
			Severity  string                 `json:"severity"`
			Details   map[string]interface{} `json:"details"`
		}
		if err := decodeJSON(r, &req); err != nil {
			respondError(w, err.Error(), http.StatusBadRequest)
			return
		}
		event := ch.correlator.RecordEvent(req.NodeID, req.AgentID, req.EventType, req.Severity, req.Details)
		respondJSON(w, event)
		return
	}
	events := ch.correlator.GetEvents("", 100)
	respondJSON(w, map[string]interface{}{"events": events, "count": len(events)})
}

func (ch *ComplianceHandlers) handleNodeEvents(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Path[len("/compliance/v1/events/"):]
	events := ch.correlator.GetEvents(nodeID, 100)
	respondJSON(w, map[string]interface{}{"events": events, "count": len(events)})
}

func (ch *ComplianceHandlers) handleFrameworks(w http.ResponseWriter, r *http.Request) {
	frameworks := ch.frameworkManager.ListFrameworks()
	respondJSON(w, map[string]interface{}{"frameworks": frameworks, "count": len(frameworks)})
}

func (ch *ComplianceHandlers) handleFramework(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/compliance/v1/frameworks/"):]
	fw, ok := ch.frameworkManager.GetFramework(FrameworkID(id))
	if !ok {
		respondError(w, "framework not found", http.StatusNotFound)
		return
	}
	respondJSON(w, fw)
}

func (ch *ComplianceHandlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		respondError(w, "node_id query param required", http.StatusBadRequest)
		return
	}
	status := ch.frameworkManager.GetNodeCompliance(nodeID)
	respondJSON(w, status)
}

func (ch *ComplianceHandlers) handleChainVerify(w http.ResponseWriter, r *http.Request) {
	valid := ch.correlator.VerifyChain()
	respondJSON(w, map[string]interface{}{
		"valid":   valid,
		"tip":     ch.correlator.TipHash(),
		"entries": len(ch.correlator.events),
	})
}

func decodeJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}
	return nil
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
