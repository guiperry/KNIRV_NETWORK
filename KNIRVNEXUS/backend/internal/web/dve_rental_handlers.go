package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/container"
	"backend_server/internal/services/dverental"
	"backend_server/internal/services/endpoints"
	"backend_server/internal/services/session"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
	"github.com/tidwall/buntdb"
)

// DVERentalHandlers handles DVE rental API requests
type DVERentalHandlers struct {
	dveRentalService      *dverental.DVERentalService
	containerOrchestrator *container.ContainerOrchestrator
	sessionManager        *session.SessionManager
	endpointRegistry      *endpoints.EndpointRegistry
	db                    *database.BuntDBManager
}

// NewDVERentalHandlers creates new DVE rental handlers
func NewDVERentalHandlers(dveRentalService *dverental.DVERentalService, containerOrchestrator *container.ContainerOrchestrator, sessionManager *session.SessionManager, endpointRegistry *endpoints.EndpointRegistry, db *database.BuntDBManager) *DVERentalHandlers {
	return &DVERentalHandlers{
		dveRentalService:      dveRentalService,
		containerOrchestrator: containerOrchestrator,
		sessionManager:        sessionManager,
		endpointRegistry:      endpointRegistry,
		db:                    db,
	}
}

// DVERentalResponse represents a standard API response for DVE rental operations
type DVERentalResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetRentalPlans handles GET /api/dve-rental/plans
func (h *DVERentalHandlers) GetRentalPlans(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DVE Rental] GetRentalPlans called")
	plans, err := h.dveRentalService.GetRentalPlans()
	if err != nil {
		log.Printf("[DVE Rental] Error fetching plans: %v", err)
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to fetch rental plans: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("[DVE Rental] Retrieved %d plans: %+v", len(plans), plans)
	response := DVERentalResponse{
		Success:   true,
		Data:      plans,
		Message:   "Rental plans retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateRental handles POST /api/dve-rental/rentals
func (h *DVERentalHandlers) CreateRental(w http.ResponseWriter, r *http.Request) {
	var req objects.RentalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Extract user ID from JWT token
	if req.UserID == "" {
		req.UserID = middleware.GetUserIDFromRequest(r)
	}
	// For development/testing, fallback to generated ID
	if req.UserID == "" {
		req.UserID = "test-user-" + time.Now().Format("20060102150405")
	}

	rentalResponse, err := h.dveRentalService.CreateRental(&req)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to create rental: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if !rentalResponse.Success {
		response := DVERentalResponse{
			Success:   false,
			Error:     rentalResponse.Error,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      rentalResponse,
		Message:   "DVE rental created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetUserRentals handles GET /api/dve-rental/rentals
func (h *DVERentalHandlers) GetUserRentals(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	rentals, err := h.dveRentalService.GetActiveRentals(userID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to fetch user rentals: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      rentals,
		Message:   "User rentals retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRentalStats handles GET /api/dve-rental/stats
func (h *DVERentalHandlers) GetRentalStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.dveRentalService.GetRentalStats()
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to fetch rental stats: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      stats,
		Message:   "Rental statistics retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ExtendRental handles POST /api/dve-rental/rentals/{id}/extend
func (h *DVERentalHandlers) ExtendRental(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	var req struct {
		AdditionalDuration int64  `json:"additional_duration"`
		PaymentTxHash      string `json:"payment_tx_hash"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.dveRentalService.ExtendRental(rentalID, req.AdditionalDuration, req.PaymentTxHash)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to extend rental: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Message:   "Rental extended successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelRental handles DELETE /api/dve-rental/rentals/{id}
func (h *DVERentalHandlers) CancelRental(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	err := h.dveRentalService.CancelRental(rentalID, userID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to cancel rental: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Message:   "Rental cancelled successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetFullAccessInfo handles GET /api/dve-rental/rentals/{id}/full-access-info
func (h *DVERentalHandlers) GetFullAccessInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Extract user ID from JWT token (middleware should have validated auth)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		// Fallback to query parameter for development
		userID = r.URL.Query().Get("user_id")
		if userID == "" {
			response := DVERentalResponse{
				Success:   false,
				Error:     "Authentication required",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Validate rental access with comprehensive checks
	if err := h.validateRentalAccess(rentalID, userID); err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get rental details (already validated)
	rental, _ := h.dveRentalService.GetRentalByID(rentalID)

	// Get SSH endpoint from registry
	sshEndpoint, err := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "ssh")
	sshInfo := map[string]interface{}{
		"endpoint": "localhost",
		"port":     rental.SSHPort,
		"username": rental.SSHUsername,
		"command":  fmt.Sprintf("ssh -i key.pem %s@localhost -p %d", rental.SSHUsername, rental.SSHPort),
	}
	if err == nil && sshEndpoint != nil {
		sshInfo["endpoint"] = sshEndpoint.Host
		sshInfo["port"] = sshEndpoint.Port
		sshInfo["protocol"] = sshEndpoint.Protocol
	}

	// Get validation endpoint from registry
	validationEndpoint, err := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "validation")
	validationInfo := map[string]interface{}{
		"endpoint_url":  fmt.Sprintf("http://localhost:%d", 23145),
		"session_token": "placeholder-token",
		"expires_at":    time.Now().Add(24 * time.Hour),
	}
	if err == nil && validationEndpoint != nil {
		validationInfo["endpoint_url"] = fmt.Sprintf("%s://%s:%d", validationEndpoint.Protocol, validationEndpoint.Host, validationEndpoint.Port)
		// Get session token from session manager
		if rental.ValidationSessionID != "" {
			if session, err := h.sessionManager.GetValidationSession(rental.ValidationSessionID); err == nil && session != nil {
				validationInfo["session_token"] = session.SessionToken
				validationInfo["expires_at"] = session.ExpiresAt
				validationInfo["session_id"] = session.ID
				validationInfo["validation_type"] = session.ValidationType
			}
		}
	}

	// Get error resolution endpoint from registry
	errorResEndpoint, err := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "error-resolution")
	errorResInfo := map[string]interface{}{
		"endpoint_url":  fmt.Sprintf("http://localhost:%d", 24145),
		"session_token": "placeholder-token",
		"expires_at":    time.Now().Add(24 * time.Hour),
	}
	if err == nil && errorResEndpoint != nil {
		errorResInfo["endpoint_url"] = fmt.Sprintf("%s://%s:%d", errorResEndpoint.Protocol, errorResEndpoint.Host, errorResEndpoint.Port)
		// Get session token from session manager
		if rental.ErrorResSessionID != "" {
			if session, err := h.sessionManager.GetErrorResolutionSession(rental.ErrorResSessionID); err == nil && session != nil {
				errorResInfo["session_token"] = session.SessionToken
				errorResInfo["expires_at"] = session.ExpiresAt
				errorResInfo["session_id"] = session.ID
				errorResInfo["supported_error_types"] = session.SupportedTypes
			}
		}
	}

	// Get container status
	containerStatus := "unknown"
	if rental.ContainerID != "" && h.containerOrchestrator != nil {
		if status, err := h.containerOrchestrator.GetContainerStatus(rental.ContainerID); err == nil {
			containerStatus = string(status)
		}
	}

	// Build full access info response
	accessInfo := map[string]interface{}{
		"rental": map[string]interface{}{
			"id":                  rental.ID,
			"status":              rental.Status,
			"provisioning_status": rental.ProvisioningStatus,
			"end_time":            rental.EndTime,
			"start_time":          rental.StartTime,
		},
		"ssh":                  sshInfo,
		"reasoning_validation": validationInfo,
		"error_resolution":     errorResInfo,
		"container_info": map[string]interface{}{
			"container_id": rental.ContainerID,
			"status":       containerStatus,
			"allocated_resources": map[string]interface{}{
				"cpu":    rental.ResourceLimits.MaxCPU,
				"memory": fmt.Sprintf("%.1fGB", float64(rental.ResourceLimits.MaxMemory)/(1024*1024*1024)),
				"disk":   fmt.Sprintf("%.1fGB", float64(rental.ResourceLimits.MaxDisk)/(1024*1024*1024)),
			},
		},
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      accessInfo,
		Message:   "Full access information retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateSSHSession handles POST /api/dve-rental/rentals/{id}/ssh-session
func (h *DVERentalHandlers) CreateSSHSession(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	vars := mux.Vars(r)
	rentalID := vars["id"]

	// Extract user ID from JWT token (middleware should have validated auth)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		// Fallback to query parameter for development
		userID = r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "test-user-default"
		}
	}

	// Log access attempt
	h.logAccessAttempt(&AccessLogEntry{
		ID:          fmt.Sprintf("ssh_create_%d", time.Now().UnixNano()),
		UserID:      userID,
		RentalID:    rentalID,
		Action:      "create_session",
		ServiceType: "ssh",
		IPAddress:   r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		Success:     false, // Will be updated
		Timestamp:   startTime,
	})

	if rentalID == "" {
		h.logError(&ErrorLogEntry{
			ID:        fmt.Sprintf("error_%d", time.Now().UnixNano()),
			UserID:    userID,
			RentalID:  rentalID,
			Endpoint:  r.URL.Path,
			Error:     "Rental ID is required",
			Severity:  "low",
			Timestamp: time.Now(),
		})

		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if rental is active
	if rental.Status != "active" || time.Now().After(rental.EndTime) {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental is not active",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get SSH private key from container orchestrator
	privateKey, err := h.containerOrchestrator.GetSSHPrivateKey(rental.ContainerID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to retrieve SSH private key: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create SSH session using session manager
	sshSession, err := h.sessionManager.CreateSSHSession(rentalID, rental.ContainerID, rental.SSHUsername, privateKey)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to create SSH session: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Register SSH endpoint
	sshEndpoint := &objects.TEEEndpoint{
		RentalID:     rentalID,
		ContainerID:  rental.ContainerID,
		EndpointType: "ssh",
		Host:         "localhost", // TODO: Get actual host
		Port:         rental.SSHPort,
		Protocol:     "ssh",
		Status:       "active",
		CreatedAt:    time.Now(),
		ExpiresAt:    sshSession.ExpiresAt,
	}
	err = h.endpointRegistry.RegisterEndpoint(rentalID, "ssh", sshEndpoint)
	if err != nil {
		log.Printf("Warning: Failed to register SSH endpoint: %v", err)
		// Continue - endpoint registration failure shouldn't block session creation
	}

	// Get SSH endpoint info
	sshEndpointInfo, _ := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "ssh")
	endpoint := "localhost"
	port := rental.SSHPort
	if sshEndpointInfo != nil {
		endpoint = sshEndpointInfo.Host
		port = sshEndpointInfo.Port
	}

	sessionInfo := map[string]interface{}{
		"id":                       sshSession.ID,
		"rental_id":                rentalID,
		"container_id":             rental.ContainerID,
		"username":                 sshSession.Username,
		"private_key_download_url": sshSession.PrivateKeyURL,
		"endpoint":                 endpoint,
		"port":                     port,
		"command":                  fmt.Sprintf("ssh -i key.pem %s@%s -p %d", sshSession.Username, endpoint, port),
		"expires_at":               sshSession.ExpiresAt,
		"public_key_hash":          sshSession.PublicKeyHash,
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      sessionInfo,
		Message:   "SSH session created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	// Log successful access attempt
	h.logAccessAttempt(&AccessLogEntry{
		ID:           fmt.Sprintf("ssh_create_success_%d", time.Now().UnixNano()),
		UserID:       userID,
		RentalID:     rentalID,
		Action:       "create_session",
		ServiceType:  "ssh",
		IPAddress:    r.RemoteAddr,
		UserAgent:    r.UserAgent(),
		Success:      true,
		Timestamp:    startTime,
		ResponseTime: time.Since(startTime),
	})

	// Log performance metrics
	h.logPerformanceMetrics(&DVEPerformanceMetrics{
		ID:           fmt.Sprintf("perf_ssh_create_%d", time.Now().UnixNano()),
		Endpoint:     r.URL.Path,
		Method:       r.Method,
		ResponseTime: time.Since(startTime),
		StatusCode:   http.StatusCreated,
		UserID:       userID,
		Timestamp:    time.Now(),
	})
}

// GetSSHSession handles GET /api/dve-rental/rentals/{id}/ssh-session
func (h *DVERentalHandlers) GetSSHSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get all sessions for this rental and find the most recent SSH session
	sessions, err := h.sessionManager.GetSessionsByRentalID(rentalID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to retrieve sessions: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Find the most recent SSH session
	var latestSSHSession *objects.SSHSession
	var latestTime time.Time

	for _, session := range sessions {
		if sshSession, ok := session.(*objects.SSHSession); ok {
			if sshSession.CreatedAt.After(latestTime) {
				latestSSHSession = sshSession
				latestTime = sshSession.CreatedAt
			}
		}
	}

	if latestSSHSession == nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "SSH session not found for this rental",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get SSH endpoint info
	sshEndpointInfo, _ := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "ssh")
	endpoint := "localhost"
	port := rental.SSHPort
	if sshEndpointInfo != nil {
		endpoint = sshEndpointInfo.Host
		port = sshEndpointInfo.Port
	}

	sessionInfo := map[string]interface{}{
		"id":                       latestSSHSession.ID,
		"rental_id":                rentalID,
		"container_id":             rental.ContainerID,
		"username":                 latestSSHSession.Username,
		"private_key_download_url": latestSSHSession.PrivateKeyURL,
		"endpoint":                 endpoint,
		"port":                     port,
		"command":                  fmt.Sprintf("ssh -i key.pem %s@%s -p %d", latestSSHSession.Username, endpoint, port),
		"expires_at":               latestSSHSession.ExpiresAt,
		"public_key_hash":          latestSSHSession.PublicKeyHash,
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      sessionInfo,
		Message:   "SSH session retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateValidationSession handles POST /api/dve-rental/rentals/{id}/validation-session
func (h *DVERentalHandlers) CreateValidationSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if rental is active
	if rental.Status != "active" || time.Now().After(rental.EndTime) {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental is not active",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create validation session using session manager
	validationSession, err := h.sessionManager.CreateValidationSession(rentalID, "reasoning")
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to create validation session: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Register validation endpoint
	validationEndpoint := &objects.TEEEndpoint{
		RentalID:     rentalID,
		ContainerID:  rental.ContainerID,
		EndpointType: "validation",
		Host:         "localhost", // TODO: Get actual host
		Port:         23145,       // TODO: Get from container spec
		Protocol:     "http",
		Status:       "active",
		CreatedAt:    time.Now(),
		ExpiresAt:    validationSession.ExpiresAt,
	}
	err = h.endpointRegistry.RegisterEndpoint(rentalID, "validation", validationEndpoint)
	if err != nil {
		log.Printf("Warning: Failed to register validation endpoint: %v", err)
		// Continue - endpoint registration failure shouldn't block session creation
	}

	// Get validation endpoint info
	validationEndpointInfo, _ := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "validation")
	endpointURL := fmt.Sprintf("http://localhost:%d", 23145)
	if validationEndpointInfo != nil {
		endpointURL = fmt.Sprintf("%s://%s:%d", validationEndpointInfo.Protocol, validationEndpointInfo.Host, validationEndpointInfo.Port)
	}

	sessionInfo := map[string]interface{}{
		"id":              validationSession.ID,
		"rental_id":       rentalID,
		"container_id":    rental.ContainerID,
		"endpoint_url":    endpointURL,
		"session_token":   validationSession.SessionToken,
		"session_id":      validationSession.ID,
		"validation_type": validationSession.ValidationType,
		"expires_at":      validationSession.ExpiresAt,
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      sessionInfo,
		Message:   "Validation session created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetValidationSession handles GET /api/dve-rental/rentals/{id}/validation-session
func (h *DVERentalHandlers) GetValidationSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get all sessions for this rental and find the most recent validation session
	sessions, err := h.sessionManager.GetSessionsByRentalID(rentalID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to retrieve sessions: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Find the most recent validation session
	var latestValidationSession *objects.ValidationSession
	var latestTime time.Time

	for _, session := range sessions {
		if validationSession, ok := session.(*objects.ValidationSession); ok {
			if validationSession.CreatedAt.After(latestTime) {
				latestValidationSession = validationSession
				latestTime = validationSession.CreatedAt
			}
		}
	}

	if latestValidationSession == nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Validation session not found for this rental",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get validation endpoint info
	validationEndpointInfo, _ := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "validation")
	endpointURL := fmt.Sprintf("http://localhost:%d", 23145)
	if validationEndpointInfo != nil {
		endpointURL = fmt.Sprintf("%s://%s:%d", validationEndpointInfo.Protocol, validationEndpointInfo.Host, validationEndpointInfo.Port)
	}

	sessionInfo := map[string]interface{}{
		"id":              latestValidationSession.ID,
		"rental_id":       rentalID,
		"container_id":    rental.ContainerID,
		"endpoint_url":    endpointURL,
		"session_token":   latestValidationSession.SessionToken,
		"session_id":      latestValidationSession.ID,
		"validation_type": latestValidationSession.ValidationType,
		"expires_at":      latestValidationSession.ExpiresAt,
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      sessionInfo,
		Message:   "Validation session retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateErrorResolutionSession handles POST /api/dve-rental/rentals/{id}/error-resolution-session
func (h *DVERentalHandlers) CreateErrorResolutionSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if rental is active
	if rental.Status != "active" || time.Now().After(rental.EndTime) {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental is not active",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create error resolution session using session manager
	supportedTypes := []string{
		"connection_timeout",
		"validation_failed",
		"resource_exhausted",
		"custom_error",
	}
	errorResSession, err := h.sessionManager.CreateErrorResolutionSession(rentalID, supportedTypes)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to create error resolution session: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Register error resolution endpoint
	errorResEndpoint := &objects.TEEEndpoint{
		RentalID:     rentalID,
		ContainerID:  rental.ContainerID,
		EndpointType: "error-resolution",
		Host:         "localhost", // TODO: Get actual host
		Port:         24145,       // TODO: Get from container spec
		Protocol:     "http",
		Status:       "active",
		CreatedAt:    time.Now(),
		ExpiresAt:    errorResSession.ExpiresAt,
	}
	err = h.endpointRegistry.RegisterEndpoint(rentalID, "error-resolution", errorResEndpoint)
	if err != nil {
		log.Printf("Warning: Failed to register error resolution endpoint: %v", err)
		// Continue - endpoint registration failure shouldn't block session creation
	}

	// Get error resolution endpoint info
	errorResEndpointInfo, _ := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "error-resolution")
	endpointURL := fmt.Sprintf("http://localhost:%d", 24145)
	if errorResEndpointInfo != nil {
		endpointURL = fmt.Sprintf("%s://%s:%d", errorResEndpointInfo.Protocol, errorResEndpointInfo.Host, errorResEndpointInfo.Port)
	}

	sessionInfo := map[string]interface{}{
		"id":                    errorResSession.ID,
		"rental_id":             rentalID,
		"container_id":          rental.ContainerID,
		"endpoint_url":          endpointURL,
		"session_token":         errorResSession.SessionToken,
		"session_id":            errorResSession.ID,
		"supported_error_types": errorResSession.SupportedTypes,
		"expires_at":            errorResSession.ExpiresAt,
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      sessionInfo,
		Message:   "Error resolution session created successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetErrorResolutionSession handles GET /api/dve-rental/rentals/{id}/error-resolution-session
func (h *DVERentalHandlers) GetErrorResolutionSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get all sessions for this rental and find the most recent error resolution session
	sessions, err := h.sessionManager.GetSessionsByRentalID(rentalID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to retrieve sessions: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Find the most recent error resolution session
	var latestErrorResSession *objects.ErrorResolutionSession
	var latestTime time.Time

	for _, session := range sessions {
		if errorResSession, ok := session.(*objects.ErrorResolutionSession); ok {
			if errorResSession.CreatedAt.After(latestTime) {
				latestErrorResSession = errorResSession
				latestTime = errorResSession.CreatedAt
			}
		}
	}

	if latestErrorResSession == nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Error resolution session not found for this rental",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get error resolution endpoint info
	errorResEndpointInfo, _ := h.endpointRegistry.GetEndpointByRentalAndType(rentalID, "error-resolution")
	endpointURL := fmt.Sprintf("http://localhost:%d", 24145)
	if errorResEndpointInfo != nil {
		endpointURL = fmt.Sprintf("%s://%s:%d", errorResEndpointInfo.Protocol, errorResEndpointInfo.Host, errorResEndpointInfo.Port)
	}

	sessionInfo := map[string]interface{}{
		"id":                    latestErrorResSession.ID,
		"rental_id":             rentalID,
		"container_id":          rental.ContainerID,
		"endpoint_url":          endpointURL,
		"session_token":         latestErrorResSession.SessionToken,
		"session_id":            latestErrorResSession.ID,
		"supported_error_types": latestErrorResSession.SupportedTypes,
		"expires_at":            latestErrorResSession.ExpiresAt,
	}

	response := DVERentalResponse{
		Success:   true,
		Data:      sessionInfo,
		Message:   "Error resolution session retrieved successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TerminateValidationSession handles DELETE /api/dve-rental/rentals/{id}/validation-session
func (h *DVERentalHandlers) TerminateValidationSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get all sessions for this rental and find the most recent validation session
	sessions, err := h.sessionManager.GetSessionsByRentalID(rentalID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to retrieve sessions: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Find the most recent validation session
	var latestValidationSession *objects.ValidationSession
	var latestTime time.Time

	for _, session := range sessions {
		if validationSession, ok := session.(*objects.ValidationSession); ok {
			if validationSession.CreatedAt.After(latestTime) {
				latestValidationSession = validationSession
				latestTime = validationSession.CreatedAt
			}
		}
	}

	if latestValidationSession == nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Validation session not found for this rental",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Terminate the validation session
	err = h.sessionManager.TerminateValidationSession(latestValidationSession.ID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to terminate validation session: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Message:   "Validation session terminated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TerminateErrorResolutionSession handles DELETE /api/dve-rental/rentals/{id}/error-resolution-session
func (h *DVERentalHandlers) TerminateErrorResolutionSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get all sessions for this rental and find the most recent error resolution session
	sessions, err := h.sessionManager.GetSessionsByRentalID(rentalID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to retrieve sessions: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Find the most recent error resolution session
	var latestErrorResSession *objects.ErrorResolutionSession
	var latestTime time.Time

	for _, session := range sessions {
		if errorResSession, ok := session.(*objects.ErrorResolutionSession); ok {
			if errorResSession.CreatedAt.After(latestTime) {
				latestErrorResSession = errorResSession
				latestTime = errorResSession.CreatedAt
			}
		}
	}

	if latestErrorResSession == nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Error resolution session not found for this rental",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Terminate the error resolution session
	err = h.sessionManager.TerminateErrorResolutionSession(latestErrorResSession.ID)
	if err != nil {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Failed to terminate error resolution session: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DVERentalResponse{
		Success:   true,
		Message:   "Error resolution session terminated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TerminateSSHSession handles DELETE /api/dve-rental/rentals/{id}/ssh-session
func (h *DVERentalHandlers) TerminateSSHSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rentalID := vars["id"]

	if rentalID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "test-user-default"
	}

	// Validate rental ownership
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil || rental == nil || rental.UserID != userID {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Rental not found or access denied",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Terminate SSH session using session manager
	response := DVERentalResponse{
		Success:   true,
		Message:   "SSH session terminated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DownloadSSHPrivateKey handles GET /api/sessions/ssh/{sessionId}/private-key
func (h *DVERentalHandlers) DownloadSSHPrivateKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	if sessionID == "" {
		response := DVERentalResponse{
			Success:   false,
			Error:     "Session ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get SSH session
	sshSession, err := h.sessionManager.GetSSHSession(sessionID)
	if err != nil {
		errorMsg := "SSH session not found: " + err.Error()
		statusCode := http.StatusNotFound
		if strings.Contains(err.Error(), "expired") {
			statusCode = http.StatusGone
			errorMsg = "SSH session has expired"
		}
		response := DVERentalResponse{
			Success:   false,
			Error:     errorMsg,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if session is expired
	if time.Now().After(sshSession.ExpiresAt) {
		response := DVERentalResponse{
			Success:   false,
			Error:     "SSH session has expired",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate rental access (optional, but good practice)
	// For now, we'll allow download if session exists and is valid

	// Set headers for file download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", "ssh_private_key.pem"))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(sshSession.PrivateKey)))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Write the private key
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(sshSession.PrivateKey))
}

// RegisterRoutes registers the DVE rental routes with the router
func (h *DVERentalHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Register SSH private key download route
	if authMiddleware != nil {
		protectedRouter := r.PathPrefix("/api/sessions").Subrouter()
		protectedRouter.Use(authMiddleware.RequireAuth)
		protectedRouter.HandleFunc("/ssh/{sessionId}/private-key", h.DownloadSSHPrivateKey).Methods("GET", "OPTIONS")
	} else {
		r.HandleFunc("/api/sessions/ssh/{sessionId}/private-key", h.DownloadSSHPrivateKey).Methods("GET", "OPTIONS")
	}

	// Create a subrouter for DVE endpoints
	rentalRouter := r.PathPrefix("/api/dve").Subrouter()

	// Public routes for viewing available services and stats
	rentalRouter.HandleFunc("/services", h.GetRentalPlans).Methods("GET", "OPTIONS")
	rentalRouter.HandleFunc("/stats", h.GetRentalStats).Methods("GET", "OPTIONS")

	// Protected routes for DVE instance management
	if authMiddleware != nil {
		protectedRentalRouter := rentalRouter.PathPrefix("").Subrouter()
		protectedRentalRouter.Use(authMiddleware.RequireAuth)
		protectedRentalRouter.HandleFunc("/instances", h.CreateRental).Methods("POST", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances", h.GetUserRentals).Methods("GET", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/extend", h.ExtendRental).Methods("POST", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}", h.CancelRental).Methods("DELETE", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/full-access-info", h.GetFullAccessInfo).Methods("GET", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/ssh-session", h.CreateSSHSession).Methods("POST", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/ssh-session", h.GetSSHSession).Methods("GET", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/ssh-session", h.TerminateSSHSession).Methods("DELETE", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/validation-session", h.CreateValidationSession).Methods("POST", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/validation-session", h.GetValidationSession).Methods("GET", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/validation-session", h.TerminateValidationSession).Methods("DELETE", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/error-resolution-session", h.CreateErrorResolutionSession).Methods("POST", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/error-resolution-session", h.GetErrorResolutionSession).Methods("GET", "OPTIONS")
		protectedRentalRouter.HandleFunc("/instances/{id}/error-resolution-session", h.TerminateErrorResolutionSession).Methods("DELETE", "OPTIONS")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		rentalRouter.HandleFunc("/instances", h.CreateRental).Methods("POST", "OPTIONS")
		rentalRouter.HandleFunc("/instances", h.GetUserRentals).Methods("GET", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/extend", h.ExtendRental).Methods("POST", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}", h.CancelRental).Methods("DELETE", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/full-access-info", h.GetFullAccessInfo).Methods("GET", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/ssh-session", h.CreateSSHSession).Methods("POST", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/ssh-session", h.GetSSHSession).Methods("GET", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/ssh-session", h.TerminateSSHSession).Methods("DELETE", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/validation-session", h.CreateValidationSession).Methods("POST", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/validation-session", h.GetValidationSession).Methods("GET", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/validation-session", h.TerminateValidationSession).Methods("DELETE", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/error-resolution-session", h.CreateErrorResolutionSession).Methods("POST", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/error-resolution-session", h.GetErrorResolutionSession).Methods("GET", "OPTIONS")
		rentalRouter.HandleFunc("/instances/{id}/error-resolution-session", h.TerminateErrorResolutionSession).Methods("DELETE", "OPTIONS")
	}
	// CORS headers are handled by the CORSMiddleware
	// No need for explicit OPTIONS handler as middleware handles it
}

// validateRentalAccess performs comprehensive access validation for DVE rentals
func (h *DVERentalHandlers) validateRentalAccess(rentalID, userID string) error {
	if rentalID == "" {
		return fmt.Errorf("rental ID is required")
	}

	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	// Get rental details
	rental, err := h.dveRentalService.GetRentalByID(rentalID)
	if err != nil {
		return fmt.Errorf("failed to fetch rental: %w", err)
	}

	if rental == nil {
		return fmt.Errorf("rental not found")
	}

	// Check user ownership
	if rental.UserID != userID {
		return fmt.Errorf("access denied: rental belongs to different user")
	}

	// Check rental status
	if rental.Status != "active" {
		return fmt.Errorf("rental is not active (status: %s)", rental.Status)
	}

	// Check if rental has expired
	if time.Now().After(rental.EndTime) {
		return fmt.Errorf("rental has expired")
	}

	// Check if rental has not started yet
	if time.Now().Before(rental.StartTime) {
		return fmt.Errorf("rental has not started yet")
	}

	// Additional security checks
	if rental.ContainerID == "" {
		return fmt.Errorf("rental has no associated container")
	}

	// Validate resource limits are reasonable
	if rental.ResourceLimits.MaxCPU > 8.0 || rental.ResourceLimits.MaxMemory > 32*1024*1024*1024 {
		log.Printf("Warning: Rental %s has high resource limits - CPU: %.1f, Memory: %dGB",
			rentalID, rental.ResourceLimits.MaxCPU, rental.ResourceLimits.MaxMemory/(1024*1024*1024))
	}

	return nil
}

// AccessLogEntry represents an access attempt log entry
type AccessLogEntry struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	RentalID     string        `json:"rental_id"`
	Action       string        `json:"action"`       // "create_session", "access_terminal", "terminate_session", etc.
	ServiceType  string        `json:"service_type"` // "ssh", "validation", "error_resolution"
	IPAddress    string        `json:"ip_address"`
	UserAgent    string        `json:"user_agent"`
	Success      bool          `json:"success"`
	ErrorMsg     string        `json:"error_msg,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	ResponseTime time.Duration `json:"response_time"`
}

// logAccessAttempt logs an access attempt to the database
func (h *DVERentalHandlers) logAccessAttempt(entry *AccessLogEntry) {
	if h.db == nil {
		log.Printf("Warning: Database not available for access logging")
		return
	}

	err := h.db.Transaction(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(fmt.Sprintf("access_log:%s", entry.ID), string(data), nil)
		return err
	})

	if err != nil {
		log.Printf("Failed to log access attempt: %v", err)
	}
}

// DVEPerformanceMetrics represents performance monitoring data for DVE operations
type DVEPerformanceMetrics struct {
	ID           string        `json:"id"`
	Endpoint     string        `json:"endpoint"`
	Method       string        `json:"method"`
	ResponseTime time.Duration `json:"response_time"`
	StatusCode   int           `json:"status_code"`
	RequestSize  int64         `json:"request_size"`
	ResponseSize int64         `json:"response_size"`
	UserID       string        `json:"user_id,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	MemoryUsage  int64         `json:"memory_usage,omitempty"`
	CPUUsage     float64       `json:"cpu_usage,omitempty"`
}

// logPerformanceMetrics logs performance metrics
func (h *DVERentalHandlers) logPerformanceMetrics(metrics *DVEPerformanceMetrics) {
	if h.db == nil {
		return
	}

	err := h.db.Transaction(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(metrics)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(fmt.Sprintf("perf_metrics:%s", metrics.ID), string(data), nil)
		return err
	})

	if err != nil {
		log.Printf("Failed to log performance metrics: %v", err)
	}
}

// ErrorLogEntry represents an error log entry
type ErrorLogEntry struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id,omitempty"`
	RentalID    string                 `json:"rental_id,omitempty"`
	Endpoint    string                 `json:"endpoint"`
	Error       string                 `json:"error"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Severity    string                 `json:"severity"` // "low", "medium", "high", "critical"
	Timestamp   time.Time              `json:"timestamp"`
	RequestData map[string]interface{} `json:"request_data,omitempty"`
}

// logError logs an error event
func (h *DVERentalHandlers) logError(entry *ErrorLogEntry) {
	if h.db == nil {
		log.Printf("Error: %s", entry.Error)
		return
	}

	err := h.db.Transaction(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(fmt.Sprintf("error_log:%s", entry.ID), string(data), nil)
		return err
	})

	if err != nil {
		log.Printf("Failed to log error: %v", err)
	}

	// Also log to standard logger with appropriate level
	log.Printf("[%s] %s", entry.Severity, entry.Error)
}
