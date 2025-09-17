package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"KNIRVORACLE/config"
	"KNIRVORACLE/dataengine"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// GoReverseProxy holds the configuration and proxy instances.
type GoReverseProxy struct {
	config          *config.ReverseProxyConfig
	frontendProxy   *httputil.ReverseProxy
	backendApiProxy *httputil.ReverseProxy
	frontendTarget  *url.URL
	backendTarget   *url.URL

	// DataEngine integration
	dataEngine *dataengine.DataEngine
	router     *mux.Router
	server     *http.Server

	// WebSocket integration
	wsUpgrader     websocket.Upgrader
	wsClients      map[*websocket.Conn]bool
	wsClientsMutex sync.RWMutex
	wsBroadcast    chan interface{}
}

// NewGoReverseProxy creates and configures a new GoReverseProxy.
// frontendTargetURL: e.g., "http://127.0.0.1:3000" (for Next.js)
// backendTargetURL: e.g., "http://127.0.0.1:5000" (for Go API)
func NewGoReverseProxy(cfg *config.ReverseProxyConfig, frontendTargetURL, backendTargetURL string, dataEngine *dataengine.DataEngine) (*GoReverseProxy, error) {
	ftURL, err := url.Parse(frontendTargetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid frontend target URL: %w", err)
	}

	btURL, err := url.Parse(backendTargetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid backend target URL: %w", err)
	}

	proxy := &GoReverseProxy{
		config:          cfg,
		frontendProxy:   httputil.NewSingleHostReverseProxy(ftURL),
		backendApiProxy: httputil.NewSingleHostReverseProxy(btURL),
		frontendTarget:  ftURL,
		backendTarget:   btURL,
		dataEngine:      dataEngine,
		router:          mux.NewRouter(),
		wsClients:       make(map[*websocket.Conn]bool),
		wsBroadcast:     make(chan interface{}, 100),
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for WebSocket connections
			},
		},
	}

	return proxy, nil
}

// ServeHTTP is the entry point for incoming requests.
func (p *GoReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("ReverseProxy: Received request for %s %s", r.Method, r.URL.Path)

	// Handle WebSocket connections
	if r.URL.Path == "/ws" && p.config.EmbedDataEngine && p.dataEngine != nil {
		p.handleWebSocket(w, r)
		return
	}

	// Handle DataEngine API routes
	if p.config.EmbedDataEngine && p.dataEngine != nil {
		// Check for DataEngine API paths
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			p.router.ServeHTTP(w, r)
			return
		}

		// Handle specific DataEngine endpoints
		switch r.URL.Path {
		case "/metrics":
			p.handleMetrics(w, r)
			return
		case "/alerts":
			p.handleAlerts(w, r)
			return
		case "/events":
			p.handleEvents(w, r)
			return
		case "/health":
			p.handleHealth(w, r)
			return
		}
	}

	// Route to backend API if path starts with /api/
	if strings.HasPrefix(r.URL.Path, "/api/") {
		log.Printf("ReverseProxy: Routing to backend API: %s", r.URL.Path)
		// To ensure the backend receives the original Host header if it needs it
		r.Host = p.backendTarget.Host
		p.backendApiProxy.ServeHTTP(w, r)
		return
	}

	// Otherwise, route to frontend
	log.Printf("ReverseProxy: Routing to frontend: %s", r.URL.Path)
	// To ensure the frontend receives the original Host header
	r.Host = p.frontendTarget.Host
	p.frontendProxy.ServeHTTP(w, r)
}

// Start runs the reverse proxy server.
func (p *GoReverseProxy) Start() error {
	if !p.config.Enabled {
		log.Println("ReverseProxy: Not starting, as it's disabled in config.")
		return nil
	}

	// Set up DataEngine routes if enabled
	if p.config.EmbedDataEngine && p.dataEngine != nil {
		log.Println("ReverseProxy: Setting up embedded DataEngine routes")
		p.setupDataEngineRoutes()

		// Start WebSocket broadcast handler
		go p.handleBroadcasts()

		// Subscribe to DataEngine events
		p.subscribeToDataEngineEvents()
	}

	log.Printf("ReverseProxy: Starting on %s, proxying frontend to %s, backend API to %s",
		p.config.ListenAddr, p.frontendTarget.String(), p.backendTarget.String())

	p.server = &http.Server{
		Addr:    p.config.ListenAddr,
		Handler: p,
	}

	// For HTTPS, you would use ListenAndServeTLS:
	// return p.server.ListenAndServeTLS(p.config.CertFile, p.config.KeyFile)
	err := p.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Printf("ReverseProxy: Server error: %v", err)
		return err
	}
	log.Println("ReverseProxy: Server stopped.")
	return nil
}

// Stop gracefully stops the reverse proxy server
func (p *GoReverseProxy) Stop() error {
	if p.server != nil {
		// Create a context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Close all WebSocket connections
		p.wsClientsMutex.Lock()
		for client := range p.wsClients {
			client.Close()
		}
		p.wsClients = make(map[*websocket.Conn]bool)
		p.wsClientsMutex.Unlock()

		// Shutdown the server
		return p.server.Shutdown(ctx)
	}
	return nil
}

// setupDataEngineRoutes sets up the API routes for the DataEngine
func (p *GoReverseProxy) setupDataEngineRoutes() {
	// API version prefix
	api := p.router.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", p.handleAPIHealth).Methods("GET")

	// Metrics
	api.HandleFunc("/metrics", p.handleGetMetrics).Methods("GET")

	// Alerts
	api.HandleFunc("/alerts", p.handleGetAlerts).Methods("GET")
	api.HandleFunc("/alerts/{id}", p.handleResolveAlert).Methods("PUT")

	// Events
	api.HandleFunc("/events", p.handleGetEvents).Methods("GET")
	api.HandleFunc("/events/search", p.handleSearchEvents).Methods("GET")
	api.HandleFunc("/events/types", p.handleGetEventTypes).Methods("GET")

	// Windows
	api.HandleFunc("/windows", p.handleGetWindows).Methods("GET")
	api.HandleFunc("/windows/range", p.handleGetWindowsInRange).Methods("GET")

	// Analytics
	api.HandleFunc("/analytics/users", p.handleGetActiveUsers).Methods("GET")
	api.HandleFunc("/analytics/rates", p.handleGetEventRates).Methods("GET")

	// Add middleware
	p.router.Use(p.loggingMiddleware)
	p.router.Use(p.corsMiddleware)
}

// loggingMiddleware logs API requests
func (p *GoReverseProxy) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the request
		log.Printf(
			"[%s] %s %s %s",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	})
}

// corsMiddleware adds CORS headers
func (p *GoReverseProxy) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// handleAPIHealth handles health check requests for the API
func (p *GoReverseProxy) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	// Create health status
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "rest-api",
	}

	// Add data engine status
	if p.dataEngine != nil {
		health["data_engine"] = map[string]interface{}{
			"running": p.dataEngine.IsRunning(),
		}

		// Add Kafka status
		if p.dataEngine.GetProducer() != nil {
			health["kafka"] = map[string]interface{}{
				"connected": p.dataEngine.GetProducer().IsConnected(),
			}
		}

		// Add ChromaDB status
		if p.dataEngine.GetChromaDB() != nil {
			health["chromadb"] = map[string]interface{}{
				"connected": p.dataEngine.GetChromaDB().IsConnected(),
			}
		}
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(health)
}

// handleGetMetrics handles requests for metrics
func (p *GoReverseProxy) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Get metrics
	metrics := p.dataEngine.GetMetrics()
	if metrics == nil {
		http.Error(w, "No metrics available", http.StatusNotFound)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(metrics)
}

// handleGetAlerts handles requests for alerts
func (p *GoReverseProxy) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetAlerting() == nil {
		http.Error(w, "Alerting system not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	activeOnly := r.URL.Query().Get("active") == "true"

	var alerts []dataengine.Alert
	if activeOnly {
		alerts = p.dataEngine.GetAlerting().GetActiveAlerts()
	} else {
		alerts = p.dataEngine.GetAlerting().GetAlerts()
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(alerts)
}

// handleResolveAlert handles requests to resolve an alert
func (p *GoReverseProxy) handleResolveAlert(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetAlerting() == nil {
		http.Error(w, "Alerting system not available", http.StatusServiceUnavailable)
		return
	}

	// Get alert ID from URL
	vars := mux.Vars(r)
	alertID := vars["id"]

	// Resolve alert
	resolved := p.dataEngine.GetAlerting().ResolveAlert(alertID)

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       alertID,
		"resolved": resolved,
	})
}

// handleGetEvents handles requests for events
func (p *GoReverseProxy) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetChromaDB() == nil || !p.dataEngine.GetChromaDB().IsConnected() {
		http.Error(w, "ChromaDB not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	eventType := r.URL.Query().Get("type")

	// Set default limit
	limit := 10
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}
	}

	var docs []dataengine.ChromaDocument
	var err error

	// Query events
	if eventType != "" {
		// Filter by event type
		docs, err = p.dataEngine.GetChromaDB().GetEventsByType(r.Context(), dataengine.EventType(eventType), limit)
	} else {
		// Get recent events
		docs, err = p.dataEngine.GetChromaDB().GetRecentEvents(r.Context(), limit)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query events: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(docs)
}

// handleSearchEvents handles requests to search events
func (p *GoReverseProxy) handleSearchEvents(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetChromaDB() == nil || !p.dataEngine.GetChromaDB().IsConnected() {
		http.Error(w, "ChromaDB not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")

	// Set default limit
	limit := 10
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}
	}

	// Search events
	docs, err := p.dataEngine.GetChromaDB().QueryEvents(r.Context(), query, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search events: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(docs)
}

// handleGetEventTypes handles requests for event types
func (p *GoReverseProxy) handleGetEventTypes(w http.ResponseWriter, r *http.Request) {
	// Define event types
	eventTypes := []string{
		string(dataengine.BlockchainEventType),
		string(dataengine.BlockCreatedEvent),
		string(dataengine.TxSubmittedEvent),
		string(dataengine.TxConfirmedEvent),
		string(dataengine.TxRejectedEvent),
		string(dataengine.NodeConnectedEvent),
		string(dataengine.NodeDisconnectedEvent),
		string(dataengine.SystemEventType),
		string(dataengine.SystemStartedEvent),
		string(dataengine.SystemStoppedEvent),
		string(dataengine.SystemErrorEvent),
		string(dataengine.UserEventType),
		string(dataengine.UserLoginEvent),
		string(dataengine.UserLogoutEvent),
		string(dataengine.UserActionEvent),
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(eventTypes)
}

// handleGetWindows handles requests for windows
func (p *GoReverseProxy) handleGetWindows(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetAggregator() == nil {
		http.Error(w, "Windowed aggregator not available", http.StatusServiceUnavailable)
		return
	}

	// Get windows
	windows := p.dataEngine.GetAggregator().GetWindows()

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(windows)
}

// handleGetWindowsInRange handles requests for windows in a time range
func (p *GoReverseProxy) handleGetWindowsInRange(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetAggregator() == nil {
		http.Error(w, "Windowed aggregator not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	// Parse start time
	var start time.Time
	var err error
	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid start time: %s", err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Default to 1 hour ago
		start = time.Now().Add(-1 * time.Hour)
	}

	// Parse end time
	var end time.Time
	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid end time: %s", err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Default to now
		end = time.Now()
	}

	// Get windows in range
	windows := p.dataEngine.GetAggregator().GetWindowsInRange(start, end)

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(windows)
}

// handleGetActiveUsers handles requests for active users
func (p *GoReverseProxy) handleGetActiveUsers(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetAggregator() == nil {
		http.Error(w, "Windowed aggregator not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	// Parse start time
	var start time.Time
	var err error
	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid start time: %s", err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Default to 1 hour ago
		start = time.Now().Add(-1 * time.Hour)
	}

	// Parse end time
	var end time.Time
	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid end time: %s", err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Default to now
		end = time.Now()
	}

	// Get active users
	activeUsers := p.dataEngine.GetAggregator().GetActiveUsers(start, end)

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"start":        start.Format(time.RFC3339),
		"end":          end.Format(time.RFC3339),
		"active_users": activeUsers,
	})
}

// handleGetEventRates handles requests for event rates
func (p *GoReverseProxy) handleGetEventRates(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetAggregator() == nil {
		http.Error(w, "Windowed aggregator not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	// Parse start time
	var start time.Time
	var err error
	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid start time: %s", err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Default to 1 hour ago
		start = time.Now().Add(-1 * time.Hour)
	}

	// Parse end time
	var end time.Time
	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid end time: %s", err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Default to now
		end = time.Now()
	}

	// Get event rate
	eventRate := p.dataEngine.GetAggregator().GetEventRate(start, end)

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"start":      start.Format(time.RFC3339),
		"end":        end.Format(time.RFC3339),
		"event_rate": eventRate,
	})
}

// handleWebSocket handles WebSocket connections
func (p *GoReverseProxy) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := p.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ReverseProxy: Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Register client
	p.wsClientsMutex.Lock()
	p.wsClients[conn] = true
	p.wsClientsMutex.Unlock()

	// Handle client messages
	go p.handleDataEngineWebSocketClient(conn)
}

// handleDataEngineWebSocketClient handles messages from a WebSocket client
func (p *GoReverseProxy) handleDataEngineWebSocketClient(conn *websocket.Conn) {
	defer func() {
		// Unregister client on disconnect
		p.wsClientsMutex.Lock()
		delete(p.wsClients, conn)
		p.wsClientsMutex.Unlock()
		conn.Close()
	}()

	for {
		// Read message
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("ReverseProxy: WebSocket read error: %v", err)
			break
		}

		// Parse message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("ReverseProxy: Failed to parse WebSocket message: %v", err)
			continue
		}

		// Handle message based on type
		msgType, ok := msg["type"].(string)
		if !ok {
			log.Printf("ReverseProxy: WebSocket message missing 'type' field")
			continue
		}

		switch msgType {
		case "subscribe":
			// Handle subscription request
			topic, ok := msg["topic"].(string)
			if !ok {
				log.Printf("ReverseProxy: Subscribe message missing 'topic' field")
				continue
			}
			log.Printf("ReverseProxy: Client subscribed to topic: %s", topic)
			// In a real implementation, you would store the subscription information
			// For now, we just acknowledge the subscription
			conn.WriteJSON(map[string]interface{}{
				"type":    "subscribed",
				"topic":   topic,
				"success": true,
			})

		case "resolve_alert":
			// Handle alert resolution request
			alertID, ok := msg["alert_id"].(string)
			if !ok {
				log.Printf("ReverseProxy: Resolve alert message missing 'alert_id' field")
				continue
			}
			if p.dataEngine != nil && p.dataEngine.GetAlerting() != nil {
				resolved := p.dataEngine.GetAlerting().ResolveAlert(alertID)
				conn.WriteJSON(map[string]interface{}{
					"type":     "alert_resolved",
					"alert_id": alertID,
					"success":  resolved,
				})
			} else {
				conn.WriteJSON(map[string]interface{}{
					"type":    "error",
					"message": "Alerting system not available",
				})
			}

		default:
			log.Printf("ReverseProxy: Unknown WebSocket message type: %s", msgType)
		}
	}
}

// handleBroadcasts handles broadcasting messages to all WebSocket clients
func (p *GoReverseProxy) handleBroadcasts() {
	for {
		// Get message from broadcast channel
		msg, ok := <-p.wsBroadcast
		if !ok {
			// Channel closed
			return
		}

		// Broadcast to all clients
		p.wsClientsMutex.RLock()
		for client := range p.wsClients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("ReverseProxy: Failed to send WebSocket message: %v", err)
				client.Close()
				p.wsClientsMutex.RUnlock()
				p.wsClientsMutex.Lock()
				delete(p.wsClients, client)
				p.wsClientsMutex.Unlock()
				p.wsClientsMutex.RLock()
			}
		}
		p.wsClientsMutex.RUnlock()
	}
}

// subscribeToDataEngineEvents is a no-op since WebSocket events are handled
// directly by the DataEngine's WebSocket server endpoints (/alerts, /metrics)
func (p *GoReverseProxy) subscribeToDataEngineEvents() {
	// Clients should connect directly to these WebSocket endpoints:
	// /alerts - for alert notifications
	// /metrics - for metrics updates
}

// handleHealth handles health check requests
func (p *GoReverseProxy) handleHealth(w http.ResponseWriter, _ *http.Request) {
	// Create health status
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "reverse-proxy",
	}

	// Add data engine status
	if p.dataEngine != nil {
		health["data_engine"] = map[string]interface{}{
			"running": p.dataEngine.IsRunning(),
		}

		// Add Kafka status
		if p.dataEngine.GetProducer() != nil {
			health["kafka"] = map[string]interface{}{
				"connected": p.dataEngine.GetProducer().IsConnected(),
			}
		}

		// Add ChromaDB status
		if p.dataEngine.GetChromaDB() != nil {
			health["chromadb"] = map[string]interface{}{
				"connected": p.dataEngine.GetChromaDB().IsConnected(),
			}
		}
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(health)
}

// handleMetrics handles metrics requests
func (p *GoReverseProxy) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	if p.dataEngine == nil {
		http.Error(w, "Data engine not available", http.StatusServiceUnavailable)
		return
	}

	// Get metrics
	metrics := p.dataEngine.GetMetrics()
	if metrics == nil {
		http.Error(w, "No metrics available", http.StatusNotFound)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(metrics)
}

// handleAlerts handles alerts requests
func (p *GoReverseProxy) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetAlerting() == nil {
		http.Error(w, "Alerting system not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	activeOnly := r.URL.Query().Get("active") == "true"

	var alerts []dataengine.Alert
	if activeOnly {
		alerts = p.dataEngine.GetAlerting().GetActiveAlerts()
	} else {
		alerts = p.dataEngine.GetAlerting().GetAlerts()
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(alerts)
}

// handleEvents handles events requests
func (p *GoReverseProxy) handleEvents(w http.ResponseWriter, r *http.Request) {
	if p.dataEngine == nil || p.dataEngine.GetChromaDB() == nil || !p.dataEngine.GetChromaDB().IsConnected() {
		http.Error(w, "ChromaDB not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	eventType := r.URL.Query().Get("type")

	// Set default limit
	limit := 10
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}
	}

	var docs []dataengine.ChromaDocument
	var err error

	// Query events
	if eventType != "" {
		// Filter by event type
		docs, err = p.dataEngine.GetChromaDB().GetEventsByType(r.Context(), dataengine.EventType(eventType), limit)
	} else {
		// Get recent events
		docs, err = p.dataEngine.GetChromaDB().GetRecentEvents(r.Context(), limit)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query events: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Write response
	json.NewEncoder(w).Encode(docs)
}
