package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

type APIGateway struct {
	router        *mux.Router
	services      map[string]*ServiceConfig
	servicesMutex sync.RWMutex
	wsConnections map[string]*websocket.Conn
	wsConnMutex   sync.RWMutex
	authService   *AuthenticationService
	rateLimiter   *RateLimiter
	metrics       *GatewayMetrics
}

type ServiceConfig struct {
	Name       string            `json:"name"`
	URL        string            `json:"url"`
	HealthPath string            `json:"health_path"`
	Routes     []RouteConfig     `json:"routes"`
	Headers    map[string]string `json:"headers"`
	Timeout    time.Duration     `json:"timeout"`
	IsHealthy  bool              `json:"is_healthy"`
	LastCheck  time.Time         `json:"last_check"`
}

type RouteConfig struct {
	Path         string   `json:"path"`
	Methods      []string `json:"methods"`
	AuthRequired bool     `json:"auth_required"`
	RateLimit    int      `json:"rate_limit"`
}

type GatewayMetrics struct {
	TotalRequests  int64                      `json:"total_requests"`
	SuccessfulReqs int64                      `json:"successful_requests"`
	FailedReqs     int64                      `json:"failed_requests"`
	ServiceMetrics map[string]*ServiceMetrics `json:"service_metrics"`
	ResponseTimes  []time.Duration            `json:"response_times"`
	mutex          sync.RWMutex
}

type ServiceMetrics struct {
	Requests    int64         `json:"requests"`
	Errors      int64         `json:"errors"`
	AvgLatency  time.Duration `json:"avg_latency"`
	LastRequest time.Time     `json:"last_request"`
}

type AuthenticationService struct {
	validTokens map[string]*TokenInfo
	mutex       sync.RWMutex
}

type TokenInfo struct {
	UserID    string    `json:"user_id"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

func NewAPIGateway() *APIGateway {
	gateway := &APIGateway{
		router:        mux.NewRouter(),
		services:      make(map[string]*ServiceConfig),
		wsConnections: make(map[string]*websocket.Conn),
		authService:   NewAuthenticationService(),
		rateLimiter:   NewRateLimiter(100, time.Minute), // 100 requests per minute
		metrics:       NewGatewayMetrics(),
	}

	gateway.setupRoutes()
	gateway.startHealthChecks()

	return gateway
}

func NewAuthenticationService() *AuthenticationService {
	return &AuthenticationService{
		validTokens: make(map[string]*TokenInfo),
	}
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func NewGatewayMetrics() *GatewayMetrics {
	return &GatewayMetrics{
		ServiceMetrics: make(map[string]*ServiceMetrics),
		ResponseTimes:  make([]time.Duration, 0),
	}
}

func (gw *APIGateway) setupRoutes() {
	// Gateway management routes
	gw.router.HandleFunc("/gateway/health", gw.handleGatewayHealth).Methods("GET")
	gw.router.HandleFunc("/gateway/metrics", gw.handleGatewayMetrics).Methods("GET")
	gw.router.HandleFunc("/gateway/services", gw.handleListServices).Methods("GET")
	gw.router.HandleFunc("/gateway/services", gw.handleRegisterService).Methods("POST")
	gw.router.HandleFunc("/gateway/services/{service}", gw.handleUpdateService).Methods("PUT")
	gw.router.HandleFunc("/gateway/services/{service}", gw.handleUnregisterService).Methods("DELETE")

	// WebSocket endpoint
	gw.router.HandleFunc("/gateway/ws", gw.handleWebSocket)

	// Authentication routes
	gw.router.HandleFunc("/auth/login", gw.handleLogin).Methods("POST")
	gw.router.HandleFunc("/auth/logout", gw.handleLogout).Methods("POST")
	gw.router.HandleFunc("/auth/validate", gw.handleValidateToken).Methods("GET")

	// Service proxy routes (catch-all)
	gw.router.PathPrefix("/").HandlerFunc(gw.handleServiceProxy)
}

func (gw *APIGateway) RegisterService(config *ServiceConfig) error {
	gw.servicesMutex.Lock()
	defer gw.servicesMutex.Unlock()

	// Validate service configuration
	if err := gw.validateServiceConfig(config); err != nil {
		return fmt.Errorf("invalid service config: %w", err)
	}

	// Initialize service metrics
	gw.metrics.mutex.Lock()
	gw.metrics.ServiceMetrics[config.Name] = &ServiceMetrics{
		Requests:    0,
		Errors:      0,
		AvgLatency:  0,
		LastRequest: time.Time{},
	}
	gw.metrics.mutex.Unlock()

	gw.services[config.Name] = config
	log.Printf("Registered service: %s at %s", config.Name, config.URL)

	return nil
}

func (gw *APIGateway) validateServiceConfig(config *ServiceConfig) error {
	if config.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if config.URL == "" {
		return fmt.Errorf("service URL is required")
	}
	if _, err := url.Parse(config.URL); err != nil {
		return fmt.Errorf("invalid service URL: %w", err)
	}
	return nil
}

func (gw *APIGateway) handleServiceProxy(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Update metrics
	gw.metrics.mutex.Lock()
	gw.metrics.TotalRequests++
	gw.metrics.mutex.Unlock()

	// Extract service name from path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) == 0 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		gw.recordFailedRequest()
		return
	}

	serviceName := pathParts[0]

	// Get service configuration
	gw.servicesMutex.RLock()
	service, exists := gw.services[serviceName]
	gw.servicesMutex.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Service '%s' not found", serviceName), http.StatusNotFound)
		gw.recordFailedRequest()
		return
	}

	// Check service health
	if !service.IsHealthy {
		http.Error(w, fmt.Sprintf("Service '%s' is unhealthy", serviceName), http.StatusServiceUnavailable)
		gw.recordFailedRequest()
		return
	}

	// Find matching route
	route := gw.findMatchingRoute(service, r.URL.Path, r.Method)
	if route == nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		gw.recordFailedRequest()
		return
	}

	// Check authentication
	if route.AuthRequired {
		if !gw.isAuthenticated(r) {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			gw.recordFailedRequest()
			return
		}
	}

	// Check rate limiting
	clientIP := gw.getClientIP(r)
	if !gw.rateLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		gw.recordFailedRequest()
		return
	}

	// Proxy the request
	if err := gw.proxyRequest(w, r, service); err != nil {
		log.Printf("Proxy error for service %s: %v", serviceName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		gw.recordFailedRequest()
		return
	}

	// Record successful request
	duration := time.Since(startTime)
	gw.recordSuccessfulRequest(serviceName, duration)
}

func (gw *APIGateway) findMatchingRoute(service *ServiceConfig, path, method string) *RouteConfig {
	// Remove service name from path
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) > 0 {
		path = "/" + strings.Join(pathParts[1:], "/")
	}

	for _, route := range service.Routes {
		if gw.pathMatches(route.Path, path) && gw.methodMatches(route.Methods, method) {
			return &route
		}
	}

	return nil
}

func (gw *APIGateway) pathMatches(routePath, requestPath string) bool {
	// Simple path matching - could be enhanced with wildcards
	return routePath == requestPath || routePath == "/*"
}

func (gw *APIGateway) methodMatches(allowedMethods []string, requestMethod string) bool {
	if len(allowedMethods) == 0 {
		return true // Allow all methods if none specified
	}

	for _, method := range allowedMethods {
		if method == requestMethod {
			return true
		}
	}
	return false
}

func (gw *APIGateway) isAuthenticated(r *http.Request) bool {
	token := gw.extractToken(r)
	if token == "" {
		return false
	}

	return gw.authService.ValidateToken(token)
}

func (gw *APIGateway) extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Check query parameter
	return r.URL.Query().Get("token")
}

func (gw *APIGateway) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Use remote address
	return strings.Split(r.RemoteAddr, ":")[0]
}

func (gw *APIGateway) proxyRequest(w http.ResponseWriter, r *http.Request, service *ServiceConfig) error {
	// Parse service URL
	serviceURL, err := url.Parse(service.URL)
	if err != nil {
		return err
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(serviceURL)

	// Modify request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Remove service name from path
		pathParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		if len(pathParts) > 0 {
			req.URL.Path = "/" + strings.Join(pathParts[1:], "/")
		}

		// Add custom headers
		for key, value := range service.Headers {
			req.Header.Set(key, value)
		}

		// Add gateway headers
		req.Header.Set("X-Gateway-Service", service.Name)
		req.Header.Set("X-Gateway-Timestamp", time.Now().Format(time.RFC3339))
	}

	// Set timeout
	if service.Timeout > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), service.Timeout)
		defer cancel()
		r = r.WithContext(ctx)
	}

	// Proxy the request
	proxy.ServeHTTP(w, r)
	return nil
}

func (gw *APIGateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Generate connection ID
	connID := fmt.Sprintf("ws_%d", time.Now().UnixNano())

	// Store connection
	gw.wsConnMutex.Lock()
	gw.wsConnections[connID] = conn
	gw.wsConnMutex.Unlock()

	// Remove connection on exit
	defer func() {
		gw.wsConnMutex.Lock()
		delete(gw.wsConnections, connID)
		gw.wsConnMutex.Unlock()
	}()

	log.Printf("WebSocket connection established: %s", connID)

	// Handle messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process message
		response := gw.processWebSocketMessage(msg)

		// Send response
		if err := conn.WriteJSON(response); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}

func (gw *APIGateway) processWebSocketMessage(msg map[string]interface{}) map[string]interface{} {
	msgType, ok := msg["type"].(string)
	if !ok {
		return map[string]interface{}{
			"type":  "error",
			"error": "Missing message type",
		}
	}

	switch msgType {
	case "ping":
		return map[string]interface{}{
			"type":      "pong",
			"timestamp": time.Now().Unix(),
		}

	case "subscribe":
		service, ok := msg["service"].(string)
		if !ok {
			return map[string]interface{}{
				"type":  "error",
				"error": "Missing service name",
			}
		}

		return map[string]interface{}{
			"type":    "subscribed",
			"service": service,
		}

	case "get_metrics":
		return map[string]interface{}{
			"type":    "metrics",
			"metrics": gw.getMetricsData(),
		}

	default:
		return map[string]interface{}{
			"type":  "error",
			"error": "Unknown message type",
		}
	}
}

func (gw *APIGateway) broadcastToWebSockets(message map[string]interface{}) {
	gw.wsConnMutex.RLock()
	defer gw.wsConnMutex.RUnlock()

	for connID, conn := range gw.wsConnections {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Failed to send WebSocket message to %s: %v", connID, err)
			// Connection will be cleaned up by the handler goroutine
		}
	}
}

func (gw *APIGateway) startHealthChecks() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			gw.performHealthChecks()
		}
	}()
}

func (gw *APIGateway) performHealthChecks() {
	gw.servicesMutex.RLock()
	services := make([]*ServiceConfig, 0, len(gw.services))
	for _, service := range gw.services {
		services = append(services, service)
	}
	gw.servicesMutex.RUnlock()

	for _, service := range services {
		go gw.checkServiceHealth(service)
	}
}

func (gw *APIGateway) checkServiceHealth(service *ServiceConfig) {
	healthURL := service.URL + service.HealthPath
	if service.HealthPath == "" {
		healthURL = service.URL + "/health"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(healthURL)

	wasHealthy := service.IsHealthy
	service.IsHealthy = (err == nil && resp != nil && resp.StatusCode == http.StatusOK)
	service.LastCheck = time.Now()

	if resp != nil {
		resp.Body.Close()
	}

	// Notify if health status changed
	if wasHealthy != service.IsHealthy {
		status := "unhealthy"
		if service.IsHealthy {
			status = "healthy"
		}

		log.Printf("Service %s is now %s", service.Name, status)

		// Broadcast health change via WebSocket
		gw.broadcastToWebSockets(map[string]interface{}{
			"type":      "health_change",
			"service":   service.Name,
			"healthy":   service.IsHealthy,
			"timestamp": time.Now().Unix(),
		})
	}
}

// Authentication methods
func (auth *AuthenticationService) ValidateToken(token string) bool {
	auth.mutex.RLock()
	defer auth.mutex.RUnlock()

	tokenInfo, exists := auth.validTokens[token]
	if !exists {
		return false
	}

	return time.Now().Before(tokenInfo.ExpiresAt)
}

func (auth *AuthenticationService) CreateToken(userID string, scopes []string, duration time.Duration) string {
	auth.mutex.Lock()
	defer auth.mutex.Unlock()

	token := fmt.Sprintf("token_%s_%d", userID, time.Now().UnixNano())
	auth.validTokens[token] = &TokenInfo{
		UserID:    userID,
		Scopes:    scopes,
		ExpiresAt: time.Now().Add(duration),
	}

	return token
}

func (auth *AuthenticationService) RevokeToken(token string) {
	auth.mutex.Lock()
	defer auth.mutex.Unlock()

	delete(auth.validTokens, token)
}

// Rate limiting methods
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Clean old requests
	if requests, exists := rl.requests[clientID]; exists {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if now.Sub(reqTime) < rl.window {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[clientID] = validRequests
	}

	// Check if under limit
	if len(rl.requests[clientID]) >= rl.limit {
		return false
	}

	// Add current request
	rl.requests[clientID] = append(rl.requests[clientID], now)
	return true
}

// Metrics methods
func (gw *APIGateway) recordSuccessfulRequest(serviceName string, duration time.Duration) {
	gw.metrics.mutex.Lock()
	defer gw.metrics.mutex.Unlock()

	gw.metrics.SuccessfulReqs++

	// Update service metrics
	if serviceMetrics, exists := gw.metrics.ServiceMetrics[serviceName]; exists {
		serviceMetrics.Requests++
		serviceMetrics.LastRequest = time.Now()

		// Update average latency
		if serviceMetrics.Requests == 1 {
			serviceMetrics.AvgLatency = duration
		} else {
			serviceMetrics.AvgLatency = time.Duration(
				(int64(serviceMetrics.AvgLatency)*(serviceMetrics.Requests-1) + int64(duration)) / serviceMetrics.Requests,
			)
		}
	}

	// Store response time (keep last 1000)
	gw.metrics.ResponseTimes = append(gw.metrics.ResponseTimes, duration)
	if len(gw.metrics.ResponseTimes) > 1000 {
		gw.metrics.ResponseTimes = gw.metrics.ResponseTimes[1:]
	}
}

func (gw *APIGateway) recordFailedRequest() {
	gw.metrics.mutex.Lock()
	defer gw.metrics.mutex.Unlock()

	gw.metrics.FailedReqs++
}

func (gw *APIGateway) getMetricsData() map[string]interface{} {
	gw.metrics.mutex.RLock()
	defer gw.metrics.mutex.RUnlock()

	return map[string]interface{}{
		"total_requests":      gw.metrics.TotalRequests,
		"successful_requests": gw.metrics.SuccessfulReqs,
		"failed_requests":     gw.metrics.FailedReqs,
		"service_metrics":     gw.metrics.ServiceMetrics,
		"avg_response_time":   gw.calculateAverageResponseTime(),
	}
}

func (gw *APIGateway) calculateAverageResponseTime() time.Duration {
	if len(gw.metrics.ResponseTimes) == 0 {
		return 0
	}

	var total int64
	for _, duration := range gw.metrics.ResponseTimes {
		total += int64(duration)
	}

	return time.Duration(total / int64(len(gw.metrics.ResponseTimes)))
}

// HTTP Handlers
func (gw *APIGateway) handleGatewayHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"services":  len(gw.services),
	})
}

func (gw *APIGateway) handleGatewayMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gw.getMetricsData())
}

func (gw *APIGateway) handleListServices(w http.ResponseWriter, r *http.Request) {
	gw.servicesMutex.RLock()
	defer gw.servicesMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gw.services)
}

func (gw *APIGateway) handleRegisterService(w http.ResponseWriter, r *http.Request) {
	var config ServiceConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := gw.RegisterService(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "registered",
		"service": config.Name,
	})
}

func (gw *APIGateway) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["service"]

	var config ServiceConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config.Name = serviceName
	if err := gw.RegisterService(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "updated",
		"service": serviceName,
	})
}

func (gw *APIGateway) handleUnregisterService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceName := vars["service"]

	gw.servicesMutex.Lock()
	delete(gw.services, serviceName)
	gw.servicesMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "unregistered",
		"service": serviceName,
	})
}

func (gw *APIGateway) handleLogin(w http.ResponseWriter, r *http.Request) {
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Simple authentication (in production, use proper password hashing)
	if loginReq.Username == "admin" && loginReq.Password == "password" {
		token := gw.authService.CreateToken(loginReq.Username, []string{"admin"}, 24*time.Hour)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
			"user":  loginReq.Username,
		})
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

func (gw *APIGateway) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := gw.extractToken(r)
	if token != "" {
		gw.authService.RevokeToken(token)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "logged_out",
	})
}

func (gw *APIGateway) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	token := gw.extractToken(r)
	valid := gw.authService.ValidateToken(token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"valid": valid,
	})
}

func main() {
	gateway := NewAPIGateway()

	// Register KNIRV services based on current port configurations
	services := []*ServiceConfig{
		{
			Name:       "knirvchain",
			URL:        "http://localhost:8080",
			HealthPath: "/health",
			Routes: []RouteConfig{
				{Path: "/wallets/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
				{Path: "/nrn/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/skill/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/llm/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/blocks", Methods: []string{"GET"}, AuthRequired: false},
			},
			Headers: map[string]string{
				"X-Service": "knirvchain",
			},
			Timeout: 30 * time.Second,
		},
		{
			Name:       "knirvgraph",
			URL:        "http://localhost:8081", // KNIRVGRAPH typically runs on 8081
			HealthPath: "/health",
			Routes: []RouteConfig{
				{Path: "/height", Methods: []string{"GET"}, AuthRequired: false},
				{Path: "/node/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
				{Path: "/edge/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
				{Path: "/graph/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
				{Path: "/account/*", Methods: []string{"GET"}, AuthRequired: false},
				{Path: "/transaction", Methods: []string{"POST"}, AuthRequired: true},
				{Path: "/nrv/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
			},
			Headers: map[string]string{
				"X-Service": "knirvgraph",
			},
			Timeout: 20 * time.Second,
		},
		{
			Name:       "knirvnexus",
			URL:        "http://localhost:8082", // KNIRVNEXUS API port
			HealthPath: "/health",
			Routes: []RouteConfig{
				{Path: "/api/v1/agents/*", Methods: []string{"GET", "POST", "PUT", "DELETE"}, AuthRequired: true},
				{Path: "/api/v1/workflows/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/api/v1/mcp/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/api/v1/inference/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/desktop/*", Methods: []string{"GET"}, AuthRequired: false},
			},
			Headers: map[string]string{
				"X-Service": "knirvnexus",
			},
			Timeout: 60 * time.Second,
		},
		{
			Name:       "knirvroot",
			URL:        "http://localhost:5000", // KNIRVROOT default port
			HealthPath: "/health",
			Routes: []RouteConfig{
				{Path: "/chain", Methods: []string{"GET"}, AuthRequired: false},
				{Path: "/block", Methods: []string{"POST"}, AuthRequired: true},
				{Path: "/transaction", Methods: []string{"POST"}, AuthRequired: true},
				{Path: "/mcp/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/payment/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/bridge/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
				{Path: "/test/faucet", Methods: []string{"POST"}, AuthRequired: false},
				{Path: "/ping", Methods: []string{"GET"}, AuthRequired: false},
			},
			Headers: map[string]string{
				"X-Service": "knirvroot",
			},
			Timeout: 30 * time.Second,
		},
		{
			Name:       "knirvrouter",
			URL:        "http://localhost:3478", // KNIRVROUTER TURN server port
			HealthPath: "/api/health",
			Routes: []RouteConfig{
				{Path: "/api/connectivity/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
				{Path: "/api/proof/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
				{Path: "/api/mint/*", Methods: []string{"POST"}, AuthRequired: true},
				{Path: "/api/stats/*", Methods: []string{"GET"}, AuthRequired: false},
				{Path: "/turn/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
				{Path: "/ws", Methods: []string{"GET"}, AuthRequired: false},
			},
			Headers: map[string]string{
				"X-Service": "knirvrouter",
			},
			Timeout: 15 * time.Second,
		},
	}

	// Register all services
	for _, service := range services {
		if err := gateway.RegisterService(service); err != nil {
			log.Fatalf("Failed to register service %s: %v", service.Name, err)
		}
	}

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(gateway.router)

	log.Println("Starting API Gateway on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", handler))
}
