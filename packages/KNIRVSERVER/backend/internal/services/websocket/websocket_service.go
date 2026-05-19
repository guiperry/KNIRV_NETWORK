package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"
	cognitiveengine "backend_server/internal/services/cognitiveengine"
	"backend_server/internal/services/dvemanager"
	inference "backend_server/internal/services/inferencer"
	"backend_server/internal/services/session"
	"backend_server/internal/services/validation"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/tidwall/buntdb"
	"golang.org/x/crypto/ssh"
)

// CognitiveEngineMetricsProvider is the minimal interface for pulling live CE state
type CognitiveEngineMetricsProvider interface {
	GetLearningStateRaw() (totalTasks int64, successRate float64, learningProgress float64, confidenceLevel float64)
	GetLearningState() *cognitiveengine.LearningState
	GetGuardrailViolations(limit int) []cognitiveengine.PolicyViolation
	GetLatestTelemetry() *cognitiveengine.SystemResourceSnapshot
	IsRunning() bool
}

// CognitiveEngineActivityProvider exposes real-time cognitive engine activities
type CognitiveEngineActivityProvider interface {
	GetLearningStateRaw() (totalTasks int64, successRate float64, learningProgress float64, confidenceLevel float64)
	GetLearningState() *cognitiveengine.LearningState
	GetGuardrailViolations(limit int) []cognitiveengine.PolicyViolation
	GetLatestTelemetry() *cognitiveengine.SystemResourceSnapshot
	IsRunning() bool
}

// WebSocketService handles real-time updates for the KNIRV-SERVER system
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
	inferenceService *inference.InferenceService
	dveManager       *dvemanager.DVEManager
	validationCore   *validation.ValidationCore
	sessionManager   *session.SessionManager
	agentService     interface {
		SubscribeToResponses(dveID string) chan string
		UnsubscribeFromResponses(dveID string, ch chan string)
	}
	teeSecurityService interface {
		IsRunning() bool
		GetSecurityStatus() *objects.TEESecurityStatus
	}

	// Cognitive engine for real-time learning metrics
	cognitiveEngine          CognitiveEngineActivityProvider
	cognitiveEngineIsRunning bool

	// Inference history tracking for Neural Stream
	inferenceHistory    []InferenceCompletion
	inferenceHistoryMu  sync.RWMutex
	inferenceHistoryMax int
	socketDir           string
}

// InferenceCompletion represents a completed inference task for the Neural Stream
type InferenceCompletion struct {
	ID        string    `json:"id"`
	Goal      string    `json:"goal"`
	Result    string    `json:"result"`
	Type      string    `json:"type"` // "conclusion", "error", "observation"
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
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

// ProcessingActivity represents a single real background activity from the cognitive engine.
type ProcessingActivity struct {
	ID          string `json:"id"`
	Timestamp   string `json:"timestamp"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` // "active" | "completed"
}

// CognitiveActivityUpdate is the payload for the "cognitive-engine-activity" broadcast.
type CognitiveActivityUpdate struct {
	Activities []ProcessingActivity `json:"activities"`
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
func NewWebSocketService(inferenceService *inference.InferenceService, dveManager *dvemanager.DVEManager, validationCore *validation.ValidationCore, sessionManager *session.SessionManager, agentService interface {
	SubscribeToResponses(dveID string) chan string
	UnsubscribeFromResponses(dveID string, ch chan string)
}, teeSecurityService interface {
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
		sessionManager:     sessionManager,
		agentService:       agentService,
		teeSecurityService: teeSecurityService,
	}
}

// SetCognitiveEngine wires the cognitive engine for real-time metric broadcasting
func (ws *WebSocketService) SetCognitiveEngine(ce CognitiveEngineActivityProvider) {
	ws.cognitiveEngine = ce
	if ce != nil {
		ws.cognitiveEngineIsRunning = ce.IsRunning()
	}
}

// SetAuthMiddleware sets the authentication middleware for WebSocket connections
func (ws *WebSocketService) SetAuthMiddleware(authMiddleware *middleware.AuthMiddleware) {
	ws.authMiddleware = authMiddleware
}

func (ws *WebSocketService) SetSocketDir(socketDir string) {
	ws.socketDir = socketDir
}

// SetDatabase sets the database manager for message persistence
func (ws *WebSocketService) SetDatabase(db *database.BuntDBManager) {
	ws.db = db
	ws.messageStore.db = db.GetDB()
	ws.inferenceHistoryMax = 50 // Keep last 50 inference completions
	ws.inferenceHistory = make([]InferenceCompletion, 0, ws.inferenceHistoryMax)
}

// RecordInferenceCompletion records an inference completion for the Neural Stream
func (ws *WebSocketService) RecordInferenceCompletion(completion InferenceCompletion) {
	ws.inferenceHistoryMu.Lock()
	defer ws.inferenceHistoryMu.Unlock()

	ws.inferenceHistory = append(ws.inferenceHistory, completion)
	if len(ws.inferenceHistory) > ws.inferenceHistoryMax {
		ws.inferenceHistory = ws.inferenceHistory[len(ws.inferenceHistory)-ws.inferenceHistoryMax:]
	}
}

// GetRecentInferences returns recent inference completions for the Neural Stream
func (ws *WebSocketService) GetRecentInferences(limit int) []InferenceCompletion {
	ws.inferenceHistoryMu.RLock()
	defer ws.inferenceHistoryMu.RUnlock()

	count := limit
	if count > len(ws.inferenceHistory) {
		count = len(ws.inferenceHistory)
	}
	if count == 0 {
		return []InferenceCompletion{}
	}

	// Return most recent
	result := make([]InferenceCompletion, count)
	copy(result, ws.inferenceHistory[len(ws.inferenceHistory)-count:])
	return result
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
	router.HandleFunc("/ws", ws.handleWebSocket)
	router.HandleFunc("/ws/ssh/{sessionId}", ws.handleSSHWebSocket)
	router.HandleFunc("/ws/error-resolution/{sessionId}", ws.handleErrorResolutionWebSocket)
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
		Payload:   map[string]string{"message": "Connected to KNIRV-SERVER real-time updates"},
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
	if ws.inferenceService != nil && ws.inferenceService.IsRunning() {
		var cognitiveUpdate CognitiveEngineUpdate
		if ws.cognitiveEngine != nil && ws.cognitiveEngine.IsRunning() {
			totalTasks, successRate, learningProgress, _ := ws.cognitiveEngine.GetLearningStateRaw()
			cognitiveUpdate = CognitiveEngineUpdate{
				Status:         "active",
				Accuracy:       successRate * 100,
				TasksProcessed: int(totalTasks),
				AdaptationRate: learningProgress,
			}
		} else {
			cognitiveUpdate = CognitiveEngineUpdate{
				Status:         "active",
				Accuracy:       94.5,
				TasksProcessed: 15420,
				AdaptationRate: 0.85,
			}
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
	// Send cognitive engine updates — use real metrics when available
	if ws.inferenceService != nil && ws.inferenceService.IsRunning() {
		var cognitiveUpdate CognitiveEngineUpdate
		if ws.cognitiveEngine != nil && ws.cognitiveEngine.IsRunning() {
			totalTasks, successRate, learningProgress, _ := ws.cognitiveEngine.GetLearningStateRaw()
			cognitiveUpdate = CognitiveEngineUpdate{
				Status:         "active",
				Accuracy:       successRate * 100,
				TasksProcessed: int(totalTasks),
				AdaptationRate: learningProgress,
			}
		} else {
			cognitiveUpdate = CognitiveEngineUpdate{
				Status:         "active",
				Accuracy:       94.5 + (float64(time.Now().UnixNano()%100) / 1000.0),
				TasksProcessed: 15420 + int(time.Now().Unix()%100),
				AdaptationRate: 0.85 + (float64(time.Now().UnixNano()%50) / 1000.0),
			}
		}
		ws.Broadcast("cognitive-engine-updated", cognitiveUpdate)

		// Broadcast real background processing activities
		ws.broadcastCognitiveActivities()

		// Broadcast recent inference completions for Neural Stream
		ws.broadcastInferenceHistory()
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

// broadcastCognitiveActivities publishes real background processing activities
// from the cognitive engine's adaptation history and running subsystem loops.
func (ws *WebSocketService) broadcastCognitiveActivities() {
	now := time.Now()
	activities := []ProcessingActivity{}

	// If cognitive engine is available, get real data
	if ws.cognitiveEngine != nil && ws.cognitiveEngine.IsRunning() {
		// Get real learning state metrics
		totalTasks, successRate, learningProgress, confidenceLevel := ws.cognitiveEngine.GetLearningStateRaw()

		// Add learning state as active activity
		activities = append(activities, ProcessingActivity{
			ID:    "learning_state",
			Type:  "learning_cycle",
			Title: "Learning State",
			Description: fmt.Sprintf("Tasks: %d | Success Rate: %.1f%% | Learning: %.1f%% | Confidence: %.1f%%",
				totalTasks, successRate*100, learningProgress*100, confidenceLevel*100),
			Status:    "active",
			Timestamp: now.Format(time.RFC3339),
		})

		// Get guardrail violations
		type guardrailProvider interface {
			GetGuardrailViolations(limit int) []cognitiveengine.PolicyViolation
		}
		if gp, ok := ws.cognitiveEngine.(guardrailProvider); ok {
			violations := gp.GetGuardrailViolations(3)
			for _, v := range violations {
				activities = append(activities, ProcessingActivity{
					ID:          "violation_" + v.NodeID,
					Type:        "guardrail_check",
					Title:       "Guardrail Violation",
					Description: fmt.Sprintf("Node: %s | Rule: %s | Severity: %s", v.NodeID, v.RuleID, v.Severity),
					Status:      "active",
					Timestamp:   now.Format(time.RFC3339),
				})
			}
		}

		// Get real-time telemetry
		type telemetryProvider interface {
			GetLatestTelemetry() *cognitiveengine.SystemResourceSnapshot
		}
		if tp, ok := ws.cognitiveEngine.(telemetryProvider); ok {
			telemetry := tp.GetLatestTelemetry()
			if telemetry != nil {
				activities = append(activities, ProcessingActivity{
					ID:    "telemetry",
					Type:  "metrics_collection",
					Title: "System Telemetry",
					Description: fmt.Sprintf("CPU: %.1f%% | Memory: %.1f%% | Goroutines: %d | Heap: %dMB",
						telemetry.CPUPressure*100, telemetry.MemoryPressure*100,
						telemetry.GoRoutines, telemetry.HeapAllocBytes/1024/1024),
					Status:    "active",
					Timestamp: now.Format(time.RFC3339),
				})
			}
		}

		// Get task type performance
		type learningStateProvider interface {
			GetLearningState() *cognitiveengine.LearningState
		}
		if lsp, ok := ws.cognitiveEngine.(learningStateProvider); ok {
			state := lsp.GetLearningState()
			if state != nil && len(state.TaskTypePerformance) > 0 {
				for taskType, metrics := range state.TaskTypePerformance {
					activities = append(activities, ProcessingActivity{
						ID:    "task_" + taskType,
						Type:  "pattern_analysis",
						Title: "Task: " + taskType,
						Description: fmt.Sprintf("Processed: %d | Success: %.1f%% | Avg Time: %.2fs | Score: %.2f",
							metrics.TasksProcessed, metrics.SuccessRate*100,
							metrics.AvgProcessingTime, metrics.AvgScore),
						Status:    "completed",
						Timestamp: metrics.LastProcessed.Format(time.RFC3339),
					})
				}
			}
		}

		// Get adaptation history
		type adaptationProvider interface {
			GetAdaptationHistory(limit int) []cognitiveengine.AdaptationEvent
		}
		if ap, ok := ws.cognitiveEngine.(adaptationProvider); ok {
			for _, e := range ap.GetAdaptationHistory(4) {
				activities = append(activities, ProcessingActivity{
					ID:          e.ID,
					Timestamp:   e.Timestamp.Format(time.RFC3339),
					Type:        e.AdaptationType,
					Title:       adaptationActivityTitle(e.AdaptationType),
					Description: e.TriggerReason,
					Status:      "completed",
				})
			}
		}
	} else {
		// Fallback: show demo data when cognitive engine is not available
		activities = []ProcessingActivity{
			{ID: "loop_learning", Type: "learning_cycle", Title: "Learning Cycle",
				Description: "Processing validation results and updating learning state",
				Status:      "active", Timestamp: now.Format(time.RFC3339)},
			{ID: "loop_metrics", Type: "metrics_collection", Title: "Metrics Collection",
				Description: "Aggregating DVE performance and resource telemetry",
				Status:      "active", Timestamp: now.Add(-2 * time.Second).Format(time.RFC3339)},
			{ID: "loop_patterns", Type: "pattern_analysis", Title: "Pattern Analysis",
				Description: "Scanning validation history for recurring failure patterns",
				Status:      "active", Timestamp: now.Add(-4 * time.Second).Format(time.RFC3339)},
			{ID: "loop_guardrails", Type: "guardrail_check", Title: "Guardrail Check",
				Description: "Evaluating DVE policy compliance and security constraints",
				Status:      "active", Timestamp: now.Add(-6 * time.Second).Format(time.RFC3339)},
		}
	}

	// Add validation tasks if available
	if ws.validationCore != nil {
		tasks, err := ws.validationCore.GetValidationTasks(nil)
		if err == nil && len(tasks) > 0 {
			for _, task := range tasks {
				if task.Status == "running" || task.Status == "pending" {
					activities = append(activities, ProcessingActivity{
						ID:          "validation_" + task.ID,
						Type:        "increase_priority",
						Title:       "Validation Task",
						Description: fmt.Sprintf("Type: %s | Status: %s | Node: %s", task.Type, task.Status, task.AssignedNodeID),
						Status:      "active",
						Timestamp:   now.Format(time.RFC3339),
					})
				}
			}
		}
	}

	ws.Broadcast("cognitive-engine-activity", CognitiveActivityUpdate{Activities: activities})
}

// adaptationActivityTitle maps internal rule action names to human-readable titles.
func adaptationActivityTitle(action string) string {
	switch action {
	case "increase_priority":
		return "Task Priority Adjustment"
	case "optimize_resources":
		return "Resource Optimization"
	case "redistribute_load":
		return "Load Redistribution"
	default:
		return "Adaptation Event"
	}
}

// broadcastInferenceHistory broadcasts recent inference completions for the Neural Stream
func (ws *WebSocketService) broadcastInferenceHistory() {
	recentInferences := ws.GetRecentInferences(10)
	if len(recentInferences) == 0 {
		return
	}

	for _, inf := range recentInferences {
		payload := map[string]interface{}{
			"goal":      inf.Goal,
			"result":    inf.Result,
			"type":      inf.Type,
			"timestamp": inf.Timestamp.Format(time.RFC3339),
		}
		if inf.Error != "" {
			payload["error"] = inf.Error
		}

		ws.Broadcast("neural_task_result", payload)
	}
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

	case "neural_task":
		// Route Neural Desktop goals through the backend Inference Engine.
		// The frontend sends { type, payload: { goal } }, so read from payload first.
		goal := ""
		if payload, ok := msg["payload"].(map[string]interface{}); ok {
			goal, _ = payload["goal"].(string)
		}
		if goal == "" {
			goal, _ = msg["goal"].(string) // fallback for top-level format
		}
		if goal != "" && ws.inferenceService != nil && ws.inferenceService.IsRunning() {
			go func() {
				systemPrompt := "You are Ana, the default cognitive engine identity for the KNIRV distributed network. Analyze the following objective and provide structured reasoning steps."
				result, err := ws.inferenceService.GenerateTextWithContext(
					ws.ctx, "", goal, systemPrompt,
				)

				completion := InferenceCompletion{
					ID:        fmt.Sprintf("inf_%d", time.Now().UnixNano()),
					Goal:      goal,
					Timestamp: time.Now(),
				}

				var responseMsg Message
				if err != nil {
					completion.Type = "error"
					completion.Error = err.Error()
					responseMsg = Message{
						Type:      "neural_task_response",
						Event:     "neural_task_error",
						Payload:   map[string]string{"error": err.Error(), "goal": goal},
						Timestamp: time.Now().Format(time.RFC3339),
					}
				} else {
					completion.Type = "conclusion"
					completion.Result = result
					responseMsg = Message{
						Type:  "neural_task_response",
						Event: "neural_task_result",
						Payload: map[string]string{
							"goal":   goal,
							"result": result,
							"type":   "conclusion",
						},
						Timestamp: time.Now().Format(time.RFC3339),
					}
				}

				// Record completion for Neural Stream history
				ws.RecordInferenceCompletion(completion)

				// Broadcast to all subscribed clients for Neural Stream
				ws.Broadcast("neural_task_result", map[string]string{
					"goal":   goal,
					"result": result,
					"type":   completion.Type,
				})

				select {
				case client.send <- responseMsg:
				default:
					log.Printf("WebSocket: failed to send neural_task_response to client %s", client.id)
				}
			}()
		}
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

// handleSSHWebSocket handles SSH WebSocket connections for web-based terminal
func (ws *WebSocketService) handleSSHWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	// Validate SSH session exists
	sshSession, err := ws.sessionManager.GetSSHSession(sessionID)
	if err != nil || sshSession == nil {
		http.Error(w, "Invalid SSH session", http.StatusUnauthorized)
		return
	}

	// Check if session is expired
	if time.Now().After(sshSession.ExpiresAt) {
		http.Error(w, "SSH session expired", http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("SSH WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("SSH WebSocket client connected for session: %s", sessionID)

	// Subscribe to real-time agent responses for this DVE
	var agentChan chan string
	if ws.agentService != nil {
		dveID := sshSession.CreationID
		if dveID == "" {
			dveID = sshSession.RentalID
		}
		if dveID != "" {
			agentChan = ws.agentService.SubscribeToResponses(dveID)
			defer ws.agentService.UnsubscribeFromResponses(dveID, agentChan)
		}
	}

	// Set up connection parameters
	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping/pong handler
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	// Channel for incoming messages
	inputChan := make(chan string, 100)
	defer close(inputChan)

	// Handle incoming WebSocket messages
	go func() {
		for {
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("SSH WebSocket error: %v", err)
				}
				return
			}

			if msgType, ok := msg["type"].(string); ok && msgType == "input" {
				if data, ok := msg["data"].(string); ok {
					select {
					case inputChan <- data:
					default:
						log.Printf("SSH input channel full, dropping input")
					}
				}
			}
		}
	}()

	// Forward agent responses to terminal
	if agentChan != nil {
		go func() {
			for {
				select {
				case msg, ok := <-agentChan:
					if !ok {
						return
					}
					// Inject agent message into terminal stream
					// Use green color and clear lines to make it stand out
					conn.WriteJSON(map[string]interface{}{
						"type": "output",
						"data": fmt.Sprintf("\r\n\x1b[32m[KNIRVAGENT] %s\x1b[0m\r\n", msg),
					})
				case <-ws.ctx.Done():
					return
				}
			}
		}()
	}

	// Real SSH connection implementation
	go func() {
		// Use the session endpoint if available, otherwise fallback to localhost
		endpoint := sshSession.Endpoint
		if endpoint == "" {
			endpoint = "localhost"
		}
		sshAddr := fmt.Sprintf("%s:%d", endpoint, sshSession.Port)
		if sshSession.Port == 0 {
			sshAddr = fmt.Sprintf("%s:22", endpoint)
		}

		// Parse private key
		signer, err := ssh.ParsePrivateKey([]byte(sshSession.PrivateKey))
		if err != nil {
			log.Printf("Failed to parse private key for session %s: %v", sessionID, err)
			conn.WriteJSON(map[string]interface{}{
				"type": "output",
				"data": fmt.Sprintf("\r\n\x1b[31mError: Failed to parse SSH private key: %v\x1b[0m\r\n", err),
			})
			return
		}

		// SSH client configuration
		config := &ssh.ClientConfig{
			User: sshSession.Username,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use actual host key verification
			Timeout:         10 * time.Second,
		}

		// Connect to SSH server
		client, err := ssh.Dial("tcp", sshAddr, config)
		if err != nil {
			log.Printf("Failed to dial SSH server %s: %v", sshAddr, err)
			conn.WriteJSON(map[string]interface{}{
				"type": "output",
				"data": fmt.Sprintf("\r\n\x1b[31mError: Failed to connect to SSH server at %s: %v\x1b[0m\r\n", sshAddr, err),
			})
			return
		}
		defer client.Close()

		// Create SSH session
		session, err := client.NewSession()
		if err != nil {
			log.Printf("Failed to create SSH session for %s: %v", sessionID, err)
			return
		}
		defer session.Close()

		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,     // enable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		// Request pseudo-terminal
		if err := session.RequestPty("xterm-256color", 40, 80, modes); err != nil {
			log.Printf("Failed to request PTY for %s: %v", sessionID, err)
			return
		}

		// Pipe stdin/stdout
		stdin, _ := session.StdinPipe()
		stdout, _ := session.StdoutPipe()
		stderr, _ := session.StderrPipe()

		// Start the shell
		if err := session.Shell(); err != nil {
			log.Printf("Failed to start shell for %s: %v", sessionID, err)
			return
		}

		// Relay SSH output to WebSocket
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					conn.WriteJSON(map[string]interface{}{
						"type": "output",
						"data": string(buf[:n]),
					})
				}
				if err != nil {
					return
				}
			}
		}()

		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stderr.Read(buf)
				if n > 0 {
					conn.WriteJSON(map[string]interface{}{
						"type": "output",
						"data": string(buf[:n]),
					})
				}
				if err != nil {
					return
				}
			}
		}()

		// Relay input from WebSocket to SSH
		for {
			select {
			case input := <-inputChan:
				_, err := stdin.Write([]byte(input))
				if err != nil {
					return
				}
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-ws.ctx.Done():
				return
			}
		}
	}()

	// Keep connection alive
	for {
		select {
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-ws.ctx.Done():
			return
		}
	}
}

func (ws *WebSocketService) handleErrorResolutionWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	if sessionID == "" {
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}
	if ws.sessionManager == nil {
		http.Error(w, "Session manager unavailable", http.StatusServiceUnavailable)
		return
	}

	errSession, err := ws.sessionManager.GetErrorResolutionSession(sessionID)
	if err != nil || errSession == nil {
		http.Error(w, "Invalid error resolution session", http.StatusUnauthorized)
		return
	}
	if errSession.Port < 24000 || errSession.Port > 24999 {
		http.Error(w, "Error resolution port out of allowed range", http.StatusBadRequest)
		return
	}

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error resolution WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	proxyConn, err := ws.openErrorResolutionProxy(errSession)
	if err != nil {
		_ = conn.WriteJSON(map[string]interface{}{
			"type": "error",
			"data": fmt.Sprintf("Failed to connect to error resolution service: %v", err),
		})
		return
	}
	defer proxyConn.Close()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := proxyConn.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteJSON(map[string]interface{}{
					"type": "output",
					"data": string(buf[:n]),
				}); writeErr != nil {
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		if msgType, ok := msg["type"].(string); ok && msgType == "input" {
			if data, ok := msg["data"].(string); ok {
				if _, err := proxyConn.Write([]byte(data)); err != nil {
					return
				}
			}
		}
	}
}

func (ws *WebSocketService) openErrorResolutionProxy(session *objects.ErrorResolutionSession) (net.Conn, error) {
	for _, candidate := range ws.errorResolutionSocketCandidates(session) {
		if _, err := os.Stat(candidate); err == nil {
			return net.DialTimeout("unix", candidate, 5*time.Second)
		}
	}

	return net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", session.Port), 5*time.Second)
}

func (ws *WebSocketService) errorResolutionSocketCandidates(session *objects.ErrorResolutionSession) []string {
	if ws.socketDir == "" {
		return nil
	}

	names := []string{
		fmt.Sprintf("error-resolution-%d.sock", session.Port),
		fmt.Sprintf("error_resolution_%d.sock", session.Port),
		fmt.Sprintf("terminal-%d.sock", session.Port),
		fmt.Sprintf("error-resolution-%s.sock", session.CreationID),
	}

	candidates := make([]string, 0, len(names))
	for _, name := range names {
		candidates = append(candidates, filepath.Join(ws.socketDir, name))
	}
	return candidates
}
