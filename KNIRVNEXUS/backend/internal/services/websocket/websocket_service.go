package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"backend-server/internal/inference"
	"backend-server/internal/objects"
	"backend-server/internal/services/dvemanager"
	"backend-server/internal/services/validation"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// WebSocketService handles real-time updates for the KNIRV-NEXUS system
type WebSocketService struct {
	clients      map[*websocket.Conn]*Client
	clientsMutex sync.RWMutex
	broadcast    chan Message
	upgrader     websocket.Upgrader
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc

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

// NewWebSocketService creates a new WebSocket service
func NewWebSocketService(inferenceService *inference.InferenceService, dveManager *dvemanager.DVEManager, validationCore *validation.ValidationCore, teeSecurityService interface {
	IsRunning() bool
	GetSecurityStatus() *objects.TEESecurityStatus
}) *WebSocketService {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketService{
		clients:   make(map[*websocket.Conn]*Client),
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
		inferenceService:   inferenceService,
		dveManager:         dveManager,
		validationCore:     validationCore,
		teeSecurityService: teeSecurityService,
	}
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

// handleClientMessage processes messages from clients
func (ws *WebSocketService) handleClientMessage(client *Client, msg map[string]interface{}) {
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

	case "request_sync":
		// Send current state to client
		ws.sendCurrentState(client)
	}
}

// handleBroadcast handles broadcasting messages to all clients
func (ws *WebSocketService) handleBroadcast() {
	for {
		select {
		case message := <-ws.broadcast:
			ws.clientsMutex.RLock()
			for _, client := range ws.clients {
				// Check if client is subscribed to this message type
				if len(client.subscriptions) == 0 || client.subscriptions[message.Event] {
					select {
					case client.send <- message:
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
				if task.Status == "running" {
					// Simulate progress based on time elapsed
					if task.StartedAt != nil {
						elapsed := time.Since(*task.StartedAt)
						maxDuration := 10 * time.Minute // Assume 10 minutes max
						progress = int((elapsed.Seconds() / maxDuration.Seconds()) * 100)
						if progress > 95 {
							progress = 95 // Don't show 100% until actually completed
						}
					}
				} else if task.Status == "completed" {
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
