// Package supervision provides a bounded, policy-scoped retrieval adviser.
package supervision

import "math"

// Clef translation rules, carried over verbatim from the Clef KNIRV design
// (Clef_KNIRV_Design.md §4.2). They are pure, testable primitives; the full
// model-output → SupervisorAdvice converter (Task 4.10) wires them in once the
// 4.3 set-head output contract and scored choices exist. Encoding them here keeps
// a single, audited implementation of each rule instead of ad-hoc copies in every
// caller.
//
// The local Clef harness under pipeline/5_HEAD_TRAINER/clef-harness is the
// calibration baseline (reported ECE 0.63); HEAD must beat it. No hosted Clef
// calls are made from Go.

// DefaultAbstainTau is the abstention threshold for rule 3 (abstain when the
// peak probability is below tau). PROVISIONAL: its value must be fitted on a
// held-out split (Task 4.6) and is not a trained label. Until then every caller
// must use this one constant so the threshold is consistent and easy to replace.
const DefaultAbstainTau = 0.5

// NoulConfidence derives confidence for a binary (noul) question from P(yes)=pYes.
// Per Clef rule 2 the confidence is max(pYes, 1-pYes) — never the raw pYes — and
// the derivation is computed explicitly rather than measured from the wrong
// quantity. Inputs are clamped to [0,1] so a malformed probability cannot yield a
// negative or >1 confidence.
func NoulConfidence(pYes float64) float64 {
	if pYes < 0 {
		pYes = 0
	} else if pYes > 1 {
		pYes = 1
	}
	return math.Max(pYes, 1-pYes)
}

// NoulProbabilities synthesises the calibrated distribution for a noul answer as
// {true: pYes, false: 1-pYes}. A single abstention rule then works across all
// three Clef question types (noul/choice/score) because each reduces to a peak
// probability.
func NoulProbabilities(pYes float64) (pTrue, pFalse float64) {
	if pYes < 0 {
		pYes = 0
	} else if pYes > 1 {
		pYes = 1
	}
	return pYes, 1 - pYes
}

// PeakProbability returns the maximum value in probs (0 when empty), i.e. the
// confidence the model places on its most likely outcome. It is the quantity the
// abstention rule thresholds on.
func PeakProbability(probs []float64) float64 {
	peak := 0.0
	for _, p := range probs {
		if p > peak {
			peak = p
		}
	}
	return peak
}

// ShouldAbstain implements Clef rule 3: abstain when the peak probability is
// below tau. Abstention is a deterministic calibration threshold, never a
// model-chosen label. Use DefaultAbstainTau until it is fitted on held-out data.
func ShouldAbstain(peakProb, tau float64) bool {
	return peakProb < tau
}

// RoundSeverityForDisplay returns severity rounded to decimals places. Per Clef
// rule 1 severity is carried as a continuous float internally and rounded only
// at the display boundary; callers must not round before storing or comparing.
func RoundSeverityForDisplay(severity float64, decimals int) float64 {
	if decimals < 0 {
		decimals = 0
	}
	scale := math.Pow(10, float64(decimals))
	return math.Round(severity*scale) / scale
}
