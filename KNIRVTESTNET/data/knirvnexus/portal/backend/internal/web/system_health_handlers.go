package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"nexus-backend/internal/models"
	"nexus-backend/internal/services/systemhealth"
	"nexus-backend/internal/web/middleware"

	"github.com/gorilla/mux"
)

type SystemHealthHandlers struct {
	systemHealthService *systemhealth.SystemHealthService
}

func NewSystemHealthHandlers(systemHealthService *systemhealth.SystemHealthService) *SystemHealthHandlers {
	return &SystemHealthHandlers{systemHealthService: systemHealthService}
}

type SystemHealthResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetSystemHealth handles GET /api/system-health
func (h *SystemHealthHandlers) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	detailed := r.URL.Query().Get("detailed") == "true"
	
	health := h.systemHealthService.GetSystemHealth(detailed)

	response := SystemHealthResponse{
		Success:   true,
		Data:      health,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSystemAlerts handles GET /api/system-health/alerts
func (h *SystemHealthHandlers) GetSystemAlerts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	var resolved *bool
	if resolvedStr := r.URL.Query().Get("resolved"); resolvedStr != "" {
		if resolvedBool, err := strconv.ParseBool(resolvedStr); err == nil {
			resolved = &resolvedBool
		}
	}
	
	severity := r.URL.Query().Get("severity")
	
	alerts := h.systemHealthService.GetAlerts(resolved, severity)

	response := SystemHealthResponse{
		Success:   true,
		Data:      alerts,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostSystemHealthAction handles POST /api/system-health/actions
func (h *SystemHealthHandlers) PostSystemHealthAction(w http.ResponseWriter, r *http.Request) {
	var action models.SystemHealthAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		response := SystemHealthResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var responseMessage string
	var responseData interface{}
	var err error

	switch action.Action {
	case "run_diagnostics":
		responseData = h.systemHealthService.RunDiagnostics()
		responseMessage = "System diagnostics completed"

	case "add_alert":
		if severity, ok := action.Parameters["severity"].(string); ok {
			if component, ok := action.Parameters["component"].(string); ok {
				if message, ok := action.Parameters["message"].(string); ok {
					metadata, _ := action.Parameters["metadata"].(map[string]interface{})
					alert := h.systemHealthService.AddAlert(severity, component, message, metadata)
					responseData = alert
					responseMessage = "Alert added successfully"
				} else {
					response := SystemHealthResponse{
						Success:   false,
						Error:     "Alert message is required",
						Timestamp: time.Now().Format(time.RFC3339),
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(response)
					return
				}
			} else {
				response := SystemHealthResponse{
					Success:   false,
					Error:     "Alert component is required",
					Timestamp: time.Now().Format(time.RFC3339),
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(response)
				return
			}
		} else {
			response := SystemHealthResponse{
				Success:   false,
				Error:     "Alert severity is required",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

	case "resolve_alert":
		if alertID, ok := action.Parameters["alert_id"].(string); ok {
			err = h.systemHealthService.ResolveAlert(alertID)
			responseMessage = "Alert " + alertID + " resolved"
		} else {
			response := SystemHealthResponse{
				Success:   false,
				Error:     "Alert ID is required",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

	default:
		response := SystemHealthResponse{
			Success:   false,
			Error:     "Invalid action: " + action.Action,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != nil {
		response := SystemHealthResponse{
			Success:   false,
			Error:     "Failed to execute action: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := SystemHealthResponse{
		Success:   true,
		Data:      responseData,
		Message:   responseMessage,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostSystemHealthAlertResolve handles POST /api/system-health/alerts/{id}/resolve
func (h *SystemHealthHandlers) PostSystemHealthAlertResolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	if alertID == "" {
		response := SystemHealthResponse{
			Success:   false,
			Error:     "Alert ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.systemHealthService.ResolveAlert(alertID)
	if err != nil {
		response := SystemHealthResponse{
			Success:   false,
			Error:     "Failed to resolve alert: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := SystemHealthResponse{
		Success:   true,
		Message:   "Alert " + alertID + " resolved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSystemHealthMetrics handles GET /api/system-health/metrics
func (h *SystemHealthHandlers) GetSystemHealthMetrics(w http.ResponseWriter, r *http.Request) {
	health := h.systemHealthService.GetSystemHealth(true)

	response := SystemHealthResponse{
		Success:   true,
		Data:      health.Metrics,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSystemHealthComponents handles GET /api/system-health/components
func (h *SystemHealthHandlers) GetSystemHealthComponents(w http.ResponseWriter, r *http.Request) {
	health := h.systemHealthService.GetSystemHealth(true)

	response := SystemHealthResponse{
		Success:   true,
		Data:      health.Components,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the system health routes with the router
func (h *SystemHealthHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for system health endpoints
	healthRouter := r.PathPrefix("/api/system-health").Subrouter()

	// Public routes for monitoring (read-only)
	healthRouter.HandleFunc("", h.GetSystemHealth).Methods("GET")
	healthRouter.HandleFunc("/alerts", h.GetSystemAlerts).Methods("GET")
	healthRouter.HandleFunc("/metrics", h.GetSystemHealthMetrics).Methods("GET")
	healthRouter.HandleFunc("/components", h.GetSystemHealthComponents).Methods("GET")

	// Protected routes for management
	if authMiddleware != nil {
		protectedHealthRouter := healthRouter.PathPrefix("").Subrouter()
		protectedHealthRouter.Use(authMiddleware.RequireAuth)
		protectedHealthRouter.HandleFunc("/actions", h.PostSystemHealthAction).Methods("POST")
		protectedHealthRouter.HandleFunc("/alerts/{id}/resolve", h.PostSystemHealthAlertResolve).Methods("POST")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		healthRouter.HandleFunc("/actions", h.PostSystemHealthAction).Methods("POST")
		healthRouter.HandleFunc("/alerts/{id}/resolve", h.PostSystemHealthAlertResolve).Methods("POST")
	}

	// Handle OPTIONS requests for CORS
	healthRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
