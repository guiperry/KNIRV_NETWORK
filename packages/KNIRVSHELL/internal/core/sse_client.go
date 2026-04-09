package core

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/config"
	"github.com/sirupsen/logrus"
)

// SSEConnection represents a Server-Sent Events connection
type SSEConnection struct {
	url      string
	client   *http.Client
	response *http.Response
	scanner  *bufio.Scanner
	handlers map[string]EventHandler
	logger   *logrus.Logger
	stopCh   chan struct{}
	mu       sync.RWMutex
}

// SSEClient manages Server-Sent Events connections
type SSEClient struct {
	connections map[string]*SSEConnection
	config      *config.RealtimeConfig
	logger      *logrus.Logger
	mu          sync.RWMutex
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	ID    string `json:"id"`
	Event string `json:"event"`
	Data  string `json:"data"`
	Retry int    `json:"retry"`
}

// NewSSEClient creates a new SSE client
func NewSSEClient(config *config.RealtimeConfig, logger *logrus.Logger) *SSEClient {
	return &SSEClient{
		connections: make(map[string]*SSEConnection),
		config:      config,
		logger:      logger,
	}
}

// Connect establishes an SSE connection to a service
func (sc *SSEClient) Connect(ctx context.Context, serviceName, sseURL string, handlers map[string]EventHandler) error {
	if !sc.config.SSE.Enabled {
		return fmt.Errorf("SSE is disabled in configuration")
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Check if already connected
	if _, exists := sc.connections[serviceName]; exists {
		return fmt.Errorf("already connected to %s", serviceName)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: sc.config.SSE.Timeout,
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", sseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("SSE connection failed with status: %d", resp.StatusCode)
	}

	// Create SSE connection
	sseConn := &SSEConnection{
		url:      sseURL,
		client:   client,
		response: resp,
		scanner:  bufio.NewScanner(resp.Body),
		handlers: handlers,
		logger:   sc.logger,
		stopCh:   make(chan struct{}),
	}

	// Set scanner buffer size
	buf := make([]byte, 0, sc.config.SSE.BufferSize)
	sseConn.scanner.Buffer(buf, sc.config.SSE.BufferSize)

	sc.connections[serviceName] = sseConn

	// Start event handling goroutine
	go sseConn.handleEvents()

	sc.logger.Infof("SSE connected to %s at %s", serviceName, sseURL)
	return nil
}

// Disconnect closes an SSE connection
func (sc *SSEClient) Disconnect(serviceName string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	conn, exists := sc.connections[serviceName]
	if !exists {
		return fmt.Errorf("not connected to %s", serviceName)
	}

	// Stop event handling
	close(conn.stopCh)

	// Close response body
	err := conn.response.Body.Close()
	if err != nil {
		sc.logger.Errorf("Error closing SSE connection to %s: %v", serviceName, err)
	}

	// Remove from connections map
	delete(sc.connections, serviceName)

	sc.logger.Infof("SSE disconnected from %s", serviceName)
	return err
}

// DisconnectAll closes all SSE connections
func (sc *SSEClient) DisconnectAll() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	var lastErr error
	for serviceName := range sc.connections {
		if err := sc.disconnect(serviceName); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// disconnect is the internal disconnect method (assumes lock is held)
func (sc *SSEClient) disconnect(serviceName string) error {
	conn, exists := sc.connections[serviceName]
	if !exists {
		return nil
	}

	close(conn.stopCh)
	err := conn.response.Body.Close()
	delete(sc.connections, serviceName)

	return err
}

// GetConnections returns the list of active connections
func (sc *SSEClient) GetConnections() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	connections := make([]string, 0, len(sc.connections))
	for serviceName := range sc.connections {
		connections = append(connections, serviceName)
	}

	return connections
}

// IsConnected checks if connected to a service
func (sc *SSEClient) IsConnected(serviceName string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	_, exists := sc.connections[serviceName]
	return exists
}

// handleEvents handles incoming SSE events
func (sseConn *SSEConnection) handleEvents() {
	defer func() {
		if r := recover(); r != nil {
			sseConn.logger.Errorf("SSE event handler panic: %v", r)
		}
	}()

	var currentEvent SSEEvent

	for {
		select {
		case <-sseConn.stopCh:
			return
		default:
			if !sseConn.scanner.Scan() {
				if err := sseConn.scanner.Err(); err != nil {
					sseConn.logger.Errorf("SSE scanner error: %v", err)
				}
				return
			}

			line := sseConn.scanner.Text()

			// Parse SSE line
			if line == "" {
				// Empty line indicates end of event
				if currentEvent.Event != "" || currentEvent.Data != "" {
					sseConn.handleEvent(&currentEvent)
					currentEvent = SSEEvent{} // Reset for next event
				}
				continue
			}

			// Parse field and value
			if strings.HasPrefix(line, ":") {
				// Comment line, ignore
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			field := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch field {
			case "id":
				currentEvent.ID = value
			case "event":
				currentEvent.Event = value
			case "data":
				if currentEvent.Data != "" {
					currentEvent.Data += "\n"
				}
				currentEvent.Data += value
			case "retry":
				// Parse retry value if needed
				// For now, we'll ignore it
			}
		}
	}
}

// handleEvent processes a single SSE event
func (sseConn *SSEConnection) handleEvent(sseEvent *SSEEvent) {
	eventType := sseEvent.Event
	if eventType == "" {
		eventType = "message" // Default event type
	}

	sseConn.mu.RLock()
	handler, exists := sseConn.handlers[eventType]
	sseConn.mu.RUnlock()

	if !exists {
		sseConn.logger.Debugf("No handler for SSE event type: %s", eventType)
		return
	}

	// Convert SSE event to Event
	event := &Event{
		Type:      eventType,
		Source:    "sse",
		Data:      map[string]interface{}{"data": sseEvent.Data, "id": sseEvent.ID},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Handle the event
	go func() {
		if err := handler.HandleEvent(event); err != nil {
			sseConn.logger.Errorf("Error handling SSE event %s: %v", event.Type, err)
		}
	}()
}

// AddHandler adds an event handler for a specific event type
func (sc *SSEClient) AddHandler(serviceName, eventType string, handler EventHandler) error {
	sc.mu.RLock()
	conn, exists := sc.connections[serviceName]
	sc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("not connected to %s", serviceName)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.handlers[eventType] = handler
	sc.logger.Debugf("Added SSE handler for %s event type %s", serviceName, eventType)

	return nil
}

// RemoveHandler removes an event handler for a specific event type
func (sc *SSEClient) RemoveHandler(serviceName, eventType string) error {
	sc.mu.RLock()
	conn, exists := sc.connections[serviceName]
	sc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("not connected to %s", serviceName)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	delete(conn.handlers, eventType)
	sc.logger.Debugf("Removed SSE handler for %s event type %s", serviceName, eventType)

	return nil
}
