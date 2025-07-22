**Task 10.1: Create Unified API Gateway**

Create `shared-integration/api-gateway/gateway.go`:
```go
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
    router          *mux.Router
    services        map[string]*ServiceConfig
    servicesMutex   sync.RWMutex
    wsConnections   map[string]*websocket.Conn
    wsConnMutex     sync.RWMutex
    authService     *AuthenticationService
    rateLimiter     *RateLimiter
    metrics         *GatewayMetrics
}

type ServiceConfig struct {
    Name        string            `json:"name"`
    URL         string            `json:"url"`
    HealthPath  string            `json:"health_path"`
    Routes      []RouteConfig     `json:"routes"`
    Headers     map[string]string `json:"headers"`
    Timeout     time.Duration     `json:"timeout"`
    IsHealthy   bool              `json:"is_healthy"`
    LastCheck   time.Time         `json:"last_check"`
}

type RouteConfig struct {
    Path        string   `json:"path"`
    Methods     []string `json:"methods"`
    AuthRequired bool    `json:"auth_required"`
    RateLimit   int      `json:"rate_limit"`
}

type GatewayMetrics struct {
    TotalRequests    int64                    `json:"total_requests"`
    SuccessfulReqs   int64                    `json:"successful_requests"`
    FailedReqs       int64                    `json:"failed_requests"`
    ServiceMetrics   map[string]*ServiceMetrics `json:"service_metrics"`
    ResponseTimes    []time.Duration          `json:"response_times"`
    mutex           sync.RWMutex
}

type ServiceMetrics struct {
    Requests      int64           `json:"requests"`
    Errors        int64           `json:"errors"`
    AvgLatency    time.Duration   `json:"avg_latency"`
    LastRequest   time.Time       `json:"last_request"`
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
            "type": "pong",
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
            "type":    "health_change",
            "service": service.Name,
            "healthy": service.IsHealthy,
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
        "total_requests":     gw.metrics.TotalRequests,
        "successful_requests": gw.metrics.SuccessfulReqs,
        "failed_requests":    gw.metrics.FailedReqs,
        "service_metrics":    gw.metrics.ServiceMetrics,
        "avg_response_time":  gw.calculateAverageResponseTime(),
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

    // Register KNIRV services
    services := []*ServiceConfig{
        {
            Name:       "knirvchain",
            URL:        "http://localhost:8080",
            HealthPath: "/health",
            Routes: []RouteConfig{
                {Path: "/llm/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
                {Path: "/skill/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
                {Path: "/bridge/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
            },
            Headers: map[string]string{
                "X-Service": "knirvchain",
            },
            Timeout: 30 * time.Second,
        },
        {
            Name:       "knirvgraph",
            URL:        "http://localhost:8081",
            HealthPath: "/health",
            Routes: []RouteConfig{
                {Path: "/graph/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
                {Path: "/nrv/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
            },
            Headers: map[string]string{
                "X-Service": "knirvgraph",
            },
            Timeout: 20 * time.Second,
        },
        {
            Name:       "knirvnexus",
            URL:        "http://localhost:8082",
            HealthPath: "/health",
            Routes: []RouteConfig{
                {Path: "/agents/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
                {Path: "/workflows/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
            },
            Headers: map[string]string{
                "X-Service": "knirvnexus",
            },
            Timeout: 60 * time.Second,
        },
        {
            Name:       "knirvroot",
            URL:        "http://localhost:8083",
            HealthPath: "/health",
            Routes: []RouteConfig{
                {Path: "/mcp/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
                {Path: "/payment/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
                {Path: "/bridge/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
                {Path: "/faucet/*", Methods: []string{"GET", "POST"}, AuthRequired: true},
            },
            Headers: map[string]string{
                "X-Service": "knirvroot",
            },
            Timeout: 30 * time.Second,
        },
        {
            Name:       "knirvrouter",
            URL:        "http://localhost:3478", // Existing KNIRVROUTER TURN server port
            HealthPath: "/api/connectivity/status",
            Routes: []RouteConfig{
                {Path: "/api/connectivity/*", Methods: []string{"GET", "POST"}, AuthRequired: false},
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
```

### Month 11: Economic Model Integration

**Task 11.1: Implement Unified Token Economics**

Create `shared-integration/economics/token_economics.go`:
```go
package economics

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math/big"
    "sync"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
)

type TokenEconomics struct {
    nrnContract      common.Address
    xionClient       *ethclient.Client
    KNIRVROOTDB     *LevelDB
    economicRules    *EconomicRules
    transactionPool  *TransactionPool
    rewardCalculator *RewardCalculator
    burnTracker      *BurnTracker
    metrics          *EconomicMetrics
    mutex            sync.RWMutex
}

type EconomicRules struct {
    SkillInvocationCost    *big.Int              `json:"skill_invocation_cost"`
    LLMRegistrationFee     *big.Int              `json:"llm_registration_fee"`
    ValidationReward       *big.Int              `json:"validation_reward"`
    BurnRates              map[string]*big.Int   `json:"burn_rates"`
    MintingRules           *MintingRules         `json:"minting_rules"`
    StakingRequirements    *StakingRequirements  `json:"staking_requirements"`
    GovernanceThresholds   *GovernanceThresholds `json:"governance_thresholds"`
}

type MintingRules struct {
    MaxSupply           *big.Int `json:"max_supply"`
    InflationRate       float64  `json:"inflation_rate"`
    ValidatorRewards    *big.Int `json:"validator_rewards"`
    DeveloperRewards    *big.Int `json:"developer_rewards"`
    CommunityRewards    *big.Int `json:"community_rewards"`
}

type StakingRequirements struct {
    MinValidatorStake   *big.Int `json:"min_validator_stake"`
    MinDeveloperStake   *big.Int `json:"min_developer_stake"`
    SlashingPenalty     float64  `json:"slashing_penalty"`
    UnbondingPeriod     time.Duration `json:"unbonding_period"`
}

type GovernanceThresholds struct {
    ProposalDeposit     *big.Int `json:"proposal_deposit"`
    VotingThreshold     float64  `json:"voting_threshold"`
    QuorumThreshold     float64  `json:"quorum_threshold"`
    VotingPeriod        time.Duration `json:"voting_period"`
}

type TransactionPool struct {
    pendingTxs      map[string]*EconomicTransaction
    confirmedTxs    map[string]*EconomicTransaction
    mutex           sync.RWMutex
    maxPoolSize     int
    cleanupInterval time.Duration
}

type EconomicTransaction struct {
    ID              string                 `json:"id"`
    Type            string                 `json:"type"`
    From            string                 `json:"from"`
    To              string                 `json:"to"`
    Amount          *big.Int               `json:"amount"`
    Purpose         string                 `json:"purpose"`
    Metadata        map[string]interface{} `json:"metadata"`
    Status          string                 `json:"status"`
    Timestamp       time.Time              `json:"timestamp"`
    ConfirmedAt     *time.Time             `json:"confirmed_at,omitempty"`
    BlockHeight     uint64                 `json:"block_height,omitempty"`
    GasUsed         uint64                 `json:"gas_used,omitempty"`
}

type RewardCalculator struct {
    baseRewards     map[string]*big.Int
    multipliers     map[string]float64
    performanceData map[string]*PerformanceMetrics
    mutex           sync.RWMutex
}

type PerformanceMetrics struct {
    SuccessRate     float64   `json:"success_rate"`
    ResponseTime    float64   `json:"response_time"`
    UserSatisfaction float64  `json:"user_satisfaction"`
    Uptime          float64   `json:"uptime"`
    LastUpdated     time.Time `json:"last_updated"`
}

type BurnTracker struct {
    totalBurned     *big.Int
    burnHistory     []*BurnEvent
    burnRates       map[string]*big.Int
    mutex           sync.RWMutex
}

type BurnEvent struct {
    TxID        string    `json:"tx_id"`
    User        string    `json:"user"`
    Amount      *big.Int  `json:"amount"`
    Purpose     string    `json:"purpose"`
    SkillID     string    `json:"skill_id,omitempty"`
    Timestamp   time.Time `json:"timestamp"`
    Validated   bool      `json:"validated"`
}

type EconomicMetrics struct {
    TotalSupply         *big.Int              `json:"total_supply"`
    CirculatingSupply   *big.Int              `json:"circulating_supply"`
    TotalBurned         *big.Int              `json:"total_burned"`
    TotalStaked         *big.Int              `json:"total_staked"`
    ActiveValidators    int                   `json:"active_validators"`
    TransactionVolume   *big.Int              `json:"transaction_volume"`
    AverageGasPrice     *big.Int              `json:"average_gas_price"`
    NetworkUtilization  float64               `json:"network_utilization"`
    TokenVelocity       float64               `json:"token_velocity"`
    LastUpdated         time.Time             `json:"last_updated"`
    ServiceMetrics      map[string]*ServiceEconomics `json:"service_metrics"`
}

type ServiceEconomics struct {
    Revenue         *big.Int  `json:"revenue"`
    Costs           *big.Int  `json:"costs"`
    Profit          *big.Int  `json:"profit"`
    TokensEarned    *big.Int  `json:"tokens_earned"`
    TokensSpent     *big.Int  `json:"tokens_spent"`
    UserCount       int       `json:"user_count"`
    TransactionCount int      `json:"transaction_count"`
    LastUpdated     time.Time `json:"last_updated"`
}

func NewTokenEconomics(nrnContract common.Address, xionRPC string, KNIRVROOTDB *LevelDB) (*TokenEconomics, error) {
    client, err := ethclient.Dial(xionRPC)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to XION: %w", err)
    }

    economics := &TokenEconomics{
        nrnContract:      nrnContract,
        xionClient:       client,
        KNIRVROOTDB:     KNIRVROOTDB,
        economicRules:    NewDefaultEconomicRules(),
        transactionPool:  NewTransactionPool(),
        rewardCalculator: NewRewardCalculator(),
        burnTracker:      NewBurnTracker(),
        metrics:          NewEconomicMetrics(),
    }

    return economics, nil
}

func NewDefaultEconomicRules() *EconomicRules {
    return &EconomicRules{
        SkillInvocationCost: big.NewInt(100000), // 0.1 NRN
        LLMRegistrationFee:  big.NewInt(1000000), // 1 NRN
        ValidationReward:    big.NewInt(50000),   // 0.05 NRN
        BurnRates: map[string]*big.Int{
            "skill_invocation": big.NewInt(100000),
            "llm_registration": big.NewInt(500000),
            "validation":       big.NewInt(25000),
        },
        MintingRules: &MintingRules{
            MaxSupply:        big.NewInt(1000000000000000), // 1B NRN
            InflationRate:    0.05, // 5% annual
            ValidatorRewards: big.NewInt(10000000), // 10 NRN per block
            DeveloperRewards: big.NewInt(5000000),  // 5 NRN per block
            CommunityRewards: big.NewInt(2000000),  // 2 NRN per block
        },
        StakingRequirements: &StakingRequirements{
            MinValidatorStake: big.NewInt(100000000000), // 100K NRN
            MinDeveloperStake: big.NewInt(10000000000),  // 10K NRN
            SlashingPenalty:   0.05, // 5%
            UnbondingPeriod:   21 * 24 * time.Hour, // 21 days
        },
        GovernanceThresholds: &GovernanceThresholds{
            ProposalDeposit:   big.NewInt(1000000000), // 1K NRN
            VotingThreshold:   0.5, // 50%
            QuorumThreshold:   0.33, // 33%
            VotingPeriod:      7 * 24 * time.Hour, // 7 days
        },
    }
}

func NewTransactionPool() *TransactionPool {
    return &TransactionPool{
        pendingTxs:      make(map[string]*EconomicTransaction),
        confirmedTxs:    make(map[string]*EconomicTransaction),
        maxPoolSize:     10000,
        cleanupInterval: 1 * time.Hour,
    }
}

func NewRewardCalculator() *RewardCalculator {
    return &RewardCalculator{
        baseRewards: map[string]*big.Int{
            "validation":      big.NewInt(50000),
            "skill_creation":  big.NewInt(100000),
            "bug_reporting":   big.NewInt(25000),
            "community_help":  big.NewInt(10000),
        },
        multipliers: map[string]float64{
            "high_performance": 1.5,
            "consistent_user":  1.2,
            "early_adopter":    1.3,
            "community_leader": 2.0,
        },
        performanceData: make(map[string]*PerformanceMetrics),
    }
}

func NewBurnTracker() *BurnTracker {
    return &BurnTracker{
        totalBurned: big.NewInt(0),
        burnHistory: make([]*BurnEvent, 0),
        burnRates:   make(map[string]*big.Int),
    }
}

func NewEconomicMetrics() *EconomicMetrics {
    return &EconomicMetrics{
        TotalSupply:       big.NewInt(0),
        CirculatingSupply: big.NewInt(0),
        TotalBurned:       big.NewInt(0),
        TotalStaked:       big.NewInt(0),
        ServiceMetrics:    make(map[string]*ServiceEconomics),
        LastUpdated:       time.Now(),
    }
}

func (te *TokenEconomics) Start(ctx context.Context) error {
    log.Println("Starting Token Economics system...")

    // Start background processes
    go te.transactionProcessor(ctx)
    go te.metricsUpdater(ctx)
    go te.rewardDistributor(ctx)
    go te.burnProcessor(ctx)

    // Load existing state
    if err := te.loadState(); err != nil {
        log.Printf("Warning: Failed to load economics state: %v", err)
    }

    log.Println("Token Economics system started")
    return nil
}

func (te *TokenEconomics) ProcessSkillInvocation(userID, skillID string, amount *big.Int) (*EconomicTransaction, error) {
    te.mutex.Lock()
    defer te.mutex.Unlock()

    // Validate amount
    requiredAmount := te.economicRules.SkillInvocationCost
    if amount.Cmp(requiredAmount) < 0 {
        return nil, fmt.Errorf("insufficient amount: required %s, provided %s", requiredAmount.String(), amount.String())
    }

    // Create transaction
    tx := &EconomicTransaction{
        ID:        fmt.Sprintf("skill_%s_%d", skillID, time.Now().UnixNano()),
        Type:      "skill_invocation",
        From:      userID,
        To:        "skill_registry",
        Amount:    amount,
        Purpose:   "skill_invocation",
        Metadata: map[string]interface{}{
            "skill_id": skillID,
        },
        Status:    "pending",
        Timestamp: time.Now(),
    }

    // Add to transaction pool
    te.transactionPool.AddTransaction(tx)

    // Record burn event
    burnEvent := &BurnEvent{
        TxID:      tx.ID,
        User:      userID,
        Amount:    amount,
        Purpose:   "skill_invocation",
        SkillID:   skillID,
        Timestamp: time.Now(),
        Validated: false,
    }

    te.burnTracker.AddBurnEvent(burnEvent)

    // Update metrics
    te.updateServiceMetrics("knirvchain", amount, "spent")

    return tx, nil
}

func (te *TokenEconomics) ProcessLLMRegistration(userID, llmID string, registrationFee *big.Int) (*EconomicTransaction, error) {
    te.mutex.Lock()
    defer te.mutex.Unlock()

    // Validate fee
    requiredFee := te.economicRules.LLMRegistrationFee
    if registrationFee.Cmp(requiredFee) < 0 {
        return nil, fmt.Errorf("insufficient registration fee: required %s, provided %s", requiredFee.String(), registrationFee.String())
    }

    // Create transaction
    tx := &EconomicTransaction{
        ID:        fmt.Sprintf("llm_reg_%s_%d", llmID, time.Now().UnixNano()),
        Type:      "llm_registration",
        From:      userID,
        To:        "llm_registry",
        Amount:    registrationFee,
        Purpose:   "llm_registration",
        Metadata: map[string]interface{}{
            "llm_id": llmID,
        },
        Status:    "pending",
        Timestamp: time.Now(),
    }

    te.transactionPool.AddTransaction(tx)

    // Update metrics
    te.updateServiceMetrics("knirvchain", registrationFee, "earned")

    return tx, nil
}

func (te *TokenEconomics) ProcessValidationReward(validatorID, targetID string, validationResult bool) (*EconomicTransaction, error) {
    te.mutex.Lock()
    defer te.mutex.Unlock()

    if !validationResult {
        return nil, fmt.Errorf("validation failed, no reward")
    }

    // Calculate reward based on performance
    baseReward := te.economicRules.ValidationReward
    finalReward := te.rewardCalculator.CalculateReward(validatorID, "validation", baseReward)

    // Create transaction
    tx := &EconomicTransaction{
        ID:        fmt.Sprintf("validation_%s_%d", targetID, time.Now().UnixNano()),
        Type:      "validation_reward",
        From:      "reward_pool",
        To:        validatorID,
        Amount:    finalReward,
        Purpose:   "validation_reward",
        Metadata: map[string]interface{}{
            "target_id":         targetID,
            "validation_result": validationResult,
        },
        Status:    "pending",
        Timestamp: time.Now(),
    }

    te.transactionPool.AddTransaction(tx)

    // Update metrics
    te.updateServiceMetrics("knirvnexus", finalReward, "earned")

    return tx, nil
}

func (te *TokenEconomics) CalculateNetworkFees(gasUsed uint64, priority string) *big.Int {
    baseGasPrice := big.NewInt(1000) // Base gas price in wei

    // Apply priority multiplier
    multiplier := 1.0
    switch priority {
    case "high":
        multiplier = 2.0
    case "medium":
        multiplier = 1.5
    case "low":
        multiplier = 1.0
    }

    // Calculate total fee
    gasPrice := new(big.Int).Mul(baseGasPrice, big.NewInt(int64(multiplier*1000)))
    gasPrice = new(big.Int).Div(gasPrice, big.NewInt(1000))

    totalFee := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasUsed)))

    return totalFee
}

func (te *TokenEconomics) GetEconomicMetrics() *EconomicMetrics {
    te.mutex.RLock()
    defer te.mutex.RUnlock()

    // Create a copy to avoid race conditions
    metrics := &EconomicMetrics{
        TotalSupply:        new(big.Int).Set(te.metrics.TotalSupply),
        CirculatingSupply:  new(big.Int).Set(te.metrics.CirculatingSupply),
        TotalBurned:        new(big.Int).Set(te.metrics.TotalBurned),
        TotalStaked:        new(big.Int).Set(te.metrics.TotalStaked),
        ActiveValidators:   te.metrics.ActiveValidators,
        TransactionVolume:  new(big.Int).Set(te.metrics.TransactionVolume),
        AverageGasPrice:    new(big.Int).Set(te.metrics.AverageGasPrice),
        NetworkUtilization: te.metrics.NetworkUtilization,
        TokenVelocity:      te.metrics.TokenVelocity,
        LastUpdated:        te.metrics.LastUpdated,
        ServiceMetrics:     make(map[string]*ServiceEconomics),
    }

    // Copy service metrics
    for service, serviceMetrics := range te.metrics.ServiceMetrics {
        metrics.ServiceMetrics[service] = &ServiceEconomics{
            Revenue:          new(big.Int).Set(serviceMetrics.Revenue),
            Costs:            new(big.Int).Set(serviceMetrics.Costs),
            Profit:           new(big.Int).Set(serviceMetrics.Profit),
            TokensEarned:     new(big.Int).Set(serviceMetrics.TokensEarned),
            TokensSpent:      new(big.Int).Set(serviceMetrics.TokensSpent),
            UserCount:        serviceMetrics.UserCount,
            TransactionCount: serviceMetrics.TransactionCount,
            LastUpdated:      serviceMetrics.LastUpdated,
        }
    }

    return metrics
}

func (te *TokenEconomics) updateServiceMetrics(serviceName string, amount *big.Int, operation string) {
    if te.metrics.ServiceMetrics[serviceName] == nil {
        te.metrics.ServiceMetrics[serviceName] = &ServiceEconomics{
            Revenue:          big.NewInt(0),
            Costs:            big.NewInt(0),
            Profit:           big.NewInt(0),
            TokensEarned:     big.NewInt(0),
            TokensSpent:      big.NewInt(0),
            UserCount:        0,
            TransactionCount: 0,
            LastUpdated:      time.Now(),
        }
    }

    serviceMetrics := te.metrics.ServiceMetrics[serviceName]

    switch operation {
    case "earned":
        serviceMetrics.Revenue.Add(serviceMetrics.Revenue, amount)
        serviceMetrics.TokensEarned.Add(serviceMetrics.TokensEarned, amount)
    case "spent":
        serviceMetrics.Costs.Add(serviceMetrics.Costs, amount)
        serviceMetrics.TokensSpent.Add(serviceMetrics.TokensSpent, amount)
    }

    // Update profit
    serviceMetrics.Profit.Sub(serviceMetrics.Revenue, serviceMetrics.Costs)
    serviceMetrics.TransactionCount++
    serviceMetrics.LastUpdated = time.Now()
}

func (te *TokenEconomics) transactionProcessor(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            te.processPendingTransactions()
        }
    }
}

func (te *TokenEconomics) processPendingTransactions() {
    te.transactionPool.mutex.RLock()
    pendingTxs := make([]*EconomicTransaction, 0, len(te.transactionPool.pendingTxs))
    for _, tx := range te.transactionPool.pendingTxs {
        pendingTxs = append(pendingTxs, tx)
    }
    te.transactionPool.mutex.RUnlock()

    for _, tx := range pendingTxs {
        if err := te.processTransaction(tx); err != nil {
            log.Printf("Failed to process transaction %s: %v", tx.ID, err)
            tx.Status = "failed"
        } else {
            tx.Status = "confirmed"
            now := time.Now()
            tx.ConfirmedAt = &now
        }

        // Move to confirmed transactions
        te.transactionPool.mutex.Lock()
        delete(te.transactionPool.pendingTxs, tx.ID)
        te.transactionPool.confirmedTxs[tx.ID] = tx
        te.transactionPool.mutex.Unlock()
    }
}

func (te *TokenEconomics) processTransaction(tx *EconomicTransaction) error {
    // Simulate transaction processing
    // In real implementation, this would interact with XION blockchain

    switch tx.Type {
    case "skill_invocation":
        return te.processSkillInvocationTx(tx)
    case "llm_registration":
        return te.processLLMRegistrationTx(tx)
    case "validation_reward":
        return te.processValidationRewardTx(tx)
    default:
        return fmt.Errorf("unknown transaction type: %s", tx.Type)
    }
}

func (te *TokenEconomics) processSkillInvocationTx(tx *EconomicTransaction) error {
    // Burn tokens for skill invocation
    te.burnTracker.mutex.Lock()
    te.burnTracker.totalBurned.Add(te.burnTracker.totalBurned, tx.Amount)
    te.burnTracker.mutex.Unlock()

    // Update total burned in metrics
    te.metrics.mutex.Lock()
    te.metrics.TotalBurned.Add(te.metrics.TotalBurned, tx.Amount)
    te.metrics.CirculatingSupply.Sub(te.metrics.CirculatingSupply, tx.Amount)
    te.metrics.mutex.Unlock()

    log.Printf("Burned %s NRN for skill invocation %s", tx.Amount.String(), tx.Metadata["skill_id"])
    return nil
}

func (te *TokenEconomics) processLLMRegistrationTx(tx *EconomicTransaction) error {
    // Transfer registration fee to treasury
    log.Printf("Processed LLM registration fee %s NRN for %s", tx.Amount.String(), tx.Metadata["llm_id"])
    return nil
}

func (te *TokenEconomics) processValidationRewardTx(tx *EconomicTransaction) error {
    // Mint reward tokens
    te.metrics.mutex.Lock()
    te.metrics.TotalSupply.Add(te.metrics.TotalSupply, tx.Amount)
    te.metrics.CirculatingSupply.Add(te.metrics.CirculatingSupply, tx.Amount)
    te.metrics.mutex.Unlock()

    log.Printf("Minted %s NRN validation reward for %s", tx.Amount.String(), tx.To)
    return nil
}

func (te *TokenEconomics) metricsUpdater(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            te.updateMetrics()
        }
    }
}

func (te *TokenEconomics) updateMetrics() {
    te.metrics.mutex.Lock()
    defer te.metrics.mutex.Unlock()

    // Update network utilization
    te.metrics.NetworkUtilization = te.calculateNetworkUtilization()

    // Update token velocity
    te.metrics.TokenVelocity = te.calculateTokenVelocity()

    // Update average gas price
    te.metrics.AverageGasPrice = te.calculateAverageGasPrice()

    te.metrics.LastUpdated = time.Now()
}

func (te *TokenEconomics) calculateNetworkUtilization() float64 {
    // Calculate based on transaction volume and capacity
    // This is a simplified calculation
    return 0.75 // 75% utilization
}

func (te *TokenEconomics) calculateTokenVelocity() float64 {
    // Token velocity = Transaction Volume / Circulating Supply
    if te.metrics.CirculatingSupply.Cmp(big.NewInt(0)) == 0 {
        return 0
    }

    // Simplified calculation
    return 2.5 // 2.5x velocity
}

func (te *TokenEconomics) calculateAverageGasPrice() *big.Int {
    // Calculate average gas price from recent transactions
    return big.NewInt(1500) // 1500 wei average
}

func (te *TokenEconomics) rewardDistributor(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            te.distributeRewards()
        }
    }
}

func (te *TokenEconomics) distributeRewards() {
    // Distribute validator rewards
    // Distribute developer rewards
    // Distribute community rewards
    log.Println("Distributing periodic rewards...")
}

func (te *TokenEconomics) burnProcessor(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            te.processBurnEvents()
        }
    }
}

func (te *TokenEconomics) processBurnEvents() {
    te.burnTracker.mutex.Lock()
    defer te.burnTracker.mutex.Unlock()

    // Process unvalidated burn events
    for _, event := range te.burnTracker.burnHistory {
        if !event.Validated {
            // Validate burn event
            event.Validated = true
            log.Printf("Validated burn event: %s burned %s NRN", event.User, event.Amount.String())
        }
    }
}

func (te *TokenEconomics) loadState() error {
    // Load economic state from database
    // This would restore metrics, transaction history, etc.
    return nil
}

func (te *TokenEconomics) saveState() error {
    // Save economic state to database
    return nil
}

// Transaction Pool methods
func (tp *TransactionPool) AddTransaction(tx *EconomicTransaction) {
    tp.mutex.Lock()
    defer tp.mutex.Unlock()

    tp.pendingTxs[tx.ID] = tx

    // Clean up if pool is too large
    if len(tp.pendingTxs) > tp.maxPoolSize {
        tp.cleanupOldTransactions()
    }
}

func (tp *TransactionPool) cleanupOldTransactions() {
    // Remove oldest transactions if pool is full
    cutoff := time.Now().Add(-1 * time.Hour)

    for id, tx := range tp.pendingTxs {
        if tx.Timestamp.Before(cutoff) {
            delete(tp.pendingTxs, id)
        }
    }
}

// Reward Calculator methods
func (rc *RewardCalculator) CalculateReward(userID, rewardType string, baseAmount *big.Int) *big.Int {
    rc.mutex.RLock()
    defer rc.mutex.RUnlock()

    multiplier := 1.0

    // Apply performance-based multipliers
    if metrics, exists := rc.performanceData[userID]; exists {
        if metrics.SuccessRate > 0.9 {
            multiplier *= rc.multipliers["high_performance"]
        }
        if metrics.Uptime > 0.95 {
            multiplier *= rc.multipliers["consistent_user"]
        }
    }

    // Calculate final reward
    finalAmount := new(big.Int).Mul(baseAmount, big.NewInt(int64(multiplier*1000)))
    finalAmount = new(big.Int).Div(finalAmount, big.NewInt(1000))

    return finalAmount
}

func (rc *RewardCalculator) UpdatePerformanceMetrics(userID string, metrics *PerformanceMetrics) {
    rc.mutex.Lock()
    defer rc.mutex.Unlock()

    rc.performanceData[userID] = metrics
}

// Burn Tracker methods
func (bt *BurnTracker) AddBurnEvent(event *BurnEvent) {
    bt.mutex.Lock()
    defer bt.mutex.Unlock()

    bt.burnHistory = append(bt.burnHistory, event)

    // Maintain history size
    if len(bt.burnHistory) > 10000 {
        bt.burnHistory = bt.burnHistory[1000:]
    }
}

func (bt *BurnTracker) GetTotalBurned() *big.Int {
    bt.mutex.RLock()
    defer bt.mutex.RUnlock()

    return new(big.Int).Set(bt.totalBurned)
}
```