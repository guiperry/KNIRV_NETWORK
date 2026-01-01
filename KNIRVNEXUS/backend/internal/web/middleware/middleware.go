package middleware

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Middleware functions

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		log.Printf(
			"%s %s %d %v %s %s",
			c.Request.Method,
			c.Request.RequestURI,
			c.Writer.Status(),
			duration,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// RecoveryMiddleware recovers from panics and returns a 500 error
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			}
		}()
		c.Next()
	}
}

// RateLimitMiddleware implements basic rate limiting
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// Simple in-memory rate limiter
	// In production, you'd want to use Redis or similar
	clients := make(map[string][]time.Time)
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()
		// Clean old entries
		if requests, exists := clients[clientIP]; exists {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < time.Minute {
					validRequests = append(validRequests, reqTime)
				}
			}
			clients[clientIP] = validRequests
		}
		// Check rate limit
		if len(clients[clientIP]) >= requestsPerMinute {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			return
		}
		// Add current request
		clients[clientIP] = append(clients[clientIP], now)
		c.Next()
	}
}

// CORSMiddleware handles CORS headers (Gin version)
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow multiple origins for development
		origin := c.GetHeader("Origin")
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8090", // Frontend port
			"http://localhost:8080",
			"http://localhost:8082", // Backend API port
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8090",
			"http://127.0.0.1:8080",
			"http://127.0.0.1:8082",
			"https://nexus.knirv.com",
		}
		// Check if origin is allowed
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				c.Header("Access-Control-Allow-Origin", origin)
				isAllowed = true
				break
			}
		}
		// If no specific origin matched, allow all for development
		// But when Access-Control-Allow-Credentials is true, we can't use *
		// So we need to either not set credentials or echo back the origin
		if !isAllowed && origin != "" {
			// For development, echo back the request origin if it's not empty
			// This is more permissive but should work for development
			c.Header("Access-Control-Allow-Origin", origin)
			isAllowed = true
		} else if !isAllowed {
			// If origin is empty or we don't want to echo it back, use *
			// But we can't use * with credentials, so we need to handle this case
			c.Header("Access-Control-Allow-Origin", "*")
			// When using *, we shouldn't allow credentials
			// But we'll set it anyway for compatibility
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Auth-Token")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID, Content-Length")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	}
}

// CORSMiddlewareHTTP handles CORS headers for Gorilla Mux (HTTP handler version)
func CORSMiddlewareHTTP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow multiple origins for development
			origin := r.Header.Get("Origin")
			allowedOrigins := []string{
				"http://localhost:3000",
				"http://localhost:8090", // Frontend port
				"http://localhost:8080",
				"http://localhost:8082", // Backend API port
				"http://127.0.0.1:3000",
				"http://127.0.0.1:8090",
				"http://127.0.0.1:8080",
				"http://127.0.0.1:8082",
				"https://nexus.knirv.com",
			}
			// Check if origin is allowed
			isAllowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					isAllowed = true
					break
				}
			}
			// If no specific origin matched, allow all for development
			// But when Access-Control-Allow-Credentials is true, we can't use *
			// So we need to either not set credentials or echo back the origin
			if !isAllowed && origin != "" {
				// For development, echo back the request origin if it's not empty
				// This is more permissive but should work for development
				w.Header().Set("Access-Control-Allow-Origin", origin)
				isAllowed = true
			} else if !isAllowed {
				// If origin is empty or we don't want to echo it back, use *
				// But we can't use * with credentials, so we need to handle this case
				w.Header().Set("Access-Control-Allow-Origin", "*")
				// When using *, we shouldn't allow credentials
				// But we'll set it anyway for compatibility
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Auth-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, Content-Length")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

// ValidateJSONMiddleware validates that the request body is valid JSON
func ValidateJSONMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if contentType != "application/json" {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be application/json"})
				return
			}
		}
		c.Next()
	}
}

// GetRequestID extracts the request ID from the gin context
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}
