package factslice

import (
	"fmt"

	"knirvmodels/pkg/evidence"
)

// Thresholds are the calibrated-probability cut points of the spectrum (section 5.7).
// They are configuration: the initial values are the plan's, to be set from the
// calibrated curve on the gold set (Task 0.5), not guessed.
type Thresholds struct {
	Refused     float64 `json:"refused"`     // a below this is level 0
	Discouraged float64 `json:"discouraged"` // below: 1
	Uncommitted float64 `json:"uncommitted"` // below: 2
	Seen        float64 `json:"seen"`        // at or above: 3 (conditions permitting)
	Attested    float64 `json:"attested"`    // 4
	Accepted    float64 `json:"accepted"`    // 5
	Committed   float64 `json:"committed"`   // 6
}

// DefaultThresholds are the plan's initial values: a<0.15/0.30/0.50/0.70/0.85/0.90.
// Provisional until fitted on the gold set (D-H2 is open).
func DefaultThresholds() Thresholds {
	return Thresholds{Refused: 0.15, Discouraged: 0.30, Uncommitted: 0.50, Seen: 0.50, Attested: 0.70, Accepted: 0.85, Committed: 0.90}
}

// Validate refuses thresholds that are not an increasing ladder inside (0,1).
func (t Thresholds) Validate() error {
	ladder := []struct {
		name string
		v    float64
	}{{"refused", t.Refused}, {"discouraged", t.Discouraged}, {"uncommitted", t.Uncommitted}, {"attested", t.Attested}, {"accepted", t.Accepted}, {"committed", t.Committed}}
	prev := 0.0
	for _, s := range ladder {
		if s.v <= prev || s.v >= 1 {
			return fmt.Errorf("threshold %s=%v must be above %v and below 1", s.name, s.v, prev)
		}
		prev = s.v
	}
	if t.Seen != t.Uncommitted {
		return fmt.Errorf("threshold seen=%v must equal uncommitted=%v: level 3 begins where level 2 ends", t.Seen, t.Uncommitted)
	}
	return nil
}

// refutedByReference reports whether the claim was refuted on reference-grade evidence.
func refutedByReference(r ClaimResult) bool { return r.Verdict == Refuted && r.ReferenceTier }

// Gate maps one claim's checks to its level. It is a pure function: the same
// ClaimResult and Thresholds always give the same level, and no model supplies it.
//
// Reading of section 5.7 (the owner confirms under O2):
//   - A degraded, failed or pending check is exactly "uncommitted": it is not a
//     judgement the claim is false, and it can never exceed level 2.
//   - No evidence (and no degradation) or a refutation by reference-tier evidence
//     is "refused", as is a calibrated a below the first threshold.
//   - Levels 3 to 6 are cumulative: each needs every condition of the levels below
//     it. Level 4 needs witnessed provenance for every cited span; level 5 adds
//     reference-tier evidence, judge agreement and corroboration; level 6 adds a
//     proof, a certificate and an anchor.
func Gate(r ClaimResult, t Thresholds) GateLevel {
	if r.Degraded || r.Failed || r.Pending {
		return GateUncommitted
	}
	if len(r.Evidence) == 0 {
		return GateRefused
	}
	a := r.Calibrated
	if refutedByReference(r) || a < t.Refused {
		return GateRefused
	}
	if r.Verdict == Refuted || r.DisagreeToRefuted || a < t.Discouraged {
		return GateDiscouraged
	}
	if r.Verdict != Supported || a < t.Uncommitted {
		return GateUncommitted
	}
	level := GateSeen // supported, a >= 0.50
	if a >= t.Attested && r.Witnessed {
		level = GateAttested
		if a >= t.Accepted && r.ReferenceTier && r.Agreement && r.Corroborated {
			level = GateAccepted
			if a >= t.Committed && r.Proof && r.Certificate && r.Anchored {
				level = GateCommitted
			}
		}
	}
	return level
}

// ResponseGate is the weakest checkable claim. With nothing checkable it is
// "uncommitted": nothing was verified, so nothing is accepted, and nothing was
// refuted either (provisional).
func ResponseGate(results []ClaimResult) GateLevel {
	if len(results) == 0 {
		return GateUncommitted
	}
	min := GateCommitted
	for _, r := range results {
		if r.Gate < min {
			min = r.Gate
		}
	}
	return min
}

// finish fills the response-level and legacy fields from the per-claim results.
func finish(res *Result) {
	res.Gate = ResponseGate(res.Claims)
	total := len(res.Claims) + len(res.Unchecked)
	if total > 0 {
		res.UncheckedFraction = float64(len(res.Unchecked)) / float64(total)
	}
	// Nothing was verified when there is no checkable claim, so nothing is charged: a
	// charge always has a matching verdict.
	res.Chargeable = len(res.Claims) > 0
	for _, c := range res.Claims {
		if c.Degraded || c.Failed || c.Pending {
			res.Chargeable = false
		}
	}
	// Legacy: refused = level 0, is_accurate = level >= 5, confidence = the weakest
	// checkable claim's calibrated probability.
	res.Refused = res.Gate == GateRefused
	res.IsAccurate = res.Gate >= GateAccepted
	res.Confidence = 0
	if len(res.Claims) > 0 {
		res.Confidence = 1
		for _, c := range res.Claims {
			if c.Calibrated < res.Confidence {
				res.Confidence = c.Calibrated
			}
		}
	}
	res.Citations = res.Citations[:0]
	offset := 0
	for _, c := range res.Claims {
		cited := map[string]bool{}
		for _, id := range c.Cited {
			cited[id] = true
		}
		for i, h := range c.Evidence {
			if cited[h.SpanID] {
				res.Citations = append(res.Citations, offset+i)
			}
		}
		offset += len(c.Evidence)
	}
}

// referenceGrade is the tier rule for level 5: reference and ontology evidence.
func referenceGrade(t evidence.Tier) bool {
	return t == evidence.TierReference || t == evidence.TierOntology
}
