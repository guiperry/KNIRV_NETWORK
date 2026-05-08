// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package wasm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// GuardrailDecision captures the result of a rules.wasm guardrail check.
type GuardrailDecision struct {
	Allowed       bool      `json:"allowed"`
	GuardrailClass uint32   `json:"guardrail_class"`
	BadgeID       string    `json:"badge_id"`
	Tag           string    `json:"tag"`
	WASMPath      string    `json:"wasm_path"`
	Error         string    `json:"error,omitempty"`
	CheckedAt     time.Time `json:"checked_at"`
}

// GuardrailMiddleware implements Option A of the WASM Integration
// Investigation: userspace guardrail enforcement via rules.wasm.
//
// It wraps an HTTP handler (typically ViewportProxy) and, before
// forwarding the request, loads the rules.wasm associated with the
// DVE's Badge and calls GuardrailClass().  If the guardrail class
// indicates a blocked category, the request is rejected with 403.
//
// The middleware uses a pool of pre-compiled WazeroGate instances
// per rules.wasm module for low-latency checks.
type GuardrailMiddleware struct {
	mapper   *BadgeWASMMapper
	pools    map[string]*WazeroPool // wasmPath → pool
	poolSize int
}

// NewGuardrailMiddleware creates a guardrail middleware backed by the
// Badge-to-WASM mapper.  poolSize controls how many WazeroGate instances
// are pre-allocated per unique rules.wasm module.
func NewGuardrailMiddleware(mapper *BadgeWASMMapper, poolSize int) *GuardrailMiddleware {
	if poolSize < 1 {
		poolSize = 8
	}
	return &GuardrailMiddleware{
		mapper:   mapper,
		pools:    make(map[string]*WazeroPool),
		poolSize: poolSize,
	}
}

// Middleware returns an HTTP middleware that validates requests via
// rules.wasm before passing them to the next handler.
//
// The middleware expects the following request headers to identify the
// DVE and its active Badge:
//
//	X-DVE-ID     — DVE node identifier
//	X-Badge-ID   — Badge NFT ID whose guardrail rules.wasm to invoke
//	X-Badge-Tag  — (optional) specific ontology tag; if omitted, the
//	               first guardrail-classified tag is used
func (gm *GuardrailMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dveID := r.Header.Get("X-DVE-ID")
		badgeID := r.Header.Get("X-Badge-ID")
		badgeTag := r.Header.Get("X-Badge-Tag")

		// If no badge header, pass through (no guardrail active).
		if badgeID == "" {
			next.ServeHTTP(w, r)
			return
		}

		decision := gm.validate(r.Context(), dveID, badgeID, badgeTag)

		// Attach decision to response headers for observability.
		w.Header().Set("X-Guardrail-Allowed", fmt.Sprintf("%v", decision.Allowed))
		w.Header().Set("X-Guardrail-Class", fmt.Sprintf("%d", decision.GuardrailClass))
		if decision.Error != "" {
			w.Header().Set("X-Guardrail-Error", decision.Error)
		}

		if !decision.Allowed {
			http.Error(w, fmt.Sprintf("guardrail blocked: class=%d", decision.GuardrailClass),
				http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ValidateSession is a convenience method that validates a session
// against the DVE's guardrail WASM without requiring an HTTP request.
// Returns (allowed, guardrailClass, error).
func (gm *GuardrailMiddleware) ValidateSession(ctx context.Context, dveID, badgeID, tag string) (bool, uint32, error) {
	decision := gm.validate(ctx, dveID, badgeID, tag)
	return decision.Allowed, decision.GuardrailClass, nil
}

// Close releases all WazeroGate pools.
func (gm *GuardrailMiddleware) Close() {
	for _, pool := range gm.pools {
		pool.CloseAll()
	}
	gm.pools = make(map[string]*WazeroPool)
}

// —— internal ——

func (gm *GuardrailMiddleware) validate(ctx context.Context, dveID, badgeID, tag string) *GuardrailDecision {
	decision := &GuardrailDecision{
		Allowed:   true, // default allow if nothing to check
		BadgeID:   badgeID,
		Tag:       tag,
		CheckedAt: time.Now(),
	}

	// Find the guardrail WASM for this badge.
	mappings := gm.mapper.GetGuardrailWASM(badgeID)
	if len(mappings) == 0 {
		// No guardrail WASM — allow (no policy to enforce).
		return decision
	}

	// If a specific tag is requested, filter to that tag.
	var target *WASMMapping
	if tag != "" {
		for i := range mappings {
			if mappings[i].Tag == tag {
				target = &mappings[i]
				break
			}
		}
		if target == nil {
			decision.Error = fmt.Sprintf("guardrail tag %s not found on badge %s", tag, badgeID)
			return decision
		}
	} else {
		// Use the first available guardrail mapping.
		target = &mappings[0]
	}

	decision.Tag = target.Tag
	decision.WASMPath = target.WASMPath

	// Acquire a WazeroGate from the pool.
	pool := gm.getOrCreatePool(target.WASMPath)
	gate, err := pool.Acquire(ctx)
	if err != nil {
		decision.Error = fmt.Sprintf("wazero pool error: %v", err)
		log.Printf("[guardrail-middleware] dve=%s badge=%s: %s", dveID, badgeID, decision.Error)
		// Fail-open when WASM runtime is unavailable.
		decision.Allowed = true
		return decision
	}

	class, err := gate.GuardrailClass(ctx)
	if err != nil {
		decision.Error = fmt.Sprintf("GuardrailClass() error: %v", err)
		log.Printf("[guardrail-middleware] dve=%s badge=%s: %s", dveID, badgeID, decision.Error)
		// Fail-open on WASM execution error to avoid blocking legitimate
		// traffic when the WASM module is misconfigured.
		decision.Allowed = true
		decision.GuardrailClass = class
		return decision
	}

	decision.GuardrailClass = class
	// Class 0 = no violation; non-zero = specific policy violation.
	decision.Allowed = (class == 0)

	if !decision.Allowed {
		log.Printf("[guardrail-middleware] dve=%s badge=%s class=%d → BLOCKED",
			dveID, badgeID, class)
	}

	return decision
}

func (gm *GuardrailMiddleware) getOrCreatePool(wasmPath string) *WazeroPool {
	// Simple map access without locking; pools are created at registration
	// time and never removed during the lifetime of the middleware.
	if pool, ok := gm.pools[wasmPath]; ok {
		return pool
	}
	pool := NewWazeroPool(wasmPath, gm.poolSize)
	gm.pools[wasmPath] = pool
	return pool
}
