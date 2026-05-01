package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"backend_server/internal/objects"
	"backend_server/internal/services/dvemanager"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocket message types for Browser DVE communication
const (
	// Server -> Client messages
	WSTaskAssigned = "task_assigned"
	WSPolicySync   = "policy_sync"
	WSBadgeRefresh = "badge_refresh"
	WSHeartbeatAck = "heartbeat_ack"

	// Client -> Server messages
	WSTaskResult       = "task_result"
	WSHeartbeat        = "heartbeat"
	WSCapabilityUpdate = "capability_update"
	WSBadgeSync        = "badge_sync"
	WSRegister         = "ws_register"
)

// BrowserDVEMessage represents a WebSocket message exchanged with a browser DVE
type BrowserDVEMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// BrowserDVEConn represents a single WebSocket connection for a browser DVE
type BrowserDVEConn struct {
	WalletAddress string
	NodeID        string
	Conn          *websocket.Conn
	Send          chan []byte
	LastHeartbeat time.Time
	mu            sync.Mutex
}

// BrowserDVEHub manages all browser DVE WebSocket connections
type BrowserDVEHub struct {
	mu          sync.RWMutex
	connections map[string]*BrowserDVEConn // walletAddress -> conn
	dveManager  *dvemanager.DVEManager
	rateLimiter *BrowserDVERateLimiter
}

// NewBrowserDVEHub creates a new Browser DVE WebSocket hub
func NewBrowserDVEHub(dveManager *dvemanager.DVEManager) *BrowserDVEHub {
	hub := &BrowserDVEHub{
		connections: make(map[string]*BrowserDVEConn),
		dveManager:  dveManager,
		rateLimiter: NewBrowserDVERateLimiter(),
	}

	// Start heartbeat monitor
	go hub.heartbeatMonitor()

	return hub
}

// HandleWebSocket handles the WebSocket upgrade and connection lifecycle
func (hub *BrowserDVEHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authenticate via JWT from query param or Authorization header
	authToken := r.URL.Query().Get("token")
	if authToken == "" {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			authToken = authHeader[7:]
		}
	}

	if authToken == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// For now, extract wallet address from the token query param or URL
	// In production, validate JWT and extract wallet address
	walletAddress := r.URL.Query().Get("wallet")
	if walletAddress == "" {
		walletAddress = "wallet-" + uuid.New().String()[:8]
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Browser DVE WebSocket upgrade error: %v", err)
		return
	}

	browserConn := &BrowserDVEConn{
		WalletAddress: walletAddress,
		NodeID:        "",
		Conn:          conn,
		Send:          make(chan []byte, 256),
		LastHeartbeat: time.Now(),
	}

	hub.mu.Lock()
	hub.connections[walletAddress] = browserConn
	hub.mu.Unlock()

	log.Printf("Browser DVE WebSocket connected: wallet=%s", walletAddress[:12])

	// Start read and write pumps
	go hub.writePump(browserConn)
	go hub.readPump(browserConn)
}

// readPump reads messages from the WebSocket connection
func (hub *BrowserDVEHub) readPump(conn *BrowserDVEConn) {
	defer func() {
		hub.DeregisterConnection(conn.WalletAddress)
		conn.Conn.Close()
	}()

	conn.Conn.SetReadLimit(65536)
	conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.mu.Lock()
		conn.LastHeartbeat = time.Now()
		conn.mu.Unlock()
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Browser DVE WebSocket read error: %v", err)
			}
			break
		}

		var msg BrowserDVEMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Browser DVE invalid message format: %v", err)
			continue
		}

		hub.handleMessage(conn, &msg)
	}
}

// writePump writes messages to the WebSocket connection
func (hub *BrowserDVEHub) writePump(conn *BrowserDVEConn) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Browser DVE WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes an incoming WebSocket message from a browser DVE
func (hub *BrowserDVEHub) handleMessage(conn *BrowserDVEConn, msg *BrowserDVEMessage) {
	switch msg.Type {
	case WSRegister:
		hub.handleRegister(conn, msg.Payload)
	case WSHeartbeat:
		hub.handleHeartbeat(conn)
	case WSTaskResult:
		hub.handleTaskResult(conn, msg.Payload)
	case WSCapabilityUpdate:
		hub.handleCapabilityUpdate(conn, msg.Payload)
	case WSBadgeSync:
		hub.handleBadgeSync(conn, msg.Payload)
	default:
		log.Printf("Browser DVE unknown message type: %s", msg.Type)
	}
}

// handleRegister processes a registration message from a browser DVE
func (hub *BrowserDVEHub) handleRegister(conn *BrowserDVEConn, payload json.RawMessage) {
	var regPayload struct {
		NodeID        string   `json:"node_id"`
		Capabilities  []string `json:"capabilities"`
		BadgeNFTIDs   []string `json:"badge_nft_ids"`
		ExtensionID   string   `json:"extension_id"`
		BrowserVersion string  `json:"browser_version"`
	}

	if err := json.Unmarshal(payload, &regPayload); err != nil {
		log.Printf("Browser DVE invalid register payload: %v", err)
		return
	}

	conn.mu.Lock()
	conn.NodeID = regPayload.NodeID
	conn.mu.Unlock()

	// Update the DVE node in the manager if it exists
	if hub.dveManager != nil {
		updates := map[string]interface{}{
			"status":          "online",
			"extension_id":    regPayload.ExtensionID,
			"browser_version": regPayload.BrowserVersion,
			"capabilities":    regPayload.Capabilities,
		}
		hub.dveManager.UpdateNode(regPayload.NodeID, updates)
	}

	log.Printf("Browser DVE registered: node=%s wallet=%s capabilities=%v",
		regPayload.NodeID[:12], conn.WalletAddress[:12], regPayload.Capabilities)

	// Send acknowledgment
	ackPayload, _ := json.Marshal(map[string]string{
		"node_id": regPayload.NodeID,
		"status":  "registered",
	})
	ackMsg, _ := json.Marshal(BrowserDVEMessage{
		Type:    WSRegister,
		Payload: ackPayload,
	})
	select {
	case conn.Send <- ackMsg:
	default:
	}
}

// handleHeartbeat processes a heartbeat from a browser DVE
func (hub *BrowserDVEHub) handleHeartbeat(conn *BrowserDVEConn) {
	conn.mu.Lock()
	conn.LastHeartbeat = time.Now()
	conn.mu.Unlock()

	// Send heartbeat acknowledgment
	ackPayload, _ := json.Marshal(map[string]interface{}{
		"timestamp": time.Now().Unix(),
	})
	ackMsg, _ := json.Marshal(BrowserDVEMessage{
		Type:    WSHeartbeatAck,
		Payload: ackPayload,
	})
	select {
	case conn.Send <- ackMsg:
	default:
	}
}

// handleTaskResult processes a task result from a browser DVE
func (hub *BrowserDVEHub) handleTaskResult(conn *BrowserDVEConn, payload json.RawMessage) {
	var resultPayload struct {
		TaskID string  `json:"task_id"`
		Status string  `json:"status"`
		Score  float64 `json:"score"`
	}
	if err := json.Unmarshal(payload, &resultPayload); err != nil {
		log.Printf("Browser DVE invalid task result payload: %v", err)
		return
	}

	log.Printf("Browser DVE task result: node=%s task=%s status=%s score=%.2f",
		conn.NodeID, resultPayload.TaskID, resultPayload.Status, resultPayload.Score)

	// Update task in manager
	if hub.dveManager != nil {
		hub.dveManager.UpdateTask(resultPayload.TaskID, map[string]interface{}{
			"status": resultPayload.Status,
		})
	}
}

// handleCapabilityUpdate processes a capability update from a browser DVE
func (hub *BrowserDVEHub) handleCapabilityUpdate(conn *BrowserDVEConn, payload json.RawMessage) {
	var capPayload struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(payload, &capPayload); err != nil {
		log.Printf("Browser DVE invalid capability update payload: %v", err)
		return
	}

	if hub.dveManager != nil && conn.NodeID != "" {
		hub.dveManager.UpdateNode(conn.NodeID, map[string]interface{}{
			"capabilities": capPayload.Capabilities,
		})
	}

	log.Printf("Browser DVE capabilities updated: node=%s capabilities=%v", conn.NodeID, capPayload.Capabilities)
}

// handleBadgeSync processes badge NFT sync from a browser DVE
func (hub *BrowserDVEHub) handleBadgeSync(conn *BrowserDVEConn, payload json.RawMessage) {
	var badgePayload struct {
		BadgeNFTIDs []string `json:"badge_nft_ids"`
	}
	if err := json.Unmarshal(payload, &badgePayload); err != nil {
		log.Printf("Browser DVE invalid badge sync payload: %v", err)
		return
	}

	log.Printf("Browser DVE badge sync: node=%s badges=%v", conn.NodeID, badgePayload.BadgeNFTIDs)
}

// DispatchTask sends a validation task to a specific wallet's browser DVE
func (hub *BrowserDVEHub) DispatchTask(walletAddress string, task *objects.ValidationTask) error {
	hub.mu.RLock()
	conn, exists := hub.connections[walletAddress]
	hub.mu.RUnlock()

	if !exists {
		return nil // Not connected via WS
	}

	// Check rate limit
	if !hub.rateLimiter.AllowTask(walletAddress) {
		log.Printf("Browser DVE rate limit exceeded for wallet: %s", walletAddress[:12])
		return nil
	}

	taskPayload, err := json.Marshal(task)
	if err != nil {
		return err
	}

	msg, err := json.Marshal(BrowserDVEMessage{
		Type:    WSTaskAssigned,
		Payload: taskPayload,
	})
	if err != nil {
		return err
	}

	select {
	case conn.Send <- msg:
		log.Printf("Browser DVE task dispatched: wallet=%s task=%s", walletAddress[:12], task.ID)
	default:
		log.Printf("Browser DVE send buffer full for wallet: %s", walletAddress[:12])
	}

	return nil
}

// BroadcastBadgeRefresh sends a badge refresh notification to a specific wallet
func (hub *BrowserDVEHub) BroadcastBadgeRefresh(walletAddress string) error {
	hub.mu.RLock()
	conn, exists := hub.connections[walletAddress]
	hub.mu.RUnlock()

	if !exists {
		return nil
	}

	payload, _ := json.Marshal(map[string]string{
		"message": "badges_updated",
	})
	msg, _ := json.Marshal(BrowserDVEMessage{
		Type:    WSBadgeRefresh,
		Payload: payload,
	})

	select {
	case conn.Send <- msg:
	default:
	}

	return nil
}

// SendHeartbeatAck sends a heartbeat acknowledgment to a specific wallet
func (hub *BrowserDVEHub) SendHeartbeatAck(walletAddress string) error {
	hub.mu.RLock()
	conn, exists := hub.connections[walletAddress]
	hub.mu.RUnlock()

	if !exists {
		return nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"timestamp": time.Now().Unix(),
	})
	msg, _ := json.Marshal(BrowserDVEMessage{
		Type:    WSHeartbeatAck,
		Payload: payload,
	})

	select {
	case conn.Send <- msg:
	default:
	}

	return nil
}

// DeregisterConnection removes a connection from the hub and marks the node offline
func (hub *BrowserDVEHub) DeregisterConnection(walletAddress string) {
	hub.mu.Lock()
	conn, exists := hub.connections[walletAddress]
	if exists {
		delete(hub.connections, walletAddress)
	}
	hub.mu.Unlock()

	if exists && hub.dveManager != nil && conn.NodeID != "" {
		hub.dveManager.UpdateNodeStatus(conn.NodeID, "offline")
		log.Printf("Browser DVE disconnected: node=%s wallet=%s", conn.NodeID[:12], walletAddress[:12])
	}
}

// GetConnectedCount returns the number of connected browser DVEs
func (hub *BrowserDVEHub) GetConnectedCount() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.connections)
}

// heartbeatMonitor periodically checks for stale connections
func (hub *BrowserDVEHub) heartbeatMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		hub.mu.Lock()
		now := time.Now()
		for walletAddr, conn := range hub.connections {
			conn.mu.Lock()
			lastBeat := conn.LastHeartbeat
			conn.mu.Unlock()

			if now.Sub(lastBeat) > 90*time.Second {
				log.Printf("Browser DVE heartbeat timeout: wallet=%s", walletAddr[:12])
				conn.Conn.Close()
				delete(hub.connections, walletAddr)

				if hub.dveManager != nil && conn.NodeID != "" {
					hub.dveManager.UpdateNodeStatus(conn.NodeID, "offline")
				}
			}
		}
		hub.mu.Unlock()
	}
}

// BrowserDVERateLimiter enforces rate limits per wallet address
type BrowserDVERateLimiter struct {
	mu       sync.Mutex
	counters map[string]*walletRateLimit
}

type walletRateLimit struct {
	taskCount    int
	windowStart  time.Time
	activeTaskID string
}

const (
	maxTasksPerMinute    = 10
	rateLimitWindow      = 1 * time.Minute
)

// NewBrowserDVERateLimiter creates a new rate limiter
func NewBrowserDVERateLimiter() *BrowserDVERateLimiter {
	return &BrowserDVERateLimiter{
		counters: make(map[string]*walletRateLimit),
	}
}

// AllowTask checks if a wallet is allowed to receive a new task
func (rl *BrowserDVERateLimiter) AllowTask(walletAddress string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limit, exists := rl.counters[walletAddress]
	if !exists {
		rl.counters[walletAddress] = &walletRateLimit{
			taskCount:   1,
			windowStart: time.Now(),
		}
		return true
	}

	// Reset window if expired
	if time.Since(limit.windowStart) > rateLimitWindow {
		limit.taskCount = 1
		limit.windowStart = time.Now()
		limit.activeTaskID = ""
		return true
	}

	// Check concurrent task limit
	if limit.activeTaskID != "" {
		return false
	}

	// Check task rate limit
	if limit.taskCount >= maxTasksPerMinute {
		return false
	}

	limit.taskCount++
	return true
}

// SetActiveTask marks a wallet as having an active task
func (rl *BrowserDVERateLimiter) SetActiveTask(walletAddress, taskID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limit, exists := rl.counters[walletAddress]; exists {
		limit.activeTaskID = taskID
	}
}

// ClearActiveTask removes the active task marker for a wallet
func (rl *BrowserDVERateLimiter) ClearActiveTask(walletAddress string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limit, exists := rl.counters[walletAddress]; exists {
		limit.activeTaskID = ""
	}
}

// SendToWallet sends a raw message to a connected wallet's browser DVE
func (hub *BrowserDVEHub) SendToWallet(walletAddress string, msgType string, payload interface{}) error {
	hub.mu.RLock()
	conn, exists := hub.connections[walletAddress]
	hub.mu.RUnlock()

	if !exists {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg, err := json.Marshal(BrowserDVEMessage{
		Type:    msgType,
		Payload: payloadBytes,
	})
	if err != nil {
		return err
	}

	select {
	case conn.Send <- msg:
	default:
	}

	return nil
}

// ensure interfaces are satisfied
var _ = (*BrowserDVEHub).HandleWebSocket
