// auth.go
package agentify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AuthProvider defines the interface for authentication providers
type AuthProvider interface {
	// Authenticate authenticates a request
	Authenticate(r *http.Request) (string, error)

	// Authorize authorizes a request
	Authorize(userID string, resource string, action string) error
}

// APIKeyAuthProvider implements AuthProvider using API keys
type APIKeyAuthProvider struct {
	apiKeys map[string]string // API key -> user ID
	mutex   sync.RWMutex
}

// NewAPIKeyAuthProvider creates a new API key authentication provider
func NewAPIKeyAuthProvider() *APIKeyAuthProvider {
	return &APIKeyAuthProvider{
		apiKeys: make(map[string]string),
	}
}

// AddAPIKey adds an API key for a user
func (p *APIKeyAuthProvider) AddAPIKey(apiKey string, userID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.apiKeys[apiKey] = userID
}

// RemoveAPIKey removes an API key
func (p *APIKeyAuthProvider) RemoveAPIKey(apiKey string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.apiKeys, apiKey)
}

// Authenticate authenticates a request using the API key
func (p *APIKeyAuthProvider) Authenticate(r *http.Request) (string, error) {
	// Get the API key from the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	// Check if the header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("invalid Authorization header format")
	}

	// Extract the API key
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")

	// Look up the user ID
	p.mutex.RLock()
	userID, ok := p.apiKeys[apiKey]
	p.mutex.RUnlock()

	if !ok {
		return "", errors.New("invalid API key")
	}

	return userID, nil
}

// Authorize authorizes a request
func (p *APIKeyAuthProvider) Authorize(userID string, resource string, action string) error {
	// In a real implementation, we would check if the user has permission to access the resource
	// For now, we'll just allow all authenticated users to access all resources
	return nil
}

// HMACAuthProvider implements AuthProvider using HMAC signatures
type HMACAuthProvider struct {
	secrets map[string]string // User ID -> secret
	mutex   sync.RWMutex
}

// NewHMACAuthProvider creates a new HMAC authentication provider
func NewHMACAuthProvider() *HMACAuthProvider {
	return &HMACAuthProvider{
		secrets: make(map[string]string),
	}
}

// AddSecret adds a secret for a user
func (p *HMACAuthProvider) AddSecret(userID string, secret string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.secrets[userID] = secret
}

// RemoveSecret removes a secret for a user
func (p *HMACAuthProvider) RemoveSecret(userID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.secrets, userID)
}

// Authenticate authenticates a request using HMAC
func (p *HMACAuthProvider) Authenticate(r *http.Request) (string, error) {
	// Get the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	// Check if the header starts with "HMAC "
	if !strings.HasPrefix(authHeader, "HMAC ") {
		return "", errors.New("invalid Authorization header format")
	}

	// Extract the HMAC parts
	parts := strings.Split(strings.TrimPrefix(authHeader, "HMAC "), ":")
	if len(parts) != 3 {
		return "", errors.New("invalid HMAC format")
	}

	userID := parts[0]
	timestamp := parts[1]
	signature := parts[2]

	// Check if the timestamp is valid
	timestampInt, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "", fmt.Errorf("invalid timestamp: %v", err)
	}

	// Check if the timestamp is within a reasonable time window
	if time.Since(timestampInt) > 5*time.Minute {
		return "", errors.New("timestamp expired")
	}

	// Get the secret for the user
	p.mutex.RLock()
	secret, ok := p.secrets[userID]
	p.mutex.RUnlock()

	if !ok {
		return "", errors.New("invalid user ID")
	}

	// Create the message to sign
	method := r.Method
	path := r.URL.Path
	message := fmt.Sprintf("%s %s %s", method, path, timestamp)

	// Create the HMAC
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare the signatures
	if signature != expectedSignature {
		return "", errors.New("invalid signature")
	}

	return userID, nil
}

// Authorize authorizes a request
func (p *HMACAuthProvider) Authorize(userID string, resource string, action string) error {
	// In a real implementation, we would check if the user has permission to access the resource
	// For now, we'll just allow all authenticated users to access all resources
	return nil
}

// JWTAuthProvider implements AuthProvider using JWT
type JWTAuthProvider struct {
	jwtSecret string
	mutex     sync.RWMutex
}

// NewJWTAuthProvider creates a new JWT authentication provider
func NewJWTAuthProvider(jwtSecret string) *JWTAuthProvider {
	return &JWTAuthProvider{
		jwtSecret: jwtSecret,
	}
}

// SetJWTSecret sets the JWT secret
func (p *JWTAuthProvider) SetJWTSecret(jwtSecret string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.jwtSecret = jwtSecret
}

// Authenticate authenticates a request using JWT
func (p *JWTAuthProvider) Authenticate(r *http.Request) (string, error) {
	// Get the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	// Check if the header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("invalid Authorization header format")
	}

	// Extract the JWT
	jwt := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse the JWT
	// In a real implementation, we would use a JWT library to parse and validate the token
	// For now, we'll just extract the user ID from the token

	// Split the JWT into parts
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid JWT format")
	}

	// Decode the payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	// Extract the user ID from the payload
	// In a real implementation, we would parse the JSON payload
	// For now, we'll just assume the payload is the user ID
	userID := string(payload)

	return userID, nil
}

// Authorize authorizes a request
func (p *JWTAuthProvider) Authorize(userID string, resource string, action string) error {
	// In a real implementation, we would check if the user has permission to access the resource
	// For now, we'll just allow all authenticated users to access all resources
	return nil
}

// AuthMiddleware is a middleware for authenticating and authorizing requests
type AuthMiddleware struct {
	provider AuthProvider
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(provider AuthProvider) *AuthMiddleware {
	return &AuthMiddleware{
		provider: provider,
	}
}

// Middleware returns an http.Handler middleware
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for certain paths
		if r.URL.Path == "/status" {
			next.ServeHTTP(w, r)
			return
		}

		// Authenticate the request
		userID, err := m.provider.Authenticate(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
			return
		}

		// Authorize the request
		if err := m.provider.Authorize(userID, r.URL.Path, r.Method); err != nil {
			http.Error(w, fmt.Sprintf("Authorization failed: %v", err), http.StatusForbidden)
			return
		}

		// Add the user ID to the request context
		ctx := context.WithValue(r.Context(), "userID", userID)

		// Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
