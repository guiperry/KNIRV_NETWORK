package core

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVCLI/config"
	"github.com/sirupsen/logrus"
)

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	conn     *websocket.Conn
	url      string
	handlers map[string]EventHandler
	logger   *logrus.Logger
	stopCh   chan struct{}
	mu       sync.RWMutex
}

// WebSocketManager manages WebSocket connections to KNIRV services
type WebSocketManager struct {
	connections map[string]*WebSocketConnection
	config      *config.RealtimeConfig
	logger      *logrus.Logger
	mu          sync.RWMutex
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp"`
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(config *config.RealtimeConfig, logger *logrus.Logger) *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[string]*WebSocketConnection),
		config:      config,
		logger:      logger,
	}
}

// Connect establishes a WebSocket connection to a service
func (wsm *WebSocketManager) Connect(ctx context.Context, serviceName, wsURL string, handlers map[string]EventHandler) error {
	if !wsm.config.WebSocket.Enabled {
		return fmt.Errorf("WebSocket is disabled in configuration")
	}

	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	// Check if already connected
	if _, exists := wsm.connections[serviceName]; exists {
		return fmt.Errorf("already connected to %s", serviceName)
	}

	// Parse WebSocket URL
	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	// Create WebSocket connection
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	wsConn := &WebSocketConnection{
		conn:     conn,
		url:      wsURL,
		handlers: handlers,
		logger:   wsm.logger,
		stopCh:   make(chan struct{}),
	}

	wsm.connections[serviceName] = wsConn

	// Start message handling goroutine
	go wsConn.handleMessages()

	wsm.logger.Infof("WebSocket connected to %s at %s", serviceName, wsURL)
	return nil
}

// Disconnect closes a WebSocket connection
func (wsm *WebSocketManager) Disconnect(serviceName string) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	conn, exists := wsm.connections[serviceName]
	if !exists {
		return fmt.Errorf("not connected to %s", serviceName)
	}

	// Stop message handling
	close(conn.stopCh)

	// Close WebSocket connection
	err := conn.conn.Close()
	if err != nil {
		wsm.logger.Errorf("Error closing WebSocket connection to %s: %v", serviceName, err)
	}

	// Remove from connections map
	delete(wsm.connections, serviceName)

	wsm.logger.Infof("WebSocket disconnected from %s", serviceName)
	return err
}

// DisconnectAll closes all WebSocket connections
func (wsm *WebSocketManager) DisconnectAll() error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	var lastErr error
	for serviceName := range wsm.connections {
		if err := wsm.disconnect(serviceName); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// disconnect is the internal disconnect method (assumes lock is held)
func (wsm *WebSocketManager) disconnect(serviceName string) error {
	conn, exists := wsm.connections[serviceName]
	if !exists {
		return nil
	}

	close(conn.stopCh)
	err := conn.conn.Close()
	delete(wsm.connections, serviceName)

	return err
}

// SendMessage sends a message through a WebSocket connection
func (wsm *WebSocketManager) SendMessage(serviceName string, message *WebSocketMessage) error {
	wsm.mu.RLock()
	conn, exists := wsm.connections[serviceName]
	wsm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("not connected to %s", serviceName)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	err := conn.conn.WriteJSON(message)
	if err != nil {
		wsm.logger.Errorf("Failed to send WebSocket message to %s: %v", serviceName, err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// GetConnections returns the list of active connections
func (wsm *WebSocketManager) GetConnections() []string {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	connections := make([]string, 0, len(wsm.connections))
	for serviceName := range wsm.connections {
		connections = append(connections, serviceName)
	}

	return connections
}

// IsConnected checks if connected to a service
func (wsm *WebSocketManager) IsConnected(serviceName string) bool {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	_, exists := wsm.connections[serviceName]
	return exists
}

// handleMessages handles incoming WebSocket messages
func (wsc *WebSocketConnection) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			wsc.logger.Errorf("WebSocket message handler panic: %v", r)
		}
	}()

	for {
		select {
		case <-wsc.stopCh:
			return
		default:
			var message WebSocketMessage
			err := wsc.conn.ReadJSON(&message)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					wsc.logger.Errorf("WebSocket read error: %v", err)
				}
				return
			}

			// Handle the message
			wsc.handleMessage(&message)
		}
	}
}

// handleMessage processes a single WebSocket message
func (wsc *WebSocketConnection) handleMessage(message *WebSocketMessage) {
	wsc.mu.RLock()
	handler, exists := wsc.handlers[message.Type]
	wsc.mu.RUnlock()

	if !exists {
		wsc.logger.Debugf("No handler for WebSocket message type: %s", message.Type)
		return
	}

	// Convert WebSocket message to Event
	event := &Event{
		Type:      message.Type,
		Source:    message.Source,
		Data:      message.Data,
		Timestamp: message.Timestamp,
	}

	// Handle the event
	go func() {
		if err := handler.HandleEvent(event); err != nil {
			wsc.logger.Errorf("Error handling WebSocket event %s: %v", event.Type, err)
		}
	}()
}

// AddHandler adds an event handler for a specific message type
func (wsm *WebSocketManager) AddHandler(serviceName, messageType string, handler EventHandler) error {
	wsm.mu.RLock()
	conn, exists := wsm.connections[serviceName]
	wsm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("not connected to %s", serviceName)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.handlers[messageType] = handler
	wsm.logger.Debugf("Added WebSocket handler for %s message type %s", serviceName, messageType)

	return nil
}

// RemoveHandler removes an event handler for a specific message type
func (wsm *WebSocketManager) RemoveHandler(serviceName, messageType string) error {
	wsm.mu.RLock()
	conn, exists := wsm.connections[serviceName]
	wsm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("not connected to %s", serviceName)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	delete(conn.handlers, messageType)
	wsm.logger.Debugf("Removed WebSocket handler for %s message type %s", serviceName, messageType)

	return nil
}
