package desktop

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// handleTargetAssignmentQR handles target assignment QR code generation
func (dh *DesktopHost) handleTargetAssignmentQR(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TargetSystemID string   `json:"target_system_id"`
		Capabilities   []string `json:"capabilities"`
		ExpiryMinutes  int      `json:"expiry_minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Generate QR code for target assignment
	qrCode, err := dh.qrLinkage.GenerateTargetAssignmentQR(
		request.TargetSystemID,
		request.Capabilities,
	)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"qr_code_data": string(qrCode.Data),
		"session_id":   qrCode.SessionID,
		"expires_at":   qrCode.ExpiresAt,
		"target_info": map[string]interface{}{
			"id":   request.TargetSystemID,
			"name": fmt.Sprintf("Target-%s", request.TargetSystemID),
			"type": "system",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTransactionSignQR handles transaction signing QR code generation
func (dh *DesktopHost) handleTransactionSignQR(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TransactionHash string `json:"transaction_hash"`
		Amount          string `json:"amount"`
		Recipient       string `json:"recipient"`
		GasFee          string `json:"gas_fee"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	transactionData := &TransactionData{
		Hash:      request.TransactionHash,
		Amount:    request.Amount,
		Recipient: request.Recipient,
		GasFee:    request.GasFee,
		Timestamp: time.Now().Unix(),
	}

	qrCode, err := dh.qrLinkage.GenerateTransactionSignQR(transactionData)
	if err != nil {
		http.Error(w, "Failed to generate transaction QR code", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"qr_code_data": string(qrCode.Data),
		"session_id":   qrCode.SessionID,
		"expires_at":   qrCode.ExpiresAt,
		"transaction": map[string]interface{}{
			"hash":      request.TransactionHash,
			"amount":    request.Amount,
			"recipient": request.Recipient,
			"gas_fee":   request.GasFee,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleQRSessionStatus handles QR session status requests
func (dh *DesktopHost) handleQRSessionStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	session, exists := dh.qrLinkage.GetSession(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"session_id":       session.SessionID,
		"status":           session.Status,
		"linkage_type":     session.LinkageType,
		"expires_at":       session.ExpiresAt,
		"mobile_id":        session.MobileID,
		"target_system_id": session.TargetSystemID,
		"created_at":       session.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMobileConnect handles mobile device connection requests
func (dh *DesktopHost) handleMobileConnect(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SessionID     string   `json:"session_id"`
		DeviceID      string   `json:"device_id"`
		WalletAddress string   `json:"wallet_address"`
		PublicKey     string   `json:"public_key"`
		Capabilities  []string `json:"capabilities"`
		Signature     string   `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Handle mobile linkage
	mobileData := &MobileLinkageData{
		DeviceID:      request.DeviceID,
		WalletAddress: request.WalletAddress,
		PublicKey:     request.PublicKey,
		Capabilities:  request.Capabilities,
		Signature:     request.Signature,
	}

	err := dh.HandleMobileLinkage(request.SessionID, mobileData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to establish mobile connection: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":          "connected",
		"desktop_id":      dh.desktopID,
		"secure_endpoint": dh.endpoint,
		"session_key":     "mock_session_key",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHRMProcess handles HRM cognitive processing requests
func (dh *DesktopHost) handleHRMProcess(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SensoryData []float32 `json:"sensory_data"`
		Context     string    `json:"context"`
		TaskType    string    `json:"task_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	input := &HRMInput{
		SensoryData: request.SensoryData,
		Context:     request.Context,
		TaskType:    request.TaskType,
	}

	output, err := dh.hrmEngine.ProcessCognitiveInput(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("HRM processing failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

// handleHRMInfo handles HRM model information requests
func (dh *DesktopHost) handleHRMInfo(w http.ResponseWriter, r *http.Request) {
	info := dh.hrmEngine.GetModelInfo()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleCreateAgentSession handles agent session creation requests
func (dh *DesktopHost) handleCreateAgentSession(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID         string `json:"user_id"`
		MobileDeviceID string `json:"mobile_device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	session, err := dh.CreateAgentSession(request.UserID, request.MobileDeviceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create agent session: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// handleAgentWebSocket handles WebSocket connections for agent communication
func (dh *DesktopHost) handleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := dh.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error": "session_id required"}`))
		return
	}

	log.Printf("WebSocket connection established for session: %s", sessionID)

	// Handle WebSocket messages
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// Echo message back for now
		response := map[string]interface{}{
			"type":       "response",
			"session_id": sessionID,
			"echo":       message,
			"timestamp":  time.Now().Unix(),
		}

		err = conn.WriteJSON(response)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}

// handleHealth handles health check requests
func (dh *DesktopHost) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":             "healthy",
		"desktop_id":         dh.desktopID,
		"hrm_initialized":    dh.hrmEngine.IsInitialized(),
		"mobile_connections": len(dh.mobileConnections),
		"agent_sessions":     len(dh.agentSessions),
		"timestamp":          time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
