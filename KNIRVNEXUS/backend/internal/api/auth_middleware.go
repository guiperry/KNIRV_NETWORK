package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	dataengine "nexus-backend/internal/services/data-engine"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	dataEngine *dataengine.BuntDBDataEngine
	jwtSecret  []byte
}

// UserClaims represents JWT claims for a user
type UserClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// AuthContext represents authentication context
type AuthContext struct {
	UserID   string
	Username string
	Role     string
	Token    string
}

// Context keys
type contextKey string

const (
	AuthContextKey contextKey = "auth_context"
)

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(dataEngine *dataengine.BuntDBDataEngine, jwtSecret string) (*AuthMiddleware, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret cannot be empty")
	}

	return &AuthMiddleware{
		dataEngine: dataEngine,
		jwtSecret:  []byte(jwtSecret),
	}, nil
}

// Authenticate middleware function
func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			am.writeAuthError(w, "Authorization header required")
			return
		}

		// Check for Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			am.writeAuthError(w, "Invalid authorization header format")
			return
		}

		tokenString := parts[1]

		// Parse and validate JWT token
		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return am.jwtSecret, nil
		})

		if err != nil {
			am.writeAuthError(w, "Invalid token")
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*UserClaims)
		if !ok || !token.Valid {
			am.writeAuthError(w, "Invalid token claims")
			return
		}

		// Check token expiration
		if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
			am.writeAuthError(w, "Token expired")
			return
		}

		// Verify token in database
		tokenHash := am.hashToken(tokenString)
		auth, err := am.dataEngine.GetBuntDBManager().GetAuthByToken(tokenHash)
		if err != nil {
			am.writeAuthError(w, "Token not found or revoked")
			return
		}

		// Update last used timestamp
		now := time.Now()
		auth.LastUsed = &now
		am.dataEngine.GetBuntDBManager().CreateAuth(auth) // This will update existing

		// Create auth context
		authCtx := &AuthContext{
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
			Token:    tokenString,
		}

		// Add auth context to request context
		ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware to check user role
func (am *AuthMiddleware) RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				am.writeAuthError(w, "Authentication required")
				return
			}

			if authCtx.Role != requiredRole {
				am.writeAuthError(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GenerateToken generates a JWT token for a user
func (am *AuthMiddleware) GenerateToken(userID, username, role string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := UserClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "knirv-nexus",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(am.jwtSecret)
	if err != nil {
		return "", err
	}

	// Store token in database
	tokenHash := am.hashToken(tokenString)
	auth := &dataengine.AuthEntry{
		ID:        fmt.Sprintf("auth_%d", time.Now().UnixNano()),
		UserID:    userID,
		TokenHash: tokenHash,
		TokenType: "jwt",
		ExpiresAt: now.Add(duration),
		CreatedAt: now,
		Revoked:   false,
		Metadata:  make(map[string]interface{}),
	}

	if err := am.dataEngine.GetBuntDBManager().CreateAuth(auth); err != nil {
		return "", fmt.Errorf("failed to store auth token: %w", err)
	}

	return tokenString, nil
}

// RevokeToken revokes a JWT token
func (am *AuthMiddleware) RevokeToken(tokenString string) error {
	tokenHash := am.hashToken(tokenString)
	auth, err := am.dataEngine.GetBuntDBManager().GetAuthByToken(tokenHash)
	if err != nil {
		return err
	}

	return am.dataEngine.GetBuntDBManager().RevokeAuth(auth.ID)
}

// ValidateToken validates a JWT token without middleware
func (am *AuthMiddleware) ValidateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check token in database
	tokenHash := am.hashToken(tokenString)
	_, err = am.dataEngine.GetBuntDBManager().GetAuthByToken(tokenHash)
	if err != nil {
		return nil, fmt.Errorf("token not found or revoked")
	}

	return claims, nil
}

// hashToken creates a hash of the token for storage
func (am *AuthMiddleware) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// writeAuthError writes an authentication error response
func (am *AuthMiddleware) writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	response := APIResponse{
		Success:   false,
		Error:     message,
		Timestamp: time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetAuthContext extracts auth context from request context
func GetAuthContext(r *http.Request) *AuthContext {
	if ctx := r.Context().Value(AuthContextKey); ctx != nil {
		if authCtx, ok := ctx.(*AuthContext); ok {
			return authCtx
		}
	}
	return nil
}

// RequireAuth is a helper function to check if request is authenticated
func RequireAuth(r *http.Request) (*AuthContext, error) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		return nil, fmt.Errorf("authentication required")
	}
	return authCtx, nil
}

// RequireRole is a helper function to check if user has required role
func RequireRole(r *http.Request, requiredRole string) (*AuthContext, error) {
	authCtx, err := RequireAuth(r)
	if err != nil {
		return nil, err
	}

	if authCtx.Role != requiredRole {
		return nil, fmt.Errorf("insufficient permissions")
	}

	return authCtx, nil
}
