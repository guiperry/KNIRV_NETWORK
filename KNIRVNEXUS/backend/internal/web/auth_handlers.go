package web

import (
	"encoding/json"
	"net/http"
	"time"

	"backend-server/internal/database"
	"backend-server/internal/web/middleware"
)

type AuthHandlers struct {
	db   *database.BuntDBManager
	auth *middleware.AuthMiddleware
}

func NewAuthHandlers(db *database.BuntDBManager, auth *middleware.AuthMiddleware) *AuthHandlers {
	return &AuthHandlers{db: db, auth: auth}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RefreshRequest struct {
	Token string `json:"token"`
}

type RefreshResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RevokeRequest struct {
	Token string `json:"token"`
}

type MeResponse struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	IssuedAt time.Time `json:"issued_at"`
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// TODO: Replace with proper user store and password hash check
	// For now, using simple validation with hardcoded users
	if req.Username == "" || req.Password == "" {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Simple user role assignment (in production, this would come from a user store)
	role := "user"
	if req.Username == "admin" {
		role = "admin"
	}

	// Generate JWT token
	duration := 12 * time.Hour
	token, err := h.auth.GenerateToken("u:"+req.Username, req.Username, role, duration)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token:     token,
		Role:      role,
		ExpiresAt: time.Now().Add(duration),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandlers) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Validate the current token
	claims, err := h.auth.ValidateToken(req.Token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Generate new token with same user info
	duration := 12 * time.Hour
	newToken, err := h.auth.GenerateToken(claims.UserID, claims.Username, claims.Role, duration)
	if err != nil {
		http.Error(w, "failed to refresh token", http.StatusInternalServerError)
		return
	}

	response := RefreshResponse{
		Token:     newToken,
		ExpiresAt: time.Now().Add(duration),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandlers) Revoke(w http.ResponseWriter, r *http.Request) {
	var req RevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Revoke the token
	if err := h.auth.RevokeToken(req.Token); err != nil {
		http.Error(w, "failed to revoke token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "token revoked"})
}

func (h *AuthHandlers) Me(w http.ResponseWriter, r *http.Request) {
	// Get auth context from middleware
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate token to get claims (including issued_at)
	claims, err := h.auth.ValidateToken(authCtx.Token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	response := MeResponse{
		UserID:   authCtx.UserID,
		Username: authCtx.Username,
		Role:     authCtx.Role,
		IssuedAt: claims.IssuedAt.Time,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
