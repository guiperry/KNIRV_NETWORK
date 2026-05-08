// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package badge provides Badge lifecycle services bridging badge creation
// to GuardrailEngine rule injection, ICME alignment enforcement, and SVG
// visual generation.
package badge

import (
	"fmt"
	"log"
	"sync"

	knirvshell "github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL"
)

// GuardrailInjector bridges badge attachment (MintBadge) to automatic
// GuardrailEngine policy injection.  When a badge is attached to a DVE:
//   1. The badge's ontology signals become guardrail domain constraints.
//   2. The badge's value signals become guardrail policy rules.
//   3. The badge's JWT claims and role bindings become auth enforcement rules.
//   4. The badge's API keys are registered as scoped credentials.
//
// This implements Workflow E Step 11-13 of the production plan:
//
//	"Badge enforcement ... every request to the DVE is checked against
//	 the badge's embedded ontology tags and value signals."
type GuardrailInjector struct {
	// guardrailEngine is the target policy engine (from the cognitiveengine package).
	// Avoids direct import cycle via interface.
	guardrailEngine GuardrailEngineInterface
	wasmMapper      WASMMapperInterface // optional Badge-to-WASM hook
	badgeRegistry   BadgeRegistryInterface
	mu              sync.Mutex
	// badgeAttachments tracks which badges are attached to which DVEs.
	badgeAttachments map[string][]string // dveID → []badgeID
}

// GuardrailEngineInterface abstracts the GuardrailEngine methods used by the injector.
type GuardrailEngineInterface interface {
	InjectBadgeRules(dveID, badgeID string, ontologyTags []string)
	RemoveBadgeRules(badgeID string)
}

// WASMMapperInterface abstracts the BadgeWASMMapper (optional, for WASM guardrail support).
type WASMMapperInterface interface {
	RegisterBadge(badgeID string, ontologyTags []string) error
	RemoveBadge(badgeID string)
}

// BadgeRegistryInterface provides badge metadata lookups.
type BadgeRegistryInterface interface {
	GetBadge(badgeID string) (*knirvshell.BadgeInfo, error)
}

// NewGuardrailInjector creates an injector wired to the target engine.
func NewGuardrailInjector(
	engine GuardrailEngineInterface,
	registry BadgeRegistryInterface,
) *GuardrailInjector {
	return &GuardrailInjector{
		guardrailEngine:  engine,
		badgeRegistry:    registry,
		badgeAttachments: make(map[string][]string),
	}
}

// SetWASMMapper optionally wires the Badge-to-WASM mapper so that guardrail
// rules also trigger .wasm loading for the attached badge.
func (gi *GuardrailInjector) SetWASMMapper(m WASMMapperInterface) {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	gi.wasmMapper = m
}

// AttachBadge is called when a badge is minted/attached to a DVE.
// It:
//   - Retrieves the badge metadata (value signals, ontology signals).
//   - Injects guardrail policies for the ontology tags.
//   - Registers the badge with the WASM mapper if configured.
//   - Tracks the attachment for later removal / stacking.
func (gi *GuardrailInjector) AttachBadge(dveID, badgeID string) error {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	// Retrieve badge metadata.
	badge, err := gi.badgeRegistry.GetBadge(badgeID)
	if err != nil {
		return fmt.Errorf("badge registry lookup %s: %w", badgeID, err)
	}

	// Collect ontology tags for guardrail rule injection.
	ontologyTags := badge.OntologySignals
	if len(ontologyTags) == 0 {
		// Fall back to metadata if ontology signals are not populated directly.
		if raw, ok := badge.Metadata["ontology"]; ok {
			if arr, ok := raw.([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						ontologyTags = append(ontologyTags, s)
					}
				}
			}
		}
	}

	// Collect value signals for alignment enforcement.
	valueSignals := badge.ValueSignals
	if len(valueSignals) == 0 {
		if raw, ok := badge.Metadata["values"]; ok {
			if arr, ok := raw.([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						valueSignals = append(valueSignals, s)
					}
				}
			}
		}
	}

	// Inject guardrail rules into the engine.
	// The ontology tags become per-DVE guardrail domain constraints.
	// The value signals become policy rules scoped to this badge.
	allTags := append(ontologyTags, valueSignals...)
	gi.guardrailEngine.InjectBadgeRules(dveID, badgeID, allTags)

	// Register with WASM mapper if configured (for rules.wasm / resolution.wasm).
	if gi.wasmMapper != nil {
		if err := gi.wasmMapper.RegisterBadge(badgeID, allTags); err != nil {
			log.Printf("[badge-injector] WASM mapper registration for badge %s: %v", badgeID, err)
		}
	}

	// Track attachment.
	gi.badgeAttachments[dveID] = append(gi.badgeAttachments[dveID], badgeID)

	log.Printf("[badge-injector] attached badge %s to DVE %s (%d ontology + %d value tags)",
		badgeID, dveID, len(ontologyTags), len(valueSignals))
	return nil
}

// DetachBadge removes a badge from a DVE and cleans up its guardrail rules.
func (gi *GuardrailInjector) DetachBadge(dveID, badgeID string) {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	// Remove from tracking.
	attached := gi.badgeAttachments[dveID]
	filtered := make([]string, 0, len(attached))
	for _, bid := range attached {
		if bid != badgeID {
			filtered = append(filtered, bid)
		}
	}
	gi.badgeAttachments[dveID] = filtered

	// Clean up guardrail rules.
	gi.guardrailEngine.RemoveBadgeRules(badgeID)

	// Clean up WASM mappings.
	if gi.wasmMapper != nil {
		gi.wasmMapper.RemoveBadge(badgeID)
	}

	log.Printf("[badge-injector] detached badge %s from DVE %s", badgeID, dveID)
}

// GetAttachedBadges returns the list of badge IDs currently attached to a DVE.
func (gi *GuardrailInjector) GetAttachedBadges(dveID string) []string {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	attached := gi.badgeAttachments[dveID]
	out := make([]string, len(attached))
	copy(out, attached)
	return out
}

// ValidateCredentialAccess checks whether a given credential (by key ID) is
// scoped within any badge attached to the DVE.  Returns the first matching
// badge ID and true if the credential is authorized.
func (gi *GuardrailInjector) ValidateCredentialAccess(dveID, keyID string) (string, bool) {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	for _, badgeID := range gi.badgeAttachments[dveID] {
		badge, err := gi.badgeRegistry.GetBadge(badgeID)
		if err != nil {
			continue
		}
		for _, key := range badge.APIKeys {
			if key.KeyID == keyID {
				return badgeID, true
			}
		}
	}
	return "", false
}

// ValidateJWTClaim checks whether a JWT claim is authorized by any badge
// attached to the DVE.  Returns true if the claim is enforced+allowed,
// or if no badge enforces this claim (default allow for unguarded claims).
func (gi *GuardrailInjector) ValidateJWTClaim(dveID, claim, value string) (allowed bool, badgeID string) {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	for _, bid := range gi.badgeAttachments[dveID] {
		badge, err := gi.badgeRegistry.GetBadge(bid)
		if err != nil {
			continue
		}
		for _, jc := range badge.JWTClaims {
			if jc.Claim == claim && jc.Action == "enforce" {
				if jc.Value == value {
					return true, bid
				}
				return false, bid
			}
		}
	}
	// No badge enforces this claim — default allow.
	return true, ""
}

// Stats returns injector statistics.
func (gi *GuardrailInjector) Stats() map[string]interface{} {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	totalDVEs := len(gi.badgeAttachments)
	totalAttachments := 0
	for _, a := range gi.badgeAttachments {
		totalAttachments += len(a)
	}

	return map[string]interface{}{
		"dves_with_badges":   totalDVEs,
		"total_attachments":  totalAttachments,
	}
}
