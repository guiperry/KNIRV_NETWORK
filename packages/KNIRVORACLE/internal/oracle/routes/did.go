package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/did"
)

func (r *OracleRoutes) handleDIDRegister(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var doc did.DIDDocument
	if err := json.NewDecoder(req.Body).Decode(&doc); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	resolver := r.oracle.GetDIDResolver()
	if err := resolver.Register(&doc); err != nil {
		http.Error(w, fmt.Sprintf("Registration failed: %v", err), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusCreated, doc)
}

func (r *OracleRoutes) handleDID(w http.ResponseWriter, req *http.Request) {
	didStr := strings.TrimPrefix(req.URL.Path, "/oracle/v3/did/")
	if didStr == "" {
		http.Error(w, "DID is required", http.StatusBadRequest)
		return
	}

	if strings.HasSuffix(didStr, "/deactivate") {
		r.handleDIDDeactivate(w, req, strings.TrimSuffix(didStr, "/deactivate"))
		return
	}

	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resolver := r.oracle.GetDIDResolver()
	doc, err := resolver.Resolve(didStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Resolution failed: %v", err), http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, doc)
}

func (r *OracleRoutes) handleDIDDeactivate(w http.ResponseWriter, req *http.Request, didStr string) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resolver := r.oracle.GetDIDResolver()
	if err := resolver.Deactivate(didStr); err != nil {
		http.Error(w, fmt.Sprintf("Deactivation failed: %v", err), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "deactivated",
		"deactivated_at": time.Now().UTC(),
	})
}

func (r *OracleRoutes) RegisterDIDRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/oracle/v3/did/register", r.handleDIDRegister)
	mux.HandleFunc("/oracle/v3/did/", r.handleDID)
}
