package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"backend_server/internal/services"
	"go.uber.org/zap"
)

type DVEProxyHandler struct {
	uriRegistry *services.DVEURIRegistry
	logger     *zap.Logger
}

func NewDVEProxyHandler(registry *services.DVEURIRegistry, logger *zap.Logger) *DVEProxyHandler {
	return &DVEProxyHandler{
		uriRegistry: registry,
		logger:     logger,
	}
}

func (h *DVEProxyHandler) RegisterRoutes(router Router) {
	router.HandleFunc("/dve/{dveId}", h.handleDVEPublicPage)
	router.HandleFunc("/dve/{dveId}/validation", h.handleValidationRecords)
	router.HandleFunc("/dve/{dveId}/metrics", h.handleDVEMetrics)
	router.HandleFunc("/dve/{dveId}/search", h.handleSearch)
	
	router.HandleFunc("/api/dve/resolve", h.handleResolveURI)
	router.HandleFunc("/api/dve/register", h.handleRegisterDVE)
	router.HandleFunc("/api/dve/{dveId}", h.handleGetDVE)
}

func (h *DVEProxyHandler) handleDVEPublicPage(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}

	dve, err := h.uriRegistry.GetByDVEID(dveID)
	if err != nil {
		h.logger.Warn("DVE not found", zap.String("dve_id", dveID), zap.Error(err))
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"dve_id":    dve.DVEID,
		"full_uri": dve.FullURI,
		"status":   dve.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *DVEProxyHandler) handleValidationRecords(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"dve_id":    dveID,
		"records":  []interface{}{},
		"total":   0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *DVEProxyHandler) handleDVEMetrics(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"dve_id":   dveID,
		"metrics": map[string]interface{}{
			"total_validations": 0,
			"success_rate":     0.0,
			"average_score":   0.0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *DVEProxyHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"query": query,
		"results": []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *DVEProxyHandler) handleResolveURI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URI string `json:"uri"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	dve, err := h.uriRegistry.Resolve(req.URI)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve URI: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dve)
}

func (h *DVEProxyHandler) handleRegisterDVE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DVEID      string `json:"dve_id"`
		FullURI    string `json:"full_uri"`
		WalletAddr string `json:"wallet_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	dve := &services.DVEURI{
		DVEID:      req.DVEID,
		FullURI:    req.FullURI,
		WalletAddr: req.WalletAddr,
	}

	if err := h.uriRegistry.Register(dve); err != nil {
		http.Error(w, fmt.Sprintf("Failed to register DVE: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dve_id":  req.DVEID,
	})
}

func (h *DVEProxyHandler) handleGetDVE(w http.ResponseWriter, r *http.Request) {
	dveID := extractURLParam(r, "dveId")
	if dveID == "" {
		http.Error(w, "DVE ID required", http.StatusBadRequest)
		return
	}

	dve, err := h.uriRegistry.GetByDVEID(dveID)
	if err != nil {
		http.Error(w, "DVE not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dve)
}

type Router interface {
	HandleFunc(pattern string, handler http.HandlerFunc)
}

func extractURLParam(r *http.Request, name string) string {
	return r.PathValue(name)
}