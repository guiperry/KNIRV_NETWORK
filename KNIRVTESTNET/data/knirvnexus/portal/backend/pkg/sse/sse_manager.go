package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SSEManager manages Server-Sent Events connections
type SSEManager struct {
	clients map[string]*SSEClient
	rooms   map[string][]string
	mutex   sync.RWMutex
}

// SSEClient represents a connected SSE client
type SSEClient struct {
	ID       string
	UserID   string
	Channel  chan SSEMessage
	Request  *http.Request
	Writer   http.ResponseWriter
	LastPing time.Time
	Rooms    []string
}

// SSEMessage represents a message sent via SSE
type SSEMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	ID    string      `json:"id,omitempty"`
}

// NewSSEManager creates a new SSE manager
func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients: make(map[string]*SSEClient),
		rooms:   make(map[string][]string),
	}
}

// AddClient adds a new SSE client
func (sm *SSEManager) AddClient(client *SSEClient) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	sm.clients[client.ID] = client
}

// RemoveClient removes an SSE client
func (sm *SSEManager) RemoveClient(clientID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	if client, exists := sm.clients[clientID]; exists {
		// Remove from all rooms
		for _, room := range client.Rooms {
			sm.removeClientFromRoom(clientID, room)
		}
		
		// Close channel
		close(client.Channel)
		
		// Remove from clients map
		delete(sm.clients, clientID)
	}
}

// SendToClient sends a message to a specific client
func (sm *SSEManager) SendToClient(clientID string, message SSEMessage) error {
	sm.mutex.RLock()
	client, exists := sm.clients[clientID]
	sm.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}
	
	select {
	case client.Channel <- message:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message to client %s", clientID)
	}
}

// BroadcastToRoom sends a message to all clients in a room
func (sm *SSEManager) BroadcastToRoom(room string, message SSEMessage) {
	sm.mutex.RLock()
	clientIDs, exists := sm.rooms[room]
	if !exists {
		sm.mutex.RUnlock()
		return
	}
	
	// Create a copy of client IDs to avoid holding the lock too long
	clientIDsCopy := make([]string, len(clientIDs))
	copy(clientIDsCopy, clientIDs)
	sm.mutex.RUnlock()
	
	for _, clientID := range clientIDsCopy {
		sm.SendToClient(clientID, message)
	}
}

// BroadcastToAll sends a message to all connected clients
func (sm *SSEManager) BroadcastToAll(message SSEMessage) {
	sm.mutex.RLock()
	clientIDs := make([]string, 0, len(sm.clients))
	for clientID := range sm.clients {
		clientIDs = append(clientIDs, clientID)
	}
	sm.mutex.RUnlock()
	
	for _, clientID := range clientIDs {
		sm.SendToClient(clientID, message)
	}
}

// JoinRoom adds a client to a room
func (sm *SSEManager) JoinRoom(clientID, room string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	client, exists := sm.clients[clientID]
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}
	
	// Add client to room
	if _, exists := sm.rooms[room]; !exists {
		sm.rooms[room] = make([]string, 0)
	}
	
	// Check if client is already in room
	for _, id := range sm.rooms[room] {
		if id == clientID {
			return nil // Already in room
		}
	}
	
	sm.rooms[room] = append(sm.rooms[room], clientID)
	client.Rooms = append(client.Rooms, room)
	
	return nil
}

// LeaveRoom removes a client from a room
func (sm *SSEManager) LeaveRoom(clientID, room string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	client, exists := sm.clients[clientID]
	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}
	
	// Remove from room
	sm.removeClientFromRoom(clientID, room)
	
	// Remove room from client's rooms list
	for i, r := range client.Rooms {
		if r == room {
			client.Rooms = append(client.Rooms[:i], client.Rooms[i+1:]...)
			break
		}
	}
	
	return nil
}

// removeClientFromRoom removes a client from a room (internal helper)
func (sm *SSEManager) removeClientFromRoom(clientID, room string) {
	if clientIDs, exists := sm.rooms[room]; exists {
		for i, id := range clientIDs {
			if id == clientID {
				sm.rooms[room] = append(clientIDs[:i], clientIDs[i+1:]...)
				break
			}
		}
		
		// Remove room if empty
		if len(sm.rooms[room]) == 0 {
			delete(sm.rooms, room)
		}
	}
}

// GetClientCount returns the number of connected clients
func (sm *SSEManager) GetClientCount() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.clients)
}

// GetRoomCount returns the number of active rooms
func (sm *SSEManager) GetRoomCount() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.rooms)
}

// GetClientsInRoom returns the client IDs in a specific room
func (sm *SSEManager) GetClientsInRoom(room string) []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	if clientIDs, exists := sm.rooms[room]; exists {
		result := make([]string, len(clientIDs))
		copy(result, clientIDs)
		return result
	}
	
	return []string{}
}

// CleanupStaleClients removes clients that haven't pinged recently
func (sm *SSEManager) CleanupStaleClients() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	staleThreshold := time.Now().Add(-5 * time.Minute)
	var staleClients []string
	
	for clientID, client := range sm.clients {
		if client.LastPing.Before(staleThreshold) {
			staleClients = append(staleClients, clientID)
		}
	}
	
	// Remove stale clients
	for _, clientID := range staleClients {
		if client, exists := sm.clients[clientID]; exists {
			// Remove from all rooms
			for _, room := range client.Rooms {
				sm.removeClientFromRoom(clientID, room)
			}
			
			// Close channel
			close(client.Channel)
			
			// Remove from clients map
			delete(sm.clients, clientID)
		}
	}
}

// Close closes all connections and cleans up
func (sm *SSEManager) Close() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	// Close all client channels
	for _, client := range sm.clients {
		close(client.Channel)
	}
	
	// Clear all data structures
	sm.clients = make(map[string]*SSEClient)
	sm.rooms = make(map[string][]string)
}

// CreateSSEHandler creates an HTTP handler for SSE connections
func (sm *SSEManager) CreateSSEHandler(userID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

		clientID := uuid.New().String()
		client := &SSEClient{
			ID:       clientID,
			UserID:   userID,
			Channel:  make(chan SSEMessage, 100),
			Request:  r,
			Writer:   w,
			LastPing: time.Now(),
			Rooms:    make([]string, 0),
		}

		sm.AddClient(client)
		defer sm.RemoveClient(clientID)

		// Send initial connection message
		sm.SendToClient(clientID, SSEMessage{
			Event: "connected",
			Data:  map[string]string{"client_id": clientID},
			ID:    fmt.Sprintf("%d", time.Now().Unix()),
		})

		// Handle client connection
		sm.handleSSEClient(client)
	}
}

// handleSSEClient handles an individual SSE client connection
func (sm *SSEManager) handleSSEClient(client *SSEClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message := <-client.Channel:
			data, _ := json.Marshal(message.Data)
			eventData := fmt.Sprintf("event: %s\ndata: %s\nid: %s\n\n",
				message.Event,
				string(data),
				message.ID)

			if _, err := client.Writer.Write([]byte(eventData)); err != nil {
				return
			}

			if flusher, ok := client.Writer.(http.Flusher); ok {
				flusher.Flush()
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			ping := fmt.Sprintf("event: ping\ndata: %d\n\n", time.Now().Unix())
			if _, err := client.Writer.Write([]byte(ping)); err != nil {
				return
			}

			if flusher, ok := client.Writer.(http.Flusher); ok {
				flusher.Flush()
			}

			client.LastPing = time.Now()

		case <-client.Request.Context().Done():
			return
		}
	}
}

// BroadcastUpdate is a convenience method for broadcasting updates
func (sm *SSEManager) BroadcastUpdate(room string, update interface{}) {
	sm.BroadcastToRoom(room, SSEMessage{
		Event: "update",
		Data:  update,
		ID:    fmt.Sprintf("%d", time.Now().Unix()),
	})
}
