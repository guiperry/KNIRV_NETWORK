package operator

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handler represents the HTTP handler for operator registry
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new operator registry handler
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers the operator registry routes with the router
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Health check endpoint
	router.HandleFunc("/operator-registry/api/health", h.handleHealth).Methods("GET")

	// Operator endpoints
	router.HandleFunc("/operator-registry/api/operators", h.handleGetAllOperators).Methods("GET")
	router.HandleFunc("/operator-registry/api/operators/{id}", h.handleGetOperatorByID).Methods("GET")
	router.HandleFunc("/operator-registry/api/operators", h.handleRegisterOperator).Methods("POST")

	// Legacy endpoints for compatibility
	router.HandleFunc("/operator-registry/", h.handleGetAllOperators).Methods("GET")
	router.HandleFunc("/operator-registry/register", h.handleRegisterOperator).Methods("POST")
}

// handleHealth handles the health check endpoint
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := h.service.GetHealth()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error("Failed to encode health response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleGetAllOperators handles getting all operators
func (h *Handler) handleGetAllOperators(w http.ResponseWriter, r *http.Request) {
	operators, err := h.service.GetAllOperators()
	if err != nil {
		h.logger.Error("Failed to get all operators", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(operators); err != nil {
		h.logger.Error("Failed to encode operators response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleGetOperatorByID handles getting an operator by ID
func (h *Handler) handleGetOperatorByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		http.Error(w, "Operator ID is required", http.StatusBadRequest)
		return
	}

	operator, err := h.service.GetOperatorByID(id)
	if err != nil {
		h.logger.Warn("Operator not found", zap.String("id", id), zap.Error(err))
		http.Error(w, "Operator not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(operator); err != nil {
		h.logger.Error("Failed to encode operator response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleRegisterOperator handles registering a new operator
func (h *Handler) handleRegisterOperator(w http.ResponseWriter, r *http.Request) {
	var req RegisterOperatorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode register operator request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.service.RegisterNewOperator(&req)
	if err != nil {
		h.logger.Error("Failed to register new operator", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode register response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
