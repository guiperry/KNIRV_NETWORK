package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/services/cognitiveengine"

	"github.com/gorilla/mux"
)

type TelemetryService interface {
	Latest() *cognitiveengine.SystemResourceSnapshot
}

type TelemetryHandlers struct {
	telemetry TelemetryService
}

func NewTelemetryHandlers(telemetry TelemetryService) *TelemetryHandlers {
	return &TelemetryHandlers{
		telemetry: telemetry,
	}
}

func (h *TelemetryHandlers) RegisterRoutes(r *mux.Router) {
	telemetryRouter := r.PathPrefix("/api/cognitive").Subrouter()

	telemetryRouter.HandleFunc("/telemetry", h.GetTelemetry).Methods("GET", "OPTIONS")
}

func (h *TelemetryHandlers) GetTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	telemetry := h.telemetry.Latest()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"timestamp":        telemetry.Timestamp.Format(time.RFC3339),
		"cpu_time_ns":      telemetry.TotalCPUTimeNs,
		"memory_bytes":     telemetry.TotalMemoryBytes,
		"net_tx_bytes":     telemetry.TotalNetTxBytes,
		"net_rx_bytes":     telemetry.TotalNetRxBytes,
		"context_switches": telemetry.ContextSwitches,
		"page_faults":      telemetry.PageFaults,
		"goroutines":       telemetry.GoRoutines,
		"heap_alloc_bytes": telemetry.HeapAllocBytes,
		"gc_count":         telemetry.NumGC,
		"cpu_pressure":     telemetry.CPUPressure,
		"memory_pressure":  telemetry.MemoryPressure,
	})
}
