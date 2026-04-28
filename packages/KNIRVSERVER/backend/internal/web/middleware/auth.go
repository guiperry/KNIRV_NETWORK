package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tidwall/buntdb"

	"backend_server/internal/database"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	db           *database.BuntDBManager
	jwtSecret    []byte
	authRequired bool
}

// SetAuthRequired enables or disables authentication requirement
func (am *AuthMiddleware) SetAuthRequired(required bool) {
	am.authRequired = required
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
func NewAuthMiddleware(db *database.BuntDBManager, jwtSecret string, authRequired bool) (*AuthMiddleware, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret cannot be empty")
	}

	return &AuthMiddleware{
		db:           db,
		jwtSecret:    []byte(jwtSecret),
		authRequired: authRequired,
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
			Issuer:    "knirv-server",
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
		// If auth is not required, skip authentication
		if !am.authRequired {
			next.ServeHTTP(w, r)
			return
		}

		// Allow OPTIONS requests for CORS preflight
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		// Check for Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "Invalid authorization format")
			return
		}

		tokenString := parts[1]

		// Try validating as JWT
		if claims, err := am.ValidateToken(tokenString); err == nil {
			authCtx := &AuthContext{
				UserID:   claims.UserID,
				Username: claims.Username,
				Role:     claims.Role,
				Token:    tokenString,
			}
			ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
			r = r.WithContext(ctx)
		} else if authCtx, err := am.validateSessionToken(tokenString); err == nil {
			// Try validating as Session Token
			ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
			r = r.WithContext(ctx)
		} else {
			writeError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

	next.ServeHTTP(w, r)
	})
}

// validateSessionToken checks for a session token in the database
func (am *AuthMiddleware) validateSessionToken(token string) (*AuthContext, error) {
	var authCtx *AuthContext

	err := am.db.ViewTransaction(func(tx *buntdb.Tx) error {
		// Check user sessions
		found := false
		tx.AscendKeys("sessions:*", func(key, value string) bool {
			var s struct {
				UserID   string    `json:"user_id"`
				Token    string    `json:"token"`
				IsActive bool      `json:"is_active"`
				ExpiresAt time.Time `json:"expires_at"`
			}
			if err := json.Unmarshal([]byte(value), &s); err == nil && s.Token == token && s.IsActive && time.Now().Before(s.ExpiresAt) {
				authCtx = &AuthContext{
					UserID: s.UserID,
					Token:  token,
					Role:   "user", // Default for session
				}
				found = true
				return false
			}
			return true
		})

		if found {
			return nil
		}

		// Check DVE rentals (access tokens)
		tx.AscendKeys("rental:*", func(key, value string) bool {
			var r struct {
				UserID      string    `json:"user_id"`
				AccessToken string    `json:"access_token"`
				Status      string    `json:"status"`
				EndTime     time.Time `json:"end_time"`
			}
			if err := json.Unmarshal([]byte(value), &r); err == nil && r.AccessToken == token && r.Status == "active" && time.Now().Before(r.EndTime) {
				authCtx = &AuthContext{
					UserID: r.UserID,
					Token:  token,
					Role:   "validator", // DVE tokens act as validator role
				}
				found = true
				return false
			}
			return true
		})

		if found {
			return nil
		}

		return fmt.Errorf("session token not found")
	})

	return authCtx, err
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

				// Try validating as JWT
				if claims, err := am.ValidateToken(tokenString); err == nil {
					authCtx := &AuthContext{
						UserID:   claims.UserID,
						Username: claims.Username,
						Role:     claims.Role,
						Token:    tokenString,
					}
					ctx := context.WithValue(r.Context(), AuthContextKey, authCtx)
					r = r.WithContext(ctx)
				} else if authCtx, err := am.validateSessionToken(tokenString); err == nil {
					// Try validating as Session Token
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

// GetUserIDFromRequest extracts the user ID from the authenticated request
// Returns an empty string if no valid auth context exists
func GetUserIDFromRequest(r *http.Request) string {
	authCtx := GetAuthContext(r)
	if authCtx != nil {
		return authCtx.UserID
	}
	return ""
}

// GetUsernameFromRequest extracts the username from the authenticated request
// Returns an empty string if no valid auth context exists
func GetUsernameFromRequest(r *http.Request) string {
	authCtx := GetAuthContext(r)
	if authCtx != nil {
		return authCtx.Username
	}
	return ""
}

// writeError writes an error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// testnetTokens maps development testnet tokens to their auth contexts.
// These tokens are only for local development and match the frontend's hardcoded testnet tokens.
// In production, this map remains nil and getTestnetAuthContext always returns nil.
var testnetTokens map[string]*AuthContext

func init() {
	// Only load testnet tokens in development/test environments
	env := os.Getenv("ENVIRONMENT")
	if env == "development" || env == "testnet" || env == "test" {
		testnetTokens = map[string]*AuthContext{
			"testnet-admin-123": {
				UserID:   "testnet-admin",
				Username: "testnet-admin",
				Role:     "admin",
			},
			"testnet-validator-456": {
				UserID:   "testnet-validator",
				Username: "testnet-validator",
				Role:     "validator",
			},
			"testnet-observer-789": {
				UserID:   "testnet-observer",
				Username: "testnet-observer",
				Role:     "observer",
			},
		}
	}
}

// getTestnetAuthContext returns an AuthContext for known development testnet tokens, or nil.
// In production (ENVIRONMENT=production), this always returns nil.
func getTestnetAuthContext(token string) *AuthContext {
	// Return nil in production to disable testnet tokens
	if os.Getenv("ENVIRONMENT") == "production" {
		return nil
	}

	// If testnet tokens aren't loaded (e.g., wrong environment), return nil
	if testnetTokens == nil {
		return nil
	}

	if ctx, ok := testnetTokens[token]; ok {
		return &AuthContext{
			UserID:   ctx.UserID,
			Username: ctx.Username,
			Role:     ctx.Role,
		}
	}
	return nil
}
