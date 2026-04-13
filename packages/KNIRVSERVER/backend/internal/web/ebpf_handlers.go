package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/services/cognitiveengine"

	"github.com/gorilla/mux"
)

type EBPFService interface {
	IsNodeIsolated(nodeID string) bool
	LatestTelemetry() *cognitiveengine.SystemResourceSnapshot
	GetXDPFilters() []*cognitiveengine.XDPFilter
	GetResourceQuotas() []*cognitiveengine.ResourceQuota
}

type EBPFHandlers struct {
	ebpf EBPFService
}

func NewEBPFHandlers(ebpf EBPFService) *EBPFHandlers {
	return &EBPFHandlers{
		ebpf: ebpf,
	}
}

func (h *EBPFHandlers) RegisterRoutes(r *mux.Router) {
	ebpfRouter := r.PathPrefix("/api/ebpf").Subrouter()

	ebpfRouter.HandleFunc("/isolation-status", h.GetIsolationStatus).Methods("GET", "OPTIONS")
	ebpfRouter.HandleFunc("/filters", h.ListFilters).Methods("GET", "OPTIONS")
	ebpfRouter.HandleFunc("/quotas", h.ListQuotas).Methods("GET", "OPTIONS")
}

func (h *EBPFHandlers) GetIsolationStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	nodeID := r.URL.Query().Get("node_id")

	if nodeID != "" {
		isolated := h.ebpf.IsNodeIsolated(nodeID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"node_id":   nodeID,
			"isolated":  isolated,
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"isolation_status": "not_implemented",
		"timestamp":        time.Now().Format(time.RFC3339),
		"note":             "Use node_id parameter to check specific node isolation status",
	})
}

func (h *EBPFHandlers) ListFilters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	filters := h.ebpf.GetXDPFilters()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"filters": filters,
		"count":   len(filters),
	})
}

func (h *EBPFHandlers) ListQuotas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	quotas := h.ebpf.GetResourceQuotas()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"quotas": quotas,
		"count":  len(quotas),
	})
}
