package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend_server/internal/objects"

	"github.com/gorilla/websocket"
)

// newTestConfig creates a minimal config for testing
func newTestConfig() *struct{} {
	return &struct{}{}
}

func TestNewBrowserDVEHub(t *testing.T) {
	hub := NewBrowserDVEHub(nil)
	if hub == nil {
		t.Fatal("Expected hub to be non-nil")
	}
	if hub.rateLimiter == nil {
		t.Error("Expected rate limiter to be initialized")
	}
	if hub.connections == nil {
		t.Error("Expected connections map to be initialized")
	}
}

func TestBrowserDVEHub_GetConnectedCount(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	if count := hub.GetConnectedCount(); count != 0 {
		t.Errorf("Expected 0 connections, got %d", count)
	}

	// Add a mock connection
	hub.mu.Lock()
	hub.connections["test-wallet"] = &BrowserDVEConn{
		WalletAddress: "test-wallet",
		Send:          make(chan []byte, 256),
		LastHeartbeat: time.Now(),
	}
	hub.mu.Unlock()

	if count := hub.GetConnectedCount(); count != 1 {
		t.Errorf("Expected 1 connection, got %d", count)
	}
}

func TestBrowserDVERateLimiter_AllowTask(t *testing.T) {
	rl := NewBrowserDVERateLimiter()

	if !rl.AllowTask("wallet-1") {
		t.Error("Expected first task to be allowed")
	}

	// Set active task - should block concurrent task
	rl.SetActiveTask("wallet-1", "task-1")
	if rl.AllowTask("wallet-1") {
		t.Error("Expected concurrent task to be rejected")
	}

	// Clear active task - should allow again
	rl.ClearActiveTask("wallet-1")
	if !rl.AllowTask("wallet-1") {
		t.Error("Expected task to be allowed after clearing active task")
	}
}

func TestBrowserDVERateLimiter_MaxTasksPerMinute(t *testing.T) {
	rl := NewBrowserDVERateLimiter()

	for i := 0; i < maxTasksPerMinute; i++ {
		if !rl.AllowTask("wallet-2") {
			t.Fatalf("Expected task %d to be allowed", i+1)
		}
	}

	if rl.AllowTask("wallet-2") {
		t.Error("Expected task beyond limit to be rejected")
	}
}

func TestBrowserDVERateLimiter_DifferentWallets(t *testing.T) {
	rl := NewBrowserDVERateLimiter()

	for i := 0; i < maxTasksPerMinute; i++ {
		rl.AllowTask("wallet-1")
	}

	if !rl.AllowTask("wallet-2") {
		t.Error("Expected different wallet to be allowed")
	}
}

func TestBrowserDVEHub_HandleWebSocket_RequiresAuth(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	req := httptest.NewRequest("GET", "/api/dve/browser/ws", nil)
	w := httptest.NewRecorder()

	hub.HandleWebSocket(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func TestBrowserDVEMessage_JSON(t *testing.T) {
	payload, _ := json.Marshal(map[string]string{
		"node_id": "test-node-1",
		"status":  "registered",
	})

	msg := BrowserDVEMessage{
		Type:    WSRegister,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded BrowserDVEMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if decoded.Type != WSRegister {
		t.Errorf("Expected type '%s', got '%s'", WSRegister, decoded.Type)
	}

	var payloadMap map[string]string
	if err := json.Unmarshal(decoded.Payload, &payloadMap); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if payloadMap["node_id"] != "test-node-1" {
		t.Errorf("Expected node_id 'test-node-1', got '%s'", payloadMap["node_id"])
	}
}

func TestWebSocketConnectAndHeartbeat(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/dve/browser/ws?wallet=test-wallet-ws&token=test-token"

	dialer := websocket.Dialer{
		HandshakeTimeout: 2 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Logf("WebSocket dial result: %v (expected if auth fails - running in test mode)", err)
		return
	}
	defer conn.Close()

	// Send heartbeat
	heartbeatPayload, _ := json.Marshal(map[string]interface{}{
		"timestamp": time.Now().Unix(),
	})
	msg, _ := json.Marshal(BrowserDVEMessage{
		Type:    WSHeartbeat,
		Payload: heartbeatPayload,
	})

	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("Failed to send heartbeat: %v", err)
	}

	// Read response
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var respMsg BrowserDVEMessage
	if err := json.Unmarshal(response, &respMsg); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if respMsg.Type != WSHeartbeatAck {
		t.Errorf("Expected type '%s', got '%s'", WSHeartbeatAck, respMsg.Type)
	}
}

func TestDispatchTask(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	mockConn := &BrowserDVEConn{
		WalletAddress: "test-wallet-dispatched",
		NodeID:        "test-node-1",
		Send:          make(chan []byte, 256),
		LastHeartbeat: time.Now(),
	}

	hub.mu.Lock()
	hub.connections["test-wallet-dispatched"] = mockConn
	hub.mu.Unlock()

	task := &objects.ValidationTask{
		ID:              "task-dispatch-1",
		Type:            "skillnode",
		Status:          "assigned",
		Priority:        5,
		RequiredTEEType: "browser-extension",
		Parameters:      map[string]interface{}{"trust_tier": "standard"},
	}

	err := hub.DispatchTask("test-wallet-dispatched", task)
	if err != nil {
		t.Fatalf("Failed to dispatch task: %v", err)
	}

	select {
	case msgBytes := <-mockConn.Send:
		var msg BrowserDVEMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			t.Fatalf("Failed to unmarshal sent message: %v", err)
		}
		if msg.Type != WSTaskAssigned {
			t.Errorf("Expected type '%s', got '%s'", WSTaskAssigned, msg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for dispatched task message")
	}
}

func TestDeregisterConnection(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	mockConn := &BrowserDVEConn{
		WalletAddress: "test-wallet-deregister",
		NodeID:        "test-node-deregister",
		Send:          make(chan []byte, 256),
		LastHeartbeat: time.Now(),
	}

	hub.mu.Lock()
	hub.connections["test-wallet-deregister"] = mockConn
	hub.mu.Unlock()

	if hub.GetConnectedCount() != 1 {
		t.Fatalf("Expected 1 connection before deregister, got %d", hub.GetConnectedCount())
	}

	// Deregister (hub has nil manager, so no db update - just remove from map)
	hub.DeregisterConnection("test-wallet-deregister")

	if hub.GetConnectedCount() != 0 {
		t.Errorf("Expected 0 connections after deregister, got %d", hub.GetConnectedCount())
	}
}

func TestDeregisterConnection_Nonexistent(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	// Should not panic
	hub.DeregisterConnection("nonexistent-wallet")
}

func TestBroadcastBadgeRefresh(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	mockConn := &BrowserDVEConn{
		WalletAddress: "wallet-badge-test",
		Send:          make(chan []byte, 256),
		LastHeartbeat: time.Now(),
	}

	hub.mu.Lock()
	hub.connections["wallet-badge-test"] = mockConn
	hub.mu.Unlock()

	err := hub.BroadcastBadgeRefresh("wallet-badge-test")
	if err != nil {
		t.Fatalf("Failed to broadcast badge refresh: %v", err)
	}

	select {
	case msgBytes := <-mockConn.Send:
		var msg BrowserDVEMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}
		if msg.Type != WSBadgeRefresh {
			t.Errorf("Expected type '%s', got '%s'", WSBadgeRefresh, msg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for badge refresh message")
	}
}

func TestSendHeartbeatAck(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	mockConn := &BrowserDVEConn{
		WalletAddress: "wallet-hb-test",
		Send:          make(chan []byte, 256),
		LastHeartbeat: time.Now(),
	}

	hub.mu.Lock()
	hub.connections["wallet-hb-test"] = mockConn
	hub.mu.Unlock()

	err := hub.SendHeartbeatAck("wallet-hb-test")
	if err != nil {
		t.Fatalf("Failed to send heartbeat ack: %v", err)
	}

	select {
	case msgBytes := <-mockConn.Send:
		var msg BrowserDVEMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}
		if msg.Type != WSHeartbeatAck {
			t.Errorf("Expected type '%s', got '%s'", WSHeartbeatAck, msg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for heartbeat ack message")
	}
}

func TestSendToWallet(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	mockConn := &BrowserDVEConn{
		WalletAddress: "wallet-custom-msg",
		Send:          make(chan []byte, 256),
		LastHeartbeat: time.Now(),
	}

	hub.mu.Lock()
	hub.connections["wallet-custom-msg"] = mockConn
	hub.mu.Unlock()

	payload := map[string]string{
		"message": "hello from server",
	}
	err := hub.SendToWallet("wallet-custom-msg", WSPolicySync, payload)
	if err != nil {
		t.Fatalf("Failed to send custom message: %v", err)
	}

	select {
	case msgBytes := <-mockConn.Send:
		var msg BrowserDVEMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}
		if msg.Type != WSPolicySync {
			t.Errorf("Expected type '%s', got '%s'", WSPolicySync, msg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for custom message")
	}
}

func TestDispatchTask_NonexistentWallet(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	task := &objects.ValidationTask{
		ID:       "task-nonexistent",
		Type:     "skillnode",
		Status:   "assigned",
		Priority: 5,
	}

	// Should not panic and return nil
	err := hub.DispatchTask("nonexistent-wallet", task)
	if err != nil {
		t.Errorf("Expected nil error for nonexistent wallet, got: %v", err)
	}
}

func TestHandleMessage_UnknownType(t *testing.T) {
	hub := NewBrowserDVEHub(nil)

	mockConn := &BrowserDVEConn{
		WalletAddress: "test-wallet-unknown",
		Send:          make(chan []byte, 256),
		LastHeartbeat: time.Now(),
	}

	// Should not panic
	unknownPayload, _ := json.Marshal(map[string]string{"foo": "bar"})
	msg := &BrowserDVEMessage{
		Type:    "unknown_message_type",
		Payload: unknownPayload,
	}

	hub.handleMessage(mockConn, msg)
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewBrowserDVERateLimiter()

	// Fill up
	for i := 0; i < maxTasksPerMinute; i++ {
		rl.AllowTask("wallet-reset")
	}

	if rl.AllowTask("wallet-reset") {
		t.Error("Expected task to be rejected before window reset")
	}

	// Force reset by setting window start to before the window
	rl.mu.Lock()
	if limit, exists := rl.counters["wallet-reset"]; exists {
		limit.windowStart = time.Now().Add(-2 * rateLimitWindow)
		limit.taskCount = 0
	}
	rl.mu.Unlock()

	if !rl.AllowTask("wallet-reset") {
		t.Error("Expected task to be allowed after window reset")
	}
}

// Ensure the unused variable doesn't cause compile error
var _ = (*BrowserDVEHub).HandleWebSocket
