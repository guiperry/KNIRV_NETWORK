// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package badge

import (
	"log"
	"sync"

	knirvshell "github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL"
)

// BadgeValueSignal carries governance values from an attached badge into the
// ICME alignment loop for continuous value-alignment enforcement.
type BadgeValueSignal struct {
	BadgeID           string            `json:"badge_id"`
	DVEID             string            `json:"dve_id"`
	ValueSignals      []string          `json:"value_signals"`
	AlignmentThreshold float64          `json:"alignment_threshold"`
	AuthConstraints   map[string]string `json:"auth_constraints,omitempty"`
}

// ICMEAlignmentBridge wires badge value signals into the existing ICME
// alignment loop so that badge-value misalignment triggers guardrail
// violations and adaptive refinement.
//
// This implements Workflow E Step 13 of the production plan:
//
//	"Value alignment: ICME AlignmentLoop scores outputs against the
//	 badge's value signals; misaligned outputs trigger guardrail violations."
type ICMEAlignmentBridge struct {
	injector   *GuardrailInjector
	registry   BadgeRegistryInterface

	// signalSink is called when badge value alignment should be evaluated.
	// Typically this is the ICME AlignmentLoop.Evaluate or a custom adapter.
	signalSink AlignmentSink

	// activeSignals tracks the current value-signal state per DVE.
	activeSignals map[string]*BadgeValueSignal // dveID → signal
	mu            sync.RWMutex
}

// AlignmentSink receives badge value signals for ICME processing.
type AlignmentSink interface {
	IngestBadgeSignal(signal *BadgeValueSignal) error
}

// NewICMEAlignmentBridge creates a bridge between badge attachment and ICME.
func NewICMEAlignmentBridge(injector *GuardrailInjector, registry BadgeRegistryInterface) *ICMEAlignmentBridge {
	return &ICMEAlignmentBridge{
		injector:      injector,
		registry:      registry,
		activeSignals: make(map[string]*BadgeValueSignal),
	}
}

// SetAlignmentSink wires the bridge into the ICME alignment loop.
func (b *ICMEAlignmentBridge) SetAlignmentSink(sink AlignmentSink) {
	b.signalSink = sink
}

// OnBadgeAttached is called after a badge is attached to a DVE.
// It constructs a BadgeValueSignal from the badge's metadata and
// pushes it to the ICME alignment loop.
func (b *ICMEAlignmentBridge) OnBadgeAttached(dveID, badgeID string) error {
	badge, err := b.registry.GetBadge(badgeID)
	if err != nil {
		return err
	}

	signal := b.buildSignal(dveID, badge)

	b.mu.Lock()
	b.activeSignals[dveID] = signal
	b.mu.Unlock()

	if b.signalSink != nil {
		if err := b.signalSink.IngestBadgeSignal(signal); err != nil {
			log.Printf("[icme-bridge] failed to ingest badge signal %s for DVE %s: %v",
				badgeID, dveID, err)
			return err
		}
	}

	log.Printf("[icme-bridge] badge %s value signals active for DVE %s (threshold=%.2f)",
		badgeID, dveID, signal.AlignmentThreshold)
	return nil
}

// OnBadgeDetached removes the badge's value signals from the DVE.
func (b *ICMEAlignmentBridge) OnBadgeDetached(dveID, badgeID string) {
	b.mu.Lock()
	delete(b.activeSignals, dveID)
	b.mu.Unlock()
	log.Printf("[icme-bridge] badge %s value signals removed for DVE %s", badgeID, dveID)
}

// GetActiveSignals returns all signals currently active across all DVEs.
func (b *ICMEAlignmentBridge) GetActiveSignals() []*BadgeValueSignal {
	b.mu.RLock()
	defer b.mu.RUnlock()

	signals := make([]*BadgeValueSignal, 0, len(b.activeSignals))
	for _, s := range b.activeSignals {
		signals = append(signals, s)
	}
	return signals
}

// GetDVESignal returns the active signal for a specific DVE, or nil.
func (b *ICMEAlignmentBridge) GetDVESignal(dveID string) *BadgeValueSignal {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.activeSignals[dveID]
}

// —— internal ——

func (b *ICMEAlignmentBridge) buildSignal(dveID string, badge *knirvshell.BadgeInfo) *BadgeValueSignal {
	signal := &BadgeValueSignal{
		BadgeID:            badge.ID,
		DVEID:              dveID,
		AlignmentThreshold: badge.AlignmentThreshold,
		AuthConstraints:    make(map[string]string),
	}

	// Collect value signals.
	signal.ValueSignals = badge.ValueSignals
	if len(signal.ValueSignals) == 0 {
		if raw, ok := badge.Metadata["values"]; ok {
			if arr, ok := raw.([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						signal.ValueSignals = append(signal.ValueSignals, s)
					}
				}
			}
		}
	}

	// Collect auth constraints from JWT claims and role bindings.
	for _, jc := range badge.JWTClaims {
		if jc.Action == "enforce" {
			signal.AuthConstraints["jwt:"+jc.Claim] = jc.Value
		}
	}
	for _, rb := range badge.RoleBindings {
		signal.AuthConstraints["role:"+rb.Role] = "enforced"
	}

	// Default alignment threshold if not set on the badge.
	if signal.AlignmentThreshold == 0 {
		signal.AlignmentThreshold = 0.75
	}

	return signal
}
