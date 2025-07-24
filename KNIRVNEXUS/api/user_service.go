package api

import (
	"Agentic_Engine/database"

	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// UserService handles user management
type UserService struct {
	userRepo       *database.UserRepository
	permissionRepo *database.PermissionRepository
	roleRepo       *database.RoleRepository
}

// NewUserService creates a new user service
func NewUserService(
	userRepo *database.UserRepository,
	permissionRepo *database.PermissionRepository,
	roleRepo *database.RoleRepository,
) *UserService {
	return &UserService{
		userRepo:       userRepo,
		permissionRepo: permissionRepo,
		roleRepo:       roleRepo,
	}
}

// RegisterHandlers registers the user API handlers
func (s *UserService) RegisterHandlers(router *mux.Router, authService *AuthService) {
	// Protected routes
	protected := router.PathPrefix("/api/v1/users").Subrouter()
	protected.Use(authService.AuthMiddleware)

	// User management (admin only)
	protected.Handle("", authService.RequirePermission("user:read")(http.HandlerFunc(s.handleListUsers))).Methods("GET")
	protected.Handle("/{id}", authService.RequirePermission("user:read")(http.HandlerFunc(s.handleGetUser))).Methods("GET")
	protected.Handle("", authService.RequirePermission("user:create")(http.HandlerFunc(s.handleCreateUser))).Methods("POST")
	protected.Handle("/{id}", authService.RequirePermission("user:update")(http.HandlerFunc(s.handleUpdateUser))).Methods("PUT")
	protected.Handle("/{id}", authService.RequirePermission("user:delete")(http.HandlerFunc(s.handleDeleteUser))).Methods("DELETE")

	// Current user profile (any authenticated user)
	protected.HandleFunc("/profile", s.handleGetProfile).Methods("GET")
	protected.HandleFunc("/profile", s.handleUpdateProfile).Methods("PUT")
	protected.HandleFunc("/profile/password", s.handleChangePassword).Methods("PUT")
}

// handleListUsers handles GET /api/v1/users
func (s *UserService) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// List users
	users, err := s.userRepo.ListUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

// handleGetUser handles GET /api/v1/users/{id}
func (s *UserService) handleGetUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get user
	user, err := s.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// handleCreateUser handles POST /api/v1/users
func (s *UserService) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username, email, and password are required")
		return
	}

	// Validate role
	if req.Role == "" {
		req.Role = "user" // Default role
	} else {
		// Check if role exists
		_, err := s.roleRepo.GetRoleByName(r.Context(), req.Role)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid role")
			return
		}
	}

	// Create user
	user := &database.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
	}

	if err := s.userRepo.CreateUser(user, req.Password); err != nil {
		if err.Error() == "UNIQUE constraint failed: users.username" {
			respondWithError(w, http.StatusConflict, "Username already exists")
			return
		}
		if err.Error() == "UNIQUE constraint failed: users.email" {
			respondWithError(w, http.StatusConflict, "Email already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"user": user,
	})
}

// handleUpdateUser handles PUT /api/v1/users/{id}
func (s *UserService) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get existing user
	existingUser, err := s.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Parse request
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update user fields
	if req.Username != "" {
		existingUser.Username = req.Username
	}

	if req.Email != "" {
		existingUser.Email = req.Email
	}

	if req.Role != "" {
		// Check if role exists
		_, err := s.roleRepo.GetRoleByName(r.Context(), req.Role)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid role")
			return
		}
		existingUser.Role = req.Role
	}

	// Update user
	if err := s.userRepo.UpdateUser(existingUser); err != nil {
		if err.Error() == "UNIQUE constraint failed: users.username" {
			respondWithError(w, http.StatusConflict, "Username already exists")
			return
		}
		if err.Error() == "UNIQUE constraint failed: users.email" {
			respondWithError(w, http.StatusConflict, "Email already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"user": existingUser,
	})
}

// handleDeleteUser handles DELETE /api/v1/users/{id}
func (s *UserService) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get current user ID from context
	currentUserID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Prevent self-deletion
	if userID == currentUserID {
		respondWithError(w, http.StatusBadRequest, "Cannot delete your own account")
		return
	}

	// Delete user
	if err := s.userRepo.DeleteUser(userID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User deleted successfully",
	})
}

// handleGetProfile handles GET /api/v1/users/profile
func (s *UserService) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get user
	user, err := s.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get user profile")
		return
	}

	// Get user permissions
	permissions, err := s.permissionRepo.GetPermissionsForUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get user permissions")
		return
	}

	// Extract permission names
	permissionNames := make([]string, len(permissions))
	for i, p := range permissions {
		permissionNames[i] = p.Name
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"profile": map[string]interface{}{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"role":        user.Role,
			"permissions": permissionNames,
			"created_at":  user.CreatedAt,
		},
	})
}

// handleUpdateProfile handles PUT /api/v1/users/profile
func (s *UserService) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get existing user
	existingUser, err := s.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get user profile")
		return
	}

	// Parse request
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Update user fields
	if req.Username != "" {
		existingUser.Username = req.Username
	}

	if req.Email != "" {
		existingUser.Email = req.Email
	}

	// Update user
	if err := s.userRepo.UpdateUser(existingUser); err != nil {
		if err.Error() == "UNIQUE constraint failed: users.username" {
			respondWithError(w, http.StatusConflict, "Username already exists")
			return
		}
		if err.Error() == "UNIQUE constraint failed: users.email" {
			respondWithError(w, http.StatusConflict, "Email already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"profile": map[string]interface{}{
			"id":         existingUser.ID,
			"username":   existingUser.Username,
			"email":      existingUser.Email,
			"role":       existingUser.Role,
			"created_at": existingUser.CreatedAt,
		},
	})
}

// handleChangePassword handles PUT /api/v1/users/profile/password
func (s *UserService) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate input
	if req.CurrentPassword == "" || req.NewPassword == "" {
		respondWithError(w, http.StatusBadRequest, "Current password and new password are required")
		return
	}

	// Get user
	user, err := s.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}

	// Verify current password
	valid, err := s.userRepo.VerifyPassword(user.Username, req.CurrentPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to verify password")
		return
	}

	if !valid {
		respondWithError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Update password
	if err := s.userRepo.UpdatePassword(userID, req.NewPassword); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Password updated successfully",
	})
}
