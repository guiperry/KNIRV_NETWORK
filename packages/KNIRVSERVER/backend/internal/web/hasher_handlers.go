package web

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// HasherHandlers provides HTTP handlers that bridge to the KNIRVHASHER gRPC service.
// These endpoints are mounted at /api/v1/hasher/* in the backend server
// and proxied through the gateway at /api/hasher/*.
type HasherHandlers struct {
	logger          *zap.Logger
	hasherAvailable func() bool
	hasherPing      func() error
}

// NewHasherHandlers creates a new HasherHandlers.
// hasherAvailable is a callback that returns whether the hasher gRPC connection
// is active. hasherPing is a callback that performs a quick health check against
// the hasher gRPC service.
func NewHasherHandlers(logger *zap.Logger, hasherAvailable func() bool, hasherPing func() error) *HasherHandlers {
	return &HasherHandlers{
		logger:          logger,
		hasherAvailable: hasherAvailable,
		hasherPing:      hasherPing,
	}
}

// RegisterRoutes registers hasher HTTP routes on the given router.
func (h *HasherHandlers) RegisterRoutes(r *http.ServeMux, prefix string) {
	if prefix == "" {
		prefix = "/api/v1/hasher"
	}
	http.HandleFunc(prefix+"/status", h.HandleStatus)
	http.HandleFunc(prefix+"/ping", h.HandlePing)
}

// HandleStatus returns the hasher service status.
func (h *HasherHandlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	available := false
	if h.hasherAvailable != nil {
		available = h.hasherAvailable()
	}

	status := "unavailable"
	if available {
		status = "available"
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status,
		"available": available,
	})
}

// HandlePing performs a health check against the hasher gRPC service.
func (h *HasherHandlers) HandlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.hasherPing == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": "hasher ping not configured",
		})
		return
	}

	if err := h.hasherPing(); err != nil {
		h.logger.Debug("Hasher ping failed", zap.Error(err))
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "hasher reachable",
	})
}
