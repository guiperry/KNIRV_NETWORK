package api

import (
	"KNIRV_Engine/database"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserIDContextKey is the key used to store user ID in context
	UserIDContextKey ContextKey = "userID"
)

// AuthService handles user authentication and authorization
type AuthService struct {
	userRepo       *database.UserRepository
	tokenRepo      *database.TokenRepository
	permissionRepo *database.PermissionRepository
	jwtSecret      []byte
	jwtExpiration  time.Duration
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo *database.UserRepository,
	tokenRepo *database.TokenRepository,
	permissionRepo *database.PermissionRepository,
	jwtSecret string,
	jwtExpiration time.Duration,
) *AuthService {
	if jwtExpiration == 0 {
		jwtExpiration = 24 * time.Hour // Default to 24 hours
	}

	return &AuthService{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		permissionRepo: permissionRepo,
		jwtSecret:      []byte(jwtSecret),
		jwtExpiration:  jwtExpiration,
	}
}

// RegisterHandlers registers the auth API handlers
func (s *AuthService) RegisterHandlers(router *mux.Router) {
	// Public auth endpoints (no auth middleware required)
	router.HandleFunc("/api/v1/auth/login", s.handleLogin).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/register", s.handleRegister).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/refresh", s.handleRefreshToken).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/logout", s.handleLogout).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/auth/csrf-token", s.handleGetCSRFToken).Methods("GET", "OPTIONS")

	// Protected auth endpoints (require auth middleware)
	router.Handle("/api/v1/auth/tokens", s.AuthMiddleware(http.HandlerFunc(s.handleListTokens))).Methods("GET", "OPTIONS")
	router.Handle("/api/v1/auth/tokens", s.AuthMiddleware(http.HandlerFunc(s.handleCreateToken))).Methods("POST", "OPTIONS")
	router.Handle("/api/v1/auth/tokens/{id}", s.AuthMiddleware(http.HandlerFunc(s.handleDeleteToken))).Methods("DELETE", "OPTIONS")
}

// handleLogin handles the login request
func (s *AuthService) handleLogin(w http.ResponseWriter, r *http.Request) {
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

	if err := parseJSONBody(r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Verify credentials
	valid, err := s.userRepo.VerifyPassword(req.Username, req.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to verify credentials")
		return
	}

	if !valid {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Get user
	user, err := s.userRepo.GetUserByUsername(req.Username)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Return token and user info
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// handleGetCSRFToken handles CSRF token requests
func (s *AuthService) handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers for the main request
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// For now, return a simple token - in production, this would be more sophisticated
	userID := "anonymous"

	// Extract user ID from token if authenticated
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
			if validatedUserID, err := s.validateJWT(tokenParts[1]); err == nil {
				userID = strconv.FormatInt(validatedUserID, 10)
			}
		}
	}

	// Generate CSRF token (simplified - in production, integrate with security middleware)
	token := fmt.Sprintf("csrf_%d_%s", time.Now().UnixNano(), userID)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"csrf_token": token,
		"expires_at": time.Now().Add(1 * time.Hour).Unix(),
	})
}

// handleRegister handles the registration request
func (s *AuthService) handleRegister(w http.ResponseWriter, r *http.Request) {
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
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username, email, and password are required")
		return
	}

	// Create user
	user := &database.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     "user", // Default role
	}

	if err := s.userRepo.CreateUser(user, req.Password); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			respondWithError(w, http.StatusConflict, "Username or email already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Return token and user info
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// handleRefreshToken handles token refresh requests
func (s *AuthService) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
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

	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}

	// Extract token from "Bearer <token>"
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		respondWithError(w, http.StatusUnauthorized, "Invalid Authorization header format")
		return
	}

	tokenString := tokenParts[1]

	// Validate token
	userID, err := s.validateJWT(tokenString)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	// Get user
	user, err := s.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Generate new token
	newToken, err := s.generateJWT(user)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Return new token
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"token": newToken,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// handleLogout handles logout requests
func (s *AuthService) handleLogout(w http.ResponseWriter, r *http.Request) {
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

	// In a stateless JWT system, the client simply discards the token
	// Server-side, we don't need to do anything special

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

// handleListTokens handles listing API tokens for a user
func (s *AuthService) handleListTokens(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers for the main request
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(UserIDContextKey).(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get tokens for user
	tokens, err := s.tokenRepo.ListTokensForUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to list tokens")
		return
	}

	// Mask token values for security
	for _, token := range tokens {
		if len(token.Token) > 8 {
			token.Token = token.Token[:8] + "..." // Show only first 8 characters
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
	})
}

// handleCreateToken handles creating a new API token
func (s *AuthService) handleCreateToken(w http.ResponseWriter, r *http.Request) {
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

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(UserIDContextKey).(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request
	var req struct {
		Description string     `json:"description"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	}

	if err := parseJSONBody(r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	tokenValue := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create token
	token := &database.APIToken{
		UserID:      userID,
		Token:       tokenValue,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := s.tokenRepo.CreateToken(r.Context(), token); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create token")
		return
	}

	// Return the full token value only once
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
	})
}

// handleDeleteToken handles deleting an API token
func (s *AuthService) handleDeleteToken(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set CORS headers for the main request
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value(UserIDContextKey).(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get token ID from URL
	vars := mux.Vars(r)
	tokenID, err := parseID(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid token ID")
		return
	}

	// Get token to verify ownership
	token, err := s.tokenRepo.GetTokenByID(r.Context(), tokenID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Token not found")
		return
	}

	// Verify ownership
	if token.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Delete token
	if err := s.tokenRepo.DeleteToken(r.Context(), tokenID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete token")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Token deleted successfully",
	})
}

// generateJWT generates a JWT token for a user
func (s *AuthService) generateJWT(user *database.User) (string, error) {
	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"name": user.Username,
		"role": user.Role,
		"exp":  time.Now().Add(s.jwtExpiration).Unix(),
	})

	// Sign and return token
	return token.SignedString(s.jwtSecret)
}

// validateJWT validates a JWT token and returns the user ID
func (s *AuthService) validateJWT(tokenString string) (int64, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return s.jwtSecret, nil
	})

	if err != nil {
		return 0, err
	}

	// Validate token claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Check expiration
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return 0, fmt.Errorf("token expired")
			}
		} else {
			return 0, fmt.Errorf("invalid exp claim")
		}

		// Get user ID
		if sub, ok := claims["sub"].(float64); ok {
			return int64(sub), nil
		}

		return 0, fmt.Errorf("invalid sub claim")
	}

	return 0, fmt.Errorf("invalid token")
}

// AuthMiddleware creates a middleware that validates JWT tokens
func (s *AuthService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight request
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Set CORS headers for the main request
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		// Extract token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			respondWithError(w, http.StatusUnauthorized, "Invalid Authorization header format")
			return
		}

		// Validate token
		userID, err := s.validateJWT(tokenParts[1])
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// Set user ID in context
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission creates middleware that checks for a specific permission
func (s *AuthService) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by auth middleware)
			userID, ok := r.Context().Value(UserIDContextKey).(int64)
			if !ok {
				respondWithError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Check permission
			hasPermission, err := s.permissionRepo.HasPermission(r.Context(), userID, permission)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to check permission")
				return
			}

			if !hasPermission {
				respondWithError(w, http.StatusForbidden, "Permission denied")
				return
			}

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions

// parseJSONBody parses the JSON body of a request into the provided struct
func parseJSONBody(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}
