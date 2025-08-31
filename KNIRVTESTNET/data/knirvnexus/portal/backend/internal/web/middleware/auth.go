package middleware

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
	"github.com/tidwall/buntdb"

	"nexus-backend/internal/database"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	db        *database.BuntDBManager
	jwtSecret []byte
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
func NewAuthMiddleware(db *database.BuntDBManager, jwtSecret string) (*AuthMiddleware, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret cannot be empty")
	}

	return &AuthMiddleware{
		db:        db,
		jwtSecret: []byte(jwtSecret),
	}, nil
}

// GenerateToken generates a JWT token for a user
func (am *AuthMiddleware) GenerateToken(userID, username, role string, duration time.Duration) (string, error) {
	claims := UserClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "knirv-nexus",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(am.jwtSecret)
}

// ValidateToken validates a JWT token and returns the claims
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

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		// Check if token is revoked
		if am.isTokenRevoked(tokenString) {
			return nil, fmt.Errorf("token has been revoked")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RevokeToken revokes a JWT token
func (am *AuthMiddleware) RevokeToken(tokenString string) error {
	// Create a hash of the token for storage
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])

	// Store the revoked token hash in the database
	// This is a simplified implementation - in production you'd want to clean up expired tokens
	return am.db.Transaction(func(tx *buntdb.Tx) error {
		// Store with expiration time to allow cleanup
		expirationTime := time.Now().Add(24 * time.Hour) // Adjust based on your token expiration
		revokedToken := map[string]interface{}{
			"token_hash": tokenHash,
			"revoked_at": time.Now(),
			"expires_at": expirationTime,
		}

		data, _ := json.Marshal(revokedToken)
		_, _, err := tx.Set("revoked_tokens:"+tokenHash, string(data), nil)
		return err
	})
}

// isTokenRevoked checks if a token has been revoked
func (am *AuthMiddleware) isTokenRevoked(tokenString string) bool {
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])

	var isRevoked bool
	am.db.ViewTransaction(func(tx *buntdb.Tx) error {
		val, err := tx.Get("revoked_tokens:" + tokenHash)
		isRevoked = (err == nil && val != "")
		return nil
	})

	return isRevoked
}

// RequireAuth middleware that requires authentication
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := am.ValidateToken(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

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

// RequireRole middleware that requires a specific role
func (am *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				writeError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			if authCtx.Role != role && authCtx.Role != "admin" { // Admin can access everything
				writeError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth middleware that optionally extracts auth context if present
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Check for Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString := parts[1]

				// Validate token
				if claims, err := am.ValidateToken(tokenString); err == nil {
					// Create auth context
					authCtx := &AuthContext{
						UserID:   claims.UserID,
						Username: claims.Username,
						Role:     claims.Role,
						Token:    tokenString,
					}

					// Add auth context to request context
					ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
					r = r.WithContext(ctx)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// GetAuthContext extracts the auth context from the request
func GetAuthContext(r *http.Request) *AuthContext {
	if authCtx := r.Context().Value(AuthContextKey); authCtx != nil {
		if ctx, ok := authCtx.(*AuthContext); ok {
			return ctx
		}
	}
	return nil
}
