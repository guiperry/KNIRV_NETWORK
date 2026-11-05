package routes

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleClearReplicaDivergence is POST /oracle/v3/replication/clear-divergence.
// It deliberately accepts no remote head or state. The caller can only
// acknowledge the hash of the local signed head retained at fork detection.
func (r *OracleRoutes) handleClearReplicaDivergence(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireOperatorToken(w, req) {
		return
	}
	var body struct {
		ExpectedHeadHash string `json:"expected_head_hash"`
		Acknowledge      bool   `json:"acknowledge"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 4096)).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request"})
		return
	}
	if !body.Acknowledge || strings.TrimSpace(body.ExpectedHeadHash) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "acknowledge and expected_head_hash are required"})
		return
	}
	if err := r.oracle.ClearReplicaDivergence(body.ExpectedHeadHash); err != nil {
		respondJSON(w, http.StatusConflict, map[string]interface{}{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"cleared": true, "paused": true})
}
