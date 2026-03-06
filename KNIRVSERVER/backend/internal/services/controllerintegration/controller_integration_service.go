package controllerintegration

import (
	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/websocket"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tidwall/buntdb"
)

// ControllerIntegrationService manages advanced controller integration
type ControllerIntegrationService struct {
	db      *database.BuntDBManager
	mu      sync.RWMutex
	running bool

	// Session management
	activeSessions  map[string]*objects.ControllerSession
	qrCodes         map[string]*objects.QRCode
	pairingRequests map[string]*objects.PairingRequest

	// Real-time communication
	websocketService interface {
		BroadcastToRoom(roomName string, message *websocket.RoomMessage)
	}
	websocketConnections map[string]*objects.WebSocketConnection
	messageQueue         map[string][]*objects.ControllerMessage

	// Configuration
	sessionTimeout     time.Duration
	qrCodeTimeout      time.Duration
	maxSessionsPerUser int
	maxQRCodesPerUser  int

	// Security
	encryptionKey []byte
	signingKey    []byte
}

// NewControllerIntegrationService creates a new controller integration service
func NewControllerIntegrationService(db *database.BuntDBManager) *ControllerIntegrationService {
	service := &ControllerIntegrationService{
		db:                   db,
		activeSessions:       make(map[string]*objects.ControllerSession),
		qrCodes:              make(map[string]*objects.QRCode),
		pairingRequests:      make(map[string]*objects.PairingRequest),
		websocketConnections: make(map[string]*objects.WebSocketConnection),
		messageQueue:         make(map[string][]*objects.ControllerMessage),
		sessionTimeout:       30 * time.Minute,
		qrCodeTimeout:        5 * time.Minute,
		maxSessionsPerUser:   10,
		maxQRCodesPerUser:    5,
	}

	// Generate encryption and signing keys
	service.generateSecurityKeys()

	// Initialize database indices
	service.initializeDatabase()

	// Load existing data
	service.loadControllerData()

	return service
}

// Start begins the controller integration service
func (cis *ControllerIntegrationService) Start() error {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	if cis.running {
		return fmt.Errorf("controller integration service already running")
	}

	cis.running = true

	log.Println("Starting controller integration service...")

	// Start background goroutines
	go cis.sessionCleanupLoop()
	go cis.qrCodeCleanupLoop()
	go cis.messageProcessingLoop()

	log.Println("Controller integration service started successfully")
	return nil
}

// Stop stops the controller integration service
func (cis *ControllerIntegrationService) Stop() error {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	if !cis.running {
		return fmt.Errorf("controller integration service not running")
	}

	cis.running = false

	log.Println("Controller integration service stopped")
	return nil
}

// IsRunning returns whether the service is running
func (cis *ControllerIntegrationService) IsRunning() bool {
	cis.mu.RLock()
	defer cis.mu.RUnlock()
	return cis.running
}

// SetWebSocketService sets the WebSocket service for real-time communication
func (cis *ControllerIntegrationService) SetWebSocketService(ws interface {
	BroadcastToRoom(roomName string, message *websocket.RoomMessage)
}) {
	cis.mu.Lock()
	defer cis.mu.Unlock()
	cis.websocketService = ws
	log.Println("WebSocket service set for controller integration")
}

// GenerateQRCode creates a new QR code for controller pairing
func (cis *ControllerIntegrationService) GenerateQRCode(userID, deviceType string, capabilities []string) (*objects.QRCode, error) {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	// Check user QR code limit
	userQRCount := cis.countUserQRCodes(userID)
	if userQRCount >= cis.maxQRCodesPerUser {
		return nil, fmt.Errorf("maximum QR codes (%d) reached for user %s", cis.maxQRCodesPerUser, userID)
	}

	// Generate secure session ID
	sessionID, err := cis.generateSecureID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Generate desktop ID
	desktopID, err := cis.generateSecureID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate desktop ID: %w", err)
	}

	// Create QR code data
	qrCode := &objects.QRCode{
		ID:           fmt.Sprintf("qr_%d", time.Now().UnixNano()),
		SessionID:    sessionID,
		DesktopID:    desktopID,
		UserID:       userID,
		DeviceType:   deviceType,
		Capabilities: capabilities,
		Status:       "active",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(cis.qrCodeTimeout),
		ScanCount:    0,
		MaxScans:     1,
		Data: &objects.QRCodeData{
			Version:      "2.0",
			Type:         "controller_pairing",
			SessionID:    sessionID,
			DesktopID:    desktopID,
			UserID:       userID,
			DeviceType:   deviceType,
			Capabilities: capabilities,
			ExpiresAt:    time.Now().Add(cis.qrCodeTimeout).Unix(),
			Timestamp:    time.Now().Unix(),
			Signature:    "",
		},
	}

	// Sign the QR code data
	signature, err := cis.signQRCodeData(qrCode.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to sign QR code: %w", err)
	}
	qrCode.Data.Signature = signature

	// Store QR code
	cis.qrCodes[qrCode.ID] = qrCode
	cis.storeQRCode(qrCode)

	log.Printf("Generated QR code: %s for user %s", qrCode.ID, userID)
	return qrCode, nil
}

// ScanQRCode processes a QR code scan and initiates pairing
func (cis *ControllerIntegrationService) ScanQRCode(qrData string, mobileDeviceID string) (*objects.PairingRequest, error) {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	// Parse QR code data
	var qrCodeData objects.QRCodeData
	if err := json.Unmarshal([]byte(qrData), &qrCodeData); err != nil {
		return nil, fmt.Errorf("invalid QR code format: %w", err)
	}

	// Verify QR code signature
	if !cis.verifyQRCodeSignature(&qrCodeData) {
		return nil, fmt.Errorf("invalid QR code signature")
	}

	// Check if QR code has expired
	if time.Now().Unix() > qrCodeData.ExpiresAt {
		return nil, fmt.Errorf("QR code has expired")
	}

	// Find the QR code in our store
	var qrCode *objects.QRCode
	for _, qr := range cis.qrCodes {
		if qr.SessionID == qrCodeData.SessionID && qr.DesktopID == qrCodeData.DesktopID {
			qrCode = qr
			break
		}
	}

	if qrCode == nil {
		return nil, fmt.Errorf("QR code not found or invalid")
	}

	// Check scan limits
	if qrCode.ScanCount >= qrCode.MaxScans {
		return nil, fmt.Errorf("QR code scan limit exceeded")
	}

	// Update scan count
	qrCode.ScanCount++
	qrCode.LastScannedAt = &time.Time{}
	*qrCode.LastScannedAt = time.Now()

	// Create pairing request
	pairingRequest := &objects.PairingRequest{
		ID:             fmt.Sprintf("pair_%d", time.Now().UnixNano()),
		QRCodeID:       qrCode.ID,
		SessionID:      qrCode.SessionID,
		DesktopID:      qrCode.DesktopID,
		UserID:         qrCode.UserID,
		MobileDeviceID: mobileDeviceID,
		Status:         "pending",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(2 * time.Minute),
		Capabilities:   qrCode.Capabilities,
		DeviceInfo: &objects.DeviceInfo{
			DeviceID:   mobileDeviceID,
			DeviceType: "mobile",
			Platform:   "unknown",
			Version:    "unknown",
		},
	}

	// Store pairing request
	cis.pairingRequests[pairingRequest.ID] = pairingRequest
	cis.storePairingRequest(pairingRequest)

	log.Printf("Created pairing request: %s for QR code %s", pairingRequest.ID, qrCode.ID)
	return pairingRequest, nil
}

// ConfirmPairing confirms a pairing request and creates a session
func (cis *ControllerIntegrationService) ConfirmPairing(pairingRequestID string, confirmed bool) (*objects.ControllerSession, error) {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	pairingRequest, exists := cis.pairingRequests[pairingRequestID]
	if !exists {
		return nil, fmt.Errorf("pairing request not found: %s", pairingRequestID)
	}

	// Check if pairing request has expired
	if time.Now().After(pairingRequest.ExpiresAt) {
		return nil, fmt.Errorf("pairing request has expired")
	}

	if !confirmed {
		// Mark as rejected
		pairingRequest.Status = "rejected"
		pairingRequest.RejectedAt = &time.Time{}
		*pairingRequest.RejectedAt = time.Now()
		cis.storePairingRequest(pairingRequest)
		return nil, fmt.Errorf("pairing request rejected")
	}

	// Check user session limit
	userSessionCount := cis.countUserSessions(pairingRequest.UserID)
	if userSessionCount >= cis.maxSessionsPerUser {
		return nil, fmt.Errorf("maximum sessions (%d) reached for user %s", cis.maxSessionsPerUser, pairingRequest.UserID)
	}

	// Create controller session
	session := &objects.ControllerSession{
		ID:             fmt.Sprintf("sess_%d", time.Now().UnixNano()),
		SessionID:      pairingRequest.SessionID,
		DesktopID:      pairingRequest.DesktopID,
		UserID:         pairingRequest.UserID,
		MobileDeviceID: pairingRequest.MobileDeviceID,
		Status:         "active",
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		ExpiresAt:      time.Now().Add(cis.sessionTimeout),
		Capabilities:   pairingRequest.Capabilities,
		DeviceInfo:     pairingRequest.DeviceInfo,
		ConnectionInfo: &objects.ConnectionInfo{
			IPAddress:      "",
			UserModel:      "",
			ConnectionType: "websocket",
			Encrypted:      true,
		},
		SessionData:  make(map[string]interface{}),
		MessageCount: 0,
	}

	// Store session
	cis.activeSessions[session.ID] = session
	cis.storeSession(session)

	// Update pairing request
	pairingRequest.Status = "confirmed"
	pairingRequest.ConfirmedAt = &time.Time{}
	*pairingRequest.ConfirmedAt = time.Now()
	pairingRequest.SessionID = session.ID
	cis.storePairingRequest(pairingRequest)

	// Mark QR code as used
	if qrCode, exists := cis.qrCodes[pairingRequest.QRCodeID]; exists {
		qrCode.Status = "used"
		qrCode.UsedAt = &time.Time{}
		*qrCode.UsedAt = time.Now()
		cis.storeQRCode(qrCode)
	}

	log.Printf("Created controller session: %s for user %s", session.ID, session.UserID)
	return session, nil
}

// GetActiveSession returns an active session by ID
func (cis *ControllerIntegrationService) GetActiveSession(sessionID string) (*objects.ControllerSession, error) {
	cis.mu.RLock()
	defer cis.mu.RUnlock()

	session, exists := cis.activeSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session has expired")
	}

	return session, nil
}

// GetUserSessions returns all active sessions for a user
func (cis *ControllerIntegrationService) GetUserSessions(userID string) ([]*objects.ControllerSession, error) {
	cis.mu.RLock()
	defer cis.mu.RUnlock()

	var sessions []*objects.ControllerSession
	for _, session := range cis.activeSessions {
		if session.UserID == userID && session.Status == "active" {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// SendMessage sends a message through a controller session
func (cis *ControllerIntegrationService) SendMessage(sessionID string, message *objects.ControllerMessage) error {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	session, exists := cis.activeSessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if session is active
	if session.Status != "active" {
		return fmt.Errorf("session is not active: %s", session.Status)
	}

	// Update session activity
	session.LastActivity = time.Now()
	session.MessageCount++

	// Add message to queue
	if cis.messageQueue[sessionID] == nil {
		cis.messageQueue[sessionID] = make([]*objects.ControllerMessage, 0)
	}
	cis.messageQueue[sessionID] = append(cis.messageQueue[sessionID], message)

	// Keep only last 100 messages per session
	if len(cis.messageQueue[sessionID]) > 100 {
		cis.messageQueue[sessionID] = cis.messageQueue[sessionID][len(cis.messageQueue[sessionID])-100:]
	}

	log.Printf("Message queued for session %s: %s", sessionID, message.Type)
	return nil
}

// TerminateSession terminates a controller session
func (cis *ControllerIntegrationService) TerminateSession(sessionID string, reason string) error {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	session, exists := cis.activeSessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Update session status
	session.Status = "terminated"
	session.TerminatedAt = &time.Time{}
	*session.TerminatedAt = time.Now()
	session.TerminationReason = reason

	// Store updated session
	cis.storeSession(session)

	// Clean up message queue
	delete(cis.messageQueue, sessionID)

	// Remove from active sessions
	delete(cis.activeSessions, sessionID)

	log.Printf("Terminated session %s: %s", sessionID, reason)
	return nil
}

// Private helper methods
func (cis *ControllerIntegrationService) generateSecurityKeys() {
	// Generate 32-byte encryption key
	cis.encryptionKey = make([]byte, 32)
	rand.Read(cis.encryptionKey)

	// Generate 32-byte signing key
	cis.signingKey = make([]byte, 32)
	rand.Read(cis.signingKey)
}

func (cis *ControllerIntegrationService) generateSecureID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (cis *ControllerIntegrationService) signQRCodeData(data *objects.QRCodeData) (string, error) {
	// Simple signature implementation (in production, use proper HMAC or digital signatures)
	payload := fmt.Sprintf("%s:%s:%s:%d", data.SessionID, data.DesktopID, data.UserID, data.Timestamp)
	signature := base64.StdEncoding.EncodeToString([]byte(payload + string(cis.signingKey)))
	return signature, nil
}

func (cis *ControllerIntegrationService) verifyQRCodeSignature(data *objects.QRCodeData) bool {
	expectedSignature, err := cis.signQRCodeData(data)
	if err != nil {
		return false
	}
	return data.Signature == expectedSignature
}

func (cis *ControllerIntegrationService) countUserQRCodes(userID string) int {
	count := 0
	for _, qr := range cis.qrCodes {
		if qr.UserID == userID && qr.Status == "active" {
			count++
		}
	}
	return count
}

func (cis *ControllerIntegrationService) countUserSessions(userID string) int {
	count := 0
	for _, session := range cis.activeSessions {
		if session.UserID == userID && session.Status == "active" {
			count++
		}
	}
	return count
}

func (cis *ControllerIntegrationService) initializeDatabase() {
	cis.db.Transaction(func(tx *buntdb.Tx) error {
		tx.CreateIndex("controller:sessions", "controller:session:*", buntdb.IndexString)
		tx.CreateIndex("controller:qrcodes", "controller:qrcode:*", buntdb.IndexString)
		tx.CreateIndex("controller:pairings", "controller:pairing:*", buntdb.IndexString)
		return nil
	})
}

func (cis *ControllerIntegrationService) loadControllerData() {
	// Load sessions from database
	cis.db.GetObjectsByPrefix("controller:session:", func(key string, value []byte) bool {
		var session objects.ControllerSession
		if json.Unmarshal(value, &session) == nil {
			cis.activeSessions[session.ID] = &session
		}
		return true
	})

	// Load QR codes from database
	cis.db.GetObjectsByPrefix("controller:qrcode:", func(key string, value []byte) bool {
		var qrCode objects.QRCode
		if json.Unmarshal(value, &qrCode) == nil {
			cis.qrCodes[qrCode.ID] = &qrCode
		}
		return true
	})

	// Load pairing requests from database
	cis.db.GetObjectsByPrefix("controller:pairing:", func(key string, value []byte) bool {
		var pairingRequest objects.PairingRequest
		if json.Unmarshal(value, &pairingRequest) == nil {
			cis.pairingRequests[pairingRequest.ID] = &pairingRequest
		}
		return true
	})
}

func (cis *ControllerIntegrationService) storeSession(session *objects.ControllerSession) {
	if data, err := json.Marshal(session); err == nil {
		cis.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("controller:session:"+session.ID, string(data), nil)
			return nil
		})
	}
}

func (cis *ControllerIntegrationService) storeQRCode(qrCode *objects.QRCode) {
	if data, err := json.Marshal(qrCode); err == nil {
		cis.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("controller:qrcode:"+qrCode.ID, string(data), nil)
			return nil
		})
	}
}

func (cis *ControllerIntegrationService) storePairingRequest(pairingRequest *objects.PairingRequest) {
	if data, err := json.Marshal(pairingRequest); err == nil {
		cis.db.Transaction(func(tx *buntdb.Tx) error {
			tx.Set("controller:pairing:"+pairingRequest.ID, string(data), nil)
			return nil
		})
	}
}

func (cis *ControllerIntegrationService) sessionCleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cis.mu.RLock()
		running := cis.running
		cis.mu.RUnlock()

		if !running {
			return
		}

		cis.cleanupExpiredSessions()
	}
}

func (cis *ControllerIntegrationService) qrCodeCleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cis.mu.RLock()
		running := cis.running
		cis.mu.RUnlock()

		if !running {
			return
		}

		cis.cleanupExpiredQRCodes()
	}
}

func (cis *ControllerIntegrationService) messageProcessingLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cis.mu.RLock()
		running := cis.running
		cis.mu.RUnlock()

		if !running {
			return
		}

		cis.processQueuedMessages()
	}
}

func (cis *ControllerIntegrationService) cleanupExpiredSessions() {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	now := time.Now()
	for sessionID, session := range cis.activeSessions {
		if now.After(session.ExpiresAt) {
			session.Status = "expired"
			session.TerminatedAt = &now
			session.TerminationReason = "expired"

			cis.storeSession(session)
			delete(cis.activeSessions, sessionID)
			delete(cis.messageQueue, sessionID)

			log.Printf("Expired session: %s", sessionID)
		}
	}
}

func (cis *ControllerIntegrationService) cleanupExpiredQRCodes() {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	now := time.Now()
	for qrID, qrCode := range cis.qrCodes {
		if now.After(qrCode.ExpiresAt) && qrCode.Status == "active" {
			qrCode.Status = "expired"
			cis.storeQRCode(qrCode)

			log.Printf("Expired QR code: %s", qrID)
		}
	}
}

// encryptMessagePayload encrypts a message payload using AES-GCM
func (cis *ControllerIntegrationService) encryptMessagePayload(payload interface{}) (map[string]interface{}, error) {
	if cis.encryptionKey == nil {
		return nil, fmt.Errorf("encryption key not available")
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	block, err := aes.NewCipher(cis.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return map[string]interface{}{
		"encrypted":  true,
		"data":       base64.StdEncoding.EncodeToString(ciphertext),
		"nonce_size": gcm.NonceSize(),
	}, nil
}


// processQueuedMessages processes messages in queue using WebSocket delivery
func (cis *ControllerIntegrationService) processQueuedMessages() {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	// Process messages in queue using WebSocket delivery
	for sessionID, messages := range cis.messageQueue {
		if len(messages) == 0 {
			continue
		}

		// Check if session is active
		session, exists := cis.activeSessions[sessionID]
		if !exists || session.Status != "active" {
			// Store messages for offline delivery if session exists but inactive
			if exists && session.Status == "inactive" {
				cis.storeOfflineMessages(sessionID, messages)
			}
			// Clear processed messages
			delete(cis.messageQueue, sessionID)
			continue
		}

		// Deliver messages via WebSocket
		roomName := fmt.Sprintf("controller:%s", sessionID)
		deliveredCount := 0

		for _, message := range messages {
			// Encrypt message if encryption is enabled
			var payload interface{} = message.Payload
			if cis.encryptionKey != nil {
				encrypted, err := cis.encryptMessagePayload(message.Payload)
				if err != nil {
					log.Printf("Failed to encrypt message %s: %v", message.ID, err)
					continue
				}
				payload = encrypted
			}

			roomMessage := &websocket.RoomMessage{
				Room:      roomName,
				Type:      "controller",
				Event:     message.Type,
				Payload:   payload,
				SenderID:  "system",
				Timestamp: message.Timestamp.Format(time.RFC3339),
			}

			// Send via WebSocket
			if cis.websocketService != nil {
				cis.websocketService.BroadcastToRoom(roomName, roomMessage)
			}

			message.Processed = true
			deliveredCount++
		}

		if deliveredCount > 0 {
			log.Printf("Delivered %d messages to session %s", deliveredCount, sessionID)
		}

		// Keep only undelivered messages in queue
		var remainingMessages []*objects.ControllerMessage
		for _, message := range messages {
			if !message.Processed {
				remainingMessages = append(remainingMessages, message)
			}
		}

		if len(remainingMessages) > 0 {
			cis.messageQueue[sessionID] = remainingMessages
		} else {
			delete(cis.messageQueue, sessionID)
		}
	}
}

// storeOfflineMessages stores messages for offline delivery
func (cis *ControllerIntegrationService) storeOfflineMessages(sessionID string, messages []*objects.ControllerMessage) {
	// Store in database for later delivery when session becomes active
	for _, message := range messages {
		offlineKey := fmt.Sprintf("controller:offline:%s:%s", sessionID, message.ID)
		if data, err := json.Marshal(message); err == nil {
			cis.db.Transaction(func(tx *buntdb.Tx) error {
				tx.Set(offlineKey, string(data), nil)
				return nil
			})
		}
	}
	log.Printf("Stored %d messages for offline delivery to session %s", len(messages), sessionID)
}


// HandleControllerCommand processes incoming controller commands
func (cis *ControllerIntegrationService) HandleControllerCommand(sessionID string, command *objects.ControllerMessage) (*objects.ControllerMessage, error) {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	session, exists := cis.activeSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status != "active" {
		return nil, fmt.Errorf("session not active: %s", session.Status)
	}

	// Update session activity
	session.LastActivity = time.Now()

	// Process command based on type
	response := &objects.ControllerMessage{
		ID:        fmt.Sprintf("resp_%d", time.Now().UnixNano()),
		SessionID: sessionID,
		Type:      "response",
		Direction: "outbound",
		Timestamp: time.Now(),
		Processed: false,
	}

	switch command.Type {
	case "ping":
		response.Payload = map[string]interface{}{
			"type":    "pong",
			"session": sessionID,
			"time":    time.Now().Unix(),
		}

	case "status":
		response.Payload = map[string]interface{}{
			"type":           "status_response",
			"session_status": session.Status,
			"capabilities":   session.Capabilities,
			"last_activity":  session.LastActivity.Unix(),
			"message_count":  session.MessageCount,
		}

	case "get_sessions":
		userSessions, err := cis.getUserSessionsInternal(session.UserID)
		if err != nil {
			response.Payload = map[string]interface{}{
				"type":  "error",
				"error": err.Error(),
			}
		} else {
			response.Payload = map[string]interface{}{
				"type":     "sessions_list",
				"sessions": userSessions,
			}
		}

	case "terminate_session":
		targetSessionID, ok := command.Payload["session_id"].(string)
		if !ok {
			response.Payload = map[string]interface{}{
				"type":  "error",
				"error": "missing session_id in payload",
			}
		} else {
			err := cis.TerminateSession(targetSessionID, "terminated by controller")
			if err != nil {
				response.Payload = map[string]interface{}{
					"type":  "error",
					"error": err.Error(),
				}
			} else {
				response.Payload = map[string]interface{}{
					"type":          "session_terminated",
					"session_id":    targetSessionID,
					"terminated_by": sessionID,
				}
			}
		}

	case "send_notification":
		// Send push notification (placeholder - would integrate with push service)
		log.Printf("Push notification requested: %v", command.Payload)
		response.Payload = map[string]interface{}{
			"type": "notification_sent",
			"id":   fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		}

	default:
		response.Payload = map[string]interface{}{
			"type":  "error",
			"error": fmt.Sprintf("unknown command type: %s", command.Type),
		}
	}

	return response, nil
}

// getUserSessionsInternal returns user sessions (internal version without locking)
func (cis *ControllerIntegrationService) getUserSessionsInternal(userID string) ([]map[string]interface{}, error) {
	var sessions []map[string]interface{}
	for _, session := range cis.activeSessions {
		if session.UserID == userID {
			sessionData := map[string]interface{}{
				"id":            session.ID,
				"status":        session.Status,
				"created_at":    session.CreatedAt.Unix(),
				"last_activity": session.LastActivity.Unix(),
				"capabilities":  session.Capabilities,
				"device_type":   session.DeviceInfo.DeviceType,
				"message_count": session.MessageCount,
			}
			sessions = append(sessions, sessionData)
		}
	}
	return sessions, nil
}

// NegotiateCapabilities performs capability negotiation for a session
func (cis *ControllerIntegrationService) NegotiateCapabilities(sessionID string, requestedCapabilities []string) ([]string, error) {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	session, exists := cis.activeSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Check which requested capabilities are supported
	var negotiatedCapabilities []string
	supportedCaps := session.Capabilities

	for _, requested := range requestedCapabilities {
		for _, supported := range supportedCaps {
			if requested == supported {
				negotiatedCapabilities = append(negotiatedCapabilities, requested)
				break
			}
		}
	}

	// Update session with negotiated capabilities
	session.Capabilities = negotiatedCapabilities
	cis.storeSession(session)

	log.Printf("Negotiated capabilities for session %s: %v", sessionID, negotiatedCapabilities)
	return negotiatedCapabilities, nil
}

// SendPushNotification sends a push notification to the controller
func (cis *ControllerIntegrationService) SendPushNotification(sessionID string, title, message string, data map[string]interface{}) error {
	cis.mu.RLock()
	session, exists := cis.activeSessions[sessionID]
	cis.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status != "active" {
		return fmt.Errorf("session not active: %s", session.Status)
	}

	// Create notification message
	notification := &objects.ControllerMessage{
		ID:        fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		SessionID: sessionID,
		Type:      "notification",
		Direction: "outbound",
		Payload: map[string]interface{}{
			"title":   title,
			"message": message,
			"data":    data,
			"time":    time.Now().Unix(),
		},
		Timestamp: time.Now(),
		Processed: false,
	}

	// Send via WebSocket
	return cis.SendMessage(sessionID, notification)
}
