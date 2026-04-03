package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"backend_server/internal/services/rollup"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type RollupHandlers struct {
	rollupService *rollup.Service
	pollInterval  time.Duration
}

type RollupResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

type RollupStatusPayload struct {
	Enabled      bool                 `json:"enabled"`
	PollInterval string               `json:"poll_interval"`
	Service      rollup.ServiceStatus `json:"service"`
}

func NewRollupHandlers(rollupService *rollup.Service, pollInterval time.Duration) *RollupHandlers {
	return &RollupHandlers{
		rollupService: rollupService,
		pollInterval:  pollInterval,
	}
}

func (h *RollupHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	rollupRouter := r.PathPrefix("/api/rollups").Subrouter()

	if authMiddleware != nil {
		rollupRouter.Use(authMiddleware.RequireAuth)
	}

	rollupRouter.HandleFunc("/status", h.GetStatus).Methods("GET", "OPTIONS")
	rollupRouter.HandleFunc("", h.ListBatches).Methods("GET", "OPTIONS")
	rollupRouter.HandleFunc("/{id}", h.GetBatch).Methods("GET", "OPTIONS")
}

func (h *RollupHandlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	if h.rollupService == nil {
		writeRollupResponse(w, http.StatusServiceUnavailable, RollupResponse{
			Success:   false,
			Error:     "Rollup service not available",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	writeRollupResponse(w, http.StatusOK, RollupResponse{
		Success: true,
		Data: RollupStatusPayload{
			Enabled:      true,
			PollInterval: h.pollInterval.String(),
			Service:      h.rollupService.GetStatus(),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

func (h *RollupHandlers) ListBatches(w http.ResponseWriter, r *http.Request) {
	if h.rollupService == nil {
		writeRollupResponse(w, http.StatusServiceUnavailable, RollupResponse{
			Success:   false,
			Error:     "Rollup service not available",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	writeRollupResponse(w, http.StatusOK, RollupResponse{
		Success:   true,
		Data:      h.rollupService.ListBatches(limit),
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

func (h *RollupHandlers) GetBatch(w http.ResponseWriter, r *http.Request) {
	if h.rollupService == nil {
		writeRollupResponse(w, http.StatusServiceUnavailable, RollupResponse{
			Success:   false,
			Error:     "Rollup service not available",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	batchID := mux.Vars(r)["id"]
	batch, ok := h.rollupService.GetBatch(batchID)
	if !ok {
		writeRollupResponse(w, http.StatusNotFound, RollupResponse{
			Success:   false,
			Error:     "Rollup batch not found",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	writeRollupResponse(w, http.StatusOK, RollupResponse{
		Success:   true,
		Data:      batch,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

func writeRollupResponse(w http.ResponseWriter, statusCode int, response RollupResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}
