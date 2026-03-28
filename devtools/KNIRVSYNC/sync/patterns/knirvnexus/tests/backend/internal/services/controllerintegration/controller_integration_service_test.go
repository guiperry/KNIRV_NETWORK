package controllerintegration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/websocket"
	"github.com/tidwall/buntdb"
)

func TestNewControllerIntegrationService(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)

	if service == nil {
		t.Fatal("NewControllerIntegrationService returned nil")
	}

	if service.db != db {
		t.Error("Database not set correctly")
	}

	if service.running {
		t.Error("Service should not be running initially")
	}

	if len(service.activeSessions) != 0 {
		t.Error("Active sessions map should be empty initially")
	}

	if len(service.qrCodes) != 0 {
		t.Error("QR codes map should be empty initially")
	}

	if len(service.pairingRequests) != 0 {
		t.Error("Pairing requests map should be empty initially")
	}
}

func TestControllerIntegrationService_Start(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)

	err = service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	if !service.IsRunning() {
		t.Error("Service should be running after Start()")
	}

	// Test starting already running service
	err = service.Start()
	if err == nil {
		t.Error("Expected error when starting already running service")
	}
}

func TestControllerIntegrationService_Stop(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()

	err = service.Stop()
	if err != nil {
		t.Fatalf("Failed to stop service: %v", err)
	}

	if service.IsRunning() {
		t.Error("Service should not be running after Stop()")
	}

	// Test stopping already stopped service
	err = service.Stop()
	if err == nil {
		t.Error("Expected error when stopping already stopped service")
	}
}

func TestControllerIntegrationService_IsRunning(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)

	if service.IsRunning() {
		t.Error("Service should not be running initially")
	}

	service.Start()

	if !service.IsRunning() {
		t.Error("Service should be running after Start()")
	}

	service.Stop()

	if service.IsRunning() {
		t.Error("Service should not be running after Stop()")
	}
}

func TestControllerIntegrationService_SetWebSocketService(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)

	mockWS := &mockWebSocketService{}
	service.SetWebSocketService(mockWS)

	if service.websocketService == nil {
		t.Error("WebSocket service not set correctly")
	}
}

func TestControllerIntegrationService_GenerateQRCode(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	userID := "test-user"
	deviceType := "mobile"
	capabilities := []string{"control", "monitor"}

	qrCode, err := service.GenerateQRCode(userID, deviceType, capabilities)
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	if qrCode == nil {
		t.Fatal("Generated QR code is nil")
	}

	if qrCode.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, qrCode.UserID)
	}

	if qrCode.DeviceType != deviceType {
		t.Errorf("Expected DeviceType %s, got %s", deviceType, qrCode.DeviceType)
	}

	if len(qrCode.Capabilities) != len(capabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(capabilities), len(qrCode.Capabilities))
	}

	if qrCode.Status != "active" {
		t.Errorf("Expected status 'active', got %s", qrCode.Status)
	}

	if qrCode.ScanCount != 0 {
		t.Errorf("Expected ScanCount 0, got %d", qrCode.ScanCount)
	}

	if qrCode.MaxScans != 1 {
		t.Errorf("Expected MaxScans 1, got %d", qrCode.MaxScans)
	}
}

func TestControllerIntegrationService_ScanQRCode(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	// Generate QR code first
	userID := "test-user"
	deviceType := "mobile"
	capabilities := []string{"control", "monitor"}

	qrCode, err := service.GenerateQRCode(userID, deviceType, capabilities)
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	// Convert QR code data to JSON for scanning
	qrData, err := json.Marshal(qrCode.Data)
	if err != nil {
		t.Fatalf("Failed to marshal QR code data: %v", err)
	}

	mobileDeviceID := "mobile-device-123"

	pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
	if err != nil {
		t.Fatalf("Failed to scan QR code: %v", err)
	}

	if pairingRequest == nil {
		t.Fatal("Pairing request is nil")
	}

	if pairingRequest.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, pairingRequest.UserID)
	}

	if pairingRequest.MobileDeviceID != mobileDeviceID {
		t.Errorf("Expected MobileDeviceID %s, got %s", mobileDeviceID, pairingRequest.MobileDeviceID)
	}

	if pairingRequest.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", pairingRequest.Status)
	}
}

func TestControllerIntegrationService_ConfirmPairing(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	// Generate QR code and scan it
	userID := "test-user"
	deviceType := "mobile"
	capabilities := []string{"control", "monitor"}

	qrCode, err := service.GenerateQRCode(userID, deviceType, capabilities)
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	qrData, err := json.Marshal(qrCode.Data)
	if err != nil {
		t.Fatalf("Failed to marshal QR code data: %v", err)
	}

	mobileDeviceID := "mobile-device-123"

	pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
	if err != nil {
		t.Fatalf("Failed to scan QR code: %v", err)
	}

	// Confirm pairing
	session, err := service.ConfirmPairing(pairingRequest.ID, true)
	if err != nil {
		t.Fatalf("Failed to confirm pairing: %v", err)
	}

	if session == nil {
		t.Fatal("Session is nil")
	}

	if session.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, session.UserID)
	}

	if session.MobileDeviceID != mobileDeviceID {
		t.Errorf("Expected MobileDeviceID %s, got %s", mobileDeviceID, session.MobileDeviceID)
	}

	if session.Status != "active" {
		t.Errorf("Expected status 'active', got %s", session.Status)
	}
}

func TestControllerIntegrationService_GetActiveSession(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	// Create a session first
	userID := "test-user"
	deviceType := "mobile"
	capabilities := []string{"control", "monitor"}

	qrCode, err := service.GenerateQRCode(userID, deviceType, capabilities)
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	qrData, err := json.Marshal(qrCode.Data)
	if err != nil {
		t.Fatalf("Failed to marshal QR code data: %v", err)
	}

	mobileDeviceID := "mobile-device-123"

	pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
	if err != nil {
		t.Fatalf("Failed to scan QR code: %v", err)
	}

	session, err := service.ConfirmPairing(pairingRequest.ID, true)
	if err != nil {
		t.Fatalf("Failed to confirm pairing: %v", err)
	}

	// Get active session
	retrievedSession, err := service.GetActiveSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get active session: %v", err)
	}

	if retrievedSession.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, retrievedSession.ID)
	}

	if retrievedSession.Status != "active" {
		t.Errorf("Expected status 'active', got %s", retrievedSession.Status)
	}
}

func TestControllerIntegrationService_GetUserSessions(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	userID := "test-user"

	// Create multiple sessions for the user
	for i := 0; i < 3; i++ {
		qrCode, err := service.GenerateQRCode(userID, "mobile", []string{"control"})
		if err != nil {
			t.Fatalf("Failed to generate QR code %d: %v", i, err)
		}

		qrData, err := json.Marshal(qrCode.Data)
		if err != nil {
			t.Fatalf("Failed to marshal QR code data %d: %v", i, err)
		}

		mobileDeviceID := fmt.Sprintf("mobile-device-%d", i)

		pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
		if err != nil {
			t.Fatalf("Failed to scan QR code %d: %v", i, err)
		}

		_, err = service.ConfirmPairing(pairingRequest.ID, true)
		if err != nil {
			t.Fatalf("Failed to confirm pairing %d: %v", i, err)
		}
	}

	sessions, err := service.GetUserSessions(userID)
	if err != nil {
		t.Fatalf("Failed to get user sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	for _, session := range sessions {
		if session.UserID != userID {
			t.Errorf("Expected UserID %s, got %s", userID, session.UserID)
		}
		if session.Status != "active" {
			t.Errorf("Expected status 'active', got %s", session.Status)
		}
	}
}

func TestControllerIntegrationService_SendMessage(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	// Create a session first
	userID := "test-user"
	qrCode, err := service.GenerateQRCode(userID, "mobile", []string{"control"})
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	qrData, err := json.Marshal(qrCode.Data)
	if err != nil {
		t.Fatalf("Failed to marshal QR code data: %v", err)
	}

	mobileDeviceID := "mobile-device-123"

	pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
	if err != nil {
		t.Fatalf("Failed to scan QR code: %v", err)
	}

	session, err := service.ConfirmPairing(pairingRequest.ID, true)
	if err != nil {
		t.Fatalf("Failed to confirm pairing: %v", err)
	}

	// Send a message
	message := &objects.ControllerMessage{
		ID:        "test-message-1",
		SessionID: session.ID,
		Type:      "test",
		Direction: "outbound",
		Payload: map[string]interface{}{
			"key": "value",
		},
		Timestamp: time.Now(),
		Processed: false,
	}

	err = service.SendMessage(session.ID, message)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Check if message is in queue
	if len(service.messageQueue[session.ID]) != 1 {
		t.Errorf("Expected 1 message in queue, got %d", len(service.messageQueue[session.ID]))
	}
}

func TestControllerIntegrationService_TerminateSession(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	// Create a session first
	userID := "test-user"
	qrCode, err := service.GenerateQRCode(userID, "mobile", []string{"control"})
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	qrData, err := json.Marshal(qrCode.Data)
	if err != nil {
		t.Fatalf("Failed to marshal QR code data: %v", err)
	}

	mobileDeviceID := "mobile-device-123"

	pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
	if err != nil {
		t.Fatalf("Failed to scan QR code: %v", err)
	}

	session, err := service.ConfirmPairing(pairingRequest.ID, true)
	if err != nil {
		t.Fatalf("Failed to confirm pairing: %v", err)
	}

	// Terminate session
	reason := "test termination"
	err = service.TerminateSession(session.ID, reason)
	if err != nil {
		t.Fatalf("Failed to terminate session: %v", err)
	}

	// Check if session is terminated
	retrievedSession, err := service.GetActiveSession(session.ID)
	if err == nil {
		t.Error("Expected error when getting terminated session")
	}

	if retrievedSession != nil {
		t.Error("Expected nil session for terminated session")
	}
}

func TestControllerIntegrationService_HandleControllerCommand(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	// Create a session first
	userID := "test-user"
	qrCode, err := service.GenerateQRCode(userID, "mobile", []string{"control"})
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	qrData, err := json.Marshal(qrCode.Data)
	if err != nil {
		t.Fatalf("Failed to marshal QR code data: %v", err)
	}

	mobileDeviceID := "mobile-device-123"

	pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
	if err != nil {
		t.Fatalf("Failed to scan QR code: %v", err)
	}

	session, err := service.ConfirmPairing(pairingRequest.ID, true)
	if err != nil {
		t.Fatalf("Failed to confirm pairing: %v", err)
	}

	// Test ping command
	pingCommand := &objects.ControllerMessage{
		ID:        "ping-1",
		SessionID: session.ID,
		Type:      "ping",
		Direction: "inbound",
		Payload:   map[string]interface{}{},
		Timestamp: time.Now(),
		Processed: false,
	}

	response, err := service.HandleControllerCommand(session.ID, pingCommand)
	if err != nil {
		t.Fatalf("Failed to handle ping command: %v", err)
	}

	if response.Type != "response" {
		t.Errorf("Expected response type 'response', got %s", response.Type)
	}

	payload := response.Payload

	if payload["type"] != "pong" {
		t.Errorf("Expected payload type 'pong', got %v", payload["type"])
	}

	// Test status command
	statusCommand := &objects.ControllerMessage{
		ID:        "status-1",
		SessionID: session.ID,
		Type:      "status",
		Direction: "inbound",
		Payload:   map[string]interface{}{},
		Timestamp: time.Now(),
		Processed: false,
	}

	response, err = service.HandleControllerCommand(session.ID, statusCommand)
	if err != nil {
		t.Fatalf("Failed to handle status command: %v", err)
	}

	payload = response.Payload

	if payload["type"] != "status_response" {
		t.Errorf("Expected payload type 'status_response', got %v", payload["type"])
	}
}

func TestControllerIntegrationService_NegotiateCapabilities(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	// Create a session first
	userID := "test-user"
	capabilities := []string{"control", "monitor", "admin"}
	qrCode, err := service.GenerateQRCode(userID, "mobile", capabilities)
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	qrData, err := json.Marshal(qrCode.Data)
	if err != nil {
		t.Fatalf("Failed to marshal QR code data: %v", err)
	}

	mobileDeviceID := "mobile-device-123"

	pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
	if err != nil {
		t.Fatalf("Failed to scan QR code: %v", err)
	}

	session, err := service.ConfirmPairing(pairingRequest.ID, true)
	if err != nil {
		t.Fatalf("Failed to confirm pairing: %v", err)
	}

	// Negotiate capabilities
	requestedCapabilities := []string{"control", "monitor", "unknown"}
	negotiatedCapabilities, err := service.NegotiateCapabilities(session.ID, requestedCapabilities)
	if err != nil {
		t.Fatalf("Failed to negotiate capabilities: %v", err)
	}

	expectedCapabilities := []string{"control", "monitor"}
	if len(negotiatedCapabilities) != len(expectedCapabilities) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCapabilities), len(negotiatedCapabilities))
	}

	for i, cap := range expectedCapabilities {
		if i >= len(negotiatedCapabilities) || negotiatedCapabilities[i] != cap {
			t.Errorf("Expected capability %s at index %d, got %v", cap, i, negotiatedCapabilities)
		}
	}
}

func TestControllerIntegrationService_SendPushNotification(t *testing.T) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	service := NewControllerIntegrationService(db)
	service.Start()
	defer service.Stop()

	// Create a session first
	userID := "test-user"
	qrCode, err := service.GenerateQRCode(userID, "mobile", []string{"control"})
	if err != nil {
		t.Fatalf("Failed to generate QR code: %v", err)
	}

	qrData, err := json.Marshal(qrCode.Data)
	if err != nil {
		t.Fatalf("Failed to marshal QR code data: %v", err)
	}

	mobileDeviceID := "mobile-device-123"

	pairingRequest, err := service.ScanQRCode(string(qrData), mobileDeviceID)
	if err != nil {
		t.Fatalf("Failed to scan QR code: %v", err)
	}

	session, err := service.ConfirmPairing(pairingRequest.ID, true)
	if err != nil {
		t.Fatalf("Failed to confirm pairing: %v", err)
	}

	// Send push notification
	title := "Test Notification"
	message := "This is a test notification"
	data := map[string]interface{}{
		"key": "value",
	}

	err = service.SendPushNotification(session.ID, title, message, data)
	if err != nil {
		t.Fatalf("Failed to send push notification: %v", err)
	}

	// Check if notification is in queue
	if len(service.messageQueue[session.ID]) != 1 {
		t.Errorf("Expected 1 message in queue, got %d", len(service.messageQueue[session.ID]))
	}

	queuedMessage := service.messageQueue[session.ID][0]
	if queuedMessage.Type != "notification" {
		t.Errorf("Expected message type 'notification', got %s", queuedMessage.Type)
	}
}

// Mock WebSocket service for testing
type mockWebSocketService struct{}

func (m *mockWebSocketService) BroadcastToRoom(roomName string, message *websocket.RoomMessage) {
	// Mock implementation - do nothing
}

// Helper function for generating test IDs
func generateTestID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// Example usage of generateTestID to prevent unused function warning
func _() {
	_ = generateTestID("test")
}