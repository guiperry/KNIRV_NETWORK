package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/config"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handler handles authentication requests
type Handler struct {
	config *config.Config
	logger *zap.Logger
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

// NewHandler creates a new auth handler
func NewHandler(cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		config: cfg,
		logger: logger,
	}
}

// RegisterRoutes registers the auth API routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/auth/login", h.handleLogin).Methods("POST")
	r.HandleFunc("/auth/logout", h.handleLogout).Methods("POST")
	r.HandleFunc("/auth/verify", h.handleVerifyToken).Methods("GET")
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
