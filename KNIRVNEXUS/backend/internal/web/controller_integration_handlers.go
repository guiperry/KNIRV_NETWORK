package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/controllerintegration"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
)

type ControllerIntegrationHandlers struct {
	controllerIntegrationService *controllerintegration.ControllerIntegrationService
}

func NewControllerIntegrationHandlers(controllerIntegrationService *controllerintegration.ControllerIntegrationService) *ControllerIntegrationHandlers {
	return &ControllerIntegrationHandlers{controllerIntegrationService: controllerIntegrationService}
}

type ControllerIntegrationResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// PostGenerateQRCode handles POST /api/controller-integration/qr-code
func (h *ControllerIntegrationHandlers) PostGenerateQRCode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID       string   `json:"user_id"`
		DeviceType   string   `json:"device_type"`
		Capabilities []string `json:"capabilities"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if request.UserID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "User ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	qrCode, err := h.controllerIntegrationService.GenerateQRCode(request.UserID, request.DeviceType, request.Capabilities)
	if err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to generate QR code: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ControllerIntegrationResponse{
		Success:   true,
		Data:      qrCode,
		Message:   "QR code generated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostScanQRCode handles POST /api/controller-integration/qr-code/scan
func (h *ControllerIntegrationHandlers) PostScanQRCode(w http.ResponseWriter, r *http.Request) {
	var request struct {
		QRData         string `json:"qr_data"`
		MobileDeviceID string `json:"mobile_device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if request.QRData == "" || request.MobileDeviceID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "QR data and mobile device ID are required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	pairingRequest, err := h.controllerIntegrationService.ScanQRCode(request.QRData, request.MobileDeviceID)
	if err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to scan QR code: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ControllerIntegrationResponse{
		Success:   true,
		Data:      pairingRequest,
		Message:   "QR code scanned successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostConfirmPairing handles POST /api/controller-integration/pairing/{id}/confirm
func (h *ControllerIntegrationHandlers) PostConfirmPairing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pairingRequestID := vars["id"]

	if pairingRequestID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Pairing request ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var request struct {
		Confirmed bool `json:"confirmed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	session, err := h.controllerIntegrationService.ConfirmPairing(pairingRequestID, request.Confirmed)
	if err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to confirm pairing: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	var responseData interface{}
	var message string

	if session != nil {
		responseData = session
		message = "Pairing confirmed and session created"
	} else {
		message = "Pairing rejected"
	}

	response := ControllerIntegrationResponse{
		Success:   true,
		Data:      responseData,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSession handles GET /api/controller-integration/sessions/{id}
func (h *ControllerIntegrationHandlers) GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if sessionID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Session ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	session, err := h.controllerIntegrationService.GetActiveSession(sessionID)
	if err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to get session: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ControllerIntegrationResponse{
		Success:   true,
		Data:      session,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserSessions handles GET /api/controller-integration/users/{id}/sessions
func (h *ControllerIntegrationHandlers) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "User ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	sessions, err := h.controllerIntegrationService.GetUserSessions(userID)
	if err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to get user sessions: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ControllerIntegrationResponse{
		Success:   true,
		Data:      sessions,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostSendMessage handles POST /api/controller-integration/sessions/{id}/messages
func (h *ControllerIntegrationHandlers) PostSendMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if sessionID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Session ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var message objects.ControllerMessage
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set message metadata
	message.ID = "msg_" + time.Now().Format("20060102150405")
	message.SessionID = sessionID
	message.Timestamp = time.Now()
	message.Processed = false

	if err := h.controllerIntegrationService.SendMessage(sessionID, &message); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to send message: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ControllerIntegrationResponse{
		Success:   true,
		Data:      &message,
		Message:   "Message sent successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteSession handles DELETE /api/controller-integration/sessions/{id}
func (h *ControllerIntegrationHandlers) DeleteSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if sessionID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Session ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var request struct {
		Reason string `json:"reason"`
	}

	// Try to decode reason, but it's optional
	json.NewDecoder(r.Body).Decode(&request)

	reason := request.Reason
	if reason == "" {
		reason = "user_requested"
	}

	if err := h.controllerIntegrationService.TerminateSession(sessionID, reason); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to terminate session: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ControllerIntegrationResponse{
		Success:   true,
		Message:   "Session terminated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostHandleCommand handles POST /api/controller-integration/sessions/{id}/commands
func (h *ControllerIntegrationHandlers) PostHandleCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if sessionID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Session ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var command objects.ControllerMessage
	if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set command metadata
	command.ID = "cmd_" + time.Now().Format("20060102150405")
	command.SessionID = sessionID
	command.Direction = "inbound"
	command.Timestamp = time.Now()
	command.Processed = false

	response, err := h.controllerIntegrationService.HandleControllerCommand(sessionID, &command)
	if err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to handle command: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	apiResponse := ControllerIntegrationResponse{
		Success:   true,
		Data:      response,
		Message:   "Command handled successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiResponse)
}

// PostNegotiateCapabilities handles POST /api/controller-integration/sessions/{id}/capabilities
func (h *ControllerIntegrationHandlers) PostNegotiateCapabilities(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if sessionID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Session ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var request struct {
		RequestedCapabilities []string `json:"requested_capabilities"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	negotiatedCapabilities, err := h.controllerIntegrationService.NegotiateCapabilities(sessionID, request.RequestedCapabilities)
	if err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to negotiate capabilities: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ControllerIntegrationResponse{
		Success: true,
		Data: map[string]interface{}{
			"negotiated_capabilities": negotiatedCapabilities,
		},
		Message:   "Capabilities negotiated successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PostSendPushNotification handles POST /api/controller-integration/sessions/{id}/notifications
func (h *ControllerIntegrationHandlers) PostSendPushNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if sessionID == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Session ID is required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var request struct {
		Title   string                 `json:"title"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Invalid request body",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if request.Title == "" || request.Message == "" {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Title and message are required",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.controllerIntegrationService.SendPushNotification(sessionID, request.Title, request.Message, request.Data)
	if err != nil {
		response := ControllerIntegrationResponse{
			Success:   false,
			Error:     "Failed to send push notification: " + err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ControllerIntegrationResponse{
		Success:   true,
		Message:   "Push notification sent successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterRoutes registers the controller integration routes with the router
func (h *ControllerIntegrationHandlers) RegisterRoutes(r *mux.Router, authMiddleware *middleware.AuthMiddleware) {
	// Create a subrouter for controller integration endpoints
	controllerRouter := r.PathPrefix("/api/controller-integration").Subrouter()

	// Public routes for QR code scanning (mobile app)
	controllerRouter.HandleFunc("/qr-code/scan", h.PostScanQRCode).Methods("POST")

	// Protected routes for controller management
	if authMiddleware != nil {
		protectedControllerRouter := controllerRouter.PathPrefix("").Subrouter()
		protectedControllerRouter.Use(authMiddleware.RequireAuth)

		// QR code generation (desktop)
		protectedControllerRouter.HandleFunc("/qr-code", h.PostGenerateQRCode).Methods("POST")

		// Pairing management
		protectedControllerRouter.HandleFunc("/pairing/{id}/confirm", h.PostConfirmPairing).Methods("POST")

		// Session management
		protectedControllerRouter.HandleFunc("/sessions/{id}", h.GetSession).Methods("GET")
		protectedControllerRouter.HandleFunc("/sessions/{id}", h.DeleteSession).Methods("DELETE")
		protectedControllerRouter.HandleFunc("/sessions/{id}/messages", h.PostSendMessage).Methods("POST")
		protectedControllerRouter.HandleFunc("/sessions/{id}/commands", h.PostHandleCommand).Methods("POST")
		protectedControllerRouter.HandleFunc("/sessions/{id}/capabilities", h.PostNegotiateCapabilities).Methods("POST")
		protectedControllerRouter.HandleFunc("/sessions/{id}/notifications", h.PostSendPushNotification).Methods("POST")

		// User sessions
		protectedControllerRouter.HandleFunc("/users/{id}/sessions", h.GetUserSessions).Methods("GET")
	} else {
		// If no auth middleware, allow all routes (for testnet mode)
		controllerRouter.HandleFunc("/qr-code", h.PostGenerateQRCode).Methods("POST")
		controllerRouter.HandleFunc("/pairing/{id}/confirm", h.PostConfirmPairing).Methods("POST")
		controllerRouter.HandleFunc("/sessions/{id}", h.GetSession).Methods("GET")
		controllerRouter.HandleFunc("/sessions/{id}", h.DeleteSession).Methods("DELETE")
		controllerRouter.HandleFunc("/sessions/{id}/messages", h.PostSendMessage).Methods("POST")
		controllerRouter.HandleFunc("/sessions/{id}/commands", h.PostHandleCommand).Methods("POST")
		controllerRouter.HandleFunc("/sessions/{id}/capabilities", h.PostNegotiateCapabilities).Methods("POST")
		controllerRouter.HandleFunc("/sessions/{id}/notifications", h.PostSendPushNotification).Methods("POST")
		controllerRouter.HandleFunc("/users/{id}/sessions", h.GetUserSessions).Methods("GET")
	}

	// Handle OPTIONS requests for CORS
	controllerRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
