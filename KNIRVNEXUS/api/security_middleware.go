// security_middleware.go - Enhanced Security Middleware for Production
package api

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SecurityMiddleware provides comprehensive security controls
type SecurityMiddleware struct {
	rateLimiter    *RateLimiter
	inputValidator *InputValidator
	auditLogger    *SecurityAuditLogger
	csrfProtection *CSRFProtection
	enabled        bool
	mutex          sync.RWMutex
}

// RateLimiter implements rate limiting per IP and per user
type RateLimiter struct {
	ipLimiters    map[string]*rate.Limiter
	userLimiters  map[string]*rate.Limiter
	ipLimit       rate.Limit
	userLimit     rate.Limit
	burst         int
	mutex         sync.RWMutex
	cleanupTicker *time.Ticker
	stopChan      chan bool
}

// InputValidator validates and sanitizes input
type InputValidator struct {
	maxRequestSize  int64
	allowedMethods  []string
	blockedPatterns []*regexp.Regexp
	sanitizeHTML    bool
}

// SecurityAuditLogger logs security events
type SecurityAuditLogger struct {
	events    []SecurityEvent
	maxEvents int
	mutex     sync.RWMutex
}

// CSRFProtection provides CSRF protection
type CSRFProtection struct {
	tokenStore map[string]CSRFToken
	secret     []byte
	mutex      sync.RWMutex
}

// SecurityEvent represents a security event
type SecurityEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	UserID    string                 `json:"user_id"`
	Method    string                 `json:"method"`
	Path      string                 `json:"path"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"timestamp"`
	Blocked   bool                   `json:"blocked"`
}

// CSRFToken represents a CSRF token
type CSRFToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware() *SecurityMiddleware {
	return &SecurityMiddleware{
		rateLimiter:    NewRateLimiter(100, 1000, 10), // 100 req/min per IP, 1000 per user, burst 10
		inputValidator: NewInputValidator(),
		auditLogger:    NewSecurityAuditLogger(10000),
		csrfProtection: NewCSRFProtection(),
		enabled:        true,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(ipLimit, userLimit rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		ipLimiters:   make(map[string]*rate.Limiter),
		userLimiters: make(map[string]*rate.Limiter),
		ipLimit:      ipLimit,
		userLimit:    userLimit,
		burst:        burst,
		stopChan:     make(chan bool),
	}

	// Start cleanup goroutine
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go rl.cleanup()

	return rl
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	// Common attack patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)union\s+select`),
		regexp.MustCompile(`(?i)drop\s+table`),
		regexp.MustCompile(`(?i)insert\s+into`),
		regexp.MustCompile(`(?i)delete\s+from`),
		regexp.MustCompile(`\.\./`), // Path traversal
		regexp.MustCompile(`\x00`),  // Null bytes
	}

	return &InputValidator{
		maxRequestSize:  10 * 1024 * 1024, // 10MB
		allowedMethods:  []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		blockedPatterns: patterns,
		sanitizeHTML:    true,
	}
}

// NewSecurityAuditLogger creates a new security audit logger
func NewSecurityAuditLogger(maxEvents int) *SecurityAuditLogger {
	return &SecurityAuditLogger{
		events:    make([]SecurityEvent, 0),
		maxEvents: maxEvents,
	}
}

// NewCSRFProtection creates a new CSRF protection
func NewCSRFProtection() *CSRFProtection {
	return &CSRFProtection{
		tokenStore: make(map[string]CSRFToken),
		secret:     []byte("csrf-secret-key"), // In production, use a secure random key
	}
}

// Middleware returns the security middleware handler
func (sm *SecurityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sm.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Set security headers
		sm.setSecurityHeaders(w)

		// Get client info
		clientIP := sm.getClientIP(r)
		userAgent := r.UserAgent()
		userID := sm.getUserID(r)

		// Rate limiting
		if !sm.rateLimiter.Allow(clientIP, userID) {
			sm.logSecurityEvent("rate_limit_exceeded", "medium", clientIP, userAgent, userID, r.Method, r.URL.Path, map[string]interface{}{
				"limit_type": "rate_limit",
			}, true)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Input validation (skip for WebSocket upgrades and health endpoints)
		if !sm.isWebSocketUpgrade(r) && !sm.isExemptFromInputValidation(r.URL.Path) {
			if err := sm.inputValidator.ValidateRequest(r); err != nil {
				sm.logSecurityEvent("input_validation_failed", "high", clientIP, userAgent, userID, r.Method, r.URL.Path, map[string]interface{}{
					"validation_error": err.Error(),
				}, true)
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
		}

		// CSRF protection for state-changing operations (exclude auth endpoints and WebSocket)
		if sm.isStateChangingOperation(r.Method) && !sm.isExemptFromCSRF(r.URL.Path) {
			if err := sm.csrfProtection.ValidateToken(r, userID); err != nil {
				sm.logSecurityEvent("csrf_validation_failed", "high", clientIP, userAgent, userID, r.Method, r.URL.Path, map[string]interface{}{
					"csrf_error": err.Error(),
				}, true)
				http.Error(w, "CSRF token validation failed", http.StatusForbidden)
				return
			}
		}

		// Log successful request
		sm.logSecurityEvent("request_processed", "info", clientIP, userAgent, userID, r.Method, r.URL.Path, nil, false)

		next.ServeHTTP(w, r)
	})
}

// setSecurityHeaders sets security headers
func (sm *SecurityMiddleware) setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
}

// getClientIP extracts the client IP address
func (sm *SecurityMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// getUserID extracts user ID from request context
func (sm *SecurityMiddleware) getUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(UserIDContextKey).(int64); ok {
		return strconv.FormatInt(userID, 10)
	}
	return "anonymous"
}

// isStateChangingOperation checks if the HTTP method changes state
func (sm *SecurityMiddleware) isStateChangingOperation(method string) bool {
	return method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH"
}

// isExemptFromCSRF checks if a path is exempt from CSRF protection
func (sm *SecurityMiddleware) isExemptFromCSRF(path string) bool {
	exemptPaths := []string{
		// Authentication endpoints
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout",
		"/api/v1/auth/csrf-token",
		"/api/v1/auth/tokens",
		"/api/v1/authkit-token",

		// WebSocket endpoints
		"/api/v1/ws",
		"/api/v1/desktop/secure-ws",

		// System endpoints
		"/api/shutdown",
		"/health",
		"/status",
		"/api/v1/health",

		// Agent endpoints (legacy)
		"/api/v1/agents/activate",
		"/api/v1/agents",
		"/api/v1/agents/discover",
		"/api/v1/agents/message",
		"/api/v1/agents/capabilities",

		// MCP endpoints (specific)
		"/api/v1/mcp/servers",
		"/api/v1/mcp/registry",
		"/api/v1/mcp/capabilities",
		"/api/v1/mcp/metrics",
		"/api/v1/mcp/logs",
		"/api/v1/mcp/alerts",
		"/api/v1/mcp/configs",

		// Capabilities endpoints
		"/api/v1/capabilities",
		"/api/v1/capabilities/mcp",
	}

	// Check for exact matches first
	for _, exemptPath := range exemptPaths {
		if path == exemptPath {
			return true
		}
	}

	// Check for path prefixes that should be exempt (for desktop operations)
	exemptPrefixes := []string{
		// Agent Development Kit (ADK) endpoints
		"/api/v1/adk/",

		// Terminal endpoints
		"/api/v1/terminal/",

		// Agent endpoints (all agent operations)
		"/api/v1/agents/",

		// Plugin endpoints (all plugin operations)
		"/api/v1/plugins/",

		// MCP (Model Context Protocol) endpoints
		"/api/v1/mcp/",

		// Inference endpoints
		"/api/v1/inference/",

		"/api/v1/debug/",

		// Authentication endpoints
		"/api/v1/auth/",

		// Settings endpoints
		"/api/v1/settings/",

		// Analytics endpoints
		"/api/v1/analytics/",

		// Target system endpoints
		"/api/v1/targets/",

		// User management endpoints
		"/api/v1/users/",

		// Web connections endpoints
		"/api/v1/web-connections/",

		// Workflow endpoints
		"/api/v1/workflows/",

		// Capabilities endpoints
		"/api/v1/capabilities/",
	}

	for _, prefix := range exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// isExemptFromInputValidation checks if a path is exempt from input validation
func (sm *SecurityMiddleware) isExemptFromInputValidation(path string) bool {
	exemptPaths := []string{
		// Health and status endpoints
		"/health",
		"/status",
		"/api/v1/health",
		"/api/v1/status",

		// Authentication endpoints (need to allow login requests)
		"/api/v1/auth/login",
		"/api/v1/auth/logout",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
		"/api/v1/auth/verify",

		// WebSocket endpoints (already handled separately)
		"/api/v1/ws",
		"/api/v1/desktop/secure-ws",

		// System endpoints that need minimal validation
		"/api/shutdown",
	}

	// Check for exact matches
	for _, exemptPath := range exemptPaths {
		if path == exemptPath {
			return true
		}
	}

	return false
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade
func (sm *SecurityMiddleware) isWebSocketUpgrade(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// Allow checks if a request is allowed by rate limiting
func (rl *RateLimiter) Allow(clientIP, userID string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Check IP-based rate limit
	ipLimiter, exists := rl.ipLimiters[clientIP]
	if !exists {
		ipLimiter = rate.NewLimiter(rl.ipLimit, rl.burst)
		rl.ipLimiters[clientIP] = ipLimiter
	}

	if !ipLimiter.Allow() {
		return false
	}

	// Check user-based rate limit (if user is authenticated)
	if userID != "anonymous" {
		userLimiter, exists := rl.userLimiters[userID]
		if !exists {
			userLimiter = rate.NewLimiter(rl.userLimit, rl.burst)
			rl.userLimiters[userID] = userLimiter
		}

		if !userLimiter.Allow() {
			return false
		}
	}

	return true
}

// cleanup removes old rate limiters
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mutex.Lock()
			// Remove limiters that haven't been used recently
			// This is a simplified cleanup - in production, track last access time
			if len(rl.ipLimiters) > 1000 {
				rl.ipLimiters = make(map[string]*rate.Limiter)
			}
			if len(rl.userLimiters) > 1000 {
				rl.userLimiters = make(map[string]*rate.Limiter)
			}
			rl.mutex.Unlock()
		case <-rl.stopChan:
			rl.cleanupTicker.Stop()
			return
		}
	}
}

// ValidateRequest validates an HTTP request
func (iv *InputValidator) ValidateRequest(r *http.Request) error {
	// Check request size
	if r.ContentLength > iv.maxRequestSize {
		return fmt.Errorf("request size %d exceeds maximum %d", r.ContentLength, iv.maxRequestSize)
	}

	// Check HTTP method
	methodAllowed := false
	for _, method := range iv.allowedMethods {
		if r.Method == method {
			methodAllowed = true
			break
		}
	}
	if !methodAllowed {
		return fmt.Errorf("HTTP method %s not allowed", r.Method)
	}

	// Validate URL path
	if err := iv.validatePath(r.URL.Path); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Validate query parameters
	for key, values := range r.URL.Query() {
		for _, value := range values {
			if err := iv.validateInput(value); err != nil {
				return fmt.Errorf("invalid query parameter %s: %w", key, err)
			}
		}
	}

	// Validate headers
	for key, values := range r.Header {
		for _, value := range values {
			if err := iv.validateInput(value); err != nil {
				return fmt.Errorf("invalid header %s: %w", key, err)
			}
		}
	}

	return nil
}

// validatePath validates URL path
func (iv *InputValidator) validatePath(path string) error {
	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected")
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("null byte detected")
	}

	return nil
}

// validateInput validates input against blocked patterns
func (iv *InputValidator) validateInput(input string) error {
	for _, pattern := range iv.blockedPatterns {
		if pattern.MatchString(input) {
			return fmt.Errorf("blocked pattern detected: %s", pattern.String())
		}
	}
	return nil
}

// ValidateToken validates CSRF token
func (cp *CSRFProtection) ValidateToken(r *http.Request, userID string) error {
	token := r.Header.Get("X-CSRF-Token")
	if token == "" {
		token = r.FormValue("csrf_token")
	}

	if token == "" {
		return fmt.Errorf("CSRF token missing")
	}

	cp.mutex.RLock()
	storedToken, exists := cp.tokenStore[userID]
	cp.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no CSRF token found for user")
	}

	if storedToken.Token != token {
		return fmt.Errorf("CSRF token mismatch")
	}

	if time.Now().After(storedToken.ExpiresAt) {
		cp.mutex.Lock()
		delete(cp.tokenStore, userID)
		cp.mutex.Unlock()
		return fmt.Errorf("CSRF token expired")
	}

	return nil
}

// GenerateToken generates a new CSRF token for a user
func (cp *CSRFProtection) GenerateToken(userID string) string {
	token := fmt.Sprintf("csrf_%d_%s", time.Now().UnixNano(), userID)

	cp.mutex.Lock()
	cp.tokenStore[userID] = CSRFToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	cp.mutex.Unlock()

	return token
}

// logSecurityEvent logs a security event
func (sm *SecurityMiddleware) logSecurityEvent(eventType, severity, ipAddress, userAgent, userID, method, path string, details map[string]interface{}, blocked bool) {
	event := SecurityEvent{
		ID:        fmt.Sprintf("sec_%d", time.Now().UnixNano()),
		Type:      eventType,
		Severity:  severity,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		UserID:    userID,
		Method:    method,
		Path:      path,
		Details:   details,
		Timestamp: time.Now(),
		Blocked:   blocked,
	}

	sm.auditLogger.LogEvent(event)

	// Log to system log for high severity events
	if severity == "high" || severity == "critical" {
		log.Printf("SECURITY ALERT [%s]: %s from %s (User: %s, Path: %s %s, Blocked: %t)",
			severity, eventType, ipAddress, userID, method, path, blocked)
	}
}

// LogEvent logs a security event
func (sal *SecurityAuditLogger) LogEvent(event SecurityEvent) {
	sal.mutex.Lock()
	defer sal.mutex.Unlock()

	sal.events = append(sal.events, event)

	// Keep only the last maxEvents events
	if len(sal.events) > sal.maxEvents {
		sal.events = sal.events[len(sal.events)-sal.maxEvents:]
	}
}

// GetEvents returns recent security events
func (sal *SecurityAuditLogger) GetEvents(limit int) []SecurityEvent {
	sal.mutex.RLock()
	defer sal.mutex.RUnlock()

	if limit <= 0 || limit > len(sal.events) {
		limit = len(sal.events)
	}

	// Return the most recent events
	start := len(sal.events) - limit
	return sal.events[start:]
}

// GetEventsByType returns events of a specific type
func (sal *SecurityAuditLogger) GetEventsByType(eventType string, limit int) []SecurityEvent {
	sal.mutex.RLock()
	defer sal.mutex.RUnlock()

	var filtered []SecurityEvent
	for i := len(sal.events) - 1; i >= 0 && len(filtered) < limit; i-- {
		if sal.events[i].Type == eventType {
			filtered = append(filtered, sal.events[i])
		}
	}

	return filtered
}

// Enable enables the security middleware
func (sm *SecurityMiddleware) Enable() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.enabled = true
	log.Println("Security middleware enabled")
}

// Disable disables the security middleware
func (sm *SecurityMiddleware) Disable() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.enabled = false
	log.Println("Security middleware disabled")
}

// GetStats returns security statistics
func (sm *SecurityMiddleware) GetStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	recentEvents := sm.auditLogger.GetEvents(100)
	blockedCount := 0
	eventTypes := make(map[string]int)

	for _, event := range recentEvents {
		if event.Blocked {
			blockedCount++
		}
		eventTypes[event.Type]++
	}

	return map[string]interface{}{
		"enabled":          sm.enabled,
		"total_events":     len(sm.auditLogger.events),
		"recent_events":    len(recentEvents),
		"blocked_requests": blockedCount,
		"event_types":      eventTypes,
		"ip_limiters":      len(sm.rateLimiter.ipLimiters),
		"user_limiters":    len(sm.rateLimiter.userLimiters),
		"csrf_tokens":      len(sm.csrfProtection.tokenStore),
	}
}

// Stop stops the security middleware
func (sm *SecurityMiddleware) Stop() {
	close(sm.rateLimiter.stopChan)
}
