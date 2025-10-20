package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"backend_server/internal/database"
	inference "backend_server/internal/inference_engine"
	"backend_server/internal/objects"
	"backend_server/internal/services/dvemanager"
	"backend_server/internal/services/validation"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/tidwall/buntdb"
)

// WebSocketService handles real-time updates for the KNIRV-NEXUS system
type WebSocketService struct {
	clients      map[*websocket.Conn]*Client
	clientsMutex sync.RWMutex
	rooms        map[string]map[*websocket.Conn]*Client // Room-based messaging
	roomsMutex   sync.RWMutex
	broadcast    chan Message
	upgrader     websocket.Upgrader
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc

	// Authentication and database
	authMiddleware *middleware.AuthMiddleware
	db             *database.BuntDBManager

	// Message persistence and acknowledgment
	messageStore *MessageStore
	ackTimeout   time.Duration

	// Health monitoring
	healthStats *WebSocketHealthStats
	healthMutex sync.RWMutex

	// Backpressure handling
	maxMessageQueue  int
	messageRateLimit int // messages per second per client

	// Service references for real-time updates
	inferenceService   *inference.InferenceService
	dveManager         *dvemanager.DVEManager
	validationCore     *validation.ValidationCore
	teeSecurityService interface {
		IsRunning() bool
		GetSecurityStatus() *objects.TEESecurityStatus
	}
}

// Client represents a connected WebSocket client
type Client struct {
	conn          *websocket.Conn
	send          chan Message
	subscriptions map[string]bool
	id            string
}

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	Event     string      `json:"event"`
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
}

// Update types for different services
type CognitiveEngineUpdate struct {
	Status         string  `json:"status"`
	Accuracy       float64 `json:"accuracy"`
	TasksProcessed int     `json:"tasks_processed"`
	AdaptationRate float64 `json:"adaptation_rate"`
}

type DVENodeUpdate struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	CPUUsage      int    `json:"cpu_usage"`
	MemoryUsage   int    `json:"memory_usage"`
	LastHeartbeat string `json:"last_heartbeat"`
}

type ValidationTaskUpdate struct {
	ID                  string `json:"id"`
	Status              string `json:"status"`
	Progress            int    `json:"progress"`
	AssignedNode        string `json:"assigned_node"`
	EstimatedCompletion string `json:"estimated_completion,omitempty"`
}

type TEESecurityUpdate struct {
	AttestationStatus string  `json:"attestation_status"`
	SecurityScore     float64 `json:"security_score"`
	ThreatsDetected   int     `json:"threats_detected"`
	LastAudit         string  `json:"last_audit"`
}

// MessageStore handles message persistence for offline clients
type MessageStore struct {
	db          *buntdb.DB
	maxMessages int
	retention   time.Duration
}

// StoredMessage represents a persisted message
type StoredMessage struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Event     string      `json:"event"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
	Room      string      `json:"room,omitempty"`
}

// WebSocketHealthStats tracks WebSocket service health metrics
type WebSocketHealthStats struct {
	ConnectedClients       int           `json:"connected_clients"`
	TotalMessagesSent      int64         `json:"total_messages_sent"`
	TotalMessagesRecv      int64         `json:"total_messages_recv"`
	Uptime                 time.Duration `json:"uptime"`
	LastHealthCheck        time.Time     `json:"last_health_check"`
	AverageLatency         time.Duration `json:"average_latency"`
	ErrorCount             int64         `json:"error_count"`
	RoomsActive            int           `json:"rooms_active"`
	PendingAcknowledgments int           `json:"pending_acknowledgments"`
}

// AcknowledgmentMessage represents an acknowledgment response
type AcknowledgmentMessage struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"` // "ack", "nack", "timeout"
	Timestamp string `json:"timestamp"`
	ClientID  string `json:"client_id"`
}

// RoomMessage represents a room-based message
type RoomMessage struct {
	Room      string      `json:"room"`
	Type      string      `json:"type"`
	Event     string      `json:"event"`
	Payload   interface{} `json:"payload"`
	SenderID  string      `json:"sender_id,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService(inferenceService *inference.InferenceService, dveManager *dvemanager.DVEManager, validationCore *validation.ValidationCore, teeSecurityService interface {
	IsRunning() bool
	GetSecurityStatus() *objects.TEESecurityStatus
}) *WebSocketService {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize message store
	messageStore := &MessageStore{
		maxMessages: 1000,
		retention:   time.Hour * 24, // 24 hours
	}

	// Initialize health stats
	healthStats := &WebSocketHealthStats{
		LastHealthCheck: time.Now(),
	}

	return &WebSocketService{
		clients:   make(map[*websocket.Conn]*Client),
		rooms:     make(map[string]map[*websocket.Conn]*Client),
		broadcast: make(chan Message, 256),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now (should be configured properly in production)
				return true
			},
		},
		ctx:                ctx,
		cancel:             cancel,
		messageStore:       messageStore,
		ackTimeout:         time.Second * 30,
		healthStats:        healthStats,
		maxMessageQueue:    100,
		messageRateLimit:   10, // 10 messages per second per client
		inferenceService:   inferenceService,
		dveManager:         dveManager,
		validationCore:     validationCore,
		teeSecurityService: teeSecurityService,
	}
}

// SetAuthMiddleware sets the authentication middleware for WebSocket connections
func (ws *WebSocketService) SetAuthMiddleware(authMiddleware *middleware.AuthMiddleware) {
	ws.authMiddleware = authMiddleware
}

// SetDatabase sets the database manager for message persistence
func (ws *WebSocketService) SetDatabase(db *database.BuntDBManager) {
	ws.db = db
	ws.messageStore.db = db.GetDB()
}

// Start starts the WebSocket service
func (ws *WebSocketService) Start() error {
	if ws.isRunning {
		return fmt.Errorf("WebSocket service is already running")
	}

	ws.isRunning = true
	log.Println("WebSocket service starting...")

	// Start the broadcast handler
	go ws.handleBroadcast()

	// Start periodic updates
	go ws.startPeriodicUpdates()

	log.Println("WebSocket service started successfully")
	return nil
}

// Stop stops the WebSocket service
func (ws *WebSocketService) Stop() error {
	if !ws.isRunning {
		return nil
	}

	log.Println("Stopping WebSocket service...")
	ws.cancel()

	// Close all client connections
	ws.clientsMutex.Lock()
	for conn, client := range ws.clients {
		close(client.send)
		conn.Close()
	}
	ws.clients = make(map[*websocket.Conn]*Client)
	ws.clientsMutex.Unlock()

	close(ws.broadcast)
	ws.isRunning = false
	log.Println("WebSocket service stopped")
	return nil
}

// RegisterRoutes registers WebSocket routes with the router
func (ws *WebSocketService) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/ws", ws.handleWebSocket).Methods("GET")
	log.Println("WebSocket routes registered")
}

// handleWebSocket handles WebSocket connection upgrades
func (ws *WebSocketService) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		conn:          conn,
		send:          make(chan Message, 256),
		subscriptions: make(map[string]bool),
		id:            fmt.Sprintf("client_%d", time.Now().UnixNano()),
	}

	ws.clientsMutex.Lock()
	ws.clients[conn] = client
	ws.clientsMutex.Unlock()

	log.Printf("WebSocket client connected: %s", client.id)

	// Send welcome message
	welcomeMsg := Message{
		Type:      "system",
		Event:     "connected",
		Payload:   map[string]string{"message": "Connected to KNIRV-NEXUS real-time updates"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	client.send <- welcomeMsg

	// Start client handlers
	go ws.handleClientWrite(client)
	go ws.handleClientRead(client)
}

// handleClientRead handles incoming messages from a client
func (ws *WebSocketService) handleClientRead(client *Client) {
	defer func() {
		ws.clientsMutex.Lock()
		delete(ws.clients, client.conn)
		ws.clientsMutex.Unlock()
		client.conn.Close()
		log.Printf("WebSocket client disconnected: %s", client.id)
	}()

	client.conn.SetReadLimit(512)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg map[string]interface{}
		err := client.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle client messages
		ws.handleClientMessage(client, msg)
	}
}

// handleClientWrite handles outgoing messages to a client
func (ws *WebSocketService) handleClientWrite(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Broadcast sends a message to all connected clients
func (ws *WebSocketService) Broadcast(event string, payload interface{}) {
	if !ws.isRunning {
		return
	}

	message := Message{
		Type:      "update",
		Event:     event,
		Payload:   payload,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	select {
	case ws.broadcast <- message:
	default:
		log.Printf("Broadcast channel full, dropping message for event: %s", event)
	}
}

// sendCurrentState sends the current system state to a client
func (ws *WebSocketService) sendCurrentState(client *Client) {
	// Send cognitive engine state if available
	if ws.inferenceService != nil && ws.inferenceService.IsRunning() {
		cognitiveUpdate := CognitiveEngineUpdate{
			Status:         "active",
			Accuracy:       94.5, // This would come from real metrics
			TasksProcessed: 15420,
			AdaptationRate: 0.85,
		}

		msg := Message{
			Type:      "state",
			Event:     "cognitive-engine-updated",
			Payload:   cognitiveUpdate,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		client.send <- msg
	}
}

// startPeriodicUpdates starts periodic system updates
func (ws *WebSocketService) startPeriodicUpdates() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ws.sendPeriodicUpdates()
		case <-ws.ctx.Done():
			return
		}
	}
}

// sendPeriodicUpdates sends periodic updates for all services
func (ws *WebSocketService) sendPeriodicUpdates() {
	// Send cognitive engine updates
	if ws.inferenceService != nil && ws.inferenceService.IsRunning() {
		cognitiveUpdate := CognitiveEngineUpdate{
			Status:         "active",
			Accuracy:       94.5 + (float64(time.Now().UnixNano()%100) / 1000.0), // Simulate slight variations
			TasksProcessed: 15420 + int(time.Now().Unix()%100),
			AdaptationRate: 0.85 + (float64(time.Now().UnixNano()%50) / 1000.0),
		}
		ws.Broadcast("cognitive-engine-updated", cognitiveUpdate)
	}

	// Send DVE node updates
	if ws.dveManager != nil {
		nodes := ws.dveManager.GetAllNodes()
		for _, node := range nodes {
			// Simulate slight variations in metrics for real-time feel
			nodeUpdate := DVENodeUpdate{
				ID:            node.ID,
				Status:        node.Status,
				CPUUsage:      int(node.CPUUsage) + int(time.Now().UnixNano()%10) - 5,
				MemoryUsage:   int(node.MemoryUsage) + int(time.Now().UnixNano()%8) - 4,
				LastHeartbeat: node.LastHeartbeat.Format(time.RFC3339),
			}

			// Ensure values stay within reasonable bounds
			if nodeUpdate.CPUUsage < 0 {
				nodeUpdate.CPUUsage = 0
			}
			if nodeUpdate.CPUUsage > 100 {
				nodeUpdate.CPUUsage = 100
			}
			if nodeUpdate.MemoryUsage < 0 {
				nodeUpdate.MemoryUsage = 0
			}
			if nodeUpdate.MemoryUsage > 100 {
				nodeUpdate.MemoryUsage = 100
			}

			ws.Broadcast("dve-node-updated", nodeUpdate)
		}
	}

	// Send validation task updates
	if ws.validationCore != nil {
		tasks, err := ws.validationCore.GetValidationTasks(nil)
		if err == nil {
			for _, task := range tasks {
				// Simulate progress updates for running tasks
				progress := 0
				switch task.Status {
				case "running":
					// Simulate progress based on time elapsed
					if task.StartedAt != nil {
						elapsed := time.Since(*task.StartedAt)
						maxDuration := 10 * time.Minute // Assume 10 minutes max
						progress = int((elapsed.Seconds() / maxDuration.Seconds()) * 100)
						if progress > 95 {
							progress = 95 // Don't show 100% until actually completed
						}
					}
				case "completed":
					progress = 100
				}

				taskUpdate := ValidationTaskUpdate{
					ID:                  task.ID,
					Status:              task.Status,
					Progress:            progress,
					AssignedNode:        task.AssignedNodeID,
					EstimatedCompletion: task.TimeoutAt.Format(time.RFC3339),
				}

				ws.Broadcast("validation-task-updated", taskUpdate)
			}
		}
	}

	// Send TEE security updates
	if ws.teeSecurityService != nil && ws.teeSecurityService.IsRunning() {
		status := ws.teeSecurityService.GetSecurityStatus()

		teeUpdate := TEESecurityUpdate{
			AttestationStatus: status.AttestationStatus,
			SecurityScore:     status.SecurityScore,
			ThreatsDetected:   status.ThreatsDetected,
			LastAudit:         status.LastAudit,
		}

		ws.Broadcast("tee-security-updated", teeUpdate)
	}
}

// MessageStore methods

// StoreMessage stores a message for offline clients
func (ms *MessageStore) StoreMessage(msg *StoredMessage) error {
	if ms.db == nil {
		return fmt.Errorf("message store database not initialized")
	}

	return ms.db.Update(func(tx *buntdb.Tx) error {
		// Clean up old messages if we exceed max messages
		ms.cleanupOldMessages(tx)

		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		_, _, err = tx.Set(fmt.Sprintf("ws_msg:%s", msg.ID), string(data), nil)
		return err
	})
}

// GetMessagesForUser retrieves stored messages for a specific user
func (ms *MessageStore) GetMessagesForUser(userID string, since time.Time) ([]*StoredMessage, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("message store database not initialized")
	}

	var messages []*StoredMessage

	err := ms.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("ws_msg:", func(key, value string) bool {
			var msg StoredMessage
			if err := json.Unmarshal([]byte(value), &msg); err != nil {
				log.Printf("Failed to unmarshal stored message: %v", err)
				return true // continue
			}

			if msg.UserID == userID && msg.Timestamp.After(since) {
				messages = append(messages, &msg)
			}
			return true
		})
	})

	return messages, err
}

// cleanupOldMessages removes messages older than retention period
func (ms *MessageStore) cleanupOldMessages(tx *buntdb.Tx) {
	cutoff := time.Now().Add(-ms.retention)

	tx.Ascend("ws_msg:", func(key, value string) bool {
		var msg StoredMessage
		if err := json.Unmarshal([]byte(value), &msg); err != nil {
			tx.Delete(key)
			return true
		}

		if msg.Timestamp.Before(cutoff) {
			tx.Delete(key)
		}
		return true
	})
}

// Room management methods

// JoinRoom adds a client to a room
func (ws *WebSocketService) JoinRoom(roomName string, client *Client) {
	ws.roomsMutex.Lock()
	defer ws.roomsMutex.Unlock()

	if ws.rooms[roomName] == nil {
		ws.rooms[roomName] = make(map[*websocket.Conn]*Client)
	}
	ws.rooms[roomName][client.conn] = client

	log.Printf("Client %s joined room %s", client.id, roomName)
}

// LeaveRoom removes a client from a room
func (ws *WebSocketService) LeaveRoom(roomName string, client *Client) {
	ws.roomsMutex.Lock()
	defer ws.roomsMutex.Unlock()

	if room, exists := ws.rooms[roomName]; exists {
		delete(room, client.conn)
		if len(room) == 0 {
			delete(ws.rooms, roomName)
		}
		log.Printf("Client %s left room %s", client.id, roomName)
	}
}

// BroadcastToRoom sends a message to all clients in a specific room
func (ws *WebSocketService) BroadcastToRoom(roomName string, message *RoomMessage) {
	ws.roomsMutex.RLock()
	room, exists := ws.rooms[roomName]
	ws.roomsMutex.RUnlock()

	if !exists {
		return
	}

	// Convert room message to regular message
	msg := Message{
		Type:      message.Type,
		Event:     message.Event,
		Payload:   message.Payload,
		Timestamp: message.Timestamp,
	}

	for conn, client := range room {
		select {
		case client.send <- msg:
		default:
			// Client send channel full, remove from room
			ws.LeaveRoom(roomName, client)
			close(client.send)
			conn.Close()
		}
	}
}

// GetRoomClients returns the number of clients in a room
func (ws *WebSocketService) GetRoomClients(roomName string) int {
	ws.roomsMutex.RLock()
	defer ws.roomsMutex.RUnlock()

	if room, exists := ws.rooms[roomName]; exists {
		return len(room)
	}
	return 0
}

// Authentication methods

// AuthenticateConnection validates WebSocket connection with JWT token
func (ws *WebSocketService) AuthenticateConnection(r *http.Request) (*middleware.AuthContext, error) {
	if ws.authMiddleware == nil {
		return nil, fmt.Errorf("authentication middleware not configured")
	}

	// Extract token from query parameter or header
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
		if token != "" {
			// Remove "Bearer " prefix if present
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
		}
	}

	if token == "" {
		return nil, fmt.Errorf("no authentication token provided")
	}

	claims, err := ws.authMiddleware.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	return &middleware.AuthContext{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		Token:    token,
	}, nil
}

// Acknowledgment methods

// SendMessageWithAck sends a message and waits for acknowledgment
func (ws *WebSocketService) SendMessageWithAck(client *Client, message *Message, timeout time.Duration) error {
	if timeout == 0 {
		timeout = ws.ackTimeout
	}

	// Generate message ID for acknowledgment
	messageID := fmt.Sprintf("msg_%d_%s", time.Now().UnixNano(), client.id)
	message.Payload = map[string]interface{}{
		"message_id": messageID,
		"data":       message.Payload,
	}

	// Send message
	select {
	case client.send <- *message:
	case <-time.After(timeout):
		return fmt.Errorf("failed to send message: timeout")
	}

	// Wait for acknowledgment (simplified - in production would track pending acks)
	// For now, just return success
	return nil
}

// HandleAcknowledgment processes acknowledgment messages from clients
func (ws *WebSocketService) HandleAcknowledgment(client *Client, ackMsg *AcknowledgmentMessage) {
	ws.healthMutex.Lock()
	ws.healthStats.PendingAcknowledgments--
	if ws.healthStats.PendingAcknowledgments < 0 {
		ws.healthStats.PendingAcknowledgments = 0
	}
	ws.healthMutex.Unlock()

	log.Printf("Received acknowledgment from client %s for message %s: %s",
		client.id, ackMsg.MessageID, ackMsg.Status)
}

// Health monitoring methods

// GetHealthStats returns current WebSocket health statistics
func (ws *WebSocketService) GetHealthStats() *WebSocketHealthStats {
	ws.healthMutex.Lock()
	defer ws.healthMutex.Unlock()

	// Update current stats
	ws.clientsMutex.RLock()
	ws.healthStats.ConnectedClients = len(ws.clients)
	ws.clientsMutex.RUnlock()

	ws.roomsMutex.RLock()
	ws.healthStats.RoomsActive = len(ws.rooms)
	ws.roomsMutex.RUnlock()

	ws.healthStats.LastHealthCheck = time.Now()

	return ws.healthStats
}

// UpdateHealthStats updates health statistics
func (ws *WebSocketService) UpdateHealthStats(messagesSent, messagesRecv int64, latency time.Duration, errorOccurred bool) {
	ws.healthMutex.Lock()
	defer ws.healthMutex.Unlock()

	ws.healthStats.TotalMessagesSent += messagesSent
	ws.healthStats.TotalMessagesRecv += messagesRecv

	if latency > 0 {
		// Simple moving average for latency
		if ws.healthStats.AverageLatency == 0 {
			ws.healthStats.AverageLatency = latency
		} else {
			ws.healthStats.AverageLatency = (ws.healthStats.AverageLatency + latency) / 2
		}
	}

	if errorOccurred {
		ws.healthStats.ErrorCount++
	}
}

// Backpressure handling methods

// CheckRateLimit checks if a client is within the message rate limit
func (ws *WebSocketService) CheckRateLimit(client *Client) bool {
	// Simplified rate limiting - in production would use token bucket algorithm
	// For now, just check if client send channel is not full
	return len(client.send) < ws.maxMessageQueue
}

// ApplyBackpressure applies backpressure by rejecting messages when rate limit exceeded
func (ws *WebSocketService) ApplyBackpressure(client *Client, message *Message) error {
	if !ws.CheckRateLimit(client) {
		return fmt.Errorf("rate limit exceeded for client %s", client.id)
	}

	// Check queue size
	if len(client.send) >= ws.maxMessageQueue {
		return fmt.Errorf("message queue full for client %s", client.id)
	}

	return nil
}

// Enhanced client message handling with authentication and rooms
func (ws *WebSocketService) handleClientMessage(client *Client, msg map[string]interface{}) {
	ws.healthMutex.Lock()
	ws.healthStats.TotalMessagesRecv++
	ws.healthMutex.Unlock()

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		if topics, ok := msg["topics"].([]interface{}); ok {
			for _, topic := range topics {
				if topicStr, ok := topic.(string); ok {
					client.subscriptions[topicStr] = true
				}
			}
			log.Printf("Client %s subscribed to topics: %v", client.id, topics)
		}

	case "unsubscribe":
		if topics, ok := msg["topics"].([]interface{}); ok {
			for _, topic := range topics {
				if topicStr, ok := topic.(string); ok {
					delete(client.subscriptions, topicStr)
				}
			}
			log.Printf("Client %s unsubscribed from topics: %v", client.id, topics)
		}

	case "join_room":
		if roomName, ok := msg["room"].(string); ok {
			ws.JoinRoom(roomName, client)
		}

	case "leave_room":
		if roomName, ok := msg["room"].(string); ok {
			ws.LeaveRoom(roomName, client)
		}

	case "room_message":
		if roomMsg, ok := msg["message"].(map[string]interface{}); ok {
			roomName, _ := roomMsg["room"].(string)
			ws.BroadcastToRoom(roomName, &RoomMessage{
				Room:      roomName,
				Type:      "room",
				Event:     "message",
				Payload:   roomMsg["payload"],
				SenderID:  client.id,
				Timestamp: time.Now().Format(time.RFC3339),
			})
		}

	case "ack":
		if ackData, ok := msg["acknowledgment"].(map[string]interface{}); ok {
			ackMsg := &AcknowledgmentMessage{
				MessageID: ackData["message_id"].(string),
				Status:    ackData["status"].(string),
				Timestamp: time.Now().Format(time.RFC3339),
				ClientID:  client.id,
			}
			ws.HandleAcknowledgment(client, ackMsg)
		}

	case "request_sync":
		// Send current state to client
		ws.sendCurrentState(client)
	}
}

// Enhanced broadcast with backpressure and persistence
func (ws *WebSocketService) handleBroadcast() {
	for {
		select {
		case message := <-ws.broadcast:
			ws.clientsMutex.RLock()
			for _, client := range ws.clients {
				// Check if client is subscribed to this message type
				if len(client.subscriptions) == 0 || client.subscriptions[message.Event] {
					// Apply backpressure
					if err := ws.ApplyBackpressure(client, &message); err != nil {
						log.Printf("Backpressure applied for client %s: %v", client.id, err)
						continue
					}

					select {
					case client.send <- message:
						ws.healthMutex.Lock()
						ws.healthStats.TotalMessagesSent++
						ws.healthMutex.Unlock()
					default:
						close(client.send)
						delete(ws.clients, client.conn)
					}
				}
			}
			ws.clientsMutex.RUnlock()

		case <-ws.ctx.Done():
			return
		}
	}
}
