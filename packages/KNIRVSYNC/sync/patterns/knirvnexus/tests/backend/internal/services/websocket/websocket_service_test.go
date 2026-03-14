package websocket

import (
	"backend_server/internal/objects"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Mock TEE security service for testing
type mockTEESecurityService struct {
	running bool
}

func (m *mockTEESecurityService) IsRunning() bool {
	return m.running
}

func (m *mockTEESecurityService) GetSecurityStatus() *objects.TEESecurityStatus {
	return &objects.TEESecurityStatus{
		AttestationStatus:  "verified",
		EnclaveCount:       3,
		SecurityScore:      95.5,
		LastAudit:          time.Now().Format(time.RFC3339),
		ThreatsDetected:    0,
		ActiveThreats:      []*objects.ThreatAlert{},
		AuditHistory:       []*objects.SecurityAudit{},
		PerformanceMetrics: &objects.TEEPerformanceMetrics{},
		TEEType:            "Intel SGX",
		LastAttestation:    time.Now().Format(time.RFC3339),
		MonitoringEnabled:  true,
	}
}

func TestNewWebSocketService(t *testing.T) {
	mockTEE := &mockTEESecurityService{running: true}

	service := NewWebSocketService(nil, nil, nil, mockTEE)

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if service.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if service.broadcast == nil {
		t.Error("Expected broadcast channel to be initialized")
	}

	if service.isRunning {
		t.Error("Expected service to not be running initially")
	}

	if service.teeSecurityService != mockTEE {
		t.Error("Expected TEE security service to be set correctly")
	}
}

func TestWebSocketService_Start(t *testing.T) {
	mockTEE := &mockTEESecurityService{running: true}

	service := NewWebSocketService(nil, nil, nil, mockTEE)

	// Test starting the service
	err := service.Start()
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	if !service.isRunning {
		t.Error("Expected service to be running after start")
	}

	// Test starting already running service
	err = service.Start()
	if err == nil {
		t.Error("Expected error when starting already running service")
	}

	// Clean up
	service.Stop()
}

func TestWebSocketService_Stop(t *testing.T) {
	mockTEE := &mockTEESecurityService{running: true}

	service := NewWebSocketService(nil, nil, nil, mockTEE)

	// Test stopping non-running service (should not error)
	err := service.Stop()
	if err != nil {
		t.Errorf("Unexpected error when stopping non-running service: %v", err)
	}

	// Start and then stop
	service.Start()
	err = service.Stop()
	if err != nil {
		t.Fatalf("Failed to stop service: %v", err)
	}

	if service.isRunning {
		t.Error("Expected service to not be running after stop")
	}
}

func TestWebSocketService_RegisterRoutes(t *testing.T) {
	mockTEE := &mockTEESecurityService{running: true}

	service := NewWebSocketService(nil, nil, nil, mockTEE)
	router := mux.NewRouter()

	// Register routes
	service.RegisterRoutes(router)

	// Test that the route was registered
	req, err := http.NewRequest("GET", "/ws", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// The request should be handled (even if it fails WebSocket upgrade)
	// We expect a 400 Bad Request because we're not sending proper WebSocket headers
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestWebSocketService_Broadcast(t *testing.T) {
	mockTEE := &mockTEESecurityService{running: true}

	service := NewWebSocketService(nil, nil, nil, mockTEE)

	// Test broadcasting when service is not running (should not panic)
	service.Broadcast("test_event", map[string]interface{}{"data": "test"})

	// Start service and test broadcasting
	service.Start()
	defer service.Stop()

	// Test broadcasting with running service
	service.Broadcast("test_event", map[string]interface{}{"data": "test"})

	// Give some time for the broadcast to be processed
	time.Sleep(10 * time.Millisecond)

	// The broadcast should not cause any errors
}

func TestWebSocketService_HandleWebSocket(t *testing.T) {
	mockTEE := &mockTEESecurityService{running: true}

	service := NewWebSocketService(nil, nil, nil, mockTEE)
	service.Start()
	defer service.Stop()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(service.handleWebSocket))
	defer server.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.1
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Test WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Send a test message
	testMsg := map[string]interface{}{
		"type": "subscribe",
		"data": map[string]interface{}{
			"events": []string{"cognitive_engine", "dve_nodes"},
		},
	}

	err = conn.WriteJSON(testMsg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Set read deadline to avoid hanging
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Try to read a response (might timeout, which is okay)
	var response map[string]interface{}
	err = conn.ReadJSON(&response)
	if err != nil {
		// Timeout or connection closed is expected in this test
		if !websocket.IsCloseError(err, websocket.CloseNormalClosure) &&
			!strings.Contains(err.Error(), "timeout") {
			t.Logf("WebSocket read error (expected): %v", err)
		}
	}
}

func TestMessage_Struct(t *testing.T) {
	timestamp := time.Now().Format(time.RFC3339)
	msg := Message{
		Type:      "test",
		Event:     "test_event",
		Payload:   map[string]interface{}{"data": "test"},
		Timestamp: timestamp,
	}

	if msg.Type != "test" {
		t.Error("Expected Type to be set correctly")
	}

	if msg.Event != "test_event" {
		t.Error("Expected Event to be set correctly")
	}

	if msg.Payload == nil {
		t.Error("Expected Payload to be set")
	}

	if msg.Timestamp == "" {
		t.Error("Expected Timestamp to be set")
	}
}

func TestClient_Struct(t *testing.T) {
	// Create a mock connection (we can't easily test the real websocket.Conn)
	client := &Client{
		send:          make(chan Message, 256),
		subscriptions: make(map[string]bool),
		id:            "test-client-1",
	}

	if client.send == nil {
		t.Error("Expected send channel to be initialized")
	}

	if client.subscriptions == nil {
		t.Error("Expected subscriptions map to be initialized")
	}

	if client.id != "test-client-1" {
		t.Error("Expected ID to be set correctly")
	}

	// Test that we can send to the channel
	testMsg := Message{Type: "test"}
	select {
	case client.send <- testMsg:
		// Success
	default:
		t.Error("Failed to send message to client channel")
	}

	// Test that we can receive from the channel
	select {
	case receivedMsg := <-client.send:
		if receivedMsg.Type != "test" {
			t.Error("Expected to receive the test message")
		}
	default:
		t.Error("Failed to receive message from client channel")
	}
}
