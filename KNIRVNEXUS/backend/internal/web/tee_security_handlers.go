package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend-server/internal/objects"
	"backend-server/internal/services/teesecurity"
	"backend-server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type TEESecurityHandlers struct {
	teeSecurityService *teesecurity.TEESecurityService
}

func NewTEESecurityHandlers(teeSecurityService *teesecurity.TEESecurityService) *TEESecurityHandlers {
	return &TEESecurityHandlers{teeSecurityService: teeSecurityService}
}

type TEESecurityResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetTEESecurityStatus handles GET /api/tee-security
func (h *TEESecurityHandlers) GetTEESecurityStatus(w http.ResponseWriter, r *http.Request) {
	status := h.teeSecurityService.GetSecurityStatus()

	response := TEESecurityResponse{
		Success:   true,
		Data:      status,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostTEESecurityAction handles POST /api/tee-security/actions
func (h *TEESecurityHandlers) PostTEESecurityAction(w http.ResponseWriter, r *http.Request) {
	var action objects.TEESecurityAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		response := TEESecurityResponse{
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
	var err error

	switch action.Action {
	case "run_security_scan":
		err = h.teeSecurityService.RunSecurityScan()
		responseMessage = "Security scan initiated"

	case "perform_attestation":
		err = h.teeSecurityService.PerformAttestation()
		responseMessage = "TEE attestation performed"

	case "update_attestation":
		if status, ok := action.Parameters["status"].(string); ok {
			err = h.teeSecurityService.UpdateAttestationStatus(status)
			responseMessage = "Attestation status updated to " + status
		} else {
			response := TEESecurityResponse{
				Success:   false,
				Error:     "Attestation status is required",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

	case "resolve_threat":
		if threatID, ok := action.Parameters["threat_id"].(string); ok {
			err = h.teeSecurityService.ResolveThreat(threatID)
			responseMessage = "Threat " + threatID + " resolved"
		} else {
			response := TEESecurityResponse{
				Success:   false,
				Error:     "Threat ID is required",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

	default:
		response := TEESecurityResponse{
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
		response := TEESecurityResponse{
			Success:   false,
			Error:     "Failed to execute action: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get updated status
	status := h.teeSecurityService.GetSecurityStatus()

	response := TEESecurityResponse{
		Success:   true,
		Data:      status,
		Message:   responseMessage,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTEESecurityMetrics handles GET /api/tee-security/metrics
func (h *TEESecurityHandlers) GetTEESecurityMetrics(w http.ResponseWriter, r *http.Request) {
	status := h.teeSecurityService.GetSecurityStatus()

	metrics := &objects.TEESecurityMetrics{
		AttestationStatus:   status.AttestationStatus,
		SecurityScore:       status.SecurityScore,
		ThreatsDetected:     status.ThreatsDetected,
		LastAudit:           status.LastAudit,
		ActiveAttestations:  1, // Simplified for now
		ExpiredAttestations: 0,
		FailedVerifications: 0,
	}

	response := TEESecurityResponse{
		Success:   true,
		Data:      metrics,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTEESecurityThreats handles GET /api/tee-security/threats
func (h *TEESecurityHandlers) GetTEESecurityThreats(w http.ResponseWriter, r *http.Request) {
	status := h.teeSecurityService.GetSecurityStatus()

	response := TEESecurityResponse{
		Success:   true,
		Data:      status.ActiveThreats,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTEESecurityAuditHistory handles GET /api/tee-security/audit-history
func (h *TEESecurityHandlers) GetTEESecurityAuditHistory(w http.ResponseWriter, r *http.Request) {
	status := h.teeSecurityService.GetSecurityStatus()

	response := TEESecurityResponse{
		Success:   true,
		Data:      status.AuditHistory,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTEESecurityPerformance handles GET /api/tee-security/performance
func (h *TEESecurityHandlers) GetTEESecurityPerformance(w http.ResponseWriter, r *http.Request) {
	status := h.teeSecurityService.GetSecurityStatus()

	response := TEESecurityResponse{
		Success:   true,
		Data:      status.PerformanceMetrics,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostTEESecurityThreatResolve handles POST /api/tee-security/threats/{id}/resolve
func (h *TEESecurityHandlers) PostTEESecurityThreatResolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	threatID := vars["id"]

	if threatID == "" {
		response := TEESecurityResponse{
			Success:   false,
			Error:     "Threat ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.teeSecurityService.ResolveThreat(threatID)
	if err != nil {
		response := TEESecurityResponse{
			Success:   false,
			Error:     "Failed to resolve threat: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := TEESecurityResponse{
		Success:   true,
		Message:   "Threat " + threatID + " resolved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the TEE security routes with the router
func (h *TEESecurityHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for TEE security endpoints
	teeRouter := r.PathPrefix("/api/tee-security").Subrouter()

	// Public routes for monitoring (read-only)
	teeRouter.HandleFunc("", h.GetTEESecurityStatus).Methods("GET")
	teeRouter.HandleFunc("/metrics", h.GetTEESecurityMetrics).Methods("GET")
	teeRouter.HandleFunc("/threats", h.GetTEESecurityThreats).Methods("GET")
	teeRouter.HandleFunc("/audit-history", h.GetTEESecurityAuditHistory).Methods("GET")
	teeRouter.HandleFunc("/performance", h.GetTEESecurityPerformance).Methods("GET")

	// Protected routes for management
	if authMiddleware != nil {
		protectedTEERouter := teeRouter.PathPrefix("").Subrouter()
		protectedTEERouter.Use(authMiddleware.RequireAuth)
		protectedTEERouter.HandleFunc("/actions", h.PostTEESecurityAction).Methods("POST")
		protectedTEERouter.HandleFunc("/threats/{id}/resolve", h.PostTEESecurityThreatResolve).Methods("POST")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		teeRouter.HandleFunc("/actions", h.PostTEESecurityAction).Methods("POST")
		teeRouter.HandleFunc("/threats/{id}/resolve", h.PostTEESecurityThreatResolve).Methods("POST")
	}

	// Handle OPTIONS requests for CORS
	teeRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
