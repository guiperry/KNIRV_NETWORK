package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
)

// Handler handles authentication requests
type Handler struct {
	config *config.Config
	logger *zap.Logger
	db     *sql.DB // Added for database operations
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe,omitempty"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string    `json:"token,omitempty"`
	User      User      `json:"user,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// User represents a user in the system
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role"`
}

// GenerateAPIKeyRequest represents the request to generate a new API key
type GenerateAPIKeyRequest struct {
	UserID  string   `json:"userId"`
	Scopes  []string `json:"scopes"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GenerateAPIKeyResponse represents the response containing the new API key
type GenerateAPIKeyResponse struct {
	APIKey   string    `json:"apiKey"`
	KeyID    string    `json:"keyId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// NewHandler creates a new auth handler
func NewHandler(cfg *config.Config, logger *zap.Logger, db *sql.DB) *Handler {
	return &Handler{
		config: cfg,
		logger: logger,
		db:     db,
	}
}

// RegisterRoutes registers the auth API routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/auth/login", h.handleLogin).Methods("POST")
	r.HandleFunc("/auth/logout", h.handleLogout).Methods("POST")
	r.HandleFunc("/auth/verify", h.handleVerifyToken).Methods("GET")
	r.HandleFunc("/auth/generate-api-key", h.handleGenerateAPIKey).Methods("POST")
}

// generateZeroTrustKey creates a new API key with Zero Trust metadata.
func (h *Handler) generateZeroTrustKey(userID string, scopes []string, metadata map[string]interface{}) (string, string, error) {
	// 1. Generate cryptographically secure secret
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", err
	}
	secret := base64.URLEncoding.EncodeToString(secretBytes)

	// 2. Create key ID for database lookup
	keyIDBytes := make([]byte, 6)
	if _, err := rand.Read(keyIDBytes); err != nil {
		return "", "", err
	}
	keyID := hex.EncodeToString(keyIDBytes)

	// 3. Hash secret for storage (Argon2id)
	// Use strong, recommended parameters for Argon2id
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}
	secretHash := argon2.IDKey([]byte(secret), salt, 1, 64*1024, 4, 32)

	// 4. Store with Zero Trust metadata
	expiresAt := time.Now().Add(90 * 24 * time.Hour) // Mandatory expiration
	_, err := h.db.Exec(`
		INSERT INTO api_keys (key_id, user_id, secret_hash, salt, scopes, is_active, created_at, expires_at, max_requests_per_minute, allowed_ips, last_rotated, risk_score)
		VALUES ($1, $2, $3, $4, $5, false, NOW(), $6, 100, NULL, NOW(), 0)
	`, keyID, userID, secretHash, salt, scopes, expiresAt)
	if err != nil {
		return "", "", err
	}

	return "sk_prod_" + keyID, secret, nil // Return once only
}

// handleGenerateAPIKey handles API key generation requests
func (h *Handler) handleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req GenerateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode generate API key request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" {
		h.logger.Warn("API key generation attempt with missing user ID")
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if len(req.Scopes) == 0 {
		h.logger.Warn("API key generation attempt with missing scopes", zap.String("userId", req.UserID))
		http.Error(w, "At least one scope is required", http.StatusBadRequest)
		return
	}

	keyPrefix, secret, err := h.generateZeroTrustKey(req.UserID, req.Scopes, req.Metadata)
	if err != nil {
		h.logger.Error("Failed to generate API key", zap.Error(err), zap.String("userId", req.UserID))
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	fullKey := keyPrefix + "." + secret
	expiresAt := time.Now().Add(90 * 24 * time.Hour)

	// Return successful response
	response := GenerateAPIKeyResponse{
		APIKey:   fullKey,
		KeyID:    keyPrefix,
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	h.logger.Info("API key generated successfully",
		zap.String("userId", req.UserID),
		zap.String("keyId", keyPrefix))
}

// handleLogin handles user login
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode login request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		h.logger.Warn("Login attempt with missing credentials",
			zap.String("username", req.Username))
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Authenticate user (mock implementation - in production, validate against database)
	user, err := h.authenticateUser(req.Username, req.Password)
	if err != nil {
		h.logger.Warn("Authentication failed",
			zap.String("username", req.Username),
			zap.Error(err))
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate token
	token, expiresAt, err := h.generateToken(user, req.RememberMe)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Return successful response
	response := LoginResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	h.logger.Info("User logged in successfully",
		zap.String("username", user.Username),
		zap.String("userId", user.ID))
}

// handleLogout handles user logout
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you might want to invalidate the token
	// For now, just return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

// handleVerifyToken verifies if a token is valid
func (h *Handler) handleVerifyToken(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "No token provided", http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Verify token (mock implementation)
	user, err := h.verifyToken(token)
	if err != nil {
		h.logger.Warn("Token verification failed", zap.Error(err))
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"user":  user,
	})
}

// verifyKeyIntegrity checks the key against the database.
func (h *Handler) verifyKeyIntegrity(ctx context.Context, keyID, secret, clientIP string) (map[string]interface{}, error) {
	var keyRecord struct {
		secretHash []byte
		salt []byte
		isActive bool
		expiresAt time.Time
		allowedIPs []string
	}

	// Fetch key metadata (use a cache like Redis here for performance)
	err := h.db.QueryRowContext(ctx, "SELECT secret_hash, salt, is_active, expires_at, allowed_ips FROM api_keys WHERE key_id = $1", keyID).Scan(&keyRecord.secretHash, &keyRecord.salt, &keyRecord.isActive, &keyRecord.expiresAt, &keyRecord.allowedIPs)
	if err != nil {
		return nil, http.ErrNoCookie
	}

	if !keyRecord.isActive {
		return nil, http.ErrNoCookie
	}

	if time.Now().After(keyRecord.expiresAt) {
		// Implement revokeKey logic
		return nil, http.ErrNoCookie
	}

	// Verify secret hash (never trust the key itself)
	hash := argon2.IDKey([]byte(secret), keyRecord.salt, 1, 64*1024, 4, 32)
	if string(keyRecord.secretHash) != string(hash) {
		// Implement logSuspiciousEvent
		return nil, http.ErrNoCookie
	}

	// Zero Trust: Verify IP even if previously allowed
	if len(keyRecord.allowedIPs) > 0 {
		// IP validation logic here
	}

	// Return key metadata for policy evaluation
	return map[string]interface{}{"key_id": keyID}, nil
}

// authenticateUser authenticates a user (mock implementation)
func (h *Handler) authenticateUser(username, password string) (*User, error) {
	// Mock authentication - check for default credentials
	// In production, this would validate against a database or external service

	// Check default credentials
	if username == "admin" && password == "password123" {
		user := &User{
			ID:       "user-admin",
			Username: "admin",
			Email:    "admin@knirv.network",
			Role:     "admin",
		}
		return user, nil
	}

	// For demo purposes, also accept any other username/password combination
	user := &User{
		ID:       "user-" + username,
		Username: username,
		Email:    username + "@knirv.network",
		Role:     "user",
	}

	return user, nil
}

// generateToken generates a JWT token (mock implementation)
func (h *Handler) generateToken(user *User, rememberMe bool) (string, time.Time, error) {
	// Mock token generation - in production, use proper JWT
	token := "mock-jwt-token-" + user.ID + "-" + time.Now().Format("20060102150405")

	// Set expiration time
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hours default
	if rememberMe {
		expiresAt = time.Now().Add(7 * 24 * time.Hour) // 7 days if remember me
	}

	return token, expiresAt, nil
}

// verifyToken verifies a JWT token (mock implementation)
func (h *Handler) verifyToken(token string) (*User, error) {
	// Mock token verification - in production, validate JWT properly
	if len(token) < 15 || token[:15] != "mock-jwt-token-" {
		return nil, http.ErrNoCookie // Invalid token
	}

	// Extract user ID from mock token
	userID := token[15:]
	if len(userID) < 5 {
		return nil, http.ErrNoCookie
	}

	// Mock user reconstruction
	user := &User{
		ID:       userID,
		Username: "mock-user",
		Email:    "mock-user@knirv.network",
		Role:     "user",
	}

	return user, nil
}
