package network

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

type traceContextKey struct{}

func TraceID(ctx context.Context) string { id, _ := ctx.Value(traceContextKey{}).(string); return id }
func tracingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			traceID := traceIDFromHeader(r.Header.Get("traceparent"))
			if traceID == "" {
				var b [16]byte
				if _, err := rand.Read(b[:]); err == nil {
					traceID = hex.EncodeToString(b[:])
				}
			}
			w.Header().Set("X-Trace-ID", traceID)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), traceContextKey{}, traceID)))
			logger.Debug("request completed", zap.String("trace_id", traceID), zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Duration("duration", time.Since(start)))
		})
	}
}
func traceIDFromHeader(header string) string {
	parts := strings.Split(header, "-")
	if len(parts) != 4 || len(parts[1]) != 32 {
		return ""
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return ""
	}
	return parts[1]
}
