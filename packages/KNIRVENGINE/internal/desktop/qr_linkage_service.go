package desktop

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LinkageType represents the type of QR code linkage
type LinkageType string

const (
	LinkageTypeTargetAssignment LinkageType = "target_assignment"
	LinkageTypeTransactionSign  LinkageType = "transaction_sign"
	LinkageTypeAgentDeploy      LinkageType = "agent_deploy"
	LinkageTypeSecureComms      LinkageType = "secure_comms"
)

// LinkageStatus represents the status of a linkage session
type LinkageStatus string

const (
	LinkageStatusPending   LinkageStatus = "pending"
	LinkageStatusConnected LinkageStatus = "connected"
	LinkageStatusExpired   LinkageStatus = "expired"
	LinkageStatusRejected  LinkageStatus = "rejected"
)

// LinkageSession represents an active QR code linkage session
type LinkageSession struct {
	SessionID      string                 `json:"session_id"`
	DesktopID      string                 `json:"desktop_id"`
	MobileID       string                 `json:"mobile_id,omitempty"`
	TargetSystemID string                 `json:"target_system_id"`
	LinkageType    LinkageType            `json:"linkage_type"`
	ExpiresAt      time.Time              `json:"expires_at"`
	Status         LinkageStatus          `json:"status"`
	EncryptionKey  []byte                 `json:"-"`
	Capabilities   []string               `json:"capabilities"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
}

// QRPayload represents the data encoded in a QR code
type QRPayload struct {
	Version          string   `json:"version"`
	Type             string   `json:"type"`
	SessionID        string   `json:"session_id"`
	DesktopID        string   `json:"desktop_id"`
	TargetID         string   `json:"target_id,omitempty"`
	ExpiresAt        int64    `json:"expires_at"`
	Endpoint         string   `json:"endpoint"`
	PublicKey        string   `json:"public_key"`
	Capabilities     []string `json:"capabilities,omitempty"`
	EncryptedPayload string   `json:"encrypted_payload,omitempty"`
	Signature        string   `json:"signature"`
}

// QRCode represents a generated QR code
type QRCode struct {
	Data      []byte    `json:"data"`
	Image     []byte    `json:"image"`
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TransactionData represents transaction data for signing
type TransactionData struct {
	Hash      string `json:"hash"`
	Amount    string `json:"amount"`
	Recipient string `json:"recipient"`
	GasFee    string `json:"gas_fee"`
	Timestamp int64  `json:"timestamp"`
}

// MobileLinkageData represents data from mobile device during linkage
type MobileLinkageData struct {
	DeviceID      string   `json:"device_id"`
	WalletAddress string   `json:"wallet_address"`
	PublicKey     string   `json:"public_key"`
	Capabilities  []string `json:"capabilities"`
	Signature     string   `json:"signature"`
}

// QRLinkageService manages QR code generation and linkage sessions
type QRLinkageService struct {
	activeSessions map[string]*LinkageSession
	desktopID      string
	endpoint       string
	publicKey      string
	mutex          sync.RWMutex
}

// QRLinkageInterface defines the subset of methods used by DesktopClient and tests
type QRLinkageInterface interface {
	GenerateTargetAssignmentQR(targetSystemID string, capabilities []string) (*QRCode, error)
	GenerateTransactionSignQR(transactionData *TransactionData) (*QRCode, error)
	GetSession(sessionID string) (*LinkageSession, bool)
	UpdateSessionStatus(sessionID string, status LinkageStatus, mobileID string) error
	StartService()
}

// NewQRLinkageService creates a new QR linkage service
func NewQRLinkageService(desktopID, endpoint, publicKey string) *QRLinkageService {
	return &QRLinkageService{
		activeSessions: make(map[string]*LinkageSession),
		desktopID:      desktopID,
		endpoint:       endpoint,
		publicKey:      publicKey,
	}
}

// GenerateTargetAssignmentQR generates a QR code for target system assignment
func (qls *QRLinkageService) GenerateTargetAssignmentQR(targetSystemID string, capabilities []string) (*QRCode, error) {
	qls.mutex.Lock()
	defer qls.mutex.Unlock()

	sessionID := uuid.New().String()
	encryptionKey := generateEncryptionKey()

	session := &LinkageSession{
		SessionID:      sessionID,
		DesktopID:      qls.desktopID,
		TargetSystemID: targetSystemID,
		LinkageType:    LinkageTypeTargetAssignment,
		ExpiresAt:      time.Now().Add(5 * time.Minute),
		Status:         LinkageStatusPending,
		EncryptionKey:  encryptionKey,
		Capabilities:   capabilities,
		Metadata: map[string]interface{}{
			"target_name": fmt.Sprintf("Target-%s", targetSystemID),
			"target_type": "system",
			"created_at":  time.Now(),
		},
		CreatedAt: time.Now(),
	}

	qls.activeSessions[sessionID] = session

	// Create QR code payload
	payload := QRPayload{
		Version:      "1.0",
		Type:         string(LinkageTypeTargetAssignment),
		SessionID:    sessionID,
		DesktopID:    qls.desktopID,
		TargetID:     targetSystemID,
		ExpiresAt:    session.ExpiresAt.Unix(),
		Endpoint:     qls.endpoint,
		PublicKey:    qls.publicKey,
		Capabilities: capabilities,
		Signature:    "mock_signature", // In production, this would be a real signature
	}

	// Generate QR code data
	qrData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal QR payload: %w", err)
	}

	// For now, we'll use a mock QR code image
	// In production, this would generate an actual QR code image
	qrImage := []byte("mock_qr_image_data")

	qrCode := &QRCode{
		Data:      qrData,
		Image:     qrImage,
		SessionID: sessionID,
		ExpiresAt: session.ExpiresAt,
	}

	log.Printf("Generated target assignment QR code: session=%s, target=%s", sessionID, targetSystemID)

	return qrCode, nil
}

// GenerateTransactionSignQR generates a QR code for transaction signing
func (qls *QRLinkageService) GenerateTransactionSignQR(transactionData *TransactionData) (*QRCode, error) {
	qls.mutex.Lock()
	defer qls.mutex.Unlock()

	sessionID := uuid.New().String()
	encryptionKey := generateEncryptionKey()

	session := &LinkageSession{
		SessionID:     sessionID,
		DesktopID:     qls.desktopID,
		LinkageType:   LinkageTypeTransactionSign,
		ExpiresAt:     time.Now().Add(2 * time.Minute), // Shorter expiry for transactions
		Status:        LinkageStatusPending,
		EncryptionKey: encryptionKey,
		Metadata: map[string]interface{}{
			"transaction_hash": transactionData.Hash,
			"amount":           transactionData.Amount,
			"recipient":        transactionData.Recipient,
			"gas_fee":          transactionData.GasFee,
			"created_at":       time.Now(),
		},
		CreatedAt: time.Now(),
	}

	qls.activeSessions[sessionID] = session

	// Encrypt transaction data (mock encryption for now)
	encryptedTxData := base64.StdEncoding.EncodeToString([]byte("encrypted_transaction_data"))

	payload := QRPayload{
		Version:          "1.0",
		Type:             string(LinkageTypeTransactionSign),
		SessionID:        sessionID,
		DesktopID:        qls.desktopID,
		ExpiresAt:        session.ExpiresAt.Unix(),
		Endpoint:         qls.endpoint,
		PublicKey:        qls.publicKey,
		EncryptedPayload: encryptedTxData,
		Signature:        "mock_signature",
	}

	qrData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction QR payload: %w", err)
	}

	qrImage := []byte("mock_transaction_qr_image_data")

	qrCode := &QRCode{
		Data:      qrData,
		Image:     qrImage,
		SessionID: sessionID,
		ExpiresAt: session.ExpiresAt,
	}

	log.Printf("Generated transaction signing QR code: session=%s, hash=%s", sessionID, transactionData.Hash)

	return qrCode, nil
}

// GetSession retrieves a linkage session by ID
func (qls *QRLinkageService) GetSession(sessionID string) (*LinkageSession, bool) {
	qls.mutex.RLock()
	defer qls.mutex.RUnlock()

	session, exists := qls.activeSessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		session.Status = LinkageStatusExpired
		return session, true
	}

	return session, true
}

// UpdateSessionStatus updates the status of a linkage session
func (qls *QRLinkageService) UpdateSessionStatus(sessionID string, status LinkageStatus, mobileID string) error {
	qls.mutex.Lock()
	defer qls.mutex.Unlock()

	session, exists := qls.activeSessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = status
	if mobileID != "" {
		session.MobileID = mobileID
	}

	log.Printf("Updated session status: session=%s, status=%s, mobile=%s", sessionID, status, mobileID)

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (qls *QRLinkageService) CleanupExpiredSessions() {
	qls.mutex.Lock()
	defer qls.mutex.Unlock()

	now := time.Now()
	expiredSessions := []string{}

	for sessionID, session := range qls.activeSessions {
		if now.After(session.ExpiresAt) {
			session.Status = LinkageStatusExpired
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	for _, sessionID := range expiredSessions {
		delete(qls.activeSessions, sessionID)
		log.Printf("Cleaned up expired session: %s", sessionID)
	}
}

// StartService starts the QR linkage service with periodic cleanup
func (qls *QRLinkageService) StartService() {
	log.Printf("Starting QR linkage service: desktop_id=%s, endpoint=%s", qls.desktopID, qls.endpoint)

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			qls.CleanupExpiredSessions()
		}
	}()
}

// generateEncryptionKey generates a random encryption key
func generateEncryptionKey() []byte {
	key := make([]byte, 32) // 256-bit key
	rand.Read(key)
	return key
}

