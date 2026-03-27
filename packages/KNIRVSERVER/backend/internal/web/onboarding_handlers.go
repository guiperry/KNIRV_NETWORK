package web

import (
	"encoding/json"
	"net/http"

	"backend_server/internal/services/onboarding"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

// OnboardingHandlers provides HTTP handlers for onboarding
type OnboardingHandlers struct {
	onboardingService *onboarding.OnboardingService
}

// NewOnboardingHandlers creates new onboarding handlers
func NewOnboardingHandlers(onboardingService *onboarding.OnboardingService) *OnboardingHandlers {
	return &OnboardingHandlers{
		onboardingService: onboardingService,
	}
}

// RegisterRoutes registers onboarding routes
func (h *OnboardingHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	onboardingRouter := r.PathPrefix("/api/onboarding").Subrouter()

	// Protected routes
	if authMiddleware != nil {
		protectedRouter := onboardingRouter.PathPrefix("").Subrouter()
		protectedRouter.Use(authMiddleware.RequireAuth)
		protectedRouter.HandleFunc("/organizations", h.OnboardOrganization).Methods("POST", "OPTIONS")
		protectedRouter.HandleFunc("/organizations", h.ListConfigurations).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/organizations/{id}", h.GetConfiguration).Methods("GET", "OPTIONS")
		protectedRouter.HandleFunc("/organizations/{id}", h.UpdateConfiguration).Methods("PUT", "OPTIONS")
		protectedRouter.HandleFunc("/organizations/{id}", h.DeleteConfiguration).Methods("DELETE", "OPTIONS")
		protectedRouter.HandleFunc("/organizations/{id}/validate", h.ValidateAction).Methods("POST", "OPTIONS")
	} else {
		// No auth required for testnet mode
		onboardingRouter.HandleFunc("/organizations", h.OnboardOrganization).Methods("POST", "OPTIONS")
		onboardingRouter.HandleFunc("/organizations", h.ListConfigurations).Methods("GET", "OPTIONS")
		onboardingRouter.HandleFunc("/organizations/{id}", h.GetConfiguration).Methods("GET", "OPTIONS")
		onboardingRouter.HandleFunc("/organizations/{id}", h.UpdateConfiguration).Methods("PUT", "OPTIONS")
		onboardingRouter.HandleFunc("/organizations/{id}", h.DeleteConfiguration).Methods("DELETE", "OPTIONS")
		onboardingRouter.HandleFunc("/organizations/{id}/validate", h.ValidateAction).Methods("POST", "OPTIONS")
	}
}

// OnboardOrganization handles POST /api/onboarding/organizations
func (h *OnboardingHandlers) OnboardOrganization(w http.ResponseWriter, r *http.Request) {
	var req onboarding.OnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.OrganizationID == "" {
		http.Error(w, `{"error":"organization_id is required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.onboardingService.OnboardOrganization(r.Context(), &req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ListConfigurations handles GET /api/onboarding/organizations
func (h *OnboardingHandlers) ListConfigurations(w http.ResponseWriter, r *http.Request) {
	configs := h.onboardingService.ListConfigurations()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"configurations": configs,
		"total":          len(configs),
	})
}

// GetConfiguration handles GET /api/onboarding/organizations/{id}
func (h *OnboardingHandlers) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	configID := vars["id"]

	config, err := h.onboardingService.GetConfiguration(configID)
	if err != nil {
		http.Error(w, `{"error":"configuration not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateConfiguration handles PUT /api/onboarding/organizations/{id}
func (h *OnboardingHandlers) UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	configID := vars["id"]

	var req onboarding.OnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	config, err := h.onboardingService.UpdateConfiguration(configID, &req)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// DeleteConfiguration handles DELETE /api/onboarding/organizations/{id}
func (h *OnboardingHandlers) DeleteConfiguration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	configID := vars["id"]

	if err := h.onboardingService.DeleteConfiguration(configID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "deleted",
		"config_id": configID,
	})
}

// ValidateAction handles POST /api/onboarding/organizations/{id}/validate
func (h *OnboardingHandlers) ValidateAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	configID := vars["id"]

	var req struct {
		Action map[string]interface{} `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	allowed, reason, err := h.onboardingService.ValidateAction(r.Context(), configID, req.Action)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed": allowed,
		"reason":  reason,
	})
}
