package wasm

import (
	"context"
	"log"
	"net/http"
	"time"
)

// GuardrailDecision captures the result of a badge guardrail check.
type GuardrailDecision struct {
	Allowed        bool      `json:"allowed"`
	GuardrailClass uint32    `json:"guardrail_class"`
	BadgeID        string    `json:"badge_id"`
	Tag            string    `json:"tag"`
	Error          string    `json:"error,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

// GuardrailMiddleware enforces badge-based access control using the
// BadgeConfigResolver — no WASM modules needed.
type GuardrailMiddleware struct {
	resolver *BadgeConfigResolver
}

// NewGuardrailMiddleware creates a guardrail middleware backed by the
// BadgeConfigResolver for badge lookups.
func NewGuardrailMiddleware(resolver *BadgeConfigResolver) *GuardrailMiddleware {
	return &GuardrailMiddleware{resolver: resolver}
}

// Middleware returns an HTTP handler that checks X-DVE-ID, X-Badge-ID,
// and X-Badge-Tag headers against the BadgeConfigResolver. If the badge
// is registered, the request is allowed; otherwise a 403 is returned.
func (gm *GuardrailMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dveID := r.Header.Get("X-DVE-ID")
		badgeID := r.Header.Get("X-Badge-ID")
		tag := r.Header.Get("X-Badge-Tag")

		if dveID == "" || badgeID == "" {
			next.ServeHTTP(w, r)
			return
		}

		allowed, class, err := gm.ValidateSession(r.Context(), dveID, badgeID, tag)
		if err != nil {
			log.Printf("[guardrail-middleware] validation error: %v", err)
			http.Error(w, `{"error":"guardrail validation failed"}`, http.StatusInternalServerError)
			return
		}
		if !allowed {
			log.Printf("[guardrail-middleware] denied: dve=%s badge=%s tag=%s class=%d",
				dveID, badgeID, tag, class)
			http.Error(w, `{"error":"guardrail validation denied"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ValidateSession checks whether a badge has a registered config.
func (gm *GuardrailMiddleware) ValidateSession(ctx context.Context, dveID, badgeID, tag string) (bool, uint32, error) {
	if gm.resolver == nil {
		return true, 0, nil
	}
	config := gm.resolver.GetConfig(badgeID)
	if config == nil {
		return false, 0, nil
	}
	return true, config.ErrorClassID, nil
}

// Close is a no-op retained for interface compatibility.
func (gm *GuardrailMiddleware) Close() {}
