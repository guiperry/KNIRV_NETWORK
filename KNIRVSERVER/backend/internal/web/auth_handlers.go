package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/auth"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
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

type RegisterRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required,min=1,max=50"`
	LastName  string `json:"last_name" validate:"required,min=1,max=50"`
	Company   string `json:"company,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Role      string `json:"role,omitempty"`
}

type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type PasswordResetRequestRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type PasswordResetRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

type UpdateProfileRequest struct {
	FirstName      string                 `json:"first_name,omitempty" validate:"omitempty,min=1,max=50"`
	LastName       string                 `json:"last_name,omitempty" validate:"omitempty,min=1,max=50"`
	Company        string                 `json:"company,omitempty"`
	Phone          string                 `json:"phone,omitempty"`
	Timezone       string                 `json:"timezone,omitempty"`
	Language       string                 `json:"language,omitempty"`
	OnboardingData map[string]interface{} `json:"onboarding_data,omitempty"`
}

type UserPreferencesResponse struct {
	OnboardingData map[string]interface{} `json:"onboarding_data,omitempty"`
	UserID         string                 `json:"user_id"`
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Get client IP for rate limiting and audit logging
	clientIP := auth.GetClientIP(r)

	// Create user service
	userService := auth.NewUserService(h.db)

	// Authenticate user
	user, err := userService.AuthenticateUser(req.Username, req.Password, clientIP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	duration := 12 * time.Hour
	token, err := h.auth.GenerateToken(user.ID, user.Username, user.Role, duration)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token:     token,
		Role:      user.Role,
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

// RegisterRoutes registers the authentication routes with the router
func (h *AuthHandlers) RegisterRoutes(r *mux.Router) {
	authRouter := r.PathPrefix("/api/auth").Subrouter()

	// Public routes
	authRouter.HandleFunc("/login", h.Login).Methods("POST")
	authRouter.HandleFunc("/register", h.Register).Methods("POST")
	authRouter.HandleFunc("/refresh", h.Refresh).Methods("POST")
	authRouter.HandleFunc("/verify-email", h.VerifyEmail).Methods("POST")
	authRouter.HandleFunc("/request-password-reset", h.RequestPasswordReset).Methods("POST")
	authRouter.HandleFunc("/reset-password", h.ResetPassword).Methods("POST")

	// Protected routes (require authentication)
	protectedRouter := authRouter.PathPrefix("").Subrouter()
	protectedRouter.Use(h.auth.RequireAuth)
	protectedRouter.HandleFunc("/me", h.Me).Methods("GET")
	protectedRouter.HandleFunc("/change-password", h.ChangePassword).Methods("POST")
	protectedRouter.HandleFunc("/update-profile", h.UpdateProfile).Methods("PUT")
	protectedRouter.HandleFunc("/preferences", h.GetPreferences).Methods("GET")
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

func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Create user service
	userService := auth.NewUserService(h.db)

	// Create registration data
	registration := &objects.UserRegistration{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Company:   req.Company,
		Phone:     req.Phone,
		Role:      req.Role,
	}

	// Create user
	user, err := userService.CreateUser(registration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := RegisterResponse{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
		Message:  "User registered successfully. Please check your email for verification instructions.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Create user service
	userService := auth.NewUserService(h.db)

	// Verify email
	if err := userService.VerifyEmail(req.Token); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Email verified successfully"})
}

func (h *AuthHandlers) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Create user service
	userService := auth.NewUserService(h.db)

	// Initiate password reset
	if err := userService.InitiatePasswordReset(req.Email); err != nil {
		// Don't reveal if email exists or not
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "If the email exists, a reset link has been sent"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "If the email exists, a reset link has been sent"})
}

func (h *AuthHandlers) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Create user service
	userService := auth.NewUserService(h.db)

	// Reset password
	reset := &objects.PasswordReset{
		Token:    req.Token,
		Password: req.Password,
	}

	if err := userService.ResetPassword(reset); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password reset successfully"})
}

func (h *AuthHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Get auth context from middleware
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Create user service
	userService := auth.NewUserService(h.db)

	// Change password
	change := &objects.ChangePassword{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}

	if err := userService.ChangePassword(authCtx.UserID, change); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password changed successfully"})
}

func (h *AuthHandlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get auth context from middleware
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Create user service
	userService := auth.NewUserService(h.db)

	// Update profile
	updates := &objects.UserUpdate{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Company:   req.Company,
		Phone:     req.Phone,
		Timezone:  req.Timezone,
		Language:  req.Language,
	}

	if err := userService.UpdateUser(authCtx.UserID, updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Save onboarding data if provided
	if req.OnboardingData != nil {
		preferencesKey := fmt.Sprintf("users:preferences:%s", authCtx.UserID)
		if err := h.db.StoreJSON(preferencesKey, req.OnboardingData); err != nil {
			log.Printf("Failed to save onboarding data: %v", err)
			// Don't fail the request, just log the error
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile updated successfully"})
}

func (h *AuthHandlers) GetPreferences(w http.ResponseWriter, r *http.Request) {
	// Get auth context from middleware
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Load onboarding data
	preferencesKey := fmt.Sprintf("users:preferences:%s", authCtx.UserID)
	var onboardingData map[string]interface{}

	err := h.db.GetJSON(preferencesKey, &onboardingData)
	if err != nil {
		// No preferences saved yet, return empty
		onboardingData = make(map[string]interface{})
	}

	response := UserPreferencesResponse{
		OnboardingData: onboardingData,
		UserID:         authCtx.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
