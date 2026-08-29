package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

// AuthServiceInterface defines the interface for authentication services
type AuthServiceInterface interface {
	RegisterHandlers(router *mux.Router)
}

// SimpleAuthService provides basic authentication for development
type SimpleAuthService struct {
	jwtSecret []byte
}

// SimpleUser represents a simple user for development
type SimpleUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// NewSimpleAuthService creates a new simple auth service for development
func NewSimpleAuthService() *SimpleAuthService {
	return &SimpleAuthService{
		jwtSecret: []byte("dev-secret-key-change-in-production"),
	}
}

// RegisterHandlers registers the auth API handlers
func (s *SimpleAuthService) RegisterHandlers(router *mux.Router) {
	// Public auth endpoints
	router.HandleFunc("/api/v1/auth/login", s.handleLogin).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/register", s.handleRegister).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/refresh", s.handleRefreshToken).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/logout", s.handleLogout).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/csrf-token", s.handleGetCSRFToken).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/v1/auth/verify", s.handleVerify).Methods("GET", "OPTIONS")
}

// handleLogin handles the login request
func (s *SimpleAuthService) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers for the main request
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Parse request
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Simple authentication for development
	// In production, this would verify against a real database
	user := s.authenticateUser(req.Username, req.Password)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Return token and user info
	response := map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// handleRegister handles user registration (simplified for development)
func (s *SimpleAuthService) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// For development, just return success
	response := map[string]interface{}{
		"message": "Registration successful (development mode)",
		"user": map[string]interface{}{
			"id":       1,
			"username": "newuser",
			"email":    "newuser@example.com",
			"role":     "user",
		},
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleRefreshToken handles token refresh
func (s *SimpleAuthService) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// For development, just return a new token
	user := &SimpleUser{ID: 1, Username: "user", Email: "user@example.com", Role: "user"}
	token, _ := s.generateJWT(user)

	response := map[string]interface{}{
		"token": token,
	}

	json.NewEncoder(w).Encode(response)
}

// handleLogout handles logout
func (s *SimpleAuthService) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"message": "Logged out successfully",
	}

	json.NewEncoder(w).Encode(response)
}

// handleVerify confirms that a locally-issued session token is still valid.
func (s *SimpleAuthService) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	const bearerPrefix = "Bearer "
	authorization := r.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, bearerPrefix) {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(strings.TrimPrefix(authorization, bearerPrefix), claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]bool{"valid": true})
}

// handleGetCSRFToken handles CSRF token generation
func (s *SimpleAuthService) handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"csrf_token": "dev-csrf-token",
	}

	json.NewEncoder(w).Encode(response)
}

// authenticateUser performs simple authentication for development
func (s *SimpleAuthService) authenticateUser(username, password string) *SimpleUser {
	// Simple hardcoded users for development
	users := map[string]map[string]interface{}{
		"admin": {"password": "admin", "role": "admin", "email": "admin@example.com"},
		"user":  {"password": "user", "role": "user", "email": "user@example.com"},
		"demo":  {"password": "demo", "role": "user", "email": "demo@example.com"},
	}

	if userInfo, exists := users[username]; exists {
		if userInfo["password"] == password {
			return &SimpleUser{
				ID:       1,
				Username: username,
				Email:    userInfo["email"].(string),
				Role:     userInfo["role"].(string),
			}
		}
	}

	return nil
}

// generateJWT generates a JWT token for a user
func (s *SimpleAuthService) generateJWT(user *SimpleUser) (string, error) {
	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"name": user.Username,
		"role": user.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	})

	// Sign and return token
	return token.SignedString(s.jwtSecret)
}
