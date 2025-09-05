package controllerintegration

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nexus-backend/internal/models"

	"github.com/tidwall/buntdb"
)

// ControllerIntegrationService manages advanced controller integration
type ControllerIntegrationService struct {
	db      *buntdb.DB
	mu      sync.RWMutex
	running bool

	// Session management
	activeSessions  map[string]*models.ControllerSession
	qrCodes         map[string]*models.QRCode
	pairingRequests map[string]*models.PairingRequest

	// Real-time communication
	websocketConnections map[string]*models.WebSocketConnection
	messageQueue         map[string][]*models.ControllerMessage

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
func NewControllerIntegrationService(db *buntdb.DB) *ControllerIntegrationService {
	service := &ControllerIntegrationService{
		db:                   db,
		activeSessions:       make(map[string]*models.ControllerSession),
		qrCodes:              make(map[string]*models.QRCode),
		pairingRequests:      make(map[string]*models.PairingRequest),
		websocketConnections: make(map[string]*models.WebSocketConnection),
		messageQueue:         make(map[string][]*models.ControllerMessage),
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

// GenerateQRCode creates a new QR code for controller pairing
func (cis *ControllerIntegrationService) GenerateQRCode(userID, deviceType string, capabilities []string) (*models.QRCode, error) {
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
	qrCode := &models.QRCode{
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
		Data: &models.QRCodeData{
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
func (cis *ControllerIntegrationService) ScanQRCode(qrData string, mobileDeviceID string) (*models.PairingRequest, error) {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	// Parse QR code data
	var qrCodeData models.QRCodeData
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
	var qrCode *models.QRCode
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
	pairingRequest := &models.PairingRequest{
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
		DeviceInfo: &models.DeviceInfo{
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
func (cis *ControllerIntegrationService) ConfirmPairing(pairingRequestID string, confirmed bool) (*models.ControllerSession, error) {
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
	session := &models.ControllerSession{
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
		ConnectionInfo: &models.ConnectionInfo{
			IPAddress:      "",
			UserAgent:      "",
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
func (cis *ControllerIntegrationService) GetActiveSession(sessionID string) (*models.ControllerSession, error) {
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
func (cis *ControllerIntegrationService) GetUserSessions(userID string) ([]*models.ControllerSession, error) {
	cis.mu.RLock()
	defer cis.mu.RUnlock()

	var sessions []*models.ControllerSession
	for _, session := range cis.activeSessions {
		if session.UserID == userID && session.Status == "active" {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// SendMessage sends a message through a controller session
func (cis *ControllerIntegrationService) SendMessage(sessionID string, message *models.ControllerMessage) error {
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
		cis.messageQueue[sessionID] = make([]*models.ControllerMessage, 0)
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

func (cis *ControllerIntegrationService) signQRCodeData(data *models.QRCodeData) (string, error) {
	// Simple signature implementation (in production, use proper HMAC or digital signatures)
	payload := fmt.Sprintf("%s:%s:%s:%d", data.SessionID, data.DesktopID, data.UserID, data.Timestamp)
	signature := base64.StdEncoding.EncodeToString([]byte(payload + string(cis.signingKey)))
	return signature, nil
}

func (cis *ControllerIntegrationService) verifyQRCodeSignature(data *models.QRCodeData) bool {
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
	cis.db.Update(func(tx *buntdb.Tx) error {
		tx.CreateIndex("controller:sessions", "controller:session:*", buntdb.IndexString)
		tx.CreateIndex("controller:qrcodes", "controller:qrcode:*", buntdb.IndexString)
		tx.CreateIndex("controller:pairings", "controller:pairing:*", buntdb.IndexString)
		return nil
	})
}

func (cis *ControllerIntegrationService) loadControllerData() {
	// Load sessions from database
	cis.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("controller:sessions", func(key, value string) bool {
			var session models.ControllerSession
			if json.Unmarshal([]byte(value), &session) == nil {
				cis.activeSessions[session.ID] = &session
			}
			return true
		})
		return nil
	})

	// Load QR codes from database
	cis.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("controller:qrcodes", func(key, value string) bool {
			var qrCode models.QRCode
			if json.Unmarshal([]byte(value), &qrCode) == nil {
				cis.qrCodes[qrCode.ID] = &qrCode
			}
			return true
		})
		return nil
	})

	// Load pairing requests from database
	cis.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("controller:pairings", func(key, value string) bool {
			var pairingRequest models.PairingRequest
			if json.Unmarshal([]byte(value), &pairingRequest) == nil {
				cis.pairingRequests[pairingRequest.ID] = &pairingRequest
			}
			return true
		})
		return nil
	})
}

func (cis *ControllerIntegrationService) storeSession(session *models.ControllerSession) {
	if data, err := json.Marshal(session); err == nil {
		cis.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("controller:session:"+session.ID, string(data), nil)
			return nil
		})
	}
}

func (cis *ControllerIntegrationService) storeQRCode(qrCode *models.QRCode) {
	if data, err := json.Marshal(qrCode); err == nil {
		cis.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("controller:qrcode:"+qrCode.ID, string(data), nil)
			return nil
		})
	}
}

func (cis *ControllerIntegrationService) storePairingRequest(pairingRequest *models.PairingRequest) {
	if data, err := json.Marshal(pairingRequest); err == nil {
		cis.db.Update(func(tx *buntdb.Tx) error {
			tx.Set("controller:pairing:"+pairingRequest.ID, string(data), nil)
			return nil
		})
	}
}

func (cis *ControllerIntegrationService) sessionCleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !cis.running {
				return
			}

			cis.cleanupExpiredSessions()
		}
	}
}

func (cis *ControllerIntegrationService) qrCodeCleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !cis.running {
				return
			}

			cis.cleanupExpiredQRCodes()
		}
	}
}

func (cis *ControllerIntegrationService) messageProcessingLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !cis.running {
				return
			}

			cis.processQueuedMessages()
		}
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

func (cis *ControllerIntegrationService) processQueuedMessages() {
	cis.mu.Lock()
	defer cis.mu.Unlock()

	// Process messages in queue (placeholder for real-time WebSocket delivery)
	for _, messages := range cis.messageQueue {
		if len(messages) > 0 {
			// Mark messages as processed
			for _, message := range messages {
				message.Processed = true
			}
		}
	}
}
