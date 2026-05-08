package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") || r.URL.Query().Get("stream") == "1" {
		h.streamTelemetry(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(h.snapshotPayload())
}

func (h *TelemetryHandlers) snapshotPayload() map[string]interface{} {
	telemetry := h.telemetry.Latest()
	return map[string]interface{}{
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
	}
}

func (h *TelemetryHandlers) streamTelemetry(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	writeEvent := func() error {
		payload, err := json.Marshal(h.snapshotPayload())
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: telemetry\ndata: %s\n\n", payload); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := writeEvent(); err != nil {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := writeEvent(); err != nil {
				return
			}
		}
	}
}
