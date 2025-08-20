package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	
	dataengine "KNIRVNEXUS/backend/internal/services/data-engine"
	"KNIRVNEXUS/backend/internal/services/cde"
)

// Health and Status Handlers

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"services": map[string]bool{
			"host_controller":    s.hostController != nil,
			"data_engine":        s.dataEngine != nil && s.dataEngine.IsRunning(),
			"agent_server":       s.agentServer != nil && s.agentServer.IsRunning(),
			"inference_service":  s.inferenceService != nil && s.inferenceService.IsRunning(),
			"cde_service":        s.cdeService != nil && s.cdeService.IsRunning(),
		},
	}
	
	s.writeJSON(w, http.StatusOK, health)
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"api_server": map[string]interface{}{
			"running": s.isRunning,
			"port":    s.port,
		},
		"services": map[string]interface{}{},
	}
	
	// Add service statuses
	if s.hostController != nil {
		status.(map[string]interface{})["services"].(map[string]interface{})["host_controller"] = s.hostController.GetStatus()
	}
	
	if s.dataEngine != nil {
		status.(map[string]interface{})["services"].(map[string]interface{})["data_engine"] = map[string]interface{}{
			"running": s.dataEngine.IsRunning(),
		}
	}
	
	if s.agentServer != nil {
		status.(map[string]interface{})["services"].(map[string]interface{})["agent_server"] = s.agentServer.GetStatus()
	}
	
	if s.inferenceService != nil {
		status.(map[string]interface{})["services"].(map[string]interface{})["inference_service"] = s.inferenceService.GetStatus()
	}
	
	if s.cdeService != nil {
		status.(map[string]interface{})["services"].(map[string]interface{})["cde_service"] = s.cdeService.GetStatus()
	}
	
	s.writeJSON(w, http.StatusOK, status)
}

func (s *APIServer) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"version":     "1.0.0",
		"build_time":  time.Now().Format(time.RFC3339),
		"go_version":  "1.21",
		"platform":    "linux/amd64",
	}
	
	if s.hostController != nil {
		systemInfo := s.hostController.GetSystemInfo()
		info["system"] = systemInfo
	}
	
	s.writeJSON(w, http.StatusOK, info)
}

// Authentication Handlers

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string                 `json:"token"`
	ExpiresAt time.Time              `json:"expires_at"`
	User      *dataengine.UserEntry  `json:"user"`
}

type RegisterRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Get user by username
	user, err := s.dataEngine.GetBuntDBManager().GetUserByUsername(req.Username)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	
	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	
	// Check user status
	if user.Status != "active" {
		s.writeError(w, http.StatusUnauthorized, "Account is not active")
		return
	}
	
	// Generate JWT token
	tokenDuration := 24 * time.Hour
	token, err := s.authMiddleware.GenerateToken(user.ID, user.Username, user.Role, tokenDuration)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	
	// Update last login
	now := time.Now()
	user.LastLogin = &now
	s.dataEngine.GetBuntDBManager().UpdateUser(user)
	
	// Remove password hash from response
	user.PasswordHash = ""
	
	response := LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(tokenDuration),
		User:      user,
	}
	
	s.writeJSON(w, http.StatusOK, response)
}

func (s *APIServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" {
		s.writeError(w, http.StatusBadRequest, "Username, email, and password are required")
		return
	}
	
	// Check if user already exists
	if _, err := s.dataEngine.GetBuntDBManager().GetUserByUsername(req.Username); err == nil {
		s.writeError(w, http.StatusConflict, "Username already exists")
		return
	}
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}
	
	// Create user
	user := &dataengine.UserEntry{
		ID:           "user_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         "user", // Default role
		Status:       "active",
		Metadata:     make(map[string]interface{}),
	}
	
	if err := s.dataEngine.GetBuntDBManager().CreateUser(user); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}
	
	// Remove password hash from response
	user.PasswordHash = ""
	
	s.writeJSON(w, http.StatusCreated, user)
}

func (s *APIServer) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// Extract current token
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	
	// Get user
	user, err := s.dataEngine.GetBuntDBManager().GetUser(authCtx.UserID)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "User not found")
		return
	}
	
	// Revoke old token
	s.authMiddleware.RevokeToken(authCtx.Token)
	
	// Generate new token
	tokenDuration := 24 * time.Hour
	newToken, err := s.authMiddleware.GenerateToken(user.ID, user.Username, user.Role, tokenDuration)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	
	response := LoginResponse{
		Token:     newToken,
		ExpiresAt: time.Now().Add(tokenDuration),
		User:      user,
	}
	
	s.writeJSON(w, http.StatusOK, response)
}

// User Management Handlers

func (s *APIServer) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	
	user, err := s.dataEngine.GetBuntDBManager().GetUser(authCtx.UserID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "User not found")
		return
	}
	
	// Remove password hash
	user.PasswordHash = ""
	
	s.writeJSON(w, http.StatusOK, user)
}

func (s *APIServer) handleUpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	
	user, err := s.dataEngine.GetBuntDBManager().GetUser(authCtx.UserID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "User not found")
		return
	}
	
	var updateReq struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Update fields
	if updateReq.FirstName != "" {
		user.FirstName = updateReq.FirstName
	}
	if updateReq.LastName != "" {
		user.LastName = updateReq.LastName
	}
	if updateReq.Email != "" {
		user.Email = updateReq.Email
	}
	
	if err := s.dataEngine.GetBuntDBManager().UpdateUser(user); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}
	
	// Remove password hash
	user.PasswordHash = ""
	
	s.writeJSON(w, http.StatusOK, user)
}

func (s *APIServer) handleGetUserSessions(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	
	// This would require implementing session tracking in the auth system
	// For now, return empty array
	sessions := []interface{}{}
	
	s.writeJSON(w, http.StatusOK, sessions)
}
