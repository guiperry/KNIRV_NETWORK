package server

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// AuthMiddleware extracts and performs initial validation on the API key.
func AuthMiddleware(logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Warn("Missing credentials", zap.String("path", r.URL.Path), zap.String("method", r.Method))
			http.Error(w, "Missing credentials", http.StatusUnauthorized)
			return
		}

		fullKey := strings.TrimPrefix(authHeader, "Bearer ")
		parts := strings.Split(fullKey, ".")
		if len(parts) != 2 {
			logger.Warn("Invalid key format", zap.String("path", r.URL.Path), zap.String("method", r.Method))
			http.Error(w, "Invalid key format", http.StatusUnauthorized)
			return
		}
		keyID, secret := parts[0], parts[1]

		clientIP := r.RemoteAddr // Simplified; use a proper IP extraction method

		ctx := context.WithValue(r.Context(), "keyID", keyID)
		ctx = context.WithValue(ctx, "secret", secret)
		ctx = context.WithValue(ctx, "clientIP", clientIP)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}